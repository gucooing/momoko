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
	trigger := rec.Trigger
	if trigger == "" {
		trigger = task.TriggerSystem
	}
	if exists {
		upd := r.data.db.Task.UpdateOneID(rec.ID).
			SetType(rec.Type).
			SetKind(enttask.Kind(rec.Kind)).
			SetStatus(enttask.Status(rec.Status)).
			SetResumePolicy(enttask.ResumePolicy(rec.Resume)).
			SetTitle(rec.Title).
			SetTarget(rec.Target).
			SetSource(rec.Source).
			SetTriggerType(enttask.TriggerType(trigger)).
			SetUserID(rec.UserID).
			SetOperator(rec.Operator).
			SetPayload(string(rec.Payload)).
			SetState(string(rec.State)).
			SetResult(marshalJSON(rec.Results)).
			SetSteps(marshalJSON(rec.Steps)).
			SetLogs(marshalJSON(rec.Logs)).
			SetProgressTotal(rec.Total).
			SetProgressFinished(rec.Finished).
			SetMessage(rec.Message).
			SetError(rec.Error).
			SetCanceledBy(rec.CanceledBy).
			SetIntervalMs(rec.IntervalMS).
			SetTimeoutMs(rec.TimeoutMS)
		if rec.StartTime != nil {
			upd.SetStartTime(*rec.StartTime)
		} else {
			upd.ClearStartTime()
		}
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
		SetTarget(rec.Target).
		SetSource(rec.Source).
		SetTriggerType(enttask.TriggerType(trigger)).
		SetUserID(rec.UserID).
		SetOperator(rec.Operator).
		SetPayload(string(rec.Payload)).
		SetState(string(rec.State)).
		SetResult(marshalJSON(rec.Results)).
		SetSteps(marshalJSON(rec.Steps)).
		SetLogs(marshalJSON(rec.Logs)).
		SetProgressTotal(rec.Total).
		SetProgressFinished(rec.Finished).
		SetMessage(rec.Message).
		SetError(rec.Error).
		SetCanceledBy(rec.CanceledBy).
		SetIntervalMs(rec.IntervalMS).
		SetTimeoutMs(rec.TimeoutMS).
		SetNillableStartTime(rec.StartTime).
		SetNillableEndTime(rec.EndTime)
	if !rec.CreateTime.IsZero() {
		c.SetCreateTime(rec.CreateTime)
	}
	return c.Exec(ctx)
}

func (r *taskRepo) SetRunning(ctx context.Context, id string, start time.Time) error {
	return ignoreNotFound(r.data.db.Task.UpdateOneID(id).
		SetStatus(enttask.StatusRunning).
		SetStartTime(start).
		Exec(ctx))
}

func (r *taskRepo) SetStatus(ctx context.Context, id string, s task.Status) error {
	return ignoreNotFound(r.data.db.Task.UpdateOneID(id).SetStatus(enttask.Status(s)).Exec(ctx))
}

func (r *taskRepo) SaveSnapshot(ctx context.Context, id string, s task.Snapshot) error {
	return ignoreNotFound(r.data.db.Task.UpdateOneID(id).
		SetProgressTotal(s.Total).
		SetProgressFinished(s.Finished).
		SetMessage(s.Message).
		SetSteps(marshalJSON(s.Steps)).
		SetLogs(marshalJSON(s.Logs)).
		Exec(ctx))
}

func (r *taskRepo) Finish(ctx context.Context, id string, s task.Status, errText, canceledBy string, end time.Time, snap task.Snapshot) error {
	return ignoreNotFound(r.data.db.Task.UpdateOneID(id).
		SetStatus(enttask.Status(s)).
		SetError(errText).
		SetCanceledBy(canceledBy).
		SetEndTime(end).
		SetProgressTotal(snap.Total).
		SetProgressFinished(snap.Finished).
		SetMessage(snap.Message).
		SetSteps(marshalJSON(snap.Steps)).
		SetLogs(marshalJSON(snap.Logs)).
		Exec(ctx))
}

func (r *taskRepo) SaveState(ctx context.Context, id string, state json.RawMessage) error {
	return ignoreNotFound(r.data.db.Task.UpdateOneID(id).SetState(string(state)).Exec(ctx))
}

func (r *taskRepo) SaveResults(ctx context.Context, id string, results []task.Result) error {
	return ignoreNotFound(r.data.db.Task.UpdateOneID(id).SetResult(marshalJSON(results)).Exec(ctx))
}

func (r *taskRepo) Get(ctx context.Context, id string) (*task.Record, error) {
	row, err := r.data.db.Task.Query().Where(enttask.IDEQ(id)).Only(ctx)
	if err != nil {
		return nil, err
	}
	return toTaskRecord(row, true), nil
}

