package file

import (
	"bytes"
	"context"
	"errors"
	"io"
	"maps"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
)

// UploadPresignTTL 是分片预签名直传 URL 的有效期，需覆盖单次上传会话的最长时长（与 biz 的 UploadPeriod 对齐）。
const UploadPresignTTL = 2 * time.Hour

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
		root:   remoteBase(cfg.Prefix),
	}, nil
}

// key 把逻辑路径转为对象 key。越界输入返回必然失败的 key（fail-closed）；
// 正常调用路径上各导出方法已在入口用 guard 拦下。
func (s *ossStore) key(logical string) string {
	k, err := remoteKey(s.root, logical)
	if err != nil {
		return invalidRemotePath
	}
	return k
}

// logical 把对象 key 转回逻辑路径（相对 root）。
// 按路径边界剥前缀：裸 TrimPrefix 会让 root="a" 时的兄弟前缀 "ab/c" 被误算成 "b/c"。
func (s *ossStore) logical(key string) string {
	return remoteLogical(s.root, key)
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
	if err := guard(dir); err != nil {
		return nil, err
	}
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
	if err := guard(p); err != nil {
		return nil, err
	}
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
	if err := guard(p); err != nil {
		return nil, nil, err
	}
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

func (s *ossStore) Write(ctx context.Context, p string, r io.Reader) error {
	if err := guard(p); err != nil {
		return err
	}
	_, err := s.client.PutObject(ctx, s.bucket, s.key(p), r, -1, minio.PutObjectOptions{ContentType: "application/octet-stream"})
	return err
}

func (s *ossStore) Create(ctx context.Context, item *v1.FileCreateItem) error {
	if item == nil {
		return errors.New("缺少创建参数")
	}
	if err := guard(item.Path); err != nil {
		return err
	}
	key := s.key(item.Path)
	if item.IsDir {
		_, err := s.client.PutObject(ctx, s.bucket, key+"/", bytes.NewReader(nil), 0, minio.PutObjectOptions{})
		return err
	}
	_, err := s.client.PutObject(ctx, s.bucket, key, bytes.NewReader(item.Content), int64(len(item.Content)), minio.PutObjectOptions{})
	return err
}

func (s *ossStore) Delete(ctx context.Context, paths []string) []*v1.FileOperationResult {
	if err := guard(paths...); err != nil {
		return errResults(paths, err)
	}
	results := make([]*v1.FileOperationResult, 0, len(paths))
	for _, p := range paths {
		result := &v1.FileOperationResult{Path: p}
		// 空路径会让 key 落在配置的 Prefix 上，随后的递归删除即清空整个来源——必须挡住。
		if rel, _ := remoteRel(p); rel == "" {
			result.Message = "不允许删除来源根目录"
			results = append(results, result)
			continue
		}
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
	srcLogical, dstLogical, err := remoteRename(p, newName)
	if err != nil {
		return "", err
	}
	if err := s.copyTree(ctx, s.key(srcLogical), s.key(dstLogical)); err != nil {
		return "", err
	}
	s.Delete(ctx, []string{srcLogical})
	return dstLogical, nil
}

func (s *ossStore) CopyToDir(ctx context.Context, paths []string, targetDir string) []*v1.FileOperationResult {
	if err := guard(append([]string{targetDir}, paths...)...); err != nil {
		return errResults(paths, err)
	}
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
	if err := guard(p); err != nil {
		return "", err
	}
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

// ---- 分片上传：浏览器直传 OSS（S3 分片上传），momoko 不在数据路径。机制全部私有于本来源。----

func (s *ossStore) core() minio.Core { return minio.Core{Client: s.client} }

// PrepareUpload 初始化（或复用）来源侧 multipart，按需建立续传句柄并逐片生成预签名直传 URL；
// 已传分片以来源 ListObjectParts 为准（合并进 u.Uploaded）。
func (s *ossStore) PrepareUpload(ctx context.Context, u *Upload) error {
	if u.Spec.Parts == 0 { // 空文件无分片：收尾时直接写空对象，无需 multipart
		u.PartURLs = map[uint64]string{}
		return nil
	}
	objectPath := u.TargetPath()
	if err := guard(objectPath); err != nil {
		return err
	}
	// 幂等建立来源侧 multipart uploadID（并发竞态由 SaveRef 收敛，败者中止自己的残留）。
	if u.Ref == "" {
		candidate, err := s.initMultipart(ctx, objectPath)
		if err != nil {
			return err
		}
		ref := candidate
		if u.SaveRef != nil {
			winner, err := u.SaveRef(ctx, candidate)
			if err != nil {
				_ = s.abortMultipart(ctx, objectPath, candidate)
				return err
			}
			if winner != candidate {
				_ = s.abortMultipart(ctx, objectPath, candidate)
				ref = winner
			}
		}
		u.Ref = ref
	}
	partURLs := make(map[uint64]string, u.Spec.Parts)
	for i := uint64(1); i <= u.Spec.Parts; i++ {
		url, err := s.presignPart(ctx, objectPath, u.Ref, i)
		if err != nil {
			return err
		}
		partURLs[i] = url
	}
	u.PartURLs = partURLs
	if uploaded, err := s.listParts(ctx, objectPath, u.Ref); err == nil {
		if u.Uploaded == nil {
			u.Uploaded = map[uint64]string{}
		}
		maps.Copy(u.Uploaded, uploaded)
	}
	return nil
}

// CompleteUpload 合并分片完成上传（空文件直接写空对象）。
func (s *ossStore) CompleteUpload(ctx context.Context, u *Upload) error {
	objectPath := u.TargetPath()
	if err := guard(objectPath); err != nil {
		return err
	}
	if u.Ref == "" { // 无 multipart 句柄（空文件）→ 写空对象
		return s.Write(ctx, objectPath, bytes.NewReader(nil))
	}
	parts, err := s.listParts(ctx, objectPath, u.Ref)
	if err != nil {
		return err
	}
	if len(parts) == 0 {
		return errors.New("没有已上传的分片")
	}
	nums := make([]int, 0, len(parts))
	for n := range parts {
		nums = append(nums, int(n))
	}
	sort.Ints(nums)
	complete := make([]minio.CompletePart, 0, len(nums))
	for _, n := range nums {
		complete = append(complete, minio.CompletePart{PartNumber: n, ETag: parts[uint64(n)]})
	}
	_, err = s.core().CompleteMultipartUpload(ctx, s.bucket, s.key(objectPath), u.Ref, complete, minio.PutObjectOptions{})
	return err
}

// CancelUpload 中止来源侧 multipart（无句柄则无操作）。
func (s *ossStore) CancelUpload(ctx context.Context, u *Upload) error {
	if u.Ref == "" {
		return nil
	}
	objectPath := u.TargetPath()
	if err := guard(objectPath); err != nil {
		return err
	}
	return s.abortMultipart(ctx, objectPath, u.Ref)
}

func (s *ossStore) initMultipart(ctx context.Context, objectPath string) (string, error) {
	return s.core().NewMultipartUpload(ctx, s.bucket, s.key(objectPath), minio.PutObjectOptions{ContentType: "application/octet-stream"})
}

func (s *ossStore) presignPart(ctx context.Context, objectPath, uploadID string, part uint64) (string, error) {
	reqParams := make(url.Values)
	reqParams.Set("uploadId", uploadID)
	reqParams.Set("partNumber", strconv.FormatUint(part, 10))
	u, err := s.client.Presign(ctx, http.MethodPut, s.bucket, s.key(objectPath), UploadPresignTTL, reqParams)
	if err != nil {
		return "", err
	}
	return u.String(), nil
}

func (s *ossStore) listParts(ctx context.Context, objectPath, uploadID string) (map[uint64]string, error) {
	parts := make(map[uint64]string)
	marker := 0
	for {
		res, err := s.core().ListObjectParts(ctx, s.bucket, s.key(objectPath), uploadID, marker, 1000)
		if err != nil {
			return nil, err
		}
		for _, op := range res.ObjectParts {
			parts[uint64(op.PartNumber)] = op.ETag
		}
		if !res.IsTruncated {
			break
		}
		marker = res.NextPartNumberMarker
	}
	return parts, nil
}

func (s *ossStore) abortMultipart(ctx context.Context, objectPath, uploadID string) error {
	return s.core().AbortMultipartUpload(ctx, s.bucket, s.key(objectPath), uploadID)
}
