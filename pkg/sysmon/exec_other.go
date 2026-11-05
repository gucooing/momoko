//go:build !windows

package sysmon

import "os/exec"

func hideWindow(*exec.Cmd) {}
