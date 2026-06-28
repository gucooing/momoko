// Package share 实现对外分享的底层逻辑：令牌生成、可用性与提取码校验、目录浏览（防穿越）、
// 以及下载（单文件直传、多选/文件夹实时打包 zip）。
//
// 分享内容抽象为一层虚拟顶层目录：被分享的若干路径（文件或文件夹）即其顶层条目，
// 单选/多选、文件/文件夹由此统一处理。分享统一走 file.Store 接口，支持任意来源
// （本地 / OSS / FTP / WebDAV）。按分层约定，这里只放与分享相关的纯逻辑与 Store 调用；
// biz 仅负责鉴权 + 仓储 + 解析来源 Store。
package share

import (
	"archive/zip"
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"mime"
	"net/http"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/file"
)

// shareServeTTL 是公开分享 302 直链的有效期。
const shareServeTTL = 5 * time.Minute

// GenToken 生成 URL 安全的随机分享令牌。
func GenToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// Prepare 通过来源 Store 校验被分享的每个路径是否存在，并返回默认展示名（name 为空时取首个条目名）。
func Prepare(ctx context.Context, store file.Store, paths []string, name string) (string, error) {
	if len(paths) == 0 {
		return "", errors.New("文件不存在")
	}
	for i, p := range paths {
		entry, err := store.Stat(ctx, p)
		if err != nil {
			return "", errors.New("文件不存在")
		}
		if i == 0 && name == "" {
			name = entry.Name
			if name == "" {
				name = baseName(p)
			}
		}
	}
	return name, nil
}

// Available 报告分享当前是否可用（启用、未过期、未超下载次数），不校验提取码。
func Available(s *gen.FileShare, now time.Time) bool {
	if !s.Enabled {
		return false
	}
	if s.ExpiresAt != nil && now.After(*s.ExpiresAt) {
		return false
	}
	if s.MaxDownloads > 0 && s.DownloadCount >= s.MaxDownloads {
		return false
	}
	return true
}

// Verify 校验一次公开访问：可用性 + 提取码。返回的 error 文本可直接回给访问者。
func Verify(s *gen.FileShare, code string, now time.Time) error {
	if !s.Enabled {
		return errors.New("分享已关闭")
	}
	if s.ExpiresAt != nil && now.After(*s.ExpiresAt) {
		return errors.New("分享已过期")
	}
	if s.MaxDownloads > 0 && s.DownloadCount >= s.MaxDownloads {
		return errors.New("分享下载次数已用尽")
	}
	if s.Code != "" && s.Code != code {
		return errors.New("提取码错误")
	}
	return nil
}

// normalizeSub 把客户端子路径规整为相对、正斜杠、无 .. 的安全相对路径。
func normalizeSub(sub string) string {
	sub = strings.ReplaceAll(sub, "\\", "/")
	cleaned := path.Clean("/" + strings.Trim(sub, "/"))
	return strings.TrimPrefix(cleaned, "/")
}

// baseName 取来源内路径的末段名（兼容正反斜杠、去尾部分隔符），即该路径在虚拟顶层目录中的展示名。
func baseName(p string) string {
	return path.Base(strings.ReplaceAll(strings.TrimRight(p, "/\\"), "\\", "/"))
}

// resolve 将公开端的虚拟子路径映射为来源内真实路径：首段对应被分享的某个顶层条目（按展示名匹配），
// 其余段原样拼到该条目真实路径之后。匹配不到（越界/不存在）返回 ok=false。
// 公开端只见虚拟路径，真实来源路径不出 pkg/share，既防越权穿越也不泄露存储细节。
func resolve(s *gen.FileShare, sub string) (string, bool) {
	sub = normalizeSub(sub)
	if sub == "" {
		return "", false
	}
	first, rest, _ := strings.Cut(sub, "/")
	for _, p := range s.Paths {
		if baseName(p) != first {
			continue
		}
		if rest == "" {
			return p, true
		}
		return strings.TrimRight(p, "/\\") + "/" + rest, true
	}
	return "", false
}

