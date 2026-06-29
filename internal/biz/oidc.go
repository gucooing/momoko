package biz

import (
	"context"
	"crypto/rsa"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/cache"
	"momoko/pkg/common"
	"momoko/pkg/oidcserver"
	"momoko/pkg/response"
)

type OIDCRepo interface {
	ListClients(ctx context.Context, page, pageSize int64, keywords *string) ([]*gen.OIDCClient, int64, error)
	GetClientByClientID(ctx context.Context, clientID string) (*gen.OIDCClient, error)
	CreateClient(ctx context.Context, client *gen.OIDCClient) (*gen.OIDCClient, error)
	UpdateClient(ctx context.Context, client *gen.OIDCClient) (*gen.OIDCClient, error)
	UpdateClientSecret(ctx context.Context, id, secret string) (*gen.OIDCClient, error)
	DeleteClient(ctx context.Context, id string) error
}

// OIDCUsecase 只负责把配置、用户、客户端数据交给 pkg/oidcserver 处理。
// OIDC 协议校验、PKCE、JWT/JWKS 等细节都在 pkg 内，避免 internal 层承载协议算法。
type OIDCUsecase struct {
	repo   OIDCRepo
	config ConfigRepo
	users  UserRepo
	codes  *cache.Cache[string, oidcserver.AuthorizationCode]
}

func NewOIDCUsecase(repo OIDCRepo, config ConfigRepo, users UserRepo) *OIDCUsecase {
	return &OIDCUsecase{
		repo:   repo,
		config: config,
		users:  users,
		codes:  cache.New[string, oidcserver.AuthorizationCode](24 * time.Hour),
	}
}

// Config 返回后台配置页需要展示的 Provider 配置和自动推导端点。
func (u *OIDCUsecase) Config(ctx context.Context, origin string) (*v1.OIDCConfig, error) {
	config, endpoints, err := u.providerConfig(ctx, origin)
	if err != nil {
		return nil, oidcErr(err)
	}
	return toOIDCConfig(config, endpoints), nil
}

// UpdateConfig 保存 OIDC Provider 配置。
// issuer 会先交给 pkg 校验，确保写入的地址能作为标准 OIDC 信任根。
func (u *OIDCUsecase) UpdateConfig(ctx context.Context, req *v1.UpdateOIDCConfigRequest, origin string) (*v1.OIDCConfig, error) {
	config := oidcserver.ProviderConfig{
		Enabled:                     req.Enabled,
		IssuerURL:                   req.IssuerUrl,
		AccessTokenTTLSeconds:       req.AccessTokenTtlSeconds,
		IDTokenTTLSeconds:           req.IdTokenTtlSeconds,
		AuthorizationCodeTTLSeconds: req.AuthorizationCodeTtlSeconds,
	}
	normalized, _, err := oidcserver.NormalizeProviderConfig(config, origin)
	if err != nil {
		return nil, oidcErr(err)
	}
	err = u.config.BatchUpdate(ctx, map[common.ConfigKey]string{
		common.ConfigOIDCEnabled:                     strconv.FormatBool(normalized.Enabled),
		common.ConfigOIDCIssuerURL:                   normalized.IssuerURL,
		common.ConfigOIDCAccessTokenTTLSeconds:       strconv.FormatInt(normalized.AccessTokenTTLSeconds, 10),
		common.ConfigOIDCIDTokenTTLSeconds:           strconv.FormatInt(normalized.IDTokenTTLSeconds, 10),
		common.ConfigOIDCAuthorizationCodeTTLSeconds: strconv.FormatInt(normalized.AuthorizationCodeTTLSeconds, 10),
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	return u.Config(ctx, origin)
}

// ListClients 返回后台客户端列表；列表接口不返回完整 Client Secret。
func (u *OIDCUsecase) ListClients(ctx context.Context, req *v1.ListOIDCClientsRequest) ([]*v1.OIDCClientInfo, int64, error) {
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}
	clients, total, err := u.repo.ListClients(ctx, req.Page, req.PageSize, req.Keywords)
	if err != nil {
		return nil, 0, ErrSystem(err)
	}
	list := make([]*v1.OIDCClientInfo, 0, len(clients))
	for _, client := range clients {
		list = append(list, toOIDCClientInfo(client, false))
	}
	return list, total, nil
}

// CreateClient 生成新的 OIDC 客户端。
// Client Secret 只在创建响应中完整返回一次，后续列表只显示脱敏值。
func (u *OIDCUsecase) CreateClient(ctx context.Context, req *v1.CreateOIDCClientRequest) (*v1.OIDCClientInfo, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrOIDCClientName
	}
	if !oidcserver.ValidRedirectURIs(req.RedirectUris) {
		return nil, ErrOIDCRedirectURI
	}
	clientID, err := oidcserver.GenerateClientID()
	if err != nil {
		return nil, ErrSystem(err)
	}
	secret, err := oidcserver.GenerateClientSecret()
	if err != nil {
		return nil, ErrSystem(err)
	}
	client, err := u.repo.CreateClient(ctx, &gen.OIDCClient{
		ID:           uuid.NewString(),
		Name:         strings.TrimSpace(req.Name),
		ClientID:     clientID,
		ClientSecret: secret,
		RedirectUris: normalizeURIList(req.RedirectUris),
		Scopes:       normalizeScopeList(req.Scopes),
		Active:       req.Active,
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toOIDCClientInfo(client, true), nil
}

// UpdateClient 更新客户端基础信息，不改 Client ID 和 Client Secret。
func (u *OIDCUsecase) UpdateClient(ctx context.Context, req *v1.UpdateOIDCClientRequest) (*v1.OIDCClientInfo, error) {
	if strings.TrimSpace(req.Name) == "" {
		return nil, ErrOIDCClientName
	}
	if !oidcserver.ValidRedirectURIs(req.RedirectUris) {
		return nil, ErrOIDCRedirectURI
	}
	client, err := u.repo.UpdateClient(ctx, &gen.OIDCClient{
		ID:           req.Id,
		Name:         strings.TrimSpace(req.Name),
		RedirectUris: normalizeURIList(req.RedirectUris),
		Scopes:       normalizeScopeList(req.Scopes),
		Active:       req.Active,
	})
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrOIDCClientNotFound
		}
		return nil, ErrSystem(err)
	}
	return toOIDCClientInfo(client, false), nil
}

