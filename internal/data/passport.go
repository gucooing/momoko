package data

import (
	"context"
	"sync"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	entlink "momoko/internal/data/ent/gen/gameaccountlink"
	entunlink "momoko/internal/data/ent/gen/gameaccountunlink"
	entrecent "momoko/internal/data/ent/gen/gamerecentserver"
	entredeem "momoko/internal/data/ent/gen/giftcoderedeem"
	entexec "momoko/internal/data/ent/gen/gmexecution"
	entselfservice "momoko/internal/data/ent/gen/gmselfservice"
	entinvite "momoko/internal/data/ent/gen/passportinvite"
	entlogin "momoko/internal/data/ent/gen/passportloginlog"
	entmsg "momoko/internal/data/ent/gen/passportmessage"
	entclient "momoko/internal/data/ent/gen/passportoidcclient"
	entgrant "momoko/internal/data/ent/gen/passportoidcgrant"
	entsession "momoko/internal/data/ent/gen/passportsession"
	entuser "momoko/internal/data/ent/gen/passportuser"
	"momoko/internal/data/ent/gen/predicate"
)

type passportRepo struct {
	data *Data
	// lastTouch 会话最近一次写入活跃时间（节流）。
	lastTouch sync.Map
}

func NewPassportRepo(data *Data) biz.PassportRepo {
	return &passportRepo{data: data}
}

// ---------------- 用户 ----------------

func (r *passportRepo) CreateUser(ctx context.Context, u *gen.PassportUser) (*gen.PassportUser, error) {
	c := r.data.db.PassportUser.Create().
		SetID(u.ID).
		SetUsername(u.Username).
		SetEmail(u.Email).
		SetEmailVerified(u.EmailVerified).
		SetPassword(u.Password).
		SetName(u.Name).
		SetLanguage(u.Language).
		SetEmailCodeLogin(u.EmailCodeLogin).
		SetStatus(u.Status).
		SetInviteID(u.InviteID).
		SetRegisterIP(u.RegisterIP)
	if u.PasswordUpdatedAt != nil {
		c.SetPasswordUpdatedAt(*u.PasswordUpdatedAt)
	}
	return c.Save(ctx)
}

func (r *passportRepo) GetUser(ctx context.Context, id string) (*gen.PassportUser, error) {
	return r.data.db.PassportUser.Get(ctx, id)
}

func (r *passportRepo) FindUserByUsername(ctx context.Context, username string) (*gen.PassportUser, error) {
	return r.data.db.PassportUser.Query().Where(entuser.UsernameEqualFold(username)).First(ctx)
}

func (r *passportRepo) FindUserByEmail(ctx context.Context, email string) (*gen.PassportUser, error) {
	if email == "" {
		return nil, &gen.NotFoundError{}
	}
	return r.data.db.PassportUser.Query().Where(entuser.EmailEqualFold(email)).Order(entuser.ByCreateTime()).First(ctx)
}

func (r *passportRepo) SaveUser(ctx context.Context, u *gen.PassportUser) (*gen.PassportUser, error) {
	up := r.data.db.PassportUser.UpdateOneID(u.ID).
		SetEmail(u.Email).
		SetEmailVerified(u.EmailVerified).
		SetPassword(u.Password).
		SetName(u.Name).
		SetBio(u.Bio).
		SetAvatar(u.Avatar).
		SetAvatarKey(u.AvatarKey).
		SetAvatarAsset(u.AvatarAsset).
		SetLanguage(u.Language).
		SetTimezone(u.Timezone).
		SetEmailCodeLogin(u.EmailCodeLogin).
		SetTotpSecret(u.TotpSecret).
		SetStatus(u.Status).
		SetLastLoginIP(u.LastLoginIP).
		SetLastLoginDevice(u.LastLoginDevice).
		SetDisabledByName(u.DisabledByName)
	if len(u.RecoveryCodes) > 0 {
		up.SetRecoveryCodes(u.RecoveryCodes)
	} else {
		up.ClearRecoveryCodes()
	}
	setTime := func(t *time.Time, set func(time.Time), clear func()) {
		if t != nil {
			set(*t)
		} else {
			clear()
		}
	}
	setTime(u.PasswordUpdatedAt, func(t time.Time) { up.SetPasswordUpdatedAt(t) }, func() { up.ClearPasswordUpdatedAt() })
	setTime(u.TotpEnabledAt, func(t time.Time) { up.SetTotpEnabledAt(t) }, func() { up.ClearTotpEnabledAt() })
	setTime(u.LastLoginAt, func(t time.Time) { up.SetLastLoginAt(t) }, func() { up.ClearLastLoginAt() })
	setTime(u.DeletionScheduledAt, func(t time.Time) { up.SetDeletionScheduledAt(t) }, func() { up.ClearDeletionScheduledAt() })
	setTime(u.DisabledAt, func(t time.Time) { up.SetDisabledAt(t) }, func() { up.ClearDisabledAt() })
	setTime(u.ReviewsSyncedAt, func(t time.Time) { up.SetReviewsSyncedAt(t) }, func() { up.ClearReviewsSyncedAt() })
	return up.Save(ctx)
}

