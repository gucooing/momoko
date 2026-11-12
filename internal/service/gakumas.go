package service

import (
	"context"

	v1 "momoko/api/gen/v1"
	gk "momoko/api/gen/v1/gakumas"
	"momoko/internal/biz"
)

// GakumasGMService 学マス GM 运营（管理端，主 server，权限 game:gm）：解析请求，调用 GakumasUsecase。
type GakumasGMService struct {
	gk.UnimplementedGakumasGMServiceServer

	gm      *biz.GmUsecase
	gakumas *biz.GakumasUsecase
}

func NewGakumasGMService(gm *biz.GmUsecase, gakumas *biz.GakumasUsecase) *GakumasGMService {
	return &GakumasGMService{
		gm:      gm,
		gakumas: gakumas,
	}
}

func (s *GakumasGMService) GetManifest(ctx context.Context, req *gk.ServerRequest) (*gk.Manifest, error) {
	return s.gakumas.Manifest(ctx, req.ServerId)
}

func (s *GakumasGMService) GetStatus(ctx context.Context, req *gk.ServerRequest) (*gk.Status, error) {
	return s.gakumas.Status(ctx, req.ServerId)
}

func (s *GakumasGMService) GetDashboard(ctx context.Context, req *gk.GetDashboardRequest) (*gk.Dashboard, error) {
	return s.gakumas.Dashboard(ctx, req)
}

func (s *GakumasGMService) ListPlayers(ctx context.Context, req *gk.ListPlayersRequest) (*gk.ListPlayersResponse, error) {
	return s.gakumas.ListPlayers(ctx, req)
}

func (s *GakumasGMService) GetPlayer(ctx context.Context, req *gk.PlayerRequest) (*gk.GetPlayerResponse, error) {
	return s.gakumas.GetPlayer(ctx, req)
}

func (s *GakumasGMService) ListPlayerSessions(ctx context.Context, req *gk.PageRequest) (*gk.SessionPage, error) {
	return s.gakumas.ListPlayerSessions(ctx, req)
}

func (s *GakumasGMService) ListPlayerIdolCards(ctx context.Context, req *gk.ListIdolCardsRequest) (*gk.IdolCardPage, error) {
	return s.gakumas.ListPlayerIdolCards(ctx, req)
}

func (s *GakumasGMService) ListPlayerSupportCards(ctx context.Context, req *gk.ListSupportCardsRequest) (*gk.SupportCardPage, error) {
	return s.gakumas.ListPlayerSupportCards(ctx, req)
}

func (s *GakumasGMService) ListPlayerItems(ctx context.Context, req *gk.ListItemsRequest) (*gk.ItemPage, error) {
	return s.gakumas.ListPlayerItems(ctx, req)
}

func (s *GakumasGMService) ListPlayerGifts(ctx context.Context, req *gk.ListGiftsRequest) (*gk.GiftPage, error) {
	return s.gakumas.ListPlayerGifts(ctx, req)
}

func (s *GakumasGMService) ListPlayerGashaDraws(ctx context.Context, req *gk.ListGashaDrawsRequest) (*gk.GashaDrawPage, error) {
	return s.gakumas.ListPlayerGashaDraws(ctx, req)
}

func (s *GakumasGMService) ListPlayerMemories(ctx context.Context, req *gk.ListMemoriesRequest) (*gk.MemoryPage, error) {
	return s.gakumas.ListPlayerMemories(ctx, req)
}

func (s *GakumasGMService) GetMemoryPhoto(ctx context.Context, req *gk.MemoryPhotoRequest) (*gk.MemoryPhoto, error) {
	return s.gakumas.MemoryPhoto(ctx, req)
}

func (s *GakumasGMService) GetPlayerMemory(ctx context.Context, req *gk.PlayerMemoryRequest) (*gk.MemoryDetail, error) {
	return s.gakumas.GetPlayerMemory(ctx, req)
}

func (s *GakumasGMService) GetPlayerMeishiImage(ctx context.Context, req *gk.PlayerRequest) (*gk.MeishiImage, error) {
	return s.gakumas.GetPlayerMeishiImage(ctx, req)
}

