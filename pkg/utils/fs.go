package utils

import (
	"errors"
	"io/fs"
	"math/rand/v2"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// CreateTemp 同 os.CreateTemp（pattern 中最后一个 "*" 替换为随机串，无 "*" 则追加在末尾），
// 区别是权限交给系统 umask（0666 &^ umask）而非固定 0600：临时文件最终会 rename 成正式文件。
func CreateTemp(dir, pattern string) (*os.File, error) {
	prefix, suffix := pattern, ""
	if i := strings.LastIndex(pattern, "*"); i >= 0 {
		prefix, suffix = pattern[:i], pattern[i+1:]
	}
	for range 100 {
		name := filepath.Join(dir, prefix+strconv.FormatUint(rand.Uint64(), 36)+suffix)
		f, err := os.OpenFile(name, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0o666)
		if errors.Is(err, fs.ErrExist) {
			continue
		}
		return f, err
	}
	return nil, errors.New("创建临时文件失败：重名过多")
}
