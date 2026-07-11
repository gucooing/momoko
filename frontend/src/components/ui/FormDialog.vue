<!-- 弹窗外壳（令牌驱动，替代 el-dialog）：teleport + 遮罩 + 头/体/脚。CRUD 表单用它。
     移动端近全宽、体区滚动（02 M-3）。页脚默认 取消/确定（UButton）。 -->
<template>
  <Teleport to="body">
    <Transition name="form-dialog">
      <div v-if="modelValue" class="form-dialog" @mousedown.self="onOverlay">
        <div class="form-dialog__panel" :style="{ maxWidth: panelWidth }" role="dialog" aria-modal="true">
          <header class="form-dialog__head">
            <h3 class="form-dialog__title">{{ title }}</h3>
            <AppIconButton icon="HOutline:XMarkIcon" :label="t('system.common.close')" :size="18" @click="close" />
          </header>

          <div class="form-dialog__body"><slot /></div>

          <footer v-if="showFooter" class="form-dialog__foot">
            <slot name="footer" :close="close" :confirm="confirm">
              <UButton color="neutral" variant="soft" @click="close">
                {{ cancelText || t('system.common.cancel') }}
              </UButton>
              <UButton color="primary" :loading="loading" @click="confirm">
                {{ confirmText || t('system.common.confirm') }}
              </UButton>
            </slot>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title?: string
    width?: number
    loading?: boolean
    showFooter?: boolean
    closeOnOverlay?: boolean
    confirmText?: string
    cancelText?: string
  }>(),
  { width: 600, showFooter: true, closeOnOverlay: true },
)
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; confirm: []; close: [] }>()

const { t } = useI18n()
const panelWidth = computed(() => `${props.width}px`)

const close = () => {
  emit('update:modelValue', false)
  emit('close')
}
const confirm = () => emit('confirm')
const onOverlay = () => {
  if (props.closeOnOverlay) close()
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
  document.documentElement.style.overflow = ''
})
</script>

<style scoped lang="scss">
.form-dialog {
  position: fixed;
  inset: 0;
  z-index: 2200;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: color-mix(in srgb, #0b1220 45%, transparent);
  backdrop-filter: blur(2px);
}
.form-dialog__panel {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-height: calc(100vh - 48px);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-lg);
  box-shadow: var(--app-shadow-lg);
  overflow: hidden;
}
.form-dialog__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 16px 16px 16px 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.form-dialog__title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.form-dialog__body {
  padding: 20px;
  overflow-y: auto;
}
.form-dialog__foot {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding: 14px 20px;
  border-top: 1px solid var(--el-border-color-lighter);
}
@media (width <= 768px) {
  .form-dialog {
    padding: 0;
    align-items: flex-end;
  }
  .form-dialog__panel {
    max-width: none !important;
    max-height: 92vh;
    border-radius: var(--app-radius-lg) var(--app-radius-lg) 0 0;
  }
}

.form-dialog-enter-active,
.form-dialog-leave-active {
  transition: opacity 0.18s ease;
}
.form-dialog-enter-active .form-dialog__panel,
.form-dialog-leave-active .form-dialog__panel {
  transition: transform 0.18s cubic-bezier(0.4, 0, 0.2, 1);
}
.form-dialog-enter-from,
.form-dialog-leave-to {
  opacity: 0;
}
.form-dialog-enter-from .form-dialog__panel,
.form-dialog-leave-to .form-dialog__panel {
  transform: translateY(8px) scale(0.98);
}
</style>
