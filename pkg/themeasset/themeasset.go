// Package themeasset 保存外观主题引用的资源文件（背景图 / 横幅图转成的 WebP、woff2 字体），
// 并在 /api/v1/theme-asset/<id> 公开提供访问（登录页也要用到全站默认主题的背景）。
package themeasset

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	httpm "github.com/go-kratos/kratos/v2/transport/http"

	"momoko/pkg/response"
)

const (
	// PublicPath 资源访问地址前缀，完整地址为 PublicPath + id。
	PublicPath = "/api/v1/theme-asset/"
	defaultDir = "data/theme-assets"

	KindImage = "image"
	KindFont  = "font"

	ExtWebP  = "webp"
	ExtWOFF2 = "woff2"
)

var idPattern = regexp.MustCompile(`^[0-9a-f]{32}\.(webp|woff2)$`)

// Manager 负责资源文件的保存、读取、删除和 HTTP 访问。
type Manager struct {
	rootDir string
}

func NewManager() (*Manager, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("获取工作目录失败: %w", err)
	}
	return NewManagerWithRoot(filepath.Join(workDir, defaultDir)), nil
}

func NewManagerWithRoot(rootDir string) *Manager {
	return &Manager{rootDir: filepath.Clean(rootDir)}
}

// ValidID 判断是否为合法的资源文件名（32 位小写十六进制 + .webp / .woff2）。
func ValidID(id string) bool {
	return idPattern.MatchString(id)
}

// KindOf 按扩展名返回资源类型（image / font）；不是合法 id 返回空串。
func KindOf(id string) string {
	if !ValidID(id) {
		return ""
	}
	if strings.HasSuffix(id, "."+ExtWOFF2) {
		return KindFont
	}
	return KindImage
}

// URL 返回资源的公开访问地址。
func URL(id string) string {
	return PublicPath + id
}

// NewID 生成新的资源文件名。
func NewID(ext string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Sprintf("themeasset: crypto/rand failed: %v", err))
	}
	return hex.EncodeToString(b[:]) + "." + ext
}

func (m *Manager) path(id string) (string, bool) {
	if !ValidID(id) {
		return "", false
	}
	return filepath.Join(m.rootDir, id), true
}

// Save 写入资源文件（先写临时文件再改名，避免读到半个文件）。
func (m *Manager) Save(id string, data []byte) error {
	target, ok := m.path(id)
	if !ok {
		return fmt.Errorf("非法的资源文件名: %q", id)
	}
	if err := os.MkdirAll(m.rootDir, 0o755); err != nil {
		return fmt.Errorf("创建主题资源目录失败: %w", err)
	}
	tmp, err := os.CreateTemp(m.rootDir, id+".*.tmp")
	if err != nil {
		return fmt.Errorf("创建主题资源临时文件失败: %w", err)
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return fmt.Errorf("写入主题资源失败: %w", err)
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("写入主题资源失败: %w", err)
	}
	if err := os.Rename(tmpPath, target); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("保存主题资源失败: %w", err)
	}
	return nil
}

// Read 读取资源文件内容。
func (m *Manager) Read(id string) ([]byte, error) {
	p, ok := m.path(id)
	if !ok {
		return nil, os.ErrNotExist
	}
	return os.ReadFile(p)
}

// Exists 判断资源文件是否存在。
func (m *Manager) Exists(id string) bool {
	p, ok := m.path(id)
	if !ok {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && info.Mode().IsRegular()
}

// Delete 删除资源文件（不存在视为成功）。
func (m *Manager) Delete(id string) error {
	p, ok := m.path(id)
	if !ok {
		return nil
	}
	if err := os.Remove(p); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("删除主题资源失败: %w", err)
	}
	return nil
}

// Filter 处理 /api/v1/theme-asset/ 下的资源访问请求（公开、无需登录）。
func (m *Manager) Filter() httpm.FilterFunc {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !strings.HasPrefix(r.URL.Path, PublicPath) {
				next.ServeHTTP(w, r)
				return
			}
			m.serveHTTP(w, r)
		})
	}
}

var errAssetNotFound = kerrors.NotFound("THEME_ASSET_NOT_FOUND", "Theme Asset Not Found")

func (m *Manager) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		response.WriteError(w, r, kerrors.New(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method Not Allowed"))
		return
	}
	id := strings.TrimPrefix(r.URL.Path, PublicPath)
	p, ok := m.path(id)
	if !ok {
		response.WriteError(w, r, errAssetNotFound)
		return
	}
	f, err := os.Open(p)
	if err != nil {
		response.WriteError(w, r, errAssetNotFound)
		return
	}
	defer f.Close()
	info, err := f.Stat()
	if err != nil || !info.Mode().IsRegular() {
		response.WriteError(w, r, errAssetNotFound)
		return
	}
	contentType := "image/webp"
	if KindOf(id) == KindFont {
		contentType = "font/woff2"
	}
	h := w.Header()
	h.Set("Content-Type", contentType)
	h.Set("Cache-Control", "public, max-age=31536000, immutable")
	h.Set("X-Content-Type-Options", "nosniff")
	http.ServeContent(w, r, id, info.ModTime(), f)
}
