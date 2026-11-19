//go:build unix

package agentinstall

import (
	"errors"
	"syscall"
)

// errDiskFull 磁盘满了。
var errDiskFull error = syscall.ENOSPC

// inUse 打开文件失败是不是因为它正被占用。
func inUse(err error) bool {
	return errors.Is(err, syscall.ETXTBSY) || errors.Is(err, syscall.EBUSY)
}

// locked 改名失败是不是因为里面的文件正被占用。Unix 上运行中的程序照样能改名、删除，基本遇不到。
func locked(err error) bool { return inUse(err) }
