package sub2api

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"time"
)

// 抽奖业务错误（由 biz 翻译为对外响应）。
var (
	ErrLotteryDisabled      = errors.New("抽奖活动未开启")
	ErrLotteryClosed        = errors.New("报名已截止")
	ErrLotteryDrawNotDue    = errors.New("未到开奖时间")
	ErrLotteryNotEligible   = errors.New("扣费未达到报名门槛")
	ErrLotteryTokenInvalid  = errors.New("用户令牌无效")
	ErrLotteryRoundNotFound = errors.New("轮次不存在")
	// ErrLotteryOutsideWindow：不在当日报名窗 [00:00,12:00)，禁止新建轮次。
	ErrLotteryOutsideWindow = errors.New("当前不在报名窗口，无法结算新轮次")
)

const (
	lotteryAccumTTL          = 45 * time.Second
	lotteryTickTimeout       = 2 * time.Minute
	lotterySettleTimeout     = 2 * time.Minute
	lotteryDrawTimeout       = 5 * time.Minute
	lotteryDistributeTimeout = 5 * time.Minute
)

// LotteryService 每日抽奖编排。
//
// 生命周期（Asia/Shanghai）：
//
//	D-1 全天      展示分组扣费 → 次日奖池来源
//	D 00:00–12:00  定时器 Tick 满足「已过 0 点 + 今日未结算」→ Settle(D)，开放报名
//	D 12:00+       定时器 Tick 满足「已过 12 点 + 轮次仍报名中」→ Draw(D)，进入历史
//
// 铁律：
//   - 进程启动 / EnsureSingleton / 重启：不做任何 Settle/Draw（定时任务也不立即首跑）。
//   - 只有定时器真正触发 Tick 时才评估墙钟条件。
//   - 结算三条件同时满足才 Settle：① 定时器触发 ② 处于报名窗 [00:00,12:00) ③ 今日轮次不存在。
//   - 开奖三条件同时满足才 Draw：① 定时器触发 ② now≥12:00 ③ 存在 registering 且未开奖。
//   - 过午绝不补建今日轮次（杜绝 15:52 重启造出 0 人历史）；漏掉的源日奖池在下次合法 Settle 并入 carry。
//   - 历史列表只含 drawn。
type LotteryService struct {
	store   LotteryStore
	config  ConfigStore
	manager *Manager

	mu sync.Mutex // 串行化 Settle/Draw

	accumMu   sync.Mutex
	accumVal  accumSnapshot
	accumTime time.Time
}

type accumSnapshot struct {
	date     string
	spend    float64
	eligible int
}

func NewLotteryService(store LotteryStore, config ConfigStore, manager *Manager) *LotteryService {
	return &LotteryService{store: store, config: config, manager: manager}
}

// Tick 仅由定时任务 Interval 触发。重启不会调用本方法（任务管理器不首跑）。
//
//	Settle：报名窗内 + 今日无轮次
//	Draw：已存在的 registering 且 now ≥ 名义 12:00
func (s *LotteryService) Tick() {
	ctx, cancel := context.WithTimeout(context.Background(), lotteryTickTimeout)
	defer cancel()
	if !s.enabled(ctx) {
		return
	}
	now := LotteryNow()
	today := LotteryDateStr(now)

	// 开奖：只动库里已有的 registering
	s.drawDue(ctx, now)

	// 结算：三条件 —— 定时器已触发(本函数) + 报名窗内 + 今日未执行
	if !InRegisterWindow(now) {
		return
	}
	round, err := s.store.GetLotteryRound(ctx, today)
	if err != nil || round != nil {
		return
	}
	_, _ = s.Settle(today)
}

func (s *LotteryService) drawDue(ctx context.Context, now time.Time) {
	rounds, err := s.store.ListRegisteringRounds(ctx)
	if err != nil {
		return
	}
	for _, r := range rounds {
		if r == nil {
			continue
		}
		drawAt, err := RoundDrawTime(r.ID)
		if err != nil || now.Before(drawAt) {
			continue
		}
		_, _ = s.Draw(r.ID)
	}
}

func (s *LotteryService) enabled(ctx context.Context) bool {
	settings, err := LoadLotterySettings(ctx, s.config)
	return err == nil && settings.Enabled
}

// ---------- 结算 ----------

