package servercore

import (
	"bufio"
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
	"unicode"
)

const (
	defaultLogLimit           = 200
	defaultSubscriberBuffer   = 128
	defaultMaxRestartAttempts = 3
	defaultRestartInterval    = time.Second
	defaultStopTimeout        = 3 * time.Second
)

var closedWaitDone = func() <-chan struct{} {
	ch := make(chan struct{})
	close(ch)
	return ch
}()

// LogSource 表示日志来源。
type LogSource string

const (
	LogSourceStdout LogSource = "stdout"
	LogSourceStderr LogSource = "stderr"
	LogSourceStdin  LogSource = "stdin"
)

// LogEntry 表示一条控制台事件。
type LogEntry struct {
	Time   time.Time
	Source LogSource
	Text   string
}

// ServerConfig 表示单个服务端实例的启动配置。
type ServerConfig struct {
	ID                 string
	Command            string
	Args               []string
	CommandLine        bool
	Dir                string
	Env                []string
	StopCommand        string
	AutoRestart        bool
	MaxRestartAttempts int
	RestartInterval    time.Duration
	LogLimit           int
	SubscriberBuffer   int
	Terminal           bool
}

// NewTerminalConfig 返回本机终端实例的默认配置。
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
	cfg        ServerConfig
	createTime time.Time

	mu              sync.RWMutex
	stdin           io.WriteCloser
	running         bool
	startTime       time.Time
	waitDone        chan struct{}
	waitFn          func() error
	stopFn          func(force bool) error
	closeFn         func()
	stdinLineEnd    string
	rawOutput       bool
	runID           uint64
	manualStop      bool
	restartAttempts int
	logs            []LogEntry
	nextSubID       uint64
	subscribers     map[uint64]chan LogEntry
	outputSanitizer *terminalOutputSanitizer
}

type startResult struct {
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	waitFn       func() error
	stopFn       func(force bool) error
	closeFn      func()
	stdinLineEnd string
	rawOutput    bool
}

// NewServer 创建一个最小可用的服务端实例。
func NewServer(cfg ServerConfig) (*Server, error) {
	normalized := cfg
	normalized.ID = strings.TrimSpace(cfg.ID)
	normalized.Command = strings.TrimSpace(cfg.Command)
	normalized.StopCommand = strings.TrimSpace(cfg.StopCommand)
	normalized.Args = append([]string(nil), cfg.Args...)
	normalized.Env = append([]string(nil), cfg.Env...)
	resolvedDir, err := normalizeDir(cfg.Dir)
	if err != nil {
		return nil, err
	}
	normalized.Dir = resolvedDir

	if normalized.ID == "" {
		return nil, errors.New("服务端 ID 不能为空")
	}
	if normalized.Command == "" {
		return nil, errors.New("启动命令不能为空")
	}
	if normalized.LogLimit <= 0 {
		normalized.LogLimit = defaultLogLimit
	}
	if normalized.SubscriberBuffer <= 0 {
		normalized.SubscriberBuffer = defaultSubscriberBuffer
	}
	if normalized.AutoRestart && normalized.MaxRestartAttempts <= 0 {
		normalized.MaxRestartAttempts = defaultMaxRestartAttempts
	}
	if normalized.RestartInterval <= 0 {
		normalized.RestartInterval = defaultRestartInterval
	}

	return &Server{
		cfg:         normalized,
		createTime:  time.Now(),
		subscribers: make(map[uint64]chan LogEntry),
	}, nil
}

// ID 返回实例 ID。
func (s *Server) ID() string {
	return s.cfg.ID
}

// CommandLine 返回实例启动命令。
func (s *Server) CommandLine() string {
	if s.cfg.CommandLine {
		return s.cfg.Command
	}
	parts := make([]string, 0, 1+len(s.cfg.Args))
	parts = append(parts, s.cfg.Command)
	parts = append(parts, s.cfg.Args...)
	return strings.Join(parts, " ")
}

// Dir 返回实例工作目录。
func (s *Server) Dir() string {
	return s.cfg.Dir
}

// Running 返回当前是否处于运行中。
func (s *Server) Running() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// Done 返回当前运行实例结束时会被关闭的通道。
func (s *Server) Done() <-chan struct{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.waitDone == nil {
		return closedWaitDone
	}
	return s.waitDone
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
	s.running = true
	s.startTime = time.Now()
	s.waitDone = waitDone
	s.manualStop = false
	s.restartAttempts = 0
	s.applyStartResultLocked(result)
	s.mu.Unlock()

	s.startReaders(result)
	go s.waitProcess(runID, waitDone)

	return nil
}

// Stop 优雅停止当前子进程。
func (s *Server) Stop() error {
	return s.stop(false)
}

// ForceStop 强制停止当前子进程。
func (s *Server) ForceStop() error {
	return s.stop(true)
}

