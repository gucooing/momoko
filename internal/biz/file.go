package biz

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"os"
	"strconv"
	"time"

	khttp "github.com/go-kratos/kratos/v2/transport/http"
	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/internal/data/ent/gen/fileupload"
	"momoko/pkg/cache"
	"momoko/pkg/file"
	"momoko/pkg/pre"
	"momoko/pkg/share"
	"momoko/pkg/utils"
)

const (
	PreFileDownload = "/api/v1/download/pre" // 下载路径
	PreFileUpload   = "/api/v1/upload/pre"   // 上传路径

	// ShareDownloadPath 公开分享下载入口；/api/v1/public/ 前缀免 JWT，业务层自校验令牌/提取码。
	ShareDownloadPath = "/api/v1/public/share/download"

	UploadPeriod = 2 * time.Hour
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

	CreateShare(ctx context.Context, userID, token, name, targetPath string, isDir bool, req *v1.CreateShareRequest) (*gen.FileShare, error)
	ListShares(ctx context.Context, userID string, page, pageSize int64, keywords string) ([]*gen.FileShare, int64, error)
	GetShareByToken(ctx context.Context, token string) (*gen.FileShare, error)
	UpdateShare(ctx context.Context, userID string, req *v1.UpdateShareRequest, targetPath string, isDir bool) (*gen.FileShare, error)
	DeleteShare(ctx context.Context, userID, id string) error
	IncrShareDownload(ctx context.Context, id string) error
}

type FileUsecase struct {
	repo FileRepo
}

// NewFileUsecase 创建文件操作用例。
func NewFileUsecase(repo FileRepo) *FileUsecase {
	if _, err := os.Stat(file.ServersPath); os.IsNotExist(err) {
		os.MkdirAll(file.ServersPath, 0755)
	}
	uc := &FileUsecase{
		repo: repo,
	}
	// 上传会话 GC 的清理逻辑位于 pkg/file，biz 仅负责装配并注入数据访问能力。
	file.StartUploadJanitor(context.Background(), repo, uploadGCInterval, 2*UploadPeriod)
	return uc
}

// uploadGCInterval 是清理废弃上传会话的周期。
const uploadGCInterval = 30 * time.Minute

func (f *FileUsecase) newSystemInstance() (*file.FileOper, error) {
	return file.NewFileOper(file.SystemPath)
}

// GetFileSystemList 获取系统文件列表。
func (f *FileUsecase) GetFileSystemList(ctx context.Context, req *v1.GetFileSystemListRequest) (*v1.GetFileSystemListResponse, error) {
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	var result []*v1.FileEntryInfo
	if req.Keywords != nil {
		result, err = fileOper.QueryDir(req.Path, req.GetKeywords(), req.GetIncludeSubDir())
	} else {
		result, err = fileOper.ListDir(req.Path, req.SortField, req.IsDesc)
	}
	if err != nil {
		return nil, ErrSystem(err)
	}
	// 分页
	pages := make([]*v1.FileEntryInfo, 0)
	total := int64(len(result))
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
	directory, err := fileOper.DirInfo(req.Path)
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
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}
	entries, err := fileOper.ListDir(req.Path, v1.FileSortField_FILE_SORT_FIELD_NAME, false)
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
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.BatchDeleteFileSystemResponse{
		Items: fileOper.BatchDelete(req.Paths),
	}, nil
}

// BatchCompressFileSystem 压缩系统文件或目录。
func (f *FileUsecase) BatchCompressFileSystem(ctx context.Context, req *v1.BatchCompressFileSystemRequest) (*v1.BatchCompressFileSystemResponse, error) {
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	outputPath, err := fileOper.BatchCompress(req.Paths, req.GetTargetPath())
	if err != nil {
		return nil, ErrSystem(err)
	}

	return &v1.BatchCompressFileSystemResponse{OutputPath: outputPath}, nil
}

