package data

import (
	"context"
	"time"

	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/sub2apigroup"
	"momoko/internal/data/ent/gen/sub2apilotteryparticipant"
	"momoko/internal/data/ent/gen/sub2apilotteryround"
	"momoko/internal/data/ent/gen/sub2apiusagerecord"
	sub2apipkg "momoko/pkg/sub2api"
)

// ---------- 抽奖：扣费聚合（ent Sum / GroupBy，禁止内存累加） ----------

// publicUsageQuery 构造「展示分组 + 时间区间」的使用记录查询。
func (r *sub2APIRepo) publicUsageQuery(start, end time.Time) *gen.Sub2APIUsageRecordQuery {
	return r.data.db.Sub2APIUsageRecord.Query().
		Where(
			sub2apiusagerecord.RequestTimeGTE(start),
			sub2apiusagerecord.RequestTimeLT(end),
			sub2apiusagerecord.HasGroupWith(sub2apigroup.PublicEnabledEQ(true)),
		)
}

// SumPublicCostInRange 展示分组在 [start,end) 内 cost 之和。
func (r *sub2APIRepo) SumPublicCostInRange(ctx context.Context, start, end time.Time) (float64, error) {
	var v []struct {
		Sum float64 `json:"sum"`
	}
	err := r.publicUsageQuery(start, end).
		Aggregate(gen.As(gen.Sum(sub2apiusagerecord.FieldCost), "sum")).
		Scan(ctx, &v)
	if err != nil {
		return 0, err
	}
	if len(v) == 0 {
		return 0, nil
	}
	return v[0].Sum, nil
}

// CountEligibleUsersInRange 展示分组 [start,end) 内，按 user 汇总 cost ≥ threshold 的用户数。
func (r *sub2APIRepo) CountEligibleUsersInRange(ctx context.Context, start, end time.Time, threshold float64) (int, error) {
	var rows []struct {
		UserID int64   `json:"user_id"`
		Sum    float64 `json:"sum"`
	}
	err := r.publicUsageQuery(start, end).
		Where(sub2apiusagerecord.UserIDNotNil()).
		GroupBy(sub2apiusagerecord.FieldUserID).
		Aggregate(gen.As(gen.Sum(sub2apiusagerecord.FieldCost), "sum")).
		Scan(ctx, &rows)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, row := range rows {
		if row.UserID > 0 && row.Sum >= threshold {
			count++
		}
	}
	return count, nil
}

// UserSpendInRange 某用户在展示分组、[start,end) 内的扣费之和（ent Sum）。
func (r *sub2APIRepo) UserSpendInRange(ctx context.Context, userID int64, start, end time.Time) (float64, error) {
	if userID <= 0 {
		return 0, nil
	}
	var v []struct {
		Sum float64 `json:"sum"`
	}
	err := r.publicUsageQuery(start, end).
		Where(sub2apiusagerecord.UserIDEQ(userID)).
		Aggregate(gen.As(gen.Sum(sub2apiusagerecord.FieldCost), "sum")).
		Scan(ctx, &v)
	if err != nil {
		return 0, err
	}
	if len(v) == 0 {
		return 0, nil
	}
	return v[0].Sum, nil
}

// ---------- 抽奖：轮次 CRUD ----------

func (r *sub2APIRepo) GetLotteryRound(ctx context.Context, id string) (*sub2apipkg.LotteryRound, error) {
	row, err := r.data.db.Sub2APILotteryRound.Get(ctx, id)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return toLotteryRound(row), nil
}

