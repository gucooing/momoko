// Package nodeauth 实现主控调用节点 gRPC 时的 API Key 时间戳签名鉴权。
//
// API Key 只是签名种子，不在网络上传输。主控每次调用在 gRPC metadata 中携带：
//
//	x-momoko-key-id     由种子派生的公开密钥标识，节点据此定位密钥
//	x-momoko-timestamp  Unix 秒级时间戳
//	x-momoko-nonce      一次性随机数
//	x-momoko-operator   主控侧操作者（URL 编码，用于节点审计）
//	x-momoko-signature  base64url(HMAC-SHA256(种子, 规范串))
//
// 规范串 = Algorithm \n key-id \n timestamp \n nonce \n gRPC 完整方法名 \n operator。
// 节点校验时间戳在 ±MaxClockSkew 内、签名一致，并在窗口内拒绝重复 nonce；
// 截获的签名既不能换方法使用，也不能在窗口内重放，过期后自然失效。
package nodeauth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/url"
	"strconv"
	"strings"
	"time"

	"google.golang.org/grpc/metadata"
)

const (
	// Algorithm 签名算法标识，同时作为规范串首行做域分离。
	Algorithm = "MOMOKO-HMAC-SHA256"

	MetadataKeyID     = "x-momoko-key-id"
	MetadataTimestamp = "x-momoko-timestamp"
	MetadataNonce     = "x-momoko-nonce"
	MetadataOperator  = "x-momoko-operator"
	MetadataSignature = "x-momoko-signature"

	// MaxClockSkew 允许的主控与节点时钟偏差；nonce 防重放窗口与之相同。
	MaxClockSkew = 5 * time.Minute

	keyIDPrefix = "mkid-"
	nonceBytes  = 16
)

var (
	ErrMissingCredential = errors.New("缺少节点签名凭据")
	ErrClockSkew         = errors.New("请求时间戳超出允许范围，请同步主控与节点的系统时间")
	ErrBadSignature      = errors.New("签名校验失败")
	ErrReplay            = errors.New("请求已被使用（疑似重放）")
)

// KeyID 由种子派生公开密钥标识：可公开展示与传输，无法反推种子。
func KeyID(seed string) string {
	sum := sha256.Sum256([]byte("momoko:api-key:id:v1:" + seed))
	return keyIDPrefix + hex.EncodeToString(sum[:12])
}

func canonical(keyID string, timestamp int64, nonce, method, operator string) string {
	return strings.Join([]string{
		Algorithm,
		keyID,
		strconv.FormatInt(timestamp, 10),
		nonce,
		method,
		operator,
	}, "\n")
}

// Sign 计算一次调用的签名。
func Sign(seed, keyID string, timestamp int64, nonce, method, operator string) string {
	mac := hmac.New(sha256.New, []byte(seed))
	mac.Write([]byte(canonical(keyID, timestamp, nonce, method, operator)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func newNonce() (string, error) {
	raw := make([]byte, nonceBytes)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

type operatorKey struct{}

// WithOperator 在调用上下文中记录主控侧操作者，签名时写入 metadata 供节点审计。
func WithOperator(ctx context.Context, operator string) context.Context {
	return context.WithValue(ctx, operatorKey{}, operator)
}

// OperatorFromContext 返回 WithOperator 记录的操作者。
func OperatorFromContext(ctx context.Context) string {
	operator, _ := ctx.Value(operatorKey{}).(string)
	return operator
}

// SignedMetadata 生成一次调用所需的签名 metadata（键值对，可直接追加到 outgoing context）。
func SignedMetadata(ctx context.Context, seed, method string, now time.Time) ([]string, error) {
	nonce, err := newNonce()
	if err != nil {
		return nil, err
	}
	keyID := KeyID(seed)
	timestamp := now.Unix()
	operator := url.QueryEscape(OperatorFromContext(ctx))
	return []string{
		MetadataKeyID, keyID,
		MetadataTimestamp, strconv.FormatInt(timestamp, 10),
		MetadataNonce, nonce,
		MetadataOperator, operator,
		MetadataSignature, Sign(seed, keyID, timestamp, nonce, method, operator),
	}, nil
}

// Signed 是节点从 metadata 解析出的签名凭据。
type Signed struct {
	KeyID     string
	Timestamp int64
	Nonce     string
	Signature string
	// operator 保留原始（URL 编码）值参与验签。
	operator string
}

// Operator 返回解码后的主控侧操作者。
func (s *Signed) Operator() string {
	operator, err := url.QueryUnescape(s.operator)
	if err != nil {
		return ""
	}
	return operator
}

// ParseMetadata 从 incoming metadata 解析签名凭据；字段缺失返回 ErrMissingCredential。
func ParseMetadata(md metadata.MD) (*Signed, error) {
	get := func(key string) string {
		if values := md.Get(key); len(values) > 0 {
			return values[0]
		}
		return ""
	}
	s := &Signed{
		KeyID:     get(MetadataKeyID),
		Nonce:     get(MetadataNonce),
		Signature: get(MetadataSignature),
		operator:  get(MetadataOperator),
	}
	rawTimestamp := get(MetadataTimestamp)
	if s.KeyID == "" || s.Nonce == "" || s.Signature == "" || rawTimestamp == "" {
		return nil, ErrMissingCredential
	}
	timestamp, err := strconv.ParseInt(rawTimestamp, 10, 64)
	if err != nil {
		return nil, ErrMissingCredential
	}
	s.Timestamp = timestamp
	return s, nil
}

// CheckTime 校验时间戳是否在允许偏差内（可在查库前先做，廉价拒绝过期请求）。
func (s *Signed) CheckTime(now time.Time) error {
	skew := now.Sub(time.Unix(s.Timestamp, 0))
	if skew > MaxClockSkew || skew < -MaxClockSkew {
		return ErrClockSkew
	}
	return nil
}

// Verify 用种子校验签名（常量时间比较）与时间窗口；method 为节点实际收到的 gRPC 完整方法名。
func (s *Signed) Verify(seed, method string, now time.Time) error {
	if err := s.CheckTime(now); err != nil {
		return err
	}
	if KeyID(seed) != s.KeyID {
		return ErrBadSignature
	}
	expected := Sign(seed, s.KeyID, s.Timestamp, s.Nonce, method, s.operator)
	if !hmac.Equal([]byte(expected), []byte(s.Signature)) {
		return ErrBadSignature
	}
	return nil
}
