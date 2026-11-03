<!-- 顶栏：高 64，半透明底 + hairline。左:折叠/汉堡 + 标题；右:节点/搜索/语言/全屏/深浅色/主题/通知。 -->
<template>
  <header class="app-topbar">
    <div class="app-topbar__left">
      <AppIconButton
        :icon="menuStore.isMobile ? 'HOutline:Bars3Icon' : collapseIcon"
        :label="menuToggleLabel"
        @click="onToggle"
      />
      <h1 class="app-topbar__title">{{ pageTitle }}</h1>
    </div>

    <div class="app-topbar__right">
      <NodeSwitcher />
      <!-- 桌面专属动作：包在普通 div 里，用作用域样式可靠隐藏（子组件多根无法透传 class） -->
      <div class="app-topbar__desktop">
        <CommandSearch />
        <LanguageMenu />
        <AppIconButton
          :icon="isFullscreen ? 'HOutline:ArrowsPointingInIcon' : 'HOutline:ArrowsPointingOutIcon'"
          :label="isFullscreen ? t('layout.exitFullscreen') : t('layout.fullscreen')"
          @click="toggleFullscreen"
        />
        <span class="app-topbar__divider" />
      </div>
      <AppIconButton
        :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
        :label="t('theme.toggleDark')"
        @click="themeStore.toggleDarkQuick()"
      />
      <ThemeQuickMenu />
      <NotificationMenu />
    </div>
  </header>
</template>

<script setup lang="ts">
import CommandSearch from './CommandSearch.vue'
import LanguageMenu from './LanguageMenu.vue'
import NodeSwitcher from './NodeSwitcher.vue'
import NotificationMenu from './NotificationMenu.vue'
import ThemeQuickMenu from './ThemeQuickMenu.vue'
import { translateKnownText } from '@/locales'
import { useFullscreen } from '@vueuse/core'
import { useI18n } from 'vue-i18n'

const route = useRoute()
const menuStore = useMenuStore()
const themeStore = useThemeStore()
const { t } = useI18n()
const { isFullscreen, toggle: toggleFullscreen } = useFullscreen()

const pageTitle = computed(() => translateKnownText((route.meta?.title as string) || ''))
const collapseIcon = computed(() =>
  menuStore.isCollapse ? 'HOutline:Bars3BottomRightIcon' : 'HOutline:Bars3BottomLeftIcon',
)
const menuToggleLabel = computed(() =>
  menuStore.isMobile
    ? t('layout.openMenu')
    : menuStore.isCollapse
      ? t('layout.expandSidebar')
      : t('layout.collapseSidebar'),
)

const onToggle = () => {
  if (menuStore.isMobile) menuStore.toggleMobileMenu()
  else menuStore.toggleCollapse()
}
</script>

<style scoped lang="scss">
.app-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  height: 64px;
  flex-shrink: 0;
  padding: 0 16px 0 12px;
  background: color-mix(in srgb, var(--el-bg-color-page) 88%, transparent);
  backdrop-filter: blur(10px);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.app-topbar__left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.app-topbar__title {
  font-size: 1.125rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.app-topbar__right {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-shrink: 0;
}
.app-topbar__desktop {
  display: flex;
  align-items: center;
  gap: 4px;
}
.app-topbar__divider {
  width: 1px;
  height: 22px;
  background: var(--el-border-color);
  margin: 0 6px;
}
@media (width <= 768px) {
  .app-topbar__desktop {
    display: none;
  }
}
</style>
