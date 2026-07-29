package biz

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/schema/sharetype"
	"momoko/pkg/auth"
	"momoko/pkg/cache"
	"momoko/pkg/file"
	"momoko/pkg/pre"
	"momoko/pkg/secretbox"
	"momoko/pkg/share"
	"momoko/pkg/task"
	"momoko/pkg/utils"
)

const (
	PreFileDownload = "/api/v1/download/pre" // 下载路径
	PreFileUpload   = "/api/v1/upload/pre"   // 上传路径

	// ShareDownloadPath 公开分享下载入口；/api/v1/public/ 前缀免 JWT，业务层用签名自校验。
	ShareDownloadPath = "/api/v1/public/share/download"

	UploadPeriod = 2 * time.Hour
	// ShareSessionTTL 分享会话签名默认有效期（限时分享下会被裁剪到不超过分享过期时间）。
	ShareSessionTTL = 2 * time.Hour
	// fileServeTTL 预签名直链(302)对外暴露的有效期。
	fileServeTTL = 5 * time.Minute
)

var uploadCache = cache.New[string, *file.ChunkedUpload](UploadPeriod) // 全局上传缓存

type FileRepo interface {
	GetOrCreate(ctx context.Context, userId string, info *file.ChunkedUpload) (*gen.FileUpload, error)
	WithTx(ctx context.Context, fn func(tx *gen.Tx) error) error
	Query(ctx context.Context, uid string) (*gen.FileUpload, error)
	QueryByUserID(ctx context.Context, userID, id string) (*gen.FileUpload, error)
	SaveChunkRecord(ctx context.Context, uploadID string, chunk uint64, hash string, size uint64) error
	ListStaleUploads(ctx context.Context, before time.Time) ([]*gen.FileUpload, error)
	DeleteUploadCascade(ctx context.Context, id string) error
	SetUploadCompleted(ctx context.Context, id string) error
	// SetUploadProviderRefIfEmpty 仅当 provider_ref 仍为空时写入（OSS multipart uploadID 初始化幂等/防竞态）。
	SetUploadProviderRefIfEmpty(ctx context.Context, id, ref string) (bool, error)

	CreateShare(ctx context.Context, userID, token, name string, items []sharetype.Item, req *v1.CreateShareRequest) (*gen.FileShare, error)
	ListShares(ctx context.Context, userID string, page, pageSize int64, keywords string) ([]*gen.FileShare, int64, error)
	GetShareByToken(ctx context.Context, token string) (*gen.FileShare, error)
	UpdateShare(ctx context.Context, userID, name string, items []sharetype.Item, req *v1.UpdateShareRequest) (*gen.FileShare, error)
	DeleteShare(ctx context.Context, userID, id string) error
	IncrShareDownload(ctx context.Context, id string) error

	CreateFileSource(ctx context.Context, userID, name, typ, configJSON string, enabled, redirect302 bool) (*gen.FileSource, error)
	ListFileSources(ctx context.Context, keywords string, enabledOnly bool) ([]*gen.FileSource, error)
	GetFileSource(ctx context.Context, id string) (*gen.FileSource, error)
	UpdateFileSource(ctx context.Context, id, name, configJSON string, enabled, redirect302 bool) (*gen.FileSource, error)
	DeleteFileSource(ctx context.Context, id string) error
}

type FileUsecase struct {
	repo  FileRepo
	box   *secretbox.Box
	tasks *task.Manager
}

// NewFileUsecase 创建文件操作用例。
func NewFileUsecase(repo FileRepo, tasks *task.Manager) *FileUsecase {
	if _, err := os.Stat(file.ServersPath); os.IsNotExist(err) {
		os.MkdirAll(file.ServersPath, 0755)
	}
	// 远端来源上传的本地分片缓冲目录。
	os.MkdirAll(file.UploadBufferDir, 0755)
	uc := &FileUsecase{
		repo:  repo,
		box:   secretbox.New(auth.AuthSecretKey),
		tasks: tasks,
	}
	// 注册文件传输/收尾任务工厂（开机/重试时按 payload 重建对应来源的 Copier/Mover/收尾会话）。
	uc.registerTaskFactories()
	// 恢复本域未完成的一次性任务：复制/剪切=不可重入(重启即标失败)，远端上传收尾=可重入(重启续做)。
	_ = tasks.Resume(context.Background(), file.TaskTypeCopy, file.TaskTypeMove, file.TaskTypeUploadFinalize)
	// 上传会话清理改为定时单例任务（开机随任务管理器启动），清理时一并中止 OSS 残留 multipart。
	gcTask := file.NewUploadGCTask(repo, uploadGCInterval, 2*UploadPeriod, uc.cancelStaleUpload)
	_ = tasks.EnsureSingleton(context.Background(), gcTask)
	return uc
}

// cancelStaleUpload 在清理残留上传会话时经来源中止其机制（删本地缓冲 / 中止 OSS multipart）。
func (f *FileUsecase) cancelStaleUpload(ctx context.Context, row *gen.FileUpload) {
	store, _, err := f.storeFor(ctx, row.SourceID)
	if err != nil {
		return
	}
	_ = store.CancelUpload(ctx, uploadFromRow(row))
}

