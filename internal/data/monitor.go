package data

import (
	"context"
	"time"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/monitorsample"
)

type monitorRepo struct {
	data *Data
}

func NewMonitorRepo(data *Data) biz.MonitorRepo {
	return &monitorRepo{data: data}
}

// InsertSamples 批量写入；同一精度同一时间已存在的行保持不动（重复聚合时幂等）。
func (r *monitorRepo) InsertSamples(ctx context.Context, rows []*biz.MonitorRow) error {
	const batch = 200
	for start := 0; start < len(rows); start += batch {
		end := min(start+batch, len(rows))
		builders := make([]*gen.MonitorSampleCreate, 0, end-start)
		for _, row := range rows[start:end] {
			builders = append(builders, r.data.db.MonitorSample.Create().
				SetTier(row.Tier).
				SetSampleTime(row.Time).
				SetData(row.Data))
		}
		if err := r.data.db.MonitorSample.CreateBulk(builders...).
			OnConflictColumns(monitorsample.FieldTier, monitorsample.FieldSampleTime).
			DoNothing().
			Exec(ctx); err != nil {
			return err
		}
	}
	return nil
}

// ListSamples 按时间升序返回某精度在 [from, to) 内的行。
func (r *monitorRepo) ListSamples(ctx context.Context, tier int8, from, to time.Time) ([]*biz.MonitorRow, error) {
	items, err := r.data.db.MonitorSample.Query().
		Where(
			monitorsample.TierEQ(tier),
			monitorsample.SampleTimeGTE(from),
			monitorsample.SampleTimeLT(to),
		).
		Order(gen.Asc(monitorsample.FieldSampleTime)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*biz.MonitorRow, 0, len(items))
	for _, item := range items {
		out = append(out, &biz.MonitorRow{Tier: item.Tier, Time: item.SampleTime, Data: item.Data})
	}
	return out, nil
}

// EdgeSample 返回某精度在 since 之后最早（first=true）或全部数据里最新（first=false）的一行；没有时返回 nil。
func (r *monitorRepo) EdgeSample(ctx context.Context, tier int8, since time.Time, first bool) (*biz.MonitorRow, error) {
	q := r.data.db.MonitorSample.Query().Where(monitorsample.TierEQ(tier))
	if first {
		q = q.Where(monitorsample.SampleTimeGTE(since)).Order(gen.Asc(monitorsample.FieldSampleTime))
	} else {
		q = q.Order(gen.Desc(monitorsample.FieldSampleTime))
	}
	item, err := q.First(ctx)
	if gen.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &biz.MonitorRow{Tier: item.Tier, Time: item.SampleTime, Data: item.Data}, nil
}

// EarliestTime 返回全部精度里最早一行的时间。
func (r *monitorRepo) EarliestTime(ctx context.Context) (time.Time, bool, error) {
	item, err := r.data.db.MonitorSample.Query().
		Order(gen.Asc(monitorsample.FieldSampleTime)).
		Select(monitorsample.FieldSampleTime).
		First(ctx)
	if gen.IsNotFound(err) {
		return time.Time{}, false, nil
	}
	if err != nil {
		return time.Time{}, false, err
	}
	return item.SampleTime, true, nil
}

// DeleteSamplesBefore 删除某精度早于 before 的行；before 为零值时删除该精度全部行。
func (r *monitorRepo) DeleteSamplesBefore(ctx context.Context, tier int8, before time.Time) (int64, error) {
	q := r.data.db.MonitorSample.Delete().Where(monitorsample.TierEQ(tier))
	if !before.IsZero() {
		q = q.Where(monitorsample.SampleTimeLT(before))
	}
	n, err := q.Exec(ctx)
	return int64(n), err
}

// DeleteAll 清空全部历史。
func (r *monitorRepo) DeleteAll(ctx context.Context) (int64, error) {
	n, err := r.data.db.MonitorSample.Delete().Exec(ctx)
	return int64(n), err
}

// StorageStats 返回行数与 data 列的总长度（字节，用于估算占用）。
func (r *monitorRepo) StorageStats(ctx context.Context) (int64, int64, error) {
	rows, err := r.data.db.QueryContext(ctx, "SELECT COUNT(*), COALESCE(SUM(LENGTH(data)), 0) FROM monitor_samples")
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()
	var count, size int64
	if rows.Next() {
		if err := rows.Scan(&count, &size); err != nil {
			return 0, 0, err
		}
	}
	return count, size, rows.Err()
}
