<!-- 外壳组装：侧栏 + 主列（顶栏 + 标签 + 内容）。手机端(<992)不显示侧栏，底部固定导航，「更多」整页面板。 -->
<template>
  <div class="app-shell">
    <AppSidebar v-if="!menuStore.isMobile" :collapsed="menuStore.isCollapse" />

    <div class="app-shell__main">
      <AppTopbar />
      <AppTabs />
      <AppContent />
    </div>

    <template v-if="menuStore.isMobile">
      <AppMorePanel />
      <AppTabbar v-if="!route.meta.hideTabbar" />
    </template>
  </div>
</template>

<script setup lang="ts">
import AppSidebar from './AppSidebar.vue'
import AppTopbar from './AppTopbar.vue'
import AppTabs from './AppTabs.vue'
import AppContent from './AppContent.vue'
import AppTabbar from './AppTabbar.vue'
import AppMorePanel from './AppMorePanel.vue'

defineOptions({ name: 'AppShell' })

const route = useRoute()
const menuStore = useMenuStore()
const userStore = useUserStore()

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
</style>
