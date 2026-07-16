package service

import (
	"context"
	"strings"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
)

// authTokenFromContext 读取用户端 X-Sub2API-Token（内嵌页鉴权）。
func authTokenFromContext(ctx context.Context) (string, error) {
	r, ok := khttp.RequestFromServerContext(ctx)
	if !ok {
		return "", biz.ErrLotteryTokenInvalid
	}
	token := strings.TrimSpace(r.Header.Get("X-Sub2API-Token"))
	if token == "" {
		// 无 token 时允许读公开状态（未鉴权），返回空串由 usecase 处理
		return "", nil
	}
	return token, nil
}

// ---------- 管理端 ----------

func (s *Sub2APIService) GetLotteryOverview(ctx context.Context, _ *v1.GetLotteryOverviewRequest) (*v1.GetLotteryOverviewResponse, error) {
	return s.uc.LotteryOverview(ctx)
}

func (s *Sub2APIService) UpdateLotterySettings(ctx context.Context, req *v1.UpdateLotterySettingsRequest) (*v1.UpdateLotterySettingsResponse, error) {
	settings, err := s.uc.UpdateLotterySettings(ctx, req)
	if err != nil {
		return nil, err
	}
	return &v1.UpdateLotterySettingsResponse{Settings: settings}, nil
}

func (s *Sub2APIService) ListLotteryRounds(ctx context.Context, req *v1.ListLotteryRoundsRequest) (*v1.ListLotteryRoundsResponse, error) {
	list, total, err := s.uc.ListLotteryRounds(ctx, int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &v1.ListLotteryRoundsResponse{
		Rounds:   list,
		Total:    int64(total),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	}, nil
}

func (s *Sub2APIService) GetLotteryRoundDetail(ctx context.Context, req *v1.GetLotteryRoundDetailRequest) (*v1.GetLotteryRoundDetailResponse, error) {
	round, winners, err := s.uc.GetLotteryRoundDetail(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.GetLotteryRoundDetailResponse{Round: round, Winners: winners}, nil
}

func (s *Sub2APIService) ListLotteryRegistrants(ctx context.Context, req *v1.ListLotteryRegistrantsRequest) (*v1.ListLotteryRegistrantsResponse, error) {
	regs, err := s.uc.LotteryRoundRegistrants(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.ListLotteryRegistrantsResponse{
		Registrants: regs,
		Total:       int32(len(regs)),
	}, nil
}

func (s *Sub2APIService) GetSub2APIUser(ctx context.Context, req *v1.GetSub2APIUserRequest) (*v1.GetSub2APIUserResponse, error) {
	user, err := s.uc.GetSub2APIUserInfo(ctx, req.GetUserId())
	if err != nil {
		return nil, err
	}
	return &v1.GetSub2APIUserResponse{User: user}, nil
}

func (s *Sub2APIService) DistributeLotteryRound(ctx context.Context, req *v1.DistributeLotteryRoundRequest) (*v1.DistributeLotteryRoundResponse, error) {
	round, err := s.uc.DistributeLotteryRound(ctx, req.GetId())
	if err != nil {
		return nil, err
	}
	return &v1.DistributeLotteryRoundResponse{Round: round}, nil
}

func (s *Sub2APIService) TriggerLotterySettle(ctx context.Context, req *v1.TriggerLotterySettleRequest) (*v1.TriggerLotterySettleResponse, error) {
	round, err := s.uc.TriggerLotterySettle(ctx, req.GetDate())
	if err != nil {
		return nil, err
	}
	return &v1.TriggerLotterySettleResponse{Round: round}, nil
}

func (s *Sub2APIService) TriggerLotteryDraw(ctx context.Context, req *v1.TriggerLotteryDrawRequest) (*v1.TriggerLotteryDrawResponse, error) {
	round, err := s.uc.TriggerLotteryDraw(ctx, req.GetDate())
	if err != nil {
		return nil, err
	}
	return &v1.TriggerLotteryDrawResponse{Round: round}, nil
}

// ---------- 用户端（公开） ----------

func (s *Sub2APIService) GetLotteryStatus(ctx context.Context, _ *v1.GetLotteryStatusRequest) (*v1.GetLotteryStatusResponse, error) {
	token, err := authTokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	return s.uc.LotteryStatus(ctx, token)
}

func (s *Sub2APIService) RegisterLottery(ctx context.Context, _ *v1.RegisterLotteryRequest) (*v1.RegisterLotteryResponse, error) {
	token, err := authTokenFromContext(ctx)
	if err != nil {
		return nil, err
	}
	if token == "" {
		return nil, biz.ErrLotteryTokenInvalid
	}
	return s.uc.RegisterLottery(ctx, token)
}

func (s *Sub2APIService) ListLotteryHistoryPublic(ctx context.Context, req *v1.ListLotteryHistoryPublicRequest) (*v1.ListLotteryHistoryPublicResponse, error) {
	token, _ := authTokenFromContext(ctx)
	items, total, err := s.uc.LotteryHistoryPublic(ctx, token, int(req.GetPage()), int(req.GetPageSize()))
	if err != nil {
		return nil, err
	}
	return &v1.ListLotteryHistoryPublicResponse{
		Items:    items,
		Total:    int64(total),
		Page:     req.GetPage(),
		PageSize: req.GetPageSize(),
	}, nil
}
