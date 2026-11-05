// Package sysmon 为持久性能监控采集整机指标：CPU / 内存 / 磁盘 / 网络 / GPU。
// 与 pkg/sysinfo（工作台实时图）互不共享计数器状态，两边各自计算速率，互不干扰。
package sysmon

import (
	"context"
	"math"
	"os"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	gcpu "github.com/shirou/gopsutil/v4/cpu"
	gdisk "github.com/shirou/gopsutil/v4/disk"
	ghost "github.com/shirou/gopsutil/v4/host"
	gload "github.com/shirou/gopsutil/v4/load"
	gmem "github.com/shirou/gopsutil/v4/mem"
	gnet "github.com/shirou/gopsutil/v4/net"
	gprocess "github.com/shirou/gopsutil/v4/process"
	gsensors "github.com/shirou/gopsutil/v4/sensors"
)

// Options 决定本次采集哪些可选项目（CPU 与内存始终采集）。
type Options struct {
	GPU     bool
	Disk    bool
	Network bool
}

// Mount 一个挂载点的元信息（容量写在 Sample.Values 的 du:/dt: 键里）。
type Mount struct {
	Mountpoint string
	Device     string
	Fstype     string
}

// Sample 一次采样。
//   - Values：写入历史的指标（键 → 值），见包内 Key* 约定；
//   - Extra：只在「当前值」里展示、不写历史的指标（如 5/15 分钟负载、显卡功耗上限）。
type Sample struct {
	Time   time.Time
	Values map[string]float64
	Extra  map[string]float64
	Mounts []Mount
}

// Host 主机静态信息（启动后缓存，显卡列表随探测结果变化）。
type Host struct {
	Hostname    string
	OS          string
	CPUModel    string
	CPUCores    int
	CPUThreads  int
	MemoryTotal uint64
	SwapTotal   uint64
	Linux       bool
}

type ioCounter struct {
	a, b   uint64 // 接收/读取字节、发送/写入字节
	ops    uint64 // 读写次数（磁盘）
	busyMs uint64 // 读写累计耗时（毫秒，仅 Linux 可靠）
}

// Sampler 有状态的采集器：速率类指标与上一次采样做差。并发安全。
type Sampler struct {
	mu sync.Mutex

	prevAt   time.Time
	prevCPU  *gcpu.TimesStat
	prevNet  map[string]ioCounter
	prevDisk map[string]ioCounter
	prevCtxt uint64

	tempFails int
	tempNext  time.Time

	host     *Host
	hostOnce sync.Once

	gpu *gpuProber
}

// NewSampler 创建采集器。
func NewSampler() *Sampler {
	return &Sampler{gpu: newGPUProber()}
}

// Close 释放 GPU 探测占用的资源。
func (s *Sampler) Close() {
	s.gpu.close()
}

// Host 返回主机静态信息。
func (s *Sampler) Host(ctx context.Context) Host {
	s.hostOnce.Do(func() {
		h := &Host{Linux: runtime.GOOS == "linux"}
		if info, err := ghost.InfoWithContext(ctx); err == nil {
			h.Hostname = info.Hostname
			h.OS = strings.TrimSpace(info.Platform + " " + info.PlatformVersion)
		}
		if h.Hostname == "" {
			h.Hostname, _ = os.Hostname()
		}
		if infos, err := gcpu.InfoWithContext(ctx); err == nil && len(infos) > 0 {
			h.CPUModel = strings.TrimSpace(infos[0].ModelName)
		}
		h.CPUCores, _ = gcpu.CountsWithContext(ctx, false)
		h.CPUThreads, _ = gcpu.CountsWithContext(ctx, true)
		s.host = h
	})
	h := *s.host
	if vm, err := gmem.VirtualMemoryWithContext(ctx); err == nil {
		h.MemoryTotal = vm.Total
	}
	if sw, err := gmem.SwapMemoryWithContext(ctx); err == nil {
		h.SwapTotal = sw.Total
	}
	return h
}

// GPUs 返回最近一次探测到的显卡与未检测到时的说明。
func (s *Sampler) GPUs() ([]GPUInfo, string) {
	return s.gpu.devices()
}

