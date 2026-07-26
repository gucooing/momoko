package file

import (
	"errors"
	"testing"

	"momoko/pkg/localfs"
)

// TestRemoteRelRejectsTraversal 远端来源的路径归一化必须拒绝越界，而不是把 ".." 折叠掉。
//
// 旧实现用 path.Join(base, logical)：Join 内部会 Clean，于是 "../x" 先把 base 吃掉，
// 配置里的 Prefix / BasePath 形同虚设。
func TestRemoteRelRejectsTraversal(t *testing.T) {
	for _, p := range []string{
		"..", "../x", "a/../..", "a/../../b", `..\x`, `a\..\..\b`,
		"...", "..../x", "/../etc", "a/b/../../../c",
	} {
		if _, err := remoteRel(p); !errors.Is(err, localfs.ErrTraversal) {
			t.Errorf("remoteRel(%q) 应返回 ErrTraversal，实际: %v", p, err)
		}
	}
	if _, err := remoteRel("a\x00b"); err == nil {
		t.Error("remoteRel 应拒绝 NUL 字节")
	}

	cases := map[string]string{
		"":          "",
		".":         "",
		"/":         "",
		"a":         "a",
		"/a/b":      "a/b",
		"a//b/":     "a/b",
		`a\b`:       "a/b",
		"./a/./b":   "a/b",
		"a/b/c.txt": "a/b/c.txt",
	}
	for in, want := range cases {
		got, err := remoteRel(in)
		if err != nil {
			t.Errorf("remoteRel(%q) 报错: %v", in, err)
			continue
		}
		if got != want {
			t.Errorf("remoteRel(%q) = %q，期望 %q", in, got, want)
		}
	}
}

