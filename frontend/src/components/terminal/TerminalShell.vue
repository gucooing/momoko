<!-- 终端外壳：圆角卡片（跟随全局配色）里放固定终端色的屏幕。
     顶栏：#lead 插槽（默认是 贴纸图标 + 标题/标签 + 副标题；SSH 页放会话标签）+ 工具（动作 / 复制 / 粘贴 / 字号 / 终端明暗 / 全屏）；
     手机端工具收进「⋯」菜单。屏幕：default 插槽承载 xterm，#overlay 插槽给断线遮罩。
     手机端屏幕下方是常用按键条（Esc/Tab/Ctrl/Alt/方向键/符号…，Ctrl/Alt 为一次性粘滞修饰，同样作用于系统键盘输入）+ 键盘开关。
     底栏：连接状态 pill + 副标题 + 终端类型 + 列×行 + 字号 + #footer-meta（如延迟）+ #footer-right。
     可选：actionsEnd（动作放工具最右）、compactActionLabels（窄屏动作仍带文字）、compactFooter=false（窄屏隐藏底栏）。
     对 TerminalConsole 暴露 applyModifiers(data)，用于把粘滞的 Ctrl/Alt 应用到下一次键盘输入。
     variant="console"（应用控制台，按原型）：整张深色卡片；顶栏 = 连接状态 + 信息 + 文字按钮（清屏 / 重新连接 / 自动滚动…），
     没有底栏；#below 插槽放快捷命令与命令输入；手机按键条由 keysOpen 控制（输入行里的键盘按钮切换）。 -->
