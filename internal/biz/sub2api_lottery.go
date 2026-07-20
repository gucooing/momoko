package biz

import (
	"context"
	"errors"

	v1 "momoko/api/gen/v1"
	sub2apipkg "momoko/pkg/sub2api"

	"google.golang.org/protobuf/types/known/timestamppb"
)

// ---------- 管理端 ----------

func (s *Sub2APIUsecase) LotteryOverview(ctx context.Context) (*v1.GetLotteryOverviewResponse, error) {
	ov, err := s.lottery.Overview(ctx)
	if err != nil {
		return nil, mapLotteryError(err)
	}
	out := &v1.GetLotteryOverviewResponse{
		Settings:       toV1LotterySettings(ov.Settings),
		AccumDate:      ov.AccumDate,
		AccumSpend:     ov.AccumSpend,
		AccumPool:      ov.AccumPool,
		AccumEligible:  int32(ov.AccumEligible),
		NextSettleTime: timestamppb.New(ov.NextSettleTime),
		NextDrawTime:   timestamppb.New(ov.NextDrawTime),
		Current:        toV1LotteryRound(ov.Current),
	}
	return out, nil
}

func (s *Sub2APIUsecase) UpdateLotterySettings(ctx context.Context, req *v1.UpdateLotterySettingsRequest) (*v1.LotterySettings, error) {
	in := fromV1LotterySettings(req.GetSettings())
	next, err := s.lottery.UpdateSettings(ctx, in)
	if err != nil {
		return nil, mapLotteryError(err)
	}
	return toV1LotterySettings(next), nil
}

func (s *Sub2APIUsecase) ListLotteryRounds(ctx context.Context, page, pageSize int) ([]*v1.LotteryRound, int, error) {
	rounds, total, err := s.lottery.History(ctx, page, pageSize)
	if err != nil {
		return nil, 0, mapLotteryError(err)
	}
	out := make([]*v1.LotteryRound, 0, len(rounds))
	for _, r := range rounds {
		out = append(out, toV1LotteryRound(r))
	}
	return out, total, nil
}

func (s *Sub2APIUsecase) GetLotteryRoundDetail(ctx context.Context, id string) (*v1.LotteryRound, []*v1.LotteryParticipant, error) {
	round, winners, err := s.lottery.RoundWinners(ctx, id)
	if err != nil {
		return nil, nil, mapLotteryError(err)
	}
	return toV1LotteryRound(round), toV1LotteryParticipants(winners), nil
}

func (s *Sub2APIUsecase) LotteryRoundRegistrants(ctx context.Context, roundID string) ([]*v1.LotteryParticipant, error) {
	regs, err := s.lottery.RoundRegistrants(ctx, roundID)
	if err != nil {
		return nil, mapLotteryError(err)
	}
	return toV1LotteryParticipants(regs), nil
}

func (s *Sub2APIUsecase) GetSub2APIUserInfo(ctx context.Context, userID int64) (*v1.Sub2APIUserInfo, error) {
	u, err := s.lottery.UserInfo(ctx, userID)
	if err != nil {
		return nil, mapLotteryError(err)
	}
	return toV1Sub2APIUserInfo(u), nil
}

func (s *Sub2APIUsecase) DistributeLotteryRound(ctx context.Context, id string) (*v1.LotteryRound, error) {
	round, err := s.lottery.DistributeRound(id)
	if err != nil {
		return nil, mapLotteryError(err)
	}
	return toV1LotteryRound(round), nil
}

func (s *Sub2APIUsecase) TriggerLotterySettle(ctx context.Context, date string) (*v1.LotteryRound, error) {
	if date == "" {
		date = sub2apipkg.LotteryDateStr(sub2apipkg.LotteryNow())
	}
	round, err := s.lottery.Settle(date)
	if err != nil {
		return nil, mapLotteryError(err)
	}
	return toV1LotteryRound(round), nil
}

func (s *Sub2APIUsecase) TriggerLotteryDraw(ctx context.Context, date string) (*v1.LotteryRound, error) {
	if date == "" {
		date = sub2apipkg.LotteryDateStr(sub2apipkg.LotteryNow())
	}
	round, err := s.lottery.Draw(date)
	if err != nil {
		return nil, mapLotteryError(err)
	}
	return toV1LotteryRound(round), nil
}

// ---------- 用户端 ----------

func (s *Sub2APIUsecase) LotteryStatus(ctx context.Context, jwt string) (*v1.GetLotteryStatusResponse, error) {
	st, err := s.lottery.UserStatus(ctx, jwt)
	if err != nil {
		return nil, mapLotteryError(err)
	}
	return toV1LotteryUserStatus(st), nil
}

