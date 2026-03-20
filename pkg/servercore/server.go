package servercore

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
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
	createTime       time.Time

	mu          sync.RWMutex
	cmd         *exec.Cmd
	stdin       io.WriteCloser
	running     bool
	startTime   time.Time
	waitDone    chan struct{}
	logs        []LogEntry
	nextSubID   uint64
	subscribers map[uint64]chan LogEntry
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

	cmd := exec.Command(s.command, s.args...)
	cmd.Dir = s.dir
	if len(s.env) > 0 {
		cmd.Env = append(os.Environ(), s.env...)
	}

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("获取 stdout 管道失败: %w", err)
	}
	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("获取 stderr 管道失败: %w", err)
	}
	stdinPipe, err := cmd.StdinPipe()
	if err != nil {
		s.mu.Unlock()
		return fmt.Errorf("获取 stdin 管道失败: %w", err)
	}
	if err := cmd.Start(); err != nil {
		s.mu.Unlock()
		return fmt.Errorf("启动子进程失败: %w", err)
	}

	waitDone := make(chan struct{})
	s.cmd = cmd
	s.stdin = stdinPipe
	s.running = true
	s.startTime = time.Now()
	s.waitDone = waitDone
	s.mu.Unlock()

	go s.readPipe(stdoutPipe, LogSourceStdout)
	go s.readPipe(stderrPipe, LogSourceStderr)
	go s.waitProcess(cmd, waitDone)

	return nil
}

// Stop 停止当前子进程。
func (s *Server) Stop() error {
	s.mu.RLock()
	cmd := s.cmd
	waitDone := s.waitDone
	running := s.running
	s.mu.RUnlock()

	if !running || cmd == nil {
		return nil
	}

	if err := cmd.Process.Kill(); err != nil && !errors.Is(err, os.ErrProcessDone) {
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
	s.mu.RUnlock()

	if !running || stdin == nil {
		return errors.New("服务端未运行")
	}

	payload := input
	if !strings.HasSuffix(payload, "\n") {
		payload += "\n"
	}

	if _, err := io.WriteString(stdin, payload); err != nil {
		return fmt.Errorf("写入 stdin 失败: %w", err)
	}

	s.publish(LogEntry{
		Time:   time.Now(),
		Source: LogSourceStdin,
		Text:   strings.TrimRight(payload, "\r\n"),
	})
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

func (s *Server) waitProcess(cmd *exec.Cmd, waitDone chan struct{}) {
	_ = cmd.Wait()

	s.mu.Lock()
	if s.cmd == cmd {
		s.cmd = nil
		s.stdin = nil
		s.running = false
		s.startTime = time.Time{}
		s.waitDone = nil
	}
	s.mu.Unlock()

	close(waitDone)
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
