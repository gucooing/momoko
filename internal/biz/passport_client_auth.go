package biz

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/passportuser"
)

// 客户端浏览器授权：PC 启动器、Android APK 与蓝色星原的游戏内登录不接触通行证的账号、密码或任何登录方式，只经浏览器取得通行证会话。
//
//  1. 客户端 POST /api/v1/auth/client 发起（不用登录），拿到 device_code（只给客户端）、user_code（XXXX-XXXX）与授权页地址；
//  2. 玩家在系统浏览器里登录通行证网页，在授权页（{public_url}/authorize?code=）核对 user_code 后授权或拒绝；
//  3. 客户端每 interval 秒用 device_code 领取（POST /api/v1/auth/client/token）：授权后为授权的用户建立这台设备的会话，
//     与登录相同的会话、令牌与登录记录（登录方式 client），device_code 随即作废。
//
// 待授权的请求与登录挑战、邮箱验证码一样只放在进程内存里，600 秒后失效；device_code 只存 SHA-256。

const (
	clientAuthTTL      = 10 * time.Minute
	clientAuthInterval = 2 * time.Second
	// clientAuthSlowStep 轮询过快时间隔增加的秒数（客户端收到 SLOW_DOWN 后同样加 2 秒）。
	clientAuthSlowStep = 2 * time.Second
	// clientAuthPollSlack 判断轮询过快时容忍的网络抖动。
	clientAuthPollSlack = 500 * time.Millisecond
	// clientAuthRateLimit / clientAuthRateWindow 同一 IP 发起授权的上限。
	clientAuthRateLimit  = 30
	clientAuthRateWindow = 10 * time.Minute
	// clientAuthMissLimit 授权页输错 user_code 的次数上限（窗口同登录失败锁定）。
	clientAuthMissLimit = 10

	userCodeAlphabet = "BCDFGHJKLMNPQRSTVWXZ"
	userCodeLen      = 8
)

// 授权请求的状态。
const (
	clientAuthPending  = "pending"
	clientAuthApproved = "approved"
	clientAuthDenied   = "denied"
)

// clientAuthClients 内置的客户端与显示名。launcher、android 是学マス 的客户端（授权页核对 user_code）；
// azurpromilia 是蓝色星原的游戏内登录：游戏照官方 SDK 的做法跳到浏览器单点授权，授权页由客户端直接打开，不核对 user_code。
var clientAuthClients = map[string]string{
	"launcher":     "Momoko 启动器",
	"android":      "学马仕 Android",
	"azurpromilia": "蓝色星原",
}

// clientAuth 一个待领取的客户端授权请求。
type clientAuth struct {
	Client    string
	DeviceID  string
	Device    string
	UserCode  string
	CreatedAt time.Time
	ExpiresAt time.Time
	// Interval 当前要求的轮询间隔；LastPoll 上一次领取的时间
	Interval time.Duration
	LastPoll time.Time
	State    string
	// UserID 授权的通行证用户
	UserID string
}

func clientAuthKey(deviceCode string) string {
	sum := sha256.Sum256([]byte(deviceCode))
	return hex.EncodeToString(sum[:])
}

// newUserCode 8 个字符，字母表 BCDFGHJKLMNPQRSTVWXZ（拒绝采样，均匀分布）。
func newUserCode() (string, error) {
	out := make([]byte, 0, userCodeLen)
	buf := make([]byte, 16)
	limit := byte(256 - 256%len(userCodeAlphabet))
	for len(out) < userCodeLen {
		if _, err := rand.Read(buf); err != nil {
			return "", err
		}
		for _, b := range buf {
			if b < limit && len(out) < userCodeLen {
				out = append(out, userCodeAlphabet[int(b)%len(userCodeAlphabet)])
			}
		}
	}
	return string(out), nil
}

// formatUserCode XXXXXXXX → XXXX-XXXX。
func formatUserCode(code string) string {
	return code[:4] + "-" + code[4:]
}

// normalizeUserCode 玩家输入的 user_code：不区分大小写，去掉横线与空白；格式不对时为空串。
func normalizeUserCode(input string) string {
	var b strings.Builder
	for _, r := range strings.ToUpper(input) {
		if r == '-' || r == ' ' || r == '\t' {
			continue
		}
		if !strings.ContainsRune(userCodeAlphabet, r) {
			return ""
		}
		b.WriteRune(r)
	}
	if b.Len() != userCodeLen {
		return ""
	}
	return b.String()
}

