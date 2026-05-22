<template>
  <el-popconfirm
    v-if="!menuStore.isMobile"
    v-bind="attrs"
    :title="title"
    :placement="placement"
    :width="width"
    :disabled="disabled"
    :confirm-button-text="confirmButtonText"
    :cancel-button-text="cancelButtonText"
    :show-after="showAfter"
    @confirm="handleConfirm"
  >
    <template #reference>
      <slot name="reference">
        <slot />
      </slot>
    </template>
  </el-popconfirm>
  <span
    v-else
    class="adaptive-confirm__reference"
    :class="{ 'is-disabled': disabled }"
    @click="handleMobileTriggerClick"
  >
    <slot name="reference">
      <slot />
    </slot>
  </span>
</template>

<script setup lang="ts">
import { getCurrentInstance, useAttrs } from 'vue'
import { useMenuStore } from '@/stores/menu'
import { Dialog } from '@/utils/dialog'

defineOptions({
  inheritAttrs: false,
})

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

defineEmits<{
  confirm: []
}>()

const props = withDefaults(defineProps<Props>(), {
  title: '',
  dialogTitle: '确认',
  width: 220,
  placement: 'top',
  disabled: false,
  confirmButtonText: '确定',
  cancelButtonText: '取消',
  showAfter: 0,
})

const attrs = useAttrs()
const instance = getCurrentInstance()
const menuStore = useMenuStore()

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

const handleMobileTriggerClick = () => {
  if (props.disabled) return

  Dialog.confirm({
    title: props.dialogTitle,
    content: props.title,
    confirmText: props.confirmButtonText,
    cancelText: props.cancelButtonText,
    onConfirm: handleConfirm,
  })
}
</script>

<style scoped lang="scss">
.adaptive-confirm__reference {
  display: inline-flex;
  max-width: 100%;
}

.adaptive-confirm__reference.is-disabled {
  cursor: not-allowed;
}
</style>
