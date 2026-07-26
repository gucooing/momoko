package localfs

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestValidateName(t *testing.T) {
	bad := []string{
		"", ".", "..",
		"a/b", `a\b`, "/abs", `C:\abs`,
		"a.txt:evil", "a.txt::$DATA",
		"NUL", "nul", "nul.txt", "CON", "com1", "LPT9", "aux.log", "CONIN$",
		"trailing ", "trailing.", " leading",
		"ctrl\x01name", "nul\x00byte",
		"lt<", "gt>", `quote"`, "pipe|", "quest?", "star*",
		strings.Repeat("x", 256),
	}
	for _, n := range bad {
		if err := ValidateName(n); err == nil {
			t.Errorf("ValidateName(%q) 应当拒绝", n)
		}
	}
	good := []string{
		"a.txt", "..hidden", "..x", "a-b_c.tar.gz", "中文名.txt", "with space.txt",
		"NULL.txt", "console.log", "comrade", strings.Repeat("x", 255),
	}
	for _, n := range good {
		if err := ValidateName(n); err != nil {
			t.Errorf("ValidateName(%q) 应当接受，实际: %v", n, err)
		}
	}
}

func TestValidateToken(t *testing.T) {
	for _, s := range []string{"", "../x", "a/b", "a.b", "a b", "a:b", strings.Repeat("a", 65)} {
		if err := ValidateToken(s); err == nil {
			t.Errorf("ValidateToken(%q) 应当拒绝", s)
		}
	}
	for _, s := range []string{"abc", "ABC-123_x", "01H8XYZ"} {
		if err := ValidateToken(s); err != nil {
			t.Errorf("ValidateToken(%q) 应当接受，实际: %v", s, err)
		}
	}
}

func TestSafeName(t *testing.T) {
	// 分隔符折叠为 '_' 后再剪掉首尾的 '.'/'_'/空格，所以路径成分被彻底抹平。
	cases := map[string]string{
		"../../etc/passwd": "etc_passwd",
		"normal-id":        "normal-id",
		"":                 "fallback",
		"...":              "fallback",
		"NUL":              "fallback",
		"a b":              "a_b",
		"用户":               "fallback",
	}
	for in, want := range cases {
		if got := SafeName(in, "fallback"); got != want {
			t.Errorf("SafeName(%q) = %q，期望 %q", in, got, want)
		}
		// 无论输入什么，结果都必须是一个合法的单层名称。
		if err := ValidateName(SafeName(in, "fallback")); err != nil {
			t.Errorf("SafeName(%q) 的结果不是合法名称: %v", in, err)
		}
	}
}

func TestCleanRelRejectsDotDot(t *testing.T) {
	for _, p := range []string{"..", "../a", "a/../..", `a\..\..`, "a/../../b"} {
		if _, err := cleanRel(p); !errors.Is(err, ErrTraversal) {
			t.Errorf("cleanRel(%q) 应返回 ErrTraversal，实际: %v", p, err)
		}
	}
	cases := map[string]string{
		"":         ".",
		".":        ".",
		"./a":      "a",
		"a//b":     "a/b",
		`a\b`:      "a/b",
		"a/./b/":   "a/b",
		"/leading": "leading",
		"a/b/c":    "a/b/c",
	}
	for in, want := range cases {
		got, err := cleanRel(in)
		if err != nil {
			t.Errorf("cleanRel(%q) 报错: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("cleanRel(%q) = %q，期望 %q", in, got, want)
		}
	}
	if _, err := cleanRel("a\x00b"); err == nil {
		t.Error("cleanRel 应拒绝 NUL 字节")
	}
}

// TestRoundTripEntryPath 列表返回的 Entry.Path 必须能原样回传继续操作。
func TestRoundTripEntryPath(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "dir"))
	mustWrite(t, filepath.Join(base, "dir", "a.txt"), "alpha")
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := fsys.List(".", ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "dir" {
		t.Fatalf("列表结果异常: %+v", entries)
	}
	sub, err := fsys.List(entries[0].Path, ListOptions{})
	if err != nil {
		t.Fatalf("用 Entry.Path 回传列目录失败: %v", err)
	}
	if len(sub) != 1 || sub[0].Name != "a.txt" {
		t.Fatalf("子目录列表异常: %+v", sub)
	}
	data, err := fsys.ReadFile(sub[0].Path)
	if err != nil || string(data) != "alpha" {
		t.Fatalf("用 Entry.Path 回传读文件失败: %q %v", data, err)
	}
}

