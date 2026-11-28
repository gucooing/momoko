package servercore

import (
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/shirou/gopsutil/v4/process"
)

// TestMain 测试程序也会被当成托管进程拉起来（带托管的管理器用当前程序起托管进程）。
func TestMain(m *testing.M) {
	RunHost()
	os.Exit(m.Run())
}

// exitRecorder 记下 OnExit 回调。
type exitRecorder struct {
	mu    sync.Mutex
	infos []ExitInfo
}

func (r *exitRecorder) record(info ExitInfo) {
	r.mu.Lock()
	r.infos = append(r.infos, info)
	r.mu.Unlock()
}

func (r *exitRecorder) list() []ExitInfo {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]ExitInfo(nil), r.infos...)
}

func hostedConsoleConfig(id string, env ...string) ServerConfig {
	return ServerConfig{
		ID:               id,
		Command:          os.Args[0],
		Args:             []string{"-test.run=TestHelperProcess"},
		Env:              append([]string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=console"}, env...),
		StopCommand:      "stop",
		SubscriberBuffer: 64,
		LogLimit:         100,
	}
}

func logsContain(s *Server, text string) bool {
	var b strings.Builder
	for _, e := range s.RecentLogs() {
		b.WriteString(e.Text)
	}
	return strings.Contains(b.String(), text)
}

// waitHostGone 等托管进程收尾（删掉套接字和口令文件）。
func waitHostGone(t *testing.T, m *ServerManager, id string) {
	t.Helper()
	sock, key := m.host.paths(id)
	waitForCondition(t, "托管进程没有收尾", func() bool {
		_, e1 := os.Stat(sock)
		_, e2 := os.Stat(key)
		return os.IsNotExist(e1) && os.IsNotExist(e2)
	})
}

func TestHostedServerStartSendStop(t *testing.T) {
	m := NewServerManager(WithHost(t.TempDir()))
	id := t.Name()
	rec := &exitRecorder{}
	cfg := hostedConsoleConfig(id)
	cfg.OnExit = rec.record
	server, err := m.Create(cfg)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	t.Cleanup(func() { _ = server.ForceStop() })
	logCh, cancel := server.Subscribe()
	defer cancel()

	if err := server.Start(); err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	if server.detachFn == nil {
		t.Fatal("应该经托管进程运行")
	}
	waitForEntry(t, logCh, func(e LogEntry) bool { return strings.Contains(e.Text, "ready") })
	if err := server.Send("ping"); err != nil {
		t.Fatalf("发送失败: %v", err)
	}
	waitForEntry(t, logCh, func(e LogEntry) bool { return strings.Contains(e.Text, "pong") })
	if err := server.Resize(100, 40); err != nil {
		t.Fatalf("调整尺寸失败: %v", err)
	}

	if err := server.Stop(); err != nil {
		t.Fatalf("停止失败: %v", err)
	}
	if server.Running() {
		t.Fatal("停止后仍在运行")
	}
	waitForCondition(t, "没有收到退出回调", func() bool { return len(rec.list()) == 1 })
	if info := rec.list()[0]; !info.Manual || info.Code != 0 {
		t.Fatalf("退出信息不对: %+v", info)
	}
	waitHostGone(t, m, id)
}

// 模拟在线更新：旧面板 Detach，应用继续运行；新面板 Attach 接回，进程号、启动时间、输出、控制台都在。
func TestHostedServerSurvivesPanelRestart(t *testing.T) {
	dir := t.TempDir()
	id := t.Name()

	old := NewServerManager(WithHost(dir))
	s1, err := old.Create(hostedConsoleConfig(id))
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	logCh, cancel := s1.Subscribe()
	if err := s1.Start(); err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	waitForEntry(t, logCh, func(e LogEntry) bool { return strings.Contains(e.Text, "ready") })
	_ = s1.Send("before-detach")
	waitForEntry(t, logCh, func(e LogEntry) bool { return strings.Contains(e.Text, "echo:before-detach") })
	cancel()
	pid := s1.PID()
	started, _ := s1.StartTime()

	if err := old.Detach(time.Second); err != nil {
		t.Fatalf("断开失败: %v", err)
	}
	if s1.Running() {
		t.Fatal("断开后旧面板不该再认为它在运行")
	}
	if ok, _ := process.PidExists(int32(pid)); !ok {
		t.Fatal("断开后应用不该退出")
	}

	fresh := NewServerManager(WithHost(dir))
	rec := &exitRecorder{}
	cfg := hostedConsoleConfig(id)
	cfg.OnExit = rec.record
	s2, err := fresh.Create(cfg)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	t.Cleanup(func() { _ = s2.ForceStop() })
	if !s2.Attach() {
		t.Fatal("新面板没接回运行中的应用")
	}
	if !s2.Running() || s2.PID() != pid {
		t.Fatalf("接回后状态不对: running=%v pid=%d want %d", s2.Running(), s2.PID(), pid)
	}
	if got, ok := s2.StartTime(); !ok || got.Sub(started).Abs() > time.Second {
		t.Fatalf("启动时间应是当初的 %v，实际 %v", started, got)
	}
	waitForCondition(t, "没回放断开前的输出", func() bool { return logsContain(s2, "echo:before-detach") })

	logCh2, cancel2 := s2.Subscribe()
	defer cancel2()
	_ = s2.Send("ping")
	waitForEntry(t, logCh2, func(e LogEntry) bool { return strings.Contains(e.Text, "pong") })

	if err := s2.Stop(); err != nil {
		t.Fatalf("停止失败: %v", err)
	}
	waitForCondition(t, "没有收到退出回调", func() bool { return len(rec.list()) == 1 })
	if info := rec.list()[0]; !info.Manual {
		t.Fatalf("应记为主动停止: %+v", info)
	}
	waitHostGone(t, fresh, id)
}

// 面板重启期间应用自己退出了：新面板收下它的输出和退出码，不当作运行中。
func TestHostedServerExitWhileDetached(t *testing.T) {
	dir := t.TempDir()
	id := t.Name()

	old := NewServerManager(WithHost(dir))
	s1, err := old.Create(hostedConsoleConfig(id, "SERVERCORE_EXIT_AFTER=300ms"))
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	logCh, cancel := s1.Subscribe()
	if err := s1.Start(); err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	waitForEntry(t, logCh, func(e LogEntry) bool { return strings.Contains(e.Text, "ready") })
	cancel()
	pid := s1.PID()
	if err := old.Detach(time.Second); err != nil {
		t.Fatalf("断开失败: %v", err)
	}
	waitForCondition(t, "应用没有自己退出", func() bool {
		ok, _ := process.PidExists(int32(pid))
		return !ok
	})

	fresh := NewServerManager(WithHost(dir))
	rec := &exitRecorder{}
	cfg := hostedConsoleConfig(id)
	cfg.OnExit = rec.record
	s2, err := fresh.Create(cfg)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	// 托管进程收完输出才报退出：还没报时会先接上、随即收到退出，结果一样
	s2.Attach()
	waitForCondition(t, "没收到退出回调", func() bool { return len(rec.list()) == 1 && !s2.Running() })
	if info := rec.list()[0]; info.Manual || info.Code != 3 {
		t.Fatalf("退出信息不对: %+v", info)
	}
	if !logsContain(s2, "bye") {
		t.Fatal("没收下面板不在时的输出")
	}
	waitHostGone(t, fresh, id)
}

// 还没接回就点了启动：接回已有的托管进程，不起第二份。
func TestHostedServerStartReusesLiveHost(t *testing.T) {
	dir := t.TempDir()
	id := t.Name()

	old := NewServerManager(WithHost(dir))
	s1, err := old.Create(hostedConsoleConfig(id))
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	logCh, cancel := s1.Subscribe()
	if err := s1.Start(); err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	waitForEntry(t, logCh, func(e LogEntry) bool { return strings.Contains(e.Text, "ready") })
	cancel()
	pid := s1.PID()
	_ = old.Detach(time.Second)

	fresh := NewServerManager(WithHost(dir))
	s2, err := fresh.Create(hostedConsoleConfig(id))
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	t.Cleanup(func() { _ = s2.ForceStop() })
	if err := s2.Start(); err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	if s2.PID() != pid {
		t.Fatalf("应接回原来的进程 %d，实际起了 %d", pid, s2.PID())
	}
	if err := s2.ForceStop(); err != nil {
		t.Fatalf("强制停止失败: %v", err)
	}
	waitHostGone(t, fresh, id)
}

// 经托管进程运行时，崩溃后自动重启照常工作，退出只记一次（不会又连上刚结束的托管进程再记一次）。
func TestHostedServerAutoRestart(t *testing.T) {
	m := NewServerManager(WithHost(t.TempDir()))
	rec := &exitRecorder{}
	cfg := hostedConsoleConfig(t.Name())
	cfg.AutoRestart = true
	cfg.MaxRestartAttempts = 2
	cfg.RestartInterval = 10 * time.Millisecond
	cfg.OnExit = rec.record
	server, err := m.Create(cfg)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	t.Cleanup(func() { _ = server.ForceStop() })
	logCh, cancel := server.Subscribe()
	defer cancel()
	if err := server.Start(); err != nil {
		t.Fatalf("启动失败: %v", err)
	}
	waitForEntry(t, logCh, func(e LogEntry) bool { return strings.Contains(e.Text, "ready") })
	first := server.PID()

	_ = server.Send("crash")
	waitForCondition(t, "崩溃后没有自动重启", func() bool {
		pid := server.PID()
		return server.Running() && pid != 0 && pid != first
	})
	time.Sleep(300 * time.Millisecond)
	infos := rec.list()
	if len(infos) != 1 || !infos[0].Restarted || infos[0].Code != 1 || infos[0].Manual {
		t.Fatalf("退出记录不对: %+v", infos)
	}
}

// 应用起不来时直接报错（不退回直接运行再失败一次）。
func TestHostedServerStartFailure(t *testing.T) {
	m := NewServerManager(WithHost(t.TempDir()))
	server, err := m.Create(ServerConfig{ID: t.Name(), Command: "momoko-servercore-no-such-command"})
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}
	err = server.Start()
	if err == nil {
		_ = server.ForceStop()
		t.Fatal("命令不存在时应该启动失败")
	}
	if !strings.Contains(err.Error(), "启动子进程失败") {
		t.Fatalf("错误不对: %v", err)
	}
	if logsContain(server, "直接运行") {
		t.Fatal("应用起不来不该退回直接运行")
	}
}
