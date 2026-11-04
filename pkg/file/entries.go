package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"

	v1 "momoko/api/gen/v1"
)

const SortBySize = v1.FileSortField_FILE_SORT_FIELD_SIZE

// SortEntries 对条目排序：目录恒在文件之前；按名称 / 修改时间 / 大小排序，desc 为倒序；
// 同值时按名称升序兜底，保证翻页稳定。各来源 List 可能不保证顺序，统一在这里排。
func SortEntries(items []*v1.FileEntryInfo, field v1.FileSortField, desc bool) {
	sort.SliceStable(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if a.IsDir != b.IsDir {
			return a.IsDir
		}
		switch field {
		case SortByModTime:
			ta, tb := a.GetUpdateTime().AsTime(), b.GetUpdateTime().AsTime()
			if !ta.Equal(tb) {
				if desc {
					return ta.After(tb)
				}
				return ta.Before(tb)
			}
		case SortBySize:
			// 目录无自身大小：目录之间按名称排，文件之间按大小排。
			if !a.IsDir && a.Size != b.Size {
				if desc {
					return a.Size > b.Size
				}
				return a.Size < b.Size
			}
		default:
			if a.Name != b.Name {
				if desc {
					return a.Name > b.Name
				}
				return a.Name < b.Name
			}
		}
		return strings.ToLower(a.Name) < strings.ToLower(b.Name)
	})
}

// ChildCounter 是能廉价统计目录直接子项数的来源（本地磁盘）。远端来源逐目录列举代价过高，不实现。
type ChildCounter interface {
	CountChildren(ctx context.Context, dir string) (int64, bool)
}

// FillChildCounts 为条目中的目录填充直接子项数（来源支持时）；应只对当前页调用，避免整目录逐个统计。
func FillChildCounts(ctx context.Context, store Store, items []*v1.FileEntryInfo) {
	counter, ok := store.(ChildCounter)
	if !ok {
		return
	}
	for _, it := range items {
		if !it.IsDir {
			continue
		}
		if n, ok := counter.CountChildren(ctx, it.Path); ok {
			it.ChildCount = &n
		}
	}
}

// UsageReporter 是能报告自身已用空间 / 总容量的来源（本地磁盘、支持 RFC 4331 配额的 WebDAV）。
// total=0 表示没有容量上限或未知。
type UsageReporter interface {
	Usage(ctx context.Context) (used, total uint64, err error)
}

// CountChildren 统计本地目录的直接子项数（只读目录项名称，不 Stat）。
func (s *LocalStore) CountChildren(_ context.Context, dir string) (int64, bool) {
	real, err := s.oper.ResolveRealPath(dir)
	if err != nil {
		return 0, false
	}
	f, err := os.Open(real)
	if err != nil {
		return 0, false
	}
	defer f.Close()
	names, err := f.Readdirnames(-1)
	if err != nil {
		return 0, false
	}
	return int64(len(names)), true
}

// Usage 返回本地来源所在磁盘的已用 / 总容量：限定根目录的来源统计其根所在磁盘，整机来源统计服务工作目录所在磁盘。
func (s *LocalStore) Usage(_ context.Context) (used, total uint64, err error) {
	total, free := LocalDiskUsage(s.oper.basePath)
	if total < free {
		return 0, total, nil
	}
	return total - free, total, nil
}

// RealPath 把本地来源内的逻辑路径解析为真实路径（含穿越防护），供分片上传落地到目标目录旁。
func (s *LocalStore) RealPath(logical string) (string, error) {
	if s.oper.isSystemRoot(logical) {
		// 整机来源的虚拟根：Windows 为盘符列表（不可写入），类 Unix 为 "/"。
		if runtime.GOOS == "windows" {
			return "", errors.New("请先进入某个磁盘")
		}
		logical = "/"
	}
	return s.oper.ResolveRealPath(logical)
}

// LocalDiskUsage 返回 base 所在磁盘的总容量与可用容量；base 为空时取服务工作目录所在磁盘。
func LocalDiskUsage(base string) (total, free uint64) {
	if base == "" {
		wd, err := os.Getwd()
		if err != nil {
			return 0, 0
		}
		base = wd
	}
	abs, err := filepath.Abs(base)
	if err != nil {
		return 0, 0
	}
	return diskUsage(abs)
}
