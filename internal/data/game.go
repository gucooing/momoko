package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	entlink "momoko/internal/data/ent/gen/gameaccountlink"
	entunlink "momoko/internal/data/ent/gen/gameaccountunlink"
	entrevision "momoko/internal/data/ent/gen/gameconfigrevision"
	entmail "momoko/internal/data/ent/gen/gamemail"
	entrecent "momoko/internal/data/ent/gen/gamerecentserver"
	entgameserver "momoko/internal/data/ent/gen/gameserver"
	entbackup "momoko/internal/data/ent/gen/gameserverbackup"
	enthistory "momoko/internal/data/ent/gen/gameserverhistory"
	entcode "momoko/internal/data/ent/gen/giftcode"
	entbatch "momoko/internal/data/ent/gen/giftcodebatch"
	entredeem "momoko/internal/data/ent/gen/giftcoderedeem"
	entselfservice "momoko/internal/data/ent/gen/gmselfservice"
	entinstance "momoko/internal/data/ent/gen/instance"
)

type gameServerRepo struct {
	data *Data
}

func NewGameServerRepo(data *Data) biz.GameServerRepo {
	return &gameServerRepo{data: data}
}

func (r *gameServerRepo) Get(ctx context.Context, id string) (*gen.GameServer, error) {
	return r.data.db.GameServer.Get(ctx, id)
}

func (r *gameServerRepo) List(ctx context.Context) ([]*gen.GameServer, error) {
	return r.data.db.GameServer.Query().Order(entgameserver.ByCreateTime()).All(ctx)
}

func (r *gameServerRepo) Create(ctx context.Context, s *gen.GameServer) (*gen.GameServer, error) {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	c := r.data.db.GameServer.Create().
		SetID(s.ID).
		SetGame(s.Game).
		SetName(s.Name).
		SetTags(s.Tags).
		SetMode(s.Mode).
		SetNodeID(s.NodeID).
		SetInstallDir(s.InstallDir).
		SetProcesses(s.Processes).
		SetOptions(s.Options).
		SetCover(s.Cover).
		SetMgpEndpoint(s.MgpEndpoint).
		SetKeyID(s.KeyID).
		SetSecret(s.Secret).
		SetTimezone(s.Timezone).
		SetRemark(s.Remark)
	if s.PassportVisibility != "" {
		c.SetPassportVisibility(s.PassportVisibility)
	}
	return c.Save(ctx)
}

func (r *gameServerRepo) Update(ctx context.Context, s *gen.GameServer) (*gen.GameServer, error) {
	u := r.data.db.GameServer.UpdateOneID(s.ID).
		SetName(s.Name).
		SetTags(s.Tags).
		SetMode(s.Mode).
		SetNodeID(s.NodeID).
		SetInstallDir(s.InstallDir).
		SetProcesses(s.Processes).
		SetOptions(s.Options).
		SetCover(s.Cover).
		SetMgpEndpoint(s.MgpEndpoint).
		SetKeyID(s.KeyID).
		SetSecret(s.Secret).
		SetPassportVisibility(s.PassportVisibility).
		SetPassportInviteCode(s.PassportInviteCode).
		SetTimezone(s.Timezone).
		SetRemark(s.Remark)
	if s.KeyRotatedAt != nil {
		u.SetKeyRotatedAt(*s.KeyRotatedAt)
	}
	if s.MgpEndpoint == "" {
		u.SetManifest("").ClearManifestAt().SetStatus("").ClearStatusAt().SetProbeOk(false)
	}
	return u.Save(ctx)
}

