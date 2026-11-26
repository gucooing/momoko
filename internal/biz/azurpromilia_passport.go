package biz

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/go-kratos/kratos/v2/log"

	v1 "momoko/api/gen/v1"
	az "momoko/api/gen/v1/azurpromilia"
	"momoko/pkg/mgp/azurpromilia"
)

// AzurPromiliaPassportUsecase 蓝色星原的用户通行证游戏空间（玩家端，AzurPromiliaPassportService）：
// 只针对当前用户在服务器上绑定的游戏账号；身份与绑定由 PassportUsecase 判定，游戏数据经 pkg/mgp/azurpromilia 读取。
type AzurPromiliaPassportUsecase struct {
	servers  *GameServerUsecase
	passport *PassportUsecase
	log      *log.Helper
}

func NewAzurPromiliaPassportUsecase(servers *GameServerUsecase, passport *PassportUsecase, logger log.Logger) *AzurPromiliaPassportUsecase {
	return &AzurPromiliaPassportUsecase{
		servers:  servers,
		passport: passport,
		log:      log.NewHelper(log.With(logger, "module", "biz/azurpromilia_passport")),
	}
}

// mine 当前用户绑定的游戏账号与运行蓝色星原的服务器的 MGP 客户端。
func (uc *AzurPromiliaPassportUsecase) mine(ctx context.Context, userID, serverID string) (*azurpromilia.Client, string, error) {
	pid, err := uc.passport.LinkedPID(ctx, userID, serverID)
	if err != nil {
		return nil, "", err
	}
	s, c, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, "", err
	}
	if s.Game != azurpromilia.ID {
		return nil, "", ErrGmGameUnsupported
	}
	return azurpromilia.New(c), pid, nil
}

// MyPlayer 我的玩家条目与概览。
func (uc *AzurPromiliaPassportUsecase) MyPlayer(ctx context.Context, userID, serverID string) (*az.PlayerDetail, error) {
	g, pid, err := uc.mine(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.Player(ctx, pid))
}

// MyLogins 我的登录记录（按登录时间倒序）。
func (uc *AzurPromiliaPassportUsecase) MyLogins(ctx context.Context, userID string, req *az.MyLoginsRequest) (*az.LoginPage, error) {
	g, pid, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Logins(ctx, pid, req.Cursor, req.Limit))
}

func (uc *AzurPromiliaPassportUsecase) MyItems(ctx context.Context, userID string, req *az.MyLoginsRequest) (*az.ItemPage, error) {
	g, pid, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Items(ctx, pid, nil, req.Cursor, req.Limit))
}

func (uc *AzurPromiliaPassportUsecase) MyCurrencies(ctx context.Context, userID, serverID string) (*az.ItemPage, error) {
	g, pid, err := uc.mine(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.Currencies(ctx, pid))
}

func (uc *AzurPromiliaPassportUsecase) MyHeroes(ctx context.Context, userID string, req *az.MyLoginsRequest) (*az.HeroPage, error) {
	g, pid, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Heroes(ctx, pid, req.Cursor, req.Limit))
}

func (uc *AzurPromiliaPassportUsecase) MyPets(ctx context.Context, userID string, req *az.MyLoginsRequest) (*az.PetPage, error) {
	g, pid, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Pets(ctx, pid, req.Cursor, req.Limit))
}

func (uc *AzurPromiliaPassportUsecase) MyMails(ctx context.Context, userID string, req *az.MyLoginsRequest) (*az.MailPage, error) {
	g, pid, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Mails(ctx, pid, req.Cursor, req.Limit))
}

// AzurPromiliaSelfService 蓝色星原玩家自助的操作（AzurPromiliaOperations 中可以开放给玩家的，管理员在「玩家权限」里开放后才出现）。
var AzurPromiliaSelfService = azurpromiliaSelfService()

func azurpromiliaSelfService() []PassportOperation {
	var out []PassportOperation
	for _, op := range AzurPromiliaOperations {
		if op.SelfService {
			out = append(out, PassportOperation{ID: op.ID, Title: op.Title, Group: op.Group, Risk: op.Risk, Target: TargetPlayer})
		}
	}
	return out
}

func azurpromiliaSelfOperation(id string) PassportOperation {
	for _, op := range AzurPromiliaSelfService {
		if op.ID == id {
			return op
		}
	}
	return PassportOperation{ID: id}
}

