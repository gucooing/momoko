package agentfs

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestFileTreeEdits(t *testing.T) {
	root := t.TempDir()

	// 新建文件（名字带 / 时中间的文件夹一起建出来），同名再建报错
	if _, err := SaveText(root, "src/a.txt", "hello", true, 0); err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := SaveText(root, "src/a.txt", "x", true, 0); err == nil {
		t.Fatal("create over existing file should fail")
	}

	// 保存：带着读到的修改时间能存，文件之后被改过就 ErrChanged
	text, err := ReadText(root, "src/a.txt", 1<<20)
	if err != nil || text.Content != "hello" || text.ModTime == 0 {
		t.Fatalf("read: %+v %v", text, err)
	}
	mod, err := SaveText(root, "src/a.txt", "v2", false, text.ModTime)
	if err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := SaveText(root, "src/a.txt", "stale", false, text.ModTime-1); !errors.Is(err, ErrChanged) {
		t.Fatalf("stale save should be ErrChanged, got %v", err)
	}
	if _, err := SaveText(root, "src/a.txt", "v3", false, mod); err != nil {
		t.Fatalf("save with new mod: %v", err)
	}
	if raw, _ := os.ReadFile(filepath.Join(root, "src", "a.txt")); string(raw) != "v3" {
		t.Fatalf("content = %q", raw)
	}

	// 新建文件夹，同名再建报错
	if err := MakeDir(root, "docs/img"); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := MakeDir(root, "docs"); err == nil {
		t.Fatal("mkdir over existing should fail")
	}

	// 改名、挪到别的文件夹；目标已存在、挪进自己里面都不行
	if err := Rename(root, "src/a.txt", "src/b.txt"); err != nil {
		t.Fatalf("rename: %v", err)
	}
	if err := Rename(root, "src/b.txt", "docs/b.txt"); err != nil {
		t.Fatalf("move: %v", err)
	}
	if _, err := SaveText(root, "docs/c.txt", "", true, 0); err != nil {
		t.Fatal(err)
	}
	if err := Rename(root, "docs/b.txt", "docs/c.txt"); err == nil {
		t.Fatal("move onto existing file should fail")
	}
	if err := Rename(root, "docs", "docs/img/docs"); err == nil {
		t.Fatal("move folder into itself should fail")
	}
	if err := Rename(root, "docs/c.txt", "docs/C.txt"); err != nil {
		t.Fatalf("case-only rename: %v", err)
	}

	// 路径不能跳出根目录、不能是根目录本身
	for _, bad := range []string{"", ".", "../x", "a/../../x", "a/./b", "a/ b"} {
		if _, err := SaveText(root, bad, "", true, 0); err == nil {
			t.Fatalf("path %q should be rejected", bad)
		}
	}
	if err := Remove(root, ""); err == nil {
		t.Fatal("removing root should fail")
	}

	// 删除文件夹连同里面的内容
	if err := Remove(root, "docs"); err != nil {
		t.Fatalf("remove: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "docs")); !os.IsNotExist(err) {
		t.Fatal("docs should be gone")
	}
	if !IsDir(root) {
		t.Fatal("root should still exist")
	}
}
