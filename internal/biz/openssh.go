package biz

import (
	"context"
	"momoko/pkg/utils"
	"strings"

	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/sshhost"
	"momoko/pkg/sshcore"
)

const OpenSSHWSPath = "/api/v1/openssh/ws"

type OpenSSHRepo interface {
	GetSSHHosts(ctx context.Context, page, pageSize int64, userID string, keywords, host *string) ([]*gen.SSHHost, int64, error)
	GetSSHHostByUserID(ctx context.Context, userID, hostID string) (*gen.SSHHost, error)
	GetSSHHostByOwnerID(ctx context.Context, ownerID, hostID string) (*gen.SSHHost, error)
	CreateSSHHost(ctx context.Context, req *v1.CreateSSHHostRequest, ownerID string, sharedUserIDs []string) (*gen.SSHHost, error)
	UpdateSSHHost(ctx context.Context, req *v1.UpdateSSHHostRequest, ownerID string) (*gen.SSHHost, error)
	DeleteSSHHost(ctx context.Context, ownerID, hostID string) error
	ShareSSHHost(ctx context.Context, ownerID, hostID string, userIDs []string) (*gen.SSHHost, error)
	GetSSHHostConfigByUserID(ctx context.Context, userID, hostID string) (sshcore.Config, error)
}

type OpenSSHUsecase struct {
	repo OpenSSHRepo
}

func NewOpenSSHUsecase(repo OpenSSHRepo) *OpenSSHUsecase {
	return &OpenSSHUsecase{repo: repo}
}

func (o *OpenSSHUsecase) GetSSHHosts(ctx context.Context, req *v1.GetSSHHostsRequest, userID string) ([]*v1.SSHHostInfo, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 50 {
		req.PageSize = 50
	}

	items, total, err := o.repo.GetSSHHosts(ctx, req.Page, req.PageSize, userID, req.Keywords, req.Host)
	if err != nil {
		return nil, 0, ErrSystem(err)
	}

	infos := make([]*v1.SSHHostInfo, 0, len(items))
	for _, item := range items {
		infos = append(infos, o.toSSHHostInfo(item, userID))
	}
	return infos, total, nil
}

func (o *OpenSSHUsecase) CreateSSHHost(ctx context.Context, req *v1.CreateSSHHostRequest, userID string) (*v1.SSHHostInfo, error) {
	if err := validateCreateSSHHost(req); err != nil {
		return nil, err
	}
	item, err := o.repo.CreateSSHHost(ctx, req, userID, utils.UniqueNonEmpty(req.SharedUserIds, userID))
	if err != nil {
		return nil, ErrSystem(err)
	}
	return o.toSSHHostInfo(item, userID), nil
}

func (o *OpenSSHUsecase) GetSSHHostByUserID(ctx context.Context, userID, hostID string) (*v1.SSHHostInfo, error) {
	item, err := o.repo.GetSSHHostByUserID(ctx, userID, hostID)
	if err != nil {
		return nil, o.wrapSSHRepoErr(err)
	}
	return o.toSSHHostInfo(item, userID), nil
}

func (o *OpenSSHUsecase) UpdateSSHHost(ctx context.Context, req *v1.UpdateSSHHostRequest, userID string) (*v1.SSHHostInfo, error) {
	item, err := o.repo.UpdateSSHHost(ctx, req, userID)
	if err != nil {
		return nil, o.wrapSSHRepoErr(err)
	}
	return o.toSSHHostInfo(item, userID), nil
}

func (o *OpenSSHUsecase) DeleteSSHHost(ctx context.Context, userID, hostID string) error {
	if err := o.repo.DeleteSSHHost(ctx, userID, hostID); err != nil {
		return o.wrapSSHRepoErr(err)
	}
	return nil
}