func TestCRUDAndListing(t *testing.T) {
	base := t.TempDir()
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}

	if err := fsys.MkdirAll("a/b"); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := fsys.CreateFile("a/b/f.txt", []byte("hello")); err != nil {
		t.Fatalf("CreateFile: %v", err)
	}
	// 重复创建必须失败，而不是静默覆盖。
	if err := fsys.CreateFile("a/b/f.txt", []byte("clobber")); !errors.Is(err, ErrExist) {
		t.Fatalf("重复 CreateFile 应返回 ErrExist，实际: %v", err)
	}
	if data, _ := fsys.ReadFile("a/b/f.txt"); string(data) != "hello" {
		t.Fatalf("内容被覆盖了: %q", data)
	}

	if _, err := fsys.WriteFile("a/b/f.txt", strings.NewReader("world")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	if data, _ := fsys.ReadFile("a/b/f.txt"); string(data) != "world" {
		t.Fatalf("WriteFile 未生效: %q", data)
	}
	// 原子写入不得留下临时文件。
	entries, err := fsys.List("a/b", ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name, tempPrefix) {
			t.Fatalf("残留了临时文件 %q", e.Name)
		}
	}

	newPath, err := fsys.Rename("a/b/f.txt", "g.txt")
	if err != nil {
		t.Fatalf("Rename: %v", err)
	}
	if filepath.Base(newPath) != "g.txt" {
		t.Fatalf("Rename 返回路径异常: %s", newPath)
	}

	st, err := fsys.DirStat("a/b")
	if err != nil {
		t.Fatal(err)
	}
	if st.Files != 1 || st.Dirs != 0 {
		t.Fatalf("DirStat 计数异常: %+v", st)
	}
	// 受限视图的展示路径应是相对形态，不泄露真实目录结构。
	if filepath.IsAbs(st.Path) {
		t.Errorf("受限视图 DirStat.Path 不应是绝对路径: %s", st.Path)
	}

	if res := fsys.Remove([]string{"a/b/g.txt"}); !res[0].OK {
		t.Fatalf("Remove: %+v", res)
	}
	if fsys.Exists("a/b/g.txt") {
		t.Fatal("删除后仍然存在")
	}
}

func TestCopyAndMove(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "src", "nested"))
	mustMkdir(t, filepath.Join(base, "dst"))
	mustWrite(t, filepath.Join(base, "src", "a.txt"), "alpha")
	mustWrite(t, filepath.Join(base, "src", "nested", "b.txt"), "bravo")
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}

	res := fsys.CopyInto([]string{"src"}, "dst")
	if len(res) != 1 || !res[0].OK {
		t.Fatalf("CopyInto: %+v", res)
	}
	if got, err := os.ReadFile(filepath.Join(base, "dst", "src", "nested", "b.txt")); err != nil || string(got) != "bravo" {
		t.Fatalf("递归复制失败: %q %v", got, err)
	}

	// 同目录内复制应自动加 -copy 后缀。
	res = fsys.CopyInto([]string{"src/a.txt"}, "src")
	if !res[0].OK {
		t.Fatalf("同目录复制失败: %+v", res)
	}
	if !fsys.Exists("src/a-copy.txt") {
		t.Fatal("未生成 a-copy.txt")
	}

	// 目录不能复制进自己内部。
	res = fsys.CopyInto([]string{"src"}, "src/nested")
	if res[0].OK {
		t.Fatal("目录复制进自身内部应当失败")
	}

	res = fsys.MoveInto([]string{"src/a.txt"}, "dst")
	if !res[0].OK {
		t.Fatalf("MoveInto: %+v", res)
	}
	if fsys.Exists("src/a.txt") {
		t.Fatal("移动后源仍存在")
	}
	if !fsys.Exists("dst/a.txt") {
		t.Fatal("移动后目标不存在")
	}
}