// StartClientAuth 客户端发起浏览器授权。publicURL 为通行证的对外地址（授权页地址由它拼出）。
func (uc *PassportUsecase) StartClientAuth(ctx context.Context, req *v1.StartPassportClientAuthRequest, publicURL string) (*v1.StartPassportClientAuthResponse, error) {
	client := strings.TrimSpace(req.Client)
	deviceID := strings.TrimSpace(req.DeviceId)
	device := strings.TrimSpace(req.Device)
	if _, ok := clientAuthClients[client]; !ok || deviceID == "" || len(deviceID) > 128 {
		return nil, ErrPassportInvalidArgument
	}
	if len([]rune(device)) > 120 {
		device = string([]rune(device)[:120])
	}
	if publicURL == "" {
		return nil, ErrPassportPublicURLUnset
	}
	now := time.Now()
	if !uc.clientAuthRate.allow(PassportClientOf(ctx).IP, now) {
		return nil, ErrPassportTooManyRequests
	}
	deviceCode, err := randomToken(32)
	if err != nil {
		return nil, ErrSystem(err)
	}
	key := clientAuthKey(deviceCode)

	uc.clientAuthMu.Lock()
	defer uc.clientAuthMu.Unlock()
	code := ""
	for attempt := 0; attempt < 5 && code == ""; attempt++ {
		next, err := newUserCode()
		if err != nil {
			return nil, ErrSystem(err)
		}
		if _, taken := uc.pendingClientAuthLocked(next, now); !taken {
			code = next
		}
	}
	if code == "" {
		return nil, ErrPassportTooManyRequests
	}
	uc.clientAuths.Set(key, &clientAuth{
		Client:    client,
		DeviceID:  deviceID,
		Device:    device,
		UserCode:  code,
		CreatedAt: now,
		ExpiresAt: now.Add(clientAuthTTL),
		Interval:  clientAuthInterval,
		State:     clientAuthPending,
	})
	uc.clientAuthCodes.Set(code, key)
	return &v1.StartPassportClientAuthResponse{
		DeviceCode:      deviceCode,
		UserCode:        formatUserCode(code),
		VerificationUrl: publicURL + "/authorize?code=" + formatUserCode(code),
		Interval:        int32(clientAuthInterval / time.Second),
		ExpiresIn:       int32(clientAuthTTL / time.Second),
	}, nil
}

// pendingClientAuthLocked 还在等待授权的请求（调用方持有 clientAuthMu）。
func (uc *PassportUsecase) pendingClientAuthLocked(code string, now time.Time) (*clientAuth, bool) {
	key, ok := uc.clientAuthCodes.Get(code)
	if !ok {
		return nil, false
	}
	a, ok := uc.clientAuths.Get(key)
	if !ok || a.State != clientAuthPending || !now.Before(a.ExpiresAt) {
		return nil, false
	}
	return a, true
}

// findClientAuthLocked 授权页按玩家输入的 user_code 找待授权的请求；输错的次数按用户计，太多时 429（调用方持有 clientAuthMu）。
func (uc *PassportUsecase) findClientAuthLocked(userID, input string, now time.Time) (*clientAuth, error) {
	if f, ok := uc.clientAuthMisses.Get(userID); ok && f.N >= clientAuthMissLimit && time.Since(f.First) < passportLockWindow {
		return nil, ErrPassportTooManyRequests
	}
	if code := normalizeUserCode(input); code != "" {
		if a, ok := uc.pendingClientAuthLocked(code, now); ok {
			return a, nil
		}
	}
	bumpFail(uc.clientAuthMisses, userID)
	return nil, ErrPassportAuthNotFound
}

// GetClientAuth 授权页显示的请求信息。
func (uc *PassportUsecase) GetClientAuth(ctx context.Context, userID, userCode string) (*v1.PassportClientAuth, error) {
	uc.clientAuthMu.Lock()
	defer uc.clientAuthMu.Unlock()
	a, err := uc.findClientAuthLocked(userID, userCode, time.Now())
	if err != nil {
		return nil, err
	}
	return &v1.PassportClientAuth{
		Client:     a.Client,
		ClientName: clientAuthClients[a.Client],
		Device:     a.Device,
		UserCode:   formatUserCode(a.UserCode),
		CreateTime: timestamppb.New(a.CreatedAt),
		ExpireTime: timestamppb.New(a.ExpiresAt),
	}, nil
}

