<!-- 游标分页的翻页条：第 N 页 · 共 M 条，上一页 / 下一页。 -->
<template>
  <div v-if="total > 0 || hasPrev" class="azp">
    <span class="azp__info m-num">{{ t('azurGm.common.page', { n: page, total }) }}</span>
    <AppIconButton icon="HOutline:ChevronLeftIcon" :label="t('azurGm.common.prev')" :box="28" :disabled="!hasPrev || loading" @click="emit('prev')" />
    <AppIconButton icon="HOutline:ChevronRightIcon" :label="t('azurGm.common.next')" :box="28" :disabled="!hasNext || loading" @click="emit('next')" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineProps<{ page: number; total: number; hasPrev: boolean; hasNext: boolean; loading?: boolean }>()
const emit = defineEmits<{ prev: []; next: [] }>()
const { t } = useI18n()
</script>

<style scoped>
.azp {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  padding-top: 10px;
}
.azp__info {
  margin-right: 6px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
</style>
