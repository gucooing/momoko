<!-- 外壳组装：侧栏 + 主列（顶栏 + 标签 + 内容）。移动端(<lg)侧栏改为令牌抽屉承载。 -->
<template>
  <div class="app-shell">
    <AppSidebar v-if="!menuStore.isMobile" :collapsed="menuStore.isCollapse" />

    <div class="app-shell__main">
      <AppTopbar />
      <AppTabs />
      <AppContent />
    </div>

    <Teleport to="body">
      <Transition name="shell-drawer">
        <div
          v-if="menuStore.isMobile && menuStore.isMobileMenuOpen"
          class="app-shell__drawer-overlay"
          @mousedown.self="menuStore.isMobileMenuOpen = false"
        >
          <aside class="app-shell__drawer" role="dialog" aria-modal="true">
            <AppSidebar :collapsed="false" mobile />
          </aside>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import AppSidebar from './AppSidebar.vue'
import AppTopbar from './AppTopbar.vue'
import AppTabs from './AppTabs.vue'
import AppContent from './AppContent.vue'

defineOptions({ name: 'AppShell' })

const menuStore = useMenuStore()
const userStore = useUserStore()

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && menuStore.isMobileMenuOpen) menuStore.isMobileMenuOpen = false
}

onMounted(() => {
  if (!userStore.userInfo) void userStore.getUserInfo()
  window.addEventListener('keydown', onKeydown)
})
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped lang="scss">
.app-shell {
  display: flex;
  width: 100%;
  height: 100vh;
  overflow: hidden;
  background: var(--el-bg-color-page);
}
.app-shell__main {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  overflow: hidden;
}
.app-shell__drawer-overlay {
  position: fixed;
  inset: 0;
  z-index: 2100;
  display: flex;
  background: color-mix(in srgb, #0b1220 42%, transparent);
  backdrop-filter: blur(2px);
}
.app-shell__drawer {
  width: 264px;
  height: 100%;
  overflow: hidden;
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-light);
  box-shadow: var(--app-shadow-lg);
}

.shell-drawer-enter-active,
.shell-drawer-leave-active {
  transition: opacity 0.18s ease;
}
.shell-drawer-enter-active .app-shell__drawer,
.shell-drawer-leave-active .app-shell__drawer {
  transition: transform 0.18s cubic-bezier(0.4, 0, 0.2, 1);
}
.shell-drawer-enter-from,
.shell-drawer-leave-to {
  opacity: 0;
}
.shell-drawer-enter-from .app-shell__drawer,
.shell-drawer-leave-to .app-shell__drawer {
  transform: translateX(-12px);
}
</style>