// linkedUsersWhere 用户在 game_account_links 里有符合条件的有效绑定（子查询）。
func linkedUsersWhere(in bool, conds func(t *sql.SelectTable) []*sql.Predicate) predicate.PassportUser {
	return func(s *sql.Selector) {
		t := sql.Table(entlink.Table)
		where := []*sql.Predicate{sql.EQ(t.C(entlink.FieldState), entlink.StateActive.String())}
		if conds != nil {
			where = append(where, conds(t)...)
		}
		sub := sql.Select(t.C(entlink.FieldPassportUserID)).From(t).Where(sql.And(where...))
		if in {
			s.Where(sql.In(s.C(entuser.FieldID), sub))
		} else {
			s.Where(sql.NotIn(s.C(entuser.FieldID), sub))
		}
	}
}

// baseUserPredicates 搜索与服务器筛选（不含状态）。
func baseUserPredicates(f biz.PassportUserFilter) []predicate.PassportUser {
	var ps []predicate.PassportUser
	if f.Q != "" {
		q := f.Q
		ps = append(ps, entuser.Or(
			entuser.UsernameContainsFold(q),
			entuser.NameContainsFold(q),
			entuser.EmailContainsFold(q),
			entuser.IDEQ(q),
			linkedUsersWhere(true, func(t *sql.SelectTable) []*sql.Predicate {
				return []*sql.Predicate{sql.Or(
					sql.ContainsFold(t.C(entlink.FieldNickname), q),
					sql.ContainsFold(t.C(entlink.FieldDisplayID), q),
					sql.ContainsFold(t.C(entlink.FieldAccount), q),
					sql.EQ(t.C(entlink.FieldPlayerID), q),
				)}
			}),
		))
	}
	if f.ServerID != "" {
		ps = append(ps, linkedUsersWhere(true, func(t *sql.SelectTable) []*sql.Predicate {
			return []*sql.Predicate{sql.EQ(t.C(entlink.FieldServerID), f.ServerID)}
		}))
	}
	return ps
}

func statusPredicate(status string) predicate.PassportUser {
	switch status {
	case "linked":
		return linkedUsersWhere(true, nil)
	case "unlinked":
		return linkedUsersWhere(false, nil)
	case "disabled":
		return entuser.StatusEQ(entuser.StatusDisabled)
	case "deleting":
		return entuser.DeletionScheduledAtNotNil()
	default:
		return nil
	}
}

func (r *passportRepo) ListUsers(ctx context.Context, f biz.PassportUserFilter) ([]*gen.PassportUser, int, error) {
	ps := baseUserPredicates(f)
	if p := statusPredicate(f.Status); p != nil {
		ps = append(ps, p)
	}
	q := r.data.db.PassportUser.Query().Where(ps...)
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	list, err := q.Order(entuser.ByCreateTime(sql.OrderDesc())).Offset(f.Offset).Limit(f.Limit).All(ctx)
	return list, total, err
}

func (r *passportRepo) UserCounts(ctx context.Context, f biz.PassportUserFilter) (map[string]int32, error) {
	base := baseUserPredicates(f)
	out := map[string]int32{}
	for _, status := range []string{"all", "linked", "unlinked", "disabled", "deleting"} {
		ps := append([]predicate.PassportUser{}, base...)
		if p := statusPredicate(status); p != nil {
			ps = append(ps, p)
		}
		n, err := r.data.db.PassportUser.Query().Where(ps...).Count(ctx)
		if err != nil {
			return nil, err
		}
		out[status] = int32(n)
	}
	return out, nil
}

