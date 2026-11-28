package agentrun

import (
	"context"
	"errors"
	"os"
	"sync"
	"time"

	"momoko/pkg/acp"
	"momoko/pkg/agentdriver"
	"momoko/pkg/agentproc"
)

// Caps Agent 在初始化时声明的能力。没声明的，对应的界面元素不出现。
type Caps struct {
	Resume          bool `json:"resume,omitempty"`
	Fork            bool `json:"fork,omitempty"`
	AdditionalDirs  bool `json:"additionalDirs,omitempty"`
	Image           bool `json:"image,omitempty"`
	EmbeddedContext bool `json:"embeddedContext,omitempty"`
	McpHTTP         bool `json:"mcpHttp,omitempty"`
	McpSSE          bool `json:"mcpSse,omitempty"`
	Close           bool `json:"close,omitempty"`
	Subagents       bool `json:"subagents,omitempty"`
	// Protocol 协商好的 ACP 版本。v2 里一个工具调用、一个子 Agent 也能单独停。
	Protocol int `json:"protocol,omitempty"`
	// Steer 一轮还没结束时能把消息插进去（v2）；不能的只能排队。
	Steer bool `json:"steer,omitempty"`
}

func capsOf(init *acp.InitializeResponse) Caps {
	f := init.Features()
	return Caps{
		Resume:          f.Resume,
		Fork:            f.Fork,
		AdditionalDirs:  f.AdditionalDirs,
		Image:           f.Image,
		EmbeddedContext: f.EmbeddedContext,
		McpHTTP:         f.McpHTTP,
		McpSSE:          f.McpSSE,
		Close:           f.Close,
		Subagents:       f.Subagents,
		Protocol:        init.ProtocolVersion,
		Steer:           init.ProtocolVersion >= acp.ProtocolV2,
	}
}

func clientInfo() acp.Implementation {
	return acp.Implementation{Name: "momoko", Title: "Momoko", Version: "1"}
}

// clientCaps 声明支持子 Agent 的子会话（Agent 才会把子 Agent 作为独立的会话流上报），
// 和表单询问（Agent 才会向人提问，比如 Claude 的 AskUserQuestion；不声明它就不开这个工具）。
func clientCaps() acp.ClientCapabilities {
	return acp.ClientCapabilities{Subagents: &struct{}{}, Elicitation: &acp.ElicitationCapabilities{Form: &struct{}{}}}
}

// Choice 一个可选值。
type Choice struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// CatalogModel 一个模型和它能选的推理强度。
type CatalogModel struct {
	ID            string   `json:"id"`
	Name          string   `json:"name"`
	Efforts       []Choice `json:"efforts,omitempty"`
	DefaultEffort string   `json:"defaultEffort,omitempty"`
}

// Catalog Agent 自己给出的可选项：权限模式、模型、每个模型的推理强度。客户端只能在它列出的值里选。
type Catalog struct {
	Modes        []Choice       `json:"modes,omitempty"`
	DefaultMode  string         `json:"defaultMode,omitempty"`
	Models       []CatalogModel `json:"models,omitempty"`
	DefaultModel string         `json:"defaultModel,omitempty"`
}

func configOf(options []acp.ConfigOption, category string) *acp.ConfigOption {
	for i := range options {
		if options[i].Category == category && options[i].Type == "select" {
			return &options[i]
		}
	}
	return nil
}

func choicesOf(opt *acp.ConfigOption) []Choice {
	values := opt.Values()
	out := make([]Choice, 0, len(values))
	for _, v := range values {
		out = append(out, Choice{ID: v.Value, Name: v.Name, Description: v.Description})
	}
	return out
}

func effortsOf(options []acp.ConfigOption) ([]Choice, string) {
	opt := configOf(options, acp.CategoryThoughtLevel)
	if opt == nil {
		return nil, ""
	}
	return choicesOf(opt), opt.Current()
}

// catalogOf 从 Agent 给的配置项里取可选项；推理强度只有当前模型的。
func catalogOf(options []acp.ConfigOption) Catalog {
	catalog := Catalog{}
	if opt := configOf(options, acp.CategoryMode); opt != nil {
		catalog.Modes, catalog.DefaultMode = choicesOf(opt), opt.Current()
	}
	if opt := configOf(options, acp.CategoryModel); opt != nil {
		efforts, current := effortsOf(options)
		for _, v := range opt.Values() {
			m := CatalogModel{ID: v.Value, Name: v.Name}
			if v.Value == opt.Current() {
				m.Efforts, m.DefaultEffort = efforts, current
			}
			catalog.Models = append(catalog.Models, m)
		}
		catalog.DefaultModel = opt.Current()
	}
	return catalog
}

// ProbeResult 检测一次 Agent 得到的东西。
type ProbeResult struct {
	// AgentName / AgentVersion Agent 自报的名字和版本（agentInfo），用来认它是哪一系。
	AgentName    string
	AgentVersion string
	Caps         Caps
	// Auth 这个环境下已有的登录；Agent 没报或报的是没登录时 AuthOK 为 false。
	Auth   acp.AuthStatus
	AuthOK bool
	// Catalog 能开会话时读到的可选项。
	Catalog Catalog
	// NeedAuth 能启动、能握手，但新建会话要求先登录。
	NeedAuth bool
}

// probeEvents 检测用的事件接收器：只关心登录状态。
type probeEvents struct {
	mu   sync.Mutex
	auth acp.AuthStatus
}

