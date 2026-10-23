package iqcheck

import (
	"context"
	"fmt"
	"strings"
)

type chatUsage struct {
	PromptTokens     int64 `json:"prompt_tokens"`
	CompletionTokens int64 `json:"completion_tokens"`
}

type chatMessage struct {
	Content textContent `json:"content"`
	Refusal string      `json:"refusal"`
}

type chatResult struct {
	Choices []struct {
		Index        int         `json:"index"`
		Delta        chatMessage `json:"delta"`
		Message      chatMessage `json:"message"`
		FinishReason string      `json:"finish_reason"`
	} `json:"choices"`
	Usage *chatUsage `json:"usage"`
	Error *apiError  `json:"error"`
}

func chatFinishError(reason string) error {
	switch reason {
	case "", "stop":
		return nil
	default:
		return fmt.Errorf("上游回复未正常完成: finish_reason=%s", reason)
	}
}

func (c *Client) completeChat(ctx context.Context, req Request) (*Result, error) {
	state := newStreamState(ctx)
	defer state.close()
	body := map[string]any{
		"model":          req.Model,
		"messages":       []map[string]string{{"role": "user", "content": req.Prompt}},
		"stream":         true,
		"stream_options": map[string]any{"include_usage": true},
	}
	if req.Effort != "" {
		body["reasoning_effort"] = req.Effort
	}
	resp, err := c.post(state.ctx, "/chat/completions", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	res := &Result{}
	applyUsage := func(u *chatUsage) {
		if u != nil {
			res.InputTokens, res.OutputTokens = u.PromptTokens, u.CompletionTokens
		}
	}
	if !isEventStream(resp) {
		var payload chatResult
		if err := decodeJSON(resp.Body, &payload); err != nil {
			return res, err
		}
		applyUsage(payload.Usage)
		if payload.Error != nil {
			return res, payload.Error
		}
		for _, choice := range payload.Choices {
			if choice.Index != 0 {
				continue
			}
			res.Text = string(choice.Message.Content) + choice.Message.Refusal
			return res, chatFinishError(choice.FinishReason)
		}
		return res, nil
	}

	var text strings.Builder
	err = state.read(resp.Body, func(event, data string) error {
		if strings.TrimSpace(data) == "[DONE]" {
			return state.finish()
		}
		var chunk chatResult
		if _, err := decodeEvent(event, data, &chunk); err != nil {
			return err
		}
		applyUsage(chunk.Usage)
		for _, choice := range chunk.Choices {
			if choice.Index != 0 {
				continue
			}
			text.WriteString(string(choice.Delta.Content))
			text.WriteString(choice.Delta.Refusal)
			if err := chatFinishError(choice.FinishReason); err != nil {
				return err
			}
			if choice.FinishReason != "" {
				state.awaitTrailer()
			}
		}
		if state.finished && chunk.Usage != nil {
			return state.finish()
		}
		return nil
	})
	res.Text = text.String()
	return res, err
}
