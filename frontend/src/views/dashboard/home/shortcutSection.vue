<!-- 快捷入口（DashboardPC / Mobile）：彩色「贴纸」图标块网格，只显示当前用户在当前节点能进入的页面。
     「自定义」挑选要显示的入口（保存在当前浏览器）。 -->
<template>
  <section v-if="visibleShortcuts.length || catalog.length" class="sc" aria-labelledby="sc-h">
    <div class="sc__head">
      <h2 id="sc-h" class="sc__title">{{ t('dashboard.home.quickEntries') }}</h2>
      <span class="sc__spacer" />
      <UButton v-if="catalog.length" color="neutral" variant="soft" size="sm" class="sc__custom" @click="openCustomize">
        {{ t('dashboard.home.customize') }}
      </UButton>
    </div>

    <div v-if="visibleShortcuts.length" class="sc__grid">
      <button
        v-for="(item, i) in visibleShortcuts"
        :key="item.key"
        type="button"
        class="sc__item"
        @click="router.push(item.to)"
      >
        <span
          class="sc__icon m-sticker"
          :class="`m-sticker--${item.color}`"
          :style="{ transform: `rotate(${ROTS[i % ROTS.length]})` }"
          aria-hidden="true"
        >
          <component :is="menuStore.iconComponents[item.icon]" />
        </span>
        <span class="sc__label">{{ t(item.labelKey) }}</span>
      </button>
    </div>
    <p v-else class="sc__empty">{{ t('dashboard.home.quickEmpty') }}</p>

    <FormDialog
      v-model="customizeOpen"
      :title="t('dashboard.home.customizeTitle')"
      :width="480"
      @confirm="saveCustomize"
    >
      <p class="sc-dlg__desc">{{ t('dashboard.home.customizeDesc') }}</p>
      <div class="sc-dlg__list">
        <label v-for="item in catalog" :key="item.key" class="sc-dlg__row">
          <span class="sc-dlg__sk m-sticker" :class="`m-sticker--${item.color}`" aria-hidden="true">
            <component :is="menuStore.iconComponents[item.icon]" />
          </span>
          <span class="sc-dlg__name">{{ t(item.labelKey) }}</span>
          <AppSwitch
            :model-value="draft.includes(item.key)"
            :aria-label="t(item.labelKey)"
            @update:model-value="toggleDraft(item.key, $event)"
          />
        </label>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="ghost" @click="resetCustomize">{{ t('dashboard.home.customizeReset') }}</UButton>
        <span class="sc-dlg__spacer" />
        <UButton color="neutral" variant="soft" @click="close">{{ t('common.cancel') }}</UButton>
        <UButton color="primary" @click="saveCustomize">{{ t('dashboard.home.customizeSave') }}</UButton>
      </template>
    </FormDialog>
  </section>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import { pageTabTo } from '@/config/pageGroups'
import { useI18n } from 'vue-i18n'

interface Shortcut {
  key: string
  labelKey: string
  icon: string
  /** 贴纸配色（design-tokens.css .m-sticker--*） */
  color: string
  path: string
  to: RouteLocationRaw
}

const STORAGE_KEY = 'dashboard-shortcuts'
const ROTS = ['-4deg', '3deg', '-2deg', '4deg', '-3deg', '2deg']

// 可选入口目录（顺序即默认顺序）
const CATALOG: Shortcut[] = [
  { key: 'files', labelKey: 'dashboard.home.shortcuts.fileManager', icon: 'HOutline:FolderIcon', color: 'solid-p', path: '/file/index', to: pageTabTo('file', 'index') },
  { key: 'ssh', labelKey: 'dashboard.home.shortcuts.ssh', icon: 'HOutline:CommandLineIcon', color: 'solid-ink', path: '/openssh/management', to: '/openssh/management' },
  { key: 'forward', labelKey: 'dashboard.home.shortcuts.portForward', icon: 'HOutline:ArrowsRightLeftIcon', color: 'solid-a', path: '/tools/port-forward', to: '/tools/port-forward' },
  { key: 'tunnel', labelKey: 'dashboard.home.shortcuts.tunnel', icon: 'HOutline:GlobeAltIcon', color: 'solid-a2', path: '/tools/tunnel', to: '/tools/tunnel' },
  { key: 'containers', labelKey: 'dashboard.home.shortcuts.containers', icon: 'HOutline:CubeIcon', color: 'p', path: '/docker', to: '/docker' },
  { key: 'shares', labelKey: 'dashboard.home.shortcuts.shares', icon: 'HOutline:ShareIcon', color: 'ok', path: '/file/share', to: pageTabTo('file', 'share') },
  { key: 'iqcheck', labelKey: 'dashboard.home.shortcuts.iqcheck', icon: 'HOutline:SparklesIcon', color: 'a', path: '/iqcheck', to: '/iqcheck' },
  { key: 'nodes', labelKey: 'dashboard.home.shortcuts.nodes', icon: 'HOutline:ServerIcon', color: 'a2', path: '/node', to: '/node' },
  { key: 'images', labelKey: 'dashboard.home.shortcuts.images', icon: 'HOutline:PhotoIcon', color: 'p', path: '/docker', to: { path: '/docker', query: { tab: 'image' } } },
  { key: 'apps', labelKey: 'dashboard.home.shortcuts.apps', icon: 'HOutline:Squares2X2Icon', color: 'solid-p', path: '/instance/list', to: '/instance/list' },
  { key: 'appTypes', labelKey: 'dashboard.home.shortcuts.appTypes', icon: 'HOutline:RectangleStackIcon', color: 'a2', path: '/instance/type', to: '/instance/type' },
  { key: 'sources', labelKey: 'dashboard.home.shortcuts.sources', icon: 'HOutline:CircleStackIcon', color: 'ok', path: '/file/source', to: pageTabTo('file', 'source') },
  { key: 'users', labelKey: 'dashboard.home.shortcuts.users', icon: 'HOutline:UserGroupIcon', color: 'a2', path: '/system/access', to: '/system/access' },
  { key: 'logs', labelKey: 'dashboard.home.shortcuts.logs', icon: 'HOutline:ClockIcon', color: 'a', path: '/system/logs', to: '/system/logs' },
  { key: 'settings', labelKey: 'dashboard.home.shortcuts.settings', icon: 'HOutline:Cog6ToothIcon', color: 'solid-ink', path: '/system/settings', to: '/system/settings' },
]
// 默认：PC 一屏 6 个（3×2），手机 8 个（4×2）
const DEFAULT_PC = ['files', 'ssh', 'forward', 'tunnel', 'containers', 'shares']
const DEFAULT_MOBILE = ['files', 'ssh', 'forward', 'tunnel', 'images', 'shares', 'iqcheck', 'nodes']

