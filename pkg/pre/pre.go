package pre

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"time"

	"github.com/go-jose/go-jose/v4/json"
)

var secretKey = func() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}()

// 下载预签名
type FileDownloadInfo struct {
	Path           string        `json:"path"`            // 路径
	CreateAt       time.Time     `json:"create_at"`       // 创建时间
	ValidityPeriod time.Duration `json:"validity_period"` // 有效时间
	Creator        string        `json:"creator"`         // 创建人
	Salt           []byte        `json:"salt"`
}

func NewFileDownloadInfo(path string, validityPeriod time.Duration, creator string) *FileDownloadInfo {
	return &FileDownloadInfo{
		Path:           path,
		CreateAt:       time.Now(),
		ValidityPeriod: validityPeriod,
		Creator:        creator,
		Salt: func() []byte {
			key := make([]byte, 32)
			rand.Read(key)
			return key
		}(),
	}
}

func (f *FileDownloadInfo) Sign() (string, error) {
	fs, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	ciphertext, err := aesGcmEncrypt(fs, secretKey)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, secretKey)
	mac.Write(ciphertext)
	signature := mac.Sum(nil)
	token := base64.RawURLEncoding.EncodeToString(ciphertext) + "." +
		base64.RawURLEncoding.EncodeToString(signature)

	return token, nil
}

func Verify(sign string) (*FileDownloadInfo, error) {
	parts := strings.Split(sign, ".")
	if len(parts) != 2 {
		return nil, errors.New("无效的 Sign 格式")
	}
	ciphertext, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return nil, errors.New("密文解码失败")
	}
	receivedSig, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, errors.New("签名解码失败")
	}
	mac := hmac.New(sha256.New, secretKey)
	mac.Write(ciphertext)
	expectedSig := mac.Sum(nil)
	if !hmac.Equal(receivedSig, expectedSig) {
		return nil, errors.New("签名验证失败，Sign 可能被篡改")
	}
	plaintext, err := aesGcmDecrypt(ciphertext, secretKey)
	if err != nil {
		return nil, errors.New("解密失败")
	}
	info := new(FileDownloadInfo)
	err = json.Unmarshal(plaintext, info)
	if err != nil {
		return nil, err
	}
	if info.CreateAt.Add(info.ValidityPeriod).Before(time.Now()) {
		return nil, errors.New("sign 已失效")
	}
	return info, nil
}

func aesGcmEncrypt(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}
	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func aesGcmDecrypt(ciphertext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("密文太短")
	}
	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}
