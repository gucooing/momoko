package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	entschema "momoko/internal/data/ent/schema"
	"momoko/pkg/mgp"
	"momoko/pkg/response"
)

var (
	ErrGmCommandNotFound   = response.BadRequest(404, "这台服务器没有这条指令")
	ErrGmTargets           = response.BadRequest(400, "指令目标不正确")
	ErrGmConfirmRequired   = response.BadRequest(400, "危险指令需要输入玩家昵称确认")
	ErrGmConfirmMismatch   = response.BadRequest(400, "输入的昵称与玩家当前昵称不一致")
	ErrGmExecutionNotFound = response.BadRequest(404, "GM 记录不存在")
	ErrGmNotPending        = response.BadRequest(400, "这条申请已经处理过了")
	ErrGmSelfServiceClosed = response.BadRequest(403, "这项功能没有开放")
	ErrGmSelfServiceLimit  = response.BadRequest(429, "今天的次数已用完")
	ErrGmSelfServiceWait   = response.BadRequest(429, "操作太频繁，请稍后再试")
	ErrGmSelfServicePend   = response.BadRequest(409, "已有一条申请在等待审批")
	ErrGmConfigConflict    = response.BadRequest(409, "服务端的配置已被修改，请刷新后再发布")
	ErrGmConfigInvalidJSON = response.BadRequest(400, "配置内容不是有效的 JSON")
	ErrGmConfigNoDraft     = response.BadRequest(400, "没有可发布的草稿")
)

// GmActor 执行 GM 指令的人。
type GmActor struct {
	UserID string
	Name   string
	// Kind admin 管理员 / player 玩家自助
	Kind string
	// Channel admin Momoko 后台 / passport 通行证 / game 游戏内
	Channel  string
	IP       string
	Platform string
}

// GmCommandInput 一次指令执行的输入。
type GmCommandInput struct {
	Targets     []string
	Params      json.RawMessage
	Reason      string
	DryRun      bool
	ConfirmName string
	Summary     map[string]string
	Source      string
}

// GmLink 绑定了通行证（passport_users）的游戏账号（GM 页面展示用）。
type GmLink struct {
	PID            string
	PassportUserID string
	Username       string
	Name           string
	Method         string
	Nickname       string
	LinkedAt       time.Time
}

// GmExecutionFilter GM 记录筛选。
type GmExecutionFilter struct {
	ServerID      string
	ActorKind     string
	ActorID       string
	Statuses      []string
	Risk          string
	Command       string
	PID           string
	Q             string
	Since, Until  *time.Time
	IncludeDryRun bool
	Page, Size    int
}

// GmRepo GM 运营在 Momoko 侧的持久化：执行记录、玩家自助、配置版本与绑定查询。
type GmRepo interface {
	CreateExecution(ctx context.Context, e *gen.GmExecution) (*gen.GmExecution, error)
	SaveExecution(ctx context.Context, e *gen.GmExecution) (*gen.GmExecution, error)
	GetExecution(ctx context.Context, id string) (*gen.GmExecution, error)
	ListExecutions(ctx context.Context, f GmExecutionFilter) ([]*gen.GmExecution, int, error)
	// CountExecutions 统计时间范围内（不含试运行）的 all / admin / player / pending / failed。
	CountExecutions(ctx context.Context, serverID string, since, until *time.Time) (map[string]int32, error)
	ListPending(ctx context.Context, serverID string, limit int) ([]*gen.GmExecution, error)
	CountPending(ctx context.Context, serverID string) (int, error)
	CountByTarget(ctx context.Context, serverID, pid string) (int, error)
	// CountUses 统计某执行者某指令在 since 之后有效（成功、部分成功、执行中、待审批）的次数。
	CountUses(ctx context.Context, serverID, actorID, command string, since time.Time) (int, error)
	// LastUse 返回某执行者某指令最近一次有效执行的时间。
	LastUse(ctx context.Context, serverID, actorID, command string) (*gen.GmExecution, error)
	// PriorUses 统计某玩家某指令此前成功执行的次数。
	PriorUses(ctx context.Context, serverID, command, pid string) (int, error)

	ListSelfService(ctx context.Context, serverID string) ([]*gen.GmSelfService, error)
	GetSelfService(ctx context.Context, serverID, command string) (*gen.GmSelfService, error)
	SaveSelfService(ctx context.Context, p *gen.GmSelfService) (*gen.GmSelfService, error)

	GetConfigDraft(ctx context.Context, serverID, domain string) (*gen.GameConfigRevision, error)
	SaveConfigDraft(ctx context.Context, r *gen.GameConfigRevision) (*gen.GameConfigRevision, error)
	DeleteConfigDraft(ctx context.Context, serverID, domain string) error
	// PublishConfig 写入一条已发布版本（seq 递增）并删除草稿。
	PublishConfig(ctx context.Context, r *gen.GameConfigRevision) (*gen.GameConfigRevision, error)
	ListConfigRevisions(ctx context.Context, serverID, domain string, page, size int) ([]*gen.GameConfigRevision, int, error)
	GetConfigRevision(ctx context.Context, id string) (*gen.GameConfigRevision, error)
	LastPublished(ctx context.Context, serverID string) (map[string]*gen.GameConfigRevision, error)

	LinksByPIDs(ctx context.Context, serverID string, pids []string) ([]*GmLink, error)
	LinkedPIDs(ctx context.Context, serverID string) (map[string]struct{}, error)
	// FindLinksByUser 按通行证用户名 / 昵称查找绑定。
	FindLinksByUser(ctx context.Context, serverID, q string, limit int) ([]*GmLink, error)
}

// GmUsecase GM 运营：指令执行与审计、审批、玩家自助规则、玩家与配置。
type GmUsecase struct {
	servers *GameServerUsecase
	repo    GmRepo
	users   UserRepo
	log     *log.Helper

	// actorLocks 串行化同一执行者同一指令的自助执行，避免并发绕过次数限制。
	actorLocks sync.Map
}

func NewGmUsecase(servers *GameServerUsecase, repo GmRepo, users UserRepo, logger log.Logger) *GmUsecase {
	return &GmUsecase{
		servers: servers,
		repo:    repo,
		users:   users,
		log:     log.NewHelper(log.With(logger, "module", "biz/game_gm")),
	}
}

