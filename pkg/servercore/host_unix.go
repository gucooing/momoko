//go:build unix

package servercore

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"golang.org/x/sys/unix"
)

// setHostProcAttr 托管进程自成一个会话：面板退出、面板所在的终端关掉或按 Ctrl+C，都不会波及它。
func setHostProcAttr(cmd *exec.Cmd, _ bool) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
}

// detachStdio 把标准输入输出换成 /dev/null（不能只关掉：文件号 0–2 会被之后打开的套接字占用）。
func detachStdio() {
	f, err := os.OpenFile(os.DevNull, os.O_RDWR, 0)
	if err != nil {
		return
	}
	for fd := 0; fd <= 2; fd++ {
		_ = unix.Dup2(int(f.Fd()), fd)
	}
	_ = f.Close()
}

// hostSignals 托管进程收到 SIGTERM（比如关机）时正常结束应用，自己等应用退出后再走；SIGHUP / SIGINT 不理。
// 用 Notify 而不是 Ignore：被忽略的信号会一直传给应用。
func hostSignals(stop func()) func() {
	ch := make(chan os.Signal, 4)
	signal.Notify(ch, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM)
	done := make(chan struct{})
	go func() {
		for {
			select {
			case sig := <-ch:
				if sig == syscall.SIGTERM {
					stop()
				}
			case <-done:
				return
			}
		}
	}()
	return func() {
		signal.Stop(ch)
		close(done)
	}
}
