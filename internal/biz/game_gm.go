package biz

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	entschema "momoko/internal/data/ent/schema"
	"momoko/pkg/mgp"
	"momoko/pkg/response"
)

var (
	ErrGmCommandNotFound        = response.BadRequest(404, "这款游戏没有这个操作")
	ErrGmTargets                = response.BadRequest(400, "操作目标不正确")
	ErrGmExecutionNotFound      = response.BadRequest(404, "GM 记录不存在")
	ErrGmNotPending             = response.BadRequest(400, "这条申请已经处理过了")
	ErrGmConfigConflict         = response.BadRequest(409, "服务端的配置已被修改，请刷新后再发布")
	ErrGmConfigInvalidJSON      = response.BadRequest(400, "配置内容不是有效的 JSON")
	ErrGmConfigNoDraft          = response.BadRequest(400, "没有可发布的草稿")
	ErrGmConfigReadOnly         = response.BadRequest(400, "这份配置只读")
	ErrGmConfigRevisionNotFound = response.BadRequest(404, "版本不存在")
	ErrGmParams                 = response.BadRequest(400, "操作参数无效")
	ErrGmGameUnsupported        = response.BadRequest(400, "这台服务器的游戏不支持这项功能")
)

// GmActor 执行操作的人。
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

// GmUsecase GM 运营（管理端）的通用部分：操作执行与审计（GM 记录）、审批、玩家自助规则的设置、配置文档的草稿与版本。
// 游戏特有的操作与配置文档由游戏的用例（学マス：GakumasUsecase）给出；玩家自助的执行在通行证一侧（PassportUsecase）。
type GmUsecase struct {
	servers *GameServerUsecase
	repo    GmRepo
	users   UserRepo
	log     *log.Helper
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
	out := &v1.GetGmContextResponse{
		Server:   uc.servers.Brief(ctx, s),
		Timezone: uc.servers.Location(s).String(),
	}
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

// Links 这些玩家绑定的通行证。
func (uc *GmUsecase) Links(ctx context.Context, serverID string, pids []string) ([]*v1.GmPlayerLink, error) {
	links, err := uc.repo.LinksByPIDs(ctx, serverID, pids)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := make([]*v1.GmPlayerLink, 0, len(links))
	for _, l := range links {
		out = append(out, gmLinkProto(l))
	}
	return out, nil
}

// LinkedPIDs 已绑定通行证的全部玩家（按 pid 排序）。
func (uc *GmUsecase) LinkedPIDs(ctx context.Context, serverID string) ([]string, error) {
	linked, err := uc.repo.LinkedPIDs(ctx, serverID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := make([]string, 0, len(linked))
	for pid := range linked {
		out = append(out, pid)
	}
	sort.Strings(out)
	return out, nil
}

// FindLinkedPIDs 按通行证用户名或昵称查找绑定的玩家。
func (uc *GmUsecase) FindLinkedPIDs(ctx context.Context, serverID, q string, limit int) ([]string, error) {
	links, err := uc.repo.FindLinksByUser(ctx, serverID, q, limit)
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := make([]string, 0, len(links))
	for _, l := range links {
		if l.PID != "" {
			out = append(out, l.PID)
		}
	}
	return out, nil
}

// CountExecutions 涉及某玩家的 GM 记录数。
func (uc *GmUsecase) CountExecutions(ctx context.Context, serverID, pid string) (int32, error) {
	n, err := uc.repo.CountByTarget(ctx, serverID, pid)
	if err != nil {
		return 0, ErrSystem(err)
	}
	return int32(n), nil
}

// 操作的风险等级：low 直接执行；medium 执行前预览；high 玩家自助时开启了两步验证需要验证码；
// danger 管理员需要输入玩家昵称确认。
const (
	RiskLow    = "low"
	RiskMedium = "medium"
	RiskHigh   = "high"
	RiskDanger = "danger"
)

// 操作目标：player 单个玩家；players 1–GmMaxTargets 个玩家；server 无目标。
const (
	TargetPlayer  = "player"
	TargetPlayers = "players"
	TargetServer  = "server"
)

// GmMaxTargets 一次操作最多的玩家数。
const GmMaxTargets = 200

// GmOperation 一个操作的权限信息：标题、分组、风险、目标与是否允许开放给玩家自助。
type GmOperation struct {
	ID          string
	Title       string
	Group       string
	Risk        string
	Target      string
	SelfService bool
}

// GmExecutionInput 一次操作的输入。Params 为操作参数（记入 GM 记录，审批时重放）。
type GmExecutionInput struct {
	Targets []string
	Params  json.RawMessage
	DryRun  bool
	Summary map[string]string
	Reason  string
	Source  string
}

// GmCall 对一个目标调用游戏服务端（服务器操作 pid 为空），返回内容。
type GmCall func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error)

// GmTargetOutcome 操作对一个目标的结果：记录格式见 entschema.GmOutcome；Result 为调用返回的内容（失败时为空）。
type GmTargetOutcome struct {
	entschema.GmOutcome
	Result proto.Message `json:"-"`
}

// targetsValid 按操作的目标类型校验目标数量。
func targetsValid(op *GmOperation, targets []string) bool {
	switch op.Target {
	case TargetServer:
		return len(targets) == 0
	case TargetPlayer:
		return len(targets) == 1 && targets[0] != ""
	default:
		if len(targets) == 0 || len(targets) > GmMaxTargets {
			return false
		}
		seen := map[string]bool{}
		for _, t := range targets {
			if t == "" || seen[t] {
				return false
			}
			seen[t] = true
		}
		return true
	}
}

func (uc *GmUsecase) newExecution(s *gen.GameServer, op *GmOperation, actor GmActor, in GmExecutionInput) *gen.GmExecution {
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
		Command:      op.ID,
		CommandTitle: op.Title,
		CommandGroup: op.Group,
		Risk:         op.Risk,
		TargetKind:   op.Target,
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
		Reason:       strings.TrimSpace(in.Reason),
		DryRun:       in.DryRun,
		RequestID:    "momoko-" + uuid.NewString(),
		Source:       in.Source,
	}
}

