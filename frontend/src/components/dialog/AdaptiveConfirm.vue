<!-- 自适应确认：桌面/移动统一走令牌 Dialog.confirm（去 el-popconfirm）。
     触发器仍是默认/reference 插槽；点击后弹确认。 -->
<template>
  <span
    class="adaptive-confirm__reference"
    :class="{ 'is-disabled': disabled }"
    @click="handleTriggerClick"
  >
    <slot name="reference">
      <slot />
    </slot>
  </span>
</template>

<script setup lang="ts">
import { getCurrentInstance } from 'vue'
import { Dialog } from '@/utils/dialog'
import { useI18n } from 'vue-i18n'

defineOptions({ inheritAttrs: false })

interface Props {
  title?: string
  dialogTitle?: string
  width?: string | number
  placement?: string
  disabled?: boolean
  confirmButtonText?: string
  cancelButtonText?: string
  showAfter?: number
}

defineEmits<{ confirm: [] }>()

const props = withDefaults(defineProps<Props>(), {
  title: '',
  width: 220,
  placement: 'top',
  disabled: false,
  showAfter: 0,
})

const instance = getCurrentInstance()
const { t } = useI18n()
const resolvedDialogTitle = computed(() => props.dialogTitle || t('dialog.title.confirm'))
const resolvedConfirmButtonText = computed(() => props.confirmButtonText || t('common.confirm'))
const resolvedCancelButtonText = computed(() => props.cancelButtonText || t('common.cancel'))

type ConfirmListener = () => void | Promise<void>

const resolveConfirmListeners = (): ConfirmListener[] => {
  const value = instance?.vnode.props?.onConfirm
  if (Array.isArray(value)) {
    return value.filter((item): item is ConfirmListener => typeof item === 'function')
  }
  return typeof value === 'function' ? [value as ConfirmListener] : []
}

const handleConfirm = async () => {
  for (const listener of resolveConfirmListeners()) {
    await listener()
  }
}

const handleTriggerClick = () => {
  if (props.disabled) return
  Dialog.confirm({
    title: resolvedDialogTitle.value,
    content: props.title,
    confirmText: resolvedConfirmButtonText.value,
    cancelText: resolvedCancelButtonText.value,
    onConfirm: handleConfirm,
  }).catch(() => {
    // 取消时 Dialog.confirm reject('cancel')，此处吞掉避免未捕获 promise
  })
}
</script>

<style scoped lang="scss">
.adaptive-confirm__reference {
  display: inline-flex;
  max-width: 100%;
  cursor: pointer;
}
.adaptive-confirm__reference.is-disabled {
  cursor: not-allowed;
  opacity: 0.55;
  pointer-events: none;
}
</style>
