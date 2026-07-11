<!-- 顶栏：高 60，底部 hairline，不投影（04 §3）。左:折叠/汉堡 + 标题；右:搜索/主题/语言/全屏/通知/用户。 -->
<template>
  <header class="app-topbar">
    <div class="app-topbar__left">
      <AppIconButton
        :icon="menuStore.isMobile ? 'HOutline:Bars3Icon' : collapseIcon"
        :label="menuStore.isMobile ? '菜单' : '折叠侧栏'"
        @click="onToggle"
      />
      <h1 class="app-topbar__title">{{ pageTitle }}</h1>
    </div>

    <div class="app-topbar__right">
      <!-- 桌面专属动作：包在普通 div 里，用作用域样式可靠隐藏（子组件多根无法透传 class） -->
      <div class="app-topbar__desktop">
        <CommandSearch />
        <AppIconButton icon="HOutline:Cog6ToothIcon" label="主题设置" @click="openThemeConfig" />
        <LanguageMenu />
        <AppIconButton
          :icon="isFullscreen ? 'HOutline:ArrowsPointingInIcon' : 'HOutline:ArrowsPointingOutIcon'"
          label="全屏"
          @click="toggleFullscreen"
        />
        <span class="app-topbar__divider" />
      </div>
      <NotificationMenu />
    </div>
  </header>
</template>

<script setup lang="ts">
import CommandSearch from './CommandSearch.vue'
import LanguageMenu from './LanguageMenu.vue'
import NotificationMenu from './NotificationMenu.vue'
import { translateKnownText } from '@/locales'
import { useFullscreen } from '@vueuse/core'

const route = useRoute()
const menuStore = useMenuStore()
const themeStore = useThemeStore()
const { isFullscreen, toggle: toggleFullscreen } = useFullscreen()

const pageTitle = computed(() => translateKnownText((route.meta?.title as string) || ''))
const collapseIcon = computed(() =>
  menuStore.isCollapse ? 'HOutline:Bars3BottomRightIcon' : 'HOutline:Bars3BottomLeftIcon',
)

const onToggle = () => {
  if (menuStore.isMobile) menuStore.toggleMobileMenu()
  else menuStore.toggleCollapse()
}
const openThemeConfig = () => {
  themeStore.themeConfigDrawerOpen = true
}
</script>

<style scoped lang="scss">
.app-topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  height: 60px;
  flex-shrink: 0;
  padding: 0 16px 0 10px;
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.app-topbar__left {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.app-topbar__title {
  font-size: 1.05rem;
  font-weight: 600;
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
