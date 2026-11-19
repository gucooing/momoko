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
}

func capsOf(init *acp.InitializeResponse) Caps {
	c := init.AgentCapabilities
	return Caps{
		Resume:          c.SessionCapabilities.Resume != nil,
		Fork:            c.SessionCapabilities.Fork != nil,
		AdditionalDirs:  c.SessionCapabilities.AdditionalDirectories != nil,
		Image:           c.PromptCapabilities.Image,
		EmbeddedContext: c.PromptCapabilities.EmbeddedContext,
		McpHTTP:         c.McpCapabilities.HTTP,
		McpSSE:          c.McpCapabilities.SSE,
		Close:           c.SessionCapabilities.Close != nil,
		Subagents:       c.SessionCapabilities.Subagents != nil,
	}
}

func clientInfo() acp.Implementation {
	return acp.Implementation{Name: "momoko", Title: "Momoko", Version: "1"}
}

// clientCaps 声明支持子 Agent 的子会话：Agent 才会把子 Agent 作为独立的会话流上报。
func clientCaps() acp.ClientCapabilities {
	return acp.ClientCapabilities{Subagents: &struct{}{}}
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
	// ContextWindow / MaxOutput 用户在管理页给这个模型设的最长上下文和最长输出（token 数），0 = 用 Agent 自己的。
	// 不是 Agent 给的，刷新清单时按模型 id 沿用。
	ContextWindow int64 `json:"contextWindow,omitempty"`
	MaxOutput     int64 `json:"maxOutput,omitempty"`
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

// MergeCatalog 用新看到的可选项刷新旧的：这次只读到当前模型的推理强度，其它模型的沿用之前读到的；
// 用户给模型设的最长上下文和最长输出按模型 id 沿用。
// keepDefaults 为 true 时默认模型和默认权限模式沿用旧的（会话里用户切过去的值不算默认）。
func MergeCatalog(old, fresh Catalog, keepDefaults bool) Catalog {
	if len(fresh.Modes) == 0 {
		fresh.Modes, fresh.DefaultMode = old.Modes, old.DefaultMode
	}
	if len(fresh.Models) == 0 {
		fresh.Models, fresh.DefaultModel = old.Models, old.DefaultModel
	} else {
		known := map[string]CatalogModel{}
		for _, m := range old.Models {
			known[m.ID] = m
		}
		for i := range fresh.Models {
			m := &fresh.Models[i]
			prev, ok := known[m.ID]
			if !ok {
				continue
			}
			if len(m.Efforts) == 0 {
				m.Efforts, m.DefaultEffort = prev.Efforts, prev.DefaultEffort
			}
			m.ContextWindow, m.MaxOutput = prev.ContextWindow, prev.MaxOutput
		}
	}
	if keepDefaults {
		if old.DefaultModel != "" {
			fresh.DefaultModel = old.DefaultModel
		}
		if old.DefaultMode != "" {
			fresh.DefaultMode = old.DefaultMode
		}
	}
	return fresh
}

// ProbeResult 检测一次 Agent 得到的东西。
type ProbeResult struct {
	// AgentName / AgentVersion Agent 自报的名字和版本（agentInfo），用来认它是哪一系。
	AgentName    string
	AgentVersion string
	Caps         Caps
	// Auth 这个环境下已有的登录；没登录时 ok 为 false。
	Auth   acp.AuthAccount
	AuthOK bool
	// Catalog 能开会话时读到的可选项。
	Catalog Catalog
	// NeedAuth 能启动、能握手，但新建会话要求先登录。
	NeedAuth bool
}

// probeEvents 检测用的事件接收器：只关心登录状态。
type probeEvents struct {
	mu   sync.Mutex
	auth acp.AuthState
}

func (e *probeEvents) SessionUpdate(*acp.SessionNotification) {}
func (e *probeEvents) RequestPermission(_ *acp.RequestPermissionRequest, reply func(*acp.RequestPermissionResponse)) {
	reply(acp.PermissionCancelled())
}
func (e *probeEvents) AuthStatus(state acp.AuthState) {
	e.mu.Lock()
	e.auth = state
	e.mu.Unlock()
}
func (e *probeEvents) account() (acp.AuthAccount, bool) {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.auth.Account()
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

// Probe 启动一次进程：握手，在 cwd（一个临时目录）里开一个会话，读它给的权限模式、模型和每个模型的推理强度。
// 用的是服务器上已有的登录。只改配置，不发消息，不产生用量。
func Probe(ctx context.Context, launch agentdriver.Launch, cwd string) (*ProbeResult, error) {
	if err := os.MkdirAll(cwd, 0o755); err != nil {
		return nil, err
	}
	proc, err := agentproc.Start(agentproc.Config{Command: launch.Command, Args: launch.Args, Dir: cwd, Env: launch.Env})
	if err != nil {
		return nil, err
	}
	defer proc.Kill()
	events := &probeEvents{}
	client := acp.NewClient(proc.Stdout(), proc.Stdin(), events)
	defer client.Close()

	initCtx, cancel := context.WithTimeout(ctx, 90*time.Second)
	defer cancel()
	init, err := client.Initialize(initCtx, clientInfo(), clientCaps())
	if err != nil {
		return nil, exitReason(proc, err)
	}
	result := &ProbeResult{Caps: capsOf(init)}
	if init.AgentInfo != nil {
		result.AgentName, result.AgentVersion = init.AgentInfo.Name, init.AgentInfo.Version
	}

	state, err := client.NewSession(initCtx, acp.SessionSetup{Cwd: cwd})
	if err != nil {
		if acp.IsAuthRequired(err) {
			result.NeedAuth = true
			result.Auth, result.AuthOK = events.account()
			return result, nil
		}
		return nil, exitReason(proc, err)
	}
	result.Catalog = catalogOf(state.ConfigOptions)

	// 推理强度跟着模型变：逐个切过去读一遍。
	if model := configOf(state.ConfigOptions, acp.CategoryModel); model != nil && len(result.Catalog.Models) <= 12 {
		for i := range result.Catalog.Models {
			m := &result.Catalog.Models[i]
			if m.ID == model.Current() {
				continue
			}
			stepCtx, stepCancel := context.WithTimeout(ctx, 15*time.Second)
			options, err := client.SetConfigOption(stepCtx, state.SessionID, model.ID, m.ID)
			stepCancel()
			if err != nil {
				continue
			}
			m.Efforts, m.DefaultEffort = effortsOf(options)
		}
	}
	if result.Caps.Close {
		closeCtx, closeCancel := context.WithTimeout(ctx, 5*time.Second)
		_ = client.CloseSession(closeCtx, state.SessionID)
		closeCancel()
	}
	result.Auth, result.AuthOK = events.account()
	return result, nil
}
