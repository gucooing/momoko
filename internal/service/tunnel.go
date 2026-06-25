package service

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
)

type TunnelService struct {
	v1.UnimplementedTunnelManagerServer

	uc *biz.TunnelUsecase
}

func NewTunnelService(uc *biz.TunnelUsecase) *TunnelService {
	return &TunnelService{uc: uc}
}

func (t *TunnelService) ListTunnels(ctx context.Context, req *v1.ListTunnelsRequest) (*v1.ListTunnelsResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return t.uc.ListTunnels(ctx, authCtx.UserID, req)
}

func (t *TunnelService) CreateTunnel(ctx context.Context, req *v1.CreateTunnelRequest) (*v1.CreateTunnelResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := t.uc.CreateTunnel(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateTunnelResponse{Info: info}, nil
}

func (t *TunnelService) GetTunnel(ctx context.Context, req *v1.GetTunnelRequest) (*v1.GetTunnelResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := t.uc.GetTunnel(ctx, authCtx.UserID, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetTunnelResponse{Info: info}, nil
}

func (t *TunnelService) UpdateTunnel(ctx context.Context, req *v1.UpdateTunnelRequest) (*v1.UpdateTunnelResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := t.uc.UpdateTunnel(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateTunnelResponse{Info: info}, nil
}

func (t *TunnelService) DeleteTunnel(ctx context.Context, req *v1.DeleteTunnelRequest) (*v1.DeleteTunnelResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	if err := t.uc.DeleteTunnel(ctx, authCtx.UserID, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteTunnelResponse{}, nil
}

func (t *TunnelService) GetTunnelStats(ctx context.Context, req *v1.GetTunnelStatsRequest) (*v1.GetTunnelStatsResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return t.uc.GetTunnelStats(ctx, authCtx.UserID, req.Id, req.StartTimeMs, req.EndTimeMs)
}

func (t *TunnelService) GetFrpsConfig(ctx context.Context, _ *v1.GetFrpsConfigRequest) (*v1.GetFrpsConfigResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, biz.ErrTokenInvalid
	}
	cfg, err := t.uc.GetFrpsConfig(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetFrpsConfigResponse{Config: cfg}, nil
}

func (t *TunnelService) UpdateFrpsConfig(ctx context.Context, req *v1.UpdateFrpsConfigRequest) (*v1.UpdateFrpsConfigResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, biz.ErrTokenInvalid
	}
	cfg, err := t.uc.UpdateFrpsConfig(ctx, req.Config)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateFrpsConfigResponse{Config: cfg}, nil
}
