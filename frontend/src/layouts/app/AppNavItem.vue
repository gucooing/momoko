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

    <!-- 链接 -->
    <button
      v-else
      type="button"
      class="nav-link"
      :class="{ 'is-active': isActive, 'is-collapsed': collapsed }"
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
import { MenuStatus, MenuType, type MenuInfo } from '@/types/v1/system'
import { translateKnownText } from '@/locales'

const props = withDefaults(
  defineProps<{ item: MenuInfo; depth?: number; collapsed?: boolean }>(),
  { depth: 0, collapsed: false },
)

const menuStore = useMenuStore()
const route = useRoute()
const router = useRouter()

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
</script>

<style scoped lang="scss">
.nav-group__label {
  padding: 0 12px;
  margin: 14px 0 4px;
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--el-text-color-placeholder);
  transition: color 0.15s;
}
.nav-group:first-child .nav-group__label {
  margin-top: 4px;
}
.nav-group__label.is-active {
  color: var(--el-color-primary);
}
.nav-group__sublabel {
  padding: 6px 12px 2px 34px;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--el-text-color-secondary);
}
.nav-group__divider {
  height: 1px;
  margin: 10px 14px;
  background: var(--el-border-color-lighter);
}

.nav-link {
  position: relative;
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 12px;
  margin: 1px 0;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-regular);
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
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
  color: var(--el-color-primary);
}
.nav-link.is-active::before {
  content: '';
  position: absolute;
  left: 0;
  top: 50%;
  transform: translateY(-50%);
  width: 3px;
  height: 18px;
  border-radius: 0 3px 3px 0;
  background: var(--el-color-primary);
}
.nav-link__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 18px;
  height: 18px;
  flex-shrink: 0;
}
.nav-link__icon :deep(svg) {
  width: 18px;
  height: 18px;
}
.nav-link__dot {
  width: 5px;
  height: 5px;
  border-radius: 999px;
  background: currentColor;
  opacity: 0.45;
  margin: auto;
}
.nav-link__label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.nav-link.is-collapsed {
  justify-content: center;
  padding: 8px;
}
</style>
