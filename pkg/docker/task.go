package docker

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/docker/go-units"
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
	taskTypeContainerCreate   = "docker.container_create"
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
	// 重建类任务每条阶段消息就是一个执行步骤；拉取（含创建容器时的拉取）按分层进度推进「下载镜像层」。
	stepByMessage := t.meta.Type == taskTypeContainerRecreate || t.meta.Type == taskTypeNetworkRecreate
	if t.meta.Type == taskTypeImagePull {
		r.Step("连接镜像仓库", "")
	}
	emit := func(info *v1.DockerTaskInfo) {
		if info == nil {
			return
		}
		t.onEmit(info)
		if p := info.GetPull(); p != nil {
			detail := fmt.Sprintf("%d / %d 层 · %s / %s", p.GetLayersDone(), p.GetLayersTotal(),
				units.HumanSize(float64(p.GetCurrent())), units.HumanSize(float64(p.GetTotal())))
			r.Step("下载镜像层", detail)
			r.SetProgress(int64(p.GetPercent()), 100, detail)
		} else if msg := strings.TrimSpace(info.Message); stepByMessage && msg != "" {
			r.Step(msg, "")
		} else if t.meta.Type == taskTypeContainerCreate && msg == containerCreateStep {
			// 创建容器：先（按需）拉取镜像，再进入「创建容器」阶段
			r.Step(containerCreateStep, "")
		}
		for _, line := range taskLogLines(info.Message) {
			r.Log(task.LogInfo, line)
		}
		if info.Error != "" {
			r.Log(task.LogError, info.Error)
		}
		// 终端流原样广播给 WS 订阅者（xterm 渲染 ANSI），任务日志已在上面清洗记录。
		r.Emit(task.Event{Message: info.Message, Progress: info.Progress, Error: info.Error, Quiet: true})
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
	target string,
	timeout time.Duration,
	fn func(context.Context, func(*v1.DockerTaskInfo)) (string, error),
) *v1.DockerTaskInfo {
	id := uuid.NewString()
	snap := &v1.DockerTaskInfo{
		Id:        id,
		Type:      taskType,
		Title:     title,
		Target:    target,
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
			Target:  target,
			Source:  dockerTaskSource(taskType),
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
	if info.Pull != nil {
		snap.Pull = info.Pull
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

// Cancel 取消任务；by 为取消人（用户 ID，空表示系统）。
func (a *dockerTasks) Cancel(id, by string) bool {
	return a.mgr.Cancel(id, by)
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
	case v1.DockerTaskType_DOCKER_TASK_TYPE_CONTAINER_CREATE:
		return taskTypeContainerCreate
	default:
		return "docker.task"
	}
}

// containerCreateStep 是创建容器任务进入创建阶段时发出的消息（同时作为任务面板里的执行步骤）。
const containerCreateStep = "创建容器"

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;?]*[A-Za-z]`)

// taskLogLines 把终端流输出清洗成任务日志行：去 ANSI 控制符、按行切分、取回车后的最终内容、丢掉进度条刷新行。
func taskLogLines(raw string) []string {
	if raw == "" {
		return nil
	}
	raw = ansiPattern.ReplaceAllString(raw, "")
	var out []string
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimRight(line, "\r")
		if i := strings.LastIndexByte(line, '\r'); i >= 0 {
			line = line[i+1:]
		}
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "[=") || strings.Contains(line, "[>") {
			continue
		}
		out = append(out, line)
	}
	return out
}

// dockerTaskSource 返回任务的发起入口（任务面板「触发方式」里的页面）。
func dockerTaskSource(t v1.DockerTaskType) string {
	switch t {
	case v1.DockerTaskType_DOCKER_TASK_TYPE_IMAGE_PULL:
		return "docker.image"
	case v1.DockerTaskType_DOCKER_TASK_TYPE_NETWORK_RECREATE:
		return "docker.network"
	default:
		return "docker.container"
	}
}

// DockerTaskTypes 返回全部 Docker 任务类型键（供业务层开机标记中断任务为失败）。
func DockerTaskTypes() []string {
	return []string{taskTypeImagePull, taskTypeContainerRecreate, taskTypeNetworkRecreate, taskTypeContainerCreate}
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
