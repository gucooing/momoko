package biz

import (
	"context"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/utils"
)

const (
	apiKeyCreateRetry = 5
)

type APIKeyRepo interface {
	CreateUserAPIKey(ctx context.Context, userID, name, apiKey string, expiresAt *time.Time, permissions []string) (*gen.UserAPIKey, error)
	GetUserAPIKey(ctx context.Context, userID, id string) (*gen.UserAPIKey, error)
	GetAPIKeyByKeyID(ctx context.Context, keyID string) (*gen.UserAPIKey, error)
	ListUserAPIKeys(ctx context.Context, userID string, page, pageSize int64, keywords *string) ([]*gen.UserAPIKey, int64, error)
	UpdateUserAPIKey(ctx context.Context, userID, id string, name *string, expiresAt *time.Time, clearExpiresAt *bool, permissions *[]string) (*gen.UserAPIKey, error)
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

		item, err := n.repo.CreateUserAPIKey(ctx, userID, req.Name, apiKey, expiration, permissions)
		if err == nil {
			return n.toAPIKeyInfo(item, true), nil
		}
		if !gen.IsConstraintError(err) {
			return nil, ErrSystem(err)
		}
	}

	return nil, ErrAPIKeyGenerate
}

func (n *NodeUsecase) ListAPIKeys(ctx context.Context, userID string, req *v1.ListAPIKeysRequest) ([]*v1.APIKeyInfo, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	items, total, err := n.repo.ListUserAPIKeys(ctx, userID, req.Page, req.PageSize, req.Keywords)
	if err != nil {
		return nil, 0, ErrSystem(err)
	}

	infos := make([]*v1.APIKeyInfo, 0, len(items))
	for _, item := range items {
		infos = append(infos, n.toAPIKeyInfo(item, false))
	}
	return infos, total, nil
}

func (n *NodeUsecase) CopyAPIKey(ctx context.Context, userID string, req *v1.CopyAPIKeyRequest) (*v1.APIKeyInfo, error) {
	item, err := n.repo.GetUserAPIKey(ctx, userID, req.Id)
	if err != nil {
		if !gen.IsNotFound(err) {
			return nil, ErrSystem(err)
		}
		return nil, ErrAPIKeyNotFound
	}
	return n.toAPIKeyInfo(item, true), nil
}

func (n *NodeUsecase) UpdateAPIKey(ctx context.Context, userID string, req *v1.UpdateAPIKeyRequest) (*v1.APIKeyInfo, error) {
	if req.Name != nil && *req.Name == "" {
		return nil, ErrAPIKeyNameEmpty
	}
	var expiration *time.Time
	if req.ExpiresAt != nil {
		expiration = new(req.ExpiresAt.AsTime())
	}
	var permissions *[]string
	if req.Permissions != nil {
		normalized, err := normalizeNodePermissions(req.Permissions.Items)
		if err != nil {
			return nil, err
		}
		permissions = &normalized
	}
	item, err := n.repo.UpdateUserAPIKey(
		ctx,
		userID,
		req.Id,
		req.Name,
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
	return n.toAPIKeyInfo(item, false), nil
}

func (n *NodeUsecase) RefreshAPIKey(ctx context.Context, userID string, req *v1.RefreshAPIKeyRequest) (*v1.APIKeyInfo, error) {
	for range apiKeyCreateRetry {
		apiKey, err := utils.GenerateAPIKey()
		if err != nil {
			return nil, ErrSystem(err)
		}

		item, err := n.repo.RefreshUserAPIKey(ctx, userID, req.Id, apiKey)
		if err == nil {
			return n.toAPIKeyInfo(item, true), nil
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
	if err := n.repo.DeleteUserAPIKey(ctx, userID, req.Id); err != nil {
		if gen.IsNotFound(err) {
			return ErrAPIKeyNotFound
		}
		return ErrSystem(err)
	}
	return nil
}

func (n *NodeUsecase) toAPIKeyInfo(item *gen.UserAPIKey, includeSecret bool) *v1.APIKeyInfo {
	info := &v1.APIKeyInfo{
		Id:          item.ID,
		Name:        item.Name,
		CreateTime:  timestamppb.New(item.CreateTime),
		UpdateTime:  timestamppb.New(item.UpdateTime),
		Permissions: item.Permissions,
		KeyId:       item.KeyID,
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
