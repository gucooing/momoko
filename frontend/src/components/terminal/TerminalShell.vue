<!-- 终端外壳（自包含 · 手动明暗 · 默认黑）：顶部工具条（身份 + 动作 + 连接状态 + 主题/全屏/字号）
     + 屏幕主体(default 插槽,承载 xterm) + 移动软键盘工具条(M-2) + 底栏(列×行 + #footer-right)。
     令牌走自建 --term-*(不受全局 html.dark 影响);动作按钮由 actions 数组渲染,统一终端观感。 -->
<template>
  <div class="term-shell" :class="[theme === 'light' ? 'is-light' : 'is-dark', { 'is-max': isMax }]">
    <header class="term-shell__bar">
      <div class="term-shell__id">
        <span class="term-shell__icon">
          <component :is="menuStore.iconComponents[icon]" />
        </span>
        <div class="term-shell__idtext">
          <div class="term-shell__titlerow">
            <span class="term-shell__title">{{ title }}</span>
            <span v-if="tag" class="term-shell__tag" :class="`is-${tag.tone || 'neutral'}`">
              {{ tag.label }}
            </span>
          </div>
          <span v-if="subtitle" class="term-shell__subtitle">{{ subtitle }}</span>
        </div>
      </div>

      <div class="term-shell__tools">
        <div v-if="visibleActions.length" class="term-shell__actions">
          <button
            v-for="act in visibleActions"
            :key="act.key"
            type="button"
            class="term-shell__act"
            :class="[`is-${act.tone || 'default'}`]"
            :disabled="busy || act.disabled"
            :title="act.label"
            :aria-label="act.label"
            @click="emit('action', act.key)"
          >
            <component :is="menuStore.iconComponents[act.icon]" />
          </button>
        </div>

        <span class="term-shell__status">
          <span class="term-shell__dot" :class="`is-${statusTone}`" />
          <span class="term-shell__status-label">{{ statusLabel }}</span>
        </span>

        <span class="term-shell__divider" />

        <button
          type="button"
          class="term-shell__ctrl"
          :title="t('terminal.fontSmaller')"
          :aria-label="t('terminal.fontSmaller')"
          @click="emit('font-bump', -1)"
        >
          <span class="term-shell__font-label">A−</span>
        </button>
        <button
          type="button"
          class="term-shell__ctrl"
          :title="t('terminal.fontLarger')"
          :aria-label="t('terminal.fontLarger')"
          @click="emit('font-bump', 1)"
        >
          <span class="term-shell__font-label">A+</span>
        </button>

        <button
          type="button"
          class="term-shell__ctrl"
          :title="theme === 'dark' ? t('theme.light') : t('theme.dark')"
          @click="emit('toggle-theme')"
        >
          <component
            :is="menuStore.iconComponents[theme === 'dark' ? 'HOutline:SunIcon' : 'HOutline:MoonIcon']"
          />
        </button>
        <button
          type="button"
          class="term-shell__ctrl"
          :title="isMax ? t('layout.exitFullscreen') : t('layout.fullscreen')"
          @click="toggleMax"
        >
          <component
            :is="menuStore.iconComponents[isMax ? 'HOutline:ArrowsPointingInIcon' : 'HOutline:ArrowsPointingOutIcon']"
          />
        </button>
      </div>
    </header>

    <div class="term-shell__body"><slot /></div>

    <!-- 移动端软键盘工具条（M-2）：常用控制键 + 修饰态 Ctrl/Alt；仅窄屏显示 -->
    <div v-if="showSoftKeys" class="term-shell__keys" role="toolbar" :aria-label="t('terminal.softKeys')">
      <button
        type="button"
        class="term-shell__key"
        :class="{ 'is-on': ctrlOn }"
        :title="t('terminal.keyCtrl')"
        @click="toggleCtrl"
      >
        Ctrl
      </button>
      <button
        type="button"
        class="term-shell__key"
        :class="{ 'is-on': altOn }"
        :title="t('terminal.keyAlt')"
        @click="toggleAlt"
      >
        Alt
      </button>
      <button
        v-for="key in softKeys"
        :key="key.id"
        type="button"
        class="term-shell__key"
        :class="{ 'is-wide': key.wide }"
        :title="key.label"
        @click="pressKey(key)"
      >
        {{ key.label }}
      </button>
    </div>

    <footer class="term-shell__foot">
      <span class="term-shell__cols">
        {{ cols }}×{{ rows }}
        <span v-if="fontSize" class="term-shell__fs">· {{ fontSize }}px</span>
      </span>
      <span class="term-shell__foot-right"><slot name="footer-right" /></span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMediaQuery } from '@vueuse/core'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import type { TerminalThemeMode } from './useTerminalTheme'