// Servers 返回游戏服务器用例。
func (uc *GmUsecase) Servers() *GameServerUsecase { return uc.servers }

// Repo 返回 GM 持久化（邮件、兑换码、通行证等同属 GM 的用例使用）。
func (uc *GmUsecase) Repo() GmRepo { return uc.repo }

// AdminActor 由当前登录的管理员生成执行者。
func (uc *GmUsecase) AdminActor(ctx context.Context, userID, ip, platform string) GmActor {
	a := GmActor{UserID: userID, Kind: "admin", Channel: "admin", IP: ip, Platform: platform}
	if u, err := uc.users.FindByID(ctx, userID); err == nil {
		a.Name = u.Name
		if a.Name == "" {
			a.Name = u.Username
		}
	}
	return a
}

// Context 返回 GM 页面上下文，并记入「最近查看」。
func (uc *GmUsecase) Context(ctx context.Context, userID, serverID string) (*v1.GetGmContextResponse, error) {
	s, err := uc.servers.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}
	uc.servers.TouchRecent(ctx, userID, serverID, GameRecentScopeAdmin)
	out := &v1.GetGmContextResponse{Timezone: uc.servers.Location(s).String()}
	if MGPConfigured(s) {
		if m, err := uc.servers.Manifest(ctx, s); err == nil {
			out.Manifest = JSONValue(m.Raw)
		}
		if st, err := uc.servers.Status(ctx, s); err == nil {
			out.Status = JSONValue(st.Raw)
		} else {
			out.StatusError = err.Error()
		}
	}
	out.Server = uc.servers.Brief(ctx, s)
	if n, err := uc.repo.CountPending(ctx, serverID); err == nil {
		out.PendingApprovals = int32(n)
	}
	if list, err := uc.repo.ListSelfService(ctx, serverID); err == nil {
		for _, p := range list {
			if p.Enabled {
				out.SelfServiceEnabled++
			}
		}
	}
	if linked, err := uc.repo.LinkedPIDs(ctx, serverID); err == nil {
		out.LinkedAccounts = int32(len(linked))
	}
	return out, nil
}

// pidOf 读出 MGP 玩家条目的 pid。
func pidOf(item json.RawMessage) string {
	var p struct {
		PID string `json:"pid"`
	}
	_ = json.Unmarshal(item, &p)
	return p.PID
}

