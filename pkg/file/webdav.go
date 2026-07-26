package file

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path"
	"strings"

	"github.com/studio-b12/gowebdav"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

// webdavStore 基于 WebDAV（gowebdav 客户端），覆盖 Nextcloud / 坚果云 等。
type webdavStore struct {
	client *gowebdav.Client
	base   string // 规整后的根路径（URL 之下的子目录）
}

func newWebDAVStore(cfg Config) (Store, error) {
	if cfg.URL == "" {
		return nil, errors.New("WebDAV 配置不完整：url 必填")
	}
	c := gowebdav.NewClient(cfg.URL, cfg.Username, cfg.Password)
	return &webdavStore{
		client: c,
		base:   remoteBase(cfg.BasePath),
	}, nil
}

// p 把逻辑路径转为远端绝对路径。越界输入返回必然失败的路径（fail-closed）；
// 正常调用路径上各导出方法已在入口用 guard 拦下。
//
// 旧实现是 "/" + path.Join(base, logical)：path.Join 会把 ".." 折叠掉，
// 于是 "../x" 先吃掉 base，多余的 ".." 还会原样留在结果里交给服务端解释，
// 等于把整个 URL 前缀之上的空间都暴露出去。
func (s *webdavStore) p(logical string) string {
	abs, err := remoteAbs(s.base, logical)
	if err != nil {
		return invalidRemotePath
	}
	return abs
}

func (s *webdavStore) List(_ context.Context, dir string, _ v1.FileSortField, _ bool) ([]*v1.FileEntryInfo, error) {
	if err := guard(dir); err != nil {
		return nil, err
	}
	infos, err := s.client.ReadDir(s.p(dir))
	if err != nil {
		return nil, err
	}
	items := make([]*v1.FileEntryInfo, 0, len(infos))
	for _, info := range infos {
		items = append(items, webdavEntry(info, path.Join(strings.Trim(dir, "/"), info.Name())))
	}
	return items, nil
}

func (s *webdavStore) Search(ctx context.Context, dir, keywords string, _ bool) ([]*v1.FileEntryInfo, error) {
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

func (s *webdavStore) DirInfo(ctx context.Context, dir string) (*v1.FileDirectoryInfo, error) {
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

func (s *webdavStore) Stat(_ context.Context, p string) (*v1.FileEntryInfo, error) {
	if err := guard(p); err != nil {
		return nil, err
	}
	info, err := s.client.Stat(s.p(p))
	if err != nil {
		return nil, errors.New("文件不存在")
	}
	return webdavEntry(info, strings.Trim(p, "/")), nil
}

func (s *webdavStore) Open(_ context.Context, p string) (io.ReadCloser, *v1.FileEntryInfo, error) {
	if err := guard(p); err != nil {
		return nil, nil, err
	}
	info, err := s.client.Stat(s.p(p))
	if err != nil {
		return nil, nil, errors.New("文件不存在")
	}
	rc, err := s.client.ReadStream(s.p(p))
	if err != nil {
		return nil, nil, err
	}
	return rc, webdavEntry(info, strings.Trim(p, "/")), nil
}

func (s *webdavStore) Write(_ context.Context, p string, r io.Reader) error {
	if err := guard(p); err != nil {
		return err
	}
	_ = s.client.MkdirAll(path.Dir(s.p(p)), 0o755)
	return s.client.WriteStream(s.p(p), r, 0o644)
}

func (s *webdavStore) Create(_ context.Context, item *v1.FileCreateItem) error {
	if item == nil {
		return errors.New("缺少创建参数")
	}
	if err := guard(item.Path); err != nil {
		return err
	}
	if item.IsDir {
		return s.client.MkdirAll(s.p(item.Path), 0o755)
	}
	_ = s.client.MkdirAll(path.Dir(s.p(item.Path)), 0o755)
	return s.client.WriteStream(s.p(item.Path), bytes.NewReader(item.Content), 0o644)
}

func (s *webdavStore) Delete(_ context.Context, paths []string) []*v1.FileOperationResult {
	if err := guard(paths...); err != nil {
		return errResults(paths, err)
	}
	results := make([]*v1.FileOperationResult, 0, len(paths))
	for _, p := range paths {
		result := &v1.FileOperationResult{Path: p}
		if err := s.client.RemoveAll(s.p(p)); err != nil {
			result.Message = err.Error()
		} else {
			result.Success = true
		}
		results = append(results, result)
	}
	return results
}

func (s *webdavStore) Rename(_ context.Context, p, newName string) (string, error) {
	srcLogical, dstLogical, err := remoteRename(p, newName)
	if err != nil {
		return "", err
	}
	if err := s.client.Rename(s.p(srcLogical), s.p(dstLogical), false); err != nil {
		return "", err
	}
	return dstLogical, nil
}

func (s *webdavStore) CopyToDir(_ context.Context, paths []string, targetDir string) []*v1.FileOperationResult {
	if err := guard(append([]string{targetDir}, paths...)...); err != nil {
		return errResults(paths, err)
	}
	results := make([]*v1.FileOperationResult, 0, len(paths))
	for _, p := range paths {
		result := &v1.FileOperationResult{Path: p}
		dst := path.Join(strings.Trim(targetDir, "/"), path.Base(strings.Trim(p, "/")))
		if err := s.client.Copy(s.p(p), s.p(dst), true); err != nil {
			result.Message = err.Error()
		} else {
			result.Success = true
			result.Message = dst
		}
		results = append(results, result)
	}
	return results
}

func (s *webdavStore) MoveToDir(_ context.Context, paths []string, targetDir string) []*v1.FileOperationResult {
	if err := guard(append([]string{targetDir}, paths...)...); err != nil {
		return errResults(paths, err)
	}
	results := make([]*v1.FileOperationResult, 0, len(paths))
	for _, p := range paths {
		result := &v1.FileOperationResult{Path: p}
		dst := path.Join(strings.Trim(targetDir, "/"), path.Base(strings.Trim(p, "/")))
		if err := s.client.Rename(s.p(p), s.p(dst), true); err != nil {
			result.Message = err.Error()
		} else {
			result.Success = true
			result.Message = dst
		}
		results = append(results, result)
	}
	return results
}

func (s *webdavStore) Caps() Caps {
	return Caps{Presign: false, Copy: true, Move: true, Compress: false, ResumableUpload: true}
}

// ---- 分片上传：分片经 momoko 签名端点落本地缓冲，收尾时整流推送到 WebDAV（缓冲型远端）----

func (s *webdavStore) PrepareUpload(_ context.Context, u *Upload) error {
	fillSignedParts(u)
	return nil
}

func (s *webdavStore) CompleteUpload(ctx context.Context, u *Upload) error {
	return completeBufferedRemote(ctx, s, u)
}

func (s *webdavStore) CancelUpload(_ context.Context, u *Upload) error {
	return cancelBufferedRemote(u)
}

// AsyncFinalize 收尾需把本地缓冲整流推送到 WebDAV，可能较慢，应作为后台任务执行。
func (s *webdavStore) AsyncFinalize() bool { return true }

func webdavEntry(info os.FileInfo, logical string) *v1.FileEntryInfo {
	return &v1.FileEntryInfo{
		Name:       info.Name(),
		Path:       strings.Trim(logical, "/"),
		IsDir:      info.IsDir(),
		Size:       uint64(info.Size()),
		UpdateTime: timestamppb.New(info.ModTime()),
	}
}
