package biz

import (
	"context"
	"strings"
	"sync"

	"golang.org/x/net/websocket"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/sshhost"
	"momoko/pkg/sshcore"
)

const OpenSSHWSPath = "/api/v1/openssh/ws"

const batchTestSSHHostLimit = 5

type OpenSSHRepo interface {
	GetSSHHosts(ctx context.Context, page, pageSize int64, userID string, keywords, host *string) ([]*gen.SSHHost, int64, error)
	GetSSHHostByUserID(ctx context.Context, userID, hostID string) (*gen.SSHHost, error)
	GetSSHHostByOwnerID(ctx context.Context, ownerID, hostID string) (*gen.SSHHost, error)
	CreateSSHHost(ctx context.Context, req *v1.CreateSSHHostRequest, ownerID string, sharedUserIDs []string) (*gen.SSHHost, error)
	UpdateSSHHost(ctx context.Context, req *v1.UpdateSSHHostRequest, ownerID string) (*gen.SSHHost, error)
	DeleteSSHHost(ctx context.Context, ownerID, hostID string) error
	ShareSSHHost(ctx context.Context, ownerID, hostID string, userIDs []string) (*gen.SSHHost, error)
	UpdateSSHHostStatus(ctx context.Context, userID, hostID string, status v1.SSHHostStatus, fingerprint string) error
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
	item, err := o.repo.CreateSSHHost(ctx, req, userID, uniqueNonEmpty(req.SharedUserIds, userID))
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
	item, err := o.repo.ShareSSHHost(ctx, userID, req.Id, uniqueNonEmpty(req.UserIds, userID))
	if err != nil {
		return nil, o.wrapSSHRepoErr(err)
	}
	return o.toSSHHostInfo(item, userID), nil
}

func (o *OpenSSHUsecase) TestSSHHost(ctx context.Context, userID, hostID string) (*v1.TestSSHHostResponse, error) {
	result, err := o.testSSHHost(ctx, userID, hostID)
	if err != nil {
		return nil, err
	}
	return &v1.TestSSHHostResponse{
		Status:      result.Status,
		Message:     result.Message,
		Fingerprint: result.Fingerprint,
	}, nil
}

func (o *OpenSSHUsecase) BatchTestSSHHosts(ctx context.Context, userID string, ids []string) (*v1.BatchTestSSHHostsResponse, error) {
	ids = uniqueNonEmpty(ids, "")
	results := make([]*v1.SSHHostTestResult, len(ids))

	jobs := make(chan int)
	var wg sync.WaitGroup
	workerCount := min(len(ids), batchTestSSHHostLimit)
	for range workerCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := range jobs {
				result, err := o.testSSHHost(ctx, userID, ids[i])
				if err != nil {
					results[i] = &v1.SSHHostTestResult{
						Id:      ids[i],
						Status:  v1.SSHHostStatus_SSH_HOST_STATUS_OFFLINE,
						Message: err.Error(),
					}
					continue
				}
				results[i] = result
			}
		}()
	}

	for i := range ids {
		select {
		case <-ctx.Done():
			close(jobs)
			wg.Wait()
			return nil, ctx.Err()
		case jobs <- i:
		}
	}
	close(jobs)
	wg.Wait()

	return &v1.BatchTestSSHHostsResponse{Results: results}, nil
}

func (o *OpenSSHUsecase) testSSHHost(ctx context.Context, userID, hostID string) (*v1.SSHHostTestResult, error) {
	cfg, err := o.repo.GetSSHHostConfigByUserID(ctx, userID, hostID)
	if err != nil {
		return nil, o.wrapSSHRepoErr(err)
	}
	result, err := sshcore.Test(ctx, cfg)
	if err != nil {
		status := v1.SSHHostStatus_SSH_HOST_STATUS_OFFLINE
		if updateErr := o.repo.UpdateSSHHostStatus(ctx, userID, hostID, status, ""); updateErr != nil {
			return nil, o.wrapSSHRepoErr(updateErr)
		}
		return &v1.SSHHostTestResult{
			Id:      hostID,
			Status:  v1.SSHHostStatus_SSH_HOST_STATUS_OFFLINE,
			Message: err.Error(),
		}, nil
	}
	status := v1.SSHHostStatus_SSH_HOST_STATUS_ONLINE
	if err := o.repo.UpdateSSHHostStatus(ctx, userID, hostID, status, result.Fingerprint); err != nil {
		return nil, o.wrapSSHRepoErr(err)
	}
	return &v1.SSHHostTestResult{
		Id:          hostID,
		Status:      status,
		Message:     "ok",
		Fingerprint: result.Fingerprint,
	}, nil
}

