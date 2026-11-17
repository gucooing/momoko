package data

import (
	"context"

	"entgo.io/ent/dialect/sql"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	entclientlog "momoko/internal/data/ent/gen/gameclientlog"
	"momoko/internal/data/ent/gen/predicate"
)

type gameClientLogRepo struct {
	data *Data
}

func NewGameClientLogRepo(data *Data) biz.GameClientLogRepo {
	return &gameClientLogRepo{data: data}
}

// newestFirst 从新到旧（同一时刻按 id，顺序稳定）。
func newestFirst() []entclientlog.OrderOption {
	return []entclientlog.OrderOption{
		entclientlog.ByCreatedAt(sql.OrderDesc()),
		entclientlog.ByID(sql.OrderDesc()),
	}
}

func (r *gameClientLogRepo) Create(ctx context.Context, l *gen.GameClientLog, keep int) (*gen.GameClientLog, error) {
	var saved *gen.GameClientLog
	err := WithTx(ctx, r.data.db, func(tx *gen.Tx) error {
		var err error
		saved, err = tx.GameClientLog.Create().
			SetID(l.ID).
			SetServerID(l.ServerID).
			SetPlayerID(l.PlayerID).
			SetReason(l.Reason).
			SetSummary(l.Summary).
			SetNote(l.Note).
			SetDevice(l.Device).
			SetClientVersion(l.ClientVersion).
			SetSize(l.Size).
			SetContent(l.Content).
			SetOccurredAt(l.OccurredAt).
			SetCreatedAt(l.CreatedAt).
			Save(ctx)
		if err != nil {
			return err
		}
		ids, err := tx.GameClientLog.Query().
			Where(entclientlog.ServerID(l.ServerID), entclientlog.PlayerID(l.PlayerID)).
			Order(newestFirst()...).
			IDs(ctx)
		if err != nil {
			return err
		}
		if len(ids) <= keep {
			return nil
		}
		_, err = tx.GameClientLog.Delete().Where(entclientlog.IDIn(ids[keep:]...)).Exec(ctx)
		return err
	})
	if err != nil {
		return nil, err
	}
	return saved, nil
}

func (r *gameClientLogRepo) List(ctx context.Context, f biz.ClientLogFilter) ([]*gen.GameClientLog, int, error) {
	var where []predicate.GameClientLog
	if f.ServerID != "" {
		where = append(where, entclientlog.ServerID(f.ServerID))
	}
	if f.PID != "" {
		where = append(where, entclientlog.PlayerID(f.PID))
	}
	if f.Reason != "" {
		where = append(where, entclientlog.Reason(f.Reason))
	}
	if f.Since != nil {
		where = append(where, entclientlog.CreatedAtGTE(*f.Since))
	}
	if f.Until != nil {
		where = append(where, entclientlog.CreatedAtLT(*f.Until))
	}
	if f.Q != "" {
		where = append(where, entclientlog.Or(
			entclientlog.SummaryContainsFold(f.Q),
			entclientlog.NoteContainsFold(f.Q),
			entclientlog.DeviceContainsFold(f.Q),
		))
	}
	total, err := r.data.db.GameClientLog.Query().Where(where...).Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	q := r.data.db.GameClientLog.Query().
		Where(where...).
		Order(newestFirst()...)
	if f.Limit > 0 {
		q = q.Offset(f.Offset).Limit(f.Limit)
	}
	list, err := q.
		Select(
			entclientlog.FieldID,
			entclientlog.FieldServerID,
			entclientlog.FieldPlayerID,
			entclientlog.FieldReason,
			entclientlog.FieldSummary,
			entclientlog.FieldNote,
			entclientlog.FieldDevice,
			entclientlog.FieldClientVersion,
			entclientlog.FieldSize,
			entclientlog.FieldOccurredAt,
			entclientlog.FieldCreatedAt,
		).
		All(ctx)
	return list, total, err
}

func (r *gameClientLogRepo) Get(ctx context.Context, id string) (*gen.GameClientLog, error) {
	return r.data.db.GameClientLog.Get(ctx, id)
}

func (r *gameClientLogRepo) Delete(ctx context.Context, id string) error {
	return r.data.db.GameClientLog.DeleteOneID(id).Exec(ctx)
}
