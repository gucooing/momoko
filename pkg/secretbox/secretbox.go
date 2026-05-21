package secretbox

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
)

const prefix = "v1:"

type Box struct {
	key [32]byte
}

func New(secret string) *Box {
	return &Box{key: sha256.Sum256([]byte(secret))}
}

func (b *Box) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}
	gcm, err := b.gcm()
	if err != nil {
		return "", err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return prefix + base64.RawURLEncoding.EncodeToString(ciphertext), nil
}

func (b *Box) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}
	if !strings.HasPrefix(ciphertext, prefix) {
		return "", errors.New("invalid secret format")
	}
	raw, err := base64.RawURLEncoding.DecodeString(strings.TrimPrefix(ciphertext, prefix))
	if err != nil {
		return "", err
	}
	gcm, err := b.gcm()
	if err != nil {
		return "", err
	}
	nonceSize := gcm.NonceSize()
	if len(raw) < nonceSize {
		return "", errors.New("ciphertext too short")
	}
	nonce, encrypted := raw[:nonceSize], raw[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, encrypted, nil)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (b *Box) gcm() (cipher.AEAD, error) {
	block, err := aes.NewCipher(b.key[:])
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}
