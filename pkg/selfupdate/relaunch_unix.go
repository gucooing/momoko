//go:build unix

package selfupdate

import (
	"os"
	"syscall"
)

// relaunch 用 exec 把当前进程原地换成新程序：进程号、命令行参数、环境变量和工作目录都不变。
// Go 打开的文件和监听的端口都带 close-on-exec，新程序启动时它们已经关掉了。
func relaunch(exe string) error {
	args := append([]string{exe}, os.Args[1:]...)
	return syscall.Exec(exe, args, os.Environ())
}
