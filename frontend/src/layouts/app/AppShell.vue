<!-- 外壳组装：侧栏 + 主列（顶栏 + 标签 + 内容）。移动端(<lg)侧栏改为抽屉承载。 -->
<template>
  <div class="app-shell">
    <AppSidebar v-if="!menuStore.isMobile" :collapsed="menuStore.isCollapse" />

    <div class="app-shell__main">
      <AppTopbar />
      <AppTabs />
      <AppContent />
    </div>

    <el-drawer
      v-if="menuStore.isMobile"
      v-model="menuStore.isMobileMenuOpen"
      direction="ltr"
      :with-header="false"
      :size="264"
      class="app-shell__drawer"
    >
      <AppSidebar :collapsed="false" mobile />
    </el-drawer>
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

onMounted(() => {
  if (!userStore.userInfo) void userStore.getUserInfo()
})
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
</style>

<style lang="scss">
.app-shell__drawer .el-drawer__body {
  padding: 0;
  overflow: hidden;
}
</style>
