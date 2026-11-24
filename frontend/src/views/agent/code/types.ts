// 会话事件：和后端 pkg/agentrun 里 Item 的 JSON 一一对应（WebSocket 推来的、导出的都是这个格式）。

export type ItemType = 'user' | 'assistant' | 'thought' | 'tool' | 'permission' | 'question' | 'turn' | 'notice' | 'subagent' | 'relay'

export interface AgentAttachmentRef {
  /** snippet：打开的文件里选中的一段，name 是「路径:起止行」；paste：粘贴的一大段文字转成的文本文件 */
  kind: 'image' | 'file' | 'paste' | 'ref' | 'snippet'
  name: string
  /** 上传的附件才有：凭它取回内容（图片显示缩略图） */
  file?: string
}

/**
 * 一次工具调用对一个文件的改动，原样来自 ACP。比对和显示都在页面上做。
 *   - ACP v1：oldText / newText，oldText 没有表示新建文件；
 *   - ACP v2：op 是 add / delete / modify / move / copy（move、copy 还有 oldPath），patch 是这个文件的 git patch 片段，
 *     二进制文件之类没有 patch。
 */
export interface AgentDiff {
  path: string
  oldText?: string | null
  newText?: string
  op?: string
  oldPath?: string
  patch?: string
}

export interface AgentTool {
  id: string
  title: string
  kind?: string
  /** pending / in_progress / paused / completed / failed / cancelled */
  status?: string
  command?: string
  /** 命令的退出码（ACP v2 的终端报了才有） */
  exitCode?: number
  /** 现在能单独停这一个（ACP v2） */
  cancel?: boolean
  locations?: { path: string; line?: number }[]
  diffs?: AgentDiff[]
  output?: string
  /** 工具调用产出的图片，凭它向会话的附件接口取回 */
  images?: string[]
  child?: string
}

export interface AgentPermOption {
  id: string
  name: string
  kind: string
}

export interface AgentPermission {
  id: string
  toolId?: string
  title: string
  kind?: string
  command?: string
  /** 给人看的说明（ACP v2 的批准自带） */
  description?: string
  options: AgentPermOption[]
  /** 空 = 还在等；selected / cancelled */
  outcome?: string
  optionId?: string
  optionKind?: string
}

export interface AgentTurnUsage {
  total: number
  input: number
  output: number
  cacheRead: number
  cacheWrite: number
  thought: number
}

export interface AgentTurn {
  stop: string
  ms: number
  usage?: AgentTurnUsage
  /** 引擎认出的出错原因：process_exited / auth_required */
  code?: string
  /** Agent 或进程给的原话 */
  error?: string
}

export interface AgentSubagent {
  id: string
  title?: string
  description?: string
  /** 它自己经 ACP 报的状态：running / requires_action / idle，unknown 是 Agent 看不到它了；没报过为空 */
  state?: string
  /** 第一次报上来、最近一次停下的时间（毫秒） */
  startAt?: number
  endAt?: number
  /** 它的工作目录（ACP v2 报了才有） */
  cwd?: string
  /** 现在能单独停它（ACP v2） */
  cancel?: boolean
}

export interface AgentItem {
  seq: number
  rev: number
  type: ItemType
  /** 子 Agent 的会话 id；主会话的事件为空 */
  sub?: string
  at: number
  text?: string
  open?: boolean
  ms?: number
  attachments?: AgentAttachmentRef[]
  /** 用户的话是在一轮还没结束时插进去的 */
  steer?: boolean
  /** 会话之间的消息（relay）从哪来、到哪去：main 是主会话，其它是子 Agent 的会话 id */
  from?: string
  to?: string
  /** 提示：level 是 info / warning / error；Agent 给的提示 title 是标题、text 是说明；code 是引擎自己的提示（resume_failed 等） */
  level?: string
  title?: string
  code?: string
  tool?: AgentTool
  permission?: AgentPermission
  question?: AgentQuestion
  turn?: AgentTurn
  subagent?: AgentSubagent
}

/** 问题字段的种类：单选、多选、文字、数字、整数、是否 */
export type AgentFieldKind = 'single' | 'multi' | 'text' | 'number' | 'integer' | 'boolean'

export interface AgentQuestionOption {
  value: string
  label: string
  description?: string
}

export interface AgentQuestionField {
  key: string
  title?: string
  description?: string
  kind: AgentFieldKind
  options?: AgentQuestionOption[]
  required?: boolean
  /** 这一格是给哪个问题补充的「其他」：放在那个问题下面 */
  for?: string
}

/** Agent 向人提的问题（ACP 表单询问，Claude 的 AskUserQuestion）：一张表，每个字段一个问题 */
export interface AgentQuestion {
  id: string
  toolId?: string
  /** 只有一个问题时就是问题本身；几个问题时是总的说明 */
  message: string
  fields: AgentQuestionField[]
  /** 空 = 还在等；accept 答了；decline 跳过；cancelled 这一轮停了 */
  outcome?: string
  /** 答了的内容：字段 key → 值 */
  answers?: Record<string, string[]>
}

export interface AgentCommand {
  name: string
  description?: string
  hint?: string
}

export interface AgentPlanEntry {
  content: string
  /** pending / in_progress / completed */
  status: string
}

/** 后台还在跑的一项（ACP v2）：type 是 terminal / tool_call / subagent，id 是工具调用 id 或子 Agent 的会话 id */
export interface AgentBackground {
  type: string
  id: string
  title?: string
}
/** 一个子 Agent 自己的计划和上下文用量 */
export interface AgentSubState {
  plan?: AgentPlanEntry[]
  contextUsed?: number
  contextSize?: number
}
export interface AgentState {
  commands?: AgentCommand[]
  plan?: AgentPlanEntry[]
  background?: AgentBackground[]
  /** 子 Agent 的会话 id → 它自己的计划、上下文用量 */
  subs?: Record<string, AgentSubState>
  /** 排着队的消息：这一轮结束后按顺序作为下一轮发出 */
  queue?: AgentQueued[]
  /** 主会话有一轮正在进行（只有子 Agent 在干活不算） */
  turn?: boolean
  /** 这个进程里的子 Agent 现在的样子（按派出的先后），状态只看它们经 ACP 报的；进程信息里的是同一份 */
  subagents?: AgentSubagent[]
}
export interface AgentQueued {
  id: string
  text: string
  attachments?: AgentAttachmentRef[]
}
