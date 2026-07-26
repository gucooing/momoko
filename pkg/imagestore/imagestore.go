// Package imagestore 封装生图图片的磁盘读写：写入图片、按相对路径打开、删除。
//
// 全部落到 pkg/localfs 的一个受限视图内，因此路径成分永远出不了 data/imagine：
// 这里的 userID 来自上游 sub2api 站点的响应、generationID 来自任务记录，两者都不是本系统可信的输入，
// 而写盘路径正是由它们拼出来的——旧实现直接 filepath.Join，一个 "../../configs" 就能写到别处。
package imagestore

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"momoko/pkg/localfs"
)

// Root 生图文件存储根目录（相对工作目录，落于 data 目录下）。
const Root = "./data/imagine"

// maxImageSize 单张图片的体积上限。
const maxImageSize = 64 << 20

var (
	once sync.Once
	view *localfs.FS
	err0 error
)

// fs 惰性打开受限视图（首次使用时创建目录）。
func fs() (*localfs.FS, error) {
	once.Do(func() {
		view, err0 = localfs.OpenDir(Root, localfs.WithMaxFileSize(maxImageSize))
	})
	return view, err0
}

// EnsureDir 在进程启动时创建存储根目录。
func EnsureDir() error {
	_, err := fs()
	return err
}

// SaveImage 将图片字节写入磁盘，返回相对存储根的 path（入库用）与文件名。
//
// userID / generationID 先经 localfs.SafeName 压成单层安全名称：它们来自外部系统，
// 不能直接充当目录名；压平后既不含分隔符也不含 ".."，再叠加受限视图形成双保险。
func SaveImage(userID, generationID string, index int, data []byte, outputFormat string) (relPath, filename string, err error) {
	view, err := fs()
	if err != nil {
		return "", "", err
	}
	ext := "png"
	switch strings.ToLower(strings.TrimSpace(outputFormat)) {
	case "jpg", "jpeg":
		ext = "jpg"
	case "webp":
		ext = "webp"
	}
	user := localfs.SafeName(userID, "anonymous")
	gen := localfs.SafeName(generationID, "unknown")
	if index < 0 {
		index = 0
	}
	filename = fmt.Sprintf("%d.%s", index, ext)

	relPath = user + "/" + gen + "/" + filename
	if err := view.MkdirAll(user + "/" + gen); err != nil {
		return "", "", err
	}
	if _, err := view.WriteFile(relPath, bytes.NewReader(data)); err != nil {
		return "", "", err
	}
	return relPath, filename, nil
}

// Open 以只读方式打开相对路径对应的文件，供 http.ServeContent 使用。
// relPath 来自数据库，仍按不可信输入对待：越界一律失败。
func Open(relPath string) (io.ReadSeekCloser, os.FileInfo, error) {
	view, err := fs()
	if err != nil {
		return nil, nil, err
	}
	f, _, err := view.OpenRead(relPath)
	if err != nil {
		return nil, nil, err
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	return f, info, nil
}

// Remove 删除相对路径对应的文件（不存在视为成功）。
func Remove(relPath string) error {
	view, err := fs()
	if err != nil {
		return err
	}
	res := view.Remove([]string{relPath})
	if len(res) > 0 && !res[0].OK {
		return fmt.Errorf("删除图片失败: %s", res[0].Message)
	}
	return nil
}