<template>
  <div
    class="term-shell"
    :class="[
      isConsole || theme !== 'light' ? 'is-dark' : 'is-light',
      { 'is-max': isMax, 'is-compact': compact, 'is-console': isConsole },
    ]"
  >
    <!-- 应用控制台顶栏（深色）：连接状态 + 信息 + 文字按钮 -->
    <header v-if="isConsole" class="term-shell__cbar">
      <span class="term-shell__conn" :class="`is-${statusVariant}`">
        <span class="term-shell__dot" aria-hidden="true" />
        {{ statusLabel }}
      </span>
      <span v-if="subtitle && !compact" class="term-shell__cinfo m-mono">{{ subtitle }}</span>
      <span class="term-shell__spacer" />
      <template v-if="!compact">
        <button
          v-for="act in visibleActions"
          :key="act.key"
          type="button"
          class="term-shell__tbtn"
          :class="`is-${act.tone || 'default'}`"
          :disabled="busy || act.disabled"
          :aria-pressed="act.toggle"
          @click="emit('action', act.key)"
        >
          <span v-if="act.toggle !== undefined" class="term-shell__tsw" :class="{ 'is-on': act.toggle }" aria-hidden="true" />
          <template v-else>
            <component :is="menuStore.iconComponents[act.icon]" v-if="menuStore.iconComponents[act.icon]" />
            <UIcon v-else :name="act.icon" />
          </template>
          {{ act.label }}
        </button>
        <button type="button" class="term-shell__tbtn" @click="emit('copy')">
          <UIcon name="i-lucide-copy" />{{ t('terminal.copy') }}
        </button>
        <button type="button" class="term-shell__tbtn" @click="emit('paste')">
          <UIcon name="i-lucide-clipboard-paste" />{{ t('terminal.paste') }}
        </button>
        <button
          type="button"
          class="term-shell__tbtn is-icon"
          :title="isMax ? t('layout.exitFullscreen') : t('layout.fullscreen')"
          :aria-label="isMax ? t('layout.exitFullscreen') : t('layout.fullscreen')"
          @click="toggleMax"
        >
          <UIcon :name="isMax ? 'i-lucide-minimize' : 'i-lucide-maximize'" />
        </button>
      </template>
      <template v-else>
        <button
          v-for="act in visibleActions.slice(0, 3)"
          :key="act.key"
          type="button"
          class="term-shell__tib"
          :class="{ 'is-on': act.toggle }"
          :disabled="busy || act.disabled"
          :title="act.label"
          :aria-label="act.label"
          :aria-pressed="act.toggle"
          @click="emit('action', act.key)"
        >
          <component :is="menuStore.iconComponents[act.icon]" v-if="menuStore.iconComponents[act.icon]" />
          <UIcon v-else :name="act.icon" />
        </button>
        <ActionMenu :items="consoleMoreItems" :label="t('terminal.more')" @select="onConsoleMore" />
      </template>
    </header>

    <header v-else class="term-shell__bar">
      <div class="term-shell__lead">
        <slot name="lead">
          <span class="term-shell__icon m-sticker m-sticker--p">
            <component :is="menuStore.iconComponents[icon]" v-if="menuStore.iconComponents[icon]" />
            <UIcon v-else name="i-lucide-terminal" />
          </span>
          <span class="term-shell__idtext">
            <span class="term-shell__titlerow">
              <span class="term-shell__title">{{ title }}</span>
              <span v-if="tag" class="term-shell__tag" :class="`is-${tag.tone || 'neutral'}`">{{ tag.label }}</span>
            </span>
            <span v-if="subtitle" class="term-shell__subtitle m-mono">{{ subtitle }}</span>
          </span>
        </slot>
      </div>

      <div class="term-shell__tools" role="toolbar" :aria-label="t('terminal.tools')">
        <template v-for="act in actionsEnd ? [] : visibleActions" :key="act.key">
          <button
            v-if="(!compact || compactActionLabels) && act.tone && act.tone !== 'default'"
            type="button"
            class="term-shell__pill"
            :class="`is-${act.tone}`"
            :disabled="busy || act.disabled"
            :title="act.label"
            @click="emit('action', act.key)"
          >
            <component :is="menuStore.iconComponents[act.icon]" v-if="menuStore.iconComponents[act.icon]" />
            <span>{{ act.label }}</span>
          </button>
          <button
            v-else
            type="button"
            class="term-shell__ib"
            :class="`is-${act.tone || 'default'}`"
            :disabled="busy || act.disabled"
            :title="act.label"
            :aria-label="act.label"
            @click="emit('action', act.key)"
          >
            <component :is="menuStore.iconComponents[act.icon]" v-if="menuStore.iconComponents[act.icon]" />
          </button>
        </template>

        <template v-if="!compact">
          <span v-if="visibleActions.length && !actionsEnd" class="term-shell__sep" />
          <button type="button" class="term-shell__pill is-soft" :title="t('terminal.copy')" @click="emit('copy')">
            <UIcon name="i-lucide-copy" />
            <span>{{ t('terminal.copy') }}</span>
          </button>
          <button type="button" class="term-shell__pill is-soft" :title="t('terminal.paste')" @click="emit('paste')">
            <UIcon name="i-lucide-clipboard-paste" />
            <span>{{ t('terminal.paste') }}</span>
          </button>
          <div class="term-shell__fsz" role="group" :aria-label="t('terminal.fontSize')">
            <button type="button" :aria-label="t('terminal.fontSmaller')" :title="t('terminal.fontSmaller')" @click="emit('font-bump', -1)">−</button>
            <span class="m-num">{{ fontSize || '—' }} px</span>
            <button type="button" :aria-label="t('terminal.fontLarger')" :title="t('terminal.fontLarger')" @click="emit('font-bump', 1)">+</button>
          </div>
          <button
            type="button"
            class="term-shell__ib"
            :title="theme === 'dark' ? t('terminal.lightScreen') : t('terminal.darkScreen')"
            :aria-label="theme === 'dark' ? t('terminal.lightScreen') : t('terminal.darkScreen')"
            @click="emit('toggle-theme')"
          >
            <UIcon :name="theme === 'dark' ? 'i-lucide-sun' : 'i-lucide-moon'" />
          </button>
          <button
            type="button"
            class="term-shell__ib"
            :title="isMax ? t('layout.exitFullscreen') : t('layout.fullscreen')"
            :aria-label="isMax ? t('layout.exitFullscreen') : t('layout.fullscreen')"
            @click="toggleMax"
          >
            <UIcon :name="isMax ? 'i-lucide-minimize' : 'i-lucide-maximize'" />
          </button>
        </template>
        <ActionMenu v-else :items="moreItems" :label="t('terminal.more')" @select="onMore" />

        <!-- actionsEnd：连接类动作放在工具最右侧（SSH 终端按原型：… 全屏 / 断开） -->
        <template v-for="act in actionsEnd ? visibleActions : []" :key="act.key">
          <button
            v-if="(!compact || compactActionLabels) && act.tone && act.tone !== 'default'"
            type="button"
            class="term-shell__pill"
            :class="`is-${act.tone}`"
            :disabled="busy || act.disabled"
            :title="act.label"
            @click="emit('action', act.key)"
          >
            <component :is="menuStore.iconComponents[act.icon]" v-if="menuStore.iconComponents[act.icon]" />
            <span>{{ act.label }}</span>
          </button>
          <button
            v-else
            type="button"
            class="term-shell__ib"
            :class="`is-${act.tone || 'default'}`"
            :disabled="busy || act.disabled"
            :title="act.label"
            :aria-label="act.label"
            @click="emit('action', act.key)"
          >
            <component :is="menuStore.iconComponents[act.icon]" v-if="menuStore.iconComponents[act.icon]" />
          </button>
        </template>
      </div>
    </header>

    <div class="term-shell__screen">
      <div class="term-shell__body"><slot /></div>
      <slot name="overlay" />
    </div>

    <!-- 手机端常用按键条（控制台卡片里由输入行的键盘按钮展开） -->
    <div v-if="compact && (!isConsole || keysOpen)" class="term-shell__keysrow">
      <div class="term-shell__keys" role="toolbar" :aria-label="t('terminal.softKeys')">
        <button
          v-for="key in softKeys"
          :key="key.id"
          type="button"
          class="term-shell__key"
          :class="{ 'is-on': (key.id === 'ctrl' && ctrlOn) || (key.id === 'alt' && altOn) }"
          :aria-label="key.aria || key.label"
          :aria-pressed="key.mod ? (key.id === 'ctrl' ? ctrlOn : altOn) : undefined"
          @pointerdown.prevent
          @click="pressKey(key)"
        >
          {{ key.label }}
        </button>
      </div>
      <button
        v-if="!isConsole"
        type="button"
        class="term-shell__kbd"
        :aria-label="t('terminal.keyboard')"
        :title="t('terminal.keyboard')"
        @pointerdown.prevent
        @click="emit('keyboard')"
      >
        <UIcon name="i-lucide-keyboard" />
      </button>
    </div>

    <slot name="below" />

    <footer v-if="!isConsole && (compactFooter || !compact)" class="term-shell__foot">
      <StatusPill :variant="statusVariant" :label="statusLabel" />
      <span v-if="subtitle && !compact" class="term-shell__sb m-mono">{{ subtitle }}</span>
      <span v-if="!compact" class="term-shell__sb">xterm-256color · UTF-8</span>
      <span class="term-shell__sb m-num">
        <UIcon name="i-lucide-scaling" />{{ cols }}×{{ rows }}
        <template v-if="fontSize && compact">· {{ fontSize }}px</template>
      </span>
      <slot name="footer-meta" />
      <span class="term-shell__spacer" />
      <span class="term-shell__sb term-shell__foot-right"><slot name="footer-right" /></span>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMediaQuery } from '@vueuse/core'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { TerminalThemeMode } from './useTerminalTheme'

