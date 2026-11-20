//go:build linux

package shellcore

import (
	"syscall"

	pty "github.com/aymanbagabas/go-pty"
)

// configureCmd Momoko 异常退出时 shell 跟着挂断（go-pty 另外会设 Setsid / Setctty）。
func configureCmd(cmd *pty.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Pdeathsig: syscall.SIGHUP}
}
