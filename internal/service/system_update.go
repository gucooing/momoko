package service

import (
	"context"
	nethttp "net/http"
	"path"

	khttp "github.com/go-kratos/kratos/v2/transport/http"

	v1 "momoko/api/gen/v1"
	"momoko/internal/biz"
	"momoko/pkg/auth"
	"momoko/pkg/response"
)

// RegisterUpdateRoutes 上传程序文件更新：请求体就是文件本身（原始 HTTP 流，几十上百兆，不走 JSON），
// 文件名放在 ?name= 里。权限在 server 的原始路由鉴权里按路径校验（system:update）。
func (s *SystemService) RegisterUpdateRoutes(srv *khttp.Server) {
	srv.HandleFunc(biz.SystemUpdateUploadPath, s.uploadUpdate)
}

func (s *SystemService) uploadUpdate(w nethttp.ResponseWriter, r *nethttp.Request) {
	if r.Method != nethttp.MethodPost {
		w.WriteHeader(nethttp.StatusMethodNotAllowed)
		return
	}
	defer r.Body.Close()
	if _, ok := auth.FromContext(r.Context()); !ok {
		response.WriteError(w, r, biz.ErrTokenInvalid)
		return
	}
	// 接收文件可能比接口超时还久，校验和业务记录都不跟着请求的超时走
	ctx := context.WithoutCancel(r.Context())
	st, err := s.uc.UploadUpdate(ctx, r.Body, r.ContentLength, path.Base(r.URL.Query().Get("name")))
	if err != nil {
		response.WriteError(w, r, err)
		return
	}
	_ = response.ResponseEncoder(w, r, &v1.UpdateStatusResponse{Status: st})
}

// UpdateStatus 在线更新进度（需 system:update 权限）。
func (s *SystemService) UpdateStatus(ctx context.Context, _ *v1.UpdateStatusRequest) (*v1.UpdateStatusResponse, error) {
	return &v1.UpdateStatusResponse{Status: s.uc.UpdateStatus(ctx)}, nil
}

// StartUpdate 从 GitHub 下载最新版本并更新（需 system:update 权限）。
func (s *SystemService) StartUpdate(ctx context.Context, _ *v1.StartUpdateRequest) (*v1.StartUpdateResponse, error) {
	st, err := s.uc.StartUpdate(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.StartUpdateResponse{Status: st}, nil
}

// ApplyUpdate 安装已上传的程序并重启（需 system:update 权限）。
func (s *SystemService) ApplyUpdate(ctx context.Context, _ *v1.ApplyUpdateRequest) (*v1.ApplyUpdateResponse, error) {
	st, err := s.uc.ApplyUpdate(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.ApplyUpdateResponse{Status: st}, nil
}

// CancelUpdate 取消更新（需 system:update 权限）。
func (s *SystemService) CancelUpdate(ctx context.Context, _ *v1.CancelUpdateRequest) (*v1.CancelUpdateResponse, error) {
	st, err := s.uc.CancelUpdate(ctx)
	if err != nil {
		return nil, err
	}
	return &v1.CancelUpdateResponse{Status: st}, nil
}

// ListDownloads 下载管理：只列出当前用户有权限的下载。
func (s *SystemService) ListDownloads(ctx context.Context, _ *v1.ListDownloadsRequest) (*v1.ListDownloadsResponse, error) {
	return s.uc.ListDownloads(ctx), nil
}

func (s *SystemService) downloadAction(ctx context.Context, id, action string) (*v1.DownloadActionResponse, error) {
	info, err := s.uc.DownloadAction(ctx, id, action)
	if err != nil {
		return nil, err
	}
	return &v1.DownloadActionResponse{Download: info}, nil
}

// PauseDownload 暂停下载。
func (s *SystemService) PauseDownload(ctx context.Context, req *v1.DownloadActionRequest) (*v1.DownloadActionResponse, error) {
	return s.downloadAction(ctx, req.GetId(), "pause")
}

// ResumeDownload 继续下载。
func (s *SystemService) ResumeDownload(ctx context.Context, req *v1.DownloadActionRequest) (*v1.DownloadActionResponse, error) {
	return s.downloadAction(ctx, req.GetId(), "resume")
}

// CancelDownload 取消下载。
func (s *SystemService) CancelDownload(ctx context.Context, req *v1.DownloadActionRequest) (*v1.DownloadActionResponse, error) {
	return s.downloadAction(ctx, req.GetId(), "cancel")
}

// RemoveDownload 删掉一条已结束的下载记录。
func (s *SystemService) RemoveDownload(ctx context.Context, req *v1.DownloadActionRequest) (*v1.RemoveDownloadResponse, error) {
	if err := s.uc.RemoveDownload(ctx, req.GetId()); err != nil {
		return nil, err
	}
	return &v1.RemoveDownloadResponse{}, nil
}

// ClearDownloads 清掉已结束的下载记录。
func (s *SystemService) ClearDownloads(ctx context.Context, _ *v1.ClearDownloadsRequest) (*v1.ClearDownloadsResponse, error) {
	return &v1.ClearDownloadsResponse{Removed: s.uc.ClearDownloads(ctx)}, nil
}
