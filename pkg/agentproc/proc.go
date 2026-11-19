// Package agentproc 启动并看管子进程：Agent 适配器、登录命令、MCP 服务器。
//
// 这些进程都跑 stdin / stdout 上的协议，所以用管道而不是 PTY（pkg/servercore 的做法）：
// stdout 原样交给协议层，stderr 单独收着，只留最近的一段用来在出错时说明原因。
// 结束时连同它启动的子进程一起结束（适配器还会再起 claude / codex，npx 起的才是真正的 MCP 服务器）。
package agentproc

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// stderrKeep 保留的 stderr 末尾字节数。
const stderrKeep = 8 * 1024

// Config 启动一个进程需要的东西。
type Config struct {
	Command string
	Args    []string
	// Dir 工作目录；带路径的相对命令也相对它。
	Dir string
	// Env 在当前进程环境之上追加或覆盖的变量；值为空字符串表示清掉这个变量。
	Env map[string]string
}

// Process 一个正在运行（或已经退出）的进程。
type Process struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	startedAt time.Time

	mu     sync.Mutex
	stderr []byte

	done     chan struct{}
	exitErr  error
	killOnce sync.Once
}

// Start 启动进程。命令找不到时返回带命令名的错误。
func Start(cfg Config) (*Process, error) {
	name := strings.TrimSpace(cfg.Command)
	if name == "" {
		return nil, errors.New("没有配置启动命令")
	}
	resolved, err := resolve(name, cfg.Dir)
	if err != nil {
		return nil, fmt.Errorf("找不到命令 %s", name)
	}
	cmd := newCommand(resolved, cfg.Args)
	cmd.Dir = cfg.Dir
	cmd.Env = mergeEnv(os.Environ(), cfg.Env)
	// 进程退出后它的子孙可能还攥着输出不放，不能让 Wait 一直等下去。
	cmd.WaitDelay = time.Second
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动 %s 失败：%v", name, err)
	}
	p := &Process{cmd: cmd, stdin: stdin, stdout: stdout, startedAt: time.Now(), done: make(chan struct{})}
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		buf := make([]byte, 4096)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				p.appendStderr(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	go func() {
		// 先把 stderr 读完再 Wait，否则最后几行（往往就是退出原因）会丢。
		<-stderrDone
		p.exitErr = cmd.Wait()
		close(p.done)
	}()
	return p, nil
}

// resolve 找到命令对应的可执行文件。带路径的相对命令相对工作目录；Windows 上 npx 这类会解析到 .cmd。
func resolve(command, dir string) (string, error) {
	if dir != "" && !filepath.IsAbs(command) && filepath.Base(command) != command {
		command = filepath.Join(dir, command)
	}
	return exec.LookPath(command)
}

func (p *Process) appendStderr(b []byte) {
	p.mu.Lock()
	p.stderr = append(p.stderr, b...)
	if over := len(p.stderr) - stderrKeep; over > 0 {
		p.stderr = append(p.stderr[:0], p.stderr[over:]...)
	}
	p.mu.Unlock()
}

// Stdin 进程的标准输入。
func (p *Process) Stdin() io.WriteCloser { return p.stdin }

// Stdout 进程的标准输出。
func (p *Process) Stdout() io.ReadCloser { return p.stdout }

// PID 进程号。
func (p *Process) PID() int { return p.cmd.Process.Pid }

// StartedAt 启动时间。
func (p *Process) StartedAt() time.Time { return p.startedAt }

// Done 在进程退出后关闭。
func (p *Process) Done() <-chan struct{} { return p.done }

// Exited 进程是否已经退出。
func (p *Process) Exited() bool {
	select {
	case <-p.done:
		return true
	default:
		return false
	}
}

// Stderr 最近的 stderr 输出。
func (p *Process) Stderr() string {
	p.mu.Lock()
	defer p.mu.Unlock()
	return string(p.stderr)
}

// LastError 从 stderr 末尾挑一行最像原因的话；没有输出时返回退出状态。
func (p *Process) LastError() string {
	lines := strings.Split(strings.ReplaceAll(strings.ToValidUTF8(p.Stderr(), ""), "\r\n", "\n"), "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" || strings.HasPrefix(line, "at ") || strings.HasPrefix(line, "Node.js v") {
			continue
		}
		if r := []rune(line); len(r) > 300 {
			line = string(r[:300]) + "…"
		}
		return line
	}
	if p.Exited() && p.exitErr != nil {
		return p.exitErr.Error()
	}
	return ""
}

// Kill 结束进程和它启动的所有子进程。
func (p *Process) Kill() {
	p.killOnce.Do(func() {
		_ = p.stdin.Close()
		if !p.Exited() {
			killTree(p.cmd)
		}
	})
}

// Wait 等进程退出，最多等 timeout。
func (p *Process) Wait(timeout time.Duration) bool {
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	select {
	case <-p.done:
		return true
	case <-timer.C:
		return false
	}
}

// MemoryBytes 进程和它所有子进程占用的内存（RSS）之和；读不到返回 0。要遍历进程树，只在有人要看时调用。
func (p *Process) MemoryBytes() int64 {
	if p.Exited() {
		return 0
	}
	root, err := process.NewProcess(int32(p.PID()))
	if err != nil {
		return 0
	}
	return treeMemory(root, 0)
}

func treeMemory(proc *process.Process, depth int) int64 {
	var total int64
	if info, err := proc.MemoryInfo(); err == nil && info != nil {
		total += int64(info.RSS)
	}
	if depth >= 4 {
		return total
	}
	children, err := proc.Children()
	if err != nil {
		return total
	}
	for _, child := range children {
		total += treeMemory(child, depth+1)
	}
	return total
}

func mergeEnv(base []string, extra map[string]string) []string {
	if len(extra) == 0 {
		return base
	}
	keyOf := func(kv string) string {
		k, _, _ := strings.Cut(kv, "=")
		// Windows 的环境变量名不分大小写。
		if runtime.GOOS == "windows" {
			k = strings.ToUpper(k)
		}
		return k
	}
	override := make(map[string]struct{}, len(extra))
	for k := range extra {
		override[keyOf(k)] = struct{}{}
	}
	out := make([]string, 0, len(base)+len(extra))
	for _, kv := range base {
		if _, ok := override[keyOf(kv)]; !ok {
			out = append(out, kv)
		}
	}
	for k, v := range extra {
		if v != "" {
			out = append(out, k+"="+v)
		}
	}
	return out
}

// Lookup 在 PATH 里找一个命令，返回它的完整路径。
func Lookup(name string) (string, bool) {
	path, err := exec.LookPath(name)
	return path, err == nil
}

// SplitCommand 把一行启动命令拆成命令和参数，认单引号和双引号。
func SplitCommand(line string) (string, []string) {
	var parts []string
	var cur strings.Builder
	var quote rune
	has := false
	flush := func() {
		if has {
			parts = append(parts, cur.String())
			cur.Reset()
			has = false
		}
	}
	for _, r := range strings.TrimSpace(line) {
		switch {
		case quote != 0:
			if r == quote {
				quote = 0
			} else {
				cur.WriteRune(r)
			}
		case r == '"' || r == '\'':
			quote = r
			has = true
		case r == ' ' || r == '\t':
			flush()
		default:
			cur.WriteRune(r)
			has = true
		}
	}
	flush()
	if len(parts) == 0 {
		return "", nil
	}
	return parts[0], parts[1:]
}
