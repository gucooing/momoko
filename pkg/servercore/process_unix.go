//go:build unix && !linux

package servercore

import (
	"errors"
	"os"
	"os/exec"
	"syscall"
)

// configureExecCmd 为子进程创建独立进程组，
// 方便停止时一次性处理整个进程树。
func configureExecCmd(cmd *exec.Cmd) {
	if cmd == nil {
		return
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}

// stopExecCmd 在类 Unix 平台按进程组发信号，
// 避免只结束直接子进程。
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
