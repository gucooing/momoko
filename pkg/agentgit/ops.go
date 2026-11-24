package agentgit

// 右侧「Git」标签里会改东西的操作：暂存、撤销、解决冲突、提交、重置、分支和标签、远程、储藏、拣选还原，
// 以及历史和提交详情。名字、路径都放在 -- 后面或者先检查过不以 - 开头，不会被当成 git 的选项。

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

// checkName 分支、标签、远程这类名字：不能空、不能以 - 开头、不能带空白和控制字符。
func checkName(kind, name string) error {
	if name == "" {
		return errors.New(kind + "名不能为空")
	}
	if strings.HasPrefix(name, "-") || strings.ContainsAny(name, " \t\r\n\x00") {
		return errors.New(kind + "名不对：" + name)
	}
	return nil
}

// checkRev 提交、分支这类引用（HEAD~1、origin/main、哈希）。
func checkRev(rev string) error {
	if rev == "" || strings.HasPrefix(rev, "-") || strings.ContainsAny(rev, " \t\r\n\x00") {
		return errors.New("提交写得不对：" + rev)
	}
	return nil
}

func hasHead(ctx context.Context, dir string) bool {
	_, err := runOut(ctx, dir, "rev-parse", "--verify", "-q", "HEAD")
	return err == nil
}

func refExists(ctx context.Context, dir, ref string) bool {
	_, err := runOut(ctx, dir, "show-ref", "--verify", "-q", ref)
	return err == nil
}

// —— 文件 ——

// StageFiles 暂存（新文件、删掉的文件也算）；all 时暂存全部。
func StageFiles(ctx context.Context, dir string, paths []string, all bool) error {
	if all {
		_, err := runOut(ctx, dir, "add", "-A")
		return err
	}
	_, err := runOut(ctx, dir, append([]string{"add", "-A", "--"}, paths...)...)
	return err
}

// UnstageFiles 取消暂存，工作区里的文件不动；all 时取消全部。还没有提交的仓库从暂存区里拿掉。
func UnstageFiles(ctx context.Context, dir string, paths []string, all bool) error {
	if !hasHead(ctx, dir) {
		args := []string{"rm", "--cached", "-r", "-q", "--ignore-unmatch", "--"}
		if all {
			args = append(args, ".")
		} else {
			args = append(args, paths...)
		}
		_, err := runOut(ctx, dir, args...)
		return err
	}
	if all {
		_, err := runOut(ctx, dir, "reset", "-q")
		return err
	}
	_, err := runOut(ctx, dir, append([]string{"restore", "--staged", "--"}, paths...)...)
	return err
}

// DiscardFiles 撤销工作区里的改动：跟踪着的恢复成暂存区里的样子，没跟踪的新文件删掉；all 时全部。
// 没跟踪的是另一个 Git 仓库（整个目录）时不删。
func DiscardFiles(ctx context.Context, dir string, paths []string, all bool) error {
	if all {
		if changed, _ := runOut(ctx, dir, "ls-files", "-m", "-d"); strings.TrimSpace(changed) != "" {
			if _, err := runOut(ctx, dir, "restore", "--worktree", "--", "."); err != nil {
				return err
			}
		}
		_, err := runOut(ctx, dir, "clean", "-f", "-d", "-q")
		return err
	}
	out, err := runOut(ctx, dir, append([]string{"-c", "core.quotepath=off", "ls-files", "-z", "--"}, paths...)...)
	if err != nil {
		return err
	}
	known := map[string]bool{}
	for _, p := range strings.Split(out, "\x00") {
		if p != "" {
			known[p] = true
		}
	}
	var keep []string
	for _, p := range paths {
		if known[p] {
			keep = append(keep, p)
			continue
		}
		full := filepath.Join(dir, filepath.FromSlash(strings.TrimSuffix(p, "/")))
		if isGit, _ := Info(full); isGit {
			return errors.New(p + " 是另一个 Git 仓库，不删")
		}
		if err := os.RemoveAll(full); err != nil {
			return errors.New("删除失败：" + err.Error())
		}
	}
	if len(keep) > 0 {
		if _, err := runOut(ctx, dir, append([]string{"restore", "--worktree", "--"}, keep...)...); err != nil {
			return err
		}
	}
	return nil
}

