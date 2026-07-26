package localfs

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// fixture 搭一个「外面有秘密、里面是受限根」的两层目录：
//
//	base/
//	  secret.txt      ← 视图外，任何操作都不该碰到
//	  outside/leak    ← 同上
//	  root/           ← 受限视图的根
//	    sub/a.txt
func fixture(t *testing.T) (base string, fsys *FS) {
	t.Helper()
	base = t.TempDir()
	root := filepath.Join(base, "root")
	mustMkdir(t, filepath.Join(root, "sub"))
	mustMkdir(t, filepath.Join(base, "outside"))
	mustWrite(t, filepath.Join(root, "sub", "a.txt"), "alpha")
	mustWrite(t, filepath.Join(base, "secret.txt"), "TOP SECRET")
	mustWrite(t, filepath.Join(base, "outside", "leak"), "LEAK")

	fsys, err := Open(root)
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	return base, fsys
}

func mustMkdir(t *testing.T, p string) {
	t.Helper()
	if err := os.MkdirAll(p, 0o755); err != nil {
		t.Fatalf("MkdirAll %s: %v", p, err)
	}
}

func mustWrite(t *testing.T, p, content string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatalf("WriteFile %s: %v", p, err)
	}
}

// escapeVectors 是一组穿越向量，会被套用到每一个接受路径的操作上。
// %BASE% 在运行时替换为受限根之外的真实绝对路径。
func escapeVectors() []string {
	return []string{
		"../secret.txt",
		"..\\secret.txt",
		"sub/../../secret.txt",
		`sub\..\..\secret.txt`,
		"sub/./../../secret.txt",
		"../../../../../../../../etc/passwd",
		"%BASE%/secret.txt",
		"/etc/passwd",
		`C:\Windows\win.ini`,
		`\\?\C:\Windows\win.ini`,
		`\\127.0.0.1\C$\Windows\win.ini`,
		"..//secret.txt",
		"....//secret.txt",
		"sub/../..",
	}
}

func vectors(base string) []string {
	raw := escapeVectors()
	out := make([]string, 0, len(raw))
	for _, v := range raw {
		out = append(out, strings.ReplaceAll(v, "%BASE%", base))
	}
	return out
}

// TestScopedReadsCannotEscape 受限视图下所有读操作都不得读到根外内容。
func TestScopedReadsCannotEscape(t *testing.T) {
	base, fsys := fixture(t)
	for _, v := range vectors(base) {
		t.Run("Stat "+v, func(t *testing.T) {
			if e, err := fsys.Stat(v); err == nil {
				t.Fatalf("Stat(%q) 竟然成功了: %+v", v, e)
			}
		})
		t.Run("ReadFile "+v, func(t *testing.T) {
			data, err := fsys.ReadFile(v)
			if err == nil {
				t.Fatalf("ReadFile(%q) 竟然成功了，读到 %q", v, data)
			}
			if bytes.Contains(data, []byte("SECRET")) {
				t.Fatalf("ReadFile(%q) 泄露了根外内容", v)
			}
		})
		t.Run("OpenRead "+v, func(t *testing.T) {
			f, _, err := fsys.OpenRead(v)
			if err == nil {
				_ = f.Close()
				t.Fatalf("OpenRead(%q) 竟然成功了", v)
			}
		})
		t.Run("List "+v, func(t *testing.T) {
			if _, err := fsys.List(v, ListOptions{}); err == nil {
				t.Fatalf("List(%q) 竟然成功了", v)
			}
		})
		t.Run("DirStat "+v, func(t *testing.T) {
			if _, err := fsys.DirStat(v); err == nil {
				t.Fatalf("DirStat(%q) 竟然成功了", v)
			}
		})
	}
}

