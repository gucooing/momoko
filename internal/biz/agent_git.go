package biz

// 右侧「Git」标签：会话目录本身是仓库，或者它下面几层里的几个仓库。repo 是仓库相对会话目录的路径（会话目录本身为空），
// 文件路径都相对仓库。具体的 git 操作在 pkg/agentgit 里。

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"google.golang.org/protobuf/types/known/timestamppb"

	v1 "momoko/api/gen/v1"
	"momoko/internal/data/ent/gen"
	"momoko/pkg/agentfs"
	"momoko/pkg/agentgit"
	"momoko/pkg/response"
)

// gitCtx git 操作用的超时：不跟着页面请求的超时走（拉取、推送、跑提交钩子、让 Agent 写说明都可能比它久），
// 最多等 110 秒（页面那边等两分钟）。
func gitCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.WithoutCancel(ctx), 110*time.Second)
}

// gitRepo 会话和它里面一个仓库的绝对路径；不是仓库时报错。
func (uc *AgentUsecase) gitRepo(ctx context.Context, id, repo string) (*gen.AgentSession, string, error) {
	s, err := uc.getSession(ctx, id)
	if err != nil {
		return nil, "", err
	}
	dir, err := agentfs.SafeJoin(s.Cwd, repo)
	if err != nil {
		return nil, "", agentErr(err)
	}
	if isGit, _ := agentgit.Info(dir); !isGit {
		name := repo
		if name == "" {
			name = "会话目录"
		}
		return nil, "", response.BadRequest(name + " 不是 Git 仓库")
	}
	return s, dir, nil
}

// checkGitPaths 文件路径都得在仓库里。
func checkGitPaths(dir string, paths []string) error {
	for _, p := range paths {
		if strings.TrimSpace(p) == "" {
			return response.BadRequest("文件路径不能为空")
		}
		if _, err := agentfs.SafeJoin(dir, p); err != nil {
			return agentErr(err)
		}
	}
	return nil
}

func gitTime(t time.Time) *timestamppb.Timestamp {
	if t.IsZero() {
		return nil
	}
	return timestamppb.New(t)
}

func gitFilesPB(files []agentgit.File) []*v1.AgentGitFile {
	out := make([]*v1.AgentGitFile, 0, len(files))
	for _, f := range files {
		out = append(out, &v1.AgentGitFile{Path: f.Path, OrigPath: f.OrigPath, Kind: f.Kind, Added: int32(f.Added), Deleted: int32(f.Deleted), Binary: f.Binary})
	}
	return out
}

func gitCommitPB(c agentgit.LogCommit) *v1.AgentGitCommit {
	return &v1.AgentGitCommit{Hash: c.Hash, ShortHash: c.Short, Parents: c.Parents, Author: c.Author, Email: c.Email, Date: gitTime(c.Date), Subject: c.Subject, Refs: c.Refs}
}

