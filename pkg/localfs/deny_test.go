package localfs

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// denyFixture 搭一个「应用目录内含受保护子目录」的场景：
//
//	base/
//	  momoko/
//	    configs/auth.secret   ← 受保护（JWT 签名密钥）
//	    data/momoko.db        ← 受保护（SQLite 库）
//	    public/               ← 普通目录
//	  outbox/                 ← 普通目录
func denyFixture(t *testing.T) (base string, fsys *FS) {
	t.Helper()
	base = t.TempDir()
	if real, err := filepath.EvalSymlinks(base); err == nil {
		base = real
	}
	app := filepath.Join(base, "momoko")
	mustMkdir(t, filepath.Join(app, "configs"))
	mustMkdir(t, filepath.Join(app, "data"))
	mustMkdir(t, filepath.Join(app, "public"))
	mustMkdir(t, filepath.Join(base, "outbox"))
	mustWrite(t, filepath.Join(app, "configs", "auth.secret"), "JWT-SIGNING-KEY")
	mustWrite(t, filepath.Join(app, "data", "momoko.db"), "SQLITE-DATA")

	fsys, err := Open(base, Deny(filepath.Join(app, "configs"), filepath.Join(app, "data")))
	if err != nil {
		t.Fatal(err)
	}
	return base, fsys
}

func secretIntact(t *testing.T, base string) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(base, "momoko", "configs", "auth.secret"))
	if err != nil {
		t.Fatalf("签名密钥文件被破坏/删除: %v", err)
	}
	if string(data) != "JWT-SIGNING-KEY" {
		t.Fatalf("签名密钥被改写: %q", data)
	}
}

// TestDenyCannotBeBypassedViaAncestor 对着保护目录的「父级」下手同样必须被拒。
//
// 这是保护清单最容易被绕开的一条路：判定只看「目标是否落在保护项内」时，
// 删除/改名/移动/复制/打包父目录都会连带处理保护项，而一次判定都不会触发。
func TestDenyCannotBeBypassedViaAncestor(t *testing.T) {
	t.Run("Remove 父目录", func(t *testing.T) {
		base, fsys := denyFixture(t)
		if res := fsys.Remove([]string{"momoko"}); res[0].OK {
			t.Error("删除含保护目录的父级竟然成功了")
		}
		secretIntact(t, base)
	})
	t.Run("Rename 父目录", func(t *testing.T) {
		base, fsys := denyFixture(t)
		if _, err := fsys.Rename("momoko", "moved"); !errors.Is(err, ErrDenied) {
			t.Errorf("改名含保护目录的父级应返回 ErrDenied，实际: %v", err)
		}
		secretIntact(t, base)
	})
	t.Run("MoveInto 父目录", func(t *testing.T) {
		base, fsys := denyFixture(t)
		if res := fsys.MoveInto([]string{"momoko"}, "outbox"); res[0].OK {
			t.Error("移动含保护目录的父级竟然成功了")
		}
		secretIntact(t, base)
	})
	t.Run("CopyInto 父目录", func(t *testing.T) {
		base, fsys := denyFixture(t)
		res := fsys.CopyInto([]string{"momoko"}, "outbox")
		if res[0].OK {
			t.Error("复制含保护目录的父级竟然成功了")
		}
		if _, err := os.Stat(filepath.Join(base, "outbox", "momoko", "configs", "auth.secret")); err == nil {
			t.Fatal("密钥被复制到了可读位置")
		}
		secretIntact(t, base)
	})
	t.Run("Compress 父目录", func(t *testing.T) {
		base, fsys := denyFixture(t)
		if _, err := fsys.Compress([]string{"momoko"}, "outbox/loot.zip"); !errors.Is(err, ErrDenied) {
			t.Errorf("打包含保护目录的父级应返回 ErrDenied，实际: %v", err)
		}
		if _, err := os.Stat(filepath.Join(base, "outbox", "loot.zip")); err == nil {
			t.Fatal("生成了含密钥的压缩包")
		}
		secretIntact(t, base)
	})
	t.Run("Extract 到父目录", func(t *testing.T) {
		base, fsys := denyFixture(t)
		writeZipFile(t, filepath.Join(base, "evil.zip"), map[string]string{
			"configs/auth.secret": "ATTACKER-CHOSEN-KEY",
			"data/momoko.db":      "WIPED",
		})
		if _, err := fsys.Extract("evil.zip", "momoko"); !errors.Is(err, ErrDenied) {
			t.Errorf("解压到含保护目录的父级应返回 ErrDenied，实际: %v", err)
		}
		secretIntact(t, base)
	})
}

// TestExtractDoesNotOverwrite 解压不得静默覆盖既有文件。
func TestExtractDoesNotOverwrite(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "www"))
	mustWrite(t, filepath.Join(base, "www", "index.html"), "ORIGINAL")
	writeZipFile(t, filepath.Join(base, "evil.zip"), map[string]string{
		"index.html": "<script>attacker</script>",
	})
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.Extract("evil.zip", "www"); !errors.Is(err, ErrExist) {
		t.Errorf("解压撞名应返回 ErrExist，实际: %v", err)
	}
	if data, _ := os.ReadFile(filepath.Join(base, "www", "index.html")); string(data) != "ORIGINAL" {
		t.Fatalf("既有文件被覆盖: %q", data)
	}
}