// Settle 创建 day D 的报名中轮次（源日 D-1 扣费 + carry）。
// 幂等：已存在直接返回。
// 已过该日 12:00：拒绝新建（防补结算后被 Tick 立刻开成 0 人历史）。
func (s *LotteryService) Settle(roundDate string) (*LotteryRound, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lotterySettleTimeout)
	defer cancel()
	s.mu.Lock()
	defer s.mu.Unlock()

	if existing, err := s.store.GetLotteryRound(ctx, roundDate); err != nil {
		return nil, err
	} else if existing != nil {
		return existing, nil
	}

	// 任意日期：若名义开奖点已过，禁止再创建 registering 轮次
	drawAt, err := RoundDrawTime(roundDate)
	if err != nil {
		return nil, err
	}
	if !LotteryNow().Before(drawAt) {
		return nil, ErrLotteryOutsideWindow
	}

	settings, err := LoadLotterySettings(ctx, s.config)
	if err != nil {
		return nil, err
	}

	sourceDate := PreviousDateStr(roundDate)
	start, end, err := LotteryDayBounds(sourceDate)
	if err != nil {
		return nil, err
	}
	spendTotal, err := s.store.SumPublicCostInRange(ctx, start, end)
	if err != nil {
		return nil, err
	}
	eligible, err := s.store.CountEligibleUsersInRange(ctx, start, end, settings.Threshold)
	if err != nil {
		return nil, err
	}
	carryIn, err := s.computeCarryIn(ctx, settings, roundDate)
	if err != nil {
		return nil, err
	}

	settleTime, _ := RoundSettleTime(roundDate)
	drawTime, _ := RoundDrawTime(roundDate)
	round := &LotteryRound{
		ID:              roundDate,
		SourceDate:      sourceDate,
		SettleTime:      settleTime,
		DrawTime:        drawTime,
		Status:          LotteryRoundRegistering,
		PoolRatio:       settings.PoolRatio,
		Threshold:       settings.Threshold,
		BaseWinners:     settings.BaseWinners,
		MaxWinners:      settings.MaxWinners,
		GroupSpendTotal: spendTotal,
		CarryIn:         carryIn,
		PoolAmount:      settings.PoolRatio*spendTotal + carryIn,
		EligibleCount:   eligible,
		AutoPayout:      settings.AutoPayout,
	}
	if err = s.store.SaveLotteryRound(ctx, round); err != nil {
		return nil, err
	}
	return round, nil
}

// computeCarryIn = 最近一轮 CarryOut + 中间漏建轮次对应源日的 ratio×spend。
// 漏日比例用上一轮快照 PoolRatio（不用当前配置），避免改配置后改写历史金额。
// 首次启用（无历史轮次）不回溯，carry=0。
func (s *LotteryService) computeCarryIn(ctx context.Context, settings LotterySettings, roundDate string) (float64, error) {
	latest, err := s.store.LatestLotteryRound(ctx)
	if err != nil {
		return 0, err
	}
	if latest == nil {
		return 0, nil
	}
	carry := latest.CarryOut
	// 漏日用上一轮结算时的比例；若快照异常则回退当前配置
	gapRatio := latest.PoolRatio
	if gapRatio < 0 || gapRatio > 1 {
		gapRatio = settings.PoolRatio
	}
	for _, m := range DatesBetweenExclusive(latest.ID, roundDate) {
		if existing, err := s.store.GetLotteryRound(ctx, m); err == nil && existing != nil {
			continue
		}
		src := PreviousDateStr(m)
		st, en, err := LotteryDayBounds(src)
		if err != nil {
			return 0, err
		}
		spend, err := s.store.SumPublicCostInRange(ctx, st, en)
		if err != nil {
			return 0, err
		}
		carry += gapRatio * spend
	}
	return carry, nil
}

// ---------- 开奖 ----------

