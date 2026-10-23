package iqcheck

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"slices"
	"sort"
	"strings"
	"time"
)

// After a finish/stop reason, only optional usage and a closing marker remain.
// A fixed deadline also handles relays that keep sending heartbeats forever.
const streamTrailerTimeout = time.Second

type streamState struct {
	ctx      context.Context
	cancel   context.CancelCauseFunc
	timer    *time.Timer
	finished bool
}

func newStreamState(ctx context.Context) *streamState {
	ctx, cancel := context.WithCancelCause(ctx)
	return &streamState{ctx: ctx, cancel: cancel}
}

func (s *streamState) close() {
	if s.timer != nil {
		s.timer.Stop()
	}
	s.cancel(nil)
}

func (s *streamState) finish() error {
	s.finished = true
	return errStopSSE
}

func (s *streamState) awaitTrailer() {
	if !s.finished {
		s.finished = true
		s.timer = time.AfterFunc(streamTrailerTimeout, func() { s.cancel(errStopSSE) })
	}
}

func (s *streamState) read(r io.Reader, fn func(string, string) error) error {
	limited := &io.LimitedReader{R: r, N: 64 << 20}
	err := readSSE(limited, fn)
	if errors.Is(err, errStopSSE) && s.finished {
		return nil
	}
	if err != nil {
		return err
	}
	if limited.N == 0 {
		return errors.New("上游事件流超过大小限制")
	}
	if !s.finished {
		return fmt.Errorf("上游事件流在结束事件前断开: %w", io.ErrUnexpectedEOF)
	}
	return nil
}

// decodeEvent accepts both named SSE events and gateways using a generic
// "message" event with the actual type in JSON. Unknown valid events can be
// ignored by the protocol handler; malformed data and upstream errors cannot.
func decodeEvent(event, data string, v any, knownEvents ...string) (string, error) {
	if strings.TrimSpace(data) == "" || event == "ping" || event == "heartbeat" || event == "keepalive" {
		return "", nil
	}
	var header struct {
		Type    string          `json:"type"`
		Error   *apiError       `json:"error"`
		Message json.RawMessage `json:"message"`
		Code    json.RawMessage `json:"code"`
	}
	if err := json.Unmarshal([]byte(data), &header); err != nil {
		return "", fmt.Errorf("上游 SSE 事件 %q 无法解析: %w", event, err)
	}
	if header.Error != nil {
		return "", header.Error
	}
	if event == "" || event == "message" {
		event = header.Type
	}
	if event == "error" || header.Type == "error" {
		var message string
		_ = json.Unmarshal(header.Message, &message)
		return "", &apiError{Message: message, Code: header.Code}
	}
	if len(knownEvents) > 0 && !slices.Contains(knownEvents, event) {
		return event, nil
	}
	if err := json.Unmarshal([]byte(data), v); err != nil {
		return "", fmt.Errorf("上游 SSE 事件 %q 无法解析: %w", event, err)
	}
	return event, nil
}

// Text snapshots replace deltas for the same item/part; indices keep interleaved
// content in output order and prevent duplicate text from *.done events.
type textPartKey struct{ item, part int }
type streamText map[textPartKey]*strings.Builder

func (t streamText) append(item, part int, text string) {
	key := textPartKey{item, part}
	if t[key] == nil {
		t[key] = &strings.Builder{}
	}
	t[key].WriteString(text)
}

func (t streamText) set(item, part int, text string) {
	key := textPartKey{item, part}
	t[key] = &strings.Builder{}
	t[key].WriteString(text)
}

func (t streamText) clearItem(item int) {
	for key := range t {
		if key.item == item {
			delete(t, key)
		}
	}
}

func (t streamText) String() string {
	keys := make([]textPartKey, 0, len(t))
	for key := range t {
		keys = append(keys, key)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i].item != keys[j].item {
			return keys[i].item < keys[j].item
		}
		return keys[i].part < keys[j].part
	})
	var text strings.Builder
	for _, key := range keys {
		text.WriteString(t[key].String())
	}
	return text.String()
}
