//go:build !windows

package agentproc

import (
	"os/exec"
	"syscall"
	"time"
)

// newCommand 建启动命令，并让它自成一个进程组，结束时才能连它启动的子进程一起带走。
func newCommand(path string, args []string) *exec.Cmd {
	cmd := exec.Command(path, args...)
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	return cmd
}

// ScriptCommandLine 只有 Windows 的 .cmd / .bat 要交给 cmd.exe。
func ScriptCommandLine(string, []string) (shell, cmdline string, ok bool) { return "", "", false }

// killTree 先给整个进程组发 SIGTERM，两秒后还没退就 SIGKILL。
func killTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	pid := cmd.Process.Pid
	_ = syscall.Kill(-pid, syscall.SIGTERM)
	time.AfterFunc(2*time.Second, func() { _ = syscall.Kill(-pid, syscall.SIGKILL) })
}
