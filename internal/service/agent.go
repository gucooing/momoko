package service

import (
	"context"
	stdhttp "net/http"

	"github.com/go-kratos/kratos/v2/transport"
	"github.com/go-kratos/kratos/v2/transport/http"
	"golang.org/x/net/websocket"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
)

type AgentService struct {
	v1.UnimplementedAgentServiceServer
	v1.UnimplementedAgentPublicServer

	uc *biz.AgentUsecase
}

func NewAgentService(uc *biz.AgentUsecase) *AgentService {
	return &AgentService{uc: uc}
}

// RegisterRawRoutes 会话事件的 WebSocket。
func (s *AgentService) RegisterRawRoutes(srv *http.Server) {
	srv.Handle(biz.AgentWsPath, websocket.Handler(s.RunWsConn))
	// 官方账户的登录终端（?runtime=运行时 id）
	srv.Handle(biz.AgentLoginWsPath, websocket.Handler(s.RunLoginWsConn))
	// 文件页 HTML 预览：把目录当静态站点给 iframe 加载（签名鉴权，见 biz.AgentPreviewPrefix）
	srv.HandlePrefix(biz.AgentPreviewPrefix, stdhttp.HandlerFunc(s.uc.ServePreview))
}

// RunWsConn 页面的长连接：收 open（打开会话）/ more（往上翻）/ close（离开会话）/ watch（Agent 管理页关注所有人的会话）/
// answer（回答 Agent 的提问），把订阅的会话事件和会话的状态变化推回去。
func (s *AgentService) RunWsConn(conn *websocket.Conn) {
	defer conn.Close()
	ctx := context.WithoutCancel(conn.Request().Context())
	c, err := s.uc.Connect(ctx)
	if err != nil {
		return
	}
	defer s.uc.Disconnect(c)
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			var msg struct {
				Op     string `json:"op"`
				ID     string `json:"id"`
				Rev    int64  `json:"rev"`
				Before int64  `json:"before"`
				On     bool   `json:"on"`
				// answer：哪张提问卡片、accept / decline、各字段填的值
				QuestionID string              `json:"questionId"`
				Action     string              `json:"action"`
				Answers    map[string][]string `json:"answers"`
			}
			if err := websocket.JSON.Receive(conn, &msg); err != nil {
				return
			}
			switch msg.Op {
			case "open":
				s.uc.OpenStream(c, msg.ID, msg.Rev)
			case "more":
				s.uc.MoreStream(c, msg.ID, msg.Before)
			case "close":
				s.uc.CloseStream(c)
			case "watch":
				s.uc.WatchAll(c, msg.On)
			case "answer":
				s.uc.AnswerStream(c, msg.ID, msg.QuestionID, msg.Action, msg.Answers)
			}
		}
	}()
	for {
		select {
		case raw, ok := <-c.Out():
			if !ok {
				return
			}
			if err := websocket.Message.Send(conn, string(raw)); err != nil {
				return
			}
		case <-done:
			return
		}
	}
}

// requestOrigin 发起请求的页面地址：没设置对外地址时，通知里的链接用它。
func requestOrigin(ctx context.Context) string {
	tr, ok := transport.FromServerContext(ctx)
	if !ok {
		return ""
	}
	ht, ok := tr.(*http.Transport)
	if !ok {
		return ""
	}
	r := ht.Request()
	if origin := r.Header.Get("Origin"); origin != "" && origin != "null" {
		return origin
	}
	scheme := "http"
	if r.TLS != nil {
		scheme = "https"
	}
	return scheme + "://" + r.Host
}

// —— 选择器 ——

func (s *AgentService) GetAgentOptions(ctx context.Context, _ *v1.GetAgentOptionsRequest) (*v1.GetAgentOptionsResponse, error) {
	return s.uc.GetOptions(ctx)
}

func (s *AgentService) GetAgentOverview(ctx context.Context, req *v1.GetAgentOverviewRequest) (*v1.GetAgentOverviewResponse, error) {
	return s.uc.GetOverview(ctx, req.GetAll())
}

// —— 会话 ——

func (s *AgentService) GetAgentSidebar(ctx context.Context, req *v1.GetAgentSidebarRequest) (*v1.GetAgentSidebarResponse, error) {
	return s.uc.GetSidebar(ctx, req.GetKeyword())
}

func (s *AgentService) ListAgentSessions(ctx context.Context, req *v1.ListAgentSessionsRequest) (*v1.ListAgentSessionsResponse, error) {
	return s.uc.ListSessions(ctx, req)
}

