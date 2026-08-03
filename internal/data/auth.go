package data

import (
	"context"
	"time"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	entauth "momoko/internal/data/ent/gen/session"
	tokenauth "momoko/pkg/auth"
	"momoko/pkg/cache"
)

type authRepo struct {
	data *Data

	cacheToken *cache.Cache[string, *gen.Session]
}

func NewAuthRepo(data *Data) biz.AuthRepo {
	return &authRepo{
		data:       data,
		cacheToken: cache.New[string, *gen.Session](5 * time.Minute),
	}
}

func (ar *authRepo) CreateAuth(ctx context.Context, authInfo *biz.Auth) (*gen.Session, error) {
	ar.data.db.Session.Delete().Where(
		entauth.And(
			entauth.UserIDEQ(authInfo.UserID),
			entauth.DeviceIDEQ(authInfo.DeviceID),
		)).Exec(ctx) // 将可能存在的旧会话移除掉

	authData, err := ar.data.db.Session.
		Create().
		SetID(authInfo.SessionID).
		SetUserID(authInfo.UserID).
		SetDeviceID(authInfo.DeviceID).
		SetDevice(authInfo.Device).
		SetIP(authInfo.IP).
		SetAccessNoise(authInfo.AccessNoise).
		SetRefreshNoise(authInfo.RefreshNoise).
		SetExpiresAt(time.Now().Add(tokenauth.RefreshTokenExpiresIn)).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	ar.cacheToken.Set(authInfo.SessionID, authData)
	return authData, nil
}

// Refresh 续签同一会话行：只更新 noise 与 expires_at（now+有效期），不换 session_id。
func (ar *authRepo) Refresh(ctx context.Context, sessionID, accessNoise, refreshNoise string) (*gen.Session, error) {
	updated, err := ar.data.db.Session.UpdateOneID(sessionID).
		SetAccessNoise(accessNoise).
		SetRefreshNoise(refreshNoise).
		SetExpiresAt(time.Now().Add(tokenauth.RefreshTokenExpiresIn)).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	ar.cacheToken.Set(sessionID, updated)
	return updated, nil
}

func (ar *authRepo) GetAuth(ctx context.Context, sessionID string) (*gen.Session, error) {
	if info, ok := ar.cacheToken.Get(sessionID); ok {
		return info, nil
	}
	return ar.data.db.Session.Query().
		Where(entauth.IDEQ(sessionID)).
		Only(ctx)
}

func (ar *authRepo) ListAuth(ctx context.Context, userId string) ([]*gen.Session, error) {
	return ar.data.db.Session.Query().
		Where(entauth.UserIDEQ(userId)).
		All(ctx)
}

func (ar *authRepo) DeleteAuth(ctx context.Context, userID string, sessionID *string) error {
	del := ar.data.db.Session.Delete().
		Where(entauth.UserIDEQ(userID))

	if sessionID != nil {
		del = del.Where(entauth.IDEQ(*sessionID))
		ar.cacheToken.Del(*sessionID)
	} else {
		ar.cacheToken.Clear()
	}

	_, err := del.Exec(ctx)
	return err
}