// Draw 开奖。幂等。未到名义 12:00 拒绝。
func (s *LotteryService) Draw(roundDate string) (*LotteryRound, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lotteryDrawTimeout)
	defer cancel()
	s.mu.Lock()
	defer s.mu.Unlock()

	round, err := s.store.GetLotteryRound(ctx, roundDate)
	if err != nil {
		return nil, err
	}
	if round == nil {
		return nil, ErrLotteryRoundNotFound
	}
	if round.Status == LotteryRoundDrawn {
		return round, nil
	}

	drawAt, err := RoundDrawTime(roundDate)
	if err != nil {
		return nil, err
	}
	if LotteryNow().Before(drawAt) {
		return nil, ErrLotteryDrawNotDue
	}
	round.DrawTime = drawAt

	participants, err := s.store.ListParticipants(ctx, roundDate)
	if err != nil {
		return nil, err
	}
	round.RegisteredCount = len(participants)

	if len(participants) == 0 {
		round.WinnerCount = 0
		round.PerWinnerAmount = 0
		round.CarryOut = round.PoolAmount
		round.Distributed = true
		round.Status = LotteryRoundDrawn
		if err = s.store.SaveLotteryRound(ctx, round); err != nil {
			return nil, err
		}
		return round, nil
	}

	// 零消费报名者在门槛配置为 0 时可能存在，但其消费占比为 0，不参与派奖。
	candidates := make([]*LotteryParticipant, 0, len(participants))
	for _, p := range participants {
		if p != nil && validLotterySpend(p.SpendSnapshot) {
			candidates = append(candidates, p)
		}
	}
	winnerCount := ComputeWinnerCount(round.PoolAmount, round.BaseWinners, round.MaxWinners, len(candidates))
	if err = secureShuffle(candidates); err != nil {
		return nil, fmt.Errorf("开奖随机失败：%w", err)
	}
	winners := candidates[:winnerCount]
	spends := make([]float64, len(winners))
	for i, p := range winners {
		spends[i] = p.SpendSnapshot
	}
	prizes, allocated := ComputeProportionalPrizes(round.PoolAmount, spends)
	if !allocated {
		round.WinnerCount = 0
		round.PerWinnerAmount = 0
		round.CarryOut = round.PoolAmount
		round.Distributed = true
		round.Status = LotteryRoundDrawn
		if err = s.store.SaveLotteryRound(ctx, round); err != nil {
			return nil, err
		}
		return round, nil
	}
	for i, p := range winners {
		p.IsWinner = true
		p.PrizeAmount = prizes[i]
		p.PayoutStatus = PayoutPending
		if err = s.store.UpdateParticipant(ctx, p); err != nil {
			return nil, err
		}
	}

	round.WinnerCount = winnerCount
	round.PerWinnerAmount = round.PoolAmount / float64(winnerCount)
	round.CarryOut = 0
	round.Status = LotteryRoundDrawn
	round.Distributed = false
	if err = s.store.SaveLotteryRound(ctx, round); err != nil {
		return nil, err
	}
	if round.AutoPayout {
		s.payoutWinners(ctx, round, winners)
		round.Distributed = allPaid(winners)
		_ = s.store.SaveLotteryRound(ctx, round)
	}
	return round, nil
}

// DistributeRound 手动补发。
func (s *LotteryService) DistributeRound(roundID string) (*LotteryRound, error) {
	ctx, cancel := context.WithTimeout(context.Background(), lotteryDistributeTimeout)
	defer cancel()
	round, err := s.store.GetLotteryRound(ctx, roundID)
	if err != nil {
		return nil, err
	}
	if round == nil {
		return nil, ErrLotteryRoundNotFound
	}
	winners, err := s.store.ListWinners(ctx, roundID)
	if err != nil {
		return nil, err
	}
	pending := make([]*LotteryParticipant, 0, len(winners))
	for _, w := range winners {
		if w.PayoutStatus != PayoutPaid {
			pending = append(pending, w)
		}
	}
	s.payoutWinners(ctx, round, pending)
	winners, _ = s.store.ListWinners(ctx, roundID)
	round.Distributed = allPaid(winners)
	if err = s.store.SaveLotteryRound(ctx, round); err != nil {
		return nil, err
	}
	return round, nil
}

func (s *LotteryService) payoutWinners(ctx context.Context, round *LotteryRound, winners []*LotteryParticipant) {
	cfg, err := s.clientConfig(ctx)
	if err != nil || !cfg.Configured() {
		for _, w := range winners {
			w.PayoutStatus = PayoutFailed
			w.PayoutError = "Sub2API 连接未配置"
			_ = s.store.UpdateParticipant(ctx, w)
		}
		return
	}
	notes := fmt.Sprintf("每日抽奖中奖（%s）", round.ID)
	for _, w := range winners {
		idem := fmt.Sprintf("lottery-%s-%d", round.ID, w.Sub2APIUserID)
		err := s.manager.AddUserBalance(cfg, w.Sub2APIUserID, w.PrizeAmount, notes, idem)
		w.PayoutIdempotencyKey = idem
		if err != nil {
			w.PayoutStatus = PayoutFailed
			w.PayoutError = err.Error()
		} else {
			w.PayoutStatus = PayoutPaid
			w.PayoutError = ""
		}
		_ = s.store.UpdateParticipant(ctx, w)
	}
}

