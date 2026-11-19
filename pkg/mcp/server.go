// Package mcp 是 Momoko 对外提供的 MCP 服务器：让 Agent 之类的 MCP 客户端回过头来操作 Momoko。
//
// 协议走 Streamable HTTP 里一问一答的 JSON 响应，只提供工具，消息格式用 pkg/jsonrpc。
// 工具的定义和分发也在这里；工具真正要做的事由外面注入（见 Agent），
// 凭证能不能调用某个工具由外面的 Guard 决定（每个工具标明它要的权限）。
package mcp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"momoko/pkg/constant"
	"momoko/pkg/jsonrpc"
	"momoko/pkg/response"
)

// Path MCP 服务的地址。请求头带 Authorization: Bearer <API Key>，凭证由 HTTP 鉴权中间件认。
const Path = "/api/v1/mcp"

// callTimeout 一次工具调用最多跑多久，不跟着 HTTP 请求的超时走（工具可能要克隆仓库、下载文件）。
const callTimeout = 10 * time.Minute

// Guard 检查这次请求的凭证能不能调用要 perm 的工具。返回的错误原样作为 HTTP 错误回给客户端。
type Guard interface {
	Authorize(ctx context.Context, perm constant.Permissions) error
}

// Tool 一个工具。
type Tool struct {
	Name        string
	Description string
	// InputSchema 参数的 JSON Schema。
	InputSchema map[string]any
	// Permission 调用它要的权限。
	Permission constant.Permissions
	// call 返回给客户端看的文字；返回错误时这次调用按失败上报，错误文字同样给客户端看。
	call func(ctx context.Context, args json.RawMessage) (string, error)
}

// Server Momoko 的 MCP 服务。建好之后只读，可以给多个请求同时用。
type Server struct {
	guard Guard
	tools []Tool
}

// New 用注入的实现建 MCP 服务。
func New(agent Agent, guard Guard) *Server {
	return &Server{guard: guard, tools: agentTools(agent)}
}

// Tools 全部工具，按 tools/list 的顺序。
func (s *Server) Tools() []Tool {
	return s.tools
}

const instructions = "Tools for changing Momoko's own Agent configuration: install a skill, or register an MCP server. " +
	"Read the source's README or install notes first, then call the tool with the exact values."

func textResult(text string, isError bool) map[string]any {
	out := map[string]any{"content": []map[string]any{{"type": "text", "text": text}}}
	if isError {
		out["isError"] = true
	}
	return out
}

// ServeHTTP 处理一条 JSON-RPC 消息。只收 POST：协议允许不开 GET 的事件流，按规定回 405。
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		response.WriteError(w, r, response.BadRequest("读不到请求内容"))
		return
	}
	req := jsonrpc.Parse(raw)
	if req == nil || req.Method == "" {
		response.WriteError(w, r, response.BadRequest("不是一条 JSON-RPC 请求"))
		return
	}
	// 通知没有 id，不用回内容。
	if !req.HasID() {
		w.WriteHeader(http.StatusAccepted)
		return
	}
	var (
		result any
		rpcErr *jsonrpc.Error
	)
	switch req.Method {
	case "initialize":
		var params struct {
			ProtocolVersion string `json:"protocolVersion"`
		}
		_ = json.Unmarshal(req.Params, &params)
		result = map[string]any{
			"protocolVersion": params.ProtocolVersion,
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "momoko", "title": "Momoko", "version": "1"},
			"instructions":    instructions,
		}
	case "ping":
		result = map[string]any{}
	case "tools/list":
		tools := make([]map[string]any, 0, len(s.tools))
		for _, t := range s.tools {
			tools = append(tools, map[string]any{"name": t.Name, "description": t.Description, "inputSchema": t.InputSchema})
		}
		result = map[string]any{"tools": tools}
	case "tools/call":
		var params struct {
			Name      string          `json:"name"`
			Arguments json.RawMessage `json:"arguments"`
		}
		_ = json.Unmarshal(req.Params, &params)
		tool := s.find(params.Name)
		if tool == nil {
			result = textResult("没有这个工具："+params.Name, true)
			break
		}
		if err := s.guard.Authorize(r.Context(), tool.Permission); err != nil {
			response.WriteError(w, r, err)
			return
		}
		result = call(r.Context(), tool, params.Arguments)
	default:
		rpcErr = jsonrpc.NewError(jsonrpc.CodeMethodNotFound, "method not found: "+req.Method)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(jsonrpc.Reply(req.ID, result, rpcErr))
}

func (s *Server) find(name string) *Tool {
	for i := range s.tools {
		if s.tools[i].Name == name {
			return &s.tools[i]
		}
	}
	return nil
}

func call(ctx context.Context, tool *Tool, args json.RawMessage) map[string]any {
	callCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), callTimeout)
	defer cancel()
	text, err := tool.call(callCtx, args)
	if err != nil {
		return textResult(err.Error(), true)
	}
	return textResult(text, false)
}
