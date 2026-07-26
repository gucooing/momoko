// Package share 实现对外分享的底层逻辑：令牌生成、可用性与提取码校验、目录浏览（防穿越）、
// 以及下载（单文件直传、多选/文件夹实时打包 zip）。
//
// 分享内容抽象为一层虚拟顶层目录：被分享的若干条目（文件或文件夹）即其顶层条目，
// 单选/多选、文件/文件夹由此统一处理。每个条目自带来源 id，可跨来源（本地 / OSS / FTP / WebDAV）
// 混合于同一分享。创建时探测并缓存各条目的名称/类型/大小/修改时间，浏览分享根目录时
// 直接用缓存、无需再访问来源。按分层约定，这里只放与分享相关的纯逻辑与 Store 调用；
// biz 仅负责鉴权 + 仓储 + 提供来源 Store 解析器。
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
	"momoko/internal/data/ent/schema/sharetype"
	"momoko/pkg/file"
	"momoko/pkg/localfs"
)

// shareServeTTL 是公开分享 302 直链的有效期。
const shareServeTTL = 5 * time.Minute

// ErrItemNotFound 表示待分享条目在其来源内不存在（biz 据此回 ErrFileNotExist）。
var ErrItemNotFound = errors.New("文件不存在")

// StoreResolver 按来源 id 解析对应存储后端（空/"local"=本地），第二返回值在远端来源时为来源记录。
// 由 biz 注入（其 storeFor），使「构造/解密来源」留在 biz，而「按条目选择来源」的编排留在本包。
type StoreResolver func(ctx context.Context, sourceID string) (file.Store, *gen.FileSource, error)

// GenToken 生成 URL 安全的随机分享令牌。
func GenToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// PrepareItems 校验每个待分享条目在其来源内存在，并回填展示名/类型/大小/修改时间（缓存进分享记录，
// 使后续浏览根目录无需再逐一 Stat）。返回补全后的条目与默认展示名（name 为空时取首个条目名）。
// 条目缺失返回 ErrItemNotFound；来源解析失败返回解析器给出的错误（如来源停用）。
func PrepareItems(ctx context.Context, resolve StoreResolver, inputs []*v1.ShareItem, name string) ([]sharetype.Item, string, error) {
	if len(inputs) == 0 {
		return nil, "", ErrItemNotFound
	}
	items := make([]sharetype.Item, 0, len(inputs))
	for i, in := range inputs {
		store, _, err := resolve(ctx, in.SourceId)
		if err != nil {
			return nil, "", err
		}
		entry, err := store.Stat(ctx, in.Path)
		if err != nil {
			return nil, "", ErrItemNotFound
		}
		nm := entry.Name
		if nm == "" {
			nm = baseName(in.Path)
		}
		var mod time.Time
		if entry.UpdateTime != nil {
			mod = entry.UpdateTime.AsTime()
		}
		items = append(items, sharetype.Item{
			SourceID:   in.SourceId,
			Path:       in.Path,
			Name:       nm,
			IsDir:      entry.IsDir,
			Size:       entry.Size,
			UpdateTime: mod,
		})
		if i == 0 && name == "" {
			name = nm
		}
	}
	return items, name, nil
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

// matchItem 按虚拟子路径的首段匹配某个被分享条目，返回该条目与其余子路径段。
func matchItem(s *gen.FileShare, sub string) (sharetype.Item, string, bool) {
	sub = normalizeSub(sub)
	if sub == "" {
		return sharetype.Item{}, "", false
	}
	first, rest, _ := strings.Cut(sub, "/")
	for _, it := range s.Items {
		if it.Name != first {
			continue
		}
		return it, rest, true
	}
	return sharetype.Item{}, "", false
}

// resolveItem 把公开端的虚拟子路径映射为「一个已限定到该条目的来源 + 来源内路径」。
//
// 本地目录条目会被收窄成锁死在该条目内的只读来源：公开分享是外部可达面，
// 不该让「首段匹配 + 其余段拼接」这层字符串逻辑成为唯一的边界。
// 收窄之后，即便 normalizeSub 将来出现纰漏，也只能在被分享的那个目录里打转。
//
// 远端来源无法收窄，其边界由来源自身的路径归一化把关（remoteRel 拒绝一切 ".."）。
func resolveItem(ctx context.Context, resolve StoreResolver, s *gen.FileShare, sub string) (sharetype.Item, file.Store, *gen.FileSource, string, error) {
	it, rest, ok := matchItem(s, sub)
	if !ok {
		return it, nil, nil, "", errors.New("路径不存在")
	}
	store, src, err := resolve(ctx, it.SourceID)
	if err != nil {
		return it, nil, nil, "", err
	}
	if local, ok := store.(*file.LocalStore); ok && it.IsDir {
		scoped, err := local.Sub(it.Path, localfs.ReadOnly())
		if err != nil {
			return it, nil, nil, "", errors.New("路径不存在")
		}
		// 收窄后 rest 就是来源内的完整路径；空表示条目根自身。
		if rest == "" {
			rest = "."
		}
		return it, scoped, src, rest, nil
	}
	if rest == "" {
		return it, store, src, it.Path, nil
	}
	return it, store, src, strings.TrimRight(it.Path, "/\\") + "/" + rest, nil
}

func joinSlash(base, name string) string {
	if base == "" {
		return name
	}
	return base + "/" + name
}

// sortShareEntries 按目录优先、名称升序排序。
func sortShareEntries(items []*v1.ShareEntry) {
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return items[i].Name < items[j].Name
	})
}