// TestExtractRollsBackOnFailure 解压中途失败必须撤销已写出的文件，
// 否则反复触发失败就能不断把半截内容留在盘上，所谓上限形同虚设。
//
// 用「中途撞名」来制造中断：总量闸门现在会在开写之前就按声明体积拒绝，
// 已经到不了循环里，因此改用撞名这一必然发生在写了若干条之后的失败。
func TestExtractRollsBackOnFailure(t *testing.T) {
	base := t.TempDir()
	entries := map[string]string{}
	for i := range 8 {
		entries[fmt.Sprintf("%c.bin", 'a'+i)] = strings.Repeat("Q", 128)
	}
	writeZipFile(t, filepath.Join(base, "many.zip"), entries)

	// 目标目录预置一个同名文件：解压到它时必然失败，此前写出的条目都应被撤销。
	mustMkdir(t, filepath.Join(base, "out"))
	mustWrite(t, filepath.Join(base, "out", "e.bin"), "PRE-EXISTING")

	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.Extract("many.zip", "out"); !errors.Is(err, ErrExist) {
		t.Fatalf("撞名应返回 ErrExist，实际: %v", err)
	}
	left, err := os.ReadDir(filepath.Join(base, "out"))
	if err != nil {
		t.Fatal(err)
	}
	if len(left) != 1 || left[0].Name() != "e.bin" {
		names := make([]string, 0, len(left))
		for _, e := range left {
			names = append(names, e.Name())
		}
		t.Fatalf("失败后应只剩预置文件，实际残留: %v", names)
	}
	if data, _ := os.ReadFile(filepath.Join(base, "out", "e.bin")); string(data) != "PRE-EXISTING" {
		t.Fatalf("预置文件被动过: %q", data)
	}
}

// TestExtractRejectsOversizeDeclaration 声明总体积超上限时，在开写之前就拒绝。
func TestExtractRejectsOversizeDeclaration(t *testing.T) {
	base := t.TempDir()
	entries := map[string]string{}
	for i := range 6 {
		entries[fmt.Sprintf("%c.bin", 'a'+i)] = strings.Repeat("Q", 40_000)
	}
	writeZipFile(t, filepath.Join(base, "big.zip"), entries)

	fsys, err := Open(base, WithArchiveLimits(0, 100_000, 1_000_000))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.Extract("big.zip", "out"); !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("超总量应返回 ErrArchiveLimit，实际: %v", err)
	}
	if _, err := os.Stat(filepath.Join(base, "out")); err == nil {
		t.Fatal("被拒绝的解压不应创建目标目录")
	}
}

// TestExtractRejectsChunkedZipBomb 把炸弹切成一堆小条目也必须被拦下：
// 逐条目的压缩比闸门对每条都不触发，只有整包聚合判定才拦得住。
func TestExtractRejectsChunkedZipBomb(t *testing.T) {
	base := t.TempDir()
	entries := map[string]string{}
	for i := range 40 {
		// 每条 60 KiB，低于单条目压缩比判定的下限（64 KiB）。
		entries[fmt.Sprintf("chunk-%02d.bin", i)] = strings.Repeat("\x00", 60<<10)
	}
	writeZipFile(t, filepath.Join(base, "bomb.zip"), entries)
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := fsys.Extract("bomb.zip", "out"); !errors.Is(err, ErrArchiveLimit) {
		t.Fatalf("分片式 zip 炸弹应返回 ErrArchiveLimit，实际: %v", err)
	}
}

// TestCopyIntoSymlinkedDstTerminates 目标是指向源内部的链接时必须直接拒绝，
// 否则 WalkDir 会一边遍历一边把新写入的条目也走一遍，永不终止。
func TestCopyIntoSymlinkedDstTerminates(t *testing.T) {
	base := t.TempDir()
	mustMkdir(t, filepath.Join(base, "src", "deep"))
	mustWrite(t, filepath.Join(base, "src", "a.txt"), strings.Repeat("x", 4096))
	if err := os.Symlink(filepath.Join("src", "deep"), filepath.Join(base, "alias")); err != nil {
		t.Skipf("当前环境无法创建符号链接: %v", err)
	}
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan []Result, 1)
	go func() { done <- fsys.CopyInto([]string{"src"}, "alias") }()
	select {
	case res := <-done:
		if res[0].OK {
			t.Error("复制到指向源内部的链接竟然成功了")
		}
	case <-time.After(15 * time.Second):
		t.Fatal("复制没有终止——递归失控")
	}
}