// ---------- 报名 ----------

func (s *LotteryService) Register(ctx context.Context, jwt string) (*LotteryUserStatus, error) {
	settings, err := LoadLotterySettings(ctx, s.config)
	if err != nil {
		return nil, err
	}
	if !settings.Enabled {
		return nil, ErrLotteryDisabled
	}
	userID, userName, err := s.validate(ctx, jwt)
	if err != nil {
		return nil, err
	}

	now := LotteryNow()
	if !InRegisterWindow(now) {
		return nil, ErrLotteryClosed
	}
	round, err := s.store.GetLotteryRound(ctx, LotteryDateStr(now))
	if err != nil {
		return nil, err
	}
	if round == nil || round.Status != LotteryRoundRegistering {
		return nil, ErrLotteryClosed
	}

	existing, err := s.store.GetParticipant(ctx, round.ID, userID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		spend, err := s.userSourceSpend(ctx, round, userID)
		if err != nil {
			return nil, err
		}
		if spend < round.Threshold {
			return nil, ErrLotteryNotEligible
		}
		if err = s.store.CreateParticipant(ctx, &LotteryParticipant{
			RoundID:        round.ID,
			Sub2APIUserID:  userID,
			UserName:       userName,
			SpendSnapshot:  spend,
			RegisteredTime: now,
			PayoutStatus:   PayoutNone,
		}); err != nil {
			return nil, err
		}
	}
	return s.userStatus(ctx, settings, userID, userName)
}

// ---------- 管理端读模型 ----------

func (s *LotteryService) Overview(ctx context.Context) (*LotteryOverview, error) {
	settings, err := LoadLotterySettings(ctx, s.config)
	if err != nil {
		return nil, err
	}
	now := LotteryNow()
	accum, err := s.accumulating(ctx, settings)
	if err != nil {
		return nil, err
	}
	current, err := s.currentOpenRound(ctx, now)
	if err != nil {
		return nil, err
	}
	return &LotteryOverview{
		Settings:       settings,
		AccumDate:      accum.date,
		AccumSpend:     accum.spend,
		AccumPool:      settings.PoolRatio * accum.spend,
		AccumEligible:  accum.eligible,
		NextSettleTime: NextBoundary(now, LotterySettleHour),
		NextDrawTime:   NextBoundary(now, LotteryDrawHour),
		Current:        current,
	}, nil
}

// History 仅 drawn。返回页内列表 + 总数（避免调用方再查一次）。
func (s *LotteryService) History(ctx context.Context, page, pageSize int) ([]*LotteryRound, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	return s.store.ListDrawnLotteryRounds(ctx, (page-1)*pageSize, pageSize)
}

func (s *LotteryService) RoundWinners(ctx context.Context, roundID string) (*LotteryRound, []*LotteryParticipant, error) {
	round, err := s.store.GetLotteryRound(ctx, roundID)
	if err != nil {
		return nil, nil, err
	}
	if round == nil {
		return nil, nil, ErrLotteryRoundNotFound
	}
	winners, err := s.store.ListWinners(ctx, roundID)
	if err != nil {
		return nil, nil, err
	}
	return round, winners, nil
}

// RoundRegistrants 某轮报名者名单（仅本地快照：用户id/用户名/消费额）。不做任何外部调用。
func (s *LotteryService) RoundRegistrants(ctx context.Context, roundID string) ([]*LotteryParticipant, error) {
	round, err := s.store.GetLotteryRound(ctx, roundID)
	if err != nil {
		return nil, err
	}
	if round == nil {
		return nil, ErrLotteryRoundNotFound
	}
	return s.store.ListParticipants(ctx, roundID)
}

// UserInfo 按 user_id 实时拉取单个 Sub2API 用户详情（点击报名者时用）。
func (s *LotteryService) UserInfo(ctx context.Context, userID int64) (*Sub2APIUserInfo, error) {
	if userID <= 0 {
		return nil, fmt.Errorf("用户 ID 不能为空")
	}
	cfg, err := s.clientConfig(ctx)
	if err != nil || !cfg.Configured() || s.manager == nil {
		return nil, fmt.Errorf("Sub2API 连接未配置")
	}
	return s.manager.GetUserByID(cfg, userID)
}