// ResolveFiles 处理冲突的文件：side 为 ours / theirs 时整个文件采用当前的 / 传入的那一边（那一边把它删了就删掉），
// 为空时按现在的样子标记为已解决（文件没了就从暂存区拿掉）。
func ResolveFiles(ctx context.Context, dir string, paths []string, side string) error {
	for _, p := range paths {
		if side == "ours" || side == "theirs" {
			if _, err := runOut(ctx, dir, "checkout", "--"+side, "--", p); err != nil {
				if !strings.Contains(err.Error(), "does not have") {
					return err
				}
				if _, err := runOut(ctx, dir, "rm", "-q", "--", p); err != nil {
					return err
				}
				continue
			}
		}
		if exists(filepath.Join(dir, filepath.FromSlash(p))) {
			if _, err := runOut(ctx, dir, "add", "--", p); err != nil {
				return err
			}
		} else if _, err := runOut(ctx, dir, "rm", "--cached", "-q", "--ignore-unmatch", "--", p); err != nil {
			return err
		}
	}
	return nil
}

// IgnoreFiles 把文件加进仓库根目录的 .gitignore（按完整路径，只忽略这一个）。
func IgnoreFiles(dir string, paths []string) error {
	gi := filepath.Join(dir, ".gitignore")
	raw, err := os.ReadFile(gi)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	text := string(raw)
	have := map[string]bool{}
	for _, l := range strings.Split(strings.ReplaceAll(text, "\r\n", "\n"), "\n") {
		have[strings.TrimSpace(l)] = true
	}
	nl := "\n"
	if strings.Contains(text, "\r\n") {
		nl = "\r\n"
	}
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += nl
	}
	for _, p := range paths {
		line := "/" + ignorePattern(p)
		if !have[line] {
			text += line + nl
			have[line] = true
		}
	}
	return os.WriteFile(gi, []byte(text), 0o644)
}

// ignorePattern 路径里 .gitignore 的特殊字符转义掉，只匹配这一个路径。
func ignorePattern(p string) string {
	var b strings.Builder
	for _, r := range strings.TrimSuffix(p, "/") {
		if strings.ContainsRune(`\*?[`, r) {
			b.WriteByte('\\')
		}
		b.WriteRune(r)
	}
	s := b.String()
	if strings.HasSuffix(s, " ") {
		s = strings.TrimRight(s, " ") + strings.Repeat(`\ `, len(s)-len(strings.TrimRight(s, " ")))
	}
	return s
}

// —— 提交 ——

// CommitOptions 提交的选项。
type CommitOptions struct {
	Message string
	// Amend 修改上一次提交；说明为空时沿用原来的。
	Amend bool
	// All 先暂存全部改动。
	All      bool
	NoVerify bool
	Signoff  bool
}

// Commit 提交，返回新提交的短哈希。说明从标准输入给 git，不受引号、换行影响。
func Commit(ctx context.Context, dir string, o CommitOptions) (string, error) {
	msg := strings.TrimSpace(o.Message)
	if msg == "" && !o.Amend {
		return "", errors.New("先写一下提交说明")
	}
	if o.All {
		if _, err := runOut(ctx, dir, "add", "-A"); err != nil {
			return "", err
		}
	}
	args := []string{"commit", "-q"}
	stdin := ""
	if o.Amend {
		args = append(args, "--amend")
	}
	if msg == "" {
		args = append(args, "--no-edit")
	} else {
		args = append(args, "-F", "-")
		stdin = msg + "\n"
	}
	if o.NoVerify {
		args = append(args, "--no-verify")
	}
	if o.Signoff {
		args = append(args, "--signoff")
	}
	if _, _, err := runCmd(ctx, dir, stdin, args...); err != nil {
		return "", err
	}
	head, _ := runOut(ctx, dir, "rev-parse", "--short", "HEAD")
	return strings.TrimSpace(head), nil
}

// Reset 把当前分支重置到 target：soft 改动留在暂存区，mixed 留在工作区，hard 全部丢掉。
// 撤销第一个提交（没有父提交）时把分支变回还没有提交，改动留在暂存区。
func Reset(ctx context.Context, dir, target, mode string) error {
	if mode != "soft" && mode != "mixed" && mode != "hard" {
		return errors.New("不认识的重置方式：" + mode)
	}
	if err := checkRev(target); err != nil {
		return err
	}
	if (target == "HEAD~1" || target == "HEAD^") && hasHead(ctx, dir) {
		if _, err := runOut(ctx, dir, "rev-parse", "--verify", "-q", "HEAD~1"); err != nil {
			if mode != "soft" {
				return errors.New("这是第一个提交，只能撤销成改动留在暂存区")
			}
			_, err := runOut(ctx, dir, "update-ref", "-d", "HEAD")
			return err
		}
	}
	_, err := runOut(ctx, dir, "reset", "--"+mode, "-q", target, "--")
	return err
}

