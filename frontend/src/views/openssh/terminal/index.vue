<!-- SSH 终端（重写 · 真 xterm.js）：调用公共组件 TerminalConsole，与实例控制台同一组件、同一协议。
     本页只负责 SSH 传输层：getSshHostInfo → wsPath → buildBackendWebSocketUrl → ws 直连，
     @data 原样发送、onmessage 写入、@resize 上报、onopen 补发初始尺寸。 -->
<template>
  <TerminalConsole
    ref="term"
    :title="connInfo.name || connInfo.username"
    :subtitle="`${connInfo.username}@${connInfo.host}:${connInfo.port}`"
    :status="socketStatus"
    :status-label="statusLabel"
    :actions="actions"
    @data="onData"
    @resize="onResize"
    @action="onAction"
  >
    <template #footer-right>
      <span v-if="socketStatus !== 'connected'" class="term-hint">
        {{ t('ssh.common.click') }}
        <button type="button" class="term-hint__link" @click="reconnect">
          {{ t('ssh.common.here') }}
        </button>
        {{ t('ssh.common.reconnectSuffix') }}
      </span>
    </template>
  </TerminalConsole>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { getSshHostInfo } from '@/api/openssh'
import { buildBackendWebSocketUrl } from '@/utils/websocket'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import type { TerminalAction } from '@/components/terminal/TerminalShell.vue'
import TerminalConsole from '@/components/terminal/TerminalConsole.vue'

defineOptions({ name: 'SshTerminalView' })

const route = useRoute()
const tabsStore = useTabsStore()
const { t } = useI18n()

const socketStatus = ref<ConsoleSocketStatus>('disconnected')
const connId = computed(() => (route.query.id as string) || '')
const connInfo = ref({ host: '-', port: '22', username: 'root', name: '' })
const wsPath = ref('')

let ws: WebSocket | null = null

const term = ref<InstanceType<typeof TerminalConsole>>()

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
      return t('ssh.common.disconnected')
  }
})

const actions = computed<TerminalAction[]>(() => [
  {
    key: 'reconnect',
    icon: 'HOutline:ArrowPathIcon',
    label: t('ssh.common.reconnect'),
    hidden: socketStatus.value === 'connected',
  },
])

const onAction = (key: string) => {
  if (key === 'reconnect') reconnect()
}

const buildWsUrl = () => {
  if (!connId.value || !wsPath.value) return ''
  return buildBackendWebSocketUrl(wsPath.value, (url) => {
    url.searchParams.set('hostID', connId.value)
  })
}

const connect = async () => {
  if (!connId.value) {
    feedback.warning(t('ssh.common.missingParams'))
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
      const tab = tabsStore.tabs.find((item) => item.fullPath === route.fullPath)
      if (tab) tab.title = info.name || `${info.username}@${info.host}`
    }
  } catch {
    wsPath.value = ''
  }

  const url = buildWsUrl()
  if (!url) {
    socketStatus.value = 'disconnected'
    feedback.warning(t('ssh.common.missingParams'))
    return
  }

  const socket = new WebSocket(url)
  ws = socket

  socket.onopen = () => {
    socketStatus.value = 'connected'
    const terminal = term.value
    // fit 发生在连接建立前，onResize 不会再触发，这里补发一次当前尺寸
    if (terminal && terminal.cols > 0 && terminal.rows > 0) {
      socket.send(JSON.stringify({ type: 'resize', cols: terminal.cols, rows: terminal.rows }))
    }
    terminal?.focus()
  }

  socket.onmessage = async (event) => {
    const text =
      typeof event.data === 'string'
        ? event.data
        : event.data instanceof Blob
          ? await event.data.text()
          : new TextDecoder().decode(event.data)
    term.value?.write(text)
  }

  socket.onerror = () => {
    socketStatus.value = 'error'
  }

  socket.onclose = () => {
    ws = null
    if (socketStatus.value !== 'disconnected') {
      socketStatus.value = 'disconnected'
      term.value?.writeln('')
      term.value?.writeln(`\x1b[31m${t('ssh.common.disconnectedNotice')}\x1b[0m`)
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
  window.setTimeout(() => void connect(), 200)
}

onMounted(() => {
  if (connId.value) window.setTimeout(() => void connect(), 200)
})

onBeforeUnmount(disconnect)
</script>

<style scoped lang="scss">
.term-hint {
  color: var(--term-fg-faint);
}
.term-hint__link {
  border: none;
  background: transparent;
  padding: 0;
  color: var(--term-accent);
  cursor: pointer;
  font: inherit;
  text-decoration: underline;
}
</style>