// registerTaskFactories 注册文件传输任务的重建工厂；store 解析（含解密/实例本地根）留在 biz。
func (f *FileUsecase) registerTaskFactories() {
	f.tasks.Register(file.TaskTypeCopy, func(ctx context.Context, rec *task.Record) (task.Task, error) {
		p, err := parseTransferPayload(rec)
		if err != nil {
			return nil, err
		}
		copier, err := f.copierForPayload(ctx, p)
		if err != nil {
			return nil, err
		}
		return file.NewCopyTask(metaFromRecord(rec), copier, p), nil
	})
	f.tasks.Register(file.TaskTypeMove, func(ctx context.Context, rec *task.Record) (task.Task, error) {
		p, err := parseTransferPayload(rec)
		if err != nil {
			return nil, err
		}
		mover, err := f.moverForPayload(ctx, p)
		if err != nil {
			return nil, err
		}
		return file.NewMoveTask(metaFromRecord(rec), mover, p), nil
	})
	f.tasks.Register(file.TaskTypeUploadFinalize, func(ctx context.Context, rec *task.Record) (task.Task, error) {
		var p file.FinalizePayload
		if err := json.Unmarshal(rec.Payload, &p); err != nil {
			return nil, err
		}
		row, err := f.repo.QueryByUserID(ctx, rec.UserID, p.UploadID)
		if err != nil {
			return nil, err
		}
		store, _, err := f.storeFor(ctx, row.SourceID)
		if err != nil {
			return nil, err
		}
		return file.NewFinalizeTask(metaFromRecord(rec), store, uploadFromRow(row), f.repo), nil
	})
}

// uploadGCInterval 是清理废弃上传会话的周期。
const uploadGCInterval = 30 * time.Minute

// storeFor 按来源 id 解析存储后端：空/"local" 为本地磁盘（系统级，含 Windows 盘符虚拟根），
// 否则从 file_source 表读取并解密配置后构造对应来源。第二返回值在远端来源时为来源记录。
func (f *FileUsecase) storeFor(ctx context.Context, sourceID string) (file.Store, *gen.FileSource, error) {
	if sourceID == "" || sourceID == "local" {
		s, err := file.NewLocalStore(file.SystemPath)
		if err != nil {
			return nil, nil, ErrSystem(err)
		}
		return s, nil, nil
	}
	rec, err := f.repo.GetFileSource(ctx, sourceID)
	if err != nil {
		return nil, nil, ErrFileSourceNotFound
	}
	if !rec.Enabled {
		return nil, nil, ErrFileSourceDisabled
	}
	cfg, err := f.decryptedConfig(rec)
	if err != nil {
		return nil, rec, ErrSystem(err)
	}
	s, err := file.NewStore(cfg)
	if err != nil {
		return nil, rec, ErrSystem(err)
	}
	return s, rec, nil
}

// GetFileSystemList 获取文件列表（按来源）。
func (f *FileUsecase) GetFileSystemList(ctx context.Context, req *v1.GetFileSystemListRequest) (*v1.GetFileSystemListResponse, error) {
	store, _, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return nil, err
	}

	var result []*v1.FileEntryInfo
	if req.Keywords != nil {
		result, err = store.Search(ctx, req.Path, req.GetKeywords(), req.GetIncludeSubDir())
	} else {
		result, err = store.List(ctx, req.Path, req.SortField, req.IsDesc)
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	// 分页
	total := int64(len(result))
	pages := make([]*v1.FileEntryInfo, 0)
	start := (req.Page - 1) * req.PageSize
	end := req.Page * req.PageSize
	if start >= total || start < 0 {
		pages = nil
	} else {
		if end > total {
			end = total
		}
		pages = result[start:end]
	}
	directory, err := store.DirInfo(ctx, req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}

	return &v1.GetFileSystemListResponse{
		Directory: directory,
		Items:     pages,
		Page:      req.Page,
		PageSize:  req.PageSize,
		Total:     total,
	}, nil
}

// GetFileSystemTree 列出指定目录的直接子项（懒加载，供编辑器文件树使用）。
func (f *FileUsecase) GetFileSystemTree(ctx context.Context, req *v1.GetFileSystemTreeRequest) (*v1.GetFileSystemTreeResponse, error) {
	store, _, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return nil, err
	}
	entries, err := store.List(ctx, req.Path, v1.FileSortField_FILE_SORT_FIELD_NAME, false)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.GetFileSystemTreeResponse{
		Path:  req.Path,
		Nodes: toTreeNodes(entries),
	}, nil
}

// toTreeNodes 把文件条目裁剪为目录树节点（仅名称/路径/是否目录）。
func toTreeNodes(entries []*v1.FileEntryInfo) []*v1.FileTreeNode {
	nodes := make([]*v1.FileTreeNode, 0, len(entries))
	for _, e := range entries {
		nodes = append(nodes, &v1.FileTreeNode{
			Name:  e.Name,
			Path:  e.Path,
			IsDir: e.IsDir,
		})
	}
	return nodes
}

// BatchDeleteFileSystem 批量删除文件。
func (f *FileUsecase) BatchDeleteFileSystem(ctx context.Context, req *v1.BatchDeleteFileSystemRequest) (*v1.BatchDeleteFileSystemResponse, error) {
	store, _, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return nil, err
	}
	return &v1.BatchDeleteFileSystemResponse{
		Items: store.Delete(ctx, req.Paths),
	}, nil
}

// BatchCompressFileSystem 压缩文件或目录（仅支持的来源）。
func (f *FileUsecase) BatchCompressFileSystem(ctx context.Context, req *v1.BatchCompressFileSystemRequest) (*v1.BatchCompressFileSystemResponse, error) {
	store, _, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return nil, err
	}
	archiver, ok := store.(file.Archiver)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	outputPath, err := archiver.Compress(ctx, req.Paths, req.GetTargetPath())
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.BatchCompressFileSystemResponse{OutputPath: outputPath}, nil
}

// UnzipFileSystem 解压压缩包（仅支持的来源）。
func (f *FileUsecase) UnzipFileSystem(ctx context.Context, req *v1.UnzipFileSystemRequest) (*v1.UnzipFileSystemResponse, error) {
	store, _, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return nil, err
	}
	archiver, ok := store.(file.Archiver)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	outputPath, err := archiver.Unzip(ctx, req.Path, req.GetTargetPath())
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.UnzipFileSystemResponse{OutputPath: outputPath}, nil
}

