package biz

import (
	"context"
	"encoding/json"
	"slices"
	"sync"
	"time"

	"google.golang.org/protobuf/proto"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	entbatch "momoko/internal/data/ent/gen/giftcodebatch"
	entredeem "momoko/internal/data/ent/gen/giftcoderedeem"
	"momoko/pkg/mgp"
	"momoko/pkg/mgp/gakumas"
)

// 兑换频率限制：每个通行证用户每分钟最多尝试的次数（被限流的尝试不计）。
const (
	redeemRateLimit  = 10
	redeemRateWindow = time.Minute
)

// 兑换结果（gift_code_redeem.result）。
const (
	RedeemSuccess         = "success"
	RedeemAlreadyRedeemed = "already_redeemed"
	RedeemExpired         = "expired"
	RedeemNotStarted      = "not_started"
	RedeemExhausted       = "exhausted"
	RedeemDisabled        = "disabled"
	RedeemNotEligible     = "not_eligible"
	RedeemBanned          = "banned"
	RedeemRateLimited     = "rate_limited"
	RedeemNotFound        = "not_found"
	RedeemCodeUsed        = "code_used"
	RedeemChannelClosed   = "channel_closed"
	RedeemFailed          = "failed"
)

// redeemGift 兑换成功时发放奖励的操作（记入 GM 记录，source = code:<批次 id>）。
var redeemGift = PassportOperation{
	ID:     "gift.send",
	Title:  "发送到礼物箱",
	Group:  "发放",
	Risk:   RiskMedium,
	Target: TargetPlayers,
}

// redeemLock 串行化同一通行证用户、同一批次的兑换，避免并发超发或绕过频率限制。
func (uc *PassportUsecase) redeemLock(key string) func() {
	l, _ := uc.redeemLocks.LoadOrStore(key, &sync.Mutex{})
	mu := l.(*sync.Mutex)
	mu.Lock()
	return mu.Unlock
}

