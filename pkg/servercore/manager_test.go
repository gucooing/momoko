package servercore

import (
	"bufio"
	"fmt"
	"os"
	"testing"
	"time"
)

const testTimeout = 5 * time.Second

// TestHelperProcess 作为测试子进程入口使用。
func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}

	switch os.Getenv("SERVERCORE_HELPER_MODE") {
	case "console":
		runConsoleHelper()
		os.Exit(0)
	case "exit_fast":
		runExitFastHelper()
		os.Exit(1)
	default:
		fmt.Fprintln(os.Stderr, "unknown helper mode")
		os.Exit(2)
	}
}

func TestServerManagerStartAndReceiveOutput(t *testing.T) {
	manager, id := newTestManager(t, 20)

	ch, cancel, err := manager.Subscribe(id)
	if err != nil {
		t.Fatalf("订阅日志失败: %v", err)
	}
	defer cancel()

	if err := manager.Start(id); err != nil {
		t.Fatalf("启动服务端失败: %v", err)
	}

	waitForEntry(t, ch, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})
	waitForEntry(t, ch, func(entry LogEntry) bool {
		return entry.Source == LogSourceStderr && entry.Text == "stderr-ready"
	})
}

func TestServerManagerSendAndBroadcast(t *testing.T) {
	manager, id := newTestManager(t, 20)

	ch1, cancel1, err := manager.Subscribe(id)
	if err != nil {
		t.Fatalf("订阅日志失败: %v", err)
	}
	defer cancel1()

	ch2, cancel2, err := manager.Subscribe(id)
	if err != nil {
		t.Fatalf("订阅日志失败: %v", err)
	}
	defer cancel2()

	if err := manager.Start(id); err != nil {
		t.Fatalf("启动服务端失败: %v", err)
	}

	waitForEntry(t, ch1, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})
	waitForEntry(t, ch2, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})

	if err := manager.Send(id, "ping"); err != nil {
		t.Fatalf("发送命令失败: %v", err)
	}

	waitForEntry(t, ch1, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdin && entry.Text == "ping"
	})
	waitForEntry(t, ch2, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdin && entry.Text == "ping"
	})
	waitForEntry(t, ch1, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "pong"
	})
	waitForEntry(t, ch2, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "pong"
	})
}

func TestServerManagerStopAndRestart(t *testing.T) {
	manager, id := newTestManager(t, 50)

	ch, cancel, err := manager.Subscribe(id)
	if err != nil {
		t.Fatalf("订阅日志失败: %v", err)
	}
	defer cancel()

	server, ok := manager.Get(id)
	if !ok {
		t.Fatal("获取服务端失败")
	}

	if err := manager.Start(id); err != nil {
		t.Fatalf("启动服务端失败: %v", err)
	}

	waitForEntry(t, ch, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})
	waitForCondition(t, "启动后未进入运行状态", func() bool {
		return server.Running()
	})
	if server.CreateTime().IsZero() {
		t.Fatal("创建时间不应为空")
	}
	if _, ok := server.StartTime(); !ok {
		t.Fatal("启动后应当存在启动时间")
	}

	if err := manager.Stop(id); err != nil {
		t.Fatalf("停止服务端失败: %v", err)
	}

	waitForCondition(t, "停止后仍处于运行状态", func() bool {
		return !server.Running()
	})
	if _, ok := server.StartTime(); ok {
		t.Fatal("停止后不应保留启动时间")
	}

	if err := manager.Restart(id); err != nil {
		t.Fatalf("重启服务端失败: %v", err)
	}

	waitForEntry(t, ch, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})
	waitForCondition(t, "重启后未进入运行状态", func() bool {
		return server.Running()
	})

	logs, err := manager.RecentLogs(id)
	if err != nil {
		t.Fatalf("获取最近日志失败: %v", err)
	}

	readyCount := 0
	for _, entry := range logs {
		if entry.Source == LogSourceStdout && entry.Text == "ready" {
			readyCount++
		}
	}
	if readyCount < 2 {
		t.Fatalf("重启后启动日志数量不正确，实际为 %d", readyCount)
	}
}