// Execute 校验目标，写一条 GM 记录，再对每个目标调用 call（单个目标失败不影响其他目标）。
// names 为目标当前的昵称（记录用）。
func (uc *GmUsecase) Execute(ctx context.Context, actor GmActor, s *gen.GameServer, op *GmOperation, in GmExecutionInput, names map[string]string, call GmCall) (*gen.GmExecution, []GmTargetOutcome, error) {
	if !targetsValid(op, in.Targets) {
		return nil, nil, ErrGmTargets
	}
	e := uc.newExecution(s, op, actor, in)
	e.Status = "running"
	e, err := uc.repo.CreateExecution(ctx, e)
	if err != nil {
		return nil, nil, ErrSystem(err)
	}
	e, outcomes := uc.run(ctx, op, e, names, call)
	return e, outcomes, nil
}

// run 逐个目标调用并把结果写回记录。幂等键为记录的 requestId（玩家操作再加上 pid），
// 审批后重放与网络重试都不会让游戏服务端重复执行。
func (uc *GmUsecase) run(ctx context.Context, op *GmOperation, e *gen.GmExecution, names map[string]string, call GmCall) (*gen.GmExecution, []GmTargetOutcome) {
	opt := mgp.OperationOptions{
		Operator: e.ActorName,
		DryRun:   e.DryRun,
	}
	start := time.Now()
	targets := e.TargetPids
	if op.Target == TargetServer {
		targets = []string{""}
	}
	outcomes := make([]GmTargetOutcome, 0, len(targets))
	for _, pid := range targets {
		opt.IdempotencyKey = e.RequestID
		if pid != "" {
			opt.IdempotencyKey += ":" + pid
		}
		o := GmTargetOutcome{
			GmOutcome: entschema.GmOutcome{
				PID:    pid,
				Name:   names[pid],
				Status: "success",
			},
		}
		result, err := call(ctx, pid, opt)
		if err == nil {
			o.Result = result
			o.Data, err = protojson.Marshal(result)
		}
		if err != nil {
			o.Status, o.Result, o.Data = "failed", nil, nil
			var me *mgp.Error
			if errors.As(err, &me) {
				o.Code, o.Message = me.Code, me.Message
			} else {
				o.Code, o.Message = "INTERNAL", err.Error()
			}
		}
		outcomes = append(outcomes, o)
	}
	now := time.Now()
	e.DurationMs, e.FinishedAt = int(now.Sub(start).Milliseconds()), &now
	e.SuccessCount, e.FailedCount = 0, 0
	for _, o := range outcomes {
		if o.Status == "success" {
			e.SuccessCount++
			continue
		}
		e.FailedCount++
		if e.ErrorCode == "" {
			e.ErrorCode, e.ErrorMessage = o.Code, o.Message
		}
	}
	for i, t := range e.Targets {
		if name := names[t.PID]; name != "" {
			e.Targets[i].Name = name
		}
	}
	switch {
	case e.FailedCount == 0:
		e.Status = "success"
	case e.SuccessCount == 0:
		e.Status = "failed"
	default:
		e.Status = "partial"
	}
	raw, _ := json.Marshal(outcomes)
	e.Result = string(raw)
	saved, err := uc.repo.SaveExecution(context.WithoutCancel(ctx), e)
	if err != nil {
		uc.log.Errorf("save gm execution %s: %v", e.ID, err)
		saved = e
	}
	return saved, outcomes
}

