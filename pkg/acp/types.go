package acp

import (
	"encoding/json"
	"errors"

	"momoko/pkg/jsonrpc"
)

// ProtocolVersion 是 Momoko 说的 ACP 协议版本。
const ProtocolVersion = 1

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
	MethodInitialize         = "initialize"
	MethodSessionNew         = "session/new"
	MethodSessionResume      = "session/resume"
	MethodSessionFork        = "session/fork"
	MethodSessionClose       = "session/close"
	MethodSessionPrompt      = "session/prompt"
	MethodSessionCancel      = "session/cancel"
	MethodSessionSetConfig   = "session/set_config_option"
	MethodSessionUpdate      = "session/update"
	MethodRequestPermission  = "session/request_permission"
	MethodAuthStatusUpdate   = "auth/status_update"
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
)

// Meta 是协议里随处可带的 _meta 扩展字段。
type Meta map[string]json.RawMessage

// Implementation 一端的名称和版本。
type Implementation struct {
	Name    string `json:"name"`
	Title   string `json:"title,omitempty"`
	Version string `json:"version"`
}

// ClientCapabilities 客户端声明的能力。Momoko 不代 Agent 读写文件，也不提供终端，所以这两项都不声明。
type ClientCapabilities struct {
	// Subagents 不为空时，Agent 把子 Agent 作为独立的子会话上报。
	Subagents *struct{} `json:"subagents,omitempty"`
	Meta      Meta      `json:"_meta,omitempty"`
}

type InitializeRequest struct {
	ProtocolVersion    int                `json:"protocolVersion"`
	ClientCapabilities ClientCapabilities `json:"clientCapabilities"`
	ClientInfo         *Implementation    `json:"clientInfo,omitempty"`
}

// PromptCapabilities Agent 能接受的消息内容。
type PromptCapabilities struct {
	Image           bool `json:"image"`
	Audio           bool `json:"audio"`
	EmbeddedContext bool `json:"embeddedContext"`
}

// McpCapabilities Agent 支持的 MCP 传输方式；stdio 是必须支持的，不在这里。
type McpCapabilities struct {
	HTTP bool `json:"http"`
	SSE  bool `json:"sse"`
}

// SessionCapabilities 需要声明才能用的会话能力：字段出现就表示支持。
type SessionCapabilities struct {
	AdditionalDirectories *json.RawMessage `json:"additionalDirectories,omitempty"`
	Fork                  *json.RawMessage `json:"fork,omitempty"`
	Resume                *json.RawMessage `json:"resume,omitempty"`
	Close                 *json.RawMessage `json:"close,omitempty"`
	Subagents             *json.RawMessage `json:"subagents,omitempty"`
}

type AgentCapabilities struct {
	PromptCapabilities  PromptCapabilities  `json:"promptCapabilities"`
	McpCapabilities     McpCapabilities     `json:"mcpCapabilities"`
	SessionCapabilities SessionCapabilities `json:"sessionCapabilities"`
	Meta                Meta                `json:"_meta,omitempty"`
}

// AuthMethod Agent 提供的登录方式。
type AuthMethod struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Type        string `json:"type,omitempty"`
}

