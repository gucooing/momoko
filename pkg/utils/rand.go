package utils

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

const (
	apiKeyAlphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
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

func GenerateAPIKey() (string, error) {
	raw := make([]byte, 20)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate api key: %w", err)
	}
	for i, b := range raw {
		raw[i] = apiKeyAlphabet[int(b)%len(apiKeyAlphabet)]
	}
	return "mk-" + string(raw), nil
}
