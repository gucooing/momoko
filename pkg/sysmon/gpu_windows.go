//go:build windows && (amd64 || arm64)

package sysmon

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sort"
	"strconv"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

func platformGPUSources() []gpuSource {
	return []gpuSource{&pdhSource{}}
}

var (
	modPdh                           = windows.NewLazySystemDLL("pdh.dll")
	procPdhOpenQueryW                = modPdh.NewProc("PdhOpenQueryW")
	procPdhAddEnglishCounterW        = modPdh.NewProc("PdhAddEnglishCounterW")
	procPdhCollectQueryData          = modPdh.NewProc("PdhCollectQueryData")
	procPdhGetFormattedCounterArrayW = modPdh.NewProc("PdhGetFormattedCounterArrayW")
	procPdhCloseQuery                = modPdh.NewProc("PdhCloseQuery")
)

const (
	pdhFmtDouble   = 0x00000200
	pdhFmtNoCap100 = 0x00008000
	pdhMoreData    = 0x800007D2
	pdhNoData      = 0x800007D5
)

// pdhItem 对应 PDH_FMT_COUNTERVALUE_ITEM_W（64 位布局：名称指针 + CStatus + 对齐 + double）。
type pdhItem struct {
	name    *uint16
	cstatus uint32
	_       uint32
	value   float64
}

type pdhAdapter struct {
	info      GPUInfo
	luid      uint64
	dedicated uint64 // 注册表里的专用显存大小
}

// pdhSource 用 Windows 性能计数器（与任务管理器同源）读取任意厂商显卡的使用率与显存：
// \GPU Engine(*)\Utilization Percentage 按适配器 LUID 汇总（取各引擎里最忙的一个），
// \GPU Adapter Memory(*)\Dedicated / Shared Usage 为显存占用；型号与专用显存大小来自 DirectX 注册表。
type pdhSource struct {
	query     uintptr
	engine    uintptr
	dedicated uintptr
	shared    uintptr
	adapters  []*pdhAdapter
}

var luidRe = regexp.MustCompile(`luid_(0x[0-9A-Fa-f]+)_(0x[0-9A-Fa-f]+)_phys_(\d+)(?:_eng_(\d+))?`)

func parseLUID(instance string) (uint64, string, bool) {
	m := luidRe.FindStringSubmatch(instance)
	if m == nil {
		return 0, "", false
	}
	hi, err1 := strconv.ParseUint(m[1][2:], 16, 32)
	lo, err2 := strconv.ParseUint(m[2][2:], 16, 32)
	if err1 != nil || err2 != nil {
		return 0, "", false
	}
	return hi<<32 | lo, m[3] + "/" + m[4], true
}

func (p *pdhSource) addCounter(path string) (uintptr, error) {
	ptr, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, err
	}
	var h uintptr
	r, _, _ := procPdhAddEnglishCounterW.Call(p.query, uintptr(unsafe.Pointer(ptr)), 0, uintptr(unsafe.Pointer(&h)))
	if r != 0 {
		return 0, fmt.Errorf("PdhAddEnglishCounter %s: 0x%x", path, r)
	}
	return h, nil
}

func (p *pdhSource) collect() error {
	r, _, _ := procPdhCollectQueryData.Call(p.query)
	if r != 0 {
		return fmt.Errorf("PdhCollectQueryData: 0x%x", r)
	}
	return nil
}

// values 读一个通配计数器的全部实例（实例名 → 值），无效数据的实例跳过。
func values(counter uintptr) (map[string]float64, error) {
	var size, count uint32
	r, _, _ := procPdhGetFormattedCounterArrayW.Call(counter, pdhFmtDouble|pdhFmtNoCap100,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), 0)
	if uint32(r) == pdhNoData || (r == 0 && count == 0) {
		return map[string]float64{}, nil
	}
	if uint32(r) != pdhMoreData || size == 0 {
		return nil, fmt.Errorf("PdhGetFormattedCounterArray: 0x%x", r)
	}
	buf := make([]uint64, (size+7)/8)
	r, _, _ = procPdhGetFormattedCounterArrayW.Call(counter, pdhFmtDouble|pdhFmtNoCap100,
		uintptr(unsafe.Pointer(&size)), uintptr(unsafe.Pointer(&count)), uintptr(unsafe.Pointer(&buf[0])))
	if r != 0 {
		return nil, fmt.Errorf("PdhGetFormattedCounterArray: 0x%x", r)
	}
	items := unsafe.Slice((*pdhItem)(unsafe.Pointer(&buf[0])), count)
	out := make(map[string]float64, count)
	for _, it := range items {
		if it.cstatus > 1 || it.name == nil {
			continue
		}
		out[windows.UTF16PtrToString(it.name)] += it.value
	}
	return out, nil
}

type dxAdapter struct {
	name      string
	vendor    uint32
	dedicated uint64
	driver    string
}

