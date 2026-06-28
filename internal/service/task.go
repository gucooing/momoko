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
	info, err := s.uc.GetTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetTaskResponse{Task: info}, nil
}

func (s *TaskService) CancelTask(ctx context.Context, req *v1.CancelTaskRequest) (*v1.CancelTaskResponse, error) {
	if err := s.uc.CancelTask(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.CancelTaskResponse{}, nil
}

func (s *TaskService) RetryTask(ctx context.Context, req *v1.RetryTaskRequest) (*v1.RetryTaskResponse, error) {
	info, err := s.uc.RetryTask(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.RetryTaskResponse{Task: info}, nil
}

func (s *TaskService) DeleteTask(ctx context.Context, req *v1.DeleteTaskRequest) (*v1.DeleteTaskResponse, error) {
	if err := s.uc.DeleteTask(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteTaskResponse{}, nil
}