export interface TerminalAction {
  key: string
  icon: string
  label: string
  tone?: 'default' | 'primary' | 'danger'
  disabled?: boolean
  hidden?: boolean
  /** 开关型按钮（如「自动滚动」）：显示开关并带 aria-pressed */
  toggle?: boolean
}

interface SoftKey {
  id: string
  label: string
  aria?: string
  /** 直接发送的序列 */
  seq?: string
  /** Ctrl / Alt 粘滞修饰键 */
  mod?: boolean
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
    variant?: 'card' | 'console'
    keysOpen?: boolean
    /** 卡片顶栏：动作按钮放在工具最右侧（默认放在最左侧） */
    actionsEnd?: boolean
    /** 窄屏时带色调的动作仍显示文字药丸（默认只显示图标） */
    compactActionLabels?: boolean
    /** 窄屏是否显示底栏（默认显示；SSH 终端把状态放到顶栏） */
    compactFooter?: boolean
  }>(),
  {
    variant: 'card',
    keysOpen: false,
    actionsEnd: false,
    compactActionLabels: false,
    compactFooter: true,
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
  copy: []
  paste: []
  keyboard: []
}>()

const menuStore = useMenuStore()
const { t } = useI18n()

const visibleActions = computed(() => props.actions.filter((a) => !a.hidden))
const isConsole = computed(() => props.variant === 'console')

