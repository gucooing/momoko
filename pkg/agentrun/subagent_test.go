package agentrun

import (
	"encoding/json"
	"testing"

	"momoko/pkg/acp"
)

// recSink 记下引擎报上来的状态、子 Agent 干不干活、一轮的开始和结束。
type recSink struct {
	state   State
	working []bool
	started int
	ended   []TurnEnd
}

func (r *recSink) Item(*Item, bool)              {}
func (r *recSink) Delta(int64, string)           {}
func (r *recSink) State(st State)                { r.state = st }
func (r *recSink) Usage(int64, int64)            {}
func (r *recSink) Title(string)                  {}
func (r *recSink) Config(string, string, string) {}
func (r *recSink) Asked(*Permission)             {}
func (r *recSink) Questioned(*Question)          {}
func (r *recSink) TurnStarted()                  { r.started++ }
func (r *recSink) SubagentsWorking(on bool)      { r.working = append(r.working, on) }
func (r *recSink) Answered(string, string, bool) {}
func (r *recSink) TurnEnded(end TurnEnd)         { r.ended = append(r.ended, end) }
func (r *recSink) Exited()                       {}
func (r *recSink) lastWorking() (on, ok bool) {
	if len(r.working) == 0 {
		return false, false
	}
	return r.working[len(r.working)-1], true
}

func newTestSession(sink Sink) *Session {
	return &Session{
		sink: sink, acpID: "main", v2: true,
		open: map[string]*Item{}, dirty: map[string]bool{}, tools: map[string]*Item{}, subs: map[string]*Item{},
		perms: map[string]*pending{}, permReqs: map[string]string{}, messages: map[string]*Item{},
		terminals: map[string]*terminal{}, toolDirty: map[string]bool{}, plans: map[string]*planList{},
		background: map[string]bool{}, done: make(chan struct{}),
	}
}

// notify 把一条 ACP session/update 原样交给引擎。
func notify(t *testing.T, s *Session, raw string) {
	t.Helper()
	n := &acp.SessionNotification{}
	if err := json.Unmarshal([]byte(raw), n); err != nil {
		t.Fatal(err)
	}
	s.SessionUpdate(n)
}

// 子 Agent 干不干活只看它经 ACP 报的状态：主会话这一轮结束了它还在跑，会话就还算在干活；
// 发消息让它接着做（又报 running）和新派出的一样算；Agent 说看不到它了（unknown）就不算。
func TestSubagentWorkingFollowsACPState(t *testing.T) {
	sink := &recSink{}
	s := newTestSession(sink)

	notify(t, s, `{"sessionId":"main","update":{"sessionUpdate":"state_update","state":"running"}}`)
	if sink.started != 1 || !sink.state.Turn {
		t.Fatalf("turn start not reported: started=%d turn=%v", sink.started, sink.state.Turn)
	}
	notify(t, s, `{"sessionId":"main","update":{"sessionUpdate":"subagent_update","sessionId":"sub-a","title":"副本","capabilities":{"cancel":{}},"state":{"state":"running"}}}`)
	if len(sink.working) != 0 {
		t.Fatalf("working reported while the main turn runs: %v", sink.working)
	}
	if got := sink.state.Subagents; len(got) != 1 || got[0].ID != "sub-a" || got[0].State != "running" || got[0].StartAt == 0 {
		t.Fatalf("subagents = %+v", got)
	}

	// 主会话这一轮结束，子 Agent 还在跑
	notify(t, s, `{"sessionId":"main","update":{"sessionUpdate":"state_update","state":"idle","stopReason":"end_turn"}}`)
	if len(sink.ended) != 1 || !sink.ended[0].Working || sink.state.Turn {
		t.Fatalf("turn end: ended=%+v turn=%v", sink.ended, sink.state.Turn)
	}
	if !s.Working() || !s.Busy() {
		t.Fatal("session should still be working while the subagent runs")
	}

	// 子会话自己报空闲
	notify(t, s, `{"sessionId":"sub-a","update":{"sessionUpdate":"state_update","state":"idle"}}`)
	if on, ok := sink.lastWorking(); !ok || on {
		t.Fatalf("working = %v", sink.working)
	}
	if s.Working() || sink.state.Subagents[0].EndAt == 0 {
		t.Fatalf("subagent should be done: %+v", sink.state.Subagents)
	}

	// 发消息让它接着做：同一个子会话又报 running
	notify(t, s, `{"sessionId":"main","update":{"sessionUpdate":"subagent_update","sessionId":"sub-a","state":{"state":"running"}}}`)
	if on, _ := sink.lastWorking(); !on || !s.Working() {
		t.Fatalf("resumed subagent not working: %v", sink.working)
	}
	if got := s.Subagents(); len(got) != 1 || got[0].State != "running" || got[0].EndAt != 0 {
		t.Fatalf("process view differs: %+v", got)
	}

	// 只带标题、没报状态的（恢复出来的上级）不算在跑
	notify(t, s, `{"sessionId":"main","update":{"sessionUpdate":"subagent_update","sessionId":"sub-b","title":"上级"}}`)
	if got := sink.state.Subagents; len(got) != 2 || got[1].State != "" {
		t.Fatalf("subagents = %+v", got)
	}

	// Agent 看不到它了
	notify(t, s, `{"sessionId":"main","update":{"sessionUpdate":"subagent_update","sessionId":"sub-a","state":{"state":"unknown"}}}`)
	if on, _ := sink.lastWorking(); on || s.Working() || sink.state.Subagents[0].State != "unknown" {
		t.Fatalf("unknown subagent still working: %v %+v", sink.working, sink.state.Subagents)
	}

	// 进程没了：还在干活的记成停下
	notify(t, s, `{"sessionId":"main","update":{"sessionUpdate":"subagent_update","sessionId":"sub-b","state":{"state":"requires_action"}}}`)
	s.mu.Lock()
	s.gone = true
	s.stopSubagents()
	s.mu.Unlock()
	if got := sink.state.Subagents[1]; got.State != "idle" || got.Cancel {
		t.Fatalf("subagent after exit = %+v", got)
	}
}
