package biz

import (
	"context"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"time"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/agentfs"
	agentskillpkg "momoko/pkg/agentskill"
	"momoko/pkg/file"
	"momoko/pkg/pre"
	"momoko/pkg/response"
	"momoko/pkg/utils"
)

// Agent 页面上的文件进出都走 Momoko 标准的文件接口，不经 JSON 传内容：
//   - 取：SignFiles 给签名下载链接（/api/v1/download/pre），浏览器直接拿原样字节，图片用 <img src> 显示；
//   - 传：PreSignUpload 走标准的分片上传，先传进这个人自己的暂存目录，发消息、导入技能时按 upload_id 取走（takeUpload）。

const (
	// agentFileTTL 签名下载链接能用多久。
	agentFileTTL = 12 * time.Hour
	// agentStagingTTL 传上来一直没用掉的文件留多久。
	agentStagingTTL = 24 * time.Hour
)

// uploadHashPattern 上传的文件指纹：前端算的，拿来当暂存目录的名字。
var uploadHashPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{8,128}$`)

// stagingDir 一个人传上来、还没用掉的文件：按指纹分目录，同一个文件续传、再传一次直接用。
func (uc *AgentUsecase) stagingDir(userID string) string {
	return uc.userDir(userID, "uploads")
}

// localPath 页面给的路径换成本机的绝对路径：绝对路径原样（file:// 地址在 Windows 上的路径部分是 /C:/…，去掉开头的 /），
// 相对路径接在 base 后面，不能跳出 base。认不出为空。
func localPath(base, p string) string {
	p = strings.TrimSpace(p)
	if runtime.GOOS == "windows" && len(p) >= 3 && (p[0] == '/' || p[0] == '\\') && p[2] == ':' {
		p = p[1:]
	}
	if filepath.IsAbs(p) {
		return filepath.Clean(p)
	}
	full, err := agentfs.SafeJoin(base, p)
	if err != nil {
		return ""
	}
	return full
}

// within full 在不在 roots 里的某一个目录下面。
func within(roots []string, full string) bool {
	for _, root := range roots {
		rel, err := filepath.Rel(root, full)
		if err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// SignFiles 会话（新会话页还没有会话时是工作区）里文件的签名下载链接。路径相对工作目录，或者是绝对路径；
// uploads 时相对会话的附件目录。只给会话目录、工作目录、附加目录（工作区时是工作区目录）里的文件，
// 不在这些目录里、不是文件的那一项为空。
func (uc *AgentUsecase) SignFiles(ctx context.Context, req *v1.SignAgentFilesRequest) ([]string, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	var base string
	var roots []string
	switch {
	case req.GetSessionId() != "":
		s, err := uc.getSession(ctx, req.GetSessionId())
		if err != nil {
			return nil, err
		}
		base = s.Cwd
		if req.GetUploads() {
			base = uc.sessionDir(s.ID, "uploads")
		}
		roots = append([]string{uc.sessionDir(s.ID), s.Cwd}, s.ExtraDirs...)
	case req.GetWorkspaceId() != "":
		w, err := uc.getWorkspace(ctx, req.GetWorkspaceId())
		if err != nil {
			return nil, err
		}
		base, roots = w.Path, []string{w.Path}
	default:
		return nil, response.BadRequest("缺少会话或工作区")
	}
	urls := make([]string, len(req.GetPaths()))
	for i, p := range req.GetPaths() {
		full := localPath(base, p)
		if full == "" || !within(roots, full) || !agentfs.IsFile(full) {
			continue
		}
		info := pre.NewFileSignInfo(utils.GenerateRandomString(10), full, userID, agentFileTTL)
		info.Inline = true
		sign, err := info.Sign()
		if err != nil {
			return nil, ErrSign
		}
		urls[i] = PreFileDownload + "?sign=" + sign
	}
	return urls, nil
}

// PreSignUpload 上传一个文件（消息附件、技能压缩包）：标准的分片上传，目标是自己暂存目录里的 <指纹>/<文件名>。
// 同一个文件传完了的直接返回已完成。
func (uc *AgentUsecase) PreSignUpload(ctx context.Context, req *v1.PreSignAgentUploadRequest) (*v1.UploadInfo, error) {
	userID, err := agentUserID(ctx)
	if err != nil {
		return nil, err
	}
	name := path.Base(filepath.ToSlash(strings.TrimSpace(req.GetFileName())))
	if name == "" || name == "." || name == "/" {
		return nil, response.BadRequest("文件名不对")
	}
	if req.GetFileSize() <= 0 {
		return nil, response.BadRequest("文件 " + name + " 是空的")
	}
	if req.GetFileSize() > agentskillpkg.MaxArchive {
		return nil, response.BadRequest("文件 " + name + " 太大了，最多 64 MB")
	}
	if !uploadHashPattern.MatchString(req.GetHash()) {
		return nil, response.BadRequest("文件指纹不对")
	}
	dir := uc.stagingDir(userID)
	sweepStaging(dir)
	store, err := file.NewLocalStore(dir)
	if err != nil {
		return nil, ErrSystem(err)
	}
	seed := file.NewChunkedUpload(req.GetHash(), filepath.Join(dir, req.GetHash(), name), name, uint64(req.GetFileSize()), true)
	row, err := uc.uploads.GetOrCreate(ctx, userID, seed)
	if err != nil {
		return nil, ErrSystem(err)
	}
	if row.Completed {
		if agentfs.IsFile(row.Path) {
			return toUploadInfo(row, &file.Upload{}), nil
		}
		// 传完的文件已经被取走或清掉了：重新传
		if err := uc.uploads.DeleteUploadCascade(ctx, row.ID); err != nil {
			return nil, ErrSystem(err)
		}
		if row, err = uc.uploads.GetOrCreate(ctx, userID, seed); err != nil {
			return nil, ErrSystem(err)
		}
	}
	u, err := buildUpload(uc.uploads, userID, row)
	if err != nil {
		return nil, err
	}
	if err := store.PrepareUpload(ctx, u); err != nil {
		return nil, ErrSystem(err)
	}
	uploadCache.Set(row.ID, &file.ChunkedUpload{FileUpload: row})
	return toUploadInfo(row, u), nil
}

// takeUpload 取走一个传完的文件：挪进 dir（放进一个新建的子目录，同名的不冲突），上传记录删掉。返回挪过去的路径。
// max 是这个用途允许的大小。
func (uc *AgentUsecase) takeUpload(ctx context.Context, userID, uploadID, dir string, max int64) (string, error) {
	row, err := uc.uploads.QueryByUserID(ctx, userID, uploadID)
	if err != nil || !row.Completed {
		return "", response.BadRequest("文件还没传完")
	}
	if int64(row.FileSize) > max {
		return "", response.BadRequest("文件 " + row.FileName + " 太大了")
	}
	staging := uc.stagingDir(userID)
	if !within([]string{staging}, row.Path) || !agentfs.IsFile(row.Path) {
		return "", response.BadRequest("文件 " + row.FileName + " 不在了，重新传一次")
	}
	full, err := agentfs.Adopt(dir, row.Path)
	if err != nil {
		return "", response.BadRequest("保存文件 " + row.FileName + " 失败：" + err.Error())
	}
	_ = os.Remove(filepath.Dir(row.Path))
	if err := uc.uploads.DeleteUploadCascade(ctx, row.ID); err != nil {
		return "", ErrSystem(err)
	}
	return full, nil
}

// sweepStaging 清掉暂存目录里放了很久没用掉的文件（传了又没发出去的附件）。
func sweepStaging(dir string) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	cutoff := time.Now().Add(-agentStagingTTL)
	for _, e := range entries {
		if info, err := e.Info(); err == nil && info.ModTime().Before(cutoff) {
			agentfs.RemoveAll(filepath.Join(dir, e.Name()))
		}
	}
}
