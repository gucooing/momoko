<!-- 终端公共组件（SSH / 实例控制台 / 容器终端唯一入口）：外壳(TerminalShell) + xterm(useTerminalX) + 手动主题(useTerminalTheme)
     + 输入输出接线，全部收敛在此。页面只负责自己的传输层：
     @data   原始输入（string=键盘流、Uint8Array=二进制鼠标编码），原样发往后端 PTY
     @resize 终端尺寸变化，页面转发为 {"type":"resize"} 控制帧
     write()/reset()/focus()/cols/rows 经模板 ref 暴露，由页面驱动输出与初始 resize。
     可选插槽：#lead（替换顶栏左侧身份区）、#overlay（屏幕遮罩，如断线提示）、#footer-right。
     复制/粘贴/按键条/粘滞 Ctrl·Alt 都在这里接到同一条 @data 输入通道。 -->
<template>
  <TerminalShell
    ref="shellRef"
    :title="title"
    :subtitle="subtitle"
    :icon="icon"
    :tag="tag"
    :status="status"
    :status-label="statusLabel"
    :theme="theme"
    :actions="actions"
    :busy="busy"
    :cols="cols"
    :rows="rows"
    :font-size="fontSize"
    :variant="variant"
    :keys-open="keysOpen"
    @toggle-theme="toggleTheme"
    @action="(key) => emit('action', key)"
    @soft-key="onSoftKey"
    @font-bump="onFontBump"
    @copy="onCopy"
    @paste="onPaste"
    @keyboard="onKeyboard"
  >
    <div ref="containerRef" class="term-screen" />

    <template v-if="$slots.lead" #lead><slot name="lead" /></template>
    <template #overlay><slot name="overlay" /></template>
    <template #footer-right><slot name="footer-right" /></template>
    <template v-if="$slots.below" #below><slot name="below" /></template>
  </TerminalShell>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import type { TerminalAction } from './TerminalShell.vue'
import TerminalShell from './TerminalShell.vue'
import { useTerminalX } from './useTerminalX'
import { useTerminalTheme } from './useTerminalTheme'

defineOptions({ name: 'TerminalConsole' })

const props = withDefaults(
  defineProps<{
    title: string
    subtitle?: string
    icon?: string
    tag?: { label: string; tone?: 'success' | 'neutral' | 'warning' | 'error' }
    status: ConsoleSocketStatus
    statusLabel: string
    actions?: TerminalAction[]
    busy?: boolean
    /** console：应用控制台深色卡片（原型样式），屏幕固定深色 */
    variant?: 'card' | 'console'
    keysOpen?: boolean
  }>(),
  { icon: 'HOutline:CommandLineIcon', actions: () => [], busy: false, variant: 'card', keysOpen: false },
)

const emit = defineEmits<{
  data: [data: string | Uint8Array]
  resize: [size: { cols: number; rows: number }]
  action: [key: string]
}>()

const { t } = useI18n()
const shellRef = ref<InstanceType<typeof TerminalShell>>()

const { theme: savedTheme, toggleTheme } = useTerminalTheme()
// 应用控制台卡片固定深色屏幕
const theme = computed(() => (props.variant === 'console' ? 'dark' : savedTheme.value))
const {
  containerRef,
  cols,
  rows,
  fontSize,
  mount,
  write,
  writeln,
  clear,
  reset,
  focus,
  blur,
  getSelection,
  paste,
  scrollToBottom,
  fit,
  bumpFontSize,
} = useTerminalX({
  theme,
  // 键盘输入先过一遍粘滞的 Ctrl/Alt（手机按键条上点亮后，作用于下一次按键）
  onData: (data) => emit('data', shellRef.value ? shellRef.value.applyModifiers(data) : data),
  onBinary: (data) => emit('data', data),
  onResize: (size) => emit('resize', size),
})

// 按键条：序列原样注入传输层（与实体键盘 onData 同路径）
const onSoftKey = (data: string) => {
  emit('data', data)
  focus()
}

const onFontBump = (delta: number) => {
  bumpFontSize(delta)
  focus()
}

const onCopy = async () => {
  const text = getSelection()
  if (!text) {
    feedback.warning(t('terminal.copyEmpty'))
    return
  }
  try {
    await navigator.clipboard.writeText(text)
    feedback.success(t('terminal.copied'))
  } catch {
    feedback.warning(t('terminal.clipboardDenied'))
  }
  focus()
}

const onPaste = async () => {
  try {
    const text = await navigator.clipboard.readText()
    if (text) paste(text)
  } catch {
    feedback.warning(t('terminal.clipboardDenied'))
  }
  focus()
}

// 手机端键盘开关：聚焦 xterm 的隐藏输入框即弹出系统键盘，再点一次收起
let keyboardOpen = false
const onKeyboard = () => {
  keyboardOpen = !keyboardOpen
  if (keyboardOpen) focus()
  else blur()
}

onMounted(mount)

defineExpose({ cols, rows, write, writeln, clear, reset, focus, blur, fit, scrollToBottom })
</script>

<style scoped lang="scss">
.term-screen {
  position: absolute;
  inset: 0;
  padding: 6px 6px 4px 10px;
}
</style>
