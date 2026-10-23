package iqcheck

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

type capturedRequest struct {
	Method string
	Path   string
	Query  string
	Header http.Header
	Body   map[string]any
}

// newServer 启动测试服务器，记录每个请求后交给 handler 应答。
func newServer(t *testing.T, handler func(w http.ResponseWriter, r *http.Request, body map[string]any)) (*httptest.Server, func() []capturedRequest) {
	t.Helper()
	var mu sync.Mutex
	var reqs []capturedRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var body map[string]any
		if len(raw) > 0 {
			_ = json.Unmarshal(raw, &body)
		}
		mu.Lock()
		reqs = append(reqs, capturedRequest{Method: r.Method, Path: r.URL.Path, Query: r.URL.RawQuery, Header: r.Header.Clone(), Body: body})
		mu.Unlock()
		handler(w, r, body)
	}))
	t.Cleanup(srv.Close)
	return srv, func() []capturedRequest {
		mu.Lock()
		defer mu.Unlock()
		return append([]capturedRequest(nil), reqs...)
	}
}

func writeSSE(w http.ResponseWriter, events ...string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	for _, e := range events {
		fmt.Fprint(w, e, "\n\n")
	}
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func mustClient(t *testing.T, cfg Config) *Client {
	t.Helper()
	c, err := NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func TestNormalizeEndpoint(t *testing.T) {
	cases := map[string]string{
		"https://api.example.com":        "https://api.example.com/v1",
		"https://api.example.com/":       "https://api.example.com/v1",
		" https://api.example.com/v1/ ":  "https://api.example.com/v1",
		"https://x.com/openai/V1":        "https://x.com/openai/V1",
		"http://127.0.0.1:3000/api?x=1":  "http://127.0.0.1:3000/api/v1",
		"https://relay.example.com/v1/x": "https://relay.example.com/v1/x/v1",
	}
	for in, want := range cases {
		got, err := NormalizeEndpoint(in)
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if got != want {
			t.Fatalf("%q = %q, 期望 %q", in, got, want)
		}
	}
	for _, bad := range []string{"", "api.example.com", "ftp://x.com", "https://"} {
		if _, err := NormalizeEndpoint(bad); err == nil {
			t.Fatalf("%q 应报错", bad)
		}
	}
}

func TestMaskAPIKeyAndEffort(t *testing.T) {
	if got := MaskAPIKey("sk-1234567890abcdef"); got != "sk-1****cdef" {
		t.Fatalf("MaskAPIKey = %q", got)
	}
	if got := MaskAPIKey("short"); got != "********" {
		t.Fatalf("MaskAPIKey(short) = %q", got)
	}
	for _, e := range append([]string{""}, Efforts...) {
		if !ValidEffort(e) {
			t.Fatalf("%q 应合法", e)
		}
	}
	if ValidEffort("ultra") {
		t.Fatal("ultra 不应合法")
	}
}

func TestCompleteChatStream(t *testing.T) {
	srv, requests := newServer(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
		writeSSE(w,
			`data: {"choices":[{"delta":{"content":"Hel"}}]}`,
			`: keepalive`,
			`data: {"choices":[{"delta":{"content":"lo"}}]}`,
			`data: {"choices":[],"usage":{"prompt_tokens":3,"completion_tokens":2}}`,
			`data: [DONE]`,
		)
	})
	c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolChat})
	res, err := c.Complete(context.Background(), Request{Model: "m", Effort: "high", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "Hello" || res.InputTokens != 3 || res.OutputTokens != 2 {
		t.Fatalf("结果 = %+v", res)
	}
	req := requests()[0]
	if req.Path != "/v1/chat/completions" || req.Method != http.MethodPost {
		t.Fatalf("请求 = %s %s", req.Method, req.Path)
	}
	if req.Header.Get("Authorization") != "Bearer k" || req.Header.Get("User-Agent") != DefaultUserAgent {
		t.Fatalf("请求头 = %v", req.Header)
	}
	if req.Body["model"] != "m" || req.Body["reasoning_effort"] != "high" || req.Body["stream"] != true {
		t.Fatalf("请求体 = %v", req.Body)
	}
}

func TestCompleteChatJSONAndErrors(t *testing.T) {
	srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request, body map[string]any) {
		switch body["model"] {
		case "json":
			writeJSON(w, http.StatusOK, map[string]any{
				"choices": []any{map[string]any{"message": map[string]any{"content": []any{
					map[string]any{"type": "text", "text": "A"}, map[string]any{"type": "text", "text": "B"},
				}}}},
				"usage": map[string]any{"prompt_tokens": 1, "completion_tokens": 1},
			})
		case "unauthorized":
			writeJSON(w, http.StatusUnauthorized, map[string]any{"error": map[string]any{"message": "bad key", "type": "auth"}})
		case "stream-error":
			writeSSE(w, `data: {"error":{"message":"boom"}}`)
		}
	})
	c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolChat, UserAgent: "custom-ua"})

	res, err := c.Complete(context.Background(), Request{Model: "json", Prompt: "hi"})
	if err != nil || res.Text != "AB" || res.InputTokens != 1 {
		t.Fatalf("json 结果 = %+v, err = %v", res, err)
	}

	_, err = c.Complete(context.Background(), Request{Model: "unauthorized", Prompt: "hi"})
	var herr *HTTPError
	if err == nil || !strings.Contains(err.Error(), "401") || !strings.Contains(err.Error(), "bad key") {
		t.Fatalf("401 错误 = %v", err)
	}
	if !asHTTPError(err, &herr) || herr.Status != http.StatusUnauthorized {
		t.Fatalf("应为 HTTPError: %v", err)
	}

	if _, err = c.Complete(context.Background(), Request{Model: "stream-error", Prompt: "hi"}); err == nil || err.Error() != "boom" {
		t.Fatalf("流内错误 = %v", err)
	}
}

