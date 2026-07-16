package sub2api

import (
	"context"
	"sync"
	"testing"
	"time"

	"momoko/pkg/common"
)

// ---------- fakes ----------

type memLotteryStore struct {
	mu         sync.Mutex
	round      map[string]*LotteryRound
	parts      map[string][]*LotteryParticipant
	spendByDay map[string]float64
	userSpend  float64 // UserSpendInRange 的返回值（本人区间扣费）
}

func newMemStore() *memLotteryStore {
	return &memLotteryStore{
		round:      map[string]*LotteryRound{},
		parts:      map[string][]*LotteryParticipant{},
		spendByDay: map[string]float64{},
	}
}

func (m *memLotteryStore) SumPublicCostInRange(_ context.Context, start, end time.Time) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 整天查询：start 对齐 00:00 且 end = start+1d
	if end.Equal(start.AddDate(0, 0, 1)) {
		return m.spendByDay[start.In(LotteryLocation).Format("2006-01-02")], nil
	}
	total := 0.0
	for d, v := range m.spendByDay {
		t, err := time.ParseInLocation("2006-01-02", d, LotteryLocation)
		if err != nil {
			continue
		}
		if !t.Before(start) && t.Before(end) {
			total += v
		}
	}
	return total, nil
}

func (m *memLotteryStore) CountEligibleUsersInRange(context.Context, time.Time, time.Time, float64) (int, error) {
	return 0, nil
}
func (m *memLotteryStore) UserSpendInRange(context.Context, int64, time.Time, time.Time) (float64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.userSpend, nil
}
func (m *memLotteryStore) GetLotteryRound(_ context.Context, id string) (*LotteryRound, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	r := m.round[id]
	if r == nil {
		return nil, nil
	}
	cp := *r
	return &cp, nil
}
func (m *memLotteryStore) LatestLotteryRound(context.Context) (*LotteryRound, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	var best *LotteryRound
	for _, r := range m.round {
		if best == nil || r.ID > best.ID {
			cp := *r
			best = &cp
		}
	}
	return best, nil
}
func (m *memLotteryStore) ListRegisteringRounds(context.Context) ([]*LotteryRound, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]*LotteryRound, 0)
	for _, r := range m.round {
		if r.Status == LotteryRoundRegistering {
			cp := *r
			out = append(out, &cp)
		}
	}
	return out, nil
}
func (m *memLotteryStore) SaveLotteryRound(_ context.Context, r *LotteryRound) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	cp := *r
	m.round[r.ID] = &cp
	return nil
}
func (m *memLotteryStore) ListDrawnLotteryRounds(_ context.Context, offset, limit int) ([]*LotteryRound, int, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	all := make([]*LotteryRound, 0)
	for _, r := range m.round {
		if r.Status == LotteryRoundDrawn {
			cp := *r
			all = append(all, &cp)
		}
	}
	total := len(all)
	if offset >= total {
		return nil, total, nil
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total, nil
}
func (m *memLotteryStore) GetParticipant(context.Context, string, int64) (*LotteryParticipant, error) {
	return nil, nil
}
func (m *memLotteryStore) CreateParticipant(context.Context, *LotteryParticipant) error { return nil }
func (m *memLotteryStore) CountParticipants(_ context.Context, roundID string) (int, error) {
	return len(m.parts[roundID]), nil
}
func (m *memLotteryStore) ListParticipants(_ context.Context, roundID string) ([]*LotteryParticipant, error) {
	return append([]*LotteryParticipant(nil), m.parts[roundID]...), nil
}
func (m *memLotteryStore) ListWinners(context.Context, string) ([]*LotteryParticipant, error) {
	return nil, nil
}
func (m *memLotteryStore) UpdateParticipant(context.Context, *LotteryParticipant) error { return nil }
func (m *memLotteryStore) ParticipationsByUser(context.Context, int64, []string) (map[string]*LotteryParticipant, error) {
	return map[string]*LotteryParticipant{}, nil
}

type memConfig struct {
	kv map[common.ConfigKey]string
}

func newEnabledConfig() *memConfig {
	return &memConfig{kv: map[common.ConfigKey]string{
		common.ConfigSub2APILotteryEnabled:     "true",
		common.ConfigSub2APILotteryPoolRatio:   "0.05",
		common.ConfigSub2APILotteryThreshold:   "2",
		common.ConfigSub2APILotteryBaseWinners: "10",
		common.ConfigSub2APILotteryMaxWinners:  "0",
		common.ConfigSub2APILotteryAutoPayout:  "false",
	}}
}

func (c *memConfig) Get(_ context.Context, key common.ConfigKey) (string, error) {
	if v, ok := c.kv[key]; ok {
		return v, nil
	}
	if d, ok := common.ConfigDefault(key); ok {
		return d, nil
	}
	return "", nil
}
func (c *memConfig) BatchUpdate(_ context.Context, configs map[common.ConfigKey]string) error {
	for k, v := range configs {
		c.kv[k] = v
	}
	return nil
}

func setClock(t *testing.T, ts time.Time) {
	t.Helper()
	old := lotteryClock
	lotteryClock = func() time.Time { return ts }
	t.Cleanup(func() { lotteryClock = old })
}

func cst(y int, m time.Month, d, h, min int) time.Time {
	return time.Date(y, m, d, h, min, 0, 0, LotteryLocation)
}

// ---------- tests ----------

// 根因场景：7/15 15:52 重启 → Tick 不得创建当日轮次，更不得开奖进历史。
func TestTick_AfternoonNoSettleNoDraw(t *testing.T) {
	setClock(t, cst(2026, 7, 15, 15, 52))
	store := newMemStore()
	store.spendByDay["2026-07-14"] = 100
	svc := NewLotteryService(store, newEnabledConfig(), nil)

	svc.Tick()
	svc.Tick() // 再跑一次也一样

	if r, _ := store.GetLotteryRound(context.Background(), "2026-07-15"); r != nil {
		t.Fatalf("下午 Tick 不得创建今日轮次, got %+v", r)
	}
	drawn, total, _ := store.ListDrawnLotteryRounds(context.Background(), 0, 20)
	if total != 0 || len(drawn) != 0 {
		t.Fatalf("下午 Tick 不得产生历史, total=%d", total)
	}
}

// 报名窗内 + 今日未结算 → Tick 才 Settle，且不得同 Tick 开奖。
func TestTick_MorningSettlesOnce(t *testing.T) {
	setClock(t, cst(2026, 7, 15, 0, 5))
	store := newMemStore()
	store.spendByDay["2026-07-14"] = 200
	svc := NewLotteryService(store, newEnabledConfig(), nil)

	svc.Tick()
	r, _ := store.GetLotteryRound(context.Background(), "2026-07-15")
	if r == nil {
		t.Fatal("报名窗内应 Settle 今日")
	}
	if r.Status != LotteryRoundRegistering {
		t.Fatalf("status=%s want registering", r.Status)
	}
	if r.SourceDate != "2026-07-14" {
		t.Fatalf("source=%s", r.SourceDate)
	}
	// 奖池 = 0.05 * 200
	if r.PoolAmount != 10 {
		t.Fatalf("pool=%v want 10", r.PoolAmount)
	}

	// 幂等：再 Tick 不改
	svc.Tick()
	r2, _ := store.GetLotteryRound(context.Background(), "2026-07-15")
	if r2 == nil || r2.Status != LotteryRoundRegistering {
		t.Fatal("二次 Tick 应保持 registering")
	}
}

// 过 12 点：已有 registering → Tick 开奖进历史。
func TestTick_NoonDrawsExisting(t *testing.T) {
	setClock(t, cst(2026, 7, 15, 12, 1))
	store := newMemStore()
	settle, _ := RoundSettleTime("2026-07-15")
	draw, _ := RoundDrawTime("2026-07-15")
	_ = store.SaveLotteryRound(context.Background(), &LotteryRound{
		ID: "2026-07-15", SourceDate: "2026-07-14",
		SettleTime: settle, DrawTime: draw,
		Status: LotteryRoundRegistering, PoolAmount: 5, AutoPayout: false,
	})
	svc := NewLotteryService(store, newEnabledConfig(), nil)

	svc.Tick()
	r, _ := store.GetLotteryRound(context.Background(), "2026-07-15")
	if r == nil || r.Status != LotteryRoundDrawn {
		t.Fatalf("应开奖, got %+v", r)
	}
	if r.RegisteredCount != 0 || r.CarryOut != 5 {
		t.Fatalf("0 人应全额结转, reg=%d carry=%v", r.RegisteredCount, r.CarryOut)
	}
	// 过午不得再 Settle 新轮（已存在）
	if _, err := svc.Settle("2026-07-15"); err != nil {
		// 已存在应直接返回 nil err + existing
		t.Fatalf("已存在 Settle 应幂等返回: %v", err)
	}
}

// 过午手动 Settle 今日 → 拒绝。
func TestSettle_RejectAfterDrawHour(t *testing.T) {
	setClock(t, cst(2026, 7, 15, 15, 52))
	svc := NewLotteryService(newMemStore(), newEnabledConfig(), nil)
	_, err := svc.Settle("2026-07-15")
	if err != ErrLotteryOutsideWindow {
		t.Fatalf("err=%v want OutsideWindow", err)
	}
}

// Draw 在 12 点前拒绝（语义是未到开奖时间，不是报名截止）。
func TestDraw_RejectBeforeNoon(t *testing.T) {
	setClock(t, cst(2026, 7, 15, 11, 59))
	store := newMemStore()
	settle, _ := RoundSettleTime("2026-07-15")
	draw, _ := RoundDrawTime("2026-07-15")
	_ = store.SaveLotteryRound(context.Background(), &LotteryRound{
		ID: "2026-07-15", SourceDate: "2026-07-14",
		SettleTime: settle, DrawTime: draw,
		Status: LotteryRoundRegistering, PoolAmount: 5,
	})
	svc := NewLotteryService(store, newEnabledConfig(), nil)
	_, err := svc.Draw("2026-07-15")
	if err != ErrLotteryDrawNotDue {
		t.Fatalf("err=%v want DrawNotDue", err)
	}
}

// 过去日期且已过其 12:00：禁止再 Settle 出 registering。
func TestSettle_RejectPastDateAfterItsDrawHour(t *testing.T) {
	setClock(t, cst(2026, 7, 16, 10, 0))
	svc := NewLotteryService(newMemStore(), newEnabledConfig(), nil)
	_, err := svc.Settle("2026-07-15") // 15 号 12:00 已过
	if err != ErrLotteryOutsideWindow {
		t.Fatalf("err=%v want OutsideWindow", err)
	}
}

// 历史只含 drawn。
func TestHistory_DrawnOnly(t *testing.T) {
	setClock(t, cst(2026, 7, 15, 10, 0))
	store := newMemStore()
	settle, _ := RoundSettleTime("2026-07-14")
	draw, _ := RoundDrawTime("2026-07-14")
	_ = store.SaveLotteryRound(context.Background(), &LotteryRound{
		ID: "2026-07-14", Status: LotteryRoundDrawn,
		SettleTime: settle, DrawTime: draw,
	})
	settle2, _ := RoundSettleTime("2026-07-15")
	draw2, _ := RoundDrawTime("2026-07-15")
	_ = store.SaveLotteryRound(context.Background(), &LotteryRound{
		ID: "2026-07-15", Status: LotteryRoundRegistering,
		SettleTime: settle2, DrawTime: draw2,
	})
	svc := NewLotteryService(store, newEnabledConfig(), nil)
	list, total, err := svc.History(context.Background(), 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if total != 1 || len(list) != 1 || list[0].ID != "2026-07-14" {
		t.Fatalf("history=%+v total=%d", list, total)
	}
}

// 用户端状态：当期累计（今日）字段——期号取次日、达标按门槛判定、实时按用户聚合。
func TestUserStatus_AccumFields(t *testing.T) {
	setClock(t, cst(2026, 7, 16, 9, 0)) // 报名窗内，当期累计=7/16
	store := newMemStore()
	store.spendByDay["2026-07-16"] = 40 // 今日组扣费 → accumPool = 0.05*40
	store.userSpend = 3                 // 本人今日扣费 3 ≥ 门槛 2 → 达标
	svc := NewLotteryService(store, newEnabledConfig(), nil)
	settings, err := LoadLotterySettings(context.Background(), svc.config)
	if err != nil {
		t.Fatal(err)
	}

	st, err := svc.userStatus(context.Background(), settings, 42, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if st.AccumRoundDate != "2026-07-17" { // 当期累计对应次日轮次
		t.Fatalf("accumRoundDate=%q want 2026-07-17", st.AccumRoundDate)
	}
	if st.Threshold != 2 {
		t.Fatalf("threshold=%v want 2", st.Threshold)
	}
	if st.AccumUserSpend != 3 {
		t.Fatalf("accumUserSpend=%v want 3", st.AccumUserSpend)
	}
	if !st.AccumEligible {
		t.Fatal("accumEligible want true (3 ≥ 2)")
	}

	// 未达门槛 → 未达标（同一 svc 复用，验证按用户实时聚合、非缓存快照）
	store.mu.Lock()
	store.userSpend = 1
	store.mu.Unlock()
	st2, err := svc.userStatus(context.Background(), settings, 42, "alice")
	if err != nil {
		t.Fatal(err)
	}
	if st2.AccumEligible {
		t.Fatal("accumEligible want false (1 < 2)")
	}

	// 未鉴权（userID=0）：不做本人聚合，但门槛/期号仍给出
	st3, err := svc.userStatus(context.Background(), settings, 0, "")
	if err != nil {
		t.Fatal(err)
	}
	if st3.AccumEligible || st3.AccumUserSpend != 0 {
		t.Fatalf("未鉴权不应达标, got eligible=%v spend=%v", st3.AccumEligible, st3.AccumUserSpend)
	}
	if st3.Threshold != 2 || st3.AccumRoundDate != "2026-07-17" {
		t.Fatalf("未鉴权仍应给门槛/期号, got threshold=%v date=%q", st3.Threshold, st3.AccumRoundDate)
	}
}

func TestInRegisterWindow(t *testing.T) {
	if !InRegisterWindow(cst(2026, 7, 15, 0, 0)) {
		t.Fatal("00:00 应在窗口")
	}
	if !InRegisterWindow(cst(2026, 7, 15, 11, 59)) {
		t.Fatal("11:59 应在窗口")
	}
	if InRegisterWindow(cst(2026, 7, 15, 12, 0)) {
		t.Fatal("12:00 不应在窗口")
	}
	if InRegisterWindow(cst(2026, 7, 15, 15, 52)) {
		t.Fatal("15:52 不应在窗口")
	}
}
