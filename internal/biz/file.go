package biz

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/file"
	"momoko/pkg/pre"
)

const (
	PreFileDownload = "/api/v1/download/pre"
)

type FileUsecase struct{}

// NewFileUsecase 创建文件操作用例。
func NewFileUsecase() *FileUsecase {
	if _, err := os.Stat(file.ServersPath); os.IsNotExist(err) {
		os.MkdirAll(file.ServersPath, 0755)
	}
	return &FileUsecase{}
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
	preInfo := pre.NewFileDownloadInfo(path, 24*time.Hour, userID)
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
	http.ServeContent(w, r, info.Name(), info.ModTime(), fs)
}
