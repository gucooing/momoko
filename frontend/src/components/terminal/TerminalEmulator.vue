<template>
  <div class="terminal" :class="{ 'terminal--maximized': isMaximized }">
    <!-- Title Bar -->
    <div class="terminal__titlebar">
      <div class="terminal__traffic-lights">
        <button class="terminal__dot terminal__dot--green" :title="isMaximized ? '还原' : '最大化'" @click="isMaximized = !isMaximized" />
      </div>
      <div class="terminal__title">
        <el-icon class="terminal__title-icon"><component :is="titleIcon" /></el-icon>
        <span class="terminal__title-text">{{ titleText }}</span>
        <span v-if="titleSub" class="terminal__title-sub">{{ titleSub }}</span>
      </div>
      <div class="terminal__titlebar-right">
        <span v-if="instanceStatusTag" class="terminal__instance-tag" :class="instanceStatusTagClass">
          {{ instanceStatusTag }}
        </span>
        <span class="terminal__status-dot" :class="statusDotClass" />
        <span class="terminal__status-text">{{ statusLabel }}</span>
      </div>
    </div>

    <!-- Screen -->
    <div ref="screenRef" class="terminal__screen" @click="focusInput">
      <div class="terminal__content">
        <p v-for="(line, idx) in renderedLines" :key="idx" class="terminal__line">
          <template v-if="line.segments.length">
            <span
              v-for="(seg, si) in line.segments"
              :key="si"
              class="terminal__segment"
              :style="seg.style"
            >{{ seg.text }}</span>
          </template>
          <span v-else>&#8203;</span>
        </p>
        <p v-if="isConnecting" class="terminal__line terminal__line--dim">
          <span class="terminal__cursor terminal__cursor--blink">&#9608;</span>
          正在连接...
        </p>
        <p v-if="isError" class="terminal__line terminal__line--error">
          连接异常，点击
          <span class="terminal__inline-action" @click="emit('reconnect')">这里</span>
          重试
        </p>
      </div>
    </div>

    <!-- Input Line -->
    <div v-if="canType" class="terminal__input-bar">
      <span v-if="mode === 'ssh'" class="terminal__prompt">
        <span class="terminal__prompt-user">{{ username }}</span>
        <span class="terminal__prompt-sep">@</span>
        <span class="terminal__prompt-host">{{ host }}</span>
        <span class="terminal__prompt-path">:~$ </span>
      </span>
      <textarea
        ref="inputRef"
        v-model="inputBuffer"
        class="terminal__textarea"
        :rows="1"
        autofocus
        autocomplete="off"
        autocorrect="off"
        autocapitalize="off"
        spellcheck="false"
        :placeholder="sendPlaceholder"
        @keydown="handleKeydown"
        @input="autoResizeInput"
      />
    </div>

    <!-- Bottom Bar -->
    <div class="terminal__bottombar">
      <div class="terminal__bottombar-left">
        <!-- Mode: system-terminal -->
        <template v-if="mode === 'system-terminal'">
          <button
            class="terminal__action"
            :class="isRunning ? 'terminal__action--danger' : 'terminal__action--primary'"
            :disabled="isBusy"
            @click="emit('togglePower')"
          >
            <el-icon size="14"><component :is="isRunning ? powerOffIcon : powerOnIcon" /></el-icon>
            <span>{{ isRunning ? '停止' : '启动' }}</span>
          </button>
          <button class="terminal__action" :disabled="isBusy" @click="emit('restart')">
            <el-icon size="14"><component :is="restartIcon" /></el-icon>
            <span>重启</span>
          </button>
        </template>

        <!-- Mode: instance-console -->
        <template v-if="mode === 'instance-console'">
          <button
            class="terminal__action"
            :class="isRunning ? 'terminal__action--danger' : 'terminal__action--primary'"
            :disabled="isBusy"
            @click="emit('togglePower')"
          >
            <el-icon size="14"><component :is="isRunning ? powerOffIcon : powerOnIcon" /></el-icon>
            <span>{{ isRunning ? '停止' : '启动' }}</span>
          </button>
          <button class="terminal__action" :disabled="isBusy" @click="emit('restart')">
            <el-icon size="14"><component :is="restartIcon" /></el-icon>
            <span>重启</span>
          </button>
          <button class="terminal__action terminal__action--force" :disabled="isBusy" @click="emit('forceStop')">
            <el-icon size="14"><component :is="noSymbolIcon" /></el-icon>
            <span>强制停止</span>
          </button>
          <button class="terminal__action terminal__action--force" :disabled="isBusy" @click="emit('forceRestart')">
            <el-icon size="14"><component :is="boltIcon" /></el-icon>
            <span>强制重启</span>
          </button>
        </template>

        <!-- Common actions -->
        <span v-if="mode !== 'ssh'" class="terminal__action-sep" />
        <button class="terminal__action" :disabled="!outputLines.length" @click="emit('clear')">
          <el-icon size="14"><component :is="sparklesIcon" /></el-icon>
          <span>清屏</span>
        </button>

        <!-- Mode: ssh -->
        <template v-if="mode === 'ssh'">
          <button class="terminal__action" @click="emit('reconnect')">
            <el-icon size="14"><component :is="restartIcon" /></el-icon>
            <span>重连</span>
          </button>
        </template>
      </div>

      <div class="terminal__bottombar-right">
        <button
          v-for="feat in featureItems"
          :key="feat.key"
          class="terminal__feature-btn"
          :title="feat.description"
          @click="emit('feature', feat.key)"
        >
          <el-icon size="14"><component :is="menuStore.iconComponents[feat.icon]" /></el-icon>
        </button>
        <span v-if="mode === 'ssh'" class="terminal__cols-rows">{{ cols }}x{{ rows }}</span>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { ConsoleFeatureItem, ConsoleSocketStatus } from '@/stores/instance/types'
