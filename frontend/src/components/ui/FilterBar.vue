<!-- 筛选条：横向筛选字段（#fields 插槽）+ 搜索/重置。移动端折叠为可展开区（M-1）。
     字段用令牌化 .app-input/.app-select（见 design-tokens.css）。 -->
<template>
  <div class="filter-bar">
    <button v-if="menuStore.isMobile" type="button" class="filter-bar__toggle" @click="expanded = !expanded">
      <component :is="menuStore.iconComponents['HOutline:AdjustmentsHorizontalIcon']" class="filter-bar__toggle-ico" />
      <span>{{ t('system.common.search') }}</span>
      <component
        :is="menuStore.iconComponents[expanded ? 'HOutline:ChevronUpIcon' : 'HOutline:ChevronDownIcon']"
        class="filter-bar__toggle-chev"
      />
    </button>

    <div v-show="!menuStore.isMobile || expanded" class="filter-bar__body">
      <div class="filter-bar__fields"><slot name="fields" /></div>
      <div class="filter-bar__actions">
        <UButton color="primary" icon="i-lucide-search" @click="$emit('search')">
          {{ t('system.common.search') }}
        </UButton>
        <UButton color="neutral" variant="soft" icon="i-lucide-rotate-ccw" @click="$emit('reset')">
          {{ t('system.common.reset') }}
        </UButton>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineEmits<{ search: []; reset: [] }>()
const menuStore = useMenuStore()
const { t } = useI18n()
const expanded = ref(false)
</script>

<style scoped lang="scss">
.filter-bar {
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  box-shadow: var(--app-shadow-card);
  padding: 16px;
}
.filter-bar__body {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 12px;
}
.filter-bar__fields {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
  flex: 1;
  min-width: 0;
}
.filter-bar__fields > :deep(*) {
  width: 216px;
}
.filter-bar__actions {
  display: flex;
  align-items: flex-end;
  gap: 8px;
}
/* 移动端折叠触发器 */
.filter-bar__toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  border: none;
  background: transparent;
  color: var(--el-text-color-regular);
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
}
.filter-bar__toggle-ico {
  width: 18px;
  height: 18px;
  color: var(--el-text-color-secondary);
}
.filter-bar__toggle-chev {
  width: 16px;
  height: 16px;
  margin-left: auto;
  color: var(--el-text-color-placeholder);
}
@media (width <= 768px) {
  .filter-bar__body {
    margin-top: 12px;
  }
  .filter-bar__fields {
    flex-direction: column;
    width: 100%;
  }
  .filter-bar__fields > :deep(*) {
    width: 100%;
  }
  .filter-bar__actions {
    width: 100%;
  }
  .filter-bar__actions > :deep(*) {
    flex: 1;
  }
}
</style>
