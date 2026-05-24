package servercore

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
	case "spawn_child":
		runSpawnChildHelper()
		os.Exit(0)
	default:
		fmt.Fprintln(os.Stderr, "unknown helper mode")
		os.Exit(2)
	}
}

func TestServerManagerShutdownStopsServersConcurrently(t *testing.T) {
	manager := NewServerManager()
	ids := []string{t.Name() + "-1", t.Name() + "-2"}
	logChs := make([]<-chan LogEntry, 0, len(ids))

	for _, id := range ids {
		server, err := manager.Create(ServerConfig{
			ID:               id,
			Command:          os.Args[0],
			Args:             []string{"-test.run=TestHelperProcess"},
			Env:              []string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=console", "SERVERCORE_STOP_DELAY=400ms"},
			StopCommand:      "stop",
			SubscriberBuffer: 32,
			LogLimit:         50,
		})
		if err != nil {
			t.Fatalf("创建服务端失败: %v", err)
		}

		logCh, cancel := server.Subscribe()
		t.Cleanup(cancel)
		logChs = append(logChs, logCh)
	}

	for _, id := range ids {
		if err := manager.Start(id); err != nil {
			t.Fatalf("启动服务端失败: %v", err)
		}
	}

	for _, logCh := range logChs {
		waitForEntry(t, logCh, func(entry LogEntry) bool {
			return entry.Source == LogSourceStdout && entry.Text == "ready"
		})
	}

	start := time.Now()
	if err := manager.Shutdown(600 * time.Millisecond); err != nil {
		t.Fatalf("并发关闭服务端失败: %v", err)
	}

	if elapsed := time.Since(start); elapsed >= 750*time.Millisecond {
		t.Fatalf("服务端未并发关闭，耗时过长: %v", elapsed)
	}
}

func TestServerManagerShutdownFallsBackToForceStop(t *testing.T) {
	manager := NewServerManager()
	id := t.Name()

	server, err := manager.Create(ServerConfig{
		ID:               id,
		Command:          os.Args[0],
		Args:             []string{"-test.run=TestHelperProcess"},
		Env:              []string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=console", "SERVERCORE_IGNORE_STOP=1"},
		StopCommand:      "stop",
		SubscriberBuffer: 32,
		LogLimit:         50,
	})
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	logCh, cancel := server.Subscribe()
	defer cancel()

	if err := manager.Start(id); err != nil {
		t.Fatalf("启动服务端失败: %v", err)
	}

	waitForEntry(t, logCh, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})

	start := time.Now()
	if err := manager.Shutdown(150 * time.Millisecond); err != nil {
		t.Fatalf("关闭服务端失败: %v", err)
	}

	if elapsed := time.Since(start); elapsed >= time.Second {
		t.Fatalf("强制停止兜底未及时生效，耗时过长: %v", elapsed)
	}

	waitForCondition(t, "关闭后服务端仍处于运行状态", func() bool {
		return !server.Running()
	})
}

func TestServerUpdateConfigAppliesStopCommandWithoutRestart(t *testing.T) {
	cfg := ServerConfig{
		ID:               t.Name(),
		Command:          os.Args[0],
		Args:             []string{"-test.run=TestHelperProcess"},
		Env:              []string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=console", "SERVERCORE_EXPECT_STOP=quit"},
		StopCommand:      "stop",
		SubscriberBuffer: 32,
		LogLimit:         50,
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	logCh, cancel := server.Subscribe()
	defer cancel()

	if err := server.Start(); err != nil {
		t.Fatalf("启动服务端失败: %v", err)
	}

	waitForEntry(t, logCh, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})

	updatedCfg := cfg
	updatedCfg.StopCommand = "quit"
	if err := server.UpdateConfig(updatedCfg); err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}

	if !server.Running() {
		t.Fatal("更新配置后实例不应退出")
	}

	start := time.Now()
	if err := server.Stop(); err != nil {
		t.Fatalf("停止服务端失败: %v", err)
	}
	if elapsed := time.Since(start); elapsed >= time.Second {
		t.Fatalf("更新后的停止命令未及时生效，耗时过长: %v", elapsed)
	}
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