import { InstanceStatus } from '@/stores/instance/types'
import { parseAnsiOutputLines } from '@/utils/ansi'

const props = withDefaults(
  defineProps<{
    mode: 'ssh' | 'system-terminal' | 'instance-console'
    host?: string
    port?: string | number
    username?: string
    terminalName?: string
    terminalType?: string
    terminalId?: string
    instanceStatus?: string
    outputLines: string[]
    socketStatus: ConsoleSocketStatus
    isBusy?: boolean
    canSendCommand?: boolean
    featureItems?: ConsoleFeatureItem[]
    sendPlaceholder?: string
    cols?: number
    rows?: number
  }>(),
  {
    host: '-',
    port: '22',
    username: 'root',
    terminalName: '',
    terminalType: '',
    terminalId: '',
    instanceStatus: '',
    isBusy: false,
    canSendCommand: true,
    featureItems: () => [],
    sendPlaceholder: '输入命令并按 Enter 执行',
    cols: 120,
    rows: 32,
  },
)

const emit = defineEmits<{
  input: [text: string]
  togglePower: []
  restart: []
  forceStop: []
  forceRestart: []
  reconnect: []
  clear: []
  feature: [key: string]
}>()

const menuStore = useMenuStore()

// Icons
const titleIcon = computed(() => {
  if (props.mode === 'ssh') return menuStore.iconComponents['HOutline:CommandLineIcon']
  return menuStore.iconComponents['HOutline:CommandLineIcon']
})
const powerOnIcon = computed(() => menuStore.iconComponents['HOutline:PlayIcon'])
const powerOffIcon = computed(() => menuStore.iconComponents['HOutline:StopIcon'])
const restartIcon = computed(() => menuStore.iconComponents['HOutline:ArrowPathIcon'])
const sparklesIcon = computed(() => menuStore.iconComponents['HOutline:SparklesIcon'])
const noSymbolIcon = computed(() => menuStore.iconComponents['HOutline:NoSymbolIcon'])
const boltIcon = computed(() => menuStore.iconComponents['HOutline:BoltIcon'])

// Refs
const screenRef = ref<HTMLElement>()
const inputRef = ref<HTMLTextAreaElement>()
const inputBuffer = ref('')
const isMaximized = ref(false)
const history = ref<string[]>([])
const historyCursor = ref(-1)

// Computed
const isRunning = computed(() =>
  props.instanceStatus === InstanceStatus.INSTANCE_STATUS_RUNNING,
)

const canType = computed(() => {
  if (props.mode === 'ssh') return props.socketStatus === 'connected'
  return props.canSendCommand ?? false
})

const isConnecting = computed(() => props.socketStatus === 'connecting')
const isError = computed(() => props.socketStatus === 'error')

const titleText = computed(() => {
  if (props.mode === 'ssh') return props.username || 'root'
  return props.terminalName || '终端'
})

const titleSub = computed(() => {
  if (props.mode === 'ssh') return props.host ? `@${props.host}` : ''
  return props.terminalType ? `— ${props.terminalType}` : ''
})

const statusLabel = computed(() => {
  switch (props.socketStatus) {
    case 'connected': return '已连接'
    case 'connecting': return '连接中'
    case 'error': return '连接失败'
    default: return '未连接'
  }
})

