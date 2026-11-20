package acp

import (
	"encoding/json"
	"errors"

	"momoko/pkg/jsonrpc"
)

// 协议版本。Momoko 两个都说：握手时请求 v2，Agent 只会 v1 时回 1，之后按 v1 说。
const (
	ProtocolV1 = 1
	ProtocolV2 = 2
	// ProtocolVersion 握手时请求的版本（Momoko 支持的最新版本）。
	ProtocolVersion = ProtocolV2
)

// ACP 约定的错误码（JSON-RPC 标准错误码之外）。
const (
	// CodeAuthRequired Agent 需要先登录。
	CodeAuthRequired = -32000
	// CodeResourceNotFound 会话或文件不存在。
	CodeResourceNotFound = -32002
)

// Error 是 JSON-RPC 的错误对象。
type Error = jsonrpc.Error

// ErrClosed 连接已经关闭（Agent 进程退出，或主动关闭）。
var ErrClosed = jsonrpc.ErrClosed

// IsAuthRequired 这个错误是不是 Agent 要求先登录。
func IsAuthRequired(err error) bool {
	var rpcErr *Error
	return errors.As(err, &rpcErr) && rpcErr.Code == CodeAuthRequired
}

// 方法名。
const (
	MethodInitialize        = "initialize"
	MethodSessionNew        = "session/new"
	MethodSessionResume     = "session/resume"
	MethodSessionFork       = "session/fork"
	MethodSessionClose      = "session/close"
	MethodSessionPrompt     = "session/prompt"
	MethodSessionCancel     = "session/cancel"
	MethodSessionSetConfig  = "session/set_config_option"
	MethodSessionUpdate     = "session/update"
	MethodRequestPermission = "session/request_permission"
	// v2：只停一个工具调用 / 一个终端（它声明了 capabilities.cancel 时）
	MethodToolCallCancel = "tool_call/cancel"
	MethodTerminalCancel = "terminal/cancel"
	// MethodCancelRequest 对端撤回它发来的某个请求（v2 里 Agent 用它撤回批准）。
	MethodCancelRequest = "$/cancel_request"
	// MethodAuthStatusUpdate Agent 推来的登录身份（claude-agent-acp、codex-acp 的扩展，v1 v2 一样）。
	MethodAuthStatusUpdate = "_auth/status_update"
)

// 会话更新的种类（SessionUpdate.Kind）。
const (
	UpdateUserMessageChunk  = "user_message_chunk"
	UpdateAgentMessageChunk = "agent_message_chunk"
	UpdateAgentThoughtChunk = "agent_thought_chunk"
	UpdateToolCall          = "tool_call"
	UpdateToolCallUpdate    = "tool_call_update"
	UpdatePlan              = "plan"
	UpdateAvailableCommands = "available_commands_update"
	UpdateConfigOption      = "config_option_update"
	UpdateSessionInfo       = "session_info_update"
	UpdateUsage             = "usage_update"
	UpdateNotice            = "notice"
	UpdateSubagent          = "subagent_update"

	// v2 才有的
	UpdateUserMessage          = "user_message"
	UpdateAgentMessage         = "agent_message"
	UpdateAgentThought         = "agent_thought"
	UpdateState                = "state_update"
	UpdateToolCallContentChunk = "tool_call_content_chunk"
	UpdateTerminal             = "terminal_update"
	UpdateTerminalOutputChunk  = "terminal_output_chunk"
	UpdatePlanV2               = "plan_update"
	UpdateSessionMessage       = "session_message"
	UpdateSessionMessageChunk  = "session_message_chunk"
)

// v2 的会话状态（state_update）。
const (
	StateRunning        = "running"
	StateIdle           = "idle"
	StateRequiresAction = "requires_action"
	StateUnknown        = "unknown"
)

// 配置项的类别。
const (
	CategoryMode         = "mode"
	CategoryModel        = "model"
	CategoryThoughtLevel = "thought_level"
)

