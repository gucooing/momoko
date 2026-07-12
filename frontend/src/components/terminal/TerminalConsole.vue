<!-- 终端公共组件（SSH / 实例控制台唯一入口）：外壳(TerminalShell) + xterm(useTerminalX) + 手动主题(useTerminalTheme)
     + 输入输出接线，全部收敛在此。页面只负责自己的传输层：
     @data   原始输入（string=键盘流、Uint8Array=二进制鼠标编码），原样发往后端 PTY
     @resize 终端尺寸变化，页面转发为 {"type":"resize"} 控制帧
     write()/reset()/focus()/cols/rows 经模板 ref 暴露，由页面驱动输出与初始 resize。 -->
<template>
  <TerminalShell
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
    @toggle-theme="toggleTheme"
    @action="(key) => emit('action', key)"
  >
    <div ref="containerRef" class="term-screen" />

    <template #footer-right><slot name="footer-right" /></template>
  </TerminalShell>
</template>

<script setup lang="ts">
import { onMounted } from 'vue'
import type { ConsoleSocketStatus } from '@/stores/instance/types'
import type { TerminalAction } from './TerminalShell.vue'
import TerminalShell from './TerminalShell.vue'
import { useTerminalX } from './useTerminalX'
import { useTerminalTheme } from './useTerminalTheme'

defineOptions({ name: 'TerminalConsole' })

withDefaults(
  defineProps<{
    title: string
    subtitle?: string
    icon?: string
    tag?: { label: string; tone?: 'success' | 'neutral' | 'warning' | 'error' }
    status: ConsoleSocketStatus
    statusLabel: string
    actions?: TerminalAction[]
    busy?: boolean
  }>(),
  { icon: 'HOutline:CommandLineIcon', actions: () => [], busy: false },
)

const emit = defineEmits<{
  data: [data: string | Uint8Array]
  resize: [size: { cols: number; rows: number }]
  action: [key: string]
}>()

const { theme, toggleTheme } = useTerminalTheme()
const { containerRef, cols, rows, mount, write, writeln, clear, reset, focus, fit } = useTerminalX({
  theme,
  onData: (data) => emit('data', data),
  onBinary: (data) => emit('data', data),
  onResize: (size) => emit('resize', size),
})

onMounted(mount)

defineExpose({ cols, rows, write, writeln, clear, reset, focus, fit })
</script>

<style scoped lang="scss">
.term-screen {
  position: absolute;
  inset: 0;
  padding: 6px 6px 4px 10px;
}
</style>