// TestRemoteAbsStaysUnderBase 归一化后的远端绝对路径必须始终落在 base 之下。
func TestRemoteAbsStaysUnderBase(t *testing.T) {
	const base = "data/files"
	for _, p := range []string{"../etc", "../../etc", "a/../../../etc", `..\..\etc`} {
		if _, err := remoteAbs(base, p); !errors.Is(err, localfs.ErrTraversal) {
			t.Errorf("remoteAbs(%q, %q) 应当拒绝", base, p)
		}
	}
	got, err := remoteAbs(base, "sub/x.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/data/files/sub/x.txt" {
		t.Fatalf("remoteAbs 结果异常: %s", got)
	}
	root, err := remoteAbs(base, "")
	if err != nil {
		t.Fatal(err)
	}
	if root != "/data/files" {
		t.Fatalf("根路径异常: %s", root)
	}
}

// TestRemoteKeyStaysUnderPrefix 对象 key 必须始终落在配置的 Prefix 之下。
func TestRemoteKeyStaysUnderPrefix(t *testing.T) {
	const root = "tenant-a"
	for _, p := range []string{"../tenant-b/secret", "..", "a/../../tenant-b"} {
		if _, err := remoteKey(root, p); !errors.Is(err, localfs.ErrTraversal) {
			t.Errorf("remoteKey(%q, %q) 应当拒绝", root, p)
		}
	}
	got, err := remoteKey(root, "docs/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "tenant-a/docs/a.txt" {
		t.Fatalf("remoteKey 结果异常: %s", got)
	}
	// 无 Prefix 的桶。
	got, err = remoteKey("", "docs/a.txt")
	if err != nil {
		t.Fatal(err)
	}
	if got != "docs/a.txt" {
		t.Fatalf("无 Prefix 时 key 异常: %s", got)
	}
}

// TestRemoteLogicalRespectsBoundary 剥前缀必须按路径边界，避免兄弟前缀被误认作子项。
func TestRemoteLogicalRespectsBoundary(t *testing.T) {
	cases := []struct{ root, key, want string }{
		{"a", "a/b/c", "b/c"},
		{"a", "a", ""},
		{"a", "ab/c", "ab/c"}, // 兄弟前缀：旧的裸 TrimPrefix 会错算成 "b/c"
		{"", "x/y", "x/y"},
		{"tenant-a", "tenant-ab/secret", "tenant-ab/secret"},
	}
	for _, c := range cases {
		if got := remoteLogical(c.root, c.key); got != c.want {
			t.Errorf("remoteLogical(%q, %q) = %q，期望 %q", c.root, c.key, got, c.want)
		}
	}
}

// TestRemoteRenameRejectsPathInName 改名的新名称不得含任何路径成分。
//
// 旧实现只查 "/" 与 "\"，于是 newName=".." 会让 path.Join 把目标顶到上级目录。
func TestRemoteRenameRejectsPathInName(t *testing.T) {
	for _, name := range []string{"..", ".", "a/b", `a\b`, "/abs", "", "NUL", "trail ", "x:ads"} {
		if _, _, err := remoteRename("dir/file.txt", name); err == nil {
			t.Errorf("remoteRename 新名称 %q 应当拒绝", name)
		}
	}
	src, dst, err := remoteRename("dir/file.txt", "new.txt")
	if err != nil {
		t.Fatal(err)
	}
	if src != "dir/file.txt" || dst != "dir/new.txt" {
		t.Fatalf("remoteRename 结果异常: %q → %q", src, dst)
	}
	// 顶层文件改名。
	if _, dst, err = remoteRename("file.txt", "new.txt"); err != nil || dst != "new.txt" {
		t.Fatalf("顶层改名异常: %q %v", dst, err)
	}
	// 根目录不可改名。
	if _, _, err := remoteRename("", "x"); !errors.Is(err, localfs.ErrRootScope) {
		t.Errorf("根目录改名应返回 ErrRootScope，实际: %v", err)
	}
}

// TestGuardRejectsAnyBadPath 入口守卫只要有一个路径越界就整体拒绝。
func TestGuardRejectsAnyBadPath(t *testing.T) {
	if err := guard("ok/a", "ok/b"); err != nil {
		t.Fatalf("合法路径不应被拒: %v", err)
	}
	if err := guard("ok/a", "../escape"); !errors.Is(err, localfs.ErrTraversal) {
		t.Fatalf("含越界路径的批量操作应整体拒绝，实际: %v", err)
	}
}

// TestUploadSeedValidatesName 上传目标文件名必须过 localfs 的名称校验。
func TestUploadSeedValidatesName(t *testing.T) {
	for _, name := range []string{
		"../../evil.txt", `..\evil.txt`, "a/b.txt", "/etc/passwd",
		"NUL", "evil.txt:ads", "", "..", "trail ", "trail.",
	} {
		if _, err := NewUploadSeed("hash", "/tmp", name, 10, ""); err == nil {
			t.Errorf("上传文件名 %q 应当被拒绝", name)
		}
	}
	seed, err := NewUploadSeed("hash", "/tmp", "report.pdf", localfs.MinPartSize+1, "")
	if err != nil {
		t.Fatalf("合法参数应当通过: %v", err)
	}
	if seed.Parts != 2 || seed.PartSize != localfs.MinPartSize {
		t.Fatalf("分片布局异常: %+v", seed)
	}
	// 空目标目录必须被接受：远端来源（OSS/FTP/WebDAV）的根目录，其逻辑路径就是空串，
	// 前端在来源根页面上传时传的正是它。
	if _, err := NewUploadSeed("hash", "", "a.txt", 1, "oss-1"); err != nil {
		t.Errorf("远端来源根目录上传应当被接受，实际: %v", err)
	}
}

// TestUploadTargetPathAtSourceRoot 来源根目录上传时目标路径就是文件名本身。
func TestUploadTargetPathAtSourceRoot(t *testing.T) {
	seed, err := NewUploadSeed("hash", "", "a.txt", 0, "oss-1")
	if err != nil {
		t.Fatal(err)
	}
	u := &Upload{Target: seed.Dir, Spec: localfs.Upload{Name: seed.Name}}
	if got := u.TargetPath(); got != "a.txt" {
		t.Fatalf("来源根目录上传的目标路径 = %q，期望 %q", got, "a.txt")
	}
}

// TestUploadTargetPath 目标逻辑路径拼接对正反斜杠都稳定。
func TestUploadTargetPath(t *testing.T) {
	cases := []struct{ dir, name, want string }{
		{"", "a.txt", "a.txt"},
		{"/", "a.txt", "a.txt"},
		{"docs", "a.txt", "docs/a.txt"},
		{"/docs/", "a.txt", "docs/a.txt"},
		{`docs\sub`, "a.txt", "docs/sub/a.txt"},
	}
	for _, c := range cases {
		u := &Upload{Target: c.dir, Spec: localfs.Upload{Name: c.name}}
		if got := u.TargetPath(); got != c.want {
			t.Errorf("TargetPath(dir=%q,name=%q) = %q，期望 %q", c.dir, c.name, got, c.want)
		}
	}
}
