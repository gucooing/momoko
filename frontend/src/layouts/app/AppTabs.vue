<!-- 极简下划线标签（复用 tabsStore 逻辑，重写视觉）。仅 PC、且多于一个标签时显示（04 §5）。 -->
<template>
  <div v-if="!menuStore.isMobile && themeStore.showTabs && tabsStore.tabs.length > 1" class="app-tabs">
    <div class="app-tabs__scroll">
      <button
        v-for="tab in tabsStore.tabs"
        :key="tab.fullPath"
        type="button"
        class="app-tab"
        :class="{ 'is-active': tab.fullPath === tabsStore.activePath }"
        @click="go(tab)"
      >
        <span class="app-tab__label">{{ tabLabel(tab) }}</span>
        <span
          v-if="tab.closable"
          class="app-tab__close"
          role="button"
          @click.stop="closeTab(tab)"
        >
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
        </span>
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { translateKnownText } from '@/locales'

interface TabLike {
  fullPath: string
  title: string
  closable: boolean
}

const router = useRouter()
const menuStore = useMenuStore()
const tabsStore = useTabsStore()
const themeStore = useThemeStore()

const tabLabel = (tab: TabLike) => translateKnownText(tab.title)
const go = (tab: TabLike) => {
  if (tab.fullPath !== tabsStore.activePath) router.push(tab.fullPath)
}
const closeTab = (tab: TabLike) => {
  tabsStore.removeTab(tab.fullPath)
  if (tabsStore.activePath && router.currentRoute.value.fullPath !== tabsStore.activePath) {
    router.push(tabsStore.activePath)
  }
}
</script>

<style scoped lang="scss">
.app-tabs {
  display: flex;
  align-items: center;
  height: 44px;
  flex-shrink: 0;
  margin: 12px 32px 0;
  background: transparent;
  border-bottom: 1px solid var(--el-border-color-light);
  overflow: hidden;
}
.app-tabs__scroll {
  display: flex;
  align-items: stretch;
  gap: 2px;
  height: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.app-tabs__scroll::-webkit-scrollbar {
  display: none;
}
.app-tab {
  position: relative;
  display: flex;
  align-items: center;
  gap: 6px;
  height: 100%;
  padding: 0 12px;
  border: none;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  white-space: nowrap;
  transition: color 0.15s;
}
.app-tab:hover {
  color: var(--el-text-color-primary);
}
.app-tab.is-active {
  color: var(--el-text-color-primary);
}
.app-tab.is-active::after {
  content: '';
  position: absolute;
  left: 10px;
  right: 10px;
  bottom: 0;
  height: 3px;
  border-radius: 3px 3px 0 0;
  background: var(--m-p);
}
.app-tab__close {
  display: flex;
  align-items: center;
  border-radius: 4px;
  opacity: 0;
  transition: opacity 0.15s, background 0.15s;
}
.app-tab:hover .app-tab__close,
.app-tab.is-active .app-tab__close {
  opacity: 0.55;
}
.app-tab__close:hover {
  opacity: 1;
  background: var(--el-fill-color);
}
.app-tab__close :deep(svg) {
  width: 13px;
  height: 13px;
}
</style>
