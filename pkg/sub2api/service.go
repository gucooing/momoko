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
	publicSnapshotTTL     = 30 * time.Second
	stateWriteTimeout     = 30 * time.Second
	testConnectionTimeout = 30 * time.Second
	autoSyncStartDelay    = 5 * time.Second

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

func NewService(store UsageStore, config ConfigStore) *Service {
	manager := NewManager()
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

// TestConnection 主动访问 Sub2API，使用独立的后台上下文，避免受 HTTP 请求超时影响。
func (s *Service) TestConnection(_ context.Context, cfg *v1.Sub2APIConfig) (bool, string) {
	client := ClientConfigFromConfig(cfg)
	if !client.Configured() {
		return false, ErrConfigRequired.Error()
	}
	ctx, cancel := context.WithTimeout(context.Background(), testConnectionTimeout)
	defer cancel()
	if err := s.manager.Test(ctx, client); err != nil {
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
	return BuildSnapshot(ctx, s.store, cfg, state)
}

// PublicSnapshot 返回脱敏后的公开快照，带短时缓存以保护未鉴权接口。
func (s *Service) PublicSnapshot(ctx context.Context) (*v1.Sub2APIUsageSnapshot, error) {
	if cached := s.cachedPublic(); cached != nil {
		return cached, nil
	}
	snapshot, err := s.Snapshot(ctx)
	if err != nil {
		return nil, err
	}
	StripForPublic(snapshot)
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

// AdminStats 返回管理端指定时间段聚合的用量概览，不受公开首页开关影响。
// 最近请求改由 RecentRequests 分页返回。
func (s *Service) AdminStats(ctx context.Context, start, end time.Time) (*v1.Sub2APIStats, error) {
	return BuildRangeStats(ctx, s.store, start, end)
}

// RecentRequests 返回管理端最近请求的分页结果（按时间倒序，最新在前），并附区间总数。
// 时间段归一化口径与 AdminStats 一致，确保翻页与概览覆盖相同区间。
func (s *Service) RecentRequests(ctx context.Context, start, end time.Time, page, pageSize int) ([]*v1.Sub2APIRecentRequest, int, error) {
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
	records, total, err := s.store.RecordsPage(ctx, &start, &end, (page-1)*pageSize, pageSize)
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