// ListDir 列出分享内某子路径的条目。subPath 为空时直接用缓存的顶层条目（不访问任何来源，
// 显著加快分享页打开）；否则经 resolveItem 映射到对应来源的真实目录后列出。
// 返回的 RelPath 均为虚拟路径，公开端据此续航，不暴露来源真实路径与来源 id。
func ListDir(ctx context.Context, resolve StoreResolver, s *gen.FileShare, subPath string) ([]*v1.ShareEntry, string, error) {
	sub := normalizeSub(subPath)
	// 根目录：直接用创建时缓存的条目属性，无需 Stat/List 任何来源。
	if sub == "" {
		items := make([]*v1.ShareEntry, 0, len(s.Items))
		for _, it := range s.Items {
			e := &v1.ShareEntry{
				Name:    it.Name,
				IsDir:   it.IsDir,
				Size:    it.Size,
				RelPath: it.Name, // 顶层条目的虚拟路径即其展示名
			}
			if !it.UpdateTime.IsZero() {
				e.UpdateTime = timestamppb.New(it.UpdateTime)
			}
			items = append(items, e)
		}
		sortShareEntries(items)
		return items, sub, nil
	}

	// 子目录：定位条目 → 解析出限定到该条目的来源 → 列目录。
	_, store, _, real, err := resolveItem(ctx, resolve, s, sub)
	if err != nil {
		return nil, "", err
	}
	entries, err := store.List(ctx, real, file.SortByName, false)
	if err != nil {
		return nil, "", errors.New("读取目录失败")
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
	sortShareEntries(items)
	return items, sub, nil
}

// ServeDownload 将分享内 subPath 指向的目标写入 w：subPath 为空时打包全部顶层条目（可跨来源）为 zip；
// 指向目录时打包该目录；指向文件时直传（可 Seek 来源支持 Range）。inline=true 以内联方式返回（用于预览）。
// 若目标条目所属来源开启 redirect_302 且支持预签名（OSS），单文件改为 302 跳转到（公开域名）预签名直链，momoko 不在数据路径。
func ServeDownload(ctx context.Context, resolve StoreResolver, s *gen.FileShare, subPath string, inline bool, w http.ResponseWriter, r *http.Request) error {
	sub := normalizeSub(subPath)
	// 根：打包全部顶层条目（可能来自不同来源）为 zip。
	if sub == "" {
		return streamZipItems(ctx, resolve, s.Items, s.Name, w)
	}
	it, store, src, real, err := resolveItem(ctx, resolve, s, sub)
	if err != nil {
		return errors.New("文件不存在")
	}
	entry, err := store.Stat(ctx, real)
	if err != nil {
		return errors.New("文件不存在")
	}
	// 子路径指向目录：实时打包为 zip。
	if entry.IsDir {
		name := entry.Name
		if real == "." { // 收窄来源的根：用分享里展示的条目名，而不是磁盘上的目录名
			name = it.Name
		}
		return streamZipRoot(ctx, store, real, name, w)
	}
	// 该条目来源开启 302 且支持预签名（OSS）→ 跳转到公开域名预签名直链，外部访客直连来源、momoko 不代理。
	if src != nil && src.Redirect302 {
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

// zipHeader 写 zip 响应头。
func zipHeader(w http.ResponseWriter, name string) {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.zip"`)
}

// streamZipItems 将多个（可跨来源）条目实时打包为 zip 写入 w：逐条目解析其来源后写入。
// 与浏览一致：本地目录条目先收窄成独立来源，打包遍历不可能爬出被分享的目录。
func streamZipItems(ctx context.Context, resolve StoreResolver, items []sharetype.Item, name string, w http.ResponseWriter) error {
	zipHeader(w, name)
	zw := zip.NewWriter(w)
	defer zw.Close()
	for _, it := range items {
		store, _, err := resolve(ctx, it.SourceID)
		if err != nil {
			return err
		}
		root := it.Path
		if local, ok := store.(*file.LocalStore); ok && it.IsDir {
			scoped, err := local.Sub(it.Path, localfs.ReadOnly())
			if err != nil {
				return err
			}
			store, root = scoped, "."
		}
		if err := zipRootNamed(ctx, store, root, it.Name, zw); err != nil {
			return err
		}
	}
	return nil
}

// streamZipRoot 将单一来源内的一个文件/目录实时打包为 zip 写入 w。
func streamZipRoot(ctx context.Context, store file.Store, root, name string, w http.ResponseWriter) error {
	zipHeader(w, name)
	zw := zip.NewWriter(w)
	defer zw.Close()
	return zipRootNamed(ctx, store, root, name, zw)
}

// zipRootNamed 把某来源内一个根（文件或目录）写入 zw：目录递归遍历（以其名为顶层），文件直接写入根层。
// name 为空时取 Stat 得到的名字；收窄来源后根目录的磁盘名未必等于分享里展示的条目名，故允许显式指定。
func zipRootNamed(ctx context.Context, store file.Store, root, name string, zw *zip.Writer) error {
	entry, err := store.Stat(ctx, root)
	if err != nil {
		return err
	}
	base := name
	if base == "" {
		base = entry.Name
	}
	if base == "" {
		base = baseName(root)
	}
	if entry.IsDir {
		return walkZip(ctx, store, root, base, zw)
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
	return err
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

// ToInfo 将分享记录映射为管理视图（含提取码与来源 id，仅创建者可见）。
func ToInfo(s *gen.FileShare) *v1.ShareInfo {
	info := &v1.ShareInfo{
		Id:            s.ID,
		Name:          s.Name,
		Items:         toItemInfos(s.Items),
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

// toItemInfos 将缓存条目映射为管理视图条目（保留来源 id 与展示属性；公开端另用 ShareEntry 不含来源）。
func toItemInfos(items []sharetype.Item) []*v1.ShareItem {
	out := make([]*v1.ShareItem, 0, len(items))
	for _, it := range items {
		out = append(out, &v1.ShareItem{
			SourceId: it.SourceID,
			Path:     it.Path,
			Name:     it.Name,
			IsDir:    it.IsDir,
			Size:     it.Size,
		})
	}
	return out
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
