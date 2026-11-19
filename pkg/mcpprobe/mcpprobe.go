// Package mcpprobe 是 MCP 服务器配置的客户端一侧：解析常见的 mcpServers 配置，以及「测试连接」——
// 连一次服务器，握手，读工具列表，然后断开。
//
// 只实现用得到的那点协议：initialize、notifications/initialized、tools/list，
// 传输是 stdio（一行一条 JSON-RPC，见 pkg/jsonrpc）和 Streamable HTTP。
package mcpprobe

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"momoko/pkg/jsonrpc"
)

// Server 一个 MCP 服务器的连接配置。
type Server struct {
	Transport string            // stdio | http
	Command   string            // stdio：命令
	Args      []string          // stdio：参数
	Env       map[string]string // stdio：额外环境变量（在当前进程环境之上）
	Dir       string            // stdio：工作目录，空 = 当前目录
	URL       string            // http：地址
	Headers   map[string]string // http：请求头
}

// Result 连接成功后读到的东西。
type Result struct {
	Name    string // 服务器自报的名称
	Version string
	Tools   []string // 工具名
}

const (
	// 握手时报给服务器的协议版本；服务器答别的版本也照用。
	protocolVersion = "2025-06-18"
	// 调用方没给期限时，整个测试最多这么久。
	overallTimeout = 20 * time.Second
	// tools/list 最多翻这么多页，防着游标一直不结束的服务器。
	maxPages = 20
	// 放进错误提示里的文字最多这么多个字。
	maxReason = 300
)

var errNotMCP = errors.New("对方的应答不是 MCP 的格式")