func (s *Sub2APIUsecase) RegisterLottery(ctx context.Context, jwt string) (*v1.RegisterLotteryResponse, error) {
	st, err := s.lottery.Register(ctx, jwt)
	if err != nil {
		return nil, mapLotteryError(err)
	}
	// Register 与 Status 字段一致，直接复用映射结果再拷到 Register 消息。
	base := toV1LotteryUserStatus(st)
	return &v1.RegisterLotteryResponse{
		Enabled:        base.GetEnabled(),
		Authenticated:  base.GetAuthenticated(),
		UserId:         base.GetUserId(),
		UserName:       base.GetUserName(),
		AccumPool:      base.GetAccumPool(),
		NextSettleTime: base.GetNextSettleTime(),
		NextDrawTime:   base.GetNextDrawTime(),
		Current:        base.GetCurrent(),
		Eligible:       base.GetEligible(),
		UserSpend:      base.GetUserSpend(),
		Registered:     base.GetRegistered(),
		Threshold:      base.GetThreshold(),
		AccumRoundDate: base.GetAccumRoundDate(),
		AccumUserSpend: base.GetAccumUserSpend(),
		AccumEligible:  base.GetAccumEligible(),
	}, nil
}

func (s *Sub2APIUsecase) LotteryHistoryPublic(ctx context.Context, jwt string, page, pageSize int) ([]*v1.LotteryHistoryItem, int, error) {
	rounds, total, mine, winnersByRound, err := s.lottery.UserHistory(ctx, jwt, page, pageSize)
	if err != nil {
		return nil, 0, mapLotteryError(err)
	}
	items := make([]*v1.LotteryHistoryItem, 0, len(rounds))
	for _, r := range rounds {
		item := &v1.LotteryHistoryItem{
			Round:    toV1LotteryRound(r),
			MyStatus: v1.LotteryMyStatus_LOTTERY_MY_STATUS_NONE,
			Winners:  toV1LotteryParticipants(winnersByRound[r.ID]),
		}
		if p, ok := mine[r.ID]; ok && p != nil {
			item.Registered = true
			item.IsWinner = p.IsWinner
			item.MyPrize = p.PrizeAmount
			// 中奖视角：只有实际发放到账才算「已获得」；已开奖未发放仍为 REGISTERED，前端据 is_winner 显示「已开奖」。
			if p.IsWinner && p.PayoutStatus == sub2apipkg.PayoutPaid {
				item.MyStatus = v1.LotteryMyStatus_LOTTERY_MY_STATUS_WON
			} else {
				item.MyStatus = v1.LotteryMyStatus_LOTTERY_MY_STATUS_REGISTERED
			}
		}
		items = append(items, item)
	}
	return items, total, nil
}
// ---------- 映射 ----------

func mapLotteryError(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, sub2apipkg.ErrLotteryDisabled):
		return ErrLotteryDisabled
	case errors.Is(err, sub2apipkg.ErrLotteryClosed):
		return ErrLotteryClosed
	case errors.Is(err, sub2apipkg.ErrLotteryDrawNotDue):
		return ErrLotteryDrawNotDue
	case errors.Is(err, sub2apipkg.ErrLotteryNotEligible):
		return ErrLotteryNotEligible
	case errors.Is(err, sub2apipkg.ErrLotteryTokenInvalid):
		return ErrLotteryTokenInvalid
	case errors.Is(err, sub2apipkg.ErrLotteryRoundNotFound):
		return ErrLotteryRoundNotFound
	case errors.Is(err, sub2apipkg.ErrLotteryOutsideWindow):
		return ErrLotteryOutsideWindow
	default:
		return ErrSystem(err)
	}
}

func toV1LotterySettings(s sub2apipkg.LotterySettings) *v1.LotterySettings {
	return &v1.LotterySettings{
		Enabled:     s.Enabled,
		PoolRatio:   s.PoolRatio,
		Threshold:   s.Threshold,
		BaseWinners: int32(s.BaseWinners),
		MaxWinners:  int32(s.MaxWinners),
		AutoPayout:  s.AutoPayout,
	}
}

func fromV1LotterySettings(s *v1.LotterySettings) sub2apipkg.LotterySettings {
	if s == nil {
		return sub2apipkg.DefaultLotterySettings()
	}
	return sub2apipkg.LotterySettings{
		Enabled:     s.Enabled,
		PoolRatio:   s.PoolRatio,
		Threshold:   s.Threshold,
		BaseWinners: int(s.BaseWinners),
		MaxWinners:  int(s.MaxWinners),
		AutoPayout:  s.AutoPayout,
	}
}