type InitializeResponse struct {
	ProtocolVersion   int               `json:"protocolVersion"`
	AgentCapabilities AgentCapabilities `json:"agentCapabilities"`
	AgentInfo         *Implementation   `json:"agentInfo,omitempty"`
	AuthMethods       []AuthMethod      `json:"authMethods,omitempty"`
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

// SessionSetup 新建、加载、恢复、复刻会话时共用的参数。
type SessionSetup struct {
	Cwd                   string      `json:"cwd"`
	AdditionalDirectories []string    `json:"additionalDirectories,omitempty"`
	McpServers            []McpServer `json:"mcpServers"`
	Meta                  Meta        `json:"_meta,omitempty"`
}

type NewSessionRequest struct {
	SessionSetup
}

// ExistingSessionRequest 加载、恢复、复刻一个已有会话。
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
			Group   string        `json:"group"`
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

// SessionState 会话建立时 Agent 给出的状态：会话 id（恢复时没有）和配置项（权限模式、模型、推理强度）。
type SessionState struct {
	SessionID     string         `json:"sessionId,omitempty"`
	ConfigOptions []ConfigOption `json:"configOptions,omitempty"`
}

type SetConfigOptionRequest struct {
	SessionID string `json:"sessionId"`
	ConfigID  string `json:"configId"`
	Value     string `json:"value"`
}

type SetConfigOptionResponse struct {
	ConfigOptions []ConfigOption `json:"configOptions"`
}

type SessionIDRequest struct {
	SessionID string `json:"sessionId"`
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

type PromptResponse struct {
	StopReason string `json:"stopReason"`
	// Usage 还是草案里的字段，Agent 不一定给。
	Usage *Usage `json:"usage,omitempty"`
	Meta  Meta   `json:"_meta,omitempty"`
}

// ToolCallContent 工具调用产生的内容。Type：content / diff / terminal / session。
type ToolCallContent struct {
	Type string `json:"type"`
	// content
	Content *ContentBlock `json:"content,omitempty"`
	// diff
	Path    string  `json:"path,omitempty"`
	OldText *string `json:"oldText,omitempty"`
	NewText string  `json:"newText,omitempty"`
	// terminal
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
	// Status：pending / in_progress / completed。
	Status string `json:"status"`
}

// AvailableCommand Agent 自己的一个斜杠命令。
type AvailableCommand struct {
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

// SessionUpdate 一条会话更新。Kind 决定哪些字段有值；同一个 content 键在消息块里是一块内容，
// 在工具调用里是内容数组，所以先原样留着，用 Chunk / ToolContent 取。
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

	// 计划
	Entries []PlanEntry `json:"entries,omitempty"`

	AvailableCommands []AvailableCommand `json:"availableCommands,omitempty"`
	ConfigOptions     []ConfigOption     `json:"configOptions,omitempty"`

	// 上下文用量
	Used *int64 `json:"used,omitempty"`
	Size *int64 `json:"size,omitempty"`
	Cost *Cost  `json:"cost,omitempty"`

	// 提示（notice）
	Severity    string `json:"severity,omitempty"`
	Description string `json:"description,omitempty"`

	// 子 Agent：它自己的会话 id 和状态
	SessionID string          `json:"sessionId,omitempty"`
	State     json.RawMessage `json:"state,omitempty"`

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

// SubagentState 取子 Agent 的状态：running / idle / requires_action，读不出来为空。
func (u *SessionUpdate) SubagentState() string {
	if len(u.State) == 0 {
		return ""
	}
	var s struct {
		State string `json:"state"`
	}
	if err := json.Unmarshal(u.State, &s); err != nil {
		return ""
	}
	return s.State
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

type RequestPermissionRequest struct {
	SessionID string             `json:"sessionId"`
	ToolCall  PermissionToolCall `json:"toolCall"`
	Options   []PermissionOption `json:"options"`
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

// AuthAccount 登录状态里的一个账号。
type AuthAccount struct {
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
	Email       string `json:"email,omitempty"`
	Active      bool   `json:"active,omitempty"`
}

// AuthState auth/status_update 报上来的登录状态。State：authenticated / unauthenticated / unknown。
type AuthState struct {
	State    string        `json:"state"`
	Accounts []AuthAccount `json:"accounts,omitempty"`
}

// Account 当前在用的账号；没登录时 ok 为 false。
func (a AuthState) Account() (AuthAccount, bool) {
	if a.State != "authenticated" {
		return AuthAccount{}, false
	}
	for _, acc := range a.Accounts {
		if acc.Active {
			return acc, true
		}
	}
	if len(a.Accounts) > 0 {
		return a.Accounts[0], true
	}
	return AuthAccount{}, false
}