const statusDotClass = computed(() => ({
  'terminal__status-dot--connected': props.socketStatus === 'connected',
  'terminal__status-dot--connecting': props.socketStatus === 'connecting',
  'terminal__status-dot--error': props.socketStatus === 'error',
  'terminal__status-dot--idle': props.socketStatus === 'disconnected',
}))

const instanceStatusTag = computed(() => {
  if (props.mode === 'ssh' || !props.instanceStatus) return ''
  const map: Record<string, string> = {
    [InstanceStatus.INSTANCE_STATUS_RUNNING]: '运行中',
    [InstanceStatus.INSTANCE_STATUS_STOPPED]: '已停止',
    [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: '维护中',
  }
  return map[props.instanceStatus] || ''
})

const instanceStatusTagClass = computed(() => ({
  'terminal__instance-tag--running': props.instanceStatus === InstanceStatus.INSTANCE_STATUS_RUNNING,
  'terminal__instance-tag--stopped': props.instanceStatus === InstanceStatus.INSTANCE_STATUS_STOPPED,
  'terminal__instance-tag--maintenance': props.instanceStatus === InstanceStatus.INSTANCE_STATUS_MAINTENANCE,
}))

const renderedLines = computed(() => parseAnsiOutputLines(props.outputLines))

// Input
const focusInput = () => {
  inputRef.value?.focus()
}

const autoResizeInput = () => {
  const el = inputRef.value
  if (!el) return
  el.style.height = 'auto'
  el.style.height = `${el.scrollHeight}px`
}

const scrollToBottom = () => {
  const el = screenRef.value
  if (!el) return
  el.scrollTop = el.scrollHeight
}

const handleKeydown = (e: KeyboardEvent) => {
  if (!canType.value) {
    e.preventDefault()
    return
  }

  if (e.key === 'Enter') {
    e.preventDefault()
    const text = inputBuffer.value
    if (props.mode === 'ssh') {
      if (text) {
        history.value.push(text)
        historyCursor.value = history.value.length
        emit('input', text + '\n')
      }
    } else {
      emit('input', text)
    }
    inputBuffer.value = ''
    nextTick(autoResizeInput)
    return
  }

  if (e.key === 'ArrowUp') {
    if (!history.value.length) return
    e.preventDefault()
    historyCursor.value = Math.max(0, historyCursor.value - 1)
    inputBuffer.value = history.value[historyCursor.value] || ''
    nextTick(autoResizeInput)
    return
  }

  if (e.key === 'ArrowDown') {
    if (!history.value.length) return
    e.preventDefault()
    if (historyCursor.value >= history.value.length - 1) {
      historyCursor.value = history.value.length
      inputBuffer.value = ''
    } else {
      historyCursor.value += 1
      inputBuffer.value = history.value[historyCursor.value] || ''
    }
    nextTick(autoResizeInput)
    return
  }

  if (props.mode === 'ssh') {
    if (e.key === 'Tab') {
      e.preventDefault()
      emit('input', '\t')
      return
    }
    if (e.key === 'c' && e.ctrlKey) {
      e.preventDefault()
      emit('input', '\x03')
      inputBuffer.value = ''
      return
    }
    if (e.key === 'd' && e.ctrlKey) {
      e.preventDefault()
      emit('input', '\x04')
      return
    }
  }
}

// Watchers
watch(
  () => [props.outputLines.length, props.outputLines[props.outputLines.length - 1]],
  () => nextTick(scrollToBottom),
)

watch(() => props.socketStatus, (status) => {
  if (status === 'connected') {
    nextTick(() => {
      focusInput()
      scrollToBottom()
    })
  }
})

onMounted(() => nextTick(scrollToBottom))
</script>

<style scoped>
/* ===== Layout ===== */
.terminal {
  display: flex;
  flex-direction: column;
  height: calc(100vh - 7.5rem);
  min-height: 28rem;
  border-radius: 10px;
  border: 1px solid var(--el-border-color);
  overflow: hidden;
  background: #0d1117;
  box-shadow:
    0 0 0 1px rgba(255, 255, 255, 0.03),
    0 4px 24px rgba(0, 0, 0, 0.4);
  transition: all 0.25s ease;
}

.terminal--maximized {
  position: fixed;
  inset: 0;
  z-index: 2000;
  height: 100vh;
  border-radius: 0;
  border: none;
}

/* ===== Title Bar ===== */
.terminal__titlebar {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 0 14px;
  height: 38px;
  background: #161b22;
  border-bottom: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
  user-select: none;
}

.terminal__traffic-lights {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.terminal__dot {
  width: 12px;
  height: 12px;
  border-radius: 50%;
  border: none;
  cursor: pointer;
  transition: filter 0.15s ease;
  padding: 0;
}

.terminal__dot:hover {
  filter: brightness(0.85);
}

.terminal__dot--green { background: #28c840; }

.terminal__title {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}

.terminal__title-icon {
  flex-shrink: 0;
  color: #58a6ff;
  font-size: 14px;
}

.terminal__title-text {
  color: #f0f6fc;
  font-size: 0.8rem;
  font-weight: 600;
  white-space: nowrap;
}

.terminal__title-sub {
  color: #8b949e;
  font-size: 0.76rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.terminal__titlebar-right {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.terminal__instance-tag {
  font-size: 0.66rem;
  padding: 2px 7px;
  border-radius: 4px;
  font-weight: 600;
}

.terminal__instance-tag--running {
  color: #3fb950;
  background: rgba(63, 185, 80, 0.12);
  border: 1px solid rgba(63, 185, 80, 0.25);
}

.terminal__instance-tag--stopped {
  color: #8b949e;
  background: rgba(139, 148, 158, 0.1);
  border: 1px solid rgba(139, 148, 158, 0.2);
}

.terminal__instance-tag--maintenance {
  color: #d29922;
  background: rgba(210, 153, 34, 0.12);
  border: 1px solid rgba(210, 153, 34, 0.25);
}

.terminal__status-dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}

.terminal__status-dot--connected {
  background: #3fb950;
  box-shadow: 0 0 6px rgba(63, 185, 80, 0.5);
}

.terminal__status-dot--connecting {
  background: #d29922;
  box-shadow: 0 0 6px rgba(210, 153, 34, 0.5);
  animation: terminal-blink 1s ease-in-out infinite;
}

.terminal__status-dot--error {
  background: #f85149;
  box-shadow: 0 0 6px rgba(248, 81, 73, 0.5);
}

.terminal__status-dot--idle {
  background: #484f58;
}

.terminal__status-text {
  font-size: 0.72rem;
  color: #8b949e;
  font-weight: 500;
}

/* ===== Screen ===== */
.terminal__screen {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 16px 18px 8px;
  background:
    radial-gradient(ellipse at 50% 0%, rgba(88, 166, 255, 0.04), transparent 60%),
    #0d1117;
  cursor: text;
}

.terminal__screen::-webkit-scrollbar { width: 6px; }
.terminal__screen::-webkit-scrollbar-track { background: transparent; }
.terminal__screen::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.08);
  border-radius: 3px;
}
.terminal__screen::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.14);
}

