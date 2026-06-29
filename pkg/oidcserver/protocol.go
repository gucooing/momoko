package oidcserver

import (
	"encoding/base64"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	ID           string
	ClientID     string
	ClientSecret string
	Name         string
	RedirectURIs []string
	Scopes       []string
	Active       bool
}

// AuthorizationCode 是授权码交换 token 所需的最小状态。
type AuthorizationCode struct {
	Code                string
	ClientID            string
	UserID              string
	RedirectURI         string
	Scope               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
	ExpiresAt           time.Time
}

type AuthorizeRequest struct {
	ResponseType        string
	ClientID            string
	RedirectURI         string
	Scope               string
	State               string
	Nonce               string
	CodeChallenge       string
	CodeChallengeMethod string
}

type TokenRequest struct {
	GrantType    string
	Code         string
	RedirectURI  string
	ClientID     string
	ClientSecret string
	CodeVerifier string
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
	IDToken     string `json:"id_token"`
	Scope       string `json:"scope,omitempty"`
}

type UserinfoResponse struct {
	Subject           string `json:"sub"`
	Name              string `json:"name,omitempty"`
	PreferredUsername string `json:"preferred_username,omitempty"`
	Email             string `json:"email,omitempty"`
}

// NewAuthorizeRequest 从 query 参数中提取标准授权请求。
func NewAuthorizeRequest(values url.Values) AuthorizeRequest {
	return AuthorizeRequest{
		ResponseType:        strings.TrimSpace(values.Get("response_type")),
		ClientID:            strings.TrimSpace(values.Get("client_id")),
		RedirectURI:         strings.TrimSpace(values.Get("redirect_uri")),
		Scope:               strings.TrimSpace(values.Get("scope")),
		State:               strings.TrimSpace(values.Get("state")),
		Nonce:               strings.TrimSpace(values.Get("nonce")),
		CodeChallenge:       strings.TrimSpace(values.Get("code_challenge")),
		CodeChallengeMethod: strings.TrimSpace(values.Get("code_challenge_method")),
	}
}

// NewTokenRequest 从表单和 Basic Auth 中提取 token 请求。
// OIDC 常见客户端会使用 client_secret_basic 或 client_secret_post，两者都支持。
func NewTokenRequest(r *http.Request) (TokenRequest, error) {
	if err := r.ParseForm(); err != nil {
		return TokenRequest{}, ErrInvalidRequest
	}
	clientID, clientSecret := parseClientAuth(r)
	if clientID == "" {
		clientID = r.Form.Get("client_id")
	}
	if clientSecret == "" {
		clientSecret = r.Form.Get("client_secret")
	}
	return TokenRequest{
		GrantType:    strings.TrimSpace(r.Form.Get("grant_type")),
		Code:         strings.TrimSpace(r.Form.Get("code")),
		RedirectURI:  strings.TrimSpace(r.Form.Get("redirect_uri")),
		ClientID:     strings.TrimSpace(clientID),
		ClientSecret: strings.TrimSpace(clientSecret),
		CodeVerifier: strings.TrimSpace(r.Form.Get("code_verifier")),
	}, nil
}

// ValidateAuthorizeRequest 校验授权端点参数和客户端配置是否匹配。
func ValidateAuthorizeRequest(req AuthorizeRequest, client Client) error {
	if req.ResponseType != "code" {
		return ErrUnsupportedResponse
	}
	if req.ClientID == "" || req.RedirectURI == "" {
		return ErrInvalidRequest
	}
	if !client.Active || req.ClientID != client.ClientID {
		return ErrInvalidClient
	}
	if !hasString(client.RedirectURIs, req.RedirectURI) {
		return ErrInvalidRedirectURI
	}
	scope := NormalizeScopes(req.Scope)
	if !hasString(strings.Fields(scope), "openid") {
		return ErrInvalidScope
	}
	for _, requested := range strings.Fields(scope) {
		if !hasString(client.Scopes, requested) {
			return ErrInvalidScope
		}
	}
	return nil
}

// ValidateTokenRequest 校验授权码交换请求。
// 授权码、客户端、redirect_uri、secret、PKCE 必须全部匹配。
func ValidateTokenRequest(req TokenRequest, client Client, code AuthorizationCode) error {
	if req.GrantType != "authorization_code" {
		return ErrUnsupportedGrant
	}
	if !client.Active || req.ClientID != client.ClientID {
		return ErrInvalidClient
	}
	if !VerifySecret(client.ClientSecret, req.ClientSecret) {
		return ErrInvalidClientSecret
	}
	if code.Code == "" || code.ClientID != client.ClientID || code.ExpiresAt.Before(time.Now()) {
		return ErrInvalidCode
	}
	if code.RedirectURI != req.RedirectURI {
		return ErrInvalidRedirectURI
	}
	if !VerifyPKCE(code.CodeChallenge, code.CodeChallengeMethod, req.CodeVerifier) {
		return ErrInvalidPKCE
	}
	return nil
}

// NormalizeScopes 去重并保证 openid 存在。
func NormalizeScopes(scope string) string {
	seen := map[string]struct{}{}
	values := strings.Fields(scope)
	if len(values) == 0 {
		values = strings.Fields(DefaultScope)
	}
	result := make([]string, 0, len(values))
	for _, item := range values {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	if !hasString(result, "openid") {
		result = append([]string{"openid"}, result...)
	}
	return strings.Join(result, " ")
}

// BuildAuthorizeRedirect 将授权码和 state 安全拼到客户端回调地址。
func BuildAuthorizeRedirect(redirectURI, code, state string) string {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return redirectURI
	}
	query := u.Query()
	query.Set("code", code)
	if state != "" {
		query.Set("state", state)
	}
	u.RawQuery = query.Encode()
	return u.String()
}

// SplitList 将用户输入的多行/逗号/空格分隔文本转为列表。
func SplitList(value string) []string {
	values := strings.FieldsFunc(value, func(r rune) bool {
		return r == '\n' || r == '\r' || r == ',' || r == ' '
	})
	result := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, item := range values {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		result = append(result, item)
	}
	return result
}

// ValidRedirectURIs 校验回调地址必须是 http/https URL，且不能带 fragment。
func ValidRedirectURIs(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, item := range values {
		u, err := url.Parse(strings.TrimSpace(item))
		if err != nil || u.Scheme == "" || u.Host == "" {
			return false
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return false
		}
		if u.Fragment != "" {
			return false
		}
	}
	return true
}

// parseClientAuth 解析 token 端点的客户端认证信息。
func parseClientAuth(r *http.Request) (string, string) {
	username, password, ok := r.BasicAuth()
	if ok {
		return username, password
	}
	header := strings.TrimSpace(r.Header.Get("Authorization"))
	if !strings.HasPrefix(header, "Basic ") {
		return "", ""
	}
	decoded, err := base64.StdEncoding.DecodeString(strings.TrimSpace(strings.TrimPrefix(header, "Basic ")))
	if err != nil {
		return "", ""
	}
	parts := strings.SplitN(string(decoded), ":", 2)
	if len(parts) != 2 {
		return "", ""
	}
	return parts[0], parts[1]
}

func hasString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}
