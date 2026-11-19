// Package agentfs 是 Agent 功能用到的文件系统操作：把路径关在某个目录里、列目录、读写文本、
// 按文件名搜（带一小段时间的缓存）、整目录复制。
package agentfs

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// CleanDir 把用户填的目录整理成绝对路径。
func CleanDir(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("先选一个目录")
	}
	if !filepath.IsAbs(path) {
		return "", errors.New("目录要填服务器上的绝对路径")
	}
	return filepath.Clean(path), nil
}

// IsDir 路径存在并且是目录。
func IsDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// SafeJoin 把相对路径接到 root 下面，不允许跳出 root。
func SafeJoin(root, rel string) (string, error) {
	rel = filepath.FromSlash(strings.TrimSpace(rel))
	if rel == "" || rel == "." {
		return root, nil
	}
	if filepath.IsAbs(rel) {
		return "", errors.New("路径不在这个目录里")
	}
	full := filepath.Join(root, rel)
	back, err := filepath.Rel(root, full)
	if err != nil || back == ".." || strings.HasPrefix(back, ".."+string(filepath.Separator)) {
		return "", errors.New("路径不在这个目录里")
	}
	return full, nil
}

// Rel 把 root 里面的绝对路径写成相对路径（正斜杠）；不在 root 里的原样返回。
func Rel(root, path string) string {
	if rel, err := filepath.Rel(root, path); err == nil && !strings.HasPrefix(rel, "..") && !filepath.IsAbs(rel) {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

// Entry 目录里的一项。
type Entry struct {
	Name string
	// Path 相对 root 的路径，正斜杠。
	Path    string
	IsDir   bool
	Size    int64
	ModTime time.Time
}

// List 列出 root 下某个子目录的内容：目录在前，按名称排序，不含 .git。
func List(root, rel string) ([]Entry, error) {
	dir, err := SafeJoin(root, rel)
	if err != nil {
		return nil, err
	}
	items, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, errors.New("目录不存在")
		}
		return nil, errors.New("读不了这个目录：" + err.Error())
	}
	out := make([]Entry, 0, len(items))
	for _, item := range items {
		if item.Name() == ".git" {
			continue
		}
		entry := Entry{
			Name:  item.Name(),
			Path:  filepath.ToSlash(filepath.Join(filepath.FromSlash(rel), item.Name())),
			IsDir: item.IsDir(),
		}
		if info, err := item.Info(); err == nil {
			if !item.IsDir() {
				entry.Size = info.Size()
			}
			entry.ModTime = info.ModTime()
			// 指向目录的符号链接按目录显示。
			if info.Mode()&os.ModeSymlink != 0 && IsDir(filepath.Join(dir, item.Name())) {
				entry.IsDir, entry.Size = true, 0
			}
		}
		out = append(out, entry)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].IsDir != out[j].IsDir {
			return out[i].IsDir
		}
		return strings.ToLower(out[i].Name) < strings.ToLower(out[j].Name)
	})
	return out, nil
}

// Dir 「浏览」里的一个子目录。
type Dir struct {
	Name string
	Path string
}

