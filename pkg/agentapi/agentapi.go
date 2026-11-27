// Package agentapi 模型供应商。每一种在自己的子包里实现 Vendor：通用的兼容接口（anthropic、openai），
// 和对某一家的专门适配（deepseek，以后还会有别家）。这里只放各家共用的东西：供应商的接口、模型和回复的结构、
// 发请求和读错误的办法。biz 只通过 Vendor（和可选的 Balancer）问。
package agentapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Agent 接第三方接口用的两种协议。
const (
	// ProtocolAnthropic Anthropic Messages 接口，给 Claude 系 Agent。
	ProtocolAnthropic = "anthropic"
	// ProtocolOpenAI OpenAI Responses 接口，给 Codex 系 Agent。
	ProtocolOpenAI = "openai"
)

// Conn 连到供应商用的地址、密钥和 HTTP 客户端。
type Conn struct {
	// BaseURL 用户填的地址；官方地址固定的供应商用不到它。
	BaseURL string
	APIKey  string
	// HTTP 发请求用的客户端（供应商配了代理时是经代理的那个）；空 = 直接联网。
	HTTP *http.Client
}

// Model 接口给的一个模型；接口没给的项为零值。
type Model struct {
	ID            string
	ContextWindow int64
	MaxOutput     int64
	// Levels 能选的推理强度，原样就是 API 参数值。
	Levels []string
}

// Reply 发一条测试消息拿回来的结果。
type Reply struct {
	// Text 模型回的文字（不含思考过程）
	Text         string
	InputTokens  int64
	OutputTokens int64
}

// Balance 账户余额。
type Balance struct {
	// Available 余额够不够继续调用
	Available bool
	// Items 每种币种一条
	Items []BalanceItem
}

// BalanceItem 一种币种的余额；金额是接口给的十进制字符串，不经过浮点数。
type BalanceItem struct {
	Currency string
	Total    string
	Granted  string
	ToppedUp string
}

// Vendor 一种模型供应商。
type Vendor interface {
	// ID 记在供应商上的类型：anthropic | openai | deepseek …
	ID() string
	// Protocols 能给哪些协议的 Agent 用（ProtocolAnthropic / ProtocolOpenAI）。
	Protocols() []string
	// BaseURL 官方的接口地址；空表示由用户填。
	BaseURL() string
	// Endpoint 给某个协议的 Agent 用的接口地址：交给 Agent 的、测试时请求的都是它。
	Endpoint(protocol string, c Conn) string
	// Models 拉可用模型。
	Models(ctx context.Context, c Conn) ([]Model, error)
	// Test 用某个协议发一条用户消息。
	Test(ctx context.Context, protocol string, c Conn, model, message string) (Reply, error)
}

// Balancer 能查余额的供应商。
type Balancer interface {
	Balance(ctx context.Context, c Conn) (Balance, error)
}

// Serves 这个供应商能不能给这个协议的 Agent 用。
func Serves(v Vendor, protocol string) bool {
	if protocol == "" {
		return false
	}
	for _, p := range v.Protocols() {
		if p == protocol {
			return true
		}
	}
	return false
}

// TestMaxTokens 测试消息最多回多少 token：够推理模型想一想再回一段话。
const TestMaxTokens = 4096

// direct 不走代理时发请求用的客户端。
var direct = &http.Client{Timeout: 3 * time.Minute}

// Do 经 c.HTTP 发一个请求（body 不为空时是 JSON），返回状态码和（截断的）响应体；连不上、超时时 err 是给人看的原因。
func (c Conn) Do(ctx context.Context, method, endpoint string, header map[string]string, body any) (int, []byte, error) {
	var reader io.Reader
	if body != nil {
		raw, err := json.Marshal(body)
		if err != nil {
			return 0, nil, err
		}
		reader = bytes.NewReader(raw)
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return 0, nil, errors.New("接口地址不对")
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range header {
		req.Header.Set(k, v)
	}
	client := c.HTTP
	if client == nil {
		client = direct
	}
	resp, err := client.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.Is(err, context.DeadlineExceeded) || (errors.As(err, &urlErr) && urlErr.Timeout()) {
			return 0, nil, errors.New("接口一直没有响应")
		}
		return 0, nil, errors.New("连不上接口地址")
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	return resp.StatusCode, raw, nil
}

// OK 状态码是不是 2xx。
func OK(status int) bool { return status >= 200 && status < 300 }

// Bearer 只带 Authorization: Bearer 的请求头。
func Bearer(apiKey string) map[string]string {
	return map[string]string{"Authorization": "Bearer " + apiKey}
}

// Join 接口地址后面接一段路径（地址末尾的 / 去掉）。
func Join(base, path string) string {
	return strings.TrimRight(base, "/") + path
}

// Reason 从接口的错误响应里取一句给人看的原因。
func Reason(status int, body []byte) error {
	var parsed struct {
		Error json.RawMessage `json:"error"`
		Msg   string          `json:"message"`
	}
	msg := ""
	if json.Unmarshal(body, &parsed) == nil {
		var obj struct {
			Message string `json:"message"`
		}
		var text string
		switch {
		case json.Unmarshal(parsed.Error, &obj) == nil && obj.Message != "":
			msg = obj.Message
		case json.Unmarshal(parsed.Error, &text) == nil && text != "":
			msg = text
		case parsed.Msg != "":
			msg = parsed.Msg
		}
	}
	if r := []rune(msg); len(r) > 160 {
		msg = string(r[:160]) + "…"
	}
	switch {
	case status == http.StatusUnauthorized || status == http.StatusForbidden:
		if msg == "" {
			return errors.New("密钥无效")
		}
		return errors.New("密钥无效：" + msg)
	case msg != "":
		return fmt.Errorf("接口返回 %d：%s", status, msg)
	default:
		return fmt.Errorf("接口返回 %d", status)
	}
}

// ErrNoModelList 接口没有模型列表。
var ErrNoModelList = errors.New("这个接口没有提供模型列表，手动添加模型 ID")

// ErrBadModelList 接口的模型列表读不懂。
var ErrBadModelList = errors.New("接口返回的模型列表看不懂，手动添加模型 ID")
