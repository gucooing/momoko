// Package acp 是 ACP（Agent Client Protocol）客户端一侧的协议：消息类型，和跑在 Agent 进程 stdin / stdout 上的客户端。
//
// v1 和 v2 都说：握手时请求 v2，Agent 选了哪个版本，之后的请求和回复就按哪个版本的形状收发；
// 两个版本的会话更新都交给同一个 Events，由调用方按 Version 分辨语义（v2 里一轮的结束看 state_update）。
// 只实现 Momoko 用到的部分：初始化、会话的新建 / 恢复 / 复刻 / 关闭、发消息、停止、设置配置项、单独停一个工具调用或终端，
// 以及 Agent 发回来的会话更新、批准请求（和它的撤回）和登录身份。某个 Agent 自己的扩展字段由 agentrun 里对应的驱动处理。
package acp

import (
	"context"
	"encoding/json"
	"io"
	"strings"

	"momoko/pkg/jsonrpc"
)

// Events 接收 Agent 主动发来的东西。方法按消息到达的顺序、在同一个协程里依次调用，实现里不能阻塞。
type Events interface {
	// SessionUpdate 一条会话更新。
	SessionUpdate(n *SessionNotification)
	// RequestPermission Agent 要做一件需要批准的事。id 是这个请求的 id，Agent 撤回它时凭它认。
	// reply 可以在返回之后、有人回答时再调用，只能调用一次。
	RequestPermission(id string, req *RequestPermissionRequest, reply func(*RequestPermissionResponse))
	// RequestCancelled Agent 撤回了它发来的某个请求（v2：这一轮停了，那张批准不再需要）。
	RequestCancelled(id string)
	// AuthStatus Agent 上报的登录身份。
	AuthStatus(status AuthStatus)
}

// Client 是 ACP 的客户端一侧。
type Client struct {
	conn    *jsonrpc.Conn
	events  Events
	version int
}

// NewClient 在 Agent 进程的输出 r 和输入 w 上建立客户端。
func NewClient(r io.Reader, w io.Writer, events Events) *Client {
	c := &Client{events: events, version: ProtocolV1}
	c.conn = jsonrpc.NewConn(r, w, (*clientHandler)(c))
	return c
}

// Done 在连接关闭（Agent 进程退出）后关闭。
func (c *Client) Done() <-chan struct{} { return c.conn.Done() }

// Close 关闭连接。
func (c *Client) Close() { c.conn.Close() }

// Version 握手后协商好的协议版本。
func (c *Client) Version() int { return c.version }

// V2 协商好的是 v2。
func (c *Client) V2() bool { return c.version >= ProtocolV2 }

// Initialize 协商协议版本和能力，必须是第一个调用。v1 的客户端能力只给只会 v1 的 Agent 看。
func (c *Client) Initialize(ctx context.Context, info Implementation, caps ClientCapabilities) (*InitializeResponse, error) {
	resp := &InitializeResponse{}
	err := c.conn.Call(ctx, MethodInitialize, &InitializeRequest{
		ProtocolVersion:    ProtocolVersion,
		Info:               info,
		ClientInfo:         &info,
		ClientCapabilities: caps,
	}, resp)
	if err != nil {
		return nil, err
	}
	if resp.ProtocolVersion != ProtocolV1 && resp.ProtocolVersion != ProtocolV2 {
		return nil, &Error{Code: jsonrpc.CodeInvalidRequest, Message: "Agent 说的 ACP 协议版本 Momoko 不支持"}
	}
	c.version = resp.ProtocolVersion
	return resp, nil
}

// NewSession 新建会话。
func (c *Client) NewSession(ctx context.Context, setup SessionSetup) (*SessionState, error) {
	return c.sessionCall(ctx, MethodSessionNew, &NewSessionRequest{SessionSetup: normalizeSetup(setup)})
}

// ResumeSession 恢复已有会话的上下文，不重放历史记录。
func (c *Client) ResumeSession(ctx context.Context, sessionID string, setup SessionSetup) (*SessionState, error) {
	return c.sessionCall(ctx, MethodSessionResume, &ExistingSessionRequest{SessionID: sessionID, SessionSetup: normalizeSetup(setup)})
}

// ForkSession 复刻会话，返回新会话；setup.Cwd 可以和原会话不同。
func (c *Client) ForkSession(ctx context.Context, sessionID string, setup SessionSetup) (*SessionState, error) {
	return c.sessionCall(ctx, MethodSessionFork, &ExistingSessionRequest{SessionID: sessionID, SessionSetup: normalizeSetup(setup)})
}

