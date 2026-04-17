package auth

import (
	"time"

	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/auth"
	"momoko/pkg/response"

	"github.com/golang-jwt/jwt/v5"
)

const (
	tokenExpiresAt = 2 * time.Hour
)

var (
	AuthSecretKey   = "123456"
	ErrTokenInvalid = response.BadRequest(401, "token invalid")
)

func GenerateToken(authDb *gen.Auth) (string, error) {
	if authDb == nil {
		return "", ErrTokenInvalid
	}
	claims := Auth{
		UserID:    authDb.UserID,
		DeviceId:  authDb.DeviceID,
		SessionID: authDb.SessionID,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        authDb.SessionID,
			Issuer:    "kratos",
			Subject:   "user",
			IssuedAt:  jwt.NewNumericDate(authDb.CreateTime),
			NotBefore: jwt.NewNumericDate(authDb.CreateTime),
			ExpiresAt: jwt.NewNumericDate(authDb.UpdateTime.Add(func() time.Duration {
				switch authDb.Type {
				case auth.TypeToken:
					return tokenExpiresAt
				default:
					return 0
				}
			}())),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(AuthSecretKey))
}

// ParseToken parses the JWT token string and returns the Auth claims.
func ParseToken(tokenStr string) (*Auth, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Auth{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(AuthSecretKey), nil
	}, jwt.WithoutClaimsValidation())
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, ErrTokenInvalid
	}
	claims, ok := token.Claims.(*Auth)
	if !ok {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}
