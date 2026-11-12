// Package mgp 是 Momoko 调用游戏服务端 MGP v1（Momoko Game Protocol）的通用客户端：
// 请求签名（pkg/mgpauth）、超时、GET 重试、统一响应信封解析与错误码映射、操作的幂等键与试运行，
// 以及服务器管理用到的 manifest、status 与事件流。各游戏的接口与类型在 pkg/mgp/<游戏> 下实现。
//
// 协议约定（与游戏服务端一致）：基础路径 /mgp/v1；响应 { code, message, data, requestId }；
// 时间为 Unix 毫秒；只返回资源 ID，不提供图片。
package mgp

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"momoko/pkg/mgpauth"
	"momoko/pkg/response"
)

// Prefix 是 MGP 路由前缀。
const Prefix = "/mgp/v1"

const (
	defaultTimeout = 15 * time.Second
	// OperationTimeout 执行操作（全服邮件等可能较慢）的超时。
	OperationTimeout = 2 * time.Minute
	maxResponse      = 64 << 20
)

// 操作的请求头与响应头（与服务端一致）。
const (
	headerIdempotencyKey = "Idempotency-Key"
	headerOperator       = "X-MGP-Operator"
	headerReplayed       = "Idempotent-Replayed"
)

// Error 是 MGP 返回的业务错误或传输错误。
type Error struct {
	// Status HTTP 状态码；0 表示没有拿到响应（网络错误、超时）。
	Status    int
	Code      string
	Message   string
	RequestID string
}

func (e *Error) Error() string {
	if e.Status == 0 {
		return fmt.Sprintf("mgp: %s: %s", e.Code, e.Message)
	}
	return fmt.Sprintf("mgp: %d %s: %s", e.Status, e.Code, e.Message)
}

// CodeOf 返回 err 携带的 MGP 错误码；不是 MGP 错误时返回空串。
func CodeOf(err error) string {
	var e *Error
	if errors.As(err, &e) {
		return e.Code
	}
	return ""
}

// ResponseError 把 MGP 错误映射为 Momoko 接口错误：连接不上与签名失败为 502（不能让前端当成 Momoko 登录失效），
// 其余沿用 MGP 的 HTTP 状态，信息为「错误码: 信息」；不是 MGP 错误时原样返回。
func ResponseError(err error) error {
	var e *Error
	if !errors.As(err, &e) {
		return err
	}
	status := e.Status
	if status == 0 || status == http.StatusUnauthorized || status < 400 {
		status = http.StatusBadGateway
	}
	return response.BadRequest(status, e.Code+": "+e.Message)
}

// IsUnreachable 判断错误是否为连接不上游戏服务端（网络错误、超时）。
func IsUnreachable(err error) bool {
	var e *Error
	return errors.As(err, &e) && e.Status == 0
}

// Credentials 是一台游戏服务器的 MGP 接入信息。
type Credentials struct {
	// Endpoint 服务端 MGP 地址，例如 http://127.0.0.1:8081（不含 /mgp/v1）。
	Endpoint string
	KeyID    string
	Secret   string
}

// Client 调用一台游戏服务器的 MGP。并发安全。
type Client struct {
	base   *url.URL
	keyID  string
	secret string
	http   *http.Client
	// stream 用于 SSE 事件流（不设整体超时）。
	stream *http.Client
}

// NormalizeEndpoint 规范化 MGP 地址：补全协议、去掉末尾的 / 与 /mgp/v1。
func NormalizeEndpoint(endpoint string) (string, error) {
	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", errors.New("MGP 地址不能为空")
	}
	if !strings.Contains(endpoint, "://") {
		endpoint = "http://" + endpoint
	}
	u, err := url.Parse(endpoint)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", fmt.Errorf("MGP 地址无效：%s", endpoint)
	}
	u.Path = strings.TrimSuffix(strings.TrimSuffix(u.Path, "/"), Prefix)
	u.Path = strings.TrimSuffix(u.Path, "/")
	u.RawQuery, u.Fragment = "", ""
	return u.String(), nil
}

