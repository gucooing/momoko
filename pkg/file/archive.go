package file

import (
	"archive/zip"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

const (
	defaultCompressedSuffix = "-compressed.zip"
	defaultUnzipSuffix      = "-unzipped"
)

type archiveSource struct {
	path string
	info fs.FileInfo
}

func (f *FileOper) BatchCompress(paths []string, targetPath string) (string, error) {
	sources, err := f.resolveArchiveSources(paths)
	if err != nil {
		return "", err
	}

	targetAbs, err := f.resolveCompressedTargetPath(sources[0].path, targetPath)
	if err != nil {
		return "", err
	}
	if err := ensureArchiveTargetOutsideSources(targetAbs, sources); err != nil {
		return "", err
	}

	targetDir := filepath.Dir(targetAbs)
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	if info, err := os.Stat(targetAbs); err == nil && info.IsDir() {
		return "", errors.New("目标压缩文件路径不能是目录")
	}

	tempFile, err := os.CreateTemp(targetDir, "."+filepath.Base(targetAbs)+".*.tmp")
	if err != nil {
		return "", fmt.Errorf("创建临时压缩文件失败: %w", err)
	}

	tempPath := tempFile.Name()
	success := false
	defer func() {
		if success {
			return
		}
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
	}()

	zw := zip.NewWriter(tempFile)
	for _, source := range sources {
		if source.info.IsDir() {
			if err := f.compressDir(zw, source.path); err != nil {
				_ = zw.Close()
				return "", err
			}
			continue
		}
		if err := addFileToZip(zw, filepath.Dir(source.path), source.path); err != nil {
			_ = zw.Close()
			return "", err
		}
	}

	if err := zw.Close(); err != nil {
		return "", fmt.Errorf("压缩文件失败: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		return "", fmt.Errorf("关闭临时压缩文件失败: %w", err)
	}
	if err := os.Remove(targetAbs); err != nil && !os.IsNotExist(err) {
		return "", fmt.Errorf("清理旧压缩文件失败: %w", err)
	}
	if err := os.Rename(tempPath, targetAbs); err != nil {
		return "", fmt.Errorf("保存压缩文件失败: %w", err)
	}

	success = true
	return targetAbs, nil
}

func (f *FileOper) Unzip(sourcePath, targetPath string) (string, error) {
	archivePath, err := f.ResolveRealPath(sourcePath)
	if err != nil {
		return "", err
	}
	info, err := os.Stat(archivePath)
	if err != nil {
		return "", errors.New("压缩文件不存在")
	}
	if info.IsDir() {
		return "", errors.New("压缩文件路径不能是目录")
	}

	targetAbs, err := f.resolveUnzipTargetPath(archivePath, targetPath)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(targetAbs, 0o755); err != nil {
		return "", fmt.Errorf("创建解压目录失败: %w", err)
	}

	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", errors.New("打开压缩文件失败")
	}
	defer reader.Close()

	for _, entry := range reader.File {
		cleanName, err := sanitizeArchiveEntryName(entry.Name)
		if err != nil {
			return "", err
		}
		if cleanName == "" {
			continue
		}

		destPath := filepath.Join(targetAbs, filepath.FromSlash(cleanName))
		if !isSameOrWithinPath(targetAbs, destPath) {
			return "", errors.New("压缩包包含非法路径")
		}

		if entry.FileInfo().IsDir() {
			if err := os.MkdirAll(destPath, entry.Mode().Perm()); err != nil {
				return "", fmt.Errorf("创建目录失败: %w", err)
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(destPath), 0o755); err != nil {
			return "", fmt.Errorf("创建解压目录失败: %w", err)
		}

		if err := extractArchiveFile(entry, destPath); err != nil {
			return "", err
		}
	}

	return targetAbs, nil
}

func (f *FileOper) resolveArchiveSources(paths []string) ([]archiveSource, error) {
	if len(paths) == 0 {
		return nil, errors.New("请选择要压缩的文件或目录")
	}

	sources := make([]archiveSource, 0, len(paths))
	for _, rawPath := range paths {
		if strings.TrimSpace(rawPath) == "" {
			continue
		}
		absPath, err := f.ResolveRealPath(rawPath)
		if err != nil {
			return nil, err
		}
		info, err := os.Stat(absPath)
		if err != nil {
			return nil, errors.New("路径不存在")
		}
		sources = append(sources, archiveSource{
			path: absPath,
			info: info,
		})
	}
	if len(sources) == 0 {
		return nil, errors.New("请选择要压缩的文件或目录")
	}

	sort.Slice(sources, func(i, j int) bool {
		if len(sources[i].path) == len(sources[j].path) {
			return sources[i].path < sources[j].path
		}
		return len(sources[i].path) < len(sources[j].path)
	})

	filtered := make([]archiveSource, 0, len(sources))
	for _, candidate := range sources {
		skip := false
		for _, kept := range filtered {
			if isSameOrWithinPath(kept.path, candidate.path) {
				skip = true
				break
			}
		}
		if skip {
			continue
		}
		filtered = append(filtered, candidate)
	}

	return filtered, nil
}

func (f *FileOper) resolveCompressedTargetPath(sourcePath, targetPath string) (string, error) {
	if strings.TrimSpace(targetPath) != "" {
		return f.ResolveRealPath(targetPath)
	}

	baseName := filepath.Base(sourcePath)
	targetName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if targetName == "" {
		targetName = baseName
	}
	if targetName == "" || targetName == "." {
		targetName = "archive"
	}

	return filepath.Join(filepath.Dir(sourcePath), targetName+defaultCompressedSuffix), nil
}

func (f *FileOper) resolveUnzipTargetPath(sourcePath, targetPath string) (string, error) {
	if strings.TrimSpace(targetPath) != "" {
		return f.ResolveRealPath(targetPath)
	}

	baseName := filepath.Base(sourcePath)
	targetName := strings.TrimSuffix(baseName, filepath.Ext(baseName))
	if targetName == "" {
		targetName = baseName
	}
	if targetName == "" || targetName == "." {
		targetName = "archive"
	}

	return filepath.Join(filepath.Dir(sourcePath), targetName+defaultUnzipSuffix), nil
}

func (f *FileOper) compressDir(zw *zip.Writer, sourcePath string) error {
	parentPath := filepath.Dir(sourcePath)
	return filepath.WalkDir(sourcePath, func(currentPath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		return addEntryToZip(zw, parentPath, currentPath, entry)
	})
}

func addFileToZip(zw *zip.Writer, parentPath, filePath string) error {
	info, err := os.Stat(filePath)
	if err != nil {
		return errors.New("读取文件失败")
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return errors.New("创建压缩条目失败")
	}

	entryName, err := filepath.Rel(parentPath, filePath)
	if err != nil {
		return errors.New("生成压缩路径失败")
	}
	header.Name = filepath.ToSlash(entryName)
	header.Method = zip.Deflate
	header.SetMode(info.Mode())

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return errors.New("写入压缩条目失败")
	}

	file, err := os.Open(filePath)
	if err != nil {
		return errors.New("打开文件失败")
	}
	defer file.Close()

	if _, err = io.Copy(writer, file); err != nil {
		return errors.New("写入压缩内容失败")
	}

	return nil
}

func addEntryToZip(zw *zip.Writer, parentPath, currentPath string, entry fs.DirEntry) error {
	info, err := entry.Info()
	if err != nil {
		return errors.New("读取文件信息失败")
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return errors.New("创建压缩条目失败")
	}

	entryName, err := filepath.Rel(parentPath, currentPath)
	if err != nil {
		return errors.New("生成压缩路径失败")
	}
	header.Name = filepath.ToSlash(entryName)
	if entry.IsDir() && !strings.HasSuffix(header.Name, "/") {
		header.Name += "/"
	}
	header.Method = zip.Deflate
	header.SetMode(info.Mode())

	writer, err := zw.CreateHeader(header)
	if err != nil {
		return errors.New("写入压缩条目失败")
	}
	if entry.IsDir() {
		return nil
	}

	file, err := os.Open(currentPath)
	if err != nil {
		return errors.New("打开文件失败")
	}
	defer file.Close()

	if _, err = io.Copy(writer, file); err != nil {
		return errors.New("写入压缩内容失败")
	}

	return nil
}

func extractArchiveFile(entry *zip.File, destPath string) error {
	reader, err := entry.Open()
	if err != nil {
		return errors.New("打开压缩条目失败")
	}
	defer reader.Close()

	mode := entry.Mode().Perm()
	if mode == 0 {
		mode = 0o644
	}

	file, err := os.OpenFile(destPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return errors.New("创建解压文件失败")
	}
	defer file.Close()

	if _, err = io.Copy(file, reader); err != nil {
		return errors.New("写入解压文件失败")
	}

	return nil
}

func sanitizeArchiveEntryName(value string) (string, error) {
	normalized := strings.ReplaceAll(value, "\\", "/")
	cleanName := path.Clean(normalized)
	if cleanName == "." || cleanName == "" {
		return "", nil
	}
	if path.IsAbs(cleanName) {
		return "", errors.New("压缩包包含非法路径")
	}
	if filepath.VolumeName(cleanName) != "" {
		return "", errors.New("压缩包包含非法路径")
	}
	if strings.HasPrefix(cleanName, "../") || cleanName == ".." {
		return "", errors.New("压缩包包含非法路径")
	}

	return cleanName, nil
}

func ensureArchiveTargetOutsideSources(targetPath string, sources []archiveSource) error {
	for _, source := range sources {
		if isSameOrWithinPath(source.path, targetPath) {
			return errors.New("目标路径不能位于压缩源内部")
		}
	}
	return nil
}

func isSameOrWithinPath(rootPath, targetPath string) bool {
	relPath, err := filepath.Rel(rootPath, targetPath)
	if err != nil {
		return false
	}
	if relPath == "." {
		return true
	}
	if relPath == ".." {
		return false
	}
	return !strings.HasPrefix(relPath, ".."+string(filepath.Separator))
}
