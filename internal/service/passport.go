package service

import (
	"context"
	"errors"
	"net/http"
	"strings"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/conf"
	"momoko/pkg/oidcserver"
	"momoko/pkg/passporttoken"
)

// passportIssuer 通行证 OIDC 的 issuer：配置文件的 passport.public_url，空时按请求推断。
func passportIssuer(c *conf.Passport, r *http.Request) string {
	if c != nil && strings.TrimSpace(c.PublicUrl) != "" {
		return strings.TrimRight(strings.TrimSpace(c.PublicUrl), "/")
	}
	return oidcserver.RequestOrigin(r)
}

// PassportService 用户通行证玩家端（只注册在 passport server 上）。
type PassportService struct {
	v1.UnimplementedPassportServiceServer

	uc      *biz.PassportUsecase
	gakumas *biz.GakumasPassportUsecase
	conf    *conf.Passport
}

func NewPassportService(uc *biz.PassportUsecase, gakumas *biz.GakumasPassportUsecase, c *conf.Passport) *PassportService {
	return &PassportService{
		uc:      uc,
		gakumas: gakumas,
		conf:    c,
	}
}

// Usecase 通行证用例（passport server 的鉴权过滤器用）。
func (s *PassportService) Usecase() *biz.PassportUsecase { return s.uc }

// me 当前登录的通行证身份（鉴权过滤器放进上下文）。
func me(ctx context.Context) (*passporttoken.Claims, error) {
	c, ok := passporttoken.FromContext(ctx)
	if !ok {
		return nil, biz.ErrPassportUnauthorized
	}
	return c, nil
}

func (s *PassportService) issuer(ctx context.Context) string {
	r, _ := khttp.RequestFromServerContext(ctx)
	return passportIssuer(s.conf, r)
}

// ---------------- 公开 ----------------

func (s *PassportService) GetPassportConfig(ctx context.Context, _ *v1.GetPassportConfigRequest) (*v1.GetPassportConfigResponse, error) {
	return s.uc.PublicConfig(ctx), nil
}

func (s *PassportService) SendPassportRegisterCode(ctx context.Context, req *v1.SendPassportEmailCodeRequest) (*v1.SendPassportEmailCodeResponse, error) {
	return s.uc.SendRegisterCode(ctx, req.Email)
}

func (s *PassportService) PassportRegister(ctx context.Context, req *v1.PassportRegisterRequest) (*v1.PassportLoginResponse, error) {
	return s.uc.Register(ctx, req)
}

func (s *PassportService) SendPassportLoginCode(ctx context.Context, req *v1.SendPassportEmailCodeRequest) (*v1.SendPassportEmailCodeResponse, error) {
	return s.uc.SendLoginCode(ctx, req.Email)
}

func (s *PassportService) PassportLogin(ctx context.Context, req *v1.PassportLoginRequest) (*v1.PassportLoginResponse, error) {
	return s.uc.Login(ctx, req)
}

func (s *PassportService) PassportLoginMfa(ctx context.Context, req *v1.PassportLoginMfaRequest) (*v1.PassportLoginResponse, error) {
	return s.uc.LoginMfa(ctx, req)
}

func (s *PassportService) PassportRefresh(ctx context.Context, req *v1.PassportRefreshRequest) (*v1.PassportLoginResponse, error) {
	return s.uc.Refresh(ctx, req.RefreshToken)
}

func (s *PassportService) SendPassportResetCode(ctx context.Context, req *v1.SendPassportEmailCodeRequest) (*v1.SendPassportEmailCodeResponse, error) {
	return s.uc.SendResetCode(ctx, req.Email)
}

func (s *PassportService) PassportResetPassword(ctx context.Context, req *v1.PassportResetPasswordRequest) (*v1.PassportEmpty, error) {
	if err := s.uc.ResetPassword(ctx, req); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

// ---------------- 账号 ----------------

func (s *PassportService) PassportLogout(ctx context.Context, _ *v1.PassportEmpty) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.Logout(ctx, c); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) GetPassportMe(ctx context.Context, _ *v1.PassportEmpty) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Me(ctx, c.UserID)
}

func (s *PassportService) UpdatePassportProfile(ctx context.Context, req *v1.UpdatePassportProfileRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.UpdateProfile(ctx, c.UserID, req)
}

func (s *PassportService) PresignPassportAvatar(ctx context.Context, req *v1.PresignPassportAvatarRequest) (*v1.PresignPassportAvatarResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.PresignAvatar(ctx, c.UserID, req.ContentType, req.Size)
}

