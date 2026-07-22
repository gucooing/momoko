<!-- 侧边栏：品牌 + 导航 + 底部用户/折叠。展开 248 / 折叠 72，右侧 hairline，整体不投影（04 §2）。 -->
<template>
  <aside
    class="app-sidebar"
    :class="{ 'app-sidebar--collapsed': collapsed, 'app-sidebar--mobile': mobile }"
  >
    <div class="app-sidebar__brand">
      <img :src="logoSrc" alt="logo" class="app-sidebar__logo" />
      <span v-show="!collapsed" class="app-sidebar__name">{{ appName }}</span>
    </div>

    <div class="app-sidebar__nav">
      <AppNav :collapsed="collapsed" />
    </div>

    <div class="app-sidebar__foot" :class="{ 'is-collapsed': collapsed }">
      <UserMenu variant="sidebar" :collapsed="collapsed" />
      <AppIconButton
        v-if="!mobile"
        :icon="collapsed ? 'HOutline:ChevronDoubleRightIcon' : 'HOutline:ChevronDoubleLeftIcon'"
        :label="sidebarToggleLabel"
        @click="menuStore.toggleCollapse()"
      />
    </div>
  </aside>
</template>

<script setup lang="ts">
import AppNav from './AppNav.vue'
import UserMenu from './UserMenu.vue'
import { APP_CONFIG } from '@/config/app.config'
import { useI18n } from 'vue-i18n'

const props = defineProps<{ collapsed?: boolean; mobile?: boolean }>()
const menuStore = useMenuStore()
const { t } = useI18n()
const appName = APP_CONFIG.name
const logoSrc = APP_CONFIG.logoSrc
const sidebarToggleLabel = computed(() =>
  props.collapsed ? t('layout.expandSidebar') : t('layout.collapseSidebar'),
)
</script>

<style scoped lang="scss">
.app-sidebar {
  display: flex;
  flex-direction: column;
  width: 248px;
  height: 100%;
  background: var(--el-bg-color);
  border-right: 1px solid var(--el-border-color-light);
  transition: width 0.2s cubic-bezier(0.4, 0, 0.2, 1);
}
.app-sidebar--collapsed {
  width: 72px;
}
.app-sidebar--mobile {
  width: 100%;
  border-right: none;
}
.app-sidebar__brand {
  display: flex;
  align-items: center;
  gap: 10px;
  height: 60px;
  padding: 0 18px;
  flex-shrink: 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.app-sidebar--collapsed .app-sidebar__brand {
  justify-content: center;
  padding: 0;
}
.app-sidebar__logo {
  width: 30px;
  height: 30px;
  border-radius: var(--app-radius-sm);
  flex-shrink: 0;
}
.app-sidebar__name {
  font-size: 1.15rem;
  font-weight: 700;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}
.app-sidebar__nav {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  overflow-x: hidden;
}
.app-sidebar__foot {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.app-sidebar__foot.is-collapsed {
  flex-direction: column;
}
</style>
