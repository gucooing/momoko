<!-- 侧边栏（AppsPC / DashboardPC）：品牌 → 节点卡片（本机/远程 · 在线 · 主机名）→ 分组导航 → 底部账号卡。
     展开 252 / 折叠 76，右侧 hairline，不投影。
     drawer：手机端的抽屉（AppShell 里从左边滑出），内容不变；品牌行右侧换成深浅色切换和关闭。 -->
<template>
  <aside
    class="app-sidebar"
    :class="{ 'app-sidebar--collapsed': collapsed, 'app-sidebar--drawer': drawer }"
    :aria-label="drawer ? t('shell.nav.main') : undefined"
  >
    <div class="app-sidebar__brand">
      <img :src="logoSrc" alt="" class="app-sidebar__logo" />
      <span v-show="!collapsed" class="app-sidebar__name">{{ appName }}</span>
      <template v-if="drawer">
        <AppIconButton
          :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
          :label="t('theme.toggleDark')"
          :box="40"
          :size="18"
          @click="themeStore.toggleDarkQuick()"
        />
        <AppIconButton
          icon="HOutline:XMarkIcon"
          :label="t('common.close')"
          :box="40"
          :size="18"
          @click="menuStore.isMobileMenuOpen = false"
        />
      </template>
      <AppIconButton
        v-else-if="!collapsed"
        class="app-sidebar__fold"
        :icon="'HOutline:ChevronDoubleLeftIcon'"
        :label="t('layout.collapseSidebar')"
        :box="32"
        :size="16"
        @click="menuStore.toggleCollapse()"
      />
    </div>
    <AppIconButton
      v-if="collapsed"
      class="app-sidebar__unfold"
      icon="HOutline:ChevronDoubleRightIcon"
      :label="t('layout.expandSidebar')"
      :box="36"
      :size="16"
      @click="menuStore.toggleCollapse()"
    />

    <div class="app-sidebar__node">
      <NodeSwitcher :collapsed="collapsed" />
    </div>

    <div class="app-sidebar__nav">
      <AppNav :collapsed="collapsed" />
    </div>

    <div class="app-sidebar__foot">
      <UserMenu :collapsed="collapsed" />
    </div>
  </aside>
</template>

<script setup lang="ts">
import AppNav from './AppNav.vue'
import NodeSwitcher from './NodeSwitcher.vue'
import UserMenu from './UserMenu.vue'
import { APP_CONFIG } from '@/config/app.config'
import { useI18n } from 'vue-i18n'

defineProps<{ collapsed?: boolean; drawer?: boolean }>()
const menuStore = useMenuStore()
const themeStore = useThemeStore()
const { t } = useI18n()
const appName = APP_CONFIG.name
const logoSrc = APP_CONFIG.logoSrc
</script>

<style scoped lang="scss">
.app-sidebar {
  display: flex;
  flex-direction: column;
  gap: 14px;
  width: var(--cmp-sidebar-width);
  height: 100%;
  padding: 20px 16px 16px;
  background: var(--app-sidebar-bg);
  border-right: 1px solid var(--el-border-color-light);
  transition: width 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
.app-sidebar--collapsed {
  width: 76px;
  padding: 20px 10px 14px;
}
.app-sidebar--drawer {
  width: min(300px, 86vw);
  gap: 12px;
  padding: calc(12px + env(safe-area-inset-top)) 12px calc(12px + env(safe-area-inset-bottom));
  background: var(--el-bg-color);
  border-right: 0;
  box-shadow: var(--app-shadow-lg);
}
.app-sidebar--drawer .app-sidebar__brand {
  gap: 4px;
  min-height: 40px;
}
.app-sidebar--drawer .app-sidebar__logo {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  margin-right: 6px;
}
.app-sidebar--drawer .app-sidebar__name {
  font-size: 1.125rem;
}
.app-sidebar--drawer .app-sidebar__nav {
  margin: 0 -12px;
}
.app-sidebar__brand {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 44px;
  padding: 0 0 0 8px;
  flex-shrink: 0;
}
.app-sidebar--collapsed .app-sidebar__brand {
  justify-content: center;
  padding: 0;
}
.app-sidebar__logo {
  width: 40px;
  height: 40px;
  border-radius: 13px;
  flex-shrink: 0;
  object-fit: cover;
  transform: rotate(-6deg);
}
.app-sidebar__name {
  flex: 1;
  min-width: 0;
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
/* 折叠按钮平时淡出，悬停侧栏或键盘聚焦时出现，不抢品牌区视觉 */
.app-sidebar__fold {
  flex: none;
  opacity: 0;
  transition: opacity 0.15s;
}
.app-sidebar:hover .app-sidebar__fold,
.app-sidebar__fold:focus-visible {
  opacity: 1;
}
@media (pointer: coarse) {
  .app-sidebar__fold {
    opacity: 1;
  }
}
.app-sidebar__unfold {
  align-self: center;
  margin-top: -6px;
}
.app-sidebar__node {
  display: flex;
  flex-shrink: 0;
}
.app-sidebar__nav {
  flex: 1;
  min-height: 0;
  margin: 0 -16px;
  overflow-y: auto;
  overflow-x: hidden;
  scrollbar-width: none;
}
.app-sidebar--collapsed .app-sidebar__nav {
  margin: 0 -10px;
}
.app-sidebar__nav::-webkit-scrollbar {
  display: none;
}
.app-sidebar__foot {
  flex-shrink: 0;
}
</style>
