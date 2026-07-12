//go:build !unix

package servercore

import (
	"os"

	pty "github.com/aymanbagabas/go-pty"
)

// ptyStdinLineEnd 是向 PTY 写入整行命令（停止命令等）时使用的行结束符。
// Windows ConPTY 的输入是按键流，回车键为 CR。
const ptyStdinLineEnd = "\r"

// configurePtyCmd 在非 Unix 平台上不需要额外的进程属性配置。
func configurePtyCmd(cmd *pty.Cmd) {
	_ = cmd
}

// stopPtyProcess 在非 Unix 平台上直接结束目标进程；
// 优雅停止依赖 StopCommand 路径，关闭 ConPTY 也会促使控制台程序退出。
func stopPtyProcess(proc *os.Process, force bool) error {
	_ = force
	if proc == nil {
		return os.ErrProcessDone
	}
	return proc.Kill()
}