func (s *LotteryService) UpdateSettings(ctx context.Context, next LotterySettings) (LotterySettings, error) {
	if err := SaveLotterySettings(ctx, s.config, next); err != nil {
		return LotterySettings{}, err
	}
	return LoadLotterySettings(ctx, s.config)
}

// ---------- 用户端读模型 ----------

func (s *LotteryService) UserStatus(ctx context.Context, jwt string) (*LotteryUserStatus, error) {
	settings, err := LoadLotterySettings(ctx, s.config)
	if err != nil {
		return nil, err
	}
	userID, userName, _ := s.validate(ctx, jwt)
	return s.userStatus(ctx, settings, userID, userName)
}

// UserHistory 用户端历史：已开奖轮次 + total + 本人参与 + 各轮中奖名单（邮箱已打码）。
func (s *LotteryService) UserHistory(ctx context.Context, jwt string, page, pageSize int) (
	rounds []*LotteryRound,
	total int,
	mine map[string]*LotteryParticipant,
	winnersByRound map[string][]*LotteryParticipant,
	err error,
) {
	if page < 1 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 10
	}
	rounds, total, err = s.store.ListDrawnLotteryRounds(ctx, (page-1)*pageSize, pageSize)
	if err != nil {
		return nil, 0, nil, nil, err
	}
	userID, _, _ := s.validate(ctx, jwt)
	mine = map[string]*LotteryParticipant{}
	winnersByRound = map[string][]*LotteryParticipant{}
	if len(rounds) == 0 {
		return rounds, total, mine, winnersByRound, nil
	}
	ids := make([]string, 0, len(rounds))
	for _, r := range rounds {
		ids = append(ids, r.ID)
	}
	if userID > 0 {
		if mine, err = s.store.ParticipationsByUser(ctx, userID, ids); err != nil {
			return nil, 0, nil, nil, err
		}
	}
	for _, id := range ids {
		ws, werr := s.store.ListWinners(ctx, id)
		if werr != nil {
			return nil, 0, nil, nil, werr
		}
		masked := make([]*LotteryParticipant, 0, len(ws))
		for _, w := range ws {
			if w == nil {
				continue
			}
			cp := *w
			cp.UserName = MaskPublicDisplayName(w.UserName)
			cp.PayoutError = ""
			cp.PayoutIdempotencyKey = ""
			masked = append(masked, &cp)
		}
		winnersByRound[id] = masked
	}
	return rounds, total, mine, winnersByRound, nil
}

// ---------- 内部 ----------

func (s *LotteryService) userStatus(ctx context.Context, settings LotterySettings, userID int64, userName string) (*LotteryUserStatus, error) {
	now := LotteryNow()
	accum, err := s.accumulating(ctx, settings)
	if err != nil {
		return nil, err
	}
	st := &LotteryUserStatus{
		Enabled:        settings.Enabled,
		Authenticated:  userID > 0,
		UserID:         userID,
		UserName:       userName,
		AccumPool:      settings.PoolRatio * accum.spend,
		NextSettleTime: NextBoundary(now, LotterySettleHour),
		NextDrawTime:   NextBoundary(now, LotteryDrawHour),
		Threshold:      settings.Threshold,
	}
	// 当期累计（今日）对应下一次结算生成的轮次（次日），作为期号展示。
	st.AccumRoundDate = LotteryDateStr(st.NextSettleTime)
	// 本人当期累计扣费 = 今日 [00:00, now)，与展示分组奖池同口径；达标即锁定次日轮次资格。
	if userID > 0 {
		if dayStart, e := LotteryDayStart(LotteryDateStr(now)); e == nil {
			spend, _ := s.store.UserSpendInRange(ctx, userID, dayStart, now)
			st.AccumUserSpend = spend
			st.AccumEligible = spend >= settings.Threshold
		}
	}
	round, err := s.currentOpenRound(ctx, now)
	if err != nil {
		return nil, err
	}
	st.Current = round
	if round != nil && userID > 0 {
		spend, _ := s.userSourceSpend(ctx, round, userID)
		st.UserSpend = spend
		st.Eligible = spend >= round.Threshold
		if p, _ := s.store.GetParticipant(ctx, round.ID, userID); p != nil {
			st.Registered = true
		}
	}
	return st, nil
}

