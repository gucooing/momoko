package biz

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	gcpu "github.com/shirou/gopsutil/v4/cpu"
	gprocess "github.com/shirou/gopsutil/v4/process"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/common"
	"momoko/pkg/response"
	"momoko/pkg/sysmon"
	"momoko/pkg/task"
)

// MonitorRow 一行历史采样：Data 为 monitorPoint 的 JSON。
type MonitorRow struct {
	Tier int8
	Time time.Time
	Data string
}

type MonitorRepo interface {
	InsertSamples(ctx context.Context, rows []*MonitorRow) error
	ListSamples(ctx context.Context, tier int8, from, to time.Time) ([]*MonitorRow, error)
	// EdgeSample first=true 时返回 since 之后最早的一行，false 时返回最新的一行；没有返回 nil
	EdgeSample(ctx context.Context, tier int8, since time.Time, first bool) (*MonitorRow, error)
	DeleteSamplesBefore(ctx context.Context, tier int8, before time.Time) (int64, error)
	DeleteAll(ctx context.Context) (int64, error)
	StorageStats(ctx context.Context) (rows int64, dataBytes int64, err error)
}

// —— 精度分级 ——
// 原始精度保留 24 小时 → 1 分钟精度保留 7 天 → 10 分钟精度保留 90 天 → 1 小时精度保留到期满；
// 保留时长更短时各级同步截短，用不到的级别不生成。

const (
	tierRaw int8 = iota
	tier1m
	tier10m
	tier1h
)

var tierSteps = [...]time.Duration{0, time.Minute, 10 * time.Minute, time.Hour}
var tierNames = [...]string{"raw", "1m", "10m", "1h"}

func tierEnabled(tier int8, days int32) bool {
	switch tier {
	case tier10m:
		return days > 7
	case tier1h:
		return days > 90
	}
	return true
}

func tierRetention(tier int8, days int32) time.Duration {
	day := 24 * time.Hour
	switch tier {
	case tierRaw:
		return day
	case tier1m:
		return time.Duration(min(days, 7)) * day
	case tier10m:
		return time.Duration(min(days, 90)) * day
	}
	return time.Duration(days) * day
}

func tierStep(tier int8, interval time.Duration) time.Duration {
	if tier == tierRaw {
		return interval
	}
	return tierSteps[tier]
}

// —— 采样点：指标键 → [平均值, 峰值, 样本数] ——

type monitorStat [3]float64

type monitorPoint map[string]monitorStat

func roundMetric(v float64) float64 {
	if math.Abs(v) >= 100 {
		return math.Round(v)
	}
	return math.Round(v*100) / 100
}

func pointFromValues(values map[string]float64) monitorPoint {
	p := make(monitorPoint, len(values))
	for k, v := range values {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			continue
		}
		r := roundMetric(v)
		p[k] = monitorStat{r, r, 1}
	}
	return p
}

// merge 把 src 合并进 p：平均值按样本数加权，峰值取最大，样本数相加。
func (p monitorPoint) merge(src monitorPoint) {
	for k, s := range src {
		cur, ok := p[k]
		if !ok {
			p[k] = s
			continue
		}
		n := cur[2] + s[2]
		if n <= 0 {
			continue
		}
		p[k] = monitorStat{(cur[0]*cur[2] + s[0]*s[2]) / n, math.Max(cur[1], s[1]), n}
	}
}

func (p monitorPoint) encode() string {
	out := make(map[string][3]float64, len(p))
	for k, s := range p {
		out[k] = [3]float64{roundMetric(s[0]), roundMetric(s[1]), s[2]}
	}
	raw, _ := json.Marshal(out)
	return string(raw)
}

func decodePoint(raw string) monitorPoint {
	var m map[string][3]float64
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return monitorPoint{}
	}
	p := make(monitorPoint, len(m))
	for k, s := range m {
		p[k] = monitorStat(s)
	}
	return p
}

// 按应用统计的键：acpu:<实例id>（占整机 CPU %）、amem:<实例id>（常驻内存字节）、apps（运行中的应用数）。
const (
	monitorKeyAppCPU = "acpu"
	monitorKeyAppMem = "amem"
	monitorKeyApps   = "apps"
)

