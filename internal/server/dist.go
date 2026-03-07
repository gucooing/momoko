package server

import (
	"bytes"
	"io/fs"
	"net/http"
	"path"
	"strings"
	"time"

	root "momoko"

	httpm "github.com/go-kratos/kratos/v2/transport/http"
)

const (
	apiPrefix  = "/api/v1"
	distIndex  = "index.html"
	distFolder = "dist"
)

type distHandler struct {
	files      fs.FS
	fileServer http.Handler
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
	return &distHandler{
		files:      distFS,
		fileServer: http.FileServer(http.FS(distFS)),
	}, true
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
		h.serveIndex(w, r)
		return true
	}

	filePath := strings.TrimPrefix(cleanPath, "/")
	if fileExists(h.files, filePath) {
		h.fileServer.ServeHTTP(w, r)
		return true
	}

	h.serveIndex(w, r)
	return true
}

func (h *distHandler) serveIndex(w http.ResponseWriter, r *http.Request) {
	data, err := fs.ReadFile(h.files, distIndex)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	http.ServeContent(w, r, distIndex, fileModTime(h.files, distIndex), bytes.NewReader(data))
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
