package biz

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

const (
	fileTaskOperationSystemCopy   = "system_copy"
	fileTaskOperationSystemCut    = "system_cut"
	fileTaskOperationInstanceCopy = "instance_copy"
	fileTaskOperationInstanceCut  = "instance_cut"
)

var fileTasks = newFileTaskManager()

type fileTaskManager struct {
	mu    sync.RWMutex
	tasks map[string]*fileTask
}

type fileTask struct {
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

func newFileTaskManager() *fileTaskManager {
	return &fileTaskManager{
		tasks: make(map[string]*fileTask),
	}
}

func startFileTransferTask(
	ctx context.Context,
	userID string,
	operation string,
	paths []string,
	transfer func(context.Context, string) *v1.FileOperationResult,
) *v1.FileTaskInfo {
	task := fileTasks.create(userID, operation, int64(len(paths)))
	taskCtx := context.WithoutCancel(ctx)
	taskPaths := append([]string(nil), paths...)

	go func() {
		defer func() {
			if recovered := recover(); recovered != nil {
				fileTasks.fail(task.TaskId, fmt.Sprintf("%v", recovered))
			}
		}()

		fileTasks.running(task.TaskId)
		for _, path := range taskPaths {
			fileTasks.appendResult(task.TaskId, transfer(taskCtx, path))
		}
		fileTasks.success(task.TaskId)
	}()

	return task
}

func (m *fileTaskManager) create(userID, operation string, total int64) *v1.FileTaskInfo {
	now := time.Now()
	task := &fileTask{
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

func (m *fileTaskManager) get(userID, id string) (*v1.FileTaskInfo, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	task, ok := m.tasks[id]
	if !ok || task.userID != userID {
		return nil, false
	}
	return task.toProto(), true
}

func (m *fileTaskManager) running(id string) {
	m.update(id, func(task *fileTask) {
		task.status = v1.FileTaskStatus_FILE_TASK_STATUS_RUNNING
		task.message = "执行中"
	})
}

func (m *fileTaskManager) appendResult(id string, result *v1.FileOperationResult) {
	m.update(id, func(task *fileTask) {
		if result == nil {
			result = &v1.FileOperationResult{Message: "文件操作未返回结果"}
		}
		task.items = append(task.items, cloneFileOperationResult(result))
		task.finished = int64(len(task.items))
	})
}

func (m *fileTaskManager) success(id string) {
	m.update(id, func(task *fileTask) {
		task.status = v1.FileTaskStatus_FILE_TASK_STATUS_SUCCESS
		task.message = "已完成"
		task.finished = int64(len(task.items))
	})
}

func (m *fileTaskManager) fail(id, message string) {
	m.update(id, func(task *fileTask) {
		task.status = v1.FileTaskStatus_FILE_TASK_STATUS_FAILED
		task.message = message
	})
}

func (m *fileTaskManager) update(id string, fn func(*fileTask)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	task, ok := m.tasks[id]
	if !ok {
		return
	}
	fn(task)
	task.updatedAt = time.Now()
}

func (task *fileTask) toProto() *v1.FileTaskInfo {
	return &v1.FileTaskInfo{
		TaskId:     task.id,
		Operation:  task.operation,
		Status:     task.status,
		Total:      task.total,
		Finished:   task.finished,
		Message:    task.message,
		Items:      cloneFileOperationResults(task.items),
		CreateTime: timestamppb.New(task.createdAt),
		UpdateTime: timestamppb.New(task.updatedAt),
	}
}

func cloneFileOperationResults(items []*v1.FileOperationResult) []*v1.FileOperationResult {
	results := make([]*v1.FileOperationResult, 0, len(items))
	for _, item := range items {
		results = append(results, cloneFileOperationResult(item))
	}
	return results
}

func cloneFileOperationResult(item *v1.FileOperationResult) *v1.FileOperationResult {
	if item == nil {
		return nil
	}
	return &v1.FileOperationResult{
		Path:    item.Path,
		Success: item.Success,
		Message: item.Message,
	}
}

func firstFileOperationResult(path string, results []*v1.FileOperationResult) *v1.FileOperationResult {
	if len(results) == 0 || results[0] == nil {
		return &v1.FileOperationResult{
			Path:    path,
			Message: "文件操作未返回结果",
		}
	}
	return results[0]
}
