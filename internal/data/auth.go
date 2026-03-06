package data

import (
	"context"

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

	create.
		OnConflict().
		UpdateNewValues().
		UpdateSessionID().
		UpdateIP()

	return create.Save(ctx)
}

func (ar *authRepo) GetAuth(ctx context.Context, sessionID string, tokenType auth.Type) (*ent.Auth, error) {
	return ar.data.db.Auth.Query().
		Where(
			auth.SessionIDEQ(sessionID),
			auth.TypeEQ(tokenType),
		).First(ctx)
}

func (ar *authRepo) ListAuth(ctx context.Context, tokenType *auth.Type) ([]*ent.Auth, error) {
	query := ar.data.db.Auth.Query()

	if tokenType != nil {
		query.Where(auth.TypeEQ(*tokenType))
	}

	return query.All(ctx)
}
