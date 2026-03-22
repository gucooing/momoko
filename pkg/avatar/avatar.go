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

	if err := os.MkdirAll(m.rootDir, 0o755); err != nil {
		return "", nil, nil, fmt.Errorf("创建头像目录失败: %w", err)
	}

	safeID := sanitizePathSegment(userID)
	pattern := safeID + ".*." + ext + ".tmp"
	tempFile, err := os.CreateTemp(m.rootDir, pattern)
	if err != nil {
		return "", nil, nil, fmt.Errorf("创建头像临时文件失败: %w", err)
	}

	tempPath := tempFile.Name()
	if _, err := tempFile.Write(data); err != nil {
		_ = tempFile.Close()
		_ = os.Remove(tempPath)
		return "", nil, nil, fmt.Errorf("写入头像临时文件失败: %w", err)
	}
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return "", nil, nil, fmt.Errorf("关闭头像临时文件失败: %w", err)
	}

	rollback = func() {
		_ = os.Remove(tempPath)
	}

	commit = func() error {
		finalPath := m.filePath(userID, ext)
		backupPath := finalPath + ".bak"

		_ = os.Remove(backupPath)
		if err := os.Rename(finalPath, backupPath); err != nil && !os.IsNotExist(err) {
			_ = os.Remove(tempPath)
			return fmt.Errorf("备份旧头像失败: %w", err)
		}

		if err := os.Rename(tempPath, finalPath); err != nil {
			_ = os.Remove(tempPath)
			if _, statErr := os.Stat(backupPath); statErr == nil {
				_ = os.Rename(backupPath, finalPath)
			}
			return fmt.Errorf("保存头像文件失败: %w", err)
		}

		_ = os.Remove(backupPath)
		if err := m.deleteOtherFormats(userID, ext); err != nil {
			return err
		}
		return nil
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
	var firstErr error
	for _, ext := range allowedExts {
		if err := os.Remove(m.filePath(userID, ext)); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = fmt.Errorf("删除头像文件失败: %w", err)
		}
	}
	return firstErr
}

func (m *Manager) IsManaged(raw string) bool {
	return strings.HasPrefix(strings.TrimSpace(raw), PublicPath)
}

func (m *Manager) serveHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		response.WriteError(w, r, kerrors.New(http.StatusMethodNotAllowed, "METHOD_NOT_ALLOWED", "Method Not Allowed"))
		return
	}

	filePath, ok := m.requestedFilePath(r.URL.Path)
	if !ok {
		response.WriteError(w, r, kerrors.NotFound("AVATAR_NOT_FOUND", "Avatar Not Found"))
		return
	}

	if _, err := os.Stat(filePath); err != nil {
		response.WriteError(w, r, kerrors.NotFound("AVATAR_NOT_FOUND", "Avatar Not Found"))
		return
	}
	http.ServeFile(w, r, filePath)
}

func (m *Manager) requestedFilePath(raw string) (string, bool) {
	userID, ok := m.requestedUserID(raw)
	if !ok {
		return "", false
	}

	for _, ext := range allowedExts {
		filePath := m.filePath(userID, ext)
		if _, err := os.Stat(filePath); err == nil {
			return filePath, true
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

func (m *Manager) filePath(userID, ext string) string {
	return filepath.Join(m.rootDir, sanitizePathSegment(userID)+"."+ext)
}

func (m *Manager) deleteOtherFormats(userID, keepExt string) error {
	var firstErr error
	for _, ext := range allowedExts {
		if ext == keepExt {
			continue
		}
		if err := os.Remove(m.filePath(userID, ext)); err != nil && !os.IsNotExist(err) && firstErr == nil {
			firstErr = fmt.Errorf("删除旧头像文件失败: %w", err)
		}
	}
	return firstErr
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
