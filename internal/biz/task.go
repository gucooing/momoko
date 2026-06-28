package biz

import (
	"context"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/task"
)

// NewTaskManager 构造通用任务管理器并返回其停止函数（接进 wire cleanup）。
func NewTaskManager(store task.Store) (*task.Manager, func()) {
	m := task.New(store)
	return m, m.Stop
}

// TaskUsecase 是「任务管理」对外面板的薄封装：列出/查看/取消/重试/删除任务管理器中的全部任务。
type TaskUsecase struct {
	manager *task.Manager
	user    UserRepo
}

func NewTaskUsecase(manager *task.Manager, user UserRepo) *TaskUsecase {
	return &TaskUsecase{manager: manager, user: user}
}

func (t *TaskUsecase) ListTasks(ctx context.Context, req *v1.ListTasksRequest) (*v1.ListTasksResponse, error) {
	filter := task.Filter{
		Status:   task.Status(req.GetStatus()),
		Kind:     task.Kind(req.GetKind()),
		Keywords: req.GetKeywords(),
	}
	if req.GetType() != "" {
		filter.TypePrefix = req.GetType()
	}
	infos, total, err := t.manager.List(ctx, filter, req.GetPage(), req.GetPageSize())
	if err != nil {
		return nil, ErrSystem(err)
	}
	names := t.resolveNames(ctx, infos)
	items := make([]*v1.TaskInfo, 0, len(infos))
	for _, info := range infos {
		items = append(items, toTaskInfo(info, names[info.UserID]))
	}
	return &v1.ListTasksResponse{
		Items:    items,
		Total:    total,
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	}, nil
}

func (t *TaskUsecase) GetTask(ctx context.Context, id string) (*v1.TaskInfo, error) {
	info, ok := t.manager.Get(id)
	if !ok {
		return nil, ErrTaskNotFound
	}
	return toTaskInfo(info, t.userName(ctx, info.UserID)), nil
}

func (t *TaskUsecase) CancelTask(ctx context.Context, id string) error {
	if !t.manager.Cancel(id) {
		return ErrTaskNotFound
	}
	return nil
}

func (t *TaskUsecase) RetryTask(ctx context.Context, id string) (*v1.TaskInfo, error) {
	info, err := t.manager.Retry(ctx, id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toTaskInfo(info, t.userName(ctx, info.UserID)), nil
}

func (t *TaskUsecase) DeleteTask(ctx context.Context, id string) error {
	if err := t.manager.Delete(id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// resolveNames 批量解析任务发起人名称（带本次调用内缓存）。
func (t *TaskUsecase) resolveNames(ctx context.Context, infos []*task.Info) map[string]string {
	names := make(map[string]string)
	for _, info := range infos {
		if info.UserID == "" {
			continue
		}
		if _, ok := names[info.UserID]; ok {
			continue
		}
		names[info.UserID] = t.userName(ctx, info.UserID)
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

func toTaskInfo(info *task.Info, userName string) *v1.TaskInfo {
	out := &v1.TaskInfo{
		Id:         info.ID,
		Type:       info.Type,
		Kind:       toTaskKind(info.Kind),
		Title:      info.Title,
		UserId:     info.UserID,
		UserName:   userName,
		Status:     toTaskStatus(info.Status),
		Total:      info.Total,
		Finished:   info.Finished,
		Message:    info.Message,
		Error:      info.Error,
		Results:    make([]*v1.TaskResult, 0, len(info.Results)),
		CreateTime: timestamppb.New(info.CreateTime),
	}
	for _, r := range info.Results {
		out.Results = append(out.Results, &v1.TaskResult{Path: r.Path, Success: r.Success, Message: r.Message})
	}
	if info.EndTime != nil {
		out.EndTime = timestamppb.New(*info.EndTime)
	}
	return out
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