func (r *passportRepo) Stats(ctx context.Context, since time.Time) (*biz.PassportStats, error) {
	db := r.data.db
	st := &biz.PassportStats{}
	var err error
	if st.Users, err = db.PassportUser.Query().Count(ctx); err != nil {
		return nil, err
	}
	active := entlink.StateEQ(entlink.StateActive)
	if st.LinkedAccounts, err = db.GameAccountLink.Query().Where(active).Count(ctx); err != nil {
		return nil, err
	}
	servers, err := db.GameAccountLink.Query().Where(active).GroupBy(entlink.FieldServerID).Strings(ctx)
	if err != nil {
		return nil, err
	}
	st.Servers = len(servers)
	if st.Games, err = db.GameAccountLink.Query().Where(active).GroupBy(entlink.FieldGame).Strings(ctx); err != nil {
		return nil, err
	}
	if st.NewThisWeek, err = db.PassportUser.Query().Where(entuser.CreateTimeGTE(since)).Count(ctx); err != nil {
		return nil, err
	}
	if st.NewThisWeekLinked, err = db.PassportUser.Query().Where(entuser.CreateTimeGTE(since), linkedUsersWhere(true, nil)).Count(ctx); err != nil {
		return nil, err
	}
	if st.ActiveThisWeek, err = db.PassportUser.Query().Where(entuser.LastLoginAtGTE(since)).Count(ctx); err != nil {
		return nil, err
	}
	if st.Disabled, err = db.PassportUser.Query().Where(entuser.StatusEQ(entuser.StatusDisabled)).Count(ctx); err != nil {
		return nil, err
	}
	if st.Deleting, err = db.PassportUser.Query().Where(entuser.DeletionScheduledAtNotNil()).Count(ctx); err != nil {
		return nil, err
	}
	return st, nil
}

