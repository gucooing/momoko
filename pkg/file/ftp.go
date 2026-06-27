package file

import (
	"bytes"
	"context"
	"crypto/tls"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/jlaffaye/ftp"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

// ftpStore 基于 FTP(可选 FTPS)。每次操作建立短连接，避免连接状态在并发下串话。
type ftpStore struct {
	addr string
	user string
	pass string
	base string // 规整后的根路径
	tls  bool
}

func newFTPStore(cfg Config) (Store, error) {
	if cfg.Host == "" {
		return nil, errors.New("FTP 配置不完整：host 必填")
	}
	port := cfg.Port
	if port == 0 {
		port = 21
	}
	return &ftpStore{
		addr: fmt.Sprintf("%s:%d", cfg.Host, port),
		user: cfg.Username,
		pass: cfg.Password,
		base: strings.Trim(strings.ReplaceAll(cfg.BasePath, "\\", "/"), "/"),
		tls:  cfg.TLS,
	}, nil
}

func (s *ftpStore) conn() (*ftp.ServerConn, error) {
	opts := []ftp.DialOption{ftp.DialWithTimeout(10 * time.Second)}
	if s.tls {
		opts = append(opts, ftp.DialWithExplicitTLS(&tls.Config{InsecureSkipVerify: true})) //nolint:gosec // 自建 FTPS 常用自签证书
	}
	c, err := ftp.Dial(s.addr, opts...)
	if err != nil {
		return nil, err
	}
	if err := c.Login(s.user, s.pass); err != nil {
		_ = c.Quit()
		return nil, err
	}
	return c, nil
}

// p 把逻辑路径转为远端绝对路径。
func (s *ftpStore) p(logical string) string {
	return path.Clean("/" + path.Join(s.base, strings.Trim(strings.ReplaceAll(logical, "\\", "/"), "/")))
}

func (s *ftpStore) List(_ context.Context, dir string, _ v1.FileSortField, _ bool) ([]*v1.FileEntryInfo, error) {
	c, err := s.conn()
	if err != nil {
		return nil, err
	}
	defer c.Quit()
	remote := s.p(dir)
	entries, err := c.List(remote)
	if err != nil {
		return nil, err
	}
	items := make([]*v1.FileEntryInfo, 0, len(entries))
	for _, e := range entries {
		if e.Name == "." || e.Name == ".." {
			continue
		}
		items = append(items, ftpEntry(e, path.Join(strings.Trim(dir, "/"), e.Name)))
	}
	return items, nil
}

func (s *ftpStore) Search(ctx context.Context, dir, keywords string, _ bool) ([]*v1.FileEntryInfo, error) {
	all, err := s.List(ctx, dir, 0, false)
	if err != nil {
		return nil, err
	}
	if keywords == "" {
		return all, nil
	}
	items := make([]*v1.FileEntryInfo, 0)
	for _, e := range all {
		if strings.Contains(e.Name, keywords) {
			items = append(items, e)
		}
	}
	return items, nil
}

func (s *ftpStore) DirInfo(ctx context.Context, dir string) (*v1.FileDirectoryInfo, error) {
	entries, err := s.List(ctx, dir, 0, false)
	if err != nil {
		return nil, err
	}
	var dirCount, fileCount int64
	for _, e := range entries {
		if e.IsDir {
			dirCount++
		} else {
			fileCount++
		}
	}
	name := path.Base(strings.Trim(dir, "/"))
	if name == "" || name == "." {
		name = "根目录"
	}
	return &v1.FileDirectoryInfo{
		Name:      name,
		Path:      strings.Trim(dir, "/"),
		DirCount:  dirCount,
		FileCount: fileCount,
		ItemCount: int64(len(entries)),
	}, nil
}

func (s *ftpStore) Stat(ctx context.Context, p string) (*v1.FileEntryInfo, error) {
	parent := path.Dir(strings.Trim(p, "/"))
	name := path.Base(strings.Trim(p, "/"))
	entries, err := s.List(ctx, parent, 0, false)
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.Name == name {
			return e, nil
		}
	}
	return nil, errors.New("文件不存在")
}

