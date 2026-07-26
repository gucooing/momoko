package avatar

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	kerrors "github.com/go-kratos/kratos/v2/errors"
	httpm "github.com/go-kratos/kratos/v2/transport/http"

	"momoko/pkg/localfs"
	"momoko/pkg/response"
)

const (
	PublicPath  = "/api/v1/avatar/"
	defaultDir  = "data/avatar"
	maxFileSize = 10 << 20
)

var (
	dataURLPattern = regexp.MustCompile(`^data:image/([a-zA-Z0-9.+-]+);base64,(.+)$`)
	allowedExts    = []string{"jpg", "png", "webp", "gif"}
)

var (
	ErrInvalidInput = errors.New("头像格式不正确，仅支持 URL 或 data:image 形式")
	ErrInvalidImage = errors.New("头像仅支持 jpg、png、webp、gif 格式")
	ErrFileTooLarge = errors.New("头像大小不能超过 10MB")
)

// Manager 负责头像文件的保存、删除和 HTTP 访问。
// 所有磁盘操作都经由锁死在头像目录内的 localfs 视图，因此文件名即便被构造得再离奇也出不了这个目录。
type Manager struct {
	rootDir string
	view    *localfs.FS
}

func NewManager() (*Manager, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("获取工作目录失败: %w", err)
	}
	return NewManagerWithRoot(filepath.Join(workDir, defaultDir))
}

// NewManagerWithRoot 打开（必要时创建）头像目录并返回管理器。
func NewManagerWithRoot(rootDir string) (*Manager, error) {
	view, err := localfs.OpenDir(rootDir, localfs.WithMaxFileSize(maxFileSize))
	if err != nil {
		return nil, fmt.Errorf("初始化头像目录失败: %w", err)
	}
	return &Manager{rootDir: filepath.Clean(rootDir), view: view}, nil
}

// Filter 处理 /api/v1/avatar/ 下的头像访问请求。
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

// ManagedValue 返回本地托管头像最终保存到数据库中的固定路径。
func (m *Manager) ManagedValue(userID string) string {
	return PublicPath + sanitizePathSegment(userID)
}

// Prepare 预处理头像输入。
// commit 在数据库写入成功后执行，rollback 在数据库写入失败后执行。
func (m *Manager) Prepare(userID, raw string) (value string, commit func() error, rollback func(), err error) {
	raw = strings.TrimSpace(raw)
	managedValue := m.ManagedValue(userID)

	switch {
	case raw == "":
		return "", func() error { return m.DeleteByUserID(userID) }, func() {}, nil
	case isHTTPURL(raw):
		return raw, func() error { return m.DeleteByUserID(userID) }, func() {}, nil
	case m.IsManaged(raw):
		if raw != managedValue {
			return "", nil, nil, ErrInvalidInput
		}
		return managedValue, func() error { return nil }, func() {}, nil
	}

	data, err := decodeDataURL(raw)
	if err != nil {
		return "", nil, nil, err
	}
	if len(data) > maxFileSize {
		return "", nil, nil, ErrFileTooLarge
	}

	ext, err := detectExt(data)
	if err != nil {
		return "", nil, nil, err
	}

	// 落盘推迟到 commit：localfs.WriteFile 本身就是「写临时文件再改名」的原子写，
	// 于是既不需要手工备份/回滚，也不会在数据库写失败时留下 .tmp / .bak 残渣。
	rollback = func() {}
	commit = func() error {
		if _, err := m.view.WriteFile(m.fileName(userID, ext), bytes.NewReader(data)); err != nil {
			return fmt.Errorf("保存头像文件失败: %w", err)
		}
		return m.deleteOtherFormats(userID, ext)
	}

	return managedValue, commit, rollback, nil
}

func (m *Manager) Delete(raw string) error {
	if !m.IsManaged(raw) {
		return nil
	}

	userID, ok := m.requestedUserID(raw)
	if !ok {
		return nil
	}
	return m.DeleteByUserID(userID)
}

// DeleteByUserID 删除指定用户的所有本地头像文件。
func (m *Manager) DeleteByUserID(userID string) error {
	names := make([]string, 0, len(allowedExts))
	for _, ext := range allowedExts {
		names = append(names, m.fileName(userID, ext))
	}
	for _, res := range m.view.Remove(names) {
		if !res.OK {
			return fmt.Errorf("删除头像文件失败: %s", res.Message)
		}
	}
	return nil
}

