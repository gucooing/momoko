//go:build unix && !linux

package servercore

import (
	"errors"
	"os"
	"syscall"

	pty "github.com/aymanbagabas/go-pty"
)

// ptyStdinLineEnd 是向 PTY 写入整行命令（停止命令等）时使用的行结束符。
const ptyStdinLineEnd = "\n"

// configurePtyCmd 在非 Linux 的类 Unix 平台不需要额外属性；
// go-pty 已为子进程 Setsid（成为会话首进程，pgid=pid）。
func configurePtyCmd(cmd *pty.Cmd) {
	_ = cmd
}

// stopPtyProcess 按进程组发信号，避免只结束直接子进程。
func stopPtyProcess(proc *os.Process, force bool) error {
	if proc == nil {
		return os.ErrProcessDone
	}

	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}

	if err := syscall.Kill(-proc.Pid, sig); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	return nil
}
