package iqcheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

var maxTokensLimitRE = regexp.MustCompile(`max_tokens: *(\d+) *> *(\d+)`)

type messageBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type messagesResult struct {
	Content    []messageBlock  `json:"content"`
	StopReason string          `json:"stop_reason"`
	Usage      *anthropicUsage `json:"usage"`
	Error      *apiError       `json:"error"`
}

func messageStopError(reason string) error {
	switch reason {
	case "", "end_turn", "stop_sequence", "refusal":
		return nil
	default:
		return fmt.Errorf("上游回复未正常完成: stop_reason=%s", reason)
	}
}

func (c *Client) completeMessages(ctx context.Context, req Request) (*Result, error) {
	state := newStreamState(ctx)
	defer state.close()
	body := map[string]any{
		"model": req.Model, "max_tokens": defaultMaxTokens,
		"messages": []map[string]string{{"role": "user", "content": req.Prompt}},
		"stream":   true,
	}
	if _, ok := anthropicEfforts[req.Effort]; ok {
		body["thinking"] = map[string]any{"type": "adaptive"}
		body["output_config"] = map[string]any{"effort": req.Effort}
	}
	resp, err := c.post(state.ctx, "/messages", body)
	// Retry exactly once, under the same request deadline, only when the server
	// rejects max_tokens and supplies a smaller valid limit.
	var herr *HTTPError
	if errors.As(err, &herr) && herr.Status == http.StatusBadRequest {
		if match := maxTokensLimitRE.FindStringSubmatch(herr.Message); match != nil {
			if limit, parseErr := strconv.Atoi(match[2]); parseErr == nil && limit > 0 && limit < defaultMaxTokens {
				body["max_tokens"] = limit
				resp, err = c.post(state.ctx, "/messages", body)
			}
		}
	}
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	res := &Result{}
	text := streamText{}
	applyMessage := func(m *messagesResult) {
		m.Usage.apply(res)
		for i, block := range m.Content {
			if block.Type == "text" {
				text.set(i, 0, block.Text)
			}
		}
	}
	if !isEventStream(resp) {
		var payload messagesResult
		if err := decodeJSON(resp.Body, &payload); err != nil {
			return res, err
		}
		applyMessage(&payload)
		res.Text = text.String()
		if payload.Error != nil {
			return res, payload.Error
		}
		return res, messageStopError(payload.StopReason)
	}

	err = state.read(resp.Body, func(event, data string) error {
		if strings.TrimSpace(data) == "[DONE]" {
			return fmt.Errorf("上游缺少 message_stop 事件: %w", io.ErrUnexpectedEOF)
		}
		var payload struct {
			Index        int             `json:"index"`
			Message      *messagesResult `json:"message"`
			ContentBlock *messageBlock   `json:"content_block"`
			Delta        struct {
				Type       string `json:"type"`
				Text       string `json:"text"`
				StopReason string `json:"stop_reason"`
			} `json:"delta"`
			Usage *anthropicUsage `json:"usage"`
		}
		kind, err := decodeEvent(event, data, &payload,
			"message_start", "content_block_start", "content_block_delta", "message_delta", "message_stop")
		if err != nil {
			return err
		}
		switch kind {
		case "message_start":
			if payload.Message != nil {
				applyMessage(payload.Message)
			}
		case "content_block_start":
			if block := payload.ContentBlock; block != nil && block.Type == "text" {
				text.set(payload.Index, 0, block.Text)
			}
		case "content_block_delta":
			if payload.Delta.Type == "text_delta" {
				text.append(payload.Index, 0, payload.Delta.Text)
			}
		case "message_delta":
			payload.Usage.apply(res)
			if err := messageStopError(payload.Delta.StopReason); err != nil {
				return err
			}
			if payload.Delta.StopReason != "" {
				state.awaitTrailer()
			}
		case "message_stop":
			return state.finish()
		}
		return nil
	})
	res.Text = text.String()
	return res, err
}