.terminal__content {
  display: flex;
  flex-direction: column;
}

.terminal__line {
  margin: 0;
  color: #c9d1d9;
  font-family: 'Cascadia Code', 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 0.82rem;
  line-height: 1.45;
  white-space: pre-wrap;
  word-break: break-all;
  min-height: 1.19rem;
}

.terminal__segment {
  white-space: inherit;
}

.terminal__line--dim {
  color: #484f58;
  font-style: italic;
}

.terminal__line--error {
  color: #f85149;
}

.terminal__cursor--blink {
  animation: terminal-cursor-blink 1s step-end infinite;
}

.terminal__inline-action {
  color: #58a6ff;
  cursor: pointer;
  text-decoration: underline;
}

.terminal__inline-action:hover {
  color: #79c0ff;
}

/* ===== Input Bar ===== */
.terminal__input-bar {
  display: flex;
  align-items: flex-start;
  gap: 0;
  padding: 8px 18px 12px;
  background: #0d1117;
  border-top: 1px solid rgba(255, 255, 255, 0.05);
  flex-shrink: 0;
}

.terminal__prompt {
  font-family: 'Cascadia Code', 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 0.82rem;
  line-height: 1.45;
  flex-shrink: 0;
  user-select: none;
  white-space: nowrap;
  padding-top: 2px;
}