// 批准选项的种类。
const (
	PermissionAllowOnce    = "allow_once"
	PermissionAllowAlways  = "allow_always"
	PermissionRejectOnce   = "reject_once"
	PermissionRejectAlways = "reject_always"
)

// 一轮结束的原因。
const (
	StopEndTurn         = "end_turn"
	StopMaxTokens       = "max_tokens"
	StopMaxTurnRequests = "max_turn_requests"
	StopRefusal         = "refusal"
	StopCancelled       = "cancelled"
	// StopError v2：消息已经收下之后出错。
	StopError = "error"
)

// Meta 是协议里随处可带的 _meta 扩展字段。
type Meta map[string]json.RawMessage

// Marker 能力标记：v2 里出现（{}）就是支持，没有或 null 是不支持。
type Marker = *json.RawMessage

// Implementation 一端的名称和版本。
type Implementation struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// ClientCapabilities v1 的客户端能力。Momoko 不代 Agent 读写文件，也不提供终端，所以这两项都不声明。
type ClientCapabilities struct {
	// Subagents 不为空时，Agent 把子 Agent 作为独立的子会话上报（v1 草案；v2 不用声明）。
	Subagents *struct{} `json:"subagents,omitempty"`
	Meta      Meta      `json:"_meta,omitempty"`
}

// ClientCapabilitiesV2 v2 的客户端能力：登录和表单询问 Momoko 都不做，什么都不声明。
type ClientCapabilitiesV2 struct{}

// InitializeRequest 两种形状一起发：v2 Agent 读 info / capabilities，只会 v1 的 Agent 读 clientInfo / clientCapabilities，
// 各自忽略认不得的字段，这样不用为了 v1 再握一次手。
type InitializeRequest struct {
	ProtocolVersion    int                  `json:"protocolVersion"`
	Info               Implementation       `json:"info"`
	Capabilities       ClientCapabilitiesV2 `json:"capabilities"`
	ClientInfo         *Implementation      `json:"clientInfo,omitempty"`
	ClientCapabilities ClientCapabilities   `json:"clientCapabilities"`
}

// PromptCapabilities v1：Agent 能接受的消息内容。
type PromptCapabilities struct {
	Image           bool `json:"image"`
	Audio           bool `json:"audio"`
	EmbeddedContext bool `json:"embeddedContext"`
}

// McpCapabilities v1：Agent 支持的 MCP 传输方式；stdio 是必须支持的，不在这里。
type McpCapabilities struct {
	HTTP bool `json:"http"`
	SSE  bool `json:"sse"`
}

// SessionCapabilities v1：需要声明才能用的会话能力，字段出现就表示支持。
type SessionCapabilities struct {
	AdditionalDirectories Marker `json:"additionalDirectories,omitempty"`
	Fork                  Marker `json:"fork,omitempty"`
	Resume                Marker `json:"resume,omitempty"`
	Close                 Marker `json:"close,omitempty"`
	Subagents             Marker `json:"subagents,omitempty"`
}

type AgentCapabilities struct {
	PromptCapabilities  PromptCapabilities  `json:"promptCapabilities"`
	McpCapabilities     McpCapabilities     `json:"mcpCapabilities"`
	SessionCapabilities SessionCapabilities `json:"sessionCapabilities"`
	Meta                Meta                `json:"_meta,omitempty"`
}

// AgentCapabilitiesV2 v2 的 Agent 能力：会话相关的都在 session 下，支持标记都是对象。
// 声明了 session 就一定支持新建、列出、恢复、关闭、发消息、停止；fork 还是草案。
type AgentCapabilitiesV2 struct {
	Session *struct {
		Prompt *struct {
			Image           Marker `json:"image"`
			Audio           Marker `json:"audio"`
			EmbeddedContext Marker `json:"embeddedContext"`
		} `json:"prompt"`
		Mcp *struct {
			Stdio Marker `json:"stdio"`
			HTTP  Marker `json:"http"`
		} `json:"mcp"`
		AdditionalDirectories Marker `json:"additionalDirectories"`
		Fork                  Marker `json:"fork"`
	} `json:"session"`
}