// DeleteClient 删除客户端配置。
func (u *OIDCUsecase) DeleteClient(ctx context.Context, id string) error {
	if err := u.repo.DeleteClient(ctx, id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// RefreshClientSecret 重新生成 Client Secret，并只在本次响应中完整返回。
func (u *OIDCUsecase) RefreshClientSecret(ctx context.Context, id string) (*v1.OIDCClientInfo, error) {
	secret, err := oidcserver.GenerateClientSecret()
	if err != nil {
		return nil, ErrSystem(err)
	}
	client, err := u.repo.UpdateClientSecret(ctx, id, secret)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrOIDCClientNotFound
		}
		return nil, ErrSystem(err)
	}
	return toOIDCClientInfo(client, true), nil
}

// AuthorizationInfo 返回授权确认页需要展示的客户端名称和授权参数。
// 这里会复用授权请求校验，避免前端展示一个后续无法授权的无效客户端。
func (u *OIDCUsecase) AuthorizationInfo(ctx context.Context, req *v1.OIDCAuthorizationInfoRequest, origin string) (*v1.OIDCAuthorizationInfo, error) {
	config, _, err := u.providerConfig(ctx, origin)
	if err != nil {
		return nil, oidcErr(err)
	}
	if !config.Enabled {
		return nil, oidcErr(oidcserver.ErrDisabled)
	}
	client, err := u.repo.GetClientByClientID(ctx, req.ClientId)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, oidcErr(oidcserver.ErrInvalidClient)
		}
		return nil, ErrSystem(err)
	}
	authReq := oidcserver.AuthorizeRequest{
		ResponseType: req.ResponseType,
		ClientID:     req.ClientId,
		RedirectURI:  req.RedirectUri,
		Scope:        req.Scope,
	}
	if err := oidcserver.ValidateAuthorizeRequest(authReq, toOIDCPkgClient(client)); err != nil {
		return nil, oidcErr(err)
	}
	return &v1.OIDCAuthorizationInfo{
		ClientName:  client.Name,
		ClientId:    client.ClientID,
		RedirectUri: req.RedirectUri,
		Scope:       oidcserver.NormalizeScopes(req.Scope),
	}, nil
}

