package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/http"
	"golang.org/x/net/websocket"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
)

type OpenSSHService struct {
	v1.UnimplementedOpenSSHManagerServer

	uc *biz.OpenSSHUsecase
}

func NewOpenSSHService(uc *biz.OpenSSHUsecase) *OpenSSHService {
	return &OpenSSHService{uc: uc}
}

func (o *OpenSSHService) RegisterWsServer(srv *http.Server) {
	srv.Handle(biz.OpenSSHWSPath, websocket.Handler(o.RunSSHWsConn))
}

func (o *OpenSSHService) GetSSHHosts(ctx context.Context, req *v1.GetSSHHostsRequest) (*v1.GetSSHHostsResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	infos, total, err := o.uc.GetSSHHosts(ctx, req, authCtx.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.GetSSHHostsResponse{
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		Infos:    infos,
	}, nil
}

func (o *OpenSSHService) CreateSSHHost(ctx context.Context, req *v1.CreateSSHHostRequest) (*v1.CreateSSHHostResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := o.uc.CreateSSHHost(ctx, req, authCtx.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.CreateSSHHostResponse{Info: info}, nil
}

func (o *OpenSSHService) GetSSHHostInfo(ctx context.Context, req *v1.GetSSHHostInfoRequest) (*v1.GetSSHHostInfoResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := o.uc.GetSSHHostByUserID(ctx, authCtx.UserID, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetSSHHostInfoResponse{Info: info}, nil
}

func (o *OpenSSHService) UpdateSSHHost(ctx context.Context, req *v1.UpdateSSHHostRequest) (*v1.UpdateSSHHostResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := o.uc.UpdateSSHHost(ctx, req, authCtx.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateSSHHostResponse{Info: info}, nil
}

func (o *OpenSSHService) DeleteSSHHost(ctx context.Context, req *v1.DeleteSSHHostRequest) (*v1.DeleteSSHHostResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	if err := o.uc.DeleteSSHHost(ctx, authCtx.UserID, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteSSHHostResponse{}, nil
}

func (o *OpenSSHService) ShareSSHHost(ctx context.Context, req *v1.ShareSSHHostRequest) (*v1.ShareSSHHostResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := o.uc.ShareSSHHost(ctx, req, authCtx.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.ShareSSHHostResponse{Info: info}, nil
}

func (o *OpenSSHService) TestSSHHost(ctx context.Context, req *v1.TestSSHHostRequest) (*v1.TestSSHHostResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return o.uc.TestSSHHost(ctx, authCtx.UserID, req.Id)
}

func (o *OpenSSHService) BatchTestSSHHosts(ctx context.Context, req *v1.BatchTestSSHHostsRequest) (*v1.BatchTestSSHHostsResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return o.uc.BatchTestSSHHosts(ctx, authCtx.UserID, req.Ids)
}

func (o *OpenSSHService) RunSSHWsConn(conn *websocket.Conn) {
	defer conn.Close()
	ctx := conn.Request().Context()
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		_ = websocket.Message.Send(conn, "缺少参数")
		return
	}
	hostID := conn.Request().URL.Query().Get("hostID")
	if hostID == "" {
		hostID = conn.Request().URL.Query().Get("id")
	}
	if hostID == "" {
		_ = websocket.Message.Send(conn, "缺少参数")
		return
	}
	_ = o.uc.StartSSHWebSocket(conn, authCtx.UserID, hostID)
}
