// Package mgpauth 实现 MGP v1（Momoko Game Protocol）请求签名，与节点 API Key 的签名方案同源：
// 密钥不在网络上传输，每个请求携带
//
//	X-MGP-Key        key_id
//	X-MGP-Timestamp  Unix 秒
//	X-MGP-Nonce      一次性随机串（最长 128 字符）
//	X-MGP-Signature  base64url（无填充）的 HMAC-SHA256(secret, 规范串)
//
// 规范串 = Algorithm \n key_id \n 时间戳 \n nonce \n HTTP 方法 \n 请求 URI（路径 + 原始查询串）\n body 的 SHA-256 十六进制。
// 服务端校验时间戳在 ±MaxClockSkew 内，并在窗口内拒绝重复 nonce（复用 nodeauth.ReplayGuard）。
package mgpauth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"momoko/pkg/nodeauth"
)

const (
	Algorithm       = "MGP-HMAC-SHA256"
	HeaderKey       = "X-MGP-Key"
	HeaderTimestamp = "X-MGP-Timestamp"
	HeaderNonce     = "X-MGP-Nonce"
	HeaderSignature = "X-MGP-Signature"
	HeaderRequestID = "X-MGP-Request-Id"

	// MaxClockSkew 允许的时钟偏差；nonce 防重放窗口与之相同。
	MaxClockSkew = nodeauth.MaxClockSkew

	maxNonce = 128
)

var (
	ErrMissing   = errors.New("SIGNATURE_MISSING")
	ErrClockSkew = errors.New("CLOCK_SKEW")
	ErrSignature = errors.New("SIGNATURE_INVALID")
	ErrReplay    = errors.New("REPLAY")
)

// Canonical 是签名覆盖的规范串。
func Canonical(keyID string, timestamp int64, nonce, method, requestURI string, body []byte) string {
	digest := sha256.Sum256(body)
	return strings.Join([]string{
		Algorithm,
		keyID,
		strconv.FormatInt(timestamp, 10),
		nonce,
		method,
		requestURI,
		hex.EncodeToString(digest[:]),
	}, "\n")
}

// Sign 计算一次请求的签名。
func Sign(secret, keyID string, timestamp int64, nonce, method, requestURI string, body []byte) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(Canonical(keyID, timestamp, nonce, method, requestURI, body)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// SignRequest 为请求写入签名头；requestURI 取 req.URL.RequestURI()。
func SignRequest(req *http.Request, keyID, secret string, body []byte, now time.Time) error {
	raw := make([]byte, 16)
	if _, err := rand.Read(raw); err != nil {
		return err
	}
	nonce := base64.RawURLEncoding.EncodeToString(raw)
	timestamp := now.Unix()
	req.Header.Set(HeaderKey, keyID)
	req.Header.Set(HeaderTimestamp, strconv.FormatInt(timestamp, 10))
	req.Header.Set(HeaderNonce, nonce)
	req.Header.Set(HeaderSignature, Sign(secret, keyID, timestamp, nonce, req.Method, req.URL.RequestURI(), body))
	return nil
}

// Verifier 校验收到的签名请求（游戏服务端一侧使用）。
type Verifier struct {
	keyID  string
	secret string
	guard  *nodeauth.ReplayGuard
}

func NewVerifier(keyID, secret string) *Verifier {
	return &Verifier{keyID: keyID, secret: secret, guard: nodeauth.NewReplayGuard()}
}

// Verify 校验签名头、时间窗与重放；body 为完整请求体。
func (v *Verifier) Verify(r *http.Request, body []byte, now time.Time) error {
	keyID, nonce, signature := r.Header.Get(HeaderKey), r.Header.Get(HeaderNonce), r.Header.Get(HeaderSignature)
	timestamp, err := strconv.ParseInt(r.Header.Get(HeaderTimestamp), 10, 64)
	if keyID == "" || nonce == "" || len(nonce) > maxNonce || signature == "" || err != nil {
		return ErrMissing
	}
	if skew := now.Sub(time.Unix(timestamp, 0)); skew > MaxClockSkew || skew < -MaxClockSkew {
		return ErrClockSkew
	}
	expected := Sign(v.secret, v.keyID, timestamp, nonce, r.Method, r.URL.RequestURI(), body)
	if !hmac.Equal([]byte(keyID), []byte(v.keyID)) || !hmac.Equal([]byte(expected), []byte(signature)) {
		return ErrSignature
	}
	if err := v.guard.Use(keyID, nonce, now); err != nil {
		return ErrReplay
	}
	return nil
}

var (
	keyIDPattern  = regexp.MustCompile(`^[A-Za-z0-9_-]{4,64}$`)
	secretPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{32,256}$`)
)

// ValidateKey 校验一组 MGP 密钥的格式：key_id 为 4–64 位字母数字、_ 或 -；
// secret 为 32–256 位 base64url 字符（前端生成的是 32 字节随机数的 base64url，43 位，与游戏服务端 mgp keygen 一致）。
func ValidateKey(keyID, secret string) error {
	if !keyIDPattern.MatchString(keyID) {
		return errors.New("MGP Key ID 格式不正确")
	}
	if !secretPattern.MatchString(secret) {
		return errors.New("MGP 密钥格式不正确")
	}
	return nil
}
