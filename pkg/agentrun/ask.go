package agentrun

import (
	"context"
	"errors"
	"strings"
	"time"

	"momoko/pkg/acp"
)

// sideRun 在同一个 Agent 进程里另开的一次性会话（比如写提交说明）：它的更新不进对话、不报给页面，
// 只把 Agent 回答的文字攒下来；它要批准、要问问题时一律拒绝（这种会话不该动手做事）。
type sideRun struct {
	order []string
	texts map[string]string
	done  chan struct{}
	ended bool
	err   string
}

func (r *sideRun) finish(errText string) {
	if r.ended {
		return
	}
	r.ended, r.err = true, errText
	close(r.done)
}

// answer 最后一条消息的文字（前面的可能是「我看一下」之类）。
func (r *sideRun) answer() string {
	for i := len(r.order) - 1; i >= 0; i-- {
		if t := strings.TrimSpace(r.texts[r.order[i]]); t != "" {
			return t
		}
	}
	return ""
}

// update 一次性会话的一条更新（在 s.mu 里调用）。
func (r *sideRun) update(u *acp.SessionUpdate) {
	put := func(id, text string, appendMode bool) {
		if _, ok := r.texts[id]; !ok {
			r.order = append(r.order, id)
		}
		if appendMode {
			r.texts[id] += text
		} else {
			r.texts[id] = text
		}
	}
	switch u.Kind {
	case acp.UpdateAgentMessageChunk:
		if b := u.Chunk(); b != nil && b.Type == "text" {
			put(u.MessageID, b.Text, true)
		}
	case acp.UpdateAgentMessage:
		if blocks, ok := u.Blocks(); ok {
			put(u.MessageID, textOf(blocks), false)
		}
	case acp.UpdateState:
		if u.StateName() == acp.StateIdle {
			msg := ""
			if u.Error != nil {
				msg = u.Error.Message
			}
			r.finish(msg)
		}
	}
}

// sideOf 这条消息是不是属于一个一次性会话（在 s.mu 里调用）。
func (s *Session) sideOf(sessionID string) *sideRun {
	if sessionID == "" || sessionID == s.acpID {
		return nil
	}
	return s.sides[sessionID]
}

// Ask 在同一个 Agent 进程里另开一个一次性会话问一句话，返回 Agent 的回答。不影响正在进行的对话，
// 用和对话一样的模型和推理强度；问完关掉这个会话（Agent 支持关闭时）。
func (s *Session) Ask(ctx context.Context, cwd, text string) (string, error) {
	s.mu.Lock()
	if s.gone || s.stopping {
		s.mu.Unlock()
		return "", errors.New("Agent 进程刚刚结束了，再试一次")
	}
	s.lastActive = time.Now()
	configs := s.configs
	s.mu.Unlock()

	state, err := s.client.NewSession(ctx, acp.SessionSetup{Cwd: cwd})
	if err != nil {
		return "", exitReason(s.proc, err)
	}
	id := state.SessionID
	run := &sideRun{texts: map[string]string{}, done: make(chan struct{})}
	s.mu.Lock()
	if s.sides == nil {
		s.sides = map[string]*sideRun{}
	}
	s.sides[id] = run
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.sides, id)
		s.lastActive = time.Now()
		s.mu.Unlock()
		if s.caps.Close {
			closeCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
			_ = s.client.CloseSession(closeCtx, id)
			cancel()
		}
	}()

	if !s.modelByEnv {
		options := state.ConfigOptions
		for _, category := range []string{acp.CategoryModel, acp.CategoryThoughtLevel} {
			want, have := configOf(configs, category), configOf(options, category)
			if want == nil || have == nil || want.Current() == have.Current() || !hasValue(have, want.Current()) {
				continue
			}
			if next, err := s.client.SetConfigOption(ctx, id, have.ID, want.Current()); err == nil {
				options = next
			}
		}
	}

	blocks := []acp.ContentBlock{{Type: "text", Text: text}}
	if s.v2 {
		// v2 的回复只说明收下了，这一问的结束看 state_update
		if _, err := s.client.Prompt(ctx, id, blocks); err != nil {
			return "", exitReason(s.proc, err)
		}
		select {
		case <-run.done:
		case <-s.client.Done():
			return "", errors.New("Agent 进程结束了")
		case <-ctx.Done():
			_ = s.client.Cancel(id)
			return "", errors.New("Agent 太久没有回答")
		}
	} else {
		resp, err := s.client.Prompt(ctx, id, blocks)
		if err != nil {
			if ctx.Err() != nil {
				_ = s.client.Cancel(id)
				return "", errors.New("Agent 太久没有回答")
			}
			return "", exitReason(s.proc, err)
		}
		s.mu.Lock()
		run.finish("")
		if resp.StopReason == acp.StopRefusal {
			run.err = "Agent 拒绝了"
		}
		s.mu.Unlock()
	}
	s.mu.Lock()
	answer, failed := run.answer(), run.err
	s.mu.Unlock()
	if failed != "" {
		return "", errors.New(failed)
	}
	if answer == "" {
		return "", errors.New("Agent 没有给出回答")
	}
	return answer, nil
}
