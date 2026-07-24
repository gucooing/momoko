package auth

import (
	"context"

	"github.com/golang-jwt/jwt/v5"
)

// TokenKind 区分 access / refresh JWT（同一会话只占一行 auth 记录）。
type TokenKind string

const (
	TokenKindAccess  TokenKind = "access"
	TokenKindRefresh TokenKind = "refresh"
)

// Auth user auth claims.
type Auth struct {
	UserID    string    `json:"id"`
	DeviceId  string    `json:"device_id"`
	SessionID string    `json:"session_id"`
	Kind      TokenKind `json:"kind"`
	// Noise 会话级随机噪声：access / refresh 各用一份，写入 JWT 并与库内值比对。
	Noise string `json:"noise"`
	// Type 仅用于区分 JWT 与 API Key 等鉴权载体，不再表示 access/refresh。
	Type AuthType `json:"type"`
	jwt.RegisteredClaims
}

type AuthType int

const (
	AuthTypeJWT AuthType = iota
	AuthTypeApiKey
)

type authKey struct{}

// NewContext returns a new Context that carries value.
func NewContext(ctx context.Context, auth *Auth) context.Context {
	return context.WithValue(ctx, authKey{}, auth)
}

// FromContext returns the Auth value stored in ctx, if any.
func FromContext(ctx context.Context) (auth *Auth, ok bool) {
	auth, ok = ctx.Value(authKey{}).(*Auth)
	return
}

func GetUserIDFromContext(ctx context.Context) *string {
	auth, ok := FromContext(ctx)
	if !ok {
		return nil
	}
	return &auth.UserID
}
