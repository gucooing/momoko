//go:build !windows

package sysenv

// refresh 其它系统上 shell 自己会读配置文件（登录 shell 读 ~/.profile 之类），当前进程的环境原样用。
func refresh(base []string) []string { return base }
