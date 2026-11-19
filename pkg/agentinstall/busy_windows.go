//go:build windows

package agentinstall

import (
	"errors"
	"syscall"
)

const (
	errorAccessDenied     = syscall.Errno(5)
	errorSharingViolation = syscall.Errno(32)
	errorLockViolation    = syscall.Errno(33)
)

// errDiskFull 是 Windows 说磁盘满了的错误码。
var errDiskFull error = syscall.Errno(112)

// inUse 打开文件失败是不是因为它正被占用（程序还在运行、别的进程开着它）。
func inUse(err error) bool {
	return errors.Is(err, errorSharingViolation) || errors.Is(err, errorLockViolation)
}

// locked 改名失败是不是因为里面的文件正被占用。这时 Windows 也常报「拒绝访问」，所以它也算。
func locked(err error) bool {
	return inUse(err) || errors.Is(err, errorAccessDenied)
}
