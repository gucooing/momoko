package sysmon

import (
	"context"
	"math"
	"strconv"
	"sync"
	"time"
)

// GPUInfo 一块显卡的静态信息。
type GPUInfo struct {
	Index       int
	Name        string
	Vendor      string // nvidia / amd / intel / other
	Driver      string
	MemoryTotal uint64
}

// gpuStat 一次显卡采样；未知的数值为 NaN。
type gpuStat struct {
	Index      int
	Util       float64
	MemUsed    float64
	MemTotal   float64
	Temp       float64
	Power      float64
	PowerLimit float64
	Fan        float64
}

func newGPUStat(index int) gpuStat {
	n := math.NaN()
	return gpuStat{Index: index, Util: n, MemUsed: n, MemTotal: n, Temp: n, Power: n, PowerLimit: n, Fan: n}
}

// gpuSource 一种显卡数据来源（nvidia-smi、Linux sysfs、Windows 性能计数器）。
type gpuSource interface {
	// probe 探测可用显卡；不可用时返回错误（静默降级，不打日志）。
	probe(ctx context.Context) ([]GPUInfo, error)
	sample(ctx context.Context) ([]gpuStat, error)
	close()
}

const (
	gpuReprobeInterval = time.Minute
	gpuMaxFailures     = 3
)

// gpuProber 按顺序探测显卡来源；没有显卡或驱动时每分钟重新探测一次，装好驱动后自动开始采集。
type gpuProber struct {
	mu        sync.Mutex
	active    gpuSource
	infos     []GPUInfo
	note      string
	lastProbe time.Time
	failures  int
}

func newGPUProber() *gpuProber {
	return &gpuProber{note: "未检测到显卡"}
}

func (g *gpuProber) devices() ([]GPUInfo, string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	out := append([]GPUInfo(nil), g.infos...)
	if len(out) > 0 {
		return out, ""
	}
	return out, g.note
}

func (g *gpuProber) close() {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.active != nil {
		g.active.close()
		g.active = nil
	}
}

func (g *gpuProber) sample(ctx context.Context, v, x map[string]float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.active == nil {
		if !g.lastProbe.IsZero() && time.Since(g.lastProbe) < gpuReprobeInterval {
			return
		}
		g.lastProbe = time.Now()
		g.probeLocked(ctx)
		if g.active == nil {
			return
		}
	}
	stats, err := g.active.sample(ctx)
	if err != nil {
		g.failures++
		if g.failures >= gpuMaxFailures {
			g.active.close()
			g.active, g.infos, g.failures = nil, nil, 0
			g.note = "显卡数据读取失败，稍后自动重试"
		}
		return
	}
	g.failures = 0
	for _, st := range stats {
		id := strconv.Itoa(st.Index)
		put := func(m map[string]float64, key string, val float64) {
			if !math.IsNaN(val) && !math.IsInf(val, 0) && val >= 0 {
				m[Key(key, id)] = val
			}
		}
		put(v, KeyGPU, clampNaN(st.Util))
		put(v, KeyVRAMUsed, st.MemUsed)
		if st.MemTotal > 0 && !math.IsNaN(st.MemUsed) {
			put(v, KeyVRAM, clampPct(st.MemUsed/st.MemTotal*100))
			put(x, extraVRAMT, st.MemTotal)
		}
		put(v, KeyGPUTemp, st.Temp)
		put(v, KeyGPUPower, st.Power)
		put(x, extraPowerL, st.PowerLimit)
		put(x, extraFan, st.Fan)
	}
}

func (g *gpuProber) probeLocked(ctx context.Context) {
	sources := append([]gpuSource{&nvidiaSource{}}, platformGPUSources()...)
	for _, src := range sources {
		pctx, cancel := context.WithTimeout(ctx, 8*time.Second)
		infos, err := src.probe(pctx)
		cancel()
		if err != nil || len(infos) == 0 {
			src.close()
			continue
		}
		g.active, g.infos, g.note, g.failures = src, infos, "", 0
		return
	}
	g.note = "未检测到显卡"
}

func clampNaN(v float64) float64 {
	if math.IsNaN(v) {
		return v
	}
	return clampPct(v)
}