func (s *Server) stop(force bool) error {
	s.mu.Lock()
	waitDone := s.waitDone
	stopFn := s.stopFn
	stdin := s.stdin
	lineEnd := s.stdinLineEnd
	stopCommand := s.cfg.StopCommand
	s.manualStop = true
	s.mu.Unlock()

	if waitDone == nil {
		return nil
	}

	if !force && stopCommand != "" && stdin != nil {
		if _, err := writeLine(stdin, stopCommand, lineEnd); err == nil {
			select {
			case <-waitDone:
				return nil
			case <-time.After(defaultStopTimeout):
			}
		}
	}

	if stopFn != nil {
		if err := stopFn(force); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("停止子进程失败: %w", err)
		}
	}

	if stopFn != nil && !force {
		select {
		case <-waitDone:
			return nil
		case <-time.After(defaultStopTimeout):
		}

		if err := stopFn(true); err != nil && !errors.Is(err, os.ErrProcessDone) {
			return fmt.Errorf("强制停止子进程失败: %w", err)
		}
	}

	<-waitDone
	return nil
}

// Restart 优雅重启当前子进程（先停再启）。
func (s *Server) Restart() error {
	if err := s.stop(false); err != nil {
		return err
	}
	return s.Start()
}

// ForceRestart 强制重启当前子进程。
func (s *Server) ForceRestart() error {
	if err := s.stop(true); err != nil {
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
	terminal := s.cfg.Terminal
	s.mu.RUnlock()

	if !running || stdin == nil {
		return errors.New("服务端未运行")
	}

	payload, err := writeLine(stdin, input, lineEnd)
	if err != nil {
		return fmt.Errorf("写入 stdin 失败: %w", err)
	}

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
func (s *Server) Subscribe() (<-chan LogEntry, func()) {
	ch := make(chan LogEntry, s.cfg.SubscriberBuffer)

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
		s.publish(LogEntry{Time: time.Now(), Source: source, Text: scanner.Text()})
	}

	if err := scanner.Err(); err != nil {
		s.publish(LogEntry{Time: time.Now(), Source: LogSourceStderr, Text: "读取输出失败: " + err.Error()})
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
				s.publish(LogEntry{Time: time.Now(), Source: source, Text: text})
			}
		}
		if err != nil {
			if !errors.Is(err, io.EOF) {
				s.publish(LogEntry{Time: time.Now(), Source: LogSourceStderr, Text: "读取输出失败: " + err.Error()})
			}
			return
		}
	}
}

func (s *Server) waitProcess(runID uint64, waitDone chan struct{}) {
	for {
		s.mu.RLock()
		waitFn := s.waitFn
		s.mu.RUnlock()

		if waitFn != nil {
			_ = waitFn()
		}

		result, restarted, closeFn, restartErr := s.handleProcessExit(runID)
		if closeFn != nil {
			closeFn()
		}
		if restartErr != nil {
			s.publish(LogEntry{Time: time.Now(), Source: LogSourceStderr, Text: restartErr.Error()})
		}
		if !restarted {
			close(waitDone)
			return
		}
		s.startReaders(result)
	}
}

func (s *Server) handleProcessExit(runID uint64) (*startResult, bool, func(), error) {
	s.mu.Lock()
	if s.runID != runID {
		s.mu.Unlock()
		return nil, false, nil, nil
	}

	closeFn := s.closeFn
	s.resetProcessStateLocked()

	if !s.shouldRestartLocked() {
		s.running = false
		s.startTime = time.Time{}
		s.waitDone = nil
		s.manualStop = false
		s.restartAttempts = 0
		s.mu.Unlock()
		return nil, false, closeFn, nil
	}

	s.restartAttempts++
	attempt := s.restartAttempts
	maxAttempts := s.cfg.MaxRestartAttempts
	restartInterval := s.cfg.RestartInterval
	s.mu.Unlock()

	if closeFn != nil {
		closeFn()
		closeFn = nil
	}
	if restartInterval > 0 {
		time.Sleep(restartInterval)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if s.runID != runID || s.manualStop {
		s.running = false
		s.startTime = time.Time{}
		s.waitDone = nil
		s.manualStop = false
		s.restartAttempts = 0
		return nil, false, nil, nil
	}

	result, err := s.startProcessLocked()
	if err != nil {
		s.running = false
		s.startTime = time.Time{}
		s.waitDone = nil
		s.manualStop = false
		s.restartAttempts = 0
		return nil, false, nil, fmt.Errorf("自动重启失败(%d/%d): %w", attempt, maxAttempts, err)
	}

	s.running = true
	s.startTime = time.Now()
	s.applyStartResultLocked(result)
	return result, true, nil, nil
}

func (s *Server) shouldRestartLocked() bool {
	return s.cfg.AutoRestart && !s.manualStop && s.cfg.MaxRestartAttempts > 0 && s.restartAttempts < s.cfg.MaxRestartAttempts
}

func (s *Server) applyStartResultLocked(result *startResult) {
	s.stdin = result.stdin
	s.waitFn = result.waitFn
	s.stopFn = result.stopFn
	s.closeFn = result.closeFn
	s.stdinLineEnd = result.stdinLineEnd
	s.rawOutput = result.rawOutput
	if s.rawOutput {
		s.outputSanitizer = newTerminalOutputSanitizer()
		return
	}
	s.outputSanitizer = nil
}

func (s *Server) resetProcessStateLocked() {
	s.stdin = nil
	s.waitFn = nil
	s.stopFn = nil
	s.closeFn = nil
	s.stdinLineEnd = ""
	s.rawOutput = false
	s.outputSanitizer = nil
}

func (s *Server) startReaders(result *startResult) {
	if result.rawOutput {
		go s.readRaw(result.stdout, LogSourceStdout)
		return
	}
	go s.readPipe(result.stdout, LogSourceStdout)
	if result.stderr != nil {
		go s.readPipe(result.stderr, LogSourceStderr)
	}
}

func (s *Server) startProcessLocked() (*startResult, error) {
	if s.cfg.Terminal {
		if result, err := startTerminalProcess(s); err == nil {
			return result, nil
		} else if runtime.GOOS == "windows" {
			return nil, err
		}
	}

	cmd := buildExecCommand(s.cfg.CommandLine, s.cfg.Command, s.cfg.Args, s.cfg.Dir)
	cmd.Dir = s.cfg.Dir
	if len(s.cfg.Env) > 0 {
		cmd.Env = append(os.Environ(), s.cfg.Env...)
	}
	configureExecCmd(cmd)

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
		waitFn:       cmd.Wait,
		stopFn:       func(force bool) error { return stopExecCmd(cmd, force) },
		stdinLineEnd: "\n",
	}, nil
}

