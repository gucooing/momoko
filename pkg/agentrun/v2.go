package agentrun

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"
	"time"

	"momoko/pkg/acp"
	"momoko/pkg/agentfs"
)

// 这里是 ACP v2 才有的东西：一轮由 state_update 开始和结束、Agent 自己的显示终端、按 id 替换的计划和整条消息、
// 后台还在跑的工作、撤回的批准、单独停一个工具调用或子 Agent。v1 的连接走不到这些分支。

// terminalKeep 一个终端最多留多少字节的原始输出（留末尾）。
const terminalKeep = 256 << 10

// terminal v2 里 Agent 自己的显示终端：输出是原始字节（可能带颜色、光标控制），显示在引用它的那个工具调用上。
type terminal struct {
	// toolKey 引用它的工具调用；终端的更新可能比引用先到，那时为空。
	toolKey string
	data    []byte
	command string
	exited  bool
	exit    *int64
	cancel  bool
}

func (s *Session) terminalOf(sub, id string) *terminal {
	key := toolKey(sub, id)
	t := s.terminals[key]
	if t == nil {
		t = &terminal{}
		s.terminals[key] = t
	}
	return t
}

// onTerminal terminal_update（整份快照、命令、退出、能不能停）和 terminal_output_chunk（接在后面的输出）。
// 输出块很密，攒着每秒报一次；其它变化马上报。
func (s *Session) onTerminal(sub string, u *acp.SessionUpdate) {
	if u.TerminalID == "" {
		return
	}
	t := s.terminalOf(sub, u.TerminalID)
	if u.Kind == acp.UpdateTerminalOutputChunk {
		data, err := base64.StdEncoding.DecodeString(u.Data)
		if err != nil {
			return
		}
		t.data = append(t.data, data...)
		if len(t.data) > terminalKeep {
			t.data = append([]byte(nil), t.data[len(t.data)-terminalKeep:]...)
		}
		if t.toolKey != "" {
			s.showTerminal(t)
			s.toolDirty[t.toolKey] = true
		}
		return
	}
	if u.Command != nil {
		t.command = *u.Command
	}
	if u.Output != nil {
		if data, err := base64.StdEncoding.DecodeString(u.Output.Data); err == nil {
			if len(data) > terminalKeep {
				data = data[len(data)-terminalKeep:]
			}
			t.data = data
		}
	}
	if u.ExitStatus != nil {
		t.exited, t.exit = true, u.ExitStatus.ExitCode
	}
	if cancel, ok := u.CanCancel(); ok {
		t.cancel = cancel
	}
	if item := s.tools[t.toolKey]; item != nil {
		s.showTerminal(t)
		delete(s.toolDirty, t.toolKey)
		s.emit(item, true)
	}
}

// showTerminal 把终端的现状写到引用它的工具调用上（不报出去）。
func (s *Session) showTerminal(t *terminal) {
	item := s.tools[t.toolKey]
	if item == nil {
		return
	}
	tool := item.Tool
	tool.Output = keepTail(plainTerminal(t.data))
	if tool.Command == "" {
		tool.Command = t.command
	}
	tool.ExitCode = t.exit
	tool.termCancel = t.cancel && !t.exited
	refreshCancel(tool)
}

// refreshCancel 工具调用有了结果就不能再停了；转到后台的例外（报了完成，命令其实还在跑）。
func refreshCancel(tool *Tool) {
	tool.Cancel = (tool.toolCancel || tool.termCancel) && (unfinished(tool.Status) || tool.background)
}

