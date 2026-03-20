package servercore

import (
	"strings"
	"testing"
)

// TestTerminalOutputSanitizerCursorMoveAsLineBreak 验证光标定位序列会被翻成纯文本换行，
// 避免提示符直接黏在上一行内容后面。
func TestTerminalOutputSanitizerCursorMoveAsLineBreak(t *testing.T) {
	s := newTerminalOutputSanitizer()

	input := "(c) Microsoft Corporation。\x1b[4;1HD:\\github\\momoko>"
	got := s.Filter(input)

	if !strings.Contains(got, "Corporation。\nD:\\github\\momoko>") {
		t.Fatalf("光标定位未被正确转换为换行，实际输出: %q", got)
	}
}