func TestReadFileSizeLimit(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, filepath.Join(base, "big.txt"), strings.Repeat("x", 2048))
	fsys, err := Open(base, WithMaxFileSize(1024))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.ReadFile("big.txt"); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("超限读取应返回 ErrTooLarge，实际: %v", err)
	}
	if _, err := fsys.WriteFile("out.txt", strings.NewReader(strings.Repeat("y", 2048))); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("超限写入应返回 ErrTooLarge，实际: %v", err)
	}
	if fsys.Exists("out.txt") {
		t.Fatal("超限写入不应留下文件")
	}
}

func TestSearchLimits(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "deep"))
	for i := range 20 {
		mustWrite(t, filepath.Join(base, "deep", string(rune('a'+i))+"-match.txt"), "x")
	}
	fsys, err := Open(base, WithSearchLimits(5, 0))
	if err != nil {
		t.Fatal(err)
	}
	hits, err := fsys.Search(".", SearchOptions{Keywords: "match", Recursive: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) > 5 {
		t.Fatalf("搜索结果 %d 条，超过上限 5", len(hits))
	}
	// 非递归搜索不应下钻。
	hits, err = fsys.Search(".", SearchOptions{Keywords: "match", Recursive: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(hits) != 0 {
		t.Fatalf("非递归搜索不该命中子目录内容，得到 %d 条", len(hits))
	}
}

// ---- 分片上传 ----

func newUpload(dir, name string, size uint64) Upload {
	ps := CalcPartSize(size)
	return Upload{ID: "test-upload-1", Dir: dir, Name: name, Size: size, PartSize: ps, Parts: CalcParts(size, ps)}
}

func TestUploadValidateRejectsDangerousName(t *testing.T) {
	for _, name := range []string{
		"../../../../evil.txt", "..\\evil.txt", "/etc/cron.d/evil", `C:\evil.txt`,
		"a/b.txt", "NUL", "evil.txt:ads", "trail ", "", "..",
	} {
		u := newUpload(".", name, 10)
		if err := u.Validate(); err == nil {
			t.Errorf("上传目标文件名 %q 应被拒绝", name)
		}
	}
}

func TestUploadPartAndCommit(t *testing.T) {
	base := t.TempDir()
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	content := strings.Repeat("A", MinPartSize) + strings.Repeat("B", 100)
	u := newUpload(".", "out.bin", uint64(len(content)))
	if u.Parts != 2 {
		t.Fatalf("分片数应为 2，实际 %d", u.Parts)
	}

	if _, _, err := fsys.WriteUploadPart(u, 1, strings.NewReader(content[:MinPartSize])); err != nil {
		t.Fatalf("写第 1 片: %v", err)
	}
	if _, _, err := fsys.WriteUploadPart(u, 2, strings.NewReader(content[MinPartSize:])); err != nil {
		t.Fatalf("写第 2 片: %v", err)
	}
	// 缓冲文件名只由会话 id 决定，与目标文件名无关。
	if _, err := os.Stat(filepath.Join(base, ".momoko-upload-test-upload-1.part")); err != nil {
		t.Fatalf("缓冲文件命名不符预期: %v", err)
	}
	// 缓冲期间它对列目录不可见。
	if listed, err := fsys.List(".", ListOptions{}); err != nil {
		t.Fatal(err)
	} else if len(listed) != 0 {
		t.Fatalf("缓冲文件不应出现在列表中: %+v", listed)
	}
	// 分片没齐时不得收尾。
	if _, err := fsys.CommitUpload(u, 1); err == nil {
		t.Fatal("漏片时 CommitUpload 应当失败")
	}
	target, err := fsys.CommitUpload(u, 2)
	if err != nil {
		t.Fatalf("CommitUpload: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Fatalf("落地内容不符：长度 %d，期望 %d", len(got), len(content))
	}
	// 收尾后目标目录里只应有落地的文件，缓冲已随 rename 消失。
	entries, err := os.ReadDir(base)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name() != "out.bin" {
		names := make([]string, 0, len(entries))
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("目标目录残留了中间产物: %v", names)
	}
}

func TestUploadPartRejectsWrongLength(t *testing.T) {
	base := t.TempDir()
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	u := newUpload(".", "out.bin", MinPartSize+10)

	// 少发：拒绝。
	if _, _, err := fsys.WriteUploadPart(u, 1, strings.NewReader("short")); err == nil {
		t.Error("分片长度不足时应当拒绝")
	}
	// 多发：拒绝（否则可越写到相邻分片区间）。
	over := strings.Repeat("A", MinPartSize+50)
	if _, _, err := fsys.WriteUploadPart(u, 1, strings.NewReader(over)); err == nil {
		t.Error("分片超长时应当拒绝")
	}
	// 越界片号：拒绝。
	for _, p := range []uint64{0, 3, 99999} {
		if _, _, err := fsys.WriteUploadPart(u, p, strings.NewReader("x")); err == nil {
			t.Errorf("片号 %d 应当拒绝", p)
		}
	}
}

func TestUploadCannotEscapeViaDir(t *testing.T) {
	base, fsys := fixture(t)
	for _, dir := range vectors(base) {
		u := newUpload(dir, "evil.txt", 10)
		if _, _, err := fsys.WriteUploadPart(u, 1, strings.NewReader(strings.Repeat("x", 10))); err == nil {
			t.Errorf("上传到 %q 竟然成功了", dir)
		}
		if _, err := fsys.CommitUpload(u, u.Parts); err == nil {
			t.Errorf("收尾到 %q 竟然成功了", dir)
		}
	}
	if _, err := os.Stat(filepath.Join(base, "evil.txt")); err == nil {
		t.Fatal("根外出现了 evil.txt")
	}
}

func TestUploadEmptyFile(t *testing.T) {
	base := t.TempDir()
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	u := newUpload(".", "empty.txt", 0)
	if u.Parts != 0 {
		t.Fatalf("空文件分片数应为 0，实际 %d", u.Parts)
	}
	target, err := fsys.CommitUpload(u, 0)
	if err != nil {
		t.Fatalf("CommitUpload 空文件: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil || info.Size() != 0 {
		t.Fatalf("空文件落地异常: %v", err)
	}
}

func TestDiscardUpload(t *testing.T) {
	base := t.TempDir()
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	u := newUpload(".", "out.bin", MinPartSize)
	if _, _, err := fsys.WriteUploadPart(u, 1, strings.NewReader(strings.Repeat("A", MinPartSize))); err != nil {
		t.Fatal(err)
	}
	if err := fsys.DiscardUpload(u); err != nil {
		t.Fatalf("DiscardUpload: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, u.bufferName())); err == nil {
		t.Fatal("缓冲文件未被删除")
	}
	// 幂等：再删一次仍应成功。
	if err := fsys.DiscardUpload(u); err != nil {
		t.Fatalf("DiscardUpload 应当幂等: %v", err)
	}
}

// ---- 整机视图 ----

func TestSystemViewBasics(t *testing.T) {
	fsys, err := OpenSystem()
	if err != nil {
		t.Fatal(err)
	}
	if !fsys.IsSystem() {
		t.Fatal("应为整机视图")
	}
	// 虚拟根（Windows「此电脑」）/ 根目录都应可列。
	entries, err := fsys.List("", ListOptions{})
	if err != nil {
		t.Fatalf("List(\"\"): %v", err)
	}
	if len(entries) == 0 {
		t.Fatal("整机视图根目录不应为空")
	}
	if runtime.GOOS == "windows" {
		for _, e := range entries {
			if !e.IsDir || !strings.HasSuffix(e.Path, `\`) {
				t.Fatalf("盘符条目形态异常: %+v", e)
			}
		}
	}
	if _, err := fsys.DirStat(""); err != nil {
		t.Fatalf("DirStat(\"\"): %v", err)
	}

	// 真实临时目录应可读写，且返回真实绝对路径。
	tmp := t.TempDir()
	mustWrite(t, filepath.Join(tmp, "a.txt"), "alpha")
	data, err := fsys.ReadFile(filepath.Join(tmp, "a.txt"))
	if err != nil || string(data) != "alpha" {
		t.Fatalf("整机视图读取失败: %q %v", data, err)
	}
	items, err := fsys.List(tmp, ListOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 || !filepath.IsAbs(items[0].Path) {
		t.Fatalf("整机视图应返回真实绝对路径: %+v", items)
	}
}

func TestSystemViewRejectsDotDotAndUNC(t *testing.T) {
	fsys, err := OpenSystem()
	if err != nil {
		t.Fatal(err)
	}
	tmp := t.TempDir()
	// ".." 即便折叠后仍落在合法范围内，也一律显式拒绝，便于审计。
	// 注意不能用 filepath.Join 构造——它会先把 ".." 折叠掉。
	for _, p := range []string{
		tmp + string(filepath.Separator) + ".." + string(filepath.Separator) + "..",
		tmp + "/../..",
		tmp + "/...",
	} {
		if _, err := fsys.List(p, ListOptions{}); !errors.Is(err, ErrTraversal) {
			t.Errorf("整机视图应拒绝 %q，实际: %v", p, err)
		}
	}
	if runtime.GOOS == "windows" {
		// UNC 会让服务端向任意主机发起 SMB 认证，必须拒绝。
		for _, p := range []string{`\\127.0.0.1\C$\Windows`, `\\evil.example.com\share\x`, `\\?\C:\Windows`} {
			if _, err := fsys.List(p, ListOptions{}); err == nil {
				t.Errorf("UNC/设备路径 %q 应当被拒绝", p)
			}
		}
	}
}

func TestSystemViewDeny(t *testing.T) {
	secretDir := t.TempDir()
	mustWrite(t, filepath.Join(secretDir, "auth.secret"), "KEY")
	fsys, err := OpenSystem(Deny(secretDir))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.ReadFile(filepath.Join(secretDir, "auth.secret")); !errors.Is(err, ErrDenied) {
		t.Fatalf("整机视图下受保护路径应返回 ErrDenied，实际: %v", err)
	}
	if _, err := fsys.List(secretDir, ListOptions{}); !errors.Is(err, ErrDenied) {
		t.Fatalf("受保护目录不应可列，实际: %v", err)
	}
}

func TestSubScope(t *testing.T) {
	base, fsys := fixture(t)
	sub, err := fsys.Sub("sub")
	if err != nil {
		t.Fatal(err)
	}
	if data, err := sub.ReadFile("a.txt"); err != nil || string(data) != "alpha" {
		t.Fatalf("子视图读取失败: %q %v", data, err)
	}
	// 子视图不得回看父视图的内容。
	if _, err := sub.ReadFile("../../secret.txt"); err == nil {
		t.Fatal("子视图越界了")
	}
	if _, err := sub.Stat(filepath.Join(base, "root")); err == nil {
		t.Fatal("子视图不应看到父目录")
	}
}

func TestOpenRejectsFileAndMissing(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, filepath.Join(base, "a.txt"), "x")
	if _, err := Open(filepath.Join(base, "a.txt")); !errors.Is(err, ErrNotDir) {
		t.Errorf("Open 一个文件应返回 ErrNotDir，实际: %v", err)
	}
	if _, err := Open(filepath.Join(base, "nope")); err == nil {
		t.Error("Open 不存在的目录应当失败")
	}
	if _, err := Open(""); err == nil {
		t.Error("Open 空路径应当失败")
	}
}

func TestOpenDirCreates(t *testing.T) {
	base := t.TempDir()
	target := filepath.Join(base, "a", "b", "c")
	fsys, err := OpenDir(target)
	if err != nil {
		t.Fatalf("OpenDir: %v", err)
	}
	if info, err := os.Stat(target); err != nil || !info.IsDir() {
		t.Fatalf("OpenDir 未创建目录: %v", err)
	}
	if err := fsys.CreateFile("x.txt", []byte("v")); err != nil {
		t.Fatalf("新建视图不可写: %v", err)
	}
}

// TestRealPathValidates RealPath 是业务层拿真实路径的唯一入口（上传目标固化、外部进程工作目录），
// 它必须与其它操作一样拒绝越界，否则拿到的路径会被后续代码当作可信路径直接使用。
func TestRealPathValidates(t *testing.T) {
	base, fsys := fixture(t)
	for _, v := range vectors(base) {
		if got, err := fsys.RealPath(v); err == nil {
			t.Errorf("RealPath(%q) 竟然成功了 → %s", v, got)
		}
	}
	// 目标尚不存在也应能解析（上传到新目录时需要）。
	got, err := fsys.RealPath("newdir/sub")
	if err != nil {
		t.Fatalf("RealPath 对不存在的路径应当可用: %v", err)
	}
	want := filepath.Join(fsys.Base(), "newdir", "sub")
	if got != want {
		t.Fatalf("RealPath = %q，期望 %q", got, want)
	}
	// 根自身。
	if got, err := fsys.RealPath("."); err != nil || got != fsys.Base() {
		t.Fatalf("RealPath(\".\") = %q %v，期望 %q", got, err, fsys.Base())
	}
	// 受保护路径同样不给真实路径。
	denied, err := Open(base, Deny(filepath.Join(base, "outside")))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := denied.RealPath("outside"); !errors.Is(err, ErrDenied) {
		t.Errorf("受保护路径的 RealPath 应返回 ErrDenied，实际: %v", err)
	}
}

// TestOpenUploadBuffer 缓冲型远端来源的收尾路径：校验齐备后打开缓冲文件整流推送。
func TestOpenUploadBuffer(t *testing.T) {
	base := t.TempDir()
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	content := strings.Repeat("Z", MinPartSize)
	u := newUpload("", "remote.bin", uint64(len(content)))
	if _, _, err := fsys.WriteUploadPart(u, 1, strings.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	// 漏片时不得放行。
	if _, err := fsys.OpenUploadBuffer(u, 0); err == nil {
		t.Fatal("分片未齐时 OpenUploadBuffer 应当失败")
	}
	f, err := fsys.OpenUploadBuffer(u, u.Parts)
	if err != nil {
		t.Fatalf("OpenUploadBuffer: %v", err)
	}
	defer f.Close()
	got, err := io.ReadAll(f)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != content {
		t.Fatalf("缓冲内容不符：长度 %d，期望 %d", len(got), len(content))
	}
	// 体积被篡改后必须拒绝（声明与实际不符）。
	tampered := u
	tampered.Size = u.Size + 1
	tampered.Parts = CalcParts(tampered.Size, tampered.PartSize)
	if _, err := fsys.OpenUploadBuffer(tampered, tampered.Parts); err == nil {
		t.Fatal("声明体积与缓冲实际大小不符时应当拒绝")
	}
}

// TestSortEntries 目录恒在文件之前，各排序字段与升降序都要正确。
func TestSortEntries(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "zdir"))
	mustWrite(t, filepath.Join(base, "a.txt"), "x")
	mustWrite(t, filepath.Join(base, "b.txt"), strings.Repeat("y", 100))
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	names := func(opt ListOptions) []string {
		items, err := fsys.List(".", opt)
		if err != nil {
			t.Fatal(err)
		}
		out := make([]string, 0, len(items))
		for _, e := range items {
			out = append(out, e.Name)
		}
		return out
	}
	// 目录优先，其余按名称升序。
	if got := names(ListOptions{}); got[0] != "zdir" {
		t.Errorf("目录应排在最前，实际: %v", got)
	}
	if got := names(ListOptions{Desc: true}); got[0] != "zdir" {
		t.Errorf("降序时目录仍应排在最前，实际: %v", got)
	}
	// 按体积。
	got := names(ListOptions{Sort: SortBySize})
	if got[1] != "a.txt" || got[2] != "b.txt" {
		t.Errorf("按体积升序异常: %v", got)
	}
	got = names(ListOptions{Sort: SortBySize, Desc: true})
	if got[1] != "b.txt" || got[2] != "a.txt" {
		t.Errorf("按体积降序异常: %v", got)
	}
	// 按修改时间不应报错且顺序稳定。
	if got := names(ListOptions{Sort: SortByModTime}); len(got) != 3 {
		t.Errorf("按时间排序条目数异常: %v", got)
	}
}

// TestWriteFilePreservesSymlinkAndInode 编辑保存必须就地覆盖，而不是换掉一个新 inode。
//
// 面板用户编辑的往往是 server.properties 这类被 symlink 到共享配置的文件；
// 若保存时走「写临时文件再 rename」，链接会被悄悄替换成普通文件、硬链接被断开、
// 属主变成 momoko 进程用户，部署拓扑就此改变而没有任何提示。
func TestWriteFilePreservesSymlinkAndInode(t *testing.T) {
	base := t.TempDir()
	shared := filepath.Join(base, "shared", "app.conf")
	mustMkdir(t, filepath.Join(base, "shared"))
	mustWrite(t, shared, "old")
	mustMkdir(t, filepath.Join(base, "app"))
	link := filepath.Join(base, "app", "server.properties")
	// 用相对链接：os.Root 把绝对链接目标当作「相对视图根」解释（类似 chroot），
	// 因此指向绝对路径的链接一律解析失败——这本身也是一层保护。
	if err := os.Symlink(filepath.Join("..", "shared", "app.conf"), link); err != nil {
		t.Skipf("当前环境无法创建符号链接: %v", err)
	}
	// 视图涵盖链接与其目标（跨视图的链接由 os.Root 另行拒绝，见下一个用例）。
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.WriteFile("app/server.properties", strings.NewReader("new")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	info, err := os.Lstat(link)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatal("符号链接被替换成了普通文件")
	}
	if data, _ := os.ReadFile(shared); string(data) != "new" {
		t.Fatalf("内容没有写到链接目标上: %q", data)
	}
}

// TestWriteFileRefusesSymlinkOutOfScope 指向视图外的符号链接不得成为写入通道。
//
// 旧实现只对父目录做包含性检查，末段的符号链接照样被 O_TRUNC 跟随出去——
// 实例目录里放一个指向 /etc 的链接就能改写系统文件。
func TestWriteFileRefusesSymlinkOutOfScope(t *testing.T) {
	base := t.TempDir()
	outside := filepath.Join(base, "outside.conf")
	mustWrite(t, outside, "SYSTEM")
	scope := filepath.Join(base, "inst")
	mustMkdir(t, scope)
	if err := os.Symlink(outside, filepath.Join(scope, "hijack.conf")); err != nil {
		t.Skipf("当前环境无法创建符号链接: %v", err)
	}
	fsys, err := Open(scope)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.WriteFile("hijack.conf", strings.NewReader("pwned")); err == nil {
		t.Fatal("经越界符号链接写入应当被拒绝")
	}
	if data, _ := os.ReadFile(outside); string(data) != "SYSTEM" {
		t.Fatalf("视图外文件被改写了: %q", data)
	}
}

// TestWriteFileOversizeLeavesTargetIntact 超限写入不得先把目标截断再报错。
func TestWriteFileOversizeLeavesTargetIntact(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, filepath.Join(base, "a.txt"), "original")
	fsys, err := Open(base, WithMaxFileSize(16))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.WriteFile("a.txt", strings.NewReader(strings.Repeat("x", 64))); !errors.Is(err, ErrTooLarge) {
		t.Fatalf("超限写入应返回 ErrTooLarge，实际: %v", err)
	}
	if data, _ := os.ReadFile(filepath.Join(base, "a.txt")); string(data) != "original" {
		t.Fatalf("超限写入把目标截断了: %q", data)
	}
}

// TestWriteFilePreservesMode 覆盖写入保留既有权限位。
func TestWriteFilePreservesMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows 无 POSIX 权限位")
	}
	base := t.TempDir()
	target := filepath.Join(base, "run.sh")
	mustWrite(t, target, "#!/bin/sh\n")
	if err := os.Chmod(target, 0o750); err != nil {
		t.Fatal(err)
	}
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.WriteFile("run.sh", strings.NewReader("#!/bin/sh\necho hi\n")); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o750 {
		t.Fatalf("权限位未保留：%v", info.Mode().Perm())
	}
}
