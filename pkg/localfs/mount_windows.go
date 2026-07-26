//go:build windows

package localfs

import "os"

// systemHasVirtualRoot Windows 有「此电脑」这一层：空路径列出全部盘符。
func systemHasVirtualRoot() bool { return true }

// systemMounts 枚举当前存在的盘符根（"C:\"、"D:\" …）。
// 每个盘符是一个独立的 os.Root 挂载点；有意不枚举网络位置，
// UNC 路径在 splitMount 处已被拒绝（避免服务端向任意 SMB 主机发起认证）。
func systemMounts() []string {
	out := make([]string, 0, 26)
	for c := byte('A'); c <= 'Z'; c++ {
		p := string(c) + `:\`
		if _, err := os.Stat(p); err != nil {
			continue
		}
		out = append(out, p)
	}
	return out
}

// mountLabel 返回挂载点的展示名（"C:\" → "C:"）。
func mountLabel(mount string) string {
	if len(mount) >= 2 && mount[1] == ':' {
		return mount[:2]
	}
	return mount
}

// systemRootLabel 是整机视图虚拟根的展示名。
const systemRootLabel = "此电脑"