func (s *PassportService) CompletePassportAvatar(ctx context.Context, req *v1.CompletePassportAvatarRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.CompleteAvatar(ctx, c.UserID, req.Token)
}

func (s *PassportService) ChangePassportPassword(ctx context.Context, req *v1.ChangePassportPasswordRequest) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.ChangePassword(ctx, c, req.OldPassword, req.NewPassword); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) SendPassportEmailChangeCode(ctx context.Context, req *v1.SendPassportEmailCodeRequest) (*v1.SendPassportEmailCodeResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.SendEmailChangeCode(ctx, c.UserID, req.Email)
}

func (s *PassportService) ChangePassportEmail(ctx context.Context, req *v1.ChangePassportEmailRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.ChangeEmail(ctx, c.UserID, req)
}

func (s *PassportService) SetPassportEmailLogin(ctx context.Context, req *v1.SetPassportEmailLoginRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.SetEmailLogin(ctx, c.UserID, req.Enabled)
}

func (s *PassportService) SetupPassportTotp(ctx context.Context, _ *v1.PassportEmpty) (*v1.SetupPassportTotpResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.SetupTotp(ctx, c.UserID)
}

func (s *PassportService) EnablePassportTotp(ctx context.Context, req *v1.EnablePassportTotpRequest) (*v1.PassportRecoveryCodes, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.EnableTotp(ctx, c.UserID, req.Code)
}

func (s *PassportService) DisablePassportTotp(ctx context.Context, req *v1.DisablePassportTotpRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.DisableTotp(ctx, c.UserID, req.Password, req.Code)
}

func (s *PassportService) RegeneratePassportRecoveryCodes(ctx context.Context, req *v1.PassportTotpCodeRequest) (*v1.PassportRecoveryCodes, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.RegenerateRecoveryCodes(ctx, c.UserID, req.Code)
}

func (s *PassportService) ListPassportDevices(ctx context.Context, _ *v1.PassportEmpty) (*v1.ListPassportDevicesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Devices(ctx, c)
}

func (s *PassportService) RevokePassportDevice(ctx context.Context, req *v1.RevokePassportDeviceRequest) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.RevokeDevice(ctx, c.UserID, req.SessionId); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) LogoutPassportOthers(ctx context.Context, _ *v1.PassportEmpty) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.LogoutOthers(ctx, c); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) ListPassportApps(ctx context.Context, _ *v1.PassportEmpty) (*v1.ListPassportAppsResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Apps(ctx, c.UserID)
}

func (s *PassportService) RevokePassportApp(ctx context.Context, req *v1.RevokePassportAppRequest) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.RevokeApp(ctx, c.UserID, req.Id); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) RequestPassportDeletion(ctx context.Context, req *v1.RequestPassportDeletionRequest) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.RequestDeletion(ctx, c.UserID, req.Password)
}

func (s *PassportService) CancelPassportDeletion(ctx context.Context, _ *v1.PassportEmpty) (*v1.PassportMe, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.CancelDeletion(ctx, c.UserID)
}

func (s *PassportService) ListPassportMessages(ctx context.Context, req *v1.ListPassportMessagesRequest) (*v1.ListPassportMessagesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Messages(ctx, c.UserID, req)
}

func (s *PassportService) ReadPassportMessages(ctx context.Context, req *v1.ReadPassportMessagesRequest) (*v1.ReadPassportMessagesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.ReadMessages(ctx, c.UserID, req.Ids, req.All)
}

func (s *PassportService) ListPassportActivity(ctx context.Context, req *v1.ListPassportActivityRequest) (*v1.ListPassportActivityResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Activity(ctx, c.UserID, int(req.Limit))
}

// ---------------- 游戏账号 ----------------

func (s *PassportService) ListPassportLinks(ctx context.Context, _ *v1.PassportEmpty) (*v1.ListPassportLinksResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Links(ctx, c.UserID)
}

func (s *PassportService) SetPassportPrimaryLink(ctx context.Context, req *v1.PassportLinkRequest) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.SetPrimary(ctx, c.UserID, req.Id); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) UnbindPassportLink(ctx context.Context, req *v1.PassportLinkRequest) (*v1.PassportEmpty, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	if err := s.uc.Unbind(ctx, c.UserID, req.Id); err != nil {
		return nil, err
	}
	return &v1.PassportEmpty{}, nil
}

func (s *PassportService) SearchPassportBindServers(ctx context.Context, req *v1.SearchPassportBindServersRequest) (*v1.SearchPassportBindServersResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.SearchBindServers(ctx, c.UserID, req.Q, req.Game)
}

