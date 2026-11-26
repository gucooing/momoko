package biz

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"

	v1 "momoko/api/gen/v1"
	az "momoko/api/gen/v1/azurpromilia"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/mgp"
	"momoko/pkg/mgp/azurpromilia"
)

// AzurPromiliaUsecase 蓝色星原的 GM（管理端，AzurPromiliaGMService）：写 GM 记录，然后调用 pkg/mgp/azurpromilia 的对应函数并返回其内容。
// 用户通行证一侧是 AzurPromiliaPassportUsecase，两者互不调用。
type AzurPromiliaUsecase struct {
	gm  *GmUsecase
	log *log.Helper
}

func NewAzurPromiliaUsecase(gm *GmUsecase, logger log.Logger) *AzurPromiliaUsecase {
	return &AzurPromiliaUsecase{
		gm:  gm,
		log: log.NewHelper(log.With(logger, "module", "biz/azurpromilia")),
	}
}

// 蓝色星原的操作。
const (
	azItemAdjust     = "item.adjust"
	azPlayerMail     = "player.mail"
	azPlayerKick     = "player.kick"
	azPlayerBan      = "player.ban"
	azPlayerUnban    = "player.unban"
	azPlayerPassword = "player.password"
	azPlayerRename   = "player.rename"
	azMailSend       = "mail.send"
	azMailRevoke     = "mail.revoke"
	azCodeCreate     = "code.create"
	azCodeDisable    = "code.disable"
	azCodeEnable     = "code.enable"
)

// AzurPromiliaOperations 蓝色星原的全部操作（展示顺序）；SelfService 为可以在「玩家权限」里开放给玩家自助的操作。
var AzurPromiliaOperations = []GmOperation{
	{ID: azPlayerMail, Title: "发送邮件（含附件）", Group: "发放", Risk: RiskMedium, Target: TargetPlayers, SelfService: true},
	{ID: azItemAdjust, Title: "调整道具与货币", Group: "资源", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: azPlayerKick, Title: "强制下线", Group: "账号", Risk: RiskLow, Target: TargetPlayers, SelfService: true},
	{ID: azPlayerBan, Title: "封禁", Group: "账号", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: azPlayerUnban, Title: "解封", Group: "账号", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: azPlayerPassword, Title: "重置登录密码", Group: "账号", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: azPlayerRename, Title: "改昵称", Group: "账号", Risk: RiskLow, Target: TargetPlayer, SelfService: true},
	{ID: azMailSend, Title: "邮件补偿", Group: "运营", Risk: RiskHigh, Target: TargetServer},
	{ID: azMailRevoke, Title: "撤回邮件", Group: "运营", Risk: RiskHigh, Target: TargetServer},
	{ID: azCodeCreate, Title: "新建兑换码", Group: "运营", Risk: RiskMedium, Target: TargetServer},
	{ID: azCodeDisable, Title: "停用兑换码", Group: "运营", Risk: RiskMedium, Target: TargetServer},
	{ID: azCodeEnable, Title: "恢复兑换码", Group: "运营", Risk: RiskMedium, Target: TargetServer},
}

func azurpromiliaOperation(id string) (*GmOperation, bool) {
	for i := range AzurPromiliaOperations {
		if AzurPromiliaOperations[i].ID == id {
			return &AzurPromiliaOperations[i], true
		}
	}
	return nil, false
}

// 操作参数（记入 GM 记录，审批时按它重放）。
type azItemsParams struct {
	Grant []*az.Reward `json:"grant"`
	Take  []*az.Reward `json:"take"`
}

type azBanParams struct {
	Until  int64  `json:"until"`
	Reason string `json:"reason"`
}

type azPasswordParams struct {
	Password string `json:"password"`
}

type azNameParams struct {
	Name string `json:"name"`
}

type azMailParams struct {
	Mail *az.MailContent `json:"mail"`
}

type azSendMailParams struct {
	Target  string          `json:"target"`
	PIDs    []string        `json:"pids"`
	Mail    *az.MailContent `json:"mail"`
	StartAt int64           `json:"startAt"`
	EndAt   int64           `json:"endAt"`
}

type azIDParams struct {
	ID string `json:"id"`
}