type InitializeResponse struct {
	ProtocolVersion int `json:"protocolVersion"`
	// v1
	AgentCapabilities AgentCapabilities `json:"agentCapabilities"`
	AgentInfo         *Implementation   `json:"agentInfo,omitempty"`
	// v2
	Info         *Implementation     `json:"info,omitempty"`
	Capabilities AgentCapabilitiesV2 `json:"capabilities"`
}

// Features 不分版本的 Agent 能力。
type Features struct {
	Image, Audio, EmbeddedContext bool
	McpHTTP, McpSSE               bool
	AdditionalDirs                bool
	Fork, Resume, Close           bool
	Subagents                     bool
	// ToolCancel v2：可以只停一个工具调用或终端（还要看那一项自己声明了没有）。
	ToolCancel bool
}

// Agent Agent 自报的名字和版本。
func (r *InitializeResponse) Agent() *Implementation {
	if r.ProtocolVersion >= ProtocolV2 {
		return r.Info
	}
	return r.AgentInfo
}

// Features 按协商好的版本读出 Agent 的能力。
func (r *InitializeResponse) Features() Features {
	if r.ProtocolVersion < ProtocolV2 {
		c := r.AgentCapabilities
		return Features{
			Image: c.PromptCapabilities.Image, Audio: c.PromptCapabilities.Audio, EmbeddedContext: c.PromptCapabilities.EmbeddedContext,
			McpHTTP: c.McpCapabilities.HTTP, McpSSE: c.McpCapabilities.SSE,
			AdditionalDirs: c.SessionCapabilities.AdditionalDirectories != nil,
			Fork:           c.SessionCapabilities.Fork != nil,
			Resume:         c.SessionCapabilities.Resume != nil,
			Close:          c.SessionCapabilities.Close != nil,
			Subagents:      c.SessionCapabilities.Subagents != nil,
		}
	}
	s := r.Capabilities.Session
	if s == nil {
		return Features{}
	}
	f := Features{
		AdditionalDirs: s.AdditionalDirectories != nil,
		Fork:           s.Fork != nil,
		Resume:         true,
		Close:          true,
		Subagents:      true,
		ToolCancel:     true,
	}
	if p := s.Prompt; p != nil {
		f.Image, f.Audio, f.EmbeddedContext = p.Image != nil, p.Audio != nil, p.EmbeddedContext != nil
	}
	if m := s.Mcp; m != nil {
		f.McpHTTP = m.HTTP != nil
	}
	return f
}

// EnvVariable MCP 服务器的一个环境变量。
type EnvVariable struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// HTTPHeader MCP 服务器的一个请求头。
type HTTPHeader struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// McpServer 交给 Agent 的一个 MCP 服务器。Type 为空是 stdio，"http" 是 HTTP。
type McpServer struct {
	Type    string
	Name    string
	Command string
	Args    []string
	Env     []EnvVariable
	URL     string
	Headers []HTTPHeader
}

// MarshalJSON 按传输方式输出协议要求的字段；数组字段即使为空也必须出现。
func (m McpServer) MarshalJSON() ([]byte, error) {
	if m.Type == "http" || m.Type == "sse" {
		headers := m.Headers
		if headers == nil {
			headers = []HTTPHeader{}
		}
		return json.Marshal(struct {
			Type    string       `json:"type"`
			Name    string       `json:"name"`
			URL     string       `json:"url"`
			Headers []HTTPHeader `json:"headers"`
		}{m.Type, m.Name, m.URL, headers})
	}
	args, env := m.Args, m.Env
	if args == nil {
		args = []string{}
	}
	if env == nil {
		env = []EnvVariable{}
	}
	return json.Marshal(struct {
		Name    string        `json:"name"`
		Command string        `json:"command"`
		Args    []string      `json:"args"`
		Env     []EnvVariable `json:"env"`
	}{m.Name, m.Command, args, env})
}