func (s *GakumasGMService) GetPlayerPvp(ctx context.Context, req *gk.PlayerRequest) (*gk.PlayerPvp, error) {
	return s.gakumas.GetPlayerPvp(ctx, req)
}

func (s *GakumasGMService) GetPlayerSeminar(ctx context.Context, req *gk.PlayerRequest) (*gk.PlayerSeminar, error) {
	return s.gakumas.GetPlayerSeminar(ctx, req)
}

func (s *GakumasGMService) ListPlayerBackups(ctx context.Context, req *gk.PlayerRequest) (*gk.BackupList, error) {
	return s.gakumas.ListPlayerBackups(ctx, req)
}

func (s *GakumasGMService) GetGashaStats(ctx context.Context, req *gk.ServerRequest) (*gk.GashaStatList, error) {
	return s.gakumas.GashaStats(ctx, req.ServerId)
}

func (s *GakumasGMService) ListProduces(ctx context.Context, req *gk.ListProducesRequest) (*gk.ProducePage, error) {
	return s.gakumas.ListProduces(ctx, req)
}

func (s *GakumasGMService) GetPvpRanking(ctx context.Context, req *gk.RankingRequest) (*gk.PvpRankingPage, error) {
	return s.gakumas.PvpRanking(ctx, req)
}

func (s *GakumasGMService) GetSeminarStats(ctx context.Context, req *gk.ServerRequest) (*gk.SeminarStatList, error) {
	return s.gakumas.SeminarStats(ctx, req.ServerId)
}

func (s *GakumasGMService) GetTowerRanking(ctx context.Context, req *gk.TowerRankingRequest) (*gk.TowerRankingPage, error) {
	return s.gakumas.TowerRanking(ctx, req)
}

func (s *GakumasGMService) ListCurrencies(ctx context.Context, req *gk.CatalogRequest) (*gk.CurrencyPage, error) {
	return s.gakumas.Currencies(ctx, req)
}

func (s *GakumasGMService) ListItemCatalog(ctx context.Context, req *gk.ItemCatalogRequest) (*gk.ItemEntryPage, error) {
	return s.gakumas.ItemCatalog(ctx, req)
}

func (s *GakumasGMService) ListIdolCardSkins(ctx context.Context, req *gk.IdolCardSkinCatalogRequest) (*gk.IdolCardSkinPage, error) {
	return s.gakumas.IdolCardSkins(ctx, req)
}

func (s *GakumasGMService) ListSupportCardCatalog(ctx context.Context, req *gk.SupportCardCatalogRequest) (*gk.SupportCardEntryPage, error) {
	return s.gakumas.SupportCardCatalog(ctx, req)
}

func (s *GakumasGMService) ListCharacters(ctx context.Context, req *gk.CatalogRequest) (*gk.CharacterPage, error) {
	return s.gakumas.Characters(ctx, req)
}

func (s *GakumasGMService) ListCostumes(ctx context.Context, req *gk.CostumeCatalogRequest) (*gk.CostumePage, error) {
	return s.gakumas.Costumes(ctx, req)
}

func (s *GakumasGMService) ListCostumeHeads(ctx context.Context, req *gk.CostumeCatalogRequest) (*gk.CostumePage, error) {
	return s.gakumas.CostumeHeads(ctx, req)
}

func (s *GakumasGMService) ListMusic(ctx context.Context, req *gk.CatalogRequest) (*gk.MusicPage, error) {
	return s.gakumas.Music(ctx, req)
}

func (s *GakumasGMService) ListGashas(ctx context.Context, req *gk.GashaCatalogRequest) (*gk.GashaEntryPage, error) {
	return s.gakumas.Gashas(ctx, req)
}

func (s *GakumasGMService) ListProduceGroups(ctx context.Context, req *gk.CatalogRequest) (*gk.ProduceGroupPage, error) {
	return s.gakumas.ProduceGroups(ctx, req)
}

