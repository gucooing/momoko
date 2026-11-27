package agentrun

import (
	"cmp"
	"context"
	"encoding/base64"
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

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
	// NoticePromptFailed v2：运行中插进去的消息 Agent 没收下，Text 是原因。
	NoticePromptFailed = "prompt_failed"
)

// Sink 引擎往外报的事情，由调用方实现（落库、推给页面、发通知）。
// Item / Delta / State / Usage / Title / Config / Asked / Questioned 在引擎的锁里按事件顺序调用，
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
	// Questioned 出现一张等人回答的提问卡片（Agent 的表单询问）。
	Questioned(q *Question)
	// TurnStarted 一轮开始了：调用方发的消息、排着队的消息接着发出、Agent 自己开始的（v2，后台任务做完了接着处理之类）。
	// 在引擎的锁里调用，和 SubagentsWorking 按发生的先后排。
	TurnStarted()
	// SubagentsWorking 主会话没有一轮在跑时，有子 Agent 开始干活（on），或者干活的都停下了：会话跟着算运行中或空闲。
	// 子 Agent 干不干活只看它们自己经 ACP 报的状态。在引擎的锁里调用。
	SubagentsWorking(on bool)
	// Answered 一张批准或提问卡片有了结果（批准是选中选项的种类，提问是 accept / decline，这一轮被停止时为空）；
	// waiting 表示还有别的卡片在等。
	Answered(id, kind string, waiting bool)
	// TurnEnded 一轮结束。
	TurnEnded(end TurnEnd)
	// Exited 进程没了（主动结束或异常退出），这个 Session 不能再用。
	Exited()
}

// TurnEnd 一轮结束时交给调用方的东西。
type TurnEnd struct {
	Turn *Turn
	// Working 这一轮结束时还有子 Agent 在干活：会话接着算运行中。
	Working bool
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
	TempDir string
	// MediaDir Agent 工具调用产出的图片存到这里，事件里只记相对它的文件名；空则不留图片。
	MediaDir   string
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

// pending 一张等人回答的卡片：批准（reply）或提问（answer），只有其中一个。
type pending struct {
	item   *Item
	reply  func(*acp.RequestPermissionResponse)
	answer func(*acp.CreateElicitationResponse)
	// req Agent 那边这个请求的 id（撤回时凭它找）。
	req string
}

// queued 一条排着队的消息。
type queued struct {
	id      string
	text    string
	uploads []Upload
}

type turnRun struct {
	started   time.Time
	cancelled bool
	lastText  string
	// active v2：Agent 已经报了运行中（或者收下了消息）；在这之前发消息失败算这一轮出错。
	active bool
}

// Session 一个连着 Agent 进程的会话。
type Session struct {
	sink       Sink
	driver     agentdriver.Driver
	cwd        string
	mediaDir   string
	modelByEnv bool
	proc       *agentproc.Process
	client     *acp.Client
	acpID      string
	caps       Caps
	// v2 协商好的是 ACP v2：一轮的结束看 state_update，运行中也能接着发消息。
	v2 bool

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

	permReqs   map[string]string    // Agent 的请求 id → 批准卡片 id
	messages   map[string]*Item     // 流 + 类型 + 消息 id → 那条文字
	terminals  map[string]*terminal // v2：流 + 终端 id → Agent 的显示终端
	toolDirty  map[string]bool      // 终端输出攒着，每秒报一次
	plans      map[string]*planList // 流 → 计划
	background map[string]bool      // v2：后台还在跑的工具调用（toolKey）和子 Agent（"sub:" + 会话 id）
	subsOn     bool                 // 有子 Agent 在干活（它们自己报的状态是 running / requires_action）
	after      []func()             // 处理一条会话更新时攒下、要在锁外做的事
	queue      []*queued            // 排着队的消息
	sides      map[string]*sideRun  // 同一个进程里另开的一次性会话（Ask）

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
	proc, err := agentproc.Start(agentproc.Config{Command: cfg.Launch.Command, Args: cfg.Launch.Args, Dir: cfg.Cwd, Env: env, Proxy: cfg.Launch.Proxy})
	if err != nil {
		return nil, nil, err
	}
	s := &Session{
		sink:        cfg.Sink,
		driver:      cfg.Driver,
		cwd:         cfg.Cwd,
		mediaDir:    cfg.MediaDir,
		modelByEnv:  cfg.ModelByEnv,
		proc:        proc,
		seq:         cfg.Seq,
		rev:         cfg.Rev,
		open:        map[string]*Item{},
		dirty:       map[string]bool{},
		tools:       map[string]*Item{},
		subs:        map[string]*Item{},
		perms:       map[string]*pending{},
		permReqs:    map[string]string{},
		messages:    map[string]*Item{},
		terminals:   map[string]*terminal{},
		toolDirty:   map[string]bool{},
		plans:       map[string]*planList{},
		background:  map[string]bool{},
		state:       cfg.State,
		titleCustom: cfg.TitleCustom,
		contextUsed: cfg.ContextUsed,
		contextSize: cfg.ContextSize,
		lastActive:  time.Now(),
		done:        make(chan struct{}),
	}
	// 排队的消息、后台的工作、一轮、子 Agent 都属于上一个进程，不接着用（新进程的由它自己报）
	s.state.Queue, s.state.Background, s.state.Turn, s.state.Subagents = nil, nil, false, nil
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
	s.v2 = s.client.V2()
	started := &Started{Caps: s.caps}
	if agent := init.Agent(); agent != nil {
		started.AgentName, started.AgentVersion = agent.Name, agent.Version
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
	// v2 建立会话的回复里就带着斜杠命令，整份替换手上的
	if s.v2 {
		s.state.Commands = commandsOf(state.AvailableCommands)
		s.sink.State(s.state)
	}
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

// Busy 有一轮正在进行，或者（v2）后台还有工作在跑：这时进程不能当空闲回收，也不能换。
func (s *Session) Busy() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.turn != nil || len(s.background) > 0 || s.subsOn
}

// Working 主会话有一轮在进行，或者有子 Agent 在干活：会话算运行中。
func (s *Session) Working() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.turn != nil || s.subsOn
}

// Running 有一轮正在进行（不算后台的工作）。
func (s *Session) Running() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.turn != nil
}

