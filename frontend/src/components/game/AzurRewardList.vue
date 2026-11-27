<!-- 蓝色星原奖励的图标列表：图标、名称与数量；compact 时只有图标与数量（名称在悬停提示里），用在表格里。
     管理端与用户通行证共用：名称由调用方按各自的语言给出（name）。 -->
<template>
  <span v-if="rewards?.length" class="arl" :class="{ 'arl--compact': compact }">
    <span v-for="(r, i) in rewards" :key="`${r.type}:${r.id}:${i}`" class="arl__item" :title="`${name(r.type, r.id)} ×${r.num}`">
      <GameImage :src="azurpromilia.rewardImage(r.type, r.id)" alt="" class="arl__icon" />
      <span v-if="!compact" class="arl__name">{{ name(r.type, r.id) }}</span>
      <span class="arl__num">×{{ r.num }}</span>
    </span>
  </span>
  <span v-else>—</span>
</template>

<script setup lang="ts">
import { azurpromilia } from '@/games'
import GameImage from './GameImage.vue'

defineProps<{
  rewards: { type: number; id: number; num: number }[] | undefined
  name: (type: number, id: number) => string
  compact?: boolean
}>()
</script>

<style scoped>
.arl {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 12px;
  min-width: 0;
}
.arl--compact {
  flex-wrap: nowrap;
  gap: 8px;
  overflow: hidden;
}
.arl__item {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}
.arl__icon {
  flex: none;
  width: 28px;
  height: 28px;
  object-fit: contain;
}
.arl--compact .arl__icon {
  width: 24px;
  height: 24px;
}
.arl__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.arl__num {
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}
</style>
