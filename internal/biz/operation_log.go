package biz

import (
	"context"
	"encoding/json"
	"momoko/pkg/utils"
	"momoko/internal/data/ent/gen"
	"time"

	"momoko/api/gen/v1"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type OperationLogRepo interface {
	CreateOperationLog(context.Context, *v1.OperationLogInfo) error
	ListOperationLogs(context.Context, *v1.ListOperationLogsRequest) ([]*gen.OperationLog, int64, error)
	CountOperationLogsBySuccess(context.Context, *v1.ListOperationLogsRequest) (int64, int64, error)
	OperationLogStats(ctx context.Context, since time.Time) (*OperationLogStats, error)
}

// OperationLogStats 操作日志统计。
type OperationLogStats struct {
	TodayTotal       int64
	TodayFailed      int64
	TodayActiveUsers int64
	TodayForwarded   int64
	Total            int64
	EarliestTime     *time.Time
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
	successCount, failedCount, err := u.repo.CountOperationLogsBySuccess(ctx, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.ListOperationLogsResponse{
		Logs:         items,
		Page:         req.Page,
		PageSize:     req.PageSize,
		Total:        total,
		SuccessCount: successCount,
		FailedCount:  failedCount,
	}, nil
}

// OperationLogStats 返回今日（服务器本地时间）操作统计与全部记录概况。
func (u *OperationLogUsecase) OperationLogStats(ctx context.Context) (*v1.OperationLogStatsResponse, error) {
	now := time.Now()
	since := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	stats, err := u.repo.OperationLogStats(ctx, since)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.OperationLogStatsResponse{
		TodayTotal:       stats.TodayTotal,
		TodayFailed:      stats.TodayFailed,
		TodayActiveUsers: stats.TodayActiveUsers,
		TodayForwarded:   stats.TodayForwarded,
		Total:            stats.Total,
	}
	if stats.EarliestTime != nil {
		out.EarliestTime = timestamppb.New(*stats.EarliestTime)
	}
	return out, nil
}

func (u *OperationLogUsecase) MyLoginLogs(ctx context.Context, userID string, req *v1.MyLoginLogsRequest) (*v1.MyLoginLogsResponse, error) {
	if req == nil {
		req = &v1.MyLoginLogsRequest{}
	}
	listReq := &v1.ListOperationLogsRequest{
		UserId:        &userID,
		OperationType: v1.OperationType_OperationTypeAuthLogin.Enum(),
		Page:          req.Page,
		PageSize:      req.PageSize,
		Success:       req.Success,
	}
	if listReq.Page <= 0 {
		listReq.Page = 1
	}
	if listReq.PageSize <= 0 {
		listReq.PageSize = 10
	}
	if listReq.PageSize > 100 {
		listReq.PageSize = 100
	}
	rows, total, err := u.repo.ListOperationLogs(ctx, listReq)
	if err != nil {
		return nil, ErrSystem(err)
	}
	successCount, failedCount, err := u.repo.CountOperationLogsBySuccess(ctx, listReq)
	if err != nil {
		return nil, ErrSystem(err)
	}
	logs := make([]*v1.LoginLogInfo, 0, len(rows))
	for _, row := range rows {
		logs = append(logs, toLoginLogInfo(row))
	}
	return &v1.MyLoginLogsResponse{
		Logs:         logs,
		Page:         listReq.Page,
		PageSize:     listReq.PageSize,
		Total:        total,
		SuccessCount: successCount,
		FailedCount:  failedCount,
	}, nil
}

// toLoginLogInfo 从登录操作日志里解析出设备、登录方式与失败原因。
func toLoginLogInfo(log *gen.OperationLog) *v1.LoginLogInfo {
	device := utils.RequestString(log.Detail, "device")
	method := LoginMethodPassword
	if utils.RequestString(log.Detail, "email") != "" {
		method = LoginMethodEmailCode
	}
	var errText string
	var detail struct {
		Error string `json:"error"`
	}
	if json.Unmarshal([]byte(log.Detail), &detail) == nil {
		errText = detail.Error
	}
	return &v1.LoginLogInfo{
		Time:       timestamppb.New(log.OperationTime),
		Ip:         log.IP,
		IpScope:    IPScope(log.IP),
		UserAgent:  log.UserAgent,
		Device:     device,
		DeviceType: DeviceType(log.UserAgent, device),
		Method:     method,
		Success:    log.Success,
		Error:      errText,
		Detail:     log.Detail,
	}
}

func toOperationLogInfo(log *gen.OperationLog) *v1.OperationLogInfo {
	return &v1.OperationLogInfo{
		UserId:        log.UserID,
		OperationType: operationTypeFromString(log.OperationType),
		Success:       log.Success,
		Detail:        log.Detail,
		Ip:            log.IP,
		UserAgent:     log.UserAgent,
		Path:          log.Path,
		DurationMs:    log.DurationMs,
		OperationTime: timestamppb.New(log.OperationTime),
	}
}

func operationTypeFromString(value string) v1.OperationType {
	if item, ok := v1.OperationType_value[value]; ok {
		return v1.OperationType(item)
	}
	return v1.OperationType_OperationTypeUncategorized
}
