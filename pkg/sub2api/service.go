package sub2api

import (
	"context"
	"errors"
	"sync"
	"time"

	v1 "momoko/api/gen/v1"
)

// 服务层错误，由 biz 层翻译为对外响应。
var (
	ErrConfigRequired       = errors.New("Sub2API 地址与管理员 API Key 不能为空")
	ErrHomeConfigIncomplete = errors.New("开启公开首页前需完成 Sub2API 地址与管理员 API Key 配置")
	ErrSyncRunning          = errors.New("同步任务进行中，请稍后再试")
)

const (
	publicSnapshotTTL  = 30 * time.Second
	stateWriteTimeout  = 30 * time.Second
	autoSyncStartDelay = 5 * time.Second

	// 最近请求分页边界。
	defaultRecentPageSize = 10
	maxRecentPageSize     = 100
)

// Service 编排 Sub2API 的配置、同步与用量聚合。所有与 Sub2API 的交互及数据处理都在此完成。
type Service struct {
	store  UsageStore
	config ConfigStore

	manager *Manager
	syncer  *Syncer

	runMu   sync.Mutex
	running bool

	stopMu sync.Mutex
	stopCh chan struct{}

	cacheMu      sync.Mutex
	publicCache  *v1.Sub2APIUsageSnapshot
	publicCacheT time.Time
}

// NewService 构造用量同步服务。manager 为 nil 时使用共享 NewManager()。
func NewService(store UsageStore, config ConfigStore, manager *Manager) *Service {
	if manager == nil {
		manager = NewManager()
	}
	return &Service{
		store:   store,
		config:  config,
		manager: manager,
		syncer:  NewSyncer(manager),
	}
}

// Start 启动后台自动同步循环。
func (s *Service) Start() {
	s.stopMu.Lock()
	defer s.stopMu.Unlock()
	if s.stopCh != nil {
		return
	}
	stopCh := make(chan struct{})
	s.stopCh = stopCh
	go s.syncLoop(stopCh)
}

// Stop 停止后台同步循环。
func (s *Service) Stop() {
	s.stopMu.Lock()
	defer s.stopMu.Unlock()
	if s.stopCh == nil {
		return
	}
	close(s.stopCh)
	s.stopCh = nil
}

func (s *Service) Config(ctx context.Context) (*v1.Sub2APIConfig, error) {
	return LoadConfig(ctx, s.config)
}

// ListGroups 返回本地分组（含是否公开）。
func (s *Service) ListGroups(ctx context.Context) ([]*Group, error) {
	return s.store.ListGroups(ctx)
}

func (s *Service) UpdateConfig(ctx context.Context, next *v1.Sub2APIConfig) (*v1.Sub2APIConfig, error) {
	if next == nil {
		next = DefaultConfig()
	}
	NormalizeConfig(next)
	if next.HomeEnabled && !ClientConfigFromConfig(next).Configured() {
		return nil, ErrHomeConfigIncomplete
	}

	old, err := LoadConfig(ctx, s.config)
	if err != nil {
		return nil, err
	}
	if err = SaveConfig(ctx, s.config, next); err != nil {
		return nil, err
	}
	// 公开分组启用状态落在分组表（非 KV）。
	if err = s.store.SetPublicGroups(ctx, next.GetPublicGroups()); err != nil {
		return nil, err
	}
	if old.BaseUrl != next.BaseUrl || old.AdminApiKey != next.AdminApiKey {
		if err = s.store.ClearUsageRecords(ctx); err != nil {
			return nil, err
		}
		if err = ResetSyncState(ctx, s.config); err != nil {
			return nil, err
		}
	}
	s.invalidatePublic()
	return next, nil
}

// TestConnection 主动访问 Sub2API；出站不接框架 ctx，由共享 Client 超时兜底。
func (s *Service) TestConnection(_ context.Context, cfg *v1.Sub2APIConfig) (bool, string) {
	client := ClientConfigFromConfig(cfg)
	if !client.Configured() {
		return false, ErrConfigRequired.Error()
	}
	if err := s.manager.Test(client); err != nil {
		return false, err.Error()
	}
	return true, "连接成功"
}

// Snapshot 返回完整的管理端聚合快照。
func (s *Service) Snapshot(ctx context.Context) (*v1.Sub2APIUsageSnapshot, error) {
	cfg, err := LoadConfig(ctx, s.config)
	if err != nil {
		return nil, err
	}
	state, err := LoadSyncState(ctx, s.config)
	if err != nil {
		return nil, err
	}
	return BuildSnapshot(ctx, s.store, cfg, state, false)
}