func (o *OpenSSHUsecase) StartSSHWebSocket(conn *websocket.Conn, userID, hostID string) error {
	cfg, err := o.repo.GetSSHHostConfigByUserID(conn.Request().Context(), userID, hostID)
	if err != nil {
		_ = websocket.Message.Send(conn, err.Error())
		return o.wrapSSHRepoErr(err)
	}
	return sshcore.ServeWebSocket(conn, cfg)
}

func validateCreateSSHHost(req *v1.CreateSSHHostRequest) error {
	req.Host = strings.TrimSpace(req.Host)
	req.Username = strings.TrimSpace(req.Username)
	req.Name = strings.TrimSpace(req.Name)
	req.Fingerprint = strings.TrimSpace(req.Fingerprint)
	if req.Name == "" || req.Host == "" || req.Username == "" {
		return ErrSSHHostInvalid
	}
	if req.Port == 0 {
		req.Port = 22
	}
	if req.Port < 1 || req.Port > 65535 {
		return ErrSSHHostInvalid
	}

	return validateCreateCredential(req.AuthType, req.Password, req.PrivateKey)
}

func validateCreateCredential(authType v1.SSHAuthType, password, privateKey string) error {
	switch authType {
	case v1.SSHAuthType_SSH_AUTH_TYPE_PASSWORD:
		if strings.TrimSpace(password) == "" {
			return ErrSSHCredentialInvalid
		}
		return nil
	case v1.SSHAuthType_SSH_AUTH_TYPE_KEY:
		if strings.TrimSpace(privateKey) == "" {
			return ErrSSHCredentialInvalid
		}
		return nil
	default:
		return ErrSSHAuthInvalid
	}
}

func toV1SSHAuthType(authType string) v1.SSHAuthType {
	switch authType {
	case sshcore.AuthPassword:
		return v1.SSHAuthType_SSH_AUTH_TYPE_PASSWORD
	case sshcore.AuthKey:
		return v1.SSHAuthType_SSH_AUTH_TYPE_KEY
	default:
		return v1.SSHAuthType_SSH_AUTH_TYPE_UNSPECIFIED
	}
}

func toV1SSHHostStatus(status sshhost.Status) v1.SSHHostStatus {
	switch status {
	case sshhost.StatusOnline:
		return v1.SSHHostStatus_SSH_HOST_STATUS_ONLINE
	case sshhost.StatusOffline:
		return v1.SSHHostStatus_SSH_HOST_STATUS_OFFLINE
	default:
		return v1.SSHHostStatus_SSH_HOST_STATUS_UNKNOWN
	}
}

func (o *OpenSSHUsecase) toSSHHostInfo(item *gen.SSHHost, userID string) *v1.SSHHostInfo {
	info := &v1.SSHHostInfo{
		Id:          item.ID,
		Name:        item.Name,
		Host:        item.Host,
		Port:        int32(item.Port),
		Username:    item.Username,
		AuthType:    toV1SSHAuthType(string(item.AuthType)),
		Fingerprint: item.Fingerprint,
		Remark:      item.Remark,
		Tags:        item.Tags,
		Status:      toV1SSHHostStatus(item.Status),
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

func uniqueNonEmpty(ids []string, exclude string) []string {
	seen := make(map[string]struct{}, len(ids))
	result := make([]string, 0, len(ids))
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || id == exclude {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
