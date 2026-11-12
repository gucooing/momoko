package biz

import (
	"context"
	"strings"

	"github.com/go-kratos/kratos/v2/log"
	"github.com/google/uuid"
	"google.golang.org/protobuf/proto"

	v1 "momoko/api/gen/v1"
	gk "momoko/api/gen/v1/gakumas"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/mgp"
	"momoko/pkg/mgp/gakumas"
)

// GakumasPassportUsecase 学マス 的用户通行证游戏空间（玩家端，GakumasPassportService）：
// 只针对当前用户在服务器上绑定的游戏账号；身份与绑定、自助规则由 PassportUsecase 判定，游戏数据经 pkg/mgp/gakumas 读取。
// 与管理端的 GakumasUsecase 互不调用。
type GakumasPassportUsecase struct {
	servers  *GameServerUsecase
	passport *PassportUsecase
	log      *log.Helper
}

func NewGakumasPassportUsecase(servers *GameServerUsecase, passport *PassportUsecase, logger log.Logger) *GakumasPassportUsecase {
	return &GakumasPassportUsecase{
		servers:  servers,
		passport: passport,
		log:      log.NewHelper(log.With(logger, "module", "biz/gakumas_passport")),
	}
}

// GakumasSelfService 学マス 玩家自助的操作（GakumasOperations 中可以开放给玩家的，管理员在「玩家权限」里开放后才出现）：
// 目标为我绑定的游戏账号；标题、分组与风险写进 GM 记录，高风险操作在开启了两步验证时要验证码。
var GakumasSelfService = gakumasSelfService()

func gakumasSelfService() []PassportOperation {
	var out []PassportOperation
	for _, op := range GakumasOperations {
		if op.SelfService {
			out = append(out, PassportOperation{ID: op.ID, Title: op.Title, Group: op.Group, Risk: op.Risk, Target: TargetPlayer})
		}
	}
	return out
}

func gakumasSelfOperation(id string) PassportOperation {
	for _, op := range GakumasSelfService {
		if op.ID == id {
			return op
		}
	}
	return PassportOperation{ID: id}
}

// selfCall 对我绑定的游戏账号调用一个 pkg/mgp/gakumas 的操作函数。
type selfCall func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error)

// client 运行学マス 的服务器的 MGP 客户端。
func (uc *GakumasPassportUsecase) client(ctx context.Context, serverID string) (*gakumas.Client, error) {
	s, c, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, err
	}
	if s.Game != gakumas.ID {
		return nil, ErrGmGameUnsupported
	}
	return gakumas.New(c), nil
}

// mine 当前用户绑定的游戏账号与服务器的 MGP 客户端。
func (uc *GakumasPassportUsecase) mine(ctx context.Context, userID, serverID string) (*gakumas.Client, string, error) {
	pid, err := uc.passport.LinkedPID(ctx, userID, serverID)
	if err != nil {
		return nil, "", err
	}
	g, err := uc.client(ctx, serverID)
	if err != nil {
		return nil, "", err
	}
	return g, pid, nil
}