func (s *AgentService) CreateAgentSession(ctx context.Context, req *v1.CreateAgentSessionRequest) (*v1.CreateAgentSessionResponse, error) {
	session, err := s.uc.CreateSession(ctx, req, requestOrigin(ctx))
	if err != nil {
		return nil, err
	}
	return &v1.CreateAgentSessionResponse{Session: session}, nil
}

func (s *AgentService) GetAgentSession(ctx context.Context, req *v1.GetAgentSessionRequest) (*v1.GetAgentSessionResponse, error) {
	return s.uc.GetSession(ctx, req.GetId())
}

func (s *AgentService) GetAgentSessionProcess(ctx context.Context, req *v1.GetAgentSessionProcessRequest) (*v1.GetAgentSessionProcessResponse, error) {
	process, err := s.uc.GetSessionProcess(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.GetAgentSessionProcessResponse{Process: process}, nil
}

func (s *AgentService) PromptAgentSession(ctx context.Context, req *v1.PromptAgentSessionRequest) (*v1.PromptAgentSessionResponse, error) {
	if err := s.uc.PromptSession(ctx, req, requestOrigin(ctx)); err != nil {
		return nil, err
	}
	return &v1.PromptAgentSessionResponse{}, nil
}

func (s *AgentService) CancelAgentSession(ctx context.Context, req *v1.CancelAgentSessionRequest) (*v1.CancelAgentSessionResponse, error) {
	if err := s.uc.CancelSession(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.CancelAgentSessionResponse{}, nil
}

func (s *AgentService) DeleteAgentQueuedPrompt(ctx context.Context, req *v1.DeleteAgentQueuedPromptRequest) (*v1.DeleteAgentQueuedPromptResponse, error) {
	if err := s.uc.UnqueuePrompt(ctx, req.GetId(), req.GetQueueId()); err != nil {
		return nil, err
	}
	return &v1.DeleteAgentQueuedPromptResponse{}, nil
}

func (s *AgentService) SendAgentQueuedPrompt(ctx context.Context, req *v1.SendAgentQueuedPromptRequest) (*v1.SendAgentQueuedPromptResponse, error) {
	if err := s.uc.SendQueuedNow(ctx, req.GetId(), req.GetQueueId()); err != nil {
		return nil, err
	}
	return &v1.SendAgentQueuedPromptResponse{}, nil
}

func (s *AgentService) StopAgentSessionItem(ctx context.Context, req *v1.StopAgentSessionItemRequest) (*v1.StopAgentSessionItemResponse, error) {
	if err := s.uc.StopSessionItem(ctx, req.GetId(), req.GetSub(), req.GetToolId()); err != nil {
		return nil, err
	}
	return &v1.StopAgentSessionItemResponse{}, nil
}

func (s *AgentService) ResolveAgentPermission(ctx context.Context, req *v1.ResolveAgentPermissionRequest) (*v1.ResolveAgentPermissionResponse, error) {
	if err := s.uc.ResolvePermission(ctx, req.GetId(), req.GetRequestId(), req.GetOptionId()); err != nil {
		return nil, err
	}
	return &v1.ResolveAgentPermissionResponse{}, nil
}

func (s *AgentService) SetAgentSessionOption(ctx context.Context, req *v1.SetAgentSessionOptionRequest) (*v1.SetAgentSessionOptionResponse, error) {
	session, err := s.uc.SetSessionOption(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.SetAgentSessionOptionResponse{Session: session}, nil
}

func (s *AgentService) RenameAgentSession(ctx context.Context, req *v1.RenameAgentSessionRequest) (*v1.RenameAgentSessionResponse, error) {
	if err := s.uc.RenameSession(ctx, req.GetId(), req.GetTitle()); err != nil {
		return nil, err
	}
	return &v1.RenameAgentSessionResponse{}, nil
}

func (s *AgentService) ForkAgentSession(ctx context.Context, req *v1.ForkAgentSessionRequest) (*v1.ForkAgentSessionResponse, error) {
	session, err := s.uc.ForkSession(ctx, req.GetId(), req.GetWorktree())
	if err != nil {
		return nil, err
	}
	return &v1.ForkAgentSessionResponse{Session: session}, nil
}

func (s *AgentService) DetachAgentSession(ctx context.Context, req *v1.DetachAgentSessionRequest) (*v1.DetachAgentSessionResponse, error) {
	if err := s.uc.DetachSession(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DetachAgentSessionResponse{}, nil
}

func (s *AgentService) ArchiveAgentSessions(ctx context.Context, req *v1.ArchiveAgentSessionsRequest) (*v1.ArchiveAgentSessionsResponse, error) {
	changed, skipped, err := s.uc.ArchiveSessions(ctx, req.GetIds(), req.GetArchived())
	if err != nil {
		return nil, err
	}
	return &v1.ArchiveAgentSessionsResponse{Changed: int32(changed), Skipped: int32(skipped)}, nil
}

func (s *AgentService) DeleteAgentSessions(ctx context.Context, req *v1.DeleteAgentSessionsRequest) (*v1.DeleteAgentSessionsResponse, error) {
	deleted, err := s.uc.DeleteSessions(ctx, req.GetIds(), req.GetWithForks())
	if err != nil {
		return nil, err
	}
	return &v1.DeleteAgentSessionsResponse{Deleted: int32(deleted)}, nil
}

func (s *AgentService) PurgeAgentSessions(ctx context.Context, _ *v1.PurgeAgentSessionsRequest) (*v1.PurgeAgentSessionsResponse, error) {
	deleted, err := s.uc.PurgeSessions(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.PurgeAgentSessionsResponse{Deleted: int32(deleted)}, nil
}

func (s *AgentService) ExportAgentSessions(ctx context.Context, req *v1.ExportAgentSessionsRequest) (*v1.ExportAgentSessionsResponse, error) {
	sessions, err := s.uc.ExportSessions(ctx, req.GetIds())
	if err != nil {
		return nil, err
	}
	return &v1.ExportAgentSessionsResponse{Sessions: sessions}, nil
}

// —— Git ——

func (s *AgentService) GetAgentGitStatus(ctx context.Context, req *v1.GetAgentGitStatusRequest) (*v1.GetAgentGitStatusResponse, error) {
	return s.uc.GitStatus(ctx, req.GetId())
}

func (s *AgentService) GetAgentGitDiff(ctx context.Context, req *v1.GetAgentGitDiffRequest) (*v1.GetAgentGitDiffResponse, error) {
	return s.uc.GitDiff(ctx, req)
}

func (s *AgentService) UpdateAgentGitFiles(ctx context.Context, req *v1.UpdateAgentGitFilesRequest) (*v1.UpdateAgentGitFilesResponse, error) {
	if err := s.uc.GitFiles(ctx, req); err != nil {
		return nil, err
	}
	return &v1.UpdateAgentGitFilesResponse{}, nil
}

func (s *AgentService) ApplyAgentGitLines(ctx context.Context, req *v1.ApplyAgentGitLinesRequest) (*v1.ApplyAgentGitLinesResponse, error) {
	if err := s.uc.GitLines(ctx, req); err != nil {
		return nil, err
	}
	return &v1.ApplyAgentGitLinesResponse{}, nil
}

func (s *AgentService) CommitAgentGit(ctx context.Context, req *v1.CommitAgentGitRequest) (*v1.CommitAgentGitResponse, error) {
	head, err := s.uc.GitCommit(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CommitAgentGitResponse{Head: head}, nil
}

func (s *AgentService) GenerateAgentGitMessage(ctx context.Context, req *v1.GenerateAgentGitMessageRequest) (*v1.GenerateAgentGitMessageResponse, error) {
	msg, err := s.uc.GitMessage(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return nil, err
	}
	return &v1.GenerateAgentGitMessageResponse{Message: msg}, nil
}

func (s *AgentService) ResetAgentGit(ctx context.Context, req *v1.ResetAgentGitRequest) (*v1.ResetAgentGitResponse, error) {
	if err := s.uc.GitReset(ctx, req); err != nil {
		return nil, err
	}
	return &v1.ResetAgentGitResponse{}, nil
}

func (s *AgentService) ListAgentGitRefs(ctx context.Context, req *v1.ListAgentGitRefsRequest) (*v1.ListAgentGitRefsResponse, error) {
	return s.uc.GitRefs(ctx, req.GetId(), req.GetRepo())
}

func (s *AgentService) UpdateAgentGitRef(ctx context.Context, req *v1.UpdateAgentGitRefRequest) (*v1.UpdateAgentGitRefResponse, error) {
	out, err := s.uc.GitRef(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateAgentGitRefResponse{Output: out}, nil
}

func (s *AgentService) SyncAgentGit(ctx context.Context, req *v1.SyncAgentGitRequest) (*v1.SyncAgentGitResponse, error) {
	out, err := s.uc.GitSync(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.SyncAgentGitResponse{Output: out}, nil
}

func (s *AgentService) UpdateAgentGitRemote(ctx context.Context, req *v1.UpdateAgentGitRemoteRequest) (*v1.UpdateAgentGitRemoteResponse, error) {
	if err := s.uc.GitRemote(ctx, req); err != nil {
		return nil, err
	}
	return &v1.UpdateAgentGitRemoteResponse{}, nil
}

func (s *AgentService) ListAgentGitStashes(ctx context.Context, req *v1.ListAgentGitStashesRequest) (*v1.ListAgentGitStashesResponse, error) {
	list, err := s.uc.GitStashes(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return nil, err
	}
	return &v1.ListAgentGitStashesResponse{Stashes: list}, nil
}

func (s *AgentService) UpdateAgentGitStash(ctx context.Context, req *v1.UpdateAgentGitStashRequest) (*v1.UpdateAgentGitStashResponse, error) {
	out, err := s.uc.GitStash(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateAgentGitStashResponse{Output: out}, nil
}

func (s *AgentService) RunAgentGitOperation(ctx context.Context, req *v1.RunAgentGitOperationRequest) (*v1.RunAgentGitOperationResponse, error) {
	out, err := s.uc.GitOperation(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.RunAgentGitOperationResponse{Output: out}, nil
}

func (s *AgentService) ListAgentGitLog(ctx context.Context, req *v1.ListAgentGitLogRequest) (*v1.ListAgentGitLogResponse, error) {
	return s.uc.GitLog(ctx, req)
}

func (s *AgentService) GetAgentGitCommit(ctx context.Context, req *v1.GetAgentGitCommitRequest) (*v1.GetAgentGitCommitResponse, error) {
	return s.uc.GitCommitDetail(ctx, req)
}

func (s *AgentService) InitAgentGitRepo(ctx context.Context, req *v1.InitAgentGitRepoRequest) (*v1.InitAgentGitRepoResponse, error) {
	if err := s.uc.GitInit(ctx, req.GetId(), req.GetPath()); err != nil {
		return nil, err
	}
	return &v1.InitAgentGitRepoResponse{}, nil
}

func (s *AgentService) ListAgentSessionFiles(ctx context.Context, req *v1.ListAgentSessionFilesRequest) (*v1.ListAgentSessionFilesResponse, error) {
	entries, err := s.uc.ListFiles(ctx, req.GetId(), req.GetDir())
	if err != nil {
		return nil, err
	}
	return &v1.ListAgentSessionFilesResponse{Entries: entries}, nil
}

func (s *AgentService) ReadAgentSessionFile(ctx context.Context, req *v1.ReadAgentSessionFileRequest) (*v1.ReadAgentSessionFileResponse, error) {
	return s.uc.ReadFile(ctx, req.GetId(), req.GetPath())
}

func (s *AgentService) GetAgentSessionUpload(ctx context.Context, req *v1.GetAgentSessionUploadRequest) (*v1.GetAgentSessionUploadResponse, error) {
	return s.uc.ReadUpload(ctx, req.GetId(), req.GetFile())
}

func (s *AgentService) SearchAgentFiles(ctx context.Context, req *v1.SearchAgentFilesRequest) (*v1.SearchAgentFilesResponse, error) {
	paths, err := s.uc.SearchFiles(ctx, req.GetSessionId(), req.GetWorkspaceId(), req.GetKeyword())
	if err != nil {
		return nil, err
	}
	return &v1.SearchAgentFilesResponse{Paths: paths}, nil
}

func (s *AgentService) SaveAgentFile(ctx context.Context, req *v1.SaveAgentFileRequest) (*v1.SaveAgentFileResponse, error) {
	return s.uc.SaveFile(ctx, req)
}

func (s *AgentService) CreateAgentDir(ctx context.Context, req *v1.CreateAgentDirRequest) (*v1.CreateAgentDirResponse, error) {
	if err := s.uc.MakeDir(ctx, req.GetSessionId(), req.GetWorkspaceId(), req.GetPath()); err != nil {
		return nil, err
	}
	return &v1.CreateAgentDirResponse{}, nil
}

func (s *AgentService) MoveAgentFile(ctx context.Context, req *v1.MoveAgentFileRequest) (*v1.MoveAgentFileResponse, error) {
	if err := s.uc.MoveFile(ctx, req.GetSessionId(), req.GetWorkspaceId(), req.GetFrom(), req.GetTo()); err != nil {
		return nil, err
	}
	return &v1.MoveAgentFileResponse{}, nil
}

func (s *AgentService) DeleteAgentFiles(ctx context.Context, req *v1.DeleteAgentFilesRequest) (*v1.DeleteAgentFilesResponse, error) {
	if err := s.uc.DeleteFiles(ctx, req.GetSessionId(), req.GetWorkspaceId(), req.GetPaths()); err != nil {
		return nil, err
	}
	return &v1.DeleteAgentFilesResponse{}, nil
}

func (s *AgentService) GetAgentFileRaw(ctx context.Context, req *v1.GetAgentFileRawRequest) (*v1.GetAgentSessionUploadResponse, error) {
	return s.uc.ReadRaw(ctx, req.GetSessionId(), req.GetWorkspaceId(), req.GetPath())
}

func (s *AgentService) GetAgentPreviewToken(ctx context.Context, req *v1.GetAgentPreviewTokenRequest) (*v1.GetAgentPreviewTokenResponse, error) {
	token, exp, err := s.uc.PreviewToken(ctx, req.GetSessionId(), req.GetWorkspaceId())
	if err != nil {
		return nil, err
	}
	return &v1.GetAgentPreviewTokenResponse{Token: token, ExpiresAt: exp}, nil
}

// —— 工作区 ——

func (s *AgentService) ListAgentWorkspaces(ctx context.Context, req *v1.ListAgentWorkspacesRequest) (*v1.ListAgentWorkspacesResponse, error) {
	workspaces, users, err := s.uc.ListWorkspaces(ctx, req.GetAll())
	if err != nil {
		return nil, err
	}
	return &v1.ListAgentWorkspacesResponse{Workspaces: workspaces, Users: users}, nil
}

func (s *AgentService) CreateAgentWorkspace(ctx context.Context, req *v1.CreateAgentWorkspaceRequest) (*v1.CreateAgentWorkspaceResponse, error) {
	workspace, err := s.uc.CreateWorkspace(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateAgentWorkspaceResponse{Workspace: workspace}, nil
}

func (s *AgentService) UpdateAgentWorkspace(ctx context.Context, req *v1.UpdateAgentWorkspaceRequest) (*v1.UpdateAgentWorkspaceResponse, error) {
	workspace, err := s.uc.UpdateWorkspace(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateAgentWorkspaceResponse{Workspace: workspace}, nil
}

func (s *AgentService) DeleteAgentWorkspace(ctx context.Context, req *v1.DeleteAgentWorkspaceRequest) (*v1.DeleteAgentWorkspaceResponse, error) {
	if err := s.uc.DeleteWorkspace(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteAgentWorkspaceResponse{}, nil
}

func (s *AgentService) ListAgentDirs(ctx context.Context, req *v1.ListAgentDirsRequest) (*v1.ListAgentDirsResponse, error) {
	return s.uc.ListDirs(ctx, req.GetPath())
}

func (s *AgentService) BrowseAgentWorkspace(ctx context.Context, req *v1.BrowseAgentWorkspaceRequest) (*v1.BrowseAgentWorkspaceResponse, error) {
	entries, err := s.uc.BrowseWorkspace(ctx, req.GetId(), req.GetDir())
	if err != nil {
		return nil, err
	}
	return &v1.BrowseAgentWorkspaceResponse{Entries: entries}, nil
}

func (s *AgentService) ReadAgentWorkspaceFile(ctx context.Context, req *v1.ReadAgentWorkspaceFileRequest) (*v1.ReadAgentSessionFileResponse, error) {
	return s.uc.ReadWorkspaceFile(ctx, req.GetId(), req.GetPath())
}

func (s *AgentService) GetAgentDocs(ctx context.Context, req *v1.GetAgentDocsRequest) (*v1.GetAgentDocsResponse, error) {
	docs, err := s.uc.GetDocs(ctx, req.GetPath())
	if err != nil {
		return nil, err
	}
	return &v1.GetAgentDocsResponse{Docs: docs}, nil
}

// —— 模型供应商 ——

func (s *AgentService) ListAgentProviders(ctx context.Context, req *v1.ListAgentProvidersRequest) (*v1.ListAgentProvidersResponse, error) {
	return s.uc.ListProviders(ctx, req.GetAll())
}

func providerOf(p *v1.AgentProvider) *v1.AgentProvider {
	if p == nil {
		return &v1.AgentProvider{}
	}
	return p
}

func (s *AgentService) CreateAgentProvider(ctx context.Context, req *v1.CreateAgentProviderRequest) (*v1.CreateAgentProviderResponse, error) {
	provider, err := s.uc.CreateProvider(ctx, providerOf(req.GetProvider()))
	if err != nil {
		return nil, err
	}
	return &v1.CreateAgentProviderResponse{Provider: provider}, nil
}

func (s *AgentService) UpdateAgentProvider(ctx context.Context, req *v1.UpdateAgentProviderRequest) (*v1.UpdateAgentProviderResponse, error) {
	provider, err := s.uc.UpdateProvider(ctx, req.GetId(), providerOf(req.GetProvider()))
	if err != nil {
		return nil, err
	}
	return &v1.UpdateAgentProviderResponse{Provider: provider}, nil
}

func (s *AgentService) DeleteAgentProvider(ctx context.Context, req *v1.DeleteAgentProviderRequest) (*v1.DeleteAgentProviderResponse, error) {
	if err := s.uc.DeleteProvider(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteAgentProviderResponse{}, nil
}

func (s *AgentService) TestAgentProvider(ctx context.Context, req *v1.TestAgentProviderRequest) (*v1.TestAgentProviderResponse, error) {
	return s.uc.TestProvider(ctx, req)
}

func (s *AgentService) FetchAgentProviderModels(ctx context.Context, req *v1.FetchAgentProviderModelsRequest) (*v1.FetchAgentProviderModelsResponse, error) {
	models, err := s.uc.FetchProviderModels(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.FetchAgentProviderModelsResponse{Models: models}, nil
}

func (s *AgentService) GetAgentProviderBalance(ctx context.Context, req *v1.GetAgentProviderBalanceRequest) (*v1.GetAgentProviderBalanceResponse, error) {
	return s.uc.ProviderBalance(ctx, req.GetId())
}

// —— MCP ——

func (s *AgentService) ListAgentMcpServers(ctx context.Context, req *v1.ListAgentMcpServersRequest) (*v1.ListAgentMcpServersResponse, error) {
	servers, users, err := s.uc.ListMcpServers(ctx, req.GetAll())
	if err != nil {
		return nil, err
	}
	return &v1.ListAgentMcpServersResponse{Servers: servers, Users: users}, nil
}

func mcpOf(m *v1.AgentMcpServer) *v1.AgentMcpServer {
	if m == nil {
		return &v1.AgentMcpServer{}
	}
	return m
}

func (s *AgentService) CreateAgentMcpServer(ctx context.Context, req *v1.CreateAgentMcpServerRequest) (*v1.CreateAgentMcpServerResponse, error) {
	server, err := s.uc.CreateMcpServer(ctx, mcpOf(req.GetServer()))
	if err != nil {
		return nil, err
	}
	return &v1.CreateAgentMcpServerResponse{Server: server}, nil
}

func (s *AgentService) UpdateAgentMcpServer(ctx context.Context, req *v1.UpdateAgentMcpServerRequest) (*v1.UpdateAgentMcpServerResponse, error) {
	server, err := s.uc.UpdateMcpServer(ctx, req.GetId(), mcpOf(req.GetServer()))
	if err != nil {
		return nil, err
	}
	return &v1.UpdateAgentMcpServerResponse{Server: server}, nil
}

func (s *AgentService) DeleteAgentMcpServer(ctx context.Context, req *v1.DeleteAgentMcpServerRequest) (*v1.DeleteAgentMcpServerResponse, error) {
	if err := s.uc.DeleteMcpServer(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteAgentMcpServerResponse{}, nil
}

func (s *AgentService) SetAgentMcpDefault(ctx context.Context, req *v1.SetAgentMcpDefaultRequest) (*v1.SetAgentMcpDefaultResponse, error) {
	if err := s.uc.SetMcpDefault(ctx, req.GetId(), req.GetDefaultOn()); err != nil {
		return nil, err
	}
	return &v1.SetAgentMcpDefaultResponse{}, nil
}

func (s *AgentService) TestAgentMcpServer(ctx context.Context, req *v1.TestAgentMcpServerRequest) (*v1.TestAgentMcpServerResponse, error) {
	return s.uc.TestMcpServer(ctx, req)
}

func (s *AgentService) ImportAgentMcpServers(ctx context.Context, req *v1.ImportAgentMcpServersRequest) (*v1.ImportAgentMcpServersResponse, error) {
	servers, err := s.uc.ImportMcpServers(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.ImportAgentMcpServersResponse{Servers: servers}, nil
}

// —— 技能 ——

func (s *AgentService) ListAgentSkills(ctx context.Context, req *v1.ListAgentSkillsRequest) (*v1.ListAgentSkillsResponse, error) {
	skills, users, err := s.uc.ListSkills(ctx, req.GetEnabled(), req.GetAll())
	if err != nil {
		return nil, err
	}
	return &v1.ListAgentSkillsResponse{Skills: skills, Users: users}, nil
}

func (s *AgentService) GetAgentSkill(ctx context.Context, req *v1.GetAgentSkillRequest) (*v1.GetAgentSkillResponse, error) {
	return s.uc.GetSkill(ctx, req.GetId())
}

func (s *AgentService) CreateAgentSkill(ctx context.Context, req *v1.CreateAgentSkillRequest) (*v1.CreateAgentSkillResponse, error) {
	skill, err := s.uc.CreateSkill(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.CreateAgentSkillResponse{Skill: skill}, nil
}

func (s *AgentService) UpdateAgentSkill(ctx context.Context, req *v1.UpdateAgentSkillRequest) (*v1.UpdateAgentSkillResponse, error) {
	skill, err := s.uc.UpdateSkill(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateAgentSkillResponse{Skill: skill}, nil
}

func (s *AgentService) DeleteAgentSkill(ctx context.Context, req *v1.DeleteAgentSkillRequest) (*v1.DeleteAgentSkillResponse, error) {
	if err := s.uc.DeleteSkill(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteAgentSkillResponse{}, nil
}

func (s *AgentService) SetAgentSkillEnabled(ctx context.Context, req *v1.SetAgentSkillEnabledRequest) (*v1.SetAgentSkillEnabledResponse, error) {
	if err := s.uc.SetSkillEnabled(ctx, req.GetId(), req.GetEnabled()); err != nil {
		return nil, err
	}
	return &v1.SetAgentSkillEnabledResponse{}, nil
}

func (s *AgentService) ImportAgentSkill(ctx context.Context, req *v1.ImportAgentSkillRequest) (*v1.ImportAgentSkillResponse, error) {
	skills, err := s.uc.ImportSkill(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.ImportAgentSkillResponse{Skills: skills}, nil
}

func (s *AgentService) ListAgentUsers(ctx context.Context, _ *v1.ListAgentUsersRequest) (*v1.ListAgentUsersResponse, error) {
	users, err := s.uc.ListUsers(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ListAgentUsersResponse{Users: users}, nil
}

// —— 运行时 ——

func (s *AgentService) ListAgentRuntimes(ctx context.Context, _ *v1.ListAgentRuntimesRequest) (*v1.ListAgentRuntimesResponse, error) {
	return s.uc.ListRuntimes(ctx)
}

func (s *AgentService) SetAgentRuntimeMode(ctx context.Context, req *v1.SetAgentRuntimeModeRequest) (*v1.SetAgentRuntimeModeResponse, error) {
	runtime, err := s.uc.SetRuntimeMode(ctx, req.GetId(), req.GetManaged())
	if err != nil {
		return nil, err
	}
	return &v1.SetAgentRuntimeModeResponse{Runtime: runtime}, nil
}

func (s *AgentService) DetectAgentRuntime(ctx context.Context, req *v1.DetectAgentRuntimeRequest) (*v1.DetectAgentRuntimeResponse, error) {
	runtime, err := s.uc.DetectRuntime(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.DetectAgentRuntimeResponse{Runtime: runtime}, nil
}

func (s *AgentService) SetAgentRuntimeModels(ctx context.Context, req *v1.SetAgentRuntimeModelsRequest) (*v1.SetAgentRuntimeModelsResponse, error) {
	runtime, err := s.uc.SetRuntimeModels(ctx, req.GetId(), req.GetModels())
	if err != nil {
		return nil, err
	}
	return &v1.SetAgentRuntimeModelsResponse{Runtime: runtime}, nil
}

func (s *AgentService) UpdateAgentRuntime(ctx context.Context, req *v1.UpdateAgentRuntimeRequest) (*v1.UpdateAgentRuntimeResponse, error) {
	return s.uc.UpdateRuntime(ctx, req.GetId())
}

func (s *AgentService) CreateAgentRuntime(ctx context.Context, req *v1.CreateAgentRuntimeRequest) (*v1.CreateAgentRuntimeResponse, error) {
	runtime, err := s.uc.CreateRuntime(ctx, req.GetName(), req.GetCommand())
	if err != nil {
		return nil, err
	}
	return &v1.CreateAgentRuntimeResponse{Runtime: runtime}, nil
}

func (s *AgentService) DeleteAgentRuntime(ctx context.Context, req *v1.DeleteAgentRuntimeRequest) (*v1.DeleteAgentRuntimeResponse, error) {
	if err := s.uc.DeleteRuntime(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.DeleteAgentRuntimeResponse{}, nil
}

func (s *AgentService) UpdateAgentRunSettings(ctx context.Context, req *v1.UpdateAgentRunSettingsRequest) (*v1.UpdateAgentRunSettingsResponse, error) {
	settings := req.GetSettings()
	if settings == nil {
		settings = &v1.AgentRunSettings{}
	}
	saved, err := s.uc.UpdateRunSettings(ctx, settings)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateAgentRunSettingsResponse{Settings: saved}, nil
}

func (s *AgentService) SetAgentRuntimeProxy(ctx context.Context, req *v1.SetAgentRuntimeProxyRequest) (*v1.SetAgentRuntimeProxyResponse, error) {
	runtime, err := s.uc.SetRuntimeProxy(ctx, req.GetId(), req.GetProxy())
	if err != nil {
		return nil, err
	}
	return &v1.SetAgentRuntimeProxyResponse{Runtime: runtime}, nil
}

func (s *AgentService) TestAgentProxy(ctx context.Context, req *v1.TestAgentProxyRequest) (*v1.TestAgentProxyResponse, error) {
	return s.uc.TestProxy(ctx, req)
}

// RunLoginWsConn 官方账户的登录终端：在服务器上跑登录命令，连接断开或命令结束为止。
func (s *AgentService) RunLoginWsConn(conn *websocket.Conn) {
	defer conn.Close()
	_ = s.uc.ServeLogin(conn, conn.Request().URL.Query().Get("runtime"))
}

// —— 通知 ——

func (s *AgentService) GetAgentNotify(ctx context.Context, _ *v1.GetAgentNotifyRequest) (*v1.GetAgentNotifyResponse, error) {
	return s.uc.GetNotify(ctx)
}

func (s *AgentService) UpdateAgentNotifyEvents(ctx context.Context, req *v1.UpdateAgentNotifyEventsRequest) (*v1.UpdateAgentNotifyEventsResponse, error) {
	if err := s.uc.UpdateNotifyEvents(ctx, req.GetEvents()); err != nil {
		return nil, err
	}
	return &v1.UpdateAgentNotifyEventsResponse{}, nil
}

func (s *AgentService) SaveAgentNotifyChannel(ctx context.Context, req *v1.SaveAgentNotifyChannelRequest) (*v1.SaveAgentNotifyChannelResponse, error) {
	channel, err := s.uc.SaveNotifyChannel(ctx, req.GetKind(), req.GetFields())
	if err != nil {
		return nil, err
	}
	return &v1.SaveAgentNotifyChannelResponse{Channel: channel}, nil
}

func (s *AgentService) SetAgentNotifyChannelEnabled(ctx context.Context, req *v1.SetAgentNotifyChannelEnabledRequest) (*v1.SetAgentNotifyChannelEnabledResponse, error) {
	if err := s.uc.SetNotifyChannelEnabled(ctx, req.GetUserId(), req.GetKind(), req.GetEnabled()); err != nil {
		return nil, err
	}
	return &v1.SetAgentNotifyChannelEnabledResponse{}, nil
}

func (s *AgentService) DeleteAgentNotifyChannel(ctx context.Context, req *v1.DeleteAgentNotifyChannelRequest) (*v1.DeleteAgentNotifyChannelResponse, error) {
	if err := s.uc.DeleteNotifyChannel(ctx, req.GetUserId(), req.GetKind()); err != nil {
		return nil, err
	}
	return &v1.DeleteAgentNotifyChannelResponse{}, nil
}

func (s *AgentService) ListAgentNotify(ctx context.Context, _ *v1.ListAgentNotifyRequest) (*v1.ListAgentNotifyResponse, error) {
	return s.uc.ListNotify(ctx)
}

func (s *AgentService) TestAgentNotifyChannel(ctx context.Context, req *v1.TestAgentNotifyChannelRequest) (*v1.TestAgentNotifyChannelResponse, error) {
	if err := s.uc.TestNotifyChannel(ctx, req.GetKind(), req.GetFields()); err != nil {
		return nil, err
	}
	return &v1.TestAgentNotifyChannelResponse{}, nil
}

// —— 公开批准页 ——

func (s *AgentService) PublicAgentApproval(ctx context.Context, req *v1.PublicAgentApprovalRequest) (*v1.PublicAgentApprovalResponse, error) {
	return s.uc.PublicApproval(ctx, req.GetToken())
}

func (s *AgentService) PublicResolveAgentApproval(ctx context.Context, req *v1.PublicResolveAgentApprovalRequest) (*v1.PublicResolveAgentApprovalResponse, error) {
	if err := s.uc.PublicResolveApproval(ctx, req.GetToken(), req.GetOptionId()); err != nil {
		return nil, err
	}
	return &v1.PublicResolveAgentApprovalResponse{}, nil
}
