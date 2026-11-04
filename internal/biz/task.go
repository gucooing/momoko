package biz

import (
	"context"
	"sort"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/auth"
	"momoko/pkg/task"
)

// NewTaskManager 构造通用任务管理器并返回其停止函数（接进 wire cleanup）。
// 发起方解析：远程主控经 API Key 转发 → 主控转发（记主控侧操作者）；有登录用户 → 手动；否则系统。
func NewTaskManager(store task.Store) (*task.Manager, func()) {
	m := task.New(store)
	m.SetOriginResolver(taskOrigin)
	return m, m.Stop
}

func taskOrigin(ctx context.Context) task.Origin {
	if p, ok := NodePrincipalFromContext(ctx); ok && p != nil {
		return task.Origin{Trigger: task.TriggerMaster, UserID: p.UserID, Operator: p.Operator}
	}
	if uid := auth.GetUserIDFromContext(ctx); uid != nil && *uid != "" {
		return task.Origin{Trigger: task.TriggerManual, UserID: *uid}
	}
	return task.Origin{Trigger: task.TriggerSystem}
}

// TaskUsecase 是「任务管理」对外面板：列出/查看/取消/重试/删除任务管理器中的全部任务。
type TaskUsecase struct {
	manager *task.Manager
	user    UserRepo
}

// taskRetentionInterval 是任务记录清理的触发周期。
const taskRetentionInterval = 6 * time.Hour

func NewTaskUsecase(manager *task.Manager, user UserRepo) *TaskUsecase {
	_ = manager.EnsureSingleton(context.Background(), manager.NewRetentionTask(task.DefaultRetention, taskRetentionInterval))
	return &TaskUsecase{manager: manager, user: user}
}