// UnzipFileSystem 解压系统压缩包。
func (f *FileUsecase) UnzipFileSystem(ctx context.Context, req *v1.UnzipFileSystemRequest) (*v1.UnzipFileSystemResponse, error) {
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	outputPath, err := fileOper.Unzip(req.Path, req.GetTargetPath())
	if err != nil {
		return nil, ErrSystem(err)
	}

	return &v1.UnzipFileSystemResponse{OutputPath: outputPath}, nil
}

// CreateFileSystem 创建文件。
func (f *FileUsecase) CreateFileSystem(ctx context.Context, req *v1.CreateFileSystemRequest) (*v1.CreateFileSystemResponse, error) {
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	err = fileOper.Create(req.Info)
	if err != nil {
		return nil, ErrSystem(err)
	}

	return &v1.CreateFileSystemResponse{}, nil
}

// RenameFileSystem 重命名文件或目录。
func (f *FileUsecase) RenameFileSystem(ctx context.Context, req *v1.RenameFileSystemRequest) (*v1.RenameFileSystemResponse, error) {
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	path, err := fileOper.Rename(req.Path, req.NewName)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.RenameFileSystemResponse{Path: path}, nil
}

func (f *FileUsecase) CopyFileSystem(
	ctx context.Context,
	userID string,
	req *v1.CopyFileSystemRequest,
) (*v1.CopyFileSystemResponse, error) {
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}
	task, err := fileOper.StartCopyToDirTask(userID, req.Paths, req.TargetPath, file.TaskOperationSystemCopy)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.CopyFileSystemResponse{Task: task}, nil
}

func (f *FileUsecase) CutFileSystem(
	ctx context.Context,
	userID string,
	req *v1.CutFileSystemRequest,
) (*v1.CutFileSystemResponse, error) {
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}
	task, err := fileOper.StartMoveToDirTask(userID, req.Paths, req.TargetPath, file.TaskOperationSystemCut)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.CutFileSystemResponse{Task: task}, nil
}

func (f *FileUsecase) GetFileTask(ctx context.Context, userID, taskID string) (*v1.FileTaskInfo, error) {
	task, ok := file.GetTask(userID, taskID)
	if !ok {
		return nil, ErrFileTaskNotFound
	}
	return task, nil
}

// OpenFileSystemFile 打开文件并返回内容。
func (f *FileUsecase) OpenFileSystemFile(ctx context.Context, req *v1.OpenFileSystemFileRequest) (*v1.OpenFileSystemFileResponse, error) {
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	content, err := fileOper.LoadFile(req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.OpenFileSystemFileResponse{Info: content}, nil
}

// EditFileSystemFile 覆盖保存文件内容。
func (f *FileUsecase) EditFileSystemFile(ctx context.Context, req *v1.EditFileSystemFileRequest) (*v1.EditFileSystemFileResponse, error) {
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}

	if err = fileOper.SaveFile(req.Path, req.Content); err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.EditFileSystemFileResponse{}, nil
}

func (f *FileUsecase) FileSystemPreSign(ctx context.Context, userID, path string) (string, error) {
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return "", ErrSystem(err)
	}
	realPath, err := fileOper.ResolveRealPath(path)
	if err != nil {
		return "", ErrSystem(err)
	}
	if _, err := os.Stat(realPath); os.IsNotExist(err) {
		return "", ErrFileNotExist
	}
	preInfo := pre.NewFileSignInfo(utils.GenerateRandomString(10), path, userID, 24*time.Hour)
	sign, err := preInfo.Sign()
	if err != nil {
		return "", ErrSign
	}
	urlPath := fmt.Sprintf("%s?sign=%s", PreFileDownload, sign)
	return urlPath, nil
}

func (f *FileUsecase) FileDownload(path string, w http.ResponseWriter, r *http.Request) {
	fs, err := os.Open(path)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	defer fs.Close()
	info, err := fs.Stat()
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	w.Header().Set("Content-Disposition", `attachment; filename="`+info.Name()+`"`)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, info.Name(), info.ModTime(), fs)
}

