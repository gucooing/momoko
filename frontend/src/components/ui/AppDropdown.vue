<!-- 轻量下拉容器：onClickOutside + Teleport + fixed 定位。浮层用柔和阴影（01 §5）。
     令牌驱动，完全掌控观感，用于顶栏下拉/命令搜索。 -->
<template>
  <div
    ref="triggerRef"
    class="app-dropdown__trigger"
    :class="{ 'app-dropdown__trigger--block': block }"
    @click="toggle"
  >
    <slot name="trigger" :open="isOpen" :toggle="toggle" />
  </div>
  <Teleport to="body">
    <Transition name="app-dropdown">
      <div
        v-if="isOpen"
        ref="panelRef"
        class="app-dropdown__panel"
        :class="[panelClass, side === 'top' ? 'app-dropdown__panel--top' : null]"
        :style="panelStyle"
      >
        <slot :close="close" />
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onClickOutside } from '@vueuse/core'

const props = withDefaults(
  defineProps<{
    align?: 'start' | 'end'
    side?: 'top' | 'bottom'
    width?: number
    panelClass?: string
    block?: boolean
  }>(),
  { align: 'end', side: 'bottom' },
)

const isOpen = ref(false)
const triggerRef = ref<HTMLElement>()
const panelRef = ref<HTMLElement>()
const panelStyle = ref<Record<string, string>>({})

const position = () => {
  const el = triggerRef.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const gap = 8
  const style: Record<string, string> = {
    position: 'fixed',
    zIndex: '2100',
  }
  if (props.side === 'top') style.bottom = `${Math.round(window.innerHeight - r.top + gap)}px`
  else style.top = `${Math.round(r.bottom + gap)}px`
  if (props.align === 'end') style.right = `${Math.round(window.innerWidth - r.right)}px`
  else style.left = `${Math.round(r.left)}px`
  if (props.width) style.width = `${props.width}px`
  panelStyle.value = style
}

const open = async () => {
  isOpen.value = true
  await nextTick()
  position()
}
const close = () => {
  isOpen.value = false
}
const toggle = () => (isOpen.value ? close() : open())

onClickOutside(
  panelRef,
  (e) => {
    if (triggerRef.value?.contains(e.target as Node)) return
    close()
  },
  { ignore: [triggerRef] },
)

const onViewportChange = () => {
  if (isOpen.value) close()
}
onMounted(() => {
  window.addEventListener('resize', onViewportChange)
  window.addEventListener('scroll', onViewportChange, true)
})
onBeforeUnmount(() => {
  window.removeEventListener('resize', onViewportChange)
  window.removeEventListener('scroll', onViewportChange, true)
})

defineExpose({ open, close, toggle })
</script>

<style scoped lang="scss">
.app-dropdown__trigger {
  display: inline-flex;
}
.app-dropdown__trigger--block {
  display: flex;
  flex: 1 1 auto;
  min-width: 0;
}
.app-dropdown__panel {
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  box-shadow: var(--app-shadow-md);
  overflow: hidden;
  min-width: 180px;
}
.app-dropdown-enter-active,
.app-dropdown-leave-active {
  transition: opacity 0.15s ease, transform 0.15s cubic-bezier(0.4, 0, 0.2, 1);
}
.app-dropdown-enter-from,
.app-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-6px);
}
/* 向上弹出时从下方滑入（两类选择器特异性高于基础规则，稳定覆盖） */
.app-dropdown__panel--top.app-dropdown-enter-from,
.app-dropdown__panel--top.app-dropdown-leave-to {
  transform: translateY(6px);
}
</style>
