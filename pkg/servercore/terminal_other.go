//go:build !windows

package servercore

import "errors"

// startTerminalProcess 在非 Windows 平台上返回回退信号，
// 让上层继续使用普通的 exec 管道模式。
func startTerminalProcess(s *Server) (*startResult, error) {
	_ = s
	return nil, errors.New("当前平台不需要专用终端实现")
}
