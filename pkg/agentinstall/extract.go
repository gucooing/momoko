package agentinstall

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"syscall"

	"momoko/pkg/archive"
)

// 一个安装包最多解出这么多：正常的在 500 MB 上下，防的是压缩炸弹。
const maxExtracted = 4 << 30

func extract(ctx context.Context, path, name, dest string) error {
	return archive.Extract(ctx, path, name, dest, maxExtracted)
}

// reason 把文件系统的错误说成人话，不带路径。
func reason(err error) string {
	switch {
	case errors.Is(err, fs.ErrPermission):
		return "没有权限"
	case errors.Is(err, syscall.ENOSPC), errors.Is(err, errDiskFull):
		return "磁盘空间不够"
	}
	// 去掉外面那层「操作 路径:」，只留系统给的原因。
	var pathErr *fs.PathError
	if errors.As(err, &pathErr) {
		return pathErr.Err.Error()
	}
	var linkErr *os.LinkError
	if errors.As(err, &linkErr) {
		return linkErr.Err.Error()
	}
	return err.Error()
}
