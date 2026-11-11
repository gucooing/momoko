<!-- 手机底部导航（AppsMobile）：首页 / 应用 / 容器 / 文件（按当前用户可访问的菜单出现）+ 更多（打开「更多」页）。
     工具、管理、个人中心等其它页面高亮「更多」。 -->
<template>
  <nav class="tabbar" :aria-label="t('shell.nav.bottom')">
    <button
      v-for="tab in tabs"
      :key="tab.path"
      type="button"
      class="tabbar__item"
      :class="{ 'is-on': !menuStore.isMobileMenuOpen && isActive(tab) }"
      :aria-current="!menuStore.isMobileMenuOpen && isActive(tab) ? 'page' : undefined"
      @click="go(tab.path)"
    >
      <span class="tabbar__icon"><component :is="ic(tab.icon)" /></span>
      <span class="tabbar__label">{{ t(tab.label) }}</span>
    </button>
    <button
      type="button"
      class="tabbar__item"
      :class="{ 'is-on': moreActive }"
      :aria-expanded="menuStore.isMobileMenuOpen"
      @click="more.toggle()"
    >
      <span class="tabbar__icon">
        <component :is="ic('HOutline:Squares2X2Icon')" />
        <span v-if="runningTasks > 0" class="tabbar__dot" aria-hidden="true" />
      </span>
      <span class="tabbar__label">{{ t('layout.tabbar.more') }}</span>
    </button>
  </nav>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import { TABBAR_ITEMS, useMorePanel, type TabbarItem } from './tabbar'
import { useRunningTaskCount } from './useRunningTasks'

const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()
const { t } = useI18n()
const more = useMorePanel()
const ic = (name: string) => menuStore.iconComponents[name]
const runningTasks = useRunningTaskCount()

const accessiblePaths = useAccessibleRoutes()

const tabs = computed(() => TABBAR_ITEMS.filter((c) => accessiblePaths.value.has(c.path)))

const isActive = (tab: TabbarItem) =>
  route.path === tab.path ||
  route.path.startsWith(`${tab.path}/`) ||
  !!tab.also?.some((prefix) => route.path.startsWith(prefix))
const moreActive = computed(() => menuStore.isMobileMenuOpen || !tabs.value.some((tab) => isActive(tab)))

const go = (path: string) => {
  more.close()
  if (route.path !== path) router.push(path)
}
</script>

<style scoped lang="scss">
.tabbar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2050;
  display: flex;
  padding: 6px 8px calc(8px + env(safe-area-inset-bottom));
  background: var(--app-sidebar-bg);
  border-top: 1px solid var(--el-border-color-light);
}
.tabbar__item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  min-height: var(--cmp-tabbar-height);
  padding-top: 6px;
  border: none;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.6875rem;
  font-weight: 600;
  cursor: pointer;
}
.tabbar__item:hover {
  color: var(--el-text-color-primary);
}
.tabbar__icon {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 30px;
  border-radius: 999px;
  transition: background 0.18s, color 0.18s;
}
.tabbar__icon :deep(svg) {
  width: var(--cmp-tabbar-icon-size);
  height: var(--cmp-tabbar-icon-size);
}
.tabbar__dot {
  position: absolute;
  top: 3px;
  right: 12px;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--m-p);
  border: 2px solid var(--el-bg-color);
}
.tabbar__item.is-on {
  color: var(--m-pInk);
}
.tabbar__item.is-on .tabbar__icon {
  background: var(--m-pSoft);
}
</style>
