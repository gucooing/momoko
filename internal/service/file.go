package service

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
)

type FileService struct {
	v1.UnimplementedFileSystemManagerServer

	uc *biz.FileUsecase
}

func NewFileService(uc *biz.FileUsecase) *FileService {
	return &FileService{uc: uc}
}

func (f *FileService) GetFileSystemList(ctx context.Context, req *v1.GetFileSystemListRequest) (*v1.GetFileSystemListResponse, error) {
	return f.uc.GetFileSystemList(ctx, req)
}

func (f *FileService) BatchDeleteFileSystem(ctx context.Context, req *v1.BatchDeleteFileSystemRequest) (*v1.BatchDeleteFileSystemResponse, error) {
	return f.uc.BatchDeleteFileSystem(ctx, req)
}

func (f *FileService) CreateFileSystem(ctx context.Context, req *v1.CreateFileSystemRequest) (*v1.CreateFileSystemResponse, error) {
	return f.uc.CreateFileSystem(ctx, req)
}

func (f *FileService) OpenFileSystemFile(ctx context.Context, req *v1.OpenFileSystemFileRequest) (*v1.OpenFileSystemFileResponse, error) {
	return f.uc.OpenFileSystemFile(ctx, req)
}