// SessionSetup 新建、恢复、复刻会话时共用的参数（两个版本一样；v1 要求 mcpServers 一定出现）。
type SessionSetup struct {
	Cwd                   string      `json:"cwd"`
	AdditionalDirectories []string    `json:"additionalDirectories,omitempty"`
	McpServers            []McpServer `json:"mcpServers"`
	Meta                  Meta        `json:"_meta,omitempty"`
}

type NewSessionRequest struct {
	SessionSetup
}

// ExistingSessionRequest 恢复、复刻一个已有会话。v2 的恢复不带 replayFrom：Momoko 自己存着对话，不要 Agent 重放。
type ExistingSessionRequest struct {
	SessionID string `json:"sessionId"`
	SessionSetup
}

// ConfigValue 配置项的一个可选值。
type ConfigValue struct {
	Value       string `json:"value"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// ConfigOption 会话配置项（权限模式、模型、推理强度等）。客户端只能在 Agent 列出的值里选。
// v1 的 id 在 v2 里叫 configId，读进来都放在 ID。
type ConfigOption struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Category    string `json:"category,omitempty"`
	// Type 是 select 或 boolean。
	Type string `json:"type"`
	// CurrentValue select 是字符串，boolean 是 true / false。
	CurrentValue json.RawMessage `json:"currentValue"`
	// Options 可以是平铺的，也可以是分组的，用 Values 取平铺后的结果。
	Options json.RawMessage `json:"options,omitempty"`
}

// UnmarshalJSON v1 的 id 和 v2 的 configId 都认。
func (o *ConfigOption) UnmarshalJSON(b []byte) error {
	type plain ConfigOption
	var v struct {
		plain
		ConfigID string `json:"configId"`
	}
	if err := json.Unmarshal(b, &v); err != nil {
		return err
	}
	*o = ConfigOption(v.plain)
	if o.ID == "" {
		o.ID = v.ConfigID
	}
	return nil
}

// Current 返回 select 配置项当前的值。
func (o *ConfigOption) Current() string {
	var s string
	if err := json.Unmarshal(o.CurrentValue, &s); err != nil {
		return ""
	}
	return s
}

// Values 返回 select 配置项的全部可选值（分组的会被摊平）。
func (o *ConfigOption) Values() []ConfigValue {
	if len(o.Options) == 0 {
		return nil
	}
	var raw []json.RawMessage
	if err := json.Unmarshal(o.Options, &raw); err != nil {
		return nil
	}
	out := make([]ConfigValue, 0, len(raw))
	for _, item := range raw {
		var group struct {
			Options []ConfigValue `json:"options"`
		}
		if err := json.Unmarshal(item, &group); err == nil && group.Options != nil {
			out = append(out, group.Options...)
			continue
		}
		var v ConfigValue
		if err := json.Unmarshal(item, &v); err == nil && v.Value != "" {
			out = append(out, v)
		}
	}
	return out
}

// SessionState 会话建立时 Agent 给出的状态：会话 id（恢复时没有）、配置项，v2 还有斜杠命令。
type SessionState struct {
	SessionID         string             `json:"sessionId,omitempty"`
	ConfigOptions     []ConfigOption     `json:"configOptions,omitempty"`
	AvailableCommands []AvailableCommand `json:"availableCommands,omitempty"`
}

// SetConfigOptionRequest Type 是 v2 才要的（select 的值是 "id"），v1 不带。
type SetConfigOptionRequest struct {
	SessionID string `json:"sessionId"`
	ConfigID  string `json:"configId"`
	Type      string `json:"type,omitempty"`
	Value     string `json:"value"`
}

type SetConfigOptionResponse struct {
	ConfigOptions []ConfigOption `json:"configOptions"`
}

type SessionIDRequest struct {
	SessionID string `json:"sessionId"`
}

type CancelToolCallRequest struct {
	SessionID  string `json:"sessionId"`
	ToolCallID string `json:"toolCallId"`
}

type CancelTerminalRequest struct {
	SessionID  string `json:"sessionId"`
	TerminalID string `json:"terminalId"`
}

// ResourceContents 嵌入到消息里的资源内容：文本或 base64 的二进制。
type ResourceContents struct {
	URI      string `json:"uri"`
	MimeType string `json:"mimeType,omitempty"`
	Text     string `json:"text,omitempty"`
	Blob     string `json:"blob,omitempty"`
}

// ContentBlock 消息里的一块内容。Type：text / image / audio / resource_link / resource。
type ContentBlock struct {
	Type        string            `json:"type"`
	Text        string            `json:"text,omitempty"`
	Data        string            `json:"data,omitempty"`
	MimeType    string            `json:"mimeType,omitempty"`
	URI         string            `json:"uri,omitempty"`
	Name        string            `json:"name,omitempty"`
	Title       string            `json:"title,omitempty"`
	Description string            `json:"description,omitempty"`
	Size        *int64            `json:"size,omitempty"`
	Resource    *ResourceContents `json:"resource,omitempty"`
}

// TextBlock 生成一块文本内容。
func TextBlock(text string) ContentBlock {
	return ContentBlock{Type: "text", Text: text}
}

type PromptRequest struct {
	SessionID string         `json:"sessionId"`
	Prompt    []ContentBlock `json:"prompt"`
}

// Usage 一轮的 token 用量。InputTokens 不含缓存命中和缓存写入。
type Usage struct {
	TotalTokens       int64  `json:"totalTokens"`
	InputTokens       int64  `json:"inputTokens"`
	OutputTokens      int64  `json:"outputTokens"`
	ThoughtTokens     *int64 `json:"thoughtTokens,omitempty"`
	CachedReadTokens  *int64 `json:"cachedReadTokens,omitempty"`
	CachedWriteTokens *int64 `json:"cachedWriteTokens,omitempty"`
}

// PromptResponse v1：这一轮结束（StopReason、Usage）；v2：消息已经收下（MessageID），一轮的结束看 state_update。
type PromptResponse struct {
	StopReason string `json:"stopReason"`
	// Usage 还是草案里的字段，Agent 不一定给。
	Usage     *Usage `json:"usage,omitempty"`
	MessageID string `json:"messageId,omitempty"`
	Meta      Meta   `json:"_meta,omitempty"`
}

// DiffChange v2 的一个文件级改动：add / delete / modify 带 Path，move / copy 还带 OldPath。
type DiffChange struct {
	Operation string `json:"operation"`
	Path      string `json:"path"`
	OldPath   string `json:"oldPath,omitempty"`
	FileType  string `json:"fileType,omitempty"`
}

// DiffPatch v2 diff 附带的可渲染文本；git_patch 是绝对路径、不带 a/ b/ 前缀的 diff --git 片段。
type DiffPatch struct {
	Format string `json:"format"`
	Text   string `json:"text"`
}

// ToolCallContent 工具调用产生的内容。Type：content / diff / terminal / session。
type ToolCallContent struct {
	Type string `json:"type"`
	// content
	Content *ContentBlock `json:"content,omitempty"`
	// diff：v1 是 Path + OldText / NewText；v2 是 Changes + Patch
	Path    string       `json:"path,omitempty"`
	OldText *string      `json:"oldText,omitempty"`
	NewText string       `json:"newText,omitempty"`
	Changes []DiffChange `json:"changes,omitempty"`
	Patch   *DiffPatch   `json:"patch,omitempty"`
	// terminal：v2 里是 Agent 自己的显示终端，内容由 terminal_update / terminal_output_chunk 报
	TerminalID string `json:"terminalId,omitempty"`
	// session：子 Agent 的会话
	SessionID string `json:"sessionId,omitempty"`
}

// ToolCallLocation 工具调用涉及的文件位置。
type ToolCallLocation struct {
	Path string `json:"path"`
	Line *int   `json:"line,omitempty"`
}

// PlanEntry 计划里的一步。
type PlanEntry struct {
	Content  string `json:"content"`
	Priority string `json:"priority,omitempty"`
	// Status：pending / in_progress / completed（v2 还有 cancelled）。
	Status string `json:"status"`
}

// PlanContent v2 的计划：按 PlanID 整份替换，稳定版只有 items 一种。
type PlanContent struct {
	Type    string      `json:"type"`
	PlanID  string      `json:"planId"`
	Entries []PlanEntry `json:"entries,omitempty"`
}

// AvailableCommand Agent 自己的一个斜杠命令（v2 多一个 type，值是 command）。
type AvailableCommand struct {
	Type        string `json:"type,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Input       *struct {
		Hint string `json:"hint"`
	} `json:"input,omitempty"`
}

