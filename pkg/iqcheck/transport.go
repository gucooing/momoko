package iqcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ErrIdleTimeout distinguishes a silent gateway from the overall request deadline.
var ErrIdleTimeout = errors.New("等待上游响应数据超时")

// do bounds every network wait, including response headers and error bodies.
// Canceling the request unblocks net/http reads without a separate reader goroutine.
func (c *Client) do(req *http.Request) (*http.Response, error) {
	ctx, cancel := context.WithCancelCause(req.Context())
	timer := time.AfterFunc(c.idleTimeout, func() { cancel(ErrIdleTimeout) })
	resp, err := c.http.Do(req.WithContext(ctx))
	if err != nil {
		timer.Stop()
		if cause := context.Cause(ctx); cause != nil {
			err = cause
		}
		cancel(nil)
		return nil, fmt.Errorf("请求上游失败: %w", err)
	}
	timer.Reset(c.idleTimeout)
	resp.Body = &idleBody{ReadCloser: resp.Body, ctx: ctx, cancel: cancel, timer: timer, timeout: c.idleTimeout}
	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		var data []byte
		var readErr error
		if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "json") {
			var payload json.RawMessage
			readErr = decodeJSON(io.LimitReader(resp.Body, maxErrorBody), &payload)
			data = payload
		} else {
			data, readErr = io.ReadAll(io.LimitReader(resp.Body, maxErrorBody))
		}
		if readErr != nil {
			return nil, fmt.Errorf("上游返回 HTTP %d，读取错误响应失败: %w", resp.StatusCode, readErr)
		}
		return nil, httpError(resp.StatusCode, data)
	}
	return resp, nil
}

type idleBody struct {
	io.ReadCloser
	ctx     context.Context
	cancel  context.CancelCauseFunc
	timer   *time.Timer
	timeout time.Duration
}

func (b *idleBody) Read(p []byte) (int, error) {
	n, err := b.ReadCloser.Read(p)
	if n > 0 {
		b.timer.Reset(b.timeout)
	}
	if err != nil {
		if cause := context.Cause(b.ctx); cause != nil {
			err = cause
		}
	}
	return n, err
}

func (b *idleBody) Close() error {
	b.timer.Stop()
	b.cancel(nil)
	return b.ReadCloser.Close()
}