func (r *sub2APIRepo) LatestLotteryRound(ctx context.Context) (*sub2apipkg.LotteryRound, error) {
	// id = YYYY-MM-DD，字典序 = 时间序
	row, err := r.data.db.Sub2APILotteryRound.Query().
		Order(gen.Desc(sub2apilotteryround.FieldID)).
		First(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return toLotteryRound(row), nil
}

// ListRegisteringRounds 全部仍处于报名中的轮次（Tick 补开奖用）。
func (r *sub2APIRepo) ListRegisteringRounds(ctx context.Context) ([]*sub2apipkg.LotteryRound, error) {
	rows, err := r.data.db.Sub2APILotteryRound.Query().
		Where(sub2apilotteryround.StatusEQ(sub2apilotteryround.StatusRegistering)).
		Order(gen.Asc(sub2apilotteryround.FieldID)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]*sub2apipkg.LotteryRound, 0, len(rows))
	for _, row := range rows {
		out = append(out, toLotteryRound(row))
	}
	return out, nil
}

// SaveLotteryRound 按 ID upsert：已存在则更新，否则创建。
func (r *sub2APIRepo) SaveLotteryRound(ctx context.Context, rd *sub2apipkg.LotteryRound) error {
	exists, err := r.data.db.Sub2APILotteryRound.Query().
		Where(sub2apilotteryround.IDEQ(rd.ID)).
		Exist(ctx)
	if err != nil {
		return err
	}
	if exists {
		return r.data.db.Sub2APILotteryRound.UpdateOneID(rd.ID).
			SetSourceDate(rd.SourceDate).
			SetSettleTime(rd.SettleTime).
			SetDrawTime(rd.DrawTime).
			SetStatus(toEntRoundStatus(rd.Status)).
			SetPoolRatio(rd.PoolRatio).
			SetThreshold(rd.Threshold).
			SetBaseWinners(rd.BaseWinners).
			SetMaxWinners(rd.MaxWinners).
			SetGroupSpendTotal(rd.GroupSpendTotal).
			SetCarryIn(rd.CarryIn).
			SetPoolAmount(rd.PoolAmount).
			SetEligibleCount(rd.EligibleCount).
			SetRegisteredCount(rd.RegisteredCount).
			SetWinnerCount(rd.WinnerCount).
			SetPerWinnerAmount(rd.PerWinnerAmount).
			SetCarryOut(rd.CarryOut).
			SetAutoPayout(rd.AutoPayout).
			SetDistributed(rd.Distributed).
			Exec(ctx)
	}
	return r.data.db.Sub2APILotteryRound.Create().
		SetID(rd.ID).
		SetSourceDate(rd.SourceDate).
		SetSettleTime(rd.SettleTime).
		SetDrawTime(rd.DrawTime).
		SetStatus(toEntRoundStatus(rd.Status)).
		SetPoolRatio(rd.PoolRatio).
		SetThreshold(rd.Threshold).
		SetBaseWinners(rd.BaseWinners).
		SetMaxWinners(rd.MaxWinners).
		SetGroupSpendTotal(rd.GroupSpendTotal).
		SetCarryIn(rd.CarryIn).
		SetPoolAmount(rd.PoolAmount).
		SetEligibleCount(rd.EligibleCount).
		SetRegisteredCount(rd.RegisteredCount).
		SetWinnerCount(rd.WinnerCount).
		SetPerWinnerAmount(rd.PerWinnerAmount).
		SetCarryOut(rd.CarryOut).
		SetAutoPayout(rd.AutoPayout).
		SetDistributed(rd.Distributed).
		Exec(ctx)
}

// ListDrawnLotteryRounds 仅已开奖轮次（真正历史）；registering 不进历史。
func (r *sub2APIRepo) ListDrawnLotteryRounds(ctx context.Context, offset, limit int) ([]*sub2apipkg.LotteryRound, int, error) {
	q := r.data.db.Sub2APILotteryRound.Query().
		Where(sub2apilotteryround.StatusEQ(sub2apilotteryround.StatusDrawn))
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	rows, err := q.
		Order(gen.Desc(sub2apilotteryround.FieldID)).
		Offset(offset).
		Limit(limit).
		All(ctx)
	if err != nil {
		return nil, 0, err
	}
	out := make([]*sub2apipkg.LotteryRound, 0, len(rows))
	for _, row := range rows {
		out = append(out, toLotteryRound(row))
	}
	return out, total, nil
}

// ---------- 抽奖：参与者 CRUD ----------