// 窄屏（≤768，与顶栏移动断点一致）：工具收进菜单 + 显示按键条
const compact = useMediaQuery('(max-width: 768px)')

const ctrlOn = ref(false)
const altOn = ref(false)

const softKeys = computed<SoftKey[]>(() => [
  { id: 'esc', label: 'Esc', seq: '\x1b' },
  { id: 'tab', label: 'Tab', seq: '\t' },
  { id: 'ctrl', label: 'Ctrl', mod: true, aria: t('terminal.keyCtrl') },
  { id: 'alt', label: 'Alt', mod: true, aria: t('terminal.keyAlt') },
  { id: 'up', label: '↑', seq: '\x1b[A', aria: t('terminal.keyUp') },
  { id: 'down', label: '↓', seq: '\x1b[B', aria: t('terminal.keyDown') },
  { id: 'left', label: '←', seq: '\x1b[D', aria: t('terminal.keyLeft') },
  { id: 'right', label: '→', seq: '\x1b[C', aria: t('terminal.keyRight') },
  { id: 'cc', label: '^C', seq: '\x03', aria: 'Ctrl+C' },
  { id: 'cd', label: '^D', seq: '\x04', aria: 'Ctrl+D' },
  { id: 'cz', label: '^Z', seq: '\x1a', aria: 'Ctrl+Z' },
  { id: 'cl', label: '^L', seq: '\x0c', aria: 'Ctrl+L' },
  { id: 'pipe', label: '|', seq: '|' },
  { id: 'slash', label: '/', seq: '/' },
  { id: 'dash', label: '-', seq: '-' },
  { id: 'tilde', label: '~', seq: '~' },
  { id: 'home', label: 'Home', seq: '\x1b[H' },
  { id: 'end', label: 'End', seq: '\x1b[F' },
  { id: 'pgup', label: 'PgUp', seq: '\x1b[5~' },
  { id: 'pgdn', label: 'PgDn', seq: '\x1b[6~' },
])

/** 把粘滞的 Ctrl/Alt 应用到一段输入（单个可打印字符才变换），用后复位。 */
const applyModifiers = (data: string): string => {
  if (!ctrlOn.value && !altOn.value) return data
  let out = data
  if (data.length === 1) {
    const code = data.toLowerCase().charCodeAt(0)
    if (ctrlOn.value) {
      // Ctrl+字母 → C0 控制符（a=1 … z=26）；Ctrl+[ \ ] ^ _ 与 Ctrl+空格 同理
      if (code >= 97 && code <= 122) out = String.fromCharCode(code - 96)
      else if (code >= 91 && code <= 95) out = String.fromCharCode(code - 64)
      else if (data === ' ') out = '\x00'
    }
    if (altOn.value) out = `\x1b${out}`
  } else if (altOn.value && !data.startsWith('\x1b')) {
    out = `\x1b${data}`
  }
  ctrlOn.value = false
  altOn.value = false
  return out
}

const pressKey = (key: SoftKey) => {
  if (key.mod) {
    if (key.id === 'ctrl') ctrlOn.value = !ctrlOn.value
    else altOn.value = !altOn.value
    return
  }
  if (!key.seq) return
  // 方向键等多字节序列只在 Alt 下加 ESC 前缀；单字符照常应用修饰
  emit('soft-key', applyModifiers(key.seq))
}