// Subdirs 列出某个目录的子目录，给「浏览」用。
// path 不存在时退回到最近的一层存在的目录（用户可能正打到一半）；为空或退到头时给根：
// Windows 上是各个盘符（此时 current 为空），其它系统是 /。
func Subdirs(path string) (current, parent string, dirs []Dir) {
	path = strings.TrimSpace(path)
	if path != "" && filepath.IsAbs(path) {
		path = filepath.Clean(path)
		for !IsDir(path) {
			up := filepath.Dir(path)
			if up == path {
				path = ""
				break
			}
			path = up
		}
	} else {
		path = ""
	}
	if path == "" {
		if runtime.GOOS == "windows" {
			for c := 'A'; c <= 'Z'; c++ {
				drive := string(c) + `:\`
				if _, err := os.Stat(drive); err == nil {
					dirs = append(dirs, Dir{Name: string(c) + ":", Path: drive})
				}
			}
			return "", "", dirs
		}
		path = "/"
	}
	current = path
	if up := filepath.Dir(path); up != path {
		parent = up
	}
	items, err := os.ReadDir(path)
	if err != nil {
		return current, parent, nil
	}
	for _, item := range items {
		name := item.Name()
		if strings.HasPrefix(name, ".") || strings.HasPrefix(name, "$") {
			continue
		}
		full := filepath.Join(path, name)
		if !item.IsDir() && (item.Type()&os.ModeSymlink == 0 || !IsDir(full)) {
			continue
		}
		dirs = append(dirs, Dir{Name: name, Path: full})
	}
	sort.SliceStable(dirs, func(i, j int) bool { return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name) })
	return current, parent, dirs
}

// IsText 内容是不是文本（合法的 UTF-8，没有 NUL）。
func IsText(raw []byte) bool {
	return utf8.Valid(raw) && !bytes.ContainsRune(raw, 0)
}

// Text 读一个文件给页面显示用的结果。
type Text struct {
	Content  string
	Size     int64
	Binary   bool
	TooLarge bool
}

// ReadText 读 root 里的一个文本文件；超过 max 字节或不是文本时不给内容。
func ReadText(root, rel string, max int64) (*Text, error) {
	full, err := SafeJoin(root, rel)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		return nil, errors.New("文件不存在")
	}
	out := &Text{Size: info.Size()}
	if info.Size() > max {
		out.TooLarge = true
		return out, nil
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return nil, errors.New("读不了这个文件：" + err.Error())
	}
	if !IsText(raw) {
		out.Binary = true
		return out, nil
	}
	out.Content = string(raw)
	return out, nil
}

// WriteText 把文本写进 root 里的一个文件。
func WriteText(root, rel, content string) error {
	full, err := SafeJoin(root, rel)
	if err != nil {
		return err
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		return errors.New("写入 " + rel + " 失败：" + err.Error())
	}
	return nil
}

// EnsureDir 保证目录存在，返回它的路径。
func EnsureDir(path string) string {
	_ = os.MkdirAll(path, 0o755)
	return path
}

// RemoveAll 删掉一个目录和里面的全部内容。
func RemoveAll(path string) { _ = os.RemoveAll(path) }

// Move 把一个文件或目录挪到另一个位置，目标已存在时先删掉；不在同一个盘上挪不了时复制过去再删掉原来的。
func Move(from, to string) error {
	_ = os.RemoveAll(to)
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		return err
	}
	if os.Rename(from, to) == nil {
		return nil
	}
	var err error
	if IsDir(from) {
		err = CopyDir(from, to)
	} else {
		err = CopyFile(from, to)
	}
	if err != nil {
		_ = os.RemoveAll(to)
		return err
	}
	return os.RemoveAll(from)
}

// Save 把一段数据存进 dir 下新建的一个子目录（只有本用户可读），返回完整路径。
// 每次一个子目录：同名文件互不覆盖，文件名又保持原样。
func Save(dir, name string, data []byte) (string, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	sub, err := os.MkdirTemp(dir, "")
	if err != nil {
		return "", err
	}
	full := filepath.Join(sub, filepath.Base(name))
	if err := os.WriteFile(full, data, 0o600); err != nil {
		_ = os.RemoveAll(sub)
		return "", err
	}
	return full, nil
}

// ReadBlob 读 root 里的一个文件，连同按扩展名（认不出时按内容）判断的类型；超过 max 字节不读。
func ReadBlob(root, rel string, max int64) ([]byte, string, error) {
	full, err := SafeJoin(root, rel)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		return nil, "", errors.New("文件不存在")
	}
	if info.Size() > max {
		return nil, "", errors.New("文件太大了")
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return nil, "", errors.New("读不了这个文件：" + err.Error())
	}
	kind := mime.TypeByExtension(filepath.Ext(full))
	if kind == "" {
		kind = http.DetectContentType(raw)
	}
	return raw, kind, nil
}

// Empty 目录不存在或里面没有东西。
func Empty(path string) bool {
	items, err := os.ReadDir(path)
	return err != nil || len(items) == 0
}

// Walk 列出 root 里的文件（相对路径），跳过明显不该翻的大目录，最多 limit 个。给不是 Git 仓库的目录搜文件用。
func Walk(root string, limit int) []string {
	var files []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", "dist", "build", "target", ".venv", "__pycache__":
				return filepath.SkipDir
			}
			return nil
		}
		if len(files) >= limit {
			return filepath.SkipAll
		}
		if rel, err := filepath.Rel(root, path); err == nil {
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	return files
}

// CopyTree 把一个目录整份复制过去（不带 .git、node_modules 和符号链接），最多 maxFiles 个文件，返回文件数。
func CopyTree(src, dst string, maxFiles int) (int, error) {
	count := 0
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if d.IsDir() {
			if d.Name() == ".git" || d.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(dst, rel), 0o755)
		}
		if d.Type()&os.ModeSymlink != 0 || count >= maxFiles {
			return nil
		}
		if err := CopyFile(path, filepath.Join(dst, rel)); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

// CopyDir 把一个目录原样整份复制过去，.git、node_modules 也带上；符号链接照建一个（建不了的跳过）。
// 源目录不存在时只建出空的目标目录。
func CopyDir(src, dst string) error {
	if !IsDir(src) {
		return os.MkdirAll(dst, 0o755)
	}
	return filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		switch {
		case d.IsDir():
			return os.MkdirAll(target, 0o755)
		case d.Type()&os.ModeSymlink != 0:
			if link, err := os.Readlink(path); err == nil {
				_ = os.Symlink(link, target)
			}
			return nil
		default:
			return CopyFile(path, target)
		}
	})
}

// CopyFile 复制一个文件，保留「能不能执行」；目标所在目录不存在时建出来。
func CopyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	info, err := in.Stat()
	if err != nil {
		return err
	}
	mode := os.FileMode(0o644)
	if info.Mode()&0o111 != 0 {
		mode = 0o755
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// PruneEmptyDirs 删掉 root 里已经空了的子目录。
func PruneEmptyDirs(root string) {
	var dirs []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err == nil && d.IsDir() && path != root {
			dirs = append(dirs, path)
		}
		return nil
	})
	for i := len(dirs) - 1; i >= 0; i-- {
		_ = os.Remove(dirs[i])
	}
}

// Slug 把名字变成能当目录名和分支名的一段。
func Slug(name string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9', r == '-', r == '_', r == '.':
			b.WriteRune(r)
		case r == ' ' || r == '/':
			b.WriteRune('-')
		}
	}
	s := strings.Trim(b.String(), "-.")
	if s == "" {
		return "ws"
	}
	return s
}
