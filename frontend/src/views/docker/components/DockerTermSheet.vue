<!-- Docker 终端类面板外壳（日志 / 终端 / 任务日志共用）：PC 居中大面板，手机底部面板（顶部拖拽条 + 大圆角）。
     头部：彩色贴纸图标 + 标题 + 副标题 + 关闭；默认插槽放 TerminalConsole 等内容（其外框由本壳接管）。 -->
<template>
  <Teleport to="body">
    <Transition name="dk-sheet">
      <div v-if="modelValue" class="dk-sheet" @mousedown.self="close">
        <section class="dk-sheet__panel" role="dialog" aria-modal="true" :aria-label="title">
          <span class="dk-sheet__grab" aria-hidden="true" />
          <header class="dk-sheet__head">
            <span class="m-sticker dk-sheet__sticker" :class="`m-sticker--${tone}`">
              <component :is="menuStore.iconComponents[icon]" />
            </span>
            <span class="dk-sheet__text">
              <span class="dk-sheet__title">{{ title }}</span>
              <span v-if="subtitle" class="dk-sheet__sub">{{ subtitle }}</span>
            </span>
            <slot name="head-extra" />
            <AppIconButton icon="HOutline:XMarkIcon" :label="t('system.common.close')" :box="44" :size="20" @click="close" />
          </header>
          <div class="dk-sheet__body"><slot /></div>
        </section>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title: string
    subtitle?: string
    icon?: string
    tone?: 'p' | 'a' | 'a2' | 'ok' | 'bad' | 'warn'
  }>(),
  { icon: 'HOutline:CommandLineIcon', tone: 'a2' },
)
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()
const menuStore = useMenuStore()

const close = () => emit('update:modelValue', false)
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
  document.documentElement.style.overflow = ''
})
</script>

<style scoped lang="scss">
.dk-sheet {
  position: fixed;
  inset: 0;
  z-index: 2200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: rgba(12, 6, 18, 0.45);
  backdrop-filter: blur(2px);
}
.dk-sheet__panel {
  display: flex;
  flex-direction: column;
  gap: 12px;
  width: min(1100px, 96vw);
  height: min(84vh, 780px);
  padding: 18px 18px 14px;
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-lg);
  overflow: hidden;
}
.dk-sheet__grab {
  display: none;
  align-self: center;
  flex: none;
  width: 44px;
  height: 5px;
  border-radius: 999px;
  background: var(--el-border-color-light);
}
.dk-sheet__head { display: flex; align-items: center; gap: 12px; min-width: 0; }
.dk-sheet__sticker {
  width: 46px;
  height: 46px;
  flex: none;
  border-radius: 15px;
  transform: rotate(-5deg);
  :deep(svg) { width: 22px; height: 22px; }
}
.dk-sheet__text { display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0; }
.dk-sheet__title {
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.dk-sheet__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.dk-sheet__body { position: relative; flex: 1; min-height: 0; display: flex; flex-direction: column; }
/* 内嵌终端外壳：外框交给本面板，避免卡中卡 */
.dk-sheet__body :deep(.term-shell) {
  flex: 1;
  min-height: 0;
  padding: 0;
  border: none;
  border-radius: 0;
  background: transparent;
}

.dk-sheet-enter-active,
.dk-sheet-leave-active { transition: opacity 0.18s ease; }
.dk-sheet-enter-active .dk-sheet__panel,
.dk-sheet-leave-active .dk-sheet__panel { transition: transform 0.22s cubic-bezier(0.4, 0, 0.2, 1); }
.dk-sheet-enter-from,
.dk-sheet-leave-to { opacity: 0; }
.dk-sheet-enter-from .dk-sheet__panel,
.dk-sheet-leave-to .dk-sheet__panel { transform: translateY(10px) scale(0.98); }

@media (width <= 768px) {
  .dk-sheet { align-items: flex-end; padding: 0; background: rgba(12, 6, 18, 0.5); }
  .dk-sheet__panel {
    width: 100vw;
    height: calc(100dvh - 28px);
    padding: 10px 12px calc(12px + env(safe-area-inset-bottom));
    gap: 10px;
    border-radius: 28px 28px 0 0;
  }
  .dk-sheet__grab { display: block; }
  .dk-sheet__sticker { width: 42px; height: 42px; border-radius: 14px; }
  .dk-sheet__title { font-size: 1.0625rem; }
  .dk-sheet__sub { font-size: 0.75rem; }
  .dk-sheet-enter-from .dk-sheet__panel,
  .dk-sheet-leave-to .dk-sheet__panel { transform: translateY(100%); }
}
</style>