// NewClient 创建客户端。
func NewClient(c Credentials) (*Client, error) {
	endpoint, err := NormalizeEndpoint(c.Endpoint)
	if err != nil {
		return nil, err
	}
	if c.KeyID == "" || c.Secret == "" {
		return nil, errors.New("MGP 密钥未配置")
	}
	base, _ := url.Parse(endpoint)
	transport := &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		DialContext:         (&net.Dialer{Timeout: 5 * time.Second, KeepAlive: 30 * time.Second}).DialContext,
		MaxIdleConnsPerHost: 8,
		IdleConnTimeout:     90 * time.Second,
	}
	return &Client{
		base:   base,
		keyID:  c.KeyID,
		secret: c.Secret,
		http:   &http.Client{Transport: transport},
		stream: &http.Client{Transport: transport},
	}, nil
}

// envelope 是 MGP 统一响应。
type envelope struct {
	Code      string          `json:"code"`
	Message   string          `json:"message"`
	Data      json.RawMessage `json:"data"`
	RequestID string          `json:"requestId"`
}

// OperationOptions 是一次操作（改变数据的请求）的选项。
type OperationOptions struct {
	// IdempotencyKey 幂等键：同一个键只执行一次，重复请求返回首次的结果；试运行可为空。
	IdempotencyKey string
	// Operator 操作者（写进服务端日志与事件）。
	Operator string
	// DryRun 试运行：不改变任何数据。
	DryRun bool
}

// Call 是一次请求的选项。
type Call struct {
	Method string
	// Path 是 /mgp/v1 之后的路径，例如 /players/12/items（各段需已转义）。
	Path  string
	Query url.Values
	// Body 为 nil 时不带请求体；否则按 JSON 编码（json.RawMessage 原样发送）。
	Body any
	// RequestID 透传为 X-MGP-Request-Id，便于两端日志对应。
	RequestID string
	// Operation 操作的幂等键、操作者与试运行；查询为 nil。
	Operation *OperationOptions
	// Timeout 为 0 时取默认超时。
	Timeout time.Duration
}

// Result 是一次请求的结果：data 的原始 JSON，以及操作是否为幂等键的重放（服务端返回首次执行的结果）。
type Result struct {
	Data     json.RawMessage
	Replayed bool
}

// Do 发送请求并返回结果。GET 在网络错误时重试一次。
func (c *Client) Do(ctx context.Context, call Call) (Result, error) {
	var body []byte
	if call.Body != nil {
		switch b := call.Body.(type) {
		case json.RawMessage:
			body = b
		case []byte:
			body = b
		default:
			raw, err := json.Marshal(b)
			if err != nil {
				return Result{}, err
			}
			body = raw
		}
	}
	attempts := 1
	if call.Method == http.MethodGet {
		attempts = 2
	}
	var err error
	for i := 0; i < attempts; i++ {
		var r Result
		r, err = c.do(ctx, call, body)
		if err == nil || !IsUnreachable(err) || ctx.Err() != nil {
			return r, err
		}
	}
	return Result{}, err
}

// Get 是 Do 的 GET 简写，返回 data 的原始 JSON。
func (c *Client) Get(ctx context.Context, path string, query url.Values) (json.RawMessage, error) {
	r, err := c.Do(ctx, Call{Method: http.MethodGet, Path: path, Query: query})
	return r.Data, err
}

// Operate 执行一次操作（POST / PUT / DELETE），body 为 nil 时不带请求体。
func (c *Client) Operate(ctx context.Context, method, path string, body any, opt OperationOptions) (Result, error) {
	return c.Do(ctx, Call{Method: method, Path: path, Body: body, Operation: &opt, RequestID: opt.IdempotencyKey, Timeout: OperationTimeout})
}

// GetInto 发送 GET 并把 data 解码进 out。
func (c *Client) GetInto(ctx context.Context, path string, query url.Values, out any) error {
	raw, err := c.Get(ctx, path, query)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, out)
}

