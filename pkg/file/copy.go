package file

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	v1 "momoko/api/gen/v1"
)

const copyNameSuffix = "-copy"

// CopyToDir copies files or directories into targetDir.
func (f *FileOper) CopyToDir(paths []string, targetDir string) []*v1.FileOperationResult {
	return f.transferToDir(paths, targetDir, f.resolveCopyTargetPath, f.copyPath)
}

// MoveToDir moves files or directories into targetDir.
func (f *FileOper) MoveToDir(paths []string, targetDir string) []*v1.FileOperationResult {
	return f.transferToDir(paths, targetDir, f.resolveMoveTargetPath, f.movePath)
}

func (f *FileOper) transferToDir(
	paths []string,
	targetDir string,
	resolveTarget func(string, string) (string, error),
	transfer func(string, string) error,
) []*v1.FileOperationResult {
	results := make([]*v1.FileOperationResult, 0, len(paths))
	for _, path := range paths {
		result := &v1.FileOperationResult{Path: path}
		if strings.TrimSpace(path) == "" {
			result.Message = "路径不能为空"
			results = append(results, result)
			continue
		}

		sourcePath, err := f.ResolveRealPath(path)
		if err != nil {
			result.Message = err.Error()
			results = append(results, result)
			continue
		}
		targetPath, err := resolveTarget(sourcePath, targetDir)
		if err != nil {
			result.Message = err.Error()
			results = append(results, result)
			continue
		}
		if err = transfer(sourcePath, targetPath); err != nil {
			result.Message = err.Error()
			results = append(results, result)
			continue
		}

		result.Success = true
		result.Message = targetPath
		results = append(results, result)
	}
	return results
}

func (f *FileOper) resolveCopyTargetPath(sourcePath, targetDir string) (string, error) {
	targetDirPath, sourceInfo, err := f.resolveTransferPaths(sourcePath, targetDir)
	if err != nil {
		return "", err
	}
	if sourceInfo.IsDir() && isSameOrWithinPath(sourcePath, targetDirPath) {
		return "", errors.New("目标目录不能位于源目录内部")
	}

	return nextCopyPath(targetDirPath, filepath.Base(sourcePath), sameFilePath(filepath.Dir(sourcePath), targetDirPath))
}

func (f *FileOper) resolveMoveTargetPath(sourcePath, targetDir string) (string, error) {
	targetDirPath, sourceInfo, err := f.resolveTransferPaths(sourcePath, targetDir)
	if err != nil {
		return "", err
	}
	if sourceInfo.IsDir() && isSameOrWithinPath(sourcePath, targetDirPath) {
		return "", errors.New("目标目录不能位于源目录内部")
	}

	targetPath := filepath.Join(targetDirPath, filepath.Base(sourcePath))
	if sameFilePath(sourcePath, targetPath) {
		return sourcePath, nil
	}
	if _, err := os.Stat(targetPath); os.IsNotExist(err) {
		return targetPath, nil
	} else if err != nil {
		return "", err
	}

	return nextCopyPath(targetDirPath, filepath.Base(sourcePath), true)
}

func (f *FileOper) resolveTransferPaths(sourcePath, targetDir string) (string, os.FileInfo, error) {
	targetDirPath, err := f.ResolveRealPath(targetDir)
	if err != nil {
		return "", nil, err
	}
	targetInfo, err := os.Stat(targetDirPath)
	if err != nil {
		return "", nil, errors.New("目标目录不存在")
	}
	if !targetInfo.IsDir() {
		return "", nil, errors.New("目标路径不是目录")
	}

	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		return "", nil, errors.New("源路径不存在")
	}

	return targetDirPath, sourceInfo, nil
}

func nextCopyPath(targetDir, name string, forceCopySuffix bool) (string, error) {
	if name == "" || name == "." || name == string(filepath.Separator) {
		return "", errors.New("源名称无效")
	}

	if !forceCopySuffix {
		targetPath := filepath.Join(targetDir, name)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			return targetPath, nil
		} else if err != nil {
			return "", err
		}
	}

	ext := filepath.Ext(name)
	base := strings.TrimSuffix(name, ext)
	if base == "" {
		base = name
		ext = ""
	}

	for index := 1; ; index++ {
		targetName := base + copyNameSuffix + ext
		if index > 1 {
			targetName = fmt.Sprintf("%s%s-%d%s", base, copyNameSuffix, index, ext)
		}

		targetPath := filepath.Join(targetDir, targetName)
		if _, err := os.Stat(targetPath); os.IsNotExist(err) {
			return targetPath, nil
		} else if err != nil {
			return "", err
		}
	}
}

func (f *FileOper) copyPath(sourcePath, targetPath string) error {
	info, err := os.Stat(sourcePath)
	if err != nil {
		return errors.New("源路径不存在")
	}
	if info.IsDir() {
		return f.copyDir(sourcePath, targetPath, info.Mode().Perm())
	}
	return copyFile(sourcePath, targetPath, info.Mode().Perm())
}

func (f *FileOper) movePath(sourcePath, targetPath string) error {
	if sameFilePath(sourcePath, targetPath) {
		return nil
	}
	if err := os.Rename(sourcePath, targetPath); err == nil {
		return nil
	} else if !isCrossDeviceError(err) {
		return err
	}

	if err := f.copyPath(sourcePath, targetPath); err != nil {
		return err
	}
	if err := os.RemoveAll(sourcePath); err != nil {
		return fmt.Errorf("删除源路径失败: %w", err)
	}
	return nil
}

func (f *FileOper) copyDir(sourcePath, targetPath string, mode fs.FileMode) error {
	if err := os.MkdirAll(targetPath, mode); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	return filepath.WalkDir(sourcePath, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if currentPath == sourcePath {
			return nil
		}
		if _, err := f.ResolveRealPath(currentPath); err != nil {
			return err
		}

		relPath, err := filepath.Rel(sourcePath, currentPath)
		if err != nil {
			return errors.New("生成复制路径失败")
		}
		destPath := filepath.Join(targetPath, relPath)
		info, err := entry.Info()
		if err != nil {
			return errors.New("读取文件信息失败")
		}
		if entry.IsDir() {
			if err = os.MkdirAll(destPath, info.Mode().Perm()); err != nil {
				return fmt.Errorf("创建目标目录失败: %w", err)
			}
			return nil
		}

		return copyFile(currentPath, destPath, info.Mode().Perm())
	})
}

func copyFile(sourcePath, targetPath string, mode fs.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o755); err != nil {
		return fmt.Errorf("创建目标目录失败: %w", err)
	}

	source, err := os.Open(sourcePath)
	if err != nil {
		return errors.New("打开源文件失败")
	}
	defer source.Close()

	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_EXCL, mode)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}

	success := false
	defer func() {
		_ = target.Close()
		if !success {
			_ = os.Remove(targetPath)
		}
	}()

	if _, err = io.Copy(target, source); err != nil {
		return errors.New("复制文件失败")
	}
	if err = target.Close(); err != nil {
		return errors.New("保存目标文件失败")
	}

	success = true
	return nil
}

func sameFilePath(a, b string) bool {
	a = filepath.Clean(a)
	b = filepath.Clean(b)
	if runtime.GOOS == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

func isCrossDeviceError(err error) bool {
	if linkErr, ok := err.(*os.LinkError); ok {
		err = linkErr.Err
	}
	return strings.Contains(strings.ToLower(err.Error()), "cross-device") ||
		strings.Contains(strings.ToLower(err.Error()), "invalid cross-device link")
}
