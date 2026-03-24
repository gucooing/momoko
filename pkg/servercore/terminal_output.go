package servercore

import "strings"

const (
	terminalStateText = iota
	terminalStateEsc
	terminalStateCSI
	terminalStateOSC
	terminalStateOSCEscape
)

// terminalOutputSanitizer 用于去掉 ConPTY 输出中的控制序列，
// 让当前前端的纯文本终端先能稳定显示内容。
type terminalOutputSanitizer struct {
	state            int
	csi              strings.Builder
	lastWasLineBreak bool
}

// newTerminalOutputSanitizer 创建一个终端输出过滤器实例。
func newTerminalOutputSanitizer() *terminalOutputSanitizer {
	return &terminalOutputSanitizer{state: terminalStateText}
}

// Filter 过滤终端原始输出中的控制序列，只保留适合纯文本展示的内容。
func (s *terminalOutputSanitizer) Filter(input string) string {
	var out strings.Builder

	for _, r := range input {
		switch s.state {
		case terminalStateText:
			if r == 0x1b {
				s.state = terminalStateEsc
				continue
			}
			out.WriteRune(r)
			s.lastWasLineBreak = r == '\n' || r == '\r'
		case terminalStateEsc:
			switch r {
			case '[':
				s.csi.Reset()
				s.state = terminalStateCSI
			case ']':
				s.state = terminalStateOSC
			default:
				s.state = terminalStateText
			}
		case terminalStateCSI:
			if r >= 0x40 && r <= 0x7e {
				s.handleCSI(&out, r)
				s.state = terminalStateText
				s.csi.Reset()
				continue
			}
			s.csi.WriteRune(r)
		case terminalStateOSC:
			switch r {
			case 0x07:
				s.state = terminalStateText
			case 0x1b:
				s.state = terminalStateOSCEscape
			}
		case terminalStateOSCEscape:
			if r == '\\' {
				s.state = terminalStateText
				continue
			}
			s.state = terminalStateOSC
		}
	}

	return out.String()
}

// handleCSI 处理已读完的 CSI 控制序列，并在需要时转换为普通文本效果。
func (s *terminalOutputSanitizer) handleCSI(out *strings.Builder, final rune) {
	switch final {
	case 'H', 'f':
		// ConPTY 在 cmd 提示符前经常用光标定位代替直接输出换行。
		// 对当前纯文本前端来说，把它翻成一条普通换行最接近用户预期。
		s.writeLineBreak(out)
	}
}

// writeLineBreak 在输出末尾补一个换行，同时避免重复写入连续空行。
func (s *terminalOutputSanitizer) writeLineBreak(out *strings.Builder) {
	if s.lastWasLineBreak {
		return
	}
	out.WriteByte('\n')
	s.lastWasLineBreak = true
}
