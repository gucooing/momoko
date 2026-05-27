package service

import (
	"context"
	stdhttp "net/http"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/pre"
	"momoko/pkg/response"
)

type FileService struct {
	v1.UnimplementedFileManagerServer

	uc *biz.FileUsecase
}

func (f *FileService) RegisterDownloadServer(srv *khttp.Server) {
	srv.HandleFunc(biz.PreFileDownload, f.preFileDownload)
	srv.HandleFunc(biz.PreFileUpload, f.preFileUpload)
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

func (f *FileService) BatchCompressFileSystem(ctx context.Context, req *v1.BatchCompressFileSystemRequest) (*v1.BatchCompressFileSystemResponse, error) {
	return f.uc.BatchCompressFileSystem(ctx, req)
}

func (f *FileService) UnzipFileSystem(ctx context.Context, req *v1.UnzipFileSystemRequest) (*v1.UnzipFileSystemResponse, error) {
	return f.uc.UnzipFileSystem(ctx, req)
}

func (f *FileService) CreateFileSystem(ctx context.Context, req *v1.CreateFileSystemRequest) (*v1.CreateFileSystemResponse, error) {
	return f.uc.CreateFileSystem(ctx, req)
}

func (f *FileService) RenameFileSystem(ctx context.Context, req *v1.RenameFileSystemRequest) (*v1.RenameFileSystemResponse, error) {
	return f.uc.RenameFileSystem(ctx, req)
}

func (f *FileService) CopyFileSystem(ctx context.Context, req *v1.CopyFileSystemRequest) (*v1.CopyFileSystemResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return f.uc.CopyFileSystem(ctx, authCtx.UserID, req)
}

func (f *FileService) CutFileSystem(ctx context.Context, req *v1.CutFileSystemRequest) (*v1.CutFileSystemResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	return f.uc.CutFileSystem(ctx, authCtx.UserID, req)
}

func (f *FileService) GetFileTask(ctx context.Context, req *v1.GetFileTaskRequest) (*v1.GetFileTaskResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	task, err := f.uc.GetFileTask(ctx, authCtx.UserID, req.TaskId)
	if err != nil {
		return nil, err
	}
	return &v1.GetFileTaskResponse{Task: task}, nil
}

func (f *FileService) OpenFileSystemFile(ctx context.Context, req *v1.OpenFileSystemFileRequest) (*v1.OpenFileSystemFileResponse, error) {
	return f.uc.OpenFileSystemFile(ctx, req)
}

func (f *FileService) EditFileSystemFile(ctx context.Context, req *v1.EditFileSystemFileRequest) (*v1.EditFileSystemFileResponse, error) {
	return f.uc.EditFileSystemFile(ctx, req)
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

func (f *FileService) FileSystemPreSignUpload(ctx context.Context, req *v1.FileSystemPreSignUploadRequest) (*v1.FileSystemPreSignUploadResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}

	info, err := f.uc.FileSystemPreSignUpload(ctx, authCtx.UserID, req)
	if err != nil {
		return nil, err
	}

	return &v1.FileSystemPreSignUploadResponse{Info: info}, nil
}

func (f *FileService) GetFileUploadStatus(ctx context.Context, req *v1.GetFileUploadStatusRequest) (*v1.GetFileUploadStatusResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}

	info, err := f.uc.GetFileUploadStatus(ctx, authCtx.UserID, req.UploadId)
	if err != nil {
		return nil, err
	}

	return &v1.GetFileUploadStatusResponse{Info: info}, nil
}

func (f *FileService) CompleteFileUpload(ctx context.Context, req *v1.CompleteFileUploadRequest) (*v1.CompleteFileUploadResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	err := f.uc.CompleteFileUpload(ctx, authCtx.UserID, req.UploadId)
	if err != nil {
		return nil, err
	}
	return &v1.CompleteFileUploadResponse{}, nil
}

func (f *FileService) CancelFileUpload(ctx context.Context, req *v1.CancelFileUploadRequest) (*v1.CancelFileUploadResponse, error) {
	authCtx, ok := auth.FromContext(ctx)
	if !ok {
		return nil, biz.ErrTokenInvalid
	}
	err := f.uc.CancelFileUpload(ctx, authCtx.UserID, req.UploadId)
	if err != nil {
		return nil, err
	}
	return &v1.CancelFileUploadResponse{}, nil
}

func (f *FileService) preFileDownload(w khttp.ResponseWriter, r *khttp.Request) {
	info, err := pre.Verify(r.URL.Query().Get("sign"))
	if err != nil {
		w.WriteHeader(stdhttp.StatusBadRequest)
		return
	}
	f.uc.FileDownload(info.Path, w, r)
}

func (f *FileService) preFileUpload(w khttp.ResponseWriter, r *khttp.Request) {
	info, err := pre.Verify(r.URL.Query().Get("sign"))
	if err != nil {
		response.WriteError(w, r, err)
		return
	}

	err = f.uc.PreFileUpload(w, r, info)
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	response.Write(w, r, []byte("ok"))
}