func (o *OpenSSHUsecase) ShareSSHHost(ctx context.Context, req *v1.ShareSSHHostRequest, userID string) (*v1.SSHHostInfo, error) {
	item, err := o.repo.ShareSSHHost(ctx, userID, req.Id, utils.UniqueNonEmpty(req.UserIds, userID))
	if err != nil {
		return nil, o.wrapSSHRepoErr(err)
	}
	return o.toSSHHostInfo(item, userID), nil
}

// TestSSHHost 用请求体草稿测连通性，不写库。
// - 有 id：以库中配置为底，请求体字段覆盖；空凭据沿用库中凭据
// - 无 id：纯草稿，必须带齐 host/username/凭据
func (o *OpenSSHUsecase) TestSSHHost(ctx context.Context, userID string, req *v1.TestSSHHostRequest) (*v1.TestSSHHostResponse, error) {
	cfg, err := o.resolveTestConfig(ctx, userID, req)
	if err != nil {
		return nil, err
	}
	result, err := sshcore.Test(ctx, cfg)
	if err != nil {
		return &v1.TestSSHHostResponse{
			Ok:      false,
			Message: err.Error(),
		}, nil
	}
	return &v1.TestSSHHostResponse{
		Ok:          true,
		Message:     "ok",
		Fingerprint: result.Fingerprint,
	}, nil
}

func (o *OpenSSHUsecase) resolveTestConfig(ctx context.Context, userID string, req *v1.TestSSHHostRequest) (sshcore.Config, error) {
	var base sshcore.Config
	// 请求是否自带可用明文凭据（编辑页重填密码/密钥时优先走明文，不再依赖库中密文）
	hasInlineCred := (req.Password != nil && strings.TrimSpace(*req.Password) != "") ||
		(req.PrivateKey != nil && strings.TrimSpace(*req.PrivateKey) != "")

	if req.Id != nil && strings.TrimSpace(*req.Id) != "" {
		if hasInlineCred {
			// 有明文凭据时，库中只取 host/user/auth 等元数据意义不大；直接跳过解密，避免旧密文坏钥拖垮测试。
			// 仍允许下面用请求体字段组装完整草稿。
		} else {
			cfg, err := o.repo.GetSSHHostConfigByUserID(ctx, userID, strings.TrimSpace(*req.Id))
			if err != nil {
				// 编辑页空凭据 + 库中密文无法解密：给可操作提示，而不是笼统 500 cipher 错误
				if strings.Contains(err.Error(), "message authentication failed") ||
					strings.Contains(err.Error(), "decrypt ssh") {
					return sshcore.Config{}, ErrSSHStoredCredentialBroken
				}
				return sshcore.Config{}, o.wrapSSHRepoErr(err)
			}
			base = cfg
		}
	}

	if req.Host != nil {
		base.Host = strings.TrimSpace(*req.Host)
	}
	if req.Port != nil {
		base.Port = int(*req.Port)
	}
	if req.Username != nil {
		base.Username = strings.TrimSpace(*req.Username)
	}
	if req.Fingerprint != nil {
		base.Fingerprint = strings.TrimSpace(*req.Fingerprint)
	}
	if req.AuthType != nil {
		switch *req.AuthType {
		case v1.SSHAuthType_SSH_AUTH_TYPE_PASSWORD:
			base.AuthType = sshcore.AuthPassword
		case v1.SSHAuthType_SSH_AUTH_TYPE_KEY:
			base.AuthType = sshcore.AuthKey
		default:
			return sshcore.Config{}, ErrSSHAuthInvalid
		}
	}
	// 凭据：仅当请求显式传了非空值才覆盖；编辑弹窗空凭据沿用 base（已存）
	if req.Password != nil && strings.TrimSpace(*req.Password) != "" {
		base.AuthType = sshcore.AuthPassword
		base.Credential = *req.Password
	}
	if req.PrivateKey != nil && strings.TrimSpace(*req.PrivateKey) != "" {
		base.AuthType = sshcore.AuthKey
		base.Credential = *req.PrivateKey
	}
	if req.Passphrase != nil {
		base.Passphrase = *req.Passphrase
	}

	if base.Port == 0 {
		base.Port = 22
	}
	if base.Host == "" || base.Username == "" {
		return sshcore.Config{}, ErrSSHHostInvalid
	}
	if base.AuthType == "" {
		return sshcore.Config{}, ErrSSHAuthInvalid
	}
	if strings.TrimSpace(base.Credential) == "" {
		return sshcore.Config{}, ErrSSHCredentialInvalid
	}
	return base, nil
}

