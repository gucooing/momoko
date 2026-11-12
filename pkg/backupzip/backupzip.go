// Package backupzip 把一个目录下的若干相对路径打包成 zip，并能把 zip 解回原位置。
// 打包时可以跳过指定文件、追加替代文件（例如数据库的一致性快照），解包时拒绝越出目标目录的条目。
package backupzip

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Extra 是额外放进 zip 的文件：Name 为 zip 内路径（/ 分隔），Path 为磁盘上的文件。
type Extra struct {
	Name string
	Path string
}

// Options 打包选项。
type Options struct {
	// Skip 返回 true 时跳过该文件（rel 为相对 base 的 / 分隔路径）。
	Skip func(rel string) bool
	// Extras 额外加入的文件。
	Extras []Extra
	// Progress 每写完一个文件回调（累计文件数、累计字节）。
	Progress func(files int, bytes int64)
}

// Create 把 base 下的 paths（文件或目录，相对 base）打包到 target（先写临时文件，成功后改名）。
// 不存在的路径跳过。返回写入的文件数与 zip 大小。
func Create(ctx context.Context, base string, paths []string, target string, opt Options) (int, int64, error) {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return 0, 0, err
	}
	tmp := target + ".part"
	f, err := os.Create(tmp)
	if err != nil {
		return 0, 0, err
	}
	zw := zip.NewWriter(f)
	files := 0
	var written int64
	add := func(name, path string, info fs.FileInfo) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		h, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}
		h.Name = name
		h.Method = zip.Deflate
		w, err := zw.CreateHeader(h)
		if err != nil {
			return err
		}
		src, err := os.Open(path)
		if err != nil {
			return err
		}
		n, err := io.Copy(w, src)
		_ = src.Close()
		if err != nil {
			return err
		}
		files++
		written += n
		if opt.Progress != nil {
			opt.Progress(files, written)
		}
		return nil
	}
	fail := func(err error) (int, int64, error) {
		_ = zw.Close()
		_ = f.Close()
		_ = os.Remove(tmp)
		return 0, 0, err
	}
	for _, p := range paths {
		root := filepath.Join(base, filepath.FromSlash(p))
		st, err := os.Stat(root)
		if errors.Is(err, os.ErrNotExist) {
			continue
		}
		if err != nil {
			return fail(err)
		}
		if !st.IsDir() {
			rel := filepath.ToSlash(p)
			if opt.Skip != nil && opt.Skip(rel) {
				continue
			}
			if err := add(rel, root, st); err != nil {
				return fail(err)
			}
			continue
		}
		err = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !d.Type().IsRegular() {
				return nil
			}
			relPath, err := filepath.Rel(base, path)
			if err != nil {
				return err
			}
			rel := filepath.ToSlash(relPath)
			if opt.Skip != nil && opt.Skip(rel) {
				return nil
			}
			info, err := d.Info()
			if err != nil {
				return err
			}
			return add(rel, path, info)
		})
		if err != nil {
			return fail(err)
		}
	}
	for _, e := range opt.Extras {
		st, err := os.Stat(e.Path)
		if err != nil {
			return fail(err)
		}
		if err := add(e.Name, e.Path, st); err != nil {
			return fail(err)
		}
	}
	if err := zw.Close(); err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return 0, 0, err
	}
	st, err := f.Stat()
	if err != nil {
		_ = f.Close()
		_ = os.Remove(tmp)
		return 0, 0, err
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(tmp)
		return 0, 0, err
	}
	if err := os.Rename(tmp, target); err != nil {
		_ = os.Remove(tmp)
		return 0, 0, err
	}
	return files, st.Size(), nil
}

// Entries 列出 zip 内的文件名（/ 分隔）。
func Entries(path string) ([]string, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return nil, err
	}
	defer zr.Close()
	out := make([]string, 0, len(zr.File))
	for _, f := range zr.File {
		if !f.FileInfo().IsDir() {
			out = append(out, f.Name)
		}
	}
	return out, nil
}

// Extract 把 zip 解到 base 下（覆盖同名文件）；条目路径越出 base 时报错。
func Extract(ctx context.Context, path, base string) (int, error) {
	zr, err := zip.OpenReader(path)
	if err != nil {
		return 0, err
	}
	defer zr.Close()
	cleanBase, err := filepath.Abs(base)
	if err != nil {
		return 0, err
	}
	n := 0
	for _, f := range zr.File {
		if err := ctx.Err(); err != nil {
			return n, err
		}
		target := filepath.Join(cleanBase, filepath.FromSlash(f.Name))
		if target != cleanBase && !strings.HasPrefix(target, cleanBase+string(os.PathSeparator)) {
			return n, fmt.Errorf("backupzip: entry %q escapes the target directory", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, 0o755); err != nil {
				return n, err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return n, err
		}
		src, err := f.Open()
		if err != nil {
			return n, err
		}
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			_ = src.Close()
			return n, err
		}
		_, err = io.Copy(dst, src)
		_ = src.Close()
		if cerr := dst.Close(); err == nil {
			err = cerr
		}
		if err != nil {
			return n, err
		}
		_ = os.Chtimes(target, f.Modified, f.Modified)
		n++
	}
	return n, nil
}
