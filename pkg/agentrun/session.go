package agentrun

import (
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"

	"momoko/pkg/acp"
	"momoko/pkg/agentdriver"
	"momoko/pkg/agentfs"
	"momoko/pkg/agentproc"
)

// ErrAuthRequired Agent 要求先登录才能开会话。
var ErrAuthRequired = errors.New("Agent 还没有登录")

// ErrBusy 这一轮还没结束。
var ErrBusy = errors.New("这个会话正在运行，先停下来")

// 引擎自己的提示和出错原因，页面按代码显示文案。
const (
	NoticeResumeFailed      = "resume_failed"
	NoticeResumeUnsupported = "resume_unsupported"
	TurnProcessExited       = "process_exited"
	TurnAuthRequired        = "auth_required"
)

// Sink 引擎往外报的事情，由调用方实现（落库、推给页面、发通知）。
// Item / Delta / State / Usage / Title / Config / Asked 在引擎的锁里按事件顺序调用，
// 实现里不能做慢事（读写库、网络），也不能回头调用 Session 的方法；Answered / TurnEnded / Exited 在锁外调用。
type Sink interface {
	// Item 一条事件出现或变化，Seq / Rev 已经分好。push 为 false 时只需要落库（正在写的文字定时落一次）。
	Item(item *Item, push bool)
	// Delta 正在写的那条文字又多了一段。
	Delta(seq int64, text string)
	// State Agent 给的命令或计划变了。
	State(state State)
	// Usage 上下文用量变了。
	Usage(used, size int64)
	// Title Agent 给了会话标题。
	Title(title string)
	// Config Agent 侧的权限模式、模型、推理强度当前值变了；空字符串表示这一项 Agent 没有。
	Config(mode, model, effort string)
	// Asked 出现一张等人回答的批准卡片。
	Asked(p *Permission)
	// Answered 一张批准卡片有了结果（kind 是选中选项的种类，这一轮被停止时为空）；waiting 表示还有别的卡片在等。
	Answered(id, kind string, waiting bool)
	// TurnEnded 一轮结束。
	TurnEnded(end TurnEnd)
	// Exited 进程没了（主动结束或异常退出），这个 Session 不能再用。
	Exited()
}

// TurnEnd 一轮结束时交给调用方的东西。
type TurnEnd struct {
	Turn *Turn
	// LastText Agent 这一轮最后一段回答，通知里当摘要用。
	LastText    string
	State       State
	ContextUsed int64
	ContextSize int64
}

// Config 启动一个会话需要的东西。
type Config struct {
	Launch agentdriver.Launch
	Driver agentdriver.Driver
	// Env 在启动命令自带的环境之上再覆盖（供应商的配置目录、接口地址、密钥等）。
	Env       map[string]string
	Cwd       string
	ExtraDirs []string
	// TempDir 进程的临时目录（TMPDIR / TEMP / TMP 指到这里），不存在就建出来。
	TempDir    string
	McpServers []McpServer
	// ResumeID 不为空时接回 Agent 侧已有的会话，否则新建。
	ResumeID string
	// Mode / Model / Effort 想要的权限模式、模型、推理强度；空表示采用 Agent 当前的。
	Mode, Model, Effort string
	// ModelByEnv 模型和推理强度已经靠 Env 定下来了（第三方接口），不再通过协议设置。
	ModelByEnv bool
	// Seq / Rev 事件序号和修订号从哪接着往下排。
	Seq, Rev    int64
	State       State
	TitleCustom bool
	ContextUsed int64
	ContextSize int64
	Sink        Sink
}

// Started 启动后实际生效的东西。
type Started struct {
	// SessionID Agent 侧的会话 id。
	SessionID           string
	Mode, Model, Effort string
	Caps                Caps
	AgentName           string
	AgentVersion        string
}

type pending struct {
	item  *Item
	reply func(*acp.RequestPermissionResponse)
}

type turnRun struct {
	started   time.Time
	cancelled bool
	lastText  string
}