// ApproveClientAuth 当前登录的用户授权这个客户端：停用的账号、登录失败锁定中的账号不能授权。
func (uc *PassportUsecase) ApproveClientAuth(ctx context.Context, userID, userCode string) error {
	u, err := uc.user(ctx, userID)
	if err != nil {
		return err
	}
	if u.Status != passportuser.StatusActive {
		return ErrPassportDisabled
	}
	if uc.lockedOut("account:"+strings.ToLower(u.Username)) || (u.Email != "" && uc.lockedOut("email:"+strings.ToLower(u.Email))) {
		return ErrPassportLocked
	}
	uc.clientAuthMu.Lock()
	defer uc.clientAuthMu.Unlock()
	a, err := uc.findClientAuthLocked(userID, userCode, time.Now())
	if err != nil {
		return err
	}
	a.State, a.UserID = clientAuthApproved, userID
	return nil
}

// DenyClientAuth 当前登录的用户拒绝这个客户端。
func (uc *PassportUsecase) DenyClientAuth(ctx context.Context, userID, userCode string) error {
	uc.clientAuthMu.Lock()
	defer uc.clientAuthMu.Unlock()
	a, err := uc.findClientAuthLocked(userID, userCode, time.Now())
	if err != nil {
		return err
	}
	a.State = clientAuthDenied
	return nil
}

// dropClientAuthLocked 作废一个请求（调用方持有 clientAuthMu）。
func (uc *PassportUsecase) dropClientAuthLocked(key string, a *clientAuth) {
	uc.clientAuths.Del(key)
	if current, ok := uc.clientAuthCodes.Get(a.UserCode); ok && current == key {
		uc.clientAuthCodes.Del(a.UserCode)
	}
}

// ClaimClientAuth 客户端领取会话：还没授权 PENDING、轮询过快 SLOW_DOWN（间隔加 2 秒）、拒绝 DENIED（随即作废）、
// 过期 / 已领取 / 不存在 EXPIRED；授权后为授权的用户建立这台设备的会话（与登录相同），只能领取一次。
func (uc *PassportUsecase) ClaimClientAuth(ctx context.Context, deviceCode string) (*v1.PassportClientToken, error) {
	deviceCode = strings.TrimSpace(deviceCode)
	if deviceCode == "" {
		return nil, ErrPassportAuthExpired
	}
	key := clientAuthKey(deviceCode)
	now := time.Now()

	uc.clientAuthMu.Lock()
	a, ok := uc.clientAuths.Get(key)
	if !ok || !now.Before(a.ExpiresAt) {
		if ok {
			uc.dropClientAuthLocked(key, a)
		}
		uc.clientAuthMu.Unlock()
		return nil, ErrPassportAuthExpired
	}
	switch a.State {
	case clientAuthDenied:
		uc.dropClientAuthLocked(key, a)
		uc.clientAuthMu.Unlock()
		return nil, ErrPassportAuthDenied
	case clientAuthPending:
		tooFast := !a.LastPoll.IsZero() && now.Sub(a.LastPoll) < a.Interval-clientAuthPollSlack
		a.LastPoll = now
		if tooFast {
			a.Interval += clientAuthSlowStep
			uc.clientAuthMu.Unlock()
			return nil, ErrPassportAuthSlowDown
		}
		uc.clientAuthMu.Unlock()
		return nil, ErrPassportAuthPending
	}
	claimed := *a
	uc.dropClientAuthLocked(key, a)
	uc.clientAuthMu.Unlock()

	u, err := uc.repo.GetUser(ctx, claimed.UserID)
	if err != nil && !gen.IsNotFound(err) {
		return nil, ErrSystem(err)
	}
	if err != nil || u.Status != passportuser.StatusActive {
		return nil, ErrPassportAuthDenied
	}
	tokens, err := uc.issue(ctx, u, claimed.DeviceID, claimed.Device, PassportLoginClient, true)
	if err != nil {
		return nil, err
	}
	return &v1.PassportClientToken{
		AccessToken:  tokens.AccessToken,
		RefreshToken: tokens.RefreshToken,
		ExpiresIn:    tokens.ExpiresIn,
	}, nil
}
