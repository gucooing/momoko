package biz

import (
	"context"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/constant"
	"momoko/pkg/utils"
)

const (
	apiKeyCreateRetry = 5
	// APIKeyExpiringWindow 距过期不足该时长的 API Key 视为“即将过期”。
	APIKeyExpiringWindow = 30 * 24 * time.Hour
)

// APIKeyStatusCounts 按有效状态统计的 API Key 数量。
type APIKeyStatusCounts struct {
	Valid    int64
	Expiring int64
	Expired  int64
}

// APIKeyOwnerCount 所属用户及其 API Key 数量。
type APIKeyOwnerCount struct {
	User  *gen.User
	Count int64
}

type APIKeyRepo interface {
	CreateUserAPIKey(ctx context.Context, userID, name, remark, apiKey string, expiresAt *time.Time, permissions []string) (*gen.UserAPIKey, error)
	GetUserAPIKey(ctx context.Context, userID, id string) (*gen.UserAPIKey, error)
	GetAPIKeyByKeyID(ctx context.Context, keyID string) (*gen.UserAPIKey, error)
	// ListUserAPIKeys 分页列出（userID 为空表示全部用户）；status 为空表示全部，counts 按关键字统计（不受 status 影响）。
	ListUserAPIKeys(ctx context.Context, userID string, page, pageSize int64, keywords *string, status *v1.APIKeyStatus) ([]*gen.UserAPIKey, int64, *APIKeyStatusCounts, error)
	// APIKeyOwners 全部 API Key 的所属用户及各自的密钥数。
	APIKeyOwners(ctx context.Context) ([]*APIKeyOwnerCount, error)
	// UpdateUserAPIKey / DeleteUserAPIKey 的 userID 为空表示不限所属用户。
	UpdateUserAPIKey(ctx context.Context, userID, id string, name, remark *string, expiresAt *time.Time, clearExpiresAt *bool, permissions *[]string) (*gen.UserAPIKey, error)
	// TouchAPIKeyUsed 记录最近一次被调用的时间与来源 IP（不改动 update_time）。
	TouchAPIKeyUsed(ctx context.Context, id string, at time.Time, ip string) error
	RefreshUserAPIKey(ctx context.Context, userID, id, apiKey string) (*gen.UserAPIKey, error)
	DeleteUserAPIKey(ctx context.Context, userID, id string) error
}

// NodeUsecase 管理两类数据：本机签发给远程主控的 API Key，以及本机作为主控接入的远程节点。
type NodeUsecase struct {
	repo    APIKeyRepo
	nodes   NodeRepo
	system  *SystemUsecase
	clients *nodeClients
	status  *nodeStatusCache
}

func NewNodeUsecase(repo APIKeyRepo, nodes NodeRepo, system *SystemUsecase) (*NodeUsecase, func()) {
	n := &NodeUsecase{
		repo:    repo,
		nodes:   nodes,
		system:  system,
		clients: newNodeClients(),
		status:  newNodeStatusCache(),
	}
	stop := n.startProber()
	return n, func() {
		stop()
		n.clients.closeAll()
	}
}

func (n *NodeUsecase) CreateAPIKey(ctx context.Context, userID string, req *v1.CreateAPIKeyRequest) (*v1.APIKeyInfo, error) {
	if req.Name == "" {
		return nil, ErrAPIKeyNameEmpty
	}
	permissions, err := normalizeNodePermissions(req.Permissions)
	if err != nil {
		return nil, err
	}
	var expiration *time.Time
	if req.ExpiresAt != nil {
		expiration = new(req.ExpiresAt.AsTime())
	}

	for range apiKeyCreateRetry {
		apiKey, err := utils.GenerateAPIKey()
		if err != nil {
			return nil, ErrSystem(err)
		}

		item, err := n.repo.CreateUserAPIKey(ctx, userID, req.Name, strings.TrimSpace(req.Remark), apiKey, expiration, permissions)
		if err == nil {
			return n.toAPIKeyInfo(item, true, userID), nil
		}
		if !gen.IsConstraintError(err) {
			return nil, ErrSystem(err)
		}
	}

	return nil, ErrAPIKeyGenerate
}

// canManageAllAPIKeys 当前用户是否可以查看 / 吊销 / 删除全部用户的 API Key。
func (n *NodeUsecase) canManageAllAPIKeys(ctx context.Context) bool {
	return n.system != nil && n.system.Check(ctx, constant.ApiKeyManage) == nil
}

// ListAPIKeys 默认只列出自己的密钥；all_users 时列出全部用户的密钥（需要 api_key:manage），并返回所属用户统计。
func (n *NodeUsecase) ListAPIKeys(ctx context.Context, userID string, req *v1.ListAPIKeysRequest) ([]*v1.APIKeyInfo, int64, *APIKeyStatusCounts, []*v1.APIKeyOwnerStat, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	scope := userID
	var owners []*v1.APIKeyOwnerStat
	if req.AllUsers {
		if !n.canManageAllAPIKeys(ctx) {
			return nil, 0, nil, nil, ErrNoPermission
		}
		scope = ""
		if req.OwnerId != nil {
			scope = strings.TrimSpace(*req.OwnerId)
		}
		stats, err := n.repo.APIKeyOwners(ctx)
		if err != nil {
			return nil, 0, nil, nil, ErrSystem(err)
		}
		owners = make([]*v1.APIKeyOwnerStat, 0, len(stats))
		for _, st := range stats {
			owners = append(owners, &v1.APIKeyOwnerStat{Owner: toAPIKeyOwner(st.User), Count: st.Count})
		}
	}

	items, total, counts, err := n.repo.ListUserAPIKeys(ctx, scope, req.Page, req.PageSize, req.Keywords, req.Status)
	if err != nil {
		return nil, 0, nil, nil, ErrSystem(err)
	}

	infos := make([]*v1.APIKeyInfo, 0, len(items))
	for _, item := range items {
		infos = append(infos, n.toAPIKeyInfo(item, false, userID))
	}
	return infos, total, counts, owners, nil
}

