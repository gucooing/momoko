package biz

import (
	"context"
	"crypto/rsa"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/passportuser"
	"momoko/pkg/common"
	"momoko/pkg/mgp"
	"momoko/pkg/mgp/gakumas"
	"momoko/pkg/oidcserver"
)

// 用户通行证的 OIDC Provider（游戏 SDK「用户通行证登录」）：协议细节都在 pkg/oidcserver，
// 这里只编排通行证用户、通行证自己的客户端（passport_oidc_clients）与授权记录（passport_oidc_grants）。
// 签名密钥独立于管理后台的 OIDC。端点挂在通行证 server 上：/.well-known/openid-configuration、
// /oidc/authorize（前端授权页）、/api/v1/oidc/token、/api/v1/oidc/userinfo、/api/v1/oidc/jwks。
//
// 应用关联了游戏服务器（passport_oidc_clients.server_id）时，由 Momoko 决定用户在那台服务器上玩哪个游戏账号：
// 同意授权前用户要在那台服务器上有绑定的游戏账号（绑定已有账号，或授权时新建一个并以 oidc 方式绑定），
// ID Token、Access Token 与 userinfo 带上 game_server_id 与 game_player_id（绑定的 MGP pid），游戏服务端据此登录。

// 应用关联了游戏服务器、用户在那台服务器上有绑定时签发的声明。
const (
	OidcClaimGameServerID = "game_server_id"
	OidcClaimGamePlayerID = "game_player_id"
)

// 授权页上不能新建游戏账号的原因（game_account_reason）。
const (
	oidcGameUnsupported = "unsupported"
	oidcGameNotVisible  = "not_visible"
	oidcGameCooldown    = "cooldown"
)

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

// oidcServer 应用关联的游戏服务器（服务器已删除时为 nil）。
func (uc *PassportUsecase) oidcServer(ctx context.Context, cl *gen.PassportOidcClient) (*gen.GameServer, error) {
	s, err := uc.Servers().Get(ctx, cl.ServerID)
	if errors.Is(err, ErrGameServerNotFound) {
		return nil, nil
	}
	return s, err
}

// oidcGameLink 用户在服务器上有效的游戏账号绑定（没有时为 nil）。
func (uc *PassportUsecase) oidcGameLink(ctx context.Context, userID, serverID string) (*gen.GameAccountLink, error) {
	l, err := uc.repo.LinkOf(ctx, userID, serverID)
	if gen.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if l.PlayerID == "" {
		return nil, nil
	}
	return l, nil
}

// oidcGameAccountBlock 为用户在服务器上新建游戏账号的限制，与绑定相同：学マス 且接入了 MGP、服务器对用户可见、
// 不在解绑冷却中（冷却中只能绑回原来的账号）。返回空串表示可以新建；冷却中同时返回冷却结束时间。
func (uc *PassportUsecase) oidcGameAccountBlock(ctx context.Context, u *gen.PassportUser, s *gen.GameServer) (string, *time.Time) {
	if s == nil || s.Game != gakumas.ID || !MGPConfigured(s) {
		return oidcGameUnsupported, nil
	}
	if visible, _ := passportVisible(s, "", uc.inviteServer(ctx, u)); !visible {
		return oidcGameNotVisible, nil
	}
	if until, _ := uc.cooldownUntil(ctx, u.ID, s.ID); until != nil {
		return oidcGameCooldown, until
	}
	return "", nil
}

func oidcGameAccountProto(l *gen.GameAccountLink) *v1.PassportOidcGameAccount {
	return &v1.PassportOidcGameAccount{
		PlayerId:    l.PlayerID,
		DisplayId:   l.DisplayID,
		Nickname:    l.Nickname,
		Level:       int32(l.Level),
		Account:     l.Account,
		AvatarAsset: l.AvatarAsset,
	}
}

// OidcAuthorization 授权确认页的应用信息；应用关联了游戏服务器时带上用户在那台服务器上的游戏账号状态。
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
		s, err := uc.oidcServer(ctx, cl)
		if err != nil {
			return nil, err
		}
		if s != nil {
			out.Server = uc.Servers().Brief(ctx, s)
		}
		l, err := uc.oidcGameLink(ctx, userID, cl.ServerID)
		if err != nil {
			return nil, ErrSystem(err)
		}
		if l != nil {
			out.GameAccount = oidcGameAccountProto(l)
		} else {
			u, err := uc.user(ctx, userID)
			if err != nil {
				return nil, err
			}
			reason, until := uc.oidcGameAccountBlock(ctx, u, s)
			out.GameAccountRequired = true
			out.GameAccountCreatable = reason == ""
			out.GameAccountReason = reason
			if until != nil {
				out.GameAccountCooldownUntil = timestamppb.New(*until)
			}
		}
	}
	if g, err := uc.repo.GetGrant(ctx, userID, cl.ID); err == nil && g != nil && covers(g.Scopes, scopes) {
		out.Granted = true
	}
	return out, nil
}