func (o *OpenSSHUsecase) StartSSHWebSocket(ctx context.Context, conn *websocket.Conn, userID, hostID string) error {
	cfg, err := o.repo.GetSSHHostConfigByUserID(ctx, userID, hostID)
	if err != nil {
		_ = websocket.Message.Send(conn, err.Error())
		return o.wrapSSHRepoErr(err)
	}
	return sshcore.ServeWebSocket(conn, cfg)
}

func validateCreateSSHHost(req *v1.CreateSSHHostRequest) error {
	if req.Name == "" || req.Host == "" || req.Username == "" {
		return ErrSSHHostInvalid
	}
	if req.Port == 0 {
		req.Port = 22
	}
	if req.Port < 1 || req.Port > 65535 {
		return ErrSSHHostInvalid
	}
	switch req.AuthType {
	case v1.SSHAuthType_SSH_AUTH_TYPE_PASSWORD:
		if req.Password == "" {
			return ErrSSHCredentialInvalid
		}
		return nil
	case v1.SSHAuthType_SSH_AUTH_TYPE_KEY:
		if req.PrivateKey == "" {
			return ErrSSHCredentialInvalid
		}
		return nil
	default:
		return ErrSSHAuthInvalid
	}
}

func toV1SSHAuthType(authType sshhost.AuthType) v1.SSHAuthType {
	switch authType {
	case sshcore.AuthPassword:
		return v1.SSHAuthType_SSH_AUTH_TYPE_PASSWORD
	case sshcore.AuthKey:
		return v1.SSHAuthType_SSH_AUTH_TYPE_KEY
	default:
		return v1.SSHAuthType_SSH_AUTH_TYPE_UNSPECIFIED
	}
}

func (o *OpenSSHUsecase) toSSHHostInfo(item *gen.SSHHost, userID string) *v1.SSHHostInfo {
	info := &v1.SSHHostInfo{
		Id:          item.ID,
		Name:        item.Name,
		Host:        item.Host,
		Port:        int32(item.Port),
		Username:    item.Username,
		AuthType:    toV1SSHAuthType(item.AuthType),
		Fingerprint: item.Fingerprint,
		Remark:      item.Remark,
		Tags:        item.Tags,
		WsPath:      OpenSSHWSPath,
		CreateTime:  timestamppb.New(item.CreateTime),
		UpdateTime:  timestamppb.New(item.UpdateTime),
	}
	if owner := item.Edges.Owner; owner != nil {
		info.OwnerUserId = owner.ID
		if owner.ID == userID {
			info.AccessRole = v1.SSHHostAccessRole_SSH_HOST_ACCESS_ROLE_OWNER
		}
	}
	if info.AccessRole == v1.SSHHostAccessRole_SSH_HOST_ACCESS_ROLE_UNSPECIFIED {
		info.AccessRole = v1.SSHHostAccessRole_SSH_HOST_ACCESS_ROLE_SHARED
	}
	for _, sharedUser := range item.Edges.SharedUsers {
		info.SharedUsers = append(info.SharedUsers, &v1.SSHHostSharedUser{
			UserId: sharedUser.ID,
			Name:   sharedUser.Name,
		})
	}
	return info
}

func (o *OpenSSHUsecase) wrapSSHRepoErr(err error) error {
	if gen.IsNotFound(err) {
		return ErrSSHHostAccess
	}
	return ErrSystem(err)
}