export interface TerminalAction {
  key: string
  icon: string
  label: string
  tone?: 'default' | 'primary' | 'danger'
  disabled?: boolean
  hidden?: boolean
}

interface SoftKey {
  id: string
  label: string
  /** 直接发送的序列；若为字母键且配合 Ctrl/Alt 修饰则按修饰规则变换 */
  seq?: string
  /** 单字母（a–z），支持 Ctrl/Alt 修饰态 */
  letter?: string
  wide?: boolean
}

const props = withDefaults(
  defineProps<{
    title: string
    subtitle?: string
    icon?: string
    tag?: { label: string; tone?: 'success' | 'neutral' | 'warning' | 'error' }
    status: ConsoleSocketStatus
    statusLabel: string
    theme: TerminalThemeMode
    actions?: TerminalAction[]
    cols?: number
    rows?: number
    fontSize?: number
    busy?: boolean
  }>(),
  {
    icon: 'HOutline:CommandLineIcon',
    actions: () => [],
    cols: 0,
    rows: 0,
    fontSize: 0,
    busy: false,
  },
)

const emit = defineEmits<{
  'toggle-theme': []
  action: [key: string]
  'soft-key': [data: string]
  'font-bump': [delta: number]
}>()

const menuStore = useMenuStore()
const { t } = useI18n()

const visibleActions = computed(() => props.actions.filter((a) => !a.hidden))

// 软键盘：≤768 显示（与顶栏移动断点一致）；桌面触屏仍可通过系统键盘
const showSoftKeys = useMediaQuery('(max-width: 768px)')

const ctrlOn = ref(false)
const altOn = ref(false)
const toggleCtrl = () => {
  ctrlOn.value = !ctrlOn.value
}
const toggleAlt = () => {
  altOn.value = !altOn.value
}

// CSI / 控制字符表（与实体键盘一致）
const softKeys: SoftKey[] = [
  { id: 'esc', label: 'Esc', seq: '\x1b' },
  { id: 'tab', label: 'Tab', seq: '\t' },
  { id: 'c', label: 'C', letter: 'c' },
  { id: 'd', label: 'D', letter: 'd' },
  { id: 'z', label: 'Z', letter: 'z' },
  { id: 'l', label: 'L', letter: 'l' },
  { id: 'up', label: '↑', seq: '\x1b[A' },
  { id: 'down', label: '↓', seq: '\x1b[B' },
  { id: 'left', label: '←', seq: '\x1b[D' },
  { id: 'right', label: '→', seq: '\x1b[C' },
  { id: 'home', label: 'Home', seq: '\x1b[H' },
  { id: 'end', label: 'End', seq: '\x1b[F' },
  { id: 'pgup', label: 'PgUp', seq: '\x1b[5~' },
  { id: 'pgdn', label: 'PgDn', seq: '\x1b[6~' },
]

const pressKey = (key: SoftKey) => {
  let data = key.seq ?? ''
  if (key.letter) {
    const code = key.letter.toLowerCase().charCodeAt(0)
    if (ctrlOn.value) {
      // Ctrl+字母 → C0 控制符（A=1 … Z=26）
      data = String.fromCharCode(code - 96)
    } else if (altOn.value) {
      // Alt+字母 → ESC 前缀
      data = `\x1b${key.letter.toLowerCase()}`
    } else {
      // 未按修饰：默认按 Ctrl 发（移动端点 C/D/Z/L 期望即中断/EOF/挂起/清屏）
      data = String.fromCharCode(code - 96)
    }
  }
  if (!data) return
  emit('soft-key', data)
  // 一次性修饰：发送后复位，避免粘滞
  ctrlOn.value = false
  altOn.value = false
}

