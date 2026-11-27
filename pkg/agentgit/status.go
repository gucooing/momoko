package agentgit

// 这里是右侧「Git」标签读的东西：仓库状态（分支、上游、进行中的操作、暂存的 / 没暂存的 / 冲突的文件）、
// 各种来源的 diff，以及按行暂存时拼出来的部分补丁。

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"unicode/utf8"
)

const (
	// maxStatusFiles 一个仓库最多列多少个改动的文件（三类加起来），多出来的不列，标上 Truncated。
	maxStatusFiles = 3000
	// maxUntrackedCount 没跟踪的文件最多数多少个的行数（要读文件）。
	maxUntrackedCount = 300
	// maxFullPatch 「复制补丁」最多给多少字节。
	maxFullPatch = 8 << 20
)

// runCmd 跑一条 git 命令，stdin 不为空时喂给它，返回标准输出和标准错误。
// 失败时的错误带上 git 说的最后几行（钩子的输出、冲突的说明都在里面）。
func runCmd(ctx context.Context, dir, stdin string, args ...string) (string, string, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	cmd.Env = gitEnv()
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := execute(ctx, cmd); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return "", "", errors.New("服务器上没有安装 git")
		}
		if ctx.Err() != nil {
			return stdout.String(), stderr.String(), errors.New("git 运行超时了")
		}
		msg := tailLines(stderr.String(), 12)
		if msg == "" {
			msg = tailLines(stdout.String(), 12)
		}
		if msg == "" {
			msg = err.Error()
		}
		return stdout.String(), stderr.String(), errors.New(msg)
	}
	return stdout.String(), stderr.String(), nil
}

// runOut 只要标准输出的 runCmd。
func runOut(ctx context.Context, dir string, args ...string) (string, error) {
	out, _, err := runCmd(ctx, dir, "", args...)
	return out, err
}

// runSay 跑一条会改东西的命令，返回 git 说的话（标准输出和标准错误的最后几行），给页面提示用。
func runSay(ctx context.Context, dir, stdin string, args ...string) (string, error) {
	out, errOut, err := runCmd(ctx, dir, stdin, args...)
	if err != nil {
		return "", err
	}
	return tailLines(out+"\n"+errOut, 12), nil
}

// tailLines 输出的最后 n 行：去掉空行、hint 行和 fatal: / error: 前缀；进度条用 \r 刷新的只留最后一段。
func tailLines(out string, n int) string {
	var keep []string
	for _, line := range strings.Split(strings.ReplaceAll(out, "\r\n", "\n"), "\n") {
		if i := strings.LastIndex(line, "\r"); i >= 0 {
			line = line[i+1:]
		}
		line = strings.TrimRight(line, " ")
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "hint:") {
			continue
		}
		line = strings.TrimPrefix(strings.TrimPrefix(line, "fatal: "), "error: ")
		keep = append(keep, line)
	}
	if len(keep) > n {
		keep = keep[len(keep)-n:]
	}
	return strings.Join(keep, "\n")
}

// gitDirOf 仓库自己的 .git 目录（工作树里 .git 是一个文件，指向它自己的那个目录）；不是仓库时为空。
func gitDirOf(dir string) string {
	gitPath := filepath.Join(dir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return ""
	}
	if info.IsDir() {
		return gitPath
	}
	raw, err := os.ReadFile(gitPath)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(raw))
	if !strings.HasPrefix(line, "gitdir:") {
		return ""
	}
	gd := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
	if !filepath.IsAbs(gd) {
		gd = filepath.Join(dir, gd)
	}
	return gd
}

