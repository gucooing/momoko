package download

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// slowServer 慢慢吐出 data，支持 Range（ranged=false 时忽略 Range 总是从头给），记下每次请求的 Range 头。
type slowServer struct {
	data    []byte
	ranged  bool
	chunk   int
	delay   time.Duration
	ranges  []string
	request atomic.Int32
}

func (s *slowServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.request.Add(1)
	s.ranges = append(s.ranges, r.Header.Get("Range"))
	if s.ranged {
		// ServeContent 自己处理 Range / If-Range，但一下子写完；这里用慢写的 writer 包一层
		w.Header().Set("ETag", `"v1"`)
		http.ServeContent(&slowWriter{ResponseWriter: w, chunk: s.chunk, delay: s.delay}, r, "f", time.Unix(0, 0), bytes.NewReader(s.data))
		return
	}
	w.Header().Set("Content-Length", strconv.Itoa(len(s.data)))
	sw := &slowWriter{ResponseWriter: w, chunk: s.chunk, delay: s.delay}
	_, _ = sw.Write(s.data)
}

type slowWriter struct {
	http.ResponseWriter
	chunk int
	delay time.Duration
}

func (w *slowWriter) Write(b []byte) (int, error) {
	written := 0
	for len(b) > 0 {
		n := min(w.chunk, len(b))
		m, err := w.ResponseWriter.Write(b[:n])
		written += m
		if err != nil {
			return written, err
		}
		if f, ok := w.ResponseWriter.(http.Flusher); ok {
			f.Flush()
		}
		b = b[n:]
		time.Sleep(w.delay)
	}
	return written, nil
}

func payload(n int) ([]byte, string) {
	data := make([]byte, n)
	for i := range data {
		data[i] = byte(i * 7)
	}
	sum := sha256.Sum256(data)
	return data, hex.EncodeToString(sum[:])
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatal("等太久了")
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestDownloadCompletesAndVerifies(t *testing.T) {
	data, sum := payload(300 << 10)
	srv := httptest.NewServer(&slowServer{data: data, ranged: true, chunk: 64 << 10})
	defer srv.Close()
	m := NewManager()
	dest := filepath.Join(t.TempDir(), "f")
	d, err := m.Start(context.Background(), Request{URL: srv.URL + "/pkg.zip", Dest: dest, Size: int64(len(data)), SHA256: sum})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, data) {
		t.Fatal("内容不对")
	}
	info := d.Info()
	if info.State != StateCompleted || info.Downloaded != int64(len(data)) || info.Name != "pkg.zip" {
		t.Fatalf("状态不对：%+v", info)
	}
}

func TestDownloadChecksumMismatch(t *testing.T) {
	data, _ := payload(10 << 10)
	srv := httptest.NewServer(&slowServer{data: data, ranged: true, chunk: 4 << 10})
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "f")
	d, _ := NewManager().Start(context.Background(), Request{URL: srv.URL, Dest: dest, SHA256: strings.Repeat("0", 64)})
	err := d.Wait(context.Background())
	if err == nil || !strings.Contains(err.Error(), "SHA256") {
		t.Fatalf("应该校验失败，得到 %v", err)
	}
	if _, err := os.Stat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("失败后应该删掉文件")
	}
}

func TestDownloadPauseResumeUsesRange(t *testing.T) {
	data, sum := payload(400 << 10)
	s := &slowServer{data: data, ranged: true, chunk: 16 << 10, delay: 20 * time.Millisecond}
	srv := httptest.NewServer(s)
	defer srv.Close()
	m := NewManager()
	dest := filepath.Join(t.TempDir(), "f")
	d, _ := m.Start(context.Background(), Request{URL: srv.URL, Dest: dest, SHA256: sum})

	waitFor(t, func() bool { return d.Info().Downloaded > 64<<10 })
	if err := m.Pause(d.ID()); err != nil {
		t.Fatal(err)
	}
	waitFor(t, func() bool { return d.Info().State == StatePaused })
	paused := d.Info().Downloaded
	time.Sleep(100 * time.Millisecond)
	if d.Info().Downloaded != paused {
		t.Fatal("暂停后还在下")
	}
	if !d.Info().Resumable {
		t.Fatal("服务器支持 Range，应该能续传")
	}
	if err := m.Resume(d.ID()); err != nil {
		t.Fatal(err)
	}
	if err := d.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(s.ranges) < 2 || !strings.HasPrefix(s.ranges[len(s.ranges)-1], "bytes=") {
		t.Fatalf("继续时应该带 Range，请求头：%v", s.ranges)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, data) {
		t.Fatal("续传后内容不对")
	}
}

