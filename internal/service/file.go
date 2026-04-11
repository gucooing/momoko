package service

import (
	"context"

	"github.com/go-kratos/kratos/v2/transport/http"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/pre"
)

type FileService struct {
	v1.UnimplementedFileSystemManagerServer

	uc *biz.FileUsecase
}

func (f *FileService) RegisterDownloadServer(srv *http.Server) {
	srv.HandleFunc(biz.PreFileDownload, f.PreFileDownload)
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

func (f *FileService) FileSystemPreSign(ctx context.Context, req *v1.FileSystemPreSignRequest) (*v1.FileSystemPreSignResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	url, err := f.uc.FileSystemPreSign(ctx, authCtx.UserID, req.Path)
	if err != nil {
		return nil, err
	}

	return &v1.FileSystemPreSignResponse{DownloadUrlPath: url}, nil
}

func (f *FileService) PreFileDownload(w http.ResponseWriter, r *http.Request) {
	info, err := pre.Verify(r.URL.Query().Get("sign"))
	if err != nil {
		w.WriteHeader(400)
		return
	}
	f.uc.FileDownload(info.Path, w, r)
}
