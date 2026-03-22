//go:build linux

package servercore

import (
	"os"
	"strconv"
	"strings"
	"syscall"
	"testing"
)

func TestServerStopTerminatesChildProcessTree(t *testing.T) {
	childPidFile := t.TempDir() + "/child.pid"
	server, err := NewServer(ServerConfig{
		ID:               t.Name(),
		Command:          os.Args[0],
		Args:             []string{"-test.run=TestHelperProcess"},
		Env:              []string{"GO_WANT_HELPER_PROCESS=1", "SERVERCORE_HELPER_MODE=spawn_child", "SERVERCORE_CHILD_PID_FILE=" + childPidFile},
		SubscriberBuffer: 32,
		LogLimit:         50,
	})
	if err != nil {
		t.Fatalf("创建服务端失败: %v", err)
	}

	logCh, cancel := server.Subscribe()
	defer cancel()

	if err := server.Start(); err != nil {
		t.Fatalf("启动服务端失败: %v", err)
	}
	defer func() {
		_ = server.ForceStop()
	}()

	waitForEntry(t, logCh, func(entry LogEntry) bool {
		return entry.Source == LogSourceStdout && entry.Text == "ready"
	})

	childPID := waitForChildPID(t, childPidFile)
	if !processExists(childPID) {
		t.Fatalf("子进程未成功启动，pid=%d", childPID)
	}

	if err := server.Stop(); err != nil {
		t.Fatalf("停止服务端失败: %v", err)
	}

	waitForCondition(t, "停止后子进程仍然存活", func() bool {
		return !processExists(childPID)
	})
}

func waitForChildPID(t *testing.T, path string) int {
	t.Helper()

	var pid int
	waitForCondition(t, "未读取到子进程 pid", func() bool {
		data, err := os.ReadFile(path)
		if err != nil {
			return false
		}

		value := strings.TrimSpace(string(data))
		if value == "" {
			return false
		}

		parsed, err := strconv.Atoi(value)
		if err != nil {
			t.Fatalf("解析子进程 pid 失败: %v", err)
		}
		pid = parsed
		return true
	})

	return pid
}

func processExists(pid int) bool {
	err := syscall.Kill(pid, 0)
	return err == nil || err == syscall.EPERM
}
