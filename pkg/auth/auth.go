package auth

import (
	"context"
	"crypto/md5"
	"encoding/hex"

	"github.com/golang-jwt/jwt/v5"
)

// Auth user auth.
type Auth struct {
	UserID    string   `json:"id"`
	DeviceId  string   `json:"device_id"`
	SessionID string   `json:"session_id"`
	Type      AuthType `json:"type"`
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

func EncodePassword(password string) string {
	sum := md5.Sum([]byte(password))
	return hex.EncodeToString(sum[:])
}

func GetUserIDFromContext(ctx context.Context) *string {
	auth, ok := FromContext(ctx)
	if !ok {
		return nil
	}
	return &auth.UserID
}
