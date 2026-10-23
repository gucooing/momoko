// Package iqcheck 封装智商检测对上游 AI 接口的调用：拉取模型列表与发送单轮对话，
// 支持 chat（OpenAI Chat Completions）、responses（OpenAI Responses）、messages（Anthropic Messages）三种协议。
// 本包零业务依赖，只处理协议细节；站点、提示词和结果的持久化与调度在 internal 层。
package iqcheck

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// Protocol 上游接口协议。
type Protocol string

const (
	// ProtocolChat OpenAI Chat Completions：POST {base}/chat/completions
	ProtocolChat Protocol = "chat"
	// ProtocolResponses OpenAI Responses：POST {base}/responses
	ProtocolResponses Protocol = "responses"
	// ProtocolMessages Anthropic Messages：POST {base}/messages
	ProtocolMessages Protocol = "messages"
)

// Valid 报告协议是否受支持。
func (p Protocol) Valid() bool {
	switch p {
	case ProtocolChat, ProtocolResponses, ProtocolMessages:
		return true
	}
	return false
}

// Efforts 推理强度可选值；空串表示不设置，由上游使用默认值。
var Efforts = []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}

// anthropicEfforts 是 Anthropic output_config.effort 接受的取值；其余值在 messages 协议下不发送。
var anthropicEfforts = map[string]struct{}{"low": {}, "medium": {}, "high": {}, "xhigh": {}, "max": {}}

// ValidEffort 报告推理强度取值是否合法（空串合法）。
func ValidEffort(effort string) bool {
	if effort == "" {
		return true
	}
	for _, e := range Efforts {
		if e == effort {
			return true
		}
	}
	return false
}

const (
	// DefaultTimeout 单次检测请求的总超时；推理模型的回复可能长达数分钟。
	DefaultTimeout = 10 * time.Minute
	// ModelsTimeout 拉取模型列表的超时。
	ModelsTimeout = 30 * time.Second
	// DefaultIdleTimeout bounds the wait for headers or the next response bytes.
	DefaultIdleTimeout = 2 * time.Minute

	anthropicVersion = "2023-06-01"
	// defaultMaxTokens messages 协议必填的输出上限；超过模型上限时按上游提示的上限重试一次。
	defaultMaxTokens = 16000
	maxErrorBody     = 1 << 20
	maxModelPages    = 20
)

// NormalizeEndpoint 规范化站点地址：必须是 http(s) 完整地址，去掉末尾斜杠，路径未以 /v1 结尾时补上。
func NormalizeEndpoint(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", errors.New("站点地址不能为空")
	}
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return "", errors.New("站点地址必须是 http:// 或 https:// 开头的完整地址")
	}
	u.Path = strings.TrimRight(u.Path, "/")
	u.RawPath = ""
	if !strings.HasSuffix(strings.ToLower(u.Path), "/v1") {
		u.Path += "/v1"
	}
	u.RawQuery = ""
	u.Fragment = ""
	return u.String(), nil
}

// MaskAPIKey 返回脱敏后的 API Key，仅保留首尾各 4 位。
func MaskAPIKey(key string) string {
	if len(key) <= 8 {
		return "********"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// DefaultUserAgent 请求上游时默认使用的 User-Agent。
const DefaultUserAgent = "pi"

// Config 客户端配置。
type Config struct {
	Endpoint    string
	APIKey      string
	Protocol    Protocol
	UserAgent   string        // 为空时使用 DefaultUserAgent
	Timeout     time.Duration // 单次检测总超时；0 使用 DefaultTimeout
	IdleTimeout time.Duration // 等待响应数据的超时；0 使用 DefaultIdleTimeout
}

// Client 一个站点的上游调用客户端。
type Client struct {
	base        string
	apiKey      string
	protocol    Protocol
	userAgent   string
	http        *http.Client
	timeout     time.Duration
	idleTimeout time.Duration
}

// NewClient 构造客户端；Endpoint 会被 NormalizeEndpoint 规范化。
func NewClient(cfg Config) (*Client, error) {
	base, err := NormalizeEndpoint(cfg.Endpoint)
	if err != nil {
		return nil, err
	}
	if !cfg.Protocol.Valid() {
		return nil, errors.New("协议不合法")
	}
	userAgent := strings.TrimSpace(cfg.UserAgent)
	if userAgent == "" {
		userAgent = DefaultUserAgent
	}
	if cfg.Timeout <= 0 {
		cfg.Timeout = DefaultTimeout
	}
	if cfg.IdleTimeout <= 0 {
		cfg.IdleTimeout = DefaultIdleTimeout
	}
	return &Client{
		base:        base,
		apiKey:      strings.TrimSpace(cfg.APIKey),
		protocol:    cfg.Protocol,
		userAgent:   userAgent,
		http:        &http.Client{},
		timeout:     cfg.Timeout,
		idleTimeout: cfg.IdleTimeout,
	}, nil
}

// Base 返回规范化后的接口根地址（以 /v1 结尾）。
func (c *Client) Base() string { return c.base }

// Request 一次检测请求。
type Request struct {
	Model  string
	Effort string
	Prompt string
}

// Result 上游回复；token 计数在上游未返回时为 0。
type Result struct {
	Text         string
	InputTokens  int64
	OutputTokens int64
}

// ListModels 拉取上游模型 id 列表（去重、排序）。
// OpenAI 兼容接口一次返回全部；Anthropic 按 has_more/last_id 分页。
func (c *Client) ListModels(ctx context.Context) ([]string, error) {
	ctx, cancel := context.WithTimeout(ctx, ModelsTimeout)
	defer cancel()
	seen := make(map[string]struct{})
	var models []string
	after := ""
	for page := 0; page < maxModelPages; page++ {
		path := "/models?limit=1000"
		if after != "" {
			path += "&after_id=" + url.QueryEscape(after)
		}
		req, err := c.newRequest(ctx, http.MethodGet, path, nil)
		if err != nil {
			return nil, err
		}
		resp, err := c.do(req)
		if err != nil {
			return nil, fmt.Errorf("请求上游失败: %w", err)
		}
		var payload struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
		}
		err = decodeJSON(resp.Body, &payload)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("上游模型列表格式无法解析: %w", err)
		}
		for _, m := range payload.Data {
			if m.ID == "" {
				continue
			}
			if _, ok := seen[m.ID]; ok {
				continue
			}
			seen[m.ID] = struct{}{}
			models = append(models, m.ID)
		}
		if !payload.HasMore || payload.LastID == "" {
			break
		}
		after = payload.LastID
	}
	sort.Strings(models)
	return models, nil
}

