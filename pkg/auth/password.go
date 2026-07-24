package auth

import (
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

// VerifyPassword 校验明文口令与存储的 bcrypt 哈希是否匹配。
func VerifyPassword(password, stored string) bool {
	if stored == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(stored), []byte(password)) == nil
}