// —— 设置 ——

var (
	monitorIntervals = map[int32]bool{5: true, 10: true, 30: true, 60: true}
	defaultMonitor   = &v1.MonitorSettings{
		RecordEnabled: true, IntervalSeconds: 10, RetentionDays: 30,
		CollectGpu: true, CollectDisk: true, CollectNetwork: true, CollectApps: true,
	}
)

const (
	monitorMinRetentionDays = 1
	monitorMaxRetentionDays = 3650
	// monitorRingWindow 内存里保留的最近原始采样（关闭记录历史后仍能看实时曲线）。
	monitorRingWindow = time.Hour
	// monitorCleanupType 「性能监控数据清理」定时单例的类型键。
	monitorCleanupType     = "monitor.cleanup"
	monitorCleanupInterval = time.Hour
	// monitorRollupLag 聚合只处理已结束的桶，并留出最后一次采样落库的时间。
	monitorRollupLag = 15 * time.Second
)

type memSample struct {
	at    time.Time
	point monitorPoint
}

// MonitorUsecase 持久性能监控：常驻采集协程 + 分级降采样 + 定时清理 + 查询。
type MonitorUsecase struct {
	repo      MonitorRepo
	config    ConfigRepo
	instances *InstanceUsecase
	sampler   *sysmon.Sampler
	procs     *instanceStatsCache

	mu       sync.RWMutex
	settings *v1.MonitorSettings
	latest   *sysmon.Sample
	ring     []memSample

	wake    chan struct{}
	stop    chan struct{}
	stopped sync.WaitGroup
}

func NewMonitorUsecase(repo MonitorRepo, config ConfigRepo, instances *InstanceUsecase, tasks *task.Manager) (*MonitorUsecase, func()) {
	m := &MonitorUsecase{
		repo:      repo,
		config:    config,
		instances: instances,
		sampler:   sysmon.NewSampler(),
		procs: &instanceStatsCache{
			procs: make(map[string]map[int32]*gprocess.Process),
			disk:  make(map[string]*instanceDiskUsage),
		},
		wake: make(chan struct{}, 1),
		stop: make(chan struct{}),
	}
	m.settings = m.loadSettings(context.Background())
	m.stopped.Add(2)
	go m.collectLoop()
	go m.rollupLoop()
	_ = tasks.EnsureSingleton(context.Background(), &monitorCleanupTask{m: m})
	return m, m.close
}

func (m *MonitorUsecase) close() {
	close(m.stop)
	m.stopped.Wait()
	m.sampler.Close()
}

func (m *MonitorUsecase) currentSettings() *v1.MonitorSettings {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return cloneMonitorSettings(m.settings)
}

func cloneMonitorSettings(s *v1.MonitorSettings) *v1.MonitorSettings {
	return &v1.MonitorSettings{
		RecordEnabled: s.GetRecordEnabled(), IntervalSeconds: s.GetIntervalSeconds(), RetentionDays: s.GetRetentionDays(),
		CollectGpu: s.GetCollectGpu(), CollectDisk: s.GetCollectDisk(), CollectNetwork: s.GetCollectNetwork(), CollectApps: s.GetCollectApps(),
	}
}

func (m *MonitorUsecase) loadSettings(ctx context.Context) *v1.MonitorSettings {
	s := cloneMonitorSettings(defaultMonitor)
	getBool := func(key common.ConfigKey, def bool) bool {
		raw, err := m.config.Get(ctx, key)
		if err != nil {
			return def
		}
		v, err := strconv.ParseBool(raw)
		if err != nil {
			return def
		}
		return v
	}
	getInt := func(key common.ConfigKey, def int32) int32 {
		raw, err := m.config.Get(ctx, key)
		if err != nil {
			return def
		}
		v, err := strconv.Atoi(strings.TrimSpace(raw))
		if err != nil {
			return def
		}
		return int32(v)
	}
	s.RecordEnabled = getBool(common.ConfigMonitorRecordEnabled, s.RecordEnabled)
	s.IntervalSeconds = getInt(common.ConfigMonitorInterval, s.IntervalSeconds)
	s.RetentionDays = getInt(common.ConfigMonitorRetentionDays, s.RetentionDays)
	s.CollectGpu = getBool(common.ConfigMonitorCollectGPU, s.CollectGpu)
	s.CollectDisk = getBool(common.ConfigMonitorCollectDisk, s.CollectDisk)
	s.CollectNetwork = getBool(common.ConfigMonitorCollectNetwork, s.CollectNetwork)
	s.CollectApps = getBool(common.ConfigMonitorCollectApps, s.CollectApps)
	if !monitorIntervals[s.IntervalSeconds] {
		s.IntervalSeconds = defaultMonitor.IntervalSeconds
	}
	if s.RetentionDays < monitorMinRetentionDays || s.RetentionDays > monitorMaxRetentionDays {
		s.RetentionDays = defaultMonitor.RetentionDays
	}
	return s
}