// —— 分支、标签、远程 ——

// Branch 一个本地或远程分支。
type Branch struct {
	Name         string
	Commit       string
	Subject      string
	Date         time.Time
	Current      bool
	Upstream     string
	UpstreamGone bool
	Ahead        int
	Behind       int
}

// Tag 一个标签。
type Tag struct {
	Name    string
	Commit  string
	Subject string
	Date    time.Time
}

// Remote 一个远程仓库。
type Remote struct {
	Name     string
	FetchURL string
	PushURL  string
}

// Refs 仓库里的分支、标签、远程。
type Refs struct {
	Branches       []Branch
	RemoteBranches []Branch
	Tags           []Tag
	Remotes        []Remote
}

// maxRefs 每一类最多列多少个。
const maxRefs = 2000

func unixTime(s string) time.Time {
	n, err := strconv.ParseInt(strings.TrimSpace(s), 10, 64)
	if err != nil || n <= 0 {
		return time.Time{}
	}
	return time.Unix(n, 0)
}

// ListRefs 分支（当前分支在前，其余按最近提交排）、远程分支、标签、远程仓库。
func ListRefs(ctx context.Context, dir string) (*Refs, error) {
	format := strings.Join([]string{
		"%(refname)", "%(objectname:short)", "%(upstream:short)", "%(upstream:track,nobracket)", "%(HEAD)",
		"%(committerdate:unix)", "%(contents:subject)", "%(*objectname:short)", "%(creatordate:unix)", "%(symref)",
	}, "%00")
	out, err := runOut(ctx, dir, "for-each-ref", "--sort=-committerdate", "--format="+format, "refs/heads", "refs/remotes", "refs/tags")
	if err != nil {
		return nil, err
	}
	refs := &Refs{}
	for _, line := range strings.Split(out, "\n") {
		f := strings.Split(line, "\x00")
		if len(f) < 10 || f[9] != "" {
			continue
		}
		switch name := f[0]; {
		case strings.HasPrefix(name, "refs/heads/") && len(refs.Branches) < maxRefs:
			b := Branch{Name: strings.TrimPrefix(name, "refs/heads/"), Commit: f[1], Upstream: f[2], Current: f[4] == "*", Date: unixTime(f[5]), Subject: f[6]}
			for _, part := range strings.Split(f[3], ", ") {
				k, v, _ := strings.Cut(part, " ")
				switch k {
				case "ahead":
					b.Ahead, _ = strconv.Atoi(v)
				case "behind":
					b.Behind, _ = strconv.Atoi(v)
				case "gone":
					b.UpstreamGone = true
				}
			}
			refs.Branches = append(refs.Branches, b)
		case strings.HasPrefix(name, "refs/remotes/") && len(refs.RemoteBranches) < maxRefs:
			refs.RemoteBranches = append(refs.RemoteBranches, Branch{Name: strings.TrimPrefix(name, "refs/remotes/"), Commit: f[1], Date: unixTime(f[5]), Subject: f[6]})
		case strings.HasPrefix(name, "refs/tags/") && len(refs.Tags) < maxRefs:
			t := Tag{Name: strings.TrimPrefix(name, "refs/tags/"), Commit: f[1], Subject: f[6], Date: unixTime(f[8])}
			if f[7] != "" {
				t.Commit = f[7]
			}
			refs.Tags = append(refs.Tags, t)
		}
	}
	// 当前分支放最前面，其余按最近提交
	sort.SliceStable(refs.Branches, func(i, j int) bool { return refs.Branches[i].Current && !refs.Branches[j].Current })
	sortTags(refs.Tags)
	refs.Remotes, err = ListRemotes(ctx, dir)
	return refs, err
}

// sortTags 标签按时间排，新的在前；时间一样按名字倒序（版本号大的在前）。
func sortTags(tags []Tag) {
	sort.SliceStable(tags, func(i, j int) bool {
		if !tags[i].Date.Equal(tags[j].Date) {
			return tags[i].Date.After(tags[j].Date)
		}
		return tags[i].Name > tags[j].Name
	})
}