// ListPlayers 玩家列表：MGP 搜索与筛选 + 通行证绑定；unlinked 由 Momoko 排除已绑定的玩家。
func (uc *GmUsecase) ListPlayers(ctx context.Context, req *v1.ListGmPlayersRequest) (*v1.ListGmPlayersResponse, error) {
	s, client, err := uc.servers.ClientOf(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	limit := int(req.Limit)
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	linked, err := uc.repo.LinkedPIDs(ctx, s.ID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListGmPlayersResponse{Summary: map[string]int32{}, LinkedCount: int32(len(linked))}
	var items []json.RawMessage
	if req.Filter == "unlinked" {
		offset, _ := strconv.Atoi(req.Cursor)
		skipped, cursor := 0, ""
		var summary json.RawMessage
		for {
			q := url.Values{"filter": {"all"}, "limit": {"200"}}
			if req.Q != "" {
				q.Set("q", req.Q)
			}
			if cursor != "" {
				q.Set("cursor", cursor)
			}
			var page mgp.Page
			if err := client.GetInto(ctx, "/players", q, &page); err != nil {
				return nil, GameMGPError(err)
			}
			if summary == nil {
				summary = page.Summary
			}
			var batch []json.RawMessage
			_ = json.Unmarshal(page.Items, &batch)
			for _, it := range batch {
				if _, ok := linked[pidOf(it)]; ok {
					continue
				}
				out.Total++
				if skipped < offset {
					skipped++
					continue
				}
				if len(items) < limit {
					items = append(items, it)
				}
			}
			if page.NextCursor == "" {
				break
			}
			cursor = page.NextCursor
		}
		if offset+len(items) < int(out.Total) {
			out.NextCursor = strconv.Itoa(offset + len(items))
		}
		uc.fillSummary(out, summary, len(linked))
	} else {
		q := url.Values{"limit": {strconv.Itoa(limit)}}
		if req.Filter != "" {
			q.Set("filter", req.Filter)
		}
		if req.Q != "" {
			q.Set("q", req.Q)
		}
		if req.Cursor != "" {
			q.Set("cursor", req.Cursor)
		}
		var page mgp.Page
		if err := client.GetInto(ctx, "/players", q, &page); err != nil {
			return nil, GameMGPError(err)
		}
		_ = json.Unmarshal(page.Items, &items)
		out.NextCursor, out.Total = page.NextCursor, int32(page.Total)
		uc.fillSummary(out, page.Summary, len(linked))
		// 按通行证用户名搜索：MGP 不认识通行证，第一页补上绑定了匹配用户的玩家。
		if req.Q != "" && req.Cursor == "" && (req.Filter == "" || req.Filter == "all") {
			have := map[string]struct{}{}
			for _, it := range items {
				have[pidOf(it)] = struct{}{}
			}
			if links, err := uc.repo.FindLinksByUser(ctx, s.ID, req.Q, 10); err == nil {
				for _, l := range links {
					if _, ok := have[l.PID]; ok || l.PID == "" {
						continue
					}
					raw, err := client.Get(ctx, "/players/"+url.PathEscape(l.PID), nil)
					if err != nil {
						continue
					}
					items = append([]json.RawMessage{raw}, items...)
					out.Total++
				}
			}
		}
	}
	if items == nil {
		items = []json.RawMessage{}
	}
	raw, _ := json.Marshal(items)
	out.Items = JSONValue(raw)
	pids := make([]string, 0, len(items))
	for _, it := range items {
		pids = append(pids, pidOf(it))
	}
	links, err := uc.repo.LinksByPIDs(ctx, s.ID, pids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	for _, l := range links {
		out.Links = append(out.Links, gmLinkProto(l))
	}
	return out, nil
}

func (uc *GmUsecase) fillSummary(out *v1.ListGmPlayersResponse, raw json.RawMessage, linked int) {
	var summary map[string]int32
	_ = json.Unmarshal(raw, &summary)
	for k, v := range summary {
		out.Summary[k] = v
	}
	if all, ok := out.Summary["all"]; ok {
		out.Summary["unlinked"] = max(all-int32(linked), 0)
	}
}

func gmLinkProto(l *GmLink) *v1.GmPlayerLink {
	return &v1.GmPlayerLink{
		Pid:            l.PID,
		PassportUserId: l.PassportUserID,
		Username:       l.Username,
		Name:           l.Name,
		Method:         l.Method,
		LinkedAt:       timestamppb.New(l.LinkedAt),
	}
}

// GetPlayer 玩家摘要 + 通行证绑定 + GM 记录数。
func (uc *GmUsecase) GetPlayer(ctx context.Context, serverID, pid string) (*v1.GetGmPlayerResponse, error) {
	s, client, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, err
	}
	raw, err := client.Get(ctx, "/players/"+url.PathEscape(pid), nil)
	if err != nil {
		return nil, GameMGPError(err)
	}
	out := &v1.GetGmPlayerResponse{Player: JSONValue(raw)}
	if links, err := uc.repo.LinksByPIDs(ctx, s.ID, []string{pid}); err == nil && len(links) > 0 {
		out.Link = gmLinkProto(links[0])
	}
	if n, err := uc.repo.CountByTarget(ctx, s.ID, pid); err == nil {
		out.GmExecutions = int32(n)
	}
	return out, nil
}

// Proxy 透传一次 MGP GET（视图、报表、目录）。
func (uc *GmUsecase) Proxy(ctx context.Context, serverID, path, query string) (json.RawMessage, error) {
	_, client, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, err
	}
	raw, err := client.Get(ctx, path, MGPQuery(query))
	if err != nil {
		return nil, GameMGPError(err)
	}
	return raw, nil
}

// targetsValid 按指令目标类型校验目标数量。
func targetsValid(cmd *mgp.Command, targets []string) bool {
	switch cmd.Target {
	case mgp.TargetServer:
		return len(targets) == 0
	case mgp.TargetPlayer:
		return len(targets) == 1 && targets[0] != ""
	default:
		if len(targets) == 0 || len(targets) > mgp.MaxTargets {
			return false
		}
		for _, t := range targets {
			if t == "" {
				return false
			}
		}
		return true
	}
}

// playerName 读取玩家当前昵称（危险指令确认用）。
func playerName(ctx context.Context, client *mgp.Client, pid string) (string, error) {
	var p mgp.Player
	if err := client.GetInto(ctx, "/players/"+url.PathEscape(pid), nil, &p); err != nil {
		return "", err
	}
	return p.Name, nil
}

// Execute 执行一条 GM 指令并写 GM 记录。管理员与玩家自助（经 ExecuteSelfService）共用。
func (uc *GmUsecase) Execute(ctx context.Context, actor GmActor, serverID, command string, in GmCommandInput) (*gen.GmExecution, json.RawMessage, error) {
	s, client, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	m, err := uc.servers.Manifest(ctx, s)
	if err != nil {
		return nil, nil, err
	}
	cmd, ok := m.Command(command)
	if !ok {
		return nil, nil, ErrGmCommandNotFound
	}
	if !targetsValid(cmd, in.Targets) {
		return nil, nil, ErrGmTargets
	}
	in.Reason = strings.TrimSpace(in.Reason)
	confirmed := false
	if !in.DryRun && actor.Kind == "admin" {
		if cmd.Risk == mgp.RiskDanger {
			if strings.TrimSpace(in.ConfirmName) == "" {
				return nil, nil, ErrGmConfirmRequired
			}
			for _, pid := range in.Targets {
				name, err := playerName(ctx, client, pid)
				if err != nil {
					return nil, nil, GameMGPError(err)
				}
				if name != strings.TrimSpace(in.ConfirmName) {
					return nil, nil, ErrGmConfirmMismatch
				}
			}
			confirmed = true
		}
	}
	e := uc.newExecution(s, cmd, actor, in)
	e.Confirmed = confirmed
	e.Status = "running"
	e, err = uc.repo.CreateExecution(ctx, e)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	return uc.run(ctx, client, cmd, e, in.Params)
}

func (uc *GmUsecase) newExecution(s *gen.GameServer, cmd *mgp.Command, actor GmActor, in GmCommandInput) *gen.GmExecution {
	params := in.Params
	if len(params) == 0 {
		params = json.RawMessage("{}")
	}
	targets := make([]entschema.GmTarget, 0, len(in.Targets))
	for _, pid := range in.Targets {
		targets = append(targets, entschema.GmTarget{PID: pid})
	}
	return &gen.GmExecution{
		ID:           uuid.NewString(),
		ServerID:     s.ID,
		Command:      cmd.ID,
		CommandTitle: cmd.Title,
		CommandGroup: cmd.Group,
		Risk:         cmd.Risk,
		TargetKind:   cmd.Target,
		ActorID:      actor.UserID,
		ActorName:    actor.Name,
		ActorKind:    gmActorKind(actor.Kind),
		Channel:      gmChannel(actor.Channel),
		ClientIP:     actor.IP,
		Platform:     actor.Platform,
		Targets:      targets,
		TargetPids:   in.Targets,
		Params:       string(params),
		Summary:      in.Summary,
		Reason:       in.Reason,
		DryRun:       in.DryRun,
		RequestID:    "momoko-" + uuid.NewString(),
		Source:       in.Source,
	}
}

// mgpReason MGP 对 high / danger 指令要求带原因（写进游戏服务端自己的日志）。管理端不再填写原因，
// 由 Momoko 按「谁执行了哪条指令、来自哪里」生成；玩家自助申请带的是玩家自己写的理由。
func mgpReason(e *gen.GmExecution) string {
	if e.Reason != "" {
		return e.Reason
	}
	who := e.ActorName
	if who == "" {
		who = "Momoko"
	}
	title := e.CommandTitle
	if title == "" {
		title = e.Command
	}
	if e.Source != "" {
		return fmt.Sprintf("Momoko · %s · %s（%s）", who, title, e.Source)
	}
	return fmt.Sprintf("Momoko · %s · %s", who, title)
}

// run 调用 MGP 执行并把结果写回记录。
func (uc *GmUsecase) run(ctx context.Context, client *mgp.Client, cmd *mgp.Command, e *gen.GmExecution, params json.RawMessage) (*gen.GmExecution, json.RawMessage, error) {
	if len(params) == 0 {
		params = json.RawMessage(e.Params)
	}
	body := mgp.CommandRequest{
		Targets:   e.TargetPids,
		Params:    params,
		Reason:    mgpReason(e),
		RequestID: e.RequestID,
		DryRun:    e.DryRun,
		Operator:  e.ActorName,
	}
	if body.Targets == nil {
		body.Targets = []string{}
	}
	start := time.Now()
	raw, err := client.Do(ctx, mgp.Call{
		Method:    "POST",
		Path:      "/commands/" + url.PathEscape(cmd.ID),
		Body:      body,
		RequestID: e.RequestID,
		Timeout:   mgp.CommandTimeout,
	})
	now := time.Now()
	e.DurationMs, e.FinishedAt = int(now.Sub(start).Milliseconds()), &now
	if err != nil {
		e.Status = "failed"
		e.ErrorCode = mgp.CodeOf(err)
		e.ErrorMessage = err.Error()
		if me, ok := err.(*mgp.Error); ok {
			e.ErrorMessage = me.Message
		}
		e.FailedCount = len(e.TargetPids)
		saved, saveErr := uc.repo.SaveExecution(context.WithoutCancel(ctx), e)
		if saveErr != nil {
			uc.log.Errorf("save gm execution %s: %v", e.ID, saveErr)
			saved = e
		}
		return saved, nil, nil
	}
	var result mgp.CommandResult
	_ = json.Unmarshal(raw, &result)
	e.Result = string(raw)
	byPID := map[string]string{}
	var backups []string
	for _, r := range result.Results {
		byPID[r.PID] = r.Name
		if r.Status == "success" {
			e.SuccessCount++
		} else {
			e.FailedCount++
			if e.ErrorCode == "" {
				e.ErrorCode, e.ErrorMessage = r.Code, r.Message
			}
		}
		if r.BackupID != "" && r.BackupID != "0" {
			backups = append(backups, r.BackupID.String())
		}
	}
	e.BackupRef = strings.Join(backups, ",")
	for i, t := range e.Targets {
		if name, ok := byPID[t.PID]; ok && name != "" {
			e.Targets[i].Name = name
		}
	}
	switch {
	case len(result.Results) == 0 || e.FailedCount == 0:
		e.Status = "success"
	case e.SuccessCount == 0:
		e.Status = "failed"
	default:
		e.Status = "partial"
	}
	saved, err := uc.repo.SaveExecution(context.WithoutCancel(ctx), e)
	if err != nil {
		uc.log.Errorf("save gm execution %s: %v", e.ID, err)
		saved = e
	}
	return saved, raw, nil
}

// ListExecutions GM 记录列表、时间范围内的计数与全部待审批。
func (uc *GmUsecase) ListExecutions(ctx context.Context, req *v1.ListGmExecutionsRequest) (*v1.ListGmExecutionsResponse, error) {
	if _, err := uc.servers.Get(ctx, req.ServerId); err != nil {
		return nil, err
	}
	f := GmExecutionFilter{
		ServerID:      req.ServerId,
		ActorKind:     req.ActorKind,
		ActorID:       req.ActorId,
		Risk:          req.Risk,
		Command:       req.Command,
		PID:           req.Pid,
		Q:             strings.TrimSpace(req.Q),
		IncludeDryRun: req.IncludeDryRun,
		Page:          int(req.Page),
		Size:          int(req.PageSize),
	}
	switch req.Status {
	case "pending":
		f.Statuses = []string{"pending"}
	case "failed":
		f.Statuses = []string{"failed", "partial", "rejected"}
	case "success":
		f.Statuses = []string{"success"}
	case "":
	default:
		f.Statuses = []string{req.Status}
	}
	if req.Since != nil {
		t := req.Since.AsTime()
		f.Since = &t
	}
	if req.Until != nil {
		t := req.Until.AsTime()
		f.Until = &t
	}
	items, total, err := uc.repo.ListExecutions(ctx, f)
	if err != nil {
		return nil, ErrSystem(err)
	}
	counts, err := uc.repo.CountExecutions(ctx, req.ServerId, f.Since, f.Until)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListGmExecutionsResponse{Total: int32(total), Counts: counts}
	for _, e := range items {
		out.Items = append(out.Items, GmExecutionProto(e, false))
	}
	if req.ActorId == "" {
		pending, err := uc.repo.ListPending(ctx, req.ServerId, 20)
		if err == nil {
			for _, e := range pending {
				p := GmExecutionProto(e, false)
				if len(e.TargetPids) == 1 {
					if n, err := uc.repo.PriorUses(ctx, e.ServerID, e.Command, e.TargetPids[0]); err == nil {
						p.PriorUses = int32(n)
					}
				}
				out.Pending = append(out.Pending, p)
			}
		}
	}
	return out, nil
}

// GmExecutionProto 转为接口结构（从不包含请求参数）；withResult 时带上逐个对象的结果。
func GmExecutionProto(e *gen.GmExecution, withResult bool) *v1.GmExecution {
	out := &v1.GmExecution{
		Id:           e.ID,
		ServerId:     e.ServerID,
		Command:      e.Command,
		CommandTitle: e.CommandTitle,
		CommandGroup: e.CommandGroup,
		Risk:         e.Risk,
		TargetKind:   e.TargetKind,
		ActorId:      e.ActorID,
		ActorName:    e.ActorName,
		ActorKind:    string(e.ActorKind),
		Channel:      string(e.Channel),
		ClientIp:     e.ClientIP,
		Platform:     e.Platform,
		Summary:      e.Summary,
		Reason:       e.Reason,
		Confirmed:    e.Confirmed,
		DryRun:       e.DryRun,
		Status:       string(e.Status),
		SuccessCount: int32(e.SuccessCount),
		FailedCount:  int32(e.FailedCount),
		BackupRef:    e.BackupRef,
		ErrorCode:    e.ErrorCode,
		ErrorMessage: e.ErrorMessage,
		RequestId:    e.RequestID,
		Source:       e.Source,
		ReviewedBy:   e.ReviewedBy,
		ReviewerName: e.ReviewerName,
		ReviewNote:   e.ReviewNote,
		CreateTime:   timestamppb.New(e.CreateTime),
		DurationMs:   int32(e.DurationMs),
	}
	if out.Summary == nil {
		out.Summary = map[string]string{}
	}
	for _, t := range e.Targets {
		out.Targets = append(out.Targets, &v1.GameTarget{Pid: t.PID, Name: t.Name, DisplayId: t.DisplayID})
	}
	if e.ReviewedAt != nil {
		out.ReviewedAt = timestamppb.New(*e.ReviewedAt)
	}
	if e.FinishedAt != nil {
		out.FinishedAt = timestamppb.New(*e.FinishedAt)
	}
	if withResult && e.Result != "" {
		// 只透传结果里的 results / summary，不含任何请求参数。
		var r struct {
			Results json.RawMessage `json:"results"`
			Summary json.RawMessage `json:"summary"`
		}
		if json.Unmarshal([]byte(e.Result), &r) == nil {
			raw, _ := json.Marshal(r)
			out.Result = JSONValue(raw)
		}
	}
	return out
}

// GetExecution GM 记录详情（不含请求参数）。
func (uc *GmUsecase) GetExecution(ctx context.Context, serverID, id string) (*v1.GmExecution, error) {
	e, err := uc.repo.GetExecution(ctx, id)
	if gen.IsNotFound(err) || (err == nil && e.ServerID != serverID) {
		return nil, ErrGmExecutionNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := GmExecutionProto(e, true)
	if e.Status == "pending" && len(e.TargetPids) == 1 {
		if n, err := uc.repo.PriorUses(ctx, e.ServerID, e.Command, e.TargetPids[0]); err == nil {
			out.PriorUses = int32(n)
		}
	}
	return out, nil
}

func (uc *GmUsecase) pendingOf(ctx context.Context, serverID, id string) (*gen.GmExecution, error) {
	e, err := uc.repo.GetExecution(ctx, id)
	if gen.IsNotFound(err) || (err == nil && e.ServerID != serverID) {
		return nil, ErrGmExecutionNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	if e.Status != "pending" {
		return nil, ErrGmNotPending
	}
	return e, nil
}

// Approve 批准玩家的申请并执行（危险指令需要输入玩家昵称）。
func (uc *GmUsecase) Approve(ctx context.Context, reviewer GmActor, serverID, id, confirmName, note string) (*v1.GmExecution, error) {
	e, err := uc.pendingOf(ctx, serverID, id)
	if err != nil {
		return nil, err
	}
	s, client, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, err
	}
	m, err := uc.servers.Manifest(ctx, s)
	if err != nil {
		return nil, err
	}
	cmd, ok := m.Command(e.Command)
	if !ok {
		return nil, ErrGmCommandNotFound
	}
	if cmd.Risk == mgp.RiskDanger {
		if strings.TrimSpace(confirmName) == "" {
			return nil, ErrGmConfirmRequired
		}
		for _, pid := range e.TargetPids {
			name, err := playerName(ctx, client, pid)
			if err != nil {
				return nil, GameMGPError(err)
			}
			if name != strings.TrimSpace(confirmName) {
				return nil, ErrGmConfirmMismatch
			}
		}
		e.Confirmed = true
	}
	now := time.Now()
	e.ReviewedBy, e.ReviewerName, e.ReviewedAt, e.ReviewNote = reviewer.UserID, reviewer.Name, &now, strings.TrimSpace(note)
	e.Status = "running"
	if e, err = uc.repo.SaveExecution(ctx, e); err != nil {
		return nil, ErrSystem(err)
	}
	e, _, err = uc.run(ctx, client, cmd, e, nil)
	if err != nil {
		return nil, err
	}
	return GmExecutionProto(e, true), nil
}

// Reject 驳回玩家的申请。
func (uc *GmUsecase) Reject(ctx context.Context, reviewer GmActor, serverID, id, note string) (*v1.GmExecution, error) {
	e, err := uc.pendingOf(ctx, serverID, id)
	if err != nil {
		return nil, err
	}
	note = strings.TrimSpace(note)
	if note == "" {
		return nil, response.BadRequest(400, "请填写驳回原因")
	}
	now := time.Now()
	e.ReviewedBy, e.ReviewerName, e.ReviewedAt, e.ReviewNote = reviewer.UserID, reviewer.Name, &now, note
	e.Status, e.FinishedAt = "rejected", &now
	if e, err = uc.repo.SaveExecution(ctx, e); err != nil {
		return nil, ErrSystem(err)
	}
	return GmExecutionProto(e, true), nil
}

// SelfServicePolicies 玩家自助设置：只列出 manifest 里允许自助的指令（没有设置的视为未开放）。
func (uc *GmUsecase) SelfServicePolicies(ctx context.Context, serverID string) ([]*v1.GmSelfServicePolicy, *gen.GameServer, error) {
	s, err := uc.servers.Get(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	m, err := uc.servers.Manifest(ctx, s)
	if err != nil {
		return nil, nil, err
	}
	rows, err := uc.repo.ListSelfService(ctx, serverID)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	byCmd := map[string]*gen.GmSelfService{}
	for _, r := range rows {
		byCmd[r.Command] = r
	}
	var out []*v1.GmSelfServicePolicy
	for i := range m.Commands {
		cmd := &m.Commands[i]
		if !cmd.SelfService {
			continue
		}
		p := &v1.GmSelfServicePolicy{
			Command: cmd.ID,
			Title:   cmd.Title,
			Group:   cmd.Group,
			Risk:    cmd.Risk,
			Target:  cmd.Target,
			Params:  JSONValue(cmd.Params),
		}
		if r, ok := byCmd[cmd.ID]; ok {
			p.Enabled, p.DailyLimit, p.CooldownSec = r.Enabled, int32(r.DailyLimit), int32(r.CooldownSec)
			p.NeedApproval, p.PlayerNote, p.UpdatedByName = r.NeedApproval, r.PlayerNote, r.UpdatedByName
			p.UpdateTime = timestamppb.New(r.UpdateTime)
		}
		out = append(out, p)
	}
	return out, s, nil
}

// UpdateSelfService 修改一条指令的玩家自助设置（立即生效）。
func (uc *GmUsecase) UpdateSelfService(ctx context.Context, actor GmActor, req *v1.UpdateGmSelfServiceRequest) (*v1.GmSelfServicePolicy, error) {
	policies, _, err := uc.SelfServicePolicies(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	var target *v1.GmSelfServicePolicy
	for _, p := range policies {
		if p.Command == req.Command {
			target = p
			break
		}
	}
	if target == nil {
		return nil, response.BadRequest(400, "这条指令不允许开放给玩家")
	}
	if req.DailyLimit < 0 || req.DailyLimit > 1000 || req.CooldownSec < 0 || req.CooldownSec > 30*86400 {
		return nil, response.BadRequest(400, "次数或间隔超出范围")
	}
	note := strings.TrimSpace(req.PlayerNote)
	if len([]rune(note)) > 200 {
		return nil, response.BadRequest(400, "说明最多 200 字")
	}
	row := &gen.GmSelfService{
		ServerID:      req.ServerId,
		Command:       req.Command,
		Enabled:       req.Enabled,
		DailyLimit:    int(req.DailyLimit),
		CooldownSec:   int(req.CooldownSec),
		NeedApproval:  req.NeedApproval,
		PlayerNote:    note,
		UpdatedBy:     actor.UserID,
		UpdatedByName: actor.Name,
	}
	saved, err := uc.repo.SaveSelfService(ctx, row)
	if err != nil {
		return nil, ErrSystem(err)
	}
	target.Enabled, target.DailyLimit, target.CooldownSec = saved.Enabled, int32(saved.DailyLimit), int32(saved.CooldownSec)
	target.NeedApproval, target.PlayerNote, target.UpdatedByName = saved.NeedApproval, saved.PlayerNote, saved.UpdatedByName
	target.UpdateTime = timestamppb.New(saved.UpdateTime)
	return target, nil
}

// SelfServiceUsage 一个玩家对一条自助指令的使用情况。
type SelfServiceUsage struct {
	Policy        *v1.GmSelfServicePolicy
	UsedToday     int
	Remaining     int // -1 不限
	NextAvailable *time.Time
	Last          *gen.GmExecution
	HasPending    bool
}

// SelfServiceFor 列出某玩家在某服务器上可用的自助指令（只含已开放的）与使用情况。
func (uc *GmUsecase) SelfServiceFor(ctx context.Context, serverID, actorID string) ([]*SelfServiceUsage, error) {
	policies, s, err := uc.SelfServicePolicies(ctx, serverID)
	if err != nil {
		return nil, err
	}
	dayStart := uc.servers.DayStart(s, time.Now())
	var out []*SelfServiceUsage
	for _, p := range policies {
		if !p.Enabled {
			continue
		}
		u := &SelfServiceUsage{Policy: p, Remaining: -1}
		if n, err := uc.repo.CountUses(ctx, serverID, actorID, p.Command, dayStart); err == nil {
			u.UsedToday = n
			if p.DailyLimit > 0 {
				u.Remaining = max(int(p.DailyLimit)-n, 0)
			}
		}
		if last, err := uc.repo.LastUse(ctx, serverID, actorID, p.Command); err == nil && last != nil {
			u.Last = last
			u.HasPending = last.Status == "pending"
			if p.CooldownSec > 0 {
				next := last.CreateTime.Add(time.Duration(p.CooldownSec) * time.Second)
				if next.After(time.Now()) {
					u.NextAvailable = &next
				}
			}
		}
		out = append(out, u)
	}
	return out, nil
}

// ExecuteSelfService 玩家执行开放给自己的 GM（目标强制为自己绑定的玩家）：
// 检查开放状态、每天次数（服务器时区的自然日）、间隔；需要审批时只写一条待审批记录。
func (uc *GmUsecase) ExecuteSelfService(ctx context.Context, actor GmActor, serverID, command, pid string, params json.RawMessage, reason string, summary map[string]string) (*gen.GmExecution, json.RawMessage, error) {
	actor.Kind = "player"
	if actor.Channel == "" {
		actor.Channel = "passport"
	}
	lockKey := serverID + "\x00" + actor.UserID + "\x00" + command
	lock, _ := uc.actorLocks.LoadOrStore(lockKey, &sync.Mutex{})
	mu := lock.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	s, client, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	m, err := uc.servers.Manifest(ctx, s)
	if err != nil {
		return nil, nil, err
	}
	cmd, ok := m.Command(command)
	if !ok || !cmd.SelfService {
		return nil, nil, ErrGmSelfServiceClosed
	}
	policy, err := uc.repo.GetSelfService(ctx, serverID, command)
	if err != nil || policy == nil || !policy.Enabled {
		return nil, nil, ErrGmSelfServiceClosed
	}
	now := time.Now()
	if policy.DailyLimit > 0 {
		n, err := uc.repo.CountUses(ctx, serverID, actor.UserID, command, uc.servers.DayStart(s, now))
		if err != nil {
			return nil, nil, ErrSystem(err)
		}
		if n >= policy.DailyLimit {
			return nil, nil, ErrGmSelfServiceLimit
		}
	}
	if last, err := uc.repo.LastUse(ctx, serverID, actor.UserID, command); err == nil && last != nil {
		if last.Status == "pending" {
			return nil, nil, ErrGmSelfServicePend
		}
		if policy.CooldownSec > 0 && now.Before(last.CreateTime.Add(time.Duration(policy.CooldownSec)*time.Second)) {
			return nil, nil, ErrGmSelfServiceWait
		}
	}
	targets := []string{pid}
	if cmd.Target == mgp.TargetServer {
		targets = nil
	}
	in := GmCommandInput{Targets: targets, Params: params, Reason: strings.TrimSpace(reason), Summary: summary, Source: "self-service"}
	e := uc.newExecution(s, cmd, actor, in)
	if policy.NeedApproval {
		e.Status = "pending"
		e, err = uc.repo.CreateExecution(ctx, e)
		if err != nil {
			return nil, nil, ErrSystem(err)
		}
		return e, nil, nil
	}
	e.Status = "running"
	e, err = uc.repo.CreateExecution(ctx, e)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	return uc.run(ctx, client, cmd, e, params)
}

// configDomainInfo 读取 MGP 配置域列表。
func (uc *GmUsecase) mgpConfigs(ctx context.Context, client *mgp.Client) ([]struct {
	ID        string `json:"id"`
	Title     string `json:"title"`
	File      string `json:"file"`
	Revision  string `json:"revision"`
	HotReload bool   `json:"hotReload"`
}, error) {
	var list []struct {
		ID        string `json:"id"`
		Title     string `json:"title"`
		File      string `json:"file"`
		Revision  string `json:"revision"`
		HotReload bool   `json:"hotReload"`
	}
	if err := client.GetInto(ctx, "/configs", nil, &list); err != nil {
		return nil, GameMGPError(err)
	}
	return list, nil
}

func configDraftProto(d *gen.GameConfigRevision) *v1.GmConfigDraft {
	if d == nil {
		return nil
	}
	return &v1.GmConfigDraft{
		Id:           d.ID,
		Content:      d.Content,
		BaseRevision: d.BaseRevision,
		AuthorName:   d.AuthorName,
		Comment:      d.Comment,
		UpdateTime:   timestamppb.New(d.UpdateTime),
	}
}

// ListConfigs 配置域列表（服务端 revision + 草稿 + 最近发布）。
func (uc *GmUsecase) ListConfigs(ctx context.Context, serverID string) (*v1.ListGmConfigsResponse, error) {
	_, client, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, err
	}
	list, err := uc.mgpConfigs(ctx, client)
	if err != nil {
		return nil, err
	}
	last, err := uc.repo.LastPublished(ctx, serverID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListGmConfigsResponse{}
	for _, c := range list {
		d := &v1.GmConfigDomain{Id: c.ID, Title: c.Title, File: c.File, HotReload: c.HotReload, Revision: c.Revision}
		if draft, err := uc.repo.GetConfigDraft(ctx, serverID, c.ID); err == nil && draft != nil {
			meta := configDraftProto(draft)
			meta.Content = ""
			d.Draft = meta
		}
		if r, ok := last[c.ID]; ok {
			d.LastSeq, d.LastPublishedBy = int32(r.Seq), r.AuthorName
			if r.PublishedAt != nil {
				d.LastPublishedAt = timestamppb.New(*r.PublishedAt)
			}
		}
		out.Items = append(out.Items, d)
	}
	return out, nil
}

// GetConfig 读取配置域：服务端内容与 revision，以及草稿。
func (uc *GmUsecase) GetConfig(ctx context.Context, serverID, domain string) (*v1.GetGmConfigResponse, error) {
	_, client, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, err
	}
	var doc struct {
		ID        string          `json:"id"`
		Title     string          `json:"title"`
		File      string          `json:"file"`
		Revision  string          `json:"revision"`
		HotReload bool            `json:"hotReload"`
		Content   json.RawMessage `json:"content"`
	}
	if err := client.GetInto(ctx, "/configs/"+url.PathEscape(domain), nil, &doc); err != nil {
		return nil, GameMGPError(err)
	}
	d := &v1.GmConfigDomain{Id: doc.ID, Title: doc.Title, File: doc.File, HotReload: doc.HotReload, Revision: doc.Revision}
	if draft, err := uc.repo.GetConfigDraft(ctx, serverID, domain); err == nil && draft != nil {
		d.Draft = configDraftProto(draft)
	}
	if last, err := uc.repo.LastPublished(ctx, serverID); err == nil {
		if r, ok := last[domain]; ok {
			d.LastSeq, d.LastPublishedBy = int32(r.Seq), r.AuthorName
			if r.PublishedAt != nil {
				d.LastPublishedAt = timestamppb.New(*r.PublishedAt)
			}
		}
	}
	return &v1.GetGmConfigResponse{Domain: d, Content: string(doc.Content)}, nil
}

func validJSON(content string) bool {
	return content != "" && json.Valid([]byte(content))
}

// SaveConfigDraft 保存草稿（每个配置域一份）。
func (uc *GmUsecase) SaveConfigDraft(ctx context.Context, actor GmActor, req *v1.SaveGmConfigDraftRequest) (*v1.GmConfigDraft, error) {
	if _, err := uc.servers.Get(ctx, req.ServerId); err != nil {
		return nil, err
	}
	if !validJSON(req.Content) {
		return nil, ErrGmConfigInvalidJSON
	}
	saved, err := uc.repo.SaveConfigDraft(ctx, &gen.GameConfigRevision{
		ServerID:     req.ServerId,
		Domain:       req.Domain,
		BaseRevision: req.BaseRevision,
		Content:      req.Content,
		AuthorID:     actor.UserID,
		AuthorName:   actor.Name,
		Comment:      strings.TrimSpace(req.Comment),
	})
	if err != nil {
		return nil, ErrSystem(err)
	}
	return configDraftProto(saved), nil
}

// DiscardConfigDraft 放弃草稿。
func (uc *GmUsecase) DiscardConfigDraft(ctx context.Context, serverID, domain string) error {
	if err := uc.repo.DeleteConfigDraft(ctx, serverID, domain); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// ValidateConfig 用服务端的加载逻辑校验内容（不发布）。
func (uc *GmUsecase) ValidateConfig(ctx context.Context, serverID, domain, content string) (*v1.ValidateGmConfigResponse, error) {
	if !validJSON(content) {
		return &v1.ValidateGmConfigResponse{Valid: false, Error: "invalid JSON"}, nil
	}
	_, client, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, err
	}
	raw, err := client.Do(ctx, mgp.Call{
		Method:  "PUT",
		Path:    "/configs/" + url.PathEscape(domain),
		Body:    mgp.ConfigPut{Content: json.RawMessage(content), DryRun: true},
		Timeout: time.Minute,
	})
	if err != nil {
		return nil, GameMGPError(err)
	}
	var r mgp.ConfigPutResult
	_ = json.Unmarshal(raw, &r)
	return &v1.ValidateGmConfigResponse{Valid: r.Valid, Error: r.Error}, nil
}

// PublishConfig 发布配置：写回服务端并热重载，记一个版本与一条 config.publish GM 记录。
func (uc *GmUsecase) PublishConfig(ctx context.Context, actor GmActor, req *v1.PublishGmConfigRequest) (*v1.PublishGmConfigResponse, error) {
	s, client, err := uc.servers.ClientOf(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	content, base := req.Content, req.BaseRevision
	draft, _ := uc.repo.GetConfigDraft(ctx, req.ServerId, req.Domain)
	if content == "" {
		if draft == nil {
			return nil, ErrGmConfigNoDraft
		}
		content = draft.Content
		if base == "" {
			base = draft.BaseRevision
		}
	}
	if !validJSON(content) {
		return nil, ErrGmConfigInvalidJSON
	}
	last, _ := uc.repo.LastPublished(ctx, req.ServerId)
	fromSeq := 0
	if r, ok := last[req.Domain]; ok {
		fromSeq = r.Seq
	}
	title := req.Domain
	if m, err := uc.servers.Manifest(ctx, s); err == nil {
		for _, c := range m.Configs {
			if c.ID == req.Domain && c.Title != "" {
				title = c.Title
			}
		}
	}
	e := &gen.GmExecution{
		ID:         uuid.NewString(),
		ServerID:   s.ID,
		Command:    "config.publish",
		Risk:       mgp.RiskHigh,
		TargetKind: mgp.TargetServer,
		ActorID:    actor.UserID,
		ActorName:  actor.Name,
		ActorKind:  gmActorKind(actor.Kind),
		Channel:    gmChannel(actor.Channel),
		ClientIP:   actor.IP,
		Platform:   actor.Platform,
		Params:     "{}",
		Summary:    map[string]string{"domain": req.Domain, "title": title, "fromSeq": strconv.Itoa(fromSeq), "toSeq": strconv.Itoa(fromSeq + 1), "comment": strings.TrimSpace(req.Comment)},
		Status:     "running",
		RequestID:  "momoko-" + uuid.NewString(),
		Source:     "config:" + req.Domain,
	}
	if e, err = uc.repo.CreateExecution(ctx, e); err != nil {
		return nil, ErrSystem(err)
	}
	start := time.Now()
	raw, err := client.Do(ctx, mgp.Call{
		Method:    "PUT",
		Path:      "/configs/" + url.PathEscape(req.Domain),
		Body:      mgp.ConfigPut{Content: json.RawMessage(content), Revision: base, Operator: actor.Name, Comment: strings.TrimSpace(req.Comment)},
		RequestID: e.RequestID,
		Timeout:   time.Minute,
	})
	now := time.Now()
	e.DurationMs, e.FinishedAt = int(now.Sub(start).Milliseconds()), &now
	if err != nil {
		e.Status, e.ErrorCode, e.ErrorMessage = "failed", mgp.CodeOf(err), err.Error()
		_, _ = uc.repo.SaveExecution(context.WithoutCancel(ctx), e)
		if mgp.CodeOf(err) == "REVISION_MISMATCH" {
			return nil, ErrGmConfigConflict
		}
		return nil, GameMGPError(err)
	}
	var r mgp.ConfigPutResult
	_ = json.Unmarshal(raw, &r)
	e.Status, e.Result, e.SuccessCount = "success", string(raw), 1
	e.Summary["revision"] = r.Revision
	rev, err := uc.repo.PublishConfig(ctx, &gen.GameConfigRevision{
		ServerID:     s.ID,
		Domain:       req.Domain,
		Revision:     r.Revision,
		BaseRevision: base,
		Content:      content,
		AuthorID:     actor.UserID,
		AuthorName:   actor.Name,
		Comment:      strings.TrimSpace(req.Comment),
		PublishedAt:  &now,
		ExecutionID:  e.ID,
	})
	if err != nil {
		uc.log.Errorf("record config revision: %v", err)
	}
	seq := fromSeq + 1
	if rev != nil {
		seq = rev.Seq
		e.Summary["toSeq"] = strconv.Itoa(seq)
	}
	_, _ = uc.repo.SaveExecution(context.WithoutCancel(ctx), e)
	return &v1.PublishGmConfigResponse{Revision: r.Revision, Seq: int32(seq), ExecutionId: e.ID}, nil
}

// ConfigRevisionProto 转为接口结构。
func ConfigRevisionProto(r *gen.GameConfigRevision, withContent bool) *v1.GmConfigRevision {
	out := &v1.GmConfigRevision{
		Id:           r.ID,
		Domain:       r.Domain,
		Seq:          int32(r.Seq),
		Revision:     r.Revision,
		BaseRevision: r.BaseRevision,
		AuthorName:   r.AuthorName,
		Comment:      r.Comment,
		State:        string(r.State),
		CreateTime:   timestamppb.New(r.CreateTime),
		ExecutionId:  r.ExecutionID,
	}
	if r.PublishedAt != nil {
		out.PublishedAt = timestamppb.New(*r.PublishedAt)
	}
	if withContent {
		out.Content = r.Content
	}
	return out
}

// ListConfigRevisions 版本记录。
func (uc *GmUsecase) ListConfigRevisions(ctx context.Context, req *v1.ListGmConfigRevisionsRequest) (*v1.ListGmConfigRevisionsResponse, error) {
	items, total, err := uc.repo.ListConfigRevisions(ctx, req.ServerId, req.Domain, int(req.Page), int(req.PageSize))
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListGmConfigRevisionsResponse{Total: int32(total)}
	for _, r := range items {
		out.Items = append(out.Items, ConfigRevisionProto(r, false))
	}
	return out, nil
}

// GetConfigRevision 某个版本的内容。
func (uc *GmUsecase) GetConfigRevision(ctx context.Context, serverID, domain, id string) (*v1.GmConfigRevision, error) {
	r, err := uc.repo.GetConfigRevision(ctx, id)
	if gen.IsNotFound(err) || (err == nil && (r.ServerID != serverID || r.Domain != domain)) {
		return nil, response.BadRequest(404, "版本不存在")
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return ConfigRevisionProto(r, true), nil
}
