//go:build !windows

package selfupdate

// ShouldSupervise 只有 Windows 需要守护进程；其它系统更新时用 exec 原地重启。
func ShouldSupervise() bool { return false }

// Supervise 其它系统不会调用。
func Supervise() int { return 0 }
