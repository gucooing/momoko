package biz

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/file"
	"momoko/pkg/pre"
	"momoko/pkg/utils"
)

type InstanceFileUsecase struct {
	instanceRepo InstanceRepo
	fileUC       *FileUsecase
}

func NewInstanceFileUsecase(instanceRepo InstanceRepo, fileUC *FileUsecase) *InstanceFileUsecase {
	return &InstanceFileUsecase{
		instanceRepo: instanceRepo,
		fileUC:       fileUC,
	}
}

func (u *InstanceFileUsecase) newFileOper(ctx context.Context, userID, instanceID string) (*file.FileOper, error) {
	item, err := u.instanceRepo.GetInstanceByUserID(ctx, userID, instanceID)
	if err != nil {
		if gen.IsNotFound(err) {
			return nil, ErrInstanceAccess
		}
		return nil, ErrSystem(err)
	}

	fileOper, err := file.NewFileOper(item.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return fileOper, nil
}

func (u *InstanceFileUsecase) GetFileList(ctx context.Context, userID string, req *v1.GetInstanceFileListRequest) (*v1.GetInstanceFileListResponse, error) {
	fileOper, err := u.newFileOper(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 10
	}
	if req.PageSize > 100 {
		req.PageSize = 100
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

	total := int64(len(result))
	start := (req.Page - 1) * req.PageSize
	end := req.Page * req.PageSize
	var pages []*v1.FileEntryInfo
	if start >= 0 && start < total {
		if end > total {
			end = total
		}
		pages = result[start:end]
	}

	directory, err := fileOper.DirInfo(req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}

	return &v1.GetInstanceFileListResponse{
		Directory: directory,
		Items:     pages,
		Page:      req.Page,
		PageSize:  req.PageSize,
		Total:     total,
	}, nil
}

func (u *InstanceFileUsecase) CreateFile(ctx context.Context, userID string, req *v1.CreateInstanceFileRequest) (*v1.CreateInstanceFileResponse, error) {
	fileOper, err := u.newFileOper(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	if req.Info == nil {
		return nil, ErrSystem(errors.New("创建信息为空"))
	}
	if err = fileOper.Create(req.Info); err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.CreateInstanceFileResponse{}, nil
}

func (u *InstanceFileUsecase) BatchDeleteFile(ctx context.Context, userID string, req *v1.BatchDeleteInstanceFileRequest) (*v1.BatchDeleteInstanceFileResponse, error) {
	fileOper, err := u.newFileOper(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	items := fileOper.BatchDelete(req.Paths)
	return &v1.BatchDeleteInstanceFileResponse{Items: items}, nil
}

func (u *InstanceFileUsecase) BatchCompressFile(ctx context.Context, userID string, req *v1.BatchCompressInstanceFileRequest) (*v1.BatchCompressInstanceFileResponse, error) {
	fileOper, err := u.newFileOper(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	outputPath, err := fileOper.BatchCompress(req.Paths, *req.TargetPath)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.BatchCompressInstanceFileResponse{OutputPath: outputPath}, nil
}

func (u *InstanceFileUsecase) UnzipFile(ctx context.Context, userID string, req *v1.UnzipInstanceFileRequest) (*v1.UnzipInstanceFileResponse, error) {
	fileOper, err := u.newFileOper(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	outputPath, err := fileOper.Unzip(req.Path, *req.TargetPath)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.UnzipInstanceFileResponse{OutputPath: outputPath}, nil
}

func (u *InstanceFileUsecase) OpenFile(ctx context.Context, userID string, req *v1.OpenInstanceFileRequest) (*v1.OpenInstanceFileResponse, error) {
	fileOper, err := u.newFileOper(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	content, err := fileOper.LoadFile(req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	return &v1.OpenInstanceFileResponse{Info: content}, nil
}

func (u *InstanceFileUsecase) FilePreSign(ctx context.Context, userID string, req *v1.InstanceFilePreSignRequest) (*v1.InstanceFilePreSignResponse, error) {
	fileOper, err := u.newFileOper(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	realPath, err := fileOper.ResolveRealPath(req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if info, err := os.Stat(realPath); err != nil || info.IsDir() {
		return nil, ErrFileNotExist
	}
	preInfo := pre.NewFileSignInfo(utils.GenerateRandomString(10), realPath, userID, 24*time.Hour)
	sign, err := preInfo.Sign()
	if err != nil {
		return nil, ErrSign
	}
	return &v1.InstanceFilePreSignResponse{DownloadUrlPath: fmt.Sprintf("%s?sign=%s", PreFileDownload, sign)}, nil
}

func (u *InstanceFileUsecase) FilePreSignUpload(ctx context.Context, userID string, req *v1.InstanceFilePreSignUploadRequest) (*v1.UploadInfo, error) {
	if req.FileSize > math.MaxInt64 {
		return nil, ErrUploadRequestInvalid
	}
	fileOper, err := u.newFileOper(ctx, userID, req.Id)
	if err != nil {
		return nil, err
	}
	realPath, err := fileOper.ResolveRealPath(req.Path)
	if err != nil {
		return nil, ErrSystem(err)
	}
	upload := file.NewChunkedUpload(req.Hash, realPath, req.FileName, req.FileSize)
	info, err := u.fileUC.repo.GetOrCreate(ctx, userID, upload)
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
	u.fileUC.uploadCache.Set(info.ID, upload)

	return toUploadInfo(info, sign), nil
}
