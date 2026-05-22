package file

import (
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

const (
	TaskOperationSystemCopy   = "system_copy"
	TaskOperationSystemCut    = "system_cut"
	TaskOperationInstanceCopy = "instance_copy"
	TaskOperationInstanceCut  = "instance_cut"
)

var tasks = newTaskManager()

type taskManager struct {
	mu    sync.RWMutex
	tasks map[string]*task
}

type task struct {
	id        string
	userID    string
	operation string
	status    v1.FileTaskStatus
	total     int64
	finished  int64
	message   string
	items     []*v1.FileOperationResult
	createdAt time.Time
	updatedAt time.Time
}

func newTaskManager() *taskManager {
	return &taskManager{
		tasks: make(map[string]*task),
	}
}

func (f *FileOper) StartCopyToDirTask(
	userID string,
	paths []string,
	targetPath string,
	operation string,
) (*v1.FileTaskInfo, error) {
	if err := validateTransferRequest(paths); err != nil {
		return nil, err
	}
	return startTransferTask(
		userID,
		operation,
		paths,
		func(path string) *v1.FileOperationResult {
			return firstOperationResult(path, f.CopyToDir([]string{path}, targetPath))
		},
	), nil
}

func (f *FileOper) StartMoveToDirTask(
	userID string,
	paths []string,
	targetPath string,
	operation string,
) (*v1.FileTaskInfo, error) {
	if err := validateTransferRequest(paths); err != nil {
		return nil, err
	}
	return startTransferTask(
		userID,
		operation,
		paths,
		func(path string) *v1.FileOperationResult {
			return firstOperationResult(path, f.MoveToDir([]string{path}, targetPath))
		},
	), nil
}

func GetTask(userID, taskID string) (*v1.FileTaskInfo, bool) {
	return tasks.get(userID, taskID)
}

func startTransferTask(
	userID string,
	operation string,
	paths []string,
	transfer func(string) *v1.FileOperationResult,
) *v1.FileTaskInfo {
	task := tasks.create(userID, operation, int64(len(paths)))
	taskPaths := append([]string(nil), paths...)

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				tasks.fail(task.TaskId, fmt.Sprintf("%v", recovered))
			}
		}()

		tasks.running(task.TaskId)
		for _, path := range taskPaths {
			tasks.appendResult(task.TaskId, transfer(path))
		}
		tasks.success(task.TaskId)
	}()

	return task
}

func (m *taskManager) create(userID, operation string, total int64) *v1.FileTaskInfo {
	now := time.Now()
	task := &task{
		id:        uuid.NewString(),
		userID:    userID,
		operation: operation,
		status:    v1.FileTaskStatus_FILE_TASK_STATUS_PENDING,
		total:     total,
		items:     make([]*v1.FileOperationResult, 0),
		createdAt: now,
		updatedAt: now,
	}

	m.mu.Lock()
	m.tasks[task.id] = task
	m.mu.Unlock()

	return task.toProto()
}

func (m *taskManager) get(userID, id string) (*v1.FileTaskInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[id]
	if !ok || task.userID != userID {
		return nil, false
	}
	return task.toProto(), true
}

func (m *taskManager) running(id string) {
	m.update(id, func(task *task) {
		task.status = v1.FileTaskStatus_FILE_TASK_STATUS_RUNNING
		task.message = "执行中"
	})
}

func (m *taskManager) appendResult(id string, result *v1.FileOperationResult) {
	m.update(id, func(task *task) {
		if result == nil {
			result = &v1.FileOperationResult{Message: "文件操作未返回结果"}
		}
		task.items = append(task.items, cloneOperationResult(result))
		task.finished = int64(len(task.items))
	})
}

func (m *taskManager) success(id string) {
	m.update(id, func(task *task) {
		task.status = v1.FileTaskStatus_FILE_TASK_STATUS_SUCCESS
		task.message = "已完成"
		task.finished = int64(len(task.items))
	})
}

func (m *taskManager) fail(id, message string) {
	m.update(id, func(task *task) {
		task.status = v1.FileTaskStatus_FILE_TASK_STATUS_FAILED
		task.message = message
	})
}

func (m *taskManager) update(id string, fn func(*task)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return
	}
	fn(task)
	task.updatedAt = time.Now()
}

func (task *task) toProto() *v1.FileTaskInfo {
	return &v1.FileTaskInfo{
		TaskId:     task.id,
		Operation:  task.operation,
		Status:     task.status,
		Total:      task.total,
		Finished:   task.finished,
		Message:    task.message,
		Items:      cloneOperationResults(task.items),
		CreateTime: timestamppb.New(task.createdAt),
		UpdateTime: timestamppb.New(task.updatedAt),
	}
}

func cloneOperationResults(items []*v1.FileOperationResult) []*v1.FileOperationResult {
	results := make([]*v1.FileOperationResult, 0, len(items))
	for _, item := range items {
		results = append(results, cloneOperationResult(item))
	}
	return results
}

func cloneOperationResult(item *v1.FileOperationResult) *v1.FileOperationResult {
	if item == nil {
		return nil
	}
	return &v1.FileOperationResult{
		Path:    item.Path,
		Success: item.Success,
		Message: item.Message,
	}
}

func firstOperationResult(path string, results []*v1.FileOperationResult) *v1.FileOperationResult {
	if len(results) == 0 || results[0] == nil {
		return &v1.FileOperationResult{
			Path:    path,
			Message: "文件操作未返回结果",
		}
	}
	return results[0]
}

func validateTransferRequest(paths []string) error {
	if len(paths) == 0 {
		return errors.New("路径列表不能为空")
	}
	return nil
}