func (m *MonitorUsecase) interval() time.Duration {
	return time.Duration(m.currentSettings().IntervalSeconds) * time.Second
}

// —— 采集 ——

func (m *MonitorUsecase) collectLoop() {
	defer m.stopped.Done()
	ctx := context.Background()
	// 先采一次建立速率基线（不落库），第一条历史就有 CPU 与网络 / 磁盘速率
	s := m.currentSettings()
	m.sampler.Sample(ctx, time.Now(), sampleOptions(s))
	if s.CollectApps {
		m.sampleApps(ctx, map[string]float64{})
	}
	for {
		iv := m.interval()
		slot := time.Now().Truncate(iv).Add(iv)
		timer := time.NewTimer(time.Until(slot))
		select {
		case <-m.stop:
			timer.Stop()
			return
		case <-m.wake:
			timer.Stop()
			continue
		case <-timer.C:
		}
		m.collect(ctx, slot)
	}
}

func sampleOptions(s *v1.MonitorSettings) sysmon.Options {
	return sysmon.Options{GPU: s.CollectGpu, Disk: s.CollectDisk, Network: s.CollectNetwork}
}

func (m *MonitorUsecase) collect(ctx context.Context, slot time.Time) {
	s := m.currentSettings()
	cctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	smp := m.sampler.Sample(cctx, slot, sampleOptions(s))
	if s.CollectApps {
		m.sampleApps(cctx, smp.Values)
	}
	point := pointFromValues(smp.Values)

	m.mu.Lock()
	m.latest = smp
	m.ring = append(m.ring, memSample{at: slot, point: point})
	cut := 0
	for cut < len(m.ring) && m.ring[cut].at.Before(slot.Add(-monitorRingWindow)) {
		cut++
	}
	m.ring = append([]memSample(nil), m.ring[cut:]...)
	m.mu.Unlock()

	if s.RecordEnabled {
		_ = m.repo.InsertSamples(cctx, []*MonitorRow{{Tier: tierRaw, Time: slot, Data: point.encode()}})
	}
}

// sampleApps 统计每个运行中应用（主进程及子进程）的 CPU 与内存；CPU 按逻辑核心数折算成占整机的百分比。
func (m *MonitorUsecase) sampleApps(ctx context.Context, values map[string]float64) {
	if m.instances == nil {
		return
	}
	running := m.instances.RunningProcesses()
	cores, _ := gcpu.CountsWithContext(ctx, true)
	if cores <= 0 {
		cores = 1
	}
	for id, pid := range running {
		procs := m.procs.processTree(ctx, id, int32(pid))
		var cpu, rss float64
		for _, p := range procs {
			if pct, err := p.PercentWithContext(ctx, 0); err == nil {
				cpu += pct
			}
			if info, err := p.MemoryInfoWithContext(ctx); err == nil && info != nil {
				rss += float64(info.RSS)
			}
		}
		values[monitorKeyAppCPU+":"+id] = math.Min(100, cpu/float64(cores))
		values[monitorKeyAppMem+":"+id] = rss
	}
	m.procs.retain(running)
	values[monitorKeyApps] = float64(len(running))
}

// retain 丢掉已不在运行的应用的进程句柄。
func (c *instanceStatsCache) retain(running map[string]int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for id := range c.procs {
		if _, ok := running[id]; !ok {
			delete(c.procs, id)
		}
	}
}

