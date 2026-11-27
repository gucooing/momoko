//go:build !linux

package agentsandbox

import (
	"os/exec"

	"momoko/pkg/agentproxy"
)

// Supported 这个系统能不能把 Agent 关进只走代理的网络（只有 Linux）。
func Supported() bool { return false }

// Check 现在能不能建隔离网络。
func Check() error { return ErrUnsupported }

// RunGate 只有 Linux 有看门进程。
func RunGate() {}

// Sandbox 只有 Linux 有。
type Sandbox struct{}

// Wrap 只有 Linux 能做。
func Wrap(*exec.Cmd) (*Sandbox, error) { return nil, ErrUnsupported }

func (*Sandbox) Attach(agentproxy.Proxy) error { return ErrUnsupported }
func (*Sandbox) Close()                        {}
