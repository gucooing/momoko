//go:build darwin

package sysservice

import (
	"encoding/xml"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// plistPath launchd 配置。
const plistPath = "/Library/LaunchDaemons/" + Name + ".plist"

// target launchctl 里的服务名。
const target = "system/" + Name

// DefaultDir 默认安装目录。
func DefaultDir() string { return "/usr/local/momoko" }

// Supported macOS 都能装。
func Supported() error { return nil }

func xmlEscape(s string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(s))
	return b.String()
}

// plistFile 生成 launchd 配置：开机加载、以 root 运行，进程退出后由 launchd 重新拉起（在线更新后就是靠这个重启的）。
func plistFile(dir, exe string) string {
	log := LogFile(dir)
	return fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Label</key>
	<string>%s</string>
	<key>ProgramArguments</key>
	<array>
		<string>%s</string>
	</array>
	<key>WorkingDirectory</key>
	<string>%s</string>
	<key>EnvironmentVariables</key>
	<dict>
		<key>%s</key>
		<string>1</string>
	</dict>
	<key>RunAtLoad</key>
	<true/>
	<key>KeepAlive</key>
	<true/>
	<key>StandardOutPath</key>
	<string>%s</string>
	<key>StandardErrorPath</key>
	<string>%s</string>
</dict>
</plist>
`, Name, xmlEscape(exe), xmlEscape(dir), managedEnv, xmlEscape(log), xmlEscape(log))
}

var (
	reProgram = regexp.MustCompile(`(?s)<key>ProgramArguments</key>\s*<array>\s*<string>(.*?)</string>`)
	reWorkDir = regexp.MustCompile(`(?s)<key>WorkingDirectory</key>\s*<string>(.*?)</string>`)
)

func xmlUnescape(s string) string {
	r := strings.NewReplacer("&lt;", "<", "&gt;", ">", "&quot;", `"`, "&#34;", `"`, "&#39;", "'", "&apos;", "'", "&amp;", "&")
	return r.Replace(s)
}

// Lookup 读 launchd 配置找安装目录和程序；没装时 ok 为 false。
func Lookup() (Info, bool, error) {
	data, err := os.ReadFile(plistPath)
	if errors.Is(err, os.ErrNotExist) {
		return Info{}, false, nil
	}
	if err != nil {
		return Info{}, false, err
	}
	m := reProgram.FindSubmatch(data)
	if m == nil {
		return Info{}, false, fmt.Errorf("%s 里没有 ProgramArguments", plistPath)
	}
	info := Info{Exe: xmlUnescape(string(m[1]))}
	if w := reWorkDir.FindSubmatch(data); w != nil {
		info.Dir = xmlUnescape(string(w[1]))
	} else {
		info.Dir = filepath.Dir(info.Exe)
	}
	info.State, _ = queryState()
	return info, true, nil
}

func queryState() (State, error) {
	out, err := exec.Command("launchctl", "print", target).Output()
	if err != nil {
		return StateStopped, nil // 没加载
	}
	s := string(out)
	switch {
	case strings.Contains(s, "state = running"):
		return StateRunning, nil
	case strings.Contains(s, "state = spawn scheduled"), strings.Contains(s, "state = spawning"):
		return StateStarting, nil
	}
	return StateStopped, nil
}

func loaded() bool { return exec.Command("launchctl", "print", target).Run() == nil }

// Register 写 launchd 配置（开机时自动加载）。要 root。
func Register(dir, exe string) error {
	if err := os.MkdirAll(filepath.Join(dir, "logs"), 0o755); err != nil {
		return err
	}
	if loaded() {
		_ = runCmd("launchctl", "bootout", target)
	}
	if err := os.WriteFile(plistPath, []byte(plistFile(dir, exe)), 0o644); err != nil {
		return fmt.Errorf("写入 %s 失败：%w", plistPath, err)
	}
	return nil
}

// Unregister 停止并删掉 launchd 配置。要 root。
func Unregister() error {
	if loaded() {
		_ = runCmd("launchctl", "bootout", target)
	}
	if err := os.Remove(plistPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除 %s 失败：%w", plistPath, err)
	}
	return nil
}

// ErrAccessDenied 没有权限控制服务（统一用 sudo 提权，不会走到这里）。
var ErrAccessDenied = errors.New("没有权限控制 Momoko 服务")

// Start 加载并启动服务。
func Start() error {
	if loaded() {
		if err := runCmd("launchctl", "kickstart", target); err != nil {
			return err
		}
	} else if err := runCmd("launchctl", "bootstrap", "system", plistPath); err != nil {
		return err
	}
	return waitState(queryState, StateRunning, 30*time.Second)
}

// Stop 停止服务（卸载，launchd 才不会把它再拉起来；配置还在，开机时照样加载）。
func Stop() error {
	if !loaded() {
		return nil
	}
	return runCmd("launchctl", "bootout", target)
}

// AllowFirewall macOS 的应用防火墙默认关着，不处理。
func AllowFirewall(string, []int) (string, error) { return "", nil }

// RemoveFirewall 不处理。
func RemoveFirewall(string) {}

// PrepareServiceProcess launchd 已经把输出写进日志文件（文件由 launchd 打开，这里不能挪走），不用处理。
func PrepareServiceProcess(string) {}

// LogHint 去哪看服务的日志。
func LogHint(dir string) string { return LogFile(dir) }
