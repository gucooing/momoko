// Package agentapi 直接调第三方模型接口：测试连接、拉可用模型。
// Anthropic 兼容的走 Messages 接口（给 Claude Code 用），OpenAI 兼容的走 Responses 接口（给 Codex 用）。
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

const (
	Anthropic = "anthropic"
	OpenAI    = "openai"
)

var client = &http.Client{Timeout: 60 * time.Second}

// request 向接口发一个请求，返回状态码和（截断的）响应体。
func request(ctx context.Context, method, vendor, endpoint, apiKey string, body any) (int, []byte, error) {
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
	req.Header.Set("Authorization", "Bearer "+apiKey)
	if vendor == Anthropic {
		req.Header.Set("x-api-key", apiKey)
		req.Header.Set("anthropic-version", "2023-06-01")
	}
	resp, err := client.Do(req)
	if err != nil {
		var urlErr *url.Error
		if errors.As(err, &urlErr) && urlErr.Timeout() {
			return 0, nil, errors.New("接口一直没有响应")
		}
		return 0, nil, errors.New("连不上接口地址")
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	return resp.StatusCode, raw, nil
}

// reason 从接口的错误响应里取一句给人看的原因。
func reason(status int, body []byte) error {
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

// Test 用指定模型发一条最小的消息，看接口地址和密钥通不通。返回耗时；不通时 err 是给人看的原因。
func Test(ctx context.Context, vendor, baseURL, apiKey, model string) (time.Duration, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	started := time.Now()
	var status int
	var body []byte
	var err error
	if vendor == Anthropic {
		status, body, err = request(ctx, http.MethodPost, vendor, baseURL+"/v1/messages", apiKey, map[string]any{
			"model": model, "max_tokens": 16,
			"messages": []map[string]string{{"role": "user", "content": "hi"}},
		})
	} else {
		status, body, err = request(ctx, http.MethodPost, vendor, baseURL+"/responses", apiKey, map[string]any{
			"model": model, "input": "hi", "max_output_tokens": 16,
		})
	}
	elapsed := time.Since(started)
	if err != nil {
		return elapsed, err
	}
	if status < 200 || status >= 300 {
		return elapsed, reason(status, body)
	}
	return elapsed, nil
}

// Models 从接口拉可用模型的 id 列表。
func Models(ctx context.Context, vendor, baseURL, apiKey string) ([]string, error) {
	baseURL = strings.TrimRight(baseURL, "/")
	endpoint := baseURL + "/models"
	if vendor == Anthropic {
		endpoint = baseURL + "/v1/models?limit=1000"
	}
	status, body, err := request(ctx, http.MethodGet, vendor, endpoint, apiKey, nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, errors.New("这个接口没有提供模型列表，手动添加模型 ID")
	}
	if status < 200 || status >= 300 {
		return nil, reason(status, body)
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, errors.New("接口返回的模型列表看不懂，手动添加模型 ID")
	}
	models := make([]string, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if id := strings.TrimSpace(m.ID); id != "" {
			models = append(models, id)
		}
	}
	return models, nil
}
