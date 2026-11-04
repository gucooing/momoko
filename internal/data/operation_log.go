package data

import (
	"context"
	"strings"
	"time"

	"momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/operationlog"
)

type operationLogRepo struct {
	data *Data
}

func NewOperationLogRepo(data *Data) biz.OperationLogRepo {
	return &operationLogRepo{
		data: data,
	}
}

func (r *operationLogRepo) CreateOperationLog(ctx context.Context, log *v1.OperationLogInfo) error {
	if log == nil {
		return nil
	}
	create := r.data.db.OperationLog.Create().
		SetOperationType(log.OperationType.String()).
		SetSuccess(log.Success).
		SetDetail(log.Detail).
		SetIP(log.Ip).
		SetUserAgent(log.UserAgent).
		SetPath(log.Path).
		SetDurationMs(log.DurationMs).
		SetOperationTime(log.OperationTime.AsTime())
	if log.UserId != nil && *log.UserId != "" {
		create.SetUserID(*log.UserId)
	}

	return create.Exec(ctx)
}

// operationLogQuery 按请求条件构造查询；withSuccess=false 时忽略「是否成功」条件（用于成功/失败计数）。
func (r *operationLogRepo) operationLogQuery(req *v1.ListOperationLogsRequest, withSuccess bool) *gen.OperationLogQuery {
	query := r.data.db.OperationLog.Query()

	if req.UserId != nil && *req.UserId != "" {
		query = query.Where(operationlog.UserIDEQ(*req.UserId))
	}
	if req.OperationType != nil {
		query = query.Where(operationlog.OperationTypeEQ(req.OperationType.String()))
	}
	if req.Module != nil && *req.Module != "" {
		query = query.Where(operationlog.OperationTypeHasPrefix("OperationType" + *req.Module))
	}
	if withSuccess && req.Success != nil {
		query = query.Where(operationlog.SuccessEQ(*req.Success))
	}
	if req.Path != nil && *req.Path != "" {
		query = query.Where(operationlog.PathEQ(*req.Path))
	}
	if req.Keywords != nil && strings.TrimSpace(*req.Keywords) != "" {
		kw := strings.TrimSpace(*req.Keywords)
		query = query.Where(operationlog.Or(
			operationlog.PathContainsFold(kw),
			operationlog.IPContains(kw),
			operationlog.DetailContainsFold(kw),
		))
	}
	if req.StartTime != nil {
		query = query.Where(operationlog.OperationTimeGTE(req.StartTime.AsTime()))
	}
	if req.EndTime != nil {
		query = query.Where(operationlog.OperationTimeLTE(req.EndTime.AsTime()))
	}
	return query
}

func (r *operationLogRepo) ListOperationLogs(ctx context.Context, req *v1.ListOperationLogsRequest) ([]*gen.OperationLog, int64, error) {
	query := r.operationLogQuery(req, true)

	total, err := query.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	logs, err := query.
		Order(gen.Desc(operationlog.FieldOperationTime)).
		Offset(int((req.Page - 1) * req.PageSize)).
		Limit(int(req.PageSize)).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}

	return logs, int64(total), nil
}

// CountOperationLogsBySuccess 统计同一组条件（忽略是否成功）下的成功数与失败数。
func (r *operationLogRepo) CountOperationLogsBySuccess(ctx context.Context, req *v1.ListOperationLogsRequest) (int64, int64, error) {
	base := r.operationLogQuery(req, false)
	failed, err := base.Clone().Where(operationlog.SuccessEQ(false)).Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	all, err := base.Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	return int64(all - failed), int64(failed), nil
}

// OperationLogStats 统计 since 之后的操作数、失败数、活跃用户数与经主控转发数，以及全部记录数和最早记录时间。
func (r *operationLogRepo) OperationLogStats(ctx context.Context, since time.Time) (*biz.OperationLogStats, error) {
	today := r.data.db.OperationLog.Query().Where(operationlog.OperationTimeGTE(since))
	out := &biz.OperationLogStats{}
	n, err := today.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	out.TodayTotal = int64(n)
	if n, err = today.Clone().Where(operationlog.SuccessEQ(false)).Count(ctx); err != nil {
		return nil, err
	}
	out.TodayFailed = int64(n)
	users, err := today.Clone().Where(operationlog.UserIDNotNil()).Unique(true).Select(operationlog.FieldUserID).Strings(ctx)
	if err != nil {
		return nil, err
	}
	out.TodayActiveUsers = int64(len(users))
	if n, err = today.Clone().Where(operationlog.Or(
		operationlog.DetailContains(`"node":"`),
		operationlog.DetailContains(`"api_key":"`),
	)).Count(ctx); err != nil {
		return nil, err
	}
	out.TodayForwarded = int64(n)
	if n, err = r.data.db.OperationLog.Query().Count(ctx); err != nil {
		return nil, err
	}
	out.Total = int64(n)
	if n > 0 {
		first, err := r.data.db.OperationLog.Query().Order(gen.Asc(operationlog.FieldOperationTime)).First(ctx)
		if err != nil {
			return nil, err
		}
		t := first.OperationTime
		out.EarliestTime = &t
	}
	return out, nil
}

// DeleteOperationLogsBefore 删除操作时间早于 before 的记录，返回删除条数。
func (r *operationLogRepo) DeleteOperationLogsBefore(ctx context.Context, before time.Time) (int64, error) {
	n, err := r.data.db.OperationLog.Delete().Where(operationlog.OperationTimeLT(before)).Exec(ctx)
	return int64(n), err
}
