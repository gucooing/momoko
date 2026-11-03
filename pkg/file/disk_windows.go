//go:build windows

package file

import "golang.org/x/sys/windows"

// diskUsage 返回 path 所在磁盘的总容量与可用容量（字节），失败返回 0。
func diskUsage(path string) (total, free uint64) {
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0
	}
	var avail, tot, totFree uint64
	if err := windows.GetDiskFreeSpaceEx(p, &avail, &tot, &totFree); err != nil {
		return 0, 0
	}
	return tot, avail
}