// Cost 这次会话累计的花费。
type Cost struct {
	Amount   float64 `json:"amount"`
	Currency string  `json:"currency"`
}

// Cancelable v2：这一项（工具调用、终端、子会话）能不能单独停。
type Cancelable struct {
	Cancel Marker `json:"cancel"`
}

// BackgroundWork v2 state_update 里还在后台跑的一项：terminal / tool_call / subagent。
type BackgroundWork struct {
	Type       string `json:"type"`
	TerminalID string `json:"terminalId,omitempty"`
	ToolCallID string `json:"toolCallId,omitempty"`
	SessionID  string `json:"sessionId,omitempty"`
}

// TerminalOutput v2 终端输出的整份快照（base64）。
type TerminalOutput struct {
	Data string `json:"data"`
}

// TerminalExitStatus v2 终端退出的情况。
type TerminalExitStatus struct {
	ExitCode *int64  `json:"exitCode,omitempty"`
	Signal   *string `json:"signal,omitempty"`
}

// SessionUpdate 一条会话更新。Kind 决定哪些字段有值；同一个 content 键在消息块里是一块内容，
// 在整条消息、工具调用里是数组，所以先原样留着，用 Chunk / Blocks / ToolContent 取。
type SessionUpdate struct {
	Kind string `json:"sessionUpdate"`

	Content   json.RawMessage `json:"content,omitempty"`
	MessageID string          `json:"messageId,omitempty"`

	// 工具调用
	ToolCallID string             `json:"toolCallId,omitempty"`
	Title      *string            `json:"title,omitempty"`
	ToolKind   string             `json:"kind,omitempty"`
	Status     string             `json:"status,omitempty"`
	Locations  []ToolCallLocation `json:"locations,omitempty"`
	RawInput   json.RawMessage    `json:"rawInput,omitempty"`
	RawOutput  json.RawMessage    `json:"rawOutput,omitempty"`
	// Capabilities v2：工具调用、终端、子会话能不能单独停
	Capabilities *Cancelable `json:"capabilities,omitempty"`

	// 计划：v1 是 Entries，v2 是 Plan
	Entries []PlanEntry  `json:"entries,omitempty"`
	Plan    *PlanContent `json:"plan,omitempty"`

	AvailableCommands []AvailableCommand `json:"availableCommands,omitempty"`
	ConfigOptions     []ConfigOption     `json:"configOptions,omitempty"`

	// 上下文用量
	Used *int64 `json:"used,omitempty"`
	Size *int64 `json:"size,omitempty"`
	Cost *Cost  `json:"cost,omitempty"`

	// 提示（notice）
	Severity    string `json:"severity,omitempty"`
	Description string `json:"description,omitempty"`

	// 子 Agent：它自己的会话 id 和状态。v1 草案里 state 是 {state}，v2 state_update 里是字符串，subagent_update 里是 {state, ...}
	SessionID string          `json:"sessionId,omitempty"`
	State     json.RawMessage `json:"state,omitempty"`

	// v2 state_update
	StopReason string           `json:"stopReason,omitempty"`
	Error      *Error           `json:"error,omitempty"`
	Usage      *Usage           `json:"usage,omitempty"`
	Background []BackgroundWork `json:"background,omitempty"`

	// v2 终端；Cwd 也是 v2 子会话的工作目录
	TerminalID string              `json:"terminalId,omitempty"`
	Command    *string             `json:"command,omitempty"`
	Cwd        *string             `json:"cwd,omitempty"`
	Output     *TerminalOutput     `json:"output,omitempty"`
	ExitStatus *TerminalExitStatus `json:"exitStatus,omitempty"`
	Data       string              `json:"data,omitempty"`

	// v2 会话之间的消息
	SenderSessionID    *string `json:"senderSessionId,omitempty"`
	RecipientSessionID *string `json:"recipientSessionId,omitempty"`

	Meta Meta `json:"_meta,omitempty"`
}