func (e *probeEvents) SessionUpdate(*acp.SessionNotification) {}
func (e *probeEvents) RequestPermission(_ string, _ *acp.RequestPermissionRequest, reply func(*acp.RequestPermissionResponse)) {
	reply(acp.PermissionCancelled())
}
func (e *probeEvents) Elicit(_ string, _ *acp.CreateElicitationRequest, reply func(*acp.CreateElicitationResponse)) {
	reply(&acp.CreateElicitationResponse{Action: acp.ElicitationCancel})
}
func (e *probeEvents) RequestCancelled(string) {}
func (e *probeEvents) AuthStatus(status acp.AuthStatus) {
	e.mu.Lock()
	e.auth = status
	e.mu.Unlock()
}
func (e *probeEvents) account() (acp.AuthStatus, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.auth, e.auth.LoggedIn()
}

// exitReason 进程没撑住时给人看的原因。
func exitReason(proc *agentproc.Process, err error) error {
	if errors.Is(err, acp.ErrClosed) || proc.Exited() {
		if reason := proc.LastError(); reason != "" {
			return errors.New("Agent 进程启动后就退出了：" + reason)
		}
		return errors.New("Agent 进程启动后就退出了")
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("Agent 进程一直没有响应")
	}
	var rpcErr *acp.Error
	if errors.As(err, &rpcErr) {
		return errors.New(rpcErr.Message)
	}
	return err
}

// probeConn 检查用的一次进程：已经握过手。用的是服务器上已有的登录，不发消息，不产生用量。
type probeConn struct {
	proc   *agentproc.Process
	client *acp.Client
	events *probeEvents
	init   *acp.InitializeResponse
	cwd    string
}

// openProbe 在 cwd（一个临时目录）里启动一次进程并握手；用完 close。
func openProbe(ctx context.Context, launch agentdriver.Launch, cwd string) (*probeConn, error) {
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		return nil, err
	}
	proc, err := agentproc.Start(agentproc.Config{Command: launch.Command, Args: launch.Args, Dir: cwd, Env: launch.Env, Sandbox: launch.Sandbox})
	if err != nil {
		return nil, err
	}
	p := &probeConn{proc: proc, events: &probeEvents{}, cwd: cwd}
	p.client = acp.NewClient(proc.Stdout(), proc.Stdin(), p.events)
	initCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	if p.init, err = p.client.Initialize(initCtx, clientInfo(), clientCaps()); err != nil {
		p.close()
		return nil, exitReason(proc, err)
	}
	return p, nil
}

func (p *probeConn) close() {
	p.client.Close()
	p.proc.Kill()
}

// newSession 在 cwd 里开一个会话；错误原样返回（调用方要看是不是要求先登录）。
func (p *probeConn) newSession(ctx context.Context) (*acp.SessionState, error) {
	stepCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	return p.client.NewSession(stepCtx, acp.SessionSetup{Cwd: p.cwd})
}

// closeSession Agent 支持时把开的会话关掉，让它收拾干净。
func (p *probeConn) closeSession(ctx context.Context, id string) {
	if !capsOf(p.init).Close {
		return
	}
	closeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	_ = p.client.CloseSession(closeCtx, id)
}

// Probe 启动一次进程：握手，在 cwd（一个临时目录）里开一个会话，读它开会话时给的权限模式、模型和当前模型的推理强度。
func Probe(ctx context.Context, launch agentdriver.Launch, cwd string) (*ProbeResult, error) {
	p, err := openProbe(ctx, launch, cwd)
	if err != nil {
		return nil, err
	}
	defer p.close()
	result := &ProbeResult{Caps: capsOf(p.init)}
	// v1 在 agentInfo、v2 在 info：按协商好的版本取
	if agent := p.init.Agent(); agent != nil {
		result.AgentName, result.AgentVersion = agent.Name, agent.Version
	}

	state, err := p.newSession(ctx)
	if err != nil {
		if acp.IsAuthRequired(err) {
			result.NeedAuth = true
			result.Auth, result.AuthOK = p.events.account()
			return result, nil
		}
		return nil, exitReason(p.proc, err)
	}
	result.Catalog = catalogOf(state.ConfigOptions)
	p.closeSession(ctx, state.SessionID)
	result.Auth, result.AuthOK = p.events.account()
	return result, nil
}

// ProbeEfforts 启动一次进程，在 cwd（一个临时目录）里开一个会话，切到 model，读 Agent 给这个模型的推理强度。
// 要先登录时返回 ErrAuthRequired；Agent 没有列出这个模型时返回错误。
func ProbeEfforts(ctx context.Context, launch agentdriver.Launch, cwd, model string) ([]Choice, error) {
	p, err := openProbe(ctx, launch, cwd)
	if err != nil {
		return nil, err
	}
	defer p.close()
	state, err := p.newSession(ctx)
	if err != nil {
		if acp.IsAuthRequired(err) {
			return nil, ErrAuthRequired
		}
		return nil, exitReason(p.proc, err)
	}
	defer p.closeSession(ctx, state.SessionID)
	options := state.ConfigOptions
	opt := configOf(options, acp.CategoryModel)
	if opt == nil || !hasValue(opt, model) {
		return nil, errors.New("Agent 没有列出这个模型")
	}
	if opt.Current() != model {
		stepCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
		defer cancel()
		if options, err = p.client.SetConfigOption(stepCtx, state.SessionID, opt.ID, model); err != nil {
			return nil, exitReason(p.proc, err)
		}
	}
	efforts, _ := effortsOf(options)
	return efforts, nil
}