// ParseConfig 解析 MCP 客户端通用的配置写法 {"mcpServers": {"名称": {command, args, env} 或 {url, headers}}}。
// 名称按字母序返回；既没有命令也没有地址的条目跳过。
func ParseConfig(raw string) ([]string, map[string]Server, error) {
	var doc struct {
		McpServers map[string]struct {
			Command string            `json:"command"`
			Args    []string          `json:"args"`
			Env     map[string]string `json:"env"`
			URL     string            `json:"url"`
			Headers map[string]string `json:"headers"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal([]byte(raw), &doc); err != nil || len(doc.McpServers) == 0 {
		return nil, nil, errors.New("没看懂这段 JSON：要的是 {\"mcpServers\": {...}} 的写法")
	}
	servers := map[string]Server{}
	for name, cfg := range doc.McpServers {
		switch {
		case strings.TrimSpace(cfg.URL) != "":
			servers[name] = Server{Transport: "http", URL: strings.TrimSpace(cfg.URL), Headers: cfg.Headers}
		case strings.TrimSpace(cfg.Command) != "":
			servers[name] = Server{Transport: "stdio", Command: strings.TrimSpace(cfg.Command), Args: cfg.Args, Env: cfg.Env}
		}
	}
	names := make([]string, 0, len(servers))
	for name := range servers {
		names = append(names, name)
	}
	sort.Strings(names)
	return names, servers, nil
}

// Probe 连一次：initialize → notifications/initialized → tools/list（翻完所有页）。
// 出错时返回给人看的中文原因，例如「命令 10 秒内没有响应」「找不到命令 node」「服务器返回 401」。
func Probe(ctx context.Context, s Server) (*Result, error) {
	if _, ok := ctx.Deadline(); !ok {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, overallTimeout)
		defer cancel()
	}
	var (
		c   conn
		err error
	)
	switch strings.ToLower(strings.TrimSpace(s.Transport)) {
	case "stdio":
		c, err = dialStdio(s)
	case "http":
		c, err = dialHTTP(s)
	default:
		return nil, fmt.Errorf("不支持的连接方式：%s", s.Transport)
	}
	if err != nil {
		return nil, err
	}
	defer c.close()

	res, err := handshake(ctx, c)
	if err != nil {
		if errors.Is(err, context.Canceled) {
			return nil, err
		}
		// 错误里有服务器说的话（报错、stderr），它可能把配置的令牌原样打出来。
		return nil, errors.New(redact(err.Error(), s))
	}
	return res, nil
}

// conn 一条到 MCP 服务器的连接，stdio 和 http 各一种。服务器答了 JSON-RPC 错误时 call 返回 *jsonrpc.Error。
type conn interface {
	call(ctx context.Context, method string, params, result any) error
	notify(ctx context.Context, method string) error
	// negotiated 握手定下协议版本之后调用。
	negotiated(version string)
	close()
}

func handshake(ctx context.Context, c conn) (*Result, error) {
	var init struct {
		ProtocolVersion string `json:"protocolVersion"`
		Capabilities    struct {
			Tools json.RawMessage `json:"tools"`
		} `json:"capabilities"`
		ServerInfo struct {
			Name    string `json:"name"`
			Version string `json:"version"`
		} `json:"serverInfo"`
	}
	err := c.call(ctx, "initialize", map[string]any{
		"protocolVersion": protocolVersion,
		"capabilities":    map[string]any{},
		"clientInfo":      map[string]any{"name": "momoko", "version": "1"},
	}, &init)
	if err != nil {
		var rpcErr *jsonrpc.Error
		if errors.As(err, &rpcErr) {
			return nil, fmt.Errorf("服务器拒绝了初始化：%s", reason(rpcErr))
		}
		return nil, err
	}
	c.negotiated(init.ProtocolVersion)
	if err := c.notify(ctx, "notifications/initialized"); err != nil {
		return nil, err
	}

	res := &Result{Name: init.ServerInfo.Name, Version: init.ServerInfo.Version, Tools: []string{}}
	// 服务器没声明 tools 能力：它没有工具，不用问。
	if len(init.Capabilities.Tools) == 0 || string(init.Capabilities.Tools) == "null" {
		return res, nil
	}
	cursor := ""
	for page := 0; page < maxPages; page++ {
		params := map[string]any{}
		if cursor != "" {
			params["cursor"] = cursor
		}
		var list struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
			NextCursor string `json:"nextCursor"`
		}
		if err := c.call(ctx, "tools/list", params, &list); err != nil {
			var rpcErr *jsonrpc.Error
			if !errors.As(err, &rpcErr) {
				return nil, err
			}
			if rpcErr.Code == jsonrpc.CodeMethodNotFound {
				// 声明了能力却不认这个方法：同样当作没有工具。
				break
			}
			return nil, fmt.Errorf("读工具列表失败：%s", reason(rpcErr))
		}
		for _, t := range list.Tools {
			res.Tools = append(res.Tools, t.Name)
		}
		if list.NextCursor == "" || list.NextCursor == cursor {
			break
		}
		cursor = list.NextCursor
	}
	return res, nil
}

func reason(e *jsonrpc.Error) string {
	if msg := strings.TrimSpace(e.Message); msg != "" {
		return clip(msg)
	}
	return fmt.Sprintf("错误码 %d", e.Code)
}

// timeoutError ctx 结束时的说法：被取消的原样交回去，到期的说是谁没响应。
func timeoutError(ctx context.Context, who string) error {
	if errors.Is(ctx.Err(), context.Canceled) {
		return context.Canceled
	}
	return fmt.Errorf("%s没有在限定时间内响应", who)
}

func clip(s string) string {
	if utf8.RuneCountInString(s) <= maxReason {
		return s
	}
	return string([]rune(s)[:maxReason-1]) + "…"
}

// redact 把配置里的请求头值、环境变量值从文字里抹掉。
func redact(text string, s Server) string {
	var secrets []string
	add := func(v string) {
		// 太短的值（true、1、production）不是密钥，抹掉反而把提示弄花。
		if v = strings.TrimSpace(v); len(v) >= 8 {
			secrets = append(secrets, v)
		}
	}
	for _, v := range s.Headers {
		add(v)
		// Authorization: Bearer xxx —— 服务器多半只回显 xxx 那一段。
		if _, token, ok := strings.Cut(strings.TrimSpace(v), " "); ok {
			add(token)
		}
	}
	for _, v := range s.Env {
		add(v)
	}
	// 长的先换，免得短的先把长的拆散。
	sort.Slice(secrets, func(i, j int) bool { return len(secrets[i]) > len(secrets[j]) })
	for _, v := range secrets {
		text = strings.ReplaceAll(text, v, "***")
	}
	return text
}
