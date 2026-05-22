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
	"momoko/pkg/utils"
)

const (
	PreFileDownload = "/api/v1/download/pre" // 下载路径
	PreFileUpload   = "/api/v1/upload/pre"   // 上传路径

	UploadPeriod = 2 * time.Hour
)

var uploadCache = cache.New[string, *file.ChunkedUpload](UploadPeriod) // 全局上传缓存

type FileRepo interface {
	GetOrCreate(ctx context.Context, userId string, info *file.ChunkedUpload) (*gen.FileUpload, error)
	WithTx(ctx context.Context, fn func(tx *gen.Tx) error) error
	Query(ctx context.Context, uid string) (*gen.FileUpload, error)
	QueryByUserID(ctx context.Context, userID, id string) (*gen.FileUpload, error)
	SaveChunkRecord(ctx context.Context, uploadID string, chunk uint64, hash string, size uint64) error
}

type FileUsecase struct {
	repo FileRepo
}

// NewFileUsecase 创建文件操作用例。
func NewFileUsecase(repo FileRepo) *FileUsecase {
	if _, err := os.Stat(file.ServersPath); os.IsNotExist(err) {
		os.MkdirAll(file.ServersPath, 0755)
	}
	return &FileUsecase{
		repo: repo,
	}
}

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
	upload.Sing = sign
	uploadCache.Set(info.ID, upload)

	return toUploadInfo(info, sign), nil
}

// 获取指定上传会话
func (f *FileUsecase) getChunkedUpload(ctx context.Context, uploadID, userID string) (*file.ChunkedUpload, error) {
	upload, ok := uploadCache.Get(uploadID)
	if !ok {
		info, err := f.repo.QueryByUserID(ctx, userID, uploadID)
		if err != nil {
			return nil, ErrSystem(err)
		}
		preInfo := pre.NewFileSignInfo(info.ID, info.Path, userID, UploadPeriod)
		sign, err := preInfo.Sign()
		if err != nil {
			return nil, ErrSign
		}
		upload.FileUpload = info
		upload.Sing = sign
		uploadCache.Set(info.ID, upload)
	}
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