func (s *GakumasGMService) ListProduceCatalog(ctx context.Context, req *gk.ProduceCatalogRequest) (*gk.ProduceEntryPage, error) {
	return s.gakumas.ProduceCatalog(ctx, req)
}

func (s *GakumasGMService) ListSeminars(ctx context.Context, req *gk.CatalogRequest) (*gk.SeminarEntryPage, error) {
	return s.gakumas.Seminars(ctx, req)
}

func (s *GakumasGMService) ListTowers(ctx context.Context, req *gk.CatalogRequest) (*gk.TowerEntryPage, error) {
	return s.gakumas.Towers(ctx, req)
}

func (s *GakumasGMService) ListProduceCards(ctx context.Context, req *gk.CatalogRequest) (*gk.ProduceCardEntryPage, error) {
	return s.gakumas.ProduceCards(ctx, req)
}

func (s *GakumasGMService) ListProduceItems(ctx context.Context, req *gk.CatalogRequest) (*gk.ProduceItemEntryPage, error) {
	return s.gakumas.ProduceItems(ctx, req)
}

func (s *GakumasGMService) ListMemoryAbilities(ctx context.Context, req *gk.CatalogRequest) (*gk.MemoryAbilityPage, error) {
	return s.gakumas.MemoryAbilities(ctx, req)
}

func (s *GakumasGMService) SendGift(ctx context.Context, req *gk.SendGiftRequest) (*gk.GiftSendResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SendGift(ctx, actor, req)
}

func (s *GakumasGMService) RevokeGifts(ctx context.Context, req *gk.RevokeGiftsRequest) (*gk.GiftRevokeResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.RevokeGifts(ctx, actor, req)
}

func (s *GakumasGMService) GrantAllIdolCards(ctx context.Context, req *gk.GrantAllRequest) (*gk.IdolCardGrantResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.GrantAllIdolCards(ctx, actor, req)
}

func (s *GakumasGMService) GrantAllSupportCards(ctx context.Context, req *gk.GrantAllRequest) (*gk.SupportCardGrantResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.GrantAllSupportCards(ctx, actor, req)
}

func (s *GakumasGMService) AdjustItem(ctx context.Context, req *gk.AdjustItemRequest) (*gk.ChangesResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.AdjustItem(ctx, actor, req)
}

func (s *GakumasGMService) AdjustJewel(ctx context.Context, req *gk.AdjustJewelRequest) (*gk.ChangesResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.AdjustJewel(ctx, actor, req)
}

func (s *GakumasGMService) RefillActionPoint(ctx context.Context, req *gk.PlayersOperationRequest) (*gk.ChangesResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.RefillActionPoint(ctx, actor, req)
}

func (s *GakumasGMService) AddExp(ctx context.Context, req *gk.AddExpRequest) (*gk.ChangesResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.AddExp(ctx, actor, req)
}

func (s *GakumasGMService) KickPlayers(ctx context.Context, req *gk.PlayersOperationRequest) (*gk.KickResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.KickPlayers(ctx, actor, req)
}

func (s *GakumasGMService) BanPlayers(ctx context.Context, req *gk.BanPlayersRequest) (*gk.BanResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.BanPlayers(ctx, actor, req)
}

func (s *GakumasGMService) UnbanPlayers(ctx context.Context, req *gk.PlayersOperationRequest) (*gk.BanResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.UnbanPlayers(ctx, actor, req)
}

func (s *GakumasGMService) ResetPassword(ctx context.Context, req *gk.ResetPasswordRequest) (*gk.PasswordResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.ResetPassword(ctx, actor, req)
}

func (s *GakumasGMService) RenamePlayer(ctx context.Context, req *gk.RenamePlayerRequest) (*gk.RenameResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.RenamePlayer(ctx, actor, req)
}

func (s *GakumasGMService) SetRentalSupportCard(ctx context.Context, req *gk.SetRentalSupportCardRequest) (*gk.RentalResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SetRentalSupportCard(ctx, actor, req)
}

