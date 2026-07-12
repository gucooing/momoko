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
        <UButton color="primary" size="sm" icon="i-lucide-search" @click="$emit('search')">
          {{ t('system.common.search') }}
        </UButton>
        <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-rotate-ccw" @click="$emit('reset')">
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
  border-radius: var(--app-radius);
  padding: 10px 12px;
}
.filter-bar__body {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 10px;
}
.filter-bar__fields {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  flex: 1;
  min-width: 0;
}
.filter-bar__fields > :deep(*) {
  width: 190px;
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
  gap: 6px;
  width: 100%;
  min-height: 28px;
  border: none;
  background: transparent;
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
}
.filter-bar__toggle-ico {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.filter-bar__toggle-chev {
  width: 14px;
  height: 14px;
  margin-left: auto;
  color: var(--el-text-color-placeholder);
}
@media (width <= 768px) {
  .filter-bar {
    padding: 8px 10px;
  }
  .filter-bar__body {
    margin-top: 8px;
  }
  .filter-bar__fields {
    flex-direction: column;
    width: 100%;
    gap: 8px;
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
