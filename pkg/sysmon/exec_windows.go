//go:build windows

package sysmon

import (
	"os/exec"
	"syscall"
)

// hideWindow 后台服务调用命令行工具时不弹出控制台窗口。
func hideWindow(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: 0x08000000} // CREATE_NO_WINDOW
}
