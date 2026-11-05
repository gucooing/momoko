package biz

import (
	"bytes"
	"context"
	"encoding/csv"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/response"
	"momoko/pkg/sysmon"
)

const (
	// monitorMaxBuckets 曲线最多的桶数；monitorMaxRows 查询时单一精度最多扫描的行数（超过就换更粗的精度）。
	monitorMaxBuckets = 360
	monitorMaxRows    = 3000
	// monitorExportMaxRows 导出时允许的最多行数。
	monitorExportMaxRows = 20000
	monitorMaxSpan       = 3660 * 24 * time.Hour
	// monitorGrowthWindow 磁盘写满预测参考的历史跨度。
	monitorGrowthWindow = 30 * 24 * time.Hour
)

var monitorSteps = []time.Duration{
	5 * time.Second, 10 * time.Second, 15 * time.Second, 30 * time.Second,
	time.Minute, 2 * time.Minute, 5 * time.Minute, 10 * time.Minute, 15 * time.Minute, 30 * time.Minute,
	time.Hour, 2 * time.Hour, 3 * time.Hour, 6 * time.Hour, 12 * time.Hour, 24 * time.Hour,
}

func monitorKeyPrefix(key string) string {
	prefix, _, _ := strings.Cut(key, ":")
	return prefix
}

// 画曲线的指标带逐桶数据；其余指标只给时段汇总（均值 / 峰值 / 累计量）。
var monitorCurvePrefixes = map[string]bool{
	sysmon.KeyCPU: true, sysmon.KeyIOWait: true, sysmon.KeyMem: true, sysmon.KeyCache: true, sysmon.KeySwap: true,
	sysmon.KeyGPU: true, sysmon.KeyVRAM: true, sysmon.KeyRx: true, sysmon.KeyTx: true,
	sysmon.KeyRead: true, sysmon.KeyWrite: true,
}

// 不作为曲线返回的键：采样间隔、磁盘容量（单独给出容量与趋势）、按应用统计（单独给出排行）。
var monitorHiddenPrefixes = map[string]bool{
	sysmon.KeyDuration: true, sysmon.KeyDiskUsed: true, sysmon.KeyDiskSize: true,
	monitorKeyAppCPU: true, monitorKeyAppMem: true, monitorKeyApps: true,
}

// 速率类指标（字节/秒），时段累计量 = Σ 速率 × 时长。
var monitorRatePrefixes = map[string]bool{
	sysmon.KeyRx: true, sysmon.KeyTx: true, sysmon.KeyRead: true, sysmon.KeyWrite: true,
}

func monitorRange(startUnix, endUnix int64, now time.Time) (time.Time, time.Time, error) {
	end := now
	if endUnix > 0 {
		end = time.Unix(endUnix, 0)
	}
	if end.After(now) {
		end = now
	}
	start := end.Add(-24 * time.Hour)
	if startUnix > 0 {
		start = time.Unix(startUnix, 0)
	}
	if !start.Before(end) {
		return time.Time{}, time.Time{}, response.BadRequest(400, "开始时间需早于结束时间")
	}
	if end.Sub(start) < 5*time.Minute {
		start = end.Add(-5 * time.Minute)
	}
	if end.Sub(start) > monitorMaxSpan {
		start = end.Add(-monitorMaxSpan)
	}
	return start, end, nil
}

// chooseTier 选能覆盖时段开头、且扫描行数不超过 maxRows 的最细精度；都不满足时用最粗的可用精度。
func chooseTier(start, end, now time.Time, days int32, interval time.Duration, maxRows int) int8 {
	span := end.Sub(start)
	best := tierRaw
	for t := tierRaw; t <= tier1h; t++ {
		if !tierEnabled(t, days) {
			continue
		}
		best = t
		covers := !start.Before(now.Add(-tierRetention(t, days) - 10*time.Minute))
		if covers && int(span/tierStep(t, interval)) <= maxRows {
			return t
		}
	}
	return best
}

func chooseStep(span, base time.Duration) time.Duration {
	// 留 5% 余量：「最近 1 小时」多出的几秒不至于把桶宽翻倍
	target := span / (monitorMaxBuckets + monitorMaxBuckets/20)
	for _, c := range monitorSteps {
		if c >= base && c >= target && c%base == 0 {
			return c
		}
	}
	return monitorSteps[len(monitorSteps)-1]
}

