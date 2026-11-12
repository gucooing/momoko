<!-- 空状态：彩色贴纸图标 + 标题 + 说明 + 可选操作。 -->
<template>
  <div class="pe" :class="{ 'pe--compact': compact }">
    <span class="pe__icon" aria-hidden="true">
      <component :is="icon || InboxIcon" />
    </span>
    <span class="pe__title">{{ title }}</span>
    <span v-if="description" class="pe__desc">{{ description }}</span>
    <div v-if="$slots.default" class="pe__actions"><slot /></div>
  </div>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import { InboxIcon } from '@heroicons/vue/24/outline'

withDefaults(defineProps<{ title: string; description?: string; icon?: Component; compact?: boolean }>(), {
  description: '',
  icon: undefined,
})
</script>

<style scoped>
.pe {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  padding: 36px 20px;
  text-align: center;
}
.pe--compact {
  padding: 20px 12px;
}
.pe__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  margin-bottom: 4px;
  border-radius: 18px;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  transform: rotate(-4deg);
}
.pe--compact .pe__icon {
  width: 44px;
  height: 44px;
  border-radius: 14px;
}
.pe__icon svg {
  width: 26px;
  height: 26px;
}
.pe__title {
  font-size: 15px;
  font-weight: 800;
}
.pe__desc {
  max-width: 360px;
  font-size: 13px;
  line-height: 1.6;
  color: var(--m-ink2);
}
.pe__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: center;
  gap: 8px;
  margin-top: 6px;
}
</style>