func (s *GakumasGMService) SetRentalMemory(ctx context.Context, req *gk.SetRentalMemoryRequest) (*gk.RentalResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SetRentalMemory(ctx, actor, req)
}

func (s *GakumasGMService) SetPvpRate(ctx context.Context, req *gk.SetPvpRateRequest) (*gk.ChangesResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SetPvpRate(ctx, actor, req)
}

func (s *GakumasGMService) ResetPvpDaily(ctx context.Context, req *gk.PlayersOperationRequest) (*gk.ChangesResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.ResetPvpDaily(ctx, actor, req)
}

func (s *GakumasGMService) ClearSeminar(ctx context.Context, req *gk.PlayersOperationRequest) (*gk.SeminarClearResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.ClearSeminar(ctx, actor, req)
}

func (s *GakumasGMService) ResetSeminar(ctx context.Context, req *gk.PlayersOperationRequest) (*gk.SeminarResetResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.ResetSeminar(ctx, actor, req)
}

func (s *GakumasGMService) ExportArchive(ctx context.Context, req *gk.PlayersOperationRequest) (*gk.ArchiveExportResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.ExportArchive(ctx, actor, req)
}

func (s *GakumasGMService) BackupArchive(ctx context.Context, req *gk.BackupArchiveRequest) (*gk.BackupResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.BackupArchive(ctx, actor, req)
}

func (s *GakumasGMService) RestoreArchive(ctx context.Context, req *gk.RestoreArchiveRequest) (*gk.RestoreResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.RestoreArchive(ctx, actor, req)
}

func (s *GakumasGMService) SetMaintenance(ctx context.Context, req *gk.SetMaintenanceRequest) (*gk.MaintenanceResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SetMaintenance(ctx, actor, req)
}

// 配置文档。

func (s *GakumasGMService) config(ctx context.Context, serverID, id string) (*v1.GetGmConfigResponse, error) {
	return s.gakumas.Config(ctx, serverID, id)
}

func (s *GakumasGMService) saveDraft(ctx context.Context, id string, req *gk.SaveConfigDraftRequest) (*v1.GmConfigDraft, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SaveConfigDraft(ctx, actor, req.ServerId, id, req.Content, req.BaseRevision, req.Comment)
}

func (s *GakumasGMService) discardDraft(ctx context.Context, serverID, id string) (*v1.DiscardGmConfigDraftResponse, error) {
	if err := s.gakumas.DiscardConfigDraft(ctx, serverID, id); err != nil {
		return nil, err
	}
	return &v1.DiscardGmConfigDraftResponse{}, nil
}

func (s *GakumasGMService) publish(ctx context.Context, id string, req *gk.PublishConfigRequest) (*v1.PublishGmConfigResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.gakumas.PublishConfig(ctx, actor, req.ServerId, id, req.Content, req.BaseRevision, req.Comment)
}

func (s *GakumasGMService) GetGashaConfig(ctx context.Context, req *gk.ServerRequest) (*v1.GetGmConfigResponse, error) {
	return s.config(ctx, req.ServerId, biz.GakumasConfigGasha)
}

func (s *GakumasGMService) SaveGashaConfigDraft(ctx context.Context, req *gk.SaveConfigDraftRequest) (*v1.GmConfigDraft, error) {
	return s.saveDraft(ctx, biz.GakumasConfigGasha, req)
}

func (s *GakumasGMService) DiscardGashaConfigDraft(ctx context.Context, req *gk.ServerRequest) (*v1.DiscardGmConfigDraftResponse, error) {
	return s.discardDraft(ctx, req.ServerId, biz.GakumasConfigGasha)
}

func (s *GakumasGMService) ValidateGashaConfig(ctx context.Context, req *gk.ValidateConfigRequest) (*v1.ValidateGmConfigResponse, error) {
	return s.gakumas.ValidateConfig(ctx, req.ServerId, biz.GakumasConfigGasha, req.Content)
}