// Chunk 取消息块 / 思考块里的那一块内容。
func (u *SessionUpdate) Chunk() *ContentBlock {
	if len(u.Content) == 0 {
		return nil
	}
	block := &ContentBlock{}
	if err := json.Unmarshal(u.Content, block); err != nil {
		return nil
	}
	return block
}

// Blocks 取 v2 整条消息的内容；没带 content 时 ok 为 false（保持原样），null 或 [] 是清空。
func (u *SessionUpdate) Blocks() (blocks []ContentBlock, ok bool) {
	if len(u.Content) == 0 {
		return nil, false
	}
	if string(u.Content) == "null" {
		return nil, true
	}
	if err := json.Unmarshal(u.Content, &blocks); err != nil {
		return nil, false
	}
	return blocks, true
}

// ToolContent 取工具调用的内容数组；这次更新没带 content 时 ok 为 false（表示保持原样）。
func (u *SessionUpdate) ToolContent() (content []ToolCallContent, ok bool) {
	if len(u.Content) == 0 || string(u.Content) == "null" {
		return nil, false
	}
	if err := json.Unmarshal(u.Content, &content); err != nil {
		return nil, false
	}
	return content, true
}

// ToolContentItem 取 v2 tool_call_content_chunk 里追加的那一项。
func (u *SessionUpdate) ToolContentItem() (ToolCallContent, bool) {
	var c ToolCallContent
	if len(u.Content) == 0 || json.Unmarshal(u.Content, &c) != nil {
		return c, false
	}
	return c, true
}

