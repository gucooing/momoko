package biz

import (
	"context"
	"fmt"
	"momoko/internal/data/ent/gen/user"
	"momoko/pkg/tools"
	"momoko/pkg/utils"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/auth"
	auth2 "momoko/pkg/auth"
	"momoko/pkg/cache"
)

const emailCodeLength = 6

type Auth struct {
	UserID    string
	DeviceID  string
	Device    string
	SessionID string
	IP        string
	Type      auth.Type
}

type EmailCodeType string

const (
	EmailCodeTypeRegister EmailCodeType = "register"
	EmailCodeTypeLogin    EmailCodeType = "login"
)

type AuthRepo interface {
	CreateAuth(context.Context, *Auth) (*gen.Auth, error)
	Refresh(ctx context.Context, userId, deviceId string) (*gen.Auth, *gen.Auth, error)
	ListAuth(ctx context.Context, tokenType *auth.Type, userId string) ([]*gen.Auth, error)
	GetAuth(ctx context.Context, sessionID string, tokenType auth.Type) (*gen.Auth, error)
	GetAuthByDeviceID(ctx context.Context, deviceID string, tokenType auth.Type) (*gen.Auth, error)
	DeleteAuth(ctx context.Context, userID string, deviceID *string) error
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
	if userInfo.Password != auth2.EncodePassword(password) {
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

	userInfo, err := a.user.CreateUser(ctx, &gen.User{
		ID:       fmt.Sprintf("user_z:%06d_%s", time.Now().Unix()%1000000, uuid.NewString()[:8]),
		Username: req.Username,
		Password: auth2.EncodePassword(req.Password),
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

func (a *AuthUsecase) NewAccessToken(ctx context.Context, userId string, req *v1.LoginRequest) (*gen.Auth, error) {
	info := &Auth{
		UserID:    userId,
		DeviceID:  req.DeviceId,
		IP:        tools.ClientIPFromContext(ctx),
		Device:    req.Device,
		SessionID: uuid.NewString(),
		Type:      auth.TypeToken,
	}
	ea, err := a.auth.CreateAuth(ctx, info)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return ea, nil
}

func (a *AuthUsecase) NewRefreshToken(ctx context.Context, userId string, req *v1.LoginRequest) (*gen.Auth, error) {
	info := &Auth{
		UserID:    userId,
		DeviceID:  req.DeviceId,
		IP:        tools.ClientIPFromContext(ctx),
		Device:    req.Device,
		SessionID: uuid.NewString(),
		Type:      auth.TypeRefreshToken,
	}
	ea, err := a.auth.CreateAuth(ctx, info)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return ea, nil
}

func (a *AuthUsecase) VerifyToken(ctx context.Context, auth *auth2.Auth, tokenType auth.Type) bool {
	info, err := a.auth.GetAuth(ctx, auth.SessionID, tokenType)
	if err != nil {
		return false
	}
	if info.DeviceID != auth.DeviceId ||
		auth.SessionID != info.SessionID {
		return false
	}
	return true
}

func (a *AuthUsecase) RefreshToken(ctx context.Context, userId, deviceId string) (*gen.Auth, *gen.Auth, error) {
	access, refresh, err := a.auth.Refresh(ctx, userId, deviceId)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	return access, refresh, nil
}

func (a *AuthUsecase) ListLoginDevice(ctx context.Context, userId string) ([]*v1.LoginDevice, error) {
	auths, err := a.auth.ListAuth(ctx, new(auth.TypeRefreshToken), userId)
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
			SessionId:  authInfo.SessionID,
			UpdateTime: timestamppb.New(authInfo.UpdateTime),
		})
	}
	return list, nil
}

func (a *AuthUsecase) Logout(ctx context.Context, userID string, deviceID string) error {
	err := a.auth.DeleteAuth(ctx, userID, &deviceID)
	if err != nil {
		return ErrSystem(err)
	}
	return nil
}

func emailCodeCacheKey(codeType EmailCodeType, email string) string {
	return fmt.Sprintf("%s:%s", codeType, strings.ToLower(email))
}