// TestScopedWritesCannotEscape 受限视图下所有写操作都不得在根外落地。
func TestScopedWritesCannotEscape(t *testing.T) {
	base, fsys := fixture(t)
	for _, v := range vectors(base) {
		t.Run("WriteFile "+v, func(t *testing.T) {
			if _, err := fsys.WriteFile(v, strings.NewReader("pwned")); err == nil {
				t.Fatalf("WriteFile(%q) 竟然成功了", v)
			}
		})
		t.Run("CreateFile "+v, func(t *testing.T) {
			if err := fsys.CreateFile(v, []byte("pwned")); err == nil {
				t.Fatalf("CreateFile(%q) 竟然成功了", v)
			}
		})
		t.Run("MkdirAll "+v, func(t *testing.T) {
			if err := fsys.MkdirAll(v); err == nil {
				t.Fatalf("MkdirAll(%q) 竟然成功了", v)
			}
		})
		t.Run("Remove "+v, func(t *testing.T) {
			res := fsys.Remove([]string{v})
			if len(res) != 1 || res[0].OK {
				t.Fatalf("Remove(%q) 竟然成功了: %+v", v, res)
			}
		})
		t.Run("Rename "+v, func(t *testing.T) {
			if _, err := fsys.Rename(v, "x.txt"); err == nil {
				t.Fatalf("Rename(%q) 竟然成功了", v)
			}
		})
		t.Run("CopyInto src "+v, func(t *testing.T) {
			res := fsys.CopyInto([]string{v}, ".")
			if len(res) != 1 || res[0].OK {
				t.Fatalf("CopyInto(%q) 竟然成功了: %+v", v, res)
			}
		})
		t.Run("CopyInto dst "+v, func(t *testing.T) {
			res := fsys.CopyInto([]string{"sub/a.txt"}, v)
			if len(res) != 1 || res[0].OK {
				t.Fatalf("CopyInto → %q 竟然成功了: %+v", v, res)
			}
		})
		t.Run("MoveInto dst "+v, func(t *testing.T) {
			res := fsys.MoveInto([]string{"sub/a.txt"}, v)
			if len(res) != 1 || res[0].OK {
				t.Fatalf("MoveInto → %q 竟然成功了: %+v", v, res)
			}
		})
		t.Run("Compress dst "+v, func(t *testing.T) {
			if _, err := fsys.Compress([]string{"sub"}, v); err == nil {
				t.Fatalf("Compress → %q 竟然成功了", v)
			}
		})
	}

	// 事后核对：根外目录必须完全未被触碰。
	if data, err := os.ReadFile(filepath.Join(base, "secret.txt")); err != nil || string(data) != "TOP SECRET" {
		t.Fatalf("根外文件被改写了: %q, %v", data, err)
	}
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]bool{"root": true, "outside": true, "secret.txt": true}
	for _, e := range entries {
		if !want[e.Name()] {
			t.Fatalf("受限根之外多出了 %q —— 发生了越界写入", e.Name())
		}
	}
}

// TestRenameCannotEscapeViaName 改名的新名称不得带任何路径成分。
func TestRenameCannotEscapeViaName(t *testing.T) {
	_, fsys := fixture(t)
	for _, name := range []string{
		"../escaped.txt", "..\\escaped.txt", "..", ".", "/abs.txt", `C:\abs.txt`,
		"a/b.txt", `a\b.txt`, "a.txt:evil", "NUL", "con.txt", "trail ", "trail.",
		"", "\x00bad", "ctrl\x01",
	} {
		if got, err := fsys.Rename("sub/a.txt", name); err == nil {
			t.Fatalf("Rename 到 %q 竟然成功了 → %s", name, got)
		}
	}
}

