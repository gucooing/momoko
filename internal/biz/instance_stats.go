package biz

import (
	"context"
	"io/fs"
	"path/filepath"
	"sync"
	"time"

	gcpu "github.com/shirou/gopsutil/v4/cpu"
	gdisk "github.com/shirou/gopsutil/v4/disk"
	gmem "github.com/shirou/gopsutil/v4/mem"
	gprocess "github.com/shirou/gopsutil/v4/process"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

const (
	// instanceDiskScanTTL 工作目录占用的统计缓存时间（遍历目录较慢，不随每次轮询重算）。
	instanceDiskScanTTL = 2 * time.Minute
	// instanceDiskScanLimit 单次统计最多遍历的条目数，防止超大目录拖慢接口。
	instanceDiskScanLimit = 200000
)

// instanceStatsCache 缓存每个实例的进程句柄（CPU 百分比需要与上次采样对比）与目录占用统计。
type instanceStatsCache struct {
	mu    sync.Mutex
	procs map[string]map[int32]*gprocess.Process // instanceID -> pid -> process
	disk  map[string]*instanceDiskUsage
}

type instanceDiskUsage struct {
	dir      string
	bytes    uint64
	scanTime time.Time
	scanning bool
}

var instanceStats = &instanceStatsCache{
	procs: make(map[string]map[int32]*gprocess.Process),
	disk:  make(map[string]*instanceDiskUsage),
}

// GetInstanceStats 返回实例的资源占用：主进程及其子进程的 CPU/内存、工作目录占用与所在磁盘容量。
func (i *InstanceUsecase) GetInstanceStats(ctx context.Context, userID, instanceID string) (*v1.GetInstanceStatsResponse, error) {
	item, err := i.repo.GetInstanceByUserID(ctx, userID, instanceID)
	if err != nil {
		return nil, i.wrapInstanceRepoErr(err)
	}
	core, err := i.ensureInstance(item)
	if err != nil {
		return nil, err
	}

	resp := &v1.GetInstanceStatsResponse{}
	if cores, err := gcpu.CountsWithContext(ctx, true); err == nil {
		resp.CpuCores = int32(cores)
	}
	if vm, err := gmem.VirtualMemoryWithContext(ctx); err == nil {
		resp.MemoryTotal = vm.Total
	}

	dir := core.Dir()
	if dir != "" {
		if usage, err := gdisk.UsageWithContext(ctx, dir); err == nil {
			resp.DiskTotal = usage.Total
			resp.DiskFree = usage.Free
		}
		if bytes, scanTime, ok := instanceStats.diskUsage(instanceID, dir); ok {
			resp.DiskUsageBytes = bytes
			resp.DiskScanTime = timestamppb.New(scanTime)
		}
	}

	pid := core.PID()
	if !core.Running() || pid <= 0 {
		instanceStats.dropProcs(instanceID)
		return resp, nil
	}
	resp.Running = true
	resp.Pid = int32(pid)
	if startTime, ok := core.StartTime(); ok {
		resp.StartTime = timestamppb.New(startTime)
	}

	procs := instanceStats.processTree(ctx, instanceID, int32(pid))
	resp.ProcessCount = int32(len(procs))
	for _, p := range procs {
		if pct, err := p.PercentWithContext(ctx, 0); err == nil {
			resp.CpuPercent += pct
		}
		if info, err := p.MemoryInfoWithContext(ctx); err == nil && info != nil {
			resp.MemoryBytes += info.RSS
		}
	}
	return resp, nil
}

// processTree 返回主进程及其全部子孙进程，复用上次的句柄以便计算 CPU 百分比。
func (c *instanceStatsCache) processTree(ctx context.Context, instanceID string, root int32) []*gprocess.Process {
	c.mu.Lock()
	defer c.mu.Unlock()

	prev := c.procs[instanceID]
	next := make(map[int32]*gprocess.Process)
	get := func(pid int32) *gprocess.Process {
		if p, ok := prev[pid]; ok {
			return p
		}
		p, err := gprocess.NewProcessWithContext(ctx, pid)
		if err != nil {
			return nil
		}
		// 首次采样建立基线，下一次轮询才有有效 CPU 百分比
		_, _ = p.PercentWithContext(ctx, 0)
		return p
	}

	queue := []int32{root}
	for len(queue) > 0 && len(next) < 512 {
		pid := queue[0]
		queue = queue[1:]
		if _, seen := next[pid]; seen {
			continue
		}
		p := get(pid)
		if p == nil {
			continue
		}
		next[pid] = p
		if children, err := p.ChildrenWithContext(ctx); err == nil {
			for _, child := range children {
				queue = append(queue, child.Pid)
			}
		}
	}
	c.procs[instanceID] = next

	list := make([]*gprocess.Process, 0, len(next))
	for _, p := range next {
		list = append(list, p)
	}
	return list
}

func (c *instanceStatsCache) dropProcs(instanceID string) {
	c.mu.Lock()
	delete(c.procs, instanceID)
	c.mu.Unlock()
}

// diskUsage 返回缓存的目录占用；缓存过期或目录变更时在后台重新统计。
func (c *instanceStatsCache) diskUsage(instanceID, dir string) (uint64, time.Time, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := c.disk[instanceID]
	if entry == nil || entry.dir != dir {
		entry = &instanceDiskUsage{dir: dir}
		c.disk[instanceID] = entry
	}
	if !entry.scanning && time.Since(entry.scanTime) > instanceDiskScanTTL {
		entry.scanning = true
		go c.scanDisk(instanceID, entry)
	}
	if entry.scanTime.IsZero() {
		return 0, time.Time{}, false
	}
	return entry.bytes, entry.scanTime, true
}

func (c *instanceStatsCache) scanDisk(instanceID string, entry *instanceDiskUsage) {
	var total uint64
	count := 0
	_ = filepath.WalkDir(entry.dir, func(_ string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		count++
		if count > instanceDiskScanLimit {
			return filepath.SkipAll
		}
		if d.Type().IsRegular() {
			if info, err := d.Info(); err == nil {
				total += uint64(info.Size())
			}
		}
		return nil
	})

	c.mu.Lock()
	defer c.mu.Unlock()
	entry.bytes = total
	entry.scanTime = time.Now()
	entry.scanning = false
	if cur := c.disk[instanceID]; cur != entry {
		// 期间目录已变更，丢弃本次结果由下一次轮询重新统计
		return
	}
}