// loadRows 读所选精度的行；该精度还没聚合到的最近一段，依次用更细的精度补齐，最后补上内存里的实时采样。
func (m *MonitorUsecase) loadRows(ctx context.Context, tier int8, from, to time.Time, days int32, interval time.Duration) ([]memSample, error) {
	out := make([]memSample, 0, 512)
	cursor := from
	for t := tier; t >= tierRaw; t-- {
		if t != tier && !tierEnabled(t, days) {
			continue
		}
		if !cursor.Before(to) {
			break
		}
		rows, err := m.repo.ListSamples(ctx, t, cursor, to)
		if err != nil {
			return nil, err
		}
		for _, r := range rows {
			out = append(out, memSample{at: r.Time, point: decodePoint(r.Data)})
		}
		if len(rows) > 0 {
			cursor = rows[len(rows)-1].Time.Add(tierStep(t, interval))
		}
	}
	m.mu.RLock()
	for _, r := range m.ring {
		if !r.at.Before(cursor) && r.at.Before(to) {
			out = append(out, r)
		}
	}
	m.mu.RUnlock()
	return out, nil
}

// GetData 查询时间范围内的监控数据。
func (m *MonitorUsecase) GetData(ctx context.Context, req *v1.GetMonitorDataRequest) (*v1.GetMonitorDataResponse, error) {
	now := time.Now()
	s := m.currentSettings()
	interval := time.Duration(s.IntervalSeconds) * time.Second
	start, end, err := monitorRange(req.GetStart(), req.GetEnd(), now)
	if err != nil {
		return nil, err
	}
	tier := chooseTier(start, end, now, s.RetentionDays, interval, monitorMaxRows)
	step := chooseStep(end.Sub(start), tierStep(tier, interval))
	from := start.Truncate(step)
	buckets := int((end.Sub(from) + step - 1) / step)
	rows, err := m.loadRows(ctx, tier, from, end, s.RetentionDays, interval)
	if err != nil {
		return nil, ErrSystem(err)
	}
	pts := make([]monitorPoint, buckets)
	for _, r := range rows {
		b := int(r.at.Sub(from) / step)
		if b < 0 || b >= buckets {
			continue
		}
		if pts[b] == nil {
			pts[b] = monitorPoint{}
		}
		pts[b].merge(r.point)
	}

	resp := &v1.GetMonitorDataResponse{
		Start:      from.Unix(),
		End:        end.Unix(),
		Step:       int32(step / time.Second),
		Buckets:    int32(buckets),
		Resolution: tierNames[tier],
		Series:     buildMonitorSeries(pts, from, step, interval),
		Gaps:       findMonitorGaps(pts, from, step, interval),
		Host:       m.hostInfo(ctx),
		Settings:   s,
		FirstTime:  m.firstTime(ctx),
	}
	latest := m.latestSample()
	if latest != nil {
		resp.Current = currentValues(latest)
		resp.CurrentTime = latest.Time.Unix()
	}
	if s.CollectDisk {
		resp.Disks = m.diskUsage(ctx, now, latest)
	}
	if s.CollectApps {
		resp.Apps = m.appUsage(ctx, pts)
	}
	return resp, nil
}

func (m *MonitorUsecase) latestSample() *sysmon.Sample {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.latest
}

func currentValues(smp *sysmon.Sample) map[string]float64 {
	out := make(map[string]float64, len(smp.Values)+len(smp.Extra))
	for _, src := range []map[string]float64{smp.Values, smp.Extra} {
		for k, v := range src {
			p := monitorKeyPrefix(k)
			if p == monitorKeyAppCPU || p == monitorKeyAppMem || math.IsNaN(v) || math.IsInf(v, 0) {
				continue
			}
			out[k] = roundMetric(v)
		}
	}
	return out
}