func asHTTPError(err error, target **HTTPError) bool {
	h, ok := err.(*HTTPError)
	if ok {
		*target = h
	}
	return ok
}

func TestCompleteResponsesStream(t *testing.T) {
	srv, requests := newServer(t, func(w http.ResponseWriter, _ *http.Request, body map[string]any) {
		if body["model"] == "no-event-line" {
			writeSSE(w,
				`data: {"type":"response.output_text.delta","delta":"Hi"}`,
				`data: {"type":"response.completed","response":{"usage":{"input_tokens":5,"output_tokens":1}}}`,
			)
			return
		}
		writeSSE(w,
			"event: response.created\ndata: {\"type\":\"response.created\"}",
			"event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"Hel\"}",
			"event: response.output_text.delta\ndata: {\"type\":\"response.output_text.delta\",\"delta\":\"lo\"}",
			"event: response.completed\ndata: {\"type\":\"response.completed\",\"response\":{\"usage\":{\"input_tokens\":5,\"output_tokens\":2}}}",
		)
	})
	c := mustClient(t, Config{Endpoint: srv.URL + "/v1", APIKey: "k", Protocol: ProtocolResponses})
	res, err := c.Complete(context.Background(), Request{Model: "m", Effort: "low", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "Hello" || res.InputTokens != 5 || res.OutputTokens != 2 {
		t.Fatalf("结果 = %+v", res)
	}
	req := requests()[0]
	if req.Path != "/v1/responses" || req.Body["input"] != "hi" {
		t.Fatalf("请求 = %s %v", req.Path, req.Body)
	}
	if reasoning, _ := req.Body["reasoning"].(map[string]any); reasoning["effort"] != "low" {
		t.Fatalf("reasoning = %v", req.Body["reasoning"])
	}

	res, err = c.Complete(context.Background(), Request{Model: "no-event-line", Prompt: "hi"})
	if err != nil || res.Text != "Hi" || res.InputTokens != 5 {
		t.Fatalf("无 event 行结果 = %+v, err = %v", res, err)
	}
}

func TestCompleteResponsesJSONAndFailed(t *testing.T) {
	srv, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request, body map[string]any) {
		if body["model"] == "failed" {
			writeSSE(w, "event: response.failed\ndata: {\"type\":\"response.failed\",\"response\":{\"error\":{\"message\":\"quota\"}}}")
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"output": []any{
				map[string]any{"type": "reasoning"},
				map[string]any{"type": "message", "content": []any{map[string]any{"type": "output_text", "text": "ok"}}},
			},
			"usage": map[string]any{"input_tokens": 2, "output_tokens": 1},
		})
	})
	c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolResponses})
	res, err := c.Complete(context.Background(), Request{Model: "m", Prompt: "hi"})
	if err != nil || res.Text != "ok" || res.OutputTokens != 1 {
		t.Fatalf("json 结果 = %+v, err = %v", res, err)
	}
	if _, err := c.Complete(context.Background(), Request{Model: "failed", Prompt: "hi"}); err == nil || err.Error() != "quota" {
		t.Fatalf("failed 事件错误 = %v", err)
	}
}

