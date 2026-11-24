package agentgit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// newRepo 建一个临时仓库，配好提交身份，不受本机全局配置影响。
func newRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("没有 git")
	}
	dir := t.TempDir()
	git(t, dir, "init", "-q", "-b", "main")
	git(t, dir, "config", "user.name", "Tester")
	git(t, dir, "config", "user.email", "tester@example.com")
	git(t, dir, "config", "core.autocrlf", "false")
	git(t, dir, "config", "commit.gpgsign", "false")
	return dir
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := Run(context.Background(), dir, args...)
	if err != nil {
		t.Fatalf("git %v: %v", args, err)
	}
	return out
}

func write(t *testing.T, dir, name, text string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

func read(t *testing.T, dir, name string) string {
	t.Helper()
	raw, err := os.ReadFile(filepath.Join(dir, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func numbered(n int, change map[int]string) string {
	var b strings.Builder
	for i := 1; i <= n; i++ {
		if s, ok := change[i]; ok {
			b.WriteString(s + "\n")
			continue
		}
		fmt.Fprintf(&b, "line%d\n", i)
	}
	return b.String()
}

func findFile(files []File, path string) *File {
	for i := range files {
		if files[i].Path == path {
			return &files[i]
		}
	}
	return nil
}

// selectLines diff 里属于第 hunk 段（从 0 数）的 + / - 行号；signs 限定只要哪种。
func selectLines(patch string, hunk int, signs string) []int {
	var out []int
	h := -1
	for i, l := range strings.Split(patch, "\n") {
		if strings.HasPrefix(l, "@@") {
			h++
			continue
		}
		if h == hunk && l != "" && strings.ContainsRune(signs, rune(l[0])) {
			out = append(out, i)
		}
	}
	return out
}

func TestStatusAndLineStaging(t *testing.T) {
	ctx := context.Background()
	dir := newRepo(t)
	write(t, dir, "a.txt", numbered(30, nil))
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")

	write(t, dir, "a.txt", numbered(30, map[int]string{2: "LINE2", 25: "LINE25"}))
	write(t, dir, "new dir/b.txt", "x\ny\n")

	st, err := GetStatus(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if st.Branch != "main" || st.Head == "" || len(st.Staged) != 0 {
		t.Fatalf("status = %+v", st)
	}
	a := findFile(st.Unstaged, "a.txt")
	b := findFile(st.Unstaged, "new dir/b.txt")
	if a == nil || a.Kind != "M" || a.Added != 2 || a.Deleted != 2 {
		t.Fatalf("a.txt = %+v", a)
	}
	if b == nil || b.Kind != "U" || b.Added != 2 {
		t.Fatalf("b.txt = %+v", b)
	}
	if st.LastMessage != "init" {
		t.Fatalf("last message = %q", st.LastMessage)
	}

	// 只暂存第一段
	d, err := GetDiff(ctx, dir, DiffUnstaged, "a.txt", "", "")
	if err != nil || strings.Count(d.Patch, "\n@@") != 1 {
		t.Fatalf("diff = %+v, %v", d, err)
	}
	if err := ApplyLines(ctx, dir, "a.txt", d.Patch, selectLines(d.Patch, 0, "+-"), LinesStage); err != nil {
		t.Fatal(err)
	}
	cached := git(t, dir, "diff", "--cached")
	if !strings.Contains(cached, "+LINE2") || strings.Contains(cached, "LINE25") {
		t.Fatalf("cached diff:\n%s", cached)
	}
	st, _ = GetStatus(ctx, dir)
	if s, u := findFile(st.Staged, "a.txt"), findFile(st.Unstaged, "a.txt"); s == nil || u == nil || s.Added != 1 || u.Added != 1 {
		t.Fatalf("after stage: staged %+v unstaged %+v", s, u)
	}

	// 暂存区里那一段只取消暂存 + 行：- 行留着，暂存区里 line2 被删掉
	sd, _ := GetDiff(ctx, dir, DiffStaged, "a.txt", "", "")
	if err := ApplyLines(ctx, dir, "a.txt", sd.Patch, selectLines(sd.Patch, 0, "+"), LinesUnstage); err != nil {
		t.Fatal(err)
	}
	cached = git(t, dir, "diff", "--cached")
	if !strings.Contains(cached, "-line2") || strings.Contains(cached, "+LINE2") {
		t.Fatalf("cached after partial unstage:\n%s", cached)
	}
	// 再把剩下的取消暂存
	sd, _ = GetDiff(ctx, dir, DiffStaged, "a.txt", "", "")
	if err := ApplyLines(ctx, dir, "a.txt", sd.Patch, selectLines(sd.Patch, 0, "+-"), LinesUnstage); err != nil {
		t.Fatal(err)
	}
	if c := git(t, dir, "diff", "--cached"); c != "" {
		t.Fatalf("still staged:\n%s", c)
	}

	// 只暂存一个 + 行（对应的 - 行没选）：暂存区里旧的那行还在，新行插在后面
	d, _ = GetDiff(ctx, dir, DiffUnstaged, "a.txt", "", "")
	if err := ApplyLines(ctx, dir, "a.txt", d.Patch, selectLines(d.Patch, 1, "+"), LinesStage); err != nil {
		t.Fatal(err)
	}
	if idx := git(t, dir, "show", ":a.txt"); !strings.Contains(idx, "line25\nLINE25\n") {
		t.Fatalf("index:\n%s", idx)
	}
	git(t, dir, "reset", "-q")

	// 撤销工作区里的第二段
	d, _ = GetDiff(ctx, dir, DiffUnstaged, "a.txt", "", "")
	if err := ApplyLines(ctx, dir, "a.txt", d.Patch, selectLines(d.Patch, 1, "+-"), LinesDiscard); err != nil {
		t.Fatal(err)
	}
	if got := read(t, dir, "a.txt"); got != numbered(30, map[int]string{2: "LINE2"}) {
		t.Fatalf("worktree:\n%s", got)
	}

	// 文件在这期间又改过：拒绝，不改错地方
	stale := d.Patch
	write(t, dir, "a.txt", numbered(30, map[int]string{2: "OTHER"}))
	if err := ApplyLines(ctx, dir, "a.txt", stale, selectLines(stale, 0, "+-"), LinesDiscard); err == nil {
		t.Fatal("stale patch applied")
	}

	// 没跟踪的文件：diff 整个算新增
	nd, _ := GetDiff(ctx, dir, DiffUnstaged, "new dir/b.txt", "", "")
	if !strings.HasPrefix(nd.Patch, "@@ -0,0 +1,2 @@\n+x\n+y") {
		t.Fatalf("new file diff = %q", nd.Patch)
	}
}

func TestLineStagingSpacesAndNoNewline(t *testing.T) {
	ctx := context.Background()
	dir := newRepo(t)
	write(t, dir, "sp ace.txt", "a\nb\nc")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")
	write(t, dir, "sp ace.txt", "A\nb\nc\nd")
	d, _ := GetDiff(ctx, dir, DiffUnstaged, "sp ace.txt", "", "")
	if err := ApplyLines(ctx, dir, "sp ace.txt", d.Patch, selectLines(d.Patch, 0, "+-"), LinesStage); err != nil {
		t.Fatalf("%v\n%s", err, d.Patch)
	}
	if idx := git(t, dir, "show", ":sp ace.txt"); idx != "A\nb\nc\nd" {
		t.Fatalf("index = %q", idx)
	}
}

func TestFilesCommitAndUndo(t *testing.T) {
	ctx := context.Background()
	dir := newRepo(t)
	write(t, dir, "a.txt", "1\n")
	write(t, dir, "b.txt", "2\n")
	if err := StageFiles(ctx, dir, []string{"a.txt"}, false); err != nil {
		t.Fatal(err)
	}
	// 还没有提交时取消暂存
	if err := UnstageFiles(ctx, dir, nil, true); err != nil {
		t.Fatal(err)
	}
	if out := git(t, dir, "ls-files"); out != "" {
		t.Fatalf("index not empty: %q", out)
	}
	if _, err := Commit(ctx, dir, CommitOptions{}); err == nil {
		t.Fatal("empty message accepted")
	}
	msg := "第一次提交 \"引号\"\n\n正文第二行"
	head, err := Commit(ctx, dir, CommitOptions{Message: msg, All: true})
	if err != nil || head == "" {
		t.Fatalf("commit: %q %v", head, err)
	}
	if got := strings.TrimSpace(git(t, dir, "log", "-1", "--format=%B")); got != msg {
		t.Fatalf("message = %q", got)
	}
	// 没有改动时提交：git 说的原因带回来
	if _, err := Commit(ctx, dir, CommitOptions{Message: "x"}); err == nil || !strings.Contains(err.Error(), "nothing") {
		t.Fatalf("commit nothing: %v", err)
	}
	// 修改上一次提交，说明沿用
	write(t, dir, "c.txt", "3\n")
	if _, err := Commit(ctx, dir, CommitOptions{Amend: true, All: true}); err != nil {
		t.Fatal(err)
	}
	if n := strings.TrimSpace(git(t, dir, "rev-list", "--count", "HEAD")); n != "1" {
		t.Fatalf("commits = %s", n)
	}
	// 撤销第一个提交：分支回到还没有提交，文件都在暂存区
	if err := Reset(ctx, dir, "HEAD~1", "soft"); err != nil {
		t.Fatal(err)
	}
	st, _ := GetStatus(ctx, dir)
	if st.Head != "" || len(st.Staged) != 3 {
		t.Fatalf("after undo: %+v", st)
	}
	if _, err := Commit(ctx, dir, CommitOptions{Message: "again"}); err != nil {
		t.Fatal(err)
	}

	// 撤销：跟踪着的恢复，新文件删掉；忽略
	write(t, dir, "a.txt", "changed\n")
	write(t, dir, "tmp.log", "x\n")
	write(t, dir, "junk.txt", "x\n")
	if err := DiscardFiles(ctx, dir, []string{"a.txt", "junk.txt"}, false); err != nil {
		t.Fatal(err)
	}
	if read(t, dir, "a.txt") != "1\n" || exists(filepath.Join(dir, "junk.txt")) {
		t.Fatal("discard did not restore")
	}
	if err := IgnoreFiles(dir, []string{"tmp.log"}); err != nil {
		t.Fatal(err)
	}
	st, _ = GetStatus(ctx, dir)
	if findFile(st.Unstaged, "tmp.log") != nil || findFile(st.Unstaged, ".gitignore") == nil {
		t.Fatalf("ignore: %+v", st.Unstaged)
	}
	git(t, dir, "add", ".gitignore")
	git(t, dir, "commit", "-q", "-m", "ignore")
	write(t, dir, "b.txt", "changed\n")
	write(t, dir, "d/e.txt", "x\n")
	if err := DiscardFiles(ctx, dir, nil, true); err != nil {
		t.Fatal(err)
	}
	st, _ = GetStatus(ctx, dir)
	if len(st.Unstaged) != 0 {
		t.Fatalf("discard all left %+v", st.Unstaged)
	}
}

func TestBranchesMergeConflict(t *testing.T) {
	ctx := context.Background()
	dir := newRepo(t)
	write(t, dir, "a.txt", "base\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")
	if err := CreateBranch(ctx, dir, "feature", "", true); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "a.txt", "feature\n")
	git(t, dir, "commit", "-q", "-am", "feature change")
	if _, err := Checkout(ctx, dir, "main"); err != nil {
		t.Fatal(err)
	}
	write(t, dir, "a.txt", "main\n")
	git(t, dir, "commit", "-q", "-am", "main change")
	if err := CreateTag(ctx, dir, "v1.0", "", "第一版"); err != nil {
		t.Fatal(err)
	}
	if err := CreateBranch(ctx, dir, "-bad", "", false); err == nil {
		t.Fatal("bad branch name accepted")
	}

	refs, err := ListRefs(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(refs.Branches) != 2 || !refs.Branches[0].Current || refs.Branches[0].Name != "main" {
		t.Fatalf("branches = %+v", refs.Branches)
	}
	if len(refs.Tags) != 1 || refs.Tags[0].Subject != "第一版" || refs.Tags[0].Commit == "" {
		t.Fatalf("tags = %+v", refs.Tags)
	}

	if _, err := Merge(ctx, dir, "feature"); err == nil {
		t.Fatal("conflicting merge succeeded")
	}
	st, _ := GetStatus(ctx, dir)
	if st.Operation != "merge" || st.OperationRef != "feature" || len(st.Conflicted) != 1 || st.Conflicted[0].Kind != "UU" {
		t.Fatalf("merge status: %+v", st)
	}
	cd, _ := GetDiff(ctx, dir, DiffConflict, "a.txt", "", "")
	if !strings.Contains(cd.Patch, " <<<<<<< HEAD") {
		t.Fatalf("conflict view = %q", cd.Patch)
	}
	if _, err := Sequence(ctx, dir, "skip"); err == nil {
		t.Fatal("merge skip accepted")
	}
	if err := ResolveFiles(ctx, dir, []string{"a.txt"}, "theirs"); err != nil {
		t.Fatal(err)
	}
	if _, err := Sequence(ctx, dir, "continue"); err != nil {
		t.Fatal(err)
	}
	st, _ = GetStatus(ctx, dir)
	if st.Operation != "" || read(t, dir, "a.txt") != "feature\n" {
		t.Fatalf("after merge: %+v", st)
	}

	// 标签切成游离 HEAD，再切回来；删分支
	if _, err := Checkout(ctx, dir, "v1.0"); err != nil {
		t.Fatal(err)
	}
	if st, _ = GetStatus(ctx, dir); st.Branch != "" {
		t.Fatalf("not detached: %+v", st)
	}
	if _, err := Push(ctx, dir, "", false, false); err == nil {
		t.Fatal("push on detached HEAD")
	}
	if _, err := Checkout(ctx, dir, "main"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteBranch(ctx, dir, "feature", false); err != nil {
		t.Fatal(err)
	}
	if _, err := Push(ctx, dir, "", false, false); err == nil || !strings.Contains(err.Error(), "远程") {
		t.Fatalf("push without remote: %v", err)
	}
}

func TestRemotesPushPull(t *testing.T) {
	ctx := context.Background()
	bare := t.TempDir()
	if _, err := Run(ctx, bare, "init", "-q", "--bare", "-b", "main"); err != nil {
		t.Skip(err)
	}
	dir := newRepo(t)
	write(t, dir, "a.txt", "1\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")
	if err := AddRemote(ctx, dir, "origin", bare); err != nil {
		t.Fatal(err)
	}
	if _, err := Push(ctx, dir, "", false, false); err != nil {
		t.Fatal(err)
	}
	st, _ := GetStatus(ctx, dir)
	if st.Upstream != "origin/main" || st.Ahead != 0 || len(st.Remotes) != 1 {
		t.Fatalf("after push: %+v", st)
	}
	write(t, dir, "a.txt", "2\n")
	git(t, dir, "commit", "-q", "-am", "second")
	if st, _ = GetStatus(ctx, dir); st.Ahead != 1 {
		t.Fatalf("ahead = %d", st.Ahead)
	}
	if _, err := Push(ctx, dir, "", false, false); err != nil {
		t.Fatal(err)
	}
	// 另一个克隆推一个提交，这边拉下来
	other := t.TempDir()
	git(t, other, "clone", "-q", bare, ".")
	git(t, other, "config", "user.name", "O")
	git(t, other, "config", "user.email", "o@example.com")
	write(t, other, "b.txt", "x\n")
	git(t, other, "add", ".")
	git(t, other, "commit", "-q", "-m", "other")
	git(t, other, "push", "-q")
	if _, err := Fetch(ctx, dir, ""); err != nil {
		t.Fatal(err)
	}
	if st, _ = GetStatus(ctx, dir); st.Behind != 1 {
		t.Fatalf("behind = %d", st.Behind)
	}
	if _, err := Pull(ctx, dir, ""); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(dir, "b.txt")) {
		t.Fatal("pull did not bring b.txt")
	}
	refs, _ := ListRefs(ctx, dir)
	if len(refs.RemoteBranches) != 1 || refs.RemoteBranches[0].Name != "origin/main" || len(refs.Remotes) != 1 || refs.Remotes[0].FetchURL == "" {
		t.Fatalf("refs = %+v", refs)
	}
	if err := RenameRemote(ctx, dir, "origin", "up"); err != nil {
		t.Fatal(err)
	}
	if err := RemoveRemote(ctx, dir, "up"); err != nil {
		t.Fatal(err)
	}
}

func TestStashLogCommit(t *testing.T) {
	ctx := context.Background()
	dir := newRepo(t)
	write(t, dir, "a.txt", "1\n")
	git(t, dir, "add", ".")
	git(t, dir, "commit", "-q", "-m", "init")
	git(t, dir, "mv", "a.txt", "b.txt")
	git(t, dir, "commit", "-q", "-m", "rename")

	write(t, dir, "b.txt", "2\n")
	write(t, dir, "new.txt", "n\n")
	if _, err := StashPush(ctx, dir, "半成品", true, false, nil); err != nil {
		t.Fatal(err)
	}
	list, err := ListStashes(ctx, dir)
	if err != nil || len(list) != 1 || list[0].Message != "半成品" || list[0].Branch != "main" {
		t.Fatalf("stashes = %+v %v", list, err)
	}
	st, _ := GetStatus(ctx, dir)
	if st.Stashes != 1 || len(st.Unstaged) != 0 {
		t.Fatalf("status after stash: %+v", st)
	}
	detail, err := GetCommit(ctx, dir, list[0].Hash, true)
	if err != nil || findFile(detail.Files, "b.txt") == nil || findFile(detail.Files, "new.txt") == nil {
		t.Fatalf("stash files = %+v %v", detail, err)
	}
	if d, _ := GetDiff(ctx, dir, DiffStash, "new.txt", "", list[0].Hash); !strings.Contains(d.Patch, "+n") {
		t.Fatalf("stash untracked diff = %q", d.Patch)
	}
	if _, err := StashApply(ctx, dir, list[0].Hash, true, false); err != nil {
		t.Fatal(err)
	}
	if read(t, dir, "new.txt") != "n\n" {
		t.Fatal("stash pop lost untracked file")
	}
	if err := StashDrop(ctx, dir, list[0].Hash); err == nil {
		t.Fatal("dropped a stash that is gone")
	}

	commits, more, err := Log(ctx, dir, LogOptions{Limit: 1})
	if err != nil || len(commits) != 1 || !more || commits[0].Subject != "rename" || len(commits[0].Refs) == 0 {
		t.Fatalf("log = %+v %v %v", commits, more, err)
	}
	commits, more, _ = Log(ctx, dir, LogOptions{Skip: 1, Limit: 5})
	if len(commits) != 1 || more || len(commits[0].Parents) != 0 {
		t.Fatalf("log page 2 = %+v", commits)
	}
	if found, _, _ := Log(ctx, dir, LogOptions{Keyword: "RENAME"}); len(found) != 1 {
		t.Fatalf("search = %+v", found)
	}
	git(t, dir, "reset", "-q", "--hard")
	git(t, dir, "clean", "-q", "-fd")
	head := strings.TrimSpace(git(t, dir, "rev-parse", "HEAD"))
	detail, err = GetCommit(ctx, dir, head, false)
	if err != nil || len(detail.Files) != 1 || detail.Files[0].Kind != "R" || detail.Files[0].OrigPath != "a.txt" || detail.Author != "Tester" {
		t.Fatalf("detail = %+v %v", detail, err)
	}
	// 根提交也能看 diff
	root := strings.TrimSpace(git(t, dir, "rev-list", "--max-parents=0", "HEAD"))
	if d, err := GetDiff(ctx, dir, DiffCommit, "a.txt", "", root); err != nil || !strings.Contains(d.Patch, "+1") {
		t.Fatalf("root diff = %+v %v", d, err)
	}
	if _, err := RevertCommit(ctx, dir, head); err != nil {
		t.Fatal(err)
	}
	if !exists(filepath.Join(dir, "a.txt")) {
		t.Fatal("revert did not bring a.txt back")
	}
}