func (m *MonitorUsecase) hostInfo(ctx context.Context) *v1.MonitorHost {
	h := m.sampler.Host(ctx)
	out := &v1.MonitorHost{
		Hostname: h.Hostname, Os: h.OS, CpuModel: h.CPUModel,
		CpuCores: int32(h.CPUCores), CpuThreads: int32(h.CPUThreads),
		MemoryTotal: h.MemoryTotal, SwapTotal: h.SwapTotal, Linux: h.Linux,
	}
	out.Gpus, out.GpuNote = m.gpuInfos()
	return out
}

type monitorSeriesAcc struct {
	idx    []int32
	avg    []float64
	sum    float64
	n      float64
	peak   float64
	peakAt int64
	total  float64
	seen   bool
}

func buildMonitorSeries(pts []monitorPoint, from time.Time, step, interval time.Duration) []*v1.MonitorSeries {
	accs := map[string]*monitorSeriesAcc{}
	for b, p := range pts {
		if len(p) == 0 {
			continue
		}
		dur := interval.Seconds()
		if d, ok := p[sysmon.KeyDuration]; ok && d[0] > 0 {
			dur = d[0]
		}
		at := from.Add(time.Duration(b) * step).Unix()
		for k, st := range p {
			prefix := monitorKeyPrefix(k)
			if monitorHiddenPrefixes[prefix] {
				continue
			}
			a := accs[k]
			if a == nil {
				a = &monitorSeriesAcc{}
				accs[k] = a
			}
			if monitorCurvePrefixes[prefix] {
				a.idx = append(a.idx, int32(b))
				a.avg = append(a.avg, roundMetric(st[0]))
			}
			a.sum += st[0] * st[2]
			a.n += st[2]
			if !a.seen || st[1] > a.peak {
				a.peak, a.peakAt, a.seen = st[1], at, true
			}
			if monitorRatePrefixes[prefix] {
				a.total += st[0] * st[2] * dur
			}
		}
	}
	keys := make([]string, 0, len(accs))
	for k := range accs {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]*v1.MonitorSeries, 0, len(keys))
	for _, k := range keys {
		a := accs[k]
		mean := 0.0
		if a.n > 0 {
			mean = a.sum / a.n
		}
		out = append(out, &v1.MonitorSeries{
			Key: k, Idx: a.idx, Avg: a.avg,
			Mean: roundMetric(mean), Peak: roundMetric(a.peak), PeakTime: a.peakAt, Total: math.Round(a.total),
		})
	}
	return out
}

