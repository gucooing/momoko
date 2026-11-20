// Package anthropic Anthropic 兼容的接口（Messages）：给 Claude 系 Agent 用，地址由用户填。
// Send / ListModels 也给专门适配的供应商（它们的 Anthropic 兼容接口）用。
package anthropic

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"momoko/pkg/agentapi"
)

// ID 记在供应商上的类型。
const ID = "anthropic"

// Vendor Anthropic 兼容的接口。
type Vendor struct{}

var _ agentapi.Vendor = Vendor{}

func (Vendor) ID() string          { return ID }
func (Vendor) Protocols() []string { return []string{agentapi.ProtocolAnthropic} }
func (Vendor) BaseURL() string     { return "" }

// Endpoint 用户填的就是 /v1/messages 之前的那一段。
func (Vendor) Endpoint(_ string, c agentapi.Conn) string { return c.BaseURL }

func (Vendor) Models(ctx context.Context, c agentapi.Conn) ([]agentapi.Model, error) {
	return ListModels(ctx, c.BaseURL, c.APIKey)
}

func (Vendor) Test(ctx context.Context, _ string, c agentapi.Conn, model, message string) (agentapi.Reply, error) {
	return Send(ctx, c.BaseURL, c.APIKey, model, message)
}

// header 密钥 x-api-key 和 Bearer 都带：第三方的兼容接口有的只认其中一种。
func header(apiKey string) map[string]string {
	return map[string]string{
		"x-api-key":         apiKey,
		"Authorization":     "Bearer " + apiKey,
		"anthropic-version": "2023-06-01",
	}
}

// Send 往 Messages 接口发一条用户消息，取回模型回的文字和用量。endpoint 是 /v1/messages 之前的那一段。
func Send(ctx context.Context, endpoint, apiKey, model, message string) (agentapi.Reply, error) {
	status, body, err := agentapi.Do(ctx, http.MethodPost, agentapi.Join(endpoint, "/v1/messages"), header(apiKey), map[string]any{
		"model":      model,
		"max_tokens": agentapi.TestMaxTokens,
		"messages":   []map[string]string{{"role": "user", "content": message}},
	})
	if err != nil {
		return agentapi.Reply{}, err
	}
	if !agentapi.OK(status) {
		return agentapi.Reply{}, agentapi.Reason(status, body)
	}
	var parsed struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(body, &parsed)
	var texts []string
	for _, c := range parsed.Content {
		if c.Type == "text" && c.Text != "" {
			texts = append(texts, c.Text)
		}
	}
	return agentapi.Reply{
		Text:         strings.Join(texts, "\n"),
		InputTokens:  parsed.Usage.InputTokens,
		OutputTokens: parsed.Usage.OutputTokens,
	}, nil
}

// ListModels 从 /v1/models 拉模型 id。
func ListModels(ctx context.Context, endpoint, apiKey string) ([]agentapi.Model, error) {
	status, body, err := agentapi.Do(ctx, http.MethodGet, agentapi.Join(endpoint, "/v1/models?limit=1000"), header(apiKey), nil)
	if err != nil {
		return nil, err
	}
	if status == http.StatusNotFound {
		return nil, agentapi.ErrNoModelList
	}
	if !agentapi.OK(status) {
		return nil, agentapi.Reason(status, body)
	}
	var parsed struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, agentapi.ErrBadModelList
	}
	models := make([]agentapi.Model, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		if id := strings.TrimSpace(m.ID); id != "" {
			models = append(models, agentapi.Model{ID: id})
		}
	}
	return models, nil
}
