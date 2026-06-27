package file

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/url"
	"path"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

// ossStore 基于 S3 兼容协议（minio-go），覆盖 AWS S3 / MinIO / 阿里云OSS(S3端点) / 腾讯COS / R2 等。
// 逻辑路径以 root(前缀) 为根；目录用「key 以 / 结尾」表示。
type ossStore struct {
	client *minio.Client
	bucket string
	root   string // 规整后的前缀（无首尾斜杠），"" 表示桶根
}

func newOSSStore(cfg Config) (Store, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" {
		return nil, errors.New("OSS 配置不完整：endpoint/bucket 必填")
	}
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: cfg.Region,
	}
	if cfg.PathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}
	client, err := minio.New(strings.TrimPrefix(strings.TrimPrefix(cfg.Endpoint, "https://"), "http://"), opts)
	if err != nil {
		return nil, err
	}
	return &ossStore{
		client: client,
		bucket: cfg.Bucket,
		root:   strings.Trim(strings.ReplaceAll(cfg.Prefix, "\\", "/"), "/"),
	}, nil
}

// key 把逻辑路径转为对象 key。
func (s *ossStore) key(logical string) string {
	p := strings.Trim(strings.ReplaceAll(logical, "\\", "/"), "/")
	if s.root == "" {
		return p
	}
	if p == "" {
		return s.root
	}
	return s.root + "/" + p
}

// logical 把对象 key 转回逻辑路径（相对 root）。
func (s *ossStore) logical(key string) string {
	rel := strings.TrimPrefix(key, s.root)
	return strings.Trim(rel, "/")
}

// listPrefix 返回列举某目录所需的前缀（带尾斜杠，桶/根目录则为空或 root/）。
func (s *ossStore) listPrefix(dir string) string {
	base := s.key(dir)
	if base == "" {
		return ""
	}
	return base + "/"
}

func (s *ossStore) List(ctx context.Context, dir string, _ v1.FileSortField, _ bool) ([]*v1.FileEntryInfo, error) {
	prefix := s.listPrefix(dir)
	items := make([]*v1.FileEntryInfo, 0)
	for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: false}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		name := strings.TrimSuffix(strings.TrimPrefix(obj.Key, prefix), "/")
		if name == "" || strings.Contains(name, "/") {
			continue
		}
		isDir := strings.HasSuffix(obj.Key, "/")
		items = append(items, &v1.FileEntryInfo{
			Name:       name,
			Path:       s.logical(obj.Key),
			IsDir:      isDir,
			Permission: "",
			Size:       uint64(obj.Size),
			UpdateTime: timestamppb.New(obj.LastModified),
		})
	}
	return items, nil
}

func (s *ossStore) Search(ctx context.Context, dir, keywords string, recursive bool) ([]*v1.FileEntryInfo, error) {
	prefix := s.listPrefix(dir)
	items := make([]*v1.FileEntryInfo, 0)
	for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: prefix, Recursive: recursive}) {
		if obj.Err != nil {
			return nil, obj.Err
		}
		name := path.Base(strings.TrimSuffix(obj.Key, "/"))
		if name == "" || (keywords != "" && !strings.Contains(name, keywords)) {
			continue
		}
		items = append(items, &v1.FileEntryInfo{
			Name:       name,
			Path:       s.logical(obj.Key),
			IsDir:      strings.HasSuffix(obj.Key, "/"),
			Size:       uint64(obj.Size),
			UpdateTime: timestamppb.New(obj.LastModified),
		})
	}
	return items, nil
}

func (s *ossStore) DirInfo(ctx context.Context, dir string) (*v1.FileDirectoryInfo, error) {
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

func (s *ossStore) Stat(ctx context.Context, p string) (*v1.FileEntryInfo, error) {
	key := s.key(p)
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err == nil {
		return &v1.FileEntryInfo{
			Name:       path.Base(key),
			Path:       s.logical(key),
			IsDir:      false,
			Size:       uint64(info.Size),
			UpdateTime: timestamppb.New(info.LastModified),
		}, nil
	}
	// 不是对象则尝试当作目录（前缀下有内容即视为目录存在）。
	for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: key + "/", MaxKeys: 1, Recursive: true}) {
		if obj.Err == nil {
			return &v1.FileEntryInfo{Name: path.Base(key), Path: s.logical(key), IsDir: true}, nil
		}
	}
	return nil, errors.New("文件不存在")
}

func (s *ossStore) Open(ctx context.Context, p string) (io.ReadCloser, *v1.FileEntryInfo, error) {
	key := s.key(p)
	obj, err := s.client.GetObject(ctx, s.bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, nil, err
	}
	info, err := obj.Stat()
	if err != nil {
		_ = obj.Close()
		return nil, nil, errors.New("文件不存在")
	}
	entry := &v1.FileEntryInfo{
		Name:       path.Base(key),
		Path:       s.logical(key),
		Size:       uint64(info.Size),
		UpdateTime: timestamppb.New(info.LastModified),
	}
	return obj, entry, nil // *minio.Object 同时实现 io.Seeker，调用方可支持 Range
}