const statusVariant = computed(() => {
  switch (props.status) {
    case 'connected':
      return 'success' as const
    case 'connecting':
      return 'warning' as const
    case 'error':
      return 'error' as const
    default:
      return 'neutral' as const
  }
})

const moreItems = computed<ActionMenuItem[]>(() => [
  { key: 'copy', label: t('terminal.copy'), icon: 'i-lucide-copy' },
  { key: 'paste', label: t('terminal.paste'), icon: 'i-lucide-clipboard-paste' },
  { key: 'font-up', label: t('terminal.fontLarger'), icon: 'i-lucide-a-arrow-up' },
  { key: 'font-down', label: t('terminal.fontSmaller'), icon: 'i-lucide-a-arrow-down' },
  {
    key: 'theme',
    label: props.theme === 'dark' ? t('terminal.lightScreen') : t('terminal.darkScreen'),
    icon: props.theme === 'dark' ? 'i-lucide-sun' : 'i-lucide-moon',
  },
  {
    key: 'max',
    label: isMax.value ? t('layout.exitFullscreen') : t('layout.fullscreen'),
    icon: isMax.value ? 'i-lucide-minimize' : 'i-lucide-maximize',
  },
])

// 控制台卡片（手机）：前 3 个动作直接显示为图标，其余动作 + 复制 / 粘贴 / 字号 / 全屏收进菜单
const consoleMoreItems = computed<ActionMenuItem[]>(() => [
  ...visibleActions.value.slice(3).map((a) => ({ key: `act:${a.key}`, label: a.label, icon: a.icon })),
  ...moreItems.value.filter((i) => i.key !== 'theme'),
])
const onConsoleMore = (key: string) => {
  if (key.startsWith('act:')) emit('action', key.slice(4))
  else onMore(key)
}

const onMore = (key: string) => {
  if (key === 'copy') emit('copy')
  else if (key === 'paste') emit('paste')
  else if (key === 'font-up') emit('font-bump', 1)
  else if (key === 'font-down') emit('font-bump', -1)
  else if (key === 'theme') emit('toggle-theme')
  else if (key === 'max') toggleMax()
}

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

defineExpose({ applyModifiers })
</script>

<style scoped lang="scss">
.term-shell {
  /* 屏幕色固定（不跟随全局配色），默认黑；外框与工具条跟随全局令牌 */
  --term-bg: #0f0b14;
  --term-fg: #e8e3ee;

  display: flex;
  flex-direction: column;
  gap: 8px;
  flex: 1;
  min-height: 0;
  min-width: 0;
  padding: 8px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  overflow: hidden;
}
.term-shell.is-light {
  --term-bg: #f8fafc;
  --term-fg: #1f2933;
}
.term-shell.is-max {
  position: fixed;
  inset: 0;
  /* 高于手机底部导航（2050） */
  z-index: 2100;
  height: 100vh;
  border: 0;
  border-radius: 0;
}
/* 实例控制台等直接把终端作为全出血页根节点时，留出与普通页一致的外边距 */
:global(.app-content--bleed .app-page > .term-shell:not(.is-max)) {
  margin: 16px;
}
@media (width >= 1024px) {
  :global(.app-content--bleed .app-page > .term-shell:not(.is-max)) {
    margin: 24px 32px;
  }
}

