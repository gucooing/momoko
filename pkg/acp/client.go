// Package acp 是 ACP（Agent Client Protocol）客户端一侧的协议：消息类型，和跑在 Agent 进程 stdin / stdout 上的客户端。
//
// 只实现 Momoko 用到的部分：初始化、会话的新建 / 恢复 / 复刻 / 关闭、发消息、停止、设置配置项，
// 以及 Agent 发回来的会话更新、批准请求和登录状态。某个 Agent 自己的扩展字段由 agentrun 里对应的驱动处理。
package acp

import (
	"context"
	"encoding/json"
	"io"

	"momoko/pkg/jsonrpc"
)

// Events 接收 Agent 主动发来的东西。方法按消息到达的顺序、在同一个协程里依次调用，实现里不能阻塞。
type Events interface {
	// SessionUpdate 一条会话更新。
	SessionUpdate(n *SessionNotification)
	// RequestPermission Agent 要做一件需要批准的事。reply 可以在返回之后、有人回答时再调用，只能调用一次。
	RequestPermission(req *RequestPermissionRequest, reply func(*RequestPermissionResponse))
	// AuthStatus Agent 上报的登录状态。
	AuthStatus(state AuthState)
}

// Client 是 ACP 的客户端一侧。
type Client struct {
	conn   *jsonrpc.Conn
	events Events
}

// NewClient 在 Agent 进程的输出 r 和输入 w 上建立客户端。
func NewClient(r io.Reader, w io.Writer, events Events) *Client {
	c := &Client{events: events}
	c.conn = jsonrpc.NewConn(r, w, (*clientHandler)(c))
	return c
}

// Done 在连接关闭（Agent 进程退出）后关闭。
func (c *Client) Done() <-chan struct{} { return c.conn.Done() }

// Close 关闭连接。
func (c *Client) Close() { c.conn.Close() }

// Initialize 协商协议版本和能力，必须是第一个调用。
func (c *Client) Initialize(ctx context.Context, info Implementation, caps ClientCapabilities) (*InitializeResponse, error) {
	resp := &InitializeResponse{}
	err := c.conn.Call(ctx, MethodInitialize, &InitializeRequest{
		ProtocolVersion:    ProtocolVersion,
		ClientCapabilities: caps,
		ClientInfo:         &info,
	}, resp)
	if err != nil {
		return nil, err
	}
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

// Prompt 发一条消息并等这一轮结束；过程中的内容通过 Events.SessionUpdate 到达。
func (c *Client) Prompt(ctx context.Context, sessionID string, blocks []ContentBlock) (*PromptResponse, error) {
	resp := &PromptResponse{}
	if err := c.conn.Call(ctx, MethodSessionPrompt, &PromptRequest{SessionID: sessionID, Prompt: blocks}, resp); err != nil {
		return nil, err
	}
	return resp, nil
}

// Cancel 停止当前这一轮。是通知，没有回复；这一轮的 Prompt 会以 cancelled 结束。
func (c *Client) Cancel(sessionID string) error {
	return c.conn.Notify(MethodSessionCancel, &SessionIDRequest{SessionID: sessionID})
}

// SetConfigOption 设置一个配置项，返回更新后的全部配置项（改模型可能连带改变能选的推理强度）。
func (c *Client) SetConfigOption(ctx context.Context, sessionID, configID, value string) ([]ConfigOption, error) {
	resp := &SetConfigOptionResponse{}
	err := c.conn.Call(ctx, MethodSessionSetConfig, &SetConfigOptionRequest{SessionID: sessionID, ConfigID: configID, Value: value}, resp)
	if err != nil {
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
	case MethodAuthStatusUpdate:
		var state AuthState
		if err := json.Unmarshal(params, &state); err != nil {
			return
		}
		h.events.AuthStatus(state)
	}
}

func (h *clientHandler) Request(method string, params json.RawMessage, reply func(result any, err *jsonrpc.Error)) {
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
		h.events.RequestPermission(req, func(resp *RequestPermissionResponse) {
			if resp == nil {
				resp = PermissionCancelled()
			}
			reply(resp, nil)
		})
	default:
		// 没声明过的能力（读写文件、终端）Agent 不该来调用。
		reply(nil, jsonrpc.NewError(jsonrpc.CodeMethodNotFound, "method not found: "+method))
	}
}
