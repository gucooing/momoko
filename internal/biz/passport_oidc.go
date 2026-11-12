package biz

import (
	"context"
	"crypto/rsa"
	"errors"
	"strings"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/passportuser"
	"momoko/pkg/common"
	"momoko/pkg/oidcserver"
)

// 用户通行证的 OIDC Provider（游戏 SDK「用户通行证登录」）：协议细节都在 pkg/oidcserver，
// 这里只编排通行证用户、通行证自己的客户端（passport_oidc_clients）与授权记录（passport_oidc_grants）。
// 签名密钥独立于管理后台的 OIDC。端点挂在通行证 server 上：/.well-known/openid-configuration、
// /oidc/authorize（前端授权页）、/api/v1/oidc/token、/api/v1/oidc/userinfo、/api/v1/oidc/jwks。

type oidcCode struct {
	oidcserver.AuthorizationCode
	GrantID string
}

// OidcError 协议错误（raw handler 转成标准 OAuth 错误）。
type OidcError struct {
	Status int
	Code   string
	Err    error
}

func (e *OidcError) Error() string { return e.Err.Error() }

func oidcFail(status int, code string, err error) error {
	return &OidcError{Status: status, Code: code, Err: err}
}

// oidcPassportErr 把协议包的错误映射为通行证错误代码（授权确认页用）。
func oidcPassportErr(err error) error {
	switch {
	case errors.Is(err, oidcserver.ErrInvalidClient):
		return ErrPassportOidcClient
	case errors.Is(err, oidcserver.ErrInvalidRedirectURI):
		return ErrPassportOidcRedirect
	case errors.Is(err, oidcserver.ErrInvalidScope):
		return ErrPassportOidcScope
	default:
		return ErrPassportOidcRequest
	}
}

func oidcClientOf(c *gen.PassportOidcClient) oidcserver.Client {
	return oidcserver.Client{
		ID:           c.ID,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Name:         c.Name,
		RedirectURIs: c.RedirectUris,
		Scopes:       c.Scopes,
		Active:       c.Active,
	}
}

// OidcEnabled 是否启用了「用户通行证登录」。
func (uc *PassportUsecase) OidcEnabled(ctx context.Context) bool { return uc.Settings(ctx).OidcEnabled }

// OidcEndpoints issuer 对应的端点（issuer 为 conf passport.public_url，空时按请求推断）。
func OidcEndpoints(issuer string) (oidcserver.Endpoints, error) {
	_, ep, err := oidcserver.NormalizeProviderConfig(oidcserver.ProviderConfig{IssuerURL: issuer}, "")
	return ep, err
}

func (uc *PassportUsecase) oidcSigningKey(ctx context.Context) (*rsa.PrivateKey, error) {
	raw, err := uc.config.Get(ctx, common.ConfigPassportOIDCPrivateKey)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		if raw, err = oidcserver.GeneratePrivateKeyPEM(); err != nil {
			return nil, err
		}
		if err := uc.config.BatchUpdate(ctx, map[common.ConfigKey]string{common.ConfigPassportOIDCPrivateKey: raw}); err != nil {
			return nil, err
		}
	}
	return oidcserver.ParsePrivateKeyPEM(raw)
}

func scopeSet(scope string) []string { return strings.Fields(oidcserver.NormalizeScopes(scope)) }

func covers(have, want []string) bool {
	set := map[string]bool{}
	for _, s := range have {
		set[s] = true
	}
	for _, s := range want {
		if !set[s] {
			return false
		}
	}
	return true
}

