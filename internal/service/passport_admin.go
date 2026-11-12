package service

import (
	"context"
	"strings"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/conf"
	"momoko/pkg/auth"
)

// PassportAdminService 用户通行证的后台管理（主 server，权限 game:passport）：解析请求，调用 PassportAdminUsecase；
// 与玩家端的 PassportService 互不调用。
type PassportAdminService struct {
	v1.UnimplementedPassportAdminServiceServer

	uc   *biz.PassportAdminUsecase
	conf *conf.Passport
}

func NewPassportAdminService(uc *biz.PassportAdminUsecase, c *conf.Passport) *PassportAdminService {
	return &PassportAdminService{uc: uc, conf: c}
}

func (s *PassportAdminService) admin(ctx context.Context) (string, string, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return "", "", biz.ErrTokenInvalid
	}
	return a.UserID, s.uc.AdminName(ctx, a.UserID), nil
}

func (s *PassportAdminService) GetPassportStats(ctx context.Context, _ *v1.GetPassportStatsRequest) (*v1.GetPassportStatsResponse, error) {
	return s.uc.AdminStats(ctx)
}

func (s *PassportAdminService) ListPassportUsers(ctx context.Context, req *v1.ListPassportUsersRequest) (*v1.ListPassportUsersResponse, error) {
	return s.uc.AdminListUsers(ctx, req)
}

func (s *PassportAdminService) GetPassportUser(ctx context.Context, req *v1.GetPassportUserRequest) (*v1.GetPassportUserResponse, error) {
	return s.uc.AdminGetUser(ctx, req.Id)
}

func (s *PassportAdminService) SetPassportUserStatus(ctx context.Context, req *v1.SetPassportUserStatusRequest) (*v1.GetPassportUserResponse, error) {
	_, name, err := s.admin(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.AdminSetStatus(ctx, name, req.Id, req.Disabled)
}

func (s *PassportAdminService) AdminUnbindPassportLink(ctx context.Context, req *v1.AdminUnbindPassportLinkRequest) (*v1.GetPassportUserResponse, error) {
	_, name, err := s.admin(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.AdminUnbind(ctx, name, req.UserId, req.LinkId)
}

func (s *PassportAdminService) AdminBindPassportLink(ctx context.Context, req *v1.AdminBindPassportLinkRequest) (*v1.GetPassportUserResponse, error) {
	_, name, err := s.admin(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.AdminBind(ctx, name, req)
}

func (s *PassportAdminService) SearchPassportServers(ctx context.Context, req *v1.SearchGameServersRequest) (*v1.SearchGameServersResponse, error) {
	id, _, err := s.admin(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.AdminSearchServers(ctx, id, req)
}

func (s *PassportAdminService) SearchPassportGamePlayers(ctx context.Context, req *v1.SearchPassportGamePlayersRequest) (*v1.SearchPassportGamePlayersResponse, error) {
	return s.uc.AdminSearchPlayers(ctx, req.ServerId, req.Q)
}

func (s *PassportAdminService) ListPassportInvites(ctx context.Context, req *v1.ListPassportInvitesRequest) (*v1.ListPassportInvitesResponse, error) {
	return s.uc.AdminInvites(ctx, req)
}

func (s *PassportAdminService) CreatePassportInvite(ctx context.Context, req *v1.CreatePassportInviteRequest) (*v1.PassportInvite, error) {
	id, name, err := s.admin(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.AdminCreateInvite(ctx, id, name, req)
}

func (s *PassportAdminService) RevokePassportInvite(ctx context.Context, req *v1.RevokePassportInviteRequest) (*v1.PassportInvite, error) {
	return s.uc.AdminRevokeInvite(ctx, req.Id)
}

func (s *PassportAdminService) serverInfo(ctx context.Context) biz.PassportServerInfo {
	info := biz.PassportServerInfo{Enabled: passportEnabled(s.conf), Addr: "0.0.0.0:22634"}
	if s.conf != nil {
		if s.conf.Addr != "" {
			info.Addr = s.conf.Addr
		}
		info.PublicURL = strings.TrimRight(strings.TrimSpace(s.conf.PublicUrl), "/")
	}
	info.Issuer = info.PublicURL
	return info
}

// passportEnabled 通行证 server 是否启用（不写时默认启用）。
func passportEnabled(c *conf.Passport) bool {
	return c == nil || c.Enabled == nil || *c.Enabled
}

func (s *PassportAdminService) GetPassportSettings(ctx context.Context, _ *v1.GetPassportSettingsRequest) (*v1.GetPassportSettingsResponse, error) {
	return s.uc.AdminSettings(ctx, s.serverInfo(ctx))
}

func (s *PassportAdminService) UpdatePassportSettings(ctx context.Context, req *v1.PassportSettings) (*v1.GetPassportSettingsResponse, error) {
	if _, err := s.uc.SaveSettings(ctx, req); err != nil {
		return nil, err
	}
	return s.uc.AdminSettings(ctx, s.serverInfo(ctx))
}

func (s *PassportAdminService) ListPassportOidcClients(ctx context.Context, _ *v1.ListPassportOidcClientsRequest) (*v1.ListPassportOidcClientsResponse, error) {
	return s.uc.ListOidcClients(ctx)
}

func (s *PassportAdminService) CreatePassportOidcClient(ctx context.Context, req *v1.SavePassportOidcClientRequest) (*v1.PassportOidcClient, error) {
	_, name, err := s.admin(ctx)
	if err != nil {
		return nil, err
	}
	req.Id = ""
	return s.uc.SaveOidcClient(ctx, name, req)
}

func (s *PassportAdminService) UpdatePassportOidcClient(ctx context.Context, req *v1.SavePassportOidcClientRequest) (*v1.PassportOidcClient, error) {
	if req.Id == "" {
		return nil, biz.ErrPassportOidcClientNotFound
	}
	return s.uc.SaveOidcClient(ctx, "", req)
}

func (s *PassportAdminService) DeletePassportOidcClient(ctx context.Context, req *v1.DeletePassportOidcClientRequest) (*v1.DeletePassportOidcClientResponse, error) {
	if err := s.uc.DeleteOidcClient(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeletePassportOidcClientResponse{}, nil
}

func (s *PassportAdminService) RefreshPassportOidcClientSecret(ctx context.Context, req *v1.DeletePassportOidcClientRequest) (*v1.PassportOidcClient, error) {
	return s.uc.RefreshOidcClientSecret(ctx, req.Id)
}