// plainTerminal 终端的原始输出变成能直接显示的文字：去掉 ANSI 控制序列（颜色、光标移动、窗口标题），
// \r 回到行首覆盖（进度条只留最后的样子），退格往回一格，其它控制字符去掉；不完整的 UTF-8 换成替换字符。
func plainTerminal(data []byte) string {
	rs := []rune(strings.ToValidUTF8(string(data), "�"))
	var out strings.Builder
	var line []rune
	col := 0
	for i := 0; i < len(rs); i++ {
		switch r := rs[i]; {
		case r == 0x1b:
			i = skipEscape(rs, i)
		case r == '\r':
			col = 0
		case r == '\n':
			out.WriteString(string(line))
			out.WriteByte('\n')
			line, col = line[:0], 0
		case r == '\b':
			col = max(col-1, 0)
		case r == '\t' || (r >= 0x20 && r != 0x7f && (r < 0x80 || r >= 0xa0)):
			if col < len(line) {
				line[col] = r
			} else {
				line = append(line, r)
			}
			col++
		}
	}
	out.WriteString(string(line))
	return out.String()
}

// skipEscape 从 ESC 开始跳过一个控制序列，返回它最后一个字符的位置。序列没写完（在末尾被截断）时跳到末尾。
func skipEscape(rs []rune, i int) int {
	if i+1 >= len(rs) {
		return i
	}
	switch rs[i+1] {
	case '[': // CSI：参数，然后一个 0x40–0x7E 的结尾
		j := i + 2
		for j < len(rs) && (rs[j] < 0x40 || rs[j] > 0x7e) {
			j++
		}
		return j
	case ']', 'P', '^', '_', 'X': // OSC / DCS / PM / APC：到 BEL 或 ESC \ 为止
		for j := i + 2; j < len(rs); j++ {
			if rs[j] == 0x07 {
				return j
			}
			if rs[j] == 0x1b && j+1 < len(rs) && rs[j+1] == '\\' {
				return j + 1
			}
		}
		return len(rs)
	default:
		return i + 1
	}
}

// onState state_update：主会话的决定一轮的开始和结束（运行中也能接着发消息，Agent 自己开始的工作也算一轮），
// 子 Agent 的只改它自己的状态。带着的后台列表整份替换。
func (s *Session) onState(sub string, u *acp.SessionUpdate) {
	state := u.StateName()
	if sub != "" {
		s.subagentState(sub, state)
		return
	}
	if u.Background != nil {
		s.setBackground(u.Background)
	}
	switch state {
	case acp.StateRunning, acp.StateRequiresAction:
		if s.turn == nil {
			s.turn = &turnRun{started: time.Now()}
			s.resetTurnState()
			s.lastActive = time.Now()
			s.turnStartedLocked()
		}
		s.turn.active = true
	case acp.StateIdle:
		if s.turn == nil {
			return
		}
		result := &Turn{Stop: u.StopReason}
		if result.Stop == "" {
			result.Stop = acp.StopEndTurn
		}
		if u.Error != nil {
			result.Stop, result.Error = acp.StopError, u.Error.Message
			if u.Error.Code == acp.CodeAuthRequired {
				result.Code = TurnAuthRequired
			}
		}
		if u.Usage != nil {
			result.Usage = usageOf(u.Usage)
		}
		s.after = append(s.after, s.endTurnLocked(result))
	}
}

// setBackground 后台还在跑的工作：记下来（这一轮结束时不把它们当成没结果，进程也不当空闲回收），并报给页面。
func (s *Session) setBackground(list []acp.BackgroundWork) {
	defer s.markBackground()
	s.background = map[string]bool{}
	s.state.Background = make([]Background, 0, len(list))
	for _, w := range list {
		switch w.Type {
		case "tool_call":
			key := toolKey("", w.ToolCallID)
			s.background[key] = true
			title := ""
			if item := s.tools[key]; item != nil {
				title = item.Tool.Title
			}
			s.state.Background = append(s.state.Background, Background{Type: w.Type, ID: w.ToolCallID, Title: title})
		case "terminal":
			t := s.terminals[toolKey("", w.TerminalID)]
			if t == nil {
				continue
			}
			bg := Background{Type: w.Type, Title: t.command}
			if t.toolKey != "" {
				s.background[t.toolKey] = true
				if item := s.tools[t.toolKey]; item != nil {
					bg.ID = item.Tool.ID
					if bg.Title == "" {
						bg.Title = item.Tool.Title
					}
				}
			}
			s.state.Background = append(s.state.Background, bg)
		case "subagent":
			s.background["sub:"+w.SessionID] = true
			title := ""
			if item := s.subs[w.SessionID]; item != nil {
				title = item.Subagent.Title
			}
			s.state.Background = append(s.state.Background, Background{Type: w.Type, ID: w.SessionID, Title: title})
		}
	}
	s.sink.State(s.state)
}

