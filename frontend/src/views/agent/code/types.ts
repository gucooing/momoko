// 会话事件：和后端 pkg/agentrun 里 Item 的 JSON 一一对应（WebSocket 推来的、导出的都是这个格式）。

export type ItemType = 'user' | 'assistant' | 'thought' | 'tool' | 'permission' | 'turn' | 'notice' | 'subagent'

export interface AgentAttachmentRef {
  kind: 'image' | 'file' | 'ref'
  name: string
  /** 上传的附件才有：凭它取回内容（图片显示缩略图） */
  file?: string
}

/** 一次工具调用对一个文件的改动，原样来自 ACP：oldText 为 null 表示新建文件。比对和显示都在页面上做。 */
export interface AgentDiff {
  path: string
  oldText: string | null
  newText: string
}

export interface AgentTool {
  id: string
  title: string
  kind?: string
  /** pending / in_progress / completed / failed / cancelled */
  status?: string
  command?: string
  locations?: { path: string; line?: number }[]
  diffs?: AgentDiff[]
  output?: string
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
  /** running / idle / requires_action */
  state?: string
  endAt?: number
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
  /** 提示：level 是 info / warning / error；Agent 给的提示 title 是标题、text 是说明；code 是引擎自己的提示（resume_failed 等） */
  level?: string
  title?: string
  code?: string
  tool?: AgentTool
  permission?: AgentPermission
  turn?: AgentTurn
  subagent?: AgentSubagent
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

export interface AgentState {
  commands?: AgentCommand[]
  plan?: AgentPlanEntry[]
}
