//go:build !unix

package servercore

import (
	"os"
	"os/exec"
)

func configureExecCmd(cmd *exec.Cmd) {
	_ = cmd
}

func stopExecCmd(cmd *exec.Cmd, force bool) error {
	_ = force
	if cmd == nil || cmd.Process == nil {
		return os.ErrProcessDone
	}
	return cmd.Process.Kill()
}
