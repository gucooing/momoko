// Package objectstore 是 S3 兼容对象存储的最小封装，用于头像这类以公开链接对外的资源：
// 签发浏览器直传用的预签名 PUT 地址（文件不经过服务端）、查对象大小、按 key 拼公开链接、按 key 删除对象。
package objectstore

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/minio/minio-go/v7/pkg/s3utils"
)

// Config 是一个 S3 兼容存储的连接与公开访问配置。
type Config struct {
	Endpoint  string // 服务地址，可带 http(s):// 前缀（会被去掉，协议由 UseSSL 决定）
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool
	PathStyle bool
	Prefix    string // 对象 key 前缀，可空
	PublicURL string // 公开访问地址（自定义域名 / CDN）；空则按服务地址与存储桶拼接
}

// Store 是按 Config 构造的对象存储客户端（构造不发起网络请求）。
type Store struct {
	client     *minio.Client
	bucket     string
	prefix     string
	publicBase string // 公开链接前缀（无尾斜杠），对象链接 = publicBase + "/" + 转义后的 key
}

// New 校验配置并构造客户端。
func New(cfg Config) (*Store, error) {
	endpoint := strings.TrimSuffix(strings.TrimPrefix(strings.TrimPrefix(strings.TrimSpace(cfg.Endpoint), "https://"), "http://"), "/")
	bucket := strings.TrimSpace(cfg.Bucket)
	if endpoint == "" || bucket == "" {
		return nil, errors.New("服务地址和存储桶不能为空")
	}
	opts := &minio.Options{
		Creds:  credentials.NewStaticV4(strings.TrimSpace(cfg.AccessKey), cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
		Region: strings.TrimSpace(cfg.Region),
	}
	if cfg.PathStyle {
		opts.BucketLookup = minio.BucketLookupPath
	}
	client, err := minio.New(endpoint, opts)
	if err != nil {
		return nil, fmt.Errorf("服务地址无效: %w", err)
	}

	publicBase := strings.TrimRight(strings.TrimSpace(cfg.PublicURL), "/")
	if publicBase != "" {
		u, err := url.Parse(publicBase)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return nil, errors.New("公开访问地址必须是 http(s) 链接")
		}
	} else {
		// 与上传时 minio 选择的寻址方式保持一致：虚拟主机风格（bucket.host）或路径风格（host/bucket）。
		ep := *client.EndpointURL()
		if !cfg.PathStyle && s3utils.IsVirtualHostSupported(ep, bucket) {
			publicBase = ep.Scheme + "://" + bucket + "." + ep.Host
		} else {
			publicBase = ep.Scheme + "://" + ep.Host + "/" + url.PathEscape(bucket)
		}
	}

	return &Store{
		client:     client,
		bucket:     bucket,
		prefix:     strings.Trim(strings.ReplaceAll(cfg.Prefix, "\\", "/"), "/ "),
		publicBase: publicBase,
	}, nil
}

// PresignPut 为一个对象（name 为前缀下的相对 key）签发预签名 PUT 地址，返回直传地址与完整对象 key。
// Content-Type 与 Content-Length 一并签进签名：浏览器上传的类型或大小与申请时不符，对象存储会直接拒绝。
func (s *Store) PresignPut(ctx context.Context, name, contentType string, size int64, ttl time.Duration) (uploadURL, key string, err error) {
	key = s.key(name)
	headers := http.Header{}
	headers.Set("Content-Type", contentType)
	headers.Set("Content-Length", strconv.FormatInt(size, 10))
	u, err := s.client.PresignHeader(ctx, http.MethodPut, s.bucket, key, ttl, nil, headers)
	if err != nil {
		return "", "", err
	}
	return u.String(), key, nil
}

// Size 返回对象大小；对象不存在时返回错误。
func (s *Store) Size(ctx context.Context, key string) (int64, error) {
	info, err := s.client.StatObject(ctx, s.bucket, key, minio.StatObjectOptions{})
	if err != nil {
		return 0, err
	}
	return info.Size, nil
}

// URL 返回对象的公开链接。
func (s *Store) URL(key string) string {
	return s.publicBase + "/" + escapeKey(key)
}

// Delete 按对象 key 删除对象。
func (s *Store) Delete(ctx context.Context, key string) error {
	return s.client.RemoveObject(ctx, s.bucket, key, minio.RemoveObjectOptions{})
}

func (s *Store) key(name string) string {
	name = strings.Trim(name, "/")
	if s.prefix == "" {
		return name
	}
	return s.prefix + "/" + name
}

// escapeKey 按段转义对象 key，保留分隔用的斜杠。
func escapeKey(key string) string {
	parts := strings.Split(key, "/")
	for i, p := range parts {
		parts[i] = url.PathEscape(p)
	}
	return strings.Join(parts, "/")
}