// markBackground 进出后台的工具调用：更新它能不能停，变了就报一次（页面上的「停止」跟着出现或消失）。
func (s *Session) markBackground() {
	for key, item := range s.tools {
		if item.Tool == nil || item.Tool.background == s.background[key] {
			continue
		}
		cancel := item.Tool.Cancel
		item.Tool.background = s.background[key]
		refreshCancel(item.Tool)
		if item.Tool.Cancel != cancel {
			s.emit(item, true)
		}
	}
}

// clearBackground 进程没了：后台的工作跟着没了，列表清空报给页面，免得一直显示还在跑。
func (s *Session) clearBackground() {
	if len(s.background) == 0 && len(s.state.Background) == 0 {
		return
	}
	s.background = map[string]bool{}
	s.markBackground()
	s.state.Background = nil
	s.sink.State(s.state)
}

// resetTurnState 新的一轮开始：上一轮的工具调用、消息不会再变，不再留着；后台还在跑的工具调用和它们的终端例外。
func (s *Session) resetTurnState() {
	tools := map[string]*Item{}
	for key, item := range s.tools {
		if s.background[key] {
			tools[key] = item
		}
	}
	s.tools = tools
	for key, t := range s.terminals {
		if t.toolKey != "" && tools[t.toolKey] == nil {
			delete(s.terminals, key)
		}
	}
	s.messages = map[string]*Item{}
}

// planList 一条流（主会话或某个子 Agent）的计划：v2 可以有好几份，按 id 整份替换，按第一次出现的先后排。
type planList struct {
	ids     []string
	entries map[string][]PlanEntry
}

// setPlan 一份计划变了（v1 只有一份，id 为空）。主会话的放在 State.Plan，子 Agent 的放在 State.Subs 里。
func (s *Session) setPlan(sub, planID string, entries []acp.PlanEntry) {
	list := s.plans[sub]
	if list == nil {
		list = &planList{entries: map[string][]PlanEntry{}}
		s.plans[sub] = list
	}
	if _, ok := list.entries[planID]; !ok {
		list.ids = append(list.ids, planID)
	}
	plan := make([]PlanEntry, 0, len(entries))
	for _, e := range entries {
		plan = append(plan, PlanEntry{Content: e.Content, Status: e.Status})
	}
	list.entries[planID] = plan
	var all []PlanEntry
	for _, id := range list.ids {
		all = append(all, list.entries[id]...)
	}
	if sub == "" {
		s.state.Plan = all
	} else {
		s.subState(sub).Plan = all
	}
	s.sink.State(s.state)
}

func (s *Session) subState(sub string) *SubState {
	if s.state.Subs == nil {
		s.state.Subs = map[string]*SubState{}
	}
	st := s.state.Subs[sub]
	if st == nil {
		st = &SubState{}
		s.state.Subs[sub] = st
	}
	return st
}

func messageKey(sub, typ, msgID string) string { return sub + "\x00" + typ + "\x00" + msgID }

// upsertMessage v2 的整条消息（user_message / agent_message / agent_thought / session_message）：
// 已经有这条（按消息 id）就整条换掉，没有就新起一条；之后同一个 id 的块接在它后面。返回这条事件。
func (s *Session) upsertMessage(sub, typ, msgID, text string) *Item {
	key := messageKey(sub, typ, msgID)
	if item := s.messages[key]; item != nil {
		item.Text = text
		if s.open[sub] == item {
			s.dirty[sub] = false
		}
		s.emit(item, true)
		return item
	}
	s.closeOpen(sub)
	item := s.newItem(typ, sub)
	item.Text, item.msgID = text, msgID
	if msgID != "" {
		s.messages[key] = item
	}
	s.emit(item, true)
	return item
}