// Session 一个连着 Agent 进程的会话。
type Session struct {
	sink       Sink
	driver     agentdriver.Driver
	cwd        string
	modelByEnv bool
	proc       *agentproc.Process
	client     *acp.Client
	acpID      string
	caps       Caps

	mu sync.Mutex
	// configs Agent 最近给的配置项（权限模式、模型、推理强度）。
	configs []acp.ConfigOption

	seq, rev int64
	open     map[string]*Item // 每条流（主会话 ""、各子会话）正在写的那条文字
	dirty    map[string]bool
	tools    map[string]*Item
	subs     map[string]*Item
	perms    map[string]*pending
	state    State

	titleCustom bool
	contextUsed int64
	contextSize int64

	turn       *turnRun
	lastActive time.Time
	stopping   bool
	gone       bool
	done       chan struct{}
}

// Start 启动进程、握手、恢复或新建会话，并把想要的权限模式、模型、推理强度应用上去。
func Start(ctx context.Context, cfg Config) (*Session, *Started, error) {
	if err := os.MkdirAll(cfg.Cwd, 0o755); err != nil {
		return nil, nil, errors.New("建不了会话目录：" + err.Error())
	}
	env := map[string]string{}
	for k, v := range cfg.Launch.Env {
		env[k] = v
	}
	for k, v := range cfg.Env {
		env[k] = v
	}
	if cfg.TempDir != "" {
		if err := os.MkdirAll(cfg.TempDir, 0o755); err != nil {
			return nil, nil, errors.New("建不了临时目录：" + err.Error())
		}
		env["TMPDIR"], env["TEMP"], env["TMP"] = cfg.TempDir, cfg.TempDir, cfg.TempDir
	}
	proc, err := agentproc.Start(agentproc.Config{Command: cfg.Launch.Command, Args: cfg.Launch.Args, Dir: cfg.Cwd, Env: env})
	if err != nil {
		return nil, nil, err
	}
	s := &Session{
		sink:        cfg.Sink,
		driver:      cfg.Driver,
		cwd:         cfg.Cwd,
		modelByEnv:  cfg.ModelByEnv,
		proc:        proc,
		seq:         cfg.Seq,
		rev:         cfg.Rev,
		open:        map[string]*Item{},
		dirty:       map[string]bool{},
		tools:       map[string]*Item{},
		subs:        map[string]*Item{},
		perms:       map[string]*pending{},
		state:       cfg.State,
		titleCustom: cfg.TitleCustom,
		contextUsed: cfg.ContextUsed,
		contextSize: cfg.ContextSize,
		lastActive:  time.Now(),
		done:        make(chan struct{}),
	}
	s.client = acp.NewClient(proc.Stdout(), proc.Stdin(), s)
	fail := func(err error) (*Session, *Started, error) {
		reason := exitReason(proc, err)
		if acp.IsAuthRequired(err) {
			reason = ErrAuthRequired
		}
		s.client.Close()
		proc.Kill()
		return nil, nil, reason
	}

	startCtx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	init, err := s.client.Initialize(startCtx, clientInfo(), clientCaps())
	if err != nil {
		return fail(err)
	}
	s.caps = capsOf(init)
	started := &Started{Caps: s.caps}
	if init.AgentInfo != nil {
		started.AgentName, started.AgentVersion = init.AgentInfo.Name, init.AgentInfo.Version
	}

	setup := acp.SessionSetup{Cwd: cfg.Cwd, McpServers: mcpServers(cfg.McpServers, s.caps)}
	if s.caps.AdditionalDirs {
		setup.AdditionalDirectories = cfg.ExtraDirs
	}
	var state *acp.SessionState
	notice, noticeText := "", ""
	if cfg.ResumeID != "" {
		if !s.caps.Resume {
			notice = NoticeResumeUnsupported
		} else if state, err = s.client.ResumeSession(startCtx, cfg.ResumeID, setup); err != nil {
			if proc.Exited() {
				return fail(err)
			}
			notice, noticeText, state = NoticeResumeFailed, exitReason(proc, err).Error(), nil
		} else {
			s.acpID = cfg.ResumeID
		}
	}
	if state == nil {
		if state, err = s.client.NewSession(startCtx, setup); err != nil {
			return fail(err)
		}
		s.acpID = state.SessionID
	}
	s.mu.Lock()
	s.configs = state.ConfigOptions
	s.mu.Unlock()

	started.SessionID = s.acpID
	started.Mode = s.applyWanted(startCtx, acp.CategoryMode, cfg.Mode)
	if !s.modelByEnv {
		started.Model = s.applyWanted(startCtx, acp.CategoryModel, cfg.Model)
		started.Effort = s.applyWanted(startCtx, acp.CategoryThoughtLevel, cfg.Effort)
	}
	if notice != "" {
		// 没能接回之前的上下文，接下来是新的上下文：页面按 Code 显示，Text 是原因。
		s.mu.Lock()
		item := s.newItem(ItemNotice, "")
		item.Level, item.Code, item.Text = "warning", notice, noticeText
		s.emit(item, true)
		s.mu.Unlock()
	}
	go s.watch()
	go s.flushLoop()
	return s, started, nil
}

