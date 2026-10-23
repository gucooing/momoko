package iqcheck

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestResponsesSnapshotsAndParts(t *testing.T) {
	srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
		writeSSE(w,
			`data: {"type":"response.output_text.delta","output_index":1,"content_index":0,"delta":"ignored partial"}`,
			`data: {"type":"response.output_text.done","output_index":1,"content_index":0,"text":"B"}`,
			`data: {"type":"response.output_text.delta","output_index":0,"content_index":0,"delta":"A"}`,
			`data: {"type":"response.output_text.done","output_index":0,"content_index":0,"text":"A"}`,
			`data: {"type":"response.content_part.done","output_index":0,"content_index":0,"part":{"type":"output_text","text":"A"}}`,
			`data: {"type":"response.output_item.done","output_index":0,"item":{"type":"message","content":[{"type":"output_text","text":"A"},{"type":"refusal","refusal":"!"}]}}`,
			`data: {"type":"response.reasoning_summary_text.delta","delta":"not answer text"}`,
			`data: {"type":"response.future_event","delta":{"value":"ignored"}}`,
			`data: {"type":"response.completed","response":{"status":"completed","usage":{"input_tokens":8,"output_tokens":4}}}`,
			`data: {invalid after completion}`,
		)
	})
	c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolResponses})
	res, err := c.Complete(context.Background(), Request{Model: "m", Prompt: "hi"})
	if err != nil || res.Text != "A!B" || res.InputTokens != 8 || res.OutputTokens != 4 {
		t.Fatalf("result=%+v error=%v", res, err)
	}
}

func TestMessagesContentAndUsage(t *testing.T) {
	srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
		writeSSE(w,
			`data: {"type":"message_start","message":{"usage":{"input_tokens":8,"output_tokens":1}}}`,
			`data: {"type":"content_block_start","index":1,"content_block":{"type":"text","text":"B"}}`,
			`data: {"type":"content_block_delta","index":1,"delta":{"type":"text_delta","text":"C"}}`,
			`data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":"A"}}`,
			`data: {"type":"content_block_delta","index":2,"delta":{"type":"thinking_delta","thinking":"hidden"}}`,
			`data: {"type":"message_delta","usage":{"output_tokens":3}}`,
			`data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":4}}`,
			`data: {"type":"message_stop"}`,
		)
	})
	c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolMessages})
	res, err := c.Complete(context.Background(), Request{Model: "m", Prompt: "hi"})
	if err != nil || res.Text != "ABC" || res.InputTokens != 8 || res.OutputTokens != 4 {
		t.Fatalf("result=%+v error=%v", res, err)
	}
}

func TestGatewayTimeouts(t *testing.T) {
	for _, stage := range []string{"headers", "body", "heartbeat", "cancel"} {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			started := make(chan struct{})
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				_, _ = io.Copy(io.Discard, r.Body)
				if stage != "headers" {
					writeSSE(w, `data: {"choices":[{"delta":{"content":"partial"}}]}`)
					w.(http.Flusher).Flush()
				}
				close(started)
				ticker := time.NewTicker(10 * time.Millisecond)
				defer ticker.Stop()
				for {
					select {
					case <-r.Context().Done():
						return
					case <-ticker.C:
						if stage == "heartbeat" {
							fmt.Fprint(w, ": ping\n\n")
							w.(http.Flusher).Flush()
						}
					}
				}
			}))
			t.Cleanup(srv.Close)
			c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolChat, Timeout: 300 * time.Millisecond, IdleTimeout: 150 * time.Millisecond})
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if stage == "cancel" {
				go func() { <-started; cancel() }()
			}
			res, err := c.Complete(ctx, Request{Model: "m", Prompt: "hi"})
			want := ErrIdleTimeout
			if stage == "heartbeat" {
				want = context.DeadlineExceeded
			}
			if stage == "cancel" {
				want = context.Canceled
			}
			if !errors.Is(err, want) {
				t.Fatalf("result=%+v error=%v; want %v", res, err, want)
			}
			if stage == "body" || stage == "heartbeat" {
				if res == nil || res.Text != "partial" {
					t.Fatalf("lost partial response: %+v", res)
				}
			}
		})
	}
}

func TestCompleteRejectsFailedOrEmptyResponses(t *testing.T) {
	cases := []struct {
		name       string
		protocol   Protocol
		body, want string
		stream     bool
	}{
		{"responses failed json", ProtocolResponses, `{"status":"failed","error":{"message":"quota"}}`, "quota", false},
		{"responses incomplete json", ProtocolResponses, `{"status":"incomplete","incomplete_details":{"reason":"max_output_tokens"},"output":[{"type":"message","content":[{"type":"output_text","text":"partial"}]}]}`, "max_output_tokens", false},
		{"responses queued json", ProtocolResponses, `{"status":"queued"}`, "queued", false},
		{"messages max tokens", ProtocolMessages, `{"content":[{"type":"text","text":"partial"}],"stop_reason":"max_tokens"}`, "max_tokens", false},
		{"messages streaming max tokens", ProtocolMessages, "data: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"max_tokens\"}}\n\n", "max_tokens", true},
		{"chat string error", ProtocolChat, "data: {\"error\":\"rate limited\"}\n\n", "rate limited", true},
		{"responses flat error", ProtocolResponses, "event: error\ndata: {\"type\":\"error\",\"message\":\"overloaded\",\"code\":\"quota\"}\n\n", "overloaded", true},
		{"messages error", ProtocolMessages, "event: error\ndata: {\"type\":\"error\",\"error\":{\"type\":\"overloaded_error\",\"message\":\"busy\"}}\n\n", "busy", true},
		{"chat empty", ProtocolChat, "data: [DONE]\n\n", "未返回回复文本", true},
		{"responses empty", ProtocolResponses, `{"status":"completed","output":[]}`, "未返回回复文本", false},
		{"messages empty", ProtocolMessages, `{"content":[]}`, "未返回回复文本", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
				if tc.stream {
					w.Header().Set("Content-Type", "text/event-stream")
				} else {
					w.Header().Set("Content-Type", "application/json")
				}
				fmt.Fprint(w, tc.body)
			})
			c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: tc.protocol})
			_, err := c.Complete(context.Background(), Request{Model: "m", Prompt: "hi"})
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error=%v; want %q", err, tc.want)
			}
		})
	}
}

