// Package agentrun 是会话引擎：启动 Agent 进程，用 ACP 和它说话，把它发回来的会话更新整理成一条条对话事件，
// 管一轮的开始、停止和批准。它不知道用户、数据库和页面——事件出现或变化时通过 Sink 报给调用方，
// 落库、推送、通知都由调用方决定。
package agentrun

import (
	"encoding/json"
	"strings"

	"momoko/pkg/acp"
)

// 对话事件的类型。
const (
	ItemUser       = "user"
	ItemAssistant  = "assistant"
	ItemThought    = "thought"
	ItemTool       = "tool"
	ItemPermission = "permission"
	ItemTurn       = "turn"
	ItemNotice     = "notice"
	ItemSubagent   = "subagent"
)

// outputKeep 工具输出最多留多少字节（留末尾）。
const outputKeep = 48 * 1024

// Item 对话里的一条事件。Seq 是它在会话里的位置，Rev 是它最后一次变化的修订号：
// 断线重连时补发 Rev 大于页面已知值的事件。
type Item struct {
	Seq  int64  `json:"seq"`
	Rev  int64  `json:"rev"`
	Type string `json:"type"`
	// Sub 子 Agent 的会话 id；主会话的事件为空。
	Sub string `json:"sub,omitempty"`
	// At 开始时间（毫秒）。
	At int64 `json:"at"`

	// 消息、思考、提示的文字；Open 表示还在往后写。
	Text string `json:"text,omitempty"`
	Open bool   `json:"open,omitempty"`
	// Ms 思考或工具调用用了多久。
	Ms          int64        `json:"ms,omitempty"`
	Attachments []Attachment `json:"attachments,omitempty"`
	// 提示：Level 是 info / warning / error。Agent 给的提示 Title 是标题、Text 是说明；
	// Code 不为空时是引擎自己的提示（resume_failed 等），页面按它显示文案，Text 只放原因。
	Level string `json:"level,omitempty"`
	Title string `json:"title,omitempty"`
	Code  string `json:"code,omitempty"`

	Tool       *Tool       `json:"tool,omitempty"`
	Permission *Permission `json:"permission,omitempty"`
	Turn       *Turn       `json:"turn,omitempty"`
	Subagent   *Subagent   `json:"subagent,omitempty"`
}

// Payload 事件的 JSON，存库用。
func (i *Item) Payload() string {
	raw, err := json.Marshal(i)
	if err != nil {
		return ""
	}
	return string(raw)
}

// Decode 从存库的一行还原事件。
func Decode(seq, rev int64, typ, sub, payload string) *Item {
	item := &Item{}
	_ = json.Unmarshal([]byte(payload), item)
	item.Seq, item.Rev, item.Type, item.Sub = seq, rev, typ, sub
	return item
}

// Settle 把中断时停在半路的事件收个尾，给页面显示用：没有进程在跑时不会再有东西往里写，
// 库里留着的「还在写」「还在等」都是上次中断留下的。
func (i *Item) Settle() {
	i.Open = false
	if i.Tool != nil && (i.Tool.Status == "pending" || i.Tool.Status == "in_progress") {
		i.Tool.Status = "failed"
	}
	if i.Permission != nil && i.Permission.Outcome == "" {
		i.Permission.Outcome = "cancelled"
	}
	if i.Subagent != nil && i.Subagent.State != "idle" {
		i.Subagent.State = "idle"
	}
}

// Attachment 用户消息里带的附件。上传的附件有 File，页面凭它取回内容（图片显示缩略图）。
type Attachment struct {
	// Kind：image / file / ref（引用的工作区文件）。
	Kind string `json:"kind"`
	Name string `json:"name"`
	File string `json:"file,omitempty"`
}

type Location struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
}