func (r *sub2APIRepo) GetParticipant(ctx context.Context, roundID string, userID int64) (*sub2apipkg.LotteryParticipant, error) {
	row, err := r.data.db.Sub2APILotteryParticipant.Query().
		Where(
			sub2apilotteryparticipant.RoundIDEQ(roundID),
			sub2apilotteryparticipant.Sub2apiUserIDEQ(userID),
		).
		Only(ctx)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return toLotteryParticipant(row), nil
}

func (r *sub2APIRepo) CreateParticipant(ctx context.Context, p *sub2apipkg.LotteryParticipant) error {
	status := p.PayoutStatus
	if status == "" {
		status = sub2apipkg.PayoutNone
	}
	return r.data.db.Sub2APILotteryParticipant.Create().
		SetRoundID(p.RoundID).
		SetSub2apiUserID(p.Sub2APIUserID).
		SetUserName(p.UserName).
		SetSpendSnapshot(p.SpendSnapshot).
		SetRegisteredTime(p.RegisteredTime).
		SetIsWinner(p.IsWinner).
		SetPrizeAmount(p.PrizeAmount).
		SetPayoutStatus(toEntPayoutStatus(status)).
		Exec(ctx)
}

func (r *sub2APIRepo) CountParticipants(ctx context.Context, roundID string) (int, error) {
	return r.data.db.Sub2APILotteryParticipant.Query().
		Where(sub2apilotteryparticipant.RoundIDEQ(roundID)).
		Count(ctx)
}