// CreateFileSystem 创建文件。
func (f *FileUsecase) CreateFileSystem(ctx context.Context, req *v1.CreateFileSystemRequest) (*v1.CreateFileSystemResponse, error) {
	store, _, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return nil, err
	}
	if err = store.Create(ctx, req.Info); err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.CreateFileSystemResponse{}, nil
}

// RenameFileSystem 重命名文件或目录。
func (f *FileUsecase) RenameFileSystem(ctx context.Context, req *v1.RenameFileSystemRequest) (*v1.RenameFileSystemResponse, error) {
	store, _, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return nil, err
	}
	path, err := store.Rename(ctx, req.Path, req.NewName)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.RenameFileSystemResponse{Path: path}, nil
}

func (f *FileUsecase) CopyFileSystem(ctx context.Context, userID string, req *v1.CopyFileSystemRequest) (*v1.CopyFileSystemResponse, error) {
	store, _, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return nil, err
	}
	copier, ok := store.(file.Copier)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	payload := file.TransferPayload{SourceID: req.SourceId, Paths: req.Paths, Target: req.TargetPath}
	t := file.NewCopyTask(task.Meta{UserID: userID, Title: transferTitle("复制", req.Paths)}, copier, payload)
	info, err := submitTransfer(ctx, f.tasks, t)
	if err != nil {
		return nil, err
	}
	return &v1.CopyFileSystemResponse{Task: info}, nil
}

func (f *FileUsecase) CutFileSystem(ctx context.Context, userID string, req *v1.CutFileSystemRequest) (*v1.CutFileSystemResponse, error) {
	store, _, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return nil, err
	}
	mover, ok := store.(file.Mover)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	payload := file.TransferPayload{SourceID: req.SourceId, Paths: req.Paths, Target: req.TargetPath}
	t := file.NewMoveTask(task.Meta{UserID: userID, Title: transferTitle("移动", req.Paths)}, mover, payload)
	info, err := submitTransfer(ctx, f.tasks, t)
	if err != nil {
		return nil, err
	}
	return &v1.CutFileSystemResponse{Task: info}, nil
}

func (f *FileUsecase) GetFileTask(ctx context.Context, userID, taskID string) (*v1.FileTaskInfo, error) {
	info, ok := f.tasks.Get(taskID)
	if !ok || info.UserID != userID {
		return nil, ErrFileTaskNotFound
	}
	return toFileTaskInfo(info), nil
}

// submitTransfer 提交一个复制/移动任务并返回其文件任务视图（系统/实例共用）。
func submitTransfer(ctx context.Context, mgr *task.Manager, t task.Task) (*v1.FileTaskInfo, error) {
	id, err := mgr.Submit(ctx, t)
	if err != nil {
		return nil, ErrSystem(err)
	}
	info, ok := mgr.Get(id)
	if !ok {
		return nil, ErrFileTaskNotFound
	}
	return toFileTaskInfo(info), nil
}

// storeForPayload 按传输任务的 payload 解析目标 Store：实例本地根优先，否则按系统来源 id。
func (f *FileUsecase) storeForPayload(ctx context.Context, p file.TransferPayload) (file.Store, error) {
	if p.BasePath != "" {
		s, err := file.NewLocalStore(p.BasePath)
		if err != nil {
			return nil, ErrSystem(err)
		}
		return s, nil
	}
	store, _, err := f.storeFor(ctx, p.SourceID)
	return store, err
}

func (f *FileUsecase) copierForPayload(ctx context.Context, p file.TransferPayload) (file.Copier, error) {
	store, err := f.storeForPayload(ctx, p)
	if err != nil {
		return nil, err
	}
	copier, ok := store.(file.Copier)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	return copier, nil
}

func (f *FileUsecase) moverForPayload(ctx context.Context, p file.TransferPayload) (file.Mover, error) {
	store, err := f.storeForPayload(ctx, p)
	if err != nil {
		return nil, err
	}
	mover, ok := store.(file.Mover)
	if !ok {
		return nil, ErrFileSourceUnsupported
	}
	return mover, nil
}

func parseTransferPayload(rec *task.Record) (file.TransferPayload, error) {
	var p file.TransferPayload
	if err := json.Unmarshal(rec.Payload, &p); err != nil {
		return p, err
	}
	return p, nil
}

func metaFromRecord(rec *task.Record) task.Meta {
	return task.Meta{ID: rec.ID, Title: rec.Title, UserID: rec.UserID}
}

func transferTitle(verb string, paths []string) string {
	if len(paths) == 1 {
		return fmt.Sprintf("%s「%s」", verb, filepath.Base(paths[0]))
	}
	return fmt.Sprintf("%s %d 个项目", verb, len(paths))
}

func toFileTaskInfo(info *task.Info) *v1.FileTaskInfo {
	if info == nil {
		return nil
	}
	out := &v1.FileTaskInfo{
		TaskId:     info.ID,
		Operation:  info.Type,
		Status:     toFileTaskStatus(info.Status),
		Total:      info.Total,
		Finished:   info.Finished,
		Message:    info.Message,
		Items:      make([]*v1.FileOperationResult, 0, len(info.Results)),
		CreateTime: timestamppb.New(info.CreateTime),
	}
	for _, r := range info.Results {
		out.Items = append(out.Items, &v1.FileOperationResult{Path: r.Path, Success: r.Success, Message: r.Message})
	}
	if info.EndTime != nil {
		out.UpdateTime = timestamppb.New(*info.EndTime)
	} else {
		out.UpdateTime = timestamppb.New(info.CreateTime)
	}
	return out
}

func toFileTaskStatus(s task.Status) v1.FileTaskStatus {
	switch s {
	case task.StatusPending:
		return v1.FileTaskStatus_FILE_TASK_STATUS_PENDING
	case task.StatusRunning:
		return v1.FileTaskStatus_FILE_TASK_STATUS_RUNNING
	case task.StatusSuccess:
		return v1.FileTaskStatus_FILE_TASK_STATUS_SUCCESS
	default:
		// 失败/取消都映射为失败（FileTaskStatus 无取消态）。
		return v1.FileTaskStatus_FILE_TASK_STATUS_FAILED
	}
}

