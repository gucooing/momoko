package data

import (
	"context"

	"momoko/internal/biz"
	"momoko/internal/data/ent"
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
	create := ar.data.db.Auth.Create()

	create.SetSessionID(auth.SessionID)
	create.SetUserID(auth.UserID)
	create.SetDeviceID(auth.DeviceID)
	create.SetIP(auth.IP)
	create.SetType(auth.Type)

	return create.Save(ctx)
}
