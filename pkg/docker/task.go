package docker

import (
	"context"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/task"
)

const (
	TaskWSPath = "/api/v1/docker/task/ws"

	wsSubscriberBuffer = 256
)

// Docker 任务类型键（注册到通用任务管理器，并在统一任务面板可见）。
const (
	taskTypeImagePull         = "docker.image_pull"
	taskTypeContainerRecreate = "docker.container_recreate"
	taskTypeNetworkRecreate   = "docker.network_recreate"
)

// dockerTasks 把 Docker 异步任务托管到通用任务管理器（pkg/task）：生命周期/入库/订阅/开机标记
// 全部走通用管理器，仅保留 DockerTaskInfo 形态与 WS 流式语义不变（快照存 Docker 特有字段）。
type dockerTasks struct {
	mgr   *task.Manager
	mu    sync.RWMutex
	snaps map[string]*v1.DockerTaskInfo
}

func newDockerTasks(mgr *task.Manager) *dockerTasks {
	return &dockerTasks{mgr: mgr, snaps: make(map[string]*v1.DockerTaskInfo)}
}

// dockerTask 实现 task.Task：Run 执行 Docker 操作 fn，并把 DockerTaskInfo 事件翻译为通用事件。
type dockerTask struct {
	meta     task.Meta
	fn       func(context.Context, func(*v1.DockerTaskInfo)) (string, error)
	onEmit   func(*v1.DockerTaskInfo)
	onResult func(resultPath string)
}

func (t *dockerTask) Meta() task.Meta { return t.meta }
func (t *dockerTask) Payload() any    { return nil } // Docker 任务不可重建（ResumeNone：重启即标失败）。
func (t *dockerTask) Run(ctx context.Context, r task.Reporter) error {
	emit := func(info *v1.DockerTaskInfo) {
		if info == nil {
			return
		}
		t.onEmit(info)
		r.Emit(task.Event{Message: info.Message, Progress: info.Progress, Error: info.Error})
	}
	resultPath, err := t.fn(ctx, emit)
	if err != nil {
		return err
	}
	t.onResult(resultPath)
	return nil
}

// Start 启动一个 Docker 异步任务，返回 DockerTaskInfo（形态与历史一致）。
func (a *dockerTasks) Start(
	parent context.Context,
	taskType v1.DockerTaskType,
	title string,
	timeout time.Duration,
	fn func(context.Context, func(*v1.DockerTaskInfo)) (string, error),
) *v1.DockerTaskInfo {
	id := uuid.NewString()
	snap := &v1.DockerTaskInfo{
		Id:        id,
		Type:      taskType,
		Title:     title,
		Status:    v1.DockerTaskStatus_DOCKER_TASK_STATUS_PENDING,
		StartTime: timestamppb.Now(),
		WsPath:    TaskWSPath,
	}
	a.mu.Lock()
	a.snaps[id] = snap
	a.mu.Unlock()

	t := &dockerTask{
		meta: task.Meta{
			ID:      id,
			Type:    dockerTypeKey(taskType),
			Kind:    task.KindOneShot,
			Title:   title,
			Timeout: timeout,
			Resume:  task.ResumeNone,
		},
		fn:       fn,
		onEmit:   func(info *v1.DockerTaskInfo) { a.updateSnap(id, info) },
		onResult: func(rp string) { a.setSnapResult(id, rp) },
	}
	// 通用管理器内部生成 detached（可选超时）ctx 执行，脱离 HTTP 请求生命周期。
	_, _ = a.mgr.Submit(parent, t)
	return cloneTaskInfo(snap)
}

func (a *dockerTasks) updateSnap(id string, info *v1.DockerTaskInfo) {
	a.mu.Lock()
	defer a.mu.Unlock()
	snap, ok := a.snaps[id]
	if !ok {
		return
	}
	if info.Progress != "" {
		snap.Progress = info.Progress
	}
	if info.Message != "" {
		snap.Message = info.Message
	}
}

func (a *dockerTasks) setSnapResult(id, resultPath string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if snap, ok := a.snaps[id]; ok {
		snap.ResultPath = resultPath
	}
}

