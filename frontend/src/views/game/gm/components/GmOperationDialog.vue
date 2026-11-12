<!-- 执行一个 GM 操作的对话框：内容是 GmOperationPanel（参数、试运行、危险确认、执行结果）；执行中不能关闭。 -->
<template>
  <FormDialog :model-value="modelValue" :title="operation?.title" :width="640" :show-footer="false" @update:model-value="close">
    <GmOperationPanel
      :operation="operation"
      :targets="targets"
      :initial-params="initialParams"
      :summary="summary"
      :exclude-rewards="excludeRewards"
      :reset-key="openCount"
      closable
      @busy="busy = $event"
      @close="close(false)"
      @done="(out, status) => emit('done', out, status)"
    />
  </FormDialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import type { OperationDef, OperationOutcome } from '@/games/mgp'
import GmOperationPanel from './GmOperationPanel.vue'
import type { GmTargetRef } from './types'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    operation?: OperationDef
    targets?: GmTargetRef[]
    /** 预填参数 */
    initialParams?: Record<string, unknown>
    /** 写进 GM 记录的可展示摘要 */
    summary?: Record<string, string>
    /** 奖励里不允许的资源类型 */
    excludeRewards?: number[]
  }>(),
  { operation: undefined, targets: () => [], initialParams: () => ({}), summary: () => ({}), excludeRewards: () => [] },
)
const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  /** 真实执行完成（不含试运行） */
  done: [result: OperationOutcome | undefined, status: string]
}>()

// 每次打开清空表单与结果
const openCount = ref(0)
const busy = ref(false)
watch(
  () => props.modelValue,
  (open) => {
    if (open) openCount.value++
  },
  { immediate: true },
)

const close = (open: boolean) => {
  if (!open && !busy.value) emit('update:modelValue', false)
}
</script>