// azurpromiliaCall 按操作与参数选择调用的 pkg/mgp/azurpromilia 函数。
func azurpromiliaCall(g *azurpromilia.Client, op string, params json.RawMessage) (GmCall, error) {
	switch op {
	case azItemAdjust:
		p, err := decodeParams[azItemsParams](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.AdjustItems(ctx, pid, p.Grant, p.Take, opt)
		}, err
	case azPlayerMail:
		p, err := decodeParams[azMailParams](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Mail(ctx, pid, p.Mail, opt)
		}, err
	case azPlayerKick:
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Kick(ctx, pid, opt)
		}, nil
	case azPlayerBan:
		p, err := decodeParams[azBanParams](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Ban(ctx, pid, p.Until, p.Reason, opt)
		}, err
	case azPlayerUnban:
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Unban(ctx, pid, opt)
		}, nil
	case azPlayerPassword:
		p, err := decodeParams[azPasswordParams](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.ResetPassword(ctx, pid, p.Password, opt)
		}, err
	case azPlayerRename:
		p, err := decodeParams[azNameParams](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Rename(ctx, pid, p.Name, opt)
		}, err
	case azMailSend:
		p, err := decodeParams[azSendMailParams](params)
		return func(ctx context.Context, _ string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SendMail(ctx, p.Target, p.PIDs, p.Mail, p.StartAt, p.EndAt, opt)
		}, err
	case azMailRevoke:
		p, err := decodeParams[azIDParams](params)
		return func(ctx context.Context, _ string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.RevokeMail(ctx, p.ID, opt)
		}, err
	case azCodeCreate:
		p, err := decodeParams[azurpromilia.CodesInput](params)
		return func(ctx context.Context, _ string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.CreateCodes(ctx, p, opt)
		}, err
	case azCodeDisable, azCodeEnable:
		p, err := decodeParams[azIDParams](params)
		return func(ctx context.Context, _ string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SetCodeBatchDisabled(ctx, p.ID, op == azCodeDisable, opt)
		}, err
	}
	return nil, ErrGmCommandNotFound
}

