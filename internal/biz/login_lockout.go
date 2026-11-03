package biz

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/cache"
	"momoko/pkg/response"
	"momoko/pkg/utils"
)

// 登录失败锁定：按「账号 + IP」计数，连续失败达到系统设置里的次数后，在锁定时长内拒绝该 IP 再尝试该账号。
// 计数保存在内存里（进程重启清零），成功登录即清除。

type loginAttempt struct {
	fails       int32
	last        time.Time
	lockedUntil time.Time
}

var (
	loginAttempts   = cache.New[string, *loginAttempt](24 * time.Hour)
	loginAttemptsMu sync.Mutex
)

// LoginAttemptKey 由登录标识（用户名 / 邮箱）与客户端 IP 组成计数键。
func LoginAttemptKey(ctx context.Context, req *v1.LoginRequest) string {
	identity := req.GetUsername()
	if identity == "" {
		identity = req.GetEmail()
	}
	return strings.ToLower(strings.TrimSpace(identity)) + "|" + utils.ClientIPFromContext(ctx)
}

// CheckLoginLocked 处于锁定期时返回错误（告知剩余分钟数）。
func CheckLoginLocked(key string, cfg *v1.LoginConfig) error {
	if cfg.GetLockoutThreshold() <= 0 {
		return nil
	}
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	a, ok := loginAttempts.Get(key)
	if !ok || a == nil || !time.Now().Before(a.lockedUntil) {
		return nil
	}
	return loginLockedError(time.Until(a.lockedUntil))
}

// RecordLoginFailure 记一次失败；达到阈值时进入锁定并返回锁定错误，否则返回 nil。
func RecordLoginFailure(key string, cfg *v1.LoginConfig) error {
	threshold := cfg.GetLockoutThreshold()
	if threshold <= 0 {
		return nil
	}
	lockFor := time.Duration(cfg.GetLockoutMinutes()) * time.Minute
	if lockFor <= 0 {
		lockFor = 15 * time.Minute
	}
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	now := time.Now()
	a, ok := loginAttempts.Get(key)
	// 距上次失败已超过一个锁定周期，重新计数
	if !ok || a == nil || now.Sub(a.last) > lockFor {
		a = &loginAttempt{}
	}
	a.fails++
	a.last = now
	if a.fails >= threshold {
		a.fails = 0
		a.lockedUntil = now.Add(lockFor)
		loginAttempts.Set(key, a)
		return loginLockedError(lockFor)
	}
	loginAttempts.Set(key, a)
	return nil
}

// ClearLoginFailures 登录成功后清除计数。
func ClearLoginFailures(key string) {
	loginAttemptsMu.Lock()
	defer loginAttemptsMu.Unlock()
	loginAttempts.Del(key)
}

// IsLoginCredentialError 判断是否为凭据类失败（只有这类失败计入锁定）。
func IsLoginCredentialError(err error) bool {
	return err == ErrInvalidPassword || err == ErrAdminNotFound || err == ErrEmailCodeInvalid
}

func loginLockedError(left time.Duration) error {
	minutes := int(math.Ceil(left.Minutes()))
	if minutes < 1 {
		minutes = 1
	}
	return response.BadRequest(429, fmt.Sprintf("登录失败次数过多，请 %d 分钟后再试", minutes))
}
