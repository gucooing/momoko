package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
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

	// minSecretLength 主密钥最小长度。
	minSecretLength = 8
)

var (
	AuthSecretKey   = "123456"
	ErrTokenInvalid = response.BadRequest(401, "token invalid")
)

// weakSecrets 是已知的弱/默认/公开密钥，禁止用于生产环境的签名与加密。
var weakSecrets = map[string]struct{}{
	"123456":        {},
	"gucooing.auth": {},
	"momoko":        {},
	"secret":        {},
	"changeme":      {},
	"password":      {},
	"admin":         {},
}

// ValidateSecret 校验主密钥强度。空、过短或命中已知弱密钥都会被拒绝。
// 应在启动认证服务前调用，弱密钥直接 fail-closed。
func ValidateSecret(secret string) error {
	trimmed := strings.TrimSpace(secret)
	if trimmed == "" {
		return errors.New("auth.secret 未配置：请在配置中设置一个高强度随机密钥（建议 ≥32 字符）")
	}
	if len(trimmed) < minSecretLength {
		return fmt.Errorf("auth.secret 过短（至少 %d 字符）：请改用高强度随机密钥", minSecretLength)
	}
	if _, ok := weakSecrets[strings.ToLower(trimmed)]; ok {
		return errors.New("auth.secret 使用了已知弱/默认密钥：请改用高强度随机密钥")
	}
	return nil
}

// jwtSigningKey 从主密钥派生独立的 JWT 签名子密钥（HMAC-SHA256 域分离），
// 与 secretbox 的静态数据加密密钥（sha256(secret)）做密钥分离，避免同一份密钥
// 既签名 JWT 又加密数据。轮换主密钥会同时失效已签发的 JWT（属预期行为）。
func jwtSigningKey() []byte {
	mac := hmac.New(sha256.New, []byte(AuthSecretKey))
	mac.Write([]byte("momoko:jwt:hs256:v1"))
	return mac.Sum(nil)
}

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
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSigningKey())
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
		return jwtSigningKey(), nil
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