// server 运行蓝色星原的服务器与它的 MGP 客户端。
func (uc *AzurPromiliaUsecase) server(ctx context.Context, serverID string) (*gen.GameServer, *azurpromilia.Client, error) {
	s, c, err := uc.gm.Servers().ClientOf(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	if s.Game != azurpromilia.ID {
		return nil, nil, ErrGmGameUnsupported
	}
	return s, azurpromilia.New(c), nil
}

func (uc *AzurPromiliaUsecase) client(ctx context.Context, serverID string) (*azurpromilia.Client, error) {
	_, g, err := uc.server(ctx, serverID)
	return g, err
}

// names 目标玩家当前的昵称（GM 记录用）。
func (uc *AzurPromiliaUsecase) names(ctx context.Context, g *azurpromilia.Client, pids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(pids) == 0 {
		return out, nil
	}
	page, err := g.Players(ctx, azurpromilia.PlayerFilter{PIDs: pids}, "", int32(len(pids)))
	if err != nil {
		return nil, GameMGPError(err)
	}
	for _, p := range page.GetItems() {
		out[p.GetPid()] = p.GetName()
	}
	return out, nil
}

// execute 以当前管理员执行一个操作：校验目标，写 GM 记录，逐个目标调用游戏服务端。
func (uc *AzurPromiliaUsecase) execute(ctx context.Context, actor GmActor, serverID, opID string, targets []string, params any, dryRun bool, summary map[string]string) (*v1.GmExecution, []GmTargetOutcome, error) {
	s, g, err := uc.server(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	op, ok := azurpromiliaOperation(opID)
	if !ok {
		return nil, nil, ErrGmCommandNotFound
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, nil, ErrGmParams
	}
	call, err := azurpromiliaCall(g, opID, raw)
	if err != nil {
		return nil, nil, err
	}
	names, err := uc.names(ctx, g, targets)
	if err != nil {
		return nil, nil, err
	}
	e, outcomes, err := uc.gm.Execute(ctx, actor, s, op, GmExecutionInput{
		Targets: targets,
		Params:  raw,
		DryRun:  dryRun,
		Summary: summary,
		Source:  "gm",
	}, names, call)
	if err != nil {
		return nil, nil, err
	}
	return GmExecutionProto(e, true), outcomes, nil
}

// Replay 审批时按记录重放：记录的操作、按保存的参数选择的函数与目标当前的昵称。
func (uc *AzurPromiliaUsecase) Replay(ctx context.Context, s *gen.GameServer, e *gen.GmExecution) (*GmOperation, GmCall, map[string]string, error) {
	if s.Game != azurpromilia.ID {
		return nil, nil, nil, ErrGmGameUnsupported
	}
	c, err := uc.gm.Servers().Client(s)
	if err != nil {
		return nil, nil, nil, err
	}
	g := azurpromilia.New(c)
	op, ok := azurpromiliaOperation(e.Command)
	if !ok {
		return nil, nil, nil, ErrGmCommandNotFound
	}
	call, err := azurpromiliaCall(g, e.Command, json.RawMessage(e.Params))
	if err != nil {
		return nil, nil, nil, err
	}
	names, err := uc.names(ctx, g, e.TargetPids)
	if err != nil {
		return nil, nil, nil, err
	}
	return op, call, names, nil
}

// Approve 批准玩家的申请并执行。
func (uc *AzurPromiliaUsecase) Approve(ctx context.Context, reviewer GmActor, serverID, id, note string) (*v1.GmExecution, error) {
	return uc.gm.Approve(ctx, reviewer, serverID, id, note, uc.Replay)
}

// SelfServicePolicies 蓝色星原可以开放给玩家自助的操作与设置（目前没有）。
func (uc *AzurPromiliaUsecase) SelfServicePolicies(ctx context.Context, serverID string) ([]*v1.GmSelfServicePolicy, error) {
	if _, err := uc.client(ctx, serverID); err != nil {
		return nil, err
	}
	list, _, err := uc.gm.SelfServicePolicies(ctx, serverID, AzurPromiliaOperations)
	return list, err
}

func (uc *AzurPromiliaUsecase) UpdateSelfService(ctx context.Context, actor GmActor, req *v1.UpdateGmSelfServiceRequest) (*v1.GmSelfServicePolicy, error) {
	if _, err := uc.client(ctx, req.ServerId); err != nil {
		return nil, err
	}
	return uc.gm.UpdateSelfService(ctx, actor, req, AzurPromiliaOperations)
}

// operationResponse 把逐个目标的结果转为接口结构；成功的目标带上调用返回的内容。
func operationResponse(e *v1.GmExecution, outcomes []GmTargetOutcome) *az.OperationResponse {
	out := &az.OperationResponse{
		Execution: e,
		Results:   make([]*az.OperationResult, 0, len(outcomes)),
	}
	for _, o := range outcomes {
		r := &az.OperationResult{
			Target: &az.OperationTarget{
				Pid:     o.PID,
				Name:    o.Name,
				Status:  o.Status,
				Code:    o.Code,
				Message: o.Message,
			},
		}
		if d, ok := o.Result.(*az.OperationData); ok {
			r.Data = d
		}
		out.Results = append(out.Results, r)
	}
	return out
}

// serverResult 是服务器操作（只有一个目标）的返回内容。
func serverResult[T proto.Message](outcomes []GmTargetOutcome) T {
	var zero T
	for _, o := range outcomes {
		if v, ok := o.Result.(T); ok {
			return v
		}
	}
	return zero
}

// ---------------- 服务器 ----------------

func (uc *AzurPromiliaUsecase) Manifest(ctx context.Context, serverID string) (*az.Manifest, error) {
	g, err := uc.client(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.Manifest(ctx))
}

func (uc *AzurPromiliaUsecase) Status(ctx context.Context, serverID string) (*az.Status, error) {
	g, err := uc.client(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.Status(ctx))
}

func (uc *AzurPromiliaUsecase) DailyStats(ctx context.Context, req *az.DailyStatsRequest) (*az.DailyStats, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.DailyStats(ctx, req.Days))
}

// ---------------- 玩家 ----------------

// ListPlayers 玩家列表与本页玩家绑定的通行证。
func (uc *AzurPromiliaUsecase) ListPlayers(ctx context.Context, req *az.ListPlayersRequest) (*az.ListPlayersResponse, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	page, err := g.Players(ctx, azurpromilia.PlayerFilter{
		Q:           strings.TrimSpace(req.Q),
		Filter:      req.Filter,
		PIDs:        req.Pids,
		ExcludePIDs: req.ExcludePids,
	}, req.Cursor, req.Limit)
	if err != nil {
		return nil, GameMGPError(err)
	}
	pids := make([]string, 0, len(page.GetItems()))
	for _, p := range page.GetItems() {
		pids = append(pids, p.GetPid())
	}
	links, err := uc.gm.Links(ctx, req.ServerId, pids)
	if err != nil {
		return nil, err
	}
	return &az.ListPlayersResponse{
		Items:      page.GetItems(),
		NextCursor: page.GetNextCursor(),
		Total:      page.GetTotal(),
		Summary:    page.GetSummary(),
		Links:      links,
	}, nil
}

// GetPlayer 玩家条目、概览，以及通行证绑定与 GM 记录数。
func (uc *AzurPromiliaUsecase) GetPlayer(ctx context.Context, req *az.PlayerRequest) (*az.GetPlayerResponse, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	p, err := g.Player(ctx, req.Pid)
	if err != nil {
		return nil, GameMGPError(err)
	}
	out := &az.GetPlayerResponse{
		Player:   p.GetPlayer(),
		Overview: p.GetOverview(),
	}
	links, err := uc.gm.Links(ctx, req.ServerId, []string{req.Pid})
	if err != nil {
		return nil, err
	}
	if len(links) > 0 {
		out.Link = links[0]
	}
	if out.GmExecutions, err = uc.gm.CountExecutions(ctx, req.ServerId, req.Pid); err != nil {
		return nil, err
	}
	return out, nil
}

func (uc *AzurPromiliaUsecase) ListPlayerLogins(ctx context.Context, req *az.PageRequest) (*az.LoginPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Logins(ctx, req.Pid, req.Cursor, req.Limit))
}