// findMonitorGaps 找出首尾数据之间的空档（服务停止或暂停记录）：连续 2 个以上空桶，桶宽 ≥ 10 分钟时 1 个也算。
func findMonitorGaps(pts []monitorPoint, from time.Time, step, interval time.Duration) []*v1.MonitorGap {
	first, last := -1, -1
	for i, p := range pts {
		if len(p) > 0 {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if first < 0 {
		return nil
	}
	minRun := 2
	if step >= 10*time.Minute {
		minRun = 1
	}
	var out []*v1.MonitorGap
	for i := first; i <= last; {
		if len(pts[i]) > 0 {
			i++
			continue
		}
		j := i
		for j <= last && len(pts[j]) == 0 {
			j++
		}
		if j-i >= minRun && time.Duration(j-i)*step >= 3*interval {
			out = append(out, &v1.MonitorGap{
				Start: from.Add(time.Duration(i) * step).Unix(),
				End:   from.Add(time.Duration(j) * step).Unix(),
			})
		}
		i = j
	}
	return out
}

// appUsage 所选时段内每个应用的平均 CPU 与内存（未运行的时刻按 0 计）。
func (m *MonitorUsecase) appUsage(ctx context.Context, pts []monitorPoint) []*v1.MonitorAppUsage {
	var samples float64
	cpu := map[string]float64{}
	mem := map[string]float64{}
	for _, p := range pts {
		if st, ok := p[monitorKeyApps]; ok {
			samples += st[2]
		}
		for k, st := range p {
			prefix, id, ok := strings.Cut(k, ":")
			if !ok {
				continue
			}
			switch prefix {
			case monitorKeyAppCPU:
				cpu[id] += st[0] * st[2]
			case monitorKeyAppMem:
				mem[id] += st[0] * st[2]
			}
		}
	}
	running := map[string]int{}
	if m.instances != nil {
		running = m.instances.RunningProcesses()
	}
	ids := make([]string, 0, len(cpu)+len(running))
	seen := map[string]bool{}
	for id := range cpu {
		ids, seen[id] = append(ids, id), true
	}
	for id := range mem {
		if !seen[id] {
			ids, seen[id] = append(ids, id), true
		}
	}
	if len(ids) == 0 || m.instances == nil {
		return nil
	}
	names := m.instances.InstanceNames(ctx, ids)
	out := make([]*v1.MonitorAppUsage, 0, len(ids))
	for _, id := range ids {
		name, ok := names[id]
		if !ok {
			continue
		}
		item := &v1.MonitorAppUsage{Id: id, Name: name}
		if samples > 0 {
			item.Cpu = roundMetric(cpu[id] / samples)
			item.Memory = math.Round(mem[id] / samples)
		}
		_, item.Running = running[id]
		out = append(out, item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Cpu != out[j].Cpu {
			return out[i].Cpu > out[j].Cpu
		}
		return out[i].Memory > out[j].Memory
	})
	if len(out) > 50 {
		out = out[:50]
	}
	return out
}

// diskUsage 各挂载点当前容量，并按最近 30 天（不足时按已有历史）的已用增长估算写满时间。
func (m *MonitorUsecase) diskUsage(ctx context.Context, now time.Time, latest *sysmon.Sample) []*v1.MonitorDiskUsage {
	if latest == nil || len(latest.Mounts) == 0 {
		return nil
	}
	var base monitorPoint
	var baseAt time.Time
	since := now.Add(-monitorGrowthWindow)
	for tier := tier1h; tier >= tierRaw; tier-- {
		row, err := m.repo.EdgeSample(ctx, tier, since, true)
		if err != nil || row == nil {
			continue
		}
		if baseAt.IsZero() || row.Time.Before(baseAt) {
			base, baseAt = decodePoint(row.Data), row.Time
		}
	}
	out := make([]*v1.MonitorDiskUsage, 0, len(latest.Mounts))
	for _, mt := range latest.Mounts {
		used := latest.Values[sysmon.Key(sysmon.KeyDiskUsed, mt.Mountpoint)]
		total := latest.Values[sysmon.Key(sysmon.KeyDiskSize, mt.Mountpoint)]
		if total <= 0 {
			continue
		}
		d := &v1.MonitorDiskUsage{
			Mountpoint: mt.Mountpoint, Device: mt.Device, Fstype: mt.Fstype,
			Used: uint64(used), Total: uint64(total), FullIn: -1,
		}
		if st, ok := base[sysmon.Key(sysmon.KeyDiskUsed, mt.Mountpoint)]; ok {
			span := now.Sub(baseAt)
			d.GrowthSpan = int64(span / time.Second)
			d.Growth = math.Round(used - st[0])
			// 至少 6 小时的历史、且增长超过容量的 0.1% 才预测；3 年以上写不满视为增长平缓
			if span >= 6*time.Hour && d.Growth > total*0.001 {
				rate := d.Growth / span.Seconds()
				if fullIn := (total - used) / rate; fullIn < 3*365*86400 {
					d.FullIn = int64(fullIn)
				}
			}
		}
		out = append(out, d)
	}
	return out
}

// —— 导出 ——

var csvUnsafe = regexp.MustCompile(`[^\w.-]+`)

// monitorColumnName 把指标键换成可读的 CSV 列名。
func monitorColumnName(key string, apps map[string]string) string {
	prefix, name, sub := strings.Cut(key, ":")
	base := map[string]string{
		sysmon.KeyCPU: "cpu_percent", sysmon.KeyIOWait: "iowait_percent", sysmon.KeyLoad1: "load1",
		sysmon.KeyProcs: "processes", sysmon.KeyCtx: "context_switches_per_s", sysmon.KeyCPUTemp: "cpu_temp_c",
		sysmon.KeyMem: "memory_percent", sysmon.KeyCache: "cache_percent", sysmon.KeyMemUsed: "memory_used_bytes",
		sysmon.KeyMemAvail: "memory_available_bytes", sysmon.KeySwap: "swap_percent", sysmon.KeySwapUsed: "swap_used_bytes",
		sysmon.KeyRx: "net_rx_bytes_per_s", sysmon.KeyTx: "net_tx_bytes_per_s", sysmon.KeyTCP: "tcp_established",
		sysmon.KeyRead: "disk_read_bytes_per_s", sysmon.KeyWrite: "disk_write_bytes_per_s", sysmon.KeyIOPS: "disk_iops",
		sysmon.KeyLatency: "disk_latency_ms", sysmon.KeyDiskUsed: "disk_used_bytes", sysmon.KeyDiskSize: "disk_total_bytes",
		sysmon.KeyGPU: "gpu_percent", sysmon.KeyVRAM: "gpu_memory_percent", sysmon.KeyVRAMUsed: "gpu_memory_used_bytes",
		sysmon.KeyGPUTemp: "gpu_temp_c", sysmon.KeyGPUPower: "gpu_power_w",
		monitorKeyAppCPU: "app_cpu_percent", monitorKeyAppMem: "app_memory_bytes", monitorKeyApps: "apps_running",
	}[prefix]
	if base == "" {
		base = prefix
	}
	if !sub {
		return base
	}
	if prefix == monitorKeyAppCPU || prefix == monitorKeyAppMem {
		if n, ok := apps[name]; ok {
			name = n
		}
	}
	return base + "[" + name + "]"
}

// Export 导出所选时段（按该时段可用的最细精度，最多 2 万行）的全部指标。
func (m *MonitorUsecase) Export(ctx context.Context, req *v1.ExportMonitorDataRequest) (*v1.ExportMonitorDataResponse, error) {
	now := time.Now()
	s := m.currentSettings()
	interval := time.Duration(s.IntervalSeconds) * time.Second
	start, end, err := monitorRange(req.GetStart(), req.GetEnd(), now)
	if err != nil {
		return nil, err
	}
	tier := chooseTier(start, end, now, s.RetentionDays, interval, monitorExportMaxRows)
	rows, err := m.loadRows(ctx, tier, start, end, s.RetentionDays, interval)
	if err != nil {
		return nil, ErrSystem(err)
	}
	keySet := map[string]bool{}
	appIDs := map[string]bool{}
	for _, r := range rows {
		for k := range r.point {
			if k == sysmon.KeyDuration {
				continue
			}
			keySet[k] = true
			if p, id, ok := strings.Cut(k, ":"); ok && (p == monitorKeyAppCPU || p == monitorKeyAppMem) {
				appIDs[id] = true
			}
		}
	}
	keys := make([]string, 0, len(keySet))
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	ids := make([]string, 0, len(appIDs))
	for id := range appIDs {
		ids = append(ids, id)
	}
	apps := map[string]string{}
	if m.instances != nil {
		apps = m.instances.InstanceNames(ctx, ids)
	}

	var buf bytes.Buffer
	buf.WriteString("\uFEFF")
	w := csv.NewWriter(&buf)
	header := []string{"time", "resolution", "samples"}
	for _, k := range keys {
		header = append(header, monitorColumnName(k, apps))
	}
	_ = w.Write(header)
	for _, r := range rows {
		samples := r.point[sysmon.KeyCPU][2]
		if samples == 0 {
			samples = 1
		}
		res := tierNames[tierRaw]
		if samples > 1 {
			res = tierNames[tier]
		}
		line := []string{r.at.Format("2006-01-02 15:04:05"), res, strconv.Itoa(int(samples))}
		for _, k := range keys {
			if st, ok := r.point[k]; ok {
				line = append(line, strconv.FormatFloat(roundMetric(st[0]), 'f', -1, 64))
			} else {
				line = append(line, "")
			}
		}
		_ = w.Write(line)
	}
	w.Flush()
	host := csvUnsafe.ReplaceAllString(m.sampler.Host(ctx).Hostname, "-")
	return &v1.ExportMonitorDataResponse{
		Filename: fmt.Sprintf("momoko-monitor-%s-%s-%s.csv", host, start.Format("20060102-1504"), end.Format("20060102-1504")),
		Content:  buf.String(),
		Rows:     int32(len(rows)),
	}, nil
}