// StateName 取状态：v2 state_update 里 state 是字符串，subagent_update（v1 草案和 v2）里是带 state 的对象。读不出来为空。
func (u *SessionUpdate) StateName() string {
	if len(u.State) == 0 {
		return ""
	}
	var name string
	if json.Unmarshal(u.State, &name) == nil {
		return name
	}
	var s struct {
		State string `json:"state"`
	}
	if json.Unmarshal(u.State, &s) != nil {
		return ""
	}
	return s.State
}

// CanCancel v2：这一项声明了可以单独停。
func (u *SessionUpdate) CanCancel() (cancel, ok bool) {
	if u.Capabilities == nil {
		return false, false
	}
	return u.Capabilities.Cancel != nil, true
}

// SessionNotification Agent 发来的 session/update。
type SessionNotification struct {
	SessionID string        `json:"sessionId"`
	Update    SessionUpdate `json:"update"`
	Meta      Meta          `json:"_meta,omitempty"`
}

// PermissionOption 批准卡片上的一个选项。
type PermissionOption struct {
	OptionID string `json:"optionId"`
	Name     string `json:"name"`
	Kind     string `json:"kind"`
}

// PermissionToolCall 要批准的那次工具调用。
type PermissionToolCall struct {
	ToolCallID string             `json:"toolCallId"`
	Title      *string            `json:"title,omitempty"`
	Kind       string             `json:"kind,omitempty"`
	Status     string             `json:"status,omitempty"`
	Content    []ToolCallContent  `json:"content,omitempty"`
	Locations  []ToolCallLocation `json:"locations,omitempty"`
	RawInput   json.RawMessage    `json:"rawInput,omitempty"`
}