// TestStripBaseNoPanicOnUnicodeFold 大小写折叠会改变字节长度，
// 按 len(base) 切字节会 panic（U+212A KELVIN SIGN 三字节，小写 'k' 一字节）。
func TestStripBaseNoPanicOnUnicodeFold(t *testing.T) {
	base := t.TempDir()
	kelvin := filepath.Join(base, strings.Repeat("K", 8))
	if err := os.MkdirAll(kelvin, 0o755); err != nil {
		t.Skipf("当前环境无法创建该目录名: %v", err)
	}
	fsys, err := Open(kelvin)
	if err != nil {
		t.Skipf("无法打开视图: %v", err)
	}
	// 不论解析成功与否，都绝不能 panic。
	_, _ = fsys.Stat(filepath.Join(base, strings.Repeat("k", 8), "a.txt"))
	_, _ = fsys.List(filepath.Join(base, strings.Repeat("k", 8)), ListOptions{})
	_, _ = fsys.RealPath(filepath.Join(base, strings.Repeat("K", 8), "b.txt"))
}

// TestUploadRejectsOversizeDeclaration 声明的体积必须受策略上限约束，
// 否则「声明 1 TiB 只发最后一片」会在稀疏文件系统上一次铺开整个体积。
func TestUploadRejectsOversizeDeclaration(t *testing.T) {
	base := t.TempDir()
	fsys, err := Open(base, WithMaxUploadSize(1<<20))
	if err != nil {
		t.Fatal(err)
	}
	u := newUpload(".", "huge.bin", 1<<40)
	if _, _, err := fsys.WriteUploadPart(u, u.Parts, strings.NewReader("x")); !errors.Is(err, ErrTooLarge) {
		t.Errorf("超上限的上传声明应返回 ErrTooLarge，实际: %v", err)
	}
	if _, err := fsys.CommitUpload(u, u.Parts); !errors.Is(err, ErrTooLarge) {
		t.Errorf("收尾同样应拒绝，实际: %v", err)
	}
	if entries, _ := os.ReadDir(base); len(entries) != 0 {
		t.Fatalf("被拒绝的上传仍留下了文件: %d 个", len(entries))
	}
}

// TestCalcPartsNoOverflow 分片数计算不得溢出——溢出会算出 Parts=0，
// 从而绕过 Validate 的自洽检查，并让收尾走「空文件」分支把既有文件清零。
func TestCalcPartsNoOverflow(t *testing.T) {
	if got := CalcParts(math.MaxUint64, math.MaxUint64-4); got == 0 {
		t.Fatal("CalcParts 溢出算出了 0")
	}
	u := Upload{ID: "abc", Name: "x.txt", Size: math.MaxInt64, PartSize: uint64(math.MaxInt64) + 4, Parts: 0}
	if err := u.Validate(); err == nil {
		t.Fatal("自相矛盾的分片布局应当被拒绝")
	}
}

// TestCommitUploadRefusesZeroPartsWithSize Parts=0 但 Size!=0 时不得落空文件，
// 否则会拿 O_TRUNC 把一个既有文件清零。
func TestCommitUploadRefusesZeroPartsWithSize(t *testing.T) {
	base := t.TempDir()
	mustWrite(t, filepath.Join(base, "important.txt"), "PRECIOUS DATA")
	fsys, err := Open(base)
	if err != nil {
		t.Fatal(err)
	}
	u := Upload{ID: "abc", Dir: ".", Name: "important.txt", Size: 1024, PartSize: MinPartSize, Parts: 0}
	if _, err := fsys.CommitUpload(u, 0); err == nil {
		t.Error("Parts=0 但 Size!=0 的收尾应当被拒绝")
	}
	if data, _ := os.ReadFile(filepath.Join(base, "important.txt")); string(data) != "PRECIOUS DATA" {
		t.Fatalf("既有文件被清零: %q", data)
	}
}

// TestDenyStillAllowsBrowsingParent 保护清单不应把父目录本身也变得不可用：
// 列目录与读取兄弟文件必须照常，只是保护项不可见、不可达。
func TestDenyStillAllowsBrowsingParent(t *testing.T) {
	base, fsys := denyFixture(t)
	mustWrite(t, filepath.Join(base, "momoko", "README.md"), "hello")

	items, err := fsys.List("momoko", ListOptions{})
	if err != nil {
		t.Fatalf("列出含保护子目录的父级应当可用: %v", err)
	}
	names := map[string]bool{}
	for _, e := range items {
		names[e.Name] = true
	}
	for _, hidden := range []string{"configs", "data"} {
		if names[hidden] {
			t.Errorf("保护目录 %q 不应出现在列表中", hidden)
		}
	}
	if !names["public"] || !names["README.md"] {
		t.Errorf("普通条目被误伤: %v", names)
	}
	if data, err := fsys.ReadFile("momoko/README.md"); err != nil || string(data) != "hello" {
		t.Errorf("读取兄弟文件失败: %q %v", data, err)
	}
	// 删除普通子目录仍应可行。
	if res := fsys.Remove([]string{"momoko/public"}); !res[0].OK {
		t.Errorf("删除普通子目录不应被保护清单拦下: %+v", res)
	}
}
