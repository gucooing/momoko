// Package file 是文件管理的「多来源」适配层：把本地磁盘、对象存储(S3)、FTP、WebDAV
// 统一到一个 Store 接口之后，供业务层按来源 id 取用。
//
// 分层约定：
//   - 本地磁盘的一切实际操作与安全把关都在 pkg/localfs，本包的 LocalStore 只做协议转换；
//   - 本包只负责「来源抽象 + 与 api/v1 类型互转 + 异步任务壳」，不含任何路径拼接逻辑；
//   - 本包不依赖 internal/*，业务层（biz）负责鉴权、持久化与来源配置的解密。
package file

import (
	"context"
	"errors"
	"io"
	"time"

	v1 "momoko/api/gen/v1"
)

// 排序字段（对 api 枚举的别名，免得各处都得 import api 包）。
const (
	SortByName    = v1.FileSortField_FILE_SORT_FIELD_NAME
	SortByModTime = v1.FileSortField_FILE_SORT_FIELD_UPDATE_TIME
)

// Store 是可插拔的文件来源抽象。
//
// path 是来源内的逻辑路径：本地来源为受视图约束的路径（越界一律失败），
// 对象存储为 key 前缀下的相对路径，FTP/WebDAV 为远端路径。
// 各实现必须自行把关边界——本地由 os.Root 在内核层保证，远端由 remotePath 归一化后校验。
type Store interface {
	List(ctx context.Context, dir string, field v1.FileSortField, desc bool) ([]*v1.FileEntryInfo, error)
	Search(ctx context.Context, dir, keywords string, recursive bool) ([]*v1.FileEntryInfo, error)
	DirInfo(ctx context.Context, dir string) (*v1.FileDirectoryInfo, error)
	Stat(ctx context.Context, path string) (*v1.FileEntryInfo, error)
	// Open 打开文件用于串流（预览/下载）。返回的 ReadCloser 若同时实现 io.Seeker，调用方可支持 Range。
	Open(ctx context.Context, path string) (io.ReadCloser, *v1.FileEntryInfo, error)
	// Write 以流写入/覆盖一个文件（保存、上传落地用）。
	Write(ctx context.Context, path string, r io.Reader) error
	Create(ctx context.Context, item *v1.FileCreateItem) error
	Delete(ctx context.Context, paths []string) []*v1.FileOperationResult
	Rename(ctx context.Context, path, newName string) (string, error)
	// PrepareUpload 准备一次分片上传：填充各分片的 PUT 直传地址（u.PartURLs）与已传分片（u.Uploaded）；
	// 来源按需在内部建立并经 u.SaveRef 持久化续传句柄。各来源自行决定机制（直传/缓冲），调用方无需感知。
	PrepareUpload(ctx context.Context, u *Upload) error
	// CompleteUpload 收尾一次分片上传（合并/落地）。
	CompleteUpload(ctx context.Context, u *Upload) error
	// CancelUpload 中止并清理一次未完成的分片上传（删缓冲/中止远端句柄）。
	CancelUpload(ctx context.Context, u *Upload) error
	// Caps 返回该来源支持的能力，供前端隐藏/禁用不支持的动作。
	Caps() Caps
}

// Caps 描述一个来源支持的可选能力。
type Caps struct {
	Presign         bool // 支持预签名直链（启用 302 跳转的前提）
	Copy            bool // 支持复制
	Move            bool // 支持移动/剪切
	Compress        bool // 支持压缩/解压
	ResumableUpload bool // 支持断点续传上传
}

// 来源类型标识。
const (
	TypeLocal  = "local"
	TypeOSS    = "oss"
	TypeFTP    = "ftp"
	TypeWebDAV = "webdav"
)

// ValidType 报告 typ 是否是受支持的网络来源类型（本地来源不经 file_source 表配置）。
func ValidType(typ string) bool {
	switch typ {
	case TypeOSS, TypeFTP, TypeWebDAV:
		return true
	default:
		return false
	}
}

// CapsForType 返回某来源类型的静态能力（供前端隐藏/禁用动作；与各 Store.Caps() 一致）。
func CapsForType(typ string) Caps {
	switch typ {
	case "", TypeLocal:
		return localCaps
	case TypeOSS:
		return Caps{Presign: true, Copy: true, Move: true, Compress: false, ResumableUpload: true}
	case TypeFTP:
		return Caps{Presign: false, Copy: false, Move: true, Compress: false, ResumableUpload: true}
	case TypeWebDAV:
		return Caps{Presign: false, Copy: true, Move: true, Compress: false, ResumableUpload: true}
	default:
		return Caps{}
	}
}

// 可选能力接口：biz 用类型断言探测，缺失即对前端返回「不支持」。
type (
	// Presigner 能生成直链预签名 URL（启用 302 的前提，目前仅对象存储支持）。
	Presigner interface {
		Presign(ctx context.Context, path string, inline bool, ttl time.Duration) (string, error)
	}
	// Copier 支持把若干文件/目录复制进目标目录。
	Copier interface {
		CopyToDir(ctx context.Context, paths []string, targetDir string) []*v1.FileOperationResult
	}
	// Mover 支持把若干文件/目录移动进目标目录。
	Mover interface {
		MoveToDir(ctx context.Context, paths []string, targetDir string) []*v1.FileOperationResult
	}
	// Archiver 支持压缩与解压（仅本地来源实现）。
	Archiver interface {
		Compress(ctx context.Context, paths []string, targetPath string) (string, error)
		Unzip(ctx context.Context, path, targetPath string) (string, error)
	}
	// AsyncFinalizer 标记来源的上传收尾可能较慢（需把本地缓冲整流推送到远端），应作为后台任务执行
	//（biz 据此决定 CompleteUpload 是否返回异步任务）。仅缓冲型远端来源（FTP/WebDAV）实现。
	AsyncFinalizer interface {
		AsyncFinalize() bool
	}
)

// Config 是构造一个网络来源所需的（已解密）配置，字段为各类型的并集。
type Config struct {
	Type string // oss / ftp / webdav

	// OSS / S3
	Endpoint  string
	Region    string
	Bucket    string
	AccessKey string
	SecretKey string
	Prefix    string
	UseSSL    bool
	PathStyle bool

	// FTP / WebDAV
	Host     string
	Port     int
	Username string
	Password string
	BasePath string
	TLS      bool

	// WebDAV
	URL string
}

// NewStore 按类型构造一个网络来源（本地来源用 NewSystemStore / NewScopedStore）。
func NewStore(cfg Config) (Store, error) {
	switch cfg.Type {
	case TypeOSS:
		return newOSSStore(cfg)
	case TypeFTP:
		return newFTPStore(cfg)
	case TypeWebDAV:
		return newWebDAVStore(cfg)
	default:
		return nil, errors.New("不支持的文件来源类型")
	}
}
