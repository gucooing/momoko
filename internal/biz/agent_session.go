package biz

import (
	"context"
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/agentsession"
	"momoko/internal/data/ent/schema"
	"momoko/pkg/acp"
	"momoko/pkg/agentdriver"
	"momoko/pkg/agentfs"
	"momoko/pkg/agentgit"
	"momoko/pkg/agenthub"
	"momoko/pkg/agentrun"
	"momoko/pkg/response"
)

const (
	// agentTitleRunes 用第一条消息当标题时最多取多少个字。
	agentTitleRunes = 48
	// agentFileMax 能在页面里打开的文件大小；agentUploadMax 单个附件的大小。
	agentFileMax   = 1 << 20
	agentUploadMax = 20 << 20
	// agentPageItems 打开会话时先给最近这么多条事件，往上翻再要更早的。
	agentPageItems = 300
)

// ———————————————————————————— 读 ————————————————————————————

// getSession 取一个当前用户能动的会话：他自己的，或者他有管理 Agent 的权限。
func (uc *AgentUsecase) getSession(ctx context.Context, id string) (*gen.AgentSession, error) {
	userID, err := uc.scope(ctx)
	if err != nil {
		return nil, err
	}
	s, err := uc.repo.GetAgentSession(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrAgentSessionNotFound
		}
		return nil, ErrSystem(err)
	}
	if userID != "" && s.UserID != userID {
		return nil, ErrAgentSessionNotFound
	}
	return s, nil
}

func agentUsagePB(u agentrun.Usage) *v1.AgentUsage {
	return &v1.AgentUsage{Total: u.Total, Input: u.Input, Output: u.Output, CacheRead: u.CacheRead, CacheWrite: u.CacheWrite, Thought: u.Thought}
}

// sessionPB 会话给页面的样子：会话自己的字段，加上运行时、供应商、工作区的名称（都来自内存里的名称缓存，不查库）。
func (uc *AgentUsecase) sessionPB(s *gen.AgentSession, names *agentNames) *v1.AgentSession {
	out := &v1.AgentSession{
		Id:           s.ID,
		Title:        s.Title,
		RuntimeId:    s.RuntimeID,
		WorkspaceId:  s.WorkspaceID,
		Cwd:          s.Cwd,
		ExtraDirs:    s.ExtraDirs,
		Worktree:     s.Worktree,
		ProviderId:   s.ProviderID,
		Model:        s.Model,
		Effort:       s.Effort,
		Mode:         s.Mode,
		ModeName:     names.mode(s.RuntimeID, s.Mode),
		Status:       string(s.Status),
		Error:        s.Error,
		ParentId:     s.ParentID,
		Detached:     s.Detached,
		UserId:       s.UserID,
		ActiveAt:     timestamppb.New(s.ActiveAt),
		CreateTime:   timestamppb.New(s.CreateTime),
		AcpSessionId: s.AcpSessionID,
		Turns:        int32(s.Turns),
		ContextUsed:  s.ContextUsed,
		ContextSize:  s.ContextSize,
	}
	out.RuntimeName, out.RuntimeKind = names.runtime(s.RuntimeID)
	out.ProviderName, out.ProviderKind, out.ProviderVendor = names.provider(s.RuntimeID, s.ProviderID)
	if w := names.workspaces[s.WorkspaceID]; w != nil {
		out.WorkspaceName = w.Name
	}
	// 用量 Agent 给过才有。
	if s.LastUsage != "" {
		out.LastUsage = agentUsagePB(agentParse[agentrun.Usage](s.LastUsage))
		out.Usage = &v1.AgentUsage{
			Total: s.UsageTotal, Input: s.UsageInput, Output: s.UsageOutput,
			CacheRead: s.UsageCacheRead, CacheWrite: s.UsageCacheWrite, Thought: s.UsageThought,
		}
	}
	return out
}

func (uc *AgentUsecase) sessionsPB(ctx context.Context, rows []*gen.AgentSession) []*v1.AgentSession {
	names := uc.loadNames(ctx)
	out := make([]*v1.AgentSession, 0, len(rows))
	for _, s := range rows {
		out = append(out, uc.sessionPB(s, names))
	}
	return out
}

// sessionDetail 打开一个会话时才给的：当前分支、开着的 MCP、启用的技能、复刻来源、Agent 自报的名字和版本。
func (uc *AgentUsecase) sessionDetail(ctx context.Context, s *gen.AgentSession) *v1.AgentSessionDetail {
	out := &v1.AgentSessionDetail{}
	_, out.Branch = agentgit.Info(s.Cwd)
	if servers, err := uc.repo.ListAgentMcpServersByIDs(ctx, s.McpIds); err == nil {
		for _, m := range servers {
			out.McpNames = append(out.McpNames, m.Name)
		}
	}
	if skills, err := uc.repo.ListEnabledAgentSkills(ctx); err == nil {
		for _, sk := range skills {
			out.SkillNames = append(out.SkillNames, sk.Name)
		}
	}
	if s.ParentID != "" {
		if parent, err := uc.repo.GetAgentSession(ctx, s.ParentID); err == nil {
			out.ParentTitle = parent.Title
		}
	}
	if rt := uc.loadNames(ctx).runtimes[s.RuntimeID]; rt != nil {
		out.AgentName, out.AgentVersion = rt.AgentName, rt.AgentVersion
	}
	return out
}

func (uc *AgentUsecase) GetSession(ctx context.Context, id string) (*v1.GetAgentSessionResponse, error) {
	s, err := uc.getSession(ctx, id)
	if err != nil {
		return nil, err
	}
	return &v1.GetAgentSessionResponse{Session: uc.sessionPB(s, uc.loadNames(ctx)), Detail: uc.sessionDetail(ctx, s)}, nil
}

