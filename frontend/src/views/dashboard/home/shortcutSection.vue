<!-- 快捷入口：安静的单色图标网格，非彩色大块（01 §8 / 06a）。 -->
<template>
  <AppPanel :title="t('dashboard.home.convenientTools')" title-icon="HOutline:Squares2X2Icon">
    <div class="sc__grid">
      <button
        v-for="item in shortcuts"
        :key="item.labelKey"
        type="button"
        class="sc__item"
        @click="router.push(item.routePath)"
      >
        <span class="sc__icon"><component :is="menuStore.iconComponents[item.icon]" /></span>
        <span class="sc__label">{{ t(item.labelKey) }}</span>
      </button>
    </div>
  </AppPanel>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const menuStore = useMenuStore()
const dashboardHomeStore = useDashboardHomeStore()
const { t } = useI18n()
const { shortcuts } = storeToRefs(dashboardHomeStore)
</script>

<style scoped lang="scss">
.sc__grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
@media (width >= 420px) {
  .sc__grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
.sc__item {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  padding: 16px 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
  cursor: pointer;
  transition: border-color 0.15s, transform 0.15s, background 0.15s;
}
.sc__item:hover {
  transform: translateY(-1px);
  border-color: color-mix(in srgb, var(--el-color-primary) 35%, var(--el-border-color));
  background: var(--el-fill-color-lighter);
}
.sc__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 38px;
  height: 38px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  transition: color 0.15s, background 0.15s;
}
.sc__item:hover .sc__icon {
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
}
.sc__icon :deep(svg) {
  width: 20px;
  height: 20px;
}
.sc__label {
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--el-text-color-regular);
  text-align: center;
}
</style>