// PublicSnapshot 返回脱敏后的公开快照（按 public_groups 过滤），带短时缓存以保护未鉴权接口。
func (s *Service) PublicSnapshot(ctx context.Context) (*v1.Sub2APIUsageSnapshot, error) {
	if cached := s.cachedPublic(); cached != nil {
		return cached, nil
	}
	cfg, err := LoadConfig(ctx, s.config)
	if err != nil {
		return nil, err
	}
	state, err := LoadSyncState(ctx, s.config)
	if err != nil {
		return nil, err
	}
	snapshot, err := BuildPublicSnapshot(ctx, s.store, cfg, state)
	if err != nil {
		return nil, err
	}
	s.storePublic(snapshot)
	return snapshot, nil
}

// PublicStats 返回指定区间的公开用量统计（无需鉴权）。
func (s *Service) PublicStats(ctx context.Context, rangeDays int32) (*v1.Sub2APIStats, error) {
	cfg, err := LoadConfig(ctx, s.config)
	if err != nil {
		return nil, err
	}
	if !cfg.HomeEnabled {
		return &v1.Sub2APIStats{RangeDays: normalizeRangeDays(rangeDays, cfg.HistoryDays)}, nil
	}
	return BuildStats(ctx, s.store, cfg, rangeDays)
}

// adminWindow 由归一化后的时间段构造聚合窗口（管理端不受公开开关影响）。
// start 为零表示不限起点。
func adminWindow(start, end time.Time) StatsWindow {
	w := StatsWindow{End: &end}
	if !start.IsZero() {
		w.Start = &start
	}
	return w
}

// AdminTotals 管理端用量汇总（标量指标 + 区间标签），概览指标带独立拉取。
func (s *Service) AdminTotals(ctx context.Context, start, end time.Time) (*v1.GetSub2APIAdminTotalsResponse, error) {
	start, end = normalizeRange(start, end)
	now := time.Now()
	totals, err := s.store.AggregateTotals(ctx, adminWindow(start, end))
	if err != nil {
		return nil, err
	}
	return &v1.GetSub2APIAdminTotalsResponse{
		RangeLabel:       rangeStatsLabel(start, end, now),
		RequestCount:     totals.RequestCount,
		SuccessCount:     totals.SuccessCount,
		SuccessRate:      percent(totals.SuccessCount, totals.RequestCount),
		TokenCount:       totals.TokenCount,
		AverageLatencyMs: totals.AverageLatencyMS,
		AverageTps:       totals.AverageTPS,
	}, nil
}

// AdminTrend 管理端用量趋势：区间 <=24h 且有明确起点按 15 分钟桶展示日内曲线，否则按天补齐缺口。
func (s *Service) AdminTrend(ctx context.Context, start, end time.Time) (*v1.GetSub2APIAdminTrendResponse, error) {
	start, end = normalizeRange(start, end)
	window := adminWindow(start, end)
	if !start.IsZero() && end.Sub(start) <= time.Duration(minutesPerDay)*time.Minute {
		series, err := s.store.IntradaySeries(ctx, window)
		if err != nil {
			return nil, err
		}
		return &v1.GetSub2APIAdminTrendResponse{Trend: intradayTrendPoints(series)}, nil
	}
	daily, err := s.store.DailyTrend(ctx, window)
	if err != nil {
		return nil, err
	}
	startDay, days := trendDayRange(start, end, daily)
	return &v1.GetSub2APIAdminTrendResponse{Trend: dailyTrendPoints(daily, startDay, days)}, nil
}

// AdminTop 管理端用量排行：模型 + 分组，均按 token 降序，一次请求返回。
func (s *Service) AdminTop(ctx context.Context, start, end time.Time, limit int) (*v1.GetSub2APIAdminTopResponse, error) {
	start, end = normalizeRange(start, end)
	window := adminWindow(start, end)
	models, err := s.store.TopItems(ctx, window, GroupByModel, limit)
	if err != nil {
		return nil, err
	}
	groups, err := s.store.TopItems(ctx, window, GroupByGroup, limit)
	if err != nil {
		return nil, err
	}
	return &v1.GetSub2APIAdminTopResponse{
		Models: mapTopItems(models),
		Groups: mapTopItems(groups),
	}, nil
}

