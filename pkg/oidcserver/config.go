package oidcserver

import (
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	// DefaultProviderName 是展示给外部客户端的默认 Provider 名称。
	DefaultProviderName                = "Momoko"
	DefaultScope                       = "openid email profile"
	DefaultFrontendCallbackPath        = "/auth/oidc/callback"
	DefaultTokenEndpointAuthMethod     = "client_secret_post"
	DefaultAllowedIDTokenSigningAlgs   = "RS256"
	DefaultClockSkewSeconds            = 120
	DefaultAccessTokenTTLSeconds       = int64(3600)
	DefaultIDTokenTTLSeconds           = int64(3600)
	DefaultAuthorizationCodeTTLSeconds = int64(300)
)

type ProviderConfig struct {
	Enabled                     bool
	IssuerURL                   string
	AccessTokenTTLSeconds       int64
	IDTokenTTLSeconds           int64
	AuthorizationCodeTTLSeconds int64
}

type Endpoints struct {
	ProviderName              string
	IssuerURL                 string
	DiscoveryURL              string
	AuthorizeURL              string
	TokenURL                  string
	UserinfoURL               string
	JWKSURL                   string
	Scopes                    string
	FrontendCallbackPath      string
	TokenEndpointAuthMethod   string
	ClockSkewSeconds          int64
	AllowedIDTokenSigningAlgs string
}

// NormalizeProviderConfig 归一化 OIDC Provider 配置。
//
// 这里属于协议层逻辑，所以放在 pkg 内：
// 1. issuer 没有显式配置时，使用传入的请求 origin 推导；
// 2. 各类 TTL 使用协议服务端的默认值；
// 3. issuer 必须是干净的 http/https origin，不能带 query/fragment。
func NormalizeProviderConfig(config ProviderConfig, origin string) (ProviderConfig, Endpoints, error) {
	config.IssuerURL = strings.TrimRight(strings.TrimSpace(config.IssuerURL), "/")
	if config.IssuerURL == "" {
		config.IssuerURL = strings.TrimRight(strings.TrimSpace(origin), "/")
	}
	if err := validateIssuer(config.IssuerURL); err != nil {
		return config, Endpoints{}, err
	}
	if config.AccessTokenTTLSeconds <= 0 {
		config.AccessTokenTTLSeconds = DefaultAccessTokenTTLSeconds
	}
	if config.IDTokenTTLSeconds <= 0 {
		config.IDTokenTTLSeconds = DefaultIDTokenTTLSeconds
	}
	if config.AuthorizationCodeTTLSeconds <= 0 {
		config.AuthorizationCodeTTLSeconds = DefaultAuthorizationCodeTTLSeconds
	}
	return config, BuildEndpoints(config.IssuerURL), nil
}

// BuildEndpoints 根据 issuer 生成 Discovery 中需要公开的标准端点。
// 管理后台只展示这些值，实际协议响应也复用同一份生成逻辑。
func BuildEndpoints(issuer string) Endpoints {
	issuer = strings.TrimRight(strings.TrimSpace(issuer), "/")
	return Endpoints{
		ProviderName:              DefaultProviderName,
		IssuerURL:                 issuer,
		DiscoveryURL:              issuer + "/.well-known/openid-configuration",
		AuthorizeURL:              issuer + "/oidc/authorize",
		TokenURL:                  issuer + "/api/v1/oidc/token",
		UserinfoURL:               issuer + "/api/v1/oidc/userinfo",
		JWKSURL:                   issuer + "/api/v1/oidc/jwks",
		Scopes:                    DefaultScope,
		FrontendCallbackPath:      DefaultFrontendCallbackPath,
		TokenEndpointAuthMethod:   DefaultTokenEndpointAuthMethod,
		ClockSkewSeconds:          DefaultClockSkewSeconds,
		AllowedIDTokenSigningAlgs: DefaultAllowedIDTokenSigningAlgs,
	}
}

// Discovery 生成 OpenID Provider Metadata。
// 返回 map 是为了让标准 JSON 字段名保持 snake_case，不受 Go/Proto 命名影响。
func Discovery(config ProviderConfig, endpoints Endpoints) map[string]any {
	return map[string]any{
		"issuer":                                endpoints.IssuerURL,
		"authorization_endpoint":                endpoints.AuthorizeURL,
		"token_endpoint":                        endpoints.TokenURL,
		"userinfo_endpoint":                     endpoints.UserinfoURL,
		"jwks_uri":                              endpoints.JWKSURL,
		"response_types_supported":              []string{"code"},
		"subject_types_supported":               []string{"public"},
		"id_token_signing_alg_values_supported": []string{"RS256"},
		"token_endpoint_auth_methods_supported": []string{"client_secret_basic", "client_secret_post"},
		"grant_types_supported":                 []string{"authorization_code"},
		"scopes_supported":                      []string{"openid", "email", "profile"},
		"claims_supported":                      []string{"sub", "name", "preferred_username", "email"},
		"code_challenge_methods_supported":      []string{"plain", "S256"},
	}
}

// AccessTokenTTL 返回 Access Token 有效期。
func AccessTokenTTL(config ProviderConfig) time.Duration {
	if config.AccessTokenTTLSeconds <= 0 {
		return time.Duration(DefaultAccessTokenTTLSeconds) * time.Second
	}
	return time.Duration(config.AccessTokenTTLSeconds) * time.Second
}

// IDTokenTTL 返回 ID Token 有效期。
func IDTokenTTL(config ProviderConfig) time.Duration {
	if config.IDTokenTTLSeconds <= 0 {
		return time.Duration(DefaultIDTokenTTLSeconds) * time.Second
	}
	return time.Duration(config.IDTokenTTLSeconds) * time.Second
}

// AuthorizationCodeTTL 返回授权码有效期。
func AuthorizationCodeTTL(config ProviderConfig) time.Duration {
	if config.AuthorizationCodeTTLSeconds <= 0 {
		return time.Duration(DefaultAuthorizationCodeTTLSeconds) * time.Second
	}
	return time.Duration(config.AuthorizationCodeTTLSeconds) * time.Second
}

// validateIssuer 限制 issuer 为 origin 级地址。
// OIDC 客户端会把 issuer 作为信任根，允许携带路径/query 会增加配置歧义。
func validateIssuer(raw string) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return ErrInvalidIssuer
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return ErrInvalidIssuer
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return ErrInvalidIssuer
	}
	if u.Path != "" && u.Path != "/" {
		return ErrInvalidIssuer
	}
	return nil
}

// RequestOrigin 从请求中推导当前服务对外 origin。
// 反向代理场景优先读取 X-Forwarded-*，未设置时使用当前请求的 scheme/host。
func RequestOrigin(r *http.Request) string {
	if r == nil {
		return ""
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	if forwardedProto := strings.TrimSpace(r.Header.Get("X-Forwarded-Proto")); forwardedProto != "" {
		scheme = strings.Split(forwardedProto, ",")[0]
	}
	host := r.Host
	if forwardedHost := strings.TrimSpace(r.Header.Get("X-Forwarded-Host")); forwardedHost != "" {
		host = strings.Split(forwardedHost, ",")[0]
	}
	return scheme + "://" + strings.TrimSpace(host)
}
