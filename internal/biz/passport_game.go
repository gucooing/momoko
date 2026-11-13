package biz

import (
	"context"
	"encoding/json"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	entgmexecution "momoko/internal/data/ent/gen/gmexecution"
	entschema "momoko/internal/data/ent/schema"
	"momoko/pkg/mgp"
)

// gameLink 当前用户在这台服务器上的绑定（游戏空间的一切数据都只针对它）。
func (uc *PassportUsecase) gameLink(ctx context.Context, userID, serverID string) (*gen.GameAccountLink, error) {
	l, err := uc.repo.LinkOf(ctx, userID, serverID)
	if gen.IsNotFound(err) || (err == nil && (l == nil || l.PlayerID == "")) {
		return nil, ErrPassportLinkNotFound
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	return l, nil
}

// Game 游戏空间上下文；touch 时记录 last_seen_at（进入游戏空间），否则只是首页卡片的摘要。
func (uc *PassportUsecase) Game(ctx context.Context, userID, serverID string, touch bool) (*v1.PassportGameResponse, error) {
	l, err := uc.gameLink(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	s, err := uc.Servers().Get(ctx, serverID)
	if err != nil {
		return nil, err
	}
	out := &v1.PassportGameResponse{
		Server:   uc.Servers().Brief(ctx, s),
		Timezone: uc.Servers().Location(s).String(),
	}
	if touch {
		now := time.Now()
		l.LastSeenAt = &now
		if err := uc.repo.TouchLink(ctx, l.ID, now); err != nil {
			uc.log.Warnf("touch link: %v", err)
		}
	}
	if client, err := uc.Servers().Client(s); err != nil {
		out.PlayerError = "PASSPORT_GAME_UNAVAILABLE"
	} else if err := uc.refreshSnapshot(ctx, s, client, l); err != nil {
		out.PlayerError = passportFirst(mgp.CodeOf(err), "PASSPORT_GAME_UNAVAILABLE")
	}
	ss := uc.selfServiceCounts(ctx)
	servers := map[string]*gen.GameServer{s.ID: s}
	out.Link = uc.linkProto(ctx, l, servers, ss)
	if touch {
		if links, err := uc.repo.ListLinks(ctx, userID); err == nil {
			sortLinks(links)
			all := uc.serverMap(ctx)
			for _, other := range links {
				if other.ID != l.ID {
					out.Others = append(out.Others, uc.linkProto(ctx, other, all, ss))
				}
			}
		}
	}
	return out, nil
}

// LinkedPlayer 当前用户在服务器上绑定的游戏账号与用户的显示名（玩家自己操作时作为执行者）。
func (uc *PassportUsecase) LinkedPlayer(ctx context.Context, userID, serverID string) (string, string, error) {
	l, err := uc.gameLink(ctx, userID, serverID)
	if err != nil {
		return "", "", err
	}
	u, err := uc.user(ctx, userID)
	if err != nil {
		return "", "", err
	}
	return l.PlayerID, displayName(u), nil
}

// LinkedPID 当前用户在服务器上绑定的游戏账号。
func (uc *PassportUsecase) LinkedPID(ctx context.Context, userID, serverID string) (string, error) {
	l, err := uc.gameLink(ctx, userID, serverID)
	if err != nil {
		return "", err
	}
	return l.PlayerID, nil
}

func riskNeedsTotp(risk string) bool { return risk == RiskHigh || risk == RiskDanger }

// PassportOperation 通行证上玩家可以执行的一个操作（由游戏的通行证用例给出）：标题、分组、风险与目标写进 GM 记录，
// 管理员在「GM 记录」里看到并审批；高风险操作在开启了两步验证时要验证码。
type PassportOperation struct {
	ID     string
	Title  string
	Group  string
	Risk   string
	Target string
}

// PassportCall 对玩家绑定的游戏账号调用游戏服务端，返回内容。
type PassportCall func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error)

// SelfService 已开放的自助操作（ops 中管理员开放了的）与我的使用情况。
func (uc *PassportUsecase) SelfService(ctx context.Context, userID, serverID string, ops []PassportOperation) (*v1.ListPassportSelfServiceResponse, error) {
	if _, err := uc.gameLink(ctx, userID, serverID); err != nil {
		return nil, err
	}
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	s, err := uc.servers.Get(ctx, serverID)
	if err != nil {
		return nil, err
	}
	rows, err := uc.records.ListSelfService(ctx, serverID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	policies := map[string]*gen.GmSelfService{}
	for _, r := range rows {
		policies[r.Command] = r
	}
	totpOn := u.TotpSecret != "" && u.TotpEnabledAt != nil
	now := time.Now()
	out := &v1.ListPassportSelfServiceResponse{}
	for _, op := range ops {
		p, ok := policies[op.ID]
		if !ok || !p.Enabled {
			continue
		}
		item := &v1.PassportSelfService{
			Command:      op.ID,
			Title:        op.Title,
			Group:        op.Group,
			Risk:         op.Risk,
			Target:       op.Target,
			Limit:        int32(p.UseLimit),
			PeriodDays:   int32(p.PeriodDays),
			NeedApproval: p.NeedApproval,
			PlayerNote:   p.PlayerNote,
			Remaining:    -1,
			NeedTotp:     totpOn && riskNeedsTotp(op.Risk),
		}
		start, resets := uc.servers.PeriodWindow(s, p.PeriodDays, now)
		if n, err := uc.records.CountUses(ctx, serverID, userID, op.ID, start); err == nil {
			item.Used = int32(n)
			if p.UseLimit > 0 {
				item.Remaining = int32(max(p.UseLimit-n, 0))
				if resets != nil {
					item.ResetsAt = timestamppb.New(*resets)
				}
			}
		}
		if last, err := uc.records.LastUse(ctx, serverID, userID, op.ID); err == nil && last != nil {
			item.LastUsedAt = timestamppb.New(last.CreateTime)
			item.LastStatus = string(last.Status)
			item.HasPending = last.Status == "pending"
		}
		out.Items = append(out.Items, item)
	}
	return out, nil
}

// passportActor 执行操作的玩家（写进 GM 记录）。
type passportActor struct {
	UserID   string
	Name     string
	IP       string
	Platform string
}

// selfServiceActor 当前用户与他在服务器上的绑定；高风险操作在开启了两步验证时校验验证码。
func (uc *PassportUsecase) selfServiceActor(ctx context.Context, userID, serverID, risk, totpCode string) (passportActor, *gen.GameAccountLink, error) {
	l, err := uc.gameLink(ctx, userID, serverID)
	if err != nil {
		return passportActor{}, nil, err
	}
	u, err := uc.user(ctx, userID)
	if err != nil {
		return passportActor{}, nil, err
	}
	if riskNeedsTotp(risk) && u.TotpSecret != "" && u.TotpEnabledAt != nil {
		if strings.TrimSpace(totpCode) == "" {
			return passportActor{}, nil, ErrPassportTotpRequired
		}
		if err := uc.checkSecondFactor(ctx, u, totpCode, ""); err != nil {
			return passportActor{}, nil, err
		}
	}
	client := PassportClientOf(ctx)
	return passportActor{
		UserID:   u.ID,
		Name:     displayName(u),
		IP:       client.IP,
		Platform: Platform(client.UserAgent),
	}, l, nil
}

// selfServiceReason 玩家写给管理员的申请理由（最多 200 字）。
func selfServiceReason(reason string) string {
	reason = strings.TrimSpace(reason)
	if r := []rune(reason); len(r) > 200 {
		reason = string(r[:200])
	}
	return reason
}

// RunSelfService 玩家执行开放给自己的操作（目标为自己绑定的游戏账号）：检查开放状态、本周期的次数（每 N 天，
// 服务器时区 0 点刷新；或永久不刷新）与待审批；需要审批时只写一条待审批的 GM 记录（返回的内容为空），否则写记录并调用 call。
// params 为操作参数（游戏服务端接口的 body，记入 GM 记录，管理员批准后按它执行）。
func (uc *PassportUsecase) RunSelfService(ctx context.Context, userID, serverID string, op PassportOperation, params any, reason, totpCode string, call PassportCall) (*gen.GmExecution, proto.Message, error) {
	actor, l, err := uc.selfServiceActor(ctx, userID, serverID, op.Risk, totpCode)
	if err != nil {
		return nil, nil, err
	}
	s, err := uc.servers.Get(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, nil, ErrPassportInvalidArgument
	}
	lock, _ := uc.selfServiceLocks.LoadOrStore(s.ID+"\x00"+actor.UserID+"\x00"+op.ID, &sync.Mutex{})
	mu := lock.(*sync.Mutex)
	mu.Lock()
	defer mu.Unlock()

	policy, err := uc.records.GetSelfService(ctx, s.ID, op.ID)
	if err != nil || policy == nil || !policy.Enabled {
		return nil, nil, ErrPassportSelfServiceClosed
	}
	now := time.Now()
	if policy.UseLimit > 0 {
		start, _ := uc.servers.PeriodWindow(s, policy.PeriodDays, now)
		n, err := uc.records.CountUses(ctx, s.ID, actor.UserID, op.ID, start)
		if err != nil {
			return nil, nil, ErrSystem(err)
		}
		if n >= policy.UseLimit {
			return nil, nil, ErrPassportSelfServiceLimit
		}
	}
	if last, err := uc.records.LastUse(ctx, s.ID, actor.UserID, op.ID); err == nil && last != nil && last.Status == "pending" {
		return nil, nil, ErrPassportSelfServicePending
	}
	e := newPlayerRecord(s, op, actor, l.PlayerID, l.Nickname, raw, selfServiceReason(reason), nil, "self-service")
	if policy.NeedApproval {
		e.Status = "pending"
		if e, err = uc.records.CreateExecution(ctx, e); err != nil {
			return nil, nil, ErrSystem(err)
		}
		return e, nil, nil
	}
	e.Status = "running"
	if e, err = uc.records.CreateExecution(ctx, e); err != nil {
		return nil, nil, ErrSystem(err)
	}
	e, result := uc.runPlayerRecord(ctx, e, l.Nickname, call)
	return e, result, nil
}

// newPlayerRecord 玩家执行的一条 GM 记录（目标为 pid；记录格式与管理端执行的相同，管理员在 GM 记录里查看与审批）。
func newPlayerRecord(s *gen.GameServer, op PassportOperation, actor passportActor, pid, name string, params json.RawMessage, reason string, summary map[string]string, source string) *gen.GmExecution {
	if len(params) == 0 {
		params = json.RawMessage("{}")
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
		ActorKind:    entgmexecution.ActorKindPlayer,
		Channel:      entgmexecution.ChannelPassport,
		ClientIP:     actor.IP,
		Platform:     actor.Platform,
		Targets:      []entschema.GmTarget{{PID: pid, Name: name}},
		TargetPids:   []string{pid},
		Params:       string(params),
		Summary:      summary,
		Reason:       reason,
		RequestID:    "momoko-" + uuid.NewString(),
		Source:       source,
	}
}

// runPlayerRecord 调用游戏服务端并把结果写回记录。幂等键为记录的 requestId 加上 pid，网络重试不会重复执行。
func (uc *PassportUsecase) runPlayerRecord(ctx context.Context, e *gen.GmExecution, name string, call PassportCall) (*gen.GmExecution, proto.Message) {
	pid := e.TargetPids[0]
	start := time.Now()
	o := entschema.GmOutcome{PID: pid, Name: name, Status: "success"}
	result, err := call(ctx, pid, mgp.OperationOptions{Operator: e.ActorName, IdempotencyKey: e.RequestID + ":" + pid})
	if err == nil {
		o.Data, err = protojson.Marshal(result)
	}
	if err != nil {
		result, o.Status, o.Data = nil, "failed", nil
		var me *mgp.Error
		if errors.As(err, &me) {
			o.Code, o.Message = me.Code, me.Message
		} else {
			o.Code, o.Message = "INTERNAL", err.Error()
		}
	}
	now := time.Now()
	e.DurationMs, e.FinishedAt = int(now.Sub(start).Milliseconds()), &now
	if o.Status == "success" {
		e.Status, e.SuccessCount, e.FailedCount = "success", 1, 0
	} else {
		e.Status, e.SuccessCount, e.FailedCount = "failed", 0, 1
		e.ErrorCode, e.ErrorMessage = o.Code, o.Message
	}
	raw, _ := json.Marshal([]entschema.GmOutcome{o})
	e.Result = string(raw)
	saved, err := uc.records.SaveExecution(context.WithoutCancel(ctx), e)
	if err != nil {
		uc.log.Errorf("save gm execution %s: %v", e.ID, err)
		saved = e
	}
	return saved, result
}

// outcomeData 记录中某个玩家的结果（JSON 文本）。
func outcomeData(e *gen.GmExecution, pid string) string {
	for _, o := range entschema.ParseGmOutcomes(e.Result) {
		if o.PID == pid {
			return string(o.Data)
		}
	}
	return ""
}

func (uc *PassportUsecase) recordOf(e *gen.GmExecution) *v1.PassportGameRecord {
	pid := ""
	if len(e.TargetPids) > 0 {
		pid = e.TargetPids[0]
	}
	return &v1.PassportGameRecord{
		Id:           e.ID,
		Kind:         "self_service",
		Command:      e.Command,
		CommandTitle: e.CommandTitle,
		Status:       string(e.Status),
		Reason:       e.Reason,
		ReviewNote:   e.ReviewNote,
		ReviewedAt:   tsOrNil(e.ReviewedAt),
		CreateTime:   timestamppb.New(e.CreateTime),
		FinishedAt:   tsOrNil(e.FinishedAt),
		Data:         outcomeData(e, pid),
		ErrorCode:    e.ErrorCode,
	}
}

func pageOf(page, size int32, def int) (int, int) {
	p, s := int(page), int(size)
	if p <= 0 {
		p = 1
	}
	if s <= 0 || s > 100 {
		s = def
	}
	return p, s
}

// Requests 我的申请（需要审批的自助操作）。
func (uc *PassportUsecase) Requests(ctx context.Context, userID string, req *v1.ListPassportGameRecordsRequest) (*v1.ListPassportGameRecordsResponse, error) {
	if _, err := uc.gameLink(ctx, userID, req.ServerId); err != nil {
		return nil, err
	}
	page, size := pageOf(req.Page, req.PageSize, 10)
	list, total, err := uc.repo.ListPlayerExecutions(ctx, PassportExecFilter{UserID: userID, ServerID: req.ServerId, Reviewable: true, Offset: (page - 1) * size, Limit: size})
	if err != nil {
		return nil, ErrSystem(err)
	}
	out := &v1.ListPassportGameRecordsResponse{Total: int32(total)}
	for _, e := range list {
		out.Items = append(out.Items, uc.recordOf(e))
	}
	return out, nil
}

// Records 操作记录：我在这台服务器上的自助操作与兑换码，从新到旧。
func (uc *PassportUsecase) Records(ctx context.Context, userID string, req *v1.ListPassportGameRecordsRequest) (*v1.ListPassportGameRecordsResponse, error) {
	if _, err := uc.gameLink(ctx, userID, req.ServerId); err != nil {
		return nil, err
	}
	page, size := pageOf(req.Page, req.PageSize, 10)
	need := page * size
	execs, execTotal, err := uc.repo.ListPlayerExecutions(ctx, PassportExecFilter{UserID: userID, ServerID: req.ServerId, Limit: need})
	if err != nil {
		return nil, ErrSystem(err)
	}
	redeems, redeemTotal, err := uc.repo.ListRedeems(ctx, userID, req.ServerId, 0, need)
	if err != nil {
		return nil, ErrSystem(err)
	}
	var items []*v1.PassportGameRecord
	for _, e := range execs {
		items = append(items, uc.recordOf(e))
	}
	for _, r := range redeems {
		items = append(items, &v1.PassportGameRecord{
			Id:         r.ID,
			Kind:       "redeem",
			Status:     string(r.Result),
			CreateTime: timestamppb.New(r.CreateTime),
			Code:       r.Code,
		})
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].CreateTime.AsTime().After(items[j].CreateTime.AsTime()) })
	out := &v1.ListPassportGameRecordsResponse{Total: int32(execTotal + redeemTotal)}
	from := (page - 1) * size
	for i := from; i < len(items) && i < from+size; i++ {
		out.Items = append(out.Items, items[i])
	}
	return out, nil
}
