package localfs

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
)

// 本包对外暴露的错误哨兵：业务层可用 errors.Is 判定并映射为各自的响应码。
// 错误文案面向最终用户，不含真实路径等内部信息，避免通过报错反推服务器目录结构。
var (
	// ErrTraversal 请求路径试图越出当前视图边界（".."、绝对路径、符号链接逃逸等）。
	ErrTraversal = errors.New("非法路径：越界访问已被拒绝")
	// ErrInvalidPath 路径本身不合法（空段、控制字符、超长等）。
	ErrInvalidPath = errors.New("非法路径")
	// ErrInvalidName 文件名不合法（含分隔符、保留设备名、结尾点或空格等）。
	ErrInvalidName = errors.New("非法文件名")
	// ErrNotExist 目标不存在。
	ErrNotExist = fs.ErrNotExist
	// ErrExist 目标已存在。
	ErrExist = fs.ErrExist
	// ErrIsDir 目标是目录，但此操作要求文件。
	ErrIsDir = errors.New("目标是目录")
	// ErrNotDir 目标不是目录，但此操作要求目录。
	ErrNotDir = errors.New("目标不是目录")
	// ErrTooLarge 超出策略允许的大小上限。
	ErrTooLarge = errors.New("文件超出大小上限")
	// ErrDenied 目标命中保护清单（如 momoko 自身的数据与配置目录）。
	ErrDenied = errors.New("目标受保护，禁止访问")
	// ErrReadOnly 当前视图为只读。
	ErrReadOnly = errors.New("只读视图，禁止写入")
	// ErrRootScope 试图对视图根自身做删除/重命名等破坏性操作。
	ErrRootScope = errors.New("不允许操作根目录自身")
	// ErrArchiveLimit 压缩包超出解压上限（条目数/总体积/压缩比），疑似 zip 炸弹。
	ErrArchiveLimit = errors.New("压缩包超出解压上限")
	// ErrBadArchive 压缩包内容非法（含绝对路径、".." 或不支持的条目类型）。
	ErrBadArchive = errors.New("压缩包内容非法")
	// ErrSystemScope 该操作在整机视图的虚拟根（此电脑）上无意义。
	ErrSystemScope = errors.New("请先选择一个磁盘")
)

// escaped 判断一个 os.Root 返回的错误是否为越界拒绝。
// os.Root 越界时返回的是包装了 "path escapes from parent" 的 *PathError，
// 且刻意不满足 fs.ErrNotExist，以便与「确实不存在」区分开。
func escaped(err error) bool {
	if err == nil {
		return false
	}
	if errors.Is(err, ErrTraversal) {
		return true
	}
	if pe, ok := errors.AsType[*os.PathError](err); ok {
		return pe.Err != nil && pe.Err.Error() == "path escapes from parent"
	}
	return false
}

// sanitize 把底层错误转换为对外安全的错误：
// 越界统一为 ErrTraversal（且不回显真实路径），其余剥掉 *PathError 外壳只保留语义，
// 使调用方既能 errors.Is 判定，又不会把服务器目录结构写进 HTTP 响应。
func sanitize(err error) error {
	if err == nil {
		return nil
	}
	if escaped(err) {
		return ErrTraversal
	}
	switch {
	case errors.Is(err, fs.ErrNotExist):
		return ErrNotExist
	case errors.Is(err, fs.ErrExist):
		return ErrExist
	case errors.Is(err, fs.ErrPermission):
		return fmt.Errorf("权限不足")
	}
	if pe, ok := errors.AsType[*os.PathError](err); ok {
		return fmt.Errorf("%s 失败: %w", pe.Op, pe.Err)
	}
	if le, ok := errors.AsType[*os.LinkError](err); ok {
		return fmt.Errorf("%s 失败: %w", le.Op, le.Err)
	}
	return err
}