// MyGame 游戏与已配置的玩法（玩家看不到服务端与资源版本）。
func (uc *GakumasPassportUsecase) MyGame(ctx context.Context, userID, serverID string) (*gk.PlayerGame, error) {
	g, _, err := uc.mine(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	m, err := g.Manifest(ctx)
	if err != nil {
		return nil, GameMGPError(err)
	}
	return &gk.PlayerGame{
		Game:  m.Game,
		Modes: m.Modes,
	}, nil
}

func (uc *GakumasPassportUsecase) MyPlayer(ctx context.Context, userID, serverID string) (*gk.PlayerDetail, error) {
	g, pid, err := uc.mine(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.Player(ctx, pid))
}

func (uc *GakumasPassportUsecase) MyIdolCards(ctx context.Context, userID string, req *gk.MyIdolCardsRequest) (*gk.IdolCardPage, error) {
	g, pid, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.IdolCards(ctx, pid, gakumas.IdolCardFilter{
		Q:           req.Q,
		CharacterID: req.CharacterId,
		Rarity:      req.Rarity,
		IDs:         req.IdolCardIds,
	}, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) MySupportCards(ctx context.Context, userID string, req *gk.MySupportCardsRequest) (*gk.SupportCardPage, error) {
	g, pid, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.SupportCards(ctx, pid, gakumas.SupportCardFilter{
		Q:      req.Q,
		Rarity: req.Rarity,
		IDs:    req.SupportCardIds,
	}, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) MyGifts(ctx context.Context, userID string, req *gk.MyGiftsRequest) (*gk.GiftPage, error) {
	g, pid, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Gifts(ctx, pid, gakumas.GiftFilter{
		State:    req.State,
		Source:   req.Source,
		Expiring: req.Expiring,
	}, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) MyGashaDraws(ctx context.Context, userID string, req *gk.MyGashaDrawsRequest) (*gk.GashaDrawPage, error) {
	g, pid, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.GashaDraws(ctx, pid, req.GashaIds, req.CardIds, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) MyMemories(ctx context.Context, userID string, req *gk.MyMemoriesRequest) (*gk.MemoryPage, error) {
	g, pid, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Memories(ctx, pid, gakumas.MemoryFilter{
		Produced:    req.Produced,
		Q:           req.Q,
		Rarity:      req.Rarity,
		Grades:      req.Grades,
		IdolCardIDs: req.IdolCardIds,
	}, req.Cursor, req.Limit))
}

// MemoryPhoto 记忆照片（绑定了这台服务器的玩家可查看，与游戏里一样）。
func (uc *GakumasPassportUsecase) MemoryPhoto(ctx context.Context, userID string, req *gk.MyMemoryPhotoRequest) (*gk.MemoryPhoto, error) {
	g, _, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.MemoryPhoto(ctx, req.Path))
}

// MyMemory 一条记忆的详情与培育记录。
func (uc *GakumasPassportUsecase) MyMemory(ctx context.Context, userID string, req *gk.MyMemoryRequest) (*gk.MemoryDetail, error) {
	g, pid, err := uc.mine(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Memory(ctx, pid, req.MemoryId))
}

// settingOptions 玩家对自己数据的设置（援助，与游戏内相同）的操作选项：
// 执行者为当前用户（通行证显示名），每次请求一个新的幂等键。
func (uc *GakumasPassportUsecase) settingOptions(ctx context.Context, userID, serverID string) (*gakumas.Client, string, mgp.OperationOptions, error) {
	pid, operator, err := uc.passport.LinkedPlayer(ctx, userID, serverID)
	if err != nil {
		return nil, "", mgp.OperationOptions{}, err
	}
	g, err := uc.client(ctx, serverID)
	if err != nil {
		return nil, "", mgp.OperationOptions{}, err
	}
	return g, pid, mgp.OperationOptions{
		Operator:       operator,
		IdempotencyKey: "passport-" + uuid.NewString(),
	}, nil
}

// SetMyRentalMemory 设为援助（资料展示并出借）的记忆，与游戏内 Profile/UpdateMemory 相同。
func (uc *GakumasPassportUsecase) SetMyRentalMemory(ctx context.Context, userID string, req *gk.MyRentalMemoryRequest) (*gk.RentalChange, error) {
	g, pid, opt, err := uc.settingOptions(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.SetRentalMemory(ctx, pid, req.MemoryId, opt))
}

// SetMyRentalSupportCard 设为援助（资料展示并出借）的支援卡，与游戏内 Profile/UpdateSupportCard 相同。
func (uc *GakumasPassportUsecase) SetMyRentalSupportCard(ctx context.Context, userID string, req *gk.MyRentalSupportCardRequest) (*gk.RentalChange, error) {
	g, pid, opt, err := uc.settingOptions(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.SetRentalSupportCard(ctx, pid, req.SupportCardId, opt))
}

// MyMeishiImage 我的资料名片的图片。
func (uc *GakumasPassportUsecase) MyMeishiImage(ctx context.Context, userID, serverID string) (*gk.MeishiImage, error) {
	g, pid, err := uc.mine(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.MeishiImage(ctx, pid))
}

func (uc *GakumasPassportUsecase) MyPvp(ctx context.Context, userID, serverID string) (*gk.PlayerPvp, error) {
	g, pid, err := uc.mine(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.PlayerPvp(ctx, pid))
}

func (uc *GakumasPassportUsecase) MySeminar(ctx context.Context, userID, serverID string) (*gk.PlayerSeminar, error) {
	g, pid, err := uc.mine(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.PlayerSeminar(ctx, pid))
}

// OngoingGashas 现在开放的卡池。
func (uc *GakumasPassportUsecase) OngoingGashas(ctx context.Context, userID, serverID string) (*gk.GashaEntryPage, error) {
	g, _, err := uc.mine(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	return result(g.Gashas(ctx, "", nil, true, "", 50))
}

// ---------------- 目录（当前用户绑定了这台服务器） ----------------

func (uc *GakumasPassportUsecase) catalog(ctx context.Context, userID, serverID string) (*gakumas.Client, error) {
	g, _, err := uc.mine(ctx, userID, serverID)
	return g, err
}

func (uc *GakumasPassportUsecase) Currencies(ctx context.Context, userID string, req *gk.MyCatalogRequest) (*gk.CurrencyPage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Currencies(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) ItemCatalog(ctx context.Context, userID string, req *gk.MyItemCatalogRequest) (*gk.ItemEntryPage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.ItemCatalog(ctx, req.Q, req.Ids, req.Type, req.CharacterId, req.Rarity, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) IdolCardSkins(ctx context.Context, userID string, req *gk.MyIdolCardSkinCatalogRequest) (*gk.IdolCardSkinPage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.IdolCardSkins(ctx, req.Q, req.Ids, req.CharacterId, req.Rarity, req.PlanType, req.IdolCardIds, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) SupportCardCatalog(ctx context.Context, userID string, req *gk.MySupportCardCatalogRequest) (*gk.SupportCardEntryPage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.SupportCardCatalog(ctx, req.Q, req.Ids, req.Rarity, req.Type, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) Characters(ctx context.Context, userID string, req *gk.MyCatalogRequest) (*gk.CharacterPage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Characters(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) Costumes(ctx context.Context, userID string, req *gk.MyCostumeCatalogRequest) (*gk.CostumePage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Costumes(ctx, req.Q, req.Ids, req.CharacterId, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) CostumeHeads(ctx context.Context, userID string, req *gk.MyCostumeCatalogRequest) (*gk.CostumePage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.CostumeHeads(ctx, req.Q, req.Ids, req.CharacterId, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) Music(ctx context.Context, userID string, req *gk.MyCatalogRequest) (*gk.MusicPage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Music(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) ProduceCatalog(ctx context.Context, userID string, req *gk.MyProduceCatalogRequest) (*gk.ProduceEntryPage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.ProduceCatalog(ctx, req.Q, req.Ids, req.ProduceGroupId, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) ProduceCards(ctx context.Context, userID string, req *gk.MyCatalogRequest) (*gk.ProduceCardEntryPage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.ProduceCards(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) ProduceItems(ctx context.Context, userID string, req *gk.MyCatalogRequest) (*gk.ProduceItemEntryPage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.ProduceItems(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) MemoryAbilities(ctx context.Context, userID string, req *gk.MyCatalogRequest) (*gk.MemoryAbilityPage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.MemoryAbilities(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

func (uc *GakumasPassportUsecase) Seminars(ctx context.Context, userID string, req *gk.MyCatalogRequest) (*gk.SeminarEntryPage, error) {
	g, err := uc.catalog(ctx, userID, req.ServerId)
	if err != nil {
		return nil, err
	}
	return result(g.Seminars(ctx, req.Q, req.Ids, req.Cursor, req.Limit))
}

// ---------------- 自助操作 ----------------

// SelfService 已开放的自助操作与我的使用情况。
func (uc *GakumasPassportUsecase) SelfService(ctx context.Context, userID, serverID string) (*v1.ListPassportSelfServiceResponse, error) {
	if _, _, err := uc.mine(ctx, userID, serverID); err != nil {
		return nil, err
	}
	return uc.passport.SelfService(ctx, userID, serverID, GakumasSelfService)
}

// run 按玩家自助规则对我绑定的游戏账号执行一个操作：需要审批时只提交申请（没有内容）。
func (uc *GakumasPassportUsecase) run(ctx context.Context, userID, serverID, opID string, params any, reason, totpCode string, call selfCall) (*gk.SelfServiceResult, proto.Message, error) {
	g, err := uc.client(ctx, serverID)
	if err != nil {
		return nil, nil, err
	}
	e, data, err := uc.passport.RunSelfService(ctx, userID, serverID, gakumasSelfOperation(opID), params, reason, totpCode,
		func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return call(g, ctx, pid, opt)
		})
	if err != nil {
		return nil, nil, err
	}
	return selfServiceResult(e), data, nil
}

// selfServiceResult 自助操作的处理情况（只有记录 id、状态与失败原因）。
func selfServiceResult(e *gen.GmExecution) *gk.SelfServiceResult {
	return &gk.SelfServiceResult{
		RecordId:     e.ID,
		Status:       string(e.Status),
		ErrorCode:    e.ErrorCode,
		ErrorMessage: e.ErrorMessage,
	}
}

// dataOf 自助操作返回的内容（失败与待审批时为空）。
func dataOf[T proto.Message](m proto.Message) T {
	v, _ := m.(T)
	return v
}

func (uc *GakumasPassportUsecase) RefillMyActionPoint(ctx context.Context, userID string, req *gk.SelfServiceRequest) (*gk.MyChangesResponse, error) {
	res, data, err := uc.run(ctx, userID, req.ServerId, gkAPRefill, struct{}{}, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.RefillActionPoint(ctx, pid, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyChangesResponse{Result: res, Data: dataOf[*gk.ChangeList](data)}, nil
}

func (uc *GakumasPassportUsecase) GrantMyIdolCards(ctx context.Context, userID string, req *gk.SelfServiceGrantRequest) (*gk.MyIdolCardGrantResponse, error) {
	p := gkMessage{Message: strings.TrimSpace(req.Message)}
	res, data, err := uc.run(ctx, userID, req.ServerId, gkIdolCardGrantAll, p, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.GrantAllIdolCards(ctx, pid, p.Message, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyIdolCardGrantResponse{Result: res, Data: dataOf[*gk.IdolCardGrant](data)}, nil
}

func (uc *GakumasPassportUsecase) GrantMySupportCards(ctx context.Context, userID string, req *gk.SelfServiceGrantRequest) (*gk.MySupportCardGrantResponse, error) {
	p := gkMessage{Message: strings.TrimSpace(req.Message)}
	res, data, err := uc.run(ctx, userID, req.ServerId, gkSupportGrantAll, p, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.GrantAllSupportCards(ctx, pid, p.Message, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MySupportCardGrantResponse{Result: res, Data: dataOf[*gk.SupportCardGrant](data)}, nil
}

func (uc *GakumasPassportUsecase) KickMe(ctx context.Context, userID string, req *gk.SelfServiceRequest) (*gk.MyKickResponse, error) {
	res, data, err := uc.run(ctx, userID, req.ServerId, gkPlayerKick, struct{}{}, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Kick(ctx, pid, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyKickResponse{Result: res, Data: dataOf[*gk.KickResult](data)}, nil
}

func (uc *GakumasPassportUsecase) RenameMe(ctx context.Context, userID string, req *gk.SelfServiceRenameRequest) (*gk.MyRenameResponse, error) {
	p := gkName{Name: req.Name}
	res, data, err := uc.run(ctx, userID, req.ServerId, gkPlayerRename, p, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Rename(ctx, pid, p.Name, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyRenameResponse{Result: res, Data: dataOf[*gk.RenameResult](data)}, nil
}

func (uc *GakumasPassportUsecase) ResetMyPvpDaily(ctx context.Context, userID string, req *gk.SelfServiceRequest) (*gk.MyChangesResponse, error) {
	res, data, err := uc.run(ctx, userID, req.ServerId, gkPvpResetDaily, struct{}{}, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.ResetPvpDaily(ctx, pid, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyChangesResponse{Result: res, Data: dataOf[*gk.ChangeList](data)}, nil
}

func (uc *GakumasPassportUsecase) ExportMyArchive(ctx context.Context, userID string, req *gk.SelfServiceRequest) (*gk.MyArchiveExportResponse, error) {
	res, data, err := uc.run(ctx, userID, req.ServerId, gkArchiveExport, struct{}{}, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, _ mgp.OperationOptions) (proto.Message, error) {
			return g.ExportArchive(ctx, pid)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyArchiveExportResponse{Result: res, Data: dataOf[*gk.ArchiveExport](data)}, nil
}

func (uc *GakumasPassportUsecase) SendMyGift(ctx context.Context, userID string, req *gk.SelfServiceGiftRequest) (*gk.MyGiftSendResponse, error) {
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
	res, data, err := uc.run(ctx, userID, req.ServerId, gkGiftSend, gift, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SendGift(ctx, pid, gift, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyGiftSendResponse{Result: res, Data: dataOf[*gk.GiftSendResult](data)}, nil
}

func (uc *GakumasPassportUsecase) RevokeMyGifts(ctx context.Context, userID string, req *gk.SelfServiceGiftRevokeRequest) (*gk.MyGiftRevokeResponse, error) {
	revoke := gakumas.GiftRevoke{
		SourceKind: req.SourceKind,
		SourceID:   strings.TrimSpace(req.SourceId),
		GiftIDs:    req.GiftIds,
	}
	res, data, err := uc.run(ctx, userID, req.ServerId, gkGiftRevoke, revoke, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.RevokeGifts(ctx, pid, revoke, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyGiftRevokeResponse{Result: res, Data: dataOf[*gk.GiftRevokeResult](data)}, nil
}

func (uc *GakumasPassportUsecase) AdjustMyJewel(ctx context.Context, userID string, req *gk.SelfServiceJewelRequest) (*gk.MyChangesResponse, error) {
	p := gkJewelParams{Free: req.Free, Paid: req.Paid}
	res, data, err := uc.run(ctx, userID, req.ServerId, gkJewelAdjust, p, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.AdjustJewel(ctx, pid, p.Free, p.Paid, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyChangesResponse{Result: res, Data: dataOf[*gk.ChangeList](data)}, nil
}

func (uc *GakumasPassportUsecase) AdjustMyItem(ctx context.Context, userID string, req *gk.SelfServiceItemRequest) (*gk.MyChangesResponse, error) {
	p := gkItemParams{ItemID: strings.TrimSpace(req.ItemId), Quantity: req.Quantity}
	res, data, err := uc.run(ctx, userID, req.ServerId, gkItemAdjust, p, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.AdjustItem(ctx, pid, p.ItemID, p.Quantity, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyChangesResponse{Result: res, Data: dataOf[*gk.ChangeList](data)}, nil
}

func (uc *GakumasPassportUsecase) AddMyExp(ctx context.Context, userID string, req *gk.SelfServiceExpRequest) (*gk.MyChangesResponse, error) {
	p := gkExp{Exp: req.Exp}
	res, data, err := uc.run(ctx, userID, req.ServerId, gkExpAdd, p, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.AddExp(ctx, pid, p.Exp, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyChangesResponse{Result: res, Data: dataOf[*gk.ChangeList](data)}, nil
}

func (uc *GakumasPassportUsecase) BanMe(ctx context.Context, userID string, req *gk.SelfServiceBanRequest) (*gk.MyBanResponse, error) {
	p := gkBan{Until: int64(req.Until), Permanent: req.Permanent, Reason: strings.TrimSpace(req.BanReason)}
	res, data, err := uc.run(ctx, userID, req.ServerId, gkPlayerBan, p, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Ban(ctx, pid, p.Until, p.Permanent, p.Reason, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyBanResponse{Result: res, Data: dataOf[*gk.BanResult](data)}, nil
}

func (uc *GakumasPassportUsecase) UnbanMe(ctx context.Context, userID string, req *gk.SelfServiceRequest) (*gk.MyBanResponse, error) {
	res, data, err := uc.run(ctx, userID, req.ServerId, gkPlayerUnban, struct{}{}, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.Unban(ctx, pid, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyBanResponse{Result: res, Data: dataOf[*gk.BanResult](data)}, nil
}

func (uc *GakumasPassportUsecase) SetMyPassword(ctx context.Context, userID string, req *gk.SelfServicePasswordRequest) (*gk.MyPasswordResponse, error) {
	p := gkPassword{Password: req.Password}
	res, data, err := uc.run(ctx, userID, req.ServerId, gkPlayerPassword, p, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SetPassword(ctx, pid, p.Password, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyPasswordResponse{Result: res, Data: dataOf[*gk.PasswordResult](data)}, nil
}

func (uc *GakumasPassportUsecase) SetMyPvpRate(ctx context.Context, userID string, req *gk.SelfServiceRateRequest) (*gk.MyChangesResponse, error) {
	p := gkRate{Rate: req.Rate}
	res, data, err := uc.run(ctx, userID, req.ServerId, gkPvpSetRate, p, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.SetPvpRate(ctx, pid, p.Rate, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MyChangesResponse{Result: res, Data: dataOf[*gk.ChangeList](data)}, nil
}

func (uc *GakumasPassportUsecase) ClearMySeminar(ctx context.Context, userID string, req *gk.SelfServiceRequest) (*gk.MySeminarClearResponse, error) {
	res, data, err := uc.run(ctx, userID, req.ServerId, gkSeminarClear, struct{}{}, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.ClearSeminar(ctx, pid, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MySeminarClearResponse{Result: res, Data: dataOf[*gk.SeminarClearResult](data)}, nil
}

func (uc *GakumasPassportUsecase) ResetMySeminar(ctx context.Context, userID string, req *gk.SelfServiceRequest) (*gk.MySeminarResetResponse, error) {
	res, data, err := uc.run(ctx, userID, req.ServerId, gkSeminarReset, struct{}{}, req.Reason, req.TotpCode,
		func(g *gakumas.Client, ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
			return g.ResetSeminar(ctx, pid, opt)
		})
	if err != nil {
		return nil, err
	}
	return &gk.MySeminarResetResponse{Result: res, Data: dataOf[*gk.SeminarResetResult](data)}, nil
}
