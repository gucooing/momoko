package biz

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"google.golang.org/protobuf/proto"

	v1 "momoko/api/gen/v1"
	gk "momoko/api/gen/v1/gakumas"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/mgp"
	"momoko/pkg/mgp/gakumas"
)

// GakumasUsecase 学マス 的 GM 运营（管理端，GakumasGMService）：判定执行者身份与操作权限（风险确认、玩家自助规则的设置、审批）、
// 写 GM 记录，然后调用 pkg/mgp/gakumas 的对应函数并返回其内容。用户通行证一侧是 GakumasPassportUsecase，两者互不调用。
type GakumasUsecase struct {
	gm  *GmUsecase
	log *log.Helper
}

func NewGakumasUsecase(gm *GmUsecase, logger log.Logger) *GakumasUsecase {
	return &GakumasUsecase{
		gm:  gm,
		log: log.NewHelper(log.With(logger, "module", "biz/gakumas")),
	}
}

// 学マス 的操作与权限：SelfService 为可以开放给玩家自助的操作（GM 操作页里对玩家的操作都可以开放）。
const (
	gkGiftSend         = "gift.send"
	gkGiftRevoke       = "gift.revoke"
	gkIdolCardGrantAll = "idolCard.grantAll"
	gkSupportGrantAll  = "supportCard.grantAll"
	gkJewelAdjust      = "jewel.adjust"
	gkItemAdjust       = "item.adjust"
	gkAPRefill         = "actionPoint.refill"
	gkExpAdd           = "exp.add"
	gkPlayerKick       = "player.kick"
	gkPlayerBan        = "player.ban"
	gkPlayerUnban      = "player.unban"
	gkPlayerPassword   = "player.password"
	gkPlayerRename     = "player.rename"
	gkSupportRental    = "supportCard.setRental"
	gkMemoryRental     = "memory.setRental"
	gkPvpSetRate       = "pvp.setRate"
	gkPvpResetDaily    = "pvp.resetDaily"
	gkSeminarClear     = "seminar.clear"
	gkSeminarReset     = "seminar.reset"
	gkArchiveExport    = "archive.export"
	gkArchiveBackup    = "archive.backup"
	gkArchiveRestore   = "archive.restore"
	gkMailSend         = "mail.send"
	gkMailRevoke       = "mail.revoke"
	gkMaintenance      = "maintenance.set"
)

// GakumasOperations 学マス 的全部操作（展示顺序）。
var GakumasOperations = []GmOperation{
	{ID: gkGiftSend, Title: "发送到礼物箱", Group: "发放", Risk: RiskMedium, Target: TargetPlayers, SelfService: true},
	{ID: gkGiftRevoke, Title: "撤回未领取的礼物", Group: "发放", Risk: RiskMedium, Target: TargetPlayers, SelfService: true},
	{ID: gkIdolCardGrantAll, Title: "全部偶像卡（含满潜能碎片）", Group: "发放", Risk: RiskDanger, Target: TargetPlayers, SelfService: true},
	{ID: gkSupportGrantAll, Title: "全部支援卡（满突破）", Group: "发放", Risk: RiskDanger, Target: TargetPlayers, SelfService: true},
	{ID: gkJewelAdjust, Title: "调整钻石（免费 / 付费）", Group: "资源", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: gkItemAdjust, Title: "调整金币与道具", Group: "资源", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: gkAPRefill, Title: "回满体力", Group: "资源", Risk: RiskLow, Target: TargetPlayers, SelfService: true},
	{ID: gkExpAdd, Title: "制作人经验 / 等级", Group: "养成", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: gkPlayerKick, Title: "强制下线", Group: "账号", Risk: RiskLow, Target: TargetPlayers, SelfService: true},
	{ID: gkPlayerBan, Title: "封禁", Group: "账号", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: gkPlayerUnban, Title: "解封", Group: "账号", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: gkPlayerPassword, Title: "重置登录密码", Group: "账号", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: gkPlayerRename, Title: "改昵称", Group: "账号", Risk: RiskLow, Target: TargetPlayers, SelfService: true},
	{ID: gkSupportRental, Title: "设为援助支援卡", Group: "账号", Risk: RiskLow, Target: TargetPlayer},
	{ID: gkMemoryRental, Title: "设为援助回忆", Group: "账号", Risk: RiskLow, Target: TargetPlayer},
	{ID: gkPvpSetRate, Title: "竞赛：调整 Rate", Group: "玩法", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: gkPvpResetDaily, Title: "竞赛：重置今日次数", Group: "玩法", Risk: RiskHigh, Target: TargetPlayers, SelfService: true},
	{ID: gkSeminarClear, Title: "研修：标记全部通关并发放奖励", Group: "玩法", Risk: RiskMedium, Target: TargetPlayers, SelfService: true},
	{ID: gkSeminarReset, Title: "研修：重置进度", Group: "玩法", Risk: RiskMedium, Target: TargetPlayers, SelfService: true},
	{ID: gkArchiveExport, Title: "导出存档（JSON）", Group: "存档", Risk: RiskMedium, Target: TargetPlayers, SelfService: true},
	{ID: gkArchiveBackup, Title: "备份当前存档", Group: "存档", Risk: RiskLow, Target: TargetPlayer},
	{ID: gkArchiveRestore, Title: "回档到备份", Group: "存档", Risk: RiskDanger, Target: TargetPlayer},
	{ID: gkMailSend, Title: "全服补偿 / 定向邮件", Group: "运营", Risk: RiskHigh, Target: TargetServer},
	{ID: gkMailRevoke, Title: "撤回邮件", Group: "运营", Risk: RiskHigh, Target: TargetServer},
	{ID: gkMaintenance, Title: "维护模式 / 登录白名单", Group: "运营", Risk: RiskHigh, Target: TargetServer},
}

