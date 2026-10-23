package iqcheck

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/iotest"
	"time"
)

func TestSSEFraming(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n", "\r"} {
		t.Run(fmt.Sprintf("%q", newline), func(t *testing.T) {
			source := "\uFEFFevent: answer" + newline + "data: first" + newline + "data:  second" + newline + newline + ": heartbeat" + newline + "event: reset" + newline + newline + "data" + newline + newline + "data: last"
			var events []string
			err := readSSE(iotest.OneByteReader(strings.NewReader(source)), func(event, data string) error {
				events = append(events, event+"|"+data)
				return nil
			})
			if err != nil || strings.Join(events, ";") != "answer|first\n second;|;|last" {
				t.Fatalf("events=%q error=%v", events, err)
			}
		})
	}
}

func TestSSETerminalLineEndings(t *testing.T) {
	for _, newline := range []string{"\n", "\r\n", "\r"} {
		t.Run(fmt.Sprintf("%q", newline), func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "\uFEFFdata: {\"type\":\"content_block_delta\",\"delta\":{\"type\":\"text_delta\",\"text\":\"ok\"}}", newline, newline, "data: {\"type\":\"message_stop\"}", newline, newline)
				w.(http.Flusher).Flush()
				<-r.Context().Done()
			}))
			t.Cleanup(srv.Close)
			c := mustClient(t, Config{Endpoint: srv.URL, APIKey: "k", Protocol: ProtocolMessages, Timeout: time.Second})
			res, err := c.Complete(context.Background(), Request{Model: "m", Prompt: "hi"})
			if err != nil || res.Text != "ok" {
				t.Fatalf("result=%+v error=%v", res, err)
			}
		})
	}
}

func TestSSELargeAndBoundedEvents(t *testing.T) {
	text := strings.Repeat("x", 128<<10)
	err := readSSE(strings.NewReader("data: "+text+"\n\n"), func(_ string, data string) error {
		if data != text {
			t.Fatalf("event truncated: %d bytes", len(data))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, source := range []io.Reader{
		strings.NewReader("data: " + strings.Repeat("x", maxSSEEventSize)),
		strings.NewReader(strings.Repeat("data: "+strings.Repeat("x", 1024)+"\n", maxSSEEventSize/1024)),
	} {
		if err := readSSE(source, func(_, _ string) error { t.Fatal("oversize event dispatched"); return nil }); err == nil {
			t.Fatal("expected size limit error")
		}
	}
}
