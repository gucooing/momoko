//go:build linux

package servercore

import (
	"errors"
	"os"
	"syscall"

	pty "github.com/aymanbagabas/go-pty"
)

// ptyStdinLineEnd 是向 PTY 写入整行命令（停止命令等）时使用的行结束符。
const ptyStdinLineEnd = "\n"

// configurePtyCmd 在 Linux 下让子进程在父进程异常退出时收到 SIGTERM。
// 注意不要设置 Setpgid：go-pty 会为子进程 Setsid（成为会话首进程，pgid=pid），
// 两者互斥；按进程组发信号依旧可用 -pid。
func configurePtyCmd(cmd *pty.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGTERM}
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
