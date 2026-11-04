package file

import (
	"bytes"
	"context"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"

	"github.com/studio-b12/gowebdav"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

// webdavStore 基于 WebDAV（gowebdav 客户端），覆盖 Nextcloud / 坚果云 等。
type webdavStore struct {
	client *gowebdav.Client
	base   string // 规整后的根路径（URL 之下的子目录）

	// 配额查询（RFC 4331 PROPFIND）需要直接发请求：保留地址与凭证。
	url      string
	username string
	password string
}

func newWebDAVStore(cfg Config) (Store, error) {
	if cfg.URL == "" {
		return nil, errors.New("WebDAV 配置不完整：url 必填")
	}
	c := gowebdav.NewClient(cfg.URL, cfg.Username, cfg.Password)
	return &webdavStore{
		client:   c,
		base:     strings.Trim(strings.ReplaceAll(cfg.BasePath, "\\", "/"), "/"),
		url:      strings.TrimRight(cfg.URL, "/"),
		username: cfg.Username,
		password: cfg.Password,
	}, nil
}

func (s *webdavStore) p(logical string) string {
	return "/" + path.Join(s.base, strings.Trim(strings.ReplaceAll(logical, "\\", "/"), "/"))
}

func (s *webdavStore) List(_ context.Context, dir string, _ v1.FileSortField, _ bool) ([]*v1.FileEntryInfo, error) {
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
	var totalSize uint64
	for _, e := range entries {
		if e.IsDir {
			dirCount++
		} else {
			fileCount++
			totalSize += e.Size
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
		TotalSize: totalSize,
	}, nil
}

func (s *webdavStore) Stat(_ context.Context, p string) (*v1.FileEntryInfo, error) {
	info, err := s.client.Stat(s.p(p))
	if err != nil {
		return nil, errors.New("文件不存在")
	}
	return webdavEntry(info, strings.Trim(p, "/")), nil
}

func (s *webdavStore) Open(_ context.Context, p string) (io.ReadCloser, *v1.FileEntryInfo, error) {
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

func (s *webdavStore) Read(ctx context.Context, p string, max int64) ([]byte, error) {
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

func (s *webdavStore) Write(_ context.Context, p string, r io.Reader) error {
	_ = s.client.MkdirAll(path.Dir(s.p(p)), 0o755)
	return s.client.WriteStream(s.p(p), r, 0o644)
}

func (s *webdavStore) Create(_ context.Context, item *v1.FileCreateItem) error {
	if item.IsDir {
		return s.client.MkdirAll(s.p(item.Path), 0o755)
	}
	_ = s.client.MkdirAll(path.Dir(s.p(item.Path)), 0o755)
	return s.client.WriteStream(s.p(item.Path), bytes.NewReader(item.Content), 0o644)
}

func (s *webdavStore) Delete(_ context.Context, paths []string) []*v1.FileOperationResult {
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
	if strings.ContainsAny(newName, `/\`) {
		return "", errors.New("新名称不能包含路径")
	}
	dstLogical := path.Join(path.Dir(strings.Trim(p, "/")), newName)
	if err := s.client.Rename(s.p(p), s.p(dstLogical), false); err != nil {
		return "", err
	}
	return dstLogical, nil
}

func (s *webdavStore) CopyToDir(_ context.Context, paths []string, targetDir string) []*v1.FileOperationResult {
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

// davQuota 是 PROPFIND 响应里配额属性的最小解析结构（按本地名匹配，忽略命名空间前缀）。
type davQuota struct {
	Responses []struct {
		Used  string `xml:"propstat>prop>quota-used-bytes"`
		Avail string `xml:"propstat>prop>quota-available-bytes"`
	} `xml:"response"`
}

const davQuotaBody = `<?xml version="1.0" encoding="utf-8"?><d:propfind xmlns:d="DAV:"><d:prop><d:quota-available-bytes/><d:quota-used-bytes/></d:prop></d:propfind>`

// Usage 用 RFC 4331 配额属性查询根目录的已用 / 可用空间（坚果云、Nextcloud 等支持）；服务端不支持时返回错误。
func (s *webdavStore) Usage(ctx context.Context) (used, total uint64, err error) {
	target := s.url + (&url.URL{Path: s.p("")}).EscapedPath()
	req, err := http.NewRequestWithContext(ctx, "PROPFIND", target, strings.NewReader(davQuotaBody))
	if err != nil {
		return 0, 0, err
	}
	req.Header.Set("Depth", "0")
	req.Header.Set("Content-Type", "application/xml; charset=utf-8")
	if s.username != "" || s.password != "" {
		req.SetBasicAuth(s.username, s.password)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMultiStatus && resp.StatusCode != http.StatusOK {
		return 0, 0, fmt.Errorf("PROPFIND 返回 %d", resp.StatusCode)
	}
	var q davQuota
	if err := xml.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&q); err != nil {
		return 0, 0, err
	}
	for _, r := range q.Responses {
		u, uerr := strconv.ParseInt(strings.TrimSpace(r.Used), 10, 64)
		if uerr != nil || u < 0 {
			continue
		}
		used = uint64(u)
		// 可用空间为负（RFC 4331：-1/-2/-3）表示不限或未知。
		if a, aerr := strconv.ParseInt(strings.TrimSpace(r.Avail), 10, 64); aerr == nil && a >= 0 {
			total = used + uint64(a)
		}
		return used, total, nil
	}
	return 0, 0, errors.New("来源未提供配额信息")
}
