package service

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
)

// MonitorService 持久性能监控（查看需 monitor:view，修改设置 / 清空历史需 monitor:edit，鉴权在中间件完成）。
type MonitorService struct {
	v1.UnimplementedMonitorManagerServer

	uc *biz.MonitorUsecase
}

func NewMonitorService(uc *biz.MonitorUsecase) *MonitorService {
	return &MonitorService{uc: uc}
}

func (s *MonitorService) GetMonitorData(ctx context.Context, req *v1.GetMonitorDataRequest) (*v1.GetMonitorDataResponse, error) {
	return s.uc.GetData(ctx, req)
}

func (s *MonitorService) GetMonitorSettings(ctx context.Context, _ *v1.GetMonitorSettingsRequest) (*v1.GetMonitorSettingsResponse, error) {
	return s.uc.GetSettings(ctx)
}

func (s *MonitorService) UpdateMonitorSettings(ctx context.Context, req *v1.UpdateMonitorSettingsRequest) (*v1.UpdateMonitorSettingsResponse, error) {
	return s.uc.UpdateSettings(ctx, req)
}

func (s *MonitorService) ExportMonitorData(ctx context.Context, req *v1.ExportMonitorDataRequest) (*v1.ExportMonitorDataResponse, error) {
	return s.uc.Export(ctx, req)
}

func (s *MonitorService) ClearMonitorHistory(ctx context.Context, _ *v1.ClearMonitorHistoryRequest) (*v1.ClearMonitorHistoryResponse, error) {
	return s.uc.ClearHistory(ctx)
}