// CreateAuthorizationCode 由前端授权确认页调用。
// 当前登录用户确认后，服务端生成一次性授权码并返回第三方客户端回调地址。
func (u *OIDCUsecase) CreateAuthorizationCode(ctx context.Context, userID string, req *v1.CreateOIDCAuthorizationCodeRequest, origin string) (string, error) {
	config, _, err := u.providerConfig(ctx, origin)
	if err != nil {
		return "", oidcErr(err)
	}
	if !config.Enabled {
		return "", oidcErr(oidcserver.ErrDisabled)
	}
	client, err := u.repo.GetClientByClientID(ctx, req.ClientId)
	if err != nil {
		if gen.IsNotFound(err) {
			return "", oidcErr(oidcserver.ErrInvalidClient)
		}
		return "", ErrSystem(err)
	}
	authReq := oidcserver.AuthorizeRequest{
		ResponseType:        req.ResponseType,
		ClientID:            req.ClientId,
		RedirectURI:         req.RedirectUri,
		Scope:               req.Scope,
		State:               req.State,
		Nonce:               req.Nonce,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
	}
	pkgClient := toOIDCPkgClient(client)
	if err := oidcserver.ValidateAuthorizeRequest(authReq, pkgClient); err != nil {
		return "", oidcErr(err)
	}
	code, err := oidcserver.GenerateAuthorizationCode()
	if err != nil {
		return "", ErrSystem(err)
	}
	u.codes.Set(code, oidcserver.AuthorizationCode{
		Code:                code,
		ClientID:            req.ClientId,
		UserID:              userID,
		RedirectURI:         req.RedirectUri,
		Scope:               oidcserver.NormalizeScopes(req.Scope),
		Nonce:               req.Nonce,
		CodeChallenge:       req.CodeChallenge,
		CodeChallengeMethod: req.CodeChallengeMethod,
		ExpiresAt:           time.Now().Add(oidcserver.AuthorizationCodeTTL(config)),
	})
	return oidcserver.BuildAuthorizeRedirect(req.RedirectUri, code, req.State), nil
}

// Discovery 返回标准 OpenID Provider Metadata。
func (u *OIDCUsecase) Discovery(ctx context.Context, origin string) (map[string]any, error) {
	config, endpoints, err := u.providerConfig(ctx, origin)
	if err != nil {
		return nil, oidcErr(err)
	}
	if !config.Enabled {
		return nil, oidcErr(oidcserver.ErrDisabled)
	}
	return oidcserver.Discovery(config, endpoints), nil
}

// JWKS 返回客户端验证 ID Token 的公钥集合。
func (u *OIDCUsecase) JWKS(ctx context.Context) (map[string]any, error) {
	config, _, err := u.providerConfig(ctx, "")
	if err != nil {
		return nil, oidcErr(err)
	}
	if !config.Enabled {
		return nil, oidcErr(oidcserver.ErrDisabled)
	}
	key, err := u.signingKey(ctx)
	if err != nil {
		return nil, oidcErr(err)
	}
	return oidcserver.JWKS(key), nil
}

// ExchangeToken 处理授权码换 token。
// 该方法只编排数据读取和响应生成，校验细节由 pkg/oidcserver 完成。
func (u *OIDCUsecase) ExchangeToken(ctx context.Context, req oidcserver.TokenRequest, origin string) (*oidcserver.TokenResponse, error) {
	config, endpoints, err := u.providerConfig(ctx, origin)
	if err != nil {
		return nil, oidcErr(err)
	}
	if !config.Enabled {
		return nil, oidcErr(oidcserver.ErrDisabled)
	}
	client, err := u.repo.GetClientByClientID(ctx, req.ClientID)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, oidcErr(oidcserver.ErrInvalidClient)
		}
		return nil, ErrSystem(err)
	}
	code, ok := u.codes.Take(req.Code)
	if !ok {
		return nil, oidcErr(oidcserver.ErrInvalidCode)
	}
	if err := oidcserver.ValidateTokenRequest(req, toOIDCPkgClient(client), code); err != nil {
		return nil, oidcErr(err)
	}
	user, err := u.users.FindByID(ctx, code.UserID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	key, err := u.signingKey(ctx)
	if err != nil {
		return nil, oidcErr(err)
	}
	userClaims := toOIDCUserClaims(user)
	accessToken, expires, err := oidcserver.SignAccessToken(key, endpoints.IssuerURL, client.ClientID, code.Scope, userClaims, oidcserver.AccessTokenTTL(config))
	if err != nil {
		return nil, ErrSystem(err)
	}
	idToken, err := oidcserver.SignIDToken(key, endpoints.IssuerURL, client.ClientID, code.Nonce, userClaims, oidcserver.IDTokenTTL(config))
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &oidcserver.TokenResponse{
		AccessToken: accessToken,
		TokenType:   "Bearer",
		ExpiresIn:   int64(time.Until(expires).Seconds()),
		IDToken:     idToken,
		Scope:       code.Scope,
	}, nil
}

