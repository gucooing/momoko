//go:build netbsd

package file

import "golang.org/x/sys/unix"

// diskUsage 返回 path 所在文件系统的总容量与可用容量（字节），失败返回 0。
// NetBSD 没有 statfs，用 statvfs；块数以 f_frsize 为单位。
func diskUsage(path string) (total, free uint64) {
	var st unix.Statvfs_t
	if err := unix.Statvfs(path, &st); err != nil {
		return 0, 0
	}
	unit := st.Frsize
	if unit == 0 {
		unit = st.Bsize
	}
	return st.Blocks * unit, st.Bavail * unit
}
