// Package agentgit 是 Agent 功能用到的 Git 操作：读仓库信息、克隆、建工作树、看未提交的改动、撤销文件。
// 读分支直接看 .git（列表里每行都要用，不起进程），其它操作调用服务器上的 git 命令，diff 也用 git 自己的输出。
package agentgit

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"momoko/pkg/agentfs"
)

// maxPatch 单个文件的改动最多带多少字节的 diff。
const maxPatch = 256 * 1024

// Info 读目录的 Git 信息：是不是仓库、当前分支（游离状态给短哈希）。
func Info(dir string) (isGit bool, branch string) {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return false, ""
	}
	headDir := gitPath
	if !info.IsDir() {
		// 工作树里的 .git 是一个文件：gitdir: <真正的目录>
		raw, err := os.ReadFile(gitPath)
		if err != nil {
			return false, ""
		}
		line := strings.TrimSpace(string(raw))
		if !strings.HasPrefix(line, "gitdir:") {
			return false, ""
		}
		headDir = strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
		if !filepath.IsAbs(headDir) {
			headDir = filepath.Join(dir, headDir)
		}
	}
	raw, err := os.ReadFile(filepath.Join(headDir, "HEAD"))
	if err != nil {
		return false, ""
	}
	head := strings.TrimSpace(string(raw))
	if ref, ok := strings.CutPrefix(head, "ref: refs/heads/"); ok {
		return true, ref
	}
	if len(head) >= 7 {
		return true, head[:7]
	}
	return true, ""
}

// Run 在 dir 里跑一条 git 命令，返回标准输出；失败时错误是 git 自己说的最后一句原因。
func Run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	// 不让 git 停下来问用户名密码：私有仓库用服务器上已经配好的凭证。
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0", "LC_ALL=C")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", errors.New("服务器上没有安装 git")
		}
		lines := strings.Split(strings.TrimSpace(stderr.String()), "\n")
		msg := strings.TrimSpace(lines[len(lines)-1])
		msg = strings.TrimPrefix(strings.TrimPrefix(msg, "fatal: "), "error: ")
		if msg == "" {
			msg = err.Error()
		}
		return "", errors.New(msg)
	}
	return stdout.String(), nil
}

func lines(out string) []string {
	var res []string
	for _, line := range strings.Split(out, "\n") {
		if line = strings.TrimSpace(line); line != "" {
			res = append(res, line)
		}
	}
	return res
}

// RepoName 从仓库地址里取仓库名。
func RepoName(gitURL string) string {
	s := strings.TrimRight(strings.TrimSpace(gitURL), "/")
	s = strings.TrimSuffix(s, ".git")
	if i := strings.LastIndexAny(s, "/:"); i >= 0 {
		s = s[i+1:]
	}
	return s
}

// Clone 把仓库克隆到 path；branch 为空用默认分支，depth 大于 0 时做浅克隆。
func Clone(ctx context.Context, gitURL, branch, path string, depth int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	args := []string{"clone"}
	if branch != "" {
		args = append(args, "--branch", branch)
	}
	if depth > 0 {
		args = append(args, "--depth", fmt.Sprint(depth))
	}
	args = append(args, gitURL, path)
	_, err := Run(ctx, filepath.Dir(path), args...)
	return err
}

// Worktree 一个新建的工作树。
type Worktree struct {
	Name string
	Path string
}

