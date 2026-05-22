package server

import (
	"bytes"
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
	files      fs.FS
	fileServer http.Handler
	distFiles  map[string]distFile
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

func newDistHandlerWithFS(distFS fs.FS) *distHandler {
	return &distHandler{
		files:      distFS,
		fileServer: http.FileServer(http.FS(distFS)),
		distFiles:  buildDistFiles(distFS),
	}
}

func (h *distHandler) tryServe(w http.ResponseWriter, r *http.Request) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}
	if r.URL == nil {
		return false
	}
	if r.URL.Path == apiPrefix || strings.HasPrefix(r.URL.Path, apiPrefix+"/") {
		return false
	}

	cleanPath := path.Clean("/" + strings.TrimPrefix(r.URL.Path, "/"))
	if cleanPath == "/" {
		return h.serveDistFile(w, r, distIndex)
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

	h.fileServer.ServeHTTP(w, requestWithPath(r, "/"+item.rawPath))
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
	return strings.HasPrefix(filePath, "assets/")
}

func requestWithPath(r *http.Request, cleanPath string) *http.Request {
	clone := r.Clone(r.Context())
	if r.URL == nil {
		return clone
	}

	urlCopy := *r.URL
	urlCopy.Path = cleanPath
	clone.URL = &urlCopy
	return clone
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
