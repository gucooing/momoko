//go:build !linux

package shellcore

import (
	pty "github.com/aymanbagabas/go-pty"
)

// configureCmd 非 Linux 平台不需要额外的进程属性。
func configureCmd(cmd *pty.Cmd) {
	_ = cmd
}
