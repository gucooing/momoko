//go:build linux || darwin || freebsd || dragonfly

// 只限这些平台：它们的 syscall.Statfs_t 才有 Bsize/Blocks/Bavail（NetBSD 没有 statfs，OpenBSD 字段名不同）。

package file

import "syscall"

// diskUsage 返回 path 所在文件系统的总容量与可用容量（字节），失败返回 0。
func diskUsage(path string) (total, free uint64) {
	var st syscall.Statfs_t
	if err := syscall.Statfs(path, &st); err != nil {
		return 0, 0
	}
	bsize := uint64(st.Bsize) //nolint:unconvert // Bsize 在不同平台类型不同
	return uint64(st.Blocks) * bsize, uint64(st.Bavail) * bsize
}
