package auth

import (
	"momoko/internal/data/ent"
	"momoko/internal/data/ent/auth"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	tokenExpiresAt        = 2 * time.Hour
	refreshTokenExpiresAt = 7 * 24 * time.Hour
)

func GenerateToken(authDb *ent.Auth, secret string) (string, error) {
	claims := Auth{
		UserID: authDb.UserID,
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
				case auth.TypeRefreshToken:
					return refreshTokenExpiresAt
				default:
					return 0
				}
			}())),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
}

// ParseToken parses the JWT token string and returns the Auth claims.
func ParseToken(tokenStr, secret string) (*Auth, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Auth{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(secret), nil
	})
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, ErrUnauthorized
	}
	claims, ok := token.Claims.(*Auth)
	if !ok {
		return nil, ErrUnauthorized
	}
	return claims, nil
}
