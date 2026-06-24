package auth

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

// bcryptCost 控制 bcrypt 计算强度（越高越慢、越抗暴力破解）。
const bcryptCost = bcrypt.DefaultCost

// HashPassword 使用 bcrypt 对明文口令做加盐慢哈希。
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword 校验明文口令与存储哈希是否匹配。
// 返回 (是否匹配, 是否需要升级哈希)。
//
// 为平滑迁移，兼容历史的无盐 MD5（32 位十六进制）：旧哈希校验通过时 needsUpgrade=true，
// 调用方应借机用 HashPassword 把该用户口令重写为 bcrypt。
func VerifyPassword(password, stored string) (ok bool, needsUpgrade bool) {
	if stored == "" {
		return false, false
	}
	if isLegacyMD5(stored) {
		legacy := legacyEncodeMD5(password)
		if subtle.ConstantTimeCompare([]byte(legacy), []byte(strings.ToLower(stored))) == 1 {
			return true, true
		}
		return false, false
	}
	if err := bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)); err != nil {
		return false, false
	}
	return true, false
}

// isLegacyMD5 判断存储值是否为历史的 32 位十六进制 MD5 哈希。
func isLegacyMD5(stored string) bool {
	if len(stored) != 32 {
		return false
	}
	for _, c := range stored {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') && (c < 'A' || c > 'F') {
			return false
		}
	}
	return true
}

// legacyEncodeMD5 复刻旧版口令哈希算法，仅用于迁移期校验，禁止用于新口令。
func legacyEncodeMD5(password string) string {
	sum := md5.Sum([]byte(password))
	return hex.EncodeToString(sum[:])
}
