package service

import (
	"context"

	gk "momoko/api/gen/v1/gakumas"
	"momoko/internal/biz"
)

// GakumasPassportService 学マス 的用户通行证游戏空间（玩家端，只注册在 passport server 上）：
// 取当前通行证用户，调用 GakumasPassportUsecase；与管理端的 GakumasGMService 互不调用。
type GakumasPassportService struct {
	gk.UnimplementedGakumasPassportServiceServer

	gakumas *biz.GakumasPassportUsecase
}

func NewGakumasPassportService(gakumas *biz.GakumasPassportUsecase) *GakumasPassportService {
	return &GakumasPassportService{
		gakumas: gakumas,
	}
}

func (s *GakumasPassportService) GetGame(ctx context.Context, req *gk.MyRequest) (*gk.PlayerGame, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MyGame(ctx, c.UserID, req.ServerId)
}

func (s *GakumasPassportService) GetMyPlayer(ctx context.Context, req *gk.MyRequest) (*gk.PlayerDetail, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MyPlayer(ctx, c.UserID, req.ServerId)
}

func (s *GakumasPassportService) ListMyIdolCards(ctx context.Context, req *gk.MyIdolCardsRequest) (*gk.IdolCardPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MyIdolCards(ctx, c.UserID, req)
}

func (s *GakumasPassportService) ListMySupportCards(ctx context.Context, req *gk.MySupportCardsRequest) (*gk.SupportCardPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MySupportCards(ctx, c.UserID, req)
}

func (s *GakumasPassportService) ListMyGifts(ctx context.Context, req *gk.MyGiftsRequest) (*gk.GiftPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MyGifts(ctx, c.UserID, req)
}

func (s *GakumasPassportService) ListMyGashaDraws(ctx context.Context, req *gk.MyGashaDrawsRequest) (*gk.GashaDrawPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MyGashaDraws(ctx, c.UserID, req)
}

func (s *GakumasPassportService) ListMyMemories(ctx context.Context, req *gk.MyMemoriesRequest) (*gk.MemoryPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MyMemories(ctx, c.UserID, req)
}

func (s *GakumasPassportService) GetMemoryPhoto(ctx context.Context, req *gk.MyMemoryPhotoRequest) (*gk.MemoryPhoto, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MemoryPhoto(ctx, c.UserID, req)
}

func (s *GakumasPassportService) GetMyMemory(ctx context.Context, req *gk.MyMemoryRequest) (*gk.MemoryDetail, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MyMemory(ctx, c.UserID, req)
}

func (s *GakumasPassportService) SetMyRentalMemory(ctx context.Context, req *gk.MyRentalMemoryRequest) (*gk.RentalChange, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SetMyRentalMemory(ctx, c.UserID, req)
}

func (s *GakumasPassportService) SetMyRentalSupportCard(ctx context.Context, req *gk.MyRentalSupportCardRequest) (*gk.RentalChange, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SetMyRentalSupportCard(ctx, c.UserID, req)
}

func (s *GakumasPassportService) GetMyMeishiImage(ctx context.Context, req *gk.MyRequest) (*gk.MeishiImage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MyMeishiImage(ctx, c.UserID, req.ServerId)
}

func (s *GakumasPassportService) GetMyPvp(ctx context.Context, req *gk.MyRequest) (*gk.PlayerPvp, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MyPvp(ctx, c.UserID, req.ServerId)
}

func (s *GakumasPassportService) GetMySeminar(ctx context.Context, req *gk.MyRequest) (*gk.PlayerSeminar, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.MySeminar(ctx, c.UserID, req.ServerId)
}

func (s *GakumasPassportService) ListOngoingGashas(ctx context.Context, req *gk.MyRequest) (*gk.GashaEntryPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.OngoingGashas(ctx, c.UserID, req.ServerId)
}

func (s *GakumasPassportService) RefillMyActionPoint(ctx context.Context, req *gk.SelfServiceRequest) (*gk.MyChangesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.RefillMyActionPoint(ctx, c.UserID, req)
}

func (s *GakumasPassportService) GrantMyIdolCards(ctx context.Context, req *gk.SelfServiceGrantRequest) (*gk.MyIdolCardGrantResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.GrantMyIdolCards(ctx, c.UserID, req)
}

func (s *GakumasPassportService) GrantMySupportCards(ctx context.Context, req *gk.SelfServiceGrantRequest) (*gk.MySupportCardGrantResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.GrantMySupportCards(ctx, c.UserID, req)
}

func (s *GakumasPassportService) KickMe(ctx context.Context, req *gk.SelfServiceRequest) (*gk.MyKickResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.KickMe(ctx, c.UserID, req)
}

func (s *GakumasPassportService) RenameMe(ctx context.Context, req *gk.SelfServiceRenameRequest) (*gk.MyRenameResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.RenameMe(ctx, c.UserID, req)
}

func (s *GakumasPassportService) ResetMyPvpDaily(ctx context.Context, req *gk.SelfServiceRequest) (*gk.MyChangesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.ResetMyPvpDaily(ctx, c.UserID, req)
}

func (s *GakumasPassportService) ExportMyArchive(ctx context.Context, req *gk.SelfServiceRequest) (*gk.MyArchiveExportResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.ExportMyArchive(ctx, c.UserID, req)
}

func (s *GakumasPassportService) SendMyGift(ctx context.Context, req *gk.SelfServiceGiftRequest) (*gk.MyGiftSendResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SendMyGift(ctx, c.UserID, req)
}

func (s *GakumasPassportService) RevokeMyGifts(ctx context.Context, req *gk.SelfServiceGiftRevokeRequest) (*gk.MyGiftRevokeResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.RevokeMyGifts(ctx, c.UserID, req)
}

func (s *GakumasPassportService) AdjustMyJewel(ctx context.Context, req *gk.SelfServiceJewelRequest) (*gk.MyChangesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.AdjustMyJewel(ctx, c.UserID, req)
}

func (s *GakumasPassportService) AdjustMyItem(ctx context.Context, req *gk.SelfServiceItemRequest) (*gk.MyChangesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.AdjustMyItem(ctx, c.UserID, req)
}

func (s *GakumasPassportService) AddMyExp(ctx context.Context, req *gk.SelfServiceExpRequest) (*gk.MyChangesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.AddMyExp(ctx, c.UserID, req)
}

func (s *GakumasPassportService) BanMe(ctx context.Context, req *gk.SelfServiceBanRequest) (*gk.MyBanResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.BanMe(ctx, c.UserID, req)
}

func (s *GakumasPassportService) UnbanMe(ctx context.Context, req *gk.SelfServiceRequest) (*gk.MyBanResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.UnbanMe(ctx, c.UserID, req)
}

func (s *GakumasPassportService) SetMyPassword(ctx context.Context, req *gk.SelfServicePasswordRequest) (*gk.MyPasswordResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SetMyPassword(ctx, c.UserID, req)
}

func (s *GakumasPassportService) SetMyPvpRate(ctx context.Context, req *gk.SelfServiceRateRequest) (*gk.MyChangesResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.SetMyPvpRate(ctx, c.UserID, req)
}

func (s *GakumasPassportService) ClearMySeminar(ctx context.Context, req *gk.SelfServiceRequest) (*gk.MySeminarClearResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.ClearMySeminar(ctx, c.UserID, req)
}

func (s *GakumasPassportService) ResetMySeminar(ctx context.Context, req *gk.SelfServiceRequest) (*gk.MySeminarResetResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.gakumas.ResetMySeminar(ctx, c.UserID, req)
}