// Redeem 兑换码（奖励进礼物箱）：目标为当前用户在这台服务器绑定的游戏账号，每次尝试都写一条 gift_code_redeem。
// 校验顺序：频率 → 码 → 停用 / 未开始 / 过期 / 用完 → 兑换入口 → 一次性码是否已用 → 每人次数 → 封禁 → 适用玩家；
// 通过后先占用次数（与一次性码），再发放（gift.send，sourceKind=code、sourceId=批次 id），失败时释放占用。
// 兑换没有成功（码不存在、过期、次数用完、发放失败……）时返回对应的 result；只有参数或系统错误才返回 error。
func (uc *PassportUsecase) Redeem(ctx context.Context, userID, serverID, code string) (*v1.RedeemPassportGiftCodeResponse, error) {
	l, err := uc.gameLink(ctx, userID, serverID)
	if err != nil {
		return nil, err
	}
	u, err := uc.user(ctx, userID)
	if err != nil {
		return nil, err
	}
	code = NormalizeGiftCode(code)
	if code == "" {
		return nil, ErrPassportInvalidArgument
	}
	if len(code) > 64 {
		code = code[:64]
	}
	ctx = context.WithoutCancel(ctx)
	s, client, err := uc.servers.ClientOf(ctx, serverID)
	if err != nil {
		return nil, err
	}
	conn := PassportClientOf(ctx)
	actor := passportActor{
		UserID:   u.ID,
		Name:     displayName(u),
		IP:       conn.IP,
		Platform: Platform(conn.UserAgent),
	}
	rec := &gen.GiftCodeRedeem{
		ServerID:       s.ID,
		Code:           code,
		PlayerID:       l.PlayerID,
		Nickname:       l.Nickname,
		PassportUserID: u.ID,
		Channel:        entredeem.Channel(GiftChannelPassport),
		Platform:       actor.Platform,
	}
	out := &v1.RedeemPassportGiftCodeResponse{}
	finish := func(result, message string) (*v1.RedeemPassportGiftCodeResponse, error) {
		rec.Result, rec.ErrorMessage = entredeem.Result(result), message
		if _, err := uc.codes.CreateRedeem(ctx, rec); err != nil {
			return nil, ErrSystem(err)
		}
		out.Result, out.ExecutionId = result, rec.ExecutionID
		return out, nil
	}

	unlockUser := uc.redeemLock("user:" + u.ID)
	defer unlockUser()
	now := time.Now()
	attempts, err := uc.codes.CountAttempts(ctx, u.ID, now.Add(-redeemRateWindow))
	if err != nil {
		return nil, ErrSystem(err)
	}
	if attempts >= redeemRateLimit {
		return finish(RedeemRateLimited, "")
	}

	found, gc, err := uc.codes.FindCode(ctx, s.ID, code)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if found == nil {
		return finish(RedeemNotFound, "")
	}
	rec.BatchID = found.ID
	unlockBatch := uc.redeemLock("batch:" + found.ID)
	defer unlockBatch()
	// 拿到锁之后重新读取，保证次数与状态是最新的。
	b, err := uc.codes.GetBatch(ctx, found.ID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	now = time.Now()
	switch GiftBatchStatus(b, now) {
	case "disabled":
		return finish(RedeemDisabled, "")
	case "not_started":
		return finish(RedeemNotStarted, "")
	case "expired":
		return finish(RedeemExpired, "")
	case "exhausted":
		return finish(RedeemExhausted, "")
	}
	if !slices.Contains(b.Channels, GiftChannelPassport) {
		return finish(RedeemChannelClosed, "")
	}
	if gc != nil {
		if gc, err = uc.codes.GetCode(ctx, gc.ID); err != nil {
			return nil, ErrSystem(err)
		}
		if gc.UsedByPlayer != "" {
			if gc.UsedByPlayer == l.PlayerID {
				return finish(RedeemAlreadyRedeemed, "")
			}
			return finish(RedeemCodeUsed, "")
		}
	}
	used, err := uc.codes.CountPlayerRedeems(ctx, s.ID, b.ID, l.PlayerID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if used >= max(b.PerPlayerLimit, 1) {
		return finish(RedeemAlreadyRedeemed, "")
	}
	if s.Game != gakumas.ID {
		return nil, ErrGmGameUnsupported
	}
	g := gakumas.New(client)
	detail, err := g.Player(ctx, l.PlayerID)
	if err != nil {
		return finish(RedeemFailed, errText(GameMGPError(err)))
	}
	player := detail.GetPlayer()
	if player.GetName() != "" {
		rec.Nickname = player.Name
	}
	if int64(player.GetBannedUntil()) > now.UnixMilli() {
		return finish(RedeemBanned, "")
	}
	if b.Eligibility == entbatch.EligibilityCondition && b.RegisteredWithinDays > 0 {
		since := now.AddDate(0, 0, -b.RegisteredWithinDays).UnixMilli()
		if created := int64(player.GetCreatedAt()); created <= 0 || created < since {
			return finish(RedeemNotEligible, "")
		}
	}

	// 占用次数与一次性码，再发放；发放失败时释放。
	ok, err := uc.codes.ReserveUse(ctx, b.ID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if !ok {
		return finish(RedeemExhausted, "")
	}
	release := func() {
		if err := uc.codes.ReleaseUse(ctx, b.ID); err != nil {
			uc.log.Errorf("release gift batch %s: %v", b.ID, err)
		}
		if gc != nil {
			if err := uc.codes.ReleaseCode(ctx, gc.ID); err != nil {
				uc.log.Errorf("release gift code %s: %v", gc.ID, err)
			}
		}
	}
	if gc != nil {
		ok, err := uc.codes.ClaimCode(ctx, gc.ID, l.PlayerID, rec.Nickname, now)
		if err != nil || !ok {
			if err := uc.codes.ReleaseUse(ctx, b.ID); err != nil {
				uc.log.Errorf("release gift batch %s: %v", b.ID, err)
			}
			if err != nil {
				return nil, ErrSystem(err)
			}
			return finish(RedeemCodeUsed, "")
		}
	}
	gift := gakumas.GiftSend{
		Rewards:    make([]gakumas.Reward, 0, len(b.Rewards)),
		Message:    b.Message,
		SourceKind: "code",
		SourceID:   b.ID,
	}
	for _, r := range b.Rewards {
		gift.Rewards = append(gift.Rewards, gakumas.Reward{
			ResourceType: r.ResourceType,
			ResourceID:   r.ResourceID,
			Quantity:     r.Quantity,
		})
	}
	params, _ := json.Marshal(gift)
	summary := map[string]string{"batchId": b.ID, "batch": b.Name, "code": code, "rewards": itoa(len(b.Rewards))}
	e := newPlayerRecord(s, redeemGift, actor, l.PlayerID, rec.Nickname, params, "", summary, "code:"+b.ID)
	e.Status = "running"
	if e, err = uc.records.CreateExecution(ctx, e); err != nil {
		release()
		return nil, ErrSystem(err)
	}
	e, _ = uc.runPlayerRecord(ctx, e, rec.Nickname, func(ctx context.Context, pid string, opt mgp.OperationOptions) (proto.Message, error) {
		return g.SendGift(ctx, pid, gift, opt)
	})
	rec.ExecutionID = e.ID
	if e.Status != "success" {
		release()
		msg := e.ErrorMessage
		if e.ErrorCode != "" {
			msg = e.ErrorCode + ": " + msg
		}
		return finish(RedeemFailed, msg)
	}
	for _, r := range b.Rewards {
		out.Rewards = append(out.Rewards, &v1.GameReward{ResourceType: r.ResourceType, ResourceId: r.ResourceID, Quantity: r.Quantity})
	}
	return finish(RedeemSuccess, "")
}
