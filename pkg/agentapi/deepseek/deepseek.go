// Package deepseek DeepSeek 官方接口：地址固定，同一个密钥两种协议都给——/anthropic 是 Anthropic 兼容（给 Claude 系），
// 根地址是 OpenAI Responses（给 Codex 系）。模型列表带最长上下文、最长输出和推理强度；能查账户余额。
package deepseek

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"momoko/pkg/agentapi"
	"momoko/pkg/agentapi/anthropic"
	"momoko/pkg/agentapi/openai"
)

// ID 记在供应商上的类型。
const ID = "deepseek"

const baseURL = "https://api.deepseek.com"

// Vendor DeepSeek 官方接口。
type Vendor struct{}

var (
	_ agentapi.Vendor   = Vendor{}
	_ agentapi.Balancer = Vendor{}
)

func (Vendor) ID() string { return ID }
func (Vendor) Protocols() []string {
	return []string{agentapi.ProtocolAnthropic, agentapi.ProtocolOpenAI}
}
func (Vendor) BaseURL() string { return baseURL }

// Endpoint Anthropic 兼容的在 /anthropic 下，Responses 在根地址下。
func (Vendor) Endpoint(protocol string, _ agentapi.Conn) string {
	if protocol == agentapi.ProtocolAnthropic {
		return baseURL + "/anthropic"
	}
	return baseURL
}

func (v Vendor) Test(ctx context.Context, protocol string, c agentapi.Conn, model, message string) (agentapi.Reply, error) {
	endpoint := v.Endpoint(protocol, c)
	if protocol == agentapi.ProtocolAnthropic {
		return anthropic.Send(ctx, endpoint, c.APIKey, model, message)
	}
	return openai.Send(ctx, endpoint, c.APIKey, model, message)
}

// Models /models 除了 id 还给最长上下文、最长输出和能选的推理强度。
func (Vendor) Models(ctx context.Context, c agentapi.Conn) ([]agentapi.Model, error) {
	status, body, err := agentapi.Do(ctx, http.MethodGet, baseURL+"/models", agentapi.Bearer(c.APIKey), nil)
	if err != nil {
		return nil, err
	}
	if !agentapi.OK(status) {
		return nil, agentapi.Reason(status, body)
	}
	var parsed struct {
		Data []struct {
			ID              string `json:"id"`
			ContextWindow   int64  `json:"context_window"`
			MaxOutputTokens int64  `json:"max_output_tokens"`
			Effort          struct {
				SupportedLevels []string `json:"supported_levels"`
			} `json:"effort"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, agentapi.ErrBadModelList
	}
	models := make([]agentapi.Model, 0, len(parsed.Data))
	for _, m := range parsed.Data {
		id := strings.TrimSpace(m.ID)
		if id == "" {
			continue
		}
		models = append(models, agentapi.Model{
			ID:            id,
			ContextWindow: m.ContextWindow,
			MaxOutput:     m.MaxOutputTokens,
			Levels:        m.Effort.SupportedLevels,
		})
	}
	return models, nil
}

// Balance GET /user/balance：每种币种的总余额、赠送余额和充值余额。
func (Vendor) Balance(ctx context.Context, c agentapi.Conn) (agentapi.Balance, error) {
	status, body, err := agentapi.Do(ctx, http.MethodGet, baseURL+"/user/balance", agentapi.Bearer(c.APIKey), nil)
	if err != nil {
		return agentapi.Balance{}, err
	}
	if !agentapi.OK(status) {
		return agentapi.Balance{}, agentapi.Reason(status, body)
	}
	var parsed struct {
		IsAvailable  bool `json:"is_available"`
		BalanceInfos []struct {
			Currency        string `json:"currency"`
			TotalBalance    string `json:"total_balance"`
			GrantedBalance  string `json:"granted_balance"`
			ToppedUpBalance string `json:"topped_up_balance"`
		} `json:"balance_infos"`
	}
	if err := json.Unmarshal(body, &parsed); err != nil {
		return agentapi.Balance{}, agentapi.Reason(status, body)
	}
	out := agentapi.Balance{Available: parsed.IsAvailable}
	for _, b := range parsed.BalanceInfos {
		out.Items = append(out.Items, agentapi.BalanceItem{
			Currency: b.Currency, Total: b.TotalBalance, Granted: b.GrantedBalance, ToppedUp: b.ToppedUpBalance,
		})
	}
	return out, nil
}
