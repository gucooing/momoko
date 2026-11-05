package sysmon

import (
	"bytes"
	"context"
	"errors"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// nvidiaSource 通过 nvidia-smi 读取 NVIDIA 显卡（Linux / Windows 通用）。
type nvidiaSource struct {
	bin string
}

const nvidiaQuery = "index,name,driver_version,utilization.gpu,memory.used,memory.total,temperature.gpu,power.draw,power.limit,fan.speed"

func findNvidiaSMI() string {
	if p, err := exec.LookPath("nvidia-smi"); err == nil {
		return p
	}
	if runtime.GOOS == "windows" {
		for _, dir := range []string{
			filepath.Join(os.Getenv("SystemRoot"), "System32"),
			filepath.Join(os.Getenv("ProgramFiles"), "NVIDIA Corporation", "NVSMI"),
		} {
			p := filepath.Join(dir, "nvidia-smi.exe")
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}
	return ""
}

func (n *nvidiaSource) run(ctx context.Context) ([][]string, error) {
	if n.bin == "" {
		return nil, errors.New("nvidia-smi not found")
	}
	cctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	cmd := exec.CommandContext(cctx, n.bin, "--query-gpu="+nvidiaQuery, "--format=csv,noheader,nounits")
	hideWindow(cmd)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	rows := make([][]string, 0, 2)
	for _, line := range bytes.Split(out, []byte("\n")) {
		line = bytes.TrimSpace(line)
		if len(line) == 0 {
			continue
		}
		parts := strings.Split(string(line), ",")
		if len(parts) < 10 {
			continue
		}
		for i := range parts {
			parts[i] = strings.TrimSpace(parts[i])
		}
		rows = append(rows, parts)
	}
	if len(rows) == 0 {
		return nil, errors.New("no nvidia gpu")
	}
	return rows, nil
}

func (n *nvidiaSource) probe(ctx context.Context) ([]GPUInfo, error) {
	n.bin = findNvidiaSMI()
	rows, err := n.run(ctx)
	if err != nil {
		return nil, err
	}
	infos := make([]GPUInfo, 0, len(rows))
	for _, r := range rows {
		idx, _ := strconv.Atoi(r[0])
		info := GPUInfo{Index: idx, Name: r[1], Vendor: "nvidia", Driver: r[2]}
		if mb := smiNumber(r[5]); !math.IsNaN(mb) {
			info.MemoryTotal = uint64(mb * 1024 * 1024)
		}
		infos = append(infos, info)
	}
	return infos, nil
}

func (n *nvidiaSource) sample(ctx context.Context) ([]gpuStat, error) {
	rows, err := n.run(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]gpuStat, 0, len(rows))
	for _, r := range rows {
		idx, _ := strconv.Atoi(r[0])
		st := newGPUStat(idx)
		st.Util = smiNumber(r[3])
		if v := smiNumber(r[4]); !math.IsNaN(v) {
			st.MemUsed = v * 1024 * 1024
		}
		if v := smiNumber(r[5]); !math.IsNaN(v) {
			st.MemTotal = v * 1024 * 1024
		}
		st.Temp = smiNumber(r[6])
		st.Power = smiNumber(r[7])
		st.PowerLimit = smiNumber(r[8])
		st.Fan = smiNumber(r[9])
		out = append(out, st)
	}
	return out, nil
}

func (n *nvidiaSource) close() {}

// smiNumber 解析 nvidia-smi 的数值列；[N/A]、[Not Supported] 等返回 NaN。
func smiNumber(s string) float64 {
	v, err := strconv.ParseFloat(strings.TrimSpace(s), 64)
	if err != nil {
		return math.NaN()
	}
	return v
}
