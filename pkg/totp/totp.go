// Package totp 实现 RFC 6238 基于时间的一次性密码（HMAC-SHA1、6 位、30 秒步长，兼容常见身份验证器 App），
// 以及两步验证用的恢复码与 otpauth 二维码。
package totp

import (
	"bytes"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/base32"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"net/url"
	"strings"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/qr"
)

const (
	// Period 步长（秒）。
	Period = 30
	// Digits 验证码位数。
	Digits = 6

	secretBytes = 20
	qrSize      = 280
	qrQuiet     = 20
)

var b32 = base32.StdEncoding.WithPadding(base32.NoPadding)

// GenerateSecret 生成一个新的 base32 密钥（160 位）。
func GenerateSecret() (string, error) {
	buf := make([]byte, secretBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return b32.EncodeToString(buf), nil
}

func decodeSecret(secret string) ([]byte, error) {
	secret = strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(secret), " ", ""))
	return b32.DecodeString(strings.TrimRight(secret, "="))
}

// Step 返回 t 所在的时间步。
func Step(t time.Time) int64 { return t.Unix() / Period }

// CodeAt 计算某个时间步的验证码。
func CodeAt(secret string, step int64) (string, error) {
	key, err := decodeSecret(secret)
	if err != nil {
		return "", err
	}
	var msg [8]byte
	binary.BigEndian.PutUint64(msg[:], uint64(step))
	mac := hmac.New(sha1.New, key)
	mac.Write(msg[:])
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	return fmt.Sprintf("%0*d", Digits, value%1000000), nil
}

// Verify 校验验证码，允许前后各 skew 个时间步的时钟误差；匹配时返回所在的时间步（用于拒绝重放）。
func Verify(secret, code string, t time.Time, skew int) (int64, bool) {
	code = strings.ReplaceAll(strings.TrimSpace(code), " ", "")
	if len(code) != Digits {
		return 0, false
	}
	now := Step(t)
	for d := -skew; d <= skew; d++ {
		step := now + int64(d)
		want, err := CodeAt(secret, step)
		if err != nil {
			return 0, false
		}
		if subtle.ConstantTimeCompare([]byte(want), []byte(code)) == 1 {
			return step, true
		}
	}
	return 0, false
}

// URI 生成身份验证器 App 识别的 otpauth:// 地址。
func URI(issuer, account, secret string) string {
	label := url.PathEscape(issuer) + ":" + url.PathEscape(account)
	q := url.Values{}
	q.Set("secret", secret)
	q.Set("issuer", issuer)
	q.Set("algorithm", "SHA1")
	q.Set("digits", fmt.Sprint(Digits))
	q.Set("period", fmt.Sprint(Period))
	return "otpauth://totp/" + label + "?" + q.Encode()
}

// QRCodePNG 把内容编码成 PNG 二维码（白底黑码，四周留白）。
func QRCodePNG(content string) ([]byte, error) {
	code, err := qr.Encode(content, qr.M, qr.Auto)
	if err != nil {
		return nil, err
	}
	code, err = barcode.Scale(code, qrSize, qrSize)
	if err != nil {
		return nil, err
	}
	size := qrSize + qrQuiet*2
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	draw.Draw(img, img.Bounds(), &image.Uniform{C: color.White}, image.Point{}, draw.Src)
	draw.Draw(img, image.Rect(qrQuiet, qrQuiet, qrQuiet+qrSize, qrQuiet+qrSize), code, image.Point{}, draw.Over)
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

const recoveryAlphabet = "abcdefghjkmnpqrstuvwxyz23456789"

// GenerateRecoveryCodes 生成 n 个一次性恢复码（xxxxx-xxxxx）。
func GenerateRecoveryCodes(n int) ([]string, error) {
	out := make([]string, 0, n)
	for i := 0; i < n; i++ {
		buf := make([]byte, 10)
		if _, err := rand.Read(buf); err != nil {
			return nil, err
		}
		var b strings.Builder
		for j, c := range buf {
			if j == 5 {
				b.WriteByte('-')
			}
			b.WriteByte(recoveryAlphabet[int(c)%len(recoveryAlphabet)])
		}
		out = append(out, b.String())
	}
	return out, nil
}

// NormalizeRecoveryCode 统一恢复码的写法（小写、去空白，补回中间的连字符）。
func NormalizeRecoveryCode(code string) string {
	code = strings.ToLower(strings.ReplaceAll(strings.TrimSpace(code), " ", ""))
	code = strings.ReplaceAll(code, "-", "")
	if len(code) == 10 {
		return code[:5] + "-" + code[5:]
	}
	return code
}