func TestServerAutoRestartSuccessResetsRetryAttempts(t *testing.T) {
	cfg := ServerConfig{
		ID:                 t.Name(),
		Command:            os.Args[0],
		Args:               []string{"-test.run=TestHelperProcess"},
		Env:                []string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=console"},
		StopCommand:        "stop",
		AutoRestart:        true,
		MaxRestartAttempts: 2,
		RestartInterval:    10 * time.Millisecond,
		LogLimit:           200,
		SubscriberBuffer:   32,
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	logCh, cancel := server.Subscribe()
	defer cancel()

	if err := server.Start(); err != nil {
		t.Fatalf("启动服务端失败: %v", err)
	}

	waitForEntry(t, logCh, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})

	if err := server.Send("crash"); err != nil {
		t.Fatalf("发送第一次异常退出命令失败: %v", err)
	}

	waitForEntry(t, logCh, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})

	missingCommand := filepath.Join(t.TempDir(), "missing-command")
	if runtime.GOOS == "windows" {
		missingCommand += ".exe"
	}

	updatedCfg := cfg
	updatedCfg.Command = missingCommand
	updatedCfg.Args = nil
	updatedCfg.CommandLine = false
	if err := server.UpdateConfig(updatedCfg); err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}

	if err := server.Send("crash"); err != nil {
		t.Fatalf("发送第二次异常退出命令失败: %v", err)
	}

	waitForCondition(t, "自动重试耗尽后服务端仍处于运行状态", func() bool {
		return !server.Running()
	})
	time.Sleep(100 * time.Millisecond)

	logs := server.RecentLogs()
	readyCount := 0
	retryFailCount := 0
	for _, entry := range logs {
		if entry.Source == LogSourceStdout && entry.Text == "ready" {
			readyCount++
		}
		if entry.Source == LogSourceStderr && strings.HasPrefix(entry.Text, "自动重启失败(") {
			retryFailCount++
		}
	}
	if readyCount < 2 {
		t.Fatalf("第一次异常退出后应成功自动重启，实际 ready 日志 %d 条", readyCount)
	}
	if retryFailCount != 2 {
		t.Fatalf("重启成功后应重置次数，第二轮应重新获得 2 次尝试，实际失败日志 %d 条", retryFailCount)
	}
}

