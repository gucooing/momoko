// Code 页的会话流：打开一个会话就订阅它的事件，先拿最近一页，往上翻再要更早的；断线重连后按修订号补齐。
// 自己的会话状态变了（标题、状态、选项）也从这里推来，会话栏跟着更新。
//
// 事件列表用 shallowRef：一条事件变了就整条换掉（不在原对象上改），只有变了的那条会重新渲染。
import type { AgentSession, AgentSessionDetail } from '@/types/v1/agent'
import { agentSend, useAgentConn, type AgentServerMessage } from '@/views/agent/composables/agentConn'
import type { AgentItem, AgentState } from './types'

export interface AgentSocketHandlers {
  /** 某个会话变了（会话栏用） */
  onSession?: (session: AgentSession) => void
  /** 某个会话被删除了 */
  onGone?: (id: string) => void
  /** Agent 配置（运行时、供应商、工作区、MCP、技能）变了 */
  onConfig?: () => void
}

export const useAgentSocket = (handlers: AgentSocketHandlers = {}) => {
  const sessionId = ref('')
  /** 打开的会话这一刻的样子（WebSocket 推来的为准） */
  const session = ref<AgentSession | null>(null)
  /** 打开会话时才给的信息：分支、开着的 MCP、技能、复刻来源、Agent 版本 */
  const detail = ref<AgentSessionDetail | null>(null)
  const items = shallowRef<AgentItem[]>([])
  const state = ref<AgentState>({})
  const contextUsed = ref(0)
  const contextSize = ref(0)
  /** 更早的事件还有没拿 */
  const more = ref(false)
  const loadingMore = ref(false)
  /** 第一次收到 open 之前为 false */
  const loaded = ref(false)
  const openError = ref('')

  let rev = 0
  let index = new Map<number, number>()

  const reindex = () => {
    index = new Map(items.value.map((it, i) => [it.seq, i]))
  }

  /** 按序号放进列表：已有的换成新的（修订号更大才换），没有的按序号插进去。还没往上翻到的那一段不收。 */
  const upsert = (list: AgentItem[], item: AgentItem) => {
    if (item.rev > rev) rev = item.rev
    const at = index.get(item.seq)
    if (at != null) {
      const old = list[at]
      if (!old || item.rev >= old.rev) list[at] = item
      return
    }
    const first = list[0]
    if (more.value && first && item.seq < first.seq) return
    const last = list[list.length - 1]
    if (!last || last.seq < item.seq) {
      list.push(item)
      index.set(item.seq, list.length - 1)
      return
    }
    list.push(item)
    list.sort((a, b) => a.seq - b.seq)
    reindex()
  }

  const apply = (incoming: AgentItem[]) => {
    const list = items.value.slice()
    for (const item of incoming) upsert(list, item)
    items.value = list
  }

  const onMessage = (msg: AgentServerMessage) => {
    if (msg.t === 'config') {
      handlers.onConfig?.()
      return
    }
    if (msg.t === 'session') {
      const s = msg.session as AgentSession | undefined
      if (s) {
        handlers.onSession?.(s)
        if (msg.id === sessionId.value) session.value = s
      }
      return
    }
    if (msg.t === 'gone') {
      handlers.onGone?.(msg.id)
      return
    }
    if (msg.id !== sessionId.value) return
    switch (msg.t) {
      case 'open': {
        loaded.value = true
        if (msg.error) {
          openError.value = String(msg.error)
          return
        }
        openError.value = ''
        session.value = (msg.session as AgentSession) ?? session.value
        detail.value = (msg.detail as AgentSessionDetail) ?? null
        if (!items.value.length) more.value = Boolean(msg.more)
        apply((msg.items as AgentItem[]) ?? [])
        rev = Math.max(rev, Number(msg.rev || 0))
        state.value = (msg.state as AgentState) ?? {}
        contextUsed.value = Number(msg.contextUsed || 0)
        contextSize.value = Number(msg.contextSize || 0)
        break
      }
      case 'more': {
        loadingMore.value = false
        if (msg.error) return
        more.value = Boolean(msg.more)
        const older = (msg.items as AgentItem[]) ?? []
        items.value = [...older.filter((it) => !index.has(it.seq)), ...items.value]
        reindex()
        break
      }
      case 'item':
        apply([msg.item as AgentItem])
        break
      case 'delta': {
        const at = index.get(Number(msg.seq))
        const item = at != null ? items.value[at] : undefined
        if (item && at != null) {
          const list = items.value.slice()
          list[at] = { ...item, text: (item.text || '') + String(msg.text || '') }
          items.value = list
        }
        break
      }
      case 'state':
        state.value = (msg.state as AgentState) ?? {}
        break
      case 'usage':
        contextUsed.value = Number(msg.contextUsed || 0)
        contextSize.value = Number(msg.contextSize || 0)
        break
    }
  }

  // 重新连上：服务端的订阅随旧连接没了，按已经拿到的修订号重新打开。
  const conn = useAgentConn(onMessage, () => {
    if (sessionId.value) agentSend({ op: 'open', id: sessionId.value, rev })
  })

  /** 打开一个会话（空字符串表示不看任何会话）。 */
  const open = (id: string) => {
    if (id === sessionId.value) return
    sessionId.value = id
    session.value = null
    detail.value = null
    items.value = []
    index = new Map()
    rev = 0
    more.value = false
    loadingMore.value = false
    state.value = {}
    contextUsed.value = 0
    contextSize.value = 0
    loaded.value = !id
    openError.value = ''
    if (id) agentSend({ op: 'open', id, rev: 0 })
    else agentSend({ op: 'close' })
  }

  /** 往上翻：要更早的一页。 */
  const loadMore = () => {
    const first = items.value[0]
    if (!more.value || loadingMore.value || !first || !sessionId.value) return
    loadingMore.value = true
    agentSend({ op: 'more', id: sessionId.value, before: first.seq })
  }

  return {
    connected: conn.connected,
    sessionId,
    session,
    detail,
    items,
    state,
    contextUsed,
    contextSize,
    more,
    loadingMore,
    loaded,
    openError,
    open,
    loadMore,
    start: conn.start,
    stop: conn.stop,
  }
}

export type AgentSocket = ReturnType<typeof useAgentSocket>
