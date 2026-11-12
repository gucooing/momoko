<!-- 培育评价徽标（S+、SS…）。 -->
<template>
  <span v-if="grade" class="gbadge m-num" :class="{ 'is-sm': small }">{{ gradeLabel(grade) }}</span>
</template>

<script setup lang="ts">
import { gradeLabel } from './game'

defineProps<{ grade?: number; small?: boolean }>()
</script>

<style scoped>
.gbadge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  min-width: 28px;
  height: 22px;
  padding: 0 8px;
  border-radius: 8px;
  background: var(--m-ink);
  color: var(--m-bg);
  font-size: 12px;
  font-weight: 800;
  line-height: 1;
}
.gbadge.is-sm {
  min-width: 22px;
  height: 18px;
  padding: 0 5px;
  border-radius: 6px;
  font-size: 10px;
}
</style>
