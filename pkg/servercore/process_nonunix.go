//go:build !unix

package servercore

import (
	"os"
	"os/exec"
)

// configureExecCmd 在非 Unix 平台上不需要额外的进程属性配置。
func configureExecCmd(cmd *exec.Cmd) {
	_ = cmd
}

// stopExecCmd 在非 Unix 平台上直接结束目标进程。
func stopExecCmd(cmd *exec.Cmd, force bool) error {
	_ = force
	if cmd == nil || cmd.Process == nil {
		return os.ErrProcessDone
	}
	return cmd.Process.Kill()
}
