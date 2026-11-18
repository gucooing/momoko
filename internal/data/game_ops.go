package data

import (
	"context"
	"time"

	"entgo.io/ent/dialect/sql"
	"github.com/google/uuid"

	"momoko/internal/biz"
	"momoko/internal/data/ent/gen"
	entcode "momoko/internal/data/ent/gen/giftcode"
	entbatch "momoko/internal/data/ent/gen/giftcodebatch"
	entredeem "momoko/internal/data/ent/gen/giftcoderedeem"
	"momoko/internal/data/ent/gen/predicate"
)

type gameOpsRepo struct {
	data *Data
}

func NewGameOpsRepo(data *Data) biz.GameOpsRepo {
	return &gameOpsRepo{data: data}
}

// pageOf 规范化分页参数。
func pageOf(page, size, def, maxSize int) (offset, limit int) {
	if size <= 0 || size > maxSize {
		size = def
	}
	if page <= 0 {
		page = 1
	}
	return (page - 1) * size, size
}

// ---------------- 兑换码批次 ----------------

const codeChunk = 500

func chunks(list []string, size int) [][]string {
	var out [][]string
	for len(list) > 0 {
		n := min(size, len(list))
		out = append(out, list[:n])
		list = list[n:]
	}
	return out
}

func (r *gameOpsRepo) CreateBatch(ctx context.Context, b *gen.GiftCodeBatch, codes []string) (*gen.GiftCodeBatch, error) {
	var out *gen.GiftCodeBatch
	err := WithTx(ctx, r.data.db, func(tx *gen.Tx) error {
		c := tx.GiftCodeBatch.Create().
			SetID(b.ID).
			SetServerID(b.ServerID).
			SetName(b.Name).
			SetKind(b.Kind).
			SetCode(b.Code).
			SetPrefix(b.Prefix).
			SetQuantity(b.Quantity).
			SetRewards(b.Rewards).
			SetMessage(b.Message).
			SetTotalLimit(b.TotalLimit).
			SetPerPlayerLimit(b.PerPlayerLimit).
			SetNillableStartAt(b.StartAt).
			SetNillableEndAt(b.EndAt).
			SetEligibility(b.Eligibility).
			SetRegisteredWithinDays(b.RegisteredWithinDays).
			SetChannels(b.Channels).
			SetCreatedBy(b.CreatedBy).
			SetCreatedByName(b.CreatedByName)
		var err error
		if out, err = c.Save(ctx); err != nil {
			return err
		}
		for _, part := range chunks(codes, codeChunk) {
			builders := make([]*gen.GiftCodeCreate, 0, len(part))
			for _, code := range part {
				builders = append(builders, tx.GiftCode.Create().
					SetID(uuid.NewString()).
					SetBatchID(b.ID).
					SetServerID(b.ServerID).
					SetCode(code))
			}
			if err := tx.GiftCode.CreateBulk(builders...).Exec(ctx); err != nil {
				return err
			}
		}
		return nil
	})
	return out, err
}

func (r *gameOpsRepo) SaveBatch(ctx context.Context, b *gen.GiftCodeBatch) (*gen.GiftCodeBatch, error) {
	u := r.data.db.GiftCodeBatch.UpdateOneID(b.ID).
		SetName(b.Name).
		SetMessage(b.Message).
		SetTotalLimit(b.TotalLimit).
		SetPerPlayerLimit(b.PerPlayerLimit).
		SetChannels(b.Channels).
		SetDisabledByName(b.DisabledByName)
	if b.EndAt != nil {
		u.SetEndAt(*b.EndAt)
	} else {
		u.ClearEndAt()
	}
	if b.DisabledAt != nil {
		u.SetDisabledAt(*b.DisabledAt)
	} else {
		u.ClearDisabledAt()
	}
	return u.Save(ctx)
}

func (r *gameOpsRepo) GetBatch(ctx context.Context, id string) (*gen.GiftCodeBatch, error) {
	return r.data.db.GiftCodeBatch.Get(ctx, id)
}