// ListRemotes 远程仓库和它们的拉取、推送地址。
func ListRemotes(ctx context.Context, dir string) ([]Remote, error) {
	out, err := runOut(ctx, dir, "remote", "-v")
	if err != nil {
		return nil, err
	}
	var list []Remote
	idx := map[string]int{}
	for _, line := range lines(out) {
		name, rest, ok := strings.Cut(line, "\t")
		if !ok {
			continue
		}
		url, kind, _ := strings.Cut(rest, " ")
		i, seen := idx[name]
		if !seen {
			i = len(list)
			idx[name] = i
			list = append(list, Remote{Name: name})
		}
		if kind == "(push)" {
			list[i].PushURL = url
		} else {
			list[i].FetchURL = url
		}
	}
	return list, nil
}

// Checkout 切换到 name：本地分支直接切；远程分支（origin/x）切到同名的本地分支，没有就建一个跟踪它的；
// 标签、提交切成游离 HEAD。
func Checkout(ctx context.Context, dir, name string) (string, error) {
	if err := checkRev(name); err != nil {
		return "", err
	}
	if refExists(ctx, dir, "refs/heads/"+name) {
		return runSay(ctx, dir, "", "switch", name)
	}
	if refExists(ctx, dir, "refs/remotes/"+name) {
		if _, local, ok := strings.Cut(name, "/"); ok && local != "" {
			if refExists(ctx, dir, "refs/heads/"+local) {
				return runSay(ctx, dir, "", "switch", local)
			}
			return runSay(ctx, dir, "", "switch", "-c", local, "--track", name)
		}
	}
	return runSay(ctx, dir, "", "switch", "--detach", name)
}

// CreateBranch 从 from（空 = HEAD）建分支，checkout 时切过去。
func CreateBranch(ctx context.Context, dir, name, from string, checkout bool) error {
	if err := checkName("分支", name); err != nil {
		return err
	}
	if _, err := runOut(ctx, dir, "check-ref-format", "--branch", name); err != nil {
		return errors.New("分支名不合规：" + name)
	}
	args := []string{"branch", name}
	if checkout {
		args = []string{"switch", "-c", name}
	}
	if from != "" {
		if err := checkRev(from); err != nil {
			return err
		}
		args = append(args, from)
	}
	_, err := runOut(ctx, dir, args...)
	return err
}

// RenameBranch 分支改名。
func RenameBranch(ctx context.Context, dir, name, newName string) error {
	if err := checkName("分支", name); err != nil {
		return err
	}
	if err := checkName("分支", newName); err != nil {
		return err
	}
	_, err := runOut(ctx, dir, "branch", "-m", name, newName)
	return err
}

// DeleteBranch 删本地分支；没合并过的要 force。
func DeleteBranch(ctx context.Context, dir, name string, force bool) error {
	if err := checkName("分支", name); err != nil {
		return err
	}
	flag := "-d"
	if force {
		flag = "-D"
	}
	_, err := runOut(ctx, dir, "branch", flag, name)
	return err
}

// DeleteRemoteBranch 删远程上的分支（origin/x）。
func DeleteRemoteBranch(ctx context.Context, dir, name string) (string, error) {
	remote, branch, ok := strings.Cut(name, "/")
	if !ok || branch == "" {
		return "", errors.New("远程分支写得不对：" + name)
	}
	if err := checkName("远程", remote); err != nil {
		return "", err
	}
	return runSay(ctx, dir, "", "push", remote, "--delete", branch)
}

// Merge 把 name 合并到当前分支；有冲突时停在合并中，等解决。
func Merge(ctx context.Context, dir, name string) (string, error) {
	if err := checkRev(name); err != nil {
		return "", err
	}
	return runSay(ctx, dir, "", "merge", "--no-edit", name)
}

// Rebase 把当前分支变基到 onto；有冲突时停在变基中。
func Rebase(ctx context.Context, dir, onto string) (string, error) {
	if err := checkRev(onto); err != nil {
		return "", err
	}
	return runSay(ctx, dir, "", "rebase", onto)
}

// CreateTag 在 at（空 = HEAD）打标签；message 不为空时是附注标签。
func CreateTag(ctx context.Context, dir, name, at, message string) error {
	if err := checkName("标签", name); err != nil {
		return err
	}
	args := []string{"tag"}
	stdin := ""
	if strings.TrimSpace(message) != "" {
		args = append(args, "-a", "-F", "-")
		stdin = strings.TrimSpace(message) + "\n"
	}
	args = append(args, name)
	if at != "" {
		if err := checkRev(at); err != nil {
			return err
		}
		args = append(args, at)
	}
	_, _, err := runCmd(ctx, dir, stdin, args...)
	return err
}

// DeleteTag 删本地标签。
func DeleteTag(ctx context.Context, dir, name string) error {
	if err := checkName("标签", name); err != nil {
		return err
	}
	_, err := runOut(ctx, dir, "tag", "-d", name)
	return err
}

