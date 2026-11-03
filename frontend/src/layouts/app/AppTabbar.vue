<!-- 手机底部导航：首页 / 应用 / 容器 / 文件（按当前用户可访问的菜单出现）+ 更多（打开完整菜单）。 -->
<template>
  <nav class="tabbar" :aria-label="t('layout.openMenu')">
    <button
      v-for="tab in tabs"
      :key="tab.path"
      type="button"
      class="tabbar__item"
      :class="{ 'is-on': isActive(tab.path) }"
      :aria-current="isActive(tab.path) ? 'page' : undefined"
      @click="go(tab.path)"
    >
      <span class="tabbar__icon"><component :is="ic(tab.icon)" /></span>
      <span class="tabbar__label">{{ t(tab.label) }}</span>
    </button>
    <button
      type="button"
      class="tabbar__item"
      :class="{ 'is-on': moreActive }"
      :aria-expanded="menuStore.isMobileMenuOpen"
      @click="menuStore.toggleMobileMenu()"
    >
      <span class="tabbar__icon"><component :is="ic('HOutline:Squares2X2Icon')" /></span>
      <span class="tabbar__label">{{ t('layout.tabbar.more') }}</span>
    </button>
  </nav>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'

const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()
const { t } = useI18n()
const ic = (name: string) => menuStore.iconComponents[name]

const CANDIDATES = [
  { path: '/dashboard/home', label: 'layout.tabbar.home', icon: 'HOutline:HomeIcon' },
  { path: '/instance/list', label: 'layout.tabbar.apps', icon: 'HOutline:CubeIcon' },
  { path: '/docker', label: 'layout.tabbar.containers', icon: 'HOutline:ServerStackIcon' },
  { path: '/file/index', label: 'layout.tabbar.files', icon: 'HOutline:FolderIcon' },
]

const accessiblePaths = useAccessibleRoutes()

const tabs = computed(() => CANDIDATES.filter((c) => accessiblePaths.value.has(c.path)))

const isActive = (path: string) => route.path === path || route.path.startsWith(`${path}/`)
const moreActive = computed(
  () => menuStore.isMobileMenuOpen || !tabs.value.some((tab) => isActive(tab.path)),
)

const go = (path: string) => {
  menuStore.isMobileMenuOpen = false
  if (route.path !== path) router.push(path)
}
</script>

<style scoped lang="scss">
.tabbar {
  position: fixed;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 2050;
  display: flex;
  padding: 6px 8px calc(8px + env(safe-area-inset-bottom));
  background: color-mix(in srgb, var(--el-bg-color) 92%, transparent);
  backdrop-filter: blur(12px);
  border-top: 1px solid var(--el-border-color-light);
}
.tabbar__item {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 3px;
  min-height: 52px;
  padding-top: 4px;
  border: none;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.6875rem;
  font-weight: 700;
  cursor: pointer;
}
.tabbar__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 54px;
  height: 30px;
  border-radius: 999px;
  transition: background 0.18s, color 0.18s;
}
.tabbar__icon :deep(svg) {
  width: 22px;
  height: 22px;
}
.tabbar__item.is-on {
  color: var(--m-pInk);
}
.tabbar__item.is-on .tabbar__icon {
  background: var(--m-pSoft);
}
</style>
