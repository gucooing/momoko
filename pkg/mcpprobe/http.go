package mcpprobe

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"momoko/pkg/jsonrpc"
	"momoko/pkg/utils"
)

const (
	// 出错时的响应体只看开头这一点。
	maxErrorBody = 64 << 10
	// 一个应答最长多少字节。
	maxBody = 16 << 20
	// 结束时删除会话是顺手做的，不为它多等。
	deleteTimeout = 3 * time.Second
)

// 不设 Transport：用默认的那个，代理环境变量才生效。也不设 Timeout：应答可能是一条事件流，期限由 ctx 管。
var httpClient = &http.Client{}

// httpConn 是 MCP 的 Streamable HTTP 传输：每条消息一个 POST，应答是 JSON 或者一条事件流。
type httpConn struct {
	url     string
	host    string
	headers map[string]string
	session string // 服务器在 initialize 时发的会话 ID，之后每个请求都要带
	version string // 握手定下的协议版本
	nextID  int64
}

func dialHTTP(s Server) (conn, error) {
	raw := strings.TrimSpace(s.URL)
	if raw == "" {
		return nil, errors.New("地址还没填")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("地址不对，要以 http:// 或 https:// 开头")
	}
	for name, value := range s.Headers {
		if !validHeader(name, value) {
			return nil, fmt.Errorf("请求头 %s 的写法不对", name)
		}
	}
	return &httpConn{url: raw, host: u.Host, headers: s.Headers}, nil
}

// validHeader 请求头的名称和值里不能有换行这类字符，否则请求根本发不出去。
func validHeader(name, value string) bool {
	if name == "" {
		return false
	}
	for _, r := range name {
		if r <= ' ' || r >= 0x7f || strings.ContainsRune(`()<>@,;:\"/[]?={}`, r) {
			return false
		}
	}
	for _, r := range value {
		if (r < ' ' && r != '\t') || r == 0x7f {
			return false
		}
	}
	return true
}

func (c *httpConn) call(ctx context.Context, method string, params, result any) error {
	c.nextID++
	id := c.nextID
	resp, err := c.post(ctx, id, method, params)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" && method == "initialize" {
		c.session = sid
	}
	var msg *jsonrpc.Message
	if kind, _, _ := mime.ParseMediaType(resp.Header.Get("Content-Type")); kind == "text/event-stream" {
		msg, err = readEvents(resp.Body, id)
	} else {
		msg, err = readJSON(resp.Body, id)
	}
	if err != nil {
		if errors.Is(err, errNotMCP) {
			return err
		}
		return c.netError(ctx, err)
	}
	raw, err := msg.Outcome()
	if err != nil || result == nil || len(raw) == 0 {
		return err
	}
	if json.Unmarshal(raw, result) != nil {
		return errNotMCP
	}
	return nil
}

func (c *httpConn) notify(ctx context.Context, method string) error {
	resp, err := c.post(ctx, 0, method, nil)
	if err != nil {
		return err
	}
	// 通知没有应答，200 / 202 / 204 都算收到了。
	resp.Body.Close()
	return nil
}

func (c *httpConn) negotiated(version string) { c.version = version }

// post 发一条消息（id 为 0 是通知）；状态码不是 2xx 时直接返回给人看的错误。
func (c *httpConn) post(ctx context.Context, id int64, method string, params any) (*http.Response, error) {
	msg, err := jsonrpc.Request(id, method, params)
	if err != nil {
		return nil, err
	}
	raw, err := json.Marshal(msg)
	if err != nil {
		return nil, err
	}
	req, err := c.request(ctx, http.MethodPost, bytes.NewReader(raw))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, c.netError(ctx, err)
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return nil, fmt.Errorf("服务器返回 %d，检查请求头里的令牌", resp.StatusCode)
	}
	// 有的服务器会在响应体里用 JSON-RPC 错误说明原因。
	body, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
	if reply := jsonrpc.Parse(body); reply != nil && reply.Error != nil && strings.TrimSpace(reply.Error.Message) != "" {
		return nil, fmt.Errorf("服务器返回 %d：%s", resp.StatusCode, reason(reply.Error))
	}
	return nil, fmt.Errorf("服务器返回 %d", resp.StatusCode)
}

