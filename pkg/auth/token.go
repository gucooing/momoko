package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"momoko/internal/data/ent/gen"
	"momoko/pkg/response"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTokenExpiresIn  = time.Hour
	RefreshTokenExpiresIn = 14 * 24 * time.Hour
	tokenIssuer           = "momoko"
	tokenSubject          = "user"

	// noiseBytes 会话噪声原始字节长度（编码后写入 JWT / DB）。
	noiseBytes = 32

	// minSecretLength 主密钥最小长度。
	minSecretLength = 8
)

var (
	AuthSecretKey   = "123456"
	ErrTokenInvalid = response.BadRequest(401, "token invalid")
)

// ValidateSecret
// 应在启动认证服务前调用，弱密钥直接 fail-closed。
func ValidateSecret(secret string) error {
	if secret == "" {
		return errors.New("auth.secret 未配置：请在配置中设置一个高强度随机密钥（建议 ≥32 字符）")
	}
	if len(secret) < minSecretLength {
		return fmt.Errorf("auth.secret 过短（至少 %d 字符）：请改用高强度随机密钥", minSecretLength)
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

// NewNoise 生成会话级随机噪声（url-safe base64，无填充）。
func NewNoise() (string, error) {
	buf := make([]byte, noiseBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// GenerateAccessToken 签发 access JWT；noise 取会话的 AccessNoise。
func GenerateAccessToken(session *gen.Session) (string, error) {
	return generateToken(session, TokenKindAccess, AccessTokenExpiresIn)
}

// GenerateRefreshToken 签发 refresh JWT；noise 取会话的 RefreshNoise。
func GenerateRefreshToken(session *gen.Session) (string, error) {
	return generateToken(session, TokenKindRefresh, RefreshTokenExpiresIn)
}

func generateToken(session *gen.Session, kind TokenKind, ttl time.Duration) (string, error) {
	if session == nil {
		return "", ErrTokenInvalid
	}
	if session.UserID == "" || session.DeviceID == "" || session.ID == "" {
		return "", ErrTokenInvalid
	}

	noise := ""
	switch kind {
	case TokenKindAccess:
		noise = session.AccessNoise
	case TokenKindRefresh:
		noise = session.RefreshNoise
	default:
		return "", ErrTokenInvalid
	}

	now := time.Now()
	// jti 混入 kind + noise 前缀，使同一会话下 access/refresh 的签名输入明显不同。
	jti := session.ID + ":" + string(kind) + ":" + noise[:min(8, len(noise))]

	claims := Auth{
		UserID:    session.UserID,
		DeviceId:  session.DeviceID,
		SessionID: session.ID,
		Kind:      kind,
		Noise:     noise,
		Type:      AuthTypeJWT,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Issuer:    tokenIssuer,
			Subject:   tokenSubject,
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(jwtSigningKey())
}

// ParseToken 校验签名与标准 claims（iss/sub/exp/方法），并要求 kind/noise 非空。
func ParseToken(tokenStr string) (*Auth, error) {
	if tokenStr == "" {
		return nil, ErrTokenInvalid
	}

	token, err := jwt.ParseWithClaims(tokenStr, &Auth{Type: AuthTypeJWT}, func(token *jwt.Token) (any, error) {
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
	if !ok ||
		claims.UserID == "" ||
		claims.DeviceId == "" ||
		claims.SessionID == "" ||
		claims.Noise == "" ||
		(claims.Kind != TokenKindAccess && claims.Kind != TokenKindRefresh) {
		return nil, ErrTokenInvalid
	}
	return claims, nil
}
