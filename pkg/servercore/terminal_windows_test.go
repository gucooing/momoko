//go:build windows

package servercore

import (
	"strings"
	"testing"
	"time"
)

const terminalTestTimeout = 15 * time.Second

// TestTerminalCmdSupportsChinese 验证 Windows 下的终端实例能正确回显和处理中文输入。
func TestTerminalCmdSupportsChinese(t *testing.T) {
	server, err := NewServer(NewTerminalConfig("terminal-test", t.TempDir()))
	if err != nil {
		t.Fatalf("创建终端实例失败: %v", err)
	}

	logCh, cancel := server.Subscribe()
	defer cancel()

	if err := server.Start(); err != nil {
		t.Fatalf("启动终端实例失败: %v", err)
	}
	defer func() {
		_ = server.Stop()
	}()

	waitForTerminalText(t, logCh, func(text string) bool {
		return strings.Contains(text, "Microsoft Windows")
	})

	if err := server.Send("我的世界"); err != nil {
		t.Fatalf("发送中文命令失败: %v", err)
	}

	output := waitForTerminalText(t, logCh, func(text string) bool {
		return strings.Contains(text, "我的世界") && strings.Contains(text, "not recognized")
	})

	if strings.Contains(output, "????") || strings.Contains(output, "����") {
		t.Fatalf("终端输出仍然存在乱码: %q", output)
	}
}

func waitForTerminalText(t *testing.T, ch <-chan LogEntry, match func(string) bool) string {
	t.Helper()

	timer := time.NewTimer(terminalTestTimeout)
	defer timer.Stop()

	var builder strings.Builder
	for {
		select {
		case entry := <-ch:
			builder.WriteString(entry.Text)
			if match(builder.String()) {
				return builder.String()
			}
		case <-timer.C:
			t.Fatalf("等待终端输出超时，当前输出: %q", builder.String())
		}
	}
}