// request 建一个带上配置的请求头、会话 ID 和协议版本的请求。
func (c *httpConn) request(ctx context.Context, method string, body io.Reader) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, method, c.url, body)
	if err != nil {
		return nil, errors.New("地址不对")
	}
	for name, value := range c.headers {
		if strings.EqualFold(name, "Host") {
			req.Host = value
			continue
		}
		req.Header.Set(name, value)
	}
	if c.session != "" {
		req.Header.Set("Mcp-Session-Id", c.session)
	}
	if c.version != "" {
		req.Header.Set("MCP-Protocol-Version", c.version)
	}
	return req, nil
}

// close 握手时服务器发过会话 ID 的话，顺手告诉它这个会话不用了。
func (c *httpConn) close() {
	if c.session == "" {
		return
	}
	// 测试的 ctx 这时可能已经到期，另起一个。
	ctx, cancel := context.WithTimeout(context.Background(), deleteTimeout)
	defer cancel()
	req, err := c.request(ctx, http.MethodDelete, nil)
	if err != nil {
		return
	}
	if resp, err := httpClient.Do(req); err == nil {
		resp.Body.Close()
	}
}

func (c *httpConn) netError(ctx context.Context, err error) error {
	if ctx.Err() != nil {
		return timeoutError(ctx, c.host+" ")
	}
	return utils.NetError(ctx, c.host, err)
}

// answers 这条消息是不是编号为 id 的请求的应答。
func answers(msg *jsonrpc.Message, id int64) bool {
	return msg != nil && msg.Method == "" && msg.HasID() && (msg.Result != nil || msg.Error != nil) &&
		msg.IDKey() == strconv.FormatInt(id, 10)
}

// readJSON 读 application/json 的应答：一条 JSON-RPC 应答，个别服务器会包成数组。
func readJSON(r io.Reader, id int64) (*jsonrpc.Message, error) {
	body, err := io.ReadAll(io.LimitReader(r, maxBody))
	if err != nil {
		return nil, err
	}
	body = bytes.TrimSpace(body)
	if len(body) > 0 && body[0] == '[' {
		var batch []*jsonrpc.Message
		if json.Unmarshal(body, &batch) != nil {
			return nil, errNotMCP
		}
		for _, msg := range batch {
			if answers(msg, id) {
				return msg, nil
			}
		}
		return nil, errNotMCP
	}
	if msg := jsonrpc.Parse(body); answers(msg, id) {
		return msg, nil
	}
	return nil, errNotMCP
}

// readEvents 读事件流，直到编号为 id 的那条应答；之前的通知和服务器发来的请求都跳过。
func readEvents(r io.Reader, id int64) (*jsonrpc.Message, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64<<10), maxBody)
	var data []byte
	for scanner.Scan() {
		line := bytes.TrimRight(scanner.Bytes(), "\r")
		switch {
		case len(line) == 0:
			// 空行是一个事件的结束。
			if msg := jsonrpc.Parse(data); answers(msg, id) {
				return msg, nil
			}
			data = data[:0]
		case bytes.HasPrefix(line, []byte("data:")):
			// 一个事件可以有多行 data，按规范用换行接起来。
			if len(data) > 0 {
				data = append(data, '\n')
			}
			data = append(data, bytes.TrimPrefix(line[len("data:"):], []byte(" "))...)
		}
	}
	// 流结束了：最后一个事件后面可能没有空行。
	if msg := jsonrpc.Parse(data); answers(msg, id) {
		return msg, nil
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return nil, io.ErrUnexpectedEOF
}