// GitStatus 每个仓库的状态。会话目录本身是仓库时只有它；否则是它下面几层里的仓库（同时读几个）。
func (uc *AgentUsecase) GitStatus(ctx context.Context, id string) (*v1.GetAgentGitStatusResponse, error) {
	s, err := uc.getSession(ctx, id)
	if err != nil {
		return nil, err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	repos := []string{""}
	if isGit, _ := agentgit.Info(s.Cwd); !isGit {
		repos = agentgit.Repos(s.Cwd, nestedRepoDepth, nestedRepoMax)
	}
	list := make([]*v1.AgentGitStatus, len(repos))
	errs := make([]error, len(repos))
	eachParallel(len(repos), func(i int) {
		st, err := agentgit.GetStatus(ctx, filepath.Join(s.Cwd, filepath.FromSlash(repos[i])))
		if err != nil {
			errs[i] = err
			return
		}
		list[i] = &v1.AgentGitStatus{
			Repo: repos[i], Branch: st.Branch, Head: st.Head, Upstream: st.Upstream, UpstreamGone: st.UpstreamGone,
			Ahead: int32(st.Ahead), Behind: int32(st.Behind),
			Operation: st.Operation, OperationRef: st.OperationRef, OperationOnto: st.OperationOnto,
			OperationStep: int32(st.OperationStep), OperationTotal: int32(st.OperationTotal),
			Staged: gitFilesPB(st.Staged), Unstaged: gitFilesPB(st.Unstaged), Conflicted: gitFilesPB(st.Conflicted),
			Stashes: int32(st.Stashes), Remotes: st.Remotes, Truncated: st.Truncated, LastMessage: st.LastMessage,
		}
	})
	resp := &v1.GetAgentGitStatusResponse{}
	for _, st := range list {
		if st != nil {
			resp.Repos = append(resp.Repos, st)
		}
	}
	// 一个都读不出来（比如服务器上没装 git、仓库不归运行 Momoko 的用户所有）：把原因告诉页面
	if len(repos) > 0 && len(resp.Repos) == 0 {
		return nil, agentErr(errs[0])
	}
	return resp, nil
}

// GitDiff 一个文件的 diff。
func (uc *AgentUsecase) GitDiff(ctx context.Context, req *v1.GetAgentGitDiffRequest) (*v1.GetAgentGitDiffResponse, error) {
	_, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return nil, err
	}
	if err := checkGitPaths(dir, nonEmpty(req.GetPath(), req.GetOrigPath())); err != nil {
		return nil, err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	d, err := agentgit.GetDiff(ctx, dir, req.GetSource(), req.GetPath(), req.GetOrigPath(), req.GetRef())
	if err != nil {
		return nil, agentErr(err)
	}
	return &v1.GetAgentGitDiffResponse{Patch: d.Patch, Binary: d.Binary, TooLarge: d.TooLarge}, nil
}

func nonEmpty(items ...string) []string {
	var out []string
	for _, s := range items {
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

// GitFiles 暂存、取消暂存、撤销、解决冲突、忽略。
func (uc *AgentUsecase) GitFiles(ctx context.Context, req *v1.UpdateAgentGitFilesRequest) error {
	_, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return err
	}
	paths, all := req.GetPaths(), req.GetAll()
	if err := checkGitPaths(dir, paths); err != nil {
		return err
	}
	if !all && len(paths) == 0 {
		return response.BadRequest("没有选文件")
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	switch req.GetAction() {
	case "stage":
		err = agentgit.StageFiles(ctx, dir, paths, all)
	case "unstage":
		err = agentgit.UnstageFiles(ctx, dir, paths, all)
	case "discard":
		err = agentgit.DiscardFiles(ctx, dir, paths, all)
	case "ours", "theirs":
		err = agentgit.ResolveFiles(ctx, dir, paths, req.GetAction())
	case "resolve":
		err = agentgit.ResolveFiles(ctx, dir, paths, "")
	case "ignore":
		err = agentgit.IgnoreFiles(dir, paths)
	default:
		return response.BadRequest("不认识的操作：" + req.GetAction())
	}
	return agentErr(err)
}

// GitLines 按行暂存、取消暂存、撤销。
func (uc *AgentUsecase) GitLines(ctx context.Context, req *v1.ApplyAgentGitLinesRequest) error {
	_, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return err
	}
	if err := checkGitPaths(dir, []string{req.GetPath()}); err != nil {
		return err
	}
	lines := make([]int, 0, len(req.GetLines()))
	for _, l := range req.GetLines() {
		lines = append(lines, int(l))
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	return agentErr(agentgit.ApplyLines(ctx, dir, req.GetPath(), req.GetPatch(), lines, req.GetAction()))
}

// GitCommit 提交。
func (uc *AgentUsecase) GitCommit(ctx context.Context, req *v1.CommitAgentGitRequest) (string, error) {
	_, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return "", err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	head, err := agentgit.Commit(ctx, dir, agentgit.CommitOptions{
		Message: req.GetMessage(), Amend: req.GetAmend(), All: req.GetAll(), NoVerify: req.GetNoVerify(), Signoff: req.GetSignoff(),
	})
	return head, agentErr(err)
}

// gitMessageDiffMax 给 Agent 看的改动最多多少字节。
const gitMessageDiffMax = 60 << 10

// GitMessage 让会话的 Agent 写提交说明：把要提交的改动和最近几次提交的说明给它，在同一个 Agent 进程里另开一个
// 一次性会话问（走 ACP，不进对话，要批准的事一律拒绝）。会话的进程没在跑时先起来（和发消息一样）。
func (uc *AgentUsecase) GitMessage(ctx context.Context, id, repo string) (string, error) {
	s, dir, err := uc.gitRepo(ctx, id, repo)
	if err != nil {
		return "", err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	diff, staged, recent := agentgit.MessageContext(ctx, dir, gitMessageDiffMax)
	if strings.TrimSpace(diff) == "" {
		return "", response.BadRequest("没有要提交的改动")
	}
	l, err := uc.ensureLive(ctx, s)
	if err != nil {
		return "", err
	}
	msg, err := l.session.Ask(ctx, dir, gitMessagePrompt(diff, staged, recent))
	if err != nil {
		return "", agentErr(err)
	}
	return cleanCommitMessage(msg), nil
}

func gitMessagePrompt(diff string, staged bool, recent []string) string {
	var b strings.Builder
	b.WriteString("Write a git commit message for the changes below.\n\n")
	b.WriteString("Rules:\n")
	b.WriteString("- Reply with the commit message only: no preamble, no explanation, no code fences.\n")
	b.WriteString("- Match the language, tone and format of the recent commit messages listed below. If there are none, write a short subject line, then a blank line and a brief body only when it helps.\n")
	b.WriteString("- Describe what changed and why, based only on the diff. Do not run tools or read files: everything you need is here.\n\n")
	if len(recent) > 0 {
		b.WriteString("Recent commit messages (newest first):\n")
		for _, r := range recent {
			b.WriteString("- " + r + "\n")
		}
		b.WriteString("\n")
	}
	if staged {
		b.WriteString("Staged changes to be committed:\n")
	} else {
		b.WriteString("Nothing is staged yet, so all of these uncommitted changes will be committed:\n")
	}
	fence := "```"
	for strings.Contains(diff, fence) {
		fence += "`"
	}
	fmt.Fprintf(&b, "%sdiff\n%s\n%s\n", fence, strings.TrimRight(diff, "\n"), fence)
	return b.String()
}

// cleanCommitMessage 去掉 Agent 可能套上的代码块。
func cleanCommitMessage(msg string) string {
	msg = strings.TrimSpace(msg)
	if strings.HasPrefix(msg, "```") {
		if i := strings.Index(msg, "\n"); i >= 0 {
			msg = msg[i+1:]
		}
		msg = strings.TrimSuffix(strings.TrimRight(msg, "` \n"), "\n")
	}
	return strings.TrimSpace(msg)
}

// GitReset 重置当前分支。
func (uc *AgentUsecase) GitReset(ctx context.Context, req *v1.ResetAgentGitRequest) error {
	_, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	return agentErr(agentgit.Reset(ctx, dir, req.GetTarget(), req.GetMode()))
}

// GitRefs 分支、远程分支、标签、远程仓库。
func (uc *AgentUsecase) GitRefs(ctx context.Context, id, repo string) (*v1.ListAgentGitRefsResponse, error) {
	_, dir, err := uc.gitRepo(ctx, id, repo)
	if err != nil {
		return nil, err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	refs, err := agentgit.ListRefs(ctx, dir)
	if err != nil {
		return nil, agentErr(err)
	}
	branch := func(b agentgit.Branch) *v1.AgentGitBranch {
		return &v1.AgentGitBranch{
			Name: b.Name, Commit: b.Commit, Subject: b.Subject, Date: gitTime(b.Date), Current: b.Current,
			Upstream: b.Upstream, UpstreamGone: b.UpstreamGone, Ahead: int32(b.Ahead), Behind: int32(b.Behind),
		}
	}
	resp := &v1.ListAgentGitRefsResponse{}
	for _, b := range refs.Branches {
		resp.Branches = append(resp.Branches, branch(b))
	}
	for _, b := range refs.RemoteBranches {
		resp.RemoteBranches = append(resp.RemoteBranches, branch(b))
	}
	for _, t := range refs.Tags {
		resp.Tags = append(resp.Tags, &v1.AgentGitTag{Name: t.Name, Commit: t.Commit, Subject: t.Subject, Date: gitTime(t.Date)})
	}
	for _, r := range refs.Remotes {
		resp.Remotes = append(resp.Remotes, &v1.AgentGitRemote{Name: r.Name, FetchUrl: r.FetchURL, PushUrl: r.PushURL})
	}
	return resp, nil
}

// GitRef 分支和标签的操作。要连远程仓库的（删远程分支、推送和删除远程标签）和 GitSync 一样走会话的代理。
func (uc *AgentUsecase) GitRef(ctx context.Context, req *v1.UpdateAgentGitRefRequest) (string, error) {
	s, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return "", err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	switch req.GetAction() {
	case "delete-remote-branch", "push-tag", "delete-remote-tag":
		proxy, err := uc.sessionProxy(ctx, s)
		if err != nil {
			return "", err
		}
		ctx = agentgit.WithProxy(ctx, proxy)
	}
	name := strings.TrimSpace(req.GetName())
	var out string
	switch req.GetAction() {
	case "checkout":
		out, err = agentgit.Checkout(ctx, dir, name)
	case "create-branch":
		err = agentgit.CreateBranch(ctx, dir, name, req.GetFrom(), req.GetCheckout())
	case "rename-branch":
		err = agentgit.RenameBranch(ctx, dir, name, strings.TrimSpace(req.GetNewName()))
	case "delete-branch":
		err = agentgit.DeleteBranch(ctx, dir, name, req.GetForce())
	case "delete-remote-branch":
		out, err = agentgit.DeleteRemoteBranch(ctx, dir, name)
	case "merge":
		out, err = agentgit.Merge(ctx, dir, name)
	case "rebase":
		out, err = agentgit.Rebase(ctx, dir, name)
	case "create-tag":
		err = agentgit.CreateTag(ctx, dir, name, req.GetFrom(), req.GetMessage())
	case "delete-tag":
		err = agentgit.DeleteTag(ctx, dir, name)
	case "push-tag":
		out, err = agentgit.PushTag(ctx, dir, req.GetRemote(), name, false)
	case "delete-remote-tag":
		out, err = agentgit.PushTag(ctx, dir, req.GetRemote(), name, true)
	default:
		return "", response.BadRequest("不认识的操作：" + req.GetAction())
	}
	return out, agentErr(err)
}

// GitSync 获取、拉取、推送。会话配了代理（它用的官方账户或第三方接口配的）就只经代理连远程仓库。
func (uc *AgentUsecase) GitSync(ctx context.Context, req *v1.SyncAgentGitRequest) (string, error) {
	s, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return "", err
	}
	proxy, err := uc.sessionProxy(ctx, s)
	if err != nil {
		return "", err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	ctx = agentgit.WithProxy(ctx, proxy)
	var out string
	switch req.GetAction() {
	case "fetch":
		out, err = agentgit.Fetch(ctx, dir, req.GetRemote())
	case "pull":
		out, err = agentgit.Pull(ctx, dir, req.GetMode())
	case "push":
		out, err = agentgit.Push(ctx, dir, req.GetRemote(), req.GetForce(), req.GetTags())
	default:
		return "", response.BadRequest("不认识的操作：" + req.GetAction())
	}
	return out, agentErr(err)
}

// GitRemote 远程仓库的增删改。
func (uc *AgentUsecase) GitRemote(ctx context.Context, req *v1.UpdateAgentGitRemoteRequest) error {
	_, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	name := strings.TrimSpace(req.GetName())
	switch req.GetAction() {
	case "add":
		err = agentgit.AddRemote(ctx, dir, name, req.GetUrl())
	case "set-url":
		err = agentgit.SetRemoteURL(ctx, dir, name, req.GetUrl())
	case "rename":
		err = agentgit.RenameRemote(ctx, dir, name, strings.TrimSpace(req.GetNewName()))
	case "remove":
		err = agentgit.RemoveRemote(ctx, dir, name)
	default:
		return response.BadRequest("不认识的操作：" + req.GetAction())
	}
	return agentErr(err)
}

// GitStashes 储藏列表。
func (uc *AgentUsecase) GitStashes(ctx context.Context, id, repo string) ([]*v1.AgentGitStash, error) {
	_, dir, err := uc.gitRepo(ctx, id, repo)
	if err != nil {
		return nil, err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	list, err := agentgit.ListStashes(ctx, dir)
	if err != nil {
		return nil, agentErr(err)
	}
	out := make([]*v1.AgentGitStash, 0, len(list))
	for _, s := range list {
		out = append(out, &v1.AgentGitStash{Index: int32(s.Index), Hash: s.Hash, Message: s.Message, Branch: s.Branch, Date: gitTime(s.Date)})
	}
	return out, nil
}

// GitStash 储藏：存、应用、弹出、删除。
func (uc *AgentUsecase) GitStash(ctx context.Context, req *v1.UpdateAgentGitStashRequest) (string, error) {
	_, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return "", err
	}
	if err := checkGitPaths(dir, req.GetPaths()); err != nil {
		return "", err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	var out string
	switch req.GetAction() {
	case "push":
		out, err = agentgit.StashPush(ctx, dir, req.GetMessage(), req.GetUntracked(), req.GetStaged(), req.GetPaths())
	case "apply", "pop":
		out, err = agentgit.StashApply(ctx, dir, req.GetHash(), req.GetAction() == "pop", req.GetIndex())
	case "drop":
		err = agentgit.StashDrop(ctx, dir, req.GetHash())
	default:
		return "", response.BadRequest("不认识的操作：" + req.GetAction())
	}
	return out, agentErr(err)
}

// GitOperation 拣选、还原一个提交，或者继续 / 跳过 / 中止进行中的操作。
func (uc *AgentUsecase) GitOperation(ctx context.Context, req *v1.RunAgentGitOperationRequest) (string, error) {
	_, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return "", err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	var out string
	switch req.GetAction() {
	case "cherry-pick":
		out, err = agentgit.CherryPick(ctx, dir, req.GetCommit())
	case "revert":
		out, err = agentgit.RevertCommit(ctx, dir, req.GetCommit())
	case "continue", "skip", "abort":
		out, err = agentgit.Sequence(ctx, dir, req.GetAction())
	default:
		return "", response.BadRequest("不认识的操作：" + req.GetAction())
	}
	return out, agentErr(err)
}

// GitLog 提交历史。
func (uc *AgentUsecase) GitLog(ctx context.Context, req *v1.ListAgentGitLogRequest) (*v1.ListAgentGitLogResponse, error) {
	_, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return nil, err
	}
	if req.GetPath() != "" {
		if err := checkGitPaths(dir, []string{req.GetPath()}); err != nil {
			return nil, err
		}
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	list, more, err := agentgit.Log(ctx, dir, agentgit.LogOptions{
		Skip: int(req.GetSkip()), Limit: int(req.GetLimit()), Ref: req.GetRef(), All: req.GetAll(), Path: req.GetPath(), Keyword: req.GetKeyword(),
	})
	if err != nil {
		return nil, agentErr(err)
	}
	resp := &v1.ListAgentGitLogResponse{More: more}
	for _, c := range list {
		resp.Commits = append(resp.Commits, gitCommitPB(c))
	}
	return resp, nil
}

// GitCommitDetail 一个提交（或储藏）的详情。
func (uc *AgentUsecase) GitCommitDetail(ctx context.Context, req *v1.GetAgentGitCommitRequest) (*v1.GetAgentGitCommitResponse, error) {
	_, dir, err := uc.gitRepo(ctx, req.GetId(), req.GetRepo())
	if err != nil {
		return nil, err
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	d, err := agentgit.GetCommit(ctx, dir, req.GetHash(), req.GetStash())
	if err != nil {
		return nil, agentErr(err)
	}
	return &v1.GetAgentGitCommitResponse{
		Commit: gitCommitPB(d.LogCommit), Body: d.Body, Committer: d.Committer, CommitDate: gitTime(d.CommitDate), Files: gitFilesPB(d.Files),
	}, nil
}

// GitInit 在会话目录（或它下面的某个目录）初始化仓库。
func (uc *AgentUsecase) GitInit(ctx context.Context, id, path string) error {
	s, err := uc.getSession(ctx, id)
	if err != nil {
		return err
	}
	dir, err := agentfs.SafeJoin(s.Cwd, path)
	if err != nil {
		return agentErr(err)
	}
	ctx, cancel := gitCtx(ctx)
	defer cancel()
	return agentErr(agentgit.Init(ctx, dir))
}