// —— 分级降采样 ——

func (m *MonitorUsecase) rollupLoop() {
	defer m.stopped.Done()
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-ticker.C:
			m.rollup(context.Background())
		}
	}
}

func (m *MonitorUsecase) rollup(ctx context.Context) {
	days := m.currentSettings().RetentionDays
	for tier := tier1m; tier <= tier1h; tier++ {
		if !tierEnabled(tier, days) {
			continue
		}
		if err := m.rollupTier(ctx, tier, tier-1, days); err != nil {
			return
		}
	}
}

// rollupTier 把 src 级的行按 tier 的桶宽聚合，补齐 tier 级从上次聚合位置到最近一个已结束桶之间的数据。
func (m *MonitorUsecase) rollupTier(ctx context.Context, tier, src int8, days int32) error {
	step := tierSteps[tier]
	boundary := time.Now().Add(-monitorRollupLag).Truncate(step)
	var from time.Time
	last, err := m.repo.EdgeSample(ctx, tier, time.Time{}, false)
	if err != nil {
		return err
	}
	if last != nil {
		from = last.Time.Add(step)
	} else {
		first, err := m.repo.EdgeSample(ctx, src, time.Time{}, true)
		if err != nil || first == nil {
			return err
		}
		from = first.Time.Truncate(step)
	}
	if oldest := boundary.Add(-tierRetention(tier, days)).Truncate(step); from.Before(oldest) {
		from = oldest
	}
	chunk := step * 360
	for from.Before(boundary) {
		to := from.Add(chunk)
		if to.After(boundary) {
			to = boundary
		}
		rows, err := m.repo.ListSamples(ctx, src, from, to)
		if err != nil {
			return err
		}
		buckets := map[int64]monitorPoint{}
		order := make([]int64, 0, 64)
		for _, row := range rows {
			key := row.Time.Truncate(step).Unix()
			b, ok := buckets[key]
			if !ok {
				b = monitorPoint{}
				buckets[key] = b
				order = append(order, key)
			}
			b.merge(decodePoint(row.Data))
		}
		out := make([]*MonitorRow, 0, len(order))
		for _, key := range order {
			out = append(out, &MonitorRow{Tier: tier, Time: time.Unix(key, 0), Data: buckets[key].encode()})
		}
		if err := m.repo.InsertSamples(ctx, out); err != nil {
			return err
		}
		from = to
	}
	return nil
}

// —— 清理 ——

func (m *MonitorUsecase) purge(ctx context.Context) (int64, error) {
	days := m.currentSettings().RetentionDays
	now := time.Now()
	var total int64
	for tier := tierRaw; tier <= tier1h; tier++ {
		before := time.Time{}
		if tierEnabled(tier, days) {
			before = now.Add(-tierRetention(tier, days))
		}
		n, err := m.repo.DeleteSamplesBefore(ctx, tier, before)
		if err != nil {
			return total, err
		}
		total += n
	}
	return total, nil
}

// monitorCleanupTask 每小时删除超出各级保留时长的监控数据（在任务面板可见）。
type monitorCleanupTask struct {
	m *MonitorUsecase
}

func (t *monitorCleanupTask) Meta() task.Meta {
	return task.Meta{
		ID:       monitorCleanupType,
		Type:     monitorCleanupType,
		Kind:     task.KindScheduled,
		Resume:   task.ResumeAlways,
		Interval: monitorCleanupInterval,
		Title:    "性能监控数据清理",
		Source:   "task",
	}
}

func (t *monitorCleanupTask) Payload() any { return nil }

func (t *monitorCleanupTask) Run(ctx context.Context, r task.Reporter) error {
	n, err := t.m.purge(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		r.Log(task.LogInfo, fmt.Sprintf("已清理 %d 条过期的性能监控数据", n))
	}
	return nil
}

// —— 设置接口 ——

