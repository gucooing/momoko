<!-- 单个 SSH 会话：TerminalConsole + SSH 传输层（getSshHostInfo → wsPath → buildBackendWebSocketUrl → ws 直连，
     @data 原样发送、文本帧写入终端、@resize 上报、onopen 补发初始尺寸）。
     服务端控制帧走 WebSocket 二进制帧（JSON），需在握手 URL 中订阅：latency=1 → {"type":"latency","ms":N}（momoko → SSH 服务端往返延迟）。
     顶栏工具按原型：复制 / 粘贴 / 字号 / 明暗 / 全屏，最右是「断开 / 重连」药丸；底栏：状态 · 地址 · 终端类型 · 列×行 · 延迟 · 会话时长 · 认证方式。
     手机：顶栏由 #lead 放主机与状态（插槽参数 latency / cols / rows），无底栏，终端满屏、下方按键条。
     同一页面可同时开多个会话（由 index.vue 用 v-show 切换）。暴露 sendCommand(cmd) 给「快捷命令」。 -->
<template>
  <TerminalConsole
    ref="term"
    class="ssh-session"
    :title="connInfo.name || connInfo.username"
    :subtitle="addr"
    :status="socketStatus"
    :status-label="statusLabel"
    :actions="actions"
    actions-end
    compact-action-labels
    :compact-footer="false"
    @data="onData"
    @resize="onResize"
    @action="onAction"
  >
    <template v-if="$slots.lead" #lead>
      <slot name="lead" :latency="latencyMs" :cols="term?.cols || 0" :rows="term?.rows || 0" />
    </template>

    <template #overlay>
      <div v-if="showOverlay" class="ssh-session__overlay">
        <div class="ssh-session__overlay-box">
          <span class="ssh-session__overlay-title">
            {{ socketStatus === 'error' ? t('ssh.common.connectionFailed') : t('ssh.term.closedTitle') }}
          </span>
          <span class="ssh-session__overlay-desc">{{ t('ssh.term.closedDesc') }}</span>
          <UButton color="primary" size="lg" icon="i-lucide-rotate-cw" @click="reconnect">
            {{ t('ssh.term.reconnectNow') }}
          </UButton>
        </div>
      </div>
    </template>

    <template #footer-meta>
      <span v-if="socketStatus === 'connected' && latencyMs" class="ssh-session__meta m-num">
        <UIcon name="i-lucide-activity" />{{ t('ssh.term.latency', { ms: latencyMs }) }}
      </span>
    </template>

    <template #footer-right>
      <span class="ssh-session__foot">
        <span v-if="startedAt" class="m-num">{{ t('ssh.term.sessionDuration', { time: durationText }) }}</span>
        <span v-if="authLabel">{{ authLabel }}</span>
      </span>
    </template>
  </TerminalConsole>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useIntervalFn } from '@vueuse/core'
import { getSshHostInfo } from '@/api/openssh'
import { buildBackendWebSocketUrl } from '@/utils/websocket'
import { SSHAuthType, type SSHHostInfo } from '@/types/v1/openssh'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import type { TerminalAction } from '@/components/terminal/TerminalShell.vue'
import TerminalConsole from '@/components/terminal/TerminalConsole.vue'

defineOptions({ name: 'SshSession' })

const props = defineProps<{ hostId: string }>()
const emit = defineEmits<{
  info: [info: SSHHostInfo]
  status: [status: ConsoleSocketStatus]
}>()

const { t } = useI18n()

const socketStatus = ref<ConsoleSocketStatus>('disconnected')
const connInfo = ref({ host: '-', port: '22', username: 'root', name: '', authType: '' as string, keyType: '' })
const wsPath = ref('')
const everConnected = ref(false)
const startedAt = ref(0)
const now = ref(Date.now())
const latencyMs = ref(0)

let ws: WebSocket | null = null
const term = ref<InstanceType<typeof TerminalConsole>>()

const setStatus = (s: ConsoleSocketStatus) => {
  socketStatus.value = s
  emit('status', s)
}

