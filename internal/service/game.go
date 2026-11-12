package service

import (
	"context"
	"encoding/json"
	"fmt"
	nethttp "net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/constant"
	"momoko/pkg/mgp"
	"momoko/pkg/response"
	"momoko/pkg/utils"
)

// gmAdmin 当前登录的管理员（GM 记录里的执行者）。
func gmAdmin(ctx context.Context, gm *biz.GmUsecase) (biz.GmActor, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return biz.GmActor{}, biz.ErrTokenInvalid
	}
	ua := biz.RequestUserAgent(ctx)
	return gm.AdminActor(ctx, a.UserID, utils.ClientIPFromContext(ctx), biz.Platform(ua)), nil
}

// 游戏服务器管理（GameServerService）见 game_server.go。

// ---------------- GM 运营 ----------------

type GameGMService struct {
	v1.UnimplementedGameGMServiceServer

	gm     *biz.GmUsecase
	system *biz.SystemUsecase
}

func NewGameGMService(gm *biz.GmUsecase, system *biz.SystemUsecase) *GameGMService {
	return &GameGMService{gm: gm, system: system}
}

func (s *GameGMService) SearchGmServers(ctx context.Context, req *v1.SearchGameServersRequest) (*v1.SearchGameServersResponse, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return s.gm.Servers().Search(ctx, a.UserID, biz.GameRecentScopeAdmin, req.Q, req.Game, int(req.Limit), nil)
}

func (s *GameGMService) GetGmContext(ctx context.Context, req *v1.GetGmContextRequest) (*v1.GetGmContextResponse, error) {
	a, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return s.gm.Context(ctx, a.UserID, req.ServerId)
}

func (s *GameGMService) ListGmPlayers(ctx context.Context, req *v1.ListGmPlayersRequest) (*v1.ListGmPlayersResponse, error) {
	return s.gm.ListPlayers(ctx, req)
}

func (s *GameGMService) GetGmPlayer(ctx context.Context, req *v1.GetGmPlayerRequest) (*v1.GetGmPlayerResponse, error) {
	return s.gm.GetPlayer(ctx, req.ServerId, req.Pid)
}

func (s *GameGMService) GetGmPlayerView(ctx context.Context, req *v1.GetGmPlayerViewRequest) (*v1.GmDataResponse, error) {
	raw, err := s.gm.Proxy(ctx, req.ServerId, "/players/"+url.PathEscape(req.Pid)+"/views/"+url.PathEscape(req.View), req.Query)
	if err != nil {
		return nil, err
	}
	return &v1.GmDataResponse{Data: biz.JSONValue(raw)}, nil
}

func (s *GameGMService) GetGmReport(ctx context.Context, req *v1.GetGmReportRequest) (*v1.GmDataResponse, error) {
	raw, err := s.gm.Proxy(ctx, req.ServerId, "/reports/"+url.PathEscape(req.Report), req.Query)
	if err != nil {
		return nil, err
	}
	return &v1.GmDataResponse{Data: biz.JSONValue(raw)}, nil
}

func (s *GameGMService) GetGmCatalog(ctx context.Context, req *v1.GetGmCatalogRequest) (*v1.GmDataResponse, error) {
	raw, err := s.gm.Proxy(ctx, req.ServerId, "/catalogs/"+url.PathEscape(req.Kind), req.Query)
	if err != nil {
		return nil, err
	}
	return &v1.GmDataResponse{Data: biz.JSONValue(raw)}, nil
}

func (s *GameGMService) ExecuteGmCommand(ctx context.Context, req *v1.ExecuteGmCommandRequest) (*v1.ExecuteGmCommandResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	var params json.RawMessage
	if req.Params != nil {
		params = biz.ValueJSON(req.Params)
	}
	e, raw, err := s.gm.Execute(ctx, actor, req.ServerId, req.Command, biz.GmCommandInput{
		Targets:     req.Targets,
		Params:      params,
		DryRun:      req.DryRun,
		ConfirmName: req.ConfirmName,
		Summary:     req.Summary,
		Source:      "gm",
	})
	if err != nil {
		return nil, err
	}
	return &v1.ExecuteGmCommandResponse{
		ExecutionId:  e.ID,
		Status:       string(e.Status),
		Result:       biz.JSONValue(raw),
		ErrorCode:    e.ErrorCode,
		ErrorMessage: e.ErrorMessage,
	}, nil
}

func (s *GameGMService) ListGmExecutions(ctx context.Context, req *v1.ListGmExecutionsRequest) (*v1.ListGmExecutionsResponse, error) {
	return s.gm.ListExecutions(ctx, req)
}

