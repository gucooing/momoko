<!-- 容器日志（重写）：令牌浮层 + TerminalConsole（只读流）。
     协议：ws 原始字节输出 → 持久 TextDecoder → write；不发输入/resize。
     工具栏：tail 行数 + 时间戳开关 + 重连/清空；reconnect 用当前参数重连。 -->
<template>
  <Teleport to="body">
    <Transition name="dk-term">
      <div v-if="visible" class="dk-term-overlay" @mousedown.self="close">
        <div class="dk-term-panel">
          <div class="dk-term-bar">
            <span class="dk-term-bar__label">{{ t('docker.common.logs') }}</span>
            <div class="dk-term-bar__field">
              <span class="dk-term-bar__hint">tail</span>
              <input v-model.number="tail" type="number" min="10" max="10000" step="100" class="app-input dk-term-num" />
            </div>
            <label class="dk-term-check">
              <input v-model="timestamps" type="checkbox" />
              <span>{{ t('docker.logsDialog.timestamps') }}</span>
            </label>
          </div>
          <div class="dk-term-body">
            <TerminalConsole
              ref="termRef"
              :title="dialogTitle"
              icon="HOutline:DocumentTextIcon"
              :status="socketStatus"
              :status-label="statusLabel"
              :actions="actions"
              @action="onAction"
            />
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import TerminalConsole from '@/components/terminal/TerminalConsole.vue'
import type { TerminalAction } from '@/components/terminal/TerminalShell.vue'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import { buildBackendWebSocketUrl } from '@/utils/websocket'

const props = defineProps<{
  modelValue: boolean
  containerId: string
  wsPath: string
  containerName?: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const termRef = ref<InstanceType<typeof TerminalConsole> | null>(null)
const tail = ref(200)
const timestamps = ref(true)
const socketStatus = ref<ConsoleSocketStatus>('disconnected')

let socket: WebSocket | null = null
let decoder: TextDecoder | null = null
let manualClose = false

const dialogTitle = computed(() => {
  const name = props.containerName || props.containerId
  return name ? t('docker.logsDialog.titleWithName', { name }) : t('docker.logsDialog.title')
})
const statusLabel = computed(() => {
  if (socketStatus.value === 'connected') return t('docker.common.connected')
  if (socketStatus.value === 'connecting') return t('docker.common.connecting')
  if (socketStatus.value === 'error') return t('docker.common.connectionFailed')
  return t('docker.common.notConnected')
})
const actions = computed<TerminalAction[]>(() => [
  { key: 'reconnect', icon: 'HOutline:ArrowPathIcon', label: t('docker.common.reconnect'), tone: 'primary' },
  { key: 'clear', icon: 'HOutline:TrashIcon', label: t('docker.common.clear') },
  { key: 'close', icon: 'HOutline:XMarkIcon', label: t('system.common.close'), tone: 'danger' },
])

const writePayload = (payload: ArrayBuffer | string) => {
  const term = termRef.value
  if (!term) return
  if (typeof payload === 'string') term.write(payload)
  else if (decoder) term.write(decoder.decode(payload, { stream: true }))
}

const closeSocket = () => {
  if (!socket) return
  manualClose = true
  socket.close()
  socket = null
  socketStatus.value = 'disconnected'
}

const connect = () => {
  if (!props.containerId || !props.wsPath) return
  closeSocket()
  manualClose = false
  decoder = new TextDecoder('utf-8', { fatal: false })
  socketStatus.value = 'connecting'

  const socketUrl = buildBackendWebSocketUrl(props.wsPath, (url) => {
    url.searchParams.set('id', props.containerId)
    url.searchParams.set('tail', String(tail.value || 200))
    url.searchParams.set('timestamps', String(timestamps.value))
  })
  if (!socketUrl) { socketStatus.value = 'error'; return }

  const next = new WebSocket(socketUrl)
  next.binaryType = 'arraybuffer'
  socket = next
  next.onopen = () => { if (socket === next) socketStatus.value = 'connected' }
  next.onmessage = (event) => { if (socket === next) writePayload(event.data as ArrayBuffer | string) }
  next.onerror = () => { if (socket === next) socketStatus.value = 'error' }
  next.onclose = () => {
    const closedManually = manualClose
    if (socket !== next) return
    socket = null
    if (!closedManually) socketStatus.value = 'disconnected'
  }
}

const reconnect = () => { termRef.value?.reset(); connect() }
const onAction = (key: string) => {
  if (key === 'reconnect') reconnect()
  else if (key === 'clear') termRef.value?.reset()
  else if (key === 'close') close()
}
const close = () => { visible.value = false }

watch(visible, (open) => {
  if (open) nextTick(() => { termRef.value?.reset(); connect() })
  else closeSocket()
})
watch([() => props.containerId, () => props.wsPath, tail, timestamps], () => {
  if (visible.value) reconnect()
})

onBeforeUnmount(closeSocket)
</script>

<style scoped lang="scss">
.dk-term-overlay {
  position: fixed;
  inset: 0;
  z-index: 2200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: color-mix(in srgb, #0b1220 55%, transparent);
  backdrop-filter: blur(2px);
}
.dk-term-panel {
  display: flex;
  flex-direction: column;
  width: min(1000px, 96vw);
  height: min(80vh, 720px);
  border-radius: var(--app-radius-lg);
  overflow: hidden;
  box-shadow: var(--app-shadow-lg);
}
.dk-term-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
  padding: 8px 12px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-light);
}
.dk-term-bar__label {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.dk-term-bar__field {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.dk-term-bar__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.dk-term-num {
  width: 96px;
}
.dk-term-check {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
  user-select: none;
}
.dk-term-check input {
  accent-color: var(--el-color-primary);
}
.dk-term-body {
  position: relative;
  flex: 1;
  min-height: 0;
  display: flex;
}
.dk-term-body :deep(.term-shell) {
  flex: 1;
  min-height: 0;
  border-radius: 0;
}

.dk-term-enter-active,
.dk-term-leave-active { transition: opacity 0.18s ease; }
.dk-term-enter-active .dk-term-panel,
.dk-term-leave-active .dk-term-panel { transition: transform 0.18s cubic-bezier(0.4, 0, 0.2, 1); }
.dk-term-enter-from,
.dk-term-leave-to { opacity: 0; }
.dk-term-enter-from .dk-term-panel,
.dk-term-leave-to .dk-term-panel { transform: translateY(8px) scale(0.98); }

@media (width <= 768px) {
  .dk-term-overlay { padding: 0; }
  .dk-term-panel { width: 100vw; height: 100vh; border-radius: 0; }
}
</style>