func toV1LotteryRound(r *sub2apipkg.LotteryRound) *v1.LotteryRound {
	if r == nil {
		return nil
	}
	return &v1.LotteryRound{
		Id:              r.ID,
		SourceDate:      r.SourceDate,
		SettleTime:      timestamppb.New(r.SettleTime),
		DrawTime:        timestamppb.New(r.DrawTime),
		Status:          toV1LotteryRoundStatus(r.Status),
		PoolRatio:       r.PoolRatio,
		Threshold:       r.Threshold,
		BaseWinners:     int32(r.BaseWinners),
		MaxWinners:      int32(r.MaxWinners),
		GroupSpendTotal: r.GroupSpendTotal,
		CarryIn:         r.CarryIn,
		PoolAmount:      r.PoolAmount,
		EligibleCount:   int32(r.EligibleCount),
		RegisteredCount: int32(r.RegisteredCount),
		WinnerCount:     int32(r.WinnerCount),
		PerWinnerAmount: r.PerWinnerAmount,
		CarryOut:        r.CarryOut,
		AutoPayout:      r.AutoPayout,
		Distributed:     r.Distributed,
	}
}

func toV1LotteryRoundStatus(s sub2apipkg.LotteryRoundStatus) v1.LotteryRoundStatus {
	switch s {
	case sub2apipkg.LotteryRoundRegistering:
		return v1.LotteryRoundStatus_LOTTERY_ROUND_STATUS_REGISTERING
	case sub2apipkg.LotteryRoundDrawn:
		return v1.LotteryRoundStatus_LOTTERY_ROUND_STATUS_DRAWN
	default:
		return v1.LotteryRoundStatus_LOTTERY_ROUND_STATUS_UNSPECIFIED
	}
}

func toV1LotteryPayoutStatus(s sub2apipkg.LotteryPayoutStatus) v1.LotteryPayoutStatus {
	switch s {
	case sub2apipkg.PayoutNone:
		return v1.LotteryPayoutStatus_LOTTERY_PAYOUT_STATUS_NONE
	case sub2apipkg.PayoutPending:
		return v1.LotteryPayoutStatus_LOTTERY_PAYOUT_STATUS_PENDING
	case sub2apipkg.PayoutPaid:
		return v1.LotteryPayoutStatus_LOTTERY_PAYOUT_STATUS_PAID
	case sub2apipkg.PayoutFailed:
		return v1.LotteryPayoutStatus_LOTTERY_PAYOUT_STATUS_FAILED
	default:
		return v1.LotteryPayoutStatus_LOTTERY_PAYOUT_STATUS_UNSPECIFIED
	}
}

func toV1LotteryParticipant(p *sub2apipkg.LotteryParticipant) *v1.LotteryParticipant {
	if p == nil {
		return nil
	}
	out := &v1.LotteryParticipant{
		Id:            int32(p.ID),
		RoundId:       p.RoundID,
		Sub2ApiUserId: p.Sub2APIUserID,
		UserName:      p.UserName,
		SpendSnapshot: p.SpendSnapshot,
		IsWinner:      p.IsWinner,
		PrizeAmount:   p.PrizeAmount,
		PayoutStatus:  toV1LotteryPayoutStatus(p.PayoutStatus),
		PayoutError:   p.PayoutError,
	}
	if !p.RegisteredTime.IsZero() {
		out.RegisteredTime = timestamppb.New(p.RegisteredTime)
	}
	return out
}

func toV1Sub2APIUserInfo(u *sub2apipkg.Sub2APIUserInfo) *v1.Sub2APIUserInfo {
	if u == nil {
		return nil
	}
	out := &v1.Sub2APIUserInfo{
		Id:       u.ID,
		Username: u.Username,
		Email:    u.Email,
		Role:     u.Role,
		Balance:  u.Balance,
		Status:   u.Status,
	}
	if !u.CreatedAt.IsZero() {
		out.CreatedAt = timestamppb.New(u.CreatedAt)
	}
	return out
}

func toV1LotteryParticipants(list []*sub2apipkg.LotteryParticipant) []*v1.LotteryParticipant {
	out := make([]*v1.LotteryParticipant, 0, len(list))
	for _, p := range list {
		out = append(out, toV1LotteryParticipant(p))
	}
	return out
}

func toV1LotteryUserStatus(st *sub2apipkg.LotteryUserStatus) *v1.GetLotteryStatusResponse {
	if st == nil {
		return &v1.GetLotteryStatusResponse{}
	}
	out := &v1.GetLotteryStatusResponse{
		Enabled:        st.Enabled,
		Authenticated:  st.Authenticated,
		UserId:         st.UserID,
		UserName:       st.UserName,
		AccumPool:      st.AccumPool,
		Current:        toV1LotteryRound(st.Current),
		Eligible:       st.Eligible,
		UserSpend:      st.UserSpend,
		Registered:     st.Registered,
		Threshold:      st.Threshold,
		AccumRoundDate: st.AccumRoundDate,
		AccumUserSpend: st.AccumUserSpend,
		AccumEligible:  st.AccumEligible,
	}
	if !st.NextSettleTime.IsZero() {
		out.NextSettleTime = timestamppb.New(st.NextSettleTime)
	}
	if !st.NextDrawTime.IsZero() {
		out.NextDrawTime = timestamppb.New(st.NextDrawTime)
	}
	return out
}

