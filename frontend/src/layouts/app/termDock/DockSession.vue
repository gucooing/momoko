<!-- 悬浮终端里的一个标签：xterm + 传输层。本机终端连 /api/v1/shell/ws（指定了目录带 ?cwd=）；SSH 先取主机详情拿 wsPath 再连（?hostID=）。
     协议同 SSH 终端：输入原样发文本帧，尺寸发 {"type":"resize"}，服务端文本帧写进终端。
     卸载即断开（关标签、关窗口）；连接断了在屏幕上盖一层「重新连接」。
     外观跟着窗口设置：字号用悬浮终端自己的；背景半透明时 xterm 不画底色，由这一层按不透明度画。 -->
<template>
  <div class="tds" :class="{ 'is-clear': dock.look.opacity < 1 }" :style="{ '--tds-bg': screenBg }">
    <div ref="containerRef" class="tds__screen" />
    <div v-if="showOverlay" class="tds__over">
      <span class="tds__over-t">{{ tab.status === 'error' ? t('termDock.failed') : t('termDock.closed') }}</span>
      <button type="button" class="ag-btn ag-btn--p ag-btn--sm" @click="reconnect">
        <UIcon name="i-lucide-rotate-cw" />{{ t('termDock.reconnect') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, toRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { getSshHostInfo } from '@/api/openssh'
import { SHELL_WS_PATH } from '@/api/shell'
import { buildBackendWebSocketUrl } from '@/utils/websocket'
import { terminalThemeFor, useTerminalX } from '@/components/terminal/useTerminalX'
import { useTerminalTheme } from '@/components/terminal/useTerminalTheme'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import { dock, updateTab, type DockTab } from './state'

defineOptions({ name: 'DockSession' })

const props = defineProps<{ tab: DockTab }>()

const { t } = useI18n()
const themeStore = useThemeStore()
const { theme } = useTerminalTheme()

let ws: WebSocket | null = null
const everConnected = ref(false)

const setStatus = (status: ConsoleSocketStatus) => updateTab(props.tab.key, { status })

const send = (data: string | Uint8Array) => {
  if (ws?.readyState === WebSocket.OPEN) ws.send(data)
}

const { containerRef, mount, write, writeln, focus, fit, cols, rows } = useTerminalX({
  theme,
  onData: send,
  onBinary: send,
  onResize: (size) => send(JSON.stringify({ type: 'resize', ...size })),
  fontSizeRef: toRef(dock.look, 'fontSize'),
})

// 屏幕留白和终端同色
const screenBg = computed(
  () => terminalThemeFor(theme.value, themeStore.hasTerminalTokens ? themeStore.tokens : null).background || '#0f0b14',
)

const showOverlay = computed(
  () => props.tab.status === 'error' || (everConnected.value && props.tab.status === 'disconnected'),
)

const resolveUrl = async () => {
  if (props.tab.kind === 'local') {
    const cwd = props.tab.cwd
    return buildBackendWebSocketUrl(SHELL_WS_PATH, (url) => {
      if (cwd) url.searchParams.set('cwd', cwd)
    })
  }
  const { data } = await getSshHostInfo({ id: props.tab.hostId })
  const info = data?.info
  if (!info?.wsPath) return ''
  updateTab(props.tab.key, { label: `${info.username}@${info.name || info.host}` })
  return buildBackendWebSocketUrl(info.wsPath, (url) => url.searchParams.set('hostID', props.tab.hostId))
}

const connect = async () => {
  setStatus('connecting')
  let url = ''
  try {
    url = await resolveUrl()
  } catch {
    url = ''
  }
  if (!url) {
    setStatus('error')
    return
  }

  const socket = new WebSocket(url)
  ws = socket

  socket.onopen = () => {
    if (ws !== socket) return
    setStatus('connected')
    everConnected.value = true
    // fit 发生在连上之前，这里补发一次当前尺寸
    if (cols.value > 0 && rows.value > 0) socket.send(JSON.stringify({ type: 'resize', cols: cols.value, rows: rows.value }))
    focus()
  }
  socket.onmessage = (event) => {
    if (typeof event.data === 'string') write(event.data)
  }
  socket.onerror = () => {
    if (ws === socket) setStatus('error')
  }
  socket.onclose = () => {
    if (ws !== socket) return
    ws = null
    if (props.tab.status !== 'error') setStatus('disconnected')
    writeln('')
    writeln(`\x1b[31m${t('ssh.common.disconnectedNotice')}\x1b[0m`)
  }
}

const reconnect = () => {
  const socket = ws
  ws = null
  socket?.close()
  void connect()
}

onMounted(() => {
  mount()
  void connect()
})

onBeforeUnmount(() => {
  const socket = ws
  ws = null
  socket?.close()
})

defineExpose({ fit, focus })
</script>

<style scoped lang="scss">
.tds {
  position: absolute;
  inset: 0;
  padding: 6px 4px 4px 10px;
  // --td-alpha 由窗口给（不透明度设置），文字不受影响
  background: color-mix(in srgb, var(--tds-bg) var(--td-alpha, 100%), transparent);
}
// 半透明：xterm 自己的底色（滚动层上的行内样式）去掉，露出上面那层
.tds.is-clear :deep(.xterm),
.tds.is-clear :deep(.xterm-viewport),
.tds.is-clear :deep(.xterm-scrollable-element) {
  background-color: transparent !important;
}
.tds__screen {
  width: 100%;
  height: 100%;
}
.tds__over {
  position: absolute;
  inset: 0;
  z-index: 2;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  background: rgba(15, 11, 20, 0.72);
}
.tds__over-t {
  font-size: 13px;
  font-weight: 600;
  color: #e8e3ee;
}
</style>
