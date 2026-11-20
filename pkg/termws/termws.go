// Package termws 是浏览器终端 WebSocket 的公共约定（SSH 终端、本机终端共用）：
// 客户端文本帧要么是原样写入终端的输入，要么是 {"type":"resize","cols":N,"rows":N} 尺寸帧；
// 服务端文本帧是终端输出，必须是合法 UTF-8。
package termws

import (
	"encoding/json"
	"strings"
	"unicode/utf8"
)

// Resize 浏览器上报的终端尺寸。
type Resize struct {
	Type string `json:"type"`
	Cols int    `json:"cols"`
	Rows int    `json:"rows"`
}

// ParseResize 判断一条文本帧是不是尺寸帧。
func ParseResize(input string) (Resize, bool) {
	if !strings.HasPrefix(strings.TrimSpace(input), "{") {
		return Resize{}, false
	}
	var control Resize
	if err := json.Unmarshal([]byte(input), &control); err != nil {
		return Resize{}, false
	}
	if control.Type != "resize" || control.Cols <= 0 || control.Rows <= 0 {
		return Resize{}, false
	}
	return control, true
}

// UTF8Boundary 返回 b 中最后一个完整 UTF-8 字符之后的位置（末尾最多 3 个字节可能是被截断的字符）。
func UTF8Boundary(b []byte) int {
	n := len(b)
	for i := 1; i <= utf8.UTFMax-1 && i <= n; i++ {
		c := b[n-i]
		if c < 0x80 {
			return n // ASCII：此前都完整
		}
		if utf8.RuneStart(c) {
			if utf8.FullRune(b[n-i:]) {
				return n
			}
			return n - i
		}
	}
	return n
}