func (r *passportRepo) DeleteUserCascade(ctx context.Context, id string) error {
	return WithTx(ctx, r.data.db, func(tx *gen.Tx) error {
		links, err := tx.GameAccountLink.Query().Where(entlink.PassportUserID(id)).All(ctx)
		if err != nil {
			return err
		}
		now := time.Now()
		for _, l := range links {
			if l.State != entlink.StateActive {
				continue
			}
			if err := tx.GameAccountUnlink.Create().
				SetID(uuid.NewString()).
				SetPassportUserID(id).
				SetServerID(l.ServerID).
				SetPlayerID(l.PlayerID).
				SetNickname(l.Nickname).
				SetBy(entunlink.BySelf).
				SetUnlinkedAt(now).
				Exec(ctx); err != nil {
				return err
			}
		}
		if _, err := tx.GameAccountLink.Delete().Where(entlink.PassportUserID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.PassportSession.Delete().Where(entsession.UserID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.PassportOidcGrant.Delete().Where(entgrant.UserID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.PassportMessage.Delete().Where(entmsg.UserID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.PassportLoginLog.Delete().Where(entlogin.UserID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.GameRecentServer.Delete().Where(entrecent.UserID(id), entrecent.ScopeEQ(entrecent.ScopePassport)).Exec(ctx); err != nil {
			return err
		}
		return tx.PassportUser.DeleteOneID(id).Exec(ctx)
	})
}

func (r *passportRepo) DueDeletions(ctx context.Context, now time.Time) ([]*gen.PassportUser, error) {
	return r.data.db.PassportUser.Query().Where(entuser.DeletionScheduledAtLTE(now)).Limit(200).All(ctx)
}

// ---------------- 会话 ----------------

func (r *passportRepo) CreateSession(ctx context.Context, s *gen.PassportSession) (*gen.PassportSession, error) {
	var out *gen.PassportSession
	err := WithTx(ctx, r.data.db, func(tx *gen.Tx) error {
		if _, err := tx.PassportSession.Delete().Where(entsession.UserID(s.UserID), entsession.DeviceID(s.DeviceID)).Exec(ctx); err != nil {
			return err
		}
		c := tx.PassportSession.Create().
			SetID(s.ID).
			SetUserID(s.UserID).
			SetDeviceID(s.DeviceID).
			SetDevice(s.Device).
			SetUserAgent(s.UserAgent).
			SetClientID(s.ClientID).
			SetLoginMethod(s.LoginMethod).
			SetIP(s.IP).
			SetLastIP(s.LastIP).
			SetAccessNoise(s.AccessNoise).
			SetRefreshNoise(s.RefreshNoise).
			SetPersistent(s.Persistent).
			SetExpiresAt(s.ExpiresAt)
		if s.LastActiveAt != nil {
			c.SetLastActiveAt(*s.LastActiveAt)
		}
		var err error
		out, err = c.Save(ctx)
		return err
	})
	return out, err
}

func (r *passportRepo) GetSession(ctx context.Context, id string) (*gen.PassportSession, error) {
	return r.data.db.PassportSession.Get(ctx, id)
}

func (r *passportRepo) RotateSession(ctx context.Context, id, accessNoise, refreshNoise string, expiresAt time.Time) (*gen.PassportSession, error) {
	return r.data.db.PassportSession.UpdateOneID(id).
		SetAccessNoise(accessNoise).
		SetRefreshNoise(refreshNoise).
		SetExpiresAt(expiresAt).
		Save(ctx)
}

func (r *passportRepo) ListSessions(ctx context.Context, userID string) ([]*gen.PassportSession, error) {
	return r.data.db.PassportSession.Query().Where(entsession.UserID(userID)).Order(entsession.ByCreateTime(sql.OrderDesc())).All(ctx)
}

func (r *passportRepo) CountSessions(ctx context.Context, userID string) (int, error) {
	return r.data.db.PassportSession.Query().Where(entsession.UserID(userID), entsession.ExpiresAtGT(time.Now())).Count(ctx)
}

func (r *passportRepo) DeleteSession(ctx context.Context, userID, sessionID string) error {
	n, err := r.data.db.PassportSession.Delete().Where(entsession.ID(sessionID), entsession.UserID(userID)).Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return &gen.NotFoundError{}
	}
	return nil
}

func (r *passportRepo) DeleteSessions(ctx context.Context, userID, except string) error {
	q := r.data.db.PassportSession.Delete().Where(entsession.UserID(userID))
	if except != "" {
		q.Where(entsession.IDNEQ(except))
	}
	_, err := q.Exec(ctx)
	return err
}

const passportTouchInterval = time.Minute

func (r *passportRepo) TouchSession(ctx context.Context, sessionID, ip string) {
	type touch struct {
		at time.Time
		ip string
	}
	now := time.Now()
	if v, ok := r.lastTouch.Load(sessionID); ok {
		t := v.(touch)
		if t.ip == ip && now.Sub(t.at) < passportTouchInterval {
			return
		}
	}
	r.lastTouch.Store(sessionID, touch{at: now, ip: ip})
	go func() {
		writeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
		defer cancel()
		_ = r.data.db.PassportSession.Update().
			Where(entsession.ID(sessionID)).
			SetLastActiveAt(now).
			SetLastIP(ip).
			Exec(writeCtx)
	}()
}

// ---------------- 登录记录 ----------------

func (r *passportRepo) AddLoginLog(ctx context.Context, l *gen.PassportLoginLog) error {
	return r.data.db.PassportLoginLog.Create().
		SetID(l.ID).
		SetUserID(l.UserID).
		SetMethod(l.Method).
		SetSuccess(l.Success).
		SetErrorCode(l.ErrorCode).
		SetIP(l.IP).
		SetDeviceID(l.DeviceID).
		SetDevice(l.Device).
		SetUserAgent(l.UserAgent).
		Exec(ctx)
}

func (r *passportRepo) ListLoginLogs(ctx context.Context, userID string, limit int) ([]*gen.PassportLoginLog, error) {
	return r.data.db.PassportLoginLog.Query().
		Where(entlogin.UserID(userID), entlogin.Success(true)).
		Order(entlogin.ByCreateTime(sql.OrderDesc())).
		Limit(limit).
		All(ctx)
}

func (r *passportRepo) LoginHistory(ctx context.Context, userID, deviceID string) (bool, bool, error) {
	anyLogin, err := r.data.db.PassportLoginLog.Query().Where(entlogin.UserID(userID), entlogin.Success(true)).Exist(ctx)
	if err != nil || !anyLogin {
		return anyLogin, false, err
	}
	same, err := r.data.db.PassportLoginLog.Query().Where(entlogin.UserID(userID), entlogin.Success(true), entlogin.DeviceID(deviceID)).Exist(ctx)
	return true, same, err
}

// ---------------- 消息 ----------------

func (r *passportRepo) AddMessages(ctx context.Context, msgs []*gen.PassportMessage) error {
	if len(msgs) == 0 {
		return nil
	}
	builders := make([]*gen.PassportMessageCreate, 0, len(msgs))
	for _, m := range msgs {
		builders = append(builders, r.data.db.PassportMessage.Create().
			SetID(m.ID).
			SetUserID(m.UserID).
			SetKind(m.Kind).
			SetServerID(m.ServerID).
			SetParams(m.Params))
	}
	return r.data.db.PassportMessage.CreateBulk(builders...).Exec(ctx)
}

func (r *passportRepo) ListMessages(ctx context.Context, userID string, unreadOnly bool, offset, limit int) ([]*gen.PassportMessage, int, error) {
	q := r.data.db.PassportMessage.Query().Where(entmsg.UserID(userID))
	if unreadOnly {
		q.Where(entmsg.ReadAtIsNil())
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	list, err := q.Order(entmsg.ByCreateTime(sql.OrderDesc())).Offset(offset).Limit(limit).All(ctx)
	return list, total, err
}

func (r *passportRepo) CountUnread(ctx context.Context, userID string) (int, error) {
	return r.data.db.PassportMessage.Query().Where(entmsg.UserID(userID), entmsg.ReadAtIsNil()).Count(ctx)
}

func (r *passportRepo) ReadMessages(ctx context.Context, userID string, ids []string, all bool) error {
	q := r.data.db.PassportMessage.Update().Where(entmsg.UserID(userID), entmsg.ReadAtIsNil())
	if !all {
		q.Where(entmsg.IDIn(ids...))
	}
	return q.SetReadAt(time.Now()).Exec(ctx)
}

// ---------------- 邀请码 ----------------

func (r *passportRepo) CreateInvite(ctx context.Context, i *gen.PassportInvite) (*gen.PassportInvite, error) {
	c := r.data.db.PassportInvite.Create().
		SetID(i.ID).
		SetCode(i.Code).
		SetEmail(i.Email).
		SetServerID(i.ServerID).
		SetMaxUses(i.MaxUses).
		SetCreatedBy(i.CreatedBy).
		SetCreatedByName(i.CreatedByName).
		SetNote(i.Note)
	if i.ExpiresAt != nil {
		c.SetExpiresAt(*i.ExpiresAt)
	}
	return c.Save(ctx)
}

func (r *passportRepo) GetInvite(ctx context.Context, id string) (*gen.PassportInvite, error) {
	return r.data.db.PassportInvite.Get(ctx, id)
}

func (r *passportRepo) GetInviteByCode(ctx context.Context, code string) (*gen.PassportInvite, error) {
	if code == "" {
		return nil, &gen.NotFoundError{}
	}
	return r.data.db.PassportInvite.Query().Where(entinvite.CodeEqualFold(code)).First(ctx)
}

func (r *passportRepo) ListInvites(ctx context.Context, offset, limit int) ([]*gen.PassportInvite, int, error) {
	q := r.data.db.PassportInvite.Query()
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	list, err := q.Order(entinvite.ByCreateTime(sql.OrderDesc())).Offset(offset).Limit(limit).All(ctx)
	return list, total, err
}

func (r *passportRepo) UseInvite(ctx context.Context, id string, now time.Time) (bool, error) {
	n, err := r.data.db.PassportInvite.Update().
		Where(
			entinvite.ID(id),
			entinvite.Revoked(false),
			entinvite.Or(entinvite.ExpiresAtIsNil(), entinvite.ExpiresAtGT(now)),
			func(s *sql.Selector) { s.Where(sql.ColumnsLT(s.C(entinvite.FieldUsed), s.C(entinvite.FieldMaxUses))) },
		).
		AddUsed(1).
		Save(ctx)
	return n > 0, err
}

func (r *passportRepo) RevokeInvite(ctx context.Context, id string) (*gen.PassportInvite, error) {
	return r.data.db.PassportInvite.UpdateOneID(id).SetRevoked(true).Save(ctx)
}

// ---------------- 游戏账号绑定 ----------------

func (r *passportRepo) ListLinks(ctx context.Context, userID string) ([]*gen.GameAccountLink, error) {
	return r.data.db.GameAccountLink.Query().
		Where(entlink.PassportUserID(userID), entlink.StateEQ(entlink.StateActive)).
		Order(entlink.ByLinkedAt()).
		All(ctx)
}

func (r *passportRepo) LinksOfUsers(ctx context.Context, userIDs []string) (map[string][]*gen.GameAccountLink, error) {
	out := map[string][]*gen.GameAccountLink{}
	if len(userIDs) == 0 {
		return out, nil
	}
	list, err := r.data.db.GameAccountLink.Query().
		Where(entlink.PassportUserIDIn(userIDs...), entlink.StateEQ(entlink.StateActive)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, l := range list {
		out[l.PassportUserID] = append(out[l.PassportUserID], l)
	}
	return out, nil
}

func (r *passportRepo) GetLink(ctx context.Context, id string) (*gen.GameAccountLink, error) {
	return r.data.db.GameAccountLink.Get(ctx, id)
}

func (r *passportRepo) LinkOf(ctx context.Context, userID, serverID string) (*gen.GameAccountLink, error) {
	return r.data.db.GameAccountLink.Query().
		Where(entlink.PassportUserID(userID), entlink.ServerID(serverID), entlink.StateEQ(entlink.StateActive)).
		First(ctx)
}

func (r *passportRepo) LinkByPlayer(ctx context.Context, serverID, pid string) (*gen.GameAccountLink, error) {
	return r.data.db.GameAccountLink.Query().
		Where(entlink.ServerID(serverID), entlink.PlayerID(pid), entlink.StateEQ(entlink.StateActive)).
		First(ctx)
}

func (r *passportRepo) LinksByPlayers(ctx context.Context, serverID string, pids []string) ([]*gen.GameAccountLink, error) {
	if len(pids) == 0 {
		return nil, nil
	}
	return r.data.db.GameAccountLink.Query().
		Where(entlink.ServerID(serverID), entlink.PlayerIDIn(pids...), entlink.StateEQ(entlink.StateActive)).
		All(ctx)
}

func (r *passportRepo) CreateLink(ctx context.Context, l *gen.GameAccountLink) (*gen.GameAccountLink, error) {
	return r.data.db.GameAccountLink.Create().
		SetID(l.ID).
		SetPassportUserID(l.PassportUserID).
		SetServerID(l.ServerID).
		SetGame(l.Game).
		SetPlayerID(l.PlayerID).
		SetDisplayID(l.DisplayID).
		SetAccount(l.Account).
		SetNickname(l.Nickname).
		SetAvatarAsset(l.AvatarAsset).
		SetLevel(l.Level).
		SetMethod(l.Method).
		SetState(l.State).
		SetIsPrimary(l.IsPrimary).
		SetLinkedAt(l.LinkedAt).
		Save(ctx)
}

func (r *passportRepo) UpdateLinkSnapshot(ctx context.Context, l *gen.GameAccountLink) error {
	return r.data.db.GameAccountLink.UpdateOneID(l.ID).
		SetNickname(l.Nickname).
		SetLevel(l.Level).
		SetDisplayID(l.DisplayID).
		SetAvatarAsset(l.AvatarAsset).
		SetAccount(l.Account).
		Exec(ctx)
}

func (r *passportRepo) TouchLink(ctx context.Context, id string, at time.Time) error {
	return r.data.db.GameAccountLink.UpdateOneID(id).SetLastSeenAt(at).Exec(ctx)
}

func (r *passportRepo) SetPrimary(ctx context.Context, userID, game, linkID string) error {
	return WithTx(ctx, r.data.db, func(tx *gen.Tx) error {
		if err := tx.GameAccountLink.Update().
			Where(entlink.PassportUserID(userID), entlink.Game(game), entlink.IDNEQ(linkID)).
			SetIsPrimary(false).
			Exec(ctx); err != nil {
			return err
		}
		return tx.GameAccountLink.UpdateOneID(linkID).SetIsPrimary(true).Exec(ctx)
	})
}

func (r *passportRepo) Unlink(ctx context.Context, l *gen.GameAccountLink, by string) error {
	return WithTx(ctx, r.data.db, func(tx *gen.Tx) error {
		if err := tx.GameAccountUnlink.Create().
			SetID(uuid.NewString()).
			SetPassportUserID(l.PassportUserID).
			SetServerID(l.ServerID).
			SetPlayerID(l.PlayerID).
			SetNickname(l.Nickname).
			SetBy(entunlink.By(by)).
			SetUnlinkedAt(time.Now()).
			Exec(ctx); err != nil {
			return err
		}
		return tx.GameAccountLink.DeleteOneID(l.ID).Exec(ctx)
	})
}

func (r *passportRepo) LastUnlink(ctx context.Context, userID, serverID string) (*gen.GameAccountUnlink, error) {
	u, err := r.data.db.GameAccountUnlink.Query().
		Where(entunlink.PassportUserID(userID), entunlink.ServerID(serverID)).
		Order(entunlink.ByUnlinkedAt(sql.OrderDesc())).
		First(ctx)
	if gen.IsNotFound(err) {
		return nil, nil
	}
	return u, err
}

func (r *passportRepo) ListUnlinks(ctx context.Context, userID string, limit int) ([]*gen.GameAccountUnlink, error) {
	return r.data.db.GameAccountUnlink.Query().
		Where(entunlink.PassportUserID(userID)).
		Order(entunlink.ByUnlinkedAt(sql.OrderDesc())).
		Limit(limit).
		All(ctx)
}

func (r *passportRepo) LinkCounts(ctx context.Context) (map[string]int, error) {
	var rows []struct {
		ServerID string `json:"server_id"`
		Count    int    `json:"count"`
	}
	if err := r.data.db.GameAccountLink.Query().
		Where(entlink.StateEQ(entlink.StateActive)).
		GroupBy(entlink.FieldServerID).
		Aggregate(gen.Count()).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.ServerID] = row.Count
	}
	return out, nil
}

func (r *passportRepo) SelfServiceCounts(ctx context.Context) (map[string]int, error) {
	var rows []struct {
		ServerID string `json:"server_id"`
		Count    int    `json:"count"`
	}
	if err := r.data.db.GmSelfService.Query().
		Where(entselfservice.Enabled(true)).
		GroupBy(entselfservice.FieldServerID).
		Aggregate(gen.Count()).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.ServerID] = row.Count
	}
	return out, nil
}

// ---------------- 玩家自己的 GM 记录与兑换 ----------------

func (r *passportRepo) ListPlayerExecutions(ctx context.Context, f biz.PassportExecFilter) ([]*gen.GmExecution, int, error) {
	q := r.data.db.GmExecution.Query().Where(
		entexec.ActorKindEQ(entexec.ActorKindPlayer),
		entexec.ActorID(f.UserID),
		entexec.DryRun(false),
	)
	if f.ServerID != "" {
		q.Where(entexec.ServerID(f.ServerID))
	}
	if f.Reviewable {
		q.Where(entexec.Or(entexec.StatusEQ(entexec.StatusPending), entexec.ReviewedAtNotNil()))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	limit := f.Limit
	if limit <= 0 {
		limit = 20
	}
	list, err := q.Order(entexec.ByCreateTime(sql.OrderDesc())).Offset(f.Offset).Limit(limit).All(ctx)
	return list, total, err
}

func (r *passportRepo) ReviewedSince(ctx context.Context, userID string, since time.Time) ([]*gen.GmExecution, error) {
	return r.data.db.GmExecution.Query().
		Where(
			entexec.ActorKindEQ(entexec.ActorKindPlayer),
			entexec.ActorID(userID),
			entexec.ReviewedAtGT(since),
		).
		Order(entexec.ByReviewedAt()).
		Limit(100).
		All(ctx)
}

func (r *passportRepo) ListRedeems(ctx context.Context, userID, serverID string, offset, limit int) ([]*gen.GiftCodeRedeem, int, error) {
	q := r.data.db.GiftCodeRedeem.Query().Where(entredeem.PassportUserID(userID))
	if serverID != "" {
		q.Where(entredeem.ServerID(serverID))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	list, err := q.Order(entredeem.ByCreateTime(sql.OrderDesc())).Offset(offset).Limit(limit).All(ctx)
	return list, total, err
}

// ---------------- OIDC ----------------

func (r *passportRepo) ListOidcClients(ctx context.Context) ([]*gen.PassportOidcClient, error) {
	return r.data.db.PassportOidcClient.Query().Order(entclient.ByCreateTime()).All(ctx)
}

func (r *passportRepo) GetOidcClient(ctx context.Context, id string) (*gen.PassportOidcClient, error) {
	return r.data.db.PassportOidcClient.Get(ctx, id)
}

func (r *passportRepo) GetOidcClientByClientID(ctx context.Context, clientID string) (*gen.PassportOidcClient, error) {
	if clientID == "" {
		return nil, &gen.NotFoundError{}
	}
	return r.data.db.PassportOidcClient.Query().Where(entclient.ClientID(clientID)).Only(ctx)
}

func (r *passportRepo) CreateOidcClient(ctx context.Context, c *gen.PassportOidcClient) (*gen.PassportOidcClient, error) {
	return r.data.db.PassportOidcClient.Create().
		SetID(c.ID).
		SetName(c.Name).
		SetClientID(c.ClientID).
		SetClientSecret(c.ClientSecret).
		SetRedirectUris(c.RedirectUris).
		SetScopes(c.Scopes).
		SetActive(c.Active).
		SetDescription(c.Description).
		SetServerID(c.ServerID).
		SetCreatedByName(c.CreatedByName).
		Save(ctx)
}

func (r *passportRepo) UpdateOidcClient(ctx context.Context, c *gen.PassportOidcClient) (*gen.PassportOidcClient, error) {
	return r.data.db.PassportOidcClient.UpdateOneID(c.ID).
		SetName(c.Name).
		SetRedirectUris(c.RedirectUris).
		SetScopes(c.Scopes).
		SetActive(c.Active).
		SetDescription(c.Description).
		SetServerID(c.ServerID).
		Save(ctx)
}

func (r *passportRepo) UpdateOidcClientSecret(ctx context.Context, id, secret string) (*gen.PassportOidcClient, error) {
	return r.data.db.PassportOidcClient.UpdateOneID(id).SetClientSecret(secret).Save(ctx)
}

func (r *passportRepo) DeleteOidcClient(ctx context.Context, id string) error {
	return WithTx(ctx, r.data.db, func(tx *gen.Tx) error {
		if _, err := tx.PassportOidcGrant.Delete().Where(entgrant.ClientID(id)).Exec(ctx); err != nil {
			return err
		}
		return tx.PassportOidcClient.DeleteOneID(id).Exec(ctx)
	})
}

func (r *passportRepo) MarkOidcClientUsed(ctx context.Context, id string, at time.Time) error {
	return r.data.db.PassportOidcClient.UpdateOneID(id).SetLastUsedAt(at).Exec(ctx)
}

func (r *passportRepo) GrantCounts(ctx context.Context) (map[string]int, error) {
	var rows []struct {
		ClientID string `json:"client_id"`
		Count    int    `json:"count"`
	}
	if err := r.data.db.PassportOidcGrant.Query().
		GroupBy(entgrant.FieldClientID).
		Aggregate(gen.Count()).
		Scan(ctx, &rows); err != nil {
		return nil, err
	}
	out := make(map[string]int, len(rows))
	for _, row := range rows {
		out[row.ClientID] = row.Count
	}
	return out, nil
}

func (r *passportRepo) GetGrant(ctx context.Context, userID, clientID string) (*gen.PassportOidcGrant, error) {
	return r.data.db.PassportOidcGrant.Query().Where(entgrant.UserID(userID), entgrant.ClientID(clientID)).Only(ctx)
}

func (r *passportRepo) SaveGrant(ctx context.Context, userID, clientID string, scopes []string) (*gen.PassportOidcGrant, error) {
	existing, err := r.GetGrant(ctx, userID, clientID)
	if err == nil {
		return existing.Update().SetScopes(scopes).SetLastUsedAt(time.Now()).Save(ctx)
	}
	if !gen.IsNotFound(err) {
		return nil, err
	}
	return r.data.db.PassportOidcGrant.Create().
		SetID(uuid.NewString()).
		SetUserID(userID).
		SetClientID(clientID).
		SetScopes(scopes).
		SetLastUsedAt(time.Now()).
		Save(ctx)
}

func (r *passportRepo) TouchGrant(ctx context.Context, id string, at time.Time) error {
	return r.data.db.PassportOidcGrant.UpdateOneID(id).SetLastUsedAt(at).Exec(ctx)
}

func (r *passportRepo) ListGrants(ctx context.Context, userID string) ([]*gen.PassportOidcGrant, error) {
	return r.data.db.PassportOidcGrant.Query().Where(entgrant.UserID(userID)).Order(entgrant.ByLastUsedAt(sql.OrderDesc())).All(ctx)
}

func (r *passportRepo) CountGrants(ctx context.Context, userID string) (int, error) {
	return r.data.db.PassportOidcGrant.Query().Where(entgrant.UserID(userID)).Count(ctx)
}

func (r *passportRepo) DeleteGrant(ctx context.Context, userID, id string) error {
	n, err := r.data.db.PassportOidcGrant.Delete().Where(entgrant.ID(id), entgrant.UserID(userID)).Exec(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return &gen.NotFoundError{}
	}
	return nil
}