// TestSymlinkEscapeBlocked 视图内指向视图外的符号链接不得成为读写通道。
func TestSymlinkEscapeBlocked(t *testing.T) {
	base, fsys := fixture(t)
	link := filepath.Join(base, "root", "esc")
	if err := os.Symlink(base, link); err != nil {
		t.Skipf("当前环境无法创建符号链接: %v", err)
	}

	if data, err := fsys.ReadFile("esc/secret.txt"); err == nil {
		t.Fatalf("经符号链接读到了根外内容: %q", data)
	}
	if _, err := fsys.List("esc", ListOptions{}); err == nil {
		t.Fatal("经符号链接列出了根外目录")
	}
	if _, err := fsys.WriteFile("esc/pwned.txt", strings.NewReader("x")); err == nil {
		t.Fatal("经符号链接写入了根外")
	}
	if _, err := os.Stat(filepath.Join(base, "pwned.txt")); err == nil {
		t.Fatal("根外出现了 pwned.txt")
	}
	// 链接本身仍应可见，且被如实标记为链接（而不是假装成它指向的目录）。
	entries, err := fsys.List(".", ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	var found *Entry
	for _, e := range entries {
		if e.Name == "esc" {
			found = e
		}
	}
	if found == nil {
		t.Fatal("符号链接条目应当可见")
	}
	if !found.IsSymlink {
		t.Error("符号链接应被标记为 IsSymlink")
	}
	if found.IsDir {
		t.Error("符号链接不应被报告为目录（须按 lstat 语义）")
	}
}

// TestWindowsJunctionEscapeBlocked Windows 目录联接（junction）同样不得越界。
func TestWindowsJunctionEscapeBlocked(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("仅 Windows")
	}
	base, fsys := fixture(t)
	junction := filepath.Join(base, "root", "junc")
	if out, err := exeCombined("cmd", "/c", "mklink", "/J", junction, base); err != nil {
		t.Skipf("无法创建 junction: %v (%s)", err, out)
	}
	if data, err := fsys.ReadFile("junc/secret.txt"); err == nil {
		t.Fatalf("经 junction 读到了根外内容: %q", data)
	}
	if _, err := fsys.WriteFile("junc/pwned.txt", strings.NewReader("x")); err == nil {
		t.Fatal("经 junction 写入了根外")
	}
}

// TestDenyList 保护清单内的路径必须完全不可达，且在列表中不可见。
func TestDenyList(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "data"))
	mustWrite(t, filepath.Join(base, "data", "momoko.db"), "SQLITE")
	mustWrite(t, filepath.Join(base, "data-backup.txt"), "not protected")
	mustWrite(t, filepath.Join(base, "ok.txt"), "fine")

	fsys, err := Open(base, Deny(filepath.Join(base, "data")))
	if err != nil {
		t.Fatal(err)
	}

	for _, p := range []string{"data", "data/momoko.db", "./data/momoko.db"} {
		if _, err := fsys.ReadFile(p); !errors.Is(err, ErrDenied) && err == nil {
			t.Fatalf("%q 应当被保护清单拒绝", p)
		}
		if _, err := fsys.Stat(p); err == nil {
			t.Fatalf("Stat(%q) 应当被拒绝", p)
		}
		if _, err := fsys.WriteFile(p, strings.NewReader("x")); err == nil {
			t.Fatalf("WriteFile(%q) 应当被拒绝", p)
		}
		if res := fsys.Remove([]string{p}); res[0].OK {
			t.Fatalf("Remove(%q) 应当被拒绝", p)
		}
	}
	// 前缀相同但不是子路径的兄弟文件不应被误伤。
	if _, err := fsys.ReadFile("data-backup.txt"); err != nil {
		t.Fatalf("data-backup.txt 被保护清单误伤: %v", err)
	}
	// 受保护目录在列目录结果中不可见。
	entries, err := fsys.List(".", ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name == "data" {
			t.Fatal("受保护目录不应出现在列表中")
		}
	}
}

// TestDenyListSymlinkBypass 不得通过「允许目录内的链接指向受保护目录」绕过保护清单。
func TestDenyListSymlinkBypass(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "data"))
	mustWrite(t, filepath.Join(base, "data", "secret.key"), "KEY")
	if err := os.Symlink(filepath.Join(base, "data"), filepath.Join(base, "shortcut")); err != nil {
		t.Skipf("当前环境无法创建符号链接: %v", err)
	}
	fsys, err := Open(base, Deny(filepath.Join(base, "data")))
	if err != nil {
		t.Fatal(err)
	}
	if data, err := fsys.ReadFile("shortcut/secret.key"); err == nil {
		t.Fatalf("经符号链接绕过了保护清单: %q", data)
	}
}