// activeBatch 进行中（含未开始）：未停用、未过期、未用完。
func activeBatch(now time.Time) predicate.GiftCodeBatch {
	return entbatch.And(
		entbatch.DisabledAtIsNil(),
		entbatch.Or(entbatch.EndAtIsNil(), entbatch.EndAtGT(now)),
		entbatch.Or(
			entbatch.TotalLimit(0),
			predicate.GiftCodeBatch(func(s *sql.Selector) {
				s.Where(sql.ColumnsLT(s.C(entbatch.FieldUsedCount), s.C(entbatch.FieldTotalLimit)))
			}),
		),
	)
}

func (r *gameOpsRepo) ListBatches(ctx context.Context, f biz.GiftBatchFilter) ([]*gen.GiftCodeBatch, int, error) {
	q := r.data.db.GiftCodeBatch.Query().Where(entbatch.ServerID(f.ServerID))
	switch f.State {
	case "active":
		q = q.Where(activeBatch(f.Now))
	case "ended":
		q = q.Where(entbatch.Not(activeBatch(f.Now)))
	}
	if kw := f.Q; kw != "" {
		preds := []predicate.GiftCodeBatch{
			entbatch.NameContainsFold(kw),
			entbatch.CodeContainsFold(kw),
			entbatch.PrefixContainsFold(kw),
		}
		// 一次性码按码本身也能搜到所属批次。
		ids, err := r.data.db.GiftCode.Query().
			Where(entcode.ServerID(f.ServerID), entcode.CodeContainsFold(kw)).
			Limit(200).
			Select(entcode.FieldBatchID).
			Strings(ctx)
		if err != nil {
			return nil, 0, err
		}
		if len(ids) > 0 {
			preds = append(preds, entbatch.IDIn(ids...))
		}
		q = q.Where(entbatch.Or(preds...))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	offset, limit := pageOf(f.Page, f.Size, 20, 100)
	items, err := q.Order(entbatch.ByCreateTime(sql.OrderDesc())).Offset(offset).Limit(limit).All(ctx)
	return items, total, err
}

func (r *gameOpsRepo) CountBatches(ctx context.Context, serverID string, now time.Time) (map[string]int32, error) {
	all, err := r.data.db.GiftCodeBatch.Query().Where(entbatch.ServerID(serverID)).Count(ctx)
	if err != nil {
		return nil, err
	}
	active, err := r.data.db.GiftCodeBatch.Query().Where(entbatch.ServerID(serverID), activeBatch(now)).Count(ctx)
	if err != nil {
		return nil, err
	}
	return map[string]int32{"all": int32(all), "active": int32(active), "ended": int32(all - active)}, nil
}

func (r *gameOpsRepo) BatchNames(ctx context.Context, ids []string) (map[string]string, error) {
	out := map[string]string{}
	if len(ids) == 0 {
		return out, nil
	}
	items, err := r.data.db.GiftCodeBatch.Query().
		Where(entbatch.IDIn(ids...)).
		Select(entbatch.FieldID, entbatch.FieldName).
		All(ctx)
	if err != nil {
		return nil, err
	}
	for _, b := range items {
		out[b.ID] = b.Name
	}
	return out, nil
}

func (r *gameOpsRepo) CodesTaken(ctx context.Context, serverID string, codes []string) (map[string]struct{}, error) {
	out := map[string]struct{}{}
	for _, part := range chunks(codes, codeChunk) {
		unique, err := r.data.db.GiftCode.Query().
			Where(entcode.ServerID(serverID), entcode.CodeIn(part...)).
			Select(entcode.FieldCode).
			Strings(ctx)
		if err != nil {
			return nil, err
		}
		universal, err := r.data.db.GiftCodeBatch.Query().
			Where(entbatch.ServerID(serverID), entbatch.CodeIn(part...)).
			Select(entbatch.FieldCode).
			Strings(ctx)
		if err != nil {
			return nil, err
		}
		for _, c := range append(unique, universal...) {
			out[c] = struct{}{}
		}
	}
	return out, nil
}

func (r *gameOpsRepo) FindCode(ctx context.Context, serverID, code string) (*gen.GiftCodeBatch, *gen.GiftCode, error) {
	gc, err := r.data.db.GiftCode.Query().Where(entcode.ServerID(serverID), entcode.Code(code)).Only(ctx)
	switch {
	case err == nil:
		b, err := r.data.db.GiftCodeBatch.Get(ctx, gc.BatchID)
		if gen.IsNotFound(err) {
			return nil, nil, nil
		}
		return b, gc, err
	case !gen.IsNotFound(err):
		return nil, nil, err
	}
	b, err := r.data.db.GiftCodeBatch.Query().
		Where(entbatch.ServerID(serverID), entbatch.KindEQ(entbatch.KindUniversal), entbatch.Code(code)).
		Order(entbatch.ByCreateTime(sql.OrderDesc())).
		First(ctx)
	if gen.IsNotFound(err) {
		return nil, nil, nil
	}
	return b, nil, err
}

func (r *gameOpsRepo) GetCode(ctx context.Context, id string) (*gen.GiftCode, error) {
	return r.data.db.GiftCode.Get(ctx, id)
}

func (r *gameOpsRepo) ReserveUse(ctx context.Context, batchID string) (bool, error) {
	n, err := r.data.db.GiftCodeBatch.Update().
		Where(
			entbatch.ID(batchID),
			entbatch.DisabledAtIsNil(),
			entbatch.Or(
				entbatch.TotalLimit(0),
				predicate.GiftCodeBatch(func(s *sql.Selector) {
					s.Where(sql.ColumnsLT(s.C(entbatch.FieldUsedCount), s.C(entbatch.FieldTotalLimit)))
				}),
			),
		).
		AddUsedCount(1).
		Save(ctx)
	return n > 0, err
}

func (r *gameOpsRepo) ReleaseUse(ctx context.Context, batchID string) error {
	return r.data.db.GiftCodeBatch.Update().
		Where(entbatch.ID(batchID), entbatch.UsedCountGT(0)).
		AddUsedCount(-1).
		Exec(ctx)
}

func (r *gameOpsRepo) ClaimCode(ctx context.Context, codeID, pid, name string, at time.Time) (bool, error) {
	n, err := r.data.db.GiftCode.Update().
		Where(entcode.ID(codeID), entcode.UsedByPlayer("")).
		SetUsedByPlayer(pid).
		SetUsedByName(name).
		SetUsedAt(at).
		Save(ctx)
	return n > 0, err
}

func (r *gameOpsRepo) ReleaseCode(ctx context.Context, codeID string) error {
	return r.data.db.GiftCode.UpdateOneID(codeID).
		SetUsedByPlayer("").
		SetUsedByName("").
		ClearUsedAt().
		Exec(ctx)
}

func (r *gameOpsRepo) ListCodes(ctx context.Context, batchID, state string, page, size int) ([]*gen.GiftCode, int, error) {
	q := r.data.db.GiftCode.Query().Where(entcode.BatchID(batchID))
	switch state {
	case "used":
		q = q.Where(entcode.UsedByPlayerNEQ(""))
	case "unused":
		q = q.Where(entcode.UsedByPlayer(""))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	offset, limit := pageOf(page, size, 100, 1000)
	items, err := q.Order(entcode.ByCode()).Offset(offset).Limit(limit).All(ctx)
	return items, total, err
}

// ---------------- 兑换记录 ----------------

func (r *gameOpsRepo) CreateRedeem(ctx context.Context, rec *gen.GiftCodeRedeem) (*gen.GiftCodeRedeem, error) {
	if rec.ID == "" {
		rec.ID = uuid.NewString()
	}
	return r.data.db.GiftCodeRedeem.Create().
		SetID(rec.ID).
		SetServerID(rec.ServerID).
		SetBatchID(rec.BatchID).
		SetCode(rec.Code).
		SetPlayerID(rec.PlayerID).
		SetNickname(rec.Nickname).
		SetPassportUserID(rec.PassportUserID).
		SetChannel(rec.Channel).
		SetPlatform(rec.Platform).
		SetResult(rec.Result).
		SetErrorMessage(rec.ErrorMessage).
		SetExecutionID(rec.ExecutionID).
		Save(ctx)
}

func (r *gameOpsRepo) ListRedeems(ctx context.Context, f biz.GiftRedeemFilter) ([]*gen.GiftCodeRedeem, int, error) {
	q := r.data.db.GiftCodeRedeem.Query().Where(entredeem.ServerID(f.ServerID))
	if f.BatchID != "" {
		q = q.Where(entredeem.BatchID(f.BatchID))
	}
	switch f.Result {
	case "":
	case "success":
		q = q.Where(entredeem.ResultEQ(entredeem.ResultSuccess))
	case "failed":
		q = q.Where(entredeem.ResultNEQ(entredeem.ResultSuccess))
	default:
		q = q.Where(entredeem.ResultEQ(entredeem.Result(f.Result)))
	}
	total, err := q.Clone().Count(ctx)
	if err != nil {
		return nil, 0, err
	}
	offset, limit := pageOf(f.Page, f.Size, 20, 500)
	items, err := q.Order(entredeem.ByCreateTime(sql.OrderDesc())).Offset(offset).Limit(limit).All(ctx)
	return items, total, err
}

func (r *gameOpsRepo) CountRedeems(ctx context.Context, batchID string) (int, int, error) {
	success, err := r.data.db.GiftCodeRedeem.Query().
		Where(entredeem.BatchID(batchID), entredeem.ResultEQ(entredeem.ResultSuccess)).
		Count(ctx)
	if err != nil {
		return 0, 0, err
	}
	failure, err := r.data.db.GiftCodeRedeem.Query().
		Where(entredeem.BatchID(batchID), entredeem.ResultNEQ(entredeem.ResultSuccess)).
		Count(ctx)
	return success, failure, err
}

func (r *gameOpsRepo) CountPlayerRedeems(ctx context.Context, serverID, batchID, pid string) (int, error) {
	return r.data.db.GiftCodeRedeem.Query().
		Where(
			entredeem.ServerID(serverID),
			entredeem.PlayerID(pid),
			entredeem.BatchID(batchID),
			entredeem.ResultEQ(entredeem.ResultSuccess),
		).
		Count(ctx)
}

func (r *gameOpsRepo) CountAttempts(ctx context.Context, passportUserID string, since time.Time) (int, error) {
	return r.data.db.GiftCodeRedeem.Query().
		Where(
			entredeem.PassportUserID(passportUserID),
			entredeem.CreateTimeGTE(since),
			entredeem.ResultNEQ(entredeem.ResultRateLimited),
		).
		Count(ctx)
}

func (r *gameOpsRepo) RedeemStats(ctx context.Context, serverID string, dayStart time.Time) (*biz.GiftRedeemStats, error) {
	out := &biz.GiftRedeemStats{TodayByBatch: map[string]int{}, TodayFailures: map[string]int{}}
	var byBatch []struct {
		BatchID string `json:"batch_id"`
		Count   int    `json:"count"`
	}
	if err := r.data.db.GiftCodeRedeem.Query().
		Where(entredeem.ServerID(serverID), entredeem.ResultEQ(entredeem.ResultSuccess), entredeem.CreateTimeGTE(dayStart)).
		GroupBy(entredeem.FieldBatchID).
		Aggregate(gen.Count()).
		Scan(ctx, &byBatch); err != nil {
		return nil, err
	}
	for _, row := range byBatch {
		out.TodayByBatch[row.BatchID] = row.Count
		out.TodaySuccess += row.Count
	}
	var byResult []struct {
		Result string `json:"result"`
		Count  int    `json:"count"`
	}
	if err := r.data.db.GiftCodeRedeem.Query().
		Where(entredeem.ServerID(serverID), entredeem.ResultNEQ(entredeem.ResultSuccess), entredeem.CreateTimeGTE(dayStart)).
		GroupBy(entredeem.FieldResult).
		Aggregate(gen.Count()).
		Scan(ctx, &byResult); err != nil {
		return nil, err
	}
	for _, row := range byResult {
		out.TodayFailures[row.Result] = row.Count
	}
	success := r.data.db.GiftCodeRedeem.Query().Where(entredeem.ServerID(serverID), entredeem.ResultEQ(entredeem.ResultSuccess))
	total, err := success.Clone().Count(ctx)
	if err != nil {
		return nil, err
	}
	out.TotalSuccess = total
	if total > 0 {
		first, err := success.Order(entredeem.ByCreateTime()).First(ctx)
		if err != nil && !gen.IsNotFound(err) {
			return nil, err
		}
		if first != nil {
			t := first.CreateTime
			out.FirstSuccess = &t
		}
	}
	return out, nil
}
