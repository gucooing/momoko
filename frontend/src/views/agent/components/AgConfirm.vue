<!-- 紧凑的确认弹窗：标题 + 一段说明（默认插槽可以再放复选框之类）+ 取消 / 确定。
     onConfirm 返回 Promise 时按钮转圈，等它结束再关；抛错则保持打开。 -->
<template>
  <AgDialog :model-value="modelValue" :title="title" :width="width" close-on-overlay @update:model-value="emit('update:modelValue', $event)">
    <div v-if="message" class="ag-confirm__msg">{{ message }}</div>
    <slot />
    <div class="ag-confirm__acts">
      <button type="button" class="ag-btn" @click="emit('update:modelValue', false)">{{ cancelText || t('agent.common.cancel') }}</button>
      <button type="button" class="ag-btn" :class="danger ? 'ag-btn--stop' : 'ag-btn--p'" :disabled="busy" @click="run">
        <UIcon v-if="busy" name="i-lucide-loader-circle" class="ag-spin" />
        {{ confirmText || t('agent.common.confirm') }}
      </button>
    </div>
  </AgDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import AgDialog from './AgDialog.vue'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title: string
    message?: string
    confirmText?: string
    cancelText?: string
    danger?: boolean
    width?: number
    /** 点确定时执行 */
    onConfirm?: () => unknown
  }>(),
  { message: '', confirmText: '', cancelText: '', danger: false, width: 400, onConfirm: undefined },
)
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const { t } = useI18n()
const busy = ref(false)

const run = async () => {
  if (busy.value) return
  busy.value = true
  try {
    await props.onConfirm?.()
    emit('update:modelValue', false)
  } catch {
    // 请求层已经提示过错误，弹窗留着
  } finally {
    busy.value = false
  }
}
</script>

<style lang="scss">
.ag-confirm__msg {
  font-size: 13px;
  line-height: 1.6;
  color: var(--m-ink2);
}
.ag-confirm__acts {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
  padding-top: 2px;
}
</style>