// Userinfo 使用 OIDC access_token 返回用户基础资料。
func (u *OIDCUsecase) Userinfo(ctx context.Context, token string, origin string) (*oidcserver.UserinfoResponse, error) {
	config, endpoints, err := u.providerConfig(ctx, origin)
	if err != nil {
		return nil, oidcErr(err)
	}
	if !config.Enabled {
		return nil, oidcErr(oidcserver.ErrDisabled)
	}
	key, err := u.signingKey(ctx)
	if err != nil {
		return nil, oidcErr(err)
	}
	claims, err := oidcserver.ParseAccessToken(strings.TrimSpace(token), key, endpoints.IssuerURL)
	if err != nil {
		return nil, oidcErr(err)
	}
	user, err := u.users.FindByID(ctx, claims.Subject)
	if err != nil {
		return nil, ErrSystem(err)
	}
	userClaims := toOIDCUserClaims(user)
	return &oidcserver.UserinfoResponse{
		Subject:           userClaims.Subject,
		Name:              userClaims.Name,
		PreferredUsername: userClaims.PreferredUsername,
		Email:             userClaims.Email,
	}, nil
}

// providerConfig 从系统配置表读取 OIDC Provider 配置并交给 pkg 归一化。
func (u *OIDCUsecase) providerConfig(ctx context.Context, origin string) (oidcserver.ProviderConfig, oidcserver.Endpoints, error) {
	enabledValue, err := u.config.Get(ctx, common.ConfigOIDCEnabled)
	if err != nil {
		return oidcserver.ProviderConfig{}, oidcserver.Endpoints{}, err
	}
	enabled, err := strconv.ParseBool(enabledValue)
	if err != nil {
		return oidcserver.ProviderConfig{}, oidcserver.Endpoints{}, err
	}
	issuerURL, err := u.config.Get(ctx, common.ConfigOIDCIssuerURL)
	if err != nil {
		return oidcserver.ProviderConfig{}, oidcserver.Endpoints{}, err
	}
	if !enabled && strings.TrimSpace(issuerURL) == "" && strings.TrimSpace(origin) == "" {
		return oidcserver.ProviderConfig{Enabled: false}, oidcserver.Endpoints{}, nil
	}
	accessTTLValue, err := u.config.Get(ctx, common.ConfigOIDCAccessTokenTTLSeconds)
	if err != nil {
		return oidcserver.ProviderConfig{}, oidcserver.Endpoints{}, err
	}
	accessTTL, err := strconv.ParseInt(accessTTLValue, 10, 64)
	if err != nil {
		return oidcserver.ProviderConfig{}, oidcserver.Endpoints{}, err
	}
	idTTLValue, err := u.config.Get(ctx, common.ConfigOIDCIDTokenTTLSeconds)
	if err != nil {
		return oidcserver.ProviderConfig{}, oidcserver.Endpoints{}, err
	}
	idTTL, err := strconv.ParseInt(idTTLValue, 10, 64)
	if err != nil {
		return oidcserver.ProviderConfig{}, oidcserver.Endpoints{}, err
	}
	codeTTLValue, err := u.config.Get(ctx, common.ConfigOIDCAuthorizationCodeTTLSeconds)
	if err != nil {
		return oidcserver.ProviderConfig{}, oidcserver.Endpoints{}, err
	}
	codeTTL, err := strconv.ParseInt(codeTTLValue, 10, 64)
	if err != nil {
		return oidcserver.ProviderConfig{}, oidcserver.Endpoints{}, err
	}
	return oidcserver.NormalizeProviderConfig(oidcserver.ProviderConfig{
		Enabled:                     enabled,
		IssuerURL:                   issuerURL,
		AccessTokenTTLSeconds:       accessTTL,
		IDTokenTTLSeconds:           idTTL,
		AuthorizationCodeTTLSeconds: codeTTL,
	}, origin)
}