func (s *PassportService) BindPassportAccount(ctx context.Context, req *v1.BindPassportAccountRequest) (*v1.PassportLink, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Bind(ctx, c.UserID, req)
}

// ---------------- 游戏空间 ----------------

func (s *PassportService) GetPassportGame(ctx context.Context, req *v1.PassportGameRequest) (*v1.PassportGameResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Game(ctx, c.UserID, req.ServerId, true)
}

func (s *PassportService) GetPassportGameBrief(ctx context.Context, req *v1.PassportGameRequest) (*v1.PassportGameResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Game(ctx, c.UserID, req.ServerId, false)
}

func (s *PassportService) ListPassportSelfService(ctx context.Context, req *v1.PassportGameRequest) (*v1.ListPassportSelfServiceResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SelfService(ctx, c.UserID, req.ServerId)
}

func (s *PassportService) ListPassportGameRequests(ctx context.Context, req *v1.ListPassportGameRecordsRequest) (*v1.ListPassportGameRecordsResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Requests(ctx, c.UserID, req)
}

func (s *PassportService) ListPassportGameRecords(ctx context.Context, req *v1.ListPassportGameRecordsRequest) (*v1.ListPassportGameRecordsResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Records(ctx, c.UserID, req)
}

func (s *PassportService) RedeemPassportGiftCode(ctx context.Context, req *v1.RedeemPassportGiftCodeRequest) (*v1.RedeemPassportGiftCodeResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.Redeem(ctx, c.UserID, req.ServerId, req.Code)
}

// ---------------- OIDC ----------------

func (s *PassportService) GetPassportOidcAuthorization(ctx context.Context, req *v1.PassportOidcAuthorizeRequest) (*v1.PassportOidcAuthorizationInfo, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.OidcAuthorization(ctx, c.UserID, req)
}

func (s *PassportService) ApprovePassportOidcAuthorization(ctx context.Context, req *v1.PassportOidcAuthorizeRequest) (*v1.PassportOidcAuthorizeResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	redirect, err := s.uc.OidcApprove(ctx, c.UserID, s.issuer(ctx), req)
	if err != nil {
		return nil, err
	}
	return &v1.PassportOidcAuthorizeResponse{RedirectUrl: redirect}, nil
}

// RegisterOIDCServer 注册通行证 OIDC 的标准端点（返回协议原始 JSON，不走统一响应信封）。
func (s *PassportService) RegisterOIDCServer(srv *khttp.Server) {
	srv.HandleFunc("/.well-known/openid-configuration", s.oidcDiscovery)
	srv.HandleFunc("/api/v1/oidc/jwks", s.oidcJWKS)
	srv.HandleFunc("/api/v1/oidc/token", s.oidcToken)
	srv.HandleFunc("/api/v1/oidc/userinfo", s.oidcUserinfo)
}

func writeOidcError(w http.ResponseWriter, err error) {
	var oe *biz.OidcError
	if errors.As(err, &oe) {
		oidcserver.WriteError(w, oe.Status, oe.Code, oe.Err)
		return
	}
	oidcserver.WriteError(w, http.StatusInternalServerError, "server_error", err)
}

func (s *PassportService) oidcDiscovery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	data, err := s.uc.OidcDiscovery(r.Context(), passportIssuer(s.conf, r))
	if err != nil {
		writeOidcError(w, err)
		return
	}
	oidcserver.WriteJSON(w, http.StatusOK, data)
}

func (s *PassportService) oidcJWKS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	data, err := s.uc.OidcJWKS(r.Context())
	if err != nil {
		writeOidcError(w, err)
		return
	}
	oidcserver.WriteJSON(w, http.StatusOK, data)
}

func (s *PassportService) oidcToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	req, err := oidcserver.NewTokenRequest(r)
	if err != nil {
		oidcserver.WriteError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	data, err := s.uc.OidcToken(r.Context(), req, passportIssuer(s.conf, r))
	if err != nil {
		writeOidcError(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	oidcserver.WriteJSON(w, http.StatusOK, data)
}

func (s *PassportService) oidcUserinfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	token := oidcserver.BearerToken(r)
	if token == "" {
		oidcserver.WriteError(w, http.StatusUnauthorized, "invalid_token", oidcserver.ErrTokenInvalid)
		return
	}
	data, err := s.uc.OidcUserinfo(r.Context(), token, passportIssuer(s.conf, r))
	if err != nil {
		writeOidcError(w, err)
		return
	}
	oidcserver.WriteJSON(w, http.StatusOK, data)
}

// ---------------- 后台（主 server，权限 game:passport） ----------------
