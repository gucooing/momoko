<!-- 发送到礼物箱（操作 gift.send）对话框：内容是 GmGiftSend（收件人可搜索添加、礼物内容、领取期限、预览与发送）；
     发送中不能关闭。 -->
<template>
  <FormDialog :model-value="modelValue" :title="operation?.title || t('gmData.gift.title')" :width="640" :show-footer="false" @update:model-value="close">
    <GmGiftSend :players="players" :reset-key="openCount" closable @busy="busy = $event" @close="close(false)" @done="emit('done')" />
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import FormDialog from '@/components/ui/FormDialog.vue'
import { useGmContext } from '../context'
import GmGiftSend, { type GiftTarget } from '../components/GmGiftSend.vue'

const props = withDefaults(defineProps<{ modelValue: boolean; players?: GiftTarget[] }>(), { players: () => [] })
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; done: [] }>()

const { t } = useI18n()
const ctx = useGmContext()
const operation = computed(() => ctx.operation('gift.send'))

// 每次打开清空内容与结果
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
