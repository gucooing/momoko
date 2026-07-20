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
  const panel = panelRef.value
  if (!el) return
  const r = el.getBoundingClientRect()
  const gap = 8
  const pad = 8
  // 用 visualViewport 兼容移动端地址栏/缩放，避免 fixed 面板飞出可视区
  const vv = window.visualViewport
  const vw = Math.round(vv?.width ?? window.innerWidth)
  const vh = Math.round(vv?.height ?? window.innerHeight)
  const vLeft = Math.round(vv?.offsetLeft ?? 0)
  const vTop = Math.round(vv?.offsetTop ?? 0)

  const panelW = Math.min(
    props.width || panel?.offsetWidth || 180,
    vw - pad * 2,
  )
  // 对齐触发器：end=右缘对齐，start=左缘对齐；再钳进视口
  let left =
    props.align === 'end' ? Math.round(r.right - panelW) : Math.round(r.left)
  left = Math.min(Math.max(left, vLeft + pad), vLeft + vw - panelW - pad)

  const style: Record<string, string> = {
    position: 'fixed',
    zIndex: '2100',
    left: `${left}px`,
    width: `${panelW}px`,
    // 不用 right，避免 DPR/visualViewport 下 right 算出界
  }

  const panelH = panel?.offsetHeight || 160
  if (props.side === 'top') {
    // 优先贴触发器上方；空间不足则翻到下方
    if (r.top - vTop >= panelH + gap + pad) {
      style.top = `${Math.round(r.top - gap - panelH)}px`
    } else {
      style.top = `${Math.round(r.bottom + gap)}px`
    }
  } else {
    let top = Math.round(r.bottom + gap)
    if (top + panelH > vTop + vh - pad) {
      top = Math.max(vTop + pad, Math.round(r.top - gap - panelH))
    }
    style.top = `${top}px`
  }

  panelStyle.value = style
}

const open = async () => {
  isOpen.value = true
  await nextTick()
  position()
  // 内容渲染后复测一次高度（语言菜单等）
  requestAnimationFrame(() => position())
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