/* ===== 应用控制台卡片（variant=console，按原型：整张深色卡片） ===== */
.term-shell.is-console {
  gap: 0;
  padding: 0;
  border: 1px solid var(--m-line);
  border-radius: 24px;
  background: var(--term-bg);
  color: var(--term-fg);
}
.term-shell.is-console.is-max {
  border-radius: 0;
}
.term-shell__cbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
  min-height: 60px;
  padding: 12px 12px 12px 18px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.08);
  flex-shrink: 0;
  user-select: none;
}
.term-shell__conn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: #9c93a8;
  white-space: nowrap;
}
.term-shell__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
  box-shadow: 0 0 0 4px color-mix(in srgb, currentColor 22%, transparent);
}
.term-shell__conn.is-success {
  color: #5fd38d;
}
.term-shell__conn.is-warning {
  color: #f5c451;
}
.term-shell__conn.is-error {
  color: #ff7a8a;
}
.term-shell__cinfo {
  font-size: 0.75rem;
  color: #9c93a8;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  min-width: 0;
}
.term-shell__tbtn {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 36px;
  padding: 0 13px;
  border: 0;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.08);
  color: #e8e3ee;
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s;
}
.term-shell__tbtn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.15);
}
.term-shell__tbtn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.term-shell__tbtn.is-icon {
  width: 36px;
  padding: 0;
  justify-content: center;
}
.term-shell__tbtn.is-danger {
  color: #ff9aa6;
}
.term-shell__tbtn :deep(svg),
.term-shell__tbtn :deep(.iconify) {
  width: 15px;
  height: 15px;
}
.term-shell__tsw {
  position: relative;
  flex: none;
  width: 30px;
  height: 18px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.22);
  transition: background 0.15s;
}
.term-shell__tsw::after {
  content: '';
  position: absolute;
  top: 3px;
  left: 3px;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  background: #e8e3ee;
  transition: transform 0.15s;
}
.term-shell__tsw.is-on {
  background: var(--m-p);
}
.term-shell__tsw.is-on::after {
  transform: translateX(12px);
}
.term-shell__tib {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 44px;
  height: 44px;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: #d6cfe0;
  cursor: pointer;
}
.term-shell__tib.is-on {
  background: rgba(255, 255, 255, 0.1);
  color: #ffffff;
}
.term-shell__tib :deep(svg),
.term-shell__tib :deep(.iconify) {
  width: 20px;
  height: 20px;
}
.term-shell.is-console .term-shell__screen {
  border-radius: 0;
}
.term-shell.is-console :deep(.action-menu__trigger) {
  color: #d6cfe0;
}
.term-shell.is-console .term-shell__keysrow {
  margin: 0;
  padding: 6px 10px 0;
}
.term-shell.is-console .term-shell__key {
  border-color: rgba(255, 255, 255, 0.16);
  background: rgba(255, 255, 255, 0.06);
  color: #e8e3ee;
}
.term-shell.is-console.is-compact {
  padding: 0;
  border-radius: 0;
  border: none;
}
.term-shell.is-console.is-compact .term-shell__cbar {
  min-height: 52px;
  padding: 4px 6px 4px 14px;
  flex-wrap: nowrap;
}