// textOf 内容块里的文字，接成一段。
func textOf(blocks []acp.ContentBlock) string {
	var parts []string
	for _, b := range blocks {
		if b.Type == "text" && b.Text != "" {
			parts = append(parts, b.Text)
		}
	}
	return strings.Join(parts, "\n")
}

// peer 会话之间的消息里的一个会话：主会话记成 main，子 Agent 记它的会话 id，没说是谁为空。
func (s *Session) peer(id *string) string {
	switch {
	case id == nil || *id == "":
		return ""
	case *id == s.acpID:
		return "main"
	default:
		return *id
	}
}

// onRelay 会话之间的消息：整条的按 id 替换，块接在后面；记下从哪来、到哪去。
func (s *Session) onRelay(sub string, u *acp.SessionUpdate) {
	var item *Item
	if u.Kind == acp.UpdateSessionMessage {
		blocks, ok := u.Blocks()
		if !ok {
			return
		}
		item = s.upsertMessage(sub, ItemRelay, u.MessageID, textOf(blocks))
	} else {
		block := u.Chunk()
		if block == nil || block.Type != "text" {
			return
		}
		s.appendText(sub, ItemRelay, block.Text, u.MessageID)
		item = s.open[sub]
	}
	if item == nil || item.Type != ItemRelay {
		return
	}
	if from, to := s.peer(u.SenderSessionID), s.peer(u.RecipientSessionID); (from != "" && item.From != from) || (to != "" && item.To != to) {
		item.From, item.To = from, to
		s.emit(item, true)
	}
}

// subagentState 子 Agent 自己报的状态（v2 的子会话会发自己的 state_update）。
func (s *Session) subagentState(sub, state string) {
	item := s.subs[sub]
	if item == nil || state == "" || item.Subagent.State == state {
		return
	}
	s.setSubagentState(sub, item.Subagent, state)
	s.emit(item, true)
	s.syncSubagents()
}

// setSubagentState 子 Agent 的状态变了：空闲就算这次做完，记下结束时间；又动起来了清掉。
// unknown 是 Agent 说它看不到这个子 Agent 了（比如对话被清空），之后也不会再报，不算在干活。
func (s *Session) setSubagentState(sub string, sa *Subagent, state string) {
	sa.State = state
	switch state {
	case acp.StateIdle:
		s.closeOpen(sub)
		if sa.EndAt == 0 {
			sa.EndAt = time.Now().UnixMilli()
		}
	case acp.StateRunning, acp.StateRequiresAction:
		sa.EndAt = 0
	}
}

// RequestCancelled Agent 撤回了一张批准或一个提问（这一轮停了，不再需要），卡片收成「已取消」。
func (s *Session) RequestCancelled(reqID string) {
	s.mu.Lock()
	permID := s.permReqs[reqID]
	p := s.perms[permID]
	if p == nil {
		s.mu.Unlock()
		return
	}
	delete(s.perms, permID)
	delete(s.permReqs, reqID)
	p.markCancelled()
	s.emit(p.item, true)
	waiting := len(s.perms) > 0
	s.mu.Unlock()
	p.replyCancelled()
	s.sink.Answered(permID, "", waiting)
}