func (f *FileUsecase) FileSystemPreSignUpload(ctx context.Context, userID string, req *v1.FileSystemPreSignUploadRequest) (*v1.UploadInfo, error) {
	if req.FileSize > math.MaxInt64 {
		return nil, ErrUploadRequestInvalid
	}
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}
	realPath, err := fileOper.ResolveRealPath(req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	upload := file.NewChunkedUpload(req.Hash, realPath, req.FileName, req.FileSize)
	info, err := f.repo.GetOrCreate(ctx, userID, upload)
	if err != nil {
		return nil, ErrSystem(err)
	}
	// 签名此次上传
	preInfo := pre.NewFileSignInfo(info.ID, info.Path, userID, UploadPeriod)
	sign, err := preInfo.Sign()
	if err != nil {
		return nil, ErrSign
	}
	upload.FileUpload = info
	uploadCache.Set(info.ID, upload)

	return toUploadInfo(info, sign), nil
}

// getChunkedUpload 获取指定上传会话。缓存未命中（典型如服务重启后内存丢失）时
// 从数据库重建会话，使断点续传不再依赖进程内缓存——这是旧实现 nil 解引用崩溃的根因。
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

func (f *FileUsecase) GetFileUploadStatus(ctx context.Context, userID, uploadID string) (*v1.UploadInfo, error) {
	info, err := f.repo.QueryByUserID(ctx, userID, uploadID)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return toUploadInfo(info, ""), nil
}

func (f *FileUsecase) CompleteFileUpload(ctx context.Context, userID, uploadID string) error {
	info, err := f.getChunkedUpload(ctx, uploadID, userID)
	if err != nil {
		return ErrSystem(err)
	}
	err = f.repo.WithTx(ctx, func(tx *gen.Tx) error {
		info.FileUpload, err = tx.FileUpload.Query().
			Where(
				fileupload.IDEQ(uploadID),
				fileupload.UserIDEQ(userID),
			).WithChunks().Only(ctx)
		if err != nil {
			return err
		}
		_, err = tx.FileUpload.UpdateOneID(uploadID).
			SetCompleted(true).Save(ctx)
		if err != nil {
			return err
		}
		err = info.Complete()
		if err != nil {
			return err
		}
		info.Completed = true

		return nil
	})
	if err != nil {
		return ErrSystem(err)
	}
	return nil
}

func (f *FileUsecase) CancelFileUpload(ctx context.Context, userID, uploadID string) error {
	info, err := f.getChunkedUpload(ctx, uploadID, userID)
	if err != nil {
		return ErrSystem(err)
	}
	err = f.repo.WithTx(ctx, func(tx *gen.Tx) error {
		_, err = tx.FileUpload.UpdateOneID(uploadID).
			SetCancel(true).Save(ctx)
		if err != nil {
			return err
		}
		info.Cancel = true
		err = info.Canceld()
		if err != nil {
			return err
		}
		uploadCache.Del(uploadID)
		return nil
	})
	if err != nil {
		return ErrSystem(err)
	}
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
	err = f.repo.SaveChunkRecord(ctx, pr.UploadId, chunk, hash, size)
	if err != nil {
		return err
	}
	return nil
}

// ---- 分享 ----
// biz 仅做鉴权 + 仓储编排；令牌生成、提取码/有效期校验、目录浏览与打包下载等逻辑在 pkg/share。

