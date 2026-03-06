package data

import (
	"context"

	"github.com/google/uuid"

	"momoko/internal/biz"
	"momoko/internal/data/ent"
	"momoko/internal/data/ent/auth"
)

type authRepo struct {
	data *Data
}

func NewAuthRepo(data *Data) biz.AuthRepo {
	return &authRepo{
		data: data,
	}
}

func (ar *authRepo) CreateAuth(ctx context.Context, auth *biz.Auth) (*ent.Auth, error) {
	create := ar.data.db.Auth.Create().
		SetSessionID(auth.SessionID).
		SetUserID(auth.UserID).
		SetDeviceID(auth.DeviceID).
		SetDevice(auth.Device).
		SetIP(auth.IP).
		SetType(auth.Type)

	return create.Save(ctx)
}

func (ar *authRepo) Refresh(ctx context.Context, userId string) (*ent.Auth, *ent.Auth, error) {
	authInfos, err := ar.data.db.Auth.Query().Where(auth.UserIDEQ(userId)).All(ctx)
	if err != nil {
		return nil, nil, err
	}
	var (
		access, refresh *ent.Auth
	)
	for _, item := range authInfos {
		authInfo, err := ar.data.db.Auth.UpdateOne(item).
			SetSessionID(uuid.NewString()).Save(ctx)
		if err != nil {
			return nil, nil, biz.ErrSystem(err)
		}
		switch item.Type {
		case auth.TypeToken:
			access = authInfo
		case auth.TypeRefreshToken:
			refresh = authInfo
		default:
			return nil, nil, biz.ErrTokenInvalid
		}
	}

	return access, refresh, nil
}

func (ar *authRepo) GetAuth(ctx context.Context, sessionID string, tokenType auth.Type) (*ent.Auth, error) {
	return ar.data.db.Auth.Query().
		Where(
			auth.SessionIDEQ(sessionID),
			auth.TypeEQ(tokenType),
		).First(ctx)
}

func (ar *authRepo) ListAuth(ctx context.Context, tokenType *auth.Type, userId string) ([]*ent.Auth, error) {
	query := ar.data.db.Auth.Query().
		Where(auth.UserIDEQ(userId))

	if tokenType != nil {
		query.Where(auth.TypeEQ(*tokenType))
	}

	return query.All(ctx)
}
