package iqcheck

import (
	"context"
	"fmt"
	"io"
	"strings"
)

type responsePart struct {
	Type    string `json:"type"`
	Text    string `json:"text"`
	Refusal string `json:"refusal"`
}

func (p responsePart) text() string {
	switch p.Type {
	case "output_text":
		return p.Text
	case "refusal":
		return p.Refusal
	default:
		return ""
	}
}

type responseItem struct {
	Type    string         `json:"type"`
	Content []responsePart `json:"content"`
}

type responsesResult struct {
	Status            string          `json:"status"`
	Output            []responseItem  `json:"output"`
	Usage             *anthropicUsage `json:"usage"`
	Error             *apiError       `json:"error"`
	IncompleteDetails *struct {
		Reason string `json:"reason"`
	} `json:"incomplete_details"`
}

func (r *responsesResult) completionError() error {
	if r.Error != nil {
		return r.Error
	}
	switch r.Status {
	case "", "completed":
		return nil
	case "incomplete":
		reason := ""
		if r.IncompleteDetails != nil {
			reason = r.IncompleteDetails.Reason
		}
		return fmt.Errorf("上游回复不完整: response.incomplete (%s)", reason)
	default:
		return fmt.Errorf("上游回复未完成: response.%s", r.Status)
	}
}

func setResponseItem(text streamText, index int, item responseItem) {
	if item.Type != "message" {
		return
	}
	text.clearItem(index)
	for i, part := range item.Content {
		text.set(index, i, part.text())
	}
}

func (c *Client) completeResponses(ctx context.Context, req Request) (*Result, error) {
	state := newStreamState(ctx)
	defer state.close()
	body := map[string]any{"model": req.Model, "input": req.Prompt, "stream": true}
	if req.Effort != "" {
		body["reasoning"] = map[string]any{"effort": req.Effort}
	}
	resp, err := c.post(state.ctx, "/responses", body)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	res := &Result{}
	text := streamText{}
	applyResponse := func(r *responsesResult) {
		r.Usage.apply(res)
		// Some relays omit the final output snapshot. Keep accumulated text then.
		if len(r.Output) > 0 {
			clear(text)
			for i, item := range r.Output {
				setResponseItem(text, i, item)
			}
		}
	}
	if !isEventStream(resp) {
		var payload responsesResult
		if err := decodeJSON(resp.Body, &payload); err != nil {
			return res, err
		}
		applyResponse(&payload)
		res.Text = text.String()
		return res, payload.completionError()
	}

	err = state.read(resp.Body, func(event, data string) error {
		if strings.TrimSpace(data) == "[DONE]" {
			return fmt.Errorf("上游缺少 response.completed 事件: %w", io.ErrUnexpectedEOF)
		}
		var payload struct {
			Delta        string           `json:"delta"`
			Text         string           `json:"text"`
			Refusal      string           `json:"refusal"`
			OutputIndex  int              `json:"output_index"`
			ContentIndex int              `json:"content_index"`
			Part         *responsePart    `json:"part"`
			Item         *responseItem    `json:"item"`
			Response     *responsesResult `json:"response"`
		}
		kind, err := decodeEvent(event, data, &payload,
			"response.output_text.delta", "response.refusal.delta", "response.output_text.done", "response.refusal.done",
			"response.content_part.added", "response.content_part.done", "response.output_item.added", "response.output_item.done",
			"response.completed", "response.failed", "response.incomplete", "response.cancelled", "response.canceled", "response.done")
		if err != nil {
			return err
		}
		switch kind {
		case "response.output_text.delta", "response.refusal.delta":
			text.append(payload.OutputIndex, payload.ContentIndex, payload.Delta)
		case "response.output_text.done":
			text.set(payload.OutputIndex, payload.ContentIndex, payload.Text)
		case "response.refusal.done":
			text.set(payload.OutputIndex, payload.ContentIndex, payload.Refusal)
		case "response.content_part.added", "response.content_part.done":
			if payload.Part != nil {
				text.set(payload.OutputIndex, payload.ContentIndex, payload.Part.text())
			}
		case "response.output_item.added", "response.output_item.done":
			if payload.Item != nil {
				setResponseItem(text, payload.OutputIndex, *payload.Item)
			}
		case "response.completed", "response.failed", "response.incomplete", "response.cancelled", "response.canceled", "response.done":
			r := payload.Response
			if r == nil {
				return fmt.Errorf("上游 %s 事件缺少 response", kind)
			}
			applyResponse(r)
			if kind != "response.done" && kind != "response.completed" {
				r.Status = strings.TrimPrefix(kind, "response.")
			}
			if kind == "response.done" && r.Status == "" {
				return fmt.Errorf("上游 response.done 事件缺少 status")
			}
			if err := r.completionError(); err != nil {
				return err
			}
			return state.finish()
		}
		return nil
	})
	res.Text = text.String()
	return res, err
}
