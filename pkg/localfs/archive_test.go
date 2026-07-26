package localfs

import (
	"archive/zip"
	"bytes"
	"errors"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// exeCombined 跑一条外部命令并返回合并输出（仅测试辅助用，如 Windows 的 mklink）。
func exeCombined(name string, args ...string) (string, error) {
	out, err := exec.Command(name, args...).CombinedOutput()
	return string(out), err
}

// writeZipFile 用给定条目造一个 zip 文件。name 以 "/" 结尾表示目录条目。
func writeZipFile(t *testing.T, target string, entries map[string]string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, content := range entries {
		if strings.HasSuffix(name, "/") {
			if _, err := zw.CreateHeader(&zip.FileHeader{Name: name}); err != nil {
				t.Fatal(err)
			}
			continue
		}
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, target, buf.String())
}

// TestExtractRejectsZipSlip 各种越界条目名都必须被拒，且根外不得出现任何文件。
func TestExtractRejectsZipSlip(t *testing.T) {
	slips := []string{
		"../escape.txt",
		"..\\escape.txt",
		"a/../../escape.txt",
		"/etc/cron.d/evil",
		`C:\Windows\evil.txt`,
		"a/b/../../../escape.txt",
	}
	for _, entryName := range slips {
		t.Run(entryName, func(t *testing.T) {
			base, fsys := fixture(t)
			archive := filepath.Join(base, "root", "bad.zip")
			writeZipFile(t, archive, map[string]string{entryName: "PWNED"})

			if _, err := fsys.Extract("bad.zip", "out"); err == nil {
				t.Fatalf("Extract 接受了越界条目 %q", entryName)
			}
			if _, err := os.Stat(filepath.Join(base, "escape.txt")); err == nil {
				t.Fatal("根外出现了 escape.txt")
			}
		})
	}
}

// TestExtractRejectsDangerousNames 条目名含保留设备名/备用数据流/结尾点空格时拒绝。
func TestExtractRejectsDangerousNames(t *testing.T) {
	for _, entryName := range []string{
		"NUL", "con.txt", "a/COM1", "file.txt:evil", "trailing ", "trailing.", "\x01ctrl",
	} {
		t.Run(entryName, func(t *testing.T) {
			base, fsys := fixture(t)
			archive := filepath.Join(base, "root", "bad.zip")
			writeZipFile(t, archive, map[string]string{entryName: "x"})
			if _, err := fsys.Extract("bad.zip", "out"); !errors.Is(err, ErrBadArchive) {
				t.Fatalf("Extract(%q) 应返回 ErrBadArchive，实际: %v", entryName, err)
			}
		})
	}
}

// TestExtractRejectsSymlinkEntry 含符号链接条目的压缩包整体拒绝。
func TestExtractRejectsSymlinkEntry(t *testing.T) {
	base, fsys := fixture(t)
	archive := filepath.Join(base, "root", "link.zip")

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	hdr := &zip.FileHeader{Name: "evil-link"}
	hdr.SetMode(fs.ModeSymlink | 0o777)
	w, err := zw.CreateHeader(hdr)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write([]byte(filepath.Join(base, "secret.txt"))); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, archive, buf.String())

	if _, err := fsys.Extract("link.zip", "out"); !errors.Is(err, ErrBadArchive) {
		t.Fatalf("含符号链接条目应返回 ErrBadArchive，实际: %v", err)
	}
}

// TestExtractRejectsZipBomb 高压缩比 / 超总量的压缩包必须被拦下。
func TestExtractRejectsZipBomb(t *testing.T) {
	base, fsys := fixture(t)
	archive := filepath.Join(base, "root", "bomb.zip")
	// 8 MiB 的零字节能压到极小，压缩比远超默认阈值。
	writeZipFile(t, archive, map[string]string{"bomb.bin": strings.Repeat("\x00", 8<<20)})

	if _, err := fsys.Extract("bomb.zip", "out"); !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("高压缩比条目应返回 ErrArchiveLimit，实际: %v", err)
	}

	// 总量闸门：把上限压到 1 KiB，正常小文件也应被拒。
	tight, err := Open(filepath.Join(base, "root"), WithArchiveLimits(0, 1<<10, 1_000_000))
	if err != nil {
		t.Fatal(err)
	}
	writeZipFile(t, filepath.Join(base, "root", "big.zip"), map[string]string{
		"big.bin": strings.Repeat("A", 4096),
	})
	if _, err := tight.Extract("big.zip", "out2"); !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("超总量应返回 ErrArchiveLimit，实际: %v", err)
	}
}

// TestExtractRejectsTooManyEntries 条目数闸门。
func TestExtractRejectsTooManyEntries(t *testing.T) {
	base, _ := fixture(t)
	root := filepath.Join(base, "root")
	entries := make(map[string]string, 10)
	for i := range 10 {
		entries[string(rune('a'+i))+".txt"] = "x"
	}
	writeZipFile(t, filepath.Join(root, "many.zip"), entries)

	fsys, err := Open(root, WithArchiveLimits(3, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.Extract("many.zip", "out"); !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("超条目数应返回 ErrArchiveLimit，实际: %v", err)
	}
}

// TestCompressAndExtractRoundTrip 正常打包/解包应完整往返。
func TestCompressAndExtractRoundTrip(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "src", "nested"))
	mustWrite(t, filepath.Join(base, "src", "a.txt"), "alpha")
	mustWrite(t, filepath.Join(base, "src", "nested", "b.txt"), "bravo")
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}

	zipPath, err := fsys.Compress([]string{"src"}, "")
	if err != nil {
		t.Fatalf("Compress: %v", err)
	}
	if !strings.HasSuffix(zipPath, ".zip") {
		t.Fatalf("默认压缩目标应以 .zip 结尾: %s", zipPath)
	}
	if _, err := os.Stat(zipPath); err != nil {
		t.Fatalf("压缩包未生成: %v", err)
	}

	outDir, err := fsys.Extract(zipPath, "")
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	for rel, want := range map[string]string{
		filepath.Join("src", "a.txt"):           "alpha",
		filepath.Join("src", "nested", "b.txt"): "bravo",
	} {
		got, err := os.ReadFile(filepath.Join(outDir, rel))
		if err != nil {
			t.Fatalf("解压结果缺少 %s: %v", rel, err)
		}
		if string(got) != want {
			t.Fatalf("%s 内容不符: 得到 %q 期望 %q", rel, got, want)
		}
	}
}

// TestCompressRejectsTargetInsideSource 压缩包不得生成在被压缩目录内部。
func TestCompressRejectsTargetInsideSource(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "src"))
	mustWrite(t, filepath.Join(base, "src", "a.txt"), "x")
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.Compress([]string{"src"}, "src/out.zip"); err == nil {
		t.Fatal("压缩包生成在源目录内部时应当拒绝")
	}
}