func (m *MonitorUsecase) GetSettings(ctx context.Context) (*v1.GetMonitorSettingsResponse, error) {
	rows, size, err := m.repo.StorageStats(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	resp := &v1.GetMonitorSettingsResponse{
		Settings:     m.currentSettings(),
		Rows:         rows,
		RowBytes:     monitorRowOverhead + 600,
		StorageBytes: size + rows*monitorRowOverhead,
		FirstTime:    m.firstTime(ctx),
	}
	if rows > 0 {
		resp.RowBytes = size/rows + monitorRowOverhead
	}
	resp.Gpus, resp.GpuNote = m.gpuInfos()
	return resp, nil
}

// monitorRowOverhead 每行除 data 外的估算开销（时间、精度、主键与唯一索引）。
const monitorRowOverhead = 64

func (m *MonitorUsecase) UpdateSettings(ctx context.Context, req *v1.UpdateMonitorSettingsRequest) (*v1.UpdateMonitorSettingsResponse, error) {
	s := req.GetSettings()
	if s == nil {
		return nil, response.BadRequest(400, "缺少监控设置")
	}
	if !monitorIntervals[s.IntervalSeconds] {
		return nil, response.BadRequest(400, "采样间隔只能是 5、10、30 或 60 秒")
	}
	if s.RetentionDays < monitorMinRetentionDays || s.RetentionDays > monitorMaxRetentionDays {
		return nil, response.BadRequest(400, fmt.Sprintf("历史保留时长需为 %d 到 %d 天", monitorMinRetentionDays, monitorMaxRetentionDays))
	}
	b := strconv.FormatBool
	if err := m.config.BatchUpdate(ctx, map[common.ConfigKey]string{
		common.ConfigMonitorRecordEnabled:  b(s.RecordEnabled),
		common.ConfigMonitorInterval:       strconv.Itoa(int(s.IntervalSeconds)),
		common.ConfigMonitorRetentionDays:  strconv.Itoa(int(s.RetentionDays)),
		common.ConfigMonitorCollectGPU:     b(s.CollectGpu),
		common.ConfigMonitorCollectDisk:    b(s.CollectDisk),
		common.ConfigMonitorCollectNetwork: b(s.CollectNetwork),
		common.ConfigMonitorCollectApps:    b(s.CollectApps),
	}); err != nil {
		return nil, ErrSystem(err)
	}
	next := cloneMonitorSettings(s)
	m.mu.Lock()
	m.settings = next
	m.mu.Unlock()
	select {
	case m.wake <- struct{}{}:
	default:
	}
	deleted, err := m.purge(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.UpdateMonitorSettingsResponse{Settings: cloneMonitorSettings(next), Deleted: deleted}, nil
}

func (m *MonitorUsecase) ClearHistory(ctx context.Context) (*v1.ClearMonitorHistoryResponse, error) {
	n, err := m.repo.DeleteAll(ctx)
	if err != nil {
		return nil, ErrSystem(err)
	}
	m.mu.Lock()
	m.ring = nil
	m.mu.Unlock()
	return &v1.ClearMonitorHistoryResponse{Deleted: n}, nil
}

// gpuInfos 最近一次探测到的显卡；关闭 GPU 采集后仍返回已知的显卡（设置里据此显示「检测到 …」），
// 从未探测过时不给「未检测到」的说明。
func (m *MonitorUsecase) gpuInfos() ([]*v1.MonitorGPU, string) {
	gpus, note := m.sampler.GPUs()
	if !m.currentSettings().CollectGpu && len(gpus) == 0 {
		note = ""
	}
	out := make([]*v1.MonitorGPU, 0, len(gpus))
	for _, g := range gpus {
		out = append(out, &v1.MonitorGPU{
			Index: int32(g.Index), Name: g.Name, Vendor: g.Vendor, Driver: g.Driver, MemoryTotal: g.MemoryTotal,
		})
	}
	return out, note
}

// firstTime 全部精度里最早一行的时间（unix 秒，没有历史为 0）。
func (m *MonitorUsecase) firstTime(ctx context.Context) int64 {
	var first time.Time
	for tier := tierRaw; tier <= tier1h; tier++ {
		row, err := m.repo.EdgeSample(ctx, tier, time.Time{}, true)
		if err != nil || row == nil {
			continue
		}
		if first.IsZero() || row.Time.Before(first) {
			first = row.Time
		}
	}
	if first.IsZero() {
		return 0
	}
	return first.Unix()
}