func (s *ossStore) Read(ctx context.Context, p string, max int64) ([]byte, error) {
	if max <= 0 {
		max = MaxLoadFileSize
	}
	obj, err := s.client.GetObject(ctx, s.bucket, s.key(p), minio.GetObjectOptions{})
	if err != nil {
		return nil, err
	}
	defer obj.Close()
	return io.ReadAll(io.LimitReader(obj, max))
}

func (s *ossStore) Write(ctx context.Context, p string, r io.Reader) error {
	_, err := s.client.PutObject(ctx, s.bucket, s.key(p), r, -1, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	return err
}

func (s *ossStore) Create(ctx context.Context, item *v1.FileCreateItem) error {
	key := s.key(item.Path)
	if item.IsDir {
		_, err := s.client.PutObject(ctx, s.bucket, key+"/", bytes.NewReader(nil), 0, minio.PutObjectOptions{})
		return err
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(item.Content), int64(len(item.Content)), minio.PutObjectOptions{})
	return err
}

func (s *ossStore) Delete(ctx context.Context, paths []string) []*v1.FileOperationResult {
	results := make([]*v1.FileOperationResult, 0, len(paths))
	for _, p := range paths {
		result := &v1.FileOperationResult{Path: p}
		key := s.key(p)
		var failed error
		// 删除目录前缀下的所有对象。
		for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: key + "/", Recursive: true}) {
			if obj.Err != nil {
				failed = obj.Err
				break
			}
			if err := s.client.RemoveObject(ctx, s.bucket, obj.Key, minio.RemoveObjectOptions{}); err != nil {
				failed = err
				break
			}
		}
		// 删除对象本身（文件或目录占位）。
		_ = s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
		_ = s.client.RemoveObject(ctx, s.bucket, key+"/", minio.RemoveObjectOptions{})
		if failed != nil {
			result.Message = failed.Error()
		} else {
			result.Success = true
		}
		results = append(results, result)
	}
	return results
}

func (s *ossStore) Rename(ctx context.Context, p, newName string) (string, error) {
	if strings.ContainsAny(newName, `/\`) {
		return "", errors.New("新名称不能包含路径")
	}
	srcKey := s.key(p)
	dstLogical := path.Join(path.Dir(strings.Trim(p, "/")), newName)
	if err := s.copyTree(ctx, srcKey, s.key(dstLogical)); err != nil {
		return "", err
	}
	s.Delete(ctx, []string{p})
	return dstLogical, nil
}

func (s *ossStore) CopyToDir(ctx context.Context, paths []string, targetDir string) []*v1.FileOperationResult {
	results := make([]*v1.FileOperationResult, 0, len(paths))
	for _, p := range paths {
		result := &v1.FileOperationResult{Path: p}
		dstKey := s.key(path.Join(strings.Trim(targetDir, "/"), path.Base(strings.Trim(p, "/"))))
		if err := s.copyTree(ctx, s.key(p), dstKey); err != nil {
			result.Message = err.Error()
		} else {
			result.Success = true
			result.Message = s.logical(dstKey)
		}
		results = append(results, result)
	}
	return results
}

func (s *ossStore) MoveToDir(ctx context.Context, paths []string, targetDir string) []*v1.FileOperationResult {
	results := s.CopyToDir(ctx, paths, targetDir)
	for i, r := range results {
		if r.Success {
			s.Delete(ctx, []string{paths[i]})
		}
	}
	return results
}

// copyTree 复制单个对象，或递归复制一个「目录前缀」下的所有对象。
func (s *ossStore) copyTree(ctx context.Context, srcKey, dstKey string) error {
	copied := false
	if _, err := s.client.StatObject(ctx, s.bucket, srcKey, minio.StatObjectOptions{}); err == nil {
		if _, err := s.client.CopyObject(ctx,
			minio.CopyDestOptions{Bucket: s.bucket, Object: dstKey},
			minio.CopySrcOptions{Bucket: s.bucket, Object: srcKey}); err != nil {
			return err
		}
		copied = true
	}
	for obj := range s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{Prefix: srcKey + "/", Recursive: true}) {
		if obj.Err != nil {
			return obj.Err
		}
		rel := strings.TrimPrefix(obj.Key, srcKey)
		if _, err := s.client.CopyObject(ctx,
			minio.CopyDestOptions{Bucket: s.bucket, Object: dstKey + rel},
			minio.CopySrcOptions{Bucket: s.bucket, Object: obj.Key}); err != nil {
			return err
		}
		copied = true
	}
	if !copied {
		return errors.New("源不存在")
	}
	return nil
}

func (s *ossStore) Presign(ctx context.Context, p string, inline bool, ttl time.Duration) (string, error) {
	reqParams := make(url.Values)
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	reqParams.Set("response-content-disposition", disposition+`; filename="`+path.Base(s.key(p))+`"`)
	u, err := s.client.PresignedGetObject(ctx, s.bucket, s.key(p), ttl, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *ossStore) Caps() Caps {
	return Caps{Presign: true, Copy: true, Move: true, Compress: false, ResumableUpload: true}
}