func (n *NodeUsecase) CopyAPIKey(ctx context.Context, userID string, req *v1.CopyAPIKeyRequest) (*v1.APIKeyInfo, error) {
	item, err := n.repo.GetUserAPIKey(ctx, userID, req.Id)
	if err != nil {
		if !gen.IsNotFound(err) {
			return nil, ErrSystem(err)
		}
		return nil, ErrAPIKeyNotFound
	}
	return n.toAPIKeyInfo(item, true, userID), nil
}

func (n *NodeUsecase) UpdateAPIKey(ctx context.Context, userID string, req *v1.UpdateAPIKeyRequest) (*v1.APIKeyInfo, error) {
	if req.Name != nil && *req.Name == "" {
		return nil, ErrAPIKeyNameEmpty
	}
	var expiration *time.Time
	if req.ExpiresAt != nil {
		expiration = new(req.ExpiresAt.AsTime())
	}
	var remark *string
	if req.Remark != nil {
		remark = new(strings.TrimSpace(*req.Remark))
	}
	var permissions *[]string
	if req.Permissions != nil {
		normalized, err := normalizeNodePermissions(req.Permissions.Items)
		if err != nil {
			return nil, err
		}
		permissions = &normalized
	}
	// 管理全部 API Key 时可以吊销 / 延期他人的密钥
	scope := userID
	if n.canManageAllAPIKeys(ctx) {
		scope = ""
	}
	item, err := n.repo.UpdateUserAPIKey(
		ctx,
		scope,
		req.Id,
		req.Name,
		remark,
		expiration,
		req.NeverExpires,
		permissions,
	)
	if err != nil {
		if !gen.IsNotFound(err) {
			return nil, ErrSystem(err)
		}
		return nil, ErrAPIKeyNotFound
	}
	return n.toAPIKeyInfo(item, false, userID), nil
}

func (n *NodeUsecase) RefreshAPIKey(ctx context.Context, userID string, req *v1.RefreshAPIKeyRequest) (*v1.APIKeyInfo, error) {
	for range apiKeyCreateRetry {
		apiKey, err := utils.GenerateAPIKey()
		if err != nil {
			return nil, ErrSystem(err)
		}

		item, err := n.repo.RefreshUserAPIKey(ctx, userID, req.Id, apiKey)
		if err == nil {
			return n.toAPIKeyInfo(item, true, userID), nil
		}
		if gen.IsNotFound(err) {
			return nil, ErrAPIKeyNotFound
		}
		if !gen.IsConstraintError(err) {
			return nil, ErrSystem(err)
		}
	}

	return nil, ErrAPIKeyGenerate
}

func (n *NodeUsecase) DeleteAPIKey(ctx context.Context, userID string, req *v1.DeleteAPIKeyRequest) error {
	scope := userID
	if n.canManageAllAPIKeys(ctx) {
		scope = ""
	}
	if err := n.repo.DeleteUserAPIKey(ctx, scope, req.Id); err != nil {
		if gen.IsNotFound(err) {
			return ErrAPIKeyNotFound
		}
		return ErrSystem(err)
	}
	return nil
}

func toAPIKeyOwner(u *gen.User) *v1.APIKeyOwner {
	if u == nil {
		return nil
	}
	return &v1.APIKeyOwner{UserId: u.ID, Name: u.Name, Username: u.Username, Avatar: u.Avatar}
}

// toAPIKeyInfo callerID 为当前用户，用于标记是否自己的密钥。
func (n *NodeUsecase) toAPIKeyInfo(item *gen.UserAPIKey, includeSecret bool, callerID string) *v1.APIKeyInfo {
	info := &v1.APIKeyInfo{
		Owner:       toAPIKeyOwner(item.Edges.User),
		Mine:        item.UserID == callerID,
		Id:          item.ID,
		Name:        item.Name,
		CreateTime:  timestamppb.New(item.CreateTime),
		UpdateTime:  timestamppb.New(item.UpdateTime),
		Permissions: item.Permissions,
		KeyId:       item.KeyID,
		Remark:      item.Remark,
		LastUsedIp:  item.LastUsedIP,
		Status:      APIKeyStatusOf(item.ExpiresAt, time.Now()),
	}
	if item.LastUsedAt != nil {
		info.LastUsedTime = timestamppb.New(*item.LastUsedAt)
	}
	if includeSecret {
		info.ApiKey = item.APIKey
	} else {
		info.ApiKey = utils.PrivacyString(item.APIKey)
	}
	if item.ExpiresAt != nil {
		info.ExpiresAt = timestamppb.New(*item.ExpiresAt)
	}
	return info
}

// APIKeyStatusOf 按过期时间判断 API Key 的有效状态。
func APIKeyStatusOf(expiresAt *time.Time, now time.Time) v1.APIKeyStatus {
	switch {
	case expiresAt == nil:
		return v1.APIKeyStatus_APIKeyStatus_Valid
	case !now.Before(*expiresAt):
		return v1.APIKeyStatus_APIKeyStatus_Expired
	case expiresAt.Sub(now) <= APIKeyExpiringWindow:
		return v1.APIKeyStatus_APIKeyStatus_Expiring
	default:
		return v1.APIKeyStatus_APIKeyStatus_Valid
	}
}