func (uc *AzurPromiliaUsecase) ListPlayerItems(ctx context.Context, req *az.ListItemsRequest) (*az.ItemPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Items(ctx, req.Pid, req.Ids, req.Cursor, req.Limit))
}

func (uc *AzurPromiliaUsecase) ListPlayerCurrencies(ctx context.Context, req *az.PlayerRequest) (*az.ItemPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Currencies(ctx, req.Pid))
}

func (uc *AzurPromiliaUsecase) ListPlayerHeroes(ctx context.Context, req *az.PageRequest) (*az.HeroPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Heroes(ctx, req.Pid, req.Cursor, req.Limit))
}

func (uc *AzurPromiliaUsecase) ListPlayerPets(ctx context.Context, req *az.PageRequest) (*az.PetPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Pets(ctx, req.Pid, req.Cursor, req.Limit))
}

func (uc *AzurPromiliaUsecase) ListPlayerMails(ctx context.Context, req *az.PageRequest) (*az.MailPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Mails(ctx, req.Pid, req.Cursor, req.Limit))
}

// ---------------- 对玩家的操作 ----------------

func (uc *AzurPromiliaUsecase) AdjustItems(ctx context.Context, actor GmActor, req *az.AdjustItemsRequest) (*az.OperationResponse, error) {
	if len(req.Grant)+len(req.Take) == 0 {
		return nil, ErrGmParams
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, azItemAdjust, req.Pids, azItemsParams{
		Grant: req.Grant,
		Take:  req.Take,
	}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return operationResponse(e, outcomes), nil
}

func (uc *AzurPromiliaUsecase) MailPlayers(ctx context.Context, actor GmActor, req *az.MailPlayersRequest) (*az.OperationResponse, error) {
	if req.Mail == nil || strings.TrimSpace(req.Mail.Title) == "" {
		return nil, ErrGmParams
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, azPlayerMail, req.Pids, azMailParams{Mail: req.Mail}, req.DryRun,
		map[string]string{"标题": req.Mail.Title})
	if err != nil {
		return nil, err
	}
	return operationResponse(e, outcomes), nil
}

func (uc *AzurPromiliaUsecase) KickPlayers(ctx context.Context, actor GmActor, req *az.PlayersOperationRequest) (*az.OperationResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, azPlayerKick, req.Pids, struct{}{}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return operationResponse(e, outcomes), nil
}

func (uc *AzurPromiliaUsecase) BanPlayers(ctx context.Context, actor GmActor, req *az.BanPlayersRequest) (*az.OperationResponse, error) {
	p := azBanParams{
		Until:  int64(req.Until),
		Reason: strings.TrimSpace(req.Reason),
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, azPlayerBan, req.Pids, p, req.DryRun, map[string]string{"原因": p.Reason})
	if err != nil {
		return nil, err
	}
	return operationResponse(e, outcomes), nil
}

func (uc *AzurPromiliaUsecase) UnbanPlayers(ctx context.Context, actor GmActor, req *az.PlayersOperationRequest) (*az.OperationResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, azPlayerUnban, req.Pids, struct{}{}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return operationResponse(e, outcomes), nil
}

func (uc *AzurPromiliaUsecase) ResetPassword(ctx context.Context, actor GmActor, req *az.ResetPasswordRequest) (*az.OperationResponse, error) {
	if req.Password == "" {
		return nil, ErrGmParams
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, azPlayerPassword, req.Pids, azPasswordParams{Password: req.Password}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return operationResponse(e, outcomes), nil
}

func (uc *AzurPromiliaUsecase) RenamePlayer(ctx context.Context, actor GmActor, req *az.RenamePlayerRequest) (*az.OperationResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrGmParams
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, azPlayerRename, req.Pids, azNameParams{Name: name}, req.DryRun,
		map[string]string{"新昵称": name})
	if err != nil {
		return nil, err
	}
	return operationResponse(e, outcomes), nil
}

// ---------------- 邮件补偿 ----------------

func (uc *AzurPromiliaUsecase) ListMails(ctx context.Context, req *az.ListPageRequest) (*az.GmMailPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.GmMails(ctx, strings.TrimSpace(req.Q), req.Cursor, req.Limit))
}

