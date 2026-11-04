<!-- 顶栏（AppsPC / DashboardPC）：宽搜索框 + 深浅色 / 主题 / 通知（工作台额外有「新建」）。
     手机端没有全局顶栏：页面标题行（PageHeader）自带返回与深浅色，工作台自带问候页头，「更多」页带通知与搜索。 -->
<template>
  <header v-if="!menuStore.isMobile" class="app-topbar">
    <CommandSearch class="app-topbar__search" />
    <span class="app-topbar__spacer" />
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
import DashboardCreateMenu from '@/views/dashboard/home/createMenu.vue'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const menuStore = useMenuStore()
const themeStore = useThemeStore()
const { t } = useI18n()

const onDashboard = computed(() => route.path === '/dashboard/home')
</script>

<style scoped lang="scss">
.app-topbar {
  display: flex;
  align-items: center;
  gap: 4px;
  min-height: 68px;
  flex-shrink: 0;
  padding: 24px 32px 0;
  background: var(--el-bg-color-page);
}
.app-topbar__search {
  flex: 1 1 320px;
  max-width: 520px;
  margin-right: 8px;
}
.app-topbar__spacer {
  flex: 1;
}
.app-topbar :deep(.create-menu) {
  margin-left: 8px;
}
</style>
