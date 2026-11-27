//go:build !linux

package agentsandbox

import (
	"os/exec"

	"momoko/pkg/agentinstall"
	"momoko/pkg/agentproxy"
)

// Supported 这个系统能不能建沙箱（只有 Linux）。
func Supported() bool { return false }

// Check 现在能不能建沙箱。
func Check() error { return ErrUnsupported }

// RunGate 只有 Linux 有沙箱。
func RunGate() {}

// Start 只有 Linux 能做。
func Start(*exec.Cmd, Config) (func() error, error) { return nil, ErrUnsupported }

// SingBoxRelease 只有 Linux 用 sing-box。
func SingBoxRelease() (agentinstall.Release, bool) { return agentinstall.Release{}, false }

// CheckProxy 只有 Linux 用 sing-box。
func CheckProxy(string, agentproxy.Proxy) error { return ErrUnsupported }