// signingKey 读取或初始化 OIDC RS256 签名私钥。
// 私钥只通过 JWKS 暴露对应公钥，不从管理接口返回。
func (u *OIDCUsecase) signingKey(ctx context.Context) (*rsa.PrivateKey, error) {
	raw, err := u.config.Get(ctx, common.ConfigOIDCPrivateKey)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(raw) == "" {
		raw, err = oidcserver.GeneratePrivateKeyPEM()
		if err != nil {
			return nil, err
		}
		if err := u.config.BatchUpdate(ctx, map[common.ConfigKey]string{common.ConfigOIDCPrivateKey: raw}); err != nil {
			return nil, err
		}
	}
	return oidcserver.ParsePrivateKeyPEM(raw)
}

// toOIDCClientInfo 控制 Client Secret 的返回范围。
func toOIDCClientInfo(client *gen.OIDCClient, includeSecret bool) *v1.OIDCClientInfo {
	info := &v1.OIDCClientInfo{
		Id:           client.ID,
		Name:         client.Name,
		ClientId:     client.ClientID,
		ClientSecret: oidcserver.MaskSecret(client.ClientSecret),
		RedirectUris: client.RedirectUris,
		Scopes:       client.Scopes,
		Active:       client.Active,
		CreateTime:   timestamppb.New(client.CreateTime),
		UpdateTime:   timestamppb.New(client.UpdateTime),
	}
	if includeSecret {
		info.ClientSecret = client.ClientSecret
	}
	return info
}

// toOIDCConfig 转换后台管理页所需的配置结构。
func toOIDCConfig(config oidcserver.ProviderConfig, endpoints oidcserver.Endpoints) *v1.OIDCConfig {
	return &v1.OIDCConfig{
		Enabled:                     config.Enabled,
		IssuerUrl:                   config.IssuerURL,
		AccessTokenTtlSeconds:       config.AccessTokenTTLSeconds,
		IdTokenTtlSeconds:           config.IDTokenTTLSeconds,
		AuthorizationCodeTtlSeconds: config.AuthorizationCodeTTLSeconds,
		Endpoints: &v1.OIDCEndpoints{
			ProviderName:              endpoints.ProviderName,
			IssuerUrl:                 endpoints.IssuerURL,
			DiscoveryUrl:              endpoints.DiscoveryURL,
			AuthorizeUrl:              endpoints.AuthorizeURL,
			TokenUrl:                  endpoints.TokenURL,
			UserinfoUrl:               endpoints.UserinfoURL,
			JwksUrl:                   endpoints.JWKSURL,
			Scopes:                    endpoints.Scopes,
			FrontendCallbackPath:      endpoints.FrontendCallbackPath,
			TokenEndpointAuthMethod:   endpoints.TokenEndpointAuthMethod,
			ClockSkewSeconds:          endpoints.ClockSkewSeconds,
			AllowedIdTokenSigningAlgs: endpoints.AllowedIDTokenSigningAlgs,
		},
	}
}

// toOIDCPkgClient 将 ent 模型转换为协议包使用的最小客户端结构。
func toOIDCPkgClient(client *gen.OIDCClient) oidcserver.Client {
	return oidcserver.Client{
		ID:           client.ID,
		ClientID:     client.ClientID,
		ClientSecret: client.ClientSecret,
		Name:         client.Name,
		RedirectURIs: client.RedirectUris,
		Scopes:       client.Scopes,
		Active:       client.Active,
	}
}

// toOIDCUserClaims 将内部用户信息转换为 OIDC claim。
// 这里只暴露 sub/name/preferred_username/email，不暴露角色和权限。
func toOIDCUserClaims(user *gen.User) oidcserver.UserClaims {
	return oidcserver.UserClaims{
		Subject:           user.ID,
		Name:              user.Name,
		PreferredUsername: user.Username,
		Email:             user.Email,
	}
}

// normalizeScopeList 保证 scopes 去重且包含 openid。
func normalizeScopeList(scopes []string) []string {
	return strings.Fields(oidcserver.NormalizeScopes(strings.Join(scopes, " ")))
}

// normalizeURIList 清理空回调地址。
func normalizeURIList(values []string) []string {
	result := make([]string, 0, len(values))
	for _, item := range values {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

// oidcErr 将协议包错误转成业务层可返回的错误。
// raw OIDC handler 会再把该错误转换成标准 OAuth JSON；管理 API 则沿用现有响应格式。
func oidcErr(err error) error {
	if err == nil {
		return nil
	}
	return response.BadRequest(500, fmt.Sprintf("OIDC 错误:%v", err))
}