.terminal__prompt-user { color: #7ee787; font-weight: 600; }
.terminal__prompt-sep { color: #484f58; }
.terminal__prompt-host { color: #79c0ff; font-weight: 600; }
.terminal__prompt-path { color: #c9d1d9; }

.terminal__textarea {
  flex: 1;
  background: transparent;
  border: none;
  outline: none;
  color: #c9d1d9;
  font-family: 'Cascadia Code', 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 0.82rem;
  line-height: 1.45;
  resize: none;
  padding: 2px 0 0;
  margin: 0;
  caret-color: #58a6ff;
  caret-shape: block;
  min-height: 1.19rem;
}

.terminal__textarea::placeholder {
  color: #30363d;
}

/* ===== Bottom Bar ===== */
.terminal__bottombar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0 10px;
  height: 34px;
  background: #161b22;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  flex-shrink: 0;
}

.terminal__bottombar-left,
.terminal__bottombar-right {
  display: flex;
  align-items: center;
  gap: 2px;
}

.terminal__action {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 4px 10px;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: #8b949e;
  font-size: 0.74rem;
  cursor: pointer;
  transition: all 0.15s ease;
  white-space: nowrap;
}

.terminal__action:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.06);
  color: #c9d1d9;
}

.terminal__action:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.terminal__action--primary {
  color: #58a6ff;
}

.terminal__action--primary:hover:not(:disabled) {
  background: rgba(88, 166, 255, 0.1);
  color: #79c0ff;
}

.terminal__action--danger {
  color: #f85149;
}

.terminal__action--danger:hover:not(:disabled) {
  background: rgba(248, 81, 73, 0.1);
  color: #ff7b72;
}

.terminal__action--force {
  color: #8b949e;
}

.terminal__action--force:hover:not(:disabled) {
  background: rgba(248, 81, 73, 0.08);
  color: #f85149;
}

.terminal__action-sep {
  width: 1px;
  height: 16px;
  background: rgba(255, 255, 255, 0.08);
  margin: 0 4px;
}

.terminal__feature-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  border: none;
  border-radius: 5px;
  background: transparent;
  color: #8b949e;
  cursor: pointer;
  transition: all 0.15s ease;
}

.terminal__feature-btn:hover {
  background: rgba(255, 255, 255, 0.06);
  color: #c9d1d9;
}

.terminal__cols-rows {
  font-size: 0.7rem;
  color: #484f58;
  font-family: 'Cascadia Code', 'JetBrains Mono', 'Fira Code', 'Consolas', monospace;
  margin-left: 6px;
}

/* ===== Animations ===== */
@keyframes terminal-cursor-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0; }
}

@keyframes terminal-blink {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.3; }
}

/* ===== Light mode ===== */
:global(html.light .terminal) {
  background: #f6f8fa;
  box-shadow:
    0 0 0 1px rgba(0, 0, 0, 0.03),
    0 4px 24px rgba(0, 0, 0, 0.1);
}

:global(html.light .terminal__titlebar),
:global(html.light .terminal__bottombar) {
  background: #ebeef1;
  border-color: rgba(0, 0, 0, 0.06);
}

:global(html.light .terminal__screen),
:global(html.light .terminal__input-bar) {
  background: #f6f8fa;
}

:global(html.light .terminal__line) { color: #24292f; }
:global(html.light .terminal__title-text) { color: #24292f; }
:global(html.light .terminal__textarea) { color: #24292f; caret-color: #0969da; }
:global(html.light .terminal__prompt-user) { color: #1a7f37; }
:global(html.light .terminal__prompt-host) { color: #0969da; }
:global(html.light .terminal__prompt-path) { color: #24292f; }

:global(html.light .terminal__action) { color: #656d76; }
:global(html.light .terminal__action:hover:not(:disabled)) {
  background: rgba(0, 0, 0, 0.04);
  color: #24292f;
}

:global(html.light .terminal__action--primary) { color: #0969da; }
:global(html.light .terminal__action--primary:hover:not(:disabled)) {
  background: rgba(9, 105, 218, 0.08);
  color: #0550ae;
}

:global(html.light .terminal__action--danger) { color: #cf222e; }
:global(html.light .terminal__action--danger:hover:not(:disabled)) {
  background: rgba(207, 34, 46, 0.08);
  color: #a40e26;
}

:global(html.light .terminal__action-sep) { background: rgba(0, 0, 0, 0.08); }
:global(html.light .terminal__cols-rows) { color: #8c959f; }
:global(html.light .terminal__feature-btn) { color: #656d76; }
:global(html.light .terminal__feature-btn:hover) {
  background: rgba(0, 0, 0, 0.04);
  color: #24292f;
}

/* ===== Responsive ===== */
@media (width <= 768px) {
  .terminal {
    height: calc(100vh - 6rem);
    min-height: 22rem;
    border-radius: 6px;
  }

  .terminal__screen {
    padding: 10px 12px 4px;
  }

  .terminal__input-bar {
    padding: 6px 12px 8px;
  }

  .terminal__line,
  .terminal__textarea,
  .terminal__prompt {
    font-size: 0.74rem;
  }

  .terminal__action span {
    display: none;
  }

  .terminal__action {
    padding: 4px 8px;
  }
}
</style>
