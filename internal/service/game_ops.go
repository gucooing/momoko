package service

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
)

// GameOpsService 邮件补偿与兑换码（权限 game:gm）。
type GameOpsService struct {
	v1.UnimplementedGameOpsServiceServer

	uc *biz.GameOpsUsecase
	gm *biz.GmUsecase
}

func NewGameOpsService(uc *biz.GameOpsUsecase, gm *biz.GmUsecase) *GameOpsService {
	return &GameOpsService{uc: uc, gm: gm}
}

// ---------------- 邮件补偿 ----------------

func (s *GameOpsService) ListGameMails(ctx context.Context, req *v1.ListGameMailsRequest) (*v1.ListGameMailsResponse, error) {
	return s.uc.ListMails(ctx, req)
}

func (s *GameOpsService) GetGameMail(ctx context.Context, req *v1.GetGameMailRequest) (*v1.GameMail, error) {
	return s.uc.GetMail(ctx, req.ServerId, req.Id)
}

func (s *GameOpsService) CreateGameMail(ctx context.Context, req *v1.SaveGameMailRequest) (*v1.GameMail, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.uc.CreateMail(ctx, actor, req)
}

func (s *GameOpsService) UpdateGameMail(ctx context.Context, req *v1.SaveGameMailRequest) (*v1.GameMail, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.uc.UpdateMail(ctx, actor, req)
}

func (s *GameOpsService) DeleteGameMail(ctx context.Context, req *v1.DeleteGameMailRequest) (*v1.DeleteGameMailResponse, error) {
	if err := s.uc.DeleteMail(ctx, req.ServerId, req.Id); err != nil {
		return nil, err
	}
	return &v1.DeleteGameMailResponse{}, nil
}

func (s *GameOpsService) DuplicateGameMail(ctx context.Context, req *v1.DuplicateGameMailRequest) (*v1.GameMail, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.uc.DuplicateMail(ctx, actor, req.ServerId, req.Id)
}

func (s *GameOpsService) DryRunGameMail(ctx context.Context, req *v1.DryRunGameMailRequest) (*v1.DryRunGameMailResponse, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.uc.DryRunMail(ctx, actor, req.ServerId, req.Id)
}

func (s *GameOpsService) SendGameMail(ctx context.Context, req *v1.SendGameMailRequest) (*v1.GameMail, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.uc.SendMail(ctx, actor, req)
}

func (s *GameOpsService) CancelGameMail(ctx context.Context, req *v1.CancelGameMailRequest) (*v1.GameMail, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.uc.CancelMail(ctx, actor, req.ServerId, req.Id)
}

func (s *GameOpsService) RevokeGameMail(ctx context.Context, req *v1.RevokeGameMailRequest) (*v1.GameMail, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.uc.RevokeMail(ctx, actor, req)
}

// ---------------- 兑换码 ----------------

func (s *GameOpsService) ListGiftCodeBatches(ctx context.Context, req *v1.ListGiftCodeBatchesRequest) (*v1.ListGiftCodeBatchesResponse, error) {
	return s.uc.ListBatches(ctx, req)
}

func (s *GameOpsService) GetGiftCodeBatch(ctx context.Context, req *v1.GetGiftCodeBatchRequest) (*v1.GetGiftCodeBatchResponse, error) {
	return s.uc.GetBatch(ctx, req.ServerId, req.Id)
}

func (s *GameOpsService) CreateGiftCodeBatch(ctx context.Context, req *v1.CreateGiftCodeBatchRequest) (*v1.GiftCodeBatch, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.uc.CreateBatch(ctx, actor, req)
}

func (s *GameOpsService) UpdateGiftCodeBatch(ctx context.Context, req *v1.UpdateGiftCodeBatchRequest) (*v1.GiftCodeBatch, error) {
	return s.uc.UpdateBatch(ctx, req)
}

func (s *GameOpsService) DisableGiftCodeBatch(ctx context.Context, req *v1.DisableGiftCodeBatchRequest) (*v1.GiftCodeBatch, error) {
	actor, err := gmAdmin(ctx, s.gm)
	if err != nil {
		return nil, err
	}
	return s.uc.DisableBatch(ctx, actor, req.ServerId, req.Id)
}

func (s *GameOpsService) ListGiftCodes(ctx context.Context, req *v1.ListGiftCodesRequest) (*v1.ListGiftCodesResponse, error) {
	return s.uc.ListCodes(ctx, req)
}

func (s *GameOpsService) ListGiftCodeRedeems(ctx context.Context, req *v1.ListGiftCodeRedeemsRequest) (*v1.ListGiftCodeRedeemsResponse, error) {
	return s.uc.ListRedeems(ctx, req)
}