// applyWanted 让新进程用上某一类配置项想要的值，返回最终生效的值；Agent 没有这一类时为空。
func (s *Session) applyWanted(ctx context.Context, category, want string) string {
	s.mu.Lock()
	opt := configOf(s.configs, category)
	s.mu.Unlock()
	if opt == nil {
		return ""
	}
	if want == "" || want == opt.Current() || !hasValue(opt, want) {
		return opt.Current()
	}
	stepCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()
	options, err := s.client.SetConfigOption(stepCtx, s.acpID, opt.ID, want)
	if err != nil {
		return opt.Current()
	}
	s.mu.Lock()
	s.configs = options
	s.mu.Unlock()
	return want
}

func hasValue(opt *acp.ConfigOption, value string) bool {
	for _, v := range opt.Values() {
		if v.Value == value {
			return true
		}
	}
	return false
}

// —— 对外的状态 ——

// Caps Agent 声明的能力。
func (s *Session) Caps() Caps { return s.caps }

// Busy 是不是有一轮正在进行。
func (s *Session) Busy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.turn != nil
}

// ActiveAt 最近一轮开始或结束的时间。只看这两个时刻：Agent 一轮结束后还会陆续报用量、配置这类更新，
// 拿它们算活动的话进程永远不算空闲，回收不掉。
func (s *Session) ActiveAt() time.Time {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastActive
}

// Alive 进程还在。
func (s *Session) Alive() bool { return !s.proc.Exited() }

// Process 底层的进程（看进程号、内存）。
func (s *Session) Process() *agentproc.Process { return s.proc }

// SetTitleCustom 用户自己起了标题之后，Agent 给的标题不再报上来。
func (s *Session) SetTitleCustom(on bool) { s.mu.Lock(); s.titleCustom = on; s.mu.Unlock() }

// Catalog Agent 现在给的可选项。
func (s *Session) Catalog() Catalog {
	s.mu.Lock()
	defer s.mu.Unlock()
	return catalogOf(s.configs)
}

// Effort Agent 现在的推理强度；没有这一项时为空。
func (s *Session) Effort() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	if opt := configOf(s.configs, acp.CategoryThoughtLevel); opt != nil {
		return opt.Current()
	}
	return ""
}

// Sync 在引擎的锁里执行 fn：正在写的文字先报一次，执行期间不会有新事件。
// 页面打开会话时用它把「补发历史」和「开始订阅」做成一步，中间不漏也不重复。
func (s *Session) Sync(fn func(state State, contextUsed, contextSize int64)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flushOpen()
	fn(s.state, s.contextUsed, s.contextSize)
}

// —— 生命周期 ——

// watch 等进程退出。
func (s *Session) watch() {
	<-s.client.Done()
	s.proc.Kill()
	s.remove()
}

func (s *Session) remove() {
	s.mu.Lock()
	if s.gone {
		s.mu.Unlock()
		return
	}
	s.gone = true
	close(s.done)
	s.flushOpen()
	s.mu.Unlock()
	s.sink.Exited()
}

// Shutdown 主动结束进程（空闲回收、切换供应商、删除会话、停服）。
func (s *Session) Shutdown() {
	s.mu.Lock()
	if s.stopping {
		s.mu.Unlock()
		return
	}
	s.stopping = true
	s.mu.Unlock()
	if s.caps.Close && s.Alive() {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		_ = s.client.CloseSession(ctx, s.acpID)
		cancel()
	}
	s.client.Close()
	s.proc.Kill()
	s.remove()
}

