// Package archive 安全地解开 .zip 和 .tar.gz：所有写入都落在目标目录里（os.Root），
// 带 ..、绝对路径、指到目录外的链接一律不收，解出的总量有上限（防压缩炸弹）。
package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"strings"
)

// Extract 把压缩包 archive 解到 dest 里，按文件名 name 认格式；解出的总量超过 limit 字节就放弃。
func Extract(ctx context.Context, archive, name, dest string, limit int64) error {
	// os.Root 保证所有写入都落在 dest 里面：包里的路径不管怎么写、中间有没有链接，都出不去。
	root, err := os.OpenRoot(dest)
	if err != nil {
		return err
	}
	defer root.Close()
	x := &extractor{ctx: ctx, root: root, left: limit}
	switch {
	case strings.HasSuffix(name, ".zip"):
		return x.zip(archive)
	case strings.HasSuffix(name, ".tar.gz"):
		return x.tarGz(archive)
	}
	return errors.New("不认识的压缩包格式")
}

type extractor struct {
	ctx  context.Context
	root *os.Root
	left int64 // 还能解出多少字节
}

var errBadArchive = errors.New("压缩包损坏了")

func (x *extractor) zip(archive string) error {
	r, err := zip.OpenReader(archive)
	if err != nil {
		return errBadArchive
	}
	defer r.Close()
	for _, f := range r.File {
		mode := f.Mode()
		switch {
		case mode.IsDir():
			err = x.dir(f.Name)
		case mode&fs.ModeSymlink != 0:
			err = x.open(f, func(rc io.Reader) error {
				target, err := io.ReadAll(io.LimitReader(rc, 4<<10))
				if err != nil {
					return errBadArchive
				}
				return x.link(f.Name, string(target))
			})
		case mode.IsRegular():
			err = x.open(f, func(rc io.Reader) error { return x.file(f.Name, mode, rc) })
		default:
			err = errUnsupportedEntry
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func (x *extractor) open(f *zip.File, use func(io.Reader) error) error {
	rc, err := f.Open()
	if err != nil {
		return errBadArchive
	}
	defer rc.Close()
	return use(rc)
}

func (x *extractor) tarGz(archive string) error {
	file, err := os.Open(archive)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return errBadArchive
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errBadArchive
		}
		switch hdr.Typeflag {
		case tar.TypeDir:
			err = x.dir(hdr.Name)
		case tar.TypeReg:
			err = x.file(hdr.Name, hdr.FileInfo().Mode(), tr)
		case tar.TypeSymlink:
			err = x.link(hdr.Name, hdr.Linkname)
		case tar.TypeXGlobalHeader:
			// 整个包的附加信息，没有对应的文件。
		default:
			err = errUnsupportedEntry
		}
		if err != nil {
			return err
		}
	}
}

var errUnsupportedEntry = errors.New("压缩包里有不认识的内容")

// local 检查包里的一个路径：带 ..、绝对路径、盘符的一律不收。返回相对目标目录的路径；包的根目录本身返回空。
func local(name string) (string, error) {
	clean := path.Clean(strings.ReplaceAll(name, `\`, "/"))
	if clean == "." {
		return "", nil
	}
	rel := filepath.FromSlash(clean)
	if !filepath.IsLocal(rel) {
		return "", fmt.Errorf("压缩包里有跑到目录外面的路径：%s", name)
	}
	return rel, nil
}

func (x *extractor) dir(name string) error {
	rel, err := local(name)
	if err != nil || rel == "" {
		return err
	}
	return x.root.MkdirAll(rel, 0o755)
}

func (x *extractor) file(name string, mode fs.FileMode, r io.Reader) error {
	rel, err := local(name)
	if err != nil {
		return err
	}
	if rel == "" {
		return errUnsupportedEntry
	}
	if err := x.root.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		return err
	}
	// 只保留「能不能执行」这一点，别的权限位不照搬。
	perm := fs.FileMode(0o644)
	if mode&0o111 != 0 {
		perm = 0o755
	}
	out, err := x.root.OpenFile(rel, os.O_WRONLY|os.O_CREATE|os.O_EXCL, perm)
	if errors.Is(err, fs.ErrExist) {
		return errors.New("压缩包里有重名的文件")
	}
	if err != nil {
		return err
	}
	if err := x.copy(out, r); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

// link 建一个符号链接。指到目标目录外面的不收。
func (x *extractor) link(name, target string) error {
	rel, err := local(name)
	if err != nil {
		return err
	}
	if rel == "" || runtime.GOOS == "windows" {
		return errUnsupportedEntry
	}
	target = filepath.FromSlash(target)
	if filepath.IsAbs(target) || !filepath.IsLocal(filepath.Join(filepath.Dir(rel), target)) {
		return fmt.Errorf("压缩包里有指到目录外面的链接：%s", name)
	}
	if err := x.root.MkdirAll(filepath.Dir(rel), 0o755); err != nil {
		return err
	}
	return x.root.Symlink(target, rel)
}

func (x *extractor) copy(dst io.Writer, src io.Reader) error {
	buf := make([]byte, 1<<20)
	for {
		if err := x.ctx.Err(); err != nil {
			return err
		}
		n, err := src.Read(buf)
		if n > 0 {
			if x.left -= int64(n); x.left < 0 {
				return errors.New("压缩包解出来的东西太大了")
			}
			if _, werr := dst.Write(buf[:n]); werr != nil {
				return werr
			}
		}
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return errBadArchive
		}
	}
}