func TestDownloadResumeWithoutRangeStartsOver(t *testing.T) {
	data, sum := payload(200 << 10)
	s := &slowServer{data: data, ranged: false, chunk: 16 << 10, delay: 20 * time.Millisecond}
	srv := httptest.NewServer(s)
	defer srv.Close()
	dest := filepath.Join(t.TempDir(), "f")
	d, _ := NewManager().Start(context.Background(), Request{URL: srv.URL, Dest: dest, SHA256: sum})
	waitFor(t, func() bool { return d.Info().Downloaded > 32<<10 })
	_ = d.Pause()
	waitFor(t, func() bool { return d.Info().State == StatePaused })
	if d.Info().Resumable {
		t.Fatal("服务器不支持 Range，不该标成能续传")
	}
	_ = d.Resume()
	if err := d.Wait(context.Background()); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(dest)
	if !bytes.Equal(got, data) {
		t.Fatal("从头下后内容不对")
	}
}

func TestDownloadCancel(t *testing.T) {
	data, _ := payload(400 << 10)
	srv := httptest.NewServer(&slowServer{data: data, ranged: true, chunk: 8 << 10, delay: 20 * time.Millisecond})
	defer srv.Close()
	m := NewManager()
	dest := filepath.Join(t.TempDir(), "f")
	d, _ := m.Start(context.Background(), Request{URL: srv.URL, Dest: dest})
	waitFor(t, func() bool { return d.Info().Downloaded > 0 })
	_ = d.Pause()
	if err := m.Cancel(d.ID()); err != nil {
		t.Fatal(err)
	}
	if err := d.Wait(context.Background()); !errors.Is(err, ErrCanceled) {
		t.Fatalf("应该返回 ErrCanceled，得到 %v", err)
	}
	if _, err := os.Stat(dest); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("取消后应该删掉已下的部分")
	}
	if n := m.RemoveFinished(nil); n != 1 || len(m.List()) != 0 {
		t.Fatal("清除已结束的记录不对")
	}
}

func TestDownloadWaitContextCancels(t *testing.T) {
	data, _ := payload(400 << 10)
	srv := httptest.NewServer(&slowServer{data: data, ranged: true, chunk: 8 << 10, delay: 20 * time.Millisecond})
	defer srv.Close()
	d, _ := NewManager().Start(context.Background(), Request{URL: srv.URL, Dest: filepath.Join(t.TempDir(), "f")})
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if err := d.Wait(ctx); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("得到 %v", err)
	}
	if d.State() != StateCanceled {
		t.Fatal("发起方不等了，下载应该被取消")
	}
}

func TestDownloadClientErrorFailsWithoutRetry(t *testing.T) {
	srv := httptest.NewServer(http.NotFoundHandler())
	defer srv.Close()
	d, _ := NewManager().Start(context.Background(), Request{URL: srv.URL, Dest: filepath.Join(t.TempDir(), "f"), Site: "测试站"})
	err := d.Wait(context.Background())
	if err == nil || !strings.Contains(err.Error(), "404") || d.Info().Retries != 0 {
		t.Fatalf("404 应该直接失败：%v %+v", err, d.Info())
	}
}

func TestParseContentRange(t *testing.T) {
	start, total, ok := parseContentRange("bytes 100-199/1000")
	if !ok || start != 100 || total != 1000 {
		t.Fatal(start, total, ok)
	}
	if _, total, ok := parseContentRange("bytes 5-9/*"); !ok || total != 0 {
		t.Fatal("总数未知时应为 0")
	}
	if _, _, ok := parseContentRange("items 1-2/3"); ok {
		t.Fatal("不认识的单位应该失败")
	}
}