func (c *Client) newRequest(ctx context.Context, call Call, body []byte) (*http.Request, error) {
	u := *c.base
	u.Path = u.Path + Prefix + call.Path
	u.RawPath = ""
	query := call.Query
	if call.Operation != nil && call.Operation.DryRun {
		query = url.Values{}
		for k, v := range call.Query {
			query[k] = v
		}
		query.Set("dryRun", "true")
	}
	if len(query) > 0 {
		u.RawQuery = query.Encode()
	}
	var reader io.Reader
	if body != nil {
		reader = bytes.NewReader(body)
	}
	req, err := http.NewRequestWithContext(ctx, call.Method, u.String(), reader)
	if err != nil {
		return nil, err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	if call.RequestID != "" {
		req.Header.Set(mgpauth.HeaderRequestID, call.RequestID)
	}
	if op := call.Operation; op != nil {
		if op.IdempotencyKey != "" {
			req.Header.Set(headerIdempotencyKey, op.IdempotencyKey)
		}
		if op.Operator != "" {
			req.Header.Set(headerOperator, op.Operator)
		}
	}
	if err := mgpauth.SignRequest(req, c.keyID, c.secret, body, time.Now()); err != nil {
		return nil, err
	}
	return req, nil
}

func (c *Client) do(ctx context.Context, call Call, body []byte) (Result, error) {
	timeout := call.Timeout
	if timeout <= 0 {
		timeout = defaultTimeout
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	req, err := c.newRequest(ctx, call, body)
	if err != nil {
		return Result{}, err
	}
	res, err := c.http.Do(req)
	if err != nil {
		code := "UNREACHABLE"
		if errors.Is(err, context.DeadlineExceeded) {
			code = "TIMEOUT"
		}
		return Result{}, &Error{Code: code, Message: err.Error()}
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(io.LimitReader(res.Body, maxResponse))
	if err != nil {
		return Result{}, &Error{Code: "UNREACHABLE", Message: err.Error()}
	}
	var env envelope
	if err := json.Unmarshal(raw, &env); err != nil || env.Code == "" {
		msg := strings.TrimSpace(string(raw))
		if len(msg) > 200 {
			msg = msg[:200]
		}
		if msg == "" {
			msg = res.Status
		}
		return Result{}, &Error{Status: res.StatusCode, Code: "BAD_RESPONSE", Message: msg}
	}
	if res.StatusCode != http.StatusOK || env.Code != "OK" {
		return Result{}, &Error{Status: res.StatusCode, Code: env.Code, Message: env.Message, RequestID: env.RequestID}
	}
	if len(env.Data) == 0 {
		env.Data = json.RawMessage("null")
	}
	return Result{Data: env.Data, Replayed: res.Header.Get(headerReplayed) == "true"}, nil
}

// Event 是 /events 事件流中的一条事件。
type Event struct {
	// Kind 事件类型，由游戏服务端定义（学マス：session.login、session.disconnect、session.kick、gasha.draw、produce.finish、gm.operation、config.publish）。
	Kind string `json:"kind"`
	// Time Unix 毫秒。
	Time         int64           `json:"time"`
	AccountID    json.Number     `json:"accountId"`
	PublicUserID string          `json:"publicUserId"`
	Name         string          `json:"name"`
	Data         json.RawMessage `json:"data"`
}

// Events 订阅 SSE 事件流，逐条回调直到 ctx 结束或连接断开（返回断开原因）。
func (c *Client) Events(ctx context.Context, fn func(Event)) error {
	req, err := c.newRequest(ctx, Call{Method: http.MethodGet, Path: "/events"}, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	res, err := c.stream.Do(req)
	if err != nil {
		return &Error{Code: "UNREACHABLE", Message: err.Error()}
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		raw, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		var env envelope
		if json.Unmarshal(raw, &env) == nil && env.Code != "" {
			return &Error{Status: res.StatusCode, Code: env.Code, Message: env.Message}
		}
		return &Error{Status: res.StatusCode, Code: "BAD_RESPONSE", Message: res.Status}
	}
	scanner := bufio.NewScanner(res.Body)
	scanner.Buffer(make([]byte, 64<<10), 4<<20)
	var kind string
	var data strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		switch {
		case line == "":
			if data.Len() > 0 {
				var ev Event
				if json.Unmarshal([]byte(data.String()), &ev) == nil {
					if ev.Kind == "" {
						ev.Kind = kind
					}
					fn(ev)
				}
			}
			kind = ""
			data.Reset()
		case strings.HasPrefix(line, ":"):
			// 保活注释
		case strings.HasPrefix(line, "event:"):
			kind = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		case strings.HasPrefix(line, "data:"):
			if data.Len() > 0 {
				data.WriteByte('\n')
			}
			data.WriteString(strings.TrimPrefix(strings.TrimPrefix(line, "data:"), " "))
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return io.EOF
}
