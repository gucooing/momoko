package service

import (
	"context"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
)

type FileService struct {
	v1.UnimplementedFileSystemManagerServer

	uc *biz.FileUsecase
}

func NewFileService(uc *biz.FileUsecase) *FileService {
	return &FileService{uc: uc}
}

func (f *FileService) GetFileSystemList(ctx context.Context, req *v1.GetFileSystemListRequest) (*v1.GetFileSystemListResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, biz.ErrTokenInvalid
	}
	return f.uc.GetFileSystemList(ctx, req)
}

func (f *FileService) BatchCalcFileSystemSize(ctx context.Context, req *v1.BatchCalcFileSystemSizeRequest) (*v1.BatchCalcFileSystemSizeResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, biz.ErrTokenInvalid
	}
	return f.uc.BatchCalcFileSystemSize(ctx, req)
}

func (f *FileService) BatchDeleteFileSystem(ctx context.Context, req *v1.BatchDeleteFileSystemRequest) (*v1.BatchDeleteFileSystemResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, biz.ErrTokenInvalid
	}
	return f.uc.BatchDeleteFileSystem(ctx, req)
}

func (f *FileService) BatchCreateFileSystem(ctx context.Context, req *v1.BatchCreateFileSystemRequest) (*v1.BatchCreateFileSystemResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, biz.ErrTokenInvalid
	}
	return f.uc.BatchCreateFileSystem(ctx, req)
}

func (f *FileService) OpenFileSystemFile(ctx context.Context, req *v1.OpenFileSystemFileRequest) (*v1.OpenFileSystemFileResponse, error) {
	if _, ok := auth.FromContext(ctx); !ok {
		return nil, biz.ErrTokenInvalid
	}
	return f.uc.OpenFileSystemFile(ctx, req)
}
