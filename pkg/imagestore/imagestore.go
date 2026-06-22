// Package imagestore 封装生图图片的磁盘读写：写入 base64 图片、按相对路径打开、删除、解析磁盘路径。
// 业务层（biz）通过本包操作文件，自身不直接调用 os/filepath。
package imagestore

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// root 生图文件存储根目录（相对工作目录，落于 data 目录下）。
const root = "./data/imagine"

// EnsureDir 在进程启动时创建存储根目录。
func EnsureDir() error {
	return os.MkdirAll(root, 0755)
}

// SaveImage 将图片字节写入磁盘，返回相对存储根的 path（入库用）与文件名。
func SaveImage(userID, generationID string, index int, data []byte, outputFormat string) (relPath, filename string, err error) {
	ext := "png"
	switch strings.ToLower(strings.TrimSpace(outputFormat)) {
	case "jpg", "jpeg":
		ext = "jpg"
	case "webp":
		ext = "webp"
	}
	dir := filepath.Join(root, userID, generationID)
	if err = os.MkdirAll(dir, 0755); err != nil {
		return "", "", err
	}
	filename = fmt.Sprintf("%d.%s", index, ext)
	if err = os.WriteFile(filepath.Join(dir, filename), data, 0644); err != nil {
		return "", "", err
	}
	relPath = filepath.Join(userID, generationID, filename)
	return relPath, filename, nil
}

// DiskPath 返回相对路径对应的磁盘绝对路径。
func DiskPath(relPath string) string {
	return filepath.Join(root, relPath)
}

// Open 以只读方式打开相对路径对应的文件，供 http.ServeContent 使用。
func Open(relPath string) (io.ReadSeekCloser, os.FileInfo, error) {
	fs, err := os.Open(DiskPath(relPath))
	if err != nil {
		return nil, nil, err
	}
	info, err := fs.Stat()
	if err != nil {
		fs.Close()
		return nil, nil, err
	}
	return fs, info, nil
}

// Remove 删除相对路径对应的文件（不存在视为成功）。
func Remove(relPath string) error {
	return os.Remove(DiskPath(relPath))
}
