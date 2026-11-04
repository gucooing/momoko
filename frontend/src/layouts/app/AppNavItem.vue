<!-- 手写导航项（递归）。目录(有可见子项)=分组标签+子项；菜单/无子目录=链接。
     激活态：primary 12% 底 + 主色文字 + 左侧 3px 主色短条（04 §2/§4）。 -->
<template>
  <template v-if="!isButton">
    <!-- 分组 -->
    <div v-if="isGroup" class="nav-group">
      <div v-if="!collapsed && depth === 0" class="nav-group__label" :class="{ 'is-active': hasActiveChild }">
        {{ label }}
      </div>
      <div v-else-if="collapsed && depth === 0" class="nav-group__divider" />
      <div v-else-if="depth > 0" class="nav-group__sublabel">{{ label }}</div>
      <AppNavItem
        v-for="child in visibleChildren"
        :key="child.id"
        :item="child"
        :depth="depth + 1"
        :collapsed="collapsed"
      />
    </div>

    <!-- 分组页的标签展开成多个入口（如「文件」：文件管理 / 分享 / 文件源），同一路由按 ?tab= 切换 -->
    <template v-else-if="tabLinks.length">
      <button
        v-for="link in tabLinks"
        :key="link.key"
        type="button"
        class="nav-link"
        :class="{ 'is-active': activeTabKey === link.key, 'is-collapsed': collapsed }"
        :aria-current="activeTabKey === link.key ? 'page' : undefined"
        :title="collapsed ? link.label : undefined"
        @click="navigateTo(link.to)"
      >
        <span class="nav-link__icon">
          <component :is="menuStore.iconComponents[link.icon]" v-if="link.icon" />
          <span v-else class="nav-link__dot" />
        </span>
        <span v-show="!collapsed" class="nav-link__label">{{ link.label }}</span>
      </button>
    </template>
    <!-- 链接 -->
    <button
      v-else
      type="button"
      class="nav-link"
      :class="{ 'is-active': isActive, 'is-collapsed': collapsed }"
      :aria-current="isActive ? 'page' : undefined"
      :title="collapsed ? label : undefined"
      @click="navigate"
    >
      <span class="nav-link__icon">
        <component :is="menuStore.iconComponents[item.icon]" v-if="item.icon" />
        <span v-else class="nav-link__dot" />
      </span>
      <span v-show="!collapsed" class="nav-link__label">{{ label }}</span>
    </button>
  </template>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { MenuStatus, MenuType, type MenuInfo } from '@/types/v1/system'
import { translateKnownText } from '@/locales'
import { PAGE_GROUPS, pageTabTo, type PageGroup, type PageGroupKey } from '@/config/pageGroups'

const props = withDefaults(
  defineProps<{ item: MenuInfo; depth?: number; collapsed?: boolean }>(),
  { depth: 0, collapsed: false },
)

const menuStore = useMenuStore()
const nodeStore = useNodeStore()
const route = useRoute()
const router = useRouter()
const { t } = useI18n()

const isButton = computed(() => props.item.type === MenuType.MenuType_Button)
const visibleChildren = computed(
  () =>
    props.item.children?.filter(
      (c) => c.type !== MenuType.MenuType_Button && c.status === MenuStatus.MenuStatus_Active,
    ) ?? [],
)
const isGroup = computed(
  () => props.item.type === MenuType.MenuType_Directory && visibleChildren.value.length > 0,
)
const label = computed(() => translateKnownText(props.item.title))
const isActive = computed(() => !!props.item.path && route.path === props.item.path)

const pathHasActive = (node: MenuInfo): boolean =>
  (!!node.path && route.path === node.path) || (node.children ?? []).some(pathHasActive)
const hasActiveChild = computed(() => visibleChildren.value.some(pathHasActive))

const navigate = () => {
  if (props.item.path && route.path !== props.item.path) router.push(props.item.path)
  if (menuStore.isMobileMenuOpen) menuStore.isMobileMenuOpen = false
}

// 侧边栏展开标签的分组页（PageGroup.sidebarTabs）：每个可见标签一个入口
const tabGroupKey = computed(() =>
  props.item.type === MenuType.MenuType_Menu
    ? (Object.keys(PAGE_GROUPS) as PageGroupKey[]).find((key) => {
        const def: PageGroup = PAGE_GROUPS[key]
        return def.sidebarTabs && def.path === props.item.path
      })
    : undefined,
)
const tabLinks = computed(() => {
  const key = tabGroupKey.value
  if (!key) return []
  const def: PageGroup = PAGE_GROUPS[key]
  return def.tabs
    .filter(
      (tab) =>
        (!tab.perm || menuStore.hasButtonPermission(tab.perm)) &&
        nodeStore.canAccessRoute(tab.nodePath || `${def.path}/${tab.key}`),
    )
    .map((tab) => ({ key: tab.key, label: t(tab.labelKey), icon: tab.icon || props.item.icon, to: pageTabTo(key, tab.key) }))
})
// 当前标签：地址上的 ?tab=，无效时回落到第一个可见标签（与 PageGroupHost 一致）
const activeTabKey = computed(() => {
  if (!tabLinks.value.length || route.path !== props.item.path) return ''
  return tabLinks.value.find((link) => link.key === route.query.tab)?.key || tabLinks.value[0]?.key || ''
})
const navigateTo = (to: RouteLocationRaw) => {
  void router.push(to)
  if (menuStore.isMobileMenuOpen) menuStore.isMobileMenuOpen = false
}
</script>

<style scoped lang="scss">
.nav-group__label {
  padding: 0 12px;
  margin: 16px 0 4px;
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  text-transform: uppercase;
  color: var(--el-text-color-secondary);
  transition: color 0.15s;
}
.nav-group:first-child .nav-group__label {
  margin-top: 4px;
}
.nav-group__label.is-active {
  color: var(--m-pInk);
}
.nav-group__sublabel {
  padding: 6px 14px 2px 46px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.nav-group__divider {
  height: 1px;
  margin: 10px 16px;
  background: var(--el-border-color-lighter);
}

.nav-link {
  position: relative;
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 42px;
  padding: 0 12px;
  margin: 2px 0;
  border: none;
  border-radius: 14px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 500;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.nav-link:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.nav-link.is-active {
  background: var(--m-pSoft);
  color: var(--m-pInk);
  font-weight: 700;
}
.nav-link__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}
.nav-link__icon :deep(svg) {
  width: 20px;
  height: 20px;
}
.nav-link.is-active .nav-link__icon :deep(svg) {
  stroke-width: 2;
}
.nav-link__dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  background: currentColor;
  opacity: 0.5;
  margin: auto;
}
.nav-link__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nav-link.is-collapsed {
  justify-content: center;
  padding: 0;
  width: 48px;
  margin: 2px auto;
}
.nav-link:focus-visible {
  outline: none;
  box-shadow: var(--app-focus-ring);
}
</style>