func (r *gameServerRepo) Delete(ctx context.Context, id string) error {
	return WithTx(ctx, r.data.db, func(tx *gen.Tx) error {
		if _, err := tx.GameAccountLink.Delete().Where(entlink.ServerID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.GameAccountUnlink.Delete().Where(entunlink.ServerID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.GmSelfService.Delete().Where(entselfservice.ServerID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.GameConfigRevision.Delete().Where(entrevision.ServerID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.GameMail.Delete().Where(entmail.ServerID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.GiftCodeRedeem.Delete().Where(entredeem.ServerID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.GiftCode.Delete().Where(entcode.ServerID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.GiftCodeBatch.Delete().Where(entbatch.ServerID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.GameRecentServer.Delete().Where(entrecent.ServerID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.GameServerHistory.Delete().Where(enthistory.ServerID(id)).Exec(ctx); err != nil {
			return err
		}
		if _, err := tx.GameServerBackup.Delete().Where(entbackup.ServerID(id)).Exec(ctx); err != nil {
			return err
		}
		return tx.GameServer.DeleteOneID(id).Exec(ctx)
	})
}

func (r *gameServerRepo) SaveManifest(ctx context.Context, id, manifest string, at time.Time) error {
	return r.data.db.GameServer.UpdateOneID(id).SetManifest(manifest).SetManifestAt(at).Exec(ctx)
}

func (r *gameServerRepo) SaveProbe(ctx context.Context, id string, p biz.GameProbe) error {
	u := r.data.db.GameServer.UpdateOneID(id).
		SetProbeOk(p.OK).
		SetProbeLatencyMs(p.LatencyMs).
		SetProbeError(p.Error).
		SetProbeAt(p.At)
	if p.OK {
		u.SetStatus(p.Status).SetStatusAt(p.At).SetLastOnlineAt(p.At)
	}
	return u.Exec(ctx)
}

func (r *gameServerRepo) TouchRecent(ctx context.Context, userID, serverID, scope string) error {
	return r.data.db.GameRecentServer.Create().
		SetUserID(userID).
		SetServerID(serverID).
		SetScope(entrecent.Scope(scope)).
		SetViewedAt(time.Now()).
		OnConflictColumns(entrecent.FieldUserID, entrecent.FieldScope, entrecent.FieldServerID).
		UpdateViewedAt().
		Exec(ctx)
}

func (r *gameServerRepo) ListRecent(ctx context.Context, userID, scope string, limit int) ([]*gen.GameRecentServer, error) {
	return r.data.db.GameRecentServer.Query().
		Where(entrecent.UserID(userID), entrecent.ScopeEQ(entrecent.Scope(scope))).
		Order(entrecent.ByViewedAt(sql.OrderDesc())).
		Limit(limit).
		All(ctx)
}

func (r *gameServerRepo) AddHistory(ctx context.Context, h *gen.GameServerHistory) error {
	if h.ID == "" {
		h.ID = uuid.NewString()
	}
	c := r.data.db.GameServerHistory.Create().
		SetID(h.ID).
		SetServerID(h.ServerID).
		SetKind(h.Kind).
		SetAction(h.Action).
		SetParams(h.Params).
		SetMessage(h.Message).
		SetOperatorID(h.OperatorID).
		SetOperatorName(h.OperatorName)
	if h.Result != "" {
		c.SetResult(h.Result)
	}
	return c.Exec(ctx)
}

func (r *gameServerRepo) FinishHistory(ctx context.Context, id, result, message string, params map[string]string) error {
	u := r.data.db.GameServerHistory.UpdateOneID(id).
		SetResult(enthistory.Result(result)).
		SetMessage(message)
	if params != nil {
		u.SetParams(params)
	}
	return u.Exec(ctx)
}

func (r *gameServerRepo) ListHistory(ctx context.Context, serverID, kind string, offset, limit int) ([]*gen.GameServerHistory, int, error) {
	q := r.data.db.GameServerHistory.Query().Where(enthistory.ServerID(serverID))
	if kind != "" {
		q.Where(enthistory.Kind(kind))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	items, err := q.Order(enthistory.ByCreateTime(sql.OrderDesc())).Offset(offset).Limit(limit).All(ctx)
	return items, total, err
}

func (r *gameServerRepo) LastHistory(ctx context.Context, serverID, kind, action, result string) (*gen.GameServerHistory, error) {
	q := r.data.db.GameServerHistory.Query().Where(enthistory.ServerID(serverID), enthistory.Kind(kind), enthistory.Action(action))
	if result != "" {
		q.Where(enthistory.ResultEQ(enthistory.Result(result)))
	}
	return q.Order(enthistory.ByCreateTime(sql.OrderDesc())).First(ctx)
}

func (r *gameServerRepo) FailInterrupted(ctx context.Context, message string) error {
	if _, err := r.data.db.GameServerHistory.Update().
		Where(enthistory.ResultEQ(enthistory.ResultRunning)).
		SetResult(enthistory.ResultFailed).SetMessage(message).Save(ctx); err != nil {
		return err
	}
	_, err := r.data.db.GameServerBackup.Update().
		Where(entbackup.StateEQ(entbackup.StateRunning)).
		SetState(entbackup.StateFailed).SetError(message).Save(ctx)
	return err
}

func (r *gameServerRepo) GetInstance(ctx context.Context, id string) (*gen.Instance, error) {
	return r.data.db.Instance.Query().Where(entinstance.IDEQ(id)).WithUser().WithType().Only(ctx)
}

func (r *gameServerRepo) ListHostedInstances(ctx context.Context, dir string) ([]*gen.Instance, error) {
	return r.data.db.Instance.Query().
		Where(entinstance.PathEQ(dir), entinstance.TagsEQ("game")).
		WithUser().
		All(ctx)
}

func (r *gameServerRepo) CreateBackup(ctx context.Context, b *gen.GameServerBackup) (*gen.GameServerBackup, error) {
	if b.ID == "" {
		b.ID = uuid.NewString()
	}
	return r.data.db.GameServerBackup.Create().
		SetID(b.ID).
		SetServerID(b.ServerID).
		SetTrigger(b.Trigger).
		SetScope(b.Scope).
		SetPath(b.Path).
		SetSize(b.Size).
		SetState(b.State).
		SetNote(b.Note).
		SetError(b.Error).
		SetOperatorID(b.OperatorID).
		SetOperatorName(b.OperatorName).
		Save(ctx)
}

func (r *gameServerRepo) UpdateBackup(ctx context.Context, b *gen.GameServerBackup) error {
	return r.data.db.GameServerBackup.UpdateOneID(b.ID).
		SetScope(b.Scope).
		SetPath(b.Path).
		SetSize(b.Size).
		SetState(b.State).
		SetError(b.Error).
		Exec(ctx)
}

func (r *gameServerRepo) GetBackup(ctx context.Context, id string) (*gen.GameServerBackup, error) {
	return r.data.db.GameServerBackup.Get(ctx, id)
}

func (r *gameServerRepo) ListBackups(ctx context.Context, serverID string, offset, limit int) ([]*gen.GameServerBackup, int, error) {
	q := r.data.db.GameServerBackup.Query().Where(entbackup.ServerID(serverID))
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	q = q.Order(entbackup.ByCreateTime(sql.OrderDesc()))
	if limit > 0 {
		q = q.Offset(offset).Limit(limit)
	}
	items, err := q.All(ctx)
	return items, total, err
}

func (r *gameServerRepo) DeleteBackup(ctx context.Context, id string) error {
	return r.data.db.GameServerBackup.DeleteOneID(id).Exec(ctx)
}

// WithTx 在事务中执行 fn，出错回滚。
func WithTx(ctx context.Context, client *gen.Client, fn func(tx *gen.Tx) error) error {
	tx, err := client.Tx(ctx)
	if err != nil {
		return err
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