// Protocol 协商好的 ACP 版本。
func (s *Session) Protocol() int { return s.client.Version() }

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

// Subagents 这个进程里派出过的子 Agent，按派出的先后：和推给页面的 State.Subagents 是同一份。
func (s *Session) Subagents() []Subagent {
	s.mu.Lock()
	defer s.mu.Unlock()
	return slices.Clone(s.state.Subagents)
}

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
	if len(s.queue) > 0 {
		s.queue = nil
		s.syncQueueLocked()
	}
	s.clearBackground()
	s.stopSubagents()
	// v1 的一轮由 Prompt 的回复收尾；v2 的没有回复可等，进程没了就在这里收尾
	report := func() {}
	if s.v2 && s.turn != nil {
		report = s.endTurnLocked(&Turn{Stop: "error", Code: TurnProcessExited, Error: s.proc.LastError()})
	}
	s.mu.Unlock()
	report()
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
	for key := range s.toolDirty {
		delete(s.toolDirty, key)
		if item := s.tools[key]; item != nil {
			s.emit(item, true)
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

// appendText 一块文字接到这条流上：和正在写的那条是同一条消息（同类型、消息 id 对得上或没有）就接在后面，
// 否则另起一条。v2 整条更新过的消息（不在写）也按消息 id 接上。
func (s *Session) appendText(sub, typ, text, msgID string) {
	if text == "" {
		return
	}
	cur := s.open[sub]
	switch {
	case cur != nil && cur.Type == typ && (msgID == "" || cur.msgID == "" || cur.msgID == msgID):
		cur.Text += text
		s.dirty[sub] = true
		s.sink.Delta(cur.Seq, text)
	case msgID != "" && s.messages[messageKey(sub, typ, msgID)] != nil:
		item := s.messages[messageKey(sub, typ, msgID)]
		item.Text += text
		s.emit(item, true)
		cur = item
	default:
		s.closeOpen(sub)
		item := s.newItem(typ, sub)
		item.Text, item.Open, item.msgID = text, true, msgID
		s.open[sub] = item
		if msgID != "" {
			s.messages[messageKey(sub, typ, msgID)] = item
		}
		s.emit(item, true)
		cur = item
	}
	if sub == "" && typ == ItemAssistant && s.turn != nil {
		s.turn.lastText = cur.Text
	}
}

// —— acp.Events ——

func (s *Session) AuthStatus(acp.AuthStatus) {}

func (s *Session) SessionUpdate(n *acp.SessionNotification) {
	s.mu.Lock()
	defer func() {
		after := s.after
		s.after = nil
		s.mu.Unlock()
		for _, fn := range after {
			fn()
		}
	}()
	if side := s.sideOf(n.SessionID); side != nil {
		side.update(&n.Update)
		return
	}
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
		s.appendText(sub, typ, block.Text, u.MessageID)
	case acp.UpdateAgentMessage, acp.UpdateAgentThought:
		blocks, ok := u.Blocks()
		if !ok {
			return
		}
		typ := ItemAssistant
		if u.Kind == acp.UpdateAgentThought {
			typ = ItemThought
		}
		item := s.upsertMessage(sub, typ, u.MessageID, textOf(blocks))
		if sub == "" && typ == ItemAssistant && s.turn != nil {
			s.turn.lastText = item.Text
		}
	case acp.UpdateUserMessageChunk, acp.UpdateUserMessage:
		// 用户的话发出去时已经记过了；子 Agent 收到的任务是例外，它只从这里来。
		if sub == "" {
			return
		}
		if u.Kind == acp.UpdateUserMessage {
			if blocks, ok := u.Blocks(); ok {
				s.upsertMessage(sub, ItemUser, u.MessageID, textOf(blocks))
			}
		} else if block := u.Chunk(); block != nil && block.Type == "text" {
			s.appendText(sub, ItemUser, block.Text, u.MessageID)
		}
	case acp.UpdateSessionMessage, acp.UpdateSessionMessageChunk:
		s.onRelay(sub, u)
	case acp.UpdateState:
		s.onState(sub, u)
	case acp.UpdateToolCall, acp.UpdateToolCallUpdate, acp.UpdateToolCallContentChunk:
		s.onTool(sub, u)
	case acp.UpdateTerminal, acp.UpdateTerminalOutputChunk:
		s.onTerminal(sub, u)
	case acp.UpdatePlan:
		s.setPlan(sub, "", u.Entries)
	case acp.UpdatePlanV2:
		// 稳定版只有 items 一种；认不得的计划类型不显示
		if u.Plan != nil && u.Plan.Type == "items" {
			s.setPlan(sub, u.Plan.PlanID, u.Plan.Entries)
		}
	case acp.UpdateAvailableCommands:
		if sub != "" {
			return
		}
		s.state.Commands = commandsOf(u.AvailableCommands)
		s.sink.State(s.state)
	case acp.UpdateConfigOption:
		if sub != "" || len(u.ConfigOptions) == 0 {
			return
		}
		s.configs = u.ConfigOptions
		s.reportConfig()
	case acp.UpdateSessionInfo:
		if u.Title == nil || *u.Title == "" {
			return
		}
		// v2 子 Agent 的标题从它自己的会话信息来
		if sub != "" {
			if item := s.subs[sub]; item != nil && item.Subagent.Title != *u.Title {
				item.Subagent.Title = *u.Title
				s.emit(item, true)
				s.syncSubagents()
			}
			return
		}
		if !s.titleCustom {
			s.sink.Title(*u.Title)
		}
	case acp.UpdateUsage:
		if sub != "" {
			st := s.subState(sub)
			if u.Used != nil {
				st.ContextUsed = *u.Used
			}
			if u.Size != nil {
				st.ContextSize = *u.Size
			}
			s.sink.State(s.state)
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

// commandsOf Agent 给的斜杠命令（v2 认不得的命令类型不要）。
func commandsOf(in []acp.AvailableCommand) []Command {
	out := make([]Command, 0, len(in))
	for _, c := range in {
		if c.Type != "" && c.Type != "command" {
			continue
		}
		cmd := Command{Name: c.Name, Description: c.Description}
		if c.Input != nil {
			cmd.Hint = c.Input.Hint
		}
		out = append(out, cmd)
	}
	return out
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
	// v2 往内容里追加一项：只动内容
	if u.Kind == acp.UpdateToolCallContentChunk {
		if c, ok := u.ToolContentItem(); ok {
			s.applyToolContent(sub, key, tool, []acp.ToolCallContent{c}, true)
			s.emit(item, true)
		}
		return
	}
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
		s.applyToolContent(sub, key, tool, content, false)
	}
	if cancel, ok := u.CanCancel(); ok {
		tool.toolCancel = cancel
	}
	refreshCancel(tool)
	// 有的 Agent 把结果放在 content 以外（rawOutput），由驱动取出来
	if whole := s.driver.ToolOutput(u.RawOutput); whole != "" {
		tool.Output = keepTail(whole)
	}
	if (tool.Status == "completed" || tool.Status == "failed") && item.Ms == 0 {
		item.Ms = max(time.Now().UnixMilli()-item.At, 1)
	}
	s.emit(item, true)
}

// applyToolContent 工具调用的内容原样记下：文字输出（只留末尾）、产出的图片（存成文件）、文件改动（v1 的新旧文本、
// v2 的文件级改动和 patch）、显示输出的终端（v2）、派出的子 Agent。appendMode 为 true 时（v2 追加的一项）接在已有的后面，
// 否则整份替换。
func (s *Session) applyToolContent(sub, key string, tool *Tool, content []acp.ToolCallContent, appendMode bool) {
	var output strings.Builder
	var diffs []Diff
	var images []string
	wrote := false
	if appendMode {
		output.WriteString(tool.Output)
		diffs, images = tool.Diffs, tool.Images
	}
	for _, c := range content {
		switch c.Type {
		case "content":
			if c.Content == nil {
				continue
			}
			switch c.Content.Type {
			case "text":
				if output.Len() > 0 {
					output.WriteString("\n")
				}
				output.WriteString(c.Content.Text)
				wrote = true
			case "image":
				if file := s.saveImage(c.Content); file != "" {
					images = append(images, file)
				}
			}
		case "diff":
			if len(c.Changes) > 0 {
				diffs = append(diffs, s.diffsOf(c)...)
			} else {
				diffs = append(diffs, Diff{Path: agentfs.Rel(s.cwd, c.Path), OldText: c.OldText, NewText: c.NewText})
			}
		case "terminal":
			if c.TerminalID != "" {
				t := s.terminalOf(sub, c.TerminalID)
				t.toolKey, tool.terminal = key, c.TerminalID
				s.showTerminal(t)
			}
		case "session":
			tool.Child = c.SessionID
		}
	}
	if diffs != nil {
		tool.Diffs = diffs
	}
	if images != nil {
		tool.Images = images
	}
	// 这次有文字才动输出：终端的输出由终端写，不能被旧的文字盖回去
	if wrote {
		tool.Output = keepTail(output.String())
	}
}

// keepTail 工具输出只留末尾 outputKeep 字节；截掉前面时从一个完整的 UTF-8 字符开始，前面标上「…」。
func keepTail(text string) string {
	if len(text) <= outputKeep {
		return text
	}
	cut := len(text) - outputKeep
	for cut < len(text) && !utf8.RuneStart(text[cut]) {
		cut++
	}
	return "…\n" + text[cut:]
}

// saveImage 工具输出里的一张图片（base64）存进 mediaDir，返回相对它的文件名；存不了为空。
func (s *Session) saveImage(b *acp.ContentBlock) string {
	if s.mediaDir == "" || b.Data == "" {
		return ""
	}
	data, err := base64.StdEncoding.DecodeString(b.Data)
	if err != nil {
		return ""
	}
	path, err := agentfs.Save(s.mediaDir, imageName(b), data)
	if err != nil {
		return ""
	}
	return agentfs.Rel(s.mediaDir, path)
}

// imageName 图片的文件名：Agent 说了它存在哪就沿用那个名字，否则按类型起一个。
func imageName(b *acp.ContentBlock) string {
	if name := b.URI[strings.LastIndexAny(b.URI, `/\`)+1:]; name != "" {
		return name
	}
	ext, _, _ := strings.Cut(strings.TrimPrefix(b.MimeType, "image/"), "+")
	if ext == "" {
		ext = "png"
	}
	return "image." + ext
}

// onSubagent 子 Agent 的出现和状态变化。
func (s *Session) onSubagent(u *acp.SessionUpdate) {
	item := s.subs[u.SessionID]
	if item == nil {
		item = s.newItem(ItemSubagent, "")
		// 状态只记它报的：有的通知只带标题、目录（恢复出来的上级），没报状态就不当它在跑
		item.Subagent = &Subagent{ID: u.SessionID, StartAt: item.At}
		s.subs[u.SessionID] = item
	}
	sa := item.Subagent
	if u.Title != nil && *u.Title != "" {
		sa.Title = *u.Title
	}
	if u.Description != "" {
		sa.Description = u.Description
	}
	if u.Cwd != nil {
		sa.Cwd = agentfs.Rel(s.cwd, *u.Cwd)
	}
	if cancel, ok := u.CanCancel(); ok {
		sa.Cancel = cancel
	}
	if state := u.StateName(); state != "" {
		s.setSubagentState(u.SessionID, sa, state)
	}
	s.emit(item, true)
	s.syncSubagents()
}

// subagentWorking 子 Agent 报的状态是在干活：跑着，或者等人回应。
func subagentWorking(state string) bool {
	return state == acp.StateRunning || state == acp.StateRequiresAction
}

// syncSubagents 子 Agent 的样子变了：整份报给页面（State.Subagents，和进程信息里的是同一份）。
// 有没有子 Agent 在干活变了、主会话又没有一轮在跑时，告诉调用方（会话状态跟着变）；有一轮在跑时由这一轮结束时一起报。
func (s *Session) syncSubagents() {
	items := make([]*Item, 0, len(s.subs))
	on := false
	for _, item := range s.subs {
		items = append(items, item)
		on = on || subagentWorking(item.Subagent.State)
	}
	slices.SortFunc(items, func(a, b *Item) int { return cmp.Compare(a.Seq, b.Seq) })
	list := make([]Subagent, 0, len(items))
	for _, item := range items {
		list = append(list, *item.Subagent)
	}
	s.state.Subagents = list
	s.sink.State(s.state)
	if on == s.subsOn {
		return
	}
	s.subsOn = on
	if s.turn == nil && !s.gone {
		s.sink.SubagentsWorking(on)
	}
}

// stopSubagents 进程没了：子 Agent 也跟着没了，还在干活的记成停下（不能再停），免得一直显示在跑。
func (s *Session) stopSubagents() {
	changed := false
	for sub, item := range s.subs {
		sa := item.Subagent
		if !subagentWorking(sa.State) && !sa.Cancel {
			continue
		}
		if subagentWorking(sa.State) {
			s.setSubagentState(sub, sa, acp.StateIdle)
		}
		sa.Cancel = false
		s.emit(item, true)
		changed = true
	}
	if changed {
		s.syncSubagents()
	}
}

// turnStartedLocked 主会话的一轮开始了：页面上的「一轮在进行」跟着变，会话改成运行中。
func (s *Session) turnStartedLocked() {
	s.state.Turn = true
	s.sink.State(s.state)
	s.sink.TurnStarted()
}

func (s *Session) RequestPermission(reqID string, req *acp.RequestPermissionRequest, reply func(*acp.RequestPermissionResponse)) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.sideOf(req.SessionID) != nil {
		reply(acp.PermissionCancelled())
		return
	}
	// v1 的批准只属于正在进行的一轮；v2 里后台的工作也会要批准，而且停下时由 Agent 自己撤回
	if !s.v2 && (s.turn == nil || s.turn.cancelled) {
		reply(acp.PermissionCancelled())
		return
	}
	sub := ""
	if req.SessionID != "" && req.SessionID != s.acpID {
		sub = req.SessionID
	}
	s.closeOpen(sub)
	item := s.newItem(ItemPermission, "")
	tc := req.Tool()
	perm := &Permission{ID: uuid.NewString(), ToolID: tc.ToolCallID, Kind: tc.Kind, Title: req.Title, Description: req.Description}
	if perm.Title == "" && tc.Title != nil {
		perm.Title = *tc.Title
	}
	perm.Command = s.driver.Command(tc.RawInput)
	if req.Subject != nil && req.Subject.Command != "" {
		perm.Command = req.Subject.Command
	}
	if tool := s.tools[toolKey(sub, tc.ToolCallID)]; tool != nil {
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
	s.perms[perm.ID] = &pending{item: item, reply: reply, req: reqID}
	s.permReqs[reqID] = perm.ID
	s.emit(item, true)
	s.sink.Asked(perm)
}

// Resolve 回答一张批准卡片。
func (s *Session) Resolve(requestID, optionID string) error {
	s.mu.Lock()
	p := s.perms[requestID]
	if p == nil || p.item.Permission == nil {
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
	delete(s.permReqs, p.req)
	perm.Outcome, perm.OptionID, perm.OptionKind = "selected", optionID, kind
	s.emit(p.item, true)
	waiting := len(s.perms) > 0
	s.mu.Unlock()

	p.reply(acp.PermissionSelected(optionID))
	s.sink.Answered(requestID, kind, waiting)
	return nil
}

// cancelPending 这一轮被停止：还在等的批准、提问都按「取消」回给 Agent。在 s.mu 里调用，返回被取消的卡片 id。
func (s *Session) cancelPending() []string {
	var ids []string
	for id, p := range s.perms {
		delete(s.perms, id)
		delete(s.permReqs, p.req)
		p.markCancelled()
		s.emit(p.item, true)
		p.replyCancelled()
		ids = append(ids, id)
	}
	return ids
}

// —— 一轮 ——

// 一轮还没结束时再发消息的两种方式。
const (
	// PromptSteer 插进正在跑的一轮：v2 直接发 session/prompt，Agent 自己插进去或排在它那边；v1 没有这回事，按排队办。
	PromptSteer = "steer"
	// PromptQueue 排队：Momoko 存着，这一轮结束后按顺序作为下一轮发出。
	PromptQueue = "queue"
)

// Prompt 记下用户的话并发给 Agent，立刻返回；这一轮的内容通过 Sink 陆续报出来，结束时报 TurnEnded。
// 一轮还没结束时不拒绝：mode 是 PromptSteer 且是 v2 就插进去，否则排队。
func (s *Session) Prompt(text string, uploads []Upload, mode string) error {
	s.mu.Lock()
	if s.gone || s.stopping {
		s.mu.Unlock()
		return errors.New("Agent 进程刚刚结束了，再发一次")
	}
	if s.turn == nil {
		send := s.beginLocked(text, uploads)
		s.mu.Unlock()
		send()
		return nil
	}
	if mode == PromptSteer && s.v2 {
		s.userItemLocked(text, uploads, true)
		s.mu.Unlock()
		go s.sendV2(blocksOf(text, uploads), nil)
		return nil
	}
	s.queue = append(s.queue, &queued{id: uuid.NewString(), text: text, uploads: uploads})
	s.syncQueueLocked()
	s.mu.Unlock()
	return nil
}

// beginLocked 开始新的一轮（在 s.mu 里调用）：记下用户的话，返回把它发出去的事（在锁外调用）。
func (s *Session) beginLocked(text string, uploads []Upload) func() {
	turn := &turnRun{started: time.Now()}
	s.turn = turn
	// 工具调用、消息只在这一轮里更新：上一轮的不会再变，留着只会越积越多（v2 后台还在跑的除外）。
	s.resetTurnState()
	s.lastActive = time.Now()
	s.turnStartedLocked()
	s.userItemLocked(text, uploads, false)
	blocks := blocksOf(text, uploads)
	if s.v2 {
		return func() { go s.sendV2(blocks, turn) }
	}
	return func() { go s.runTurn(blocks) }
}

// userItemLocked 记一条用户的话；steer 表示是插进正在跑的一轮的。
func (s *Session) userItemLocked(text string, uploads []Upload, steer bool) {
	item := s.newItem(ItemUser, "")
	item.Text, item.Steer = text, steer
	for _, u := range uploads {
		item.Attachments = append(item.Attachments, Attachment{Kind: u.Kind, Name: u.Name, File: u.File})
	}
	s.emit(item, true)
}

// sendQueued 一轮结束了：排着的第一条作为下一轮发出去。
func (s *Session) sendQueued() {
	s.mu.Lock()
	if s.turn != nil || len(s.queue) == 0 || s.gone || s.stopping {
		s.mu.Unlock()
		return
	}
	q := s.queue[0]
	s.queue = s.queue[1:]
	s.syncQueueLocked()
	send := s.beginLocked(q.text, q.uploads)
	s.mu.Unlock()
	send()
}

// Unqueue 撤回一条排着队的消息。
func (s *Session) Unqueue(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i, q := range s.queue {
		if q.id == id {
			s.queue = append(s.queue[:i:i], s.queue[i+1:]...)
			s.syncQueueLocked()
			return nil
		}
	}
	return errors.New("这条消息已经发出去了")
}

// SendNow 排着队的一条马上发。v2 插进正在跑的一轮；v1 插不进去，就停下这一轮，排到最前面，这一轮收尾后作为下一轮发出
// （排在它后面的接着排）。这一轮刚好已经结束时，它排到最前面直接发。
func (s *Session) SendNow(id string) error {
	s.mu.Lock()
	idx := -1
	for i, q := range s.queue {
		if q.id == id {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.mu.Unlock()
		return errors.New("这条消息已经发出去了")
	}
	q := s.queue[idx]
	rest := append(s.queue[:idx:idx], s.queue[idx+1:]...)
	if s.turn != nil && s.v2 {
		s.queue = rest
		s.syncQueueLocked()
		s.userItemLocked(q.text, q.uploads, true)
		s.mu.Unlock()
		go s.sendV2(blocksOf(q.text, q.uploads), nil)
		return nil
	}
	s.queue = append([]*queued{q}, rest...)
	s.syncQueueLocked()
	if s.turn == nil {
		s.mu.Unlock()
		go s.sendQueued()
		return nil
	}
	// v1：停下这一轮，但不清掉排着的消息
	s.turn.cancelled = true
	cancelled := s.cancelPending()
	s.mu.Unlock()
	for _, id := range cancelled {
		s.sink.Answered(id, "", false)
	}
	_ = s.client.Cancel(s.acpID)
	return nil
}

// syncQueueLocked 排队的消息报给页面（在 s.mu 里调用）。
func (s *Session) syncQueueLocked() {
	s.state.Queue = make([]Queued, 0, len(s.queue))
	for _, q := range s.queue {
		out := Queued{ID: q.id, Text: q.text}
		for _, u := range q.uploads {
			out.Attachments = append(out.Attachments, Attachment{Kind: u.Kind, Name: u.Name, File: u.File})
		}
		s.state.Queue = append(s.state.Queue, out)
	}
	s.sink.State(s.state)
}

// runTurn v1：Prompt 的回复就是这一轮的结束。
func (s *Session) runTurn(blocks []acp.ContentBlock) {
	resp, err := s.client.Prompt(context.Background(), s.acpID, blocks)

	s.mu.Lock()
	result := &Turn{}
	exited := false
	switch {
	case err == nil:
		result.Stop = resp.StopReason
		if resp.Usage != nil {
			result.Usage = usageOf(resp.Usage)
		}
	case s.turn.cancelled || s.stopping:
		result.Stop = acp.StopCancelled
	default:
		result, exited = s.turnError(err)
	}
	report := s.endTurnLocked(result)
	s.mu.Unlock()

	report()
	if exited {
		s.proc.Kill()
		s.remove()
	}
}

// turnError 发消息出错时这一轮的结果；exited 表示是进程没了。
func (s *Session) turnError(err error) (*Turn, bool) {
	result := &Turn{Stop: acp.StopError}
	var rpcErr *acp.Error
	switch {
	case errors.Is(err, acp.ErrClosed) || s.proc.Exited():
		result.Code, result.Error = TurnProcessExited, s.proc.LastError()
		return result, true
	case errors.As(err, &rpcErr):
		result.Error = rpcErr.Message
		if rpcErr.Code == acp.CodeAuthRequired {
			result.Code = TurnAuthRequired
		}
	default:
		result.Error = err.Error()
	}
	return result, false
}

// turnErrorText 发消息出错的原因，给人看。
func (s *Session) turnErrorText(err error) string {
	var rpcErr *acp.Error
	if errors.As(err, &rpcErr) {
		return rpcErr.Message
	}
	return err.Error()
}

// endTurnLocked 收尾这一轮（在 s.mu 里调用）：正在写的文字收尾，没结果的工具调用别再转圈，记一条一轮结束。
// v1 里还在等的批准一并取消（v2 的由 Agent 自己撤回）；v2 后台还在跑的工具调用、子 Agent 里的不动。
// 返回要在锁外做的事：报批准被取消、报这一轮结束。
func (s *Session) endTurnLocked(result *Turn) func() {
	turn := s.turn
	for sub := range s.open {
		s.closeOpen(sub)
	}
	var cancelled []string
	if !s.v2 {
		cancelled = s.cancelPending()
	}
	result.Ms = time.Since(turn.started).Milliseconds()
	if turn.cancelled && result.Stop == "" {
		result.Stop = acp.StopCancelled
	}
	for key, tool := range s.tools {
		if s.background[key] || (s.v2 && tool.Sub != "") || !unfinished(tool.Tool.Status) {
			continue
		}
		tool.Tool.Status = "failed"
		if result.Stop == acp.StopCancelled {
			tool.Tool.Status = "cancelled"
		}
		refreshCancel(tool.Tool)
		s.emit(tool, true)
	}
	item := s.newItem(ItemTurn, "")
	item.Turn = result
	s.emit(item, true)
	s.turn = nil
	s.lastActive = time.Now()
	s.state.Turn = false
	s.sink.State(s.state)
	end := TurnEnd{Turn: result, Working: s.subsOn, LastText: turn.lastText, State: s.state, ContextUsed: s.contextUsed, ContextSize: s.contextSize}
	return func() {
		for _, id := range cancelled {
			s.sink.Answered(id, "", false)
		}
		s.sink.TurnEnded(end)
		s.sendQueued()
	}
}

// Cancel 停止当前这一轮。v1 还在等的批准马上按取消回；v2 由 Agent 撤回它不再需要的批准，后台的工作不受影响。
func (s *Session) Cancel() {
	s.mu.Lock()
	if s.turn == nil {
		s.mu.Unlock()
		return
	}
	s.turn.cancelled = true
	var cancelled []string
	if !s.v2 {
		cancelled = s.cancelPending()
	}
	if len(s.queue) > 0 {
		s.queue = nil
		s.syncQueueLocked()
	}
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