// StopItem v2：只停一个工具调用（toolID 不为空），或者停一个子 Agent（toolID 为空，sub 是它的会话 id）；
// 只有它声明了能单独停才行。sub 为空表示主会话里的工具调用。
func (s *Session) StopItem(ctx context.Context, sub, toolID string) error {
	if !s.v2 {
		return errors.New("这个 Agent 不能单独停一项")
	}
	s.mu.Lock()
	if toolID == "" {
		item := s.subs[sub]
		ok := item != nil && item.Subagent.Cancel
		s.mu.Unlock()
		if !ok {
			return errors.New("这个子 Agent 不能单独停")
		}
		return s.client.Cancel(sub)
	}
	sessionID := s.acpID
	if sub != "" {
		sessionID = sub
	}
	item := s.tools[toolKey(sub, toolID)]
	if item == nil || !item.Tool.Cancel {
		s.mu.Unlock()
		return errors.New("这个工具调用已经结束了，或者不能单独停")
	}
	tool := item.Tool
	terminalID := tool.terminal
	// 工具调用自己还没结束、也声明了能停，就停它；转到后台、工具调用已经报了完成的，停它还在跑的终端
	byTool := tool.toolCancel && (unfinished(tool.Status) || !tool.termCancel || terminalID == "")
	s.mu.Unlock()
	stopCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	var err error
	if byTool {
		err = s.client.CancelToolCall(stopCtx, sessionID, toolID)
	} else {
		err = s.client.CancelTerminal(stopCtx, sessionID, terminalID)
	}
	if err != nil {
		return exitReason(s.proc, err)
	}
	return nil
}

// sendV2 v2 发消息：回复只说明消息收下了，这一轮的进展和结束都从 state_update 来。
// 没收下时：这条消息自己开的一轮（还没动起来）记成出错结束；运行中插进去的那条记一条提示。
func (s *Session) sendV2(blocks []acp.ContentBlock, turn *turnRun) {
	_, err := s.client.Prompt(context.Background(), s.acpID, blocks)
	s.mu.Lock()
	if err == nil {
		if turn != nil && s.turn == turn {
			turn.active = true
		}
		s.mu.Unlock()
		return
	}
	// 进程没了：remove 会收尾
	if errors.Is(err, acp.ErrClosed) || s.proc.Exited() {
		s.mu.Unlock()
		return
	}
	if turn != nil && s.turn == turn && !turn.active {
		result, _ := s.turnError(err)
		report := s.endTurnLocked(result)
		s.mu.Unlock()
		report()
		return
	}
	item := s.newItem(ItemNotice, "")
	item.Level, item.Code, item.Text = "error", NoticePromptFailed, s.turnErrorText(err)
	s.emit(item, true)
	s.mu.Unlock()
}

// diffsOf v2 的 diff：每个文件级改动一条，带上 patch 里属于这个文件的那一段。
func (s *Session) diffsOf(c acp.ToolCallContent) []Diff {
	sections := patchSections(c.Patch)
	out := make([]Diff, 0, len(c.Changes))
	for _, ch := range c.Changes {
		d := Diff{Path: agentfs.Rel(s.cwd, ch.Path), Op: ch.Operation, Patch: sectionOf(sections, ch.Path)}
		if ch.OldPath != "" {
			d.OldPath = agentfs.Rel(s.cwd, ch.OldPath)
		}
		out = append(out, d)
	}
	return out
}

// patchSections git patch 按文件切开：每段从一行 diff --git 开始。
func patchSections(p *acp.DiffPatch) []string {
	if p == nil || p.Format != "git_patch" || p.Text == "" {
		return nil
	}
	var out []string
	start := -1
	for at := 0; at < len(p.Text); {
		end := strings.IndexByte(p.Text[at:], '\n')
		next := len(p.Text)
		if end >= 0 {
			next = at + end + 1
		}
		if strings.HasPrefix(p.Text[at:], "diff --git ") {
			if start >= 0 {
				out = append(out, p.Text[start:at])
			}
			start = at
		}
		at = next
	}
	if start >= 0 {
		out = append(out, p.Text[start:])
	}
	return out
}

// sectionOf 头一行里写着这个路径的那一段。Agent 写的路径分隔符、大小写不一定和改动里的一致（Windows 上常见），
// 都按正斜杠、不分大小写比；只有一段时就是它。
func sectionOf(sections []string, path string) string {
	norm := func(s string) string { return strings.ToLower(strings.ReplaceAll(s, `\`, "/")) }
	want := norm(path)
	for _, sec := range sections {
		header, _, _ := strings.Cut(sec, "\n")
		if strings.Contains(norm(header), want) {
			return sec
		}
	}
	if len(sections) == 1 {
		return sections[0]
	}
	return ""
}