func gakumasOperation(id string) (*GmOperation, bool) {
	for i := range GakumasOperations {
		if GakumasOperations[i].ID == id {
			return &GakumasOperations[i], true
		}
	}
	return nil, false
}

// 操作参数（记入 GM 记录，审批时按它重放）。

type gkMessage struct {
	Message string `json:"message,omitempty"`
}

type gkItemParams struct {
	ItemID   string `json:"itemId"`
	Quantity int64  `json:"quantity"`
}

type gkJewelParams struct {
	Free int64 `json:"free"`
	Paid int64 `json:"paid"`
}

type gkExp struct {
	Exp int64 `json:"exp"`
}

type gkBan struct {
	Until     int64  `json:"until,omitempty"`
	Permanent bool   `json:"permanent,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

type gkPassword struct {
	Password string `json:"password"`
}

type gkName struct {
	Name string `json:"name"`
}

type gkSupportCardID struct {
	SupportCardID string `json:"supportCardId"`
}

type gkMemoryID struct {
	MemoryID string `json:"memoryId"`
}

type gkRate struct {
	Rate int32 `json:"rate"`
}

type gkNote struct {
	Note string `json:"note,omitempty"`
}

type gkRestore struct {
	BackupID int64 `json:"backupId"`
}

type gkMaintenanceParams struct {
	Enabled   bool     `json:"enabled"`
	Whitelist []string `json:"whitelist"`
}

type gkMailID struct {
	MailID string `json:"mailId"`
}

func decodeParams[T any](params json.RawMessage) (T, error) {
	var p T
	if len(params) > 0 && json.Unmarshal(params, &p) != nil {
		return p, ErrGmParams
	}
	return p, nil
}

// gakumasCall 按操作与参数选择调用的 pkg/mgp/gakumas 函数。
func gakumasCall(g *gakumas.Client, op string, params json.RawMessage) (GmCall, error) {
	switch op {
	case gkGiftSend:
		p, err := decodeParams[gakumas.GiftSend](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SendGift(ctx, pid, p, opt)
		}, err
	case gkGiftRevoke:
		p, err := decodeParams[gakumas.GiftRevoke](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.RevokeGifts(ctx, pid, p, opt)
		}, err
	case gkIdolCardGrantAll:
		p, err := decodeParams[gkMessage](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.GrantAllIdolCards(ctx, pid, p.Message, opt)
		}, err
	case gkSupportGrantAll:
		p, err := decodeParams[gkMessage](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.GrantAllSupportCards(ctx, pid, p.Message, opt)
		}, err
	case gkJewelAdjust:
		p, err := decodeParams[gkJewelParams](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.AdjustJewel(ctx, pid, p.Free, p.Paid, opt)
		}, err
	case gkItemAdjust:
		p, err := decodeParams[gkItemParams](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.AdjustItem(ctx, pid, p.ItemID, p.Quantity, opt)
		}, err
	case gkAPRefill:
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.RefillActionPoint(ctx, pid, opt)
		}, nil
	case gkExpAdd:
		p, err := decodeParams[gkExp](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.AddExp(ctx, pid, p.Exp, opt)
		}, err
	case gkPlayerKick:
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Kick(ctx, pid, opt)
		}, nil
	case gkPlayerBan:
		p, err := decodeParams[gkBan](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Ban(ctx, pid, p.Until, p.Permanent, p.Reason, opt)
		}, err
	case gkPlayerUnban:
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Unban(ctx, pid, opt)
		}, nil
	case gkPlayerPassword:
		p, err := decodeParams[gkPassword](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SetPassword(ctx, pid, p.Password, opt)
		}, err
	case gkPlayerRename:
		p, err := decodeParams[gkName](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Rename(ctx, pid, p.Name, opt)
		}, err
	case gkSupportRental:
		p, err := decodeParams[gkSupportCardID](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SetRentalSupportCard(ctx, pid, p.SupportCardID, opt)
		}, err
	case gkMemoryRental:
		p, err := decodeParams[gkMemoryID](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SetRentalMemory(ctx, pid, p.MemoryID, opt)
		}, err
	case gkPvpSetRate:
		p, err := decodeParams[gkRate](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SetPvpRate(ctx, pid, p.Rate, opt)
		}, err
	case gkPvpResetDaily:
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.ResetPvpDaily(ctx, pid, opt)
		}, nil
	case gkSeminarClear:
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.ClearSeminar(ctx, pid, opt)
		}, nil
	case gkSeminarReset:
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.ResetSeminar(ctx, pid, opt)
		}, nil
	case gkArchiveExport:
		return func(ctx context.Context, pid string, _ mgp.OperationOptions) (proto.Message, error) {
			return g.ExportArchive(ctx, pid)
		}, nil
	case gkArchiveBackup:
		p, err := decodeParams[gkNote](params)
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Backup(ctx, pid, p.Note, opt)
		}, err
	case gkArchiveRestore:
		p, err := decodeParams[gkRestore](params)
		if err == nil && p.BackupID <= 0 {
			err = ErrGmParams
		}
		return func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Restore(ctx, pid, p.BackupID, opt)
		}, err
	case gkMailSend:
		p, err := decodeParams[gakumas.Mail](params)
		return func(ctx context.Context, _ string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SendMail(ctx, p, opt)
		}, err
	case gkMailRevoke:
		p, err := decodeParams[gkMailID](params)
		if err == nil && p.MailID == "" {
			err = ErrGmParams
		}
		return func(ctx context.Context, _ string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.RevokeMail(ctx, p.MailID, opt)
		}, err
	case gkMaintenance:
		p, err := decodeParams[gkMaintenanceParams](params)
		return func(ctx context.Context, _ string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SetMaintenance(ctx, p.Enabled, p.Whitelist, opt)
		}, err
	}
	return nil, ErrGmCommandNotFound
}

// server 运行学マス 的服务器与它的 MGP 客户端。
func (uc *GakumasUsecase) server(ctx context.Context, serverID string) (*gen.GameServer, *gakumas.Client, error) {
	s, c, err := uc.gm.Servers().ClientOf(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	if s.Game != gakumas.ID {
		return nil, nil, ErrGmGameUnsupported
	}
	return s, gakumas.New(c), nil
}

// client 运行学マス 的服务器的 MGP 客户端。
func (uc *GakumasUsecase) client(ctx context.Context, serverID string) (*gakumas.Client, error) {
	_, g, err := uc.server(ctx, serverID)
	return g, err
}

// names 目标玩家当前的昵称（危险操作确认与 GM 记录用）。
func (uc *GakumasUsecase) names(ctx context.Context, g *gakumas.Client, pids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(pids) == 0 {
		return out, nil
	}
	page, err := g.Players(ctx, gakumas.PlayerFilter{PIDs: pids}, "", int32(len(pids)))
	if err != nil {
		return nil, GameMGPError(err)
	}
	for _, p := range page.Items {
		out[p.Pid] = p.Name
	}
	return out, nil
}

// execute 以当前管理员执行一个操作：校验目标，写 GM 记录，逐个目标调用游戏服务端。
func (uc *GakumasUsecase) execute(ctx context.Context, actor GmActor, serverID, opID string, targets []string, params any, dryRun bool, summary map[string]string) (*v1.GmExecution, []GmTargetOutcome, error) {
	s, g, err := uc.server(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	op, ok := gakumasOperation(opID)
	if !ok {
		return nil, nil, ErrGmCommandNotFound
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, nil, ErrGmParams
	}
	call, err := gakumasCall(g, opID, raw)
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
	e = uc.gm.SetBackupRef(ctx, e, backupsOf(outcomes))
	return GmExecutionProto(e, true), outcomes, nil
}

// backupsOf 操作产生的存档备份（备份与回档前的备份）。
func backupsOf(outcomes []GmTargetOutcome) []string {
	var out []string
	for _, o := range outcomes {
		switch r := o.Result.(type) {
		case *gk.Backup:
			out = append(out, itoa(int(r.GetId())))
		case *gk.RestoreResult:
			if id := r.GetKept().GetId(); id > 0 {
				out = append(out, itoa(int(id)))
			}
		}
	}
	return out
}

// Replay 审批时按记录重放：记录的操作、按保存的参数选择的函数与目标当前的昵称。
func (uc *GakumasUsecase) Replay(ctx context.Context, s *gen.GameServer, e *gen.GmExecution) (*GmOperation, GmCall, map[string]string, error) {
	if s.Game != gakumas.ID {
		return nil, nil, nil, ErrGmGameUnsupported
	}
	c, err := uc.gm.Servers().Client(s)
	if err != nil {
		return nil, nil, nil, err
	}
	g := gakumas.New(c)
	op, ok := gakumasOperation(e.Command)
	if !ok {
		return nil, nil, nil, ErrGmCommandNotFound
	}
	call, err := gakumasCall(g, e.Command, json.RawMessage(e.Params))
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
func (uc *GakumasUsecase) Approve(ctx context.Context, reviewer GmActor, serverID, id, note string) (*v1.GmExecution, error) {
	return uc.gm.Approve(ctx, reviewer, serverID, id, note, uc.Replay)
}

// SelfServicePolicies 学マス 可以开放给玩家自助的操作与设置。
func (uc *GakumasUsecase) SelfServicePolicies(ctx context.Context, serverID string) ([]*v1.GmSelfServicePolicy, error) {
	if _, err := uc.client(ctx, serverID); err != nil {
		return nil, err
	}
	list, _, err := uc.gm.SelfServicePolicies(ctx, serverID, GakumasOperations)
	return list, err
}

// UpdateSelfService 修改一个操作的玩家自助设置。
func (uc *GakumasUsecase) UpdateSelfService(ctx context.Context, actor GmActor, req *v1.UpdateGmSelfServiceRequest) (*v1.GmSelfServicePolicy, error) {
	if _, err := uc.client(ctx, req.ServerId); err != nil {
		return nil, err
	}
	return uc.gm.UpdateSelfService(ctx, actor, req, GakumasOperations)
}

// targetsOf 把逐个目标的结果转为接口结构；成功的目标带上调用返回的内容。
func targetsOf[T proto.Message, R any](outcomes []GmTargetOutcome, build func(*gk.OperationTarget, T) R) []R {
	out := make([]R, 0, len(outcomes))
	for _, o := range outcomes {
		var data T
		if v, ok := o.Result.(T); ok {
			data = v
		}
		out = append(out, build(&gk.OperationTarget{
			Pid:     o.PID,
			Name:    o.Name,
			Status:  o.Status,
			Code:    o.Code,
			Message: o.Message,
		}, data))
	}
	return out
}

func changesResponse(e *v1.GmExecution, outcomes []GmTargetOutcome) *gk.ChangesResponse {
	return &gk.ChangesResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.ChangeList) *gk.ChangesTarget {
			return &gk.ChangesTarget{
				Target: t,
				Data:   d,
			}
		}),
	}
}

func idolCardGrantResponse(e *v1.GmExecution, outcomes []GmTargetOutcome) *gk.IdolCardGrantResponse {
	return &gk.IdolCardGrantResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.IdolCardGrant) *gk.IdolCardGrantTarget {
			return &gk.IdolCardGrantTarget{
				Target: t,
				Data:   d,
			}
		}),
	}
}

func supportCardGrantResponse(e *v1.GmExecution, outcomes []GmTargetOutcome) *gk.SupportCardGrantResponse {
	return &gk.SupportCardGrantResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.SupportCardGrant) *gk.SupportCardGrantTarget {
			return &gk.SupportCardGrantTarget{
				Target: t,
				Data:   d,
			}
		}),
	}
}

func kickResponse(e *v1.GmExecution, outcomes []GmTargetOutcome) *gk.KickResponse {
	return &gk.KickResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.KickResult) *gk.KickTarget {
			return &gk.KickTarget{
				Target: t,
				Data:   d,
			}
		}),
	}
}

func banResponse(e *v1.GmExecution, outcomes []GmTargetOutcome) *gk.BanResponse {
	return &gk.BanResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.BanResult) *gk.BanTarget {
			return &gk.BanTarget{
				Target: t,
				Data:   d,
			}
		}),
	}
}

func renameResponse(e *v1.GmExecution, outcomes []GmTargetOutcome) *gk.RenameResponse {
	return &gk.RenameResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.RenameResult) *gk.RenameTarget {
			return &gk.RenameTarget{
				Target: t,
				Data:   d,
			}
		}),
	}
}

func rentalResponse(e *v1.GmExecution, outcomes []GmTargetOutcome) *gk.RentalResponse {
	return &gk.RentalResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.RentalChange) *gk.RentalTarget {
			return &gk.RentalTarget{
				Target: t,
				Data:   d,
			}
		}),
	}
}

func archiveExportResponse(e *v1.GmExecution, outcomes []GmTargetOutcome) *gk.ArchiveExportResponse {
	return &gk.ArchiveExportResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.ArchiveExport) *gk.ArchiveExportTarget {
			return &gk.ArchiveExportTarget{
				Target: t,
				Data:   d,
			}
		}),
	}
}

// ---------------- 服务器 ----------------

func (uc *GakumasUsecase) Manifest(ctx context.Context, serverID string) (*gk.Manifest, error) {
	g, err := uc.client(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.Manifest(ctx))
}

func (uc *GakumasUsecase) Status(ctx context.Context, serverID string) (*gk.Status, error) {
	g, err := uc.client(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.Status(ctx))
}

func (uc *GakumasUsecase) Dashboard(ctx context.Context, req *gk.GetDashboardRequest) (*gk.Dashboard, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Dashboard(ctx, req.Days))
}

// ---------------- 玩家 ----------------

// ListPlayers 玩家列表与通行证绑定：filter=unlinked 时排除已绑定通行证的玩家（由游戏服务端分页）；
// 第一页的关键字搜索补上通行证用户名或昵称匹配的绑定玩家。
func (uc *GakumasUsecase) ListPlayers(ctx context.Context, req *gk.ListPlayersRequest) (*gk.ListPlayersResponse, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	linked, err := uc.gm.LinkedPIDs(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	limit := req.Limit
	if limit <= 0 || limit > 200 {
		limit = 20
	}
	f := gakumas.PlayerFilter{
		Q:      req.Q,
		Filter: req.Filter,
	}
	if req.Filter == "unlinked" {
		f.Filter, f.ExcludePIDs = "all", linked
	}
	page, err := g.Players(ctx, f, req.Cursor, limit)
	if err != nil {
		return nil, GameMGPError(err)
	}
	out := &gk.ListPlayersResponse{
		Items:       page.Items,
		NextCursor:  page.NextCursor,
		Total:       page.Total,
		Summary:     page.Summary,
		LinkedCount: int32(len(linked)),
	}
	if out.Summary != nil {
		out.Unlinked = max(out.Summary.All-int32(len(linked)), 0)
	}
	if req.Q != "" && req.Cursor == "" && (req.Filter == "" || req.Filter == "all") {
		found, err := uc.gm.FindLinkedPIDs(ctx, req.ServerId, req.Q, 10)
		if err != nil {
			return nil, err
		}
		have := map[string]bool{}
		for _, p := range out.Items {
			have[p.Pid] = true
		}
		var missing []string
		for _, pid := range found {
			if !have[pid] {
				missing = append(missing, pid)
			}
		}
		if len(missing) > 0 {
			extra, err := g.Players(ctx, gakumas.PlayerFilter{PIDs: missing}, "", int32(len(missing)))
			if err != nil {
				return nil, GameMGPError(err)
			}
			out.Items = append(extra.Items, out.Items...)
			out.Total += int32(len(extra.Items))
		}
	}
	pids := make([]string, 0, len(out.Items))
	for _, p := range out.Items {
		pids = append(pids, p.Pid)
	}
	if out.Links, err = uc.gm.Links(ctx, req.ServerId, pids); err != nil {
		return nil, err
	}
	return out, nil
}

// GetPlayer 玩家条目、概览、登录次数，以及通行证绑定与 GM 记录数。
func (uc *GakumasUsecase) GetPlayer(ctx context.Context, req *gk.PlayerRequest) (*gk.GetPlayerResponse, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	p, err := g.Player(ctx, req.Pid)
	if err != nil {
		return nil, GameMGPError(err)
	}
	out := &gk.GetPlayerResponse{
		Player:   p.Player,
		Overview: p.Overview,
		Logins:   p.Logins,
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

func (uc *GakumasUsecase) ListPlayerSessions(ctx context.Context, req *gk.PageRequest) (*gk.SessionPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Sessions(ctx, req.Pid, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) ListPlayerIdolCards(ctx context.Context, req *gk.ListIdolCardsRequest) (*gk.IdolCardPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.IdolCards(ctx, req.Pid, gakumas.IdolCardFilter{
		Q:           req.Q,
		CharacterID: req.CharacterId,
		Rarity:      req.Rarity,
		IDs:         req.IdolCardIds,
	}, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) ListPlayerSupportCards(ctx context.Context, req *gk.ListSupportCardsRequest) (*gk.SupportCardPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.SupportCards(ctx, req.Pid, gakumas.SupportCardFilter{
		Q:      req.Q,
		Rarity: req.Rarity,
		IDs:    req.SupportCardIds,
	}, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) ListPlayerItems(ctx context.Context, req *gk.ListItemsRequest) (*gk.ItemPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Items(ctx, req.Pid, req.Type, req.ItemIds, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) ListPlayerGifts(ctx context.Context, req *gk.ListGiftsRequest) (*gk.GiftPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Gifts(ctx, req.Pid, gakumas.GiftFilter{
		State:       req.State,
		ResourceIDs: req.ResourceIds,
		Source:      req.Source,
		Expiring:    req.Expiring,
	}, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) ListPlayerGashaDraws(ctx context.Context, req *gk.ListGashaDrawsRequest) (*gk.GashaDrawPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.GashaDraws(ctx, req.Pid, req.GashaIds, req.CardIds, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) ListPlayerMemories(ctx context.Context, req *gk.ListMemoriesRequest) (*gk.MemoryPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Memories(ctx, req.Pid, gakumas.MemoryFilter{
		Produced:    req.Produced,
		Q:           req.Q,
		CharacterID: req.CharacterId,
		Rarity:      req.Rarity,
		Grades:      req.Grades,
		IdolCardIDs: req.IdolCardIds,
	}, req.Cursor, req.Limit))
}

// MemoryPhoto 记忆照片。
func (uc *GakumasUsecase) MemoryPhoto(ctx context.Context, req *gk.MemoryPhotoRequest) (*gk.MemoryPhoto, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.MemoryPhoto(ctx, req.Path))
}

// GetPlayerMemory 一条记忆的详情与培育记录。
func (uc *GakumasUsecase) GetPlayerMemory(ctx context.Context, req *gk.PlayerMemoryRequest) (*gk.MemoryDetail, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Memory(ctx, req.Pid, req.MemoryId))
}

// GetPlayerMeishiImage 玩家资料名片的图片。
func (uc *GakumasUsecase) GetPlayerMeishiImage(ctx context.Context, req *gk.PlayerRequest) (*gk.MeishiImage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.MeishiImage(ctx, req.Pid))
}

func (uc *GakumasUsecase) GetPlayerPvp(ctx context.Context, req *gk.PlayerRequest) (*gk.PlayerPvp, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.PlayerPvp(ctx, req.Pid))
}

func (uc *GakumasUsecase) GetPlayerSeminar(ctx context.Context, req *gk.PlayerRequest) (*gk.PlayerSeminar, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.PlayerSeminar(ctx, req.Pid))
}

func (uc *GakumasUsecase) ListPlayerBackups(ctx context.Context, req *gk.PlayerRequest) (*gk.BackupList, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Backups(ctx, req.Pid))
}

// ---------------- 全服 ----------------

func (uc *GakumasUsecase) GashaStats(ctx context.Context, serverID string) (*gk.GashaStatList, error) {
	g, err := uc.client(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.GashaStats(ctx))
}

func (uc *GakumasUsecase) ListProduces(ctx context.Context, req *gk.ListProducesRequest) (*gk.ProducePage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Produces(ctx, gakumas.ProduceFilter{
		ProduceGroupID: req.ProduceGroupId,
		ProduceID:      req.ProduceId,
		CharacterID:    req.CharacterId,
		Since:          int64(req.Since),
		Until:          int64(req.Until),
		Q:              req.Q,
	}, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) PvpRanking(ctx context.Context, req *gk.RankingRequest) (*gk.PvpRankingPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.PvpRanking(ctx, req.Q, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) SeminarStats(ctx context.Context, serverID string) (*gk.SeminarStatList, error) {
	g, err := uc.client(ctx, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.SeminarStats(ctx))
}

func (uc *GakumasUsecase) TowerRanking(ctx context.Context, req *gk.TowerRankingRequest) (*gk.TowerRankingPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.TowerRanking(ctx, req.TowerId, req.Q, req.Cursor, req.Limit))
}

// ---------------- 目录 ----------------

func (uc *GakumasUsecase) Currencies(ctx context.Context, req *gk.CatalogRequest) (*gk.CurrencyPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Currencies(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) ItemCatalog(ctx context.Context, req *gk.ItemCatalogRequest) (*gk.ItemEntryPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.ItemCatalog(ctx, req.Q, req.Ids, req.Type, req.CharacterId, req.Rarity, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) IdolCardSkins(ctx context.Context, req *gk.IdolCardSkinCatalogRequest) (*gk.IdolCardSkinPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.IdolCardSkins(ctx, req.Q, req.Ids, req.CharacterId, req.Rarity, req.PlanType, req.IdolCardIds, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) SupportCardCatalog(ctx context.Context, req *gk.SupportCardCatalogRequest) (*gk.SupportCardEntryPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.SupportCardCatalog(ctx, req.Q, req.Ids, req.Rarity, req.Type, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) Characters(ctx context.Context, req *gk.CatalogRequest) (*gk.CharacterPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Characters(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) Costumes(ctx context.Context, req *gk.CostumeCatalogRequest) (*gk.CostumePage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Costumes(ctx, req.Q, req.Ids, req.CharacterId, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) CostumeHeads(ctx context.Context, req *gk.CostumeCatalogRequest) (*gk.CostumePage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.CostumeHeads(ctx, req.Q, req.Ids, req.CharacterId, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) Music(ctx context.Context, req *gk.CatalogRequest) (*gk.MusicPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Music(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) Gashas(ctx context.Context, req *gk.GashaCatalogRequest) (*gk.GashaEntryPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Gashas(ctx, req.Q, req.Ids, req.Open, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) ProduceGroups(ctx context.Context, req *gk.CatalogRequest) (*gk.ProduceGroupPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.ProduceGroups(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) ProduceCatalog(ctx context.Context, req *gk.ProduceCatalogRequest) (*gk.ProduceEntryPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.ProduceCatalog(ctx, req.Q, req.Ids, req.ProduceGroupId, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) Seminars(ctx context.Context, req *gk.CatalogRequest) (*gk.SeminarEntryPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Seminars(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) ProduceCards(ctx context.Context, req *gk.CatalogRequest) (*gk.ProduceCardEntryPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.ProduceCards(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) ProduceItems(ctx context.Context, req *gk.CatalogRequest) (*gk.ProduceItemEntryPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.ProduceItems(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) MemoryAbilities(ctx context.Context, req *gk.CatalogRequest) (*gk.MemoryAbilityPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.MemoryAbilities(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasUsecase) Towers(ctx context.Context, req *gk.CatalogRequest) (*gk.TowerEntryPage, error) {
	g, err := uc.client(ctx, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Towers(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

// ---------------- 操作（管理员） ----------------

func one(pid string) []string { return []string{pid} }

func (uc *GakumasUsecase) SendGift(ctx context.Context, actor GmActor, req *gk.SendGiftRequest) (*gk.GiftSendResponse, error) {
	gift := gakumas.GiftSend{
		Rewards:    make([]gakumas.Reward, 0, len(req.Rewards)),
		Message:    strings.TrimSpace(req.Message),
		ExpireDays: req.ExpireDays,
	}
	for _, r := range req.Rewards {
		gift.Rewards = append(gift.Rewards, gakumas.Reward{
			ResourceType: r.ResourceType,
			ResourceID:   r.ResourceId,
			Quantity:     r.Quantity,
		})
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkGiftSend, req.Pids, gift, req.DryRun, req.Summary)
	if err != nil {
		return nil, err
	}
	return &gk.GiftSendResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.GiftSendResult) *gk.GiftSendTarget {
			return &gk.GiftSendTarget{
				Target: t,
				Data:   d,
			}
		}),
	}, nil
}

func (uc *GakumasUsecase) RevokeGifts(ctx context.Context, actor GmActor, req *gk.RevokeGiftsRequest) (*gk.GiftRevokeResponse, error) {
	revoke := gakumas.GiftRevoke{
		SourceKind: req.SourceKind,
		SourceID:   req.SourceId,
		GiftIDs:    req.GiftIds,
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkGiftRevoke, req.Pids, revoke, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return &gk.GiftRevokeResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.GiftRevokeResult) *gk.GiftRevokeTarget {
			return &gk.GiftRevokeTarget{
				Target: t,
				Data:   d,
			}
		}),
	}, nil
}

func (uc *GakumasUsecase) GrantAllIdolCards(ctx context.Context, actor GmActor, req *gk.GrantAllRequest) (*gk.IdolCardGrantResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkIdolCardGrantAll, req.Pids, gkMessage{Message: req.Message}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return idolCardGrantResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) GrantAllSupportCards(ctx context.Context, actor GmActor, req *gk.GrantAllRequest) (*gk.SupportCardGrantResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkSupportGrantAll, req.Pids, gkMessage{Message: req.Message}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return supportCardGrantResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) AdjustItem(ctx context.Context, actor GmActor, req *gk.AdjustItemRequest) (*gk.ChangesResponse, error) {
	p := gkItemParams{
		ItemID:   req.ItemId,
		Quantity: req.Quantity,
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkItemAdjust, req.Pids, p, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return changesResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) AdjustJewel(ctx context.Context, actor GmActor, req *gk.AdjustJewelRequest) (*gk.ChangesResponse, error) {
	p := gkJewelParams{
		Free: req.Free,
		Paid: req.Paid,
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkJewelAdjust, req.Pids, p, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return changesResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) RefillActionPoint(ctx context.Context, actor GmActor, req *gk.PlayersOperationRequest) (*gk.ChangesResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkAPRefill, req.Pids, struct{}{}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return changesResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) AddExp(ctx context.Context, actor GmActor, req *gk.AddExpRequest) (*gk.ChangesResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkExpAdd, req.Pids, gkExp{Exp: req.Exp}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return changesResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) KickPlayers(ctx context.Context, actor GmActor, req *gk.PlayersOperationRequest) (*gk.KickResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkPlayerKick, req.Pids, struct{}{}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return kickResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) BanPlayers(ctx context.Context, actor GmActor, req *gk.BanPlayersRequest) (*gk.BanResponse, error) {
	p := gkBan{
		Until:     int64(req.Until),
		Permanent: req.Permanent,
		Reason:    strings.TrimSpace(req.Reason),
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkPlayerBan, req.Pids, p, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return banResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) UnbanPlayers(ctx context.Context, actor GmActor, req *gk.PlayersOperationRequest) (*gk.BanResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkPlayerUnban, req.Pids, struct{}{}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return banResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) ResetPassword(ctx context.Context, actor GmActor, req *gk.ResetPasswordRequest) (*gk.PasswordResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkPlayerPassword, req.Pids, gkPassword{Password: req.Password}, false, nil)
	if err != nil {
		return nil, err
	}
	return &gk.PasswordResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.PasswordResult) *gk.PasswordTarget {
			return &gk.PasswordTarget{
				Target: t,
				Data:   d,
			}
		}),
	}, nil
}

func (uc *GakumasUsecase) RenamePlayer(ctx context.Context, actor GmActor, req *gk.RenamePlayerRequest) (*gk.RenameResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkPlayerRename, req.Pids, gkName{Name: req.Name}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return renameResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) SetRentalSupportCard(ctx context.Context, actor GmActor, req *gk.SetRentalSupportCardRequest) (*gk.RentalResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkSupportRental, one(req.Pid), gkSupportCardID{SupportCardID: req.SupportCardId}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return rentalResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) SetRentalMemory(ctx context.Context, actor GmActor, req *gk.SetRentalMemoryRequest) (*gk.RentalResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkMemoryRental, one(req.Pid), gkMemoryID{MemoryID: req.MemoryId}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return rentalResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) SetPvpRate(ctx context.Context, actor GmActor, req *gk.SetPvpRateRequest) (*gk.ChangesResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkPvpSetRate, req.Pids, gkRate{Rate: req.Rate}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return changesResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) ResetPvpDaily(ctx context.Context, actor GmActor, req *gk.PlayersOperationRequest) (*gk.ChangesResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkPvpResetDaily, req.Pids, struct{}{}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return changesResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) ClearSeminar(ctx context.Context, actor GmActor, req *gk.PlayersOperationRequest) (*gk.SeminarClearResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkSeminarClear, req.Pids, struct{}{}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return &gk.SeminarClearResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.SeminarClearResult) *gk.SeminarClearTarget {
			return &gk.SeminarClearTarget{
				Target: t,
				Data:   d,
			}
		}),
	}, nil
}

func (uc *GakumasUsecase) ResetSeminar(ctx context.Context, actor GmActor, req *gk.PlayersOperationRequest) (*gk.SeminarResetResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkSeminarReset, req.Pids, struct{}{}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return &gk.SeminarResetResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.SeminarResetResult) *gk.SeminarResetTarget {
			return &gk.SeminarResetTarget{
				Target: t,
				Data:   d,
			}
		}),
	}, nil
}

func (uc *GakumasUsecase) ExportArchive(ctx context.Context, actor GmActor, req *gk.PlayersOperationRequest) (*gk.ArchiveExportResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkArchiveExport, req.Pids, struct{}{}, false, nil)
	if err != nil {
		return nil, err
	}
	return archiveExportResponse(e, outcomes), nil
}

func (uc *GakumasUsecase) BackupArchive(ctx context.Context, actor GmActor, req *gk.BackupArchiveRequest) (*gk.BackupResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkArchiveBackup, one(req.Pid), gkNote{Note: strings.TrimSpace(req.Note)}, false, nil)
	if err != nil {
		return nil, err
	}
	return &gk.BackupResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.Backup) *gk.BackupTarget {
			return &gk.BackupTarget{
				Target: t,
				Data:   d,
			}
		}),
	}, nil
}

func (uc *GakumasUsecase) RestoreArchive(ctx context.Context, actor GmActor, req *gk.RestoreArchiveRequest) (*gk.RestoreResponse, error) {
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkArchiveRestore, one(req.Pid), gkRestore{BackupID: int64(req.BackupId)}, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	return &gk.RestoreResponse{
		Execution: e,
		Results: targetsOf(outcomes, func(t *gk.OperationTarget, d *gk.RestoreResult) *gk.RestoreTarget {
			return &gk.RestoreTarget{
				Target: t,
				Data:   d,
			}
		}),
	}, nil
}

func (uc *GakumasUsecase) SetMaintenance(ctx context.Context, actor GmActor, req *gk.SetMaintenanceRequest) (*gk.MaintenanceResponse, error) {
	p := gkMaintenanceParams{
		Enabled:   req.Enabled,
		Whitelist: req.Whitelist,
	}
	e, outcomes, err := uc.execute(ctx, actor, req.ServerId, gkMaintenance, nil, p, req.DryRun, nil)
	if err != nil {
		return nil, err
	}
	out := &gk.MaintenanceResponse{
		Execution: e,
	}
	if len(outcomes) > 0 {
		out.Data, _ = outcomes[0].Result.(*gk.Maintenance)
	}
	return out, nil
}
