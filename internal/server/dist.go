package server

import (
	"bytes"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"

	root "momoko"

	httpm "github.com/go-kratos/kratos/v2/transport/http"
)

const (
	apiPrefix                 = "/api/v1"
	distIndex                 = "index.html"
	distFolder                = "frontend/dist"
	distEncodingBrotli        = "br"
	distEncodingGzip          = "gzip"
	distFileSuffixBrotli      = ".br"
	distFileSuffixGzip        = ".gz"
	distCacheControlNoCache   = "no-cache"
	distCacheControlImmutable = "public, max-age=31536000, immutable"
)

type distFile struct {
	rawPath    string
	modTime    time.Time
	brotliPath string
	gzipPath   string
}

func (f distFile) hasCompressedVariant() bool {
	return f.brotliPath != "" || f.gzipPath != ""
}

type distHandler struct {
	files     fs.FS
	distFiles map[string]distFile
}

func distMiddleware() httpm.FilterFunc {
	handler, ok := newDistHandler()
	if !ok {
		return passthroughFilter()
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if handler.tryServe(w, r) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func passthroughFilter() httpm.FilterFunc {
	return func(next http.Handler) http.Handler {
		return next
	}
}

func newDistHandler() (*distHandler, bool) {
	distFS, ok := embeddedDistFS()
	if !ok {
		return nil, false
	}
	return newDistHandlerWithFS(distFS), true
}

// passportDistMiddleware 用户通行证 server 的前端：frontend/dist-passport（SPA 回退到它的 index.html），
// 以及游戏素材 /games/（静态文件在管理端产物 frontend/dist/games/ 里，两端共用一份）。
// 通行证自己的页面也在 /games/ 下（/games/<服务器>?tab=…）：带扩展名的路径是素材，找不到就 404；
// 不带扩展名的是页面，交给 SPA 回退（否则在游戏页里刷新会 404）。
func passportDistMiddleware() httpm.FilterFunc {
	app := passportDistHandler()
	var shared *distHandler
	if distFS, ok := embeddedDistFS(); ok {
		shared = newDistHandlerWithFS(distFS)
	}
	if app == nil && shared == nil {
		return passthroughFilter()
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if (r.Method == http.MethodGet || r.Method == http.MethodHead) && strings.HasPrefix(r.URL.Path, "/games/") {
				filePath := strings.TrimPrefix(path.Clean(r.URL.Path), "/")
				if shared != nil {
					if _, ok := shared.distFiles[filePath]; ok && shared.serveDistFile(w, r, filePath) {
						return
					}
				}
				if path.Ext(filePath) != "" || app == nil {
					http.NotFound(w, r)
					return
				}
			}
			if app != nil && app.tryServe(w, r) {
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

func newDistHandlerWithFS(distFS fs.FS) *distHandler {
	return &distHandler{
		files:     distFS,
		distFiles: buildDistFiles(distFS),
	}
}

func (h *distHandler) tryServe(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if r.URL == nil {
		return false
	}
	// 所有 /api/ 路径都交给接口处理，不做 SPA 回退。
	if r.URL.Path == "/api" || strings.HasPrefix(r.URL.Path, "/api/") || r.URL.Path == apiPrefix {
		return false
	}

	cleanPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
	if cleanPath == "/" {
		return h.serveDistFile(w, r, distIndex)
	}
	if cleanPath == "/.well-known/openid-configuration" {
		return false
	}

	filePath := strings.TrimPrefix(cleanPath, "/")
	if _, ok := h.distFiles[filePath]; ok {
		return h.serveDistFile(w, r, filePath)
	}

	return h.serveDistFile(w, r, distIndex)
}

func (h *distHandler) serveDistFile(w http.ResponseWriter, r *http.Request, filePath string) bool {
	item, ok := h.distFiles[filePath]
	if !ok || item.rawPath == "" {
		return false
	}

	setDistCacheHeaders(w.Header(), filePath)
	if h.serveCompressedVariant(w, r, filePath, item) {
		return true
	}
	if item.hasCompressedVariant() {
		appendVary(w.Header(), "Accept-Encoding")
	}

	// 不经 http.FileServer：它会把 /index.html 重定向到 ./，未压缩的入口页会陷入 301 循环。
	file, err := h.files.Open(item.rawPath)
	if err != nil {
		return false
	}
	defer file.Close()
	content, ok := file.(io.ReadSeeker)
	if !ok {
		data, err := io.ReadAll(file)
		if err != nil {
			return false
		}
		content = bytes.NewReader(data)
	}
	http.ServeContent(w, r, path.Base(filePath), item.modTime, content)
	return true
}

func (h *distHandler) serveCompressedVariant(w http.ResponseWriter, r *http.Request, filePath string, item distFile) bool {
	encoding, compressedPath, ok := selectCompressedVariant(r, item)
	if !ok {
		return false
	}

	data, err := fs.ReadFile(h.files, compressedPath)
	if err != nil {
		return false
	}

	appendVary(w.Header(), "Accept-Encoding")
	w.Header().Set("Content-Encoding", encoding)
	if contentType := mime.TypeByExtension(path.Ext(filePath)); contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	http.ServeContent(w, r, path.Base(filePath), item.modTime, bytes.NewReader(data))
	return true
}

func selectCompressedVariant(r *http.Request, item distFile) (string, string, bool) {
	if r == nil || r.Header.Get("Range") != "" {
		return "", "", false
	}

	acceptEncoding := r.Header.Get("Accept-Encoding")
	brotliQ := acceptedEncodingQuality(acceptEncoding, distEncodingBrotli)
	gzipQ := acceptedEncodingQuality(acceptEncoding, distEncodingGzip)

	switch {
	case item.brotliPath != "" && brotliQ > 0 && brotliQ >= gzipQ:
		return distEncodingBrotli, item.brotliPath, true
	case item.gzipPath != "" && gzipQ > 0:
		return distEncodingGzip, item.gzipPath, true
	case item.brotliPath != "" && brotliQ > 0:
		return distEncodingBrotli, item.brotliPath, true
	default:
		return "", "", false
	}
}

func acceptedEncodingQuality(value, target string) float64 {
	wildcardQ := -1.0
	for _, part := range strings.Split(value, ",") {
		part = strings.TrimSpace(strings.ToLower(part))
		if part == "" {
			continue
		}

		name := part
		qValue := 1.0
		if idx := strings.Index(name, ";"); idx >= 0 {
			params := strings.Split(name[idx+1:], ";")
			name = strings.TrimSpace(name[:idx])
			for _, param := range params {
				param = strings.TrimSpace(param)
				if !strings.HasPrefix(param, "q=") {
					continue
				}
				value, err := strconv.ParseFloat(strings.TrimSpace(strings.TrimPrefix(param, "q=")), 64)
				if err == nil {
					qValue = value
				}
				break
			}
		}

		if qValue <= 0 {
			continue
		}
		if name == target {
			return qValue
		}
		if name == "*" {
			wildcardQ = qValue
		}
	}
	if wildcardQ > 0 {
		return wildcardQ
	}
	return 0
}

func buildDistFiles(fsys fs.FS) map[string]distFile {
	files := make(map[string]distFile)
	_ = fs.WalkDir(fsys, ".", func(name string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}

		switch {
		case strings.HasSuffix(name, distFileSuffixBrotli):
			original := strings.TrimSuffix(name, distFileSuffixBrotli)
			if !fileExists(fsys, original) {
				return nil
			}
			item := files[original]
			item.brotliPath = name
			files[original] = item
		case strings.HasSuffix(name, distFileSuffixGzip):
			original := strings.TrimSuffix(name, distFileSuffixGzip)
			if !fileExists(fsys, original) {
				return nil
			}
			item := files[original]
			item.gzipPath = name
			files[original] = item
		default:
			item := files[name]
			item.rawPath = name
			item.modTime = fileModTime(fsys, name)
			files[name] = item
		}
		return nil
	})

	for name, item := range files {
		if item.rawPath == "" {
			delete(files, name)
		}
	}
	return files
}

func setDistCacheHeaders(h http.Header, filePath string) {
	switch {
	case filePath == distIndex:
		h.Set("Cache-Control", distCacheControlNoCache)
	case isImmutableDistAsset(filePath):
		h.Set("Cache-Control", distCacheControlImmutable)
	default:
		h.Set("Cache-Control", distCacheControlNoCache)
	}
}

func isImmutableDistAsset(filePath string) bool {
	// assets/ 是 Vite 带哈希的产物；games/ 是游戏素材（文件名 <name>.<hash>.webp，内容变了文件名就变）。
	return strings.HasPrefix(filePath, "assets/") || strings.HasPrefix(filePath, "games/")
}

func fileExists(fsys fs.FS, name string) bool {
	info, err := fs.Stat(fsys, name)
	return err == nil && !info.IsDir()
}

func fileModTime(fsys fs.FS, name string) time.Time {
	info, err := fs.Stat(fsys, name)
	if err != nil || info.IsDir() {
		return time.Time{}
	}
	return info.ModTime()
}

// passportDistHandler 用户通行证前端产物（没有构建时为 nil）。
func passportDistHandler() *distHandler {
	distFS, err := fs.Sub(root.EmbeddedPassportDist, "frontend/dist-passport")
	if err != nil {
		return nil
	}
	if info, err := fs.Stat(distFS, distIndex); err != nil || info.IsDir() {
		return nil
	}
	return newDistHandlerWithFS(distFS)
}

func embeddedDistFS() (fs.FS, bool) {
	distFS, err := fs.Sub(root.EmbeddedDist, distFolder)
	if err != nil {
		return nil, false
	}
	info, err := fs.Stat(distFS, distIndex)
	if err != nil || info.IsDir() {
		return nil, false
	}
	return distFS, true
}