func joinSlash(base, name string) string {
	if base == "" {
		return name
	}
	return base + "/" + name
}

// ListDir 列出分享内某子路径的条目（目录优先、名称升序）。subPath 为空时列出虚拟顶层目录
// （即被分享的各顶层条目）；否则经 resolve 映射到来源真实目录后列出。
// 返回的 RelPath 均为虚拟路径，公开端据此续航，不暴露来源真实路径。
func ListDir(ctx context.Context, store file.Store, s *gen.FileShare, subPath string) ([]*v1.ShareEntry, string, error) {
	sub := normalizeSub(subPath)
	var entries []*v1.FileEntryInfo
	if sub == "" {
		entries = make([]*v1.FileEntryInfo, 0, len(s.Paths))
		for _, p := range s.Paths {
			entry, err := store.Stat(ctx, p)
			if err != nil {
				return nil, "", errors.New("读取目录失败")
			}
			entries = append(entries, entry)
		}
	} else {
		real, ok := resolve(s, sub)
		if !ok {
			return nil, "", errors.New("路径不存在")
		}
		var err error
		entries, err = store.List(ctx, real, file.SortByName, false)
		if err != nil {
			return nil, "", errors.New("读取目录失败")
		}
	}

	items := make([]*v1.ShareEntry, 0, len(entries))
	for _, e := range entries {
		items = append(items, &v1.ShareEntry{
			Name:       e.Name,
			IsDir:      e.IsDir,
			Size:       e.Size,
			UpdateTime: e.UpdateTime,
			// 回传虚拟路径（相对虚拟顶层目录），不暴露来源真实路径。
			RelPath: joinSlash(sub, e.Name),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return items[i].Name < items[j].Name
	})
	return items, sub, nil
}

// ServeDownload 将分享内 subPath 指向的目标写入 w：subPath 为空时打包全部顶层条目为 zip；
// 指向目录时打包该目录；指向文件时直传（可 Seek 来源支持 Range）。inline=true 以内联方式返回（用于预览）。
// redirect302=true 且来源支持预签名（OSS）时，单文件改为 302 跳转到（公开域名）预签名直链，momoko 不在数据路径。
func ServeDownload(ctx context.Context, store file.Store, s *gen.FileShare, subPath string, inline, redirect302 bool, w http.ResponseWriter, r *http.Request) error {
	sub := normalizeSub(subPath)
	if sub == "" {
		return streamZip(ctx, store, s.Paths, s.Name, w)
	}
	real, ok := resolve(s, sub)
	if !ok {
		return errors.New("文件不存在")
	}
	entry, err := store.Stat(ctx, real)
	if err != nil {
		return errors.New("文件不存在")
	}
	// 子路径指向目录：实时打包为 zip。
	if entry.IsDir {
		return streamZip(ctx, store, []string{real}, entry.Name, w)
	}
	// 开启 302 且来源支持预签名（OSS）→ 跳转到公开域名预签名直链，外部访客直连来源、momoko 不代理。
	if redirect302 {
		if presigner, ok := store.(file.Presigner); ok {
			if u, err := presigner.Presign(ctx, real, inline, shareServeTTL); err == nil {
				http.Redirect(w, r, u, http.StatusFound)
				return nil
			}
		}
	}

	rc, entry, err := store.Open(ctx, real)
	if err != nil {
		return errors.New("文件不存在")
	}
	defer rc.Close()

	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", disposition+`; filename="`+entry.Name+`"`)

	var mod time.Time
	if entry.UpdateTime != nil {
		mod = entry.UpdateTime.AsTime()
	}
	// 可 Seek 的来源（本地、OSS）走 ServeContent，支持 Range 续传/拖动。
	if seeker, ok := rc.(io.ReadSeeker); ok {
		w.Header().Set("Accept-Ranges", "bytes")
		http.ServeContent(w, r, entry.Name, mod, seeker)
		return nil
	}
	// 不可 Seek 的来源（FTP/WebDAV）：整流返回。
	ctype := mime.TypeByExtension(filepath.Ext(entry.Name))
	if ctype == "" {
		ctype = "application/octet-stream"
	}
	w.Header().Set("Content-Type", ctype)
	if entry.Size > 0 {
		w.Header().Set("Content-Length", strconv.FormatUint(entry.Size, 10))
	}
	_, err = io.Copy(w, rc)
	return err
}

// streamZip 将 roots（文件或目录）实时打包为 zip 写入 w（不落临时文件）：目录递归遍历，文件直接写入根层。
func streamZip(ctx context.Context, store file.Store, roots []string, name string, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.zip"`)
	zw := zip.NewWriter(w)
	defer zw.Close()

	for _, root := range roots {
		entry, err := store.Stat(ctx, root)
		if err != nil {
			return err
		}
		base := entry.Name
		if base == "" {
			base = baseName(root)
		}
		if entry.IsDir {
			if err := walkZip(ctx, store, root, base, zw); err != nil {
				return err
			}
			continue
		}
		rc, _, err := store.Open(ctx, root)
		if err != nil {
			return err
		}
		fw, err := zw.Create(base)
		if err != nil {
			_ = rc.Close()
			return err
		}
		_, err = io.Copy(fw, rc)
		_ = rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

func walkZip(ctx context.Context, store file.Store, dir, relBase string, zw *zip.Writer) error {
	entries, err := store.List(ctx, dir, file.SortByName, false)
	if err != nil {
		return err
	}
	for _, e := range entries {
		rel := joinSlash(relBase, e.Name)
		if e.IsDir {
			// e.Path 为来源原生路径，回传给 Store 可正确续游目录。
			if err := walkZip(ctx, store, e.Path, rel, zw); err != nil {
				return err
			}
			continue
		}
		rc, _, err := store.Open(ctx, e.Path)
		if err != nil {
			return err
		}
		fw, err := zw.Create(rel)
		if err != nil {
			_ = rc.Close()
			return err
		}
		_, err = io.Copy(fw, rc)
		_ = rc.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// ToInfo 将分享记录映射为管理视图（含提取码，仅创建者可见）。
func ToInfo(s *gen.FileShare) *v1.ShareInfo {
	info := &v1.ShareInfo{
		Id:            s.ID,
		Name:          s.Name,
		Paths:         s.Paths,
		SourceId:      s.SourceID,
		Token:         s.Token,
		Code:          s.Code,
		MaxDownloads:  s.MaxDownloads,
		DownloadCount: s.DownloadCount,
		Enabled:       s.Enabled,
		CreateTime:    timestamppb.New(s.CreateTime),
		UpdateTime:    timestamppb.New(s.UpdateTime),
	}
	if s.ExpiresAt != nil {
		info.ExpiresAt = timestamppb.New(*s.ExpiresAt)
	}
	return info
}

// ToMeta 将分享记录映射为公开元信息（不暴露提取码与真实路径）；owner 为分享者（可空）。
func ToMeta(s *gen.FileShare, owner *gen.User, now time.Time) *v1.GetShareMetaResponse {
	meta := &v1.GetShareMetaResponse{
		Name:          s.Name,
		NeedCode:      s.Code != "",
		MaxDownloads:  s.MaxDownloads,
		DownloadCount: s.DownloadCount,
		Available:     Available(s, now),
	}
	if owner != nil {
		meta.OwnerName = owner.Name
		if meta.OwnerName == "" {
			meta.OwnerName = owner.Username
		}
		meta.OwnerAvatar = owner.Avatar
	}
	if s.ExpiresAt != nil {
		meta.ExpiresAt = timestamppb.New(*s.ExpiresAt)
	}
	return meta
}
