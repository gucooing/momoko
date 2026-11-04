package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/common"
	"momoko/pkg/response"
	"momoko/pkg/task"
	"momoko/pkg/utils"

	"google.golang.org/protobuf/types/known/timestamppb"
)

type OperationLogRepo interface {
	CreateOperationLog(context.Context, *v1.OperationLogInfo) error
	ListOperationLogs(context.Context, *v1.ListOperationLogsRequest) ([]*gen.OperationLog, int64, error)
	CountOperationLogsBySuccess(context.Context, *v1.ListOperationLogsRequest) (int64, int64, error)
	OperationLogStats(ctx context.Context, since time.Time) (*OperationLogStats, error)
	DeleteOperationLogsBefore(ctx context.Context, before time.Time) (int64, error)
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
	repo   OperationLogRepo
	config ConfigRepo
}

// operationLogCleanupType 是「操作记录清理」定时单例的类型键；operationLogCleanupInterval 为触发周期。
const (
	operationLogCleanupType     = "audit.cleanup"
	operationLogCleanupInterval = 6 * time.Hour
)

func NewOperationLogUsecase(repo OperationLogRepo, config ConfigRepo, tasks *task.Manager) *OperationLogUsecase {
	uc := &OperationLogUsecase{repo: repo, config: config}
	_ = tasks.EnsureSingleton(context.Background(), &operationLogCleanupTask{uc: uc})
	return uc
}

// RetentionDays 返回操作记录保留天数（0=永久保留）。
func (u *OperationLogUsecase) RetentionDays(ctx context.Context) int32 {
	raw, err := u.config.Get(ctx, common.ConfigAuditRetentionDays)
	if err != nil {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return int32(n)
}

// UpdateRetention 修改保留天数并立即清理一次超期记录。
func (u *OperationLogUsecase) UpdateRetention(ctx context.Context, days int32) (*v1.UpdateOperationLogRetentionResponse, error) {
	if days != 0 && (days < 7 || days > 3650) {
		return nil, response.BadRequest(400, "保留天数需为 0（永久保留）或 7 到 3650 天")
	}
	if err := u.config.BatchUpdate(ctx, map[common.ConfigKey]string{
		common.ConfigAuditRetentionDays: strconv.Itoa(int(days)),
	}); err != nil {
		return nil, ErrSystem(err)
	}
	deleted, err := u.purge(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.UpdateOperationLogRetentionResponse{RetentionDays: days, Deleted: deleted}, nil
}

// purge 按保留天数删除超期记录；永久保留时不做任何事。
func (u *OperationLogUsecase) purge(ctx context.Context) (int64, error) {
	days := u.RetentionDays(ctx)
	if days <= 0 {
		return 0, nil
	}
	return u.repo.DeleteOperationLogsBefore(ctx, time.Now().AddDate(0, 0, -int(days)))
}

// operationLogCleanupTask 周期性删除超出保留期的操作记录（在任务面板可见）。
type operationLogCleanupTask struct {
	uc *OperationLogUsecase
}

func (t *operationLogCleanupTask) Meta() task.Meta {
	return task.Meta{
		ID:       operationLogCleanupType,
		Type:     operationLogCleanupType,
		Kind:     task.KindScheduled,
		Resume:   task.ResumeAlways,
		Interval: operationLogCleanupInterval,
		Title:    "操作记录清理",
		Source:   "task",
	}
}

func (t *operationLogCleanupTask) Payload() any { return nil }

func (t *operationLogCleanupTask) Run(ctx context.Context, r task.Reporter) error {
	n, err := t.uc.purge(ctx)
	if err != nil {
		return err
	}
	if days := t.uc.RetentionDays(ctx); days > 0 {
		r.Log(task.LogInfo, fmt.Sprintf("已清理 %d 条超过 %d 天的操作记录", n, days))
	}
	return nil
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
		RetentionDays:    u.RetentionDays(ctx),
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
		Id:            int64(log.ID),
		UserId:        log.UserID,
		OperationType: operationTypeFromString(log.OperationType),
		Success:       log.Success,
		Detail:        log.Detail,
		Ip:            log.IP,
		IpScope:       IPScope(log.IP),
		UserAgent:     log.UserAgent,
		Path:          log.Path,
		DurationMs:    log.DurationMs,
		OperationTime: timestamppb.New(log.OperationTime),
		StatusCode:    operationStatusCode(log),
	}
}

// operationStatusCode 取记录的响应状态码：详情里记了就用，否则成功 200 / 失败 500。
func operationStatusCode(log *gen.OperationLog) int32 {
	var d struct {
		Code int32 `json:"code"`
	}
	if json.Unmarshal([]byte(log.Detail), &d) == nil && d.Code > 0 {
		return d.Code
	}
	if log.Success {
		return 200
	}
	return 500
}

func operationTypeFromString(value string) v1.OperationType {
	if item, ok := v1.OperationType_value[value]; ok {
		return v1.OperationType(item)
	}
	return v1.OperationType_OperationTypeUncategorized
}
