// Package share 实现文件/文件夹对外分享的底层逻辑：令牌生成、可用性与提取码校验、
// 目录浏览（防穿越）、以及下载（单文件直传、文件夹实时打包 zip）。
//
// 按分层约定，这里只放与分享相关的纯逻辑与文件系统操作；biz 仅负责鉴权 + 仓储调用，
// 不直接编写文件系统/打包/校验等底层代码。
package share

import (
	"archive/zip"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/file"
)

// GenToken 生成 URL 安全的随机分享令牌。
func GenToken() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// Prepare 检查被分享的真实路径，返回是否目录与默认展示名（路径基名）。
func Prepare(realPath string) (isDir bool, name string, err error) {
	fi, err := os.Stat(realPath)
	if err != nil {
		return false, "", errors.New("文件不存在")
	}
	return fi.IsDir(), fi.Name(), nil
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

// resolve 将分享内的相对子路径解析为真实路径（限制在分享根内，防止路径穿越）。
func resolve(s *gen.FileShare, subPath string) (string, error) {
	if !s.IsDir {
		return s.TargetPath, nil
	}
	if subPath == "" || subPath == "." || subPath == "/" {
		return s.TargetPath, nil
	}
	oper, err := file.NewFileOper(s.TargetPath)
	if err != nil {
		return "", err
	}
	return oper.ResolveRealPath(subPath)
}

// ListDir 列出文件夹分享下某子路径的条目（按目录优先、名称升序），返回相对分享根的路径。
func ListDir(s *gen.FileShare, subPath string) ([]*v1.ShareEntry, string, error) {
	if !s.IsDir {
		return nil, "", errors.New("该分享不是文件夹")
	}
	dir, err := resolve(s, subPath)
	if err != nil {
		return nil, "", err
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, "", errors.New("读取目录失败")
	}
	items := make([]*v1.ShareEntry, 0, len(entries))
	for _, e := range entries {
		fi, err := e.Info()
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(s.TargetPath, filepath.Join(dir, e.Name()))
		if err != nil {
			continue
		}
		items = append(items, &v1.ShareEntry{
			Name:       e.Name(),
			IsDir:      e.IsDir(),
			Size:       uint64(fi.Size()),
			UpdateTime: timestamppb.New(fi.ModTime()),
			RelPath:    filepath.ToSlash(rel),
		})
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].IsDir != items[j].IsDir {
			return items[i].IsDir
		}
		return items[i].Name < items[j].Name
	})
	return items, filepath.ToSlash(subPath), nil
}

// ServeDownload 将分享内 subPath 指向的目标写入 w：文件直接传输（支持 Range），
// 文件夹则实时打包为 zip 流式返回。inline=true 时以内联方式返回（用于预览，不强制下载）。
func ServeDownload(s *gen.FileShare, subPath string, inline bool, w http.ResponseWriter, r *http.Request) error {
	real, err := resolve(s, subPath)
	if err != nil {
		return err
	}
	fi, err := os.Stat(real)
	if err != nil {
		return errors.New("文件不存在")
	}
	if fi.IsDir() {
		return streamZip(real, fi.Name(), w)
	}
	f, err := os.Open(real)
	if err != nil {
		return errors.New("文件打开失败")
	}
	defer f.Close()
	disposition := "attachment"
	if inline {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition", disposition+`; filename="`+fi.Name()+`"`)
	w.Header().Set("Accept-Ranges", "bytes")
	http.ServeContent(w, r, fi.Name(), fi.ModTime(), f)
	return nil
}

// streamZip 将 dir 下所有文件实时打包为 zip 写入 w（不落临时文件）。
func streamZip(dir, name string, w http.ResponseWriter) error {
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`.zip"`)
	zw := zip.NewWriter(w)
	defer zw.Close()
	return filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		fw, err := zw.Create(filepath.ToSlash(rel))
		if err != nil {
			return err
		}
		src, err := os.Open(p)
		if err != nil {
			return err
		}
		defer src.Close()
		_, err = io.Copy(fw, src)
		return err
	})
}

// ToInfo 将分享记录映射为管理视图（含提取码，仅创建者可见）。
func ToInfo(s *gen.FileShare) *v1.ShareInfo {
	info := &v1.ShareInfo{
		Id:            s.ID,
		Name:          s.Name,
		TargetPath:    s.TargetPath,
		IsDir:         s.IsDir,
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

// ToMeta 将分享记录映射为公开元信息（不暴露提取码与真实路径）。
func ToMeta(s *gen.FileShare, now time.Time) *v1.GetShareMetaResponse {
	meta := &v1.GetShareMetaResponse{
		Name:          s.Name,
		IsDir:         s.IsDir,
		NeedCode:      s.Code != "",
		MaxDownloads:  s.MaxDownloads,
		DownloadCount: s.DownloadCount,
		Available:     Available(s, now),
	}
	if s.ExpiresAt != nil {
		meta.ExpiresAt = timestamppb.New(*s.ExpiresAt)
	}
	if !s.IsDir {
		if fi, err := os.Stat(s.TargetPath); err == nil {
			meta.Size = uint64(fi.Size())
		}
	}
	return meta
}
