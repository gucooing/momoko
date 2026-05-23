package biz

import (
	"context"
	"momoko/internal/data/ent/gen"

	"momoko/api/gen/v1"
	"momoko/pkg/common"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type OperationLogRepo interface {
	CreateOperationLog(context.Context, *v1.OperationLogInfo) error
	ListOperationLogs(context.Context, *v1.ListOperationLogsRequest) ([]*gen.OperationLog, int64, error)
}

type OperationLogUsecase struct {
	repo OperationLogRepo
}

func NewOperationLogUsecase(repo OperationLogRepo) *OperationLogUsecase {
	return &OperationLogUsecase{
		repo: repo,
	}
}

func (u *OperationLogUsecase) CreateOperationLog(ctx context.Context, log *v1.OperationLogInfo) error {
	if log == nil {
		return nil
	}
	if err := u.repo.CreateOperationLog(ctx, log); err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (u *OperationLogUsecase) ListOperationLogs(ctx context.Context, req *v1.ListOperationLogsRequest) (*v1.ListOperationLogsResponse, error) {
	if req == nil {
		req = &v1.ListOperationLogsRequest{}
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	logs, total, err := u.repo.ListOperationLogs(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	items := make([]*v1.OperationLogInfo, 0, len(logs))
	for _, item := range logs {
		items = append(items, toOperationLogInfo(item))
	}
	return &v1.ListOperationLogsResponse{
		Logs:     items,
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
	}, nil
}

func (u *OperationLogUsecase) MyLoginLogs(ctx context.Context, userID string, req *v1.MyLoginLogsRequest) (*v1.MyLoginLogsResponse, error) {
	if req == nil {
		req = &v1.MyLoginLogsRequest{}
	}

	logs, err := u.ListOperationLogs(ctx, &v1.ListOperationLogsRequest{
		UserId:        &userID,
		OperationType: new(common.OperationTypeLogin.String()),
		Page:          req.Page,
		PageSize:      req.PageSize,
	})
	if err != nil {
		return nil, err
	}

	return &v1.MyLoginLogsResponse{
		Logs:     logs.Logs,
		Page:     logs.Page,
		PageSize: logs.PageSize,
		Total:    logs.Total,
	}, nil
}

func toOperationLogInfo(log *gen.OperationLog) *v1.OperationLogInfo {
	return &v1.OperationLogInfo{
		UserId:        log.UserID,
		OperationType: log.OperationType,
		Success:       log.Success,
		Detail:        log.Detail,
		Ip:            log.IP,
		UserAgent:     log.UserAgent,
		Path:          log.Path,
		DurationMs:    log.DurationMs,
		OperationTime: timestamppb.New(log.OperationTime),
	}
}
