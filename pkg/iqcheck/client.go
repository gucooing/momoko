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
	"regexp"
	"sort"
	"strconv"
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
	Endpoint  string
	APIKey    string
	Protocol  Protocol
	UserAgent string // 为空时使用 DefaultUserAgent
}

// Client 一个站点的上游调用客户端。
type Client struct {
	base      string
	apiKey    string
	protocol  Protocol
	userAgent string
	http      *http.Client
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
	return &Client{
		base:      base,
		apiKey:    strings.TrimSpace(cfg.APIKey),
		protocol:  cfg.Protocol,
		userAgent: userAgent,
		http:      &http.Client{},
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
		resp, err := c.http.Do(req)
		if err != nil {
			return nil, fmt.Errorf("请求上游失败: %w", err)
		}
		body, err := io.ReadAll(io.LimitReader(resp.Body, 8<<20))
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("读取上游响应失败: %w", err)
		}
		if resp.StatusCode != http.StatusOK {
			return nil, httpError(resp.StatusCode, body)
		}
		var payload struct {
			Data []struct {
				ID string `json:"id"`
			} `json:"data"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
		}
		if err := json.Unmarshal(body, &payload); err != nil {
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

// Complete 按协议发送一轮对话并返回回复文本。
func (c *Client) Complete(ctx context.Context, req Request) (*Result, error) {
	ctx, cancel := context.WithTimeout(ctx, DefaultTimeout)
	defer cancel()
	switch c.protocol {
	case ProtocolChat:
		return c.completeChat(ctx, req)
	case ProtocolResponses:
		return c.completeResponses(ctx, req)
	case ProtocolMessages:
		return c.completeMessages(ctx, req, defaultMaxTokens)
	}
	return nil, errors.New("协议不合法")
}

// completeChat OpenAI Chat Completions；优先按 SSE 流式读取，上游返回普通 JSON 时也能解析。
func (c *Client) completeChat(ctx context.Context, req Request) (*Result, error) {
	body := map[string]any{
		"model":          req.Model,
		"messages":       []map[string]string{{"role": "user", "content": req.Prompt}},
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	if req.Effort != "" {
		body["reasoning_effort"] = req.Effort
	}
	resp, err := c.post(ctx, "/chat/completions", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	res := &Result{}
	if isEventStream(resp) {
		var sb strings.Builder
		err := readSSE(resp.Body, func(_ string, data string) error {
			if data == "[DONE]" {
				return nil
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content textContent `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
				Usage *chatUsage `json:"usage"`
				Error *apiError  `json:"error"`
			}
			if err := json.Unmarshal([]byte(data), &chunk); err != nil {
				return nil // 无法解析的片段（如心跳）忽略
			}
			if chunk.Error != nil {
				return chunk.Error
			}
			for _, choice := range chunk.Choices {
				sb.WriteString(string(choice.Delta.Content))
			}
			if chunk.Usage != nil {
				res.InputTokens, res.OutputTokens = chunk.Usage.PromptTokens, chunk.Usage.CompletionTokens
			}
			return nil
		})
		res.Text = sb.String()
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	var payload struct {
		Choices []struct {
			Message struct {
				Content textContent `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Usage *chatUsage `json:"usage"`
		Error *apiError  `json:"error"`
	}
	if err := decodeJSON(resp.Body, &payload); err != nil {
		return nil, err
	}
	if payload.Error != nil {
		return nil, payload.Error
	}
	var sb strings.Builder
	for _, choice := range payload.Choices {
		sb.WriteString(string(choice.Message.Content))
	}
	res.Text = sb.String()
	if payload.Usage != nil {
		res.InputTokens, res.OutputTokens = payload.Usage.PromptTokens, payload.Usage.CompletionTokens
	}
	return res, nil
}

// completeResponses OpenAI Responses；事件名缺失时按 data.type 识别。
func (c *Client) completeResponses(ctx context.Context, req Request) (*Result, error) {
	body := map[string]any{
		"model":  req.Model,
		"input":  req.Prompt,
		"stream": true,
	}
	if req.Effort != "" {
		body["reasoning"] = map[string]any{"effort": req.Effort}
	}
	resp, err := c.post(ctx, "/responses", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	res := &Result{}
	if isEventStream(resp) {
		var sb strings.Builder
		err := readSSE(resp.Body, func(event string, data string) error {
			var payload struct {
				Type     string           `json:"type"`
				Delta    string           `json:"delta"`
				Response *responsesResult `json:"response"`
				Error    *apiError        `json:"error"`
				Message  string           `json:"message"`
			}
			if err := json.Unmarshal([]byte(data), &payload); err != nil {
				return nil
			}
			if event == "" {
				event = payload.Type
			}
			switch event {
			case "response.output_text.delta":
				sb.WriteString(payload.Delta)
			case "response.completed", "response.incomplete":
				if payload.Response != nil && payload.Response.Usage != nil {
					res.InputTokens, res.OutputTokens = payload.Response.Usage.InputTokens, payload.Response.Usage.OutputTokens
				}
			case "response.failed":
				if payload.Response != nil && payload.Response.Error != nil {
					return payload.Response.Error
				}
				return errors.New("上游返回 response.failed")
			case "error":
				if payload.Error != nil {
					return payload.Error
				}
				if payload.Message != "" {
					return errors.New(payload.Message)
				}
				return errors.New("上游返回错误事件")
			}
			return nil
		})
		res.Text = sb.String()
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	var payload struct {
		responsesResult
		Error *apiError `json:"error"`
	}
	if err := decodeJSON(resp.Body, &payload); err != nil {
		return nil, err
	}
	if payload.Error != nil {
		return nil, payload.Error
	}
	var sb strings.Builder
	for _, item := range payload.Output {
		if item.Type != "message" {
			continue
		}
		for _, part := range item.Content {
			if part.Type == "output_text" {
				sb.WriteString(part.Text)
			}
		}
	}
	res.Text = sb.String()
	if payload.Usage != nil {
		res.InputTokens, res.OutputTokens = payload.Usage.InputTokens, payload.Usage.OutputTokens
	}
	return res, nil
}

var maxTokensLimitRE = regexp.MustCompile(`max_tokens: *(\d+) *> *(\d+)`)

// completeMessages Anthropic Messages；推理强度映射为 adaptive thinking + output_config.effort。
func (c *Client) completeMessages(ctx context.Context, req Request, maxTokens int) (*Result, error) {
	body := map[string]any{
		"model":      req.Model,
		"max_tokens": maxTokens,
		"messages":   []map[string]string{{"role": "user", "content": req.Prompt}},
		"stream":     true,
	}
	if _, ok := anthropicEfforts[req.Effort]; ok {
		body["thinking"] = map[string]any{"type": "adaptive"}
		body["output_config"] = map[string]any{"effort": req.Effort}
	}
	resp, err := c.post(ctx, "/messages", body)
	if err != nil {
		// 上游提示 max_tokens 超过模型上限时，按提示的上限重试一次。
		var herr *HTTPError
		if errors.As(err, &herr) && herr.Status == http.StatusBadRequest {
			if m := maxTokensLimitRE.FindStringSubmatch(herr.Message); m != nil {
				if limit, perr := strconv.Atoi(m[2]); perr == nil && limit > 0 && limit < maxTokens {
					return c.completeMessages(ctx, req, limit)
				}
			}
		}
		return nil, err
	}
	defer resp.Body.Close()

	res := &Result{}
	if isEventStream(resp) {
		var sb strings.Builder
		err := readSSE(resp.Body, func(event string, data string) error {
			var payload struct {
				Type    string `json:"type"`
				Message *struct {
					Usage *anthropicUsage `json:"usage"`
				} `json:"message"`
				Delta *struct {
					Type string `json:"type"`
					Text string `json:"text"`
				} `json:"delta"`
				Usage *anthropicUsage `json:"usage"`
				Error *apiError       `json:"error"`
			}
			if err := json.Unmarshal([]byte(data), &payload); err != nil {
				return nil
			}
			if event == "" {
				event = payload.Type
			}
			switch event {
			case "message_start":
				if payload.Message != nil && payload.Message.Usage != nil {
					res.InputTokens = payload.Message.Usage.InputTokens
				}
			case "content_block_delta":
				if payload.Delta != nil && payload.Delta.Type == "text_delta" {
					sb.WriteString(payload.Delta.Text)
				}
			case "message_delta":
				if payload.Usage != nil {
					res.OutputTokens = payload.Usage.OutputTokens
				}
			case "error":
				if payload.Error != nil {
					return payload.Error
				}
				return errors.New("上游返回错误事件")
			}
			return nil
		})
		res.Text = sb.String()
		if err != nil {
			return nil, err
		}
		return res, nil
	}

	var payload struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage *anthropicUsage `json:"usage"`
		Error *apiError       `json:"error"`
	}
	if err := decodeJSON(resp.Body, &payload); err != nil {
		return nil, err
	}
	if payload.Error != nil {
		return nil, payload.Error
	}
	var sb strings.Builder
	for _, block := range payload.Content {
		if block.Type == "text" {
			sb.WriteString(block.Text)
		}
	}
	res.Text = sb.String()
	if payload.Usage != nil {
		res.InputTokens, res.OutputTokens = payload.Usage.InputTokens, payload.Usage.OutputTokens
	}
	return res, nil
}

// —— 协议数据结构 ——

type chatUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
}

type anthropicUsage struct {
	InputTokens  int64 `json:"input_tokens"`
	OutputTokens int64 `json:"output_tokens"`
}

type responsesResult struct {
	Output []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Usage *anthropicUsage `json:"usage"`
	Error *apiError       `json:"error"`
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
	*t = ""
	return nil
}

// apiError 上游 JSON 中的 error 对象。
type apiError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

func (e *apiError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if e.Type != "" {
		return e.Type
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
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求上游失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		resp.Body.Close()
		return nil, httpError(resp.StatusCode, data)
	}
	return resp, nil
}

func isEventStream(resp *http.Response) bool {
	return strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "text/event-stream")
}

func decodeJSON(r io.Reader, v any) error {
	data, err := io.ReadAll(io.LimitReader(r, 32<<20))
	if err != nil {
		return fmt.Errorf("读取上游响应失败: %w", err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		return fmt.Errorf("上游响应格式无法解析: %s", abbreviate(string(data), 200))
	}
	return nil
}