// flushLoop 正在写的文字每秒报一次让调用方落库：页面靠增量实时显示，库里的那份是给重连和重启用的。
func (s *Session) flushLoop() {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-s.done:
			return
		case <-ticker.C:
			s.mu.Lock()
			s.flushOpen()
			s.mu.Unlock()
		}
	}
}

func (s *Session) flushOpen() {
	for sub, dirty := range s.dirty {
		if !dirty {
			continue
		}
		s.dirty[sub] = false
		if item := s.open[sub]; item != nil {
			s.emit(item, false)
		}
	}
}

// —— 事件（都在 s.mu 里调用）——

func (s *Session) newItem(typ, sub string) *Item {
	s.seq++
	return &Item{Seq: s.seq, Type: typ, Sub: sub, At: time.Now().UnixMilli()}
}

func (s *Session) emit(item *Item, push bool) {
	s.rev++
	item.Rev = s.rev
	s.sink.Item(item, push)
}

// closeOpen 结束某条流上正在写的那条文字。
func (s *Session) closeOpen(sub string) {
	item := s.open[sub]
	if item == nil {
		return
	}
	delete(s.open, sub)
	delete(s.dirty, sub)
	item.Open = false
	if item.Type == ItemThought {
		item.Ms = time.Now().UnixMilli() - item.At
	}
	s.emit(item, true)
}

func (s *Session) appendText(sub, typ, text string) {
	if text == "" {
		return
	}
	if cur := s.open[sub]; cur != nil && cur.Type == typ {
		cur.Text += text
		s.dirty[sub] = true
		s.sink.Delta(cur.Seq, text)
	} else {
		s.closeOpen(sub)
		item := s.newItem(typ, sub)
		item.Text, item.Open = text, true
		s.open[sub] = item
		s.emit(item, true)
	}
	if sub == "" && typ == ItemAssistant && s.turn != nil {
		s.turn.lastText = s.open[sub].Text
	}
}

// —— acp.Events ——

func (s *Session) AuthStatus(acp.AuthState) {}

func (s *Session) SessionUpdate(n *acp.SessionNotification) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := &n.Update
	sub := ""
	if n.SessionID != "" && s.acpID != "" && n.SessionID != s.acpID {
		sub = n.SessionID
	}
	switch u.Kind {
	case acp.UpdateAgentMessageChunk, acp.UpdateAgentThoughtChunk:
		block := u.Chunk()
		if block == nil || block.Type != "text" {
			return
		}
		typ := ItemAssistant
		if u.Kind == acp.UpdateAgentThoughtChunk {
			typ = ItemThought
		}
		s.appendText(sub, typ, block.Text)
	case acp.UpdateUserMessageChunk:
		// 用户的话发出去时已经记过了；子 Agent 收到的任务是例外，它只从这里来。
		if sub == "" {
			return
		}
		if block := u.Chunk(); block != nil && block.Type == "text" {
			s.appendText(sub, ItemUser, block.Text)
		}
	case acp.UpdateToolCall, acp.UpdateToolCallUpdate:
		s.onTool(sub, u)
	case acp.UpdatePlan:
		if sub != "" {
			return
		}
		s.state.Plan = make([]PlanEntry, 0, len(u.Entries))
		for _, e := range u.Entries {
			s.state.Plan = append(s.state.Plan, PlanEntry{Content: e.Content, Status: e.Status})
		}
		s.sink.State(s.state)
	case acp.UpdateAvailableCommands:
		if sub != "" {
			return
		}
		s.state.Commands = make([]Command, 0, len(u.AvailableCommands))
		for _, c := range u.AvailableCommands {
			cmd := Command{Name: c.Name, Description: c.Description}
			if c.Input != nil {
				cmd.Hint = c.Input.Hint
			}
			s.state.Commands = append(s.state.Commands, cmd)
		}
		s.sink.State(s.state)
	case acp.UpdateConfigOption:
		if sub != "" || len(u.ConfigOptions) == 0 {
			return
		}
		s.configs = u.ConfigOptions
		s.reportConfig()
	case acp.UpdateSessionInfo:
		if sub != "" || u.Title == nil || *u.Title == "" || s.titleCustom {
			return
		}
		s.sink.Title(*u.Title)
	case acp.UpdateUsage:
		if sub != "" {
			return
		}
		if u.Used != nil {
			s.contextUsed = *u.Used
		}
		if u.Size != nil {
			s.contextSize = *u.Size
		}
		s.sink.Usage(s.contextUsed, s.contextSize)
	case acp.UpdateNotice:
		if u.Title == nil {
			return
		}
		item := s.newItem(ItemNotice, "")
		item.Level, item.Title, item.Text = u.Severity, *u.Title, u.Description
		s.emit(item, true)
	case acp.UpdateSubagent:
		if u.SessionID != "" {
			s.onSubagent(u)
		}
	}
}