// merged 合并 Docker 特有快照 + 通用管理器的权威状态（状态/消息/错误/结束时间）。
func (a *dockerTasks) merged(id string) (*v1.DockerTaskInfo, bool) {
	a.mu.RLock()
	snap, ok := a.snaps[id]
	if ok {
		snap = cloneTaskInfo(snap)
	}
	a.mu.RUnlock()
	info, found := a.mgr.Get(id)
	if !ok && !found {
		return nil, false
	}
	if snap == nil {
		snap = &v1.DockerTaskInfo{Id: id, WsPath: TaskWSPath}
	}
	if found {
		snap.Status = taskToDockerStatus(info.Status)
		if info.Message != "" {
			snap.Message = info.Message
		}
		if info.Error != "" {
			snap.Error = info.Error
		}
		if info.EndTime != nil {
			snap.EndTime = timestamppb.New(*info.EndTime)
		}
	}
	return snap, true
}

func (a *dockerTasks) Get(id string) (*v1.DockerTaskInfo, bool) {
	return a.merged(id)
}

func (a *dockerTasks) List() []*v1.DockerTaskInfo {
	a.mu.RLock()
	ids := make([]string, 0, len(a.snaps))
	for id := range a.snaps {
		ids = append(ids, id)
	}
	a.mu.RUnlock()
	tasks := make([]*v1.DockerTaskInfo, 0, len(ids))
	for _, id := range ids {
		if info, ok := a.merged(id); ok {
			tasks = append(tasks, info)
		}
	}
	return tasks
}

func (a *dockerTasks) Cancel(id string) bool {
	return a.mgr.Cancel(id)
}

// Subscribe 订阅任务实时事件：复用通用管理器的回放+广播语义，把通用事件翻译回 DockerTaskInfo。
func (a *dockerTasks) Subscribe(id string) (<-chan *v1.DockerTaskInfo, func(), bool) {
	evCh, cancel, ok := a.mgr.Subscribe(id)
	if !ok {
		return nil, nil, false
	}
	out := make(chan *v1.DockerTaskInfo, wsSubscriberBuffer)
	go func() {
		defer close(out)
		for ev := range evCh {
			out <- &v1.DockerTaskInfo{
				Id:       id,
				Message:  ev.Message,
				Progress: ev.Progress,
				Error:    ev.Error,
			}
		}
	}()
	return out, cancel, true
}

func dockerTypeKey(t v1.DockerTaskType) string {
	switch t {
	case v1.DockerTaskType_DOCKER_TASK_TYPE_IMAGE_PULL:
		return taskTypeImagePull
	case v1.DockerTaskType_DOCKER_TASK_TYPE_CONTAINER_RECREATE:
		return taskTypeContainerRecreate
	case v1.DockerTaskType_DOCKER_TASK_TYPE_NETWORK_RECREATE:
		return taskTypeNetworkRecreate
	default:
		return "docker.task"
	}
}

// DockerTaskTypes 返回全部 Docker 任务类型键（供业务层开机标记中断任务为失败）。
func DockerTaskTypes() []string {
	return []string{taskTypeImagePull, taskTypeContainerRecreate, taskTypeNetworkRecreate}
}

func taskToDockerStatus(s task.Status) v1.DockerTaskStatus {
	switch s {
	case task.StatusPending:
		return v1.DockerTaskStatus_DOCKER_TASK_STATUS_PENDING
	case task.StatusRunning:
		return v1.DockerTaskStatus_DOCKER_TASK_STATUS_RUNNING
	case task.StatusSuccess:
		return v1.DockerTaskStatus_DOCKER_TASK_STATUS_SUCCESS
	case task.StatusCanceled:
		return v1.DockerTaskStatus_DOCKER_TASK_STATUS_CANCELED
	default:
		return v1.DockerTaskStatus_DOCKER_TASK_STATUS_FAILED
	}
}

func cloneTaskInfo(task *v1.DockerTaskInfo) *v1.DockerTaskInfo {
	if task == nil {
		return nil
	}
	return proto.Clone(task).(*v1.DockerTaskInfo)
}
