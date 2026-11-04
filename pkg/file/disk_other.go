//go:build !linux && !darwin && !freebsd && !dragonfly && !netbsd && !windows

package file

// diskUsage 在不支持的平台上返回 0（未知）。
func diskUsage(string) (total, free uint64) { return 0, 0 }
