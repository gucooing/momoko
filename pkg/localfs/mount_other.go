//go:build !windows

package localfs

// systemHasVirtualRoot 类 Unix 只有一个根，空路径直接等价于 "/"，没有虚拟根这一层。
func systemHasVirtualRoot() bool { return false }

// systemMounts 类 Unix 的唯一挂载点就是 "/"。
func systemMounts() []string { return []string{"/"} }

// mountLabel 返回挂载点的展示名。
func mountLabel(mount string) string { return mount }

// systemRootLabel 是整机视图虚拟根的展示名（类 Unix 下不会用到）。
const systemRootLabel = "/"
