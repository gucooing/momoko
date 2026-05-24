package service

import (
	"context"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/internal/initialize"
)

type InitializeService struct {
	v1.UnimplementedInitializeServer

	uc *biz.InitializeUsecase
}

func NewInitializeService(uc *biz.InitializeUsecase) *InitializeService {
	return &InitializeService{
		uc: uc,
	}
}

func (s *InitializeService) InitializeStatus(ctx context.Context, req *v1.InitializeStatusRequest) (*v1.InitializeStatusResponse, error) {
	return s.uc.Status(), nil
}

func (s *InitializeService) ConfirmInitialize(ctx context.Context, req *v1.ConfirmInitializeRequest) (*v1.ConfirmInitializeResponse, error) {
	resp, err := s.uc.Confirm(ctx, req)
	if err != nil {
		return nil, err
	}

	go func() {
		time.Sleep(5 * time.Second)
		initialize.RequestRestart()
	}()
	return resp, nil
}

func (s *InitializeService) TestInitializeDatabase(ctx context.Context, req *v1.TestInitializeDatabaseRequest) (*v1.TestInitializeDatabaseResponse, error) {
	return s.uc.TestDatabase(ctx, req)
}