// RecentRequests 返回管理端最近请求的分页结果（按时间倒序，最新在前），并附区间总数。
// 时间段归一化口径与概览各接口一致，确保翻页与概览覆盖相同区间；filter 承载多维度筛选。
// start 为零表示不限制起点（全部记录）。
func (s *Service) RecentRequests(ctx context.Context, start, end time.Time, page, pageSize int, filter RecordFilter) ([]*v1.Sub2APIRecentRequest, int, error) {
	start, end = normalizeRange(start, end)
	if page < 1 {
		page = 1
	}
	switch {
	case pageSize <= 0:
		pageSize = defaultRecentPageSize
	case pageSize > maxRecentPageSize:
		pageSize = maxRecentPageSize
	}
	var startPtr *time.Time
	if !start.IsZero() {
		startPtr = &start
	}
	records, total, err := s.store.RecordsPage(ctx, startPtr, &end, (page-1)*pageSize, pageSize, false, filter)
	if err != nil {
		return nil, 0, err
	}
	return toRecentRequests(records), total, nil
}

// Sync 执行一次同步（手动或自动）。整个流程跑在独立的后台上下文中，
// 不受触发它的 HTTP 请求超时/取消影响。
func (s *Service) Sync(_ context.Context, full bool) (*v1.Sub2APIUsageSnapshot, error) {
	if !s.acquireSync() {
		return nil, ErrSyncRunning
	}
	defer s.releaseSync()

	syncCtx, cancel := context.WithTimeout(context.Background(), SyncTimeout)
	defer cancel()

	cfg, err := LoadConfig(syncCtx, s.config)
	if err != nil {
		return nil, err
	}
	client := ClientConfigFromConfig(cfg)
	if !client.Configured() {
		return nil, ErrConfigRequired
	}
	if full {
		if err = s.store.ClearUsageRecords(syncCtx); err != nil {
			return nil, err
		}
	}

	start := time.Now()
	nextTime := start.Add(time.Duration(cfg.SyncIntervalMinutes) * time.Minute)
	if err = s.writeSyncState(SyncState{Status: SyncStatusSyncing, NextSyncTime: &nextTime}); err != nil {
		return nil, err
	}

	result, syncErr := s.syncer.Sync(syncCtx, client, s.store, SyncOptions{
		Full:     full,
		PageSize: int(cfg.PageSize),
	})
	if err = s.writeSyncState(NewFinishedSyncState(start, nextTime, result, syncErr)); err != nil {
		return nil, err
	}
	if syncErr != nil {
		return nil, syncErr
	}
	s.invalidatePublic()
	return s.Snapshot(syncCtx)
}

func (s *Service) syncLoop(stopCh <-chan struct{}) {
	timer := time.NewTimer(autoSyncStartDelay)
	defer timer.Stop()
	for {
		select {
		case <-stopCh:
			return
		case <-timer.C:
			s.runAutoSync()
			timer.Reset(s.nextInterval())
		}
	}
}

func (s *Service) runAutoSync() {
	cfg, err := LoadConfig(context.Background(), s.config)
	if err != nil {
		return
	}
	if !cfg.SyncEnabled || !ClientConfigFromConfig(cfg).Configured() {
		return
	}
	_, _ = s.Sync(context.Background(), false)
}

func (s *Service) nextInterval() time.Duration {
	cfg, err := LoadConfig(context.Background(), s.config)
	if err != nil {
		return time.Duration(DefaultSyncIntervalMinutes) * time.Minute
	}
	return time.Duration(cfg.SyncIntervalMinutes) * time.Minute
}

func (s *Service) writeSyncState(state SyncState) error {
	ctx, cancel := context.WithTimeout(context.Background(), stateWriteTimeout)
	defer cancel()
	return SaveSyncState(ctx, s.config, state)
}

func (s *Service) acquireSync() bool {
	s.runMu.Lock()
	defer s.runMu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	return true
}

func (s *Service) releaseSync() {
	s.runMu.Lock()
	s.running = false
	s.runMu.Unlock()
}

func (s *Service) cachedPublic() *v1.Sub2APIUsageSnapshot {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	if s.publicCache != nil && time.Since(s.publicCacheT) < publicSnapshotTTL {
		return s.publicCache
	}
	return nil
}

func (s *Service) storePublic(snapshot *v1.Sub2APIUsageSnapshot) {
	s.cacheMu.Lock()
	s.publicCache = snapshot
	s.publicCacheT = time.Now()
	s.cacheMu.Unlock()
}

func (s *Service) invalidatePublic() {
	s.cacheMu.Lock()
	s.publicCache = nil
	s.cacheMu.Unlock()
}
