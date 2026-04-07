package file

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

const (
	SortByName    = "name"
	SortByModTime = "modtime"

	SortAsc  = "asc"
	SortDesc = "desc"

	// MaxLoadFileSize 最大在线读取大小
	MaxLoadFileSize = 10 * 1024 * 1024
)

type FileOper struct {
	basePath string // 底线路径
}

func NewFileOper(basePath string) (*FileOper, error) {
	baseAbs, err := filepath.Abs(basePath)
	if err != nil {
		return nil, err
	}
	return &FileOper{
		basePath: baseAbs,
	}, nil
}

// ResolveRealPath 返回 targetPath 对应的真实路径。
// 如果目标越过 basePath，返回错误。
func (f *FileOper) ResolveRealPath(targetPath string) (string, error) {
	targetAbs, err := filepath.Abs(targetPath)
	if err != nil {
		return "", errors.New("路径不存在")
	}

	targetReal, err := filepath.EvalSymlinks(targetAbs)
	if err != nil {
		return "", errors.New("路径不存在")
	}

	rel, err := filepath.Rel(f.basePath, targetReal)
	if err != nil {
		return "", errors.New("路径不存在")
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("路径穿越！")
	}

	return targetReal, nil
}

// ListDir 获取文件列表
func (f *FileOper) ListDir(dir, field, order string) error {
	path, err := f.ResolveRealPath(dir)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].IsDir() != entries[j].IsDir() { // 文件夹和文件分开
			return entries[i].IsDir()
		}
		switch field {
		case SortByModTime: // 时间排序
			t1Info, err := entries[i].Info()
			if err != nil {
				return false
			}
			t2Info, err := entries[j].Info()
			if err != nil {
				return false
			}
			t1 := t1Info.ModTime()
			t2 := t2Info.ModTime()

			if !t1.Equal(t2) {
				if order == SortDesc {
					return t1.After(t2)
				}
				return t1.Before(t2)
			}

		case SortByName: // 名称排序
			fallthrough
		default:
		}

		n1 := entries[i].Name()
		n2 := entries[j].Name()

		if order == SortDesc {
			return n1 > n2
		}
		return n1 < n2
	})

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			return err
		}
		fullPath := filepath.Join(dir, entry.Name())
		if entry.IsDir() {
			fmt.Printf("[目录] %-50s %-20s\n", fullPath, info.ModTime().String())
		} else {
			fmt.Printf("[文件] %-50s %-20s\n", fullPath, info.ModTime().String())
		}
	}
	return nil
}

// LoadFile 读取文件
func (f *FileOper) LoadFile(path string) ([]byte, error) {
	path, err := f.ResolveRealPath(path)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, errors.New("文件不存在")
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if info.Size() > MaxLoadFileSize {
		return nil, errors.New("文件太大")
	}

	return io.ReadAll(file)
}