// Complete sends one request. On failure it also returns any text and usage
// already received, so callers can persist a useful failed result.
func (c *Client) Complete(ctx context.Context, req Request) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()
	var res *Result
	var err error
	switch c.protocol {
	case ProtocolChat:
		res, err = c.completeChat(ctx, req)
	case ProtocolResponses:
		res, err = c.completeResponses(ctx, req)
	case ProtocolMessages:
		res, err = c.completeMessages(ctx, req)
	default:
		return nil, errors.New("协议不合法")
	}
	if err == nil && (res == nil || strings.TrimSpace(res.Text) == "") {
		err = errors.New("上游未返回回复文本")
	}
	return res, err
}

type anthropicUsage struct {
	InputTokens  *int64 `json:"input_tokens"`
	OutputTokens *int64 `json:"output_tokens"`
}

func (u *anthropicUsage) apply(res *Result) {
	if u == nil {
		return
	}
	if u.InputTokens != nil {
		res.InputTokens = *u.InputTokens
	}
	if u.OutputTokens != nil {
		res.OutputTokens = *u.OutputTokens
	}
}

// textContent 兼容 content 为字符串或 [{type:"text",text:"..."}] 分片数组的两种写法。
type textContent string

func (t *textContent) UnmarshalJSON(b []byte) error {
	if len(b) == 0 || string(b) == "null" {
		*t = ""
		return nil
	}
	if b[0] == '"' {
		var s string
		if err := json.Unmarshal(b, &s); err != nil {
			return err
		}
		*t = textContent(s)
		return nil
	}
	if b[0] == '[' {
		var parts []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		}
		if err := json.Unmarshal(b, &parts); err != nil {
			return err
		}
		var sb strings.Builder
		for _, p := range parts {
			sb.WriteString(p.Text)
		}
		*t = textContent(sb.String())
		return nil
	}
	return errors.New("不支持的 content 格式")
}

// apiError 上游 JSON 中的 error 对象。
type apiError struct {
	Message string          `json:"message"`
	Type    string          `json:"type"`
	Code    json.RawMessage `json:"code"`
}

func (e *apiError) UnmarshalJSON(data []byte) error {
	if len(data) > 0 && data[0] == '"' {
		return json.Unmarshal(data, &e.Message)
	}
	type plain apiError
	return json.Unmarshal(data, (*plain)(e))
}

func (e *apiError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Type != "" {
		return e.Type
	}
	if len(e.Code) > 0 && string(e.Code) != "null" {
		return strings.Trim(string(e.Code), "\"")
	}
	return "上游返回错误"
}

// HTTPError 上游返回非 200 状态码。
type HTTPError struct {
	Status  int
	Message string
}

func (e *HTTPError) Error() string {
	return fmt.Sprintf("上游返回 HTTP %d: %s", e.Status, e.Message)
}

// httpError 从响应体中提取可读的错误信息。
func httpError(status int, body []byte) error {
	msg := extractErrorMessage(body)
	if msg == "" {
		msg = abbreviate(string(body), 300)
	}
	if msg == "" {
		msg = http.StatusText(status)
	}
	return &HTTPError{Status: status, Message: msg}
}

func extractErrorMessage(body []byte) string {
	var payload struct {
		Error   json.RawMessage `json:"error"`
		Message string          `json:"message"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return ""
	}
	if len(payload.Error) > 0 {
		var obj apiError
		if err := json.Unmarshal(payload.Error, &obj); err == nil && obj.Message != "" {
			return obj.Message
		}
		var s string
		if err := json.Unmarshal(payload.Error, &s); err == nil && s != "" {
			return s
		}
	}
	return payload.Message
}

func abbreviate(s string, n int) string {
	s = strings.TrimSpace(s)
	r := []rune(s)
	if len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

// —— HTTP 基础 ——

func (c *Client) newRequest(ctx context.Context, method, path string, body any) (*http.Request, error) {
	var reader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		reader = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base+path, reader)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	if c.protocol == ProtocolMessages {
		req.Header.Set("x-api-key", c.apiKey)
		req.Header.Set("anthropic-version", anthropicVersion)
	}
	return req, nil
}

// post 发送请求；非 200 时读取响应体并转为 HTTPError，调用方负责关闭成功响应的 Body。
func (c *Client) post(ctx context.Context, path string, body any) (*http.Response, error) {
	req, err := c.newRequest(ctx, http.MethodPost, path, body)
	if err != nil {
		return nil, err
	}
	return c.do(req)
}

func isEventStream(resp *http.Response) bool {
	return strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream")
}

func decodeJSON(r io.Reader, v any) error {
	// Decode one JSON value without waiting for the gateway to close its body.
	if err := json.NewDecoder(io.LimitReader(r, 32<<20)).Decode(v); err != nil {
		return fmt.Errorf("上游响应格式无法解析: %w", err)
	}
	return nil
}
