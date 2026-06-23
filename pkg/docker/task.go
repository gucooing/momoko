package docker

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

const (
	TaskWSPath = "/api/v1/docker/task/ws"

	taskSubscriberBuffer = 4096
)

type taskRunner struct {
	mu    sync.RWMutex
	tasks map[string]*taskState
}

type taskState struct {
	task   *v1.DockerTaskInfo
	ctx    context.Context
	cancel context.CancelFunc
	events []*v1.DockerTaskInfo
	subs   map[uint64]chan *v1.DockerTaskInfo
	nextID uint64
}

func newTaskRunner() *taskRunner {
	return &taskRunner{
		tasks: make(map[string]*taskState),
	}
}

func (r *taskRunner) Start(
	parent context.Context,
	taskType v1.DockerTaskType,
	title string,
	timeout time.Duration,
	fn func(context.Context, func(*v1.DockerTaskInfo)) (string, error),
) *v1.DockerTaskInfo {
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(context.WithoutCancel(parent), timeout)
	id := uuid.NewString()
	task := &v1.DockerTaskInfo{
		Id:        id,
		Type:      taskType,
		Title:     title,
		Status:    v1.DockerTaskStatus_DOCKER_TASK_STATUS_PENDING,
		StartTime: timestamppb.Now(),
		WsPath:    TaskWSPath,
	}
	state := &taskState{
		task:   cloneTaskInfo(task),
		ctx:    ctx,
		cancel: cancel,
		events: []*v1.DockerTaskInfo{},
		subs:   make(map[uint64]chan *v1.DockerTaskInfo),
	}

	r.mu.Lock()
	r.tasks[id] = state
	r.mu.Unlock()

	go func() {
		defer cancel()
		r.setStatus(id, v1.DockerTaskStatus_DOCKER_TASK_STATUS_RUNNING, "", "")
		resultPath, err := fn(ctx, func(event *v1.DockerTaskInfo) {
			r.addEvent(id, event)
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				r.setStatus(id, v1.DockerTaskStatus_DOCKER_TASK_STATUS_CANCELED, "", err.Error())
				return
			}
			r.setStatus(id, v1.DockerTaskStatus_DOCKER_TASK_STATUS_FAILED, "", err.Error())
			return
		}
		r.setResult(id, resultPath)
		r.setStatus(id, v1.DockerTaskStatus_DOCKER_TASK_STATUS_SUCCESS, "完成", "")
	}()

	return cloneTaskInfo(task)
}

func (r *taskRunner) Get(id string) (*v1.DockerTaskInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.tasks[id]
	if !ok {
		return nil, false
	}
	return cloneTaskInfo(state.task), true
}

func (r *taskRunner) List() []*v1.DockerTaskInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tasks := make([]*v1.DockerTaskInfo, 0, len(r.tasks))
	for _, state := range r.tasks {
		tasks = append(tasks, cloneTaskInfo(state.task))
	}
	return tasks
}

func (r *taskRunner) Cancel(id string) bool {
	r.mu.RLock()
	state, ok := r.tasks[id]
	r.mu.RUnlock()
	if !ok {
		return false
	}
	state.cancel()
	return true
}

func (r *taskRunner) Subscribe(id string) (<-chan *v1.DockerTaskInfo, func(), bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.tasks[id]
	if !ok {
		return nil, nil, false
	}
	events := cloneTaskInfos(state.events)
	ch := make(chan *v1.DockerTaskInfo, len(events)+taskSubscriberBuffer)
	for _, event := range events {
		ch <- event
	}
	state.nextID++
	subID := state.nextID
	state.subs[subID] = ch
	cancel := func() {
		r.mu.Lock()
		defer r.mu.Unlock()
		if state, ok := r.tasks[id]; ok {
			delete(state.subs, subID)
			close(ch)
		}
	}
	return ch, cancel, true
}

func (r *taskRunner) setResult(id, resultPath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.tasks[id]
	if !ok {
		return
	}
	state.task.ResultPath = resultPath
}

func (r *taskRunner) setStatus(id string, status v1.DockerTaskStatus, message, errText string) {
	now := timestamppb.Now()
	var closeSubs []chan *v1.DockerTaskInfo

	r.mu.Lock()
	state, ok := r.tasks[id]
	if !ok {
		r.mu.Unlock()
		return
	}
	state.task.Status = status
	if message != "" {
		state.task.Message = message
	}
	if errText != "" {
		state.task.Error = errText
	}
	if isTerminalTaskStatus(status) {
		state.task.EndTime = now
	}
	if isTerminalTaskStatus(status) {
		for _, ch := range state.subs {
			closeSubs = append(closeSubs, ch)
		}
		delete(r.tasks, id)
	}
	r.mu.Unlock()

	for _, ch := range closeSubs {
		close(ch)
	}
}

func (r *taskRunner) addEvent(id string, event *v1.DockerTaskInfo) {
	if event == nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.tasks[id]
	if !ok {
		return
	}
	if event.Message != "" {
		state.task.Message = event.Message
	}
	if event.Progress != "" {
		state.task.Progress = event.Progress
	}
	if event.Error != "" {
		state.task.Error = event.Error
	}
	if event.Status != v1.DockerTaskStatus_DOCKER_TASK_STATUS_UNKNOWN {
		state.task.Status = event.Status
	}
	cloned := cloneTaskInfo(event)
	state.events = append(state.events, cloned)
	notifySubscribers(state, cloned)
}

func notifySubscribers(state *taskState, event *v1.DockerTaskInfo) {
	for _, ch := range state.subs {
		select {
		case ch <- cloneTaskInfo(event):
		default:
		}
	}
}

func isTerminalTaskStatus(status v1.DockerTaskStatus) bool {
	return status == v1.DockerTaskStatus_DOCKER_TASK_STATUS_SUCCESS ||
		status == v1.DockerTaskStatus_DOCKER_TASK_STATUS_FAILED ||
		status == v1.DockerTaskStatus_DOCKER_TASK_STATUS_CANCELED
}

func cloneTaskInfo(task *v1.DockerTaskInfo) *v1.DockerTaskInfo {
	if task == nil {
		return nil
	}
	return proto.Clone(task).(*v1.DockerTaskInfo)
}

func cloneTaskInfos(tasks []*v1.DockerTaskInfo) []*v1.DockerTaskInfo {
	result := make([]*v1.DockerTaskInfo, 0, len(tasks))
	for _, task := range tasks {
		result = append(result, cloneTaskInfo(task))
	}
	return result
}