const readStored = (): string[] | null => {
  try {
    const raw = localStorage.getItem(STORAGE_KEY)
    if (!raw) return null
    const list = JSON.parse(raw)
    return Array.isArray(list) ? list.filter((k): k is string => typeof k === 'string') : null
  } catch {
    return null
  }
}

const router = useRouter()
const menuStore = useMenuStore()
const { t } = useI18n()
const nodeStore = useNodeStore()
const accessible = useAccessibleRoutes()

const stored = ref<string[] | null>(readStored())

// 只展示菜单里存在、且当前节点授权的入口
const catalog = computed(() =>
  CATALOG.filter((item) => accessible.value.has(item.path) && nodeStore.canAccessRoute(item.path)),
)
const selectedKeys = computed(
  () => stored.value ?? (menuStore.isMobile ? DEFAULT_MOBILE : DEFAULT_PC),
)
const visibleShortcuts = computed(() => {
  const byKey = new Map(catalog.value.map((c) => [c.key, c]))
  return selectedKeys.value.map((k) => byKey.get(k)).filter((c): c is Shortcut => !!c)
})

// ---- 自定义 ----
const customizeOpen = ref(false)
const draft = ref<string[]>([])

const openCustomize = () => {
  draft.value = [...selectedKeys.value]
  customizeOpen.value = true
}
const toggleDraft = (key: string, on: boolean) => {
  if (on) {
    if (!draft.value.includes(key)) draft.value = [...draft.value, key]
  } else {
    draft.value = draft.value.filter((k) => k !== key)
  }
}
const saveCustomize = () => {
  // 按目录顺序保存，保证网格排列稳定
  const ordered = CATALOG.map((c) => c.key).filter((k) => draft.value.includes(k))
  stored.value = ordered
  try {
    localStorage.setItem(STORAGE_KEY, JSON.stringify(ordered))
  } catch {
    // 无法持久化时本次会话内生效
  }
  customizeOpen.value = false
}
const resetCustomize = () => {
  stored.value = null
  try {
    localStorage.removeItem(STORAGE_KEY)
  } catch {
    // 忽略
  }
  customizeOpen.value = false
}
</script>

<style scoped lang="scss">
.sc {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
  padding: calc(var(--cmp-card-unit) * 22);
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.sc__head {
  display: flex;
  align-items: center;
}
.sc__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.sc__spacer {
  flex: 1;
}
.sc__custom {
  min-height: 36px;
  padding-inline: 14px;
  border-radius: 999px;
  font-weight: 700;
}
.sc__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 12px;
}
.sc__item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
  padding: 12px 10px;
  border: 1px solid var(--el-border-color-light);
  border-radius: 18px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s;
}
.sc__item:hover {
  background: var(--el-fill-color-light);
}
.sc__icon {
  width: 40px;
  height: 40px;
  border-radius: 13px;
}
.sc__icon :deep(svg) {
  width: 20px;
  height: 20px;
}
.sc__label {
  display: -webkit-box;
  min-width: 0;
  overflow: hidden;
  font-size: 0.875rem;
  font-weight: 700;
  line-height: 1.25;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.sc__empty {
  margin: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
@media (width < 1200px) and (width >= 640px) {
  .sc__grid {
    grid-template-columns: repeat(auto-fill, minmax(150px, 1fr));
  }
}
@media (width < 640px) {
  .sc {
    padding: 16px;
    gap: 14px;
  }
  .sc__title {
    font-size: 1rem;
  }
  .sc__grid {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 8px 4px;
  }
  .sc__item {
    flex-direction: column;
    gap: 8px;
    padding: 4px 0;
    border: none;
    border-radius: 16px;
    text-align: center;
  }
  .sc__icon {
    width: 52px;
    height: 52px;
    border-radius: 18px;
  }
  .sc__label {
    display: block;
    max-width: 100%;
    font-size: 0.75rem;
    font-weight: 600;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
}

/* ---- 自定义弹窗 ---- */
.sc-dlg__desc {
  margin: 0 0 12px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.sc-dlg__list {
  display: flex;
  flex-direction: column;
}
.sc-dlg__row {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 52px;
  padding: 6px 0;
  cursor: pointer;
}
.sc-dlg__row + .sc-dlg__row {
  border-top: 1px solid var(--el-border-color-lighter);
}
.sc-dlg__sk {
  width: 34px;
  height: 34px;
  border-radius: 11px;
}
.sc-dlg__sk :deep(svg) {
  width: 17px;
  height: 17px;
}
.sc-dlg__name {
  flex: 1;
  min-width: 0;
  font-size: 0.875rem;
  font-weight: 700;
}
.sc-dlg__spacer {
  flex: 1;
}
</style>