func (s *GakumasGMService) PublishGashaConfig(ctx context.Context, req *gk.PublishConfigRequest) (*v1.PublishGmConfigResponse, error) {
	return s.publish(ctx, biz.GakumasConfigGasha, req)
}

func (s *GakumasGMService) ListGashaConfigRevisions(ctx context.Context, req *gk.ConfigRevisionsRequest) (*v1.ListGmConfigRevisionsResponse, error) {
	return s.gakumas.ConfigRevisions(ctx, req.ServerId, biz.GakumasConfigGasha, req.Page, req.PageSize)
}

func (s *GakumasGMService) GetGashaConfigRevision(ctx context.Context, req *gk.ConfigRevisionRequest) (*v1.GmConfigRevision, error) {
	return s.gakumas.ConfigRevision(ctx, req.ServerId, biz.GakumasConfigGasha, req.Id)
}

func (s *GakumasGMService) GetNoticeConfig(ctx context.Context, req *gk.ServerRequest) (*v1.GetGmConfigResponse, error) {
	return s.config(ctx, req.ServerId, biz.GakumasConfigNotice)
}

func (s *GakumasGMService) SaveNoticeConfigDraft(ctx context.Context, req *gk.SaveConfigDraftRequest) (*v1.GmConfigDraft, error) {
	return s.saveDraft(ctx, biz.GakumasConfigNotice, req)
}

func (s *GakumasGMService) DiscardNoticeConfigDraft(ctx context.Context, req *gk.ServerRequest) (*v1.DiscardGmConfigDraftResponse, error) {
	return s.discardDraft(ctx, req.ServerId, biz.GakumasConfigNotice)
}

func (s *GakumasGMService) ValidateNoticeConfig(ctx context.Context, req *gk.ValidateConfigRequest) (*v1.ValidateGmConfigResponse, error) {
	return s.gakumas.ValidateConfig(ctx, req.ServerId, biz.GakumasConfigNotice, req.Content)
}

func (s *GakumasGMService) PublishNoticeConfig(ctx context.Context, req *gk.PublishConfigRequest) (*v1.PublishGmConfigResponse, error) {
	return s.publish(ctx, biz.GakumasConfigNotice, req)
}

func (s *GakumasGMService) ListNoticeConfigRevisions(ctx context.Context, req *gk.ConfigRevisionsRequest) (*v1.ListGmConfigRevisionsResponse, error) {
	return s.gakumas.ConfigRevisions(ctx, req.ServerId, biz.GakumasConfigNotice, req.Page, req.PageSize)
}

func (s *GakumasGMService) GetNoticeConfigRevision(ctx context.Context, req *gk.ConfigRevisionRequest) (*v1.GmConfigRevision, error) {
	return s.gakumas.ConfigRevision(ctx, req.ServerId, biz.GakumasConfigNotice, req.Id)
}

func (s *GakumasGMService) GetPvpRateConfig(ctx context.Context, req *gk.ServerRequest) (*v1.GetGmConfigResponse, error) {
	return s.config(ctx, req.ServerId, biz.GakumasConfigPvpRate)
}

func (s *GakumasGMService) GetHomeConfig(ctx context.Context, req *gk.ServerRequest) (*v1.GetGmConfigResponse, error) {
	return s.config(ctx, req.ServerId, biz.GakumasConfigHome)
}

func (s *GakumasGMService) GetResearchConfig(ctx context.Context, req *gk.ServerRequest) (*v1.GetGmConfigResponse, error) {
	return s.config(ctx, req.ServerId, biz.GakumasConfigResearch)
}

func (s *GakumasGMService) GetStoryEventConfig(ctx context.Context, req *gk.ServerRequest) (*v1.GetGmConfigResponse, error) {
	return s.config(ctx, req.ServerId, biz.GakumasConfigStoryEvent)
}

func (s *GakumasGMService) GetLoginBonusConfig(ctx context.Context, req *gk.ServerRequest) (*v1.GetGmConfigResponse, error) {
	return s.config(ctx, req.ServerId, biz.GakumasConfigLoginBonus)
}