// CreateShare 为指定路径创建一条分享。
func (f *FileUsecase) CreateShare(ctx context.Context, userID string, req *v1.CreateShareRequest) (*v1.ShareInfo, error) {
	fileOper, err := f.newSystemInstance()
	if err != nil {
		return nil, ErrSystem(err)
	}
	realPath, err := fileOper.ResolveRealPath(req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	isDir, baseName, err := share.Prepare(realPath)
	if err != nil {
		return nil, ErrFileNotExist
	}
	name := req.Name
	if name == "" {
		name = baseName
	}
	rec, err := f.repo.CreateShare(ctx, userID, share.GenToken(), name, realPath, isDir, req)
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
// req.Path 非空时改换分享目标（文件/文件夹），token 不变、原链接继续有效。
func (f *FileUsecase) UpdateShare(ctx context.Context, userID string, req *v1.UpdateShareRequest) (*v1.ShareInfo, error) {
	var targetPath string
	var isDir bool
	if req.Path != "" {
		fileOper, err := f.newSystemInstance()
		if err != nil {
			return nil, ErrSystem(err)
		}
		realPath, err := fileOper.ResolveRealPath(req.Path)
		if err != nil {
			return nil, ErrSystem(err)
		}
		d, _, err := share.Prepare(realPath)
		if err != nil {
			return nil, ErrFileNotExist
		}
		targetPath = realPath
		isDir = d
	}
	rec, err := f.repo.UpdateShare(ctx, userID, req, targetPath, isDir)
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

// GetShareMeta 公开：返回分享元信息（不含提取码与真实路径）。
func (f *FileUsecase) GetShareMeta(ctx context.Context, token string) (*v1.GetShareMetaResponse, error) {
	rec, err := f.repo.GetShareByToken(ctx, token)
	if err != nil {
		return nil, ErrShareNotFound
	}
	return share.ToMeta(rec, rec.Edges.User, time.Now()), nil
}

// ListShareDir 公开：浏览文件夹分享的子目录（校验提取码与可用性）。
func (f *FileUsecase) ListShareDir(ctx context.Context, req *v1.ListShareDirRequest) (*v1.ListShareDirResponse, error) {
	rec, err := f.repo.GetShareByToken(ctx, req.Token)
	if err != nil {
		return nil, ErrShareNotFound
	}
	if err := share.Verify(rec, req.Code, time.Now()); err != nil {
		return nil, ErrShareForbidden
	}
	items, sub, err := share.ListDir(rec, req.SubPath)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.ListShareDirResponse{Items: items, SubPath: sub}, nil
}

// ShareDownload 公开：校验后将分享目标写入响应（文件直传 / 文件夹打包 zip），成功计一次下载。
// Range 续传请求不重复计数。
func (f *FileUsecase) ShareDownload(w http.ResponseWriter, r *http.Request) {
	ctx := context.Background()
	q := r.URL.Query()
	rec, err := f.repo.GetShareByToken(ctx, q.Get("token"))
	if err != nil {
		w.WriteHeader(http.StatusNotFound)
		return
	}
	if err := share.Verify(rec, q.Get("code"), time.Now()); err != nil {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte(err.Error()))
		return
	}
	countable := r.Header.Get("Range") == ""
	inline := q.Get("inline") == "1"
	// 预览（inline）不计入下载次数，仅真正下载（attachment 且非续传分段）计数。
	if inline {
		countable = false
	}
	if err := share.ServeDownload(rec, q.Get("path"), inline, w, r); err != nil {
		return
	}
	if countable {
		_ = f.repo.IncrShareDownload(ctx, rec.ID)
	}
}

func toUploadInfo(d *gen.FileUpload, sign string) *v1.UploadInfo {
	info := &v1.UploadInfo{
		UploadId:                  d.ID,
		UploadPartUrlPathTemplate: fmt.Sprintf("%s?sign=%s&chunk={partNumber}", PreFileUpload, sign),
		PartSize:                  d.ChunkSize,
		FileSize:                  d.FileSize,
		TotalParts:                d.TotalChunks,
		UploadedParts:             make(map[uint64]string),
		Completed:                 d.Completed,
		Cancel:                    d.Cancel,
		ExpiredAt:                 timestamppb.New(d.CreateTime.Add(UploadPeriod)),
	}
	if chunks := d.Edges.Chunks; chunks != nil {
		for _, chunk := range chunks {
			info.UploadedParts[chunk.Chunk] = chunk.Hash
		}
	}
	return info
}