// OidcApprove 同意授权：记录授权并生成授权码，返回应用的回调地址。
// 应用关联了游戏服务器时，用户在那台服务器上要有绑定的游戏账号；没有时 create_game_account 为 true 则先新建一个。
func (uc *PassportUsecase) OidcApprove(ctx context.Context, userID, issuer string, req *v1.PassportOidcAuthorizeRequest) (string, error) {
	cl, err := uc.oidcClientFor(ctx, req)
	if err != nil {
		return "", err
	}
	if _, err := OidcEndpoints(issuer); err != nil {
		return "", ErrPassportOidcRequest
	}
	if cl.ServerID != "" {
		if err := uc.oidcEnsureGameAccount(ctx, userID, cl, req.CreateGameAccount); err != nil {
			return "", err
		}
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

// oidcEnsureGameAccount 用户在应用关联的服务器上有绑定的游戏账号时直接通过；没有时 create 为 false 返回
// ErrPassportOidcGameAccount，为 true 则按绑定的限制经 MGP 新建一个游戏账号（执行者为通行证显示名，新的幂等键）
// 并以 oidc 方式绑定。同一用户在同一服务器上的新建串行执行，并发的两次同意只建一个账号。
func (uc *PassportUsecase) oidcEnsureGameAccount(ctx context.Context, userID string, cl *gen.PassportOidcClient, create bool) error {
	if l, err := uc.oidcGameLink(ctx, userID, cl.ServerID); err != nil {
		return ErrSystem(err)
	} else if l != nil {
		return nil
	}
	if !create {
		return ErrPassportOidcGameAccount
	}
	u, err := uc.user(ctx, userID)
	if err != nil {
		return err
	}
	s, err := uc.oidcServer(ctx, cl)
	if err != nil {
		return err
	}
	lock := uc.bindLock(userID + "\x00" + cl.ServerID)
	lock.Lock()
	defer lock.Unlock()
	if l, err := uc.oidcGameLink(ctx, userID, cl.ServerID); err != nil {
		return ErrSystem(err)
	} else if l != nil {
		return nil
	}
	switch reason, _ := uc.oidcGameAccountBlock(ctx, u, s); reason {
	case oidcGameUnsupported:
		return ErrPassportBindUnsupported
	case oidcGameNotVisible:
		return ErrPassportServerNotBindable
	case oidcGameCooldown:
		return ErrPassportBindCooldown
	}
	client, err := uc.Servers().Client(s)
	if err != nil {
		return ErrPassportBindUnsupported
	}
	p, err := gakumas.New(client).CreateAccount(ctx, mgp.OperationOptions{
		Operator:       displayName(u),
		IdempotencyKey: "passport-" + uuid.NewString(),
	})
	if err != nil {
		if mgp.IsUnreachable(err) {
			return ErrPassportGameUnavailable
		}
		return GameMGPError(err)
	}
	if p.GetPid() == "" {
		return ErrPassportGameUnavailable
	}
	if _, err := uc.createLink(ctx, u, s, p, PassportBindOIDC); err != nil {
		uc.log.Errorf("oidc: game account %s created on server %s but not linked to passport user %s: %v", p.GetPid(), s.ID, u.ID, err)
		return err
	}
	return nil
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

// userClaims 按授权范围生成用户声明；应用关联了游戏服务器且用户在那台服务器上有绑定时，加上 game_server_id 与 game_player_id。
func (uc *PassportUsecase) userClaims(ctx context.Context, u *gen.PassportUser, cl *gen.PassportOidcClient, scope string) (oidcserver.UserClaims, error) {
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
	if cl.ServerID != "" {
		l, err := uc.oidcGameLink(ctx, u.ID, cl.ServerID)
		if err != nil {
			return c, err
		}
		if l != nil {
			c.Extra = map[string]any{
				OidcClaimGameServerID: cl.ServerID,
				OidcClaimGamePlayerID: l.PlayerID,
			}
		}
	}
	return c, nil
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
	return oidcserver.Discovery(oidcserver.ProviderConfig{
		Enabled:     true,
		IssuerURL:   ep.IssuerURL,
		ExtraClaims: []string{OidcClaimGameServerID, OidcClaimGamePlayerID},
	}, ep), nil
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
	claims, err := uc.userClaims(ctx, u, cl, held.V.Scope)
	if err != nil {
		return nil, oidcFail(500, "server_error", err)
	}
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
	c, err := uc.userClaims(ctx, u, cl, claims.Scope)
	if err != nil {
		return nil, oidcFail(500, "server_error", err)
	}
	return &oidcserver.UserinfoResponse{
		Subject:           c.Subject,
		Name:              c.Name,
		PreferredUsername: c.PreferredUsername,
		Email:             c.Email,
		Picture:           c.Picture,
		EmailVerified:     c.EmailVerified,
		Extra:             c.Extra,
	}, nil
}