func (c *Client) sessionCall(ctx context.Context, method string, req any) (*SessionState, error) {
	state := &SessionState{}
	if err := c.conn.Call(ctx, method, req, state); err != nil {
		return nil, err
	}
	return state, nil
}

func normalizeSetup(setup SessionSetup) SessionSetup {
	if setup.McpServers == nil {
		setup.McpServers = []McpServer{}
	}
	return setup
}

// CloseSession 让 Agent 释放这个会话占用的资源，会话记录保留。
func (c *Client) CloseSession(ctx context.Context, sessionID string) error {
	return c.conn.Call(ctx, MethodSessionClose, &SessionIDRequest{SessionID: sessionID}, nil)
}

// Prompt 发一条消息。v1：等这一轮结束才返回，回复里是结束原因；v2：消息收下就返回（回复里是消息 id），
// 一轮的进展和结束都从 state_update 来。过程中的内容通过 Events.SessionUpdate 到达。
func (c *Client) Prompt(ctx context.Context, sessionID string, blocks []ContentBlock) (*PromptResponse, error) {
	resp := &PromptResponse{}
	if err := c.conn.Call(ctx, MethodSessionPrompt, &PromptRequest{SessionID: sessionID, Prompt: blocks}, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Cancel 停止当前在做的事。是通知，没有回复；v1 里这一轮的 Prompt 以 cancelled 结束，v2 里 Agent 报一个 cancelled 的 idle。
func (c *Client) Cancel(sessionID string) error {
	return c.conn.Notify(MethodSessionCancel, &SessionIDRequest{SessionID: sessionID})
}

// CancelToolCall v2：只停一个工具调用（它声明了 capabilities.cancel 时）。
func (c *Client) CancelToolCall(ctx context.Context, sessionID, toolCallID string) error {
	return c.conn.Call(ctx, MethodToolCallCancel, &CancelToolCallRequest{SessionID: sessionID, ToolCallID: toolCallID}, nil)
}

// CancelTerminal v2：只停一个终端里的进程（它声明了 capabilities.cancel 时）。
func (c *Client) CancelTerminal(ctx context.Context, sessionID, terminalID string) error {
	return c.conn.Call(ctx, MethodTerminalCancel, &CancelTerminalRequest{SessionID: sessionID, TerminalID: terminalID}, nil)
}

// SetConfigOption 设置一个配置项，返回更新后的全部配置项（改模型可能连带改变能选的推理强度）。
func (c *Client) SetConfigOption(ctx context.Context, sessionID, configID, value string) ([]ConfigOption, error) {
	req := &SetConfigOptionRequest{SessionID: sessionID, ConfigID: configID, Value: value}
	if c.V2() {
		req.Type = "id"
	}
	resp := &SetConfigOptionResponse{}
	if err := c.conn.Call(ctx, MethodSessionSetConfig, req, resp); err != nil {
		return nil, err
	}
	return resp.ConfigOptions, nil
}

// clientHandler 把连接上收到的通知和请求分给 Events。
type clientHandler Client

func (h *clientHandler) Notification(method string, params json.RawMessage) {
	if h.events == nil {
		return
	}
	switch method {
	case MethodSessionUpdate:
		n := &SessionNotification{}
		if err := json.Unmarshal(params, n); err != nil {
			return
		}
		h.events.SessionUpdate(n)
	case MethodCancelRequest:
		var p CancelRequestParams
		if err := json.Unmarshal(params, &p); err != nil || len(p.RequestID) == 0 {
			return
		}
		h.events.RequestCancelled(strings.Trim(string(p.RequestID), `"`))
	case MethodAuthStatusUpdate:
		var n AuthStatusNotification
		if err := json.Unmarshal(params, &n); err != nil {
			return
		}
		h.events.AuthStatus(n.AuthStatus)
	}
}

func (h *clientHandler) Request(id, method string, params json.RawMessage, reply func(result any, err *jsonrpc.Error)) {
	switch method {
	case MethodRequestPermission:
		req := &RequestPermissionRequest{}
		if err := json.Unmarshal(params, req); err != nil {
			reply(nil, jsonrpc.NewError(jsonrpc.CodeInvalidParams, err.Error()))
			return
		}
		if h.events == nil {
			reply(PermissionCancelled(), nil)
			return
		}
		h.events.RequestPermission(id, req, func(resp *RequestPermissionResponse) {
			if resp == nil {
				resp = PermissionCancelled()
			}
			reply(resp, nil)
		})
	default:
		// 没声明过的能力（读写文件、终端、表单询问）Agent 不该来调用。
		reply(nil, jsonrpc.NewError(jsonrpc.CodeMethodNotFound, "method not found: "+method))
	}
}
