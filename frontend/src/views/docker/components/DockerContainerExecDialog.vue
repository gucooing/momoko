<!-- 容器终端：DockerTermSheet 外壳（PC 居中大面板 / 手机底部面板）+ 复用 TerminalConsole。
     协议与原实现一致：ws 原始输入(onData→send) / 原始字节输出(ArrayBuffer→持久 TextDecoder→write)，不发 resize 帧。
     终端顶栏左侧：要执行的命令（回车或「连接」用新命令重连）+ 常用 shell 快捷切换。保留 props/emits 契约。 -->
<template>
  <DockerTermSheet
    v-model="visible"
    :title="dialogTitle"
    :subtitle="t('docker.terminalDialog.sub')"
    icon="HOutline:CommandLineIcon"
    tone="ok"
  >
    <TerminalConsole
      ref="termRef"
      :title="dialogTitle"
      icon="HOutline:CommandLineIcon"
      :status="socketStatus"
      :status-label="statusLabel"
      :actions="actions"
      @data="onData"
      @action="onAction"
    >
      <template #lead>
        <div class="dk-execopts">
          <label class="dk-execopts__cmd">
            <span class="dk-execopts__prompt m-mono">$</span>
            <input
              v-model="commandText"
              class="m-mono"
              placeholder="/bin/sh"
              :aria-label="t('docker.terminalDialog.shellLabel')"
              @keyup.enter="reconnect"
            />
            <button type="button" class="dk-execopts__go" :disabled="socketStatus === 'connecting'" @click="reconnect">
              {{ t('docker.terminalDialog.connect') }}
            </button>
          </label>
          <div class="dk-execopts__shells" role="group" :aria-label="t('docker.terminalDialog.quickShell')">
            <button
              v-for="sh in QUICK_SHELLS"
              :key="sh"
              type="button"
              class="dk-execopts__shell m-mono"
              :class="{ 'is-on': commandText.trim() === sh }"
              @click="useShell(sh)"
            >{{ sh.split('/').pop() }}</button>
          </div>
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

const QUICK_SHELLS = ['/bin/sh', '/bin/bash', '/bin/ash']
const termRef = ref<InstanceType<typeof TerminalConsole> | null>(null)
const commandText = ref('/bin/sh')
const socketStatus = ref<ConsoleSocketStatus>('disconnected')

let socket: WebSocket | null = null
let decoder: TextDecoder | null = null
let manualClose = false

const dialogTitle = computed(() => {
  const name = props.containerName || props.containerId
  return name ? t('docker.terminalDialog.titleWithName', { name }) : t('docker.terminalDialog.title')
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

const execCommand = () => {
  const cmd = commandText.value.trim().split(/\s+/).filter(Boolean)
  return cmd.length ? cmd : ['/bin/sh']
}

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
    url.searchParams.set('tty', 'true')
    execCommand().forEach((item) => url.searchParams.append('cmd', item))
  })
  if (!socketUrl) { socketStatus.value = 'error'; return }

  const next = new WebSocket(socketUrl)
  next.binaryType = 'arraybuffer'
  socket = next
  next.onopen = () => { if (socket === next) { socketStatus.value = 'connected'; termRef.value?.focus() } }
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
const useShell = (sh: string) => {
  commandText.value = sh
  reconnect()
}
const onData = (data: string | Uint8Array) => { if (socket?.readyState === WebSocket.OPEN) socket.send(data) }
const onAction = (key: string) => {
  if (key === 'reconnect') reconnect()
  else if (key === 'clear') termRef.value?.reset()
  else if (key === 'close') close()
}
const close = () => { visible.value = false }

watch(visible, (open) => {
  if (open) {
    nextTick(() => { termRef.value?.reset(); connect() })
  } else {
    closeSocket()
  }
})
watch([() => props.containerId, () => props.wsPath], () => { if (visible.value) reconnect() })

onBeforeUnmount(closeSocket)
</script>

<style scoped lang="scss">
.dk-execopts { display: flex; align-items: center; gap: 10px; min-width: 0; }
.dk-execopts__cmd {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 340px;
  min-width: 0;
  height: 40px;
  padding: 0 4px 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  input {
    flex: 1;
    min-width: 0;
    border: none;
    background: transparent;
    color: var(--el-text-color-primary);
    font-size: 0.8125rem;
    font-weight: 600;
    outline: none;
  }
}
.dk-execopts__cmd:focus-within { border-color: var(--m-p); }
.dk-execopts__prompt { color: var(--m-ok); font-weight: 800; }
.dk-execopts__go {
  flex: none;
  height: 32px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.dk-execopts__go:disabled { opacity: 0.6; cursor: default; }
.dk-execopts__shells { display: inline-flex; flex: none; gap: 2px; padding: 3px; border-radius: 999px; background: var(--el-fill-color-light); }
.dk-execopts__shell {
  min-height: 32px;
  padding: 0 11px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
}
.dk-execopts__shell.is-on { background: var(--el-bg-color); color: var(--el-text-color-primary); box-shadow: var(--app-shadow-sm); }
@media (width <= 768px) {
  .dk-execopts { width: 100%; }
  .dk-execopts__cmd { flex: 1 1 auto; }
  .dk-execopts__shells { display: none; }
}
</style>
