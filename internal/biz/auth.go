package biz

import (
	"context"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	"momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/auth"
	auth2 "momoko/pkg/auth"
)

type Auth struct {
	UserID    string
	DeviceID  string
	Device    string
	SessionID string
	IP        string
	Type      auth.Type
}

type AuthRepo interface {
	CreateAuth(context.Context, *Auth) (*gen.Auth, error)
	Refresh(ctx context.Context, userId, deviceId string) (*gen.Auth, *gen.Auth, error)
	ListAuth(ctx context.Context, tokenType *auth.Type, userId string) ([]*gen.Auth, error)
	GetAuth(ctx context.Context, sessionID string, tokenType auth.Type) (*gen.Auth, error)
	GetAuthByDeviceID(ctx context.Context, deviceID string, tokenType auth.Type) (*gen.Auth, error)
	DeleteAuth(ctx context.Context, userID string, deviceID *string) error
}

type AuthUsecase struct {
	auth AuthRepo
}

func NewAuthUsecase(auth AuthRepo) *AuthUsecase {
	return &AuthUsecase{auth: auth}
}

func (a *AuthUsecase) NewAccessToken(ctx context.Context, userId string, req *v1.LoginRequest) (*gen.Auth, error) {
	info := &Auth{
		UserID:    userId,
		DeviceID:  req.DeviceId,
		IP:        req.Ip,
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
		IP:        req.Ip,
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
