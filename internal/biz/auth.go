package biz

import (
	"context"
	"fmt"
	"momoko/internal/data/ent/gen/user"
	"momoko/pkg/utils"
	"net/mail"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	auth2 "momoko/pkg/auth"
	"momoko/pkg/cache"
)

const emailCodeLength = 6

// 最近登录方式（users.last_login_method）。
const (
	LoginMethodPassword  = "password"
	LoginMethodEmailCode = "email_code"
)

// EmailCodeTTL 邮箱验证码有效期。
const EmailCodeTTL = 5 * time.Minute

// Auth 创建/轮换会话时的输入。
type Auth struct {
	UserID       string
	DeviceID     string
	Device       string
	SessionID    string
	IP           string
	UserAgent    string
	LoginMethod  string
	AccessNoise  string
	RefreshNoise string
}

type EmailCodeType string

const (
	EmailCodeTypeRegister EmailCodeType = "register"
	EmailCodeTypeLogin    EmailCodeType = "login"
	EmailCodeTypeReset    EmailCodeType = "reset"
)

type AuthRepo interface {
	CreateAuth(context.Context, *Auth) (*gen.Session, error)
	Refresh(ctx context.Context, sessionID, accessNoise, refreshNoise string) (*gen.Session, error)
	ListAuth(ctx context.Context, userId string) ([]*gen.Session, error)
	GetAuth(ctx context.Context, sessionID string) (*gen.Session, error)
	DeleteAuth(ctx context.Context, userID string, sessionID *string) error
	// TouchSession 记录会话最近活跃时间与 IP（实现方自行节流）。
	TouchSession(ctx context.Context, sessionID, ip string)
	// LoginHistory 返回该用户此前是否有过成功登录，以及是否在该设备上成功登录过。
	LoginHistory(ctx context.Context, userID, deviceID string) (anyLogin bool, sameDevice bool, err error)

	MessageRepo
}

type AuthUsecase struct {
	auth       AuthRepo
	user       UserRepo
	emailCodes *cache.Cache[string, string]
	// failNotified 登录失败提醒节流：同一用户 10 分钟内只发一条。
	failNotified *cache.Cache[string, time.Time]
}

func NewAuthUsecase(auth AuthRepo, user UserRepo) *AuthUsecase {
	return &AuthUsecase{
		auth:       auth,
		user:       user,
		emailCodes:   cache.New[string, string](EmailCodeTTL),
		failNotified: cache.New[string, time.Time](loginFailNotifyInterval),
	}
}

func (a *AuthUsecase) LoginByUsername(ctx context.Context, username, password string) (*gen.User, error) {
	userInfo, err := a.user.FindByName(ctx, username)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, ErrSystem(err)
	}
	MarkLoginTrace(ctx, userInfo.ID)
	if !auth2.VerifyPassword(password, userInfo.Password) {
		a.notifyLoginFailed(ctx, userInfo.ID, LoginMethodPassword)
		return nil, ErrInvalidPassword
	}
	if userInfo.Status != user.StatusActive {
		return nil, ErrUserInactive
	}
	return userInfo, nil
}

func (a *AuthUsecase) LoginByEmail(ctx context.Context, email, code string) (*gen.User, error) {
	userInfo, err := a.user.FindByEmail(ctx, email)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAdminNotFound
		}
		return nil, ErrSystem(err)
	}
	MarkLoginTrace(ctx, userInfo.ID)
	if userInfo.Status != user.StatusActive {
		return nil, ErrUserInactive
	}
	if err = a.VerifyEmailCode(email, code, EmailCodeTypeLogin); err != nil {
		a.notifyLoginFailed(ctx, userInfo.ID, LoginMethodEmailCode)
		return nil, err
	}
	return userInfo, nil
}

