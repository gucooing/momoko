<!-- 控制台：托管进程的伪终端（复用应用实例的控制台 WebSocket 协议与 TerminalConsole 组件），
     多个进程（游戏服务端 / 资源服务器）时可切换；底部是一行命令输入。 -->
<template>
  <div class="ct">
    <div v-if="roles.length > 1" class="ct__roles" role="tablist" :aria-label="t('gameServers.console.roles')">
      <button
        v-for="r in roles"
        :key="r.role"
        type="button"
        role="tab"
        class="ct__role"
        :class="{ 'is-on': role === r.role }"
        :aria-selected="role === r.role"
        @click="switchRole(r.role)"
      >
        {{ t(`gameServers.role.${r.role}`, r.role) }}
        <StatusPill :variant="stateVariant(r.state)" :label="t(`gameServers.state.${r.state}`, r.state)" />
      </button>
    </div>
    <div class="ct__term">
      <TerminalConsole
        ref="term"
        variant="console"
        :title="t('gameServers.console.title', { role: t(`gameServers.role.${role}`, role) })"
        :subtitle="subtitle"
        :status="status"
        :status-label="statusLabel"
        :actions="actions"
        @data="sendRaw"
        @resize="({ cols, rows }) => sendResize(cols, rows)"
        @action="onAction"
      >
        <template #below>
          <form class="ct__cmd" @submit.prevent="submit">
            <span class="ct__prompt m-mono" aria-hidden="true">&gt;</span>
            <input
              v-model="command"
              class="m-mono"
              type="text"
              autocomplete="off"
              spellcheck="false"
              enterkeyhint="send"
              :aria-label="t('gameServers.actions.console')"
              :disabled="status !== 'connected'"
            />
            <UButton type="submit" color="primary" icon="i-lucide-send-horizontal" :disabled="status !== 'connected' || !command.trim()" :aria-label="t('gameServers.actions.console')" />
          </form>
        </template>
      </TerminalConsole>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onActivated, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import TerminalConsole from '@/components/terminal/TerminalConsole.vue'
import type { TerminalAction } from '@/components/terminal/TerminalShell.vue'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import { gameServerConsolePath } from '@/api/gameServer'
import { normalizeAuthToken, toBearerAuthHeader } from '@/utils/request'
import type { GameAdapter, GameServerInfo } from '@/types/v1/game_server'
import { stateVariant } from '../../shared'

const props = defineProps<{ server: GameServerInfo; adapter?: GameAdapter | null }>()
const { t } = useI18n()

const term = ref<InstanceType<typeof TerminalConsole>>()
const roles = computed(() => props.server.processes || [])
const role = ref(roles.value[0]?.role || 'main')
const status = ref<ConsoleSocketStatus>('disconnected')
const command = ref('')
let socket: WebSocket | null = null
let decoder = new TextDecoder('utf-8')
let retry: ReturnType<typeof setTimeout> | undefined
let closing = false

const proc = computed(() => roles.value.find((r) => r.role === role.value))
const subtitle = computed(() => [proc.value?.command, proc.value?.pid ? `PID ${proc.value.pid}` : ''].filter(Boolean).join(' · '))
const statusLabel = computed(() =>
  t(status.value === 'connected' ? 'gameServers.console.connected' : status.value === 'connecting' ? 'gameServers.console.connecting' : 'gameServers.console.disconnected'),
)
const actions = computed<TerminalAction[]>(() => [
  { key: 'reconnect', icon: 'i-lucide-plug-zap', label: t('gameServers.console.reconnect'), hidden: status.value === 'connected' || status.value === 'connecting' },
])

// 浏览器 WebSocket 不能带请求头：token 走查询参数（与应用控制台一致）。不附加节点参数：托管进程在主控本机。
const socketUrl = () => {
  const api = new URL(import.meta.env.VITE_API_BASE_URL || window.location.origin, window.location.origin)
  const url = new URL(gameServerConsolePath, `${api.protocol === 'https:' ? 'wss:' : 'ws:'}//${api.host}`)
  url.searchParams.set('id', props.server.brief?.id || '')
  url.searchParams.set('role', role.value)
  const token = normalizeAuthToken(localStorage.getItem('accessToken') || '')
  if (token) url.searchParams.set('accessToken', toBearerAuthHeader(token))
  return url.toString()
}

const disconnect = () => {
  clearTimeout(retry)
  closing = true
  socket?.close()
  socket = null
  closing = false
}

const connect = () => {
  disconnect()
  if (!props.server.brief?.id) return
  decoder = new TextDecoder('utf-8')
  term.value?.reset()
  status.value = 'connecting'
  const ws = new WebSocket(socketUrl())
  ws.binaryType = 'arraybuffer'
  socket = ws
  ws.onopen = () => {
    if (socket !== ws) return
    status.value = 'connected'
    const cols = term.value?.cols
    const rows = term.value?.rows
    if (cols && rows) sendResize(Number(cols), Number(rows))
  }
  ws.onmessage = (e) => {
    if (socket !== ws) return
    const text = typeof e.data === 'string' ? e.data : decoder.decode(e.data as ArrayBuffer, { stream: true })
    term.value?.write(text)
  }
  ws.onclose = () => {
    if (socket !== ws || closing) return
    socket = null
    status.value = 'disconnected'
    retry = setTimeout(connect, 5000)
  }
  ws.onerror = () => {
    if (socket === ws) status.value = 'error'
  }
}

const sendRaw = (data: string | Uint8Array) => {
  if (!data.length || !socket || socket.readyState !== WebSocket.OPEN) return
  socket.send(data)
}
const sendResize = (cols: number, rows: number) => {
  if (cols <= 0 || rows <= 0 || !socket || socket.readyState !== WebSocket.OPEN) return
  socket.send(JSON.stringify({ type: 'resize', cols, rows }))
}
const submit = () => {
  const cmd = command.value.trim()
  if (!cmd) return
  sendRaw(`${cmd}\r`)
  command.value = ''
}
const onAction = (key: string) => {
  if (key === 'reconnect') connect()
}
const switchRole = (r: string) => {
  if (r === role.value) return
  role.value = r
  connect()
}

onMounted(() => nextTick(connect))
onActivated(() => {
  if (!socket) connect()
  term.value?.fit()
})
onBeforeUnmount(disconnect)
</script>

<style scoped lang="scss">
.ct {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.ct__roles {
  display: flex;
  gap: 8px;
  overflow-x: auto;
  scrollbar-width: none;
}
.ct__roles::-webkit-scrollbar {
  display: none;
}
.ct__role {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 12px 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-weight: 700;
  cursor: pointer;
}
.ct__role.is-on {
  border-color: var(--m-p);
  color: var(--el-text-color-primary);
}
.ct__term {
  height: min(70vh, 640px);
  min-height: 420px;
  display: flex;
  flex-direction: column;
  border-radius: var(--app-radius-lg);
  overflow: hidden;
}
.ct__term > :deep(*) {
  flex: 1;
  min-height: 0;
}
.ct__cmd {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 10px;
}
.ct__prompt {
  color: var(--el-text-color-secondary);
}
.ct__cmd input {
  flex: 1;
  min-width: 0;
  min-height: 40px;
  border: none;
  outline: none;
  background: transparent;
  color: inherit;
  font-size: 0.875rem;
}
</style>
