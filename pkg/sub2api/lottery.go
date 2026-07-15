package sub2api

import (
	"context"
	"strconv"
	"time"

	"momoko/pkg/common"
)

// LotteryLocation 抽奖统一时区（UTC+8）。所有日界、结算/开奖时间都以此为准。
var LotteryLocation = mustLoadLottery()

func mustLoadLottery() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		return time.FixedZone("CST", 8*3600)
	}
	return loc
}

// lotteryClock 可测时钟（单测可替换）；生产恒为 time.Now。
var lotteryClock = func() time.Time { return time.Now() }

const (
	// LotterySettleHour 结算时刻（当天 00:00）：生成「报名中」轮次。
	LotterySettleHour = 0
	// LotteryDrawHour 开奖时刻（当天 12:00）：关闭报名并抽奖。
	LotteryDrawHour = 12
	// LotteryPrizeCap 每位中奖者金额上限：超过则中奖人数翻倍。
	LotteryPrizeCap = 10.0
)

// LotteryRoundStatus 轮次状态。
// 只有 drawn 才算历史；registering 是当日进行中的报名窗口。
type LotteryRoundStatus string

const (
	LotteryRoundRegistering LotteryRoundStatus = "registering"
	LotteryRoundDrawn       LotteryRoundStatus = "drawn"
)

// LotteryPayoutStatus 派奖状态。
type LotteryPayoutStatus string

const (
	PayoutNone    LotteryPayoutStatus = "none"
	PayoutPending LotteryPayoutStatus = "pending"
	PayoutPaid    LotteryPayoutStatus = "paid"
	PayoutFailed  LotteryPayoutStatus = "failed"
)

// LotterySettings 每日抽奖活动配置（存 KV）。
type LotterySettings struct {
	Enabled     bool
	PoolRatio   float64 // 奖池比例（0~1，默认 0.05）
	Threshold   float64 // 报名门槛金额（默认 2）
	BaseWinners int     // 基准中奖人数（默认 10）
	MaxWinners  int     // 最大中奖人数（0=无限）
	AutoPayout  bool    // 是否自动发放
}

// DefaultLotterySettings 返回默认配置。
func DefaultLotterySettings() LotterySettings {
	return LotterySettings{
		Enabled:     false,
		PoolRatio:   0.05,
		Threshold:   2,
		BaseWinners: 10,
		MaxWinners:  0,
		AutoPayout:  true,
	}
}

// NormalizeLotterySettings 约束取值范围。
func NormalizeLotterySettings(s *LotterySettings) {
	if s == nil {
		return
	}
	if s.PoolRatio < 0 {
		s.PoolRatio = 0
	}
	if s.PoolRatio > 1 {
		s.PoolRatio = 1
	}
	if s.Threshold < 0 {
		s.Threshold = 0
	}
	if s.BaseWinners < 1 {
		s.BaseWinners = 1
	}
	if s.MaxWinners < 0 {
		s.MaxWinners = 0
	}
}

// LoadLotterySettings 从 KV 读取抽奖配置。
func LoadLotterySettings(ctx context.Context, store ConfigStore) (LotterySettings, error) {
	s := LotterySettings{}
	var err error
	if s.Enabled, err = getBool(ctx, store, common.ConfigSub2APILotteryEnabled); err != nil {
		return s, err
	}
	if s.PoolRatio, err = getFloat(ctx, store, common.ConfigSub2APILotteryPoolRatio); err != nil {
		return s, err
	}
	if s.Threshold, err = getFloat(ctx, store, common.ConfigSub2APILotteryThreshold); err != nil {
		return s, err
	}
	base, err := getInt32(ctx, store, common.ConfigSub2APILotteryBaseWinners)
	if err != nil {
		return s, err
	}
	s.BaseWinners = int(base)
	maxW, err := getInt32(ctx, store, common.ConfigSub2APILotteryMaxWinners)
	if err != nil {
		return s, err
	}
	s.MaxWinners = int(maxW)
	if s.AutoPayout, err = getBool(ctx, store, common.ConfigSub2APILotteryAutoPayout); err != nil {
		return s, err
	}
	NormalizeLotterySettings(&s)
	return s, nil
}