func (a *AuthUsecase) Register(ctx context.Context, req *v1.RegisterRequest, registerEmailVerificationRequired bool) (*gen.User, error) {
	if req.Username == "" {
		return nil, ErrUsernameEmpty
	}
	if req.Password == "" {
		return nil, ErrPasswordEmpty
	}
	if req.Email == "" {
		return nil, ErrEmailEmpty
	}
	if _, err := mail.ParseAddress(req.GetEmail()); err != nil {
		return nil, ErrEmailInvalid
	}

	if _, err := a.user.FindByName(ctx, req.Username); err == nil {
		return nil, ErrUsernameRegistered
	} else if !gen.IsNotFound(err) {
		return nil, ErrSystem(err)
	}
	if _, err := a.user.FindByEmail(ctx, req.Email); err == nil {
		return nil, ErrEmailRegistered
	} else if !gen.IsNotFound(err) {
		return nil, ErrSystem(err)
	}
	if registerEmailVerificationRequired {
		if err := a.VerifyEmailCode(req.Email, req.Code, EmailCodeTypeRegister); err != nil {
			return nil, err
		}
	}

	passwordHash, err := auth2.HashPassword(req.Password)
	if err != nil {
		return nil, ErrSystem(err)
	}
	userInfo, err := a.user.CreateUser(ctx, &gen.User{
		ID:       fmt.Sprintf("user_z:%06d_%s", time.Now().Unix()%1000000, uuid.NewString()[:8]),
		Username: req.Username,
		Password: passwordHash,
		Email:    req.Email,
		Status:   user.StatusActive,
		Name:     req.Username,
	}, "")
	if err != nil {
		if gen.IsConstraintError(err) {
			return nil, ErrUsernameEmailRegistered
		}
		return nil, ErrSystem(err)
	}
	return userInfo, nil
}

func (a *AuthUsecase) NewEmailCode(email string, codeType EmailCodeType) (string, error) {
	key := emailCodeCacheKey(codeType, email)

	code, err := utils.GenerateEmailCode(emailCodeLength)
	if err != nil {
		return "", ErrSystem(err)
	}
	a.emailCodes.Set(key, code)
	return code, nil
}

func (a *AuthUsecase) VerifyEmailCode(email string, code string, codeType EmailCodeType) error {
	key := emailCodeCacheKey(codeType, email)
	cachedCode, ok := a.emailCodes.Get(key)
	if !ok || cachedCode != strings.TrimSpace(code) {
		return ErrEmailCodeInvalid
	}
	a.emailCodes.Del(key)
	return nil
}

// NewSession 为设备创建/覆盖一行登录会话（含两份随机噪声）。
func (a *AuthUsecase) NewSession(ctx context.Context, userId string, req *v1.LoginRequest) (*gen.Session, error) {
	accessNoise, err := auth2.NewNoise()
	if err != nil {
		return nil, ErrSystem(err)
	}
	refreshNoise, err := auth2.NewNoise()
	if err != nil {
		return nil, ErrSystem(err)
	}
	device := strings.TrimSpace(req.Device)
	if device == "" {
		device = "unknown"
	}
	method := LoginMethodPassword
	if _, ok := req.GetIdentity().(*v1.LoginRequest_Email); ok {
		method = LoginMethodEmailCode
	}
	info := &Auth{
		UserID:       userId,
		DeviceID:     req.DeviceId,
		IP:           utils.ClientIPFromContext(ctx),
		UserAgent:    userAgentFromContext(ctx),
		LoginMethod:  method,
		Device:       device,
		SessionID:    uuid.NewString(),
		AccessNoise:  accessNoise,
		RefreshNoise: refreshNoise,
	}
	// 新设备提醒：此前登录过、但从未在这台设备上成功登录。查询失败不影响登录。
	anyLogin, sameDevice, historyErr := a.auth.LoginHistory(ctx, userId, req.DeviceId)
	ea, err := a.auth.CreateAuth(ctx, info)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if historyErr == nil && anyLogin && !sameDevice {
		a.notify(ctx, userId, MessageCategorySecurity, MessageKindLoginNewDevice, map[string]string{
			"device":      info.Device,
			"device_type": DeviceType(info.UserAgent, info.Device),
			"ip":          info.IP,
			"ip_scope":    IPScope(info.IP),
			"method":      method,
		})
	}
	// 最近登录时间仅作展示，写失败不影响登录。
	_ = a.user.TouchLastLogin(ctx, userId, info.IP, method)
	return ea, nil
}

// ResetPassword 通过邮箱验证码重置密码，并让该用户的全部登录会话失效。
func (a *AuthUsecase) ResetPassword(ctx context.Context, email, code, newPassword string) error {
	email = strings.TrimSpace(email)
	if email == "" {
		return ErrEmailEmpty
	}
	if newPassword == "" {
		return ErrPasswordEmpty
	}
	userInfo, err := a.user.FindByEmail(ctx, email)
	if err != nil {
		if gen.IsNotFound(err) {
			return ErrAdminNotFound
		}
		return ErrSystem(err)
	}
	if userInfo.Status != user.StatusActive {
		return ErrUserInactive
	}
	if err = a.VerifyEmailCode(email, code, EmailCodeTypeReset); err != nil {
		return err
	}
	passwordHash, err := auth2.HashPassword(newPassword)
	if err != nil {
		return ErrSystem(err)
	}
	if _, err = a.user.UpdatePassword(ctx, userInfo.ID, passwordHash); err != nil {
		return ErrSystem(err)
	}
	if err = a.auth.DeleteAuth(ctx, userInfo.ID, nil); err != nil {
		return ErrSystem(err)
	}
	ip := utils.ClientIPFromContext(ctx)
	a.notify(ctx, userInfo.ID, MessageCategorySecurity, MessageKindPasswordReset, map[string]string{
		"ip":       ip,
		"ip_scope": IPScope(ip),
	})
	return nil
}

