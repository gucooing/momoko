//go:build windows

package servercore

import (
	"os"
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// setHostProcAttr 托管进程不挂在面板的控制台上（面板的 Ctrl+C、关窗口不波及它），自成进程组；
// breakaway 时还脱离面板所在的作业对象，免得面板退出时被作业一起结束（作业不允许时由调用方去掉重试）。
func setHostProcAttr(cmd *exec.Cmd, breakaway bool) {
	flags := uint32(windows.CREATE_NEW_PROCESS_GROUP | windows.DETACHED_PROCESS)
	if breakaway {
		flags |= windows.CREATE_BREAKAWAY_FROM_JOB
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: flags, HideWindow: true}
}

// detachStdio 关掉面板给的标准输入输出管道。
func detachStdio() {
	null, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	_ = os.Stdin.Close()
	_ = os.Stdout.Close()
	_ = os.Stderr.Close()
	if err == nil {
		os.Stdin, os.Stdout, os.Stderr = null, null, null
	}
}

// hostSignals Windows 上托管进程没有控制台，收不到 Ctrl 事件。
func hostSignals(func()) func() { return func() {} }
