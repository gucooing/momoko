package data

import (
	"context"
	"time"

	"github.com/google/uuid"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	entauth "momoko/internal/data/ent/gen/auth"
	tokenauth "momoko/pkg/auth"
	"momoko/pkg/cache"
)

type authRepo struct {
	data *Data

	cacheToken *cache.Cache[string, *gen.Auth]
}

func NewAuthRepo(data *Data) biz.AuthRepo {
	return &authRepo{
		data:       data,
		cacheToken: cache.New[string, *gen.Auth](5 * time.Minute),
	}
}

func (ar *authRepo) CreateAuth(ctx context.Context, authInfo *biz.Auth) (*gen.Auth, error) {
	now := time.Now()
	err := ar.data.db.Auth.
		Create().
		SetSessionID(authInfo.SessionID).
		SetUserID(authInfo.UserID).
		SetDeviceID(authInfo.DeviceID).
		SetDevice(authInfo.Device).
		SetIP(authInfo.IP).
		SetType(authInfo.Type).
		SetExpiresAt(now.Add(tokenauth.TokenExpiresIn(authInfo.Type))).
		OnConflictColumns(entauth.FieldDeviceID, entauth.FieldType).
		UpdateNewValues().
		Exec(ctx)
	if err != nil {
		return nil, err
	}
	authData, err := ar.data.db.Auth.
		Query().
		Where(
			entauth.DeviceIDEQ(authInfo.DeviceID),
			entauth.TypeEQ(authInfo.Type),
		).
		Only(ctx)
	if err != nil {
		return nil, err
	}
	if authInfo.Type == entauth.TypeToken {
		ar.cacheToken.Set(authInfo.DeviceID, authData)
	}
	return authData, nil
}

func (ar *authRepo) Refresh(ctx context.Context, userId, deviceId string) (*gen.Auth, *gen.Auth, error) {
	rows, err := ar.data.db.Auth.Query().
		Where(
			entauth.UserIDEQ(userId),
			entauth.DeviceIDEQ(deviceId),
		).All(ctx)
	if err != nil {
		return nil, nil, err
	}

	var (
		accessAuth  *gen.Auth
		refreshAuth *gen.Auth
	)
	for _, row := range rows {
		switch row.Type {
		case entauth.TypeToken:
			accessAuth = row
		case entauth.TypeRefreshToken:
			refreshAuth = row
		}
	}

	if accessAuth == nil || refreshAuth == nil {
		return nil, nil, biz.ErrTokenInvalid
	}

	now := time.Now()
	sessionID := uuid.NewString()
	access, err := ar.data.db.Auth.UpdateOneID(accessAuth.ID).
		SetExpiresAt(now.Add(tokenauth.TokenExpiresIn(entauth.TypeToken))).
		SetSessionID(sessionID).
		Save(ctx)
	if err != nil {
		return nil, nil, err
	}
	refresh, err := ar.data.db.Auth.UpdateOneID(refreshAuth.ID).
		SetExpiresAt(now.Add(tokenauth.TokenExpiresIn(entauth.TypeRefreshToken))).
		SetSessionID(sessionID).
		Save(ctx)
	if err != nil {
		return nil, nil, err
	}
	ar.cacheToken.Set(access.DeviceID, access)

	return access, refresh, nil
}

func (ar *authRepo) GetAuth(ctx context.Context, sessionID string, tokenType entauth.Type) (*gen.Auth, error) {
	return ar.data.db.Auth.Query().
		Where(
			entauth.SessionIDEQ(sessionID),
			entauth.TypeEQ(tokenType),
		).First(ctx)
}

func (ar *authRepo) ListAuth(ctx context.Context, tokenType *entauth.Type, userId string) ([]*gen.Auth, error) {
	query := ar.data.db.Auth.Query().
		Where(entauth.UserIDEQ(userId))

	if tokenType != nil {
		query.Where(entauth.TypeEQ(*tokenType))
	}

	return query.All(ctx)
}

func (ar *authRepo) GetAuthByDeviceID(ctx context.Context, deviceID string, tokenType entauth.Type) (*gen.Auth, error) {
	add := func() (*gen.Auth, error) {
		return ar.data.db.Auth.Query().
			Where(
				entauth.DeviceIDEQ(deviceID),
				entauth.TypeEQ(tokenType),
			).First(ctx)
	}
	if tokenType == entauth.TypeToken {
		authData, ok := ar.cacheToken.GetByAdd(deviceID, add)
		if !ok {
			return nil, biz.ErrTokenInvalid
		}
		return authData, nil
	}
	return add()
}

func (ar *authRepo) DeleteAuth(ctx context.Context, userID string, deviceID *string) error {
	del := ar.data.db.Auth.Delete().
		Where(entauth.UserIDEQ(userID))

	if deviceID != nil {
		del = del.Where(entauth.DeviceIDEQ(*deviceID))
	}

	_, err := del.Exec(ctx)
	return err
}
