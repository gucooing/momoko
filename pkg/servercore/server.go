package servercore

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"
)

const (
	// 默认日志缓存条数，保持最小够用。
	defaultLogLimit = 200
	// 默认订阅通道缓冲，避免正常测试下轻易阻塞。
	defaultSubscriberBuffer = 128
)

// LogSource 表示日志来源。
type LogSource string

const (
	// LogSourceStdout 表示标准输出。
	LogSourceStdout LogSource = "stdout"
	// LogSourceStderr 表示错误输出。
	LogSourceStderr LogSource = "stderr"
	// LogSourceStdin 表示输入命令。
	LogSourceStdin LogSource = "stdin"
)

// LogEntry 表示一条控制台事件。
type LogEntry struct {
	Time   time.Time
	Source LogSource
	Text   string
}

// ServerConfig 表示单个服务端实例的最小启动配置。
type ServerConfig struct {
	ID               string
	Command          string
	Args             []string
	Dir              string
	Env              []string
	LogLimit         int
	SubscriberBuffer int
	Terminal         bool
}

// NewTerminalConfig 返回本机终端实例的默认配置。
// 终端具体用什么命令、是否启用平台专用实现，都收口在 servercore 中维护。
func NewTerminalConfig(id, dir string) ServerConfig {
	cfg := ServerConfig{
		ID:       id,
		Dir:      dir,
		Terminal: true,
	}

	if runtime.GOOS == "windows" {
		cfg.Command = "cmd.exe"
		return cfg
	}

	cfg.Command = "sh"
	return cfg
}

// Server 表示一个只管理单个子进程的服务端实例。
type Server struct {
	id               string
	command          string
	args             []string
	dir              string
	env              []string
	logLimit         int
	subscriberBuffer int
	terminal         bool
	createTime       time.Time

	mu              sync.RWMutex
	cmd             *exec.Cmd
	stdin           io.WriteCloser
	running         bool
	startTime       time.Time
	waitDone        chan struct{}
	waitFn          func() error
	stopFn          func() error
	closeFn         func()
	stdinLineEnd    string
	rawOutput       bool
	runID           uint64
	logs            []LogEntry
	nextSubID       uint64
	subscribers     map[uint64]chan LogEntry
	outputSanitizer *terminalOutputSanitizer
}

type startResult struct {
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	cmd          *exec.Cmd
	waitFn       func() error
	stopFn       func() error
	closeFn      func()
	stdinLineEnd string
	rawOutput    bool
}

// NewServer 创建一个最小可用的服务端实例。
func NewServer(cfg ServerConfig) (*Server, error) {
	if strings.TrimSpace(cfg.ID) == "" {
		return nil, errors.New("服务端 ID 不能为空")
	}
	if strings.TrimSpace(cfg.Command) == "" {
		return nil, errors.New("启动命令不能为空")
	}
	if cfg.LogLimit <= 0 {
		cfg.LogLimit = defaultLogLimit
	}
	if cfg.SubscriberBuffer <= 0 {
		cfg.SubscriberBuffer = defaultSubscriberBuffer
	}

	return &Server{
		id:               cfg.ID,
		command:          cfg.Command,
		args:             append([]string(nil), cfg.Args...),
		dir:              cfg.Dir,
		env:              append([]string(nil), cfg.Env...),
		logLimit:         cfg.LogLimit,
		subscriberBuffer: cfg.SubscriberBuffer,
		terminal:         cfg.Terminal,
		createTime:       time.Now(),
		subscribers:      make(map[uint64]chan LogEntry),
	}, nil
}

// ID 返回实例 ID。
func (s *Server) ID() string {
	return s.id
}

// CommandLine 返回实例启动命令。
func (s *Server) CommandLine() string {
	parts := make([]string, 0, 1+len(s.args))
	parts = append(parts, s.command)
	parts = append(parts, s.args...)
	return strings.Join(parts, " ")
}

// Dir 返回实例工作目录。
func (s *Server) Dir() string {
	return s.dir
}

// Running 返回当前是否处于运行中。
func (s *Server) Running() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// CreateTime 返回实例创建时间。
func (s *Server) CreateTime() time.Time {
	return s.createTime
}

// StartTime 返回当前运行中的启动时间。
func (s *Server) StartTime() (time.Time, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.running || s.startTime.IsZero() {
		return time.Time{}, false
	}
	return s.startTime, true
}

// Start 启动子进程，并异步读取 stdout 和 stderr。
func (s *Server) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return errors.New("服务端已经在运行")
	}

	result, err := s.startProcessLocked()
	if err != nil {
		s.mu.Unlock()
		return err
	}

	waitDone := make(chan struct{})
	s.runID++
	runID := s.runID
	s.cmd = result.cmd
	s.stdin = result.stdin
	s.running = true
	s.startTime = time.Now()
	s.waitDone = waitDone
	s.waitFn = result.waitFn
	s.stopFn = result.stopFn
	s.closeFn = result.closeFn
	s.stdinLineEnd = result.stdinLineEnd
	s.rawOutput = result.rawOutput
	if s.rawOutput {
		s.outputSanitizer = newTerminalOutputSanitizer()
	} else {
		s.outputSanitizer = nil
	}
	s.mu.Unlock()

	if result.rawOutput {
		go s.readRaw(result.stdout, LogSourceStdout)
	} else {
		go s.readPipe(result.stdout, LogSourceStdout)
		if result.stderr != nil {
			go s.readPipe(result.stderr, LogSourceStderr)
		}
	}
	go s.waitProcess(runID, waitDone)

	return nil
}

