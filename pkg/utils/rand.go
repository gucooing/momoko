package utils

import (
	"crypto/rand"
	"encoding/base64"
)

func GenerateRandomString(length int) string {
	bytes := make([]byte, length)
	_, err := rand.Read(bytes)
	if err != nil {
		return ""
	}
	return base64.URLEncoding.EncodeToString(bytes)[:length]
}

func GenerateEmailCode(length int) (string, error) {
	raw := make([]byte, length)
	if _, err := rand.Read(raw); err != nil {
		return "", err
	}
	for i, b := range raw {
		raw[i] = '0' + b%10
	}
	return string(raw), nil
}
