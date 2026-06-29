package service

import (
	"context"
	"net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/oidcserver"
)

type OIDCService struct {
	v1.UnimplementedOIDCServiceServer

	uc *biz.OIDCUsecase
}

func NewOIDCService(uc *biz.OIDCUsecase) *OIDCService {
	return &OIDCService{uc: uc}
}

// RegisterOIDCServer 注册标准 OIDC 端点。
// 这些端点必须返回协议原始 JSON，因此不通过 proto response encoder。
func (s *OIDCService) RegisterOIDCServer(srv *khttp.Server) {
	srv.HandleFunc("/.well-known/openid-configuration", s.discovery)
	srv.HandleFunc("/api/v1/oidc/.well-known/openid-configuration", s.discovery)
	srv.HandleFunc("/api/v1/oidc/jwks", s.jwks)
	srv.HandleFunc("/api/v1/oidc/token", s.token)
	srv.HandleFunc("/api/v1/oidc/userinfo", s.userinfo)
}

func (s *OIDCService) OIDCConfig(ctx context.Context, _ *v1.OIDCConfigRequest) (*v1.OIDCConfigResponse, error) {
	config, err := s.uc.Config(ctx, oidcRequestOrigin(ctx))
	if err != nil {
		return nil, err
	}
	return &v1.OIDCConfigResponse{Config: config}, nil
}

func (s *OIDCService) UpdateOIDCConfig(ctx context.Context, req *v1.UpdateOIDCConfigRequest) (*v1.UpdateOIDCConfigResponse, error) {
	config, err := s.uc.UpdateConfig(ctx, req, oidcRequestOrigin(ctx))
	if err != nil {
		return nil, err
	}
	return &v1.UpdateOIDCConfigResponse{Config: config}, nil
}

func (s *OIDCService) ListOIDCClients(ctx context.Context, req *v1.ListOIDCClientsRequest) (*v1.ListOIDCClientsResponse, error) {
	clients, total, err := s.uc.ListClients(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.ListOIDCClientsResponse{
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		Clients:  clients,
	}, nil
}

func (s *OIDCService) CreateOIDCClient(ctx context.Context, req *v1.CreateOIDCClientRequest) (*v1.CreateOIDCClientResponse, error) {
	client, err := s.uc.CreateClient(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateOIDCClientResponse{Client: client}, nil
}

func (s *OIDCService) UpdateOIDCClient(ctx context.Context, req *v1.UpdateOIDCClientRequest) (*v1.UpdateOIDCClientResponse, error) {
	client, err := s.uc.UpdateClient(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateOIDCClientResponse{Client: client}, nil
}

func (s *OIDCService) DeleteOIDCClient(ctx context.Context, req *v1.DeleteOIDCClientRequest) (*v1.DeleteOIDCClientResponse, error) {
	if err := s.uc.DeleteClient(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteOIDCClientResponse{}, nil
}

func (s *OIDCService) RefreshOIDCClientSecret(ctx context.Context, req *v1.RefreshOIDCClientSecretRequest) (*v1.RefreshOIDCClientSecretResponse, error) {
	client, err := s.uc.RefreshClientSecret(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.RefreshOIDCClientSecretResponse{Client: client}, nil
}

func (s *OIDCService) OIDCAuthorizationInfo(ctx context.Context, req *v1.OIDCAuthorizationInfoRequest) (*v1.OIDCAuthorizationInfoResponse, error) {
	info, err := s.uc.AuthorizationInfo(ctx, req, oidcRequestOrigin(ctx))
	if err != nil {
		return nil, err
	}
	return &v1.OIDCAuthorizationInfoResponse{Info: info}, nil
}

func (s *OIDCService) CreateOIDCAuthorizationCode(ctx context.Context, req *v1.CreateOIDCAuthorizationCodeRequest) (*v1.CreateOIDCAuthorizationCodeResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	redirectURL, err := s.uc.CreateAuthorizationCode(ctx, authCtx.UserID, req, oidcRequestOrigin(ctx))
	if err != nil {
		return nil, err
	}
	return &v1.CreateOIDCAuthorizationCodeResponse{RedirectUrl: redirectURL}, nil
}

// discovery 输出 OpenID Provider Metadata。
func (s *OIDCService) discovery(w khttp.ResponseWriter, r *khttp.Request) {
	data, err := s.uc.Discovery(r.Context(), oidcserver.RequestOrigin(r))
	if err != nil {
		oidcserver.WriteError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	oidcserver.WriteJSON(w, http.StatusOK, data)
}

// jwks 输出用于验证 RS256 Token 的公钥集合。
func (s *OIDCService) jwks(w khttp.ResponseWriter, r *khttp.Request) {
	data, err := s.uc.JWKS(r.Context())
	if err != nil {
		oidcserver.WriteError(w, http.StatusBadRequest, "server_error", err)
		return
	}
	oidcserver.WriteJSON(w, http.StatusOK, data)
}

// token 处理 authorization_code 换 token。
func (s *OIDCService) token(w khttp.ResponseWriter, r *khttp.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	req, err := oidcserver.NewTokenRequest(r)
	if err != nil {
		oidcserver.WriteError(w, http.StatusBadRequest, "invalid_request", err)
		return
	}
	data, err := s.uc.ExchangeToken(r.Context(), req, oidcserver.RequestOrigin(r))
	if err != nil {
		oidcserver.WriteError(w, http.StatusBadRequest, "invalid_grant", err)
		return
	}
	oidcserver.WriteJSON(w, http.StatusOK, data)
}

// userinfo 使用 OIDC access_token 返回用户资料。
func (s *OIDCService) userinfo(w khttp.ResponseWriter, r *khttp.Request) {
	token := oidcserver.BearerToken(r)
	if token == "" {
		oidcserver.WriteError(w, http.StatusUnauthorized, "invalid_token", biz.ErrTokenInvalid)
		return
	}
	data, err := s.uc.Userinfo(r.Context(), token, oidcserver.RequestOrigin(r))
	if err != nil {
		oidcserver.WriteError(w, http.StatusUnauthorized, "invalid_token", err)
		return
	}
	oidcserver.WriteJSON(w, http.StatusOK, data)
}

// oidcRequestOrigin 从 Kratos HTTP 上下文提取请求 origin；具体推导规则在 pkg/oidcserver 内。
func oidcRequestOrigin(ctx context.Context) string {
	req, ok := khttp.RequestFromServerContext(ctx)
	if !ok {
		return ""
	}
	return oidcserver.RequestOrigin(req)
}