// Sample 采集一次。at 为本次采样槽时间（写库用），速率按真实耗时计算。
func (s *Sampler) Sample(ctx context.Context, at time.Time, opts Options) *Sample {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	dt := 0.0
	if !s.prevAt.IsZero() {
		dt = now.Sub(s.prevAt).Seconds()
	}
	out := &Sample{Time: at, Values: make(map[string]float64, 48), Extra: make(map[string]float64, 16)}
	v, x := out.Values, out.Extra

	s.sampleCPU(ctx, v, x, dt)
	sampleMemory(ctx, v, x)
	if opts.Network {
		s.sampleNetwork(ctx, v, dt)
	} else {
		s.prevNet = nil
	}
	if opts.Disk {
		out.Mounts = s.sampleDisk(ctx, v, dt)
	} else {
		s.prevDisk = nil
	}
	if opts.GPU {
		s.gpu.sample(ctx, v, x)
	}
	if dt > 0 {
		v[KeyDuration] = dt
	}
	s.prevAt = now
	return out
}

// 指标键约定（键:名称 表示按网卡 / 设备 / 挂载点 / 显卡序号细分）。
const (
	KeyCPU      = "cpu"    // CPU 使用率 %
	KeyIOWait   = "iow"    // I/O 等待 %（Linux）
	KeyLoad1    = "load1"  // 1 分钟负载
	KeyProcs    = "procs"  // 进程数
	KeyCtx      = "ctx"    // 上下文切换 /s（Linux）
	KeyCPUTemp  = "ctemp"  // CPU 温度 °C
	KeyMem      = "mem"    // 内存已用 %
	KeyCache    = "cache"  // 缓存 %（Linux）
	KeyMemUsed  = "memu"   // 已用字节
	KeyMemAvail = "mema"   // 可用字节
	KeySwap     = "swap"   // 交换已用 %
	KeySwapUsed = "swapu"  // 交换已用字节
	KeyRx       = "rx"     // 接收 B/s（rx:网卡）
	KeyTx       = "tx"     // 发送 B/s（tx:网卡）
	KeyTCP      = "tcp"    // 已建立的 TCP 连接数
	KeyRead     = "rd"     // 读取 B/s（rd:设备）
	KeyWrite    = "wr"     // 写入 B/s（wr:设备）
	KeyIOPS     = "iops"   // 读写次数 /s（iops:设备）
	KeyLatency  = "lat"    // 平均读写延迟 ms（Linux，lat:设备）
	KeyDiskUsed = "du"     // du:挂载点 已用字节
	KeyDiskSize = "dt"     // dt:挂载点 总字节
	KeyGPU      = "gpu"    // gpu:序号 使用率 %
	KeyVRAM     = "vram"   // vram:序号 显存 %
	KeyVRAMUsed = "vramu"  // vramu:序号 显存已用字节
	KeyGPUTemp  = "gtemp"  // gtemp:序号 温度 °C
	KeyGPUPower = "gpow"   // gpow:序号 功耗 W
	KeyDuration = "dur"    // 与上一次采样的间隔秒数（用于把速率换算成时段累计量）
	extraLoad5  = "load5"  // 仅当前值
	extraLoad15 = "load15" // 仅当前值
	extraSwapT  = "swapt"  // 仅当前值：交换总字节
	extraPowerL = "gpowl"  // 仅当前值：gpowl:序号 功耗上限 W
	extraFan    = "gfan"   // 仅当前值：gfan:序号 风扇 %
	extraVRAMT  = "vramt"  // 仅当前值：vramt:序号 显存总字节
)

// Key 拼出细分指标键。
func Key(prefix, name string) string { return prefix + ":" + name }

func (s *Sampler) sampleCPU(ctx context.Context, v, x map[string]float64, dt float64) {
	if times, err := gcpu.TimesWithContext(ctx, false); err == nil && len(times) > 0 {
		cur := times[0]
		if s.prevCPU != nil {
			total, busy, iow := cpuDelta(*s.prevCPU, cur)
			if total > 0 {
				v[KeyCPU] = clampPct(busy / total * 100)
				if runtime.GOOS == "linux" {
					v[KeyIOWait] = clampPct(iow / total * 100)
				}
			}
		}
		s.prevCPU = &cur
	}
	if avg, err := gload.AvgWithContext(ctx); err == nil && avg != nil {
		v[KeyLoad1] = avg.Load1
		x[extraLoad5] = avg.Load5
		x[extraLoad15] = avg.Load15
	}
	if pids, err := gprocess.PidsWithContext(ctx); err == nil {
		v[KeyProcs] = float64(len(pids))
	}
	if runtime.GOOS == "linux" {
		if misc, err := gload.MiscWithContext(ctx); err == nil && misc != nil {
			cur := uint64(misc.Ctxt)
			if s.prevCtxt > 0 && cur >= s.prevCtxt && dt > 0 {
				v[KeyCtx] = float64(cur-s.prevCtxt) / dt
			}
			s.prevCtxt = cur
		}
	}
	if t, ok := s.cpuTemperature(ctx); ok {
		v[KeyCPUTemp] = t
	}
}

