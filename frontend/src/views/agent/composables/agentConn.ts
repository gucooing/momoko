// Agent 页面和后端之间的那一条 WebSocket：Code 页和管理页（都在 KeepAlive 里）共用，有人用时连着、
// 都不用了就断开；断线自动重连（1、2、4… 秒，最多 10 秒），重连后让各自补上自己的订阅。
import { AGENT_WS_PATH } from '@/api/agent'
import { buildBackendWebSocketUrl } from '@/utils/websocket'

export interface AgentServerMessage {
  t: 'open' | 'more' | 'item' | 'delta' | 'state' | 'usage' | 'session' | 'gone' | 'config'
  id: string
  [key: string]: unknown
}

type Listener = (msg: AgentServerMessage) => void

const listeners = new Set<Listener>()
const reconnectHooks = new Set<() => void>()
const connected = ref(false)
let ws: WebSocket | null = null
let users = 0
let retry = 0
let timer: ReturnType<typeof setTimeout> | undefined

const connect = () => {
  if (!users || ws) return
  const url = buildBackendWebSocketUrl(AGENT_WS_PATH)
  if (!url) return
  const socket = new WebSocket(url)
  ws = socket
  socket.onopen = () => {
    connected.value = true
    retry = 0
    for (const hook of reconnectHooks) hook()
  }
  socket.onmessage = (e) => {
    let msg: AgentServerMessage
    try {
      msg = JSON.parse(String(e.data)) as AgentServerMessage
    } catch {
      return
    }
    for (const fn of listeners) fn(msg)
  }
  socket.onclose = () => {
    if (ws !== socket) return
    ws = null
    connected.value = false
    if (!users) return
    const wait = Math.min(10_000, 1000 * 2 ** retry)
    retry += 1
    timer = setTimeout(connect, wait)
  }
}

/** 发一条消息；没连上时丢掉（连上后由重连钩子补发订阅）。 */
export const agentSend = (msg: Record<string, unknown>) => {
  if (ws && ws.readyState === WebSocket.OPEN) ws.send(JSON.stringify(msg))
}

/**
 * 用这条连接：listener 收所有消息，onReconnect 在每次（重新）连上时调用。返回停止使用的函数。
 */
export const useAgentConn = (listener: Listener, onReconnect: () => void) => {
  let active = false
  const start = () => {
    if (active) return
    active = true
    users += 1
    listeners.add(listener)
    reconnectHooks.add(onReconnect)
    if (ws?.readyState === WebSocket.OPEN) onReconnect()
    else connect()
  }
  const stop = () => {
    if (!active) return
    active = false
    users -= 1
    listeners.delete(listener)
    reconnectHooks.delete(onReconnect)
    if (!users) {
      if (timer) clearTimeout(timer)
      const socket = ws
      ws = null
      connected.value = false
      socket?.close()
    }
  }
  onBeforeUnmount(stop)
  return { connected, start, stop }
}
