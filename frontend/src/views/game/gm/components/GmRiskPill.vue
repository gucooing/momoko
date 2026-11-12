<!-- 指令风险等级：低 / 中 / 高 / 危险（只决定交互：高要填原因，危险要输入玩家昵称确认）。 -->
<template>
  <StatusPill :variant="variant" :label="t(`gm.risk.${risk || 'low'}`)" />
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ risk?: string }>()
const { t } = useI18n()

const variant = computed(() => {
  switch (props.risk) {
    case 'danger':
      return 'error'
    case 'high':
      return 'warning'
    case 'medium':
      return 'primary'
    default:
      return 'neutral'
  }
})
</script>
