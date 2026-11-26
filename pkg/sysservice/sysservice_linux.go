//go:build linux

package sysservice

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// unitPath systemd 单元文件。
const unitPath = "/etc/systemd/system/" + Name + ".service"

// DefaultDir 默认安装目录。
func DefaultDir() string { return "/opt/momoko" }

// Supported 要有 systemd。
func Supported() error {
	if st, err := os.Stat("/run/systemd/system"); err != nil || !st.IsDir() {
		return errors.New("这台机器没有用 systemd，装不了系统服务；可以直接运行 momoko，或自己配置开机启动")
	}
	return nil
}

// unitFile 生成 systemd 单元：以 root 运行，工作目录是安装目录；进程不管怎么退出都 3 秒后重新拉起
// （在线更新换好程序后就是靠这个重启的）。KillMode=process：停止、重启时只结束 momoko 本身，
// 它启动的程序由它自己收尾，和以前一样。
func unitFile(dir, exe string) string {
	execStart := exe
	if strings.ContainsAny(exe, " ") {
		execStart = `"` + exe + `"`
	}
	return fmt.Sprintf(`[Unit]
Description=%s
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=root
WorkingDirectory=%s
ExecStart=%s
Environment=%s=1
Restart=always
RestartSec=3
KillMode=process
LimitNOFILE=1048576

[Install]
WantedBy=multi-user.target
`, Description, dir, execStart, managedEnv)
}

// Lookup 读单元文件找安装目录和程序；没装时 ok 为 false。
func Lookup() (Info, bool, error) {
	data, err := os.ReadFile(unitPath)
	if errors.Is(err, os.ErrNotExist) {
		return Info{}, false, nil
	}
	if err != nil {
		return Info{}, false, err
	}
	info, err := parseUnit(data)
	if err != nil {
		return Info{}, false, err
	}
	info.State, _ = queryState()
	return info, true, nil
}

// parseUnit 从单元文件里取出工作目录和程序路径。
func parseUnit(data []byte) (Info, error) {
	var info Info
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if v, ok := strings.CutPrefix(line, "WorkingDirectory="); ok {
			info.Dir = v
		} else if v, ok := strings.CutPrefix(line, "ExecStart="); ok {
			if rest, ok := strings.CutPrefix(v, `"`); ok {
				v, _, _ = strings.Cut(rest, `"`)
			} else {
				v, _, _ = strings.Cut(v, " ")
			}
			info.Exe = v
		}
	}
	if info.Exe == "" {
		return Info{}, fmt.Errorf("%s 里没有 ExecStart", unitPath)
	}
	return info, nil
}

func queryState() (State, error) {
	out, _ := exec.Command("systemctl", "show", Name, "--property=ActiveState", "--value").Output()
	switch strings.TrimSpace(string(out)) {
	case "active", "reloading":
		return StateRunning, nil
	case "activating":
		return StateStarting, nil
	case "deactivating":
		return StateStopping, nil
	case "inactive", "failed":
		return StateStopped, nil
	}
	return StateUnknown, nil
}

// Register 写单元文件并设为开机启动。要 root。
func Register(dir, exe string) error {
	if err := os.WriteFile(unitPath, []byte(unitFile(dir, exe)), 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败：%w", unitPath, err)
	}
	if err := runCmd("systemctl", "daemon-reload"); err != nil {
		return err
	}
	return runCmd("systemctl", "enable", Name)
}

// Unregister 停止、取消开机启动并删掉单元文件。要 root。
func Unregister() error {
	_ = runCmd("systemctl", "disable", "--now", Name)
	if err := os.Remove(unitPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除 %s 失败：%w", unitPath, err)
	}
	_ = runCmd("systemctl", "daemon-reload")
	_ = runCmd("systemctl", "reset-failed", Name)
	return nil
}

// ErrAccessDenied 没有权限控制服务（Linux 上统一用 sudo 提权，不会走到这里）。
var ErrAccessDenied = errors.New("没有权限控制 Momoko 服务")

// Start 启动服务。
func Start() error {
	if err := runCmd("systemctl", "start", Name); err != nil {
		return err
	}
	return waitState(queryState, StateRunning, 30*time.Second)
}

// Stop 停止服务。
func Stop() error { return runCmd("systemctl", "stop", Name) }

// AllowFirewall ufw / firewalld 在运行时放行这些 TCP 端口；都没开时什么也不做。
func AllowFirewall(_ string, ports []int) (string, error) {
	if out, err := exec.Command("ufw", "status").Output(); err == nil && strings.Contains(string(out), "Status: active") {
		for _, p := range ports {
			if err := runCmd("ufw", "allow", strconv.Itoa(p)+"/tcp"); err != nil {
				return "", err
			}
		}
		return "已在 ufw 里放行端口 " + joinPorts(ports), nil
	}
	if out, err := exec.Command("firewall-cmd", "--state").Output(); err == nil && strings.TrimSpace(string(out)) == "running" {
		for _, p := range ports {
			if err := runCmd("firewall-cmd", "--permanent", "--add-port="+strconv.Itoa(p)+"/tcp"); err != nil {
				return "", err
			}
		}
		if err := runCmd("firewall-cmd", "--reload"); err != nil {
			return "", err
		}
		return "已在 firewalld 里放行端口 " + joinPorts(ports), nil
	}
	return "", nil
}

// RemoveFirewall 卸载时不动防火墙（端口可能别的程序也在用）。
func RemoveFirewall(string) {}

func joinPorts(ports []int) string {
	s := make([]string, len(ports))
	for i, p := range ports {
		s[i] = strconv.Itoa(p)
	}
	return strings.Join(s, "、")
}

// PrepareServiceProcess systemd 把输出收进 journald，不用处理。
func PrepareServiceProcess(string) {}

// LogHint 去哪看服务的日志。
func LogHint(string) string { return "journalctl -u " + Name + " -f" }
