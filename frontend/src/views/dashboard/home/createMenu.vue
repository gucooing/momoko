<!-- 工作台顶栏「新建 ▾」（DashboardPC）：只列当前用户在当前节点能进入的创建入口。 -->
<template>
  <!-- AppDropdown 渲染多根节点（触发器 + Teleport），class 透传不上去，外面包一层承接顶栏的间距样式 -->
  <div v-if="items.length" class="create-menu">
    <AppDropdown align="end" :width="232">
      <template #trigger="{ open }">
        <UButton color="primary" class="create-menu__btn" :aria-expanded="open">
          <component :is="ic('HOutline:PlusIcon')" class="create-menu__ic" />
          {{ t('dashboard.home.create') }}
          <component :is="ic('HOutline:ChevronDownIcon')" class="create-menu__caret" />
        </UButton>
      </template>
      <template #default="{ close }">
        <div class="create-menu__list">
          <button
            v-for="(it, i) in items"
            :key="it.path"
            type="button"
            class="create-menu__item"
            @click="go(it.to, close)"
          >
            <span class="create-menu__sk m-sticker" :class="`m-sticker--${TONES[i % TONES.length]}`">
              <component :is="ic(it.icon)" />
            </span>
            <span>{{ t(it.label) }}</span>
          </button>
        </div>
      </template>
    </AppDropdown>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { RouteLocationRaw } from 'vue-router'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import { pageTabTo } from '@/config/pageGroups'

const TONES = ['p', 'a2', 'a', 'ok', 'p', 'a2']
// path 判断能否进入（分组标签用 <分组路径>/<标签> 虚拟路径），to 是跳转地址
const CREATE_ITEMS: { path: string; to: RouteLocationRaw; label: string; icon: string }[] = [
  { path: '/instance/list', to: '/instance/list', label: 'dashboard.home.createApp', icon: 'HOutline:CubeIcon' },
  { path: '/docker', to: '/docker', label: 'dashboard.home.createContainer', icon: 'HOutline:ServerStackIcon' },
  { path: '/file/index', to: pageTabTo('file', 'index'), label: 'dashboard.home.uploadFile', icon: 'HOutline:ArrowUpTrayIcon' },
  { path: '/file/share', to: pageTabTo('file', 'share'), label: 'dashboard.home.createShare', icon: 'HOutline:ShareIcon' },
  { path: '/tools/port-forward', to: '/tools/port-forward', label: 'dashboard.home.createForward', icon: 'HOutline:ArrowsRightLeftIcon' },
  { path: '/tools/tunnel', to: '/tools/tunnel', label: 'dashboard.home.createTunnel', icon: 'HOutline:GlobeAltIcon' },
]

const router = useRouter()
const menuStore = useMenuStore()
const nodeStore = useNodeStore()
const accessible = useAccessibleRoutes()
const { t } = useI18n()
const ic = (name: string) => menuStore.iconComponents[name]

const items = computed(() =>
  CREATE_ITEMS.filter((it) => accessible.value.has(it.path) && nodeStore.canAccessRoute(it.path)),
)

const go = (to: RouteLocationRaw, close: () => void) => {
  close()
  void router.push(to)
}
</script>

<style scoped lang="scss">
.create-menu__btn {
  min-height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  font-weight: 700;
  gap: 8px;
}
.create-menu__ic {
  width: 20px;
  height: 20px;
}
.create-menu__caret {
  width: 16px;
  height: 16px;
}
.create-menu__list {
  display: flex;
  flex-direction: column;
  padding: 6px;
}
.create-menu__item {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 48px;
  padding: 0 10px;
  border: none;
  border-radius: 14px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  text-align: left;
  cursor: pointer;
}
.create-menu__item:hover {
  background: var(--el-fill-color-light);
}
.create-menu__sk {
  width: 32px;
  height: 32px;
  border-radius: 10px;
}
.create-menu__sk :deep(svg) {
  width: 16px;
  height: 16px;
}
</style>