// currentOpenRound 今日仍开放报名的轮次（registering 且 now < 12:00）；否则 nil。
func (s *LotteryService) currentOpenRound(ctx context.Context, now time.Time) (*LotteryRound, error) {
	round, err := s.store.GetLotteryRound(ctx, LotteryDateStr(now))
	if err != nil || round == nil || round.Status != LotteryRoundRegistering {
		return nil, err
	}
	drawAt, err := RoundDrawTime(round.ID)
	if err != nil || !now.Before(drawAt) {
		return nil, err
	}
	if cnt, err := s.store.CountParticipants(ctx, round.ID); err == nil {
		round.RegisteredCount = cnt
	}
	return round, nil
}

func (s *LotteryService) accumulating(ctx context.Context, settings LotterySettings) (accumSnapshot, error) {
	now := LotteryNow()
	today := LotteryDateStr(now)

	s.accumMu.Lock()
	if s.accumVal.date == today && time.Since(s.accumTime) < lotteryAccumTTL {
		v := s.accumVal
		s.accumMu.Unlock()
		return v, nil
	}
	s.accumMu.Unlock()

	start, _ := LotteryDayStart(today)
	spend, err := s.store.SumPublicCostInRange(ctx, start, now)
	if err != nil {
		return accumSnapshot{}, err
	}
	eligible, err := s.store.CountEligibleUsersInRange(ctx, start, now, settings.Threshold)
	if err != nil {
		return accumSnapshot{}, err
	}
	snap := accumSnapshot{date: today, spend: spend, eligible: eligible}
	s.accumMu.Lock()
	s.accumVal = snap
	s.accumTime = time.Now()
	s.accumMu.Unlock()
	return snap, nil
}

func (s *LotteryService) userSourceSpend(ctx context.Context, round *LotteryRound, userID int64) (float64, error) {
	start, end, err := LotteryDayBounds(round.SourceDate)
	if err != nil {
		return 0, err
	}
	return s.store.UserSpendInRange(ctx, userID, start, end)
}

func (s *LotteryService) validate(ctx context.Context, jwt string) (userID int64, userName string, err error) {
	if jwt == "" {
		return 0, "", ErrLotteryTokenInvalid
	}
	cfg, err := s.clientConfig(ctx)
	if err != nil || !cfg.Configured() {
		return 0, "", ErrLotteryTokenInvalid
	}
	uid, uname, err := s.manager.ValidateUserToken(cfg, jwt)
	if err != nil {
		return 0, "", ErrLotteryTokenInvalid
	}
	return uid, uname, nil
}

func (s *LotteryService) clientConfig(ctx context.Context) (ClientConfig, error) {
	cfg, err := LoadConfig(ctx, s.config)
	if err != nil {
		return ClientConfig{}, err
	}
	return ClientConfigFromConfig(cfg), nil
}

// ---------- 结果结构 ----------

type LotteryOverview struct {
	Settings       LotterySettings
	AccumDate      string
	AccumSpend     float64
	AccumPool      float64
	AccumEligible  int
	NextSettleTime time.Time
	NextDrawTime   time.Time
	Current        *LotteryRound
}

// LotteryRegistrant 报名者本地快照 + 实时 Sub2API 用户信息（拉取失败 User=nil，原因见 UserError）。
type LotteryRegistrant struct {
	Participant *LotteryParticipant
	User        *Sub2APIUserInfo
	UserError   string
}

type LotteryUserStatus struct {
	Enabled        bool
	Authenticated  bool
	UserID         int64
	UserName       string
	AccumPool      float64
	NextSettleTime time.Time
	NextDrawTime   time.Time
	Current        *LotteryRound
	Eligible       bool
	UserSpend      float64
	Registered     bool
	Threshold      float64 // 参与门槛（统一快照）
	AccumRoundDate string  // 当期累计对应轮次日期（次日）
	AccumUserSpend float64 // 本人当期累计扣费（今日实时）
	AccumEligible  bool    // 本人是否达到当期参与条件
}

// secureShuffle 密码学乱序；rand 失败则中止（禁止半截洗牌导致中奖偏置）。
func secureShuffle[T any](s []T) error {
	for i := len(s) - 1; i > 0; i-- {
		jBig, err := rand.Int(rand.Reader, big.NewInt(int64(i+1)))
		if err != nil {
			return err
		}
		j := int(jBig.Int64())
		s[i], s[j] = s[j], s[i]
	}
	return nil
}

func allPaid(winners []*LotteryParticipant) bool {
	if len(winners) == 0 {
		return true
	}
	for _, w := range winners {
		if w.PayoutStatus != PayoutPaid {
			return false
		}
	}
	return true
}
