<!-- 外壳组装：侧栏 + 主列（顶栏 + 标签 + 内容）。手机端(<992)侧栏收成抽屉：顶栏左侧的菜单按钮从左边滑出，
     抽屉里就是电脑端的侧栏（节点、完整菜单、账号），两端找功能的路径一样。切换页面、按 Esc、点遮罩时收起。
     悬浮终端（TermDock）挂在这里：切换页面不卸载，连接不断。 -->
<template>
  <div class="app-shell">
    <AppSidebar v-if="!menuStore.isMobile" :collapsed="menuStore.isCollapse" />

    <div class="app-shell__main">
      <AppTopbar />
      <AppTabs />
      <AppContent />
    </div>

    <Transition name="app-drawer">
      <div v-if="menuStore.isMobile && menuStore.isMobileMenuOpen" class="app-drawer" @click.self="closeDrawer">
        <AppSidebar drawer />
      </div>
    </Transition>

    <TermDock v-if="dock.open" />
  </div>
</template>

<script setup lang="ts">
import { useEventListener } from '@vueuse/core'
import AppSidebar from './AppSidebar.vue'
import AppTopbar from './AppTopbar.vue'
import AppTabs from './AppTabs.vue'
import AppContent from './AppContent.vue'
import TermDock from './termDock/TermDock.vue'
import { dock } from './termDock/state'

defineOptions({ name: 'AppShell' })

const route = useRoute()
const menuStore = useMenuStore()
const userStore = useUserStore()

const closeDrawer = () => {
  menuStore.isMobileMenuOpen = false
}
watch(() => route.fullPath, closeDrawer)
watch(
  () => menuStore.isMobile,
  (mobile) => {
    if (!mobile) closeDrawer()
  },
)
useEventListener('keydown', (e: KeyboardEvent) => {
  if (e.key === 'Escape' && menuStore.isMobileMenuOpen) closeDrawer()
})

onMounted(() => {
  if (!userStore.userInfo) void userStore.getUserInfo()
})
</script>

<style scoped lang="scss">
.app-shell {
  display: flex;
  width: 100%;
  height: 100vh;
  height: 100dvh;
  overflow: hidden;
  background: var(--app-surface-page);
}
.app-shell__main {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  overflow: hidden;
}
.app-drawer {
  position: fixed;
  inset: 0;
  z-index: 2000;
  display: flex;
  background: color-mix(in srgb, #0b1220 45%, transparent);
}
.app-drawer-enter-active,
.app-drawer-leave-active {
  transition: opacity 0.2s ease;
}
.app-drawer-enter-active :deep(.app-sidebar),
.app-drawer-leave-active :deep(.app-sidebar) {
  transition: transform 0.22s cubic-bezier(0.4, 0, 0.2, 1);
}
.app-drawer-enter-from,
.app-drawer-leave-to {
  opacity: 0;
}
.app-drawer-enter-from :deep(.app-sidebar),
.app-drawer-leave-to :deep(.app-sidebar) {
  transform: translateX(-100%);
}
</style>
