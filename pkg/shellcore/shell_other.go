//go:build !linux

package shellcore

import (
	"os"

	pty "github.com/aymanbagabas/go-pty"

	"momoko/pkg/agentsandbox"
)

// startSandboxed 只有 Linux 有沙箱。
func startSandboxed(pty.Pty, Spec) (*os.Process, func() error, error) {
	return nil, nil, agentsandbox.ErrUnsupported
}

// configureCmd 非 Linux 平台不需要额外的进程属性。
func configureCmd(cmd *pty.Cmd) {
	_ = cmd
}
