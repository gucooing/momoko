package localfs

import (
	"io/fs"
	"strings"
	"time"
)

// isInternalName 报告某个条目是否是 momoko 自己的中间产物
// （原子写入的临时文件、分片上传的缓冲文件）。
//
// 这类文件必须与目标同目录，收尾的 rename 才是同卷、瞬时且原子的；
// 但它们是纯粹的实现细节，不该出现在用户眼前。因此不搬家、只隐藏：
// 列目录与搜索都跳过它们，性能一分不让，观感上也不再散落各处。
func isInternalName(name string) bool {
	return strings.HasPrefix(name, tempPrefix) || strings.HasPrefix(name, BufferPrefix)
}

// Entry 是一个文件或目录条目。纯 Go 类型，不依赖 protobuf，由调用方自行映射为对外结构。
type Entry struct {
	// Name 条目名（末段）。
	Name string
	// Path 真实绝对路径，可直接作为后续请求的 vpath 回传。
	Path string
	// IsDir 是否目录。符号链接即便指向目录也为 false（按 lstat 语义，不跟随链接）。
	IsDir bool
	// IsSymlink 是否符号链接 / Windows 重解析点。
	IsSymlink bool
	// Size 字节数（目录的值由文件系统决定，不代表递归体积）。
	Size int64
	// Mode 文件模式位。
	Mode fs.FileMode
	// ModTime 最后修改时间。
	ModTime time.Time
	// UID/GID/User/Group 属主信息，仅类 Unix 有值。
	UID   uint32
	GID   uint32
	User  string
	Group string
}

// Perm 返回权限位的字符串表示（如 "-rw-r--r--"）。
// 只取权限位、不含类型字符，与前端既有展示保持一致（目录是否为目录另有 IsDir 字段）。
func (e *Entry) Perm() string { return e.Mode.Perm().String() }

// DirStat 是一个目录的概览统计。
type DirStat struct {
	Name       string
	Path       string
	ParentPath string
	Dirs       int64
	Files      int64
	Items      int64
}

// Result 是批量操作中单个目标的结果。失败时 Message 为面向用户的原因，
// 成功时 Message 视操作而定（复制/移动会回填目标路径）。
type Result struct {
	Path    string
	OK      bool
	Message string
}

// SortField 是列目录的排序字段。
type SortField uint8

const (
	// SortByName 按名称排序（默认）。
	SortByName SortField = iota
	// SortByModTime 按修改时间排序。
	SortByModTime
	// SortBySize 按体积排序。
	SortBySize
)

// ListOptions 是列目录选项。目录恒排在文件之前，不受排序字段影响。
type ListOptions struct {
	Sort SortField
	Desc bool
}

// SearchOptions 是搜索选项。Keywords 为空表示匹配全部。
type SearchOptions struct {
	Keywords string
	// Recursive 为 false 时只搜索当前目录一层。
	Recursive bool
	// Limit 覆盖策略里的返回条目上限（<=0 表示沿用策略值）。
	Limit int
}

// newEntry 由 FileInfo 与真实路径构造条目。
func newEntry(info fs.FileInfo, real string) *Entry {
	e := &Entry{
		Name:      info.Name(),
		Path:      real,
		IsDir:     info.IsDir(),
		IsSymlink: info.Mode()&fs.ModeSymlink != 0,
		Size:      info.Size(),
		Mode:      info.Mode(),
		ModTime:   info.ModTime(),
	}
	if e.Size < 0 {
		e.Size = 0
	}
	fillOwner(e, info)
	return e
}

// mountEntry 把一个挂载点（盘符根）包装为条目，用于「此电脑」虚拟根的列表。
func mountEntry(mount string, info fs.FileInfo) *Entry {
	e := &Entry{
		Name:  mountLabel(mount),
		Path:  mount,
		IsDir: true,
		Mode:  fs.ModeDir | 0o777,
	}
	if info != nil {
		e.ModTime = info.ModTime()
	}
	return e
}