func (r *sub2APIRepo) ListParticipants(ctx context.Context, roundID string) ([]*sub2apipkg.LotteryParticipant, error) {
	rows, err := r.data.db.Sub2APILotteryParticipant.Query().
		Where(sub2apilotteryparticipant.RoundIDEQ(roundID)).
		Order(gen.Asc(sub2apilotteryparticipant.FieldRegisteredTime)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return toLotteryParticipants(rows), nil
}

func (r *sub2APIRepo) ListWinners(ctx context.Context, roundID string) ([]*sub2apipkg.LotteryParticipant, error) {
	rows, err := r.data.db.Sub2APILotteryParticipant.Query().
		Where(
			sub2apilotteryparticipant.RoundIDEQ(roundID),
			sub2apilotteryparticipant.IsWinnerEQ(true),
		).
		Order(gen.Desc(sub2apilotteryparticipant.FieldPrizeAmount)).
		All(ctx)
	if err != nil {
		return nil, err
	}
	return toLotteryParticipants(rows), nil
}

func (r *sub2APIRepo) UpdateParticipant(ctx context.Context, p *sub2apipkg.LotteryParticipant) error {
	return r.data.db.Sub2APILotteryParticipant.UpdateOneID(p.ID).
		SetIsWinner(p.IsWinner).
		SetPrizeAmount(p.PrizeAmount).
		SetPayoutStatus(toEntPayoutStatus(p.PayoutStatus)).
		SetPayoutIdempotencyKey(p.PayoutIdempotencyKey).
		SetPayoutError(p.PayoutError).
		Exec(ctx)
}

func (r *sub2APIRepo) ParticipationsByUser(ctx context.Context, userID int64, roundIDs []string) (map[string]*sub2apipkg.LotteryParticipant, error) {
	out := make(map[string]*sub2apipkg.LotteryParticipant)
	if userID <= 0 || len(roundIDs) == 0 {
		return out, nil
	}
	rows, err := r.data.db.Sub2APILotteryParticipant.Query().
		Where(
			sub2apilotteryparticipant.Sub2apiUserIDEQ(userID),
			sub2apilotteryparticipant.RoundIDIn(roundIDs...),
		).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, row := range rows {
		out[row.RoundID] = toLotteryParticipant(row)
	}
	return out, nil
}

// ---------- 转换 ----------

func toLotteryRound(row *gen.Sub2APILotteryRound) *sub2apipkg.LotteryRound {
	if row == nil {
		return nil
	}
	return &sub2apipkg.LotteryRound{
		ID:              row.ID,
		SourceDate:      row.SourceDate,
		SettleTime:      row.SettleTime,
		DrawTime:        row.DrawTime,
		Status:          fromEntRoundStatus(row.Status),
		PoolRatio:       row.PoolRatio,
		Threshold:       row.Threshold,
		BaseWinners:     row.BaseWinners,
		MaxWinners:      row.MaxWinners,
		GroupSpendTotal: row.GroupSpendTotal,
		CarryIn:         row.CarryIn,
		PoolAmount:      row.PoolAmount,
		EligibleCount:   row.EligibleCount,
		RegisteredCount: row.RegisteredCount,
		WinnerCount:     row.WinnerCount,
		PerWinnerAmount: row.PerWinnerAmount,
		CarryOut:        row.CarryOut,
		AutoPayout:      row.AutoPayout,
		Distributed:     row.Distributed,
		CreateTime:      row.CreateTime,
		UpdateTime:      row.UpdateTime,
	}
}

func toLotteryParticipant(row *gen.Sub2APILotteryParticipant) *sub2apipkg.LotteryParticipant {
	if row == nil {
		return nil
	}
	return &sub2apipkg.LotteryParticipant{
		ID:                   row.ID,
		RoundID:              row.RoundID,
		Sub2APIUserID:        row.Sub2apiUserID,
		UserName:             row.UserName,
		SpendSnapshot:        row.SpendSnapshot,
		RegisteredTime:       row.RegisteredTime,
		IsWinner:             row.IsWinner,
		PrizeAmount:          row.PrizeAmount,
		PayoutStatus:         fromEntPayoutStatus(row.PayoutStatus),
		PayoutIdempotencyKey: row.PayoutIdempotencyKey,
		PayoutError:          row.PayoutError,
	}
}

// 领域枚举 ↔ ent 枚举（一一对应，禁止 string 裸转）。
func toEntRoundStatus(s sub2apipkg.LotteryRoundStatus) sub2apilotteryround.Status {
	switch s {
	case sub2apipkg.LotteryRoundDrawn:
		return sub2apilotteryround.StatusDrawn
	default:
		return sub2apilotteryround.StatusRegistering
	}
}

func fromEntRoundStatus(s sub2apilotteryround.Status) sub2apipkg.LotteryRoundStatus {
	switch s {
	case sub2apilotteryround.StatusDrawn:
		return sub2apipkg.LotteryRoundDrawn
	default:
		return sub2apipkg.LotteryRoundRegistering
	}
}

func toEntPayoutStatus(s sub2apipkg.LotteryPayoutStatus) sub2apilotteryparticipant.PayoutStatus {
	switch s {
	case sub2apipkg.PayoutPending:
		return sub2apilotteryparticipant.PayoutStatusPending
	case sub2apipkg.PayoutPaid:
		return sub2apilotteryparticipant.PayoutStatusPaid
	case sub2apipkg.PayoutFailed:
		return sub2apilotteryparticipant.PayoutStatusFailed
	default:
		return sub2apilotteryparticipant.PayoutStatusNone
	}
}

func fromEntPayoutStatus(s sub2apilotteryparticipant.PayoutStatus) sub2apipkg.LotteryPayoutStatus {
	switch s {
	case sub2apilotteryparticipant.PayoutStatusPending:
		return sub2apipkg.PayoutPending
	case sub2apilotteryparticipant.PayoutStatusPaid:
		return sub2apipkg.PayoutPaid
	case sub2apilotteryparticipant.PayoutStatusFailed:
		return sub2apipkg.PayoutFailed
	default:
		return sub2apipkg.PayoutNone
	}
}

func toLotteryParticipants(rows []*gen.Sub2APILotteryParticipant) []*sub2apipkg.LotteryParticipant {
	out := make([]*sub2apipkg.LotteryParticipant, 0, len(rows))
	for _, row := range rows {
		out = append(out, toLotteryParticipant(row))
	}
	return out
}