// GetSessionProcess 会话的 Agent 进程：命令、进程号、内存、启动时间。内存要遍历进程树，只在有人要看时读。
func (uc *AgentUsecase) GetSessionProcess(ctx context.Context, id string) (*v1.AgentProcess, error) {
	if _, err := uc.getSession(ctx, id); err != nil {
		return nil, err
	}
	l := uc.live(id)
	if l == nil || l.session == nil {
		return nil, nil
	}
	proc := l.session.Process()
	return &v1.AgentProcess{
		Command:     l.launch.Command,
		Args:        l.launch.Args,
		Pid:         int32(proc.PID()),
		MemoryBytes: proc.MemoryBytes(),
		StartedAt:   timestamppb.New(proc.StartedAt()),
	}, nil
}

// GetSidebar 会话栏：我的未归档会话，和所有工作区。
func (uc *AgentUsecase) GetSidebar(ctx context.Context, keyword string) (*v1.GetAgentSidebarResponse, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := uc.repo.ListAgentSidebarSessions(ctx, userID, keyword, 500)
	if err != nil {
		return nil, ErrSystem(err)
	}
	resp := &v1.GetAgentSidebarResponse{Sessions: uc.sessionsPB(ctx, rows)}
	names := uc.loadNames(ctx)
	workspaces := make([]*gen.AgentWorkspace, 0, len(names.workspaces))
	for _, w := range names.workspaces {
		workspaces = append(workspaces, w)
	}
	sortWorkspaces(workspaces)
	for _, w := range workspaces {
		item := &v1.AgentSidebarWorkspace{Id: w.ID, Name: w.Name, Path: w.Path}
		item.IsGit, item.Branch = agentgit.Info(w.Path)
		resp.Workspaces = append(resp.Workspaces, item)
	}
	return resp, nil
}

