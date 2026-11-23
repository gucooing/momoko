package biz

import (
	"context"
	"errors"
	"io"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/constant"
	"momoko/pkg/download"
	"momoko/pkg/response"
	"momoko/pkg/selfupdate"
	"momoko/pkg/version"
)

// UpdateStatus 在线更新的当前状态。
func (s *SystemUsecase) UpdateStatus(_ context.Context) *v1.UpdateStatus {
	return toUpdateStatus(s.updater.Status())
}

// StartUpdate 从 GitHub 更新到当前更新通道的最新版本：后台下载、校验、替换程序并重启。
func (s *SystemUsecase) StartUpdate(ctx context.Context) (*v1.UpdateStatus, error) {
	return s.startUpdate(ctx, s.operatorName(ctx))
}

func (s *SystemUsecase) startUpdate(ctx context.Context, operator string) (*v1.UpdateStatus, error) {
	channel := v1.UpdateChannel_UpdateChannel_Stable
	if site, err := s.config.SiteConfig(ctx); err == nil {
		channel = site.UpdateChannel
	}
	rel, err := version.CheckLatestChannel(ctx, channel == v1.UpdateChannel_UpdateChannel_Beta)
	if err != nil {
		return nil, response.BadRequest(err.Error())
	}
	if !version.HasUpdate(rel.TagName) {
		if version.Current() == "dev" {
			return nil, response.BadRequest("开发版本不能从 GitHub 在线更新，请上传程序文件更新")
		}
		return nil, response.BadRequest("已是最新版本（" + version.Current() + "）")
	}
	if err := s.updater.StartRelease(ctx, rel, operator); err != nil {
		return nil, response.BadRequest(err.Error())
	}
	return toUpdateStatus(s.updater.Status()), nil
}

// UploadUpdate 接收上传的程序文件（或发行版的 .zip / .tar.gz），校验通过后等确认安装。
func (s *SystemUsecase) UploadUpdate(ctx context.Context, body io.Reader, size int64, fileName string) (*v1.UpdateStatus, error) {
	st, err := s.updater.Upload(body, size, fileName, s.operatorName(ctx))
	if err != nil {
		return nil, response.BadRequest(err.Error())
	}
	return toUpdateStatus(st), nil
}

// ApplyUpdate 安装已上传的程序并重启。
func (s *SystemUsecase) ApplyUpdate(ctx context.Context) (*v1.UpdateStatus, error) {
	st, err := s.updater.Apply(s.operatorName(ctx))
	if err != nil {
		return nil, response.BadRequest(err.Error())
	}
	return toUpdateStatus(st), nil
}

// CancelUpdate 取消下载 / 丢掉已上传的文件 / 清掉失败状态。
func (s *SystemUsecase) CancelUpdate(_ context.Context) (*v1.UpdateStatus, error) {
	st, err := s.updater.Cancel()
	if err != nil {
		return nil, response.BadRequest(err.Error())
	}
	return toUpdateStatus(st), nil
}

func toUpdateStatus(st selfupdate.Status) *v1.UpdateStatus {
	out := &v1.UpdateStatus{
		State:             v1.UpdateState(st.State),
		Source:            v1.UpdateSource(st.Source),
		TargetVersion:     st.TargetVersion,
		Downloaded:        st.Downloaded,
		Total:             st.Total,
		Error:             st.Error,
		Operator:          st.Operator,
		CurrentVersion:    version.Current(),
		Platform:          selfupdate.Platform(),
		Executable:        st.Executable,
		Supported:         st.Supported,
		UnsupportedReason: st.UnsupportedReason,
		BootId:            selfupdate.BootID,
		Backup:            st.Backup,
	}
	if !st.StartedAt.IsZero() {
		out.StartedAt = timestamppb.New(st.StartedAt)
	}
	if p := st.Package; p != nil {
		pkg := &v1.UpdatePackageInfo{
			Version:  p.Version,
			Goos:     p.GOOS,
			Goarch:   p.GOARCH,
			Revision: p.Revision,
			Size:     p.Size,
			Sha256:   p.SHA256,
			FileName: p.FileName,
		}
		if !p.CommitTime.IsZero() {
			pkg.CommitTime = timestamppb.New(p.CommitTime)
		}
		if cmp, ok := version.Compare(p.Version, version.Current()); ok {
			pkg.Downgrade = cmp < 0
			pkg.SameVersion = cmp == 0
		}
		out.Package = pkg
	}
	if st.Download != nil {
		out.Download = toDownloadInfo(*st.Download)
	}
	return out
}

// ———— 下载管理 ————