func (s *ftpStore) Open(_ context.Context, p string) (io.ReadCloser, *v1.FileEntryInfo, error) {
	c, err := s.conn()
	if err != nil {
		return nil, nil, err
	}
	resp, err := c.Retr(s.p(p))
	if err != nil {
		_ = c.Quit()
		return nil, nil, err
	}
	size, _ := c.FileSize(s.p(p))
	entry := &v1.FileEntryInfo{Name: path.Base(strings.Trim(p, "/")), Path: strings.Trim(p, "/"), Size: uint64(size)}
	// 连接需在读取期间保持打开，Close 时一并退出连接。
	return &ftpReadCloser{r: resp, c: c}, entry, nil
}

func (s *ftpStore) Read(ctx context.Context, p string, max int64) ([]byte, error) {
	rc, _, err := s.Open(ctx, p)
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	if max <= 0 {
		max = MaxLoadFileSize
	}
	return io.ReadAll(io.LimitReader(rc, max))
}

func (s *ftpStore) Write(_ context.Context, p string, r io.Reader) error {
	c, err := s.conn()
	if err != nil {
		return err
	}
	defer c.Quit()
	return c.Stor(s.p(p), r)
}

func (s *ftpStore) Create(_ context.Context, item *v1.FileCreateItem) error {
	c, err := s.conn()
	if err != nil {
		return err
	}
	defer c.Quit()
	if item.IsDir {
		return c.MakeDir(s.p(item.Path))
	}
	return c.Stor(s.p(item.Path), bytes.NewReader(item.Content))
}

func (s *ftpStore) Delete(_ context.Context, paths []string) []*v1.FileOperationResult {
	c, err := s.conn()
	results := make([]*v1.FileOperationResult, 0, len(paths))
	if err != nil {
		for _, p := range paths {
			results = append(results, &v1.FileOperationResult{Path: p, Message: err.Error()})
		}
		return results
	}
	defer c.Quit()
	for _, p := range paths {
		result := &v1.FileOperationResult{Path: p}
		remote := s.p(p)
		if delErr := c.Delete(remote); delErr != nil {
			if rmErr := c.RemoveDirRecur(remote); rmErr != nil {
				result.Message = rmErr.Error()
				results = append(results, result)
				continue
			}
		}
		result.Success = true
		results = append(results, result)
	}
	return results
}

func (s *ftpStore) Rename(_ context.Context, p, newName string) (string, error) {
	if strings.ContainsAny(newName, `/\`) {
		return "", errors.New("新名称不能包含路径")
	}
	c, err := s.conn()
	if err != nil {
		return "", err
	}
	defer c.Quit()
	dstLogical := path.Join(path.Dir(strings.Trim(p, "/")), newName)
	if err := c.Rename(s.p(p), s.p(dstLogical)); err != nil {
		return "", err
	}
	return dstLogical, nil
}

func (s *ftpStore) MoveToDir(_ context.Context, paths []string, targetDir string) []*v1.FileOperationResult {
	c, err := s.conn()
	results := make([]*v1.FileOperationResult, 0, len(paths))
	if err != nil {
		for _, p := range paths {
			results = append(results, &v1.FileOperationResult{Path: p, Message: err.Error()})
		}
		return results
	}
	defer c.Quit()
	for _, p := range paths {
		result := &v1.FileOperationResult{Path: p}
		dst := path.Join(strings.Trim(targetDir, "/"), path.Base(strings.Trim(p, "/")))
		if err := c.Rename(s.p(p), s.p(dst)); err != nil {
			result.Message = err.Error()
		} else {
			result.Success = true
			result.Message = dst
		}
		results = append(results, result)
	}
	return results
}

func (s *ftpStore) Caps() Caps {
	return Caps{Presign: false, Copy: false, Move: true, Compress: false, ResumableUpload: true}
}

func ftpEntry(e *ftp.Entry, logical string) *v1.FileEntryInfo {
	return &v1.FileEntryInfo{
		Name:       e.Name,
		Path:       strings.Trim(logical, "/"),
		IsDir:      e.Type == ftp.EntryTypeFolder,
		Size:       e.Size,
		UpdateTime: timestamppb.New(e.Time),
	}
}

type ftpReadCloser struct {
	r io.ReadCloser
	c *ftp.ServerConn
}

func (f *ftpReadCloser) Read(p []byte) (int, error) { return f.r.Read(p) }

func (f *ftpReadCloser) Close() error {
	err := f.r.Close()
	_ = f.c.Quit()
	return err
}
