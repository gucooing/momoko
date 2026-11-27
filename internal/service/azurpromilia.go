package service

import (
	"context"

	az "momoko/api/gen/v1/azurpromilia"
	"momoko/internal/biz"
)

// AzurPromiliaGMService 蓝色星原的 GM（管理端，权限 game:gm）：操作以当前管理员执行并写 GM 记录。
type AzurPromiliaGMService struct {
	az.UnimplementedAzurPromiliaGMServiceServer

	gm   *biz.GmUsecase
	azur *biz.AzurPromiliaUsecase
}

func NewAzurPromiliaGMService(gm *biz.GmUsecase, azur *biz.AzurPromiliaUsecase) *AzurPromiliaGMService {
	return &AzurPromiliaGMService{
		gm:   gm,
		azur: azur,
	}
}

func (s *AzurPromiliaGMService) GetManifest(ctx context.Context, req *az.ServerRequest) (*az.Manifest, error) {
	return s.azur.Manifest(ctx, req.ServerId)
}

func (s *AzurPromiliaGMService) GetStatus(ctx context.Context, req *az.ServerRequest) (*az.Status, error) {
	return s.azur.Status(ctx, req.ServerId)
}

func (s *AzurPromiliaGMService) GetDailyStats(ctx context.Context, req *az.DailyStatsRequest) (*az.DailyStats, error) {
	return s.azur.DailyStats(ctx, req)
}

func (s *AzurPromiliaGMService) ListPlayers(ctx context.Context, req *az.ListPlayersRequest) (*az.ListPlayersResponse, error) {
	return s.azur.ListPlayers(ctx, req)
}

func (s *AzurPromiliaGMService) GetPlayer(ctx context.Context, req *az.PlayerRequest) (*az.GetPlayerResponse, error) {
	return s.azur.GetPlayer(ctx, req)
}

func (s *AzurPromiliaGMService) ListPlayerLogins(ctx context.Context, req *az.PageRequest) (*az.LoginPage, error) {
	return s.azur.ListPlayerLogins(ctx, req)
}

func (s *AzurPromiliaGMService) ListPlayerItems(ctx context.Context, req *az.ListByIDsRequest) (*az.ItemPage, error) {
	return s.azur.ListPlayerItems(ctx, req)
}

func (s *AzurPromiliaGMService) ListPlayerCurrencies(ctx context.Context, req *az.PlayerRequest) (*az.ItemPage, error) {
	return s.azur.ListPlayerCurrencies(ctx, req)
}

func (s *AzurPromiliaGMService) ListPlayerHeroes(ctx context.Context, req *az.ListByIDsRequest) (*az.HeroPage, error) {
	return s.azur.ListPlayerHeroes(ctx, req)
}

func (s *AzurPromiliaGMService) ListPlayerPets(ctx context.Context, req *az.ListByIDsRequest) (*az.PetPage, error) {
	return s.azur.ListPlayerPets(ctx, req)
}

func (s *AzurPromiliaGMService) ListPlayerMails(ctx context.Context, req *az.PageRequest) (*az.MailPage, error) {
	return s.azur.ListPlayerMails(ctx, req)
}

func (s *AzurPromiliaGMService) AdjustItems(ctx context.Context, req *az.AdjustItemsRequest) (*az.OperationResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.AdjustItems(ctx, actor, req)
}

func (s *AzurPromiliaGMService) MailPlayers(ctx context.Context, req *az.MailPlayersRequest) (*az.OperationResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.MailPlayers(ctx, actor, req)
}

func (s *AzurPromiliaGMService) KickPlayers(ctx context.Context, req *az.PlayersOperationRequest) (*az.OperationResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.KickPlayers(ctx, actor, req)
}

func (s *AzurPromiliaGMService) BanPlayers(ctx context.Context, req *az.BanPlayersRequest) (*az.OperationResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.BanPlayers(ctx, actor, req)
}

func (s *AzurPromiliaGMService) UnbanPlayers(ctx context.Context, req *az.PlayersOperationRequest) (*az.OperationResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.UnbanPlayers(ctx, actor, req)
}

func (s *AzurPromiliaGMService) ResetPassword(ctx context.Context, req *az.ResetPasswordRequest) (*az.OperationResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.ResetPassword(ctx, actor, req)
}

func (s *AzurPromiliaGMService) SetPlayerHero(ctx context.Context, req *az.SetHeroRequest) (*az.OperationResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.SetPlayerHero(ctx, actor, req)
}

func (s *AzurPromiliaGMService) SetPlayerPet(ctx context.Context, req *az.SetPetRequest) (*az.OperationResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.SetPlayerPet(ctx, actor, req)
}

func (s *AzurPromiliaGMService) RenamePlayer(ctx context.Context, req *az.RenamePlayerRequest) (*az.OperationResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.RenamePlayer(ctx, actor, req)
}

func (s *AzurPromiliaGMService) ListMails(ctx context.Context, req *az.ListPageRequest) (*az.GmMailPage, error) {
	return s.azur.ListMails(ctx, req)
}

func (s *AzurPromiliaGMService) GetMail(ctx context.Context, req *az.IDRequest) (*az.GmMail, error) {
	return s.azur.GetMail(ctx, req)
}

func (s *AzurPromiliaGMService) SendMail(ctx context.Context, req *az.SendMailRequest) (*az.SendMailResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.SendMail(ctx, actor, req)
}

func (s *AzurPromiliaGMService) RevokeMail(ctx context.Context, req *az.IDOperationRequest) (*az.MailResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.RevokeMail(ctx, actor, req)
}

func (s *AzurPromiliaGMService) ListCodeBatches(ctx context.Context, req *az.ListPageRequest) (*az.CodeBatchPage, error) {
	return s.azur.ListCodeBatches(ctx, req)
}

func (s *AzurPromiliaGMService) GetCodeBatch(ctx context.Context, req *az.IDRequest) (*az.CodeBatch, error) {
	return s.azur.GetCodeBatch(ctx, req)
}

func (s *AzurPromiliaGMService) CreateCodes(ctx context.Context, req *az.CreateCodesRequest) (*az.CreateCodesResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.CreateCodes(ctx, actor, req)
}

func (s *AzurPromiliaGMService) SetCodeBatchDisabled(ctx context.Context, req *az.SetCodeBatchDisabledRequest) (*az.CodeBatchResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.azur.SetCodeBatchDisabled(ctx, actor, req)
}

func (s *AzurPromiliaGMService) ListBatchCodes(ctx context.Context, req *az.IDPageRequest) (*az.CodePage, error) {
	return s.azur.ListBatchCodes(ctx, req)
}

func (s *AzurPromiliaGMService) ListBatchRedemptions(ctx context.Context, req *az.IDPageRequest) (*az.RedemptionPage, error) {
	return s.azur.ListBatchRedemptions(ctx, req)
}