const addr = computed(() => `${connInfo.value.username}@${connInfo.value.host}:${connInfo.value.port}`)

const authLabel = computed(() => {
  if (connInfo.value.authType === SSHAuthType.SSH_AUTH_TYPE_KEY) {
    return connInfo.value.keyType ? `${t('ssh.term.authKey')} · ${connInfo.value.keyType}` : t('ssh.term.authKey')
  }
  if (connInfo.value.authType === SSHAuthType.SSH_AUTH_TYPE_PASSWORD) return t('ssh.term.authPassword')
  return ''
})

useIntervalFn(() => {
  now.value = Date.now()
}, 1000)

const durationText = computed(() => {
  const total = Math.max(0, Math.floor((now.value - startedAt.value) / 1000))
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(Math.floor(total / 3600))}:${pad(Math.floor((total % 3600) / 60))}:${pad(total % 60)}`
})

const showOverlay = computed(
  () => (everConnected.value && socketStatus.value === 'disconnected') || socketStatus.value === 'error',
)

const onData = (data: string | Uint8Array) => {
  if (ws?.readyState === WebSocket.OPEN) ws.send(data)
}

const onResize = ({ cols, rows }: { cols: number; rows: number }) => {
  if (ws?.readyState === WebSocket.OPEN) ws.send(JSON.stringify({ type: 'resize', cols, rows }))
}

const statusLabel = computed(() => {
  switch (socketStatus.value) {
    case 'connected':
      return t('ssh.common.connected')
    case 'connecting':
      return t('ssh.common.connecting')
    case 'error':
      return t('ssh.common.connectionFailed')
    default:
      return t('ssh.term.closed')
  }
})

const actions = computed<TerminalAction[]>(() => {
  const live = socketStatus.value === 'connected' || socketStatus.value === 'connecting'
  return [
    live
      ? { key: 'disconnect', icon: 'HOutline:PowerIcon', label: t('ssh.term.disconnectShort'), tone: 'danger' }
      : { key: 'reconnect', icon: 'HOutline:ArrowPathIcon', label: t('ssh.term.reconnectShort'), tone: 'primary' },
  ]
})

const onAction = (key: string) => {
  if (key === 'reconnect') reconnect()
  else if (key === 'disconnect') disconnect()
}

const buildWsUrl = () => {
  if (!props.hostId || !wsPath.value) return ''
  return buildBackendWebSocketUrl(wsPath.value, (url) => {
    url.searchParams.set('hostID', props.hostId)
    url.searchParams.set('latency', '1')
  })
}

// 服务端控制帧（二进制 JSON）
const onControl = (raw: ArrayBuffer) => {
  try {
    const msg = JSON.parse(new TextDecoder().decode(raw)) as { type?: string; ms?: number }
    if (msg.type === 'latency' && typeof msg.ms === 'number') latencyMs.value = Math.max(1, Math.round(msg.ms))
  } catch {
    /* 非控制帧的二进制数据忽略 */
  }
}

/** 重新拉取主机信息（快捷命令等改了之后刷新），不影响当前连接 */
const loadInfo = async () => {
  try {
    const { data } = await getSshHostInfo({ id: props.hostId })
    if (!data?.info) return null
    const info = data.info
    wsPath.value = info.wsPath
    connInfo.value = {
      host: info.host,
      port: String(info.port),
      username: info.username,
      name: info.name,
      authType: info.authType,
      keyType: info.keyType,
    }
    emit('info', info)
    return info
  } catch {
    return null
  }
}

const connect = async () => {
  if (!props.hostId) {
    feedback.warning(t('ssh.common.missingParams'))
    return
  }

  setStatus('connecting')
  latencyMs.value = 0
  if (!(await loadInfo())) wsPath.value = ''

  const url = buildWsUrl()
  if (!url) {
    setStatus('disconnected')
    feedback.warning(t('ssh.common.missingParams'))
    return
  }

  const socket = new WebSocket(url)
  socket.binaryType = 'arraybuffer'
  ws = socket

  socket.onopen = () => {
    if (ws !== socket) return
    setStatus('connected')
    everConnected.value = true
    startedAt.value = Date.now()
    const terminal = term.value
    // fit 发生在连接建立前，onResize 不会再触发，这里补发一次当前尺寸
    if (terminal && terminal.cols > 0 && terminal.rows > 0) {
      socket.send(JSON.stringify({ type: 'resize', cols: terminal.cols, rows: terminal.rows }))
    }
    terminal?.focus()
  }

  socket.onmessage = (event) => {
    if (typeof event.data === 'string') term.value?.write(event.data)
    else if (event.data instanceof ArrayBuffer) onControl(event.data)
  }

  socket.onerror = () => {
    if (ws !== socket) return
    setStatus('error')
  }

  socket.onclose = () => {
    if (ws !== socket) return
    ws = null
    startedAt.value = 0
    latencyMs.value = 0
    if (socketStatus.value !== 'disconnected') {
      if (socketStatus.value !== 'error') setStatus('disconnected')
      term.value?.writeln('')
      term.value?.writeln(`\x1b[31m${t('ssh.common.disconnectedNotice')}\x1b[0m`)
    }
  }
}

const disconnect = () => {
  const socket = ws
  ws = null
  startedAt.value = 0
  latencyMs.value = 0
  setStatus('disconnected')
  if (socket) {
    socket.close()
    term.value?.writeln('')
    term.value?.writeln(`\x1b[31m${t('ssh.common.disconnectedNotice')}\x1b[0m`)
  }
}

const reconnect = () => {
  disconnect()
  window.setTimeout(() => void connect(), 200)
}

/** 快捷命令：整行发送并回车 */
const sendCommand = (cmd: string) => {
  if (ws?.readyState !== WebSocket.OPEN) {
    feedback.warning(t('ssh.term.notConnected'))
    return
  }
  ws.send(`${cmd}\r`)
  term.value?.focus()
}

const focus = () => term.value?.focus()
const fit = () => term.value?.fit()

onMounted(() => {
  if (props.hostId) window.setTimeout(() => void connect(), 200)
})

onBeforeUnmount(() => {
  const socket = ws
  ws = null
  socket?.close()
})

defineExpose({ reconnect, disconnect, focus, fit, sendCommand, reloadInfo: loadInfo })
</script>

<style scoped lang="scss">
.ssh-session__overlay {
  position: absolute;
  inset: 0;
  z-index: 5;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 24px;
  background: rgba(15, 11, 20, 0.78);
}
.ssh-session__overlay-box {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 12px;
  text-align: center;
}
.ssh-session__overlay-title {
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  color: #e8e3ee;
}
.ssh-session__overlay-desc {
  font-size: 0.875rem;
  color: #b8b0c2;
}
.ssh-session__meta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.ssh-session__meta :deep(.iconify) {
  width: 14px;
  height: 14px;
}
.ssh-session__foot {
  display: inline-flex;
  align-items: center;
  gap: 18px;
}

/* 手机（按原型）：终端满屏 —— 顶栏实底 + 下边线，屏幕无圆角，按键条实底 + 上边线 */
@media (width <= 768px) {
  .ssh-session.term-shell {
    gap: 0;
    padding: 0;
    border: 0;
    border-radius: 0;
  }
  .ssh-session :deep(.term-shell__bar) {
    min-height: 60px;
    padding: 6px 10px 6px 6px;
    background: var(--el-bg-color);
    border-bottom: 1px solid var(--el-border-color-light);
  }
  .ssh-session :deep(.term-shell__lead) {
    gap: 2px;
  }
  .ssh-session :deep(.term-shell__screen) {
    border-radius: 0;
  }
  .ssh-session :deep(.term-shell__keysrow) {
    margin: 0;
    padding: 8px 0 8px 8px;
    gap: 0;
    background: var(--el-bg-color);
    border-top: 1px solid var(--el-border-color-light);
  }
  .ssh-session :deep(.term-shell__keys) {
    padding-right: 8px;
  }
  .ssh-session :deep(.term-shell__kbd) {
    width: 52px;
    height: 60px;
    margin: -8px 0;
  }
}
</style>
