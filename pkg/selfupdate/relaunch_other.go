//go:build !unix

package selfupdate

import (
	"os"
	"os/exec"
)

// relaunch Windows 没有 exec。由守护进程启动的（正常情况）：以 RestartExitCode 退出，守护进程马上启动换好的新程序。
// 没有守护进程时（开发构建）：用同样的参数、环境和工作目录启动新程序，共用当前控制台，然后退出当前进程。
func relaunch(exe string) error {
	if supervised {
		os.Exit(RestartExitCode)
	}
	cmd := exec.Command(exe, os.Args[1:]...)
	cmd.Stdin, cmd.Stdout, cmd.Stderr = os.Stdin, os.Stdout, os.Stderr
	cmd.Env = os.Environ()
	if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	os.Exit(0)
	return nil
}