// Diff 一次工具调用对一个文件的改动，原样来自 ACP：OldText 为 nil 表示新建文件。怎么比对、怎么显示是页面的事。
type Diff struct {
	Path    string  `json:"path"`
	OldText *string `json:"oldText"`
	NewText string  `json:"newText"`
}

type Tool struct {
	ID        string     `json:"id"`
	Title     string     `json:"title"`
	Kind      string     `json:"kind,omitempty"`
	Status    string     `json:"status,omitempty"`
	Command   string     `json:"command,omitempty"`
	Locations []Location `json:"locations,omitempty"`
	Diffs     []Diff     `json:"diffs,omitempty"`
	Output    string     `json:"output,omitempty"`
	// Child 这次调用派出的子 Agent 的会话 id。
	Child string `json:"child,omitempty"`
}

type PermissionOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// Permission 一张批准卡片。
type Permission struct {
	ID      string             `json:"id"`
	ToolID  string             `json:"toolId,omitempty"`
	Title   string             `json:"title"`
	Kind    string             `json:"kind,omitempty"`
	Command string             `json:"command,omitempty"`
	Options []PermissionOption `json:"options"`
	// Outcome 空 = 还在等；selected = 选了某个选项；cancelled = 这一轮被停止。
	Outcome    string `json:"outcome,omitempty"`
	OptionID   string `json:"optionId,omitempty"`
	OptionKind string `json:"optionKind,omitempty"`
}

// Usage 一轮的 token 用量。Input 不含缓存命中和缓存写入。
type Usage struct {
	Total      int64 `json:"total"`
	Input      int64 `json:"input"`
	Output     int64 `json:"output"`
	CacheRead  int64 `json:"cacheRead"`
	CacheWrite int64 `json:"cacheWrite"`
	Thought    int64 `json:"thought"`
}

func usageOf(u *acp.Usage) *Usage {
	out := &Usage{Total: u.TotalTokens, Input: u.InputTokens, Output: u.OutputTokens}
	if u.CachedReadTokens != nil {
		out.CacheRead = *u.CachedReadTokens
	}
	if u.CachedWriteTokens != nil {
		out.CacheWrite = *u.CachedWriteTokens
	}
	if u.ThoughtTokens != nil {
		out.Thought = *u.ThoughtTokens
	}
	return out
}

// Turn 一轮结束时的摘要。
type Turn struct {
	// Stop：end_turn / cancelled / max_tokens / max_turn_requests / refusal / error。
	Stop string `json:"stop"`
	Ms   int64  `json:"ms"`
	// Usage Agent 给了才有。
	Usage *Usage `json:"usage,omitempty"`
	// Code 引擎自己认出的出错原因：process_exited（进程异常退出）、auth_required（登录失效）；
	// Error 是 Agent 或进程给的原话。
	Code  string `json:"code,omitempty"`
	Error string `json:"error,omitempty"`
}

// Subagent 主会话派出的一个子 Agent。它自己的对话事件带着它的会话 id（Item.Sub）单独成一条流。
type Subagent struct {
	ID          string `json:"id"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	// State：running / idle / requires_action。
	State string `json:"state,omitempty"`
	EndAt int64  `json:"endAt,omitempty"`
}

// Command Agent 自己的一个斜杠命令。
type Command struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Hint        string `json:"hint,omitempty"`
}

type PlanEntry struct {
	Content string `json:"content"`
	Status  string `json:"status"`
}

// State 会话状态快照：Agent 给的命令和计划。
type State struct {
	Commands []Command   `json:"commands,omitempty"`
	Plan     []PlanEntry `json:"plan,omitempty"`
}

// Summary 取一段回答的开头，通知里当结束说明用。
func Summary(text string) string {
	text = strings.TrimSpace(text)
	if i := strings.Index(text, "\n\n"); i > 0 {
		text = text[:i]
	}
	text = strings.Join(strings.Fields(text), " ")
	if r := []rune(text); len(r) > 200 {
		text = string(r[:200]) + "…"
	}
	return text
}
