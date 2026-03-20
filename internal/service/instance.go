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
