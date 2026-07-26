package file

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	v1 "momoko/api/gen/v1"
	"momoko/pkg/localfs"
)

// TestSystemStoreProtectsMomokoSecrets 整机来源必须够不到 momoko 自身的数据与配置目录。
//
// data/ 里是 SQLite 库（会话、令牌、来源凭据密文），configs/ 里是 auth.secret——
// 后者一旦泄露即可伪造 JWT 与全部预签名 URL，等于绕过整套鉴权。
func TestSystemStoreProtectsMomokoSecrets(t *testing.T) {
	// 先确认接线：保护清单必须覆盖 data/ 与 configs/。
	// （这一段不依赖工作目录，避免与 ProtectedPaths 的 OnceValue 缓存相互干扰。）
	guarded := map[string]bool{}
	for _, p := range ProtectedPaths() {
		guarded[p] = true
	}
	for _, want := range []string{DataDir, ConfigDir} {
		if !guarded[want] {
			t.Fatalf("保护清单缺少 %q，实际为 %v", want, ProtectedPaths())
		}
	}

	// 再验证机制：把临时目录当作 data/ 与 configs/，确认任何访问都被拒。
	work := t.TempDir()
	if real, err := filepath.EvalSymlinks(work); err == nil {
		work = real
	}
	dataDir := filepath.Join(work, "data")
	configDir := filepath.Join(work, "configs")
	mustMkdirAll(t, dataDir)
	mustMkdirAll(t, configDir)
	mustWriteFile(t, filepath.Join(dataDir, "momoko.db"), "SQLITE-DATA")
	mustWriteFile(t, filepath.Join(configDir, "config.yaml"), "auth:\n  secret: TOPSECRET\n")
	mustWriteFile(t, filepath.Join(work, "public.txt"), "ok")

	store, err := NewSystemStore(localfs.Deny(dataDir, configDir), localfs.WithMaxFileSize(MaxEditSize))
	if err != nil {
		t.Fatalf("NewSystemStore: %v", err)
	}
	ctx := context.Background()

	protected := []string{
		dataDir,
		filepath.Join(dataDir, "momoko.db"),
		configDir,
		filepath.Join(configDir, "config.yaml"),
	}
	for _, p := range protected {
		t.Run("deny "+p, func(t *testing.T) {
			if _, err := store.Stat(ctx, p); !errors.Is(err, localfs.ErrDenied) {
				t.Errorf("Stat(%q) 应返回 ErrDenied，实际: %v", p, err)
			}
			if rc, _, err := store.Open(ctx, p); err == nil {
				_ = rc.Close()
				t.Errorf("Open(%q) 竟然成功了", p)
			}
			if err := store.Write(ctx, p, strings.NewReader("pwned")); err == nil {
				t.Errorf("Write(%q) 竟然成功了", p)
			}
			if res := store.Delete(ctx, []string{p}); res[0].Success {
				t.Errorf("Delete(%q) 竟然成功了", p)
			}
		})
	}

	// 密钥文件必须原样存活。
	if data, err := os.ReadFile(filepath.Join(configDir, "config.yaml")); err != nil ||
		!strings.Contains(string(data), "TOPSECRET") {
		t.Fatalf("配置文件被破坏了: %q %v", data, err)
	}

	// 受保护目录不得出现在列表里，正常文件不受影响。
	items, err := store.List(ctx, work, SortByName, false)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	names := map[string]bool{}
	for _, e := range items {
		names[e.Name] = true
	}
	for _, hidden := range []string{"data", "configs"} {
		if names[hidden] {
			t.Errorf("受保护目录 %q 不应出现在列表中", hidden)
		}
	}
	if !names["public.txt"] {
		t.Error("普通文件被误伤，未出现在列表中")
	}
}

