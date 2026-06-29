package oidcserver

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"math/big"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type UserClaims struct {
	Subject           string
	Name              string
	PreferredUsername string
	Email             string
}

type TokenClaims struct {
	Subject  string
	ClientID string
	Scope    string
	Expires  time.Time
}

// GeneratePrivateKeyPEM 生成 OIDC Provider 用于 RS256 签名的 RSA 私钥。
// 私钥由业务层保存到配置存储；pkg 层只负责密钥格式和签名细节。
func GeneratePrivateKeyPEM() (string, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return "", err
	}
	bytes := x509.MarshalPKCS1PrivateKey(key)
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: bytes}
	return string(pem.EncodeToMemory(block)), nil
}

// ParsePrivateKeyPEM 支持 PKCS#1 和 PKCS#8 两种 PEM 私钥格式。
func ParsePrivateKeyPEM(raw string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(raw))
	if block == nil {
		return nil, ErrSigningKeyUnavailable
	}
	key, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return key, nil
	}
	parsed, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, ErrSigningKeyUnavailable
	}
	rsaKey, ok := parsed.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrSigningKeyUnavailable
	}
	return rsaKey, nil
}

// KeyID 通过公钥参数生成稳定 kid，便于客户端缓存 JWKS。
func KeyID(key *rsa.PrivateKey) string {
	if key == nil {
		return ""
	}
	pub := key.PublicKey
	sum := sha256.Sum256(append(pub.N.Bytes(), big.NewInt(int64(pub.E)).Bytes()...))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// JWKS 输出 OIDC 客户端验证 ID Token 所需的公钥集合。
func JWKS(key *rsa.PrivateKey) map[string]any {
	pub := key.PublicKey
	return map[string]any{
		"keys": []map[string]any{
			{
				"kty": "RSA",
				"use": "sig",
				"kid": KeyID(key),
				"alg": "RS256",
				"n":   base64.RawURLEncoding.EncodeToString(pub.N.Bytes()),
				"e":   base64.RawURLEncoding.EncodeToString(big.NewInt(int64(pub.E)).Bytes()),
			},
		},
	}
}

// SignIDToken 签发标准 ID Token。
// claims 中只放用户身份和客户端需要的基础资料，避免暴露内部权限信息。
func SignIDToken(key *rsa.PrivateKey, issuer, clientID, nonce string, user UserClaims, ttl time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"iss":                issuer,
		"sub":                user.Subject,
		"aud":                clientID,
		"iat":                now.Unix(),
		"nbf":                now.Unix(),
		"exp":                now.Add(ttl).Unix(),
		"name":               user.Name,
		"preferred_username": user.PreferredUsername,
		"email":              user.Email,
	}
	if nonce != "" {
		claims["nonce"] = nonce
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = KeyID(key)
	return token.SignedString(key)
}

// SignAccessToken 签发访问 userinfo 端点的 Bearer Token。
func SignAccessToken(key *rsa.PrivateKey, issuer, clientID, scope string, user UserClaims, ttl time.Duration) (string, time.Time, error) {
	now := time.Now()
	expires := now.Add(ttl)
	jti, err := GenerateAccessTokenID()
	if err != nil {
		return "", time.Time{}, err
	}
	claims := jwt.MapClaims{
		"iss":   issuer,
		"sub":   user.Subject,
		"aud":   clientID,
		"iat":   now.Unix(),
		"nbf":   now.Unix(),
		"exp":   expires.Unix(),
		"jti":   jti,
		"typ":   "access_token",
		"scope": scope,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = KeyID(key)
	signed, err := token.SignedString(key)
	if err != nil {
		return "", time.Time{}, err
	}
	return signed, expires, nil
}

// ParseAccessToken 校验 userinfo 端点收到的 Bearer Token。
// 这里要求 RS256、issuer、过期时间和 typ 都正确，避免混用其他 JWT。
func ParseAccessToken(tokenString string, key *rsa.PrivateKey, issuer string) (TokenClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method.Alg() != jwt.SigningMethodRS256.Alg() {
			return nil, ErrTokenInvalid
		}
		return &key.PublicKey, nil
	}, jwt.WithIssuer(issuer), jwt.WithValidMethods([]string{jwt.SigningMethodRS256.Alg()}))
	if err != nil || !token.Valid {
		return TokenClaims{}, ErrTokenInvalid
	}
	if claims["typ"] != "access_token" {
		return TokenClaims{}, ErrTokenInvalid
	}
	expiresAt, err := claims.GetExpirationTime()
	if err != nil || expiresAt == nil || expiresAt.Before(time.Now()) {
		return TokenClaims{}, ErrTokenInvalid
	}
	subject, err := claims.GetSubject()
	if err != nil || subject == "" {
		return TokenClaims{}, ErrTokenInvalid
	}
	audience, err := claims.GetAudience()
	if err != nil || len(audience) == 0 {
		return TokenClaims{}, ErrTokenInvalid
	}
	scope, _ := claims["scope"].(string)
	return TokenClaims{
		Subject:  subject,
		ClientID: audience[0],
		Scope:    scope,
		Expires:  expiresAt.Time,
	}, nil
}
