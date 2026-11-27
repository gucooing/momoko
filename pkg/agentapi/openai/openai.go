// Package openai OpenAI 兼容的接口（Responses）：给 Codex 系 Agent 用，地址由用户填。
// 只支持 Chat Completions 的服务 Codex 用不了。Send / ListModels 也给专门适配的供应商用。
package openai

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"momoko/pkg/agentapi"
)

// ID 记在供应商上的类型。
const ID = "openai"

// Vendor OpenAI Responses 兼容的接口。
type Vendor struct{}

var _ agentapi.Vendor = Vendor{}

func (Vendor) ID() string          { return ID }
func (Vendor) Protocols() []string { return []string{agentapi.ProtocolOpenAI} }
func (Vendor) BaseURL() string     { return "" }

// Endpoint 用户填的就是 /responses 之前的那一段。
func (Vendor) Endpoint(_ string, c agentapi.Conn) string { return c.BaseURL }

func (Vendor) Models(ctx context.Context, c agentapi.Conn) ([]agentapi.Model, error) {
	return ListModels(ctx, c, c.BaseURL)
}

func (Vendor) Test(ctx context.Context, _ string, c agentapi.Conn, model, message string) (agentapi.Reply, error) {
	return Send(ctx, c, c.BaseURL, model, message)
}

// Send 经 c 往 Responses 接口发一条用户消息，取回模型回的文字和用量。endpoint 是 /responses 之前的那一段。
func Send(ctx context.Context, c agentapi.Conn, endpoint, model, message string) (agentapi.Reply, error) {
	status, body, err := c.Do(ctx, http.MethodPost, agentapi.Join(endpoint, "/responses"), agentapi.Bearer(c.APIKey), map[string]any{
		"model":             model,
		"input":             message,
		"max_output_tokens": agentapi.TestMaxTokens,
	})
	if err != nil {
		return agentapi.Reply{}, err
	}
	if !agentapi.OK(status) {
		return agentapi.Reply{}, agentapi.Reason(status, body)
	}
	// 回复在 output 里 type 为 message 的那几项；reasoning 项是思考过程，不要
	var parsed struct {
		Output []struct {
			Type    string `json:"type"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
		} `json:"output"`
		Usage struct {
			InputTokens  int64 `json:"input_tokens"`
			OutputTokens int64 `json:"output_tokens"`
		} `json:"usage"`
	}
	_ = json.Unmarshal(body, &parsed)
	var texts []string
	for _, o := range parsed.Output {
		if o.Type != "message" {
			continue
		}
		for _, c := range o.Content {
			if c.Type == "output_text" && c.Text != "" {
				texts = append(texts, c.Text)
			}
		}
	}
	return agentapi.Reply{
		Text:         strings.Join(texts, "\n"),
		InputTokens:  parsed.Usage.InputTokens,
		OutputTokens: parsed.Usage.OutputTokens,
	}, nil
}

// ListModels 从 /models 拉模型 id。
func ListModels(ctx context.Context, c agentapi.Conn, endpoint string) ([]agentapi.Model, error) {
	status, body, err := c.Do(ctx, http.MethodGet, agentapi.Join(endpoint, "/models"), agentapi.Bearer(c.APIKey), nil)
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