func buildExecCommand(commandLine bool, command string, args []string, dir string) *exec.Cmd {
	if commandLine {
		parts := splitCommandLine(command)
		if len(parts) == 0 {
			return exec.Command(resolveCommandPath(command, dir))
		}
		return exec.Command(resolveCommandPath(parts[0], dir), parts[1:]...)
	}
	return exec.Command(resolveCommandPath(command, dir), args...)
}

func resolveCommandPath(command, dir string) string {
	name := strings.TrimSpace(command)
	if name == "" {
		return command
	}

	if filepath.IsAbs(name) {
		return filepath.Clean(name)
	}

	if hasPathSeparator(name) {
		if dir == "" {
			return filepath.Clean(name)
		}
		return filepath.Join(dir, name)
	}

	if dir == "" {
		return name
	}

	if candidate, ok := findInDir(name, dir); ok {
		return candidate
	}
	return name
}

func findInDir(name, dir string) (string, bool) {
	base := filepath.Join(dir, name)
	if isFile(base) {
		return base, true
	}

	if runtime.GOOS != "windows" || filepath.Ext(name) != "" {
		return "", false
	}

	for _, ext := range pathextValues() {
		candidate := base + ext
		if isFile(candidate) {
			return candidate, true
		}
	}
	return "", false
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func hasPathSeparator(name string) bool {
	for _, r := range name {
		if os.IsPathSeparator(uint8(r)) || r == '/' || r == '\\' {
			return true
		}
	}
	return false
}

func pathextValues() []string {
	raw := os.Getenv("PATHEXT")
	if raw == "" {
		return []string{".COM", ".EXE", ".BAT", ".CMD"}
	}

	parts := strings.Split(raw, ";")
	values := make([]string, 0, len(parts))
	for _, ext := range parts {
		ext = strings.TrimSpace(ext)
		if ext == "" {
			continue
		}
		values = append(values, ext)
	}
	if len(values) == 0 {
		return []string{".COM", ".EXE", ".BAT", ".CMD"}
	}
	return values
}

func splitCommandLine(line string) []string {
	line = strings.TrimSpace(line)
	if line == "" {
		return nil
	}

	parts := make([]string, 0, 4)
	var b strings.Builder
	inSingle := false
	inDouble := false

	flush := func() {
		if b.Len() == 0 {
			return
		}
		parts = append(parts, b.String())
		b.Reset()
	}

	for _, r := range line {
		switch {
		case r == '\'' && !inDouble:
			inSingle = !inSingle
		case r == '"' && !inSingle:
			inDouble = !inDouble
		case unicode.IsSpace(r) && !inSingle && !inDouble:
			flush()
		default:
			b.WriteRune(r)
		}
	}
	flush()
	return parts
}

func normalizeDir(dir string) (string, error) {
	trimmed := strings.TrimSpace(dir)
	if trimmed == "" {
		return "", nil
	}
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), nil
	}

	absDir, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("解析工作目录失败: %w", err)
	}
	return filepath.Clean(absDir), nil
}

func writeLine(w io.Writer, input, lineEnd string) (string, error) {
	if lineEnd == "" {
		lineEnd = "\n"
	}

	payload := input
	if !strings.HasSuffix(payload, lineEnd) {
		payload += lineEnd
	}
	_, err := io.WriteString(w, payload)
	return payload, err
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
	if overflow := len(s.logs) - s.cfg.LogLimit; overflow > 0 {
		copy(s.logs, s.logs[overflow:])
		s.logs = s.logs[:s.cfg.LogLimit]
	}

	subs := make([]chan LogEntry, 0, len(s.subscribers))
	for _, ch := range s.subscribers {
		subs = append(subs, ch)
	}
	return subs
}
