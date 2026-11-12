package biz

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/passportuser"
	auth2 "momoko/pkg/auth"
	"momoko/pkg/passporttoken"
	"momoko/pkg/totp"
	"momoko/pkg/utils"
)

// 邮箱验证码用途。
const (
	passportCodeRegister = "register"
	passportCodeLogin    = "login"
	passportCodeReset    = "reset"
	passportCodeChange   = "change"
)

// PassportClient 发起请求的客户端信息（IP、User-Agent）。
type PassportClient struct {
	IP        string
	UserAgent string
}

// PassportClientOf 从请求上下文读出客户端信息。
func PassportClientOf(ctx context.Context) PassportClient {
	return PassportClient{IP: utils.ClientIPFromContext(ctx), UserAgent: userAgentFromContext(ctx)}
}

type mfaChallenge struct {
	UserID     string
	DeviceID   string
	Device     string
	Method     string
	Persistent bool
	Attempts   int
}

// ---------------- 邮箱验证码 ----------------

func passportCodeKey(purpose, email string) string {
	return purpose + ":" + strings.ToLower(strings.TrimSpace(email))
}

// sendCode 生成并发送验证码（同一邮箱同一用途 60 秒内只能发一次；发送失败不计入节流）。
func (uc *PassportUsecase) sendCode(ctx context.Context, purpose, email string, tmpl v1.EmailTemplateType, data map[string]string) (*v1.SendPassportEmailCodeResponse, error) {
	if !uc.EmailEnabled(ctx) {
		return nil, ErrPassportEmailUnavailable
	}
	key := passportCodeKey(purpose, email)
	if sentAt, ok := uc.emailCodeSent.Get(key); ok {
		if wait := EmailCodeResendInterval - time.Since(sentAt); wait > 0 {
			return nil, ErrPassportCodeTooFrequent
		}
	}
	code, err := utils.GenerateEmailCode(emailCodeLength)
	if err != nil {
		return nil, ErrSystem(err)
	}
	uc.emailCodes.Set(key, expiring[string]{V: code, Exp: time.Now().Add(EmailCodeTTL)})
	uc.emailCodeSent.Set(key, time.Now())
	data["code"] = code
	data["email"] = email
	if err := uc.system.SendEmail(ctx, tmpl, email, data); err != nil {
		uc.emailCodes.Del(key)
		uc.emailCodeSent.Del(key)
		uc.log.Warnf("send passport %s code: %v", purpose, err)
		return nil, ErrPassportEmailUnavailable
	}
	return &v1.SendPassportEmailCodeResponse{
		ExpiresInSeconds:   int64(EmailCodeTTL.Seconds()),
		ResendAfterSeconds: int64(EmailCodeResendInterval.Seconds()),
	}, nil
}

// verifyCode 校验并消费验证码。
func (uc *PassportUsecase) verifyCode(purpose, email, code string) error {
	key := passportCodeKey(purpose, email)
	v, ok := uc.emailCodes.Get(key)
	if !ok || !v.alive() || v.V != strings.TrimSpace(code) {
		return ErrPassportCodeInvalid
	}
	uc.emailCodes.Del(key)
	return nil
}

func normalizeEmail(email string) (string, error) {
	email = strings.TrimSpace(email)
	if email == "" {
		return "", ErrPassportEmailRequired
	}
	addr, err := mail.ParseAddress(email)
	if err != nil || addr.Address != email || !strings.Contains(email, "@") {
		return "", ErrPassportEmailInvalid
	}
	return email, nil
}

// SendRegisterCode 发送注册验证码（邮箱已被使用时拒绝）。
func (uc *PassportUsecase) SendRegisterCode(ctx context.Context, email string) (*v1.SendPassportEmailCodeResponse, error) {
	s := uc.Settings(ctx)
	if s.RegisterMode == PassportRegisterClosed {
		return nil, ErrPassportRegisterClosed
	}
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	if _, err := uc.repo.FindUserByEmail(ctx, email); err == nil {
		return nil, ErrPassportEmailTaken
	} else if !gen.IsNotFound(err) {
		return nil, ErrSystem(err)
	}
	return uc.sendCode(ctx, passportCodeRegister, email, v1.EmailTemplateType_EmailTemplateType_Register, map[string]string{})
}

