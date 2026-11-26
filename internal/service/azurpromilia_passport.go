package service

import (
	"context"

	az "momoko/api/gen/v1/azurpromilia"
	"momoko/internal/biz"
)

// AzurPromiliaPassportService 蓝色星原的用户通行证游戏空间（玩家端，只注册在 passport server 上）：
// 取当前通行证用户，调用 AzurPromiliaPassportUsecase。
type AzurPromiliaPassportService struct {
	az.UnimplementedAzurPromiliaPassportServiceServer

	azur *biz.AzurPromiliaPassportUsecase
}

func NewAzurPromiliaPassportService(azur *biz.AzurPromiliaPassportUsecase) *AzurPromiliaPassportService {
	return &AzurPromiliaPassportService{
		azur: azur,
	}
}

func (s *AzurPromiliaPassportService) GetMyPlayer(ctx context.Context, req *az.MyRequest) (*az.PlayerDetail, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.MyPlayer(ctx, c.UserID, req.ServerId)
}

func (s *AzurPromiliaPassportService) ListMyLogins(ctx context.Context, req *az.MyLoginsRequest) (*az.LoginPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.MyLogins(ctx, c.UserID, req)
}

func (s *AzurPromiliaPassportService) ListMyItems(ctx context.Context, req *az.MyLoginsRequest) (*az.ItemPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.MyItems(ctx, c.UserID, req)
}

func (s *AzurPromiliaPassportService) ListMyCurrencies(ctx context.Context, req *az.MyRequest) (*az.ItemPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.MyCurrencies(ctx, c.UserID, req.ServerId)
}

func (s *AzurPromiliaPassportService) ListMyHeroes(ctx context.Context, req *az.MyLoginsRequest) (*az.HeroPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.MyHeroes(ctx, c.UserID, req)
}

func (s *AzurPromiliaPassportService) ListMyPets(ctx context.Context, req *az.MyLoginsRequest) (*az.PetPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.MyPets(ctx, c.UserID, req)
}

func (s *AzurPromiliaPassportService) ListMyMails(ctx context.Context, req *az.MyLoginsRequest) (*az.MailPage, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.MyMails(ctx, c.UserID, req)
}

func (s *AzurPromiliaPassportService) MailMe(ctx context.Context, req *az.SelfServiceMailRequest) (*az.MySelfServiceResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.MailMe(ctx, c.UserID, req)
}

func (s *AzurPromiliaPassportService) AdjustMyItems(ctx context.Context, req *az.SelfServiceItemsRequest) (*az.MySelfServiceResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.AdjustMyItems(ctx, c.UserID, req)
}

func (s *AzurPromiliaPassportService) KickMe(ctx context.Context, req *az.SelfServiceRequest) (*az.MySelfServiceResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.KickMe(ctx, c.UserID, req)
}

func (s *AzurPromiliaPassportService) BanMe(ctx context.Context, req *az.SelfServiceBanRequest) (*az.MySelfServiceResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.BanMe(ctx, c.UserID, req)
}

func (s *AzurPromiliaPassportService) UnbanMe(ctx context.Context, req *az.SelfServiceRequest) (*az.MySelfServiceResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.UnbanMe(ctx, c.UserID, req)
}

func (s *AzurPromiliaPassportService) SetMyPassword(ctx context.Context, req *az.SelfServicePasswordRequest) (*az.MySelfServiceResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.SetMyPassword(ctx, c.UserID, req)
}

func (s *AzurPromiliaPassportService) RenameMe(ctx context.Context, req *az.SelfServiceRenameRequest) (*az.MySelfServiceResponse, error) {
	c, err := me(ctx)
	if err != nil {
		return nil, err
	}
	return s.azur.RenameMe(ctx, c.UserID, req)
}
