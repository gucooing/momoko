<template>
  <BaseDialog
    v-model="visible"
    :title="dialogTitle"
    width="960"
    :show-footer="false"
    @opened="handleOpened"
    @close="handleClosed"
  >
    <div class="docker-log-dialog">
      <div class="docker-log-toolbar">
        <el-input-number
          v-model="tail"
          :min="10"
          :max="10000"
          :step="100"
          size="small"
          controls-position="right"
          class="docker-log-tail"
        />
        <el-checkbox v-model="timestamps" size="small">时间戳</el-checkbox>
        <el-button
          size="small"
          :icon="menuStore.iconComponents.Refresh"
          :loading="socketStatus === 'connecting'"
          @click="reconnect"
        >
          重连
        </el-button>
        <el-button size="small" :icon="menuStore.iconComponents.Delete" @click="clearTerminal">
          清空
        </el-button>
        <span class="docker-log-status" :class="statusClass">
          <span class="docker-log-status__dot" />
          {{ statusLabel }}
        </span>
      </div>
      <div class="docker-log-terminal-shell">
        <div ref="terminalRef" class="docker-log-terminal" />
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { buildBackendWebSocketUrl } from '@/utils/websocket'

type SocketStatus = 'disconnected' | 'connecting' | 'connected' | 'error'

const props = defineProps<{
  modelValue: boolean
  containerId: string
  wsPath: string
  containerName?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const menuStore = useMenuStore()
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const terminalRef = ref<HTMLElement>()
const tail = ref(200)
const timestamps = ref(true)
const socketStatus = ref<SocketStatus>('disconnected')

let socket: WebSocket | null = null
let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null
let manualSocketClose = false

const dialogTitle = computed(() => {
  const name = props.containerName || props.containerId
  return name ? `容器日志 - ${name}` : '容器日志'
})

const statusLabel = computed(() => {
  if (socketStatus.value === 'connected') return '已连接'
  if (socketStatus.value === 'connecting') return '连接中'
  if (socketStatus.value === 'error') return '连接失败'
  return '未连接'
})

const statusClass = computed(() => ({
  'docker-log-status--connected': socketStatus.value === 'connected',
  'docker-log-status--connecting': socketStatus.value === 'connecting',
  'docker-log-status--error': socketStatus.value === 'error',
}))

const buildSocketUrl = () => {
  return buildBackendWebSocketUrl(props.wsPath, (url) => {
    url.searchParams.set('id', props.containerId)
    url.searchParams.set('tail', String(tail.value || 200))
    url.searchParams.set('timestamps', String(timestamps.value))
  })
}

const mountTerminal = () => {
  const el = terminalRef.value
  if (!el || terminal) return

  terminal = new Terminal({
    convertEol: true,
    cursorBlink: false,
    disableStdin: true,
    fontFamily: 'Consolas, Menlo, Monaco, monospace',
    fontSize: 13,
    theme: {
      background: '#0b1020',
      foreground: '#d5dde8',
    },
  })
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(el)
  fitAddon.fit()

  resizeObserver = new ResizeObserver(() => fitAddon?.fit())
  resizeObserver.observe(el)
}

const resetTerminal = () => {
  resizeObserver?.disconnect()
  resizeObserver = null
  terminal?.dispose()
  terminal = null
  fitAddon = null
}

const clearTerminal = () => {
  terminal?.clear()
}

const closeSocket = () => {
  if (!socket) return
  manualSocketClose = true
  socket.close()
  socket = null
  socketStatus.value = 'disconnected'
}

const writePayload = (payload: ArrayBuffer | string) => {
  if (!terminal) return
  if (typeof payload === 'string') {
    terminal.write(payload)
    return
  }
  terminal.write(new Uint8Array(payload))
}

const connect = () => {
  if (!props.containerId || !props.wsPath) return

  closeSocket()
  manualSocketClose = false
  socketStatus.value = 'connecting'

  const socketUrl = buildSocketUrl()
  if (!socketUrl) {
    socketStatus.value = 'error'
    return
  }

  const nextSocket = new WebSocket(socketUrl)
  nextSocket.binaryType = 'arraybuffer'
  socket = nextSocket

  nextSocket.onopen = () => {
    if (socket !== nextSocket) return
    socketStatus.value = 'connected'
  }

  nextSocket.onmessage = (event) => {
    if (socket !== nextSocket) return
    writePayload(event.data as ArrayBuffer | string)
  }

  nextSocket.onerror = () => {
    if (socket !== nextSocket) return
    socketStatus.value = 'error'
  }

  nextSocket.onclose = () => {
    const closedManually = manualSocketClose
    if (socket !== nextSocket) return
    socket = null
    if (!closedManually) {
      socketStatus.value = 'disconnected'
    }
  }
}

const reconnect = () => {
  terminal?.clear()
  connect()
}

const handleOpened = async () => {
  await nextTick()
  mountTerminal()
  terminal?.clear()
  connect()
}

const handleClosed = () => {
  closeSocket()
  resetTerminal()
}

watch([() => props.containerId, () => props.wsPath, tail, timestamps], () => {
  if (!visible.value) return
  reconnect()
})

onBeforeUnmount(() => {
  closeSocket()
  resetTerminal()
})
</script>

<style scoped lang="scss">
.docker-log-dialog {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  min-height: 24rem;
}

.docker-log-toolbar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.docker-log-tail {
  width: 8rem;
}

.docker-log-status {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 0.78rem;
}

.docker-log-status__dot {
  width: 0.45rem;
  height: 0.45rem;
  border-radius: 50%;
  background: var(--el-text-color-placeholder);
}

.docker-log-status--connected .docker-log-status__dot {
  background: var(--el-color-success);
}

.docker-log-status--connecting .docker-log-status__dot {
  background: var(--el-color-warning);
}

.docker-log-status--error .docker-log-status__dot {
  background: var(--el-color-danger);
}

.docker-log-terminal-shell {
  height: min(64vh, 580px);
  min-height: 360px;
  padding: 0.5rem;
  border-radius: 6px;
  background: #0b1020;
}

.docker-log-terminal {
  width: 100%;
  height: 100%;
}

.docker-log-terminal :deep(.xterm) {
  height: 100%;
}

.docker-log-terminal :deep(.xterm-viewport) {
  overflow-y: auto;
}

@media (width <= 768px) {
  .docker-log-dialog {
    min-height: 18rem;
  }

  .docker-log-terminal-shell {
    height: min(70vh, 520px);
    min-height: 280px;
  }

  .docker-log-status {
    width: 100%;
    margin-left: 0;
  }
}
</style>
