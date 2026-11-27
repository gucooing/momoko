// Package agentgit 是 Agent 功能用到的 Git 操作：读仓库信息、克隆、建工作树，以及右侧「Git」标签里的状态、diff、暂存、提交、分支、同步、储藏、历史。
// 读分支直接看 .git（列表里每行都要用，不起进程），其它操作调用服务器上的 git 命令，diff 也用 git 自己的输出。
package agentgit

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"momoko/pkg/agentfs"
)

// maxPatch 单个文件的改动最多带多少字节的 diff。
const maxPatch = 256 * 1024

// Info 读目录的 Git 信息：是不是仓库、当前分支（游离状态给短哈希）。
func Info(dir string) (isGit bool, branch string) {
	headDir := gitDirOf(dir)
	if headDir == "" {
		return false, ""
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

// gitEnv 跑 git 用的环境：不停下来问任何东西。用户名密码用服务器上已经配好的凭证（凭证管理器也不弹窗）；
// 要写说明的地方（合并、继续变基、还原）直接用默认的说明，不打开编辑器。
func gitEnv() []string {
	return append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0", "GCM_INTERACTIVE=never", "LC_ALL=C",
		"GIT_EDITOR=:", "GIT_SEQUENCE_EDITOR=:", "GIT_MERGE_AUTOEDIT=no",
	)
}

// Run 在 dir 里跑一条 git 命令，返回标准输出；失败时错误是 git 自己说的最后一句原因。
func Run(ctx context.Context, dir string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv()
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

// Repos 找 root 下面 maxDepth 层以内的 Git 仓库（root 自己不算），返回相对路径（正斜杠），按路径排序，最多 limit 个。
// 找到一个仓库就不再往它里面翻；隐藏目录和依赖、产物目录不翻。给本身不是仓库、但下面放着几个仓库的会话目录用。
func Repos(root string, maxDepth, limit int) []string {
	var repos []string
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || !d.IsDir() || path == root {
			return nil
		}
		if len(repos) >= limit {
			return filepath.SkipAll
		}
		name := d.Name()
		if strings.HasPrefix(name, ".") {
			return filepath.SkipDir
		}
		switch name {
		case "node_modules", "vendor", "dist", "build", "target", "__pycache__":
			return filepath.SkipDir
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return filepath.SkipDir
		}
		if isGit, _ := Info(path); isGit {
			repos = append(repos, filepath.ToSlash(rel))
			return filepath.SkipDir
		}
		if strings.Count(rel, string(filepath.Separator))+1 >= maxDepth {
			return filepath.SkipDir
		}
		return nil
	})
	sort.Strings(repos)
	return repos
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