// LogoutOthers 下线该用户除 keepSessionID 外的全部会话。
func (a *AuthUsecase) LogoutOthers(ctx context.Context, userID, keepSessionID string) error {
	sessions, err := a.auth.ListAuth(ctx, userID)
	if err != nil {
		return ErrSystem(err)
	}
	for _, sess := range sessions {
		if sess.ID == keepSessionID {
			continue
		}
		id := sess.ID
		if err := a.auth.DeleteAuth(ctx, userID, &id); err != nil {
			return ErrSystem(err)
		}
	}
	return nil
}

// VerifyToken 校验 JWT claims 与库内会话一致，并匹配对应 kind 的噪声。
func (a *AuthUsecase) VerifyToken(ctx context.Context, claims *auth2.Auth, kind auth2.TokenKind) bool {
	if claims == nil || claims.Kind != kind {
		return false
	}
	info, err := a.auth.GetAuth(ctx, claims.SessionID)
	if err != nil {
		return false
	}
	if info.DeviceID != claims.DeviceId ||
		info.UserID != claims.UserID ||
		info.ID != claims.SessionID {
		return false
	}
	switch kind {
	case auth2.TokenKindAccess:
		return info.AccessNoise != "" && info.AccessNoise == claims.Noise
	case auth2.TokenKindRefresh:
		if info.ExpiresAt.Before(time.Now()) {
			return false
		}
		return info.RefreshNoise != "" && info.RefreshNoise == claims.Noise
	default:
		return false
	}
}

// RefreshToken 续签同一会话：固定 expires_at = now+有效期，更新两份 noise，不换 session_id。
// 旧 token/rt 因 noise 变更立即失效；随后用同一 session 重新签发 access + refresh。
func (a *AuthUsecase) RefreshToken(ctx context.Context, sessionID string) (*gen.Session, error) {
	accessNoise, err := auth2.NewNoise()
	if err != nil {
		return nil, ErrSystem(err)
	}
	refreshNoise, err := auth2.NewNoise()
	if err != nil {
		return nil, ErrSystem(err)
	}
	session, err := a.auth.Refresh(ctx, sessionID, accessNoise, refreshNoise)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return session, nil
}

func (a *AuthUsecase) ListLoginDevice(ctx context.Context, userId string) ([]*v1.LoginDevice, error) {
	auths, err := a.auth.ListAuth(ctx, userId)
	if err != nil {
		return nil, ErrSystem(err)
	}
	list := make([]*v1.LoginDevice, 0, len(auths))
	for _, authInfo := range auths {
		lastIP := authInfo.LastIP
		if lastIP == "" {
			lastIP = authInfo.IP
		}
		lastActive := authInfo.CreateTime
		if authInfo.LastActiveAt != nil && authInfo.LastActiveAt.After(lastActive) {
			lastActive = *authInfo.LastActiveAt
		}
		list = append(list, &v1.LoginDevice{
			LoginTime:      timestamppb.New(authInfo.CreateTime),
			Device:         authInfo.Device,
			Ip:             authInfo.IP,
			DeviceId:       authInfo.DeviceID,
			SessionId:      authInfo.ID,
			UpdateTime:     timestamppb.New(authInfo.UpdateTime),
			UserAgent:      authInfo.UserAgent,
			DeviceType:     DeviceType(authInfo.UserAgent, authInfo.Device),
			IpScope:        IPScope(lastIP),
			LastActiveTime: timestamppb.New(lastActive),
			LastIp:         lastIP,
			LoginMethod:    authInfo.LoginMethod,
			ExpireTime:     timestamppb.New(authInfo.ExpiresAt),
		})
	}
	// 最近活跃的排前面。
	sort.SliceStable(list, func(i, j int) bool {
		return list[i].LastActiveTime.AsTime().After(list[j].LastActiveTime.AsTime())
	})
	return list, nil
}

func (a *AuthUsecase) Logout(ctx context.Context, userID string, sessionID *string) error {
	err := a.auth.DeleteAuth(ctx, userID, sessionID)
	if err != nil {
		return ErrSystem(err)
	}
	return nil
}

func emailCodeCacheKey(codeType EmailCodeType, email string) string {
	return fmt.Sprintf("%s:%s", codeType, strings.ToLower(email))
}
