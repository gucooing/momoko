package biz

import (
	"context"
	"fmt"
	"momoko/internal/data/ent/gen/user"
	"momoko/pkg/utils"
	"net/mail"
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

// Auth 创建/轮换会话时的输入。
type Auth struct {
	UserID       string
	DeviceID     string
	Device       string
	SessionID    string
	IP           string
	AccessNoise  string
	RefreshNoise string
}

type EmailCodeType string

const (
	EmailCodeTypeRegister EmailCodeType = "register"
	EmailCodeTypeLogin    EmailCodeType = "login"
)

type AuthRepo interface {
	CreateAuth(context.Context, *Auth) (*gen.Session, error)
	Refresh(ctx context.Context, sessionID, accessNoise, refreshNoise string) (*gen.Session, error)
	ListAuth(ctx context.Context, userId string) ([]*gen.Session, error)
	GetAuth(ctx context.Context, sessionID string) (*gen.Session, error)
	DeleteAuth(ctx context.Context, userID string, sessionID *string) error
}

type AuthUsecase struct {
	auth       AuthRepo
	user       UserRepo
	emailCodes *cache.Cache[string, string]
}

func NewAuthUsecase(auth AuthRepo, user UserRepo) *AuthUsecase {
	return &AuthUsecase{
		auth:       auth,
		user:       user,
		emailCodes: cache.New[string, string](5 * time.Minute),
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
	if !auth2.VerifyPassword(password, userInfo.Password) {
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
	if userInfo.Status != user.StatusActive {
		return nil, ErrUserInactive
	}
	if err = a.VerifyEmailCode(email, code, EmailCodeTypeLogin); err != nil {
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
	info := &Auth{
		UserID:       userId,
		DeviceID:     req.DeviceId,
		IP:           utils.ClientIPFromContext(ctx),
		Device:       req.Device,
		SessionID:    uuid.NewString(),
		AccessNoise:  accessNoise,
		RefreshNoise: refreshNoise,
	}
	ea, err := a.auth.CreateAuth(ctx, info)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return ea, nil
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
		list = append(list, &v1.LoginDevice{
			LoginTime:  timestamppb.New(authInfo.CreateTime),
			Device:     authInfo.Device,
			Ip:         authInfo.IP,
			DeviceId:   authInfo.DeviceID,
			SessionId:  authInfo.ID,
			UpdateTime: timestamppb.New(authInfo.UpdateTime),
		})
	}
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
