package data

import (
	"context"
	"encoding/json"
	"time"

	"momoko/internal/data/ent/gen"
	enttask "momoko/internal/data/ent/gen/task"
	"momoko/pkg/task"
)

// taskRepo 把通用任务管理器（pkg/task）的持久化落到 ent task 表。
type taskRepo struct {
	data *Data
}

// NewTaskRepo 创建任务持久化仓储。
func NewTaskRepo(data *Data) task.Store {
	return &taskRepo{data: data}
}

var _ task.Store = (*taskRepo)(nil)

func (r *taskRepo) Upsert(ctx context.Context, rec *task.Record) error {
	exists, err := r.data.db.Task.Query().Where(enttask.IDEQ(rec.ID)).Exist(ctx)
	if err != nil {
		return err
	}
	resultJSON := marshalResults(rec.Results)
	if exists {
		upd := r.data.db.Task.UpdateOneID(rec.ID).
			SetType(rec.Type).
			SetKind(enttask.Kind(rec.Kind)).
			SetStatus(enttask.Status(rec.Status)).
			SetResumePolicy(enttask.ResumePolicy(rec.Resume)).
			SetTitle(rec.Title).
			SetUserID(rec.UserID).
			SetPayload(string(rec.Payload)).
			SetState(string(rec.State)).
			SetResult(resultJSON).
			SetProgressTotal(rec.Total).
			SetProgressFinished(rec.Finished).
			SetMessage(rec.Message).
			SetError(rec.Error).
			SetIntervalMs(rec.IntervalMS).
			SetTimeoutMs(rec.TimeoutMS)
		if rec.EndTime != nil {
			upd.SetEndTime(*rec.EndTime)
		} else {
			upd.ClearEndTime()
		}
		return upd.Exec(ctx)
	}
	c := r.data.db.Task.Create().
		SetID(rec.ID).
		SetType(rec.Type).
		SetKind(enttask.Kind(rec.Kind)).
		SetStatus(enttask.Status(rec.Status)).
		SetResumePolicy(enttask.ResumePolicy(rec.Resume)).
		SetTitle(rec.Title).
		SetUserID(rec.UserID).
		SetPayload(string(rec.Payload)).
		SetState(string(rec.State)).
		SetResult(resultJSON).
		SetProgressTotal(rec.Total).
		SetProgressFinished(rec.Finished).
		SetMessage(rec.Message).
		SetError(rec.Error).
		SetIntervalMs(rec.IntervalMS).
		SetTimeoutMs(rec.TimeoutMS)
	if !rec.CreateTime.IsZero() {
		c.SetCreateTime(rec.CreateTime)
	}
	if rec.EndTime != nil {
		c.SetEndTime(*rec.EndTime)
	}
	return c.Exec(ctx)
}

func (r *taskRepo) SetStatus(ctx context.Context, id string, s task.Status, message, errText string, end *time.Time) error {
	upd := r.data.db.Task.UpdateOneID(id).SetStatus(enttask.Status(s))
	if message != "" {
		upd.SetMessage(message)
	}
	if errText != "" {
		upd.SetError(errText)
	}
	if end != nil {
		upd.SetEndTime(*end)
	}
	return ignoreNotFound(upd.Exec(ctx))
}

func (r *taskRepo) SetProgress(ctx context.Context, id string, finished, total int64, message string) error {
	upd := r.data.db.Task.UpdateOneID(id).
		SetProgressFinished(finished).
		SetProgressTotal(total)
	if message != "" {
		upd.SetMessage(message)
	}
	return ignoreNotFound(upd.Exec(ctx))
}

func (r *taskRepo) SaveState(ctx context.Context, id string, state json.RawMessage) error {
	return ignoreNotFound(r.data.db.Task.UpdateOneID(id).SetState(string(state)).Exec(ctx))
}

func (r *taskRepo) SaveResults(ctx context.Context, id string, results []task.Result) error {
	return ignoreNotFound(r.data.db.Task.UpdateOneID(id).SetResult(marshalResults(results)).Exec(ctx))
}

func (r *taskRepo) Get(ctx context.Context, id string) (*task.Record, error) {
	row, err := r.data.db.Task.Query().Where(enttask.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return toTaskRecord(row), nil
}

func (r *taskRepo) List(ctx context.Context, f task.Filter, page, pageSize int64) ([]*task.Record, int64, error) {
	q := r.data.db.Task.Query()
	if f.UserID != "" {
		q = q.Where(enttask.UserIDEQ(f.UserID))
	}
	if f.TypePrefix != "" {
		q = q.Where(enttask.TypeHasPrefix(f.TypePrefix))
	}
	if f.Status != "" {
		q = q.Where(enttask.StatusEQ(enttask.Status(f.Status)))
	}
	if f.Kind != "" {
		q = q.Where(enttask.KindEQ(enttask.Kind(f.Kind)))
	}
	if f.Keywords != "" {
		q = q.Where(enttask.Or(
			enttask.TitleContainsFold(f.Keywords),
			enttask.TypeContainsFold(f.Keywords),
		))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	rows, err := q.
		Order(gen.Desc(enttask.FieldCreateTime)).
		Offset(int((page - 1) * pageSize)).
		Limit(int(pageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	recs := make([]*task.Record, 0, len(rows))
	for _, row := range rows {
		recs = append(recs, toTaskRecord(row))
	}
	return recs, int64(total), nil
}

func (r *taskRepo) LoadResumable(ctx context.Context) ([]*task.Record, error) {
	rows, err := r.data.db.Task.Query().
		Where(enttask.StatusIn(enttask.StatusPending, enttask.StatusRunning)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	recs := make([]*task.Record, 0, len(rows))
	for _, row := range rows {
		recs = append(recs, toTaskRecord(row))
	}
	return recs, nil
}

func (r *taskRepo) Delete(ctx context.Context, id string) error {
	return ignoreNotFound(r.data.db.Task.DeleteOneID(id).Exec(ctx))
}

// ---- 映射 ----

func toTaskRecord(row *gen.Task) *task.Record {
	rec := &task.Record{
		ID:         row.ID,
		Type:       row.Type,
		Kind:       task.Kind(row.Kind),
		Status:     task.Status(row.Status),
		Resume:     task.ResumePolicy(row.ResumePolicy),
		Title:      row.Title,
		UserID:     row.UserID,
		Payload:    json.RawMessage(row.Payload),
		State:      json.RawMessage(row.State),
		Results:    unmarshalResults(row.Result),
		Total:      row.ProgressTotal,
		Finished:   row.ProgressFinished,
		Message:    row.Message,
		Error:      row.Error,
		IntervalMS: row.IntervalMs,
		TimeoutMS:  row.TimeoutMs,
		CreateTime: row.CreateTime,
		EndTime:    row.EndTime,
	}
	return rec
}

func marshalResults(results []task.Result) string {
	if len(results) == 0 {
		return ""
	}
	b, err := json.Marshal(results)
	if err != nil {
		return ""
	}
	return string(b)
}

func unmarshalResults(s string) []task.Result {
	if s == "" {
		return nil
	}
	var results []task.Result
	if err := json.Unmarshal([]byte(s), &results); err != nil {
		return nil
	}
	return results
}

// ignoreNotFound 把「行已被删」视作幂等成功（任务终态写入与删除可能并发）。
func ignoreNotFound(err error) error {
	if err != nil && gen.IsNotFound(err) {
		return nil
	}
	return err
}