// SendLoginCode 发送登录验证码。为了不暴露邮箱是否注册，找不到账号时也返回成功（但不发送）。
func (uc *PassportUsecase) SendLoginCode(ctx context.Context, email string) (*v1.SendPassportEmailCodeResponse, error) {
	if !uc.Settings(ctx).EmailLoginEnabled {
		return nil, ErrPassportEmailLoginOff
	}
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	u, err := uc.repo.FindUserByEmail(ctx, email)
	if err != nil || !u.EmailVerified || !u.EmailCodeLogin || u.Status != passportuser.StatusActive {
		if err != nil && !gen.IsNotFound(err) {
			return nil, ErrSystem(err)
		}
		return &v1.SendPassportEmailCodeResponse{ExpiresInSeconds: int64(EmailCodeTTL.Seconds()), ResendAfterSeconds: int64(EmailCodeResendInterval.Seconds())}, nil
	}
	return uc.sendCode(ctx, passportCodeLogin, email, v1.EmailTemplateType_EmailTemplateType_Login, map[string]string{"name": displayName(u)})
}

// SendResetCode 发送找回密码验证码（同样不暴露邮箱是否注册）。
func (uc *PassportUsecase) SendResetCode(ctx context.Context, email string) (*v1.SendPassportEmailCodeResponse, error) {
	email, err := normalizeEmail(email)
	if err != nil {
		return nil, err
	}
	u, err := uc.repo.FindUserByEmail(ctx, email)
	if err != nil || !u.EmailVerified || u.Status != passportuser.StatusActive {
		if err != nil && !gen.IsNotFound(err) {
			return nil, ErrSystem(err)
		}
		return &v1.SendPassportEmailCodeResponse{ExpiresInSeconds: int64(EmailCodeTTL.Seconds()), ResendAfterSeconds: int64(EmailCodeResendInterval.Seconds())}, nil
	}
	return uc.sendCode(ctx, passportCodeReset, email, v1.EmailTemplateType_EmailTemplateType_ResetPassword, map[string]string{"name": displayName(u)})
}

// ---------------- 注册 ----------------

func validPassportPassword(p string) error {
	if len([]rune(p)) < MinPasswordLength {
		return ErrPassportPasswordTooShort
	}
	return nil
}