func cpuDelta(a, b gcpu.TimesStat) (total, busy, iowait float64) {
	sum := func(t gcpu.TimesStat) float64 {
		return t.User + t.System + t.Idle + t.Nice + t.Iowait + t.Irq + t.Softirq + t.Steal
	}
	total = sum(b) - sum(a)
	idle := (b.Idle - a.Idle) + (b.Iowait - a.Iowait)
	busy = total - idle
	iowait = b.Iowait - a.Iowait
	if busy < 0 {
		busy = 0
	}
	if iowait < 0 {
		iowait = 0
	}
	return total, busy, iowait
}

// cpuTemperature 读 CPU 封装温度；读不到时退避（Windows 上通常需要管理员权限，失败 3 次后不再尝试）。
func (s *Sampler) cpuTemperature(ctx context.Context) (float64, bool) {
	if s.tempFails >= 3 || time.Now().Before(s.tempNext) {
		return 0, false
	}
	temps, err := gsensors.TemperaturesWithContext(ctx)
	if len(temps) == 0 {
		s.tempFails++
		s.tempNext = time.Now().Add(time.Minute)
		_ = err
		return 0, false
	}
	best, rank := 0.0, -1
	for _, t := range temps {
		k := strings.ToLower(t.SensorKey)
		if t.Temperature <= 0 || t.Temperature > 150 {
			continue
		}
		r := 0
		switch {
		case strings.Contains(k, "package") || strings.Contains(k, "tctl") || strings.Contains(k, "tdie") || strings.Contains(k, "x86_pkg"):
			r = 4
		case strings.Contains(k, "coretemp") || strings.Contains(k, "k10temp") || strings.Contains(k, "zenpower") || strings.Contains(k, "cpu"):
			r = 3
		case strings.Contains(k, "soc") || strings.Contains(k, "thermal"):
			r = 2
		case strings.Contains(k, "acpitz") || strings.Contains(k, "thermalzone"):
			r = 1
		default:
			continue
		}
		if r > rank || (r == rank && t.Temperature > best) {
			best, rank = t.Temperature, r
		}
	}
	if rank < 0 {
		s.tempFails++
		s.tempNext = time.Now().Add(time.Minute)
		return 0, false
	}
	s.tempFails = 0
	return best, true
}

func sampleMemory(ctx context.Context, v, x map[string]float64) {
	if vm, err := gmem.VirtualMemoryWithContext(ctx); err == nil && vm.Total > 0 {
		v[KeyMem] = clampPct(vm.UsedPercent)
		v[KeyMemUsed] = float64(vm.Used)
		v[KeyMemAvail] = float64(vm.Available)
		if runtime.GOOS == "linux" {
			cache := vm.Cached + vm.Buffers + vm.Sreclaimable
			v[KeyCache] = clampPct(float64(cache) / float64(vm.Total) * 100)
		}
	}
	if sw, err := gmem.SwapMemoryWithContext(ctx); err == nil && sw.Total > 0 {
		v[KeySwap] = clampPct(sw.UsedPercent)
		v[KeySwapUsed] = float64(sw.Used)
		x[extraSwapT] = float64(sw.Total)
	}
}

// sampleNetwork 按网卡统计收发速率：去掉回环、未启用与 veth*（容器一端，流量已在网桥上计过）。
func (s *Sampler) sampleNetwork(ctx context.Context, v map[string]float64, dt float64) {
	counters, err := gnet.IOCountersWithContext(ctx, true)
	if err != nil {
		return
	}
	include := map[string]bool{}
	if ifaces, err := gnet.InterfacesWithContext(ctx); err == nil {
		for _, it := range ifaces {
			up, loop := false, false
			for _, f := range it.Flags {
				switch strings.ToLower(f) {
				case "up":
					up = true
				case "loopback":
					loop = true
				}
			}
			include[it.Name] = up && !loop
		}
	}
	next := make(map[string]ioCounter, len(counters))
	var rx, tx float64
	for _, c := range counters {
		name := c.Name
		lower := strings.ToLower(name)
		if lower == "lo" || strings.HasPrefix(lower, "veth") || strings.Contains(lower, "loopback") {
			continue
		}
		if ok, known := include[name]; known && !ok {
			continue
		}
		if c.BytesRecv == 0 && c.BytesSent == 0 {
			continue
		}
		cur := ioCounter{a: c.BytesRecv, b: c.BytesSent}
		next[name] = cur
		prev, had := s.prevNet[name]
		if !had || dt <= 0 {
			continue
		}
		r := rate(prev.a, cur.a, dt)
		t := rate(prev.b, cur.b, dt)
		v[Key(KeyRx, name)] = r
		v[Key(KeyTx, name)] = t
		rx += r
		tx += t
	}
	if s.prevNet != nil && dt > 0 {
		v[KeyRx] = rx
		v[KeyTx] = tx
	}
	s.prevNet = next
	if n, ok := tcpEstablished(ctx); ok {
		v[KeyTCP] = float64(n)
	}
}