func TestCompleteMessagesStream(t *testing.T) {
	srv, requests := newServer(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
		writeSSE(w,
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"usage\":{\"input_tokens\":7}}}",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\"}",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"thinking_delta\",\"thinking\":\"...\"}}",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"A\"}}",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"B\"}}",
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":2}}",
			"event: message_stop\ndata: {\"type\":\"message_stop\"}",
		)
	})
	c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolMessages})
	res, err := c.Complete(context.Background(), Request{Model: "claude", Effort: "medium", Prompt: "hi"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Text != "AB" || res.InputTokens != 7 || res.OutputTokens != 2 {
		t.Fatalf("结果 = %+v", res)
	}
	req := requests()[0]
	if req.Path != "/v1/messages" || req.Header.Get("x-api-key") != "k" || req.Header.Get("anthropic-version") != anthropicVersion {
		t.Fatalf("请求 = %s %v", req.Path, req.Header)
	}
	if req.Body["max_tokens"] != float64(defaultMaxTokens) {
		t.Fatalf("max_tokens = %v", req.Body["max_tokens"])
	}
	if thinking, _ := req.Body["thinking"].(map[string]any); thinking["type"] != "adaptive" {
		t.Fatalf("thinking = %v", req.Body["thinking"])
	}
	if cfg, _ := req.Body["output_config"].(map[string]any); cfg["effort"] != "medium" {
		t.Fatalf("output_config = %v", req.Body["output_config"])
	}
}

func TestCompleteMessagesMaxTokensRetryAndNoEffort(t *testing.T) {
	srv, requests := newServer(t, func(w http.ResponseWriter, _ *http.Request, body map[string]any) {
		if body["max_tokens"] == float64(defaultMaxTokens) {
			writeJSON(w, http.StatusBadRequest, map[string]any{"type": "error", "error": map[string]any{
				"type":    "invalid_request_error",
				"message": "max_tokens: 16000 > 8192, which is the maximum allowed number of output tokens for claude-x",
			}})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{
			"content": []any{map[string]any{"type": "text", "text": "ok"}},
			"usage":   map[string]any{"input_tokens": 1, "output_tokens": 1},
		})
	})
	c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolMessages})
	res, err := c.Complete(context.Background(), Request{Model: "claude-x", Effort: "none", Prompt: "hi"})
	if err != nil || res.Text != "ok" {
		t.Fatalf("结果 = %+v, err = %v", res, err)
	}
	reqs := requests()
	if len(reqs) != 2 || reqs[1].Body["max_tokens"] != float64(8192) {
		t.Fatalf("应按上游上限重试一次: %d 次, 第二次 max_tokens = %v", len(reqs), reqs[len(reqs)-1].Body["max_tokens"])
	}
	for _, r := range reqs {
		if _, ok := r.Body["thinking"]; ok {
			t.Fatalf("effort=none 不应发送 thinking: %v", r.Body)
		}
		if _, ok := r.Body["output_config"]; ok {
			t.Fatalf("effort=none 不应发送 output_config: %v", r.Body)
		}
	}
}