// checkInvite 校验邀请码（不消费）。
func (uc *PassportUsecase) checkInvite(ctx context.Context, code, email string) (*gen.PassportInvite, error) {
	inv, err := uc.repo.GetInviteByCode(ctx, strings.TrimSpace(code))
	if gen.IsNotFound(err) {
		return nil, ErrPassportInviteInvalid
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	if inv.Revoked || (inv.ExpiresAt != nil && time.Now().After(*inv.ExpiresAt)) || inv.Used >= inv.MaxUses {
		return nil, ErrPassportInviteInvalid
	}
	if inv.Email != "" && !strings.EqualFold(inv.Email, strings.TrimSpace(email)) {
		return nil, ErrPassportInviteInvalid
	}
	return inv, nil
}

// Register 注册并直接登录。
func (uc *PassportUsecase) Register(ctx context.Context, req *v1.PassportRegisterRequest) (*v1.PassportLoginResponse, error) {
	s := uc.Settings(ctx)
	if s.RegisterMode == PassportRegisterClosed {
		return nil, ErrPassportRegisterClosed
	}
	username := strings.TrimSpace(req.Username)
	if !registerUsernamePattern.MatchString(username) {
		return nil, ErrPassportUsernameInvalid
	}
	if err := validPassportPassword(req.Password); err != nil {
		return nil, err
	}
	needCode := s.RegisterEmailVerification && uc.EmailEnabled(ctx)
	email := strings.TrimSpace(req.Email)
	if email != "" || needCode {
		var err error
		if email, err = normalizeEmail(email); err != nil {
			return nil, err
		}
	}
	var invite *gen.PassportInvite
	if code := strings.TrimSpace(req.InviteCode); code != "" || s.RegisterMode == PassportRegisterInviteOnly {
		if code == "" {
			return nil, ErrPassportInviteRequired
		}
		var err error
		if invite, err = uc.checkInvite(ctx, code, email); err != nil {
			return nil, err
		}
	}
	if _, err := uc.repo.FindUserByUsername(ctx, username); err == nil {
		return nil, ErrPassportUsernameTaken
	} else if !gen.IsNotFound(err) {
		return nil, ErrSystem(err)
	}
	if email != "" {
		if _, err := uc.repo.FindUserByEmail(ctx, email); err == nil {
			return nil, ErrPassportEmailTaken
		} else if !gen.IsNotFound(err) {
			return nil, ErrSystem(err)
		}
	}
	if needCode {
		if err := uc.verifyCode(passportCodeRegister, email, req.Code); err != nil {
			return nil, err
		}
	}
	if invite != nil {
		ok, err := uc.repo.UseInvite(ctx, invite.ID, time.Now())
		if err != nil {
			return nil, ErrSystem(err)
		}
		if !ok {
			return nil, ErrPassportInviteInvalid
		}
	}
	hash, err := auth2.HashPassword(req.Password)
	if err != nil {
		return nil, ErrSystem(err)
	}
	now := time.Now()
	client := PassportClientOf(ctx)
	u := &gen.PassportUser{
		ID:                "pp_" + uuid.NewString(),
		Username:          username,
		Email:             email,
		EmailVerified:     needCode,
		Password:          hash,
		PasswordUpdatedAt: &now,
		Name:              username,
		Language:          normalizeLanguage(req.Language),
		EmailCodeLogin:    true,
		Status:            passportuser.StatusActive,
		RegisterIP:        client.IP,
	}
	if invite != nil {
		u.InviteID = invite.ID
	}
	created, err := uc.repo.CreateUser(ctx, u)
	if err != nil {
		if gen.IsConstraintError(err) {
			return nil, ErrPassportUsernameTaken
		}
		return nil, ErrSystem(err)
	}
	return uc.issue(ctx, created, req.DeviceId, req.Device, PassportLoginRegister, true)
}

func normalizeLanguage(lang string) string {
	switch lang {
	case "zh-CN", "zh-HK", "en-US":
		return lang
	default:
		return ""
	}
}

// ---------------- 登录 ----------------

// lockedOut 账号是否处于失败锁定中。
func (uc *PassportUsecase) lockedOut(key string) bool {
	f, ok := uc.loginFails.Get(key)
	return ok && f.N >= passportLockThreshold && time.Since(f.First) < passportLockWindow
}

func bumpFail(c interface {
	Get(string) (failCount, bool)
	Set(string, failCount) bool
}, key string) int {
	f, ok := c.Get(key)
	if !ok || time.Since(f.First) >= passportLockWindow {
		f = failCount{First: time.Now()}
	}
	f.N++
	c.Set(key, f)
	return f.N
}

func (uc *PassportUsecase) recordLogin(ctx context.Context, userID, deviceID, device, method string, success bool, code string) {
	client := PassportClientOf(ctx)
	_ = uc.repo.AddLoginLog(context.WithoutCancel(ctx), &gen.PassportLoginLog{
		ID:        uuid.NewString(),
		UserID:    userID,
		Method:    method,
		Success:   success,
		ErrorCode: code,
		IP:        client.IP,
		DeviceID:  deviceID,
		Device:    device,
		UserAgent: client.UserAgent,
	})
}

// Login 用户名 / 邮箱 + 密码，或邮箱 + 验证码。开启两步验证时返回 mfa_token。
func (uc *PassportUsecase) Login(ctx context.Context, req *v1.PassportLoginRequest) (*v1.PassportLoginResponse, error) {
	if strings.TrimSpace(req.DeviceId) == "" {
		return nil, ErrPassportInvalidArgument
	}
	var (
		u      *gen.PassportUser
		err    error
		method string
	)
	if strings.TrimSpace(req.Code) != "" {
		method = PassportLoginEmailCode
		if !uc.Settings(ctx).EmailLoginEnabled || !uc.EmailEnabled(ctx) {
			return nil, ErrPassportEmailLoginOff
		}
		email := strings.TrimSpace(req.Email)
		if uc.lockedOut("email:" + strings.ToLower(email)) {
			return nil, ErrPassportLocked
		}
		u, err = uc.repo.FindUserByEmail(ctx, email)
		if err != nil || !u.EmailVerified || !u.EmailCodeLogin {
			if err != nil && !gen.IsNotFound(err) {
				return nil, ErrSystem(err)
			}
			bumpFail(uc.loginFails, "email:"+strings.ToLower(email))
			return nil, ErrPassportCodeInvalid
		}
		if err := uc.verifyCode(passportCodeLogin, email, req.Code); err != nil {
			bumpFail(uc.loginFails, "email:"+strings.ToLower(email))
			uc.recordLogin(ctx, u.ID, req.DeviceId, req.Device, method, false, "code_invalid")
			return nil, err
		}
	} else {
		method = PassportLoginPassword
		account := strings.TrimSpace(req.Account)
		if account == "" || req.Password == "" {
			return nil, ErrPassportCredentials
		}
		key := "account:" + strings.ToLower(account)
		if uc.lockedOut(key) {
			return nil, ErrPassportLocked
		}
		u, err = uc.repo.FindUserByUsername(ctx, account)
		if gen.IsNotFound(err) && strings.Contains(account, "@") {
			u, err = uc.repo.FindUserByEmail(ctx, account)
		}
		if err != nil {
			if !gen.IsNotFound(err) {
				return nil, ErrSystem(err)
			}
			bumpFail(uc.loginFails, key)
			return nil, ErrPassportCredentials
		}
		if !auth2.VerifyPassword(req.Password, u.Password) {
			bumpFail(uc.loginFails, key)
			uc.recordLogin(ctx, u.ID, req.DeviceId, req.Device, method, false, "password_invalid")
			return nil, ErrPassportCredentials
		}
		uc.loginFails.Del(key)
	}
	if u.Status != passportuser.StatusActive {
		return nil, ErrPassportDisabled
	}
	if u.TotpSecret != "" && u.TotpEnabledAt != nil {
		token, err := randomToken(32)
		if err != nil {
			return nil, ErrSystem(err)
		}
		uc.mfaChallenges.Set(token, expiring[mfaChallenge]{
			V:   mfaChallenge{UserID: u.ID, DeviceID: req.DeviceId, Device: req.Device, Method: method, Persistent: req.Remember},
			Exp: time.Now().Add(passportMfaTTL),
		})
		return &v1.PassportLoginResponse{MfaRequired: true, MfaToken: token}, nil
	}
	return uc.issue(ctx, u, req.DeviceId, req.Device, method, req.Remember)
}

// LoginMfa 两步验证：验证码或恢复码。
func (uc *PassportUsecase) LoginMfa(ctx context.Context, req *v1.PassportLoginMfaRequest) (*v1.PassportLoginResponse, error) {
	ch, ok := uc.mfaChallenges.Get(req.MfaToken)
	if !ok || !ch.alive() {
		return nil, ErrPassportMfaExpired
	}
	u, err := uc.repo.GetUser(ctx, ch.V.UserID)
	if err != nil {
		return nil, ErrPassportMfaExpired
	}
	if u.Status != passportuser.StatusActive {
		return nil, ErrPassportDisabled
	}
	if err := uc.checkSecondFactor(ctx, u, req.Code, req.RecoveryCode); err != nil {
		ch.V.Attempts++
		if ch.V.Attempts >= passportLockThreshold {
			uc.mfaChallenges.Del(req.MfaToken)
			return nil, ErrPassportMfaExpired
		}
		uc.mfaChallenges.Set(req.MfaToken, ch)
		uc.recordLogin(ctx, u.ID, ch.V.DeviceID, ch.V.Device, ch.V.Method, false, "mfa_invalid")
		return nil, err
	}
	uc.mfaChallenges.Del(req.MfaToken)
	return uc.issue(ctx, u, ch.V.DeviceID, ch.V.Device, ch.V.Method, ch.V.Persistent)
}

// checkSecondFactor 校验身份验证器验证码（拒绝重放）或恢复码（用过即删）。
func (uc *PassportUsecase) checkSecondFactor(ctx context.Context, u *gen.PassportUser, code, recovery string) error {
	if u.TotpSecret == "" || u.TotpEnabledAt == nil {
		return ErrPassportTotpDisabled
	}
	if strings.TrimSpace(code) != "" {
		secret, err := gameBox().Decrypt(u.TotpSecret)
		if err != nil {
			return ErrSystem(err)
		}
		step, ok := totp.Verify(secret, code, time.Now(), 1)
		if !ok {
			return ErrPassportMfaInvalid
		}
		if last, used := uc.totpUsed.Get(u.ID); used && step <= last {
			return ErrPassportMfaInvalid
		}
		uc.totpUsed.Set(u.ID, step)
		return nil
	}
	recovery = totp.NormalizeRecoveryCode(recovery)
	if recovery == "" {
		return ErrPassportMfaInvalid
	}
	for i, hash := range u.RecoveryCodes {
		if auth2.VerifyPassword(recovery, hash) {
			u.RecoveryCodes = append(append([]string{}, u.RecoveryCodes[:i]...), u.RecoveryCodes[i+1:]...)
			if _, err := uc.repo.SaveUser(ctx, u); err != nil {
				return ErrSystem(err)
			}
			return nil
		}
	}
	return ErrPassportMfaInvalid
}

func randomToken(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

// sessionTTL 会话有效期：保持登录按设置的天数，否则 1 天。
func (uc *PassportUsecase) sessionTTL(ctx context.Context, persistent bool) time.Duration {
	if !persistent {
		return passportShortSession
	}
	return time.Duration(uc.Settings(ctx).SessionDays) * 24 * time.Hour
}

// issue 创建会话并签发令牌；记录登录、新设备提醒。
func (uc *PassportUsecase) issue(ctx context.Context, u *gen.PassportUser, deviceID, device, method string, persistent bool) (*v1.PassportLoginResponse, error) {
	deviceID = strings.TrimSpace(deviceID)
	if deviceID == "" {
		deviceID = uuid.NewString()
	}
	device = strings.TrimSpace(device)
	if len([]rune(device)) > 120 {
		device = string([]rune(device)[:120])
	}
	accessNoise, err := passporttoken.NewNoise()
	if err != nil {
		return nil, ErrSystem(err)
	}
	refreshNoise, err := passporttoken.NewNoise()
	if err != nil {
		return nil, ErrSystem(err)
	}
	client := PassportClientOf(ctx)
	anyLogin, sameDevice, historyErr := uc.repo.LoginHistory(ctx, u.ID, deviceID)
	now := time.Now()
	sess, err := uc.repo.CreateSession(ctx, &gen.PassportSession{
		ID:           uuid.NewString(),
		UserID:       u.ID,
		DeviceID:     deviceID,
		Device:       device,
		UserAgent:    client.UserAgent,
		LoginMethod:  method,
		IP:           client.IP,
		LastIP:       client.IP,
		LastActiveAt: &now,
		AccessNoise:  accessNoise,
		RefreshNoise: refreshNoise,
		Persistent:   persistent,
		ExpiresAt:    now.Add(uc.sessionTTL(ctx, persistent)),
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	uc.recordLogin(ctx, u.ID, deviceID, device, method, true, "")
	if historyErr == nil && anyLogin && !sameDevice {
		uc.notify(ctx, u.ID, PassportMsgLoginNewDevice, "", map[string]string{
			"device":      device,
			"device_type": DeviceType(client.UserAgent, device),
			"platform":    Platform(client.UserAgent),
			"ip":          client.IP,
			"ip_scope":    IPScope(client.IP),
			"method":      method,
		})
	}
	u.LastLoginAt, u.LastLoginIP, u.LastLoginDevice = &now, client.IP, passportFirst(device, Platform(client.UserAgent))
	if _, err := uc.repo.SaveUser(ctx, u); err != nil {
		uc.log.Warnf("touch passport last login: %v", err)
	}
	return uc.tokens(sess)
}

func passportFirst(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func (uc *PassportUsecase) tokens(sess *gen.PassportSession) (*v1.PassportLoginResponse, error) {
	access, err := passporttoken.Sign(auth2.AuthSecretKey, sess.UserID, sess.ID, sess.DeviceID, passporttoken.KindAccess, sess.AccessNoise, PassportAccessTTL)
	if err != nil {
		return nil, ErrSystem(err)
	}
	refreshTTL := time.Until(sess.ExpiresAt)
	if refreshTTL <= 0 {
		return nil, ErrPassportUnauthorized
	}
	refresh, err := passporttoken.Sign(auth2.AuthSecretKey, sess.UserID, sess.ID, sess.DeviceID, passporttoken.KindRefresh, sess.RefreshNoise, refreshTTL)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.PassportLoginResponse{AccessToken: access, RefreshToken: refresh, ExpiresIn: int64(PassportAccessTTL.Seconds())}, nil
}

// Refresh 用刷新令牌续期同一会话（两份噪声都换新，旧令牌立即失效）。
func (uc *PassportUsecase) Refresh(ctx context.Context, token string) (*v1.PassportLoginResponse, error) {
	claims, err := passporttoken.Parse(auth2.AuthSecretKey, strings.TrimSpace(token))
	if err != nil || claims.Kind != passporttoken.KindRefresh {
		return nil, ErrPassportUnauthorized
	}
	sess, err := uc.repo.GetSession(ctx, claims.SessionID)
	if err != nil || sess.UserID != claims.UserID || sess.DeviceID != claims.DeviceID ||
		sess.RefreshNoise == "" || sess.RefreshNoise != claims.Noise || time.Now().After(sess.ExpiresAt) {
		return nil, ErrPassportUnauthorized
	}
	u, err := uc.repo.GetUser(ctx, sess.UserID)
	if err != nil || u.Status != passportuser.StatusActive {
		return nil, ErrPassportUnauthorized
	}
	accessNoise, err := passporttoken.NewNoise()
	if err != nil {
		return nil, ErrSystem(err)
	}
	refreshNoise, err := passporttoken.NewNoise()
	if err != nil {
		return nil, ErrSystem(err)
	}
	sess, err = uc.repo.RotateSession(ctx, sess.ID, accessNoise, refreshNoise, time.Now().Add(uc.sessionTTL(ctx, sess.Persistent)))
	if err != nil {
		return nil, ErrSystem(err)
	}
	return uc.tokens(sess)
}

// Authenticate 校验访问令牌：签名与声明、会话噪声、账号状态（通行证 server 的鉴权过滤器调用）。
func (uc *PassportUsecase) Authenticate(ctx context.Context, token string) (*passporttoken.Claims, error) {
	claims, err := passporttoken.Parse(auth2.AuthSecretKey, token)
	if err != nil || claims.Kind != passporttoken.KindAccess {
		return nil, ErrPassportUnauthorized
	}
	sess, err := uc.repo.GetSession(ctx, claims.SessionID)
	if err != nil || sess.UserID != claims.UserID || sess.DeviceID != claims.DeviceID ||
		sess.AccessNoise == "" || sess.AccessNoise != claims.Noise || time.Now().After(sess.ExpiresAt) {
		return nil, ErrPassportUnauthorized
	}
	u, err := uc.repo.GetUser(ctx, claims.UserID)
	if err != nil || u.Status != passportuser.StatusActive {
		return nil, ErrPassportUnauthorized
	}
	uc.repo.TouchSession(ctx, sess.ID, PassportClientOf(ctx).IP)
	return claims, nil
}

// Logout 退出当前会话。
func (uc *PassportUsecase) Logout(ctx context.Context, c *passporttoken.Claims) error {
	if err := uc.repo.DeleteSession(ctx, c.UserID, c.SessionID); err != nil && !gen.IsNotFound(err) {
		return ErrSystem(err)
	}
	return nil
}

// ResetPassword 用邮箱验证码重设密码，全部设备下线。
func (uc *PassportUsecase) ResetPassword(ctx context.Context, req *v1.PassportResetPasswordRequest) error {
	email, err := normalizeEmail(req.Email)
	if err != nil {
		return err
	}
	if err := validPassportPassword(req.Password); err != nil {
		return err
	}
	u, err := uc.repo.FindUserByEmail(ctx, email)
	if err != nil || !u.EmailVerified {
		if err != nil && !gen.IsNotFound(err) {
			return ErrSystem(err)
		}
		return ErrPassportCodeInvalid
	}
	if u.Status != passportuser.StatusActive {
		return ErrPassportDisabled
	}
	if err := uc.verifyCode(passportCodeReset, email, req.Code); err != nil {
		return err
	}
	hash, err := auth2.HashPassword(req.Password)
	if err != nil {
		return ErrSystem(err)
	}
	now := time.Now()
	u.Password, u.PasswordUpdatedAt = hash, &now
	if _, err := uc.repo.SaveUser(ctx, u); err != nil {
		return ErrSystem(err)
	}
	if err := uc.repo.DeleteSessions(ctx, u.ID, ""); err != nil {
		return ErrSystem(err)
	}
	client := PassportClientOf(ctx)
	uc.notify(ctx, u.ID, PassportMsgPasswordReset, "", map[string]string{"ip": client.IP, "ip_scope": IPScope(client.IP)})
	return nil
}

// notify 写一条消息；失败只记日志。
func (uc *PassportUsecase) notify(ctx context.Context, userID, kind, serverID string, params map[string]string) {
	if userID == "" {
		return
	}
	if params == nil {
		params = map[string]string{}
	}
	if err := uc.repo.AddMessages(context.WithoutCancel(ctx), []*gen.PassportMessage{{
		ID:       NewMessageID(),
		UserID:   userID,
		Kind:     kind,
		ServerID: serverID,
		Params:   params,
	}}); err != nil {
		uc.log.Warnf("add passport message: %v", err)
	}
}
