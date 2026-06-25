package servercore

import (
	"bufio"
	"context"
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
	AutoRestart        bool          // 子进程非主动退出后是否自动重启。
	MaxRestartAttempts int           // 非主动退出后的单轮最大重试次数，启动成功后会清零，不包含当前这次已启动。
	RestartInterval    time.Duration // 自动重启前的等待间隔。
	LogLimit           int
	SubscriberBuffer   int
}

// Server 表示一个只管理单个子进程的服务端实例。
type Server struct {
	cfg        ServerConfig
	createTime time.Time

	mu           sync.RWMutex
	stdin        io.WriteCloser
	running      bool
	startTime    time.Time
	waitDone     chan struct{}
	waitFn       func() error
	stopFn       func(force bool) error
	closeFn      func()
	stdinLineEnd string
	runID        uint64
	manualStop   bool
	logs         []LogEntry
	nextSubID    uint64
	subscribers  map[uint64]*logSubscriber
}

type logSubscriber struct {
	ch     chan LogEntry
	ctx    context.Context
	cancel context.CancelFunc
}

type startResult struct {
	stdin        io.WriteCloser
	stdout       io.ReadCloser
	stderr       io.ReadCloser
	waitFn       func() error
	stopFn       func(force bool) error
	closeFn      func()
	stdinLineEnd string
}