// canSeeDownload 当前用户能不能看、能不能操作要 perm 权限的下载：远程主控转发来的还要 API Key 授予了这个权限。
func (s *SystemUsecase) canSeeDownload(ctx context.Context, perm string) bool {
	if perm == "" {
		return true
	}
	p := constant.Permissions(perm)
	if np, ok := NodePrincipalFromContext(ctx); ok && np != nil && !np.Granted(p) {
		return false
	}
	return s.Check(ctx, p) == nil
}

// ListDownloads 列出当前用户有权限的下载，新的在前。
func (s *SystemUsecase) ListDownloads(ctx context.Context) *v1.ListDownloadsResponse {
	resp := &v1.ListDownloadsResponse{Downloads: []*v1.DownloadInfo{}}
	allowed := map[string]bool{}
	for _, info := range download.Default.List() {
		ok, seen := allowed[info.Perm]
		if !seen {
			ok = s.canSeeDownload(ctx, info.Perm)
			allowed[info.Perm] = ok
		}
		if !ok {
			continue
		}
		resp.Downloads = append(resp.Downloads, toDownloadInfo(info))
		if !info.State.Terminal() {
			resp.Active++
		}
	}
	return resp
}

// DownloadAction 暂停 / 继续 / 取消一个下载。
func (s *SystemUsecase) DownloadAction(ctx context.Context, id, action string) (*v1.DownloadInfo, error) {
	d, err := s.ownedDownload(ctx, id)
	if err != nil {
		return nil, err
	}
	switch action {
	case "pause":
		err = d.Pause()
	case "resume":
		err = d.Resume()
	case "cancel":
		err = d.Cancel()
	default:
		err = errors.New("不认识的操作")
	}
	if err != nil {
		return nil, response.BadRequest(err.Error())
	}
	return toDownloadInfo(d.Info()), nil
}

// RemoveDownload 从列表里删掉一条已结束的下载记录。
func (s *SystemUsecase) RemoveDownload(ctx context.Context, id string) error {
	if _, err := s.ownedDownload(ctx, id); err != nil {
		return err
	}
	if err := download.Default.Remove(id); err != nil {
		return response.BadRequest(err.Error())
	}
	return nil
}

// ClearDownloads 清掉当前用户有权限的、已结束的下载记录。
func (s *SystemUsecase) ClearDownloads(ctx context.Context) int32 {
	allowed := map[string]bool{}
	n := download.Default.RemoveFinished(func(info download.Info) bool {
		ok, seen := allowed[info.Perm]
		if !seen {
			ok = s.canSeeDownload(ctx, info.Perm)
			allowed[info.Perm] = ok
		}
		return ok
	})
	return int32(n)
}

// ownedDownload 取一个当前用户有权限操作的下载（没权限的当作不存在，不暴露它的存在）。
func (s *SystemUsecase) ownedDownload(ctx context.Context, id string) (*download.Download, error) {
	d, ok := download.Default.Get(id)
	if !ok || !s.canSeeDownload(ctx, d.Info().Perm) {
		return nil, response.BadRequest(download.ErrNotFound.Error())
	}
	return d, nil
}

var downloadStates = map[download.State]v1.DownloadState{
	download.StateDownloading: v1.DownloadState_DownloadState_Downloading,
	download.StatePaused:      v1.DownloadState_DownloadState_Paused,
	download.StateRetrying:    v1.DownloadState_DownloadState_Retrying,
	download.StateCompleted:   v1.DownloadState_DownloadState_Completed,
	download.StateFailed:      v1.DownloadState_DownloadState_Failed,
	download.StateCanceled:    v1.DownloadState_DownloadState_Canceled,
}

func toDownloadInfo(info download.Info) *v1.DownloadInfo {
	out := &v1.DownloadInfo{
		Id:         info.ID,
		Name:       info.Name,
		Title:      info.Title,
		Source:     info.Source,
		Url:        info.URL,
		State:      downloadStates[info.State],
		Downloaded: info.Downloaded,
		Total:      info.Total,
		Speed:      info.Speed,
		Resumable:  info.Resumable,
		Retries:    int32(info.Retries),
		Error:      info.Error,
		Operator:   info.Operator,
		CreatedAt:  timestamppb.New(info.CreatedAt),
	}
	if !info.FinishedAt.IsZero() {
		out.FinishedAt = timestamppb.New(info.FinishedAt)
	}
	return out
}

// SystemUpdateUploadPath 上传程序文件更新的原始 HTTP 路由（请求体是文件本身）。
const SystemUpdateUploadPath = "/api/v1/system/update/upload"
