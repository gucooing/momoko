//go:build linux

package servercore

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// configureExecCmd 为子进程创建独立进程组，
// 同时在父进程异常退出时给子进程发送 SIGTERM。
func configureExecCmd(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid:   true,
		Pdeathsig: syscall.SIGTERM,
	}
}

// stopExecCmd 在 Linux 下按进程组发信号，
// 避免只退出父进程而遗留子进程。
func stopExecCmd(cmd *exec.Cmd, force bool) error {
	if cmd == nil || cmd.Process == nil {
		return os.ErrProcessDone
	}

	sig := syscall.SIGTERM
	if force {
		sig = syscall.SIGKILL
	}

	if err := syscall.Kill(-cmd.Process.Pid, sig); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	return nil
}