// TestScopedStoreCannotEscapeInstanceDir 实例受限来源必须锁死在实例目录内。
func TestScopedStoreCannotEscapeInstanceDir(t *testing.T) {
	base := t.TempDir()
	inst := filepath.Join(base, "servers", "inst-1")
	mustMkdirAll(t, inst)
	mustWriteFile(t, filepath.Join(inst, "config.yml"), "ok")
	mustWriteFile(t, filepath.Join(base, "outside.txt"), "SECRET")

	store, err := NewScopedStore(inst)
	if err != nil {
		t.Fatalf("NewScopedStore: %v", err)
	}
	ctx := context.Background()

	for _, p := range []string{
		"../outside.txt", `..\outside.txt`, "../../outside.txt",
		filepath.Join(base, "outside.txt"), "/etc/passwd", `C:\Windows\win.ini`,
	} {
		if _, err := store.Stat(ctx, p); err == nil {
			t.Errorf("Stat(%q) 越界成功了", p)
		}
		if err := store.Write(ctx, p, strings.NewReader("pwned")); err == nil {
			t.Errorf("Write(%q) 越界成功了", p)
		}
		if res := store.Delete(ctx, []string{p}); res[0].Success {
			t.Errorf("Delete(%q) 越界成功了", p)
		}
		if _, err := store.Rename(ctx, "config.yml", "../escaped.yml"); err == nil {
			t.Error("Rename 到上级目录成功了")
		}
	}
	if data, _ := os.ReadFile(filepath.Join(base, "outside.txt")); string(data) != "SECRET" {
		t.Fatal("实例目录之外的文件被改动了")
	}

	// 正常操作仍应可用。
	if err := store.Create(ctx, &v1.FileCreateItem{Path: "logs/latest.log", Content: []byte("hello")}); err != nil {
		t.Fatalf("实例目录内创建文件失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(inst, "logs", "latest.log")); err != nil {
		t.Fatalf("文件未创建: %v", err)
	}
}

func mustMkdirAll(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestTempFilesAreHiddenNotRelocated momoko 自己的中间产物不搬家、只隐藏：
// 与目标同目录才能保住同卷 rename，而列目录/搜索都看不见它们。
func TestTempFilesAreHiddenNotRelocated(t *testing.T) {
	// 远端来源的缓冲目录仍集中在 data/ 之下（那类收尾本就要整流推送，集中零代价）。
	if !strings.HasPrefix(UploadTempDir, DataDir+"/") {
		t.Errorf("%q 不在 %q 之下", UploadTempDir, DataDir)
	}
	guarded := false
	for _, p := range ProtectedPaths() {
		if p == DataDir {
			guarded = true
		}
	}
	if !guarded {
		t.Fatalf("保护清单未覆盖 %q", DataDir)
	}

	// 打包过程中的临时文件与目标同目录，但不出现在列表里。
	work := t.TempDir()
	fsys, err := localfs.Open(work, SystemStoreOptions()...)
	if err != nil {
		t.Fatal(err)
	}
	mustMkdirAll(t, filepath.Join(work, "src"))
	mustWriteFile(t, filepath.Join(work, "src", "a.txt"), strings.Repeat("x", 4096))
	if _, err := fsys.Compress([]string{"src"}, "out.zip"); err != nil {
		t.Fatalf("Compress: %v", err)
	}
	items, err := fsys.List(".", localfs.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range items {
		if strings.HasPrefix(e.Name, ".momoko-") {
			t.Errorf("中间产物出现在列表中: %q", e.Name)
		}
	}
	// 手工造一个缓冲文件，确认列目录与搜索都看不到它。
	mustWriteFile(t, filepath.Join(work, localfs.BufferPrefix+"abc.part"), "buffering")
	items, err = fsys.List(".", localfs.ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range items {
		if strings.HasPrefix(e.Name, localfs.BufferPrefix) {
			t.Errorf("上传缓冲文件出现在列表中: %q", e.Name)
		}
	}
	hits, err := fsys.Search(".", localfs.SearchOptions{Keywords: "momoko", Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Errorf("上传缓冲文件出现在搜索结果中: %+v", hits)
	}
}