// SelfService 我能执行的自助操作与它们的规则、使用情况。
func (uc *AzurPromiliaPassportUsecase) SelfService(ctx context.Context, userID, serverID string) (*v1.ListPassportSelfServiceResponse, error) {
	if _, _, err := uc.mine(ctx, userID, serverID); err != nil {
		return nil, err
	}
	return uc.passport.SelfService(ctx, userID, serverID, AzurPromiliaSelfService)
}

// run 按玩家自助规则对我绑定的游戏账号执行一个操作（参数与管理端相同，审批时按它重放）：需要审批时只提交申请（没有内容）。
func (uc *AzurPromiliaPassportUsecase) run(ctx context.Context, userID, serverID, opID string, params any, reason, totpCode string) (*az.MySelfServiceResponse, error) {
	g, _, err := uc.mine(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(params)
	if err != nil {
		return nil, ErrGmParams
	}
	call, err := azurpromiliaCall(g, opID, raw)
	if err != nil {
		return nil, err
	}
	e, data, err := uc.passport.RunSelfService(ctx, userID, serverID, azurpromiliaSelfOperation(opID), params, strings.TrimSpace(reason), totpCode, PassportCall(call))
	if err != nil {
		return nil, err
	}
	out := &az.MySelfServiceResponse{
		Result: &az.SelfServiceResult{
			RecordId:     e.ID,
			Status:       string(e.Status),
			ErrorCode:    e.ErrorCode,
			ErrorMessage: e.ErrorMessage,
		},
	}
	if d, ok := data.(*az.OperationData); ok {
		out.Data = d
	}
	return out, nil
}

// MailMe 给我的游戏账号发一封邮件（含附件）。
func (uc *AzurPromiliaPassportUsecase) MailMe(ctx context.Context, userID string, req *az.SelfServiceMailRequest) (*az.MySelfServiceResponse, error) {
	title := strings.TrimSpace(req.Title)
	if title == "" {
		return nil, ErrGmParams
	}
	return uc.run(ctx, userID, req.ServerId, azPlayerMail, azMailParams{Mail: &az.MailContent{
		Sender:  "",
		Title:   title,
		Content: req.Content,
		Rewards: req.Rewards,
		Effect:  0,
	}}, req.Reason, req.TotpCode)
}

func (uc *AzurPromiliaPassportUsecase) AdjustMyItems(ctx context.Context, userID string, req *az.SelfServiceItemsRequest) (*az.MySelfServiceResponse, error) {
	if len(req.Grant)+len(req.Take) == 0 {
		return nil, ErrGmParams
	}
	return uc.run(ctx, userID, req.ServerId, azItemAdjust, azItemsParams{Grant: req.Grant, Take: req.Take}, req.Reason, req.TotpCode)
}

func (uc *AzurPromiliaPassportUsecase) KickMe(ctx context.Context, userID string, req *az.SelfServiceRequest) (*az.MySelfServiceResponse, error) {
	return uc.run(ctx, userID, req.ServerId, azPlayerKick, struct{}{}, req.Reason, req.TotpCode)
}

func (uc *AzurPromiliaPassportUsecase) BanMe(ctx context.Context, userID string, req *az.SelfServiceBanRequest) (*az.MySelfServiceResponse, error) {
	return uc.run(ctx, userID, req.ServerId, azPlayerBan, azBanParams{Until: int64(req.Until), Reason: strings.TrimSpace(req.Reason)}, req.Reason, req.TotpCode)
}

func (uc *AzurPromiliaPassportUsecase) UnbanMe(ctx context.Context, userID string, req *az.SelfServiceRequest) (*az.MySelfServiceResponse, error) {
	return uc.run(ctx, userID, req.ServerId, azPlayerUnban, struct{}{}, req.Reason, req.TotpCode)
}

func (uc *AzurPromiliaPassportUsecase) SetMyPassword(ctx context.Context, userID string, req *az.SelfServicePasswordRequest) (*az.MySelfServiceResponse, error) {
	if req.Password == "" {
		return nil, ErrGmParams
	}
	return uc.run(ctx, userID, req.ServerId, azPlayerPassword, azPasswordParams{Password: req.Password}, req.Reason, req.TotpCode)
}

func (uc *AzurPromiliaPassportUsecase) RenameMe(ctx context.Context, userID string, req *az.SelfServiceRenameRequest) (*az.MySelfServiceResponse, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, ErrGmParams
	}
	return uc.run(ctx, userID, req.ServerId, azPlayerRename, azNameParams{Name: name}, req.Reason, req.TotpCode)
}
