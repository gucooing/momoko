package docker

import (
	"context"
	"errors"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
)

type TaskStatus string

const (
	TaskStatusPending  TaskStatus = "pending"
	TaskStatusRunning  TaskStatus = "running"
	TaskStatusSuccess  TaskStatus = "success"
	TaskStatusFailed   TaskStatus = "failed"
	TaskStatusCanceled TaskStatus = "canceled"
)

type TaskEvent struct {
	Time     time.Time
	Status   string
	Progress string
	ID       string
	Message  string
	Error    string
}

type Task struct {
	ID         string
	Type       string
	Status     TaskStatus
	Progress   string
	Message    string
	Error      string
	ResultPath string
	StartTime  time.Time
	EndTime    *time.Time
	Events     []TaskEvent
}

type taskRunner struct {
	mu    sync.RWMutex
	tasks map[string]*taskState
}

type taskState struct {
	task   *Task
	ctx    context.Context
	cancel context.CancelFunc
	subs   map[uint64]chan TaskEvent
	nextID uint64
}

func newTaskRunner() *taskRunner {
	return &taskRunner{tasks: make(map[string]*taskState)}
}

func (r *taskRunner) Start(parent context.Context, typ string, timeout time.Duration, fn func(context.Context, func(TaskEvent)) (string, error)) *Task {
	if timeout <= 0 {
		timeout = 30 * time.Minute
	}
	ctx, cancel := context.WithTimeout(parent, timeout)
	id := uuid.NewString()
	task := &Task{
		ID:        id,
		Type:      typ,
		Status:    TaskStatusPending,
		StartTime: time.Now(),
		Events:    []TaskEvent{},
	}
	state := &taskState{
		task:   cloneTask(task),
		ctx:    ctx,
		cancel: cancel,
		subs:   make(map[uint64]chan TaskEvent),
	}

	r.mu.Lock()
	r.tasks[id] = state
	r.mu.Unlock()

	go func() {
		defer cancel()
		r.setStatus(id, TaskStatusRunning, "", "")
		resultPath, err := fn(ctx, func(event TaskEvent) {
			r.addEvent(id, normalizeTaskEvent(event))
		})
		if err != nil {
			if errors.Is(err, context.Canceled) {
				r.setStatus(id, TaskStatusCanceled, "", err.Error())
				return
			}
			r.setStatus(id, TaskStatusFailed, "", err.Error())
			return
		}
		r.setResult(id, resultPath)
		r.setStatus(id, TaskStatusSuccess, "完成", "")
	}()

	return task
}

func (r *taskRunner) Get(id string) (*Task, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	state, ok := r.tasks[id]
	if !ok {
		return nil, false
	}
	return cloneTask(state.task), true
}

func (r *taskRunner) List() []*Task {
	r.mu.RLock()
	defer r.mu.RUnlock()
	tasks := make([]*Task, 0, len(r.tasks))
	for _, state := range r.tasks {
		tasks = append(tasks, cloneTask(state.task))
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

func (r *taskRunner) Subscribe(id string) (<-chan TaskEvent, func(), bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.tasks[id]
	if !ok {
		return nil, nil, false
	}
	ch := make(chan TaskEvent, 64)
	state.nextID++
	subID := state.nextID
	state.subs[subID] = ch
	for _, event := range state.task.Events {
		ch <- event
	}
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

func (r *taskRunner) setStatus(id string, status TaskStatus, message, errText string) {
	now := time.Now()
	event := TaskEvent{
		Time:    now,
		Status:  string(status),
		Message: message,
		Error:   errText,
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	state, ok := r.tasks[id]
	if !ok {
		return
	}
	state.task.Status = status
	if message != "" {
		state.task.Message = message
	}
	if errText != "" {
		state.task.Error = errText
	}
	if status == TaskStatusSuccess || status == TaskStatusFailed || status == TaskStatusCanceled {
		state.task.EndTime = &now
	}
	state.task.Events = append(state.task.Events, event)
	notifySubscribers(state, event)
}

func (r *taskRunner) addEvent(id string, event TaskEvent) {
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
	state.task.Events = append(state.task.Events, event)
	notifySubscribers(state, event)
}

func normalizeTaskEvent(event TaskEvent) TaskEvent {
	if event.Time.IsZero() {
		event.Time = time.Now()
	}
	return event
}

func notifySubscribers(state *taskState, event TaskEvent) {
	for _, ch := range state.subs {
		select {
		case ch <- event:
		default:
		}
	}
}

func cloneTask(task *Task) *Task {
	if task == nil {
		return nil
	}
	cloned := *task
	cloned.Events = append([]TaskEvent(nil), task.Events...)
	return &cloned
}

func streamEvents(ctx context.Context, r io.Reader, emit func(TaskEvent)) error {
	buf := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return err
		}
		n, err := r.Read(buf)
		if n > 0 {
			emit(TaskEvent{Message: string(buf[:n])})
		}
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
	}
}
