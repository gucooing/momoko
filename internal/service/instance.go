package service

import (
	"context"
	"github.com/go-kratos/kratos/v2/transport/http"
	"golang.org/x/net/websocket"
	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
)

type InstanceService struct {
	v1.UnimplementedInstanceManagerServer

	uc *biz.InstanceUsecase
}

func NewInstanceService(uc *biz.InstanceUsecase) *InstanceService {
	return &InstanceService{
		uc: uc,
	}
}

func (i *InstanceService) RegisterWsServer(srv *http.Server) {
	srv.Handle(biz.TerminalWSPath, websocket.Handler(i.RunTerminalWsConn))
}

func (i *InstanceService) GetInstanceTypes(ctx context.Context, req *v1.GetInstanceTypesRequest) (*v1.GetInstanceTypesResponse, error) {
	types, err := i.uc.GetTypes(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.GetInstanceTypesResponse{Types: types}, nil
}

func (i *InstanceService) CreateInstanceType(ctx context.Context, req *v1.CreateInstanceTypeRequest) (*v1.CreateInstanceTypeResponse, error) {
	typeInfo, err := i.uc.CreateType(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateInstanceTypeResponse{Type: typeInfo}, nil
}

func (i *InstanceService) UpdateInstanceType(ctx context.Context, req *v1.UpdateInstanceTypeRequest) (*v1.UpdateInstanceTypeResponse, error) {
	typeInfo, err := i.uc.UpdateType(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateInstanceTypeResponse{Type: typeInfo}, nil
}

func (i *InstanceService) DelInstanceType(ctx context.Context, req *v1.DelInstanceTypeRequest) (*v1.DelInstanceTypeResponse, error) {
	if err := i.uc.DeleteType(ctx, req.Id); err != nil {
		return nil, err
	}
	return &v1.DelInstanceTypeResponse{}, nil
}

func (i *InstanceService) GetTerminalInfo(ctx context.Context, req *v1.GetTerminalInfoRequest) (*v1.GetTerminalInfoResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := i.uc.GetTerminalInfo(ctx, authCtx.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.GetTerminalInfoResponse{Info: info}, nil
}

func (i *InstanceService) StartTerminal(ctx context.Context, req *v1.StartTerminalRequest) (*v1.StartTerminalResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	if err := i.uc.StartTerminal(ctx, authCtx.UserID); err != nil {
		return nil, err
	}
	return &v1.StartTerminalResponse{}, nil
}

func (i *InstanceService) StopTerminal(ctx context.Context, req *v1.StopTerminalRequest) (*v1.StopTerminalResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	if err := i.uc.StopTerminal(ctx, authCtx.UserID); err != nil {
		return nil, err
	}
	return &v1.StopTerminalResponse{}, nil
}

func (i *InstanceService) RestartTerminal(ctx context.Context, req *v1.RestartTerminalRequest) (*v1.RestartTerminalResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	if err := i.uc.RestartTerminal(ctx, authCtx.UserID); err != nil {
		return nil, err
	}
	return &v1.RestartTerminalResponse{}, nil
}

func (i *InstanceService) RunTerminalWsConn(conn *websocket.Conn) {
	defer conn.Close()
	ctx := conn.Request().Context()
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		websocket.Message.Send(conn, "缺少参数")
		return
	}
	terminal, err := i.uc.GetTerminalServer(ctx, authCtx.UserID)
	if err != nil {
		websocket.Message.Send(conn, err.Error())
		return
	}
	i.uc.StartInstanceWsConn(conn, terminal)
}