// reportConfig 把 Agent 侧当前的权限模式、模型、推理强度报出去。
func (s *Session) reportConfig() {
	current := func(category string) string {
		if opt := configOf(s.configs, category); opt != nil {
			return opt.Current()
		}
		return ""
	}
	mode, model, effort := current(acp.CategoryMode), "", ""
	if !s.modelByEnv {
		model, effort = current(acp.CategoryModel), current(acp.CategoryThoughtLevel)
	}
	s.sink.Config(mode, model, effort)
}

func toolKey(sub, id string) string { return sub + "\x00" + id }

// onTool 一次工具调用的出现或更新。
func (s *Session) onTool(sub string, u *acp.SessionUpdate) {
	if u.ToolCallID == "" {
		return
	}
	key := toolKey(sub, u.ToolCallID)
	item := s.tools[key]
	if item == nil {
		s.closeOpen(sub)
		item = s.newItem(ItemTool, sub)
		item.Tool = &Tool{ID: u.ToolCallID, Status: "pending"}
		s.tools[key] = item
	}
	tool := item.Tool
	if u.Title != nil && *u.Title != "" {
		tool.Title = *u.Title
	}
	if u.ToolKind != "" {
		tool.Kind = u.ToolKind
	}
	if u.Status != "" {
		tool.Status = u.Status
	}
	if len(u.Locations) > 0 {
		tool.Locations = make([]Location, 0, len(u.Locations))
		for _, loc := range u.Locations {
			out := Location{Path: agentfs.Rel(s.cwd, loc.Path)}
			if loc.Line != nil {
				out.Line = *loc.Line
			}
			tool.Locations = append(tool.Locations, out)
		}
	}
	if cmd := s.driver.Command(u.RawInput); cmd != "" {
		tool.Command = cmd
	}
	if content, ok := u.ToolContent(); ok {
		s.applyToolContent(tool, content)
	}
	if (tool.Status == "completed" || tool.Status == "failed") && item.Ms == 0 {
		item.Ms = max(time.Now().UnixMilli()-item.At, 1)
	}
	s.emit(item, true)
}

// applyToolContent 工具调用的内容原样记下：文字输出（只留末尾）、文件改动（ACP 给的新旧文本）、派出的子 Agent。
func (s *Session) applyToolContent(tool *Tool, content []acp.ToolCallContent) {
	var output strings.Builder
	var diffs []Diff
	for _, c := range content {
		switch c.Type {
		case "content":
			if c.Content != nil && c.Content.Type == "text" {
				if output.Len() > 0 {
					output.WriteString("\n")
				}
				output.WriteString(c.Content.Text)
			}
		case "diff":
			diffs = append(diffs, Diff{Path: agentfs.Rel(s.cwd, c.Path), OldText: c.OldText, NewText: c.NewText})
		case "session":
			tool.Child = c.SessionID
		}
	}
	if diffs != nil {
		tool.Diffs = diffs
	}
	if text := output.String(); text != "" {
		if len(text) > outputKeep {
			text = "…\n" + text[len(text)-outputKeep:]
		}
		tool.Output = text
	}
}

// onSubagent 子 Agent 的出现和状态变化。
func (s *Session) onSubagent(u *acp.SessionUpdate) {
	item := s.subs[u.SessionID]
	if item == nil {
		item = s.newItem(ItemSubagent, "")
		item.Subagent = &Subagent{ID: u.SessionID, State: "running"}
		s.subs[u.SessionID] = item
	}
	sa := item.Subagent
	if u.Title != nil && *u.Title != "" {
		sa.Title = *u.Title
	}
	if u.Description != "" {
		sa.Description = u.Description
	}
	if state := u.SubagentState(); state != "" {
		sa.State = state
		if state == "idle" {
			s.closeOpen(u.SessionID)
			if sa.EndAt == 0 {
				sa.EndAt = time.Now().UnixMilli()
			}
		} else {
			sa.EndAt = 0
		}
	}
	s.emit(item, true)
}