// dxAdapters 读 HKLM\SOFTWARE\Microsoft\DirectX 下的适配器（LUID → 型号 / 厂商 / 专用显存 / 驱动版本）。
func dxAdapters() map[uint64]dxAdapter {
	out := map[uint64]dxAdapter{}
	root, err := registry.OpenKey(registry.LOCAL_MACHINE, `SOFTWARE\Microsoft\DirectX`, registry.ENUMERATE_SUB_KEYS|registry.READ)
	if err != nil {
		return out
	}
	defer root.Close()
	names, err := root.ReadSubKeyNames(-1)
	if err != nil {
		return out
	}
	for _, sub := range names {
		k, err := registry.OpenKey(root, sub, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		luid, _, err := k.GetIntegerValue("AdapterLuid")
		if err == nil {
			a := dxAdapter{}
			a.name, _, _ = k.GetStringValue("Description")
			if v, _, err := k.GetIntegerValue("VendorId"); err == nil {
				a.vendor = uint32(v)
			}
			if v, _, err := k.GetIntegerValue("DedicatedVideoMemory"); err == nil {
				a.dedicated = v
			}
			if v, _, err := k.GetIntegerValue("DriverVersion"); err == nil && v > 0 {
				a.driver = fmt.Sprintf("%d.%d.%d.%d", (v>>48)&0xffff, (v>>32)&0xffff, (v>>16)&0xffff, v&0xffff)
			}
			out[luid] = a
		}
		k.Close()
	}
	return out
}

func vendorName(id uint32) string {
	switch id {
	case 0x10DE:
		return "nvidia"
	case 0x1002, 0x1022:
		return "amd"
	case 0x8086:
		return "intel"
	}
	return "other"
}

func (p *pdhSource) probe(context.Context) ([]GPUInfo, error) {
	if err := modPdh.Load(); err != nil {
		return nil, err
	}
	r, _, _ := procPdhOpenQueryW.Call(0, 0, uintptr(unsafe.Pointer(&p.query)))
	if r != 0 {
		return nil, fmt.Errorf("PdhOpenQuery: 0x%x", r)
	}
	var err error
	if p.engine, err = p.addCounter(`\GPU Engine(*)\Utilization Percentage`); err != nil {
		return nil, err
	}
	p.dedicated, _ = p.addCounter(`\GPU Adapter Memory(*)\Dedicated Usage`)
	p.shared, _ = p.addCounter(`\GPU Adapter Memory(*)\Shared Usage`)
	if err := p.collect(); err != nil {
		return nil, err
	}
	time.Sleep(300 * time.Millisecond)
	if err := p.collect(); err != nil {
		return nil, err
	}
	eng, err := values(p.engine)
	if err != nil {
		return nil, err
	}
	luids := map[uint64]bool{}
	for inst := range eng {
		if luid, _, ok := parseLUID(inst); ok {
			luids[luid] = true
		}
	}
	dx := dxAdapters()
	for luid := range luids {
		a, ok := dx[luid]
		if !ok || a.vendor == 0x1414 { // Microsoft Basic Render Driver 等虚拟适配器
			continue
		}
		p.adapters = append(p.adapters, &pdhAdapter{
			info:      GPUInfo{Name: a.name, Vendor: vendorName(a.vendor), Driver: a.driver},
			luid:      luid,
			dedicated: a.dedicated,
		})
	}
	if len(p.adapters) == 0 {
		return nil, errors.New("no gpu adapter")
	}
	// 独显（专用显存大的）排前面
	sort.Slice(p.adapters, func(i, j int) bool {
		if p.adapters[i].dedicated != p.adapters[j].dedicated {
			return p.adapters[i].dedicated > p.adapters[j].dedicated
		}
		return p.adapters[i].luid < p.adapters[j].luid
	})
	infos := make([]GPUInfo, 0, len(p.adapters))
	for i, a := range p.adapters {
		a.info.Index = i
		if a.dedicated >= 512<<20 {
			a.info.MemoryTotal = a.dedicated
		}
		infos = append(infos, a.info)
	}
	return infos, nil
}

func (p *pdhSource) sample(context.Context) ([]gpuStat, error) {
	if err := p.collect(); err != nil {
		return nil, err
	}
	eng, err := values(p.engine)
	if err != nil {
		return nil, err
	}
	// 每个引擎（luid + phys + eng）把各进程的占用相加，适配器使用率取最忙的引擎（与任务管理器一致）
	busy := map[uint64]map[string]float64{}
	for inst, v := range eng {
		luid, engine, ok := parseLUID(inst)
		if !ok {
			continue
		}
		if busy[luid] == nil {
			busy[luid] = map[string]float64{}
		}
		busy[luid][engine] += v
	}
	sumBy := func(counter uintptr) map[uint64]float64 {
		out := map[uint64]float64{}
		if counter == 0 {
			return out
		}
		vals, err := values(counter)
		if err != nil {
			return out
		}
		for inst, v := range vals {
			if luid, _, ok := parseLUID(inst); ok {
				out[luid] += v
			}
		}
		return out
	}
	ded, shr := sumBy(p.dedicated), sumBy(p.shared)
	out := make([]gpuStat, 0, len(p.adapters))
	for _, a := range p.adapters {
		st := newGPUStat(a.info.Index)
		util := 0.0
		for _, v := range busy[a.luid] {
			util = math.Max(util, v)
		}
		st.Util = clampPct(util)
		if a.info.MemoryTotal > 0 {
			st.MemUsed = ded[a.luid]
			st.MemTotal = float64(a.info.MemoryTotal)
		} else if v := ded[a.luid] + shr[a.luid]; v > 0 {
			st.MemUsed = v
		}
		out = append(out, st)
	}
	return out, nil
}

func (p *pdhSource) close() {
	if p.query != 0 {
		_, _, _ = procPdhCloseQuery.Call(p.query)
		p.query = 0
	}
	p.adapters = nil
}