// SaveLotterySettings 持久化抽奖配置。
func SaveLotterySettings(ctx context.Context, store ConfigStore, s LotterySettings) error {
	NormalizeLotterySettings(&s)
	return store.BatchUpdate(ctx, map[common.ConfigKey]string{
		common.ConfigSub2APILotteryEnabled:     strconv.FormatBool(s.Enabled),
		common.ConfigSub2APILotteryPoolRatio:   strconv.FormatFloat(s.PoolRatio, 'f', -1, 64),
		common.ConfigSub2APILotteryThreshold:   strconv.FormatFloat(s.Threshold, 'f', -1, 64),
		common.ConfigSub2APILotteryBaseWinners: strconv.Itoa(s.BaseWinners),
		common.ConfigSub2APILotteryMaxWinners:  strconv.Itoa(s.MaxWinners),
		common.ConfigSub2APILotteryAutoPayout:  strconv.FormatBool(s.AutoPayout),
	})
}

// LotteryRound 一期抽奖轮次。
// id = 报名/开奖当天（day D）；奖池来自 D-1 扣费，D 00:00 结算开启报名，D 12:00 开奖。
type LotteryRound struct {
	ID              string
	SourceDate      string
	SettleTime      time.Time
	DrawTime        time.Time
	Status          LotteryRoundStatus
	PoolRatio       float64
	Threshold       float64
	BaseWinners     int
	MaxWinners      int
	GroupSpendTotal float64
	CarryIn         float64
	PoolAmount      float64
	EligibleCount   int
	RegisteredCount int
	WinnerCount     int
	PerWinnerAmount float64
	CarryOut        float64
	AutoPayout      bool
	Distributed     bool
	CreateTime      time.Time
	UpdateTime      time.Time
}

// LotteryParticipant 一位用户在某一轮的报名/中奖记录。
type LotteryParticipant struct {
	ID                   int
	RoundID              string
	Sub2APIUserID        int64
	UserName             string
	SpendSnapshot        float64
	RegisteredTime       time.Time
	IsWinner             bool
	PrizeAmount          float64
	PayoutStatus         LotteryPayoutStatus
	PayoutIdempotencyKey string
	PayoutError          string
}

// LotteryStore 抽奖数据访问接口。
// 扣费合计必须在 DB 层 Aggregate/GroupBy 完成，禁止拉全量再内存累加。
type LotteryStore interface {
	SumPublicCostInRange(ctx context.Context, start, end time.Time) (float64, error)
	CountEligibleUsersInRange(ctx context.Context, start, end time.Time, threshold float64) (int, error)
	UserSpendInRange(ctx context.Context, userID int64, start, end time.Time) (float64, error)

	GetLotteryRound(ctx context.Context, id string) (*LotteryRound, error)
	// LatestLotteryRound 按 id 降序取最近一轮（id=YYYY-MM-DD 可字典序比较）。
	LatestLotteryRound(ctx context.Context) (*LotteryRound, error)
	// ListRegisteringRounds 全部仍处于报名中的轮次（供 Tick 补开奖）。
	ListRegisteringRounds(ctx context.Context) ([]*LotteryRound, error)
	SaveLotteryRound(ctx context.Context, r *LotteryRound) error
	// ListDrawnLotteryRounds 仅已开奖轮次（历史列表）；offset/limit 分页。
	ListDrawnLotteryRounds(ctx context.Context, offset, limit int) ([]*LotteryRound, int, error)

	GetParticipant(ctx context.Context, roundID string, userID int64) (*LotteryParticipant, error)
	CreateParticipant(ctx context.Context, p *LotteryParticipant) error
	CountParticipants(ctx context.Context, roundID string) (int, error)
	ListParticipants(ctx context.Context, roundID string) ([]*LotteryParticipant, error)
	ListWinners(ctx context.Context, roundID string) ([]*LotteryParticipant, error)
	UpdateParticipant(ctx context.Context, p *LotteryParticipant) error
	ParticipationsByUser(ctx context.Context, userID int64, roundIDs []string) (map[string]*LotteryParticipant, error)
}

