<!-- 工具页（端口转发 / 内网穿透）统计面板外壳：PC 从右侧滑出的抽屉（左侧大圆角、整列内部滚动），
     手机为底部面板（顶部拖动条、最高 92vh）。头部 / 关闭按钮 / 内容由使用方放进默认插槽（插槽参数 close）。
     遮罩点击与 Esc 关闭；打开时锁定页面滚动。层级低于确认框（2200），确认断开连接等操作可叠在上面。 -->
<template>
  <Teleport to="body">
    <Transition name="tool-sheet">
      <div v-if="modelValue" class="tool-sheet" @mousedown.self="close">
        <section
          class="tool-sheet__panel"
          role="dialog"
          aria-modal="true"
          :aria-label="label"
          :style="{ '--tool-sheet-w': `${width}px` }"
        >
          <span class="tool-sheet__grip" aria-hidden="true" />
          <slot :close="close" />
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
const props = withDefaults(defineProps<{ modelValue: boolean; label?: string; width?: number }>(), {
  label: '',
  width: 500,
})
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; close: [] }>()

const close = () => {
  emit('update:modelValue', false)
  emit('close')
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.modelValue) close()
}
watch(
  () => props.modelValue,
  (open) => {
    document.documentElement.style.overflow = open ? 'hidden' : ''
  },
)
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  if (props.modelValue) document.documentElement.style.overflow = ''
})
</script>

<style scoped lang="scss">
.tool-sheet {
  position: fixed;
  inset: 0;
  z-index: 2100;
  background: color-mix(in srgb, var(--m-ink) 32%, transparent);
}
.tool-sheet__panel {
  position: absolute;
  top: 0;
  right: 0;
  bottom: 0;
  display: flex;
  flex-direction: column;
  gap: 18px;
  width: min(var(--tool-sheet-w), 100vw);
  padding: 24px;
  border-left: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-xl) 0 0 var(--app-radius-xl);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-lg);
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: none;
}
.tool-sheet__panel::-webkit-scrollbar {
  display: none;
}
.tool-sheet__grip {
  display: none;
}

@media (width <= 768px) {
  .tool-sheet {
    background: color-mix(in srgb, var(--m-ink) 50%, transparent);
  }
  .tool-sheet__panel {
    top: auto;
    left: 0;
    width: auto;
    max-height: 92vh;
    gap: 14px;
    padding: 10px 16px calc(24px + env(safe-area-inset-bottom));
    border-left: 0;
    border-top: 1px solid var(--el-border-color-light);
    border-radius: 30px 30px 0 0;
  }
  .tool-sheet__grip {
    display: block;
    flex: none;
    align-self: center;
    width: 44px;
    height: 5px;
    border-radius: 999px;
    background: var(--el-border-color-light);
  }
}

.tool-sheet-enter-active,
.tool-sheet-leave-active {
  transition: opacity 0.2s ease;
}
.tool-sheet-enter-active .tool-sheet__panel,
.tool-sheet-leave-active .tool-sheet__panel {
  transition: transform 0.24s cubic-bezier(0.2, 0.8, 0.2, 1);
}
.tool-sheet-enter-from,
.tool-sheet-leave-to {
  opacity: 0;
}
.tool-sheet-enter-from .tool-sheet__panel,
.tool-sheet-leave-to .tool-sheet__panel {
  transform: translateX(100%);
}
@media (width <= 768px) {
  .tool-sheet-enter-from .tool-sheet__panel,
  .tool-sheet-leave-to .tool-sheet__panel {
    transform: translateY(100%);
  }
}
@media (prefers-reduced-motion: reduce) {
  .tool-sheet-enter-active .tool-sheet__panel,
  .tool-sheet-leave-active .tool-sheet__panel {
    transition: none;
  }
}
</style>