const statusTone = computed(() => {
  switch (props.status) {
    case 'connected':
      return 'connected'
    case 'connecting':
      return 'connecting'
    case 'error':
      return 'error'
    default:
      return 'idle'
  }
})

// —— 全屏（脱离内容区，覆盖全窗口）——
const isMax = ref(false)
const toggleMax = () => {
  isMax.value = !isMax.value
}
const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && isMax.value) isMax.value = false
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped lang="scss">
.term-shell {
  /* ---- 暗色（默认）自包含令牌 ---- */
  --term-bg: #0d1117;
  --term-chrome: #161b22;
  --term-border: #262c36;
  --term-fg: #d5dde8;
  --term-fg-dim: #8b949e;
  --term-fg-faint: #6e7681;
  --term-hover: rgba(255, 255, 255, 0.08);
  --term-accent: #2dd4bf;

  /* 全屏页语义：占满整个内容区(fullBleed 路由)，不做卡片外框 */
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  background: var(--term-bg);
}
.term-shell.is-light {
  --term-bg: #f8fafc;
  --term-chrome: #eef1f6;
  --term-border: #dce1e9;
  --term-fg: #1f2933;
  --term-fg-dim: #64748b;
  --term-fg-faint: #94a3b8;
  --term-hover: rgba(15, 23, 42, 0.06);
  --term-accent: #0d9488;
}
.term-shell.is-max {
  position: fixed;
  inset: 0;
  z-index: 2000;
  height: 100vh;
}

/* ===== 顶部工具条 ===== */
.term-shell__bar {
  display: flex;
  align-items: center;
  gap: 12px;
  height: 40px;
  padding: 0 8px 0 12px;
  flex-shrink: 0;
  background: var(--term-chrome);
  border-bottom: 1px solid var(--term-border);
  user-select: none;
}
.term-shell__id {
  display: flex;
  align-items: center;
  gap: 9px;
  flex: 1;
  min-width: 0;
}
.term-shell__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  color: var(--term-accent);
}
.term-shell__icon :deep(svg) {
  width: 17px;
  height: 17px;
}
.term-shell__idtext {
  display: flex;
  flex-direction: column;
  min-width: 0;
  line-height: 1.25;
}
.term-shell__titlerow {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
}
.term-shell__title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--term-fg);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.term-shell__tag {
  flex-shrink: 0;
  font-size: 0.625rem;
  font-weight: 600;
  padding: 1px 6px;
  border-radius: 999px;
  line-height: 1.6;
}
.term-shell__tag.is-success {
  color: #3fb950;
  background: rgba(63, 185, 80, 0.14);
}
.term-shell__tag.is-warning {
  color: #d29922;
  background: rgba(210, 153, 34, 0.14);
}
.term-shell__tag.is-error {
  color: #f85149;
  background: rgba(248, 81, 73, 0.14);
}
.term-shell__tag.is-neutral {
  color: var(--term-fg-dim);
  background: var(--term-hover);
}
.term-shell__subtitle {
  font-size: 0.6875rem;
  color: var(--term-fg-dim);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-family: 'Cascadia Code', 'JetBrains Mono', Consolas, monospace;
}

.term-shell__tools {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}
.term-shell__actions {
  display: flex;
  align-items: center;
  gap: 2px;
}
.term-shell__act,
.term-shell__ctrl {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--term-fg-dim);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.term-shell__act :deep(svg),
.term-shell__ctrl :deep(svg) {
  width: 16px;
  height: 16px;
}
.term-shell__act:hover:not(:disabled),
.term-shell__ctrl:hover {
  background: var(--term-hover);
  color: var(--term-fg);
}
.term-shell__act:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.term-shell__act.is-primary {
  color: var(--term-accent);
}
.term-shell__act.is-primary:hover:not(:disabled) {
  background: color-mix(in srgb, var(--term-accent) 16%, transparent);
  color: var(--term-accent);
}
.term-shell__act.is-danger {
  color: #f85149;
}
.term-shell__act.is-danger:hover:not(:disabled) {
  background: rgba(248, 81, 73, 0.12);
  color: #f85149;
}