// TestReadOnly 只读视图必须拒绝一切写操作。
func TestReadOnly(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, filepath.Join(base, "a.txt"), "data")
	fsys, err := Open(base, ReadOnly())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.ReadFile("a.txt"); err != nil {
		t.Fatalf("只读视图应当仍可读: %v", err)
	}
	checks := map[string]error{}
	_, checks["WriteFile"] = fsys.WriteFile("a.txt", strings.NewReader("x"))
	checks["CreateFile"] = fsys.CreateFile("b.txt", nil)
	checks["MkdirAll"] = fsys.MkdirAll("d")
	_, checks["Rename"] = fsys.Rename("a.txt", "c.txt")
	_, checks["Compress"] = fsys.Compress([]string{"a.txt"}, "")
	_, checks["Extract"] = fsys.Extract("a.txt", "")
	_, _, checks["WriteUploadPart"] = fsys.WriteUploadPart(Upload{ID: "abc", Name: "x", PartSize: MinPartSize}, 1, strings.NewReader(""))
	for name, err := range checks {
		if !errors.Is(err, ErrReadOnly) {
			t.Errorf("%s 在只读视图下应返回 ErrReadOnly，实际: %v", name, err)
		}
	}
	if res := fsys.Remove([]string{"a.txt"}); res[0].OK {
		t.Error("Remove 在只读视图下应失败")
	}
	if res := fsys.CopyInto([]string{"a.txt"}, "."); res[0].OK {
		t.Error("CopyInto 在只读视图下应失败")
	}
}

// TestRootScopeProtected 不得删除或重命名视图根自身。
func TestRootScopeProtected(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, filepath.Join(base, "a.txt"), "x")
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []string{"", ".", "./", base} {
		if res := fsys.Remove([]string{p}); res[0].OK {
			t.Fatalf("Remove(%q) 删掉了视图根", p)
		}
		if _, err := fsys.Rename(p, "renamed"); err == nil {
			t.Fatalf("Rename(%q) 改掉了视图根", p)
		}
	}
	if _, err := os.Stat(base); err != nil {
		t.Fatalf("视图根被破坏: %v", err)
	}
}

// TestSubCannotWidenViaSymlink 收窄视图不得因目标是符号链接而反向扩大范围。
//
// FS.Sub 若只按词法路径开新视图，一个指向视图外的链接就会把「子视图」开到外面去。
func TestSubCannotWidenViaSymlink(t *testing.T) {
	base, fsys := fixture(t)
	link := filepath.Join(base, "root", "escdir")
	if err := os.Symlink(filepath.Join(base, "outside"), link); err != nil {
		t.Skipf("当前环境无法创建符号链接: %v", err)
	}
	sub, err := fsys.Sub("escdir")
	if err == nil {
		if data, readErr := sub.ReadFile("leak"); readErr == nil {
			t.Fatalf("子视图被开到了视图外，读到 %q", data)
		}
		t.Fatalf("Sub 到越界链接应当失败，实际得到根为 %q 的视图", sub.Base())
	}
}

// TestSubRejectsFileAndEscape Sub 只接受视图内的目录。
func TestSubRejectsFileAndEscape(t *testing.T) {
	base, fsys := fixture(t)
	if _, err := fsys.Sub("sub/a.txt"); !errors.Is(err, ErrNotDir) {
		t.Errorf("Sub 到文件应返回 ErrNotDir，实际: %v", err)
	}
	for _, v := range vectors(base) {
		if _, err := fsys.Sub(v); err == nil {
			t.Errorf("Sub(%q) 竟然成功了", v)
		}
	}
	if _, err := fsys.Sub("nope"); err == nil {
		t.Error("Sub 到不存在的目录应当失败")
	}
}

// TestCreateFileValidatesParentNames 自动创建的中间目录名同样要过名称校验。
func TestCreateFileValidatesParentNames(t *testing.T) {
	_, fsys := fixture(t)
	// 结尾点/空格的分段在 Windows 上更早一步被 cleanRel 拒掉（路径规范化会错位），
	// 其余非法名由 ValidateName 拒掉；两者都必须拒绝。
	for _, p := range []string{"NUL/x.txt", "a/COM1/x.txt", "trail /x.txt", "trail./x.txt"} {
		err := fsys.CreateFile(p, []byte("x"))
		if !errors.Is(err, ErrInvalidName) && !errors.Is(err, ErrInvalidPath) {
			t.Errorf("CreateFile(%q) 应因中间目录名非法而拒绝，实际: %v", p, err)
		}
	}
}
