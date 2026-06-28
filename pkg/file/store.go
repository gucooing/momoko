package file

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
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

// Upload 是一次分片上传会话的通用载体：biz 在 PrepareUpload/CompleteUpload/CancelUpload 间盲传它，
// 各来源在接口内部自行处理机制（OSS=预签名直传多分片；本地/FTP/WebDAV=分片经 momoko 签名端点落本地缓冲后放置）。
type Upload struct {
	ID         string // 上传会话 id（= file_upload 行 id）
	UserID     string // 发起用户
	Path       string // 目标逻辑目录（本地来源为真实路径）
	FileName   string // 目标文件名
	FileSize   uint64
	PartSize   uint64
	TotalParts uint64

	// Ref 是来源侧续传句柄（如 S3 multipart uploadID），对调用方不透明；为空表示尚未建立。
	// 续传/收尾/取消前由 biz 预填入；来源在 PrepareUpload 内按需建立并经 SaveRef 持久化。
	Ref string

	// SignPart 由 biz 注入：返回某分片经 momoko 签名的 PUT URL（缓冲型来源用，OSS 忽略）。
	SignPart func(part uint64) string
	// SaveRef 由 biz 注入：幂等持久化来源续传句柄并返回最终生效值（并发竞态下的胜出者）。
	SaveRef func(ctx context.Context, ref string) (string, error)

	// PartURLs 由来源在 PrepareUpload 内填充：分片号→PUT 直传地址（前端逐片直传，来源透明）。
	PartURLs map[uint64]string
	// Uploaded 已完成分片号→标识(hash/etag)：biz 先以本地分片表预填，来源可在 PrepareUpload 内补充（OSS 以远端为准）。
	Uploaded map[uint64]string
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

// ResolveLocalPath 把某本地根（basePath，""=系统全盘）下的逻辑路径解析为真实路径，并做穿越防护。
// 供需要真实文件系统路径的本地场景（下载签名、分享打包）使用，避免业务层依赖 FileOper。
func ResolveLocalPath(basePath, logical string) (string, error) {
	oper, err := NewFileOper(basePath)
	if err != nil {
		return "", err
	}
	return oper.ResolveRealPath(logical)
}

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

// ---- 本地分片上传：分片经 momoko 签名端点落本地缓冲，收尾时原子 rename 到目标真实路径 ----

func (s *LocalStore) PrepareUpload(_ context.Context, u *Upload) error {
	fillSignedParts(u)
	return nil
}

func (s *LocalStore) CompleteUpload(_ context.Context, u *Upload) error {
	finalPath := filepath.Join(u.Path, u.FileName)
	if u.TotalParts == 0 { // 空文件无分片，直接落一个空文件
		f, err := os.OpenFile(finalPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
		if err != nil {
			return fmt.Errorf("创建文件失败: %w", err)
		}
		return f.Close()
	}
	bufPath := bufferPath(true, u.Path, u.FileName, u.ID)
	if err := verifyBuffer(bufPath, uint64(len(u.Uploaded)), u.TotalParts, u.FileSize); err != nil {
		return err
	}
	if err := os.Rename(bufPath, finalPath); err != nil {
		return fmt.Errorf("重命名文件失败: %w", err)
	}
	return nil
}

func (s *LocalStore) CancelUpload(_ context.Context, u *Upload) error {
	if err := os.Remove(bufferPath(true, u.Path, u.FileName, u.ID)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// ---- 缓冲型上传共用助手（本地/FTP/WebDAV）----

// fillSignedParts 用注入的 SignPart 把各分片填成 momoko 签名 PUT URL（前端逐片直传到 momoko 端点）。
func fillSignedParts(u *Upload) {
	u.PartURLs = make(map[uint64]string, u.TotalParts)
	if u.SignPart == nil {
		return
	}
	for i := uint64(1); i <= u.TotalParts; i++ {
		u.PartURLs[i] = u.SignPart(i)
	}
}

// verifyBuffer 校验缓冲文件已齐备：分片数齐全且文件实际大小等于声明大小。
// 每片大小/偏移已在分片接收（UploadFilePart）时校验，这里只把关「漏片」与「总长」。
func verifyBuffer(bufPath string, partCount, totalParts, fileSize uint64) error {
	if partCount != totalParts {
		return fmt.Errorf("分片未上传完成：%d/%d", partCount, totalParts)
	}
	fi, err := os.Stat(bufPath)
	if err != nil {
		return fmt.Errorf("临时文件缺失: %w", err)
	}
	if uint64(fi.Size()) != fileSize {
		return fmt.Errorf("文件大小不一致：实际 %d，声明 %d", fi.Size(), fileSize)
	}
	return nil
}

// completeBufferedRemote 把缓冲文件整流推送到远端来源，成功后删缓冲。供 FTP/WebDAV 等缓冲型远端共用。
func completeBufferedRemote(ctx context.Context, dst Store, u *Upload) error {
	finalPath := path.Join(u.Path, u.FileName)
	if u.TotalParts == 0 { // 空文件直接写空对象
		return dst.Write(ctx, finalPath, strings.NewReader(""))
	}
	bufPath := bufferPath(false, u.Path, u.FileName, u.ID)
	if err := verifyBuffer(bufPath, uint64(len(u.Uploaded)), u.TotalParts, u.FileSize); err != nil {
		return err
	}
	f, err := os.Open(bufPath)
	if err != nil {
		return fmt.Errorf("打开临时文件失败: %w", err)
	}
	if err := dst.Write(ctx, finalPath, f); err != nil {
		_ = f.Close()
		return fmt.Errorf("上传到来源失败: %w", err)
	}
	_ = f.Close()
	_ = os.Remove(bufPath)
	return nil
}

// cancelBufferedRemote 删除远端缓冲型上传的本地缓冲。
func cancelBufferedRemote(u *Upload) error {
	if err := os.Remove(bufferPath(false, u.Path, u.FileName, u.ID)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}
