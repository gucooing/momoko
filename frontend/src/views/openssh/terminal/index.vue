<template>
  <div class="ssh-page">
    <!-- Top bar -->
    <header class="ssh-bar">
      <div class="ssh-bar__left">
        <el-button
          :icon="menuStore.iconComponents['HOutline:ArrowLeftIcon']"
          text
          size="small"
          @click="goBack"
        >
          <span class="ssh-bar__btn-label">返回</span>
        </el-button>
      </div>
      <div class="ssh-bar__center">
        <span class="ssh-bar__user">{{ connInfo.username }}</span>
        <span class="ssh-bar__sep">@</span>
        <span class="ssh-bar__host">{{ connInfo.host }}:{{ connInfo.port }}</span>
      </div>
      <div class="ssh-bar__right">
        <span class="ssh-bar__dot" :class="statusClass" />
        <span class="ssh-bar__label">{{ statusLabel }}</span>
        <el-button
          v-if="socketStatus !== 'connected'"
          :icon="menuStore.iconComponents['HOutline:ArrowPathIcon']"
          text
          size="small"
          :loading="socketStatus === 'connecting'"
          @click="reconnect"
        >
          <span class="ssh-bar__btn-label">重连</span>
        </el-button>
      </div>
    </header>

    <!-- xterm container -->
    <div ref="containerRef" class="ssh-terminal" />

    <!-- Bottom hint -->
    <footer class="ssh-foot">
      <span>{{ cols }}x{{ rows }}</span>
      <span v-if="socketStatus !== 'connected'" class="ssh-foot__hint">
        — 点击
        <span class="ssh-foot__link" @click="reconnect">这里</span>
        重新连接
      </span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { Terminal } from '@xterm/xterm'
import { FitAddon } from '@xterm/addon-fit'
import '@xterm/xterm/css/xterm.css'
import { getSshHostInfo } from '@/api/openssh'
import { buildBackendWebSocketUrl } from '@/utils/websocket'
import type { ConsoleSocketStatus } from '@/stores/instance/types'

defineOptions({ name: 'SshTerminalView' })

const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()
const tabsStore = useTabsStore()

const containerRef = ref<HTMLElement>()
const socketStatus = ref<ConsoleSocketStatus>('disconnected')
const cols = ref(120)
const rows = ref(32)

let ws: WebSocket | null = null
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null

const connId = computed(() => (route.query.id as string) || '')
const connInfo = ref({ host: '-', port: '22', username: 'root', name: '' })
const wsPath = ref('')

const statusLabel = computed(() => {
  switch (socketStatus.value) {
    case 'connected': return '已连接'
    case 'connecting': return '连接中'
    case 'error': return '连接失败'
    default: return '未连接'
  }
})

const statusClass = computed(() => ({
  'ssh-bar__dot--connected': socketStatus.value === 'connected',
  'ssh-bar__dot--connecting': socketStatus.value === 'connecting',
  'ssh-bar__dot--error': socketStatus.value === 'error',
  'ssh-bar__dot--idle': socketStatus.value === 'disconnected',
}))

const buildWsUrl = () => {
  if (!connId.value || !wsPath.value) return ''
  return buildBackendWebSocketUrl(wsPath.value, (url) => {
    url.searchParams.set('hostID', connId.value)
  })
}

const goBack = () => {
  router.back()
}

const resizeTerminal = () => {
  fitAddon?.fit()
  if (!terminal) return
  cols.value = terminal.cols
  rows.value = terminal.rows

  if (ws?.readyState === WebSocket.OPEN) {
    ws.send(
      JSON.stringify({
        type: 'resize',
        cols: terminal.cols,
        rows: terminal.rows,
      }),
    )
  }
}

const mountTerminal = () => {
  const el = containerRef.value
  if (!el || terminal) return

  terminal = new Terminal({
    cursorBlink: true,
    convertEol: true,
    fontFamily: 'Consolas, Menlo, Monaco, monospace',
    fontSize: 14,
    theme: {
      background: '#0b1020',
      foreground: '#d5dde8',
    },
  })

  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(el)
  fitAddon.fit()
  cols.value = terminal.cols
  rows.value = terminal.rows

  terminal.onData((data) => {
    if (ws?.readyState === WebSocket.OPEN) {
      ws.send(data)
    }
  })
}

const connect = async () => {
  if (!connId.value) {
    ElMessage.warning('缺少连接参数')
    return
  }

  socketStatus.value = 'connecting'

  try {
    const { data } = await getSshHostInfo({ id: connId.value })
    if (data?.info) {
      const info = data.info
      wsPath.value = info.wsPath
      connInfo.value = {
        host: info.host,
        port: String(info.port),
        username: info.username,
        name: info.name,
      }
      const tab = tabsStore.tabs.find((t) => t.fullPath === route.fullPath)
      if (tab) tab.title = info.name || `${info.username}@${info.host}`
    }
  } catch {
    wsPath.value = ''
  }

  const url = buildWsUrl()
  if (!url) {
    socketStatus.value = 'disconnected'
    ElMessage.warning('缺少连接参数')
    return
  }

  const socket = new WebSocket(url)
  ws = socket

  socket.onopen = () => {
    socketStatus.value = 'connected'
    resizeTerminal()
  }

  socket.onmessage = async (event) => {
    const text =
      typeof event.data === 'string'
        ? event.data
        : event.data instanceof Blob
          ? await event.data.text()
          : new TextDecoder().decode(event.data)

    terminal?.write(text)
  }

  socket.onerror = () => {
    socketStatus.value = 'error'
  }

  socket.onclose = () => {
    ws = null
    if (socketStatus.value !== 'disconnected') {
      socketStatus.value = 'disconnected'
      terminal?.writeln('')
      terminal?.writeln('\x1b[31m[连接已断开]\x1b[0m')
    }
  }
}