// sampleDisk 采集磁盘读写速率（按物理设备）与各挂载点容量。
func (s *Sampler) sampleDisk(ctx context.Context, v map[string]float64, dt float64) []Mount {
	if counters, err := gdisk.IOCountersWithContext(ctx); err == nil {
		next := make(map[string]ioCounter, len(counters))
		var rd, wr, ops, busy, opsAll float64
		for name, c := range counters {
			if !isPhysicalDisk(name) {
				continue
			}
			cur := ioCounter{a: c.ReadBytes, b: c.WriteBytes, ops: c.ReadCount + c.WriteCount, busyMs: c.ReadTime + c.WriteTime}
			next[name] = cur
			prev, had := s.prevDisk[name]
			if !had || dt <= 0 {
				continue
			}
			r := rate(prev.a, cur.a, dt)
			w := rate(prev.b, cur.b, dt)
			o := rate(prev.ops, cur.ops, dt)
			v[Key(KeyRead, name)] = r
			v[Key(KeyWrite, name)] = w
			v[Key(KeyIOPS, name)] = o
			rd += r
			wr += w
			ops += o
			if runtime.GOOS == "linux" && cur.ops > prev.ops && cur.busyMs >= prev.busyMs {
				n := float64(cur.ops - prev.ops)
				b := float64(cur.busyMs - prev.busyMs)
				v[Key(KeyLatency, name)] = b / n
				busy += b
				opsAll += n
			}
		}
		if s.prevDisk != nil && dt > 0 {
			v[KeyRead] = rd
			v[KeyWrite] = wr
			v[KeyIOPS] = ops
			if opsAll > 0 {
				v[KeyLatency] = busy / opsAll
			}
		}
		s.prevDisk = next
	}
	return sampleMounts(ctx, v)
}

var skipFstypes = map[string]bool{
	"squashfs": true, "overlay": true, "tmpfs": true, "devtmpfs": true, "iso9660": true, "udf": true,
	"cdfs": true, "ramfs": true, "nsfs": true, "fuse.lxcfs": true,
}

func sampleMounts(ctx context.Context, v map[string]float64) []Mount {
	parts, err := gdisk.PartitionsWithContext(ctx, false)
	if err != nil {
		return nil
	}
	sort.Slice(parts, func(i, j int) bool { return len(parts[i].Mountpoint) < len(parts[j].Mountpoint) })
	seen := map[string]bool{}
	out := make([]Mount, 0, len(parts))
	for _, p := range parts {
		if skipFstypes[strings.ToLower(p.Fstype)] || strings.HasPrefix(p.Mountpoint, "/snap/") {
			continue
		}
		// 同一设备的多次挂载（bind mount、容器卷）只保留最短的挂载点
		if p.Device != "" && seen[p.Device] {
			continue
		}
		usage, err := gdisk.UsageWithContext(ctx, p.Mountpoint)
		if err != nil || usage.Total == 0 {
			continue
		}
		seen[p.Device] = true
		v[Key(KeyDiskUsed, p.Mountpoint)] = float64(usage.Used)
		v[Key(KeyDiskSize, p.Mountpoint)] = float64(usage.Total)
		out = append(out, Mount{Mountpoint: p.Mountpoint, Device: p.Device, Fstype: p.Fstype})
		if len(out) >= 12 {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Mountpoint < out[j].Mountpoint })
	return out
}

func rate(prev, cur uint64, dt float64) float64 {
	if cur < prev || dt <= 0 {
		return 0
	}
	return float64(cur-prev) / dt
}

func clampPct(v float64) float64 {
	if math.IsNaN(v) || v < 0 {
		return 0
	}
	if v > 100 {
		return 100
	}
	return v
}
