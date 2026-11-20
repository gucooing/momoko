//go:build unix

package shellcore

import (
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
)

// shellCommand 用户的登录 shell（$SHELL），没有就依次找 bash、sh。
// 分页器一律换成 cat：长输出（git log、systemctl、man …）整段打出来，由用户往上翻。
func shellCommand() (string, []string, []string) {
	env := []string{
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"PAGER=cat",
		"GIT_PAGER=cat",
		"SYSTEMD_PAGER=cat",
		"MANPAGER=cat",
	}
	if sh := os.Getenv("SHELL"); sh != "" && filepath.IsAbs(sh) {
		if info, err := os.Stat(sh); err == nil && !info.IsDir() {
			return sh, []string{"-l"}, env
		}
	}
	for _, name := range []string{"bash", "sh"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, []string{"-l"}, env
		}
	}
	return "/bin/sh", []string{"-l"}, env
}

// terminate 先挂断 shell 所在的进程组（go-pty 让 shell 成为会话首进程，pgid=pid），收尾时间过了还在就强制结束。
func terminate(proc *os.Process, exited <-chan struct{}) {
	if proc == nil {
		return
	}
	_ = syscall.Kill(-proc.Pid, syscall.SIGHUP)
	select {
	case <-exited:
	case <-time.After(killGrace):
		_ = syscall.Kill(-proc.Pid, syscall.SIGKILL)
	}
}