func TestServerUnexpectedExitWithoutAutoRestartDoesNotRestart(t *testing.T) {
	manager := NewServerManager()
	id := t.Name()

	_, err := manager.Create(ServerConfig{
		ID:               id,
		Command:          os.Args[0],
		Args:             []string{"-test.run=TestHelperProcess"},
		Env:              []string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=exit_fast"},
		RestartInterval:  10 * time.Millisecond,
		LogLimit:         200,
		SubscriberBuffer: 32,
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

	waitForCondition(t, "未开启自动重启时异常退出后仍在运行", func() bool {
		return !server.Running()
	})
	time.Sleep(100 * time.Millisecond)

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
	if bootCount != 1 {
		t.Fatalf("未开启自动重启时不应重复启动，实际启动 %d 次", bootCount)
	}
}

func TestServerStopDoesNotTriggerAutoRestart(t *testing.T) {
	server, err := NewServer(ServerConfig{
		ID:                 t.Name(),
		Command:            os.Args[0],
		Args:               []string{"-test.run=TestHelperProcess"},
		Env:                []string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=console"},
		StopCommand:        "stop",
		AutoRestart:        true,
		MaxRestartAttempts: 2,
		RestartInterval:    10 * time.Millisecond,
		LogLimit:           200,
		SubscriberBuffer:   32,
	})
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	logCh, cancel := server.Subscribe()
	defer cancel()

	if err := server.Start(); err != nil {
		t.Fatalf("启动服务端失败: %v", err)
	}

	waitForEntry(t, logCh, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})

	if err := server.Stop(); err != nil {
		t.Fatalf("停止服务端失败: %v", err)
	}

	waitForCondition(t, "主动停止后服务端仍处于运行状态", func() bool {
		return !server.Running()
	})
	time.Sleep(100 * time.Millisecond)

	logs := server.RecentLogs()
	readyCount := 0
	for _, entry := range logs {
		if entry.Source == LogSourceStdout && entry.Text == "ready" {
			readyCount++
		}
	}
	if readyCount != 1 {
		t.Fatalf("主动停止后不应触发自动重启，实际启动 %d 次", readyCount)
	}
}

func TestServerAutoRestartRetriesFailedStartsUntilMaxAttempts(t *testing.T) {
	cfg := ServerConfig{
		ID:                 t.Name(),
		Command:            os.Args[0],
		Args:               []string{"-test.run=TestHelperProcess"},
		Env:                []string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=console"},
		StopCommand:        "stop",
		AutoRestart:        true,
		MaxRestartAttempts: 3,
		RestartInterval:    10 * time.Millisecond,
		LogLimit:           200,
		SubscriberBuffer:   32,
	}

	server, err := NewServer(cfg)
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	logCh, cancel := server.Subscribe()
	defer cancel()

	if err := server.Start(); err != nil {
		t.Fatalf("启动服务端失败: %v", err)
	}

	waitForEntry(t, logCh, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})

	missingCommand := filepath.Join(t.TempDir(), "missing-command")
	if runtime.GOOS == "windows" {
		missingCommand += ".exe"
	}

	updatedCfg := cfg
	updatedCfg.Command = missingCommand
	updatedCfg.Args = nil
	updatedCfg.CommandLine = false
	if err := server.UpdateConfig(updatedCfg); err != nil {
		t.Fatalf("更新配置失败: %v", err)
	}

	if err := server.Send("crash"); err != nil {
		t.Fatalf("发送异常退出命令失败: %v", err)
	}

	waitForCondition(t, "自动重试耗尽后服务端仍处于运行状态", func() bool {
		return !server.Running()
	})
	time.Sleep(100 * time.Millisecond)

	logs := server.RecentLogs()
	retryFailCount := 0
	for _, entry := range logs {
		if entry.Source == LogSourceStderr && strings.HasPrefix(entry.Text, "自动重启失败(") {
			retryFailCount++
		}
	}
	if retryFailCount != 3 {
		t.Fatalf("启动失败后应继续重试到上限，期望 3 次失败日志，实际 %d", retryFailCount)
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

func TestServerClearLogs(t *testing.T) {
	server, err := NewServer(ServerConfig{
		ID:      "clear-logs-test",
		Command: "test",
	})
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	server.publish(LogEntry{Source: LogSourceStdout, Text: "line-1"})
	server.publish(LogEntry{Source: LogSourceStdout, Text: "line-2"})

	server.ClearLogs()

	if logs := server.RecentLogs(); len(logs) != 0 {
		t.Fatalf("清空日志后应为空，实际 %d 条", len(logs))
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

	stopDelay, _ := time.ParseDuration(os.Getenv("SERVERCORE_STOP_DELAY"))
	ignoreStop := os.Getenv("SERVERCORE_IGNORE_STOP") == "1"
	expectedStop := os.Getenv("SERVERCORE_EXPECT_STOP")
	if expectedStop == "" {
		expectedStop = "stop"
	}
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := scanner.Text()
		switch line {
		case "ping":
			fmt.Fprintln(os.Stdout, "pong")
		case "crash":
			os.Exit(1)
		case expectedStop:
			if ignoreStop {
				continue
			}
			if stopDelay > 0 {
				time.Sleep(stopDelay)
			}
			return
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

func runSpawnChildHelper() {
	childPidFile := os.Getenv("SERVERCORE_CHILD_PID_FILE")
	if childPidFile == "" {
		fmt.Fprintln(os.Stderr, "missing child pid file")
		os.Exit(2)
	}

	cmd := exec.Command("sh", "-c", "echo $$ > \"$SERVERCORE_CHILD_PID_FILE\"; while :; do sleep 1; done")
	cmd.Env = append(os.Environ(), "SERVERCORE_CHILD_PID_FILE="+childPidFile)
	if err := cmd.Start(); err != nil {
		fmt.Fprintln(os.Stderr, "start child failed:", err)
		os.Exit(2)
	}

	fmt.Fprintln(os.Stdout, "ready")
	for {
		time.Sleep(time.Second)
	}
}
