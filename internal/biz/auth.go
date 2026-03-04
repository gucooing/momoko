package biz

import (
	"context"

	"github.com/google/uuid"

	"momoko/internal/data/ent"
	"momoko/internal/data/ent/auth"
)

type Auth struct {
	UserID    string
	DeviceID  string
	SessionID string
	IP        string
	Type      auth.Type
}

type AuthRepo interface {
	CreateAuth(context.Context, *Auth) (*ent.Auth, error)
}

type AuthUsecase struct {
	auth AuthRepo
}

func NewAuthUsecase(auth AuthRepo) *AuthUsecase {
	return &AuthUsecase{auth: auth}
}

func (a *AuthUsecase) NewAccessToken(ctx context.Context, userId, deviceId, ip string) (*ent.Auth, error) {
	info := &Auth{
		UserID:    userId,
		DeviceID:  deviceId,
		SessionID: uuid.NewString(),
		IP:        ip,
		Type:      auth.TypeToken,
	}
	ea, err := a.auth.CreateAuth(ctx, info)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return ea, nil
}

func (a *AuthUsecase) NewRefreshToken(ctx context.Context, userId, deviceId, ip string) (*ent.Auth, error) {
	info := &Auth{
		UserID:    userId,
		DeviceID:  deviceId,
		SessionID: uuid.NewString(),
		IP:        ip,
		Type:      auth.TypeRefreshToken,
	}
	ea, err := a.auth.CreateAuth(ctx, info)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return ea, nil
}
