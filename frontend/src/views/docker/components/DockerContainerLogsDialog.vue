<!-- 容器日志：DockerTermSheet 外壳（PC 居中大面板 / 手机底部面板）+ TerminalConsole（只读流）。
     协议：ws 原始字节输出 → 持久 TextDecoder → write；不发输入/resize。
     终端顶栏左侧放 最近行数（快捷档 + 自定义）与 时间戳开关；右侧 重连/清空；tail/时间戳变化即重连。 -->
<template>
  <DockerTermSheet
    v-model="visible"
    :title="dialogTitle"
    :subtitle="t('docker.logsDialog.sub')"
    icon="HOutline:DocumentTextIcon"
    tone="a2"
  >
    <TerminalConsole
      ref="termRef"
      :title="dialogTitle"
      icon="HOutline:DocumentTextIcon"
      :status="socketStatus"
      :status-label="statusLabel"
      :actions="actions"
      @action="onAction"
    >
      <template #lead>
        <div class="dk-logopts">
          <div class="dk-logopts__seg" role="radiogroup" :aria-label="t('docker.logsDialog.tailLabel')">
            <button
              v-for="n in TAIL_PRESETS"
              :key="n"
              type="button"
              role="radio"
              class="dk-logopts__btn m-num"
              :class="{ 'is-on': tail === n }"
              :aria-checked="tail === n"
              @click="tail = n"
            >{{ n }}</button>
          </div>
          <label class="dk-logopts__num">
            <span>tail</span>
            <input
              v-model.lazy.number="tail"
              type="number"
              min="10"
              max="10000"
              step="100"
              class="m-num"
              :aria-label="t('docker.logsDialog.tailLabel')"
            />
          </label>
          <label class="dk-logopts__switch">
            <AppSwitch v-model="timestamps" />
            <span>{{ t('docker.logsDialog.timestamps') }}</span>
          </label>
        </div>
      </template>
    </TerminalConsole>
  </DockerTermSheet>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import TerminalConsole from '@/components/terminal/TerminalConsole.vue'
import DockerTermSheet from '@/views/docker/components/DockerTermSheet.vue'
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

const TAIL_PRESETS = [100, 200, 500, 1000]
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
.dk-logopts { display: flex; align-items: center; gap: 10px; min-width: 0; overflow-x: auto; scrollbar-width: none; }
.dk-logopts::-webkit-scrollbar { display: none; }
.dk-logopts__seg { display: inline-flex; flex: none; gap: 2px; padding: 3px; border-radius: 999px; background: var(--el-fill-color-light); }
.dk-logopts__btn {
  min-height: 32px;
  padding: 0 11px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.dk-logopts__btn.is-on { background: var(--el-bg-color); color: var(--el-text-color-primary); box-shadow: var(--app-shadow-sm); }
.dk-logopts__num {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 6px;
  height: 38px;
  padding: 0 4px 0 12px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  input {
    width: 64px;
    border: none;
    background: transparent;
    color: var(--el-text-color-primary);
    font-size: 0.8125rem;
    font-weight: 700;
    outline: none;
  }
}
.dk-logopts__num:focus-within { border-color: var(--m-p); }
.dk-logopts__switch {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-regular);
  white-space: nowrap;
  cursor: pointer;
}
@media (width <= 768px) {
  .dk-logopts__num { display: none; }
  .dk-logopts__btn { padding: 0 9px; }
}
</style>
