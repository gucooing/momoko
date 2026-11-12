<!-- 右侧抽屉（手机上铺满全屏）：标题 + 关闭、可滚动的内容、底部操作。
     传送到 body 时带上游戏 id 与 game-skin，抽屉里仍是游戏配色。兑换码与 GM 记录共用。 -->
<template>
  <Teleport to="body">
    <Transition name="sdw">
      <div v-if="modelValue" class="sdw game-skin" :data-game="game || undefined" @mousedown.self="close">
        <aside class="sdw__panel" role="dialog" aria-modal="true" :aria-label="title" :style="{ maxWidth: `${width}px` }">
          <header class="sdw__head">
            <span class="sdw__title">{{ title }}</span>
            <slot name="head" />
            <AppIconButton icon="HOutline:XMarkIcon" :label="t('gmOps.common.close')" @click="close" />
          </header>
          <div class="sdw__body"><slot /></div>
          <footer v-if="$slots.footer" class="sdw__foot"><slot name="footer" /></footer>
        </aside>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const props = withDefaults(defineProps<{ modelValue: boolean; title?: string; width?: number; game?: string }>(), {
  title: '',
  width: 600,
  game: '',
})
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const { t } = useI18n()

const close = () => emit('update:modelValue', false)
const onKey = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.modelValue) close()
}
watch(
  () => props.modelValue,
  (open) => {
    document.documentElement.style.overflow = open ? 'hidden' : ''
  },
)
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKey)
  if (props.modelValue) document.documentElement.style.overflow = ''
})
</script>

<style scoped>
.sdw {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: flex;
  justify-content: flex-end;
  background: color-mix(in srgb, var(--m-ink) 32%, transparent);
}
.sdw__panel {
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: -12px 0 32px -16px color-mix(in srgb, var(--m-ink) 40%, transparent);
}
.sdw__head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 16px 20px;
  border-bottom: 1px solid var(--el-border-color-light);
}
.sdw__title {
  flex: 1;
  min-width: 0;
  font-family: var(--app-font-display);
  font-size: 20px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sdw__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: 20px 22px;
  scrollbar-width: thin;
}
.sdw__foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 14px 22px;
  border-top: 1px solid var(--el-border-color-light);
}
.sdw-enter-active,
.sdw-leave-active {
  transition: opacity 0.2s ease;
}
.sdw-enter-active .sdw__panel,
.sdw-leave-active .sdw__panel {
  transition: transform 0.25s ease;
}
.sdw-enter-from,
.sdw-leave-to {
  opacity: 0;
}
.sdw-enter-from .sdw__panel,
.sdw-leave-to .sdw__panel {
  transform: translateX(32px);
}
@media (width < 768px) {
  .sdw__body {
    padding: 16px;
  }
  .sdw__foot {
    padding: 12px 16px;
  }
  .sdw__foot :deep(button) {
    min-height: 44px;
  }
}
@media (prefers-reduced-motion: reduce) {
  .sdw-enter-active,
  .sdw-leave-active,
  .sdw-enter-active .sdw__panel,
  .sdw-leave-active .sdw__panel {
    transition: none;
  }
}
</style>