func (s *Session) RequestPermission(req *acp.RequestPermissionRequest, reply func(*acp.RequestPermissionResponse)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.turn == nil || s.turn.cancelled {
		reply(acp.PermissionCancelled())
		return
	}
	sub := ""
	if req.SessionID != "" && req.SessionID != s.acpID {
		sub = req.SessionID
	}
	s.closeOpen(sub)
	item := s.newItem(ItemPermission, "")
	perm := &Permission{ID: uuid.NewString(), ToolID: req.ToolCall.ToolCallID, Kind: req.ToolCall.Kind}
	if req.ToolCall.Title != nil {
		perm.Title = *req.ToolCall.Title
	}
	perm.Command = s.driver.Command(req.ToolCall.RawInput)
	if tool := s.tools[toolKey(sub, req.ToolCall.ToolCallID)]; tool != nil {
		if perm.Title == "" {
			perm.Title = tool.Tool.Title
		}
		if perm.Kind == "" {
			perm.Kind = tool.Tool.Kind
		}
		if perm.Command == "" {
			perm.Command = tool.Tool.Command
		}
	}
	for _, o := range req.Options {
		perm.Options = append(perm.Options, PermissionOption{ID: o.OptionID, Name: o.Name, Kind: o.Kind})
	}
	item.Permission = perm
	s.perms[perm.ID] = &pending{item: item, reply: reply}
	s.emit(item, true)
	s.sink.Asked(perm)
}

// Resolve 回答一张批准卡片。
func (s *Session) Resolve(requestID, optionID string) error {
	s.mu.Lock()
	p := s.perms[requestID]
	if p == nil {
		s.mu.Unlock()
		return errors.New("这个批准已经处理过了，或者这一轮已经结束")
	}
	perm := p.item.Permission
	kind := ""
	for _, o := range perm.Options {
		if o.ID == optionID {
			kind = o.Kind
		}
	}
	if kind == "" {
		s.mu.Unlock()
		return errors.New("没有这个选项")
	}
	delete(s.perms, requestID)
	perm.Outcome, perm.OptionID, perm.OptionKind = "selected", optionID, kind
	s.emit(p.item, true)
	waiting := len(s.perms) > 0
	s.mu.Unlock()

	p.reply(acp.PermissionSelected(optionID))
	s.sink.Answered(requestID, kind, waiting)
	return nil
}

// cancelPending 这一轮被停止：还在等的批准都按「取消」回给 Agent。在 s.mu 里调用，返回被取消的卡片 id。
func (s *Session) cancelPending() []string {
	var ids []string
	for id, p := range s.perms {
		delete(s.perms, id)
		p.item.Permission.Outcome = "cancelled"
		s.emit(p.item, true)
		p.reply(acp.PermissionCancelled())
		ids = append(ids, id)
	}
	return ids
}

// —— 一轮 ——

// Prompt 记下用户的话并发给 Agent，立刻返回；这一轮的内容通过 Sink 陆续报出来，结束时报 TurnEnded。
func (s *Session) Prompt(text string, uploads []Upload) error {
	s.mu.Lock()
	if s.gone || s.stopping {
		s.mu.Unlock()
		return errors.New("Agent 进程刚刚结束了，再发一次")
	}
	if s.turn != nil {
		s.mu.Unlock()
		return ErrBusy
	}
	s.turn = &turnRun{started: time.Now()}
	// 工具调用只在这一轮里更新：上一轮的不会再变，留着只会越积越多。
	s.tools = map[string]*Item{}
	s.lastActive = time.Now()
	item := s.newItem(ItemUser, "")
	item.Text = text
	for _, u := range uploads {
		item.Attachments = append(item.Attachments, Attachment{Kind: u.Kind, Name: u.Name, File: u.File})
	}
	s.emit(item, true)
	s.mu.Unlock()
	go s.runTurn(blocksOf(text, uploads))
	return nil
}