// ListSessions 管理页的会话列表。有管理权限的人看所有人的，其他人只看自己的。
func (uc *AgentUsecase) ListSessions(ctx context.Context, req *v1.ListAgentSessionsRequest) (*v1.ListAgentSessionsResponse, error) {
	userID, err := uc.scope(ctx)
	if err != nil {
		return nil, err
	}
	filter := AgentSessionFilter{UserID: userID, Status: req.GetStatus(), Keyword: req.GetKeyword(), Page: int(req.GetPage()), PageSize: int(req.GetPageSize())}
	filter.Page = max(filter.Page, 1)
	if filter.PageSize < 1 || filter.PageSize > 200 {
		filter.PageSize = 20
	}
	rows, total, counts, err := uc.repo.ListAgentSessions(ctx, filter)
	if err != nil {
		return nil, ErrSystem(err)
	}
	ids := make([]string, 0, len(rows))
	owners := make([]string, 0, len(rows))
	for _, s := range rows {
		ids = append(ids, s.ID)
		owners = append(owners, s.UserID)
	}
	forks, err := uc.repo.CountAgentForks(ctx, ids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	users, err := uc.repo.AgentUsers(ctx, owners)
	if err != nil {
		return nil, ErrSystem(err)
	}
	resp := &v1.ListAgentSessionsResponse{
		Sessions:      uc.sessionsPB(ctx, rows),
		Total:         total,
		CountAll:      counts.All,
		CountRunning:  counts.Running,
		CountWaiting:  counts.Waiting,
		CountIdle:     counts.Idle,
		CountError:    counts.Error,
		CountArchived: counts.Archived,
		ForkCounts:    map[string]int32{},
	}
	for id, n := range forks {
		resp.ForkCounts[id] = int32(n)
	}
	for _, u := range users {
		resp.Users = append(resp.Users, &v1.AgentUser{Id: u.ID, Name: u.Name, Username: u.Username})
	}
	return resp, nil
}

// ———————————————————————————— 新建和发消息 ————————————————————————————

// agentTitle 会话还没有标题时用第一条消息的开头；Agent 之后给了标题就换成它的。
func agentTitle(text string) string {
	line, _, _ := strings.Cut(strings.TrimSpace(text), "\n")
	line = strings.Join(strings.Fields(line), " ")
	if r := []rune(line); len(r) > agentTitleRunes {
		line = string(r[:agentTitleRunes])
	}
	return line
}

// checkProvider 会话能不能用这个供应商：接口协议要对得上这个 Agent。空表示服务器上的登录，最近一次检查时要登录着。
func (uc *AgentUsecase) checkProvider(ctx context.Context, rt *gen.AgentRuntime, providerID string) (*gen.AgentProvider, error) {
	if providerID == "" {
		if !rt.LoggedIn {
			return nil, response.BadRequest(rt.Name + " 在服务器上还没有登录：在服务器上登录后，到「模型供应商」里点「检查」")
		}
		return nil, nil
	}
	p, err := uc.getProvider(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if !uc.serves(p, uc.driverOf(string(rt.Kind))) {
		return nil, response.BadRequest("这个供应商的接口和 " + rt.Name + " 不兼容")
	}
	return p, nil
}

// checkAttachments 附件先校验，校验通过才动磁盘。ref 是 root 里的相对路径。
func checkAttachments(root, text string, in []*v1.AgentAttachment) error {
	for _, a := range in {
		switch a.GetKind() {
		case "ref":
			if _, err := agentfs.SafeJoin(root, a.GetPath()); err != nil {
				return agentErr(err)
			}
		case "image", "file":
			name := filepath.Base(filepath.ToSlash(strings.TrimSpace(a.GetName())))
			if len(a.GetData()) == 0 {
				return response.BadRequest("附件 " + name + " 是空的")
			}
			if len(a.GetData()) > agentUploadMax {
				return response.BadRequest("附件 " + name + " 太大了")
			}
		default:
			return response.BadRequest("不认识的附件类型")
		}
	}
	if text == "" && len(in) == 0 {
		return response.BadRequest("先写点什么")
	}
	return nil
}

// saveUploads 把附件整理成交给引擎的样子。上传的附件都存进会话目录的 uploads/，事件里记下 File，
// 页面凭它取回来显示；Agent 收图片时图片另外直接带内容，其它附件让 Agent 按路径自己去读。
func (uc *AgentUsecase) saveUploads(sessionID, cwd string, caps agentrun.Caps, in []*v1.AgentAttachment) ([]agentrun.Upload, error) {
	dir := uc.sessionDir(sessionID, "uploads")
	out := make([]agentrun.Upload, 0, len(in))
	for _, a := range in {
		if a.GetKind() == "ref" {
			path, _ := agentfs.SafeJoin(cwd, a.GetPath())
			out = append(out, agentrun.Upload{Kind: "ref", Name: a.GetPath(), Path: path})
			continue
		}
		name := filepath.Base(filepath.ToSlash(strings.TrimSpace(a.GetName())))
		path, err := agentfs.Save(dir, name, a.GetData())
		if err != nil {
			return nil, response.BadRequest("保存附件失败：" + err.Error())
		}
		u := agentrun.Upload{Kind: a.GetKind(), Name: name, Mime: a.GetMime(), Path: path, File: agentfs.Rel(dir, path)}
		if a.GetKind() == "image" && caps.Image {
			u.Data = a.GetData()
		}
		out = append(out, u)
	}
	return out, nil
}

// startTurn 起进程（需要的话）并开始一轮。起进程可能要等上一阵，所以不在请求里做：
// 失败了把原因记在会话上，推给页面，并按「出错时」通知。
func (uc *AgentUsecase) startTurn(s *gen.AgentSession, text string, uploads []agentrun.Upload) {
	ctx := context.Background()
	l, err := uc.ensureLive(ctx, s)
	if err == nil {
		err = l.session.Prompt(text, uploads)
	}
	if err == nil {
		return
	}
	_, _ = uc.updateSession(ctx, s.ID, func(u *gen.AgentSessionUpdateOne) {
		u.SetStatus(agentsession.StatusError).SetError(err.Error())
	})
	uc.notify(s.ID, "error", err.Error(), nil)
}

// CreateSession 新建会话并发出第一条消息。立刻返回，后面的事件走 WebSocket。
// origin 是发起请求的页面地址，没设置对外地址时通知里的链接用它。
func (uc *AgentUsecase) CreateSession(ctx context.Context, req *v1.CreateAgentSessionRequest, origin string) (*v1.AgentSession, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	rt, err := uc.getRuntime(ctx, req.GetRuntimeId())
	if err != nil {
		return nil, err
	}
	if !rt.Ready {
		return nil, ErrAgentRuntimeNotReady
	}
	if _, err := uc.checkProvider(ctx, rt, req.GetProviderId()); err != nil {
		return nil, err
	}
	text := strings.TrimSpace(req.GetText())
	now := time.Now()
	s := &gen.AgentSession{
		ID:         "ses_" + uuid.NewString(),
		ActiveAt:   now,
		UserID:     userID,
		Title:      agentTitle(text),
		RuntimeID:  rt.ID,
		ProviderID: req.GetProviderId(),
		Model:      req.GetModel(),
		Effort:     req.GetEffort(),
		Mode:       req.GetMode(),
		Status:     agentsession.StatusRunning,
		McpIds:     req.GetMcpIds(),
		Origin:     origin,
	}
	for _, dir := range req.GetExtraDirs() {
		clean, err := agentfs.CleanDir(dir)
		if err != nil {
			return nil, agentErr(err)
		}
		if !agentfs.IsDir(clean) {
			return nil, response.BadRequest("服务器上没有这个目录：" + clean)
		}
		s.ExtraDirs = append(s.ExtraDirs, clean)
	}
	var workspace *gen.AgentWorkspace
	if req.GetWorkspaceId() != "" {
		if workspace, err = uc.getWorkspace(ctx, req.GetWorkspaceId()); err != nil {
			return nil, err
		}
		if workspace.CloneStatus != "" {
			return nil, response.BadRequest("这个工作区的仓库还没克隆好")
		}
		if !agentfs.IsDir(workspace.Path) {
			return nil, response.BadRequest("工作区的目录已经不存在了")
		}
		if req.GetWorktree() {
			if isGit, _ := agentgit.Info(workspace.Path); !isGit {
				return nil, response.BadRequest("这个目录不是 Git 仓库，不能用独立工作树")
			}
		}
		s.WorkspaceID, s.Cwd = workspace.ID, workspace.Path
	} else {
		s.Cwd = uc.sessionDir(s.ID, "work")
	}
	if err := checkAttachments(s.Cwd, text, req.GetAttachments()); err != nil {
		return nil, err
	}

	// 校验都过了才动磁盘；后面哪一步失败，把已经建出来的收拾掉。
	bg := context.WithoutCancel(ctx)
	var worktree *agentgit.Worktree
	cleanup := func() {
		if worktree != nil {
			agentgit.RemoveWorktree(bg, workspace.Path, worktree, true)
		}
		agentfs.RemoveAll(uc.sessionDir(s.ID))
	}
	if workspace != nil && req.GetWorktree() {
		if worktree, err = agentgit.AddWorktree(bg, workspace.Path, uc.sessionDir(s.ID, "worktree"), agentfs.Slug(workspace.Name), "HEAD"); err != nil {
			cleanup()
			return nil, agentErr(err)
		}
		s.Cwd, s.Worktree = worktree.Path, worktree.Name
	}
	uploads, err := uc.saveUploads(s.ID, s.Cwd, agentParse[agentrun.Caps](rt.Caps), req.GetAttachments())
	if err != nil {
		cleanup()
		return nil, err
	}
	created, err := uc.repo.CreateAgentSession(ctx, s)
	if err != nil {
		cleanup()
		return nil, ErrSystem(err)
	}
	uc.pushSession(ctx, created)
	go uc.startTurn(created, text, uploads)
	return uc.sessionPB(created, uc.loadNames(ctx)), nil
}

// PromptSession 发一条消息，开始新的一轮。
func (uc *AgentUsecase) PromptSession(ctx context.Context, req *v1.PromptAgentSessionRequest, origin string) error {
	s, err := uc.getSession(ctx, req.GetId())
	if err != nil {
		return err
	}
	rt, err := uc.getRuntime(ctx, s.RuntimeID)
	if err != nil {
		return err
	}
	text := strings.TrimSpace(req.GetText())
	if err := checkAttachments(s.Cwd, text, req.GetAttachments()); err != nil {
		return err
	}
	// 只有空闲和出错的会话能开始新的一轮：条件写在更新里，两个请求同时来时只有一个成功。
	untitled := s.Title == "" && !s.TitleCustom
	started, err := uc.updateSession(ctx, s.ID, func(u *gen.AgentSessionUpdateOne) {
		u.Where(agentsession.StatusIn(agentsession.StatusIdle, agentsession.StatusError)).
			SetStatus(agentsession.StatusRunning).SetError("").SetActiveAt(time.Now())
		if origin != "" {
			u.SetOrigin(origin)
		}
		if untitled {
			u.SetTitle(agentTitle(text))
		}
	})
	if err != nil {
		if gen.IsNotFound(err) {
			if s.Status == agentsession.StatusArchived {
				return ErrAgentSessionArchived
			}
			return ErrAgentSessionBusy
		}
		return ErrSystem(err)
	}
	uploads, err := uc.saveUploads(started.ID, started.Cwd, agentParse[agentrun.Caps](rt.Caps), req.GetAttachments())
	if err != nil {
		_, _ = uc.updateSession(ctx, started.ID, func(u *gen.AgentSessionUpdateOne) { u.SetStatus(agentsession.StatusIdle) })
		return err
	}
	go uc.startTurn(started, text, uploads)
	return nil
}

// CancelSession 停止当前这一轮。
func (uc *AgentUsecase) CancelSession(ctx context.Context, id string) error {
	if _, err := uc.getSession(ctx, id); err != nil {
		return err
	}
	if l := uc.live(id); l != nil {
		l.session.Cancel()
	}
	return nil
}

func (uc *AgentUsecase) ResolvePermission(ctx context.Context, id, requestID, optionID string) error {
	if _, err := uc.getSession(ctx, id); err != nil {
		return err
	}
	return uc.resolvePermission(id, requestID, optionID)
}

// ———————————————————————————— 会话里的选项 ————————————————————————————

// effortFor 换了模型之后推理强度用哪个：原来的这个模型也有就沿用，没有就用这个模型默认的。
// 第三方接口看模型目录里声明的等级；服务器上的登录看模型表里这个模型的那一行（loginTo）。
func effortFor(p *gen.AgentProvider, loginTo *gen.AgentRuntimeModel, model, current string) string {
	if p != nil {
		ids, def := agentLevels(p, model)
		if slices.Contains(ids, current) {
			return current
		}
		return def
	}
	if loginTo == nil {
		return ""
	}
	if slices.ContainsFunc(loginTo.Efforts, func(e schema.AgentChoice) bool { return e.ID == current }) {
		return current
	}
	return loginTo.DefaultEffort
}

// restartFor 这次改动要换一个进程才生效：正在跑的一轮不能打断，空闲的进程直接结束，下次发消息时按新的配置起。
func restartFor(l *agentLive) error {
	if l == nil {
		return nil
	}
	if l.Busy() {
		return ErrAgentSessionBusy
	}
	l.Shutdown()
	return nil
}

// SetSessionOption 切换权限模式、模型或推理强度。
//   - 权限模式，和服务器上的登录里换模型、换推理强度：进程在就直接告诉 Agent；
//   - 换供应商，第三方接口里换模型、换推理等级，或者服务器上的登录里换到上下文、输出设得不一样的模型：
//     这些是启动进程时定下来的，要换一个进程。
func (uc *AgentUsecase) SetSessionOption(ctx context.Context, req *v1.SetAgentSessionOptionRequest) (*v1.AgentSession, error) {
	s, err := uc.getSession(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	if s.Status == agentsession.StatusArchived {
		return nil, ErrAgentSessionArchived
	}
	rt, err := uc.getRuntime(ctx, s.RuntimeID)
	if err != nil {
		return nil, err
	}
	value := strings.TrimSpace(req.GetValue())
	if value == "" {
		return nil, response.BadRequest("先选一个值")
	}
	bg := context.WithoutCancel(ctx)
	l := uc.live(s.ID)
	mode, model, effort, providerID := s.Mode, s.Model, s.Effort, s.ProviderID
	switch req.GetKind() {
	case "mode":
		if l != nil {
			if err := l.session.SetOption(bg, acp.CategoryMode, value); err != nil {
				return nil, agentErr(err)
			}
		}
		mode = value
	case "model":
		p, err := uc.checkProvider(ctx, rt, req.GetProviderId())
		if err != nil {
			return nil, err
		}
		byEnv := p != nil
		if byEnv {
			known := false
			for _, m := range p.Models {
				known = known || m.ID == value
			}
			if !known {
				return nil, response.BadRequest("这个供应商的模型目录里没有 " + value)
			}
		}
		var loginTo *gen.AgentRuntimeModel
		if !byEnv {
			if loginTo, err = uc.loginModel(ctx, rt, value); err != nil {
				return nil, err
			}
		}
		restart := byEnv || req.GetProviderId() != s.ProviderID
		// 服务器上的登录里两个模型设的上下文、输出不一样：这是启动时交给 Agent 的，也要换一个进程
		if !restart {
			loginFrom, err := uc.loginModel(ctx, rt, s.Model)
			if err != nil {
				return nil, err
			}
			ctxFrom, outFrom := loginLimits(loginFrom)
			ctxTo, outTo := loginLimits(loginTo)
			restart = ctxFrom != ctxTo || outFrom != outTo
		}
		providerID, model, effort = req.GetProviderId(), value, effortFor(p, loginTo, value, s.Effort)
		if restart {
			if err := restartFor(l); err != nil {
				return nil, err
			}
		} else if l != nil {
			if err := l.session.SetOption(bg, acp.CategoryModel, value); err != nil {
				return nil, agentErr(err)
			}
			effort = l.session.Effort()
		}
	case "effort":
		var p *gen.AgentProvider
		if s.ProviderID != "" {
			if p, err = uc.getProvider(ctx, s.ProviderID); err != nil {
				return nil, err
			}
		}
		if p != nil {
			if _, ok := agentLevelValue(p, s.Model, value); !ok {
				return nil, response.BadRequest("这个模型没有声明这个推理等级")
			}
			if err := restartFor(l); err != nil {
				return nil, err
			}
		} else if l != nil {
			if err := l.session.SetOption(bg, acp.CategoryThoughtLevel, value); err != nil {
				return nil, agentErr(err)
			}
		}
		effort = value
	default:
		return nil, response.BadRequest("不认识的选项")
	}
	s, err = uc.updateSession(bg, s.ID, func(u *gen.AgentSessionUpdateOne) {
		u.SetMode(mode).SetModel(model).SetEffort(effort).SetProviderID(providerID)
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	return uc.sessionPB(s, uc.loadNames(ctx)), nil
}

// RenameSession 用户起的标题只存在 Momoko 这边，之后 Agent 给的标题不再覆盖它。
func (uc *AgentUsecase) RenameSession(ctx context.Context, id, title string) error {
	if _, err := uc.getSession(ctx, id); err != nil {
		return err
	}
	if title = strings.TrimSpace(title); title == "" {
		return response.BadRequest("先填一个名称")
	}
	if l := uc.live(id); l != nil {
		l.session.SetTitleCustom(true)
	}
	if _, err := uc.updateSession(ctx, id, func(u *gen.AgentSessionUpdateOne) { u.SetTitle(title).SetTitleCustom(true) }); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// ———————————————————————————— 复刻、归档、删除、导出 ————————————————————————————

// ForkSession 复刻对话：Agent 侧分出一份上下文，Momoko 这边把对话记录整份带过去。复刻出来的会话有自己的会话目录，删哪个都不影响另一个：
//   - 没选工作区的：把原会话的工作目录复制一份过去；
//   - 原会话在独立工作树里，或 worktree 为 true：在自己的会话目录里建一个新工作树。原会话在工作树里的，
//     从它现在的提交分出来，还没提交的改动也带过去；直接在工作区里的，从工作区当前的 HEAD 分出来；
//   - 其它（直接在工作区里，不要工作树）：和原会话用同一个工作区目录。
func (uc *AgentUsecase) ForkSession(ctx context.Context, id string, worktree bool) (*v1.AgentSession, error) {
	s, err := uc.getSession(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.Status == agentsession.StatusArchived {
		return nil, ErrAgentSessionArchived
	}
	if s.AcpSessionID == "" {
		return nil, response.BadRequest("这个会话还没有开始，没有可以复刻的对话")
	}
	rt, err := uc.getRuntime(ctx, s.RuntimeID)
	if err != nil {
		return nil, err
	}
	if !agentParse[agentrun.Caps](rt.Caps).Fork {
		return nil, response.BadRequest("这个 Agent 不支持复刻对话")
	}
	repo := ""
	if w := uc.loadNames(ctx).workspaces[s.WorkspaceID]; w != nil {
		repo = w.Path
	}
	if repo == "" && worktree {
		return nil, response.BadRequest("没有工作区的会话复刻时会复制一份工作目录，用不上工作树")
	}
	worktree = worktree || s.Worktree != ""
	if worktree {
		if isGit, _ := agentgit.Info(repo); !isGit {
			return nil, response.BadRequest("这个目录不是 Git 仓库，不能用独立工作树")
		}
	}

	bg := context.WithoutCancel(ctx)
	forkID := "ses_" + uuid.NewString()
	cwd, wtName := s.Cwd, ""
	var wt *agentgit.Worktree
	fail := func(err error) (*v1.AgentSession, error) {
		if wt != nil {
			agentgit.RemoveWorktree(bg, repo, wt, true)
		}
		agentfs.RemoveAll(uc.sessionDir(forkID))
		return nil, err
	}
	switch {
	case worktree:
		base := "HEAD"
		if s.Worktree != "" {
			if base, err = agentgit.Head(bg, s.Cwd); err != nil {
				return fail(agentErr(err))
			}
		}
		if wt, err = agentgit.AddWorktree(bg, repo, uc.sessionDir(forkID, "worktree"), agentfs.Slug(filepath.Base(repo)), base); err != nil {
			return fail(agentErr(err))
		}
		if s.Worktree != "" {
			if err := agentgit.CopyChanges(bg, s.Cwd, wt.Path); err != nil {
				return fail(response.BadRequest("带过去没提交的改动失败：" + err.Error()))
			}
		}
		cwd, wtName = wt.Path, wt.Name
	case repo == "":
		cwd = uc.sessionDir(forkID, "work")
		if err := agentfs.CopyDir(s.Cwd, cwd); err != nil {
			return fail(response.BadRequest("复制工作目录失败：" + err.Error()))
		}
	}
	// 复刻带着原来的消息，消息里的附件也一起带过去：页面是按会话取回附件的。
	if err := agentfs.CopyDir(uc.sessionDir(s.ID, "uploads"), uc.sessionDir(forkID, "uploads")); err != nil {
		return fail(response.BadRequest("复制附件失败：" + err.Error()))
	}
	// 复刻要在原会话的进程里做（ACP 的 session/fork 是对一个连着的会话说的）。
	l, err := uc.ensureLive(bg, s)
	if err != nil {
		return fail(err)
	}
	acpID, err := l.session.Fork(bg, cwd, uc.agentDirs(forkID, s.ExtraDirs))
	if err != nil {
		return fail(agentErr(err))
	}
	// 起进程时可能刚换过模型，以库里现在的为准。
	if s, err = uc.repo.GetAgentSession(bg, s.ID); err != nil {
		return fail(ErrSystem(err))
	}
	fork := &gen.AgentSession{
		ID:           forkID,
		ActiveAt:     time.Now(),
		UserID:       s.UserID,
		Title:        s.Title,
		TitleCustom:  s.TitleCustom,
		RuntimeID:    s.RuntimeID,
		WorkspaceID:  s.WorkspaceID,
		Cwd:          cwd,
		ExtraDirs:    s.ExtraDirs,
		Worktree:     wtName,
		ProviderID:   s.ProviderID,
		Model:        s.Model,
		Effort:       s.Effort,
		Mode:         s.Mode,
		Status:       agentsession.StatusIdle,
		AcpSessionID: acpID,
		ParentID:     s.ID,
		ContextUsed:  s.ContextUsed,
		ContextSize:  s.ContextSize,
		McpIds:       s.McpIds,
		State:        s.State,
		Origin:       s.Origin,
	}
	created, err := uc.repo.ForkAgentSession(bg, fork, s.ID)
	if err != nil {
		return fail(ErrSystem(err))
	}
	uc.pushSession(bg, created)
	return uc.sessionPB(created, uc.loadNames(bg)), nil
}

// DetachSession 把复刻出来的会话移到外层。
func (uc *AgentUsecase) DetachSession(ctx context.Context, id string) error {
	if _, err := uc.getSession(ctx, id); err != nil {
		return err
	}
	if _, err := uc.updateSession(ctx, id, func(u *gen.AgentSessionUpdateOne) { u.SetDetached(true) }); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// ownSessions 这些 id 里当前用户能动的会话（筛选在 SQL 里）。
func (uc *AgentUsecase) ownSessions(ctx context.Context, ids []string) ([]*gen.AgentSession, error) {
	userID, err := uc.scope(ctx)
	if err != nil {
		return nil, err
	}
	rows, err := uc.repo.ListAgentSessionsByIDs(ctx, ids, userID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return rows, nil
}

// ArchiveSessions 归档或取消归档。还在运行、等待批准的不归档；归档的会话不留进程。
func (uc *AgentUsecase) ArchiveSessions(ctx context.Context, ids []string, archived bool) (changed, skipped int, err error) {
	rows, err := uc.ownSessions(ctx, ids)
	if err != nil {
		return 0, 0, err
	}
	own := make([]string, 0, len(rows))
	for _, s := range rows {
		own = append(own, s.ID)
	}
	updated, err := uc.repo.ArchiveAgentSessions(ctx, own, archived)
	if err != nil {
		return 0, 0, ErrSystem(err)
	}
	for _, s := range updated {
		if l := uc.live(s.ID); archived && l != nil {
			l.Shutdown()
		}
		uc.pushSession(ctx, s)
	}
	return len(updated), len(own) - len(updated), nil
}

func (uc *AgentUsecase) DeleteSessions(ctx context.Context, ids []string, withForks bool) (int, error) {
	rows, err := uc.ownSessions(ctx, ids)
	if err != nil {
		return 0, err
	}
	return uc.deleteSessions(ctx, rows, withForks)
}

// PurgeSessions 删除全部归档会话：有管理权限的人删所有人的，其他人只删自己的。
func (uc *AgentUsecase) PurgeSessions(ctx context.Context) (int, error) {
	userID, err := uc.scope(ctx)
	if err != nil {
		return 0, err
	}
	rows, err := uc.repo.ListArchivedAgentSessions(ctx, userID)
	if err != nil {
		return 0, ErrSystem(err)
	}
	return uc.deleteSessions(ctx, rows, false)
}

// deleteSessions 删除会话和它们的对话记录（Momoko 库里的，和 Agent 在它配置目录里按会话 id 存的），
// 连同整个会话目录（工作目录、工作树、附件、临时文件）；
// 工作树的分支已经合并过的一起删，没合并的提交留在分支上。工作区目录本身是用户的，不动。
// withForks 为 true 时连同复刻出来的会话一起删，否则把它们移到外层留着。
func (uc *AgentUsecase) deleteSessions(ctx context.Context, rows []*gen.AgentSession, withForks bool) (int, error) {
	bg := context.WithoutCancel(ctx)
	seen := map[string]bool{}
	ids := make([]string, 0, len(rows))
	for _, s := range rows {
		seen[s.ID] = true
		ids = append(ids, s.ID)
	}
	if withForks {
		for frontier := ids; len(frontier) > 0; {
			forks, err := uc.repo.ListAgentForks(bg, frontier)
			if err != nil {
				return 0, ErrSystem(err)
			}
			frontier = nil
			for _, f := range forks {
				if !seen[f.ID] {
					seen[f.ID] = true
					rows, ids, frontier = append(rows, f), append(ids, f.ID), append(frontier, f.ID)
				}
			}
		}
	} else if err := uc.repo.DetachAgentForks(bg, ids); err != nil {
		return 0, ErrSystem(err)
	}
	var stopped []*agentLive
	for _, s := range rows {
		if l := uc.live(s.ID); l != nil {
			// 先标记再结束进程：收尾时引擎还会报几条事件上来，不能再写进库里。
			l.dead.Store(true)
			l.Shutdown()
			stopped = append(stopped, l)
		}
	}
	if err := uc.repo.DeleteAgentSessions(bg, ids); err != nil {
		return 0, ErrSystem(err)
	}
	// 进程还占着目录时删不掉（Windows），等它们退出再删。
	for _, l := range stopped {
		l.session.Process().Wait(5 * time.Second)
	}
	workspaces := uc.loadNames(bg).workspaces
	drivers := map[string]agentdriver.Driver{}
	for _, s := range rows {
		if s.AcpSessionID != "" {
			d, ok := drivers[s.RuntimeID]
			if !ok {
				if rt, err := uc.getRuntime(bg, s.RuntimeID); err == nil {
					d = uc.driverOf(string(rt.Kind))
				}
				drivers[s.RuntimeID] = d
			}
			if d != nil {
				d.RemoveTranscript(s.Cwd, s.AcpSessionID)
			}
		}
		if s.Worktree != "" {
			if w := workspaces[s.WorkspaceID]; w != nil {
				agentgit.RemoveWorktree(bg, w.Path, &agentgit.Worktree{Name: s.Worktree, Path: s.Cwd}, false)
			}
		}
		agentfs.RemoveAll(uc.sessionDir(s.ID))
		if raw, err := json.Marshal(map[string]any{"t": "gone", "id": s.ID}); err == nil {
			uc.hub.ToSessionOrUser(s.ID, s.UserID, raw)
		}
	}
	return len(ids), nil
}

// itemsOf 存库的事件给页面的样子。settle 为 true 时（没有进程在跑）把中断时停在半路的事件收个尾；
// 否则原样给存库的 JSON，不用解开再编码。
func itemsOf(rows []*gen.AgentEvent, settle bool) []json.RawMessage {
	out := make([]json.RawMessage, 0, len(rows))
	for _, row := range rows {
		if !settle {
			out = append(out, json.RawMessage(row.Payload))
			continue
		}
		item := agentrun.Decode(row.Seq, row.Rev, row.Type, row.Sub, row.Payload)
		item.Settle()
		out = append(out, json.RawMessage(item.Payload()))
	}
	return out
}

// ExportSessions 导出记录：返回会话和它的全部事件，排版成文件是页面的事。
func (uc *AgentUsecase) ExportSessions(ctx context.Context, ids []string) ([]*v1.AgentSessionExport, error) {
	rows, err := uc.ownSessions(ctx, ids)
	if err != nil {
		return nil, err
	}
	names := uc.loadNames(ctx)
	out := make([]*v1.AgentSessionExport, 0, len(rows))
	for _, s := range rows {
		events, err := uc.repo.ListAgentEventsAfter(ctx, s.ID, 0)
		if err != nil {
			return nil, ErrSystem(err)
		}
		out = append(out, &v1.AgentSessionExport{Session: uc.sessionPB(s, names), Events: agentJSON(itemsOf(events, uc.live(s.ID) == nil))})
	}
	return out, nil
}

// ———————————————————————————— 改动和文件 ————————————————————————————

// GetChanges 会话目录相对当前分支的未提交改动。
func (uc *AgentUsecase) GetChanges(ctx context.Context, id string) (*v1.GetAgentSessionChangesResponse, error) {
	s, err := uc.getSession(ctx, id)
	if err != nil {
		return nil, err
	}
	resp := &v1.GetAgentSessionChangesResponse{}
	if resp.IsGit, resp.Branch = agentgit.Info(s.Cwd); !resp.IsGit {
		return resp, nil
	}
	for _, c := range agentgit.Changes(ctx, s.Cwd) {
		resp.Files = append(resp.Files, &v1.AgentChangedFile{Path: c.Path, Kind: c.Kind, Added: int32(c.Added), Deleted: int32(c.Deleted), Patch: c.Patch})
	}
	return resp, nil
}

func (uc *AgentUsecase) RevertFile(ctx context.Context, id, path string) error {
	s, err := uc.getSession(ctx, id)
	if err != nil {
		return err
	}
	return agentErr(agentgit.Revert(ctx, s.Cwd, path))
}

// ListFiles 会话目录里某一层的文件，带上未提交改动的标记（目录里有改动的也标上）。只让 git 看这一层。
func (uc *AgentUsecase) ListFiles(ctx context.Context, id, dir string) ([]*v1.AgentFileEntry, error) {
	s, err := uc.getSession(ctx, id)
	if err != nil {
		return nil, err
	}
	entries, err := agentfs.List(s.Cwd, dir)
	if err != nil {
		return nil, agentErr(err)
	}
	out := agentEntriesPB(entries)
	marks := agentgit.Marks(ctx, s.Cwd, dir)
	for _, e := range out {
		e.Change = marks[e.Path]
	}
	return out, nil
}

func (uc *AgentUsecase) ReadFile(ctx context.Context, id, path string) (*v1.ReadAgentSessionFileResponse, error) {
	s, err := uc.getSession(ctx, id)
	if err != nil {
		return nil, err
	}
	text, err := agentfs.ReadText(s.Cwd, path, agentFileMax)
	if err != nil {
		return nil, agentErr(err)
	}
	return &v1.ReadAgentSessionFileResponse{Path: path, Content: text.Content, Binary: text.Binary, TooLarge: text.TooLarge, Size: text.Size}, nil
}

// ReadUpload 取回会话附件目录里的一个文件（file 是事件里记的那个：消息的附件，或工具调用产出的图片），页面拿来显示图片。
func (uc *AgentUsecase) ReadUpload(ctx context.Context, id, file string) (*v1.GetAgentSessionUploadResponse, error) {
	if _, err := uc.getSession(ctx, id); err != nil {
		return nil, err
	}
	data, kind, err := agentfs.ReadBlob(uc.sessionDir(id, "uploads"), file, agentUploadMax)
	if err != nil {
		return nil, agentErr(err)
	}
	return &v1.GetAgentSessionUploadResponse{Mime: kind, Data: data}, nil
}

// SearchFiles @ 引用文件时按文件名搜：有会话搜会话目录，新会话还没有会话时搜选中的工作区。
// 目录的文件列表缓存一小段时间，连着敲字不会每次都重新列。
func (uc *AgentUsecase) SearchFiles(ctx context.Context, sessionID, workspaceID, keyword string) ([]string, error) {
	dir := ""
	switch {
	case sessionID != "":
		s, err := uc.getSession(ctx, sessionID)
		if err != nil {
			return nil, err
		}
		dir = s.Cwd
	case workspaceID != "":
		w, err := uc.getWorkspace(ctx, workspaceID)
		if err != nil {
			return nil, err
		}
		dir = w.Path
	default:
		return nil, nil
	}
	return uc.files.Search(ctx, dir, keyword, 50), nil
}

// ———————————————————————————— 页面的长连接 ————————————————————————————

// Connect 页面连上来：之后这个人的会话状态变化都会推给它。
func (uc *AgentUsecase) Connect(ctx context.Context) (*agenthub.Conn, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	return uc.hub.Add(userID, uc.canManage(ctx)), nil
}

func (uc *AgentUsecase) Disconnect(c *agenthub.Conn) {
	uc.hub.Remove(c)
}

// CloseStream 页面离开了它开着的会话。
func (uc *AgentUsecase) CloseStream(c *agenthub.Conn) {
	uc.hub.Subscribe(c, "")
}

// WatchAll 管理页关注（或不再关注）所有人的会话状态变化；只对有管理权限的连接生效。
func (uc *AgentUsecase) WatchAll(c *agenthub.Conn, on bool) {
	uc.hub.Watch(c, on)
}

func streamReply(c *agenthub.Conn, msg map[string]any) {
	if raw, err := json.Marshal(msg); err == nil {
		c.Push(raw)
	}
}

// streamSession 连接能看的会话；看不了时回一条带错误的消息。
func (uc *AgentUsecase) streamSession(ctx context.Context, c *agenthub.Conn, t, id string) *gen.AgentSession {
	s, err := uc.repo.GetAgentSession(ctx, id)
	if err != nil || (s.UserID != c.UserID && !c.Manage) {
		streamReply(c, map[string]any{"t": t, "id": id, "error": ErrAgentSessionNotFound.Error()})
		return nil
	}
	return s
}

// OpenStream 页面打开一个会话：订阅它之后的事件，并把页面还没有的事件补给它。
// rev 为 0 时给最近的一页（往上翻再要更早的），否则给修订号大于 rev 的（断线重连）。
// open 消息在订阅的同时排进连接的发送队列，排在它之后推来的增量一条不重、一条不漏。
func (uc *AgentUsecase) OpenStream(c *agenthub.Conn, id string, rev int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	s := uc.streamSession(ctx, c, "open", id)
	if s == nil {
		return
	}
	session, _ := agentJSONOptions.Marshal(uc.sessionPB(s, uc.loadNames(ctx)))
	detail, _ := agentJSONOptions.Marshal(uc.sessionDetail(ctx, s))
	send := func(state agentrun.State, used, size int64, settle bool) {
		uc.hub.Subscribe(c, id)
		var (
			rows []*gen.AgentEvent
			more bool
			err  error
		)
		if rev > 0 {
			rows, err = uc.repo.ListAgentEventsAfter(ctx, id, rev)
		} else {
			rows, more, err = uc.repo.ListAgentEventsBefore(ctx, id, 0, agentPageItems)
		}
		var last int64
		if err == nil {
			_, last, err = uc.repo.MaxAgentEvent(ctx, id)
		}
		if err != nil {
			streamReply(c, map[string]any{"t": "open", "id": id, "error": "读取对话记录失败"})
			return
		}
		streamReply(c, map[string]any{
			"t":           "open",
			"id":          id,
			"session":     json.RawMessage(session),
			"detail":      json.RawMessage(detail),
			"state":       state,
			"items":       itemsOf(rows, settle),
			"more":        more,
			"rev":         last,
			"contextUsed": used,
			"contextSize": size,
		})
	}
	if l := uc.live(id); l != nil && l.session != nil {
		// 在引擎的锁里订阅、读库、排队：正在写的文字先报一次、落库队列写完，之后的增量都排在 open 后面。
		l.session.Sync(func(state agentrun.State, used, size int64) {
			l.w.flush()
			send(state, used, size, false)
		})
		return
	}
	send(agentParse[agentrun.State](s.State), s.ContextUsed, s.ContextSize, true)
}

// MoreStream 页面往上翻：给序号小于 before 的上一页事件。
func (uc *AgentUsecase) MoreStream(c *agenthub.Conn, id string, before int64) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	if uc.streamSession(ctx, c, "more", id) == nil {
		return
	}
	rows, more, err := uc.repo.ListAgentEventsBefore(ctx, id, before, agentPageItems)
	if err != nil {
		streamReply(c, map[string]any{"t": "more", "id": id, "error": "读取对话记录失败"})
		return
	}
	streamReply(c, map[string]any{"t": "more", "id": id, "items": itemsOf(rows, uc.live(id) == nil), "more": more})
}