func readTrim(path string) string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(raw))
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func short(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

// File 改动列表里的一个文件。
type File struct {
	// Path 相对仓库的路径（正斜杠）。
	Path string
	// OrigPath 改名、复制前的路径。
	OrigPath string
	// Kind M 修改 | A 新增 | D 删除 | R 改名 | C 复制 | T 类型变了 | U 没跟踪；冲突的是两边的状态（UU、AA、DU……）。
	Kind    string
	Added   int
	Deleted int
	Binary  bool
}

// Status 一个仓库的状态。
type Status struct {
	// Branch 当前分支；游离 HEAD 时为空。
	Branch string
	// Head HEAD 的短哈希；还没有提交时为空。
	Head         string
	Upstream     string
	UpstreamGone bool
	Ahead        int
	Behind       int
	// Operation 进行中的操作：merge | rebase | am | cherry-pick | revert | bisect，没有为空。
	Operation string
	// OperationRef 合并进来的分支、拣选 / 还原的提交，或者正在变基的分支。
	OperationRef string
	// OperationOnto 变基到的提交。
	OperationOnto  string
	OperationStep  int
	OperationTotal int
	Staged         []File
	Unstaged       []File
	Conflicted     []File
	Stashes        int
	Remotes        []string
	Truncated      bool
	// LastMessage 上一次提交的完整说明（修改上一次提交时放进输入框）。
	LastMessage string
}

// GetStatus 读仓库的状态：一次 git status 拿到分支和文件，另外几条（增删行数、远程、上一次提交的说明）同时跑。
func GetStatus(ctx context.Context, dir string) (*Status, error) {
	out, err := runOut(ctx, dir, "-c", "core.quotepath=off", "status", "--porcelain=v2", "--branch", "--show-stash", "-z", "--untracked-files=all")
	if err != nil {
		return nil, err
	}
	st := &Status{}
	var untracked []string
	ab := false
	fields := strings.Split(out, "\x00")
	count := 0
	add := func(list *[]File, f File) {
		if count >= maxStatusFiles {
			st.Truncated = true
			return
		}
		count++
		*list = append(*list, f)
	}
	for i := 0; i < len(fields); i++ {
		f := fields[i]
		if f == "" {
			continue
		}
		switch f[0] {
		case '#':
			key, val, _ := strings.Cut(strings.TrimPrefix(f, "# "), " ")
			switch key {
			case "branch.oid":
				if val != "(initial)" {
					st.Head = short(val)
				}
			case "branch.head":
				if val != "(detached)" {
					st.Branch = val
				}
			case "branch.upstream":
				st.Upstream = val
			case "branch.ab":
				ab = true
				a, b, _ := strings.Cut(val, " ")
				st.Ahead, _ = strconv.Atoi(strings.TrimPrefix(a, "+"))
				st.Behind, _ = strconv.Atoi(strings.TrimPrefix(b, "-"))
			case "stash":
				st.Stashes, _ = strconv.Atoi(val)
			}
		case '1', '2':
			// 1 XY sub mH mI mW hH hI path；2 XY sub mH mI mW hH hI Xscore path\0origPath
			n := 9
			if f[0] == '2' {
				n = 10
			}
			parts := strings.SplitN(f, " ", n)
			if len(parts) < n {
				continue
			}
			xy, path := parts[1], parts[n-1]
			orig := ""
			if f[0] == '2' && i+1 < len(fields) {
				orig = fields[i+1]
				i++
			}
			if x := xy[0]; x != '.' {
				add(&st.Staged, File{Path: path, OrigPath: orig, Kind: string(x)})
			}
			if y := xy[1]; y != '.' {
				add(&st.Unstaged, File{Path: path, Kind: string(y)})
			}
		case 'u':
			// u XY sub m1 m2 m3 mW h1 h2 h3 path
			parts := strings.SplitN(f, " ", 11)
			if len(parts) < 11 {
				continue
			}
			add(&st.Conflicted, File{Path: parts[10], Kind: parts[1]})
		case '?':
			path := strings.TrimPrefix(f, "? ")
			untracked = append(untracked, path)
			add(&st.Unstaged, File{Path: path, Kind: "U"})
		}
	}
	// 有上游却没有领先落后：上游分支已经没了
	st.UpstreamGone = st.Upstream != "" && !ab

	var wg sync.WaitGroup
	var staged, unstaged map[string]numstat
	wg.Add(4)
	go func() {
		defer wg.Done()
		if len(st.Staged) > 0 {
			o, _ := runOut(ctx, dir, "-c", "core.quotepath=off", "diff", "--cached", "--numstat", "-z", "-M", "--no-ext-diff")
			staged = parseNumstat(o)
		}
	}()
	go func() {
		defer wg.Done()
		if len(st.Unstaged) > len(untracked) {
			o, _ := runOut(ctx, dir, "-c", "core.quotepath=off", "diff", "--numstat", "-z", "--no-ext-diff")
			unstaged = parseNumstat(o)
		}
	}()
	go func() {
		defer wg.Done()
		o, _ := runOut(ctx, dir, "remote")
		st.Remotes = lines(o)
	}()
	go func() {
		defer wg.Done()
		if st.Head != "" {
			o, _ := runOut(ctx, dir, "log", "-1", "--format=%B")
			st.LastMessage = strings.TrimSpace(o)
		}
	}()
	wg.Wait()
	for i := range st.Staged {
		if n, ok := staged[st.Staged[i].Path]; ok {
			st.Staged[i].Added, st.Staged[i].Deleted, st.Staged[i].Binary = n.added, n.deleted, n.binary
		}
	}
	counted := 0
	for i := range st.Unstaged {
		f := &st.Unstaged[i]
		if f.Kind != "U" {
			if n, ok := unstaged[f.Path]; ok {
				f.Added, f.Deleted, f.Binary = n.added, n.deleted, n.binary
			}
			continue
		}
		if counted >= maxUntrackedCount {
			continue
		}
		counted++
		f.Added, f.Binary = countLines(filepath.Join(dir, filepath.FromSlash(f.Path)))
	}
	operationOf(ctx, dir, st)
	return st, nil
}

type numstat struct {
	added, deleted int
	binary         bool
}

// parseNumstat git diff --numstat -z 的输出：「增\t删\t路径\0」，改名的是「增\t删\t\0旧路径\0新路径\0」，二进制文件的增删是 -。
func parseNumstat(out string) map[string]numstat {
	res := map[string]numstat{}
	parts := strings.Split(out, "\x00")
	for i := 0; i < len(parts); i++ {
		cols := strings.SplitN(parts[i], "\t", 3)
		if len(cols) < 3 {
			continue
		}
		path := cols[2]
		if path == "" && i+2 < len(parts) {
			path = parts[i+2]
			i += 2
		}
		a, errA := strconv.Atoi(cols[0])
		d, _ := strconv.Atoi(cols[1])
		res[path] = numstat{added: a, deleted: d, binary: errA != nil}
	}
	return res
}

// countLines 新文件有几行；不是文本（或太大）时 binary 为 true。
func countLines(full string) (int, bool) {
	info, err := os.Stat(full)
	if err != nil || info.IsDir() {
		return 0, false
	}
	if info.Size() > maxPatch {
		return 0, true
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return 0, false
	}
	if !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 {
		return 0, true
	}
	if len(raw) == 0 {
		return 0, false
	}
	n := bytes.Count(raw, []byte("\n"))
	if raw[len(raw)-1] != '\n' {
		n++
	}
	return n, false
}

// operationOf 从 .git 目录里的标记文件看出进行中的合并、变基、拣选、还原。
func operationOf(ctx context.Context, dir string, st *Status) {
	gd := gitDirOf(dir)
	if gd == "" {
		return
	}
	atoi := func(s string) int { n, _ := strconv.Atoi(s); return n }
	rebaseIn := func(sub, step, total string) {
		base := filepath.Join(gd, sub)
		st.OperationRef = strings.TrimPrefix(readTrim(filepath.Join(base, "head-name")), "refs/heads/")
		st.OperationOnto = short(readTrim(filepath.Join(base, "onto")))
		st.OperationStep = atoi(readTrim(filepath.Join(base, step)))
		st.OperationTotal = atoi(readTrim(filepath.Join(base, total)))
	}
	firstLine := func(name string) string {
		line, _, _ := strings.Cut(readTrim(filepath.Join(gd, name)), "\n")
		return strings.TrimSpace(line)
	}
	switch {
	case exists(filepath.Join(gd, "rebase-merge")):
		st.Operation = "rebase"
		rebaseIn("rebase-merge", "msgnum", "end")
	case exists(filepath.Join(gd, "rebase-apply")):
		st.Operation = "rebase"
		if exists(filepath.Join(gd, "rebase-apply", "applying")) {
			st.Operation = "am"
		}
		rebaseIn("rebase-apply", "next", "last")
	case exists(filepath.Join(gd, "MERGE_HEAD")):
		st.Operation = "merge"
		hash := firstLine("MERGE_HEAD")
		st.OperationRef = short(hash)
		if name, err := runOut(ctx, dir, "name-rev", "--name-only", "--no-undefined", hash); err == nil {
			st.OperationRef = strings.TrimPrefix(strings.TrimSpace(name), "remotes/")
		}
	case exists(filepath.Join(gd, "CHERRY_PICK_HEAD")):
		st.Operation = "cherry-pick"
		st.OperationRef = short(firstLine("CHERRY_PICK_HEAD"))
	case exists(filepath.Join(gd, "REVERT_HEAD")):
		st.Operation = "revert"
		st.OperationRef = short(firstLine("REVERT_HEAD"))
	case exists(filepath.Join(gd, "BISECT_LOG")):
		st.Operation = "bisect"
	}
}

// —— diff ——

// diff 的来源
const (
	// DiffUnstaged 工作区相对暂存区（没跟踪的新文件整个算新增）。
	DiffUnstaged = "unstaged"
	// DiffStaged 暂存区相对 HEAD。
	DiffStaged = "staged"
	// DiffConflict 冲突的文件现在的内容（带着冲突标记），全部当上下文行。
	DiffConflict = "conflict"
	// DiffCommit 某次提交相对它的第一个父提交。
	DiffCommit = "commit"
	// DiffStash 某个储藏（没跟踪的文件在它的第三个父提交里）。
	DiffStash = "stash"
	// DiffHead 整个仓库相对 HEAD 的完整补丁（带文件头，复制用）。
	DiffHead = "head"
)

// DiffResult 一个文件的 diff。
type DiffResult struct {
	// Patch 从 @@ 开始的各段（DiffHead 是带文件头的完整补丁）。
	Patch    string
	Binary   bool
	TooLarge bool
}

var hashRe = regexp.MustCompile(`^[0-9a-fA-F]{4,64}$`)

// IsHash 是不是一个提交哈希（完整或缩写）。
func IsHash(s string) bool { return hashRe.MatchString(s) }

// GetDiff 读一个文件的 diff；orig 是改名前的路径（暂存区里的改名、提交里的改名要两个路径一起给 git 才认得出）。
func GetDiff(ctx context.Context, dir, source, path, orig, ref string) (*DiffResult, error) {
	if source == DiffHead {
		return fullPatch(ctx, dir)
	}
	if path == "" {
		return nil, errors.New("缺少文件路径")
	}
	paths := []string{path}
	if orig != "" && orig != path {
		paths = []string{orig, path}
	}
	base := []string{"-c", "core.quotepath=off"}
	opts := []string{"--no-color", "--no-ext-diff", "-U3"}
	var out string
	var err error
	switch source {
	case DiffUnstaged:
		if !tracked(ctx, dir, path) {
			return newFilePatch(filepath.Join(dir, filepath.FromSlash(path)), "+"), nil
		}
		out, err = runOut(ctx, dir, append(append(append(base, "diff"), opts...), append([]string{"--"}, paths...)...)...)
	case DiffStaged:
		out, err = runOut(ctx, dir, append(append(append(base, "diff", "--cached", "-M"), opts...), append([]string{"--"}, paths...)...)...)
	case DiffConflict:
		return newFilePatch(filepath.Join(dir, filepath.FromSlash(path)), " "), nil
	case DiffCommit:
		if !IsHash(ref) {
			return nil, errors.New("提交哈希不对")
		}
		args := append(append(base, "show", "--format=", "-M", "--diff-merges=first-parent"), opts...)
		out, err = runOut(ctx, dir, append(append(args, ref, "--"), paths...)...)
	case DiffStash:
		if !IsHash(ref) {
			return nil, errors.New("储藏的哈希不对")
		}
		args := append(append(base, "diff", "-M"), opts...)
		out, err = runOut(ctx, dir, append(append(args, ref+"^1", ref, "--"), paths...)...)
		if err == nil && strings.TrimSpace(out) == "" {
			// 储藏里没跟踪的文件
			if _, e := runOut(ctx, dir, "rev-parse", "--verify", "-q", ref+"^3"); e == nil {
				args := append(append(base, "show", "--format="), opts...)
				out, err = runOut(ctx, dir, append(append(args, ref+"^3", "--"), paths...)...)
			}
		}
	default:
		return nil, errors.New("不认识的 diff 来源：" + source)
	}
	if err != nil {
		return nil, err
	}
	return hunksOf(out), nil
}

// tracked 文件是不是已经跟踪着（在暂存区里）。
func tracked(ctx context.Context, dir, path string) bool {
	_, err := runOut(ctx, dir, "ls-files", "--error-unmatch", "--", path)
	return err == nil
}

// hunksOf 从 git diff 的输出里取出第一个文件从 @@ 开始的各段。
func hunksOf(out string) *DiffResult {
	r := &DiffResult{}
	start := -1
	if strings.HasPrefix(out, "@@") {
		start = 0
	} else if i := strings.Index(out, "\n@@"); i >= 0 {
		start = i + 1
	}
	if start < 0 {
		r.Binary = strings.Contains(out, "Binary files ") || strings.Contains(out, "GIT binary patch")
		return r
	}
	body := out[start:]
	if i := strings.Index(body, "\ndiff --git "); i >= 0 {
		body = body[:i+1]
	}
	if len(body) > maxPatch {
		r.TooLarge = true
		return r
	}
	r.Patch = body
	return r
}

// newFilePatch 把整个文件写成一段 diff：sign 是 + 时整个算新增，是空格时全是上下文行（看冲突的文件）。
func newFilePatch(full, sign string) *DiffResult {
	r := &DiffResult{}
	info, err := os.Stat(full)
	if err != nil || info.IsDir() || info.Size() == 0 {
		return r
	}
	if info.Size() > maxPatch {
		r.TooLarge = true
		return r
	}
	raw, err := os.ReadFile(full)
	if err != nil {
		return r
	}
	if !utf8.Valid(raw) || bytes.IndexByte(raw, 0) >= 0 {
		r.Binary = true
		return r
	}
	text := strings.TrimSuffix(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")
	rows := strings.Split(text, "\n")
	if sign == "+" {
		r.Patch = fmt.Sprintf("@@ -0,0 +1,%d @@\n+%s\n", len(rows), strings.Join(rows, "\n+"))
	} else {
		r.Patch = fmt.Sprintf("@@ -1,%d +1,%d @@\n %s\n", len(rows), len(rows), strings.Join(rows, "\n "))
	}
	return r
}

// fullPatch 整个仓库相对 HEAD 的补丁（已跟踪的改动加上没跟踪的新文件），带文件头。
func fullPatch(ctx context.Context, dir string) (*DiffResult, error) {
	diff, err := runOut(ctx, dir, "-c", "core.quotepath=off", "diff", "HEAD", "--no-color", "--no-ext-diff", "-M")
	if err != nil {
		// 还没有任何提交：看暂存区
		diff, _ = runOut(ctx, dir, "-c", "core.quotepath=off", "diff", "--cached", "--no-color", "--no-ext-diff", "-M")
	}
	var b strings.Builder
	b.WriteString(diff)
	others, _ := runOut(ctx, dir, "-c", "core.quotepath=off", "ls-files", "--others", "--exclude-standard")
	for _, path := range lines(others) {
		if b.Len() > maxFullPatch {
			return &DiffResult{TooLarge: true}, nil
		}
		p := newFilePatch(filepath.Join(dir, filepath.FromSlash(path)), "+")
		name := quoteName(path)
		fmt.Fprintf(&b, "diff --git a/%s b/%s\nnew file mode 100644\n", name, name)
		switch {
		case p.Binary || p.TooLarge:
			fmt.Fprintf(&b, "Binary files /dev/null and b/%s differ\n", name)
		case p.Patch != "":
			fmt.Fprintf(&b, "--- /dev/null\n+++ b/%s%s\n%s", name, nameTab(path), p.Patch)
		}
	}
	if b.Len() > maxFullPatch {
		return &DiffResult{TooLarge: true}, nil
	}
	return &DiffResult{Patch: b.String()}, nil
}

// quoteName 补丁文件头里的路径：有引号、反斜杠、控制字符时按 git 的规矩加引号转义。
func quoteName(path string) string {
	if !strings.ContainsAny(path, "\"\\\t\n\r") {
		return path
	}
	var b strings.Builder
	b.WriteByte('"')
	for _, r := range path {
		switch r {
		case '"', '\\':
			b.WriteByte('\\')
			b.WriteRune(r)
		case '\t':
			b.WriteString(`\t`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		default:
			b.WriteRune(r)
		}
	}
	b.WriteByte('"')
	return b.String()
}

// nameTab --- / +++ 行里带空格的路径后面要跟一个制表符，git apply 才知道路径到哪结束。
func nameTab(path string) string {
	if strings.Contains(path, " ") {
		return "\t"
	}
	return ""
}

// —— 按行暂存 ——

// 按行操作
const (
	// LinesStage 从工作区的 diff 里把选中的行暂存。
	LinesStage = "stage"
	// LinesUnstage 从暂存区的 diff 里把选中的行取消暂存。
	LinesUnstage = "unstage"
	// LinesDiscard 撤销工作区里选中的这几行改动。
	LinesDiscard = "discard"
)

var hunkRe = regexp.MustCompile(`^@@ -(\d+)(?:,(\d+))? \+(\d+)(?:,(\d+))? @@`)

// partialPatch 从页面上那份 diff（从 @@ 开始）里只留选中的 + / - 行，拼成能交给 git apply 的补丁。
// selected 是 diff 按 \n 拆开后的行号。没选中的行：正向（暂存）时 - 行变成上下文、+ 行丢掉；
// 反向（取消暂存、撤销）时反过来，+ 行变成上下文、- 行丢掉。一行都没选中时 ok 为 false。
func partialPatch(path, patch string, selected map[int]bool, reverse bool) (string, bool) {
	rows := strings.Split(patch, "\n")
	name := quoteName(path)
	var b strings.Builder
	fmt.Fprintf(&b, "diff --git a/%s b/%s\n--- a/%s%s\n+++ b/%s%s\n", name, name, name, nameTab(path), name, nameTab(path))
	any := false
	delta := 0
	for i := 0; i < len(rows); {
		m := hunkRe.FindStringSubmatch(rows[i])
		if m == nil {
			i++
			continue
		}
		oldStart, _ := strconv.Atoi(m[1])
		newStart, _ := strconv.Atoi(m[3])
		var body []string
		oldN, newN := 0, 0
		changed := false
		// 上一行留下了没有：「\ No newline at end of file」跟着它说的那一行走
		kept := true
		for i++; i < len(rows) && !strings.HasPrefix(rows[i], "@@"); i++ {
			r := rows[i]
			if r == "" {
				continue
			}
			switch r[0] {
			case ' ':
				body = append(body, r)
				oldN++
				newN++
				kept = true
			case '\\':
				if kept {
					body = append(body, r)
				}
			case '+', '-':
				switch {
				case selected[i]:
					body = append(body, r)
					if r[0] == '+' {
						newN++
					} else {
						oldN++
					}
					changed, kept = true, true
				case (r[0] == '-') != reverse:
					body = append(body, " "+r[1:])
					oldN++
					newN++
					kept = true
				default:
					kept = false
				}
			}
		}
		if !changed {
			continue
		}
		any = true
		// 正向时旧的一边就是 git 要找的位置，新的一边跟着前面几段的增减走；反向时反过来
		if reverse {
			oldStart = newStart - delta
		} else {
			newStart = oldStart + delta
		}
		delta += newN - oldN
		fmt.Fprintf(&b, "@@ -%d,%d +%d,%d @@\n%s\n", oldStart, oldN, newStart, newN, strings.Join(body, "\n"))
	}
	return b.String(), any
}

// ApplyLines 按行暂存、取消暂存或撤销。patch 是页面上显示的那份 diff，lines 是选中的行号；
// 文件在这期间又改过的话上下文对不上，git 会拒绝，不会改错地方。
func ApplyLines(ctx context.Context, dir, path, patch string, lines []int, action string) error {
	verb := map[string]string{LinesStage: "暂存", LinesUnstage: "取消暂存", LinesDiscard: "撤销"}[action]
	if verb == "" {
		return errors.New("不认识的操作：" + action)
	}
	sel := map[int]bool{}
	for _, l := range lines {
		sel[l] = true
	}
	reverse := action != LinesStage
	text, ok := partialPatch(path, patch, sel, reverse)
	if !ok {
		return errors.New("没有选中要" + verb + "的行")
	}
	args := []string{"apply", "--recount", "--whitespace=nowarn"}
	if action != LinesDiscard {
		args = append(args, "--cached")
	}
	if reverse {
		args = append(args, "--reverse")
	}
	args = append(args, "-")
	if _, _, err := runCmd(ctx, dir, text, args...); err != nil {
		return fmt.Errorf("这几行没能%s，文件可能刚刚又改过，刷新后再试（%s）", verb, err.Error())
	}
	return nil
}
