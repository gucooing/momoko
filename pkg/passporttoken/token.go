// Package passporttoken 用户通行证（玩家端）的登录令牌：HS256 JWT，签名密钥由主密钥按独立的域派生，
// 签发者、受众与主题都和 Momoko 管理后台的令牌不同，所以两边的令牌在结构上互不认（不需要任何交叉检查）。
//
// 令牌只携带会话标识与一份会话噪声；服务端每次校验时还要和 passport_sessions 里的噪声比对，
// 刷新或下线即令旧令牌失效。
package passporttoken

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	// Issuer / Audience / Subject 通行证令牌的固定声明（管理后台令牌是 iss=momoko、sub=user）。
	Issuer   = "momoko-passport"
	Audience = "passport"
	Subject  = "passport_user"

	signingDomain = "momoko:passport:jwt:hs256:v1"
	noiseBytes    = 32
)

// Kind 区分访问令牌与刷新令牌。
type Kind string

const (
	KindAccess  Kind = "access"
	KindRefresh Kind = "refresh"
)

// ErrInvalid 令牌无效（签名、声明、过期或格式不对）。
var ErrInvalid = errors.New("passport token invalid")

// Claims 通行证令牌的声明。
type Claims struct {
	UserID    string `json:"uid"`
	SessionID string `json:"sid"`
	DeviceID  string `json:"did"`
	Kind      Kind   `json:"kind"`
	// Noise 会话噪声：access / refresh 各一份，和库里的值比对。
	Noise string `json:"noise"`
	jwt.RegisteredClaims
}

// signingKey 从主密钥派生通行证专用的签名子密钥（与管理后台 JWT、静态数据加密的子密钥都不同）。
func signingKey(secret string) []byte {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(signingDomain))
	return mac.Sum(nil)
}

// NewNoise 生成会话噪声（url-safe base64，无填充）。
func NewNoise() (string, error) {
	buf := make([]byte, noiseBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// Sign 签发一枚令牌。
func Sign(secret, userID, sessionID, deviceID string, kind Kind, noise string, ttl time.Duration) (string, error) {
	if userID == "" || sessionID == "" || deviceID == "" || noise == "" || (kind != KindAccess && kind != KindRefresh) {
		return "", ErrInvalid
	}
	now := time.Now()
	claims := Claims{
		UserID:    userID,
		SessionID: sessionID,
		DeviceID:  deviceID,
		Kind:      kind,
		Noise:     noise,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        sessionID + ":" + string(kind) + ":" + noise[:min(8, len(noise))],
			Issuer:    Issuer,
			Subject:   Subject,
			Audience:  jwt.ClaimStrings{Audience},
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ttl)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(signingKey(secret))
}

// Parse 校验签名与声明（算法、签发者、受众、主题、过期），并要求会话字段齐全。
func Parse(secret, token string) (*Claims, error) {
	if token == "" {
		return nil, ErrInvalid
	}
	parsed, err := jwt.ParseWithClaims(token, &Claims{}, func(*jwt.Token) (any, error) {
		return signingKey(secret), nil
	},
		jwt.WithExpirationRequired(),
		jwt.WithIssuer(Issuer),
		jwt.WithSubject(Subject),
		jwt.WithAudience(Audience),
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)
	if err != nil || !parsed.Valid {
		return nil, ErrInvalid
	}
	c, ok := parsed.Claims.(*Claims)
	if !ok || c.UserID == "" || c.SessionID == "" || c.DeviceID == "" || c.Noise == "" ||
		(c.Kind != KindAccess && c.Kind != KindRefresh) {
		return nil, ErrInvalid
	}
	return c, nil
}

type ctxKey struct{}

// NewContext 把已校验的通行证身份放进请求上下文。
func NewContext(ctx context.Context, c *Claims) context.Context {
	return context.WithValue(ctx, ctxKey{}, c)
}

// FromContext 读取请求上下文里的通行证身份。
func FromContext(ctx context.Context) (*Claims, bool) {
	c, ok := ctx.Value(ctxKey{}).(*Claims)
	return c, ok && c != nil
}