// normalizeServerConfig 清洗配置内容并补齐默认值。
func normalizeServerConfig(cfg ServerConfig) (ServerConfig, error) {
	normalized := cfg
	normalized.ID = strings.TrimSpace(cfg.ID)
	normalized.Command = strings.TrimSpace(cfg.Command)
	normalized.StopCommand = strings.TrimSpace(cfg.StopCommand)
	normalized.Args = append([]string(nil), cfg.Args...)
	normalized.Env = append([]string(nil), cfg.Env...)
	resolvedDir, err := normalizeDir(cfg.Dir)
	if err != nil {
		return ServerConfig{}, err
	}
	normalized.Dir = resolvedDir

	if normalized.ID == "" {
		return ServerConfig{}, errors.New("服务端 ID 不能为空")
	}
	if normalized.Command == "" {
		return ServerConfig{}, errors.New("启动命令不能为空")
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

	return normalized, nil
}

// NewServer 创建一个最小可用的服务端实例。
func NewServer(cfg ServerConfig) (*Server, error) {
	normalized, err := normalizeServerConfig(cfg)
	if err != nil {
		return nil, err
	}

	return &Server{
		cfg:         normalized,
		createTime:  time.Now(),
		subscribers: make(map[uint64]*logSubscriber),
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

// UpdateConfig 更新实例配置。
// 如果实例正在运行，仅更新后续控制和下次启动使用的配置，不会主动重启进程。
func (s *Server) UpdateConfig(cfg ServerConfig) error {
	normalized, err := normalizeServerConfig(cfg)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if normalized.ID != s.cfg.ID {
		return errors.New("服务端 ID 不允许修改")
	}

	s.cfg = normalized
	s.trimLogsLocked()
	return nil
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

// stop 按优雅或强制模式停止当前子进程。
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
	s.mu.RUnlock()

	if !running || stdin == nil {
		return errors.New("服务端未运行")
	}

	payload, err := writeLine(stdin, input, lineEnd)
	if err != nil {
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
func (s *Server) Subscribe() (<-chan LogEntry, func()) {
	ch, _, cancel := s.SubscribeWithDone()
	return ch, cancel
}

// SubscribeWithDone 订阅实时日志，并返回订阅被取消时关闭的通道。
func (s *Server) SubscribeWithDone() (<-chan LogEntry, <-chan struct{}, func()) {
	ch := make(chan LogEntry, s.cfg.SubscriberBuffer)
	ctx, cancelCtx := context.WithCancel(context.Background())
	sub := &logSubscriber{
		ch:     ch,
		ctx:    ctx,
		cancel: cancelCtx,
	}

	s.mu.Lock()
	subID := s.nextSubID
	s.nextSubID++
	s.subscribers[subID] = sub
	s.mu.Unlock()

	var once sync.Once
	cancel := func() {
		once.Do(func() {
			s.mu.Lock()
			delete(s.subscribers, subID)
			s.mu.Unlock()
			cancelCtx()
		})
	}

	return ch, ctx.Done(), cancel
}

// RecentLogs 返回最近日志的副本。
func (s *Server) RecentLogs() []LogEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()

	out := make([]LogEntry, len(s.logs))
	copy(out, s.logs)
	return out
}

// ClearLogs 清空当前实例保存的历史日志。
func (s *Server) ClearLogs() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = nil
}

// readPipe 按行读取标准输出或标准错误，并转成日志事件。
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

// waitProcess 等待当前子进程退出，并根据配置决定是否自动重启。
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

// handleProcessExit 在子进程退出后更新状态，并在需要时准备下一次启动结果。
func (s *Server) handleProcessExit(runID uint64) (*startResult, bool, func(), error) {
	s.mu.Lock()
	if s.runID != runID {
		s.mu.Unlock()
		return nil, false, nil, nil
	}

	closeFn := s.closeFn
	s.resetProcessStateLocked()

	if !s.shouldAutoRestartOnExitLocked() {
		s.markStoppedLocked()
		s.mu.Unlock()
		return nil, false, closeFn, nil
	}

	maxAttempts := s.cfg.MaxRestartAttempts
	restartInterval := s.cfg.RestartInterval
	attempt := 0
	s.mu.Unlock()

	if closeFn != nil {
		closeFn()
		closeFn = nil
	}

	for {
		if restartInterval > 0 {
			time.Sleep(restartInterval)
		}

		s.mu.Lock()
		if s.runID != runID || s.manualStop || !s.shouldAutoRestartOnExitLocked() {
			s.markStoppedLocked()
			s.mu.Unlock()
			return nil, false, nil, nil
		}

		attempt++
		result, err := s.startProcessLocked()
		if err == nil {
			s.running = true
			s.startTime = time.Now()
			s.applyStartResultLocked(result)
			s.mu.Unlock()
			return result, true, nil, nil
		}

		if attempt >= maxAttempts {
			s.markStoppedLocked()
			s.mu.Unlock()
			return nil, false, nil, fmt.Errorf("自动重启失败(%d/%d): %w", attempt, maxAttempts, err)
		}

		s.mu.Unlock()
		s.publish(LogEntry{
			Time:   time.Now(),
			Source: LogSourceStderr,
			Text:   fmt.Sprintf("自动重启失败(%d/%d): %v", attempt, maxAttempts, err),
		})
	}
}

// shouldAutoRestartOnExitLocked 只在子进程非主动退出时判定是否继续自动重试。
func (s *Server) shouldAutoRestartOnExitLocked() bool {
	return s.cfg.AutoRestart && !s.manualStop && s.cfg.MaxRestartAttempts > 0
}

// markStoppedLocked 把实例恢复到未运行状态。
func (s *Server) markStoppedLocked() {
	s.running = false
	s.startTime = time.Time{}
	s.waitDone = nil
	s.manualStop = false
}

// applyStartResultLocked 写入本次启动生成的运行时句柄和输出配置。
func (s *Server) applyStartResultLocked(result *startResult) {
	s.stdin = result.stdin
	s.waitFn = result.waitFn
	s.stopFn = result.stopFn
	s.closeFn = result.closeFn
	s.stdinLineEnd = result.stdinLineEnd
}

// resetProcessStateLocked 清空当前进程相关的运行时状态。
func (s *Server) resetProcessStateLocked() {
	s.stdin = nil
	s.waitFn = nil
	s.stopFn = nil
	s.closeFn = nil
	s.stdinLineEnd = ""
}

// startReaders 启动标准输出和标准错误的日志读取协程。
func (s *Server) startReaders(result *startResult) {
	go s.readPipe(result.stdout, LogSourceStdout)
	if result.stderr != nil {
		go s.readPipe(result.stderr, LogSourceStderr)
	}
}

// startProcessLocked 按当前配置启动子进程，并返回本次启动需要的运行时句柄。
func (s *Server) startProcessLocked() (*startResult, error) {
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

// buildExecCommand 根据配置构造 exec.Cmd，并解析命令路径。
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

// resolveCommandPath 结合工作目录解析命令的实际可执行路径。
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

// findInDir 在指定目录中查找命令文件，Windows 下会补齐 PATHEXT 后缀尝试。
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

// isFile 判断路径是否存在且为普通文件。
func isFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// hasPathSeparator 判断命令名中是否显式带有路径分隔符。
func hasPathSeparator(name string) bool {
	for _, r := range name {
		if os.IsPathSeparator(uint8(r)) || r == '/' || r == '\\' {
			return true
		}
	}
	return false
}

// pathextValues 返回 Windows PATHEXT 环境变量中的可执行扩展名列表。
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

// splitCommandLine 按简单引号规则拆分命令行字符串。
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

// normalizeDir 把工作目录统一解析为干净的绝对路径。
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

// writeLine 按指定换行符向写入端发送一行输入，并返回实际写入内容。
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

// publish 追加一条日志并广播给当前所有订阅者。
func (s *Server) publish(entry LogEntry) {
	subs := s.appendLogAndSnapshotSubscribers(entry)
	for _, sub := range subs {
		select {
		case <-sub.ctx.Done():
		case sub.ch <- entry:
		}
	}
}

// appendLogAndSnapshotSubscribers 写入日志缓冲，并返回订阅者快照。
func (s *Server) appendLogAndSnapshotSubscribers(entry LogEntry) []*logSubscriber {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.logs = append(s.logs, entry)
	s.trimLogsLocked()

	subs := make([]*logSubscriber, 0, len(s.subscribers))
	for _, sub := range s.subscribers {
		subs = append(subs, sub)
	}
	return subs
}

// CancelSubscribers 取消当前全部日志订阅，让依赖方自行结束会话。
func (s *Server) CancelSubscribers() {
	s.mu.Lock()
	subs := make([]*logSubscriber, 0, len(s.subscribers))
	for id, sub := range s.subscribers {
		subs = append(subs, sub)
		delete(s.subscribers, id)
	}
	s.mu.Unlock()

	for _, sub := range subs {
		sub.cancel()
	}
}

// trimLogsLocked 按日志上限裁剪最早的历史记录。
func (s *Server) trimLogsLocked() {
	if overflow := len(s.logs) - s.cfg.LogLimit; overflow > 0 {
		copy(s.logs, s.logs[overflow:])
		s.logs = s.logs[:s.cfg.LogLimit]
	}
}
