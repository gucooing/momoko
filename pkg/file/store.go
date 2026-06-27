package file

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"time"

	v1 "momoko/api/gen/v1"
)

// Store 是可插拔的文件存储后端抽象：本地磁盘为默认实现，另有 OSS(S3) / FTP / WebDAV。
// path 为来源内的逻辑路径（本地=真实路径并受穿越保护；对象存储=key 前缀下的相对路径；FTP/WebDAV=远端路径）。
// 各实现自行做边界与穿越防护。
type Store interface {
	List(ctx context.Context, dir string, field v1.FileSortField, desc bool) ([]*v1.FileEntryInfo, error)
	Search(ctx context.Context, dir, keywords string, recursive bool) ([]*v1.FileEntryInfo, error)
	DirInfo(ctx context.Context, dir string) (*v1.FileDirectoryInfo, error)
	Stat(ctx context.Context, path string) (*v1.FileEntryInfo, error)
	// Open 打开文件用于串流（预览/下载）。返回的 ReadCloser 若同时实现 io.Seeker，调用方可支持 Range。
	Open(ctx context.Context, path string) (io.ReadCloser, *v1.FileEntryInfo, error)
	// Read 读取小文件全部内容（编辑器用），max<=0 表示用默认上限。
	Read(ctx context.Context, path string, max int64) ([]byte, error)
	// Write 以流写入/覆盖一个文件（保存、上传落地用）。
	Write(ctx context.Context, path string, r io.Reader) error
	Create(ctx context.Context, item *v1.FileCreateItem) error
	Delete(ctx context.Context, paths []string) []*v1.FileOperationResult
	Rename(ctx context.Context, path, newName string) (string, error)
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

// CapsForType 返回某来源类型的静态能力（供前端隐藏/禁用动作；与各 Store.Caps() 一致）。
func CapsForType(typ string) Caps {
	switch typ {
	case "", "local":
		return Caps{Presign: false, Copy: true, Move: true, Compress: true, ResumableUpload: true}
	case "oss":
		return Caps{Presign: true, Copy: true, Move: true, Compress: false, ResumableUpload: true}
	case "ftp":
		return Caps{Presign: false, Copy: false, Move: true, Compress: false, ResumableUpload: true}
	case "webdav":
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
	// Archiver 支持压缩与解压（v1 仅本地实现）。
	Archiver interface {
		Compress(ctx context.Context, paths []string, targetPath string) (string, error)
		Unzip(ctx context.Context, path, targetPath string) (string, error)
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

// NewStore 按类型构造一个网络来源（本地来源用 NewLocalStore）。
func NewStore(cfg Config) (Store, error) {
	switch cfg.Type {
	case "oss":
		return newOSSStore(cfg)
	case "ftp":
		return newFTPStore(cfg)
	case "webdav":
		return newWebDAVStore(cfg)
	default:
		return nil, errors.New("不支持的文件来源类型")
	}
}

// LocalStore 是本地磁盘的 Store 实现，包装既有 FileOper（行为与历史一致）。
type LocalStore struct {
	oper *FileOper
}

// NewLocalStore 创建本地来源；basePath 为 SystemPath("") 时为系统级（含 Windows 盘符虚拟根）。
func NewLocalStore(basePath string) (*LocalStore, error) {
	oper, err := NewFileOper(basePath)
	if err != nil {
		return nil, err
	}
	return &LocalStore{oper: oper}, nil
}

// Oper 暴露底层 FileOper，供需要本地真实路径的逻辑（如下载/分享）使用。
func (s *LocalStore) Oper() *FileOper { return s.oper }

func (s *LocalStore) List(_ context.Context, dir string, field v1.FileSortField, desc bool) ([]*v1.FileEntryInfo, error) {
	return s.oper.ListDir(dir, field, desc)
}

func (s *LocalStore) Search(_ context.Context, dir, keywords string, recursive bool) ([]*v1.FileEntryInfo, error) {
	return s.oper.QueryDir(dir, keywords, recursive)
}

func (s *LocalStore) DirInfo(_ context.Context, dir string) (*v1.FileDirectoryInfo, error) {
	return s.oper.DirInfo(dir)
}

func (s *LocalStore) Stat(_ context.Context, path string) (*v1.FileEntryInfo, error) {
	real, err := s.oper.ResolveRealPath(path)
	if err != nil {
		return nil, err
	}
	info, err := os.Stat(real)
	if err != nil {
		return nil, errors.New("文件不存在")
	}
	return toFileEntryInfo(info, real), nil
}

func (s *LocalStore) Open(_ context.Context, path string) (io.ReadCloser, *v1.FileEntryInfo, error) {
	real, err := s.oper.ResolveRealPath(path)
	if err != nil {
		return nil, nil, err
	}
	f, err := os.Open(real)
	if err != nil {
		return nil, nil, errors.New("文件不存在")
	}
	info, err := f.Stat()
	if err != nil {
		_ = f.Close()
		return nil, nil, err
	}
	if info.IsDir() {
		_ = f.Close()
		return nil, nil, errors.New("目标是目录")
	}
	return f, toFileEntryInfo(info, real), nil
}

func (s *LocalStore) Read(_ context.Context, path string, _ int64) ([]byte, error) {
	return s.oper.LoadFile(path)
}

func (s *LocalStore) Write(_ context.Context, path string, r io.Reader) error {
	realDir, err := s.oper.ResolveRealPath(filepath.Dir(path))
	if err != nil {
		return err
	}
	target := filepath.Join(realDir, filepath.Base(path))
	f, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, r)
	return err
}

func (s *LocalStore) Create(_ context.Context, item *v1.FileCreateItem) error {
	return s.oper.Create(item)
}

func (s *LocalStore) Delete(_ context.Context, paths []string) []*v1.FileOperationResult {
	return s.oper.BatchDelete(paths)
}

func (s *LocalStore) Rename(_ context.Context, path, newName string) (string, error) {
	return s.oper.Rename(path, newName)
}

func (s *LocalStore) Caps() Caps {
	return Caps{Presign: false, Copy: true, Move: true, Compress: true, ResumableUpload: true}
}

// 本地实现可选能力（委托 FileOper；忽略 ctx）。
func (s *LocalStore) CopyToDir(_ context.Context, paths []string, targetDir string) []*v1.FileOperationResult {
	return s.oper.CopyToDir(paths, targetDir)
}

func (s *LocalStore) MoveToDir(_ context.Context, paths []string, targetDir string) []*v1.FileOperationResult {
	return s.oper.MoveToDir(paths, targetDir)
}

func (s *LocalStore) Compress(_ context.Context, paths []string, targetPath string) (string, error) {
	return s.oper.BatchCompress(paths, targetPath)
}

func (s *LocalStore) Unzip(_ context.Context, path, targetPath string) (string, error) {
	return s.oper.Unzip(path, targetPath)
}

var (
	_ Store    = (*LocalStore)(nil)
	_ Copier   = (*LocalStore)(nil)
	_ Mover    = (*LocalStore)(nil)
	_ Archiver = (*LocalStore)(nil)
)