func (s *GameGMService) GetGmExecution(ctx context.Context, req *v1.GetGmExecutionRequest) (*v1.GmExecution, error) {
	return s.gm.GetExecution(ctx, req.ServerId, req.Id)
}

func (s *GameGMService) ApproveGmExecution(ctx context.Context, req *v1.ApproveGmExecutionRequest) (*v1.GmExecution, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gm.Approve(ctx, actor, req.ServerId, req.Id, req.ConfirmName, req.Note)
}

func (s *GameGMService) RejectGmExecution(ctx context.Context, req *v1.RejectGmExecutionRequest) (*v1.GmExecution, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gm.Reject(ctx, actor, req.ServerId, req.Id, req.Note)
}

func (s *GameGMService) ListGmSelfService(ctx context.Context, req *v1.ListGmSelfServiceRequest) (*v1.ListGmSelfServiceResponse, error) {
	items, _, err := s.gm.SelfServicePolicies(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	out := &v1.ListGmSelfServiceResponse{Items: items}
	for _, p := range items {
		if p.Enabled {
			out.Enabled++
		}
	}
	return out, nil
}

func (s *GameGMService) UpdateGmSelfService(ctx context.Context, req *v1.UpdateGmSelfServiceRequest) (*v1.GmSelfServicePolicy, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gm.UpdateSelfService(ctx, actor, req)
}

func (s *GameGMService) ListGmConfigs(ctx context.Context, req *v1.ListGmConfigsRequest) (*v1.ListGmConfigsResponse, error) {
	return s.gm.ListConfigs(ctx, req.ServerId)
}

func (s *GameGMService) GetGmConfig(ctx context.Context, req *v1.GetGmConfigRequest) (*v1.GetGmConfigResponse, error) {
	return s.gm.GetConfig(ctx, req.ServerId, req.Domain)
}

func (s *GameGMService) SaveGmConfigDraft(ctx context.Context, req *v1.SaveGmConfigDraftRequest) (*v1.GmConfigDraft, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gm.SaveConfigDraft(ctx, actor, req)
}

func (s *GameGMService) DiscardGmConfigDraft(ctx context.Context, req *v1.DiscardGmConfigDraftRequest) (*v1.DiscardGmConfigDraftResponse, error) {
	if err := s.gm.DiscardConfigDraft(ctx, req.ServerId, req.Domain); err != nil {
		return nil, err
	}
	return &v1.DiscardGmConfigDraftResponse{}, nil
}

func (s *GameGMService) ValidateGmConfig(ctx context.Context, req *v1.ValidateGmConfigRequest) (*v1.ValidateGmConfigResponse, error) {
	return s.gm.ValidateConfig(ctx, req.ServerId, req.Domain, strings.TrimSpace(req.Content))
}

func (s *GameGMService) PublishGmConfig(ctx context.Context, req *v1.PublishGmConfigRequest) (*v1.PublishGmConfigResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gm.PublishConfig(ctx, actor, req)
}

func (s *GameGMService) ListGmConfigRevisions(ctx context.Context, req *v1.ListGmConfigRevisionsRequest) (*v1.ListGmConfigRevisionsResponse, error) {
	return s.gm.ListConfigRevisions(ctx, req)
}

func (s *GameGMService) GetGmConfigRevision(ctx context.Context, req *v1.GetGmConfigRevisionRequest) (*v1.GmConfigRevision, error) {
	return s.gm.GetConfigRevision(ctx, req.ServerId, req.Domain, req.Id)
}

// GmEventsPrefix GM 实时动态（SSE）：/api/v1/game/gm-events/{serverId}，转发游戏服务端 MGP /events。
// 原始 HTTP 路由不经 operation 中间件，这里自行校验 game:gm 权限。
const GmEventsPrefix = "/api/v1/game/gm-events/"

// RegisterEventsServer 注册 GM 实时动态的 SSE 路由。
func (s *GameGMService) RegisterEventsServer(srv *khttp.Server) {
	srv.HandlePrefix(GmEventsPrefix, nethttp.HandlerFunc(s.serveEvents))
}

func (s *GameGMService) serveEvents(w nethttp.ResponseWriter, r *nethttp.Request) {
	ctx := r.Context()
	if _, ok := auth.FromContext(ctx); !ok {
		response.WriteError(w, r, biz.ErrTokenInvalid)
		return
	}
	if err := s.system.Check(ctx, constant.GameGM); err != nil {
		response.WriteError(w, r, err)
		return
	}
	serverID := strings.Trim(strings.TrimPrefix(r.URL.Path, GmEventsPrefix), "/")
	_, client, err := s.gm.Servers().ClientOf(ctx, serverID)
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
