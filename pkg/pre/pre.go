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
	"sync"
	"time"

	"github.com/go-jose/go-jose/v4/json"

	"momoko/pkg/auth"
)

// signingKey 从全局 auth.AuthSecretKey 派生一个稳定的 32 字节密钥（带域分隔），
// 取代旧的「进程启动时随机生成」——旧实现会让所有预签名 URL 在服务重启后立即失效，
// 导致正在进行的上传/下载（以及未来的长效分享）全部中断。延迟到首次使用时派生，
// 以保证此时 main 已从配置写入 auth.AuthSecretKey。
var (
	keyOnce    sync.Once
	derivedKey []byte
)

func signingKey() []byte {
	keyOnce.Do(func() {
		mac := hmac.New(sha256.New, []byte(auth.AuthSecretKey))
		mac.Write([]byte("momoko:presign:aesgcm:v1"))
		derivedKey = mac.Sum(nil)
	})
	return derivedKey
}

// 下载/上传/分享 预签名
type FileSignInfo struct {
	UploadId       string        `json:"upload_id"`           // 上传id
	Path           string        `json:"path"`                // 路径
	CreateAt       time.Time     `json:"create_at"`           // 创建时间
	ValidityPeriod time.Duration `json:"validity_period"`     // 有效时间
	Creator        string        `json:"creator"`             // 创建人
	Salt           []byte        `json:"salt"`                // 随机数
	SourceID       string        `json:"source_id,omitempty"` // 文件来源id，空=本地
	Inline         bool          `json:"inline,omitempty"`    // 预览(inline) 还是 下载(attachment)
	ShareID        string        `json:"share_id,omitempty"`  // 分享会话签名所属分享id
	// BasePath 是本地受限根（如实例工作目录）。非空时取件端会重建同一个受限视图再解析 Path，
	// 使签发与取件两侧的边界一致；为空则按整机视图（仅受保护清单约束）解析。
	BasePath string `json:"base_path,omitempty"`
}

func newSalt() []byte {
	key := make([]byte, 32)
	rand.Read(key)
	return key
}

func NewFileSignInfo(uploadId, path, creator string, validityPeriod time.Duration) *FileSignInfo {
	return &FileSignInfo{
		UploadId:       uploadId,
		Path:           path,
		CreateAt:       time.Now(),
		ValidityPeriod: validityPeriod,
		Creator:        creator,
		Salt:           newSalt(),
	}
}

// NewShareSignInfo 生成一次性分享会话签名信息（仅绑定分享id；有效期由调用方按「不超过分享过期」给定）。
func NewShareSignInfo(shareID string, validityPeriod time.Duration) *FileSignInfo {
	return &FileSignInfo{
		ShareID:        shareID,
		CreateAt:       time.Now(),
		ValidityPeriod: validityPeriod,
		Salt:           newSalt(),
	}
}

func (f *FileSignInfo) Sign() (string, error) {
	fs, err := json.Marshal(f)
	if err != nil {
		return "", err
	}
	key := signingKey()
	ciphertext, err := aesGcmEncrypt(fs, key)
	if err != nil {
		return "", err
	}
	mac := hmac.New(sha256.New, key)
	mac.Write(ciphertext)
	signature := mac.Sum(nil)
	token := base64.RawURLEncoding.EncodeToString(ciphertext) + "." +
		base64.RawURLEncoding.EncodeToString(signature)

	return token, nil
}

func Verify(sign string) (*FileSignInfo, error) {
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
	key := signingKey()
	mac := hmac.New(sha256.New, key)
	mac.Write(ciphertext)
	expectedSig := mac.Sum(nil)
	if !hmac.Equal(receivedSig, expectedSig) {
		return nil, errors.New("签名验证失败，Sign 可能被篡改")
	}
	plaintext, err := aesGcmDecrypt(ciphertext, key)
	if err != nil {
		return nil, errors.New("解密失败")
	}
	info := new(FileSignInfo)
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
