<!-- 顶栏（AppsPC / DashboardPC）：宽搜索框 + 下载 / 终端 / 深浅色 / 主题 / 通知（工作台额外有「新建」）。
     「下载」只在服务器有下载记录（系统更新、Agent 安装包等）时出现，可暂停、继续、取消。
     「终端」打开悬浮终端窗口（本机终端 / SSH），角标是开着的终端数；开着时再点是最小化 / 还原。
     手机端：菜单按钮（打开侧栏抽屉，见 AppShell）+ 搜索 + 通知，深浅色与账号在抽屉里。 -->
<template>
  <header v-if="menuStore.isMobile" class="app-mbar">
    <AppIconButton icon="HOutline:Bars3Icon" :label="t('layout.openMenu')" :box="40" @click="menuStore.toggleMobileMenu()" />
    <CommandSearch class="app-mbar__search" />
    <NotificationMenu />
  </header>
  <header v-else class="app-topbar">
    <CommandSearch class="app-topbar__search" />
    <span class="app-topbar__spacer" />
    <DownloadMenu />
    <AppIconButton
      v-if="canTerm"
      icon="HOutline:CommandLineIcon"
      :label="t('termDock.title')"
      :box="44"
      :active="dock.open && !dock.minimized"
      :badge="dock.tabs.length || undefined"
      @click="toggleDock()"
    />
    <AppIconButton
      :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
      :label="t('theme.toggleDark')"
      :box="44"
      @click="themeStore.toggleDarkQuick()"
    />
    <ThemeQuickMenu />
    <NotificationMenu />
    <DashboardCreateMenu v-if="onDashboard" />
  </header>
</template>

<script setup lang="ts">
import CommandSearch from './CommandSearch.vue'
import NotificationMenu from './NotificationMenu.vue'
import ThemeQuickMenu from './ThemeQuickMenu.vue'
import DownloadMenu from './downloads/DownloadMenu.vue'
import DashboardCreateMenu from '@/views/dashboard/home/createMenu.vue'
import { dock, toggleDock } from './termDock/state'
import { SHELL_PERMISSION } from '@/api/shell'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const menuStore = useMenuStore()
const themeStore = useThemeStore()
const { t } = useI18n()

const onDashboard = computed(() => route.path === '/dashboard/home')
const canTerm = computed(
  () => menuStore.hasButtonPermission(SHELL_PERMISSION) || menuStore.hasButtonPermission('ssh:manage'),
)
</script>

<style scoped lang="scss">
.app-topbar {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: var(--cmp-topbar-height);
  flex-shrink: 0;
  padding: var(--cmp-page-padding-top) var(--cmp-page-padding-x) 0;
  background: var(--app-surface-page);
}
.app-topbar__search {
  flex: 1 1 320px;
  max-width: var(--cmp-search-width);
  margin-right: 8px;
}
.app-topbar__spacer {
  flex: 1;
}
.app-topbar :deep(.create-menu) {
  margin-left: 8px;
}
.app-mbar {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
  padding: calc(6px + env(safe-area-inset-top)) 8px 6px;
  background: var(--app-surface-page);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.app-mbar__search {
  flex: 1;
  min-width: 0;
  margin: 0 2px;
}
</style>
