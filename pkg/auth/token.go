package auth

import (
	"time"

	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/auth"
	"momoko/pkg/response"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenExpiresIn  = time.Hour
	RefreshTokenExpiresIn = 14 * 24 * time.Hour
	tokenIssuer           = "momoko"
	tokenSubject          = "user"
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
			Issuer:    tokenIssuer,
			Subject:   tokenSubject,
			IssuedAt:  jwt.NewNumericDate(authDb.CreateTime),
			NotBefore: jwt.NewNumericDate(authDb.UpdateTime),
			ExpiresAt: jwt.NewNumericDate(authDb.ExpiresAt),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(AuthSecretKey))
}

func TokenExpiresIn(tokenType auth.Type) time.Duration {
	switch tokenType {
	case auth.TypeRefreshToken:
		return RefreshTokenExpiresIn
	default:
		return AccessTokenExpiresIn
	}
}

// ParseToken parses the JWT token string and returns the Auth claims.
func ParseToken(tokenStr string) (*Auth, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Auth{Type: AuthTypeJWT}, func(token *jwt.Token) (interface{}, error) {
		return []byte(AuthSecretKey), nil
	},
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithSubject(tokenSubject),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
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
