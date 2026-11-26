//go:build linux || darwin

package sysservice

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"syscall"
)

func isWindowsService() bool { return false }

// IsAdmin 是不是 root。
func IsAdmin() bool { return os.Geteuid() == 0 }

// ValidateDir 安装目录要是绝对路径，且不带会被服务定义当成特殊字符的符号。
func ValidateDir(dir string) error {
	if !filepath.IsAbs(dir) {
		return fmt.Errorf("安装目录要写完整路径，比如 %s", DefaultDir())
	}
	if strings.ContainsAny(dir, "\"\\%$`'\n\r\t") {
		return errors.New(`安装目录里不能有引号、反斜杠、% 和 $ 这些符号`)
	}
	return nil
}

// ErrElevationCanceled 没拿到 root 权限。
var ErrElevationCanceled = errors.New("没有拿到 root 权限，已取消")

// Elevate 用 sudo 重新运行 exe args（同一个终端，sudo 会在这里问密码）。成功时不返回。
func Elevate(exe string, args []string, _ string, _ io.Writer) (int, error) {
	sudo, err := exec.LookPath("sudo")
	if err != nil {
		return 1, errors.New("需要 root 权限：请切换到 root 后再运行")
	}
	argv := append([]string{"sudo", exe}, args...)
	if err := syscall.Exec(sudo, argv, os.Environ()); err != nil {
		return 1, fmt.Errorf("用 sudo 重新运行失败：%w", err)
	}
	return 0, nil
}

// OwnConsole Linux / macOS 不会双击开出新终端。
func OwnConsole() bool { return false }

// commandLink 终端里 momoko 命令的位置。
const commandLink = "/usr/local/bin/momoko"

// LinkCommand 在 /usr/local/bin 建 momoko 的链接（已有同名的普通文件时不动它）。
func LinkCommand(_, exe string) (string, error) {
	if fi, err := os.Lstat(commandLink); err == nil {
		if fi.Mode()&os.ModeSymlink == 0 {
			return "", fmt.Errorf("%s 已经有一个文件了，没有覆盖；可以直接用 %s", commandLink, exe)
		}
		if target, _ := os.Readlink(commandLink); target == exe {
			return "终端里可以直接用 momoko 命令（" + commandLink + "）", nil
		}
		if err := os.Remove(commandLink); err != nil {
			return "", err
		}
	}
	if err := os.MkdirAll(filepath.Dir(commandLink), 0o755); err != nil {
		return "", err
	}
	if err := os.Symlink(exe, commandLink); err != nil {
		return "", fmt.Errorf("创建 %s 失败：%w", commandLink, err)
	}
	return "终端里可以直接用 momoko 命令（" + commandLink + " → " + exe + "）", nil
}

// UnlinkCommand 删掉指向 exe 的 /usr/local/bin/momoko。
func UnlinkCommand(_, exe string) error {
	if target, err := os.Readlink(commandLink); err == nil && target == exe {
		return os.Remove(commandLink)
	}
	return nil
}

// Run Linux / macOS 的服务就是普通进程，停止信号（SIGTERM）由 serve 自己处理。
func Run(serve func(stop <-chan struct{}) int) int { return serve(nil) }

// runCmd 运行命令，失败时把输出带进错误。
func runCmd(name string, args ...string) error {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			return fmt.Errorf("%s %s 失败：%w", name, strings.Join(args, " "), err)
		}
		return fmt.Errorf("%s %s 失败：%s", name, strings.Join(args, " "), msg)
	}
	return nil
}
