package server

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
)

func TestDistHandlerServesPrecompressedBrotliAsset(t *testing.T) {
	brotliData := []byte("precompressed-brotli")
	gzipData := []byte("precompressed-gzip")
	handler := newDistHandlerWithFS(fstest.MapFS{
		distIndex:                 &fstest.MapFile{Data: []byte("<!doctype html><html><body>ok</body></html>")},
		"assets/app-abc123.js":    &fstest.MapFile{Data: []byte("console.log('momoko');")},
		"assets/app-abc123.js.br": &fstest.MapFile{Data: brotliData},
		"assets/app-abc123.js.gz": &fstest.MapFile{Data: gzipData},
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/app-abc123.js", nil)
	req.Header.Set("Accept-Encoding", "gzip, br")
	rec := httptest.NewRecorder()

	if !handler.tryServe(rec, req) {
		t.Fatal("expected static asset to be served")
	}

	resp := rec.Result()
	if got := resp.Header.Get("Content-Encoding"); got != distEncodingBrotli {
		t.Fatalf("expected brotli encoding, got %q", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != distCacheControlImmutable {
		t.Fatalf("expected immutable cache control, got %q", got)
	}
	if got := resp.Header.Get("Vary"); !strings.Contains(got, "Accept-Encoding") {
		t.Fatalf("expected Vary header to contain Accept-Encoding, got %q", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if string(body) != string(brotliData) {
		t.Fatalf("unexpected precompressed body: %q", string(body))
	}
}

func TestDistHandlerServesPrecompressedGzipAsset(t *testing.T) {
	gzipData := []byte("precompressed-gzip")
	handler := newDistHandlerWithFS(fstest.MapFS{
		distIndex:                 &fstest.MapFile{Data: []byte("<!doctype html><html><body>ok</body></html>")},
		"assets/app-abc123.js":    &fstest.MapFile{Data: []byte("console.log('momoko');")},
		"assets/app-abc123.js.gz": &fstest.MapFile{Data: gzipData},
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/app-abc123.js", nil)
	req.Header.Set("Accept-Encoding", "gzip")
	rec := httptest.NewRecorder()

	if !handler.tryServe(rec, req) {
		t.Fatal("expected static asset to be served")
	}

	resp := rec.Result()
	if got := resp.Header.Get("Content-Encoding"); got != distEncodingGzip {
		t.Fatalf("expected gzip encoding, got %q", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if string(body) != string(gzipData) {
		t.Fatalf("unexpected precompressed body: %q", string(body))
	}
}

func TestDistHandlerServesCompressedIndexForSPAFallback(t *testing.T) {
	brotliData := []byte("compressed-index")
	handler := newDistHandlerWithFS(fstest.MapFS{
		distIndex:              &fstest.MapFile{Data: []byte("<!doctype html><html><body>spa</body></html>")},
		"index.html.br":        &fstest.MapFile{Data: brotliData},
		"assets/app-abc123.js": &fstest.MapFile{Data: []byte("console.log('momoko');")},
	})

	req := httptest.NewRequest(http.MethodGet, "/dashboard/profile", nil)
	req.Header.Set("Accept-Encoding", "br")
	rec := httptest.NewRecorder()

	if !handler.tryServe(rec, req) {
		t.Fatal("expected SPA fallback to be served")
	}

	resp := rec.Result()
	if got := resp.Header.Get("Content-Encoding"); got != distEncodingBrotli {
		t.Fatalf("expected brotli encoding, got %q", got)
	}
	if got := resp.Header.Get("Cache-Control"); got != distCacheControlNoCache {
		t.Fatalf("expected no-cache for index, got %q", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if string(body) != string(brotliData) {
		t.Fatalf("unexpected index body: %q", string(body))
	}
}

func TestDistHandlerSkipsCompressedVariantForRangeRequests(t *testing.T) {
	rawJS := "console.log('momoko');"
	handler := newDistHandlerWithFS(fstest.MapFS{
		distIndex:                 &fstest.MapFile{Data: []byte("<html></html>")},
		"assets/app-abc123.js":    &fstest.MapFile{Data: []byte(rawJS)},
		"assets/app-abc123.js.br": &fstest.MapFile{Data: []byte("precompressed-brotli")},
		"assets/app-abc123.js.gz": &fstest.MapFile{Data: []byte("precompressed-gzip")},
	})

	req := httptest.NewRequest(http.MethodGet, "/assets/app-abc123.js", nil)
	req.Header.Set("Accept-Encoding", "br, gzip")
	req.Header.Set("Range", "bytes=0-6")
	rec := httptest.NewRecorder()

	if !handler.tryServe(rec, req) {
		t.Fatal("expected ranged asset to be served")
	}

	resp := rec.Result()
	if got := resp.Header.Get("Content-Encoding"); got != "" {
		t.Fatalf("expected no compressed encoding for range request, got %q", got)
	}
	if got := resp.StatusCode; got != http.StatusPartialContent {
		t.Fatalf("expected partial content, got %d", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("read response body: %v", err)
	}
	if string(body) != rawJS[:7] {
		t.Fatalf("unexpected partial body: %q", string(body))
	}
}

func TestDistHandlerBypassesAPIRequests(t *testing.T) {
	handler := newDistHandlerWithFS(fstest.MapFS{
		distIndex: &fstest.MapFile{Data: []byte("<html></html>")},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/instance/list", nil)
	rec := httptest.NewRecorder()

	if handler.tryServe(rec, req) {
		t.Fatal("expected api request to bypass dist handler")
	}
}
