//go:build windows

package agentproc

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
)

// createNoWindow 不给子进程分配控制台窗口。
const createNoWindow = 0x08000000

// newCommand 建启动命令。npx、pnpm 这类在 Windows 上是 .cmd 脚本，CreateProcess 不能直接跑，要交给 cmd.exe。
func newCommand(path string, args []string) *exec.Cmd {
	shell, cmdline, ok := ScriptCommandLine(path, args)
	if !ok {
		cmd := exec.Command(path, args...)
		cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
		return cmd
	}
	cmd := exec.Command(shell)
	cmd.SysProcAttr = &syscall.SysProcAttr{CmdLine: cmdline, HideWindow: true, CreationFlags: createNoWindow}
	return cmd
}

// ScriptCommandLine .cmd / .bat 脚本不能直接交给 CreateProcess：返回替它跑的 cmd.exe 和整条命令行；不是脚本时 ok 为 false。
func ScriptCommandLine(path string, args []string) (shell, cmdline string, ok bool) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext != ".cmd" && ext != ".bat" {
		return "", "", false
	}
	shell = os.Getenv("ComSpec")
	if shell == "" {
		shell = "cmd.exe"
	}
	// cmd.exe 解析命令行的规则和别的程序不一样，Go 默认的转义对它不对（路径带空格、参数里有 & 都会出错），
	// 所以自己拼整条命令行：/s 加最外层一对引号，里面的内容 cmd 就原样当命令行处理。
	parts := []string{caret(path)}
	for _, arg := range args {
		parts = append(parts, caret(quoteArg(arg)))
	}
	return shell, syscall.EscapeArg(shell) + ` /d /s /c "` + strings.Join(parts, " ") + `"`, true
}

// quoteArg 按 C 运行时的规则把一个参数包进引号：脚本最后启动的程序（node 等）是按这套规则拆参数的。
func quoteArg(arg string) string {
	var b strings.Builder
	b.WriteByte('"')
	slashes := 0
	for i := 0; i < len(arg); i++ {
		switch arg[i] {
		case '\\':
			slashes++
			continue
		case '"':
			// 引号前面的反斜杠要成对，引号自己再转义。
			b.WriteString(strings.Repeat(`\`, slashes*2+1))
		default:
			b.WriteString(strings.Repeat(`\`, slashes))
		}
		slashes = 0
		b.WriteByte(arg[i])
	}
	// 结尾的反斜杠后面紧跟收尾的引号，同样要成对。
	b.WriteString(strings.Repeat(`\`, slashes*2))
	b.WriteByte('"')
	return b.String()
}

// caret 在 cmd.exe 的每个特殊字符前加 ^，让它们（包括引号和空格）都只是普通字符。
func caret(s string) string {
	const meta = "()[]%!^\"`<>&|;, *?"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		if strings.IndexByte(meta, s[i]) >= 0 {
			b.WriteByte('^')
		}
		b.WriteByte(s[i])
	}
	return b.String()
}

// killTree 用 taskkill 连同子进程一起结束：Windows 没有进程组信号，只杀父进程会留下 claude / codex / node。
func killTree(cmd *exec.Cmd) {
	if cmd.Process == nil {
		return
	}
	kill := exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(cmd.Process.Pid))
	kill.SysProcAttr = &syscall.SysProcAttr{HideWindow: true, CreationFlags: createNoWindow}
	if err := kill.Run(); err != nil {
		_ = cmd.Process.Kill()
	}
}