func (r *taskRepo) List(ctx context.Context, f task.Filter, page, pageSize int64) ([]*task.Record, int64, error) {
	q := r.applyFilter(r.data.db.Task.Query(), f)
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
		recs = append(recs, toTaskRecord(row, false))
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
		recs = append(recs, toTaskRecord(row, true))
	}
	return recs, nil
}

func (r *taskRepo) Delete(ctx context.Context, ids []string) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	n, err := r.data.db.Task.Delete().Where(enttask.IDIn(ids...)).Exec(ctx)
	return int64(n), err
}

func (r *taskRepo) applyFilter(q *gen.TaskQuery, f task.Filter) *gen.TaskQuery {
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
			enttask.TargetContainsFold(f.Keywords),
		))
	}
	return q
}

func (r *taskRepo) CountByStatus(ctx context.Context, f task.Filter) (map[task.Status]int64, error) {
	var rows []struct {
		Status string `json:"status"`
		Count  int    `json:"count"`
	}
	err := r.applyFilter(r.data.db.Task.Query(), f).
		GroupBy(enttask.FieldStatus).
		Aggregate(gen.Count()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	out := make(map[task.Status]int64, len(rows))
	for _, row := range rows {
		out[task.Status(row.Status)] = int64(row.Count)
	}
	return out, nil
}

func (r *taskRepo) CountByType(ctx context.Context) (map[string]int64, error) {
	var rows []struct {
		Type  string `json:"type"`
		Count int    `json:"count"`
	}
	err := r.data.db.Task.Query().
		GroupBy(enttask.FieldType).
		Aggregate(gen.Count()).
		Scan(ctx, &rows)
	if err != nil {
		return nil, err
	}
	out := make(map[string]int64, len(rows))
	for _, row := range rows {
		out[row.Type] = int64(row.Count)
	}
	return out, nil
}

func (r *taskRepo) DeleteByStatus(ctx context.Context, statuses []task.Status) (int64, error) {
	if len(statuses) == 0 {
		return 0, nil
	}
	values := make([]enttask.Status, 0, len(statuses))
	for _, s := range statuses {
		values = append(values, enttask.Status(s))
	}
	n, err := r.data.db.Task.Delete().Where(enttask.StatusIn(values...)).Exec(ctx)
	return int64(n), err
}

func (r *taskRepo) DeleteFinishedBefore(ctx context.Context, before time.Time) (int64, error) {
	n, err := r.data.db.Task.Delete().Where(
		enttask.StatusIn(enttask.StatusSuccess, enttask.StatusFailed, enttask.StatusCanceled),
		enttask.EndTimeLT(before),
	).Exec(ctx)
	return int64(n), err
}

// ---- 映射 ----

// toTaskRecord 映射一行；withLogs=false 时不解析日志（列表只需要行数，日志按需单查）。
func toTaskRecord(row *gen.Task, withLogs bool) *task.Record {
	rec := &task.Record{
		ID:         row.ID,
		Type:       row.Type,
		Kind:       task.Kind(row.Kind),
		Status:     task.Status(row.Status),
		Resume:     task.ResumePolicy(row.ResumePolicy),
		Title:      row.Title,
		Target:     row.Target,
		Source:     row.Source,
		Trigger:    task.Trigger(row.TriggerType),
		UserID:     row.UserID,
		Operator:   row.Operator,
		Payload:    json.RawMessage(row.Payload),
		State:      json.RawMessage(row.State),
		Results:    unmarshalJSON[task.Result](row.Result),
		Steps:      unmarshalJSON[task.Step](row.Steps),
		Total:      row.ProgressTotal,
		Finished:   row.ProgressFinished,
		Message:    row.Message,
		Error:      row.Error,
		CanceledBy: row.CanceledBy,
		IntervalMS: row.IntervalMs,
		TimeoutMS:  row.TimeoutMs,
		CreateTime: row.CreateTime,
		StartTime:  row.StartTime,
		EndTime:    row.EndTime,
	}
	if withLogs {
		rec.Logs = unmarshalJSON[task.LogEntry](row.Logs)
	}
	return rec
}

func marshalJSON[T any](items []T) string {
	if len(items) == 0 {
		return ""
	}
	b, err := json.Marshal(items)
	if err != nil {
		return ""
	}
	return string(b)
}

func unmarshalJSON[T any](s string) []T {
	if s == "" {
		return nil
	}
	var items []T
	if err := json.Unmarshal([]byte(s), &items); err != nil {
		return nil
	}
	return items
}

// ignoreNotFound 把「行已被删」视作幂等成功（任务终态写入与删除可能并发）。
func ignoreNotFound(err error) error {
	if err != nil && gen.IsNotFound(err) {
		return nil
	}
	return err
}