func TestMessagesRetriesOnlyOnce(t *testing.T) {
	srv, requests := newServer(t, func(w http.ResponseWriter, _ *http.Request, body map[string]any) {
		limit := int(body["max_tokens"].(float64))
		writeJSON(w, http.StatusBadRequest, map[string]any{"error": map[string]any{"message": fmt.Sprintf("max_tokens: %d > %d", limit, limit/2)}})
	})
	c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolMessages})
	_, err := c.Complete(context.Background(), Request{Model: "m", Prompt: "hi"})
	if err == nil || len(requests()) != 2 {
		t.Fatalf("requests=%d error=%v", len(requests()), err)
	}
}

func TestHTTPErrorDoesNotWaitForEOF(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusTooManyRequests)
		fmt.Fprint(w, `{"error":{"message":"rate limited"}}`)
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	t.Cleanup(srv.Close)
	c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolChat, Timeout: time.Second})
	_, err := c.Complete(context.Background(), Request{Model: "m", Prompt: "hi"})
	var httpErr *HTTPError
	if !errors.As(err, &httpErr) || httpErr.Status != http.StatusTooManyRequests || httpErr.Message != "rate limited" {
		t.Fatalf("expected immediate HTTP error, got %v", err)
	}
}

// A relay can finish a response while leaving the HTTP body open indefinitely.
func TestGatewayCompletion(t *testing.T) {
	cases := []struct {
		name          string
		protocol      Protocol
		contentType   string
		body          string
		text          string
		input, output int64
		errText       string
	}{
		{"chat finish without done", ProtocolChat, "text/event-stream", "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\n", "ok", 0, 0, ""},
		{"chat trailing usage", ProtocolChat, "text/event-stream", "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"},\"finish_reason\":\"stop\"}]}\n\ndata: {\"choices\":[],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2}}\n\n", "ok", 3, 2, ""},
		{"chat length", ProtocolChat, "text/event-stream", "data: {\"choices\":[{\"delta\":{\"content\":\"partial\"},\"finish_reason\":\"length\"}]}\n\n", "partial", 0, 0, "length"},
		{"responses snapshot", ProtocolResponses, "text/event-stream", "event: message\ndata: {\"type\":\"response.completed\",\"response\":{\"status\":\"completed\",\"output\":[{\"type\":\"message\",\"content\":[{\"type\":\"output_text\",\"text\":\"ok\"}]}],\"usage\":{\"input_tokens\":3,\"output_tokens\":2}}}\n\n", "ok", 3, 2, ""},
		{"responses incomplete", ProtocolResponses, "text/event-stream", "data: {\"type\":\"response.output_text.delta\",\"delta\":\"partial\"}\n\ndata: {\"type\":\"response.incomplete\",\"response\":{\"incomplete_details\":{\"reason\":\"max_output_tokens\"}}}\n\n", "partial", 0, 0, "max_output_tokens"},
		{"messages stop reason", ProtocolMessages, "text/event-stream", "data: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"ok\"}}\n\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"input_tokens\":3,\"output_tokens\":2}}\n\n", "ok", 3, 2, ""},
		{"json without eof", ProtocolChat, "application/json", `{"choices":[{"message":{"content":"ok"},"finish_reason":"stop"}],"usage":{"prompt_tokens":3,"completion_tokens":2}}`, "ok", 3, 2, ""},
		{"malformed event", ProtocolResponses, "text/event-stream", "data: {broken}\n\n", "", 0, 0, "解析"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", tc.contentType)
				fmt.Fprint(w, tc.body)
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			}))
			t.Cleanup(srv.Close)
			c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: tc.protocol})
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
			defer cancel()
			res, err := c.Complete(ctx, Request{Model: "m", Prompt: "hi"})
			if tc.errText == "" && err != nil || tc.errText != "" && (err == nil || !strings.Contains(err.Error(), tc.errText)) {
				t.Fatalf("result=%+v error=%v; want error containing %q", res, err, tc.errText)
			}
			if tc.text != "" && (res == nil || res.Text != tc.text || res.InputTokens != tc.input || res.OutputTokens != tc.output) {
				t.Fatalf("result=%+v; want text=%q usage=%d/%d", res, tc.text, tc.input, tc.output)
			}
		})
	}
}

func TestTruncatedStreamFails(t *testing.T) {
	for _, protocol := range []Protocol{ProtocolChat, ProtocolResponses, ProtocolMessages} {
		t.Run(string(protocol), func(t *testing.T) {
			srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
				chunks := map[Protocol]string{
					ProtocolChat:      `data: {"choices":[{"delta":{"content":"partial"}}]}`,
					ProtocolResponses: `data: {"type":"response.output_text.delta","delta":"partial"}`,
					ProtocolMessages:  `data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"partial"}}`,
				}
				writeSSE(w, chunks[protocol])
			})
			c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: protocol})
			res, err := c.Complete(context.Background(), Request{Model: "m", Prompt: "hi"})
			if err == nil || res == nil || res.Text != "partial" {
				t.Fatalf("truncated response must retain partial text and fail: result=%+v error=%v", res, err)
			}
		})
	}
}