func (t *TaskUsecase) ListTasks(ctx context.Context, req *v1.ListTasksRequest) (*v1.ListTasksResponse, error) {
	filter := task.Filter{
		Status:     task.Status(req.GetStatus()),
		Kind:       task.Kind(req.GetKind()),
		Keywords:   req.GetKeywords(),
		TypePrefix: req.GetType(),
	}
	infos, total, err := t.manager.List(ctx, filter, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, ErrSystem(err)
	}
	names := t.resolveNames(ctx, infos)
	items := make([]*v1.TaskInfo, 0, len(infos))
	for _, info := range infos {
		items = append(items, t.toTaskInfo(info, names))
	}
	byStatus, err := t.manager.CountByStatus(ctx, filter)
	if err != nil {
		return nil, ErrSystem(err)
	}
	counts := &v1.TaskStatusCounts{
		Pending:  byStatus[task.StatusPending],
		Running:  byStatus[task.StatusRunning],
		Success:  byStatus[task.StatusSuccess],
		Failed:   byStatus[task.StatusFailed],
		Canceled: byStatus[task.StatusCanceled],
	}
	for _, n := range byStatus {
		counts.All += n
	}
	byType, err := t.manager.CountByType(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	types := make([]*v1.TaskTypeCount, 0, len(byType))
	for typ, n := range byType {
		types = append(types, &v1.TaskTypeCount{Type: typ, Count: n})
	}
	sort.Slice(types, func(i, j int) bool {
		if types[i].Count != types[j].Count {
			return types[i].Count > types[j].Count
		}
		return types[i].Type < types[j].Type
	})
	return &v1.ListTasksResponse{
		Items:         items,
		Total:         total,
		Page:          req.GetPage(),
		PageSize:      req.GetPageSize(),
		Counts:        counts,
		Types:         types,
		RetentionDays: int32(task.DefaultRetention.Hours() / 24),
	}, nil
}

func (t *TaskUsecase) GetTask(ctx context.Context, id string) (*v1.TaskInfo, error) {
	info, ok := t.manager.Get(id)
	if !ok {
		return nil, ErrTaskNotFound
	}
	return t.toTaskInfo(info, t.resolveNames(ctx, []*task.Info{info})), nil
}

// CancelTasks 取消若干运行中的任务，取消人取当前登录用户。
func (t *TaskUsecase) CancelTasks(ctx context.Context, ids []string) *v1.TaskBatchResponse {
	by := ""
	if uid := auth.GetUserIDFromContext(ctx); uid != nil {
		by = *uid
	}
	out := &v1.TaskBatchResponse{}
	for _, id := range ids {
		if t.manager.Cancel(id, by) {
			out.Succeeded++
		} else {
			out.Failures = append(out.Failures, &v1.TaskBatchFailure{Id: id, Message: "任务不在运行中"})
		}
	}
	return out
}

// RetryTasks 重投若干已结束的任务。
func (t *TaskUsecase) RetryTasks(ctx context.Context, ids []string) *v1.TaskBatchResponse {
	out := &v1.TaskBatchResponse{}
	for _, id := range ids {
		if _, err := t.manager.Retry(ctx, id); err != nil {
			out.Failures = append(out.Failures, &v1.TaskBatchFailure{Id: id, Message: err.Error()})
			continue
		}
		out.Succeeded++
	}
	return out
}

// DeleteTasks 删除若干已结束的任务记录（运行中的跳过）。
func (t *TaskUsecase) DeleteTasks(ctx context.Context, ids []string) (*v1.TaskBatchResponse, error) {
	deleted, running, err := t.manager.Delete(ids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.TaskBatchResponse{Succeeded: deleted}
	for _, id := range running {
		out.Failures = append(out.Failures, &v1.TaskBatchFailure{Id: id, Message: task.ErrTaskRunning.Error()})
	}
	return out, nil
}

// ClearTasks 批量清理终态任务；status 为空时清理全部成功/失败/已取消任务。
func (t *TaskUsecase) ClearTasks(ctx context.Context, status string) (int64, error) {
	var statuses []task.Status
	if status != "" {
		statuses = append(statuses, task.Status(status))
	}
	n, err := t.manager.Clear(ctx, statuses...)
	if err != nil {
		return 0, ErrSystem(err)
	}
	return n, nil
}

// resolveNames 批量解析任务发起人与取消人名称（带本次调用内缓存）。
func (t *TaskUsecase) resolveNames(ctx context.Context, infos []*task.Info) map[string]string {
	names := make(map[string]string)
	add := func(id string) {
		if id == "" {
			return
		}
		if _, ok := names[id]; ok {
			return
		}
		names[id] = t.userName(ctx, id)
	}
	for _, info := range infos {
		add(info.UserID)
		add(info.CanceledBy)
	}
	return names
}

func (t *TaskUsecase) userName(ctx context.Context, userID string) string {
	if userID == "" || t.user == nil {
		return ""
	}
	u, err := t.user.FindByID(ctx, userID)
	if err != nil || u == nil {
		return ""
	}
	if u.Name != "" {
		return u.Name
	}
	return u.Username
}

func (t *TaskUsecase) toTaskInfo(info *task.Info, names map[string]string) *v1.TaskInfo {
	out := &v1.TaskInfo{
		Id:             info.ID,
		Type:           info.Type,
		Kind:           toTaskKind(info.Kind),
		Title:          info.Title,
		Target:         info.Target,
		Source:         info.Source,
		Trigger:        toTaskTrigger(info.Trigger),
		UserId:         info.UserID,
		UserName:       names[info.UserID],
		Operator:       info.Operator,
		Status:         toTaskStatus(info.Status),
		Total:          info.Total,
		Finished:       info.Finished,
		Message:        info.Message,
		Error:          info.Error,
		Results:        make([]*v1.TaskResult, 0, len(info.Results)),
		Steps:          make([]*v1.TaskStep, 0, len(info.Steps)),
		Logs:           make([]*v1.TaskLog, 0, len(info.Logs)),
		IntervalMs:     info.Interval.Milliseconds(),
		Retryable:      t.manager.CanRetry(info.Type),
		CanceledBy:     info.CanceledBy,
		CanceledByName: names[info.CanceledBy],
		CreateTime:     timestamppb.New(info.CreateTime),
		StartTime:      optTimestamp(info.StartTime),
		EndTime:        optTimestamp(info.EndTime),
		NextRunTime:    optTimestamp(info.NextRunTime),
	}
	for _, r := range info.Results {
		out.Results = append(out.Results, &v1.TaskResult{Path: r.Path, Success: r.Success, Message: r.Message})
	}
	for _, s := range info.Steps {
		out.Steps = append(out.Steps, &v1.TaskStep{
			Title:     s.Title,
			State:     toTaskStepState(s.State),
			Detail:    s.Detail,
			StartTime: optTimestamp(s.StartTime),
			EndTime:   optTimestamp(s.EndTime),
		})
	}
	for _, l := range info.Logs {
		out.Logs = append(out.Logs, &v1.TaskLog{Time: timestamppb.New(l.Time), Level: toTaskLogLevel(l.Level), Message: l.Message})
	}
	return out
}

func optTimestamp(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func toTaskStatus(s task.Status) v1.TaskStatus {
	switch s {
	case task.StatusPending:
		return v1.TaskStatus_TASK_STATUS_PENDING
	case task.StatusRunning:
		return v1.TaskStatus_TASK_STATUS_RUNNING
	case task.StatusSuccess:
		return v1.TaskStatus_TASK_STATUS_SUCCESS
	case task.StatusFailed:
		return v1.TaskStatus_TASK_STATUS_FAILED
	case task.StatusCanceled:
		return v1.TaskStatus_TASK_STATUS_CANCELED
	default:
		return v1.TaskStatus_TASK_STATUS_UNKNOWN
	}
}

func toTaskKind(k task.Kind) v1.TaskKind {
	switch k {
	case task.KindOneShot:
		return v1.TaskKind_TASK_KIND_ONESHOT
	case task.KindScheduled:
		return v1.TaskKind_TASK_KIND_SCHEDULED
	case task.KindDaemon:
		return v1.TaskKind_TASK_KIND_DAEMON
	default:
		return v1.TaskKind_TASK_KIND_UNKNOWN
	}
}

func toTaskTrigger(t task.Trigger) v1.TaskTrigger {
	switch t {
	case task.TriggerManual:
		return v1.TaskTrigger_TASK_TRIGGER_MANUAL
	case task.TriggerSchedule:
		return v1.TaskTrigger_TASK_TRIGGER_SCHEDULE
	case task.TriggerMaster:
		return v1.TaskTrigger_TASK_TRIGGER_MASTER
	case task.TriggerSystem:
		return v1.TaskTrigger_TASK_TRIGGER_SYSTEM
	default:
		return v1.TaskTrigger_TASK_TRIGGER_UNKNOWN
	}
}

func toTaskStepState(s task.StepState) v1.TaskStepState {
	switch s {
	case task.StepWaiting:
		return v1.TaskStepState_TASK_STEP_STATE_WAITING
	case task.StepRunning:
		return v1.TaskStepState_TASK_STEP_STATE_RUNNING
	case task.StepSuccess:
		return v1.TaskStepState_TASK_STEP_STATE_SUCCESS
	case task.StepFailed:
		return v1.TaskStepState_TASK_STEP_STATE_FAILED
	case task.StepSkipped:
		return v1.TaskStepState_TASK_STEP_STATE_SKIPPED
	default:
		return v1.TaskStepState_TASK_STEP_STATE_UNKNOWN
	}
}

func toTaskLogLevel(l task.LogLevel) v1.TaskLogLevel {
	switch l {
	case task.LogSuccess:
		return v1.TaskLogLevel_TASK_LOG_LEVEL_SUCCESS
	case task.LogWarn:
		return v1.TaskLogLevel_TASK_LOG_LEVEL_WARN
	case task.LogError:
		return v1.TaskLogLevel_TASK_LOG_LEVEL_ERROR
	default:
		return v1.TaskLogLevel_TASK_LOG_LEVEL_INFO
	}
}