// ---------- 纯逻辑 ----------

// ComputeWinnerCount 依据奖池、基准/上限人数与报名人数计算实际中奖人数。
func ComputeWinnerCount(pool float64, base, maxWinners, registered int) int {
	if registered <= 0 || pool <= 0 {
		return 0
	}
	if base < 1 {
		base = 1
	}
	winners := base
	for pool/float64(winners) > LotteryPrizeCap {
		if winners >= registered {
			break
		}
		next := winners * 2
		if maxWinners > 0 && next > maxWinners {
			break
		}
		winners = next
	}
	if maxWinners > 0 && winners > maxWinners {
		winners = maxWinners
	}
	if winners > registered {
		winners = registered
	}
	return winners
}

// LotteryNow 返回抽奖时区当前时间。
func LotteryNow() time.Time {
	return lotteryClock().In(LotteryLocation)
}

// LotteryDateStr 返回抽奖时区下 t 所在日期字符串。
func LotteryDateStr(t time.Time) string {
	return t.In(LotteryLocation).Format("2006-01-02")
}

// LotteryDayStart 返回抽奖时区下某日期的 00:00。
func LotteryDayStart(dateStr string) (time.Time, error) {
	return time.ParseInLocation("2006-01-02", dateStr, LotteryLocation)
}

// LotteryDayBounds 返回某日期的 [00:00, 次日 00:00)。
func LotteryDayBounds(dateStr string) (time.Time, time.Time, error) {
	start, err := LotteryDayStart(dateStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}
	return start, start.AddDate(0, 0, 1), nil
}

// PreviousDateStr 返回 dateStr 的前一天。
func PreviousDateStr(dateStr string) string {
	t, err := LotteryDayStart(dateStr)
	if err != nil {
		return dateStr
	}
	return t.AddDate(0, 0, -1).Format("2006-01-02")
}

// RoundSettleTime 某轮名义结算时刻（day D 00:00）。
func RoundSettleTime(roundDate string) (time.Time, error) {
	return LotteryDayStart(roundDate)
}

// RoundDrawTime 某轮名义开奖时刻（day D 12:00）。
func RoundDrawTime(roundDate string) (time.Time, error) {
	start, err := LotteryDayStart(roundDate)
	if err != nil {
		return time.Time{}, err
	}
	return start.Add(time.Duration(LotteryDrawHour) * time.Hour), nil
}

// InRegisterWindow 是否处于 day D 报名窗 [D 00:00, D 12:00)。
// 仅此窗口内允许 Settle 新建当日轮次。
func InRegisterWindow(now time.Time) bool {
	n := now.In(LotteryLocation)
	date := LotteryDateStr(n)
	draw, err := RoundDrawTime(date)
	if err != nil {
		return false
	}
	dayStart, err := LotteryDayStart(date)
	if err != nil {
		return false
	}
	return !n.Before(dayStart) && n.Before(draw)
}

// NextBoundary 返回 now 之后下一个 hour:00（抽奖时区）时刻。
func NextBoundary(now time.Time, hour int) time.Time {
	n := now.In(LotteryLocation)
	next := time.Date(n.Year(), n.Month(), n.Day(), hour, 0, 0, 0, LotteryLocation)
	if !next.After(n) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

// DatesBetweenExclusive 返回 (after, before) 之间的日期列表（不含两端）。
// after/before 为 YYYY-MM-DD；若 after 为空则返回空（避免无限回溯）。
func DatesBetweenExclusive(after, before string) []string {
	if after == "" || before == "" || after >= before {
		return nil
	}
	cur, err := LotteryDayStart(after)
	if err != nil {
		return nil
	}
	end, err := LotteryDayStart(before)
	if err != nil {
		return nil
	}
	var out []string
	for d := cur.AddDate(0, 0, 1); d.Before(end); d = d.AddDate(0, 0, 1) {
		out = append(out, d.Format("2006-01-02"))
	}
	return out
}