func (m *Manager) IsManaged(raw string) bool {
	return strings.HasPrefix(strings.TrimSpace(raw), PublicPath)
}

func (m *Manager) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		response.WriteError(w, r, kerrors.New(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method Not Allowed"))
		return
	}

	name, ok := m.requestedFileName(r.URL.Path)
	if !ok {
		response.WriteError(w, r, kerrors.NotFound("AVATAR_NOT_FOUND", "Avatar Not Found"))
		return
	}
	// 经受限视图打开而不是 http.ServeFile(绝对路径)：读取同样被锁在头像目录内，
	// 且不会顺着符号链接把目录外的文件吐出去。
	f, entry, err := m.view.OpenRead(name)
	if err != nil {
		response.WriteError(w, r, kerrors.NotFound("AVATAR_NOT_FOUND", "Avatar Not Found"))
		return
	}
	defer f.Close()
	http.ServeContent(w, r, entry.Name, entry.ModTime, f)
}

// requestedFileName 把请求路径映射为头像目录内的文件名（挑第一个真实存在的扩展名）。
func (m *Manager) requestedFileName(raw string) (string, bool) {
	userID, ok := m.requestedUserID(raw)
	if !ok {
		return "", false
	}
	for _, ext := range allowedExts {
		name := m.fileName(userID, ext)
		if m.view.Exists(name) {
			return name, true
		}
	}
	return "", false
}

func (m *Manager) requestedUserID(raw string) (string, bool) {
	cleanPath := strings.TrimSpace(raw)
	if !strings.HasPrefix(cleanPath, PublicPath) {
		return "", false
	}

	relPath := strings.TrimPrefix(cleanPath, PublicPath)
	if relPath == "" || strings.Contains(relPath, "/") || strings.Contains(relPath, "\\") {
		return "", false
	}
	if relPath == "." || relPath == ".." {
		return "", false
	}

	cleanRelPath := strings.TrimPrefix(path.Clean("/"+relPath), "/")
	if cleanRelPath == "" || cleanRelPath == "." || cleanRelPath == ".." {
		return "", false
	}
	if cleanRelPath != relPath {
		return "", false
	}
	if sanitizePathSegment(cleanRelPath) != cleanRelPath {
		return "", false
	}
	return cleanRelPath, true
}

// fileName 返回头像在受限视图内的文件名（单层，不含任何路径成分）。
func (m *Manager) fileName(userID, ext string) string {
	return sanitizePathSegment(userID) + "." + ext
}

func (m *Manager) deleteOtherFormats(userID, keepExt string) error {
	names := make([]string, 0, len(allowedExts))
	for _, ext := range allowedExts {
		if ext == keepExt {
			continue
		}
		names = append(names, m.fileName(userID, ext))
	}
	for _, res := range m.view.Remove(names) {
		if !res.OK {
			return fmt.Errorf("删除旧头像文件失败: %s", res.Message)
		}
	}
	return nil
}

func decodeDataURL(raw string) ([]byte, error) {
	matches := dataURLPattern.FindStringSubmatch(strings.TrimSpace(raw))
	if len(matches) != 3 {
		return nil, ErrInvalidInput
	}

	data, err := base64.StdEncoding.DecodeString(matches[2])
	if err != nil {
		return nil, ErrInvalidInput
	}
	return data, nil
}

func detectExt(data []byte) (string, error) {
	switch {
	case len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff:
		return "jpg", nil
	case len(data) >= 8 && bytes.Equal(data[:8], []byte{0x89, 'P', 'N', 'G', '\r', '\n', 0x1a, '\n'}):
		return "png", nil
	case len(data) >= 6 && (bytes.Equal(data[:6], []byte("GIF87a")) || bytes.Equal(data[:6], []byte("GIF89a"))):
		return "gif", nil
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return "webp", nil
	default:
		return "", ErrInvalidImage
	}
}

func isHTTPURL(raw string) bool {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return false
	}
	return (parsed.Scheme == "http" || parsed.Scheme == "https") && parsed.Host != ""
}

func sanitizePathSegment(value string) string {
	if value == "" {
		return "anonymous"
	}

	var b strings.Builder
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r >= 'A' && r <= 'Z':
			b.WriteRune(r)
		case r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_' || r == '.':
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}

	sanitized := strings.Trim(b.String(), "._")
	if sanitized == "" {
		return "anonymous"
	}
	return sanitized
}