func TestServerForceStopAndForceRestart(t *testing.T) {
	manager := NewServerManager()
	id := t.Name()

	_, err := manager.Create(ServerConfig{
		ID:               id,
		Command:          os.Args[0],
		Args:             []string{"-test.run=TestHelperProcess"},
		Env:              []string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=console"},
		StopCommand:      "invalid-stop-command",
		SubscriberBuffer: 32,
		LogLimit:         50,
	})
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	ch, cancel, err := manager.Subscribe(id)
	if err != nil {
		t.Fatalf("订阅日志失败: %v", err)
	}
	defer cancel()

	if err := manager.Start(id); err != nil {
		t.Fatalf("启动服务端失败: %v", err)
	}
	waitForEntry(t, ch, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})

	start := time.Now()
	if err := manager.ForceStop(id); err != nil {
		t.Fatalf("强制停止失败: %v", err)
	}
	if elapsed := time.Since(start); elapsed > time.Second {
		t.Fatalf("强制停止耗时过长: %v", elapsed)
	}

	if err := manager.ForceRestart(id); err != nil {
		t.Fatalf("强制重启失败: %v", err)
	}
	waitForEntry(t, ch, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})

	_ = manager.ForceStop(id)
}

func TestServerAutoRestartMaxAttempts(t *testing.T) {
	manager := NewServerManager()
	id := t.Name()

	_, err := manager.Create(ServerConfig{
		ID:                 id,
		Command:            os.Args[0],
		Args:               []string{"-test.run=TestHelperProcess"},
		Env:                []string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=exit_fast"},
		AutoRestart:        true,
		MaxRestartAttempts: 2,
		RestartInterval:    10 * time.Millisecond,
		LogLimit:           200,
		SubscriberBuffer:   32,
	})
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	server, ok := manager.Get(id)
	if !ok {
		t.Fatal("获取服务端失败")
	}

	if err := manager.Start(id); err != nil {
		t.Fatalf("启动服务端失败: %v", err)
	}

	waitForCondition(t, "自动重启超过上限后仍在运行", func() bool {
		return !server.Running()
	})

	logs, err := manager.RecentLogs(id)
	if err != nil {
		t.Fatalf("获取最近日志失败: %v", err)
	}
	bootCount := 0
	for _, entry := range logs {
		if entry.Source == LogSourceStdout && entry.Text == "boot" {
			bootCount++
		}
	}
	if bootCount != 3 {
		t.Fatalf("自动重启次数不正确，期望 3 次启动，实际 %d", bootCount)
	}
}

func TestServerRecentLogsLimit(t *testing.T) {
	server, err := NewServer(ServerConfig{
		ID:       "buffer-test",
		Command:  "test",
		LogLimit: 3,
	})
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	server.publish(LogEntry{Source: LogSourceStdout, Text: "line-1"})
	server.publish(LogEntry{Source: LogSourceStdout, Text: "line-2"})
	server.publish(LogEntry{Source: LogSourceStdout, Text: "line-3"})
	server.publish(LogEntry{Source: LogSourceStdout, Text: "line-4"})

	logs := server.RecentLogs()
	if len(logs) != 3 {
		t.Fatalf("最近日志数量不正确，期望 3，实际 %d", len(logs))
	}

	want := []string{"line-2", "line-3", "line-4"}
	for i, text := range want {
		if logs[i].Text != text {
			t.Fatalf("最近日志顺序不正确，位置 %d 期望 %s，实际 %s", i, text, logs[i].Text)
		}
	}
}

func newTestManager(t *testing.T, logLimit int) (*ServerManager, string) {
	t.Helper()

	manager := NewServerManager()
	id := t.Name()

	_, err := manager.Create(ServerConfig{
		ID:               id,
		Command:          os.Args[0],
		Args:             []string{"-test.run=TestHelperProcess"},
		Env:              []string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=console"},
		LogLimit:         logLimit,
		SubscriberBuffer: 32,
	})
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	t.Cleanup(func() {
		_ = manager.ForceStop(id)
	})

	return manager, id
}

func waitForEntry(t *testing.T, ch <-chan LogEntry, match func(LogEntry) bool) LogEntry {
	t.Helper()

	timer := time.NewTimer(testTimeout)
	defer timer.Stop()

	for {
		select {
		case entry := <-ch:
			if match(entry) {
				return entry
			}
		case <-timer.C:
			t.Fatal("等待日志超时")
		}
	}
}

func waitForCondition(t *testing.T, message string, check func() bool) {
	t.Helper()

	deadline := time.Now().Add(testTimeout)
	for time.Now().Before(deadline) {
		if check() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}

	t.Fatal(message)
}

func runConsoleHelper() {
	fmt.Fprintln(os.Stdout, "ready")
	fmt.Fprintln(os.Stderr, "stderr-ready")

	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		switch line {
		case "ping":
			fmt.Fprintln(os.Stdout, "pong")
		default:
			fmt.Fprintln(os.Stdout, "echo:"+line)
		}
	}

	if err := scanner.Err(); err != nil {
		fmt.Fprintln(os.Stderr, "scanner error:", err)
	}
}

func runExitFastHelper() {
	fmt.Fprintln(os.Stdout, "boot")
}
