package oidcserver

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"strings"
)

func GenerateClientID() (string, error) {
	value, err := randomToken(24)
	if err != nil {
		return "", err
	}
	return "momoko_" + value, nil
}

// GenerateClientSecret 生成只展示一次的客户端密钥。
// 使用 crypto/rand，避免可预测随机数导致客户端伪造。
func GenerateClientSecret() (string, error) {
	return randomToken(48)
}

func GenerateAuthorizationCode() (string, error) {
	return randomToken(32)
}

func GenerateAccessTokenID() (string, error) {
	return randomToken(32)
}

// VerifySecret 使用常量时间比较校验 Client Secret，减少时序侧信道。
func VerifySecret(expected, actual string) bool {
	expected = strings.TrimSpace(expected)
	actual = strings.TrimSpace(actual)
	if expected == "" || actual == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(expected), []byte(actual)) == 1
}

// MaskSecret 用于列表页展示密钥，避免接口无意返回完整 secret。
func MaskSecret(value string) string {
	value = strings.TrimSpace(value)
	if len(value) <= 8 {
		return "********"
	}
	return value[:4] + "********" + value[len(value)-4:]
}

// VerifyPKCE 校验授权码交换时的 PKCE 参数。
// 没有 code_challenge 时允许通过，以兼容普通 confidential client。
func VerifyPKCE(challenge, method, verifier string) bool {
	challenge = strings.TrimSpace(challenge)
	method = strings.TrimSpace(method)
	verifier = strings.TrimSpace(verifier)
	if challenge == "" {
		return true
	}
	if verifier == "" {
		return false
	}
	switch method {
	case "", "plain":
		return subtle.ConstantTimeCompare([]byte(challenge), []byte(verifier)) == 1
	case "S256":
		sum := sha256.Sum256([]byte(verifier))
		encoded := base64.RawURLEncoding.EncodeToString(sum[:])
		return subtle.ConstantTimeCompare([]byte(challenge), []byte(encoded)) == 1
	default:
		return false
	}
}

func randomToken(size int) (string, error) {
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}
