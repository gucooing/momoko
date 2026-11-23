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
	// ItemQuestion Agent 向人提的问题（ACP 表单询问），等人回答。
	ItemQuestion = "question"
	ItemTurn     = "turn"
	ItemNotice   = "notice"
	ItemSubagent = "subagent"
	// ItemRelay 会话之间的消息（v2：主会话交给子 Agent 的话、子 Agent 交回的结果），不是用户说的，也不是回答用户的。
	ItemRelay = "relay"
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
	// Steer 用户的话是在一轮还没结束时插进去的。
	Steer bool `json:"steer,omitempty"`
	// From / To 会话之间的消息从哪个会话来、到哪个会话去：main 是主会话，其它是子 Agent 的会话 id。
	From string `json:"from,omitempty"`
	To   string `json:"to,omitempty"`
	// msgID Agent 给这条消息的 id（v2 必有）：同一条消息的块接在一起，整条更新按它替换。
	msgID string
	// 提示：Level 是 info / warning / error。Agent 给的提示 Title 是标题、Text 是说明；
	// Code 不为空时是引擎自己的提示（resume_failed 等），页面按它显示文案，Text 只放原因。
	Level string `json:"level,omitempty"`
	Title string `json:"title,omitempty"`
	Code  string `json:"code,omitempty"`

	Tool       *Tool       `json:"tool,omitempty"`
	Permission *Permission `json:"permission,omitempty"`
	Question   *Question   `json:"question,omitempty"`
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
	if i.Tool != nil {
		if unfinished(i.Tool.Status) {
			i.Tool.Status = "failed"
		}
		i.Tool.Cancel = false
	}
	if i.Permission != nil && i.Permission.Outcome == "" {
		i.Permission.Outcome = "cancelled"
	}
	if i.Question != nil && i.Question.Outcome == "" {
		i.Question.Outcome = "cancelled"
	}
	if i.Subagent != nil {
		if i.Subagent.State != "idle" {
			i.Subagent.State = "idle"
		}
		i.Subagent.Cancel = false
	}
}

// unfinished 工具调用还没有结果：等着、在跑、暂停（v2）。
func unfinished(status string) bool {
	return status == "pending" || status == "in_progress" || status == "paused"
}

// Attachment 用户消息里带的附件。上传的附件有 File，页面凭它取回内容（图片显示缩略图）。
type Attachment struct {
	// Kind：image / file / paste（粘贴的一大段文字）/ ref（引用的工作区文件）/ snippet（选中的一段）。
	Kind string `json:"kind"`
	Name string `json:"name"`
	File string `json:"file,omitempty"`
}

type Location struct {
	Path string `json:"path"`
	Line int    `json:"line,omitempty"`
}

// Diff 一次工具调用对一个文件的改动，原样来自 ACP。怎么比对、怎么显示是页面的事。
//   - v1：OldText / NewText，OldText 为 nil 表示新建文件；
//   - v2：Op 是 add / delete / modify / move / copy（move、copy 还有 OldPath），Patch 是这个文件的 git patch 片段，
//     二进制文件、符号链接之类没有 Patch。
type Diff struct {
	Path    string  `json:"path"`
	OldText *string `json:"oldText,omitempty"`
	NewText string  `json:"newText,omitempty"`
	Op      string  `json:"op,omitempty"`
	OldPath string  `json:"oldPath,omitempty"`
	Patch   string  `json:"patch,omitempty"`
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
	// Images 工具调用产出的图片（生图、截图等），存在会话的附件目录里，页面凭它取回，同附件的 File。
	Images []string `json:"images,omitempty"`
	// Child 这次调用派出的子 Agent 的会话 id。
	Child string `json:"child,omitempty"`
	// ExitCode 命令的退出码（v2 的终端报了才有）。
	ExitCode *int64 `json:"exitCode,omitempty"`
	// Cancel 现在能单独停这一个（v2：工具调用或它的终端声明了 capabilities.cancel）。
	Cancel bool `json:"cancel,omitempty"`

	// terminal 显示它输出的终端 id；toolCancel / termCancel 工具调用、终端各自能不能单独停。
	terminal   string
	toolCancel bool
	termCancel bool
	// background v2：转到后台还在跑（Agent 的 state_update 里列着它）。这时工具调用本身可能已经报了完成，
	// 但它的终端还在跑，照样能停。
	background bool
}

type PermissionOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Kind string `json:"kind"`
}

// Permission 一张批准卡片。
type Permission struct {
	ID      string `json:"id"`
	ToolID  string `json:"toolId,omitempty"`
	Title   string `json:"title"`
	Kind    string `json:"kind,omitempty"`
	Command string `json:"command,omitempty"`
	// Description 给人看的说明（v2 的批准自带）。
	Description string             `json:"description,omitempty"`
	Options     []PermissionOption `json:"options"`
	// Outcome 空 = 还在等；selected = 选了某个选项；cancelled = 这一轮被停止。
	Outcome    string `json:"outcome,omitempty"`
	OptionID   string `json:"optionId,omitempty"`
	OptionKind string `json:"optionKind,omitempty"`
}

// Question Agent 向人提的问题：ACP 的表单询问（elicitation/create，form），Claude 的 AskUserQuestion 就走它。
// 一张表里可以有几个问题，每个是一个字段。
type Question struct {
	ID     string `json:"id"`
	ToolID string `json:"toolId,omitempty"`
	// Message 只有一个问题时就是问题本身；几个问题时是总的说明。
	Message string          `json:"message"`
	Fields  []QuestionField `json:"fields"`
	// Outcome 空 = 还在等；accept = 答了；decline = 跳过；cancelled = 这一轮停了，或者 Agent 撤回了。
	Outcome string `json:"outcome,omitempty"`
	// Answers 答了的内容：字段 key → 值（单选、文字是一个，多选是几个）。
	Answers map[string][]string `json:"answers,omitempty"`
}

// 问题字段的种类。
const (
	FieldSingle  = "single"
	FieldMulti   = "multi"
	FieldText    = "text"
	FieldNumber  = "number"
	FieldInteger = "integer"
	FieldBoolean = "boolean"
)

// QuestionField 表里的一个字段。
type QuestionField struct {
	Key         string `json:"key"`
	Title       string `json:"title,omitempty"`
	Description string `json:"description,omitempty"`
	// Kind 见 Field* 常量。
	Kind     string           `json:"kind"`
	Options  []QuestionOption `json:"options,omitempty"`
	Required bool             `json:"required,omitempty"`
	// For 这一格是给哪个问题补充的「其他」（Claude 的 AskUserQuestion 每个问题都配一格），页面把它放在那个问题下面。
	For string `json:"for,omitempty"`
}

// QuestionOption 一个候选项：Value 是回答时交回去的值，Label 给人看。
type QuestionOption struct {
	Value       string `json:"value"`
	Label       string `json:"label"`
	Description string `json:"description,omitempty"`
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
	// Cwd 它的工作目录（v2 报了才有）。
	Cwd string `json:"cwd,omitempty"`
	// Cancel 现在能单独停它（v2：它声明了 capabilities.cancel）。
	Cancel bool `json:"cancel,omitempty"`
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

// State 会话状态快照：Agent 给的命令和计划，后台还在跑的，和每个子 Agent 自己的计划、上下文用量。
type State struct {
	Commands []Command   `json:"commands,omitempty"`
	Plan     []PlanEntry `json:"plan,omitempty"`
	// Background v2：一轮结束后还在后台跑的终端、工具调用、子 Agent。
	Background []Background `json:"background,omitempty"`
	// Subs 子 Agent 的会话 id → 它自己的计划、上下文用量。
	Subs map[string]*SubState `json:"subs,omitempty"`
	// Queue 排着队的消息：这一轮结束后按顺序作为下一轮发出。只在进程活着时有意义。
	Queue []Queued `json:"queue,omitempty"`
}

// Queued 一条排着队的消息。
type Queued struct {
	ID          string       `json:"id"`
	Text        string       `json:"text"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// Background 后台还在跑的一项。Type：terminal / tool_call / subagent；ID 是工具调用 id 或子 Agent 的会话 id
// （终端算在显示它的那个工具调用上）；Title 给人看。
type Background struct {
	Type  string `json:"type"`
	ID    string `json:"id"`
	Title string `json:"title,omitempty"`
}

// SubState 一个子 Agent 自己的计划和上下文用量。
type SubState struct {
	Plan        []PlanEntry `json:"plan,omitempty"`
	ContextUsed int64       `json:"contextUsed,omitempty"`
	ContextSize int64       `json:"contextSize,omitempty"`
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
