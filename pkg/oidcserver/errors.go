package oidcserver

import "errors"

var (
	ErrDisabled              = errors.New("OIDC 服务未启用")
	ErrInvalidIssuer         = errors.New("Issuer URL 必须是 http/https 域名地址")
	ErrInvalidRequest        = errors.New("OIDC 请求参数不正确")
	ErrInvalidClient         = errors.New("OIDC 客户端无效")
	ErrInvalidClientSecret   = errors.New("OIDC Client Secret 无效")
	ErrInvalidRedirectURI    = errors.New("OIDC Redirect URI 无效")
	ErrInvalidScope          = errors.New("OIDC Scope 必须包含 openid")
	ErrUnsupportedResponse   = errors.New("OIDC response_type 仅支持 code")
	ErrUnsupportedGrant      = errors.New("OIDC grant_type 仅支持 authorization_code")
	ErrInvalidCode           = errors.New("OIDC 授权码无效或已过期")
	ErrInvalidPKCE           = errors.New("OIDC PKCE 校验失败")
	ErrSigningKeyUnavailable = errors.New("OIDC 签名密钥不可用")
	ErrTokenInvalid          = errors.New("OIDC token 无效")
)
