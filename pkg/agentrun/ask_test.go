package agentrun

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"momoko/pkg/acp"
)

// fakeAgent 一个只会 session/new、session/prompt 的 ACP Agent（v1）：问的时候先要一次批准，
// 再分两块回答，最后回复这一问结束。permission 收到 Momoko 对那次批准的回答。
func fakeAgent(t *testing.T, in io.Reader, out io.Writer, permission chan<- string) {
	t.Helper()
	send := func(v any) {
		raw, _ := json.Marshal(v)
		_, _ = out.Write(append(raw, '\n'))
	}
	update := func(text string) {
		send(map[string]any{"jsonrpc": "2.0", "method": acp.MethodSessionUpdate, "params": map[string]any{
			"sessionId": "side-1",
			"update":    map[string]any{"sessionUpdate": acp.UpdateAgentMessageChunk, "messageId": "m1", "content": map[string]any{"type": "text", "text": text}},
		}})
	}
	go func() {
		sc := bufio.NewScanner(in)
		var promptID json.RawMessage
		for sc.Scan() {
			var msg struct {
				ID     json.RawMessage `json:"id"`
				Method string          `json:"method"`
				Result json.RawMessage `json:"result"`
			}
			if json.Unmarshal(sc.Bytes(), &msg) != nil {
				continue
			}
			switch {
			case msg.Method == acp.MethodSessionNew:
				send(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": map[string]any{"sessionId": "side-1"}})
			case msg.Method == acp.MethodSessionPrompt:
				promptID = msg.ID
				send(map[string]any{"jsonrpc": "2.0", "id": "perm-1", "method": acp.MethodRequestPermission, "params": map[string]any{
					"sessionId": "side-1",
					"toolCall":  map[string]any{"toolCallId": "t1", "title": "git diff"},
					"options":   []map[string]any{{"optionId": "allow", "name": "Allow", "kind": "allow_once"}},
				}})
			case msg.Method == "" && string(msg.ID) == `"perm-1"`:
				// 批准被拒绝后照常回答
				permission <- string(msg.Result)
				update("feat: 加上 ")
				update("Git 面板")
				send(map[string]any{"jsonrpc": "2.0", "id": promptID, "result": map[string]any{"stopReason": acp.StopEndTurn}})
			}
		}
	}()
}

func TestAskSideSession(t *testing.T) {
	toAgentR, toAgentW := io.Pipe()
	fromAgentR, fromAgentW := io.Pipe()
	defer toAgentW.Close()
	defer fromAgentW.Close()
	permission := make(chan string, 1)
	fakeAgent(t, toAgentR, fromAgentW, permission)

	s := &Session{acpID: "main", modelByEnv: true}
	s.client = acp.NewClient(fromAgentR, toAgentW, s)
	defer s.client.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	answer, err := s.Ask(ctx, t.TempDir(), "写提交说明")
	if err != nil {
		t.Fatal(err)
	}
	if answer != "feat: 加上 Git 面板" {
		t.Fatalf("answer = %q", answer)
	}
	select {
	case got := <-permission:
		if !strings.Contains(got, "cancelled") {
			t.Fatalf("side session permission was not refused: %s", got)
		}
	default:
		t.Fatal("no permission reply")
	}
	if len(s.sides) != 0 {
		t.Fatalf("side session left behind: %v", s.sides)
	}
	if len(s.messages) != 0 || len(s.open) != 0 {
		t.Fatal("side session text leaked into the conversation")
	}
}