// SetBackupRef 记下操作产生的存档备份 id。
func (uc *GmUsecase) SetBackupRef(ctx context.Context, e *gen.GmExecution, ids []string) *gen.GmExecution {
	if len(ids) == 0 {
		return e
	}
	e.BackupRef = strings.Join(ids, ",")
	saved, err := uc.repo.SaveExecution(context.WithoutCancel(ctx), e)
	if err != nil {
		uc.log.Errorf("save gm execution %s: %v", e.ID, err)
		return e
	}
	return saved
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
	if withResult {
		for _, o := range entschema.ParseGmOutcomes(e.Result) {
			out.Results = append(out.Results, &v1.GmExecutionResult{
				Pid:     o.PID,
				Name:    o.Name,
				Status:  o.Status,
				Code:    o.Code,
				Message: o.Message,
				Data:    string(o.Data),
			})
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

// GmReplay 按记录给出它的操作、按保存的参数调用游戏服务端的函数与目标当前的昵称。
type GmReplay func(ctx context.Context, s *gen.GameServer, e *gen.GmExecution) (*GmOperation, GmCall, map[string]string, error)

// Approve 批准玩家的申请并执行。
func (uc *GmUsecase) Approve(ctx context.Context, reviewer GmActor, serverID, id, note string, replay GmReplay) (*v1.GmExecution, error) {
	e, err := uc.pendingOf(ctx, serverID, id)
	if err != nil {
		return nil, err
	}
	s, err := uc.servers.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}
	op, call, names, err := replay(ctx, s, e)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	e.ReviewedBy, e.ReviewerName, e.ReviewedAt, e.ReviewNote = reviewer.UserID, reviewer.Name, &now, strings.TrimSpace(note)
	e.Status = "running"
	if e, err = uc.repo.SaveExecution(ctx, e); err != nil {
		return nil, ErrSystem(err)
	}
	e, _ = uc.run(ctx, op, e, names, call)
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

// SelfServicePolicies 玩家自助设置：只列出 ops 中允许开放给玩家自助的操作（没有设置的视为未开放）。
func (uc *GmUsecase) SelfServicePolicies(ctx context.Context, serverID string, ops []GmOperation) ([]*v1.GmSelfServicePolicy, *gen.GameServer, error) {
	s, err := uc.servers.Get(ctx, serverID)
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
	for _, op := range ops {
		if !op.SelfService {
			continue
		}
		p := &v1.GmSelfServicePolicy{
			Command: op.ID,
			Title:   op.Title,
			Group:   op.Group,
			Risk:    op.Risk,
			Target:  op.Target,
		}
		if r, ok := byCmd[op.ID]; ok {
			p.Enabled, p.DailyLimit, p.CooldownSec = r.Enabled, int32(r.DailyLimit), int32(r.CooldownSec)
			p.NeedApproval, p.PlayerNote, p.UpdatedByName = r.NeedApproval, r.PlayerNote, r.UpdatedByName
			p.UpdateTime = timestamppb.New(r.UpdateTime)
		}
		out = append(out, p)
	}
	return out, s, nil
}

// UpdateSelfService 修改一个操作的玩家自助设置（立即生效）。
func (uc *GmUsecase) UpdateSelfService(ctx context.Context, actor GmActor, req *v1.UpdateGmSelfServiceRequest, ops []GmOperation) (*v1.GmSelfServicePolicy, error) {
	policies, _, err := uc.SelfServicePolicies(ctx, req.ServerId, ops)
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
		return nil, response.BadRequest(400, "这个操作不允许开放给玩家")
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

func validJSON(content string) bool {
	return content != "" && json.Valid([]byte(content))
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