// PushTag 推送一个标签；remove 时删掉远程上的这个标签。
func PushTag(ctx context.Context, dir, remote, name string, remove bool) (string, error) {
	if err := checkName("标签", name); err != nil {
		return "", err
	}
	if remote == "" {
		var err error
		if remote, err = defaultRemote(ctx, dir, ""); err != nil {
			return "", err
		}
	}
	if err := checkName("远程", remote); err != nil {
		return "", err
	}
	if remove {
		return runSay(ctx, dir, "", "push", remote, "--delete", "refs/tags/"+name)
	}
	return runSay(ctx, dir, "", "push", remote, "refs/tags/"+name)
}

// defaultRemote 推送到哪个远程：分支设的 pushRemote、仓库设的 pushDefault、origin、唯一的那个。
func defaultRemote(ctx context.Context, dir, branch string) (string, error) {
	if branch != "" {
		if r, _ := runOut(ctx, dir, "config", "--get", "branch."+branch+".pushRemote"); strings.TrimSpace(r) != "" {
			return strings.TrimSpace(r), nil
		}
	}
	if r, _ := runOut(ctx, dir, "config", "--get", "remote.pushDefault"); strings.TrimSpace(r) != "" {
		return strings.TrimSpace(r), nil
	}
	out, _ := runOut(ctx, dir, "remote")
	names := lines(out)
	for _, n := range names {
		if n == "origin" {
			return n, nil
		}
	}
	if len(names) == 1 {
		return names[0], nil
	}
	if len(names) == 0 {
		return "", errors.New("还没有远程仓库，先添加一个")
	}
	return "", errors.New("有好几个远程仓库，选一个再推送")
}

// Fetch 获取；remote 为空时获取全部远程。顺带清掉远程上已经删了的分支。
func Fetch(ctx context.Context, dir, remote string) (string, error) {
	if remote == "" {
		return runSay(ctx, dir, "", "fetch", "--all", "--prune")
	}
	if err := checkName("远程", remote); err != nil {
		return "", err
	}
	return runSay(ctx, dir, "", "fetch", "--prune", remote)
}

// Pull 拉取当前分支的上游。mode：merge | rebase | ff-only，空时按仓库的设置，没设过用 merge。
func Pull(ctx context.Context, dir, mode string) (string, error) {
	args := []string{"pull"}
	switch mode {
	case "merge":
		args = append(args, "--no-rebase")
	case "rebase":
		args = append(args, "--rebase")
	case "ff-only":
		args = append(args, "--ff-only")
	case "":
		rebase, _ := runOut(ctx, dir, "config", "--get", "pull.rebase")
		ff, _ := runOut(ctx, dir, "config", "--get", "pull.ff")
		if strings.TrimSpace(rebase) == "" && strings.TrimSpace(ff) == "" {
			args = append(args, "--no-rebase")
		}
	default:
		return "", errors.New("不认识的拉取方式：" + mode)
	}
	return runSay(ctx, dir, "", args...)
}

