package data

import (
	"context"
	"sync"
	"time"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	entauth "momoko/internal/data/ent/gen/session"
	"momoko/pkg/cache"
)

type authRepo struct {
	data *Data

	cacheToken *cache.Cache[string, *gen.Session]
	// lastTouch 会话最近一次写入活跃时间（节流用）。
	lastTouch sync.Map
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
		SetLastIP(authInfo.IP).
		SetLastActiveAt(time.Now()).
		SetUserAgent(authInfo.UserAgent).
		SetLoginMethod(authInfo.LoginMethod).
		SetAccessNoise(authInfo.AccessNoise).
		SetRefreshNoise(authInfo.RefreshNoise).
		SetPersistent(authInfo.Persistent).
		SetExpiresAt(time.Now().Add(biz.SessionTTL(authInfo.Persistent))).
		Save(ctx)
	if err != nil {
		return nil, err
	}
	ar.cacheToken.Set(authInfo.SessionID, authData)
	return authData, nil
}

// Refresh 续签同一会话行：只更新 noise 与 expires_at（now+有效期，按会话是否保持登录），不换 session_id。
func (ar *authRepo) Refresh(ctx context.Context, sessionID, accessNoise, refreshNoise string) (*gen.Session, error) {
	current, err := ar.GetAuth(ctx, sessionID)
	if err != nil {
		return nil, err
	}
	updated, err := ar.data.db.Session.UpdateOneID(sessionID).
		SetAccessNoise(accessNoise).
		SetRefreshNoise(refreshNoise).
		SetExpiresAt(time.Now().Add(biz.SessionTTL(current.Persistent))).
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

// sessionTouchInterval 同一会话两次写入活跃时间的最小间隔。
const sessionTouchInterval = time.Minute

// TouchSession 记录会话最近活跃时间与 IP；同一会话 1 分钟内（IP 不变时）只写一次。
func (ar *authRepo) TouchSession(ctx context.Context, sessionID, ip string) {
	type touch struct {
		at time.Time
		ip string
	}
	now := time.Now()
	if v, ok := ar.lastTouch.Load(sessionID); ok {
		t := v.(touch)
		if t.ip == ip && now.Sub(t.at) < sessionTouchInterval {
			return
		}
	}
	ar.lastTouch.Store(sessionID, touch{at: now, ip: ip})
	go func() {
		writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		_, _ = ar.data.db.Session.Update().
			Where(entauth.IDEQ(sessionID)).
			SetLastActiveAt(now).
			SetLastIP(ip).
			Save(writeCtx)
	}()
}