// EditFileSystemFile 覆盖保存文件内容。
func (f *FileUsecase) EditFileSystemFile(ctx context.Context, req *v1.EditFileSystemFileRequest) (*v1.EditFileSystemFileResponse, error) {
	store, _, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return nil, err
	}
	if len(req.Content) > file.MaxLoadFileSize {
		return nil, ErrSystem(fmt.Errorf("文件太大"))
	}
	if err = store.Write(ctx, req.Path, bytes.NewReader(req.Content)); err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.EditFileSystemFileResponse{}, nil
}

// FileSystemPreSign 生成预览/下载用的短期签名 URL（inline=true 为内联预览）。
func (f *FileUsecase) FileSystemPreSign(ctx context.Context, userID string, req *v1.FileSystemPreSignRequest) (string, error) {
	store, src, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return "", err
	}
	if _, err := store.Stat(ctx, req.Path); err != nil {
		return "", ErrFileNotExist
	}
	// 来源开启 302 且支持预签名（OSS）→ 直接返回来源预签名 GET 直链，浏览器直连来源、momoko 不在数据路径。
	if src != nil && src.Redirect302 {
		if presigner, ok := store.(file.Presigner); ok {
			if u, err := presigner.Presign(ctx, req.Path, req.Inline, fileServeTTL); err == nil {
				return u, nil
			}
		}
	}
	// 其余来源：返回 momoko 签名下载 URL（由 FileServe 代理串流/Range，或 302 兜底）。
	preInfo := pre.NewFileSignInfo(utils.GenerateRandomString(10), req.Path, userID, 24*time.Hour)
	preInfo.SourceID = req.SourceId
	preInfo.Inline = req.Inline
	sign, err := preInfo.Sign()
	if err != nil {
		return "", ErrSign
	}
	return fmt.Sprintf("%s?sign=%s", PreFileDownload, sign), nil
}

// FileServe 校验签名后服务文件：可直链(302)的来源且开启 redirect_302 时跳转到来源预签名 URL，
// 否则由后端代理串流（支持 Range，inline 预览 / attachment 下载）。
func (f *FileUsecase) FileServe(ctx context.Context, info *pre.FileSignInfo, w http.ResponseWriter, r *http.Request) {
	store, src, err := f.storeFor(ctx, info.SourceID)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	// 来源开启 302 且支持预签名 → 直链跳转，省服务器带宽。
	if src != nil && src.Redirect302 {
		if presigner, ok := store.(file.Presigner); ok {
			if u, err := presigner.Presign(ctx, info.Path, info.Inline, fileServeTTL); err == nil {
				http.Redirect(w, r, u, http.StatusFound)
				return
			}
		}
	}
	rc, entry, err := store.Open(ctx, info.Path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer rc.Close()

	disposition := "attachment"
	if info.Inline {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", disposition+`; filename="`+entry.Name+`"`)

	var mod time.Time
	if entry.UpdateTime != nil {
		mod = entry.UpdateTime.AsTime()
	}
	// 可 Seek 的来源（本地、OSS）走 ServeContent，支持 Range 续传/拖动。
	if seeker, ok := rc.(io.ReadSeeker); ok {
		w.Header().Set("Accept-Ranges", "bytes")
		http.ServeContent(w, r, entry.Name, mod, seeker)
		return
	}
	// 不可 Seek 的来源（FTP/WebDAV）：整流返回，不支持 Range。
	ctype := mime.TypeByExtension(filepath.Ext(entry.Name))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	if entry.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatUint(entry.Size, 10))
	}
	_, _ = io.Copy(w, rc)
}

func (f *FileUsecase) FileSystemPreSignUpload(ctx context.Context, userID string, req *v1.FileSystemPreSignUploadRequest) (*v1.UploadInfo, error) {
	if req.FileSize > math.MaxInt64 {
		return nil, ErrUploadRequestInvalid
	}
	store, _, err := f.storeFor(ctx, req.SourceId)
	if err != nil {
		return nil, err
	}
	// 本地来源落地到真实路径（便于原子 rename）；远端来源用逻辑路径（来源在接口内自行处理）。
	targetPath := req.Path
	if req.SourceId == "" {
		if targetPath, err = file.ResolveLocalPath(file.SystemPath, req.Path); err != nil {
			return nil, ErrSystem(err)
		}
	}
	seed := file.NewChunkedUpload(req.Hash, targetPath, req.FileName, req.FileSize)
	seed.SourceID = req.SourceId
	row, err := f.repo.GetOrCreate(ctx, userID, seed)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if row.Completed {
		return toUploadInfo(row, &file.Upload{}), nil
	}
	u, err := buildUpload(f.repo, userID, row)
	if err != nil {
		return nil, err
	}
	// 盲调用：来源在接口内自行处理机制（OSS 预签名直传 / 本地·FTP·WebDAV momoko 缓冲）。
	if err := store.PrepareUpload(ctx, u); err != nil {
		return nil, ErrSystem(err)
	}
	row.ProviderRef = u.Ref // 来源可能在 Prepare 内建立并持久化续传句柄，回写以保持内存行一致
	uploadCache.Set(row.ID, &file.ChunkedUpload{FileUpload: row})
	return toUploadInfo(row, u), nil
}

// uploadFromRow 由数据库行构造通用上传载体（不含签名/续传闭包，供取消/收尾的盲调用）。
func uploadFromRow(row *gen.FileUpload) *file.Upload {
	u := &file.Upload{
		ID:         row.ID,
		UserID:     row.UserID,
		Path:       row.Path,
		FileName:   row.FileName,
		FileSize:   row.FileSize,
		PartSize:   row.ChunkSize,
		TotalParts: row.TotalChunks,
		Ref:        row.ProviderRef,
		Uploaded:   map[uint64]string{},
	}
	for _, c := range row.Edges.Chunks {
		u.Uploaded[c.Chunk] = c.Hash
	}
	return u
}

