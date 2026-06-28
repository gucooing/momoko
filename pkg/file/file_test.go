package file

import (
	"context"
	"errors"
	"io"
	stdhttp "net/http"
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/go-kratos/kratos/v2/transport/http"

	v1 "momoko/api/gen/v1"
)

func Test_ResolveRealPath(t *testing.T) {
	basePath := t.TempDir()
	f, err := NewFileOper(basePath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.ResolveRealPath(filepath.Dir(basePath)); err == nil {
		t.Errorf("ResolveRealPath 失效")
		return
	}
}

func Test_ListDir(t *testing.T) {
	f, err := NewFileOper("")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = f.ListDir("", v1.FileSortField_FILE_SORT_FIELD_NAME, true); err != nil {
		t.Fatal(err)
	}
}

func Test_Rename(t *testing.T) {
	basePath := t.TempDir()
	f, err := NewFileOper(basePath)
	if err != nil {
		t.Fatal(err)
	}

	sourcePath := filepath.Join(basePath, "old.txt")
	if err = os.WriteFile(sourcePath, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	targetPath, err := f.Rename("old.txt", "new.txt")
	if err != nil {
		t.Fatal(err)
	}
	if targetPath != filepath.Join(basePath, "new.txt") {
		t.Fatalf("unexpected target path: %s", targetPath)
	}

	if _, err = f.Rename("", "out"); err == nil {
		t.Fatal("expected base path rename to be rejected")
	}
	if _, err = f.Rename("new.txt", "../bad.txt"); err == nil {
		t.Fatal("expected path name to be rejected")
	}
}

func Test_Download(t *testing.T) {
	t.Skip("manual HTTP server smoke test; skipped in automated test runs")

	f, err := NewFileOper("")
	if err != nil {
		t.Fatal(err)
	}
	srv := http.NewServer()
	testCtx := context.Background()
	router := srv.Route("/")
	router.GET("/d", func(ctx http.Context) error {
		path, err := f.ResolveRealPath(ctx.Query().Get("path"))
		if err != nil {
			return err
		}
		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return err
		}

		w := ctx.Response()

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", `attachment; filename="`+info.Name()+`"`)
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))

		_, err = io.Copy(w, f)
		if err != nil {
			return err
		}
		srv.Stop(testCtx)
		return nil
	})

	if err := srv.Start(testCtx); err != nil {
		return
	}
	t.Log("Download End")
}

func Test_Download_Speed(t *testing.T) {
	t.Skip("manual large-file download smoke test; skipped in automated test runs")

	srv := http.NewServer()
	testCtx := context.Background()
	router := srv.Route("/")
	router.GET("/d/speed", func(ctx http.Context) error {
		f, err := os.Open("big.test")
		if err != nil {
			if os.IsNotExist(err) {
				f, err = os.Create("big.test")
				if err != nil {
					return err
				}
				if err = f.Truncate(10 * 1024 * 1024 * 1024); err != nil {
					f.Close()
					return err
				}
				f.Close()

				f, err = os.Open("big.test")
				if err != nil {
					return err
				}
			} else {
				return err
			}
		}
		defer f.Close()

		info, err := f.Stat()
		if err != nil {
			return err
		}

		w := ctx.Response()
		r := ctx.Request()

		w.Header().Set("Content-Disposition", `attachment; filename="`+info.Name()+`"`)

		stdhttp.ServeContent(w, r, info.Name(), info.ModTime(), f)
		return nil
	})

	go func() {
		if err := srv.Start(testCtx); err != nil {
			return
		}
	}()
	<-testCtx.Done()
	srv.Close()
	t.Log("Download_Speed End")
}

func Test_SaveFile(t *testing.T) {
	t.Skip("manual multipart upload smoke test; skipped in automated test runs")

	f, err := NewFileOper("")
	if err != nil {
		t.Fatal(err)
	}
	srv := http.NewServer()
	testCtx := context.Background()
	router := srv.Route("/")
	router.POST("/put", func(ctx http.Context) error {
		path, err := f.ResolveRealPath("./")
		if err != nil {
			return err
		}
		r := ctx.Request()
		w := ctx.Response()

		mr, err := r.MultipartReader()
		if err != nil {
			return err
		}
		wrote := false
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			dstPath := filepath.Join(path, part.FileName())
			dst, err := os.Create(dstPath)
			if err != nil {
				return err
			}
			_, copyErr := io.Copy(dst, part)
			closeErr1 := dst.Close()
			closeErr2 := part.Close()

			if copyErr != nil {
				_ = os.Remove(dstPath)
				return copyErr
			}
			if closeErr1 != nil {
				_ = os.Remove(dstPath)
				return closeErr1
			}
			if closeErr2 != nil {
				_ = os.Remove(dstPath)
				return closeErr2
			}
			wrote = true
		}

		if !wrote {
			return errors.New("没有找到上传文件")
		}
		w.WriteHeader(200)
		return nil
	})
	go func() {
		if err := srv.Start(testCtx); err != nil {
			return
		}
	}()
	<-testCtx.Done()
	srv.Close()
	t.Log("SaveFile End")
}