const disconnect = () => {
  if (ws) {
    ws.close()
    ws = null
  }
  socketStatus.value = 'disconnected'
}

const reconnect = () => {
  disconnect()
  setTimeout(() => {
    void connect()
  }, 200)
}

let resizeObserver: ResizeObserver | null = null

onMounted(() => {
  mountTerminal()

  resizeObserver = new ResizeObserver(() => {
    resizeTerminal()
  })
  if (containerRef.value) {
    resizeObserver.observe(containerRef.value)
  }
  window.addEventListener('resize', resizeTerminal)

  if (connId.value) {
    setTimeout(() => {
      void connect()
    }, 300)
  }
})

onActivated(() => {
  resizeObserver?.observe(containerRef.value!)
  window.addEventListener('resize', resizeTerminal)
  fitAddon?.fit()
})

onDeactivated(() => {
  window.removeEventListener('resize', resizeTerminal)
  resizeObserver?.disconnect()
})

onBeforeUnmount(() => {
  window.removeEventListener('resize', resizeTerminal)
  resizeObserver?.disconnect()
  disconnect()
  terminal?.dispose()
  terminal = null
  fitAddon = null
})
</script>

<style scoped>
.ssh-page {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 7rem);
  min-height: 24rem;
  border-radius: 8px;
  overflow: hidden;
  background: #0b1020;
  border: 1px solid var(--el-border-color);
}

/* Top bar */
.ssh-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 10px;
  height: 36px;
  background: #111827;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
  user-select: none;
}

.ssh-bar__left,
.ssh-bar__right {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}

.ssh-bar__center {
  display: flex;
  align-items: center;
  gap: 2px;
  min-width: 0;
  font-family: 'Consolas', 'Menlo', 'Monaco', monospace;
  font-size: 0.8rem;
}

.ssh-bar__user {
  color: #7ee787;
  font-weight: 600;
}

.ssh-bar__sep {
  color: #484f58;
}

.ssh-bar__host {
  color: #8b949e;
}

.ssh-bar__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.ssh-bar__dot--connected {
  background: #3fb950;
  box-shadow: 0 0 6px rgba(63, 185, 80, 0.5);
}

.ssh-bar__dot--connecting {
  background: #d29922;
  box-shadow: 0 0 6px rgba(210, 153, 34, 0.5);
  animation: ssh-blink 1s ease-in-out infinite;
}

.ssh-bar__dot--error {
  background: #f85149;
  box-shadow: 0 0 6px rgba(248, 81, 73, 0.5);
}

.ssh-bar__dot--idle {
  background: #484f58;
}

.ssh-bar__label {
  font-size: 0.72rem;
  color: #8b949e;
}

/* Terminal */
.ssh-terminal {
  flex: 1;
  overflow: hidden;
}

.ssh-terminal :deep(.xterm) {
  height: 100%;
  padding: 4px;
}

.ssh-terminal :deep(.xterm-viewport) {
  scrollbar-width: thin;
  scrollbar-color: rgba(255, 255, 255, 0.08) transparent;
}

.ssh-terminal :deep(.xterm-viewport::-webkit-scrollbar) {
  width: 6px;
}

.ssh-terminal :deep(.xterm-viewport::-webkit-scrollbar-track) {
  background: transparent;
}

.ssh-terminal :deep(.xterm-viewport::-webkit-scrollbar-thumb) {
  background: rgba(255, 255, 255, 0.08);
  border-radius: 3px;
}

/* Foot */
.ssh-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 10px;
  height: 26px;
  background: #111827;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
  font-size: 0.68rem;
  color: #484f58;
  font-family: 'Consolas', 'Menlo', 'Monaco', monospace;
  user-select: none;
}

.ssh-foot__link {
  color: #58a6ff;
  cursor: pointer;
  text-decoration: underline;
}

.ssh-foot__link:hover {
  color: #79c0ff;
}

/* Light mode */
:global(html.light .ssh-page) {
  background: #f6f8fa;
}

:global(html.light .ssh-bar),
:global(html.light .ssh-foot) {
  background: #ebeef1;
  border-color: rgba(0, 0, 0, 0.06);
}

:global(html.light .ssh-bar__user) {
  color: #1a7f37;
}

:global(html.light .ssh-bar__sep) {
  color: #8c959f;
}

:global(html.light .ssh-bar__host) {
  color: #0969da;
}

:global(html.light .ssh-bar__label) {
  color: #656d76;
}

/* Animations */
@keyframes ssh-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

/* Responsive */
@media (width <= 768px) {
  .ssh-page {
    height: calc(100dvh - 3.5rem);
    min-height: 16rem;
    border-radius: 0;
    border-left: none;
    border-right: none;
  }

  .ssh-bar {
    padding: 0 8px;
    height: 32px;
    gap: 4px;
  }

  .ssh-bar__btn-label {
    display: none;
  }

  .ssh-bar__center {
    font-size: 0.72rem;
    flex: 1;
    overflow: hidden;
  }

  .ssh-bar__host {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .ssh-bar__label {
    display: none;
  }

  .ssh-foot {
    padding: 0 8px;
    height: 22px;
    font-size: 0.62rem;
  }
}
</style>
