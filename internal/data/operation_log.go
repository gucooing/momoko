package data

import (
	"context"

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

func (r *operationLogRepo) ListOperationLogs(ctx context.Context, req *v1.ListOperationLogsRequest) ([]*gen.OperationLog, int64, error) {
	query := r.data.db.OperationLog.Query()

	if req.UserId != nil && *req.UserId != "" {
		query = query.Where(operationlog.UserIDEQ(*req.UserId))
	}
	if req.OperationType != nil {
		query = query.Where(operationlog.OperationTypeEQ(req.OperationType.String()))
	}
	if req.Success != nil {
		query = query.Where(operationlog.SuccessEQ(*req.Success))
	}
	if req.Path != nil && *req.Path != "" {
		query = query.Where(operationlog.PathEQ(*req.Path))
	}
	if req.StartTime != nil {
		query = query.Where(operationlog.OperationTimeGTE(req.StartTime.AsTime()))
	}
	if req.EndTime != nil {
		query = query.Where(operationlog.OperationTimeLTE(req.EndTime.AsTime()))
	}

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
