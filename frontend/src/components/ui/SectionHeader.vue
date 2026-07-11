<!-- 区块标题：比面板更轻，用于卡片流之上。字重克制、次要信息降级为灰。 -->
<template>
  <div class="section-header">
    <div class="section-header__text">
      <h3>
        {{ title }}
        <span v-if="count != null" class="section-header__count">{{ count }}</span>
      </h3>
      <p v-if="caption">{{ caption }}</p>
    </div>
    <div v-if="$slots.actions" class="section-header__actions"><slot name="actions" /></div>
  </div>
</template>

<script setup lang="ts">
defineProps<{ title?: string; caption?: string; count?: number | null }>()
</script>

<style scoped lang="scss">
.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-bottom: 0.85rem;
}
.section-header__text {
  min-width: 0;
}
.section-header h3 {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 0.9375rem;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}
.section-header__count {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  padding: 1px 8px;
  border-radius: 999px;
  font-variant-numeric: tabular-nums;
}
.section-header p {
  margin-top: 2px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.section-header__actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}
</style>