func (uc *PassportUsecase) oidcClientFor(ctx context.Context, req *v1.PassportOidcAuthorizeRequest) (*gen.PassportOidcClient, error) {
	if !uc.OidcEnabled(ctx) {
		return nil, ErrPassportOidcDisabled
	}
	cl, err := uc.repo.GetOidcClientByClientID(ctx, req.ClientId)
	if gen.IsNotFound(err) {
		return nil, ErrPassportOidcClient
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	if err := oidcserver.ValidateAuthorizeRequest(oidcserver.AuthorizeRequest{
		ResponseType: req.ResponseType,
		ClientID:     req.ClientId,
		RedirectURI:  req.RedirectUri,
		Scope:        req.Scope,
	}, oidcClientOf(cl)); err != nil {
		return nil, oidcPassportErr(err)
	}
	return cl, nil
}

// OidcAuthorization 授权确认页的应用信息。
func (uc *PassportUsecase) OidcAuthorization(ctx context.Context, userID string, req *v1.PassportOidcAuthorizeRequest) (*v1.PassportOidcAuthorizationInfo, error) {
	cl, err := uc.oidcClientFor(ctx, req)
	if err != nil {
		return nil, err
	}
	scopes := scopeSet(req.Scope)
	out := &v1.PassportOidcAuthorizationInfo{
		ClientName:  cl.Name,
		ClientId:    cl.ClientID,
		Description: cl.Description,
		RedirectUri: req.RedirectUri,
		Scopes:      scopes,
	}
	if cl.ServerID != "" {
		if s, err := uc.Servers().Get(ctx, cl.ServerID); err == nil {
			out.Server = uc.Servers().Brief(ctx, s)
		}
	}
	if g, err := uc.repo.GetGrant(ctx, userID, cl.ID); err == nil && g != nil && covers(g.Scopes, scopes) {
		out.Granted = true
	}
	return out, nil
}

// OidcApprove 同意授权：记录授权并生成授权码，返回应用的回调地址。
func (uc *PassportUsecase) OidcApprove(ctx context.Context, userID, issuer string, req *v1.PassportOidcAuthorizeRequest) (string, error) {
	cl, err := uc.oidcClientFor(ctx, req)
	if err != nil {
		return "", err
	}
	if _, err := OidcEndpoints(issuer); err != nil {
		return "", ErrPassportOidcRequest
	}
	scopes := scopeSet(req.Scope)
	if g, err := uc.repo.GetGrant(ctx, userID, cl.ID); err == nil && g != nil {
		scopes = mergeScopes(g.Scopes, scopes)
	}
	g, err := uc.repo.SaveGrant(ctx, userID, cl.ID, scopes)
	if err != nil {
		return "", ErrSystem(err)
	}
	code, err := oidcserver.GenerateAuthorizationCode()
	if err != nil {
		return "", ErrSystem(err)
	}
	ttl := oidcserver.AuthorizationCodeTTL(oidcserver.ProviderConfig{})
	uc.oidcCodes.Set(code, expiring[oidcCode]{
		V: oidcCode{
			AuthorizationCode: oidcserver.AuthorizationCode{
				Code:                code,
				ClientID:            cl.ClientID,
				UserID:              userID,
				RedirectURI:         req.RedirectUri,
				Scope:               oidcserver.NormalizeScopes(req.Scope),
				Nonce:               req.Nonce,
				CodeChallenge:       req.CodeChallenge,
				CodeChallengeMethod: req.CodeChallengeMethod,
				ExpiresAt:           time.Now().Add(ttl),
			},
			GrantID: g.ID,
		},
		Exp: time.Now().Add(ttl),
	})
	return oidcserver.BuildAuthorizeRedirect(req.RedirectUri, code, req.State), nil
}

func mergeScopes(a, b []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, s := range append(append([]string{}, a...), b...) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func (uc *PassportUsecase) userClaims(u *gen.PassportUser, scope string) oidcserver.UserClaims {
	scopes := strings.Fields(scope)
	has := func(s string) bool {
		for _, x := range scopes {
			if x == s {
				return true
			}
		}
		return false
	}
	c := oidcserver.UserClaims{Subject: u.ID}
	if has("profile") {
		c.Name, c.PreferredUsername, c.Picture = displayName(u), u.Username, u.Avatar
	}
	if has("email") && u.Email != "" {
		verified := u.EmailVerified
		c.Email, c.EmailVerified = u.Email, &verified
	}
	return c
}

// OidcDiscovery OpenID Provider Metadata。
func (uc *PassportUsecase) OidcDiscovery(ctx context.Context, issuer string) (map[string]any, error) {
	if !uc.OidcEnabled(ctx) {
		return nil, oidcFail(404, "not_found", oidcserver.ErrDisabled)
	}
	ep, err := OidcEndpoints(issuer)
	if err != nil {
		return nil, oidcFail(500, "server_error", err)
	}
	return oidcserver.Discovery(oidcserver.ProviderConfig{Enabled: true, IssuerURL: ep.IssuerURL}, ep), nil
}

// OidcJWKS 公钥集合。
func (uc *PassportUsecase) OidcJWKS(ctx context.Context) (map[string]any, error) {
	if !uc.OidcEnabled(ctx) {
		return nil, oidcFail(404, "not_found", oidcserver.ErrDisabled)
	}
	key, err := uc.oidcSigningKey(ctx)
	if err != nil {
		return nil, oidcFail(500, "server_error", err)
	}
	return oidcserver.JWKS(key), nil
}

// OidcToken 授权码换令牌。
func (uc *PassportUsecase) OidcToken(ctx context.Context, req oidcserver.TokenRequest, issuer string) (*oidcserver.TokenResponse, error) {
	if !uc.OidcEnabled(ctx) {
		return nil, oidcFail(400, "invalid_request", oidcserver.ErrDisabled)
	}
	ep, err := OidcEndpoints(issuer)
	if err != nil {
		return nil, oidcFail(500, "server_error", err)
	}
	cl, err := uc.repo.GetOidcClientByClientID(ctx, req.ClientID)
	if err != nil {
		return nil, oidcFail(401, "invalid_client", oidcserver.ErrInvalidClient)
	}
	held, ok := uc.oidcCodes.Take(req.Code)
	if !ok || !held.alive() {
		return nil, oidcFail(400, "invalid_grant", oidcserver.ErrInvalidCode)
	}
	if err := oidcserver.ValidateTokenRequest(req, oidcClientOf(cl), held.V.AuthorizationCode); err != nil {
		status, code := 400, "invalid_grant"
		if errors.Is(err, oidcserver.ErrInvalidClient) || errors.Is(err, oidcserver.ErrInvalidClientSecret) {
			status, code = 401, "invalid_client"
		} else if errors.Is(err, oidcserver.ErrUnsupportedGrant) {
			code = "unsupported_grant_type"
		}
		return nil, oidcFail(status, code, err)
	}
	g, err := uc.repo.GetGrant(ctx, held.V.UserID, cl.ID)
	if err != nil || g == nil || g.ID != held.V.GrantID {
		return nil, oidcFail(400, "invalid_grant", oidcserver.ErrInvalidCode)
	}
	u, err := uc.repo.GetUser(ctx, held.V.UserID)
	if err != nil || u.Status != passportuser.StatusActive {
		return nil, oidcFail(400, "invalid_grant", oidcserver.ErrInvalidCode)
	}
	key, err := uc.oidcSigningKey(ctx)
	if err != nil {
		return nil, oidcFail(500, "server_error", err)
	}
	cfg := oidcserver.ProviderConfig{}
	claims := uc.userClaims(u, held.V.Scope)
	access, expires, err := oidcserver.SignAccessToken(key, ep.IssuerURL, cl.ClientID, held.V.Scope, claims, oidcserver.AccessTokenTTL(cfg))
	if err != nil {
		return nil, oidcFail(500, "server_error", err)
	}
	idToken, err := oidcserver.SignIDToken(key, ep.IssuerURL, cl.ClientID, held.V.Nonce, claims, oidcserver.IDTokenTTL(cfg))
	if err != nil {
		return nil, oidcFail(500, "server_error", err)
	}
	now := time.Now()
	_ = uc.repo.MarkOidcClientUsed(context.WithoutCancel(ctx), cl.ID, now)
	_ = uc.repo.TouchGrant(context.WithoutCancel(ctx), g.ID, now)
	return &oidcserver.TokenResponse{
		AccessToken: access,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(expires).Seconds()),
		IDToken:     idToken,
		Scope:       held.V.Scope,
	}, nil
}

// OidcUserinfo 用访问令牌读取用户资料（授权被撤销或账号停用后令牌即失效）。
func (uc *PassportUsecase) OidcUserinfo(ctx context.Context, token, issuer string) (*oidcserver.UserinfoResponse, error) {
	if !uc.OidcEnabled(ctx) {
		return nil, oidcFail(401, "invalid_token", oidcserver.ErrDisabled)
	}
	ep, err := OidcEndpoints(issuer)
	if err != nil {
		return nil, oidcFail(500, "server_error", err)
	}
	key, err := uc.oidcSigningKey(ctx)
	if err != nil {
		return nil, oidcFail(500, "server_error", err)
	}
	claims, err := oidcserver.ParseAccessToken(strings.TrimSpace(token), key, ep.IssuerURL)
	if err != nil {
		return nil, oidcFail(401, "invalid_token", err)
	}
	cl, err := uc.repo.GetOidcClientByClientID(ctx, claims.ClientID)
	if err != nil || !cl.Active {
		return nil, oidcFail(401, "invalid_token", oidcserver.ErrTokenInvalid)
	}
	if g, err := uc.repo.GetGrant(ctx, claims.Subject, cl.ID); err != nil || g == nil {
		return nil, oidcFail(401, "invalid_token", oidcserver.ErrTokenInvalid)
	}
	u, err := uc.repo.GetUser(ctx, claims.Subject)
	if err != nil || u.Status != passportuser.StatusActive {
		return nil, oidcFail(401, "invalid_token", oidcserver.ErrTokenInvalid)
	}
	c := uc.userClaims(u, claims.Scope)
	return &oidcserver.UserinfoResponse{
		Subject:           c.Subject,
		Name:              c.Name,
		PreferredUsername: c.PreferredUsername,
		Email:             c.Email,
		Picture:           c.Picture,
		EmailVerified:     c.EmailVerified,
	}, nil
}