// Stop 停止当前子进程。
func (s *Server) Stop() error {
	s.mu.RLock()
	waitDone := s.waitDone
	stopFn := s.stopFn
	running := s.running
	s.mu.RUnlock()

	if !running || stopFn == nil {
		return nil
	}

	if err := stopFn(); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return fmt.Errorf("停止子进程失败: %w", err)
	}

	if waitDone != nil {
		<-waitDone
	}
	return nil
}

// Restart 重启当前子进程。
func (s *Server) Restart() error {
	if err := s.Stop(); err != nil {
		return err
	}
	return s.Start()
}

// Send 向子进程 stdin 写入一条命令，并把输入作为事件广播。
func (s *Server) Send(input string) error {
	s.mu.RLock()
	stdin := s.stdin
	running := s.running
	lineEnd := s.stdinLineEnd
	terminal := s.terminal
	s.mu.RUnlock()

	if !running || stdin == nil {
		return errors.New("服务端未运行")
	}

	if lineEnd == "" {
		lineEnd = "\n"
	}

	payload := input
	if !strings.HasSuffix(payload, lineEnd) {
		payload += lineEnd
	}

	if _, err := io.WriteString(stdin, payload); err != nil {
		return fmt.Errorf("写入 stdin 失败: %w", err)
	}

	// 普通服务实例保留输入事件广播，便于测试和订阅方观察。
	// 终端实例由 shell 自己回显，避免重复插入一份人为构造的输入日志。
	if !terminal {
		s.publish(LogEntry{
			Time:   time.Now(),
			Source: LogSourceStdin,
			Text:   strings.TrimRight(payload, "\r\n"),
		})
	}
	return nil
}

// Subscribe 订阅实时日志。
// 取消订阅后只会停止继续广播，不会主动关闭通道，避免并发发送时触发关闭通道异常。
func (s *Server) Subscribe() (<-chan LogEntry, func()) {
	ch := make(chan LogEntry, s.subscriberBuffer)

	s.mu.Lock()
	subID := s.nextSubID
	s.nextSubID++
	s.subscribers[subID] = ch
	s.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.subscribers, subID)
			s.mu.Unlock()
		})
	}

	return ch, cancel
}

// RecentLogs 返回最近日志的副本。
func (s *Server) RecentLogs() []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]LogEntry, len(s.logs))
	copy(out, s.logs)
	return out
}

func (s *Server) readPipe(r io.Reader, source LogSource) {
	if r == nil {
		return
	}

	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		s.publish(LogEntry{
			Time:   time.Now(),
			Source: source,
			Text:   scanner.Text(),
		})
	}

	if err := scanner.Err(); err != nil {
		s.publish(LogEntry{
			Time:   time.Now(),
			Source: LogSourceStderr,
			Text:   "读取输出失败: " + err.Error(),
		})
	}
}

func (s *Server) readRaw(r io.Reader, source LogSource) {
	if r == nil {
		return
	}

	buf := make([]byte, 4096)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			text := string(buf[:n])
			if s.outputSanitizer != nil {
				text = s.outputSanitizer.Filter(text)
			}
			if text != "" {
				s.publish(LogEntry{
					Time:   time.Now(),
					Source: source,
					Text:   text,
				})
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.publish(LogEntry{
					Time:   time.Now(),
					Source: LogSourceStderr,
					Text:   "读取输出失败: " + err.Error(),
				})
			}
			return
		}
	}
}

func (s *Server) waitProcess(runID uint64, waitDone chan struct{}) {
	if s.waitFn != nil {
		_ = s.waitFn()
	}

	s.mu.Lock()
	if s.runID == runID {
		closeFn := s.closeFn
		s.cmd = nil
		s.stdin = nil
		s.running = false
		s.startTime = time.Time{}
		s.waitDone = nil
		s.waitFn = nil
		s.stopFn = nil
		s.closeFn = nil
		s.stdinLineEnd = ""
		s.rawOutput = false
		s.outputSanitizer = nil
		s.mu.Unlock()

		if closeFn != nil {
			closeFn()
		}
		close(waitDone)
		return
	}
	s.mu.Unlock()

	close(waitDone)
}

func (s *Server) startProcessLocked() (*startResult, error) {
	if s.terminal {
		if result, err := startTerminalProcess(s); err == nil {
			return result, nil
		} else if runtime.GOOS == "windows" {
			return nil, err
		}
	}

	cmd := exec.Command(s.command, s.args...)
	cmd.Dir = s.dir
	if len(s.env) > 0 {
		cmd.Env = append(os.Environ(), s.env...)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("获取 stdout 管道失败: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("获取 stderr 管道失败: %w", err)
	}
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("获取 stdin 管道失败: %w", err)
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("启动子进程失败: %w", err)
	}

	return &startResult{
		stdin:        stdinPipe,
		stdout:       stdoutPipe,
		stderr:       stderrPipe,
		cmd:          cmd,
		waitFn:       cmd.Wait,
		stopFn:       cmd.Process.Kill,
		stdinLineEnd: "\n",
	}, nil
}

func (s *Server) publish(entry LogEntry) {
	subs := s.appendLogAndSnapshotSubscribers(entry)
	for _, ch := range subs {
		ch <- entry
	}
}

func (s *Server) appendLogAndSnapshotSubscribers(entry LogEntry) []chan LogEntry {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs = append(s.logs, entry)
	if overflow := len(s.logs) - s.logLimit; overflow > 0 {
		copy(s.logs, s.logs[overflow:])
		s.logs = s.logs[:s.logLimit]
	}

	subs := make([]chan LogEntry, 0, len(s.subscribers))
	for _, ch := range s.subscribers {
		subs = append(subs, ch)
	}
	return subs
}