// PermissionSubject v2 批准的对象：tool_call 带工具调用，command 带命令和目录。
type PermissionSubject struct {
	Type       string              `json:"type"`
	ToolCall   *PermissionToolCall `json:"toolCall,omitempty"`
	Command    string              `json:"command,omitempty"`
	Cwd        string              `json:"cwd,omitempty"`
	ToolCallID string              `json:"toolCallId,omitempty"`
	TerminalID string              `json:"terminalId,omitempty"`
}

// RequestPermissionRequest v1 带 ToolCall；v2 带自己的 Title / Description 和 Subject。
type RequestPermissionRequest struct {
	SessionID   string             `json:"sessionId"`
	ToolCall    PermissionToolCall `json:"toolCall"`
	Title       string             `json:"title,omitempty"`
	Description string             `json:"description,omitempty"`
	Subject     *PermissionSubject `json:"subject,omitempty"`
	Options     []PermissionOption `json:"options"`
}

// Tool 这张批准关于哪次工具调用：v1 的 toolCall，v2 的 tool_call 对象，或 command 对象里带的工具调用 id。
func (r *RequestPermissionRequest) Tool() PermissionToolCall {
	if r.Subject == nil {
		return r.ToolCall
	}
	if r.Subject.ToolCall != nil {
		return *r.Subject.ToolCall
	}
	return PermissionToolCall{ToolCallID: r.Subject.ToolCallID}
}

// PermissionOutcome 批准的结果：selected 带选中的选项，cancelled 表示这一轮被停止了。
type PermissionOutcome struct {
	Outcome  string `json:"outcome"`
	OptionID string `json:"optionId,omitempty"`
}

type RequestPermissionResponse struct {
	Outcome PermissionOutcome `json:"outcome"`
}

// PermissionSelected 选中某个选项。
func PermissionSelected(optionID string) *RequestPermissionResponse {
	return &RequestPermissionResponse{Outcome: PermissionOutcome{Outcome: "selected", OptionID: optionID}}
}

// PermissionCancelled 这一轮被停止，批准不再需要。
func PermissionCancelled() *RequestPermissionResponse {
	return &RequestPermissionResponse{Outcome: PermissionOutcome{Outcome: "cancelled"}}
}

// CancelRequestParams $/cancel_request 的参数：要撤回的请求 id（数字或字符串）。
type CancelRequestParams struct {
	RequestID json.RawMessage `json:"requestId"`
}

// AuthStatus _auth/status_update 报上来的登录身份。Kind：account / api_key / gateway / external / none。
type AuthStatus struct {
	Kind   string `json:"kind"`
	Label  string `json:"label"`
	Detail string `json:"detail,omitempty"`
	// Account 账号的邮箱、组织、套餐，Agent 知道才有。
	Account *struct {
		Email        string `json:"email,omitempty"`
		Organization string `json:"organization,omitempty"`
		Plan         string `json:"plan,omitempty"`
	} `json:"account,omitempty"`
}

// LoggedIn 有登录身份（none 是确知没登录）。
func (a AuthStatus) LoggedIn() bool { return a.Kind != "" && a.Kind != "none" }

// Description 给人看的第二行：Agent 给的说明，没有时用账号邮箱。
func (a AuthStatus) Description() string {
	if a.Detail != "" {
		return a.Detail
	}
	if a.Account != nil {
		return a.Account.Email
	}
	return ""
}

// AuthStatusNotification _auth/status_update 的参数。
type AuthStatusNotification struct {
	AuthStatus AuthStatus `json:"authStatus"`
}
