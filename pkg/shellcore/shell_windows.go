//go:build windows

package shellcore

import (
	"os"
	"os/exec"
	"syscall"

	pty "github.com/aymanbagabas/go-pty"

	"momoko/pkg/agentproc"
)

// command 建伪终端里的命令；.cmd / .bat 脚本（npm 装的命令多是这样）交给 cmd.exe。
func command(ptmx pty.Pty, path string, args []string) *pty.Cmd {
	if shell, line, ok := agentproc.ScriptCommandLine(path, args); ok {
		cmd := ptmx.Command(shell)
		cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: line}
		return cmd
	}
	return ptmx.Command(path, args...)
}

// noPagerScript 长输出不分页，整段打出来由用户往上翻：Windows PowerShell 的 help 走 more 函数 → more.com，
// PowerShell 7 的 help 直接调 more.com，都按窗口高度一屏一屏停。会话开头把两者换成直通。
const noPagerScript = `function global:more { param([string[]]$paths) if ($paths) { Get-Content $paths } else { $input } }; ` +
	`function global:more.com { $input }`

// shellCommand 优先 PowerShell 7（pwsh），其次 Windows PowerShell，最后 cmd。
// go-pty 在 Windows 下不查 PATH，这里一律解析成绝对路径。git 不开分页（GIT_PAGER=cat 时 git 直接输出，不会真去找 cat）。
func shellCommand() (string, []string, []string) {
	env := []string{"GIT_PAGER=cat"}
	for _, name := range []string{"pwsh.exe", "powershell.exe"} {
		if path, err := exec.LookPath(name); err == nil {
			return path, []string{"-NoLogo", "-NoExit", "-Command", noPagerScript}, env
		}
	}
	if comspec := os.Getenv("COMSPEC"); comspec != "" {
		return comspec, nil, env
	}
	if path, err := exec.LookPath("cmd.exe"); err == nil {
		return path, nil, env
	}
	return `C:\Windows\System32\cmd.exe`, nil, env
}

// terminate 结束 shell；关闭 ConPTY 也会让挂在同一控制台上的子进程退出。
func terminate(proc *os.Process, exited <-chan struct{}) {
	if proc == nil {
		return
	}
	select {
	case <-exited:
	default:
		_ = proc.Kill()
	}
}