.term-shell__status {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 0 2px;
}
.term-shell__dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  flex-shrink: 0;
  background: var(--term-fg-faint);
}
.term-shell__dot.is-connected {
  background: #3fb950;
  box-shadow: 0 0 6px rgba(63, 185, 80, 0.5);
}
.term-shell__dot.is-connecting {
  background: #d29922;
  box-shadow: 0 0 6px rgba(210, 153, 34, 0.5);
  animation: term-blink 1s ease-in-out infinite;
}
.term-shell__dot.is-error {
  background: #f85149;
  box-shadow: 0 0 6px rgba(248, 81, 73, 0.5);
}
.term-shell__status-label {
  font-size: 0.6875rem;
  color: var(--term-fg-dim);
  white-space: nowrap;
}
.term-shell__divider {
  width: 1px;
  height: 16px;
  background: var(--term-border);
  margin: 0 2px;
}

/* ===== 屏幕主体 ===== */
.term-shell__body {
  flex: 1;
  min-height: 0;
  position: relative;
  background: var(--term-bg);
}

/* ===== 底栏 ===== */
.term-shell__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  height: 26px;
  padding: 0 12px;
  flex-shrink: 0;
  background: var(--term-chrome);
  border-top: 1px solid var(--term-border);
  font-size: 0.6875rem;
  color: var(--term-fg-faint);
  font-family: 'Cascadia Code', 'JetBrains Mono', Consolas, monospace;
  user-select: none;
}
.term-shell__cols {
  font-variant-numeric: tabular-nums;
}
.term-shell__foot-right {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.term-shell__font-label {
  font-size: 0.6875rem;
  font-weight: 700;
  font-family: 'Cascadia Code', 'JetBrains Mono', Consolas, monospace;
  letter-spacing: -0.02em;
  line-height: 1;
}

.term-shell__fs {
  margin-left: 2px;
  opacity: 0.85;
}

/* ===== 移动软键盘 ===== */
.term-shell__keys {
  display: flex;
  flex-wrap: nowrap;
  align-items: center;
  gap: 4px;
  padding: 6px 8px calc(6px + env(safe-area-inset-bottom, 0px));
  flex-shrink: 0;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  background: var(--term-chrome);
  border-top: 1px solid var(--term-border);
  scrollbar-width: none;
  user-select: none;
}
.term-shell__keys::-webkit-scrollbar {
  display: none;
}
.term-shell__key {
  flex: 0 0 auto;
  min-width: 40px;
  height: 36px;
  padding: 0 10px;
  border: 1px solid var(--term-border);
  border-radius: var(--app-radius-sm, 6px);
  background: var(--term-bg);
  color: var(--term-fg);
  font-size: 0.75rem;
  font-weight: 600;
  font-family: 'Cascadia Code', 'JetBrains Mono', Consolas, system-ui, sans-serif;
  line-height: 1;
  cursor: pointer;
  touch-action: manipulation;
  transition: background 0.12s, color 0.12s, border-color 0.12s;
}
.term-shell__key.is-wide {
  min-width: 52px;
}
.term-shell__key:active {
  background: var(--term-hover);
}
.term-shell__key.is-on {
  color: var(--term-accent);
  border-color: color-mix(in srgb, var(--term-accent) 55%, var(--term-border));
  background: color-mix(in srgb, var(--term-accent) 14%, var(--term-bg));
}

@keyframes term-blink {
  0%,
  100% {
    opacity: 1;
  }
  50% {
    opacity: 0.3;
  }
}

@media (width <= 768px) {
  .term-shell__subtitle,
  .term-shell__status-label {
    display: none;
  }
  .term-shell__bar {
    gap: 8px;
    height: 44px;
  }
  .term-shell__act,
  .term-shell__ctrl {
    width: 32px;
    height: 32px;
  }
}
</style>