func TestListModels(t *testing.T) {
	srv, requests := newServer(t, func(w http.ResponseWriter, r *http.Request, _ map[string]any) {
		if r.URL.Query().Get("after_id") == "m2" {
			writeJSON(w, http.StatusOK, map[string]any{"data": []any{map[string]any{"id": "m3"}, map[string]any{"id": "m1"}}, "has_more": false})
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"data": []any{map[string]any{"id": "m2"}, map[string]any{"id": "m1"}}, "has_more": true, "last_id": "m2"})
	})
	c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolMessages})
	models, err := c.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(models, ",") != "m1,m2,m3" {
		t.Fatalf("models = %v", models)
	}
	reqs := requests()
	if len(reqs) != 2 || reqs[0].Path != "/v1/models" || reqs[0].Method != http.MethodGet {
		t.Fatalf("请求 = %+v", reqs)
	}

	bad, _ := newServer(t, func(w http.ResponseWriter, _ *http.Request, _ map[string]any) {
		writeJSON(w, http.StatusForbidden, map[string]any{"error": "forbidden"})
	})
	c = mustClient(t, Config{Endpoint: bad.URL, APIKey: "k", Protocol: ProtocolChat})
	if _, err := c.ListModels(context.Background()); err == nil || !strings.Contains(err.Error(), "forbidden") {
		t.Fatalf("403 错误 = %v", err)
	}
}

func TestReadSSE(t *testing.T) {
	src := "event: a\ndata: 1\ndata: 2\n\n: comment\r\ndata: x\r\n\r\ndata: last"
	var got []string
	err := readSSE(strings.NewReader(src), func(event, data string) error {
		got = append(got, event+"|"+data)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"a|1\n2", "|x", "|last"}
	if strings.Join(got, ";") != strings.Join(want, ";") {
		t.Fatalf("events = %q, 期望 %q", got, want)
	}
}

func TestRandomSlug(t *testing.T) {
	a, b := RandomSlug(12), RandomSlug(12)
	if len(a) != 12 || len(b) != 12 || a == b {
		t.Fatalf("slug = %q, %q", a, b)
	}
	for _, r := range a + b {
		if !strings.ContainsRune(slugAlphabet, r) {
			t.Fatalf("非法字符 %q", r)
		}
	}
}

// 上游发完结束事件后不关闭连接（keep-alive 中转站很常见）：客户端必须立即返回，而不是等到超时。
func TestCompleteStopsAtTerminalEvent(t *testing.T) {
	release := make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		var events []string
		switch r.URL.Path {
		case "/v1/chat/completions":
			events = []string{`data: {"choices":[{"delta":{"content":"ok"}}]}`, `data: [DONE]`}
		case "/v1/responses":
			events = []string{`data: {"type":"response.output_text.delta","delta":"ok"}`, `data: {"type":"response.completed","response":{"usage":{"input_tokens":1,"output_tokens":1}}}`}
		case "/v1/messages":
			events = []string{`data: {"type":"content_block_delta","delta":{"type":"text_delta","text":"ok"}}`, `data: {"type":"message_stop"}`}
		}
		for _, e := range events {
			fmt.Fprint(w, e, "\n\n")
		}
		w.(http.Flusher).Flush()
		<-release // 挂住连接不关闭
	}))
	t.Cleanup(srv.Close)
	t.Cleanup(func() { close(release) }) // 先放开挂住的 handler，再关服务器
	for _, protocol := range []Protocol{ProtocolChat, ProtocolResponses, ProtocolMessages} {
		c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: protocol})
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		start := time.Now()
		res, err := c.Complete(ctx, Request{Model: "m", Prompt: "hi"})
		cancel()
		if err != nil || res.Text != "ok" {
			t.Fatalf("%s: 结果 = %+v, err = %v", protocol, res, err)
		}
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Fatalf("%s: 结束事件后 %s 才返回", protocol, elapsed)
		}
	}
}