// buildUpload 在 uploadFromRow 基础上注入分片签名端点与续传句柄持久化闭包（供准备/状态/收尾）。
// SignPart 返回 momoko 签名 PUT URL（缓冲型来源用）；SaveRef 幂等持久化来源句柄并返回竞态胜出者。
func buildUpload(repo FileRepo, userID string, row *gen.FileUpload) (*file.Upload, error) {
	sign, err := pre.NewFileSignInfo(row.ID, row.Path, userID, UploadPeriod).Sign()
	if err != nil {
		return nil, ErrSign
	}
	u := uploadFromRow(row)
	u.UserID = userID
	u.SignPart = func(part uint64) string {
		return fmt.Sprintf("%s?sign=%s&chunk=%d", PreFileUpload, sign, part)
	}
	u.SaveRef = func(ctx context.Context, ref string) (string, error) {
		set, err := repo.SetUploadProviderRefIfEmpty(ctx, row.ID, ref)
		if err != nil {
			return "", err
		}
		if set {
			return ref, nil
		}
		cur, err := repo.QueryByUserID(ctx, userID, row.ID)
		if err != nil {
			return "", err
		}
		return cur.ProviderRef, nil
	}
	return u, nil
}

// getChunkedUpload 获取 momoko 缓冲型上传会话（供分片接收端点 PreFileUpload 写缓冲）。
// 缓存未命中（如服务重启）时从数据库重建，使断点续传不依赖进程内缓存。
func (f *FileUsecase) getChunkedUpload(ctx context.Context, uploadID, userID string) (*file.ChunkedUpload, error) {
	if upload, ok := uploadCache.Get(uploadID); ok {
		return upload, nil
	}
	info, err := f.repo.QueryByUserID(ctx, userID, uploadID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	upload := &file.ChunkedUpload{FileUpload: info}
	uploadCache.Set(uploadID, upload)
	return upload, nil
}

// GetFileUploadStatus 返回上传会话的已传分片与分片直传地址（前端据此补传缺片）。
// 盲调用 PrepareUpload：来源在接口内刷新 u.Uploaded（OSS 以来源为准，其余以本地分片表为准）。
func (f *FileUsecase) GetFileUploadStatus(ctx context.Context, userID, uploadID string) (*v1.UploadInfo, error) {
	row, err := f.repo.QueryByUserID(ctx, userID, uploadID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if row.Completed || row.Cancel {
		return toUploadInfo(row, uploadFromRow(row)), nil
	}
	u, err := buildUpload(f.repo, userID, row)
	if err != nil {
		return nil, err
	}
	if store, _, err := f.storeFor(ctx, row.SourceID); err == nil {
		_ = store.PrepareUpload(ctx, u)
		row.ProviderRef = u.Ref
	}
	return toUploadInfo(row, u), nil
}

// CompleteFileUpload 收尾上传：来源在接口内自处理（本地 rename / OSS 合并分片 / 远端整流推送）。
// 收尾可能较慢的来源（缓冲型远端）放进任务管理器异步执行，返回非空 task 供前端轮询。
func (f *FileUsecase) CompleteFileUpload(ctx context.Context, userID, uploadID string) (*v1.FileTaskInfo, error) {
	row, err := f.repo.QueryByUserID(ctx, userID, uploadID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if row.Completed {
		return nil, nil
	}
	store, _, err := f.storeFor(ctx, row.SourceID)
	if err != nil {
		return nil, err
	}
	// 收尾较慢的来源（缓冲型远端）→ 放进任务管理器用不超时 ctx 执行，前端轮询任务。
	if fa, ok := store.(file.AsyncFinalizer); ok && fa.AsyncFinalize() {
		meta := task.Meta{UserID: userID, Title: "上传收尾「" + row.FileName + "」"}
		t := file.NewFinalizeTask(meta, store, uploadFromRow(row), f.repo)
		return submitTransfer(ctx, f.tasks, t)
	}
	// 瞬时收尾（本地 rename / OSS 合并分片）：同步完成。
	if err := store.CompleteUpload(ctx, uploadFromRow(row)); err != nil {
		return nil, ErrSystem(err)
	}
	if err := f.repo.SetUploadCompleted(ctx, uploadID); err != nil {
		return nil, ErrSystem(err)
	}
	uploadCache.Del(uploadID)
	return nil, nil
}

// CancelFileUpload 取消上传：盲调用来源 CancelUpload（删缓冲 / 中止远端句柄），再标记取消并清缓存。
func (f *FileUsecase) CancelFileUpload(ctx context.Context, userID, uploadID string) error {
	row, err := f.repo.QueryByUserID(ctx, userID, uploadID)
	if err != nil {
		return ErrSystem(err)
	}
	if store, _, err := f.storeFor(ctx, row.SourceID); err == nil {
		_ = store.CancelUpload(ctx, uploadFromRow(row))
	}
	if err := f.repo.WithTx(ctx, func(tx *gen.Tx) error {
		_, e := tx.FileUpload.UpdateOneID(uploadID).SetCancel(true).Save(ctx)
		return e
	}); err != nil {
		return ErrSystem(err)
	}
	uploadCache.Del(uploadID)
	return nil
}

func (f *FileUsecase) PreFileUpload(w khttp.ResponseWriter, r *khttp.Request, pr *pre.FileSignInfo) error {
	ctx := context.Background()
	chunk, err := strconv.ParseUint(r.URL.Query().Get("chunk"), 10, 64)
	if err != nil {
		return err
	}
	info, err := f.getChunkedUpload(ctx, pr.UploadId, pr.Creator)
	if err != nil {
		return ErrSystem(err)
	}
	size, hash, err := info.UploadFilePart(r.Body, chunk)
	if err != nil {
		return err
	}
	// 写入hash
	if err = f.repo.SaveChunkRecord(ctx, pr.UploadId, chunk, hash, size); err != nil {
		return err
	}
	return nil
}

// ---- 文件来源（OSS/FTP/WebDAV，全局/管理员维护）----

func validSourceType(typ string) bool {
	return typ == "oss" || typ == "ftp" || typ == "webdav"
}

// parseStoredConfig 解析数据库中存储的配置 JSON（密钥字段仍为加密态）。
func (f *FileUsecase) parseStoredConfig(rec *gen.FileSource) (file.Config, error) {
	var cfg file.Config
	if rec.Config != "" {
		if err := json.Unmarshal([]byte(rec.Config), &cfg); err != nil {
			return cfg, err
		}
	}
	cfg.Type = string(rec.Type)
	return cfg, nil
}

// decryptedConfig 解析并解密密钥，得到可用于构造 Store 的明文配置。
func (f *FileUsecase) decryptedConfig(rec *gen.FileSource) (file.Config, error) {
	cfg, err := f.parseStoredConfig(rec)
	if err != nil {
		return cfg, err
	}
	if cfg.SecretKey != "" {
		if v, err := f.box.Decrypt(cfg.SecretKey); err == nil {
			cfg.SecretKey = v
		}
	}
	if cfg.Password != "" {
		if v, err := f.box.Decrypt(cfg.Password); err == nil {
			cfg.Password = v
		}
	}
	return cfg, nil
}

// buildStoredConfig 由请求配置生成入库 JSON（密钥加密）；existing 提供「更新时密钥留空=保留原值」的回退。
func (f *FileUsecase) buildStoredConfig(typ string, c *v1.FileSourceConfig, existing *file.Config) (string, error) {
	if c == nil {
		c = &v1.FileSourceConfig{}
	}
	cfg := file.Config{
		Type:      typ,
		Endpoint:  c.GetEndpoint(),
		Region:    c.GetRegion(),
		Bucket:    c.GetBucket(),
		AccessKey: c.GetAccessKey(),
		Prefix:    c.GetPrefix(),
		UseSSL:    c.GetUseSsl(),
		PathStyle: c.GetPathStyle(),
		Host:      c.GetHost(),
		Port:      int(c.GetPort()),
		Username:  c.GetUsername(),
		BasePath:  c.GetBasePath(),
		TLS:       c.GetTls(),
		URL:       c.GetUrl(),
	}
	if c.GetSecretKey() != "" {
		enc, err := f.box.Encrypt(c.GetSecretKey())
		if err != nil {
			return "", err
		}
		cfg.SecretKey = enc
	} else if existing != nil {
		cfg.SecretKey = existing.SecretKey // 已是加密态
	}
	if c.GetPassword() != "" {
		enc, err := f.box.Encrypt(c.GetPassword())
		if err != nil {
			return "", err
		}
		cfg.Password = enc
	} else if existing != nil {
		cfg.Password = existing.Password
	}
	b, err := json.Marshal(cfg)
	return string(b), err
}

// requestToConfig 直接用请求里的明文配置（测试未保存来源时使用）。
func requestToConfig(typ string, c *v1.FileSourceConfig) file.Config {
	if c == nil {
		c = &v1.FileSourceConfig{}
	}
	return file.Config{
		Type:      typ,
		Endpoint:  c.GetEndpoint(),
		Region:    c.GetRegion(),
		Bucket:    c.GetBucket(),
		AccessKey: c.GetAccessKey(),
		SecretKey: c.GetSecretKey(),
		Prefix:    c.GetPrefix(),
		UseSSL:    c.GetUseSsl(),
		PathStyle: c.GetPathStyle(),
		Host:      c.GetHost(),
		Port:      int(c.GetPort()),
		Username:  c.GetUsername(),
		Password:  c.GetPassword(),
		BasePath:  c.GetBasePath(),
		TLS:       c.GetTls(),
		URL:       c.GetUrl(),
	}
}

// toFileSourceInfo 映射为对外信息：密钥脱敏（清空），附带能力。
func (f *FileUsecase) toFileSourceInfo(rec *gen.FileSource) *v1.FileSourceInfo {
	cfg, _ := f.parseStoredConfig(rec)
	caps := file.CapsForType(string(rec.Type))
	creator := ""
	if rec.Edges.User != nil {
		creator = rec.Edges.User.Name
		if creator == "" {
			creator = rec.Edges.User.Username
		}
	}
	return &v1.FileSourceInfo{
		Id:           rec.ID,
		Name:         rec.Name,
		Type:         string(rec.Type),
		Enabled:      rec.Enabled,
		Redirect_302: rec.Redirect302,
		Config: &v1.FileSourceConfig{
			Endpoint:  cfg.Endpoint,
			Region:    cfg.Region,
			Bucket:    cfg.Bucket,
			AccessKey: cfg.AccessKey,
			Prefix:    cfg.Prefix,
			UseSsl:    cfg.UseSSL,
			PathStyle: cfg.PathStyle,
			Host:      cfg.Host,
			Port:      int32(cfg.Port),
			Username:  cfg.Username,
			BasePath:  cfg.BasePath,
			Tls:       cfg.TLS,
			Url:       cfg.URL,
			// SecretKey / Password 不返回
		},
		Caps: &v1.FileSourceCaps{
			Presign:         caps.Presign,
			Copy:            caps.Copy,
			Move:            caps.Move,
			Compress:        caps.Compress,
			ResumableUpload: caps.ResumableUpload,
		},
		CreatorName: creator,
		CreateTime:  timestamppb.New(rec.CreateTime),
		UpdateTime:  timestamppb.New(rec.UpdateTime),
	}
}

func (f *FileUsecase) ListFileSources(ctx context.Context, req *v1.ListFileSourcesRequest) (*v1.ListFileSourcesResponse, error) {
	items, err := f.repo.ListFileSources(ctx, req.GetKeywords(), req.GetEnabledOnly())
	if err != nil {
		return nil, ErrSystem(err)
	}
	resp := &v1.ListFileSourcesResponse{Items: make([]*v1.FileSourceInfo, 0, len(items))}
	for _, it := range items {
		resp.Items = append(resp.Items, f.toFileSourceInfo(it))
	}
	return resp, nil
}

func (f *FileUsecase) CreateFileSource(ctx context.Context, userID string, req *v1.CreateFileSourceRequest) (*v1.FileSourceInfo, error) {
	if !validSourceType(req.Type) {
		return nil, ErrFileSourceType
	}
	cfgJSON, err := f.buildStoredConfig(req.Type, req.Config, nil)
	if err != nil {
		return nil, ErrSystem(err)
	}
	rec, err := f.repo.CreateFileSource(ctx, userID, req.Name, req.Type, cfgJSON, req.Enabled, req.Redirect_302)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if full, err := f.repo.GetFileSource(ctx, rec.ID); err == nil {
		rec = full
	}
	return f.toFileSourceInfo(rec), nil
}

func (f *FileUsecase) UpdateFileSource(ctx context.Context, req *v1.UpdateFileSourceRequest) (*v1.FileSourceInfo, error) {
	rec, err := f.repo.GetFileSource(ctx, req.Id)
	if err != nil {
		return nil, ErrFileSourceNotFound
	}
	existing, _ := f.parseStoredConfig(rec)
	cfgJSON, err := f.buildStoredConfig(string(rec.Type), req.Config, &existing)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if _, err := f.repo.UpdateFileSource(ctx, req.Id, req.Name, cfgJSON, req.Enabled, req.Redirect_302); err != nil {
		return nil, ErrSystem(err)
	}
	full, err := f.repo.GetFileSource(ctx, req.Id)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return f.toFileSourceInfo(full), nil
}

func (f *FileUsecase) DeleteFileSource(ctx context.Context, id string) error {
	if err := f.repo.DeleteFileSource(ctx, id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// TestFileSource 测试来源连通性：传 id 测已存在来源；否则用请求里的明文配置测未保存的来源。
func (f *FileUsecase) TestFileSource(ctx context.Context, req *v1.TestFileSourceRequest) (*v1.TestFileSourceResponse, error) {
	var cfg file.Config
	if req.Id != "" {
		rec, err := f.repo.GetFileSource(ctx, req.Id)
		if err != nil {
			return &v1.TestFileSourceResponse{Ok: false, Message: "来源不存在"}, nil
		}
		cfg, err = f.decryptedConfig(rec)
		if err != nil {
			return &v1.TestFileSourceResponse{Ok: false, Message: err.Error()}, nil
		}
	} else {
		if !validSourceType(req.Type) {
			return &v1.TestFileSourceResponse{Ok: false, Message: "类型不合法"}, nil
		}
		cfg = requestToConfig(req.Type, req.Config)
	}
	store, err := file.NewStore(cfg)
	if err != nil {
		return &v1.TestFileSourceResponse{Ok: false, Message: err.Error()}, nil
	}
	if _, err := store.List(ctx, "", v1.FileSortField_FILE_SORT_FIELD_NAME, false); err != nil {
		return &v1.TestFileSourceResponse{Ok: false, Message: err.Error()}, nil
	}
	return &v1.TestFileSourceResponse{Ok: true, Message: "连接成功"}, nil
}

// ---- 分享 ----
// biz 仅做鉴权 + 仓储编排；令牌生成、提取码/有效期校验、目录浏览与打包下载等逻辑在 pkg/share。

// CreateShare 为选定条目（可跨来源）创建一条分享。逐条目经其来源探测并缓存名称/类型/大小/修改时间。
func (f *FileUsecase) CreateShare(ctx context.Context, userID string, req *v1.CreateShareRequest) (*v1.ShareInfo, error) {
	items, name, err := share.PrepareItems(ctx, f.storeFor, req.Items, req.Name)
	if err != nil {
		if errors.Is(err, share.ErrItemNotFound) {
			return nil, ErrFileNotExist
		}
		return nil, err
	}
	rec, err := f.repo.CreateShare(ctx, userID, share.GenToken(), name, items, req)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return share.ToInfo(rec), nil
}

// ListShares 返回某用户的分享列表。
func (f *FileUsecase) ListShares(ctx context.Context, userID string, req *v1.ListSharesRequest) (*v1.ListSharesResponse, error) {
	items, total, err := f.repo.ListShares(ctx, userID, req.Page, req.PageSize, req.GetKeywords())
	if err != nil {
		return nil, ErrSystem(err)
	}
	resp := &v1.ListSharesResponse{
		Page:     req.Page,
		PageSize: req.PageSize,
		Total:    total,
		Items:    make([]*v1.ShareInfo, 0, len(items)),
	}
	for _, it := range items {
		resp.Items = append(resp.Items, share.ToInfo(it))
	}
	return resp, nil
}

// UpdateShare 编辑分享（含启停/续期，过期分享亦可二次编辑开启）。
// items 非空时覆盖分享内容并重新探测/缓存各条目属性；为空则仅改元信息，不动内容。
func (f *FileUsecase) UpdateShare(ctx context.Context, userID string, req *v1.UpdateShareRequest) (*v1.ShareInfo, error) {
	name := req.Name
	var items []sharetype.Item
	if len(req.Items) > 0 {
		var err error
		items, name, err = share.PrepareItems(ctx, f.storeFor, req.Items, req.Name)
		if err != nil {
			if errors.Is(err, share.ErrItemNotFound) {
				return nil, ErrFileNotExist
			}
			return nil, err
		}
	}
	rec, err := f.repo.UpdateShare(ctx, userID, name, items, req)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrShareNotFound
		}
		return nil, ErrSystem(err)
	}
	return share.ToInfo(rec), nil
}

// DeleteShare 删除分享。
func (f *FileUsecase) DeleteShare(ctx context.Context, userID, id string) error {
	if err := f.repo.DeleteShare(ctx, userID, id); err != nil {
		return ErrSystem(err)
	}
	return nil
}

// GetShareMeta 公开：返回分享元信息（仅 name/isDir/needCode/可用性/分享者等，不含目录与文件数据，
// 作为「需验证」引导面，本身不构成越权面）。
func (f *FileUsecase) GetShareMeta(ctx context.Context, token string) (*v1.GetShareMetaResponse, error) {
	rec, err := f.repo.GetShareByToken(ctx, token)
	if err != nil {
		return nil, ErrShareNotFound
	}
	return share.ToMeta(rec, rec.Edges.User, time.Now()), nil
}

// CreateShareSession 公开：用 token(+提取码) 换取一次性会话签名。
// 限时分享下，签名有效期被裁剪到不超过分享过期时间。无签名/失效签名时前端据 need_code/message 弹验证。
func (f *FileUsecase) CreateShareSession(ctx context.Context, token, code string) (*v1.CreateShareSessionResponse, error) {
	rec, err := f.repo.GetShareByToken(ctx, token)
	if err != nil {
		return nil, ErrShareNotFound
	}
	if !rec.Enabled {
		return &v1.CreateShareSessionResponse{Message: "分享已关闭"}, nil
	}
	if rec.ExpiresAt != nil && time.Now().After(*rec.ExpiresAt) {
		return &v1.CreateShareSessionResponse{Message: "分享已过期"}, nil
	}
	if rec.MaxDownloads > 0 && rec.DownloadCount >= rec.MaxDownloads {
		return &v1.CreateShareSessionResponse{Message: "分享下载次数已用尽"}, nil
	}
	if rec.Code != "" && rec.Code != code {
		resp := &v1.CreateShareSessionResponse{NeedCode: true}
		if code != "" {
			resp.Message = "提取码错误"
		}
		return resp, nil
	}
	ttl := ShareSessionTTL
	if rec.ExpiresAt != nil {
		if remain := time.Until(*rec.ExpiresAt); remain < ttl {
			ttl = remain
		}
	}
	si := pre.NewShareSignInfo(rec.ID, ttl)
	sign, err := si.Sign()
	if err != nil {
		return nil, ErrSign
	}
	return &v1.CreateShareSessionResponse{
		Sign:      sign,
		ExpiresAt: timestamppb.New(time.Now().Add(ttl)),
	}, nil
}

// verifyShareSign 校验分享会话签名：签名有效、绑定到本分享、且分享仍可用。
func (f *FileUsecase) verifyShareSign(sign string, rec *gen.FileShare) error {
	info, err := pre.Verify(sign)
	if err != nil || info.ShareID != rec.ID {
		return ErrShareSignInvalid
	}
	if !share.Available(rec, time.Now()) {
		return ErrShareForbidden
	}
	return nil
}

// ListShareDir 公开：浏览文件夹分享的子目录（需会话签名）。
func (f *FileUsecase) ListShareDir(ctx context.Context, req *v1.ListShareDirRequest) (*v1.ListShareDirResponse, error) {
	rec, err := f.repo.GetShareByToken(ctx, req.Token)
	if err != nil {
		return nil, ErrShareNotFound
	}
	if err := f.verifyShareSign(req.Sign, rec); err != nil {
		return nil, err
	}
	// 传入来源解析器：根目录用缓存条目、无需访问来源；进入子目录时按该条目来源解析 Store。
	items, sub, err := share.ListDir(ctx, f.storeFor, rec, req.SubPath)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.ListShareDirResponse{Items: items, SubPath: sub}, nil
}

// ShareDownload 公开：校验会话签名后将分享目标写入响应（文件直传/文件夹打包 zip），
// 成功（且非预览、非 Range 续传）计一次下载。
func (f *FileUsecase) ShareDownload(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	q := r.URL.Query()
	rec, err := f.repo.GetShareByToken(ctx, q.Get("token"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err := f.verifyShareSign(q.Get("sign"), rec); err != nil {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("分享会话已失效，请重新验证"))
		return
	}
	inline := q.Get("inline") == "1"
	// 预览(inline)与 Range 续传分段不计入下载次数。
	countable := r.Header.Get("Range") == "" && !inline
	// 传入来源解析器：目标条目按其来源解析 Store，并由该来源自身的 redirect_302 决定是否 302 直链。
	if err := share.ServeDownload(ctx, f.storeFor, rec, q.Get("path"), inline, w, r); err != nil {
		return
	}
	if countable {
		_ = f.repo.IncrShareDownload(ctx, rec.ID)
	}
}

// toUploadInfo 由数据库行 + 通用上传载体构造上传信息：part_urls/uploaded 来自来源在接口内填充的结果，
// 来源透明（OSS=来源预签名直链；本地/FTP/WebDAV=momoko 签名端点）。
func toUploadInfo(row *gen.FileUpload, u *file.Upload) *v1.UploadInfo {
	info := &v1.UploadInfo{
		UploadId:      row.ID,
		PartSize:      row.ChunkSize,
		FileSize:      row.FileSize,
		TotalParts:    row.TotalChunks,
		UploadedParts: u.Uploaded,
		Completed:     row.Completed,
		Cancel:        row.Cancel,
		ExpiredAt:     timestamppb.New(row.CreateTime.Add(UploadPeriod)),
		PartUrls:      u.PartURLs,
	}
	if info.UploadedParts == nil {
		info.UploadedParts = make(map[uint64]string)
	}
	return info
}
