<template>
  <div class="overview-strip">
    <div
      v-for="card in cards"
      :key="card.label"
      :class="['overview-item', `overview-item--${card.skin}`]"
    >
      <div class="overview-icon">
        <el-icon size="16">
          <component :is="menuStore.iconComponents[card.icon]" />
        </el-icon>
      </div>
      <div class="overview-body">
        <div class="overview-label">{{ card.label }}</div>
        <div class="overview-value">{{ card.value }}</div>
      </div>
      <div class="overview-note">{{ card.note }}</div>
    </div>
  </div>
</template>

<script setup lang="ts">
import type { OverviewCardItem } from '@/stores/instance/types'

defineProps<{
  cards: OverviewCardItem[]
}>()

const menuStore = useMenuStore()
</script>

<style scoped lang="scss">
.overview-strip {
  display: flex;
  gap: 1rem;
}

.overview-item {
  display: flex;
  align-items: center;
  gap: 0.65rem;
  flex: 1;
  min-width: 0;
  padding: 0.7rem 0.9rem;
  border-radius: 0.75rem;
  border: 1px solid var(--el-border-color-extra-light);
  background: var(--el-bg-color);
}

.overview-icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 2rem;
  height: 2rem;
  border-radius: 0.6rem;
  flex-shrink: 0;
}

.overview-item--tone-a .overview-icon {
  background: color-mix(in srgb, #6366f1 12%, var(--el-bg-color));
  color: #6366f1;
}

.overview-item--tone-b .overview-icon {
  background: color-mix(in srgb, #10b981 12%, var(--el-bg-color));
  color: #10b981;
}

.overview-item--tone-c .overview-icon {
  background: color-mix(in srgb, #f59e0b 12%, var(--el-bg-color));
  color: #f59e0b;
}

.overview-body {
  min-width: 0;
}

.overview-value {
  font-size: 1.25rem;
  font-weight: 800;
  line-height: 1.2;
  color: var(--el-text-color-primary);
}

.overview-label {
  font-size: 0.72rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}

.overview-note {
  margin-left: auto;
  font-size: 0.72rem;
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}

:global(html.dark .overview-item) {
  background: color-mix(in srgb, #ffffff 3%, var(--el-bg-color));
  border-color: color-mix(in srgb, var(--el-border-color) 60%, transparent);
}

@media (width <= 768px) {
  .overview-strip {
    flex-direction: column;
  }

  .overview-item {
    padding: 0.55rem 0.8rem;
  }
}
</style>
