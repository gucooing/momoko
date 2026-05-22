<template>
  <div class="overview-strip">
    <div
      v-for="card in cards"
      :key="card.label"
      :class="['overview-item', `overview-item--${card.skin}`]"
    >
      <div class="overview-content">
        <div class="overview-icon">
          <el-icon size="16">
            <component :is="menuStore.iconComponents[card.icon]" />
          </el-icon>
        </div>

        <div class="overview-main">
          <div class="overview-label">{{ card.label }}</div>
          <div class="overview-meta">
            <div class="overview-value">{{ card.value }}</div>
            <div class="overview-note">{{ card.note }}</div>
          </div>
        </div>
      </div>
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
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 0;
  padding: 0;
}

.overview-item {
  position: relative;
  min-width: 0;
  padding: 0.55rem 0.85rem;
}

.overview-item:not(:last-child)::after {
  position: absolute;
  top: 0.4rem;
  right: 0;
  height: calc(100% - 0.8rem);
  width: 1px;
  background: var(--el-border-color-extra-light);
  content: '';
}

.overview-item--tone-a {
  .overview-icon {
    background: color-mix(in srgb, #6366f1 10%, var(--el-bg-color-page));
    color: #6366f1;
  }
}

.overview-item--tone-b {
  .overview-icon {
    background: color-mix(in srgb, #10b981 10%, var(--el-bg-color-page));
    color: #10b981;
  }
}

.overview-item--tone-c {
  .overview-icon {
    background: color-mix(in srgb, #f59e0b 10%, var(--el-bg-color-page));
    color: #f59e0b;
  }
}

.overview-content {
  display: flex;
  min-height: 3.15rem;
  align-items: center;
  gap: 0.75rem;
}

.overview-icon {
  display: flex;
  height: 1.95rem;
  width: 1.95rem;
  flex-shrink: 0;
  align-items: center;
  justify-content: center;
  border-radius: 0.68rem;
}

.overview-main {
  display: flex;
  min-width: 0;
  flex-direction: column;
  gap: 0.16rem;
  flex: 1;
}

.overview-meta {
  display: flex;
  min-width: 0;
  align-items: baseline;
  gap: 0.5rem;
}

.overview-value {
  font-size: 1.45rem;
  font-weight: 800;
  line-height: 1.05;
  color: var(--el-text-color-primary);
}

.overview-label {
  font-size: 0.76rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}

.overview-note {
  overflow: hidden;
  font-size: 0.76rem;
  color: var(--el-text-color-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}

:global(html.dark .overview-item:not(:last-child)::after) {
  background: color-mix(in srgb, var(--el-border-color) 80%, transparent);
}

@media (width <= 768px) {
  .overview-strip {
    grid-template-columns: 1fr;
  }

  .overview-item {
    padding: 0.45rem 0;
  }

  .overview-item:not(:last-child)::after {
    top: auto;
    bottom: 0;
    height: 1px;
    width: 100%;
  }

  .overview-content {
    min-height: 2.9rem;
    gap: 0.65rem;
  }

  .overview-icon {
    height: 1.8rem;
    width: 1.8rem;
  }

  .overview-meta {
    gap: 0.42rem;
  }

  .overview-value {
    font-size: 1.28rem;
  }

  .overview-label {
    font-size: 0.72rem;
  }

  .overview-note {
    font-size: 0.72rem;
  }
}
</style>