func (s *Session) runTurn(blocks []acp.ContentBlock) {
	resp, err := s.client.Prompt(context.Background(), s.acpID, blocks)

	s.mu.Lock()
	turn := s.turn
	for sub := range s.open {
		s.closeOpen(sub)
	}
	cancelled := s.cancelPending()
	item := s.newItem(ItemTurn, "")
	result := &Turn{Ms: time.Since(turn.started).Milliseconds()}
	item.Turn = result
	exited := false
	switch {
	case err == nil:
		result.Stop = resp.StopReason
		if resp.Usage != nil {
			result.Usage = usageOf(resp.Usage)
		}
	case turn.cancelled || s.stopping:
		result.Stop = acp.StopCancelled
	default:
		result.Stop = "error"
		var rpcErr *acp.Error
		switch {
		case errors.Is(err, acp.ErrClosed) || s.proc.Exited():
			exited = true
			result.Code, result.Error = TurnProcessExited, s.proc.LastError()
		case errors.As(err, &rpcErr):
			result.Error = rpcErr.Message
			if rpcErr.Code == acp.CodeAuthRequired {
				result.Code = TurnAuthRequired
			}
		default:
			result.Error = err.Error()
		}
	}
	// 没接完的工具调用不会再有结果了，别让它们一直转圈。
	for _, tool := range s.tools {
		if tool.Tool.Status == "pending" || tool.Tool.Status == "in_progress" {
			tool.Tool.Status = "failed"
			if result.Stop == acp.StopCancelled {
				tool.Tool.Status = "cancelled"
			}
			s.emit(tool, true)
		}
	}
	s.emit(item, true)
	s.turn = nil
	s.lastActive = time.Now()
	end := TurnEnd{Turn: result, LastText: turn.lastText, State: s.state, ContextUsed: s.contextUsed, ContextSize: s.contextSize}
	s.mu.Unlock()

	for _, id := range cancelled {
		s.sink.Answered(id, "", false)
	}
	s.sink.TurnEnded(end)
	if exited {
		s.proc.Kill()
		s.remove()
	}
}

// Cancel 停止当前这一轮。
func (s *Session) Cancel() {
	s.mu.Lock()
	if s.turn == nil {
		s.mu.Unlock()
		return
	}
	s.turn.cancelled = true
	cancelled := s.cancelPending()
	s.mu.Unlock()
	for _, id := range cancelled {
		s.sink.Answered(id, "", false)
	}
	_ = s.client.Cancel(s.acpID)
}

// SetOption 切换某一类配置项（acp.CategoryMode / CategoryModel / CategoryThoughtLevel），只能选 Agent 列出的值。
func (s *Session) SetOption(ctx context.Context, category, value string) error {
	s.mu.Lock()
	opt := configOf(s.configs, category)
	s.mu.Unlock()
	if opt == nil {
		return errors.New("这个 Agent 不能在这里调整这一项")
	}
	if !hasValue(opt, value) {
		return errors.New("Agent 没有提供这个选项")
	}
	stepCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	options, err := s.client.SetConfigOption(stepCtx, s.acpID, opt.ID, value)
	if err != nil {
		return exitReason(s.proc, err)
	}
	s.mu.Lock()
	s.configs = options
	s.mu.Unlock()
	return nil
}

// Fork 复刻这个会话，返回新会话在 Agent 侧的 id；cwd 可以和原会话不同（Agent 可能拒绝）。
// 新会话之后在它自己的进程里跑，所以这里让当前进程立刻放掉它。
func (s *Session) Fork(ctx context.Context, cwd string, extraDirs []string) (string, error) {
	setup := acp.SessionSetup{Cwd: cwd}
	if s.caps.AdditionalDirs {
		setup.AdditionalDirectories = extraDirs
	}
	forkCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	state, err := s.client.ForkSession(forkCtx, s.acpID, setup)
	if err != nil {
		return "", exitReason(s.proc, err)
	}
	if s.caps.Close {
		closeCtx, closeCancel := context.WithTimeout(ctx, 5*time.Second)
		_ = s.client.CloseSession(closeCtx, state.SessionID)
		closeCancel()
	}
	return state.SessionID, nil
}
