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

	uc     *biz.InstanceUsecase
	fileUC *biz.InstanceFileUsecase
}

func NewInstanceService(uc *biz.InstanceUsecase, fileUC *biz.InstanceFileUsecase) *InstanceService {
	return &InstanceService{
		uc:     uc,
		fileUC: fileUC,
	}
}

func (i *InstanceService) RegisterWsServer(srv *http.Server) {
	srv.Handle(biz.TerminalWSPath, websocket.Handler(i.RunTerminalWsConn))
	srv.Handle(biz.InstanceWsPath, websocket.Handler(i.RunInstanceWsConn))
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

func (i *InstanceService) GetInstances(ctx context.Context, req *v1.GetInstancesRequest) (*v1.GetInstancesResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	infos, total, err := i.uc.GetInstances(ctx, req, authCtx.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.GetInstancesResponse{Infos: infos, Total: total}, nil
}

func (i *InstanceService) CreateInstance(ctx context.Context, req *v1.CreateInstanceRequest) (*v1.CreateInstanceResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := i.uc.CreateInstance(ctx, req, authCtx.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.CreateInstanceResponse{Info: info}, nil
}

func (i *InstanceService) DelInstance(ctx context.Context, req *v1.DelInstanceRequest) (*v1.DelInstanceResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	if err := i.uc.DeleteInstance(ctx, authCtx.UserID, req.Id); err != nil {
		return nil, err
	}
	return &v1.DelInstanceResponse{}, nil
}

func (i *InstanceService) UpdateInstance(ctx context.Context, req *v1.UpdateInstanceRequest) (*v1.UpdateInstanceResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := i.uc.UpdateInstance(ctx, req, authCtx.UserID)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateInstanceResponse{Info: info}, nil
}

func (i *InstanceService) GetInstanceInfo(ctx context.Context, req *v1.GetInstanceInfoRequest) (*v1.GetInstanceInfoResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := i.uc.GetInstanceByUserID(ctx, authCtx.UserID, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.GetInstanceInfoResponse{Info: info}, nil
}

func (i *InstanceService) StartInstance(ctx context.Context, req *v1.StartInstanceRequest) (*v1.StartInstanceResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	info, err := i.uc.StartInstance(ctx, authCtx.UserID, req.Id)
	if err != nil {
		return nil, err
	}
	return &v1.StartInstanceResponse{Info: info}, nil
}

func (i *InstanceService) StopInstance(ctx context.Context, req *v1.StopInstanceRequest) (*v1.StopInstanceResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	if err := i.uc.StopInstance(ctx, authCtx.UserID, req.Id, req.Force); err != nil {
		return nil, err
	}
	return &v1.StopInstanceResponse{}, nil
}

func (i *InstanceService) RestartInstance(ctx context.Context, req *v1.RestartInstanceRequest) (*v1.RestartInstanceResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	if err := i.uc.RestartInstance(ctx, authCtx.UserID, req.Id, req.Force); err != nil {
		return nil, err
	}
	return &v1.RestartInstanceResponse{}, nil
}

func (i *InstanceService) DelInstanceLog(ctx context.Context, req *v1.DelInstanceLogRequest) (*v1.DelInstanceLogResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	if err := i.uc.DelInstanceLog(ctx, authCtx.UserID, req.Id); err != nil {
		return nil, err
	}
	return &v1.DelInstanceLogResponse{}, nil
}

func (i *InstanceService) GetInstanceFileList(ctx context.Context, req *v1.GetInstanceFileListRequest) (*v1.GetInstanceFileListResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return i.fileUC.GetFileList(ctx, authCtx.UserID, req)
}

func (i *InstanceService) CreateInstanceFile(ctx context.Context, req *v1.CreateInstanceFileRequest) (*v1.CreateInstanceFileResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return i.fileUC.CreateFile(ctx, authCtx.UserID, req)
}

func (i *InstanceService) BatchDeleteInstanceFile(ctx context.Context, req *v1.BatchDeleteInstanceFileRequest) (*v1.BatchDeleteInstanceFileResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return i.fileUC.BatchDeleteFile(ctx, authCtx.UserID, req)
}

func (i *InstanceService) BatchCompressInstanceFile(ctx context.Context, req *v1.BatchCompressInstanceFileRequest) (*v1.BatchCompressInstanceFileResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return i.fileUC.BatchCompressFile(ctx, authCtx.UserID, req)
}

func (i *InstanceService) UnzipInstanceFile(ctx context.Context, req *v1.UnzipInstanceFileRequest) (*v1.UnzipInstanceFileResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return i.fileUC.UnzipFile(ctx, authCtx.UserID, req)
}

func (i *InstanceService) OpenInstanceFile(ctx context.Context, req *v1.OpenInstanceFileRequest) (*v1.OpenInstanceFileResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return i.fileUC.OpenFile(ctx, authCtx.UserID, req)
}

func (i *InstanceService) InstanceFilePreSign(ctx context.Context, req *v1.InstanceFilePreSignRequest) (*v1.InstanceFilePreSignResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return i.fileUC.FilePreSign(ctx, authCtx.UserID, req)
}

func (i *InstanceService) InstanceFilePreSignUpload(ctx context.Context, req *v1.InstanceFilePreSignUploadRequest) (*v1.InstanceFilePreSignUploadResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	item, err := i.fileUC.FilePreSignUpload(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}
	return &v1.InstanceFilePreSignUploadResponse{Info: item}, nil
}

// RunTerminalWsConn 启动终端连接
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

// RunInstanceWsConn 启动应用连接
func (i *InstanceService) RunInstanceWsConn(conn *websocket.Conn) {
	defer conn.Close()
	ctx := conn.Request().Context()
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		websocket.Message.Send(conn, "缺少参数")
		return
	}
	instanceId := conn.Request().URL.Query().Get("instanceID")
	core, err := i.uc.GetInstanceCore(ctx, authCtx.UserID, instanceId)
	if err != nil {
		websocket.Message.Send(conn, err.Error())
		return
	}
	i.uc.StartInstanceWsConn(conn, core)
}
