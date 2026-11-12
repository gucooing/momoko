package service

import (
	"context"
	"encoding/json"
	"fmt"
	nethttp "net/http"
	"strings"
	"sync"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"golang.org/x/net/websocket"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/constant"
	"momoko/pkg/mgp"
	"momoko/pkg/response"
)

// GameServerService 游戏服务器管理（权限 game:server）：接入、托管进程、专用设置、更新、备份、文件、日志与实时动态。
type GameServerService struct {
	v1.UnimplementedGameServerServiceServer

	uc      *biz.GameServerUsecase
	hosting *biz.GameHostingUsecase
	system  *biz.SystemUsecase
}

func NewGameServerService(uc *biz.GameServerUsecase, hosting *biz.GameHostingUsecase, system *biz.SystemUsecase) *GameServerService {
	return &GameServerService{uc: uc, hosting: hosting, system: system}
}

func (s *GameServerService) info(ctx context.Context, id string) (*v1.GameServerInfo, error) {
	srv, err := s.uc.Get(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.uc.Info(ctx, srv), nil
}

func (s *GameServerService) ListGameAdapters(ctx context.Context, _ *v1.ListGameAdaptersRequest) (*v1.ListGameAdaptersResponse, error) {
	out := &v1.ListGameAdaptersResponse{}
	for _, a := range biz.GameAdapters() {
		out.Items = append(out.Items, a.ToProto())
	}
	return out, nil
}

func (s *GameServerService) ListGameServers(ctx context.Context, req *v1.ListGameServersRequest) (*v1.ListGameServersResponse, error) {
	return s.uc.ListServers(ctx, req)
}

func (s *GameServerService) GetGameServer(ctx context.Context, req *v1.GetGameServerRequest) (*v1.GameServerInfo, error) {
	return s.info(ctx, req.Id)
}

func (s *GameServerService) CreateGameServer(ctx context.Context, req *v1.CreateGameServerRequest) (*v1.GameServerInfo, error) {
	if req.Mode == "hosted" {
		srv, err := s.hosting.CreateHosted(ctx, req)
		if err != nil {
			return nil, err
		}
		return s.uc.Info(ctx, srv), nil
	}
	srv, err := s.uc.CreateServer(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.uc.Info(ctx, srv), nil
}

func (s *GameServerService) UpdateGameServer(ctx context.Context, req *v1.UpdateGameServerRequest) (*v1.GameServerInfo, error) {
	srv, err := s.uc.UpdateServer(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.uc.Info(ctx, srv), nil
}

func (s *GameServerService) DeleteGameServer(ctx context.Context, req *v1.DeleteGameServerRequest) (*v1.DeleteGameServerResponse, error) {
	if err := s.hosting.Delete(ctx, req); err != nil {
		return nil, err
	}
	return &v1.DeleteGameServerResponse{}, nil
}

func (s *GameServerService) TestGameServer(ctx context.Context, req *v1.TestGameServerRequest) (*v1.TestGameServerResponse, error) {
	return s.uc.TestServer(ctx, req)
}

func (s *GameServerService) RefreshGameServer(ctx context.Context, req *v1.RefreshGameServerRequest) (*v1.GameServerInfo, error) {
	srv, err := s.uc.RefreshServer(ctx, req.Id)
	if err != nil {
		return nil, err
	}
	return s.uc.Info(ctx, srv), nil
}

// ---- 托管 ----

func (s *GameServerService) DetectGameServerDir(ctx context.Context, req *v1.DetectGameServerDirRequest) (*v1.DetectGameServerDirResponse, error) {
	return s.hosting.Detect(ctx, req)
}

func (s *GameServerService) SwitchGameServerMode(ctx context.Context, req *v1.SwitchGameServerModeRequest) (*v1.GameServerInfo, error) {
	srv, err := s.hosting.SwitchMode(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.uc.Info(ctx, srv), nil
}

func (s *GameServerService) RotateGameServerKey(ctx context.Context, req *v1.RotateGameServerKeyRequest) (*v1.RotateGameServerKeyResponse, error) {
	out, srv, err := s.hosting.RotateKey(ctx, req)
	if err != nil {
		return nil, err
	}
	out.Info = s.uc.Info(ctx, srv)
	return out, nil
}

func (s *GameServerService) ControlGameServer(ctx context.Context, req *v1.ControlGameServerRequest) (*v1.GameServerInfo, error) {
	srv, err := s.hosting.Control(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.uc.Info(ctx, srv), nil
}

func (s *GameServerService) UpdateGameServerHosting(ctx context.Context, req *v1.UpdateGameServerHostingRequest) (*v1.GameServerInfo, error) {
	srv, err := s.hosting.UpdateHosting(ctx, req)
	if err != nil {
		return nil, err
	}
	return s.uc.Info(ctx, srv), nil
}

func (s *GameServerService) GetGameServerOverview(ctx context.Context, req *v1.GetGameServerOverviewRequest) (*v1.GameServerOverview, error) {
	return s.hosting.Overview(ctx, req.Id)
}

// ---- 专用设置 ----

func (s *GameServerService) GetGameServerSettings(ctx context.Context, req *v1.GetGameServerSettingsRequest) (*v1.GameServerSettings, error) {
	return s.hosting.GetSettings(ctx, req.Id)
}

func (s *GameServerService) SaveGameServerSettings(ctx context.Context, req *v1.SaveGameServerSettingsRequest) (*v1.SaveGameServerSettingsResponse, error) {
	return s.hosting.SaveSettings(ctx, req)
}

// ---- 更新 ----

func (s *GameServerService) GetGameServerBuild(ctx context.Context, req *v1.GetGameServerBuildRequest) (*v1.GameServerBuild, error) {
	return s.hosting.GetBuild(ctx, req.Id)
}

func (s *GameServerService) CheckGameServerBuild(ctx context.Context, req *v1.CheckGameServerBuildRequest) (*v1.GameServerBuildInfo, error) {
	return s.hosting.CheckBuild(ctx, req)
}

func (s *GameServerService) UpdateGameServerBuild(ctx context.Context, req *v1.UpdateGameServerBuildRequest) (*v1.GameServerTaskResponse, error) {
	id, err := s.hosting.UpdateBuild(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.GameServerTaskResponse{TaskId: id}, nil
}

func (s *GameServerService) GetGameServerResource(ctx context.Context, req *v1.GetGameServerResourceRequest) (*v1.GameServerResource, error) {
	return s.hosting.GetResource(ctx, req.Id)
}

func (s *GameServerService) RunGameServerResourceTask(ctx context.Context, req *v1.RunGameServerResourceTaskRequest) (*v1.GameServerTaskResponse, error) {
	id, err := s.hosting.RunResourceTask(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.GameServerTaskResponse{TaskId: id}, nil
}

func (s *GameServerService) GetGameServerTask(ctx context.Context, req *v1.GetGameServerTaskRequest) (*v1.GameServerTask, error) {
	return s.hosting.GetTask(ctx, req.Id, req.TaskId)
}

func (s *GameServerService) CancelGameServerTask(ctx context.Context, req *v1.CancelGameServerTaskRequest) (*v1.GameServerTask, error) {
	return s.hosting.CancelTask(ctx, req.Id, req.TaskId)
}

func (s *GameServerService) ListGameServerHistory(ctx context.Context, req *v1.ListGameServerHistoryRequest) (*v1.ListGameServerHistoryResponse, error) {
	return s.hosting.ListHistory(ctx, req)
}

// ---- 备份 ----

func (s *GameServerService) ListGameServerBackups(ctx context.Context, req *v1.ListGameServerBackupsRequest) (*v1.ListGameServerBackupsResponse, error) {
	return s.hosting.ListBackups(ctx, req)
}

func (s *GameServerService) CreateGameServerBackup(ctx context.Context, req *v1.CreateGameServerBackupRequest) (*v1.GameServerTaskResponse, error) {
	id, err := s.hosting.CreateBackup(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.GameServerTaskResponse{TaskId: id}, nil
}

func (s *GameServerService) DeleteGameServerBackup(ctx context.Context, req *v1.DeleteGameServerBackupRequest) (*v1.DeleteGameServerBackupResponse, error) {
	if err := s.hosting.DeleteBackup(ctx, req); err != nil {
		return nil, err
	}
	return &v1.DeleteGameServerBackupResponse{}, nil
}

func (s *GameServerService) RestoreGameServerBackup(ctx context.Context, req *v1.RestoreGameServerBackupRequest) (*v1.GameServerTaskResponse, error) {
	id, err := s.hosting.RestoreBackup(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.GameServerTaskResponse{TaskId: id}, nil
}

func (s *GameServerService) DownloadGameServerBackup(ctx context.Context, req *v1.DownloadGameServerBackupRequest) (*v1.DownloadGameServerBackupResponse, error) {
	path, err := s.hosting.DownloadBackup(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.DownloadGameServerBackupResponse{DownloadUrlPath: path}, nil
}

func (s *GameServerService) UpdateGameServerBackupPolicy(ctx context.Context, req *v1.UpdateGameServerBackupPolicyRequest) (*v1.GameServerBackupPolicy, error) {
	return s.hosting.UpdateBackupPolicy(ctx, req)
}

// ---- 日志 ----

func (s *GameServerService) ListGameServerLogs(ctx context.Context, req *v1.ListGameServerLogsRequest) (*v1.ListGameServerLogsResponse, error) {
	return s.hosting.ListLogs(ctx, req.Id)
}

func (s *GameServerService) TailGameServerLog(ctx context.Context, req *v1.TailGameServerLogRequest) (*v1.TailGameServerLogResponse, error) {
	return s.hosting.TailLog(ctx, req)
}

// ---- 文件 ----

func (s *GameServerService) GetGameServerFileList(ctx context.Context, req *v1.GetInstanceFileListRequest) (*v1.GetInstanceFileListResponse, error) {
	return s.hosting.FileList(ctx, req)
}

func (s *GameServerService) GetGameServerFileTree(ctx context.Context, req *v1.GetInstanceFileTreeRequest) (*v1.GetInstanceFileTreeResponse, error) {
	return s.hosting.FileTree(ctx, req)
}

func (s *GameServerService) CreateGameServerFile(ctx context.Context, req *v1.CreateInstanceFileRequest) (*v1.CreateInstanceFileResponse, error) {
	return s.hosting.CreateFile(ctx, req)
}

func (s *GameServerService) RenameGameServerFile(ctx context.Context, req *v1.RenameInstanceFileRequest) (*v1.RenameInstanceFileResponse, error) {
	return s.hosting.RenameFile(ctx, req)
}

func (s *GameServerService) CopyGameServerFile(ctx context.Context, req *v1.CopyInstanceFileRequest) (*v1.CopyInstanceFileResponse, error) {
	return s.hosting.CopyFile(ctx, req)
}

func (s *GameServerService) CutGameServerFile(ctx context.Context, req *v1.CutInstanceFileRequest) (*v1.CutInstanceFileResponse, error) {
	return s.hosting.CutFile(ctx, req)
}

func (s *GameServerService) DeleteGameServerFiles(ctx context.Context, req *v1.BatchDeleteInstanceFileRequest) (*v1.BatchDeleteInstanceFileResponse, error) {
	return s.hosting.DeleteFiles(ctx, req)
}

func (s *GameServerService) CompressGameServerFiles(ctx context.Context, req *v1.BatchCompressInstanceFileRequest) (*v1.BatchCompressInstanceFileResponse, error) {
	return s.hosting.CompressFiles(ctx, req)
}

func (s *GameServerService) UnzipGameServerFile(ctx context.Context, req *v1.UnzipInstanceFileRequest) (*v1.UnzipInstanceFileResponse, error) {
	return s.hosting.UnzipFile(ctx, req)
}

func (s *GameServerService) EditGameServerFile(ctx context.Context, req *v1.EditInstanceFileRequest) (*v1.EditInstanceFileResponse, error) {
	return s.hosting.EditFile(ctx, req)
}

func (s *GameServerService) GameServerFilePreSign(ctx context.Context, req *v1.InstanceFilePreSignRequest) (*v1.InstanceFilePreSignResponse, error) {
	return s.hosting.FilePreSign(ctx, req)
}

func (s *GameServerService) GameServerFilePreSignUpload(ctx context.Context, req *v1.InstanceFilePreSignUploadRequest) (*v1.InstanceFilePreSignUploadResponse, error) {
	info, err := s.hosting.FilePreSignUpload(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.InstanceFilePreSignUploadResponse{Info: info}, nil
}

// ---- 原始路由：控制台 WebSocket 与实时动态 SSE ----

// RegisterRawRoutes 注册托管控制台（权限由 authz 的 wsRoutePermissions 校验）与实时动态（handler 内校验 game:server）。
func (s *GameServerService) RegisterRawRoutes(srv *khttp.Server) {
	srv.Handle(biz.GameServerConsoleWsPath, websocket.Handler(s.runConsole))
	srv.HandlePrefix(biz.GameServerEventsPrefix, nethttp.HandlerFunc(s.serveEvents))
}

func (s *GameServerService) runConsole(conn *websocket.Conn) {
	defer conn.Close()
	req := conn.Request()
	ctx := context.WithoutCancel(req.Context())
	if _, ok := auth.FromContext(ctx); !ok {
		_ = websocket.Message.Send(conn, "unauthorized")
		return
	}
	q := req.URL.Query()
	core, err := s.hosting.ConsoleCore(ctx, q.Get("id"), q.Get("role"))
	if err != nil {
		_ = websocket.Message.Send(conn, err.Error())
		return
	}
	s.hosting.Instances().StartInstanceWsConn(conn, core)
}

// serveEvents 转发游戏服务端的 MGP /events（会话、抽卡、培育、GM 指令、配置发布…）。
func (s *GameServerService) serveEvents(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx := r.Context()
	if _, ok := auth.FromContext(ctx); !ok {
		response.WriteError(w, r, biz.ErrTokenInvalid)
		return
	}
	if err := s.system.Check(ctx, constant.GameServer); err != nil {
		response.WriteError(w, r, err)
		return
	}
	serverID := strings.Trim(strings.TrimPrefix(r.URL.Path, biz.GameServerEventsPrefix), "/")
	_, client, err := s.uc.ClientOf(ctx, serverID)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	flusher, ok := w.(nethttp.Flusher)
	if !ok {
		response.WriteError(w, r, biz.ErrSystem(nil))
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(nethttp.StatusOK)
	_, _ = fmt.Fprint(w, ": connected\n\n")
	flusher.Flush()
	var mu sync.Mutex
	stop := make(chan struct{})
	go func() {
		ticker := time.NewTicker(15 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-stop:
				return
			case <-ticker.C:
				mu.Lock()
				_, _ = fmt.Fprint(w, ": keepalive\n\n")
				flusher.Flush()
				mu.Unlock()
			}
		}
	}()
	defer close(stop)
	err = client.Events(ctx, func(ev mgp.Event) {
		raw, err := json.Marshal(ev)
		if err != nil {
			return
		}
		mu.Lock()
		_, _ = fmt.Fprintf(w, "event: %s\ndata: %s\n\n", ev.Kind, raw)
		flusher.Flush()
		mu.Unlock()
	})
	if err != nil && ctx.Err() == nil {
		mu.Lock()
		_, _ = fmt.Fprintf(w, "event: error\ndata: %q\n\n", err.Error())
		flusher.Flush()
		mu.Unlock()
	}
}
