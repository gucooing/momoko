//go:build linux

package shellcore

import (
	"errors"
	"os"
	"os/exec"
	"syscall"

	pty "github.com/aymanbagabas/go-pty"

	"momoko/pkg/agentsandbox"
)

// configureCmd Momoko 异常退出时 shell 跟着挂断（go-pty 另外会设 Setsid / Setctty）。
func configureCmd(cmd *pty.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGHUP}
}

// startSandboxed 在沙箱里起命令（见 pkg/agentsandbox）：go-pty 的命令带不了额外的文件，这里自己接上伪终端的从端。
func startSandboxed(ptmx pty.Pty, spec Spec) (*os.Process, func() error, error) {
	up, ok := ptmx.(pty.UnixPty)
	if !ok {
		return nil, nil, errors.New("伪终端不对")
	}
	cmd := exec.Command(spec.Path, spec.Args...)
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	cmd.Stdin, cmd.Stdout, cmd.Stderr = up.Slave(), up.Slave(), up.Slave()
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true, Setctty: true, Pdeathsig: syscall.SIGHUP}
	wait, err := agentsandbox.Start(cmd, *spec.Sandbox)
	if err != nil {
		return nil, nil, err
	}
	return cmd.Process, wait, nil
}