/* ===== 顶栏 ===== */
.term-shell__bar {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 2px 0 4px;
  flex-shrink: 0;
  user-select: none;
}
.term-shell__lead {
  display: flex;
  align-items: center;
  gap: 10px;
  flex: 1;
  min-width: 0;
}
.term-shell__icon {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  flex: none;
  transform: rotate(-4deg);
}
.term-shell__icon :deep(svg),
.term-shell__icon :deep(.iconify) {
  width: 18px;
  height: 18px;
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
  gap: 8px;
  min-width: 0;
}
.term-shell__title {
  font-family: var(--app-font-display);
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.term-shell__tag {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 8px;
  border-radius: var(--app-radius-pill);
  font-size: 0.6875rem;
  font-weight: 700;
}
.term-shell__tag.is-success {
  color: var(--m-ok);
  background: var(--m-okSoft);
}
.term-shell__tag.is-warning {
  color: var(--m-warn);
  background: var(--m-warnSoft);
}
.term-shell__tag.is-error {
  color: var(--m-bad);
  background: var(--m-badSoft);
}
.term-shell__tag.is-neutral {
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
}
.term-shell__subtitle {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.term-shell__tools {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.term-shell__sep {
  width: 1px;
  height: 20px;
  margin: 0 4px;
  background: var(--el-border-color-light);
}
.term-shell__ib {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border: none;
  border-radius: 12px;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.term-shell__ib :deep(svg),
.term-shell__ib :deep(.iconify) {
  width: 18px;
  height: 18px;
}
.term-shell__ib:hover:not(:disabled) {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.term-shell__ib:disabled,
.term-shell__pill:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.term-shell__ib.is-primary {
  color: var(--m-pInk);
}
.term-shell__ib.is-danger {
  color: var(--m-bad);
}
.term-shell__ib.is-danger:hover:not(:disabled) {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.term-shell__pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 12px;
  border: none;
  border-radius: var(--app-radius-pill);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition: filter 0.15s, background 0.15s, color 0.15s;
}
.term-shell__pill :deep(svg),
.term-shell__pill :deep(.iconify) {
  width: 16px;
  height: 16px;
}
.term-shell__pill.is-soft {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.term-shell__pill.is-soft:hover {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.term-shell__pill.is-primary {
  background: var(--m-p);
  color: var(--m-pOn);
}
.term-shell__pill.is-primary:hover:not(:disabled) {
  filter: brightness(1.08);
}
.term-shell__pill.is-danger {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.term-shell__pill.is-danger:hover:not(:disabled) {
  filter: brightness(0.97);
}
.term-shell__fsz {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  margin: 0 4px;
  padding: 2px;
  border-radius: var(--app-radius-pill);
  background: var(--el-fill-color-light);
}
.term-shell__fsz button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 32px;
  border: 0;
  border-radius: var(--app-radius-pill);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 1rem;
  font-weight: 800;
  cursor: pointer;
}
.term-shell__fsz button:hover {
  background: var(--el-bg-color);
}
.term-shell__fsz span {
  min-width: 44px;
  text-align: center;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

/* ===== 屏幕 ===== */
.term-shell__screen {
  flex: 1;
  min-height: 0;
  position: relative;
  border-radius: 18px;
  background: var(--term-bg);
  overflow: hidden;
}
.term-shell__body {
  position: absolute;
  inset: 0;
}
.term-shell__body :deep(.term-screen) {
  padding: 14px 8px 8px 16px;
}

/* ===== 手机按键条 ===== */
.term-shell__keysrow {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
  margin: 0 -2px;
}
.term-shell__keys {
  flex: 1;
  min-width: 0;
  display: flex;
  gap: 6px;
  overflow-x: auto;
  -webkit-overflow-scrolling: touch;
  scrollbar-width: none;
  user-select: none;
}
.term-shell__keys::-webkit-scrollbar {
  display: none;
}
.term-shell__key {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 44px;
  height: 44px;
  padding: 0 10px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 12px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font-family: var(--app-font-mono);
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
  touch-action: manipulation;
}
.term-shell__key:active {
  background: var(--el-fill-color-light);
}
.term-shell__key.is-on,
.term-shell__key.is-on:active {
  background: var(--m-p);
  border-color: var(--m-p);
  color: var(--m-pOn);
}
.term-shell__kbd {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 44px;
  border: none;
  border-left: 1px solid var(--el-border-color-light);
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.term-shell__kbd :deep(.iconify) {
  width: 20px;
  height: 20px;
}

/* ===== 底栏 ===== */
.term-shell__foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 18px;
  min-height: 34px;
  padding: 0 10px 2px;
  flex-shrink: 0;
  user-select: none;
}
.term-shell__sb {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.term-shell__sb :deep(.iconify) {
  width: 14px;
  height: 14px;
}
.term-shell__spacer {
  flex: 1;
}
.term-shell__foot-right {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.term-shell.is-compact {
  padding: 6px;
  gap: 6px;
}
.term-shell.is-compact .term-shell__screen {
  border-radius: 14px;
}
.term-shell.is-compact .term-shell__foot {
  gap: 4px 12px;
  padding: 0 6px;
}
.term-shell.is-compact :deep(.action-menu__trigger) {
  width: 44px;
  height: 44px;
  border-radius: 14px;
}
.term-shell.is-compact .term-shell__pill {
  min-height: 44px;
  padding: 0 14px;
}
</style>
