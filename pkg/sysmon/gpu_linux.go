//go:build linux

package sysmon

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

func platformGPUSources() []gpuSource {
	return []gpuSource{&sysfsSource{}}
}

var drmCardRe = regexp.MustCompile(`^card[0-9]+$`)

type sysfsCard struct {
	info GPUInfo
	dir  string // /sys/class/drm/cardN
	dev  string // /sys/class/drm/cardN/device
	// Intel：RC6 / gtidle 空闲累计毫秒，使用率 = 1 - Δ空闲 / Δ时间
	idleFile string
	prevIdle float64
	prevAt   time.Time
}

// sysfsSource 直接读内核驱动的 sysfs：AMD（amdgpu：gpu_busy_percent / 显存 / hwmon）与
// Intel 核显（i915 的 RC6 驻留时间、xe 的 gtidle 驻留时间），不依赖额外工具。
type sysfsSource struct {
	cards []*sysfsCard
}

func readTrim(path string) string {
	b, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(b))
}

func readFloat(path string) float64 {
	v, err := strconv.ParseFloat(readTrim(path), 64)
	if err != nil {
		return math.NaN()
	}
	return v
}

func (s *sysfsSource) probe(ctx context.Context) ([]GPUInfo, error) {
	entries, err := os.ReadDir("/sys/class/drm")
	if err != nil {
		return nil, err
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if drmCardRe.MatchString(e.Name()) {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	amdNames := rocmProductNames(ctx)
	for _, name := range names {
		dir := filepath.Join("/sys/class/drm", name)
		dev := filepath.Join(dir, "device")
		vendor := readTrim(filepath.Join(dev, "vendor"))
		card := &sysfsCard{dir: dir, dev: dev}
		driver := ""
		if link, err := os.Readlink(filepath.Join(dev, "driver")); err == nil {
			driver = filepath.Base(link)
		}
		card.info.Driver = driver
		switch vendor {
		case "0x1002":
			if _, err := os.Stat(filepath.Join(dev, "gpu_busy_percent")); err != nil {
				continue
			}
			card.info.Vendor = "amd"
			card.info.Name = amdNames[name]
			if v := readFloat(filepath.Join(dev, "mem_info_vram_total")); v > 0 {
				card.info.MemoryTotal = uint64(v)
			}
		case "0x8086":
			for _, f := range []string{
				filepath.Join(dir, "gt", "gt0", "rc6_residency_ms"),
				filepath.Join(dir, "power", "rc6_residency_ms"),
				filepath.Join(dev, "tile0", "gt0", "gtidle", "idle_residency_ms"),
			} {
				if _, err := os.Stat(f); err == nil {
					card.idleFile = f
					break
				}
			}
			if card.idleFile == "" {
				continue
			}
			card.info.Vendor = "intel"
			card.prevIdle = readFloat(card.idleFile)
			card.prevAt = time.Now()
		default:
			continue
		}
		if card.info.Name == "" {
			card.info.Name = lspciName(ctx, dev)
		}
		if card.info.Name == "" {
			if card.info.Vendor == "amd" {
				card.info.Name = "AMD Radeon " + readTrim(filepath.Join(dev, "device"))
			} else {
				card.info.Name = "Intel Graphics " + readTrim(filepath.Join(dev, "device"))
			}
		}
		card.info.Index = len(s.cards)
		s.cards = append(s.cards, card)
	}
	if len(s.cards) == 0 {
		return nil, errors.New("no sysfs gpu")
	}
	infos := make([]GPUInfo, 0, len(s.cards))
	for _, c := range s.cards {
		infos = append(infos, c.info)
	}
	return infos, nil
}

func (s *sysfsSource) sample(context.Context) ([]gpuStat, error) {
	out := make([]gpuStat, 0, len(s.cards))
	for _, c := range s.cards {
		st := newGPUStat(c.info.Index)
		switch c.info.Vendor {
		case "amd":
			st.Util = readFloat(filepath.Join(c.dev, "gpu_busy_percent"))
			st.MemUsed = readFloat(filepath.Join(c.dev, "mem_info_vram_used"))
			st.MemTotal = readFloat(filepath.Join(c.dev, "mem_info_vram_total"))
		case "intel":
			now := time.Now()
			idle := readFloat(c.idleFile)
			if !math.IsNaN(idle) && !c.prevAt.IsZero() {
				wall := float64(now.Sub(c.prevAt).Milliseconds())
				if wall > 0 && idle >= c.prevIdle {
					st.Util = clampPct((1 - (idle-c.prevIdle)/wall) * 100)
				}
			}
			c.prevIdle, c.prevAt = idle, now
		}
		readHwmon(c.dev, &st)
		out = append(out, st)
	}
	return out, nil
}

func (s *sysfsSource) close() { s.cards = nil }

// readHwmon 读显卡 hwmon：温度（毫摄氏度）、功耗（微瓦）、风扇（PWM 占空比）。
func readHwmon(dev string, st *gpuStat) {
	dirs, _ := filepath.Glob(filepath.Join(dev, "hwmon", "hwmon*"))
	for _, h := range dirs {
		if v := readFloat(filepath.Join(h, "temp1_input")); !math.IsNaN(v) {
			st.Temp = v / 1000
		}
		for _, f := range []string{"power1_average", "power1_input"} {
			if v := readFloat(filepath.Join(h, f)); !math.IsNaN(v) {
				st.Power = v / 1e6
				break
			}
		}
		if v := readFloat(filepath.Join(h, "power1_cap")); !math.IsNaN(v) && v > 0 {
			st.PowerLimit = v / 1e6
		}
		pwm := readFloat(filepath.Join(h, "pwm1"))
		pwmMax := readFloat(filepath.Join(h, "pwm1_max"))
		if math.IsNaN(pwmMax) {
			pwmMax = 255
		}
		if !math.IsNaN(pwm) && pwmMax > 0 {
			st.Fan = clampPct(pwm / pwmMax * 100)
		}
	}
}

// rocmProductNames 有 rocm-smi 时读 AMD 显卡型号（card0 → 名称），没有就返回空表。
func rocmProductNames(ctx context.Context) map[string]string {
	out := map[string]string{}
	bin, err := exec.LookPath("rocm-smi")
	if err != nil {
		return out
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	raw, err := exec.CommandContext(cctx, bin, "--showproductname", "--json").Output()
	if err != nil {
		return out
	}
	var parsed map[string]map[string]string
	if json.Unmarshal(raw, &parsed) != nil {
		return out
	}
	for card, fields := range parsed {
		for _, k := range []string{"Card Series", "Card series", "Card SKU", "Card model", "Card Model"} {
			if v := strings.TrimSpace(fields[k]); v != "" && v != "N/A" {
				out[card] = v
				break
			}
		}
	}
	return out
}

// lspciName 用 lspci -mm 取 PCI 设备名（第 4 个引号字段），没有 lspci 时返回空。
func lspciName(ctx context.Context, dev string) string {
	link, err := os.Readlink(dev)
	if err != nil {
		return ""
	}
	addr := filepath.Base(link)
	bin, err := exec.LookPath("lspci")
	if err != nil {
		return ""
	}
	cctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	raw, err := exec.CommandContext(cctx, bin, "-mm", "-s", addr).Output()
	if err != nil {
		return ""
	}
	quoted := regexp.MustCompile(`"([^"]*)"`).FindAllStringSubmatch(string(raw), -1)
	if len(quoted) >= 3 {
		return strings.TrimSpace(quoted[2][1])
	}
	return ""
}