func (uc *AzurPromiliaUsecase) GetMail(ctx context.Context, req *az.IDRequest) (*az.GmMail, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.GmMail(ctx, req.Id))
}

func (uc *AzurPromiliaUsecase) SendMail(ctx context.Context, actor GmActor, req *az.SendMailRequest) (*az.SendMailResponse, error) {
	if req.Mail == nil || strings.TrimSpace(req.Mail.Title) == "" {
		return nil, ErrGmParams
	}
	p := azSendMailParams{
		Target:  req.Target,
		PIDs:    req.Pids,
		Mail:    req.Mail,
		StartAt: int64(req.StartAt),
		EndAt:   int64(req.EndAt),
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, azMailSend, nil, p, req.DryRun, map[string]string{"标题": req.Mail.Title})
	if err != nil {
		return nil, err
	}
	return &az.SendMailResponse{
		Execution: e,
		Result:    serverResult[*az.SendMailResult](outcomes),
	}, nil
}

func (uc *AzurPromiliaUsecase) RevokeMail(ctx context.Context, actor GmActor, req *az.IDOperationRequest) (*az.MailResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, azMailRevoke, nil, azIDParams{ID: req.Id}, req.DryRun, map[string]string{"邮件": req.Id})
	if err != nil {
		return nil, err
	}
	return &az.MailResponse{
		Execution: e,
		Mail:      serverResult[*az.GmMail](outcomes),
	}, nil
}

// ---------------- 兑换码 ----------------

func (uc *AzurPromiliaUsecase) ListCodeBatches(ctx context.Context, req *az.ListPageRequest) (*az.CodeBatchPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.CodeBatches(ctx, strings.TrimSpace(req.Q), req.Cursor, req.Limit))
}

func (uc *AzurPromiliaUsecase) GetCodeBatch(ctx context.Context, req *az.IDRequest) (*az.CodeBatch, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.CodeBatch(ctx, req.Id))
}

func (uc *AzurPromiliaUsecase) CreateCodes(ctx context.Context, actor GmActor, req *az.CreateCodesRequest) (*az.CreateCodesResponse, error) {
	if req.Mail == nil || strings.TrimSpace(req.Name) == "" {
		return nil, ErrGmParams
	}
	p := azurpromilia.CodesInput{
		Name:     strings.TrimSpace(req.Name),
		Codes:    req.Codes,
		Count:    req.Count,
		CodeUses: req.CodeUses,
		MinLevel: req.MinLevel,
		StartAt:  int64(req.StartAt),
		EndAt:    int64(req.EndAt),
		Mail:     req.Mail,
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, azCodeCreate, nil, p, req.DryRun, map[string]string{"批次": p.Name})
	if err != nil {
		return nil, err
	}
	return &az.CreateCodesResponse{
		Execution: e,
		Result:    serverResult[*az.CreateCodesResult](outcomes),
	}, nil
}

func (uc *AzurPromiliaUsecase) SetCodeBatchDisabled(ctx context.Context, actor GmActor, req *az.SetCodeBatchDisabledRequest) (*az.CodeBatchResponse, error) {
	op := azCodeEnable
	if req.Disabled {
		op = azCodeDisable
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, op, nil, azIDParams{ID: req.Id}, req.DryRun, map[string]string{"批次": req.Id})
	if err != nil {
		return nil, err
	}
	return &az.CodeBatchResponse{
		Execution: e,
		Batch:     serverResult[*az.CodeBatch](outcomes),
	}, nil
}

func (uc *AzurPromiliaUsecase) ListBatchCodes(ctx context.Context, req *az.IDPageRequest) (*az.CodePage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.BatchCodes(ctx, req.Id, req.Cursor, req.Limit))
}

func (uc *AzurPromiliaUsecase) ListBatchRedemptions(ctx context.Context, req *az.IDPageRequest) (*az.RedemptionPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.BatchRedemptions(ctx, req.Id, req.Cursor, req.Limit))
}