// Push 推送当前分支。没有上游（或者指定了别的远程）时推到 remote（空 = 默认远程）上的同名分支并设成上游。
// force 用 --force-with-lease（远程上有别人新推的提交时会拒绝），tags 连同指向推送提交的附注标签。
func Push(ctx context.Context, dir, remote string, force, tags bool) (string, error) {
	branch, err := runOut(ctx, dir, "symbolic-ref", "--short", "-q", "HEAD")
	if err != nil || strings.TrimSpace(branch) == "" {
		return "", errors.New("现在不在任何分支上（游离 HEAD），先建一个分支再推送")
	}
	branch = strings.TrimSpace(branch)
	upstream, _ := runOut(ctx, dir, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{u}")
	upstream = strings.TrimSpace(upstream)
	args := []string{"push"}
	if force {
		args = append(args, "--force-with-lease")
	}
	if tags {
		args = append(args, "--follow-tags")
	}
	if upstream != "" && (remote == "" || strings.HasPrefix(upstream, remote+"/")) {
		return runSay(ctx, dir, "", args...)
	}
	if remote == "" {
		if remote, err = defaultRemote(ctx, dir, branch); err != nil {
			return "", err
		}
	}
	if err := checkName("远程", remote); err != nil {
		return "", err
	}
	return runSay(ctx, dir, "", append(args, "-u", remote, branch)...)
}

// AddRemote 添加远程仓库。
func AddRemote(ctx context.Context, dir, name, url string) error {
	if err := checkName("远程", name); err != nil {
		return err
	}
	if url = strings.TrimSpace(url); url == "" || strings.HasPrefix(url, "-") {
		return errors.New("远程地址不对")
	}
	_, err := runOut(ctx, dir, "remote", "add", name, url)
	return err
}

// SetRemoteURL 改远程仓库的地址。
func SetRemoteURL(ctx context.Context, dir, name, url string) error {
	if err := checkName("远程", name); err != nil {
		return err
	}
	if url = strings.TrimSpace(url); url == "" || strings.HasPrefix(url, "-") {
		return errors.New("远程地址不对")
	}
	_, err := runOut(ctx, dir, "remote", "set-url", name, url)
	return err
}

// RenameRemote 远程仓库改名（远程分支跟着改）。
func RenameRemote(ctx context.Context, dir, name, newName string) error {
	if err := checkName("远程", name); err != nil {
		return err
	}
	if err := checkName("远程", newName); err != nil {
		return err
	}
	_, err := runOut(ctx, dir, "remote", "rename", name, newName)
	return err
}

// RemoveRemote 删除远程仓库（它的远程分支一起删掉）。
func RemoveRemote(ctx context.Context, dir, name string) error {
	if err := checkName("远程", name); err != nil {
		return err
	}
	_, err := runOut(ctx, dir, "remote", "remove", name)
	return err
}

// —— 储藏 ——

// Stash 一个储藏。
type Stash struct {
	Index   int
	Hash    string
	Message string
	// Branch 储藏时所在的分支。
	Branch string
	Date   time.Time
}

// ListStashes 储藏列表，新的在前。
func ListStashes(ctx context.Context, dir string) ([]Stash, error) {
	out, err := runOut(ctx, dir, "stash", "list", "--format=%H%x00%gs%x00%ct")
	if err != nil {
		return nil, err
	}
	var list []Stash
	for _, line := range lines(out) {
		f := strings.Split(line, "\x00")
		if len(f) < 3 {
			continue
		}
		s := Stash{Index: len(list), Hash: f[0], Message: f[1], Date: unixTime(f[2])}
		// 「WIP on main: abc1234 说明」或「On main: 自己写的说明」
		subject := strings.TrimPrefix(strings.TrimPrefix(f[1], "WIP on "), "On ")
		if b, msg, ok := strings.Cut(subject, ": "); ok {
			s.Branch, s.Message = b, msg
		}
		list = append(list, s)
	}
	return list, nil
}

// stashRef 按哈希找到储藏现在的位置 stash@{n}（列表在这期间变了也不会弄错）。
func stashRef(ctx context.Context, dir, hash string) (string, error) {
	list, err := ListStashes(ctx, dir)
	if err != nil {
		return "", err
	}
	for _, s := range list {
		if s.Hash == hash || (len(hash) >= 7 && strings.HasPrefix(s.Hash, hash)) {
			return fmt.Sprintf("stash@{%d}", s.Index), nil
		}
	}
	return "", errors.New("这个储藏已经不在了，刷新一下")
}

// StashPush 储藏改动：untracked 连同没跟踪的文件，staged 只储藏暂存了的，paths 不为空时只储藏这些文件。
func StashPush(ctx context.Context, dir, message string, untracked, staged bool, paths []string) (string, error) {
	args := []string{"stash", "push"}
	if untracked {
		args = append(args, "--include-untracked")
	}
	if staged {
		args = append(args, "--staged")
	}
	if m := strings.TrimSpace(message); m != "" {
		args = append(args, "-m", m)
	}
	if len(paths) > 0 {
		args = append(append(args, "--"), paths...)
	}
	return runSay(ctx, dir, "", args...)
}

// StashApply 应用储藏；pop 时应用后删掉它，index 时连暂存状态一起恢复。
func StashApply(ctx context.Context, dir, hash string, pop, index bool) (string, error) {
	ref, err := stashRef(ctx, dir, hash)
	if err != nil {
		return "", err
	}
	args := []string{"stash", "apply"}
	if pop {
		args[1] = "pop"
	}
	if index {
		args = append(args, "--index")
	}
	return runSay(ctx, dir, "", append(args, ref)...)
}

// StashDrop 删掉一个储藏。
func StashDrop(ctx context.Context, dir, hash string) error {
	ref, err := stashRef(ctx, dir, hash)
	if err != nil {
		return err
	}
	_, err = runOut(ctx, dir, "stash", "drop", ref)
	return err
}

// —— 拣选、还原，进行中的操作 ——

func isMergeCommit(ctx context.Context, dir, hash string) bool {
	out, err := runOut(ctx, dir, "rev-list", "--parents", "-n", "1", hash)
	return err == nil && len(strings.Fields(out)) > 2
}

// CherryPick 把一个提交拣选到当前分支；合并提交按第一个父提交算。
func CherryPick(ctx context.Context, dir, hash string) (string, error) {
	if !IsHash(hash) {
		return "", errors.New("提交哈希不对")
	}
	args := []string{"cherry-pick"}
	if isMergeCommit(ctx, dir, hash) {
		args = append(args, "-m", "1")
	}
	return runSay(ctx, dir, "", append(args, hash)...)
}

// RevertCommit 新建一个提交撤回某个提交的改动；合并提交按第一个父提交算。
func RevertCommit(ctx context.Context, dir, hash string) (string, error) {
	if !IsHash(hash) {
		return "", errors.New("提交哈希不对")
	}
	args := []string{"revert", "--no-edit"}
	if isMergeCommit(ctx, dir, hash) {
		args = append(args, "-m", "1")
	}
	return runSay(ctx, dir, "", append(args, hash)...)
}

// Sequence 进行中的合并、变基、拣选、还原：action 是 continue | skip | abort。
func Sequence(ctx context.Context, dir, action string) (string, error) {
	if action != "continue" && action != "skip" && action != "abort" {
		return "", errors.New("不认识的操作：" + action)
	}
	st := &Status{}
	operationOf(ctx, dir, st)
	cmd := st.Operation
	switch cmd {
	case "":
		return "", errors.New("现在没有进行中的合并、变基、拣选或还原")
	case "merge":
		if action == "skip" {
			return "", errors.New("合并不能跳过")
		}
	case "bisect":
		if action != "abort" {
			return "", errors.New("二分查找只能在命令行里继续")
		}
		return runSay(ctx, dir, "", "bisect", "reset")
	}
	return runSay(ctx, dir, "", cmd, "--"+action)
}

// —— 历史 ——

// LogCommit 历史里的一个提交。
type LogCommit struct {
	Hash    string
	Short   string
	Parents []string
	Author  string
	Email   string
	Date    time.Time
	Subject string
	// Refs 指向它的分支、标签：HEAD -> main、origin/main、tag: v1.0。
	Refs []string
}

// LogOptions 读历史的条件。
type LogOptions struct {
	Skip  int
	Limit int
	// Ref 从哪个分支 / 提交往前看（空 = HEAD）；All 时看所有分支。
	Ref  string
	All  bool
	Path string
	// Keyword 在提交说明里搜（不分大小写）。
	Keyword string
}

const logFormat = "%H%x00%h%x00%P%x00%an%x00%ae%x00%at%x00%s%x00%D"

func parseLogCommit(f []string) LogCommit {
	c := LogCommit{Hash: f[0], Short: f[1], Parents: strings.Fields(f[2]), Author: f[3], Email: f[4], Date: unixTime(f[5]), Subject: f[6]}
	if f[7] != "" {
		c.Refs = strings.Split(f[7], ", ")
	}
	return c
}

// Log 读历史，more 表示后面还有。
func Log(ctx context.Context, dir string, o LogOptions) ([]LogCommit, bool, error) {
	if !hasHead(ctx, dir) && !o.All {
		return nil, false, nil
	}
	if o.Limit <= 0 || o.Limit > 500 {
		o.Limit = 100
	}
	args := []string{"-c", "core.quotepath=off", "log", "--format=" + logFormat + "%x1e", "--skip=" + strconv.Itoa(max(o.Skip, 0)), "--max-count=" + strconv.Itoa(o.Limit+1)}
	if kw := strings.TrimSpace(o.Keyword); kw != "" {
		args = append(args, "-i", "--fixed-strings", "--grep="+kw)
	}
	switch {
	case o.All:
		args = append(args, "--all")
	case o.Ref != "":
		if err := checkRev(o.Ref); err != nil {
			return nil, false, err
		}
		args = append(args, o.Ref)
	}
	args = append(args, "--")
	if o.Path != "" {
		args = append(args, o.Path)
	}
	out, err := runOut(ctx, dir, args...)
	if err != nil {
		return nil, false, err
	}
	var list []LogCommit
	for _, rec := range strings.Split(out, "\x1e") {
		f := strings.Split(strings.TrimLeft(rec, "\r\n"), "\x00")
		if len(f) < 8 {
			continue
		}
		list = append(list, parseLogCommit(f))
	}
	more := len(list) > o.Limit
	if more {
		list = list[:o.Limit]
	}
	return list, more, nil
}

// CommitDetail 一个提交的详情。
type CommitDetail struct {
	LogCommit
	// Body 说明标题后面的部分。
	Body       string
	Committer  string
	CommitDate time.Time
	Files      []File
}

// GetCommit 读一个提交（或储藏）的详情和改了哪些文件；合并提交按第一个父提交算，储藏连同没跟踪的文件。
func GetCommit(ctx context.Context, dir, hash string, stash bool) (*CommitDetail, error) {
	if !IsHash(hash) {
		return nil, errors.New("提交哈希不对")
	}
	out, err := runOut(ctx, dir, "show", "-s", "--format="+logFormat+"%x00%cn%x00%ct%x00%b", hash)
	if err != nil {
		return nil, err
	}
	f := strings.SplitN(out, "\x00", 11)
	if len(f) < 11 {
		return nil, errors.New("读不到这个提交")
	}
	d := &CommitDetail{LogCommit: parseLogCommit(f), Committer: f[8], CommitDate: unixTime(f[9]), Body: strings.TrimSpace(f[10])}
	if stash {
		d.Files, err = changedFiles(ctx, dir, "diff", hash+"^1", hash)
		if err != nil {
			return nil, err
		}
		if _, e := runOut(ctx, dir, "rev-parse", "--verify", "-q", hash+"^3"); e == nil {
			extra, err := changedFiles(ctx, dir, "show", "--format=", hash+"^3")
			if err != nil {
				return nil, err
			}
			d.Files = append(d.Files, extra...)
		}
		return d, nil
	}
	d.Files, err = changedFiles(ctx, dir, "show", "--format=", "--diff-merges=first-parent", hash)
	return d, err
}

// changedFiles 一次 diff / show 改了哪些文件：--name-status 拿种类，--numstat 拿增删行数。
func changedFiles(ctx context.Context, dir string, args ...string) ([]File, error) {
	base := append([]string{"-c", "core.quotepath=off"}, args...)
	out, err := runOut(ctx, dir, append(append([]string{}, base...), "-M", "--name-status", "-z", "--no-ext-diff")...)
	if err != nil {
		return nil, err
	}
	nums, _ := runOut(ctx, dir, append(append([]string{}, base...), "-M", "--numstat", "-z", "--no-ext-diff")...)
	counts := parseNumstat(nums)
	var files []File
	parts := strings.Split(out, "\x00")
	for i := 0; i < len(parts); i++ {
		code := parts[i]
		if code == "" || i+1 >= len(parts) {
			continue
		}
		f := File{Kind: code[:1]}
		if f.Kind == "R" || f.Kind == "C" {
			if i+2 >= len(parts) {
				break
			}
			f.OrigPath, f.Path = parts[i+1], parts[i+2]
			i += 2
		} else {
			f.Path = parts[i+1]
			i++
		}
		if n, ok := counts[f.Path]; ok {
			f.Added, f.Deleted, f.Binary = n.added, n.deleted, n.binary
		}
		files = append(files, f)
		if len(files) >= maxStatusFiles {
			break
		}
	}
	return files, nil
}

// Init 把目录初始化成仓库。
func Init(ctx context.Context, dir string) error {
	if isGit, _ := Info(dir); isGit {
		return errors.New("这里已经是 Git 仓库了")
	}
	_, err := runOut(ctx, dir, "init")
	return err
}

// —— 写提交说明用的材料 ——

// MessageContext 让 Agent 写提交说明时给它看的：要提交的改动（有暂存的就是暂存的，否则是全部）和最近几次提交的说明。
// diff 太长时截断。
func MessageContext(ctx context.Context, dir string, maxDiff int) (diff string, staged bool, recent []string) {
	staged = true
	stat, _ := runOut(ctx, dir, "-c", "core.quotepath=off", "diff", "--cached", "--stat", "--no-color")
	if strings.TrimSpace(stat) != "" {
		diff, _ = runOut(ctx, dir, "-c", "core.quotepath=off", "diff", "--cached", "--no-color", "--no-ext-diff", "-M")
		diff = stat + "\n" + diff
	} else {
		staged = false
		if r, err := fullPatch(ctx, dir); err == nil {
			diff = r.Patch
		}
	}
	if len(diff) > maxDiff {
		diff = diff[:maxDiff] + "\n…（后面的改动太长，略去）\n"
	}
	if out, err := runOut(ctx, dir, "log", "-15", "--format=%s"); err == nil {
		recent = lines(out)
	}
	return diff, staged, recent
}
