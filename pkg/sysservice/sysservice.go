// Package sysservice 把 Momoko 装成系统服务：Windows 服务（LocalSystem）、Linux systemd（root）、macOS launchd（root）。
// 装好后开机自启、在后台运行（没有窗口、不用登录），在线更新换好程序后由服务管理器重新拉起新程序；
// 安装目录加进系统 PATH（Linux / macOS 在 /usr/local/bin 建链接），终端里直接敲 momoko 就能管理。
//
// 没安装时 momoko 和普通程序一样运行（双击、命令行，有控制台窗口），这时不能在线更新。
package sysservice

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

const (
	// Name 服务名（Windows 服务名 / systemd 单元名 / launchd 标签）。
	Name = "momoko"
	// DisplayName 服务显示名。
	DisplayName = "Momoko"
	// Description 服务说明。
	Description = "Momoko 管理面板"

	// managedEnv systemd / launchd 启动 momoko 时带上（见服务定义），程序据此知道自己是系统服务。
	// Windows 不需要：由服务管理器启动的进程能直接判断出来。
	managedEnv = "MOMOKO_SERVICE"
)

// ErrUnsupported 这个系统装不了系统服务。
var ErrUnsupported = errors.New("这个系统不支持安装成系统服务（只支持 Windows、带 systemd 的 Linux 和 macOS）")

// State 服务的运行状态。
type State int

const (
	StateUnknown State = iota
	StateStopped
	StateStarting
	StateRunning
	StateStopping
)

func (s State) String() string {
	switch s {
	case StateStopped:
		return "已停止"
	case StateStarting:
		return "正在启动"
	case StateRunning:
		return "运行中"
	case StateStopping:
		return "正在停止"
	default:
		return "未知"
	}
}

// Info 已安装的服务。
type Info struct {
	Dir   string // 安装目录（服务的工作目录，配置和数据都在这里）
	Exe   string // 程序文件
	State State
}

// ExeName 程序文件名。
func ExeName() string {
	if runtime.GOOS == "windows" {
		return "momoko.exe"
	}
	return "momoko"
}

// Executable 当前运行的程序文件（解开符号链接）。
func Executable() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return filepath.Abs(exe)
}

// SamePath 两个路径是不是同一个文件（Windows 不分大小写）。
func SamePath(a, b string) bool {
	a, b = filepath.Clean(a), filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// managed 当前进程是不是由服务管理器启动的。环境变量读完就去掉，免得它再启动的程序（终端、Agent、托管的应用）继承过去。
var managed = func() bool {
	on := os.Getenv(managedEnv) == "1"
	_ = os.Unsetenv(managedEnv)
	return on || isWindowsService()
}()

// Managed 当前进程是不是作为系统服务运行（只有这时在线更新后才有人把新程序拉起来）。
func Managed() bool { return managed }

// EnterInstallDir 作为服务运行时把工作目录换到程序所在的安装目录（Windows 服务默认在 System32），
// 配置、数据库、data/ 这些相对路径都以它为准。返回安装目录。
func EnterInstallDir() (string, error) {
	exe, err := Executable()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(exe)
	return dir, os.Chdir(dir)
}

// Installed 当前程序是不是已安装的那一份（返回服务信息）。
func Installed() (Info, bool) {
	info, ok, err := Lookup()
	if err != nil || !ok {
		return Info{}, false
	}
	exe, err := Executable()
	if err != nil || !SamePath(exe, info.Exe) {
		return info, false
	}
	return info, true
}

// WaitPort 等本机端口能连上（服务启动好了），最多等 timeout。
func WaitPort(port int, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	addr := net.JoinHostPort("127.0.0.1", fmt.Sprint(port))
	for {
		c, err := net.DialTimeout("tcp", addr, time.Second)
		if err == nil {
			_ = c.Close()
			return true
		}
		if time.Now().After(deadline) {
			return false
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// waitState 轮询服务状态直到 want，最多等 timeout。
func waitState(query func() (State, error), want State, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for {
		st, err := query()
		if err != nil {
			return err
		}
		if st == want {
			return nil
		}
		if time.Now().After(deadline) {
			return fmt.Errorf("等了 %d 秒服务还是「%s」", int(timeout/time.Second), st)
		}
		time.Sleep(300 * time.Millisecond)
	}
}

// rotateLog 日志超过 20 MB 时把它挪成 .1（只留一份旧的），服务启动时调用。
func rotateLog(path string) {
	if st, err := os.Stat(path); err == nil && st.Size() > 20<<20 {
		_ = os.Remove(path + ".1")
		_ = os.Rename(path, path+".1")
	}
}

// LogFile 服务输出写到的日志文件（Windows、macOS）；Linux 交给 journald，返回空。
func LogFile(dir string) string {
	if runtime.GOOS == "linux" {
		return ""
	}
	return filepath.Join(dir, "logs", "momoko.log")
}
