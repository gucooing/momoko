<!-- 弹窗外壳（令牌驱动，替代 el-dialog）：Teleport + 遮罩 + 头/体/脚。
     保留命令式 Dialog.* 契约：attrs.onConfirm（异步 + 确认按钮 loading）、emit close/update:modelValue、
     #header/#default/#footer 插槽、showClose/showFooter/showCancel/showConfirm、width + 移动端自适应。
     去掉模板遗留的 全屏/拖拽/缩放（新设计克制，统一固定居中卡，与 FormDialog/FileDialog 一致）。 -->
<template>
  <Teleport to="body">
    <Transition name="base-dialog">
      <div v-if="modelValue" class="base-dialog-overlay" @mousedown.self="onOverlay">
        <div
          class="base-dialog-panel"
          :style="{ maxWidth: panelWidth }"
          role="dialog"
          aria-modal="true"
        >
          <header class="base-dialog-head">
            <div class="base-dialog-title">
              <slot name="header">{{ title }}</slot>
            </div>
            <AppIconButton
              v-if="showClose"
              icon="HOutline:XMarkIcon"
              :label="t('common.close')"
              :size="18"
              @click="close"
            />
          </header>

          <div class="base-dialog-body"><slot /></div>

          <footer v-if="hasFooter" class="base-dialog-foot">
            <slot name="footer">
              <UButton v-if="showCancelButton" color="neutral" variant="soft" @click="close">
                {{ resolvedCancelText }}
              </UButton>
              <UButton
                v-if="showConfirmButton"
                color="primary"
                :loading="showConfirmLoading ? confirmLoading : false"
                @click="confirm"
              >
                {{ resolvedConfirmText }}
              </UButton>
            </slot>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useAttrs } from 'vue'
import { useWindowSize } from '@vueuse/core'
import { useI18n } from 'vue-i18n'

// 命令式 Dialog.* 透传 onConfirm 需从 attrs 读取，禁用自动继承避免落到 DOM。
defineOptions({ inheritAttrs: false })

interface IProps {
  modelValue: boolean
  title?: string
  showClose?: boolean
  showCancelButton?: boolean
  showConfirmButton?: boolean
  showFooter?: boolean
  cancelText?: string
  confirmText?: string
  // 确认按钮加载态（配合 attrs.onConfirm 的 Promise）
  showConfirmLoading?: boolean
  width?: string | number
  closeOnOverlay?: boolean
  // 移动端自适应宽度
  mobileAdaptive?: boolean
  mobileWidth?: string | number
  mobileBreakpoint?: number
}

const props = withDefaults(defineProps<IProps>(), {
  showClose: true,
  showFooter: true,
  showCancelButton: true,
  showConfirmButton: true,
  showConfirmLoading: true,
  width: 460,
  closeOnOverlay: true,
  mobileAdaptive: true,
  mobileWidth: '90%',
  mobileBreakpoint: 768,
})

const emits = defineEmits<{
  'update:modelValue': [value: boolean]
  close: []
  /** 打开时同步触发（兼容旧 el-dialog @open） */
  open: []
  /** 打开后下一帧触发（兼容旧 el-dialog @opened，便于量测 DOM） */
  opened: []
}>()

const { t } = useI18n()
const attrs = useAttrs()
const slots = useSlots()

const confirmLoading = ref(false)
const resolvedCancelText = computed(() => props.cancelText || t('common.cancel'))
const resolvedConfirmText = computed(() => props.confirmText || t('common.confirm'))
const hasFooter = computed(() => !!slots.footer || props.showFooter)

const { width: windowWidth } = useWindowSize()
const isMobile = computed(() => windowWidth.value < (props.mobileBreakpoint ?? 768))

const panelWidth = computed(() => {
  const raw = props.mobileAdaptive && isMobile.value ? props.mobileWidth : props.width
  if (typeof raw === 'number') return `${raw}px`
  const s = String(raw ?? '').trim()
  // '440' → '440px'；'90%' / 'calc(...)' / '640px' 原样
  return /^\d+$/.test(s) ? `${s}px` : s
})

const close = () => {
  emits('update:modelValue', false)
  emits('close')
}

const onOverlay = () => {
  if (props.closeOnOverlay) close()
}

// 确认：读取命令式/声明式透传的 onConfirm，带加载态等待其完成（不自动关闭，交由调用方）。
const confirm = async () => {
  if (props.showConfirmLoading) confirmLoading.value = true
  try {
    const onConfirm = attrs.onConfirm as (() => Promise<void> | void) | undefined
    if (onConfirm) await onConfirm()
  } finally {
    if (props.showConfirmLoading) confirmLoading.value = false
  }
}

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && props.modelValue) close()
}

watch(
  () => props.modelValue,
  async (open) => {
    document.documentElement.style.overflow = open ? 'hidden' : ''
    if (open) {
      emits('open')
      await nextTick()
      emits('opened')
    }
  },
)

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  document.documentElement.style.overflow = ''
})
</script>

<style scoped lang="scss">
.base-dialog-overlay {
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

.base-dialog-panel {
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

.base-dialog-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  padding: 16px 16px 16px 20px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.base-dialog-title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.base-dialog-body {
  padding: 20px;
  overflow-y: auto;
}

.base-dialog-foot {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding: 14px 20px;
  border-top: 1px solid var(--el-border-color-lighter);
}

@media (width <= 768px) {
  .base-dialog-overlay {
    padding: 0;
    align-items: flex-end;
  }
  .base-dialog-panel {
    max-width: none !important;
    max-height: 92vh;
    border-radius: var(--app-radius-lg) var(--app-radius-lg) 0 0;
  }
}

.base-dialog-enter-active,
.base-dialog-leave-active {
  transition: opacity 0.18s ease;
}
.base-dialog-enter-active .base-dialog-panel,
.base-dialog-leave-active .base-dialog-panel {
  transition: transform 0.18s cubic-bezier(0.4, 0, 0.2, 1);
}
.base-dialog-enter-from,
.base-dialog-leave-to {
  opacity: 0;
}
.base-dialog-enter-from .base-dialog-panel,
.base-dialog-leave-to .base-dialog-panel {
  transform: translateY(8px) scale(0.98);
}
</style>
