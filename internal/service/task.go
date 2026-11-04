package service

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
)

// TaskService 暴露通用任务管理器的统一面板接口（鉴权由 /v1.TaskManager/ 前缀统一要求 task:manage）。
type TaskService struct {
	v1.UnimplementedTaskManagerServer

	uc *biz.TaskUsecase
}

func NewTaskService(uc *biz.TaskUsecase) *TaskService {
	return &TaskService{uc: uc}
}

func (s *TaskService) ListTasks(ctx context.Context, req *v1.ListTasksRequest) (*v1.ListTasksResponse, error) {
	return s.uc.ListTasks(ctx, req)
}

func (s *TaskService) GetTask(ctx context.Context, req *v1.GetTaskRequest) (*v1.GetTaskResponse, error) {
	info, err := s.uc.GetTask(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.GetTaskResponse{Task: info}, nil
}

func (s *TaskService) CancelTasks(ctx context.Context, req *v1.TaskIdsRequest) (*v1.TaskBatchResponse, error) {
	return s.uc.CancelTasks(ctx, req.GetIds()), nil
}

func (s *TaskService) RetryTasks(ctx context.Context, req *v1.TaskIdsRequest) (*v1.TaskBatchResponse, error) {
	return s.uc.RetryTasks(ctx, req.GetIds()), nil
}

func (s *TaskService) DeleteTasks(ctx context.Context, req *v1.TaskIdsRequest) (*v1.TaskBatchResponse, error) {
	return s.uc.DeleteTasks(ctx, req.GetIds())
}

func (s *TaskService) ClearTasks(ctx context.Context, req *v1.ClearTasksRequest) (*v1.ClearTasksResponse, error) {
	n, err := s.uc.ClearTasks(ctx, req.GetStatus())
	if err != nil {
		return nil, err
	}
	return &v1.ClearTasksResponse{Deleted: n}, nil
}
