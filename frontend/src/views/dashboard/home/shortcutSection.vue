<!-- 快捷入口：彩色「贴纸」图标块网格，只显示当前用户在当前节点可进入的页面。 -->
<template>
  <AppPanel v-if="visibleShortcuts.length" :title="t('dashboard.home.convenientTools')" title-icon="HOutline:Squares2X2Icon">
    <div class="sc__grid">
      <button
        v-for="(item, i) in visibleShortcuts"
        :key="item.labelKey"
        type="button"
        class="sc__item"
        @click="router.push(item.routePath)"
      >
        <span
          class="sc__icon m-sticker"
          :class="`m-sticker--${item.color}`"
          :style="{ transform: `rotate(${ROTS[i % ROTS.length]})` }"
        >
          <component :is="menuStore.iconComponents[item.icon]" />
        </span>
        <span class="sc__label">{{ t(item.labelKey) }}</span>
      </button>
    </div>
  </AppPanel>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import { useI18n } from 'vue-i18n'

const ROTS = ['-4deg', '3deg', '-2deg', '4deg', '-3deg', '2deg']

const router = useRouter()
const menuStore = useMenuStore()
const dashboardHomeStore = useDashboardHomeStore()
const { t } = useI18n()
const nodeStore = useNodeStore()
const accessible = useAccessibleRoutes()
const { shortcuts } = storeToRefs(dashboardHomeStore)
// 只展示菜单里存在、且当前节点授权的入口
const visibleShortcuts = computed(() =>
  shortcuts.value.filter(
    (item) => accessible.value.has(item.routePath) && nodeStore.canAccessRoute(item.routePath),
  ),
)
</script>

<style scoped lang="scss">
.sc__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px 6px;
}
@media (width >= 640px) {
  .sc__grid {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
    gap: 10px;
  }
}
.sc__item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 6px 2px;
  border: none;
  border-radius: var(--app-radius);
  background: transparent;
  color: var(--el-text-color-primary);
  cursor: pointer;
  transition: background 0.15s;
}
@media (width >= 640px) {
  .sc__item {
    flex-direction: row;
    gap: 12px;
    padding: 12px;
    border: 1px solid var(--el-border-color-light);
    border-radius: 18px;
  }
}
.sc__item:hover {
  background: var(--el-fill-color-light);
}
.sc__icon {
  width: 46px;
  height: 46px;
  border-radius: 15px;
}
.sc__icon :deep(svg) {
  width: 22px;
  height: 22px;
}
.sc__label {
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 100%;
}
@media (width >= 640px) {
  .sc__label {
    font-size: 0.875rem;
  }
}
</style>