// AddWorktree 在 path 给仓库建一个新工作树：名字是 <slug>-wt<序号>，分支叫 momoko/<名字>，从 base（提交或分支）分出来。
func AddWorktree(ctx context.Context, repo, path, slug, base string) (*Worktree, error) {
	if isGit, _ := Info(repo); !isGit {
		return nil, errors.New("这个目录不是 Git 仓库，不能用独立工作树")
	}
	if _, err := os.Stat(path); err == nil {
		return nil, errors.New("工作树目录已经存在：" + path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, errors.New("建不了工作树目录：" + err.Error())
	}
	for n := 1; n < 1000; n++ {
		name := fmt.Sprintf("%s-wt%d", slug, n)
		_, err := Run(ctx, repo, "worktree", "add", "-b", "momoko/"+name, path, base)
		if err != nil {
			// 分支名被占了就换下一个序号，其它错误直接报。
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return nil, errors.New("建工作树失败：" + err.Error())
		}
		return &Worktree{Name: name, Path: path}, nil
	}
	return nil, errors.New("工作树太多了，先清理一些")
}

// Head 目录当前所在的提交。
func Head(ctx context.Context, dir string) (string, error) {
	out, err := Run(ctx, dir, "rev-parse", "HEAD")
	return strings.TrimSpace(out), err
}

// CopyChanges 把工作树 from 里没提交的改动（改过的、新加的、删掉的文件，不含被忽略的）照搬到 to。
// to 是从 from 的 HEAD 分出来的另一个工作树，搬完两边的文件一样。
func CopyChanges(ctx context.Context, from, to string) error {
	out, err := Run(ctx, from, "-c", "core.quotepath=off", "status", "--porcelain=v1", "-z", "--no-renames", "--untracked-files=all")
	if err != nil {
		return err
	}
	for _, entry := range strings.Split(out, "\x00") {
		// 每条是「XY 路径」：XY 是暂存区和工作区的状态
		if len(entry) < 4 {
			continue
		}
		rel := filepath.FromSlash(entry[3:])
		if strings.Contains(entry[:2], "D") {
			_ = os.Remove(filepath.Join(to, rel))
			continue
		}
		if err := agentfs.CopyFile(filepath.Join(from, rel), filepath.Join(to, rel)); err != nil {
			return err
		}
	}
	return nil
}

// RemoveWorktree 删掉 AddWorktree 建的工作树：目录连同没提交的改动一起删掉，再删分支。
// force 为 false 时分支只在已经合并过时才删，没合并的提交留在分支上。
func RemoveWorktree(ctx context.Context, repo string, wt *Worktree, force bool) {
	_, _ = Run(ctx, repo, "worktree", "remove", "--force", wt.Path)
	_ = os.RemoveAll(wt.Path)
	_, _ = Run(ctx, repo, "worktree", "prune")
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, _ = Run(ctx, repo, "branch", flag, "momoko/"+wt.Name)
}

// Change 一个文件的未提交改动。
type Change struct {
	Path string
	// Kind：M 修改，A 新增，D 删除。
	Kind    string
	Added   int
	Deleted int
	// Patch git 给的各段 diff（从 @@ 开始，不含文件头）；太大或不是文本时为空。
	Patch string
}

// Changes 目录相对 HEAD 的未提交改动：已跟踪文件的改动用 git diff，没跟踪的新文件整个算新增。
func Changes(ctx context.Context, dir string) []Change {
	diff, err := Run(ctx, dir, "-c", "core.quotepath=off", "diff", "HEAD", "--no-color", "--no-ext-diff", "--no-renames", "-U3")
	if err != nil {
		// 还没有任何提交的仓库没有 HEAD：只看暂存区。
		diff, _ = Run(ctx, dir, "-c", "core.quotepath=off", "diff", "--cached", "--no-color", "--no-ext-diff", "--no-renames", "-U3")
	}
	files := splitDiff(diff)

	untracked, _ := Run(ctx, dir, "-c", "core.quotepath=off", "ls-files", "--others", "--exclude-standard")
	for _, path := range lines(untracked) {
		if len(files) >= 500 {
			break
		}
		file := Change{Path: path, Kind: "A"}
		full := filepath.Join(dir, filepath.FromSlash(path))
		if info, err := os.Stat(full); err == nil && info.Size() > 0 && info.Size() <= maxPatch {
			if raw, err := os.ReadFile(full); err == nil && utf8.Valid(raw) && !bytes.ContainsRune(raw, 0) {
				text := strings.TrimSuffix(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
				added := strings.Split(text, "\n")
				file.Added = len(added)
				file.Patch = fmt.Sprintf("@@ -0,0 +1,%d @@\n+%s\n", len(added), strings.Join(added, "\n+"))
			}
		}
		files = append(files, file)
	}
	sort.SliceStable(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	return files
}

// splitDiff 把 git diff 的输出按文件拆开，数出每个文件的增删行数。
func splitDiff(diff string) []Change {
	var files []Change
	var cur *Change
	var patch strings.Builder
	inHunk := false
	flush := func() {
		if cur == nil {
			return
		}
		if patch.Len() <= maxPatch {
			cur.Patch = patch.String()
		}
		files = append(files, *cur)
		patch.Reset()
	}
	for _, line := range strings.Split(diff, "\n") {
		switch {
		case strings.HasPrefix(line, "diff --git "):
			flush()
			cur = &Change{Kind: "M"}
			inHunk = false
			if i := strings.Index(line, " b/"); i >= 0 {
				cur.Path = line[i+3:]
			}
		case cur == nil:
		case !inHunk && strings.HasPrefix(line, "new file mode"):
			cur.Kind = "A"
		case !inHunk && strings.HasPrefix(line, "deleted file mode"):
			cur.Kind = "D"
		case !inHunk && strings.HasPrefix(line, "+++ b/"):
			cur.Path = strings.TrimPrefix(line, "+++ b/")
		case strings.HasPrefix(line, "@@"):
			inHunk = true
			patch.WriteString(line + "\n")
		case inHunk && line != "":
			// 空字符串是输出末尾换行之后的那一段，不是一行（空的上下文行是一个空格）。
			if strings.HasPrefix(line, "+") {
				cur.Added++
			} else if strings.HasPrefix(line, "-") {
				cur.Deleted++
			}
			patch.WriteString(line + "\n")
		}
	}
	flush()
	return files
}

// Marks 仓库 dir 里子目录 sub（相对路径，空 = 整个仓库）下有未提交改动的文件：路径 → M（改过）/ A（新增）；
// 含有改动的目录也标上 M。只让 git 看这个子目录，不扫整个仓库。不是仓库时返回空表。
func Marks(ctx context.Context, dir, sub string) map[string]string {
	marks := map[string]string{}
	if isGit, _ := Info(dir); !isGit {
		return marks
	}
	spec := "."
	if sub = strings.Trim(filepath.ToSlash(sub), "/"); sub != "" {
		spec = sub
	}
	out, err := Run(ctx, dir, "-c", "core.quotepath=off", "status", "--porcelain", "--untracked-files=all", "--", spec)
	if err != nil {
		return marks
	}
	for _, line := range strings.Split(out, "\n") {
		if len(line) < 4 {
			continue
		}
		code, path := line[:2], strings.TrimSpace(line[3:])
		if i := strings.Index(path, " -> "); i >= 0 {
			path = path[i+4:]
		}
		mark := "M"
		if code == "??" || strings.Contains(code, "A") {
			mark = "A"
		}
		path = strings.Trim(path, `"`)
		marks[path] = mark
		for d := filepath.ToSlash(filepath.Dir(path)); d != "." && d != "/" && marks[d] == ""; d = filepath.ToSlash(filepath.Dir(d)) {
			marks[d] = "M"
		}
	}
	return marks
}

// Revert 撤销一个文件的改动：已跟踪的恢复到 HEAD，没跟踪的新文件删掉。rel 是相对仓库目录的路径。
func Revert(ctx context.Context, dir, rel string) error {
	if isGit, _ := Info(dir); !isGit {
		return errors.New("这个目录不是 Git 仓库，撤销不了")
	}
	rel = filepath.ToSlash(rel)
	if _, err := Run(ctx, dir, "ls-files", "--error-unmatch", "--", rel); err != nil {
		if err := os.Remove(filepath.Join(dir, filepath.FromSlash(rel))); err != nil && !os.IsNotExist(err) {
			return errors.New("删除失败：" + err.Error())
		}
		return nil
	}
	if _, err := Run(ctx, dir, "checkout", "HEAD", "--", rel); err != nil {
		return errors.New("撤销失败：" + err.Error())
	}
	return nil
}

// Files 仓库里的文件（已跟踪的加上没被忽略的新文件），相对路径。不是仓库时 ok 为 false。
func Files(ctx context.Context, dir string) (files []string, ok bool) {
	if isGit, _ := Info(dir); !isGit {
		return nil, false
	}
	out, err := Run(ctx, dir, "-c", "core.quotepath=off", "ls-files", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, false
	}
	return lines(out), true
}
