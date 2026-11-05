<!-- 手机「更多」页（MoreMobile）：底部导航「更多」打开的整页面板。
     页头（更多 + 深浅色 + 通知）→ 主色账号卡（头像 / 昵称 / 邮箱 → 个人中心，节点切换）→ 功能搜索 →
     按菜单分组的彩色贴纸入口（同一页面里的标签直接展开成入口，后台任务带进行中角标）→ 我的（个人中心 / 外观 / 语言 / 退出）→ 版本与节点数。 -->
<template>
  <Teleport to="body">
    <Transition name="more-panel">
      <div
        v-if="menuStore.isMobile && menuStore.isMobileMenuOpen"
        class="more"
        role="dialog"
        aria-modal="true"
        :aria-label="t('layout.tabbar.more')"
      >
        <div class="more__scroll">
          <header class="more__head">
            <h1 class="more__title">{{ t('layout.tabbar.more') }}</h1>
            <AppIconButton
              :icon="themeStore.isDarkTheme ? 'HOutline:SunIcon' : 'HOutline:MoonIcon'"
              :label="t('theme.toggleDark')"
              :box="44"
              @click="themeStore.toggleDarkQuick()"
            />
            <NotificationMenu />
          </header>

          <section class="more__hero" :aria-label="t('shell.more.account')">
            <span class="more__blob" aria-hidden="true" />
            <span class="more__ring" aria-hidden="true" />
            <button type="button" class="more__me" @click="go({ path: '/profile', query: { tab: 'info' } })">
              <span class="more__av">
                <AppAvatar :src="userStore.resolvedUserAvatar" :size="54" :name="displayName" />
              </span>
              <span class="more__me-text">
                <span class="more__me-name">{{ displayName }}</span>
                <span class="more__me-sub">{{ meSub }}</span>
              </span>
              <span class="more__me-go" aria-hidden="true">
                <component :is="ic('HOutline:ChevronRightIcon')" />
              </span>
            </button>
            <div class="more__node">
              <NodeSwitcher variant="hero" />
            </div>
          </section>

          <label class="more__search">
            <component :is="ic('HOutline:MagnifyingGlassIcon')" class="more__search-ic" />
            <input
              ref="searchRef"
              v-model="keyword"
              type="search"
              :placeholder="t('shell.more.searchPlaceholder')"
              :aria-label="t('shell.more.search')"
            />
          </label>

          <div v-if="shownGroups.length" class="more__groups">
            <section
              v-for="g in shownGroups"
              :key="g.key"
              class="more__group"
              :style="{ flex: g.flex }"
              :aria-label="g.title"
            >
              <h2 v-if="g.title" class="more__h">{{ g.title }}</h2>
              <div class="more__tiles" :style="{ gridTemplateColumns: `repeat(${g.cols}, minmax(0, 1fr))` }">
                <button
                  v-for="it in g.items"
                  :key="it.key"
                  type="button"
                  class="more__tile"
                  @click="go(it.to)"
                >
                  <span
                    class="more__stk m-sticker"
                    :class="`m-sticker--${it.tone}`"
                    :style="{ transform: `rotate(${it.rot})` }"
                  >
                    <component :is="ic(it.icon)" />
                    <span
                      v-if="it.badge"
                      class="more__bdg m-num"
                      :aria-label="t('shell.more.runningBadge', { n: it.badge })"
                    >{{ it.badge }}</span>
                  </span>
                  {{ it.label }}
                </button>
              </div>
            </section>
          </div>
          <p v-else class="more__empty">{{ t('shell.more.noMatch') }}</p>

          <section aria-labelledby="more-mine-h">
            <h2 id="more-mine-h" class="more__h">{{ t('shell.more.mine') }}</h2>
            <div class="more__grp">
              <button type="button" class="more__row" @click="go({ path: '/profile', query: { tab: 'info' } })">
                <span class="more__sk m-sticker m-sticker--solid-p" style="transform: rotate(-3deg)">
                  <component :is="ic('HOutline:UserIcon')" />
                </span>
                <span class="more__row-t">{{ t('layout.profile') }}</span>
                <span class="more__row-v">{{ t('shell.more.profileHint') }}</span>
                <component :is="ic('HOutline:ChevronRightIcon')" class="more__chev" />
              </button>
              <button type="button" class="more__row" @click="go({ path: '/profile', query: { tab: 'appearance' } })">
                <span class="more__sk m-sticker m-sticker--solid-a" style="transform: rotate(3deg)">
                  <component :is="ic('HOutline:SwatchIcon')" />
                </span>
                <span class="more__row-t">{{ t('theme.title') }}</span>
                <span class="more__row-v">{{ appearanceSummary }}</span>
                <component :is="ic('HOutline:ChevronRightIcon')" class="more__chev" />
              </button>
              <AppDropdown align="end" :width="220" block>
                <template #trigger="{ open }">
                  <button type="button" class="more__row" :aria-expanded="open">
                    <span class="more__sk m-sticker m-sticker--solid-a2" style="transform: rotate(-2deg)">
                      <component :is="ic('HOutline:GlobeAltIcon')" />
                    </span>
                    <span class="more__row-t">{{ t('language.tooltip') }}</span>
                    <span class="more__row-v">{{ currentLanguage }}</span>
                    <component :is="ic('HOutline:ChevronRightIcon')" class="more__chev" />
                  </button>
                </template>
                <template #default="{ close }">
                  <div class="more__langs">
                    <button
                      v-for="item in languageOptions"
                      :key="item.code"
                      type="button"
                      class="more__lang"
                      :class="{ 'is-on': locale === item.code }"
                      :aria-pressed="locale === item.code"
                      @click="pickLocale(item.code, close)"
                    >
                      <span>{{ t(item.labelKey) }}</span>
                      <component :is="ic('HOutline:CheckIcon')" v-if="locale === item.code" />
                    </button>
                  </div>
                </template>
              </AppDropdown>
              <button type="button" class="more__row more__row--bad" @click="onLogout">
                <span class="more__sk m-sticker m-sticker--bad">
                  <component :is="ic('HOutline:ArrowRightOnRectangleIcon')" />
                </span>
                <span class="more__row-t">{{ t('layout.logout') }}</span>
              </button>
            </div>
          </section>

          <p class="more__foot m-num">{{ footText }}</p>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import type { RouteLocationRaw } from 'vue-router'
import { languageOptions, setAppLocale, translateKnownText } from '@/locales'
import { PAGE_GROUPS, type PageGroup, type PageGroupKey } from '@/config/pageGroups'
import { MenuStatus, MenuType, type MenuInfo } from '@/types/v1/system'
import { Dialog } from '@/utils/dialog'
import NodeSwitcher from './NodeSwitcher.vue'
import NotificationMenu from './NotificationMenu.vue'
import { TAB_ROOTS, moreSearchFocusTick, useMorePanel } from './tabbar'
import { useRunningTaskCount } from './useRunningTasks'

interface Tile {
  key: string
  label: string
  icon: string
  to: RouteLocationRaw
  badge?: number
  tone: string
  rot: string
}
interface Group {
  key: string
  title: string
  items: Tile[]
  cols: number
  flex: string
}

const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()
const nodeStore = useNodeStore()
const userStore = useUserStore()
const themeStore = useThemeStore()
const { t, locale } = useI18n()
const more = useMorePanel()
const ic = (name: string) => menuStore.iconComponents[name]
const runningTasks = useRunningTaskCount()

const keyword = ref('')
const searchRef = ref<HTMLInputElement>()

// 贴纸配色与倾角循环（与原型一致）
const TONES = ['solid-p', 'solid-a2', 'solid-a', 'p', 'a2', 'ok', 'a', 'solid-ink']
const ROTS = ['-4deg', '3deg', '-2deg', '4deg', '-3deg', '2deg']

// 内置目录 id（internal/data/default_rbac.go）
const HOME_DIR_ID = 'menu_1'
const APPS_DIR_ID = 'menu_2'
const FILE_DIR_ID = 'menu_3'

// 同一页面里的标签直接展开成入口（智商检测只是一个工具，不展开）
const EXPAND_GROUPS: PageGroupKey[] = ['file', 'docker', 'node', 'access', 'logs', 'settings']
const TAB_ICONS: Record<string, string> = {
  'docker.container': 'HOutline:CubeIcon',
  'docker.image': 'HOutline:PhotoIcon',
  'docker.network': 'HOutline:RectangleGroupIcon',
  'docker.config': 'HOutline:LinkIcon',
  'node.list': 'HOutline:ServerIcon',
  'node.key': 'HOutline:KeyIcon',
  'access.user': 'HOutline:UserGroupIcon',
  'access.role': 'HOutline:ShieldCheckIcon',
  'access.menu': 'HOutline:Bars3Icon',
  'logs.task': 'HOutline:ClockIcon',
  'logs.operation': 'HOutline:DocumentTextIcon',
  'settings.general': 'HOutline:Cog6ToothIcon',
  'settings.oidc': 'HOutline:IdentificationIcon',
}
const TAB_LABELS: Record<string, string> = {
  'settings.general': 'shell.more.settings',
  'settings.oidc': 'shell.more.oidc',
}

const visibleTabs = (key: PageGroupKey) => {
  const def: PageGroup = PAGE_GROUPS[key]
  return def.tabs.filter(
    (tab) =>
      (!tab.perm || menuStore.hasButtonPermission(tab.perm)) &&
      nodeStore.canAccessRoute(tab.nodePath || `${def.path}/${tab.key}`),
  )
}

const tilesOfMenu = (menu: MenuInfo): Omit<Tile, 'tone' | 'rot'>[] => {
  const path = menu.path
  const groupKey = EXPAND_GROUPS.find((k) => PAGE_GROUPS[k].path === path)
  if (groupKey) {
    let tabs = visibleTabs(groupKey)
    // 一级页面（如 Docker 容器）已在底部导航里，只列其余标签
    if (TAB_ROOTS.includes(path)) tabs = tabs.slice(1)
    return tabs.map((tab) => ({
      key: `${groupKey}.${tab.key}`,
      label: t(TAB_LABELS[`${groupKey}.${tab.key}`] || tab.labelKey),
      icon: TAB_ICONS[`${groupKey}.${tab.key}`] || tab.icon || menu.icon || 'HOutline:Squares2X2Icon',
      to: { path, query: { tab: tab.key } },
      badge: groupKey === 'logs' && tab.key === 'task' && runningTasks.value > 0 ? runningTasks.value : undefined,
    }))
  }
  if (TAB_ROOTS.includes(path)) return []
  return [
    {
      key: path,
      label: translateKnownText(menu.title),
      icon: menu.icon || 'HOutline:Squares2X2Icon',
      to: path,
    },
  ]
}

const isActiveMenu = (m: MenuInfo) =>
  m.type !== MenuType.MenuType_Button && m.status === MenuStatus.MenuStatus_Active

const groups = computed<Group[]>(() => {
  const out: { key: string; title: string; items: Omit<Tile, 'tone' | 'rot'>[] }[] = []
  const loose: Omit<Tile, 'tone' | 'rot'>[] = []
  const collect = (list: MenuInfo[], into: Omit<Tile, 'tone' | 'rot'>[]) => {
    for (const m of list) {
      if (!isActiveMenu(m)) continue
      if (m.type === MenuType.MenuType_Menu && m.path) into.push(...tilesOfMenu(m))
      else if (m.children?.length) collect(m.children, into)
    }
  }
  for (const top of menuStore.sidebarMenuList) {
    if (!isActiveMenu(top)) continue
    if (top.type === MenuType.MenuType_Directory) {
      const items: Omit<Tile, 'tone' | 'rot'>[] = []
      collect(top.children || [], items)
      if (items.length) out.push({ key: top.id, title: translateKnownText(top.title), items })
    } else if (top.type === MenuType.MenuType_Menu && top.path) {
      loose.push(...tilesOfMenu(top))
    }
  }
  if (loose.length) out.push({ key: 'loose', title: '', items: loose })

  // 「主页」目录里除工作台（在底部导航）外的入口（性能监控）并进「应用」组，组名改为「应用与监控」（MoreMobile）
  const home = out.find((g) => g.key === HOME_DIR_ID)
  const appsGroup = out.find((g) => g.key === APPS_DIR_ID)
  if (home && appsGroup) {
    appsGroup.items.push(...home.items)
    appsGroup.title = t('shell.more.appsAndMonitor')
    out.splice(out.indexOf(home), 1)
    // 与原型一致：「文件」在左、「应用与监控」在右
    const fileIdx = out.findIndex((g) => g.key === FILE_DIR_ID)
    const appsIdx = out.indexOf(appsGroup)
    if (fileIdx > appsIdx) {
      out.splice(appsIdx, 1)
      out.splice(fileIdx, 0, appsGroup)
    }
  }

  let n = 0
  return out.map((g) => {
    const items = g.items.map((it) => {
      const tile = { ...it, tone: TONES[n % TONES.length]!, rot: ROTS[n % ROTS.length]! }
      n += 1
      return tile
    })
    const len = items.length
    const cols = len <= 4 ? len : len % 4 === 0 ? 4 : 3
    // 入口很少的分组两两并排（如「文件」和「应用与监控」），其余独占一行
    const flex = len === 1 ? '1 1 100px' : len === 2 ? '1 1 150px' : '1 1 100%'
    return { ...g, items, cols, flex }
  })
})

const shownGroups = computed(() => {
  const q = keyword.value.trim().toLowerCase()
  if (!q) return groups.value
  return groups.value
    .map((g) => {
      const items = g.items.filter(
        (it) => it.label.toLowerCase().includes(q) || g.title.toLowerCase().includes(q),
      )
      const len = items.length
      return { ...g, items, cols: Math.min(Math.max(len, 1), 4), flex: '1 1 100%' }
    })
    .filter((g) => g.items.length)
})

const displayName = computed(() => userStore.userInfo?.name || userStore.userInfo?.username || '—')
const meSub = computed(() => {
  const email = userStore.userInfo?.email
  return email ? `${email} · ${t('layout.profile')}` : t('layout.profile')
})
const appearanceSummary = computed(
  () => `${t(themeStore.currentPalette.labelKey)} · ${t(`theme.${themeStore.themeMode}`)}`,
)
const currentLanguage = computed(() => {
  const item = languageOptions.find((o) => o.code === locale.value)
  return item ? t(item.labelKey) : ''
})
const footText = computed(() => {
  const version = `Momoko ${menuStore.currentVersion || 'dev'}`
  if (!menuStore.hasButtonPermission('node:view') || !nodeStore.nodes.length) return version
  return `${version} · ${t('shell.more.nodesConnected', { n: nodeStore.nodes.length })}`
})

const go = (to: RouteLocationRaw) => {
  more.close()
  keyword.value = ''
  void router.push(to)
}

const pickLocale = (code: string, close: () => void) => {
  setAppLocale(code as never)
  close()
}

const onLogout = () => {
  Dialog.info({
    showCancelButton: true,
    content: t('layout.logoutConfirmContent'),
    confirmText: t('layout.logoutConfirmText'),
    cancelText: t('layout.logoutCancelText'),
    onConfirm: async () => {
      more.close()
      await userStore.logout()
    },
  })
}

// 切换页面（包括浏览器返回）时收起
watch(
  () => route.fullPath,
  () => more.close(),
)
// 离开手机布局时收起
watch(
  () => menuStore.isMobile,
  (mobile) => {
    if (!mobile) more.close()
  },
)
// 请求聚焦搜索框
watch(moreSearchFocusTick, async () => {
  await nextTick()
  searchRef.value?.focus()
})
watch(
  () => menuStore.isMobileMenuOpen,
  (open) => {
    if (!open) keyword.value = ''
  },
)

const onKeydown = (e: KeyboardEvent) => {
  if (e.key === 'Escape' && menuStore.isMobileMenuOpen) more.close()
}
onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped lang="scss">
.more {
  position: fixed;
  inset: 0;
  z-index: 2040;
  background: var(--el-bg-color-page);
}
.more__scroll {
  height: 100%;
  overflow-y: auto;
  overscroll-behavior: contain;
  display: flex;
  flex-direction: column;
  gap: 20px;
  padding: 0 16px calc(96px + env(safe-area-inset-bottom));
  scrollbar-width: none;
}
.more__scroll::-webkit-scrollbar {
  display: none;
}
/* 滚动容器是纵向 flex：子块不能被压扁（账号卡带 overflow:hidden，默认会缩到很矮） */
.more__scroll > * {
  flex-shrink: 0;
}
.more__head {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 22px 0 0 4px;
  margin-right: -4px;
  margin-bottom: -10px;
}
.more__title {
  flex: 1;
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 2rem;
  font-weight: 800;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
}

/* ---- 账号卡 ---- */
.more__hero {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px;
  border-radius: var(--app-radius-xl);
  background: var(--m-p);
  color: var(--m-pOn);
}
.more__blob {
  position: absolute;
  right: -46px;
  top: -62px;
  width: 160px;
  height: 160px;
  border-radius: 50%;
  background: var(--m-a);
}
.more__ring {
  position: absolute;
  right: 110px;
  bottom: -26px;
  width: 54px;
  height: 54px;
  border-radius: 50%;
  border: 10px solid var(--m-a2);
}
.more__me {
  position: relative;
  display: flex;
  align-items: center;
  gap: 14px;
  min-height: 64px;
  padding: 0;
  border: 0;
  background: transparent;
  color: var(--m-pOn);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.more__av {
  display: inline-flex;
  flex: none;
  border: 3px solid var(--m-pOn);
  border-radius: 20px;
  overflow: hidden;
  background: var(--m-a);
  transform: rotate(-5deg);
  :deep(.app-avatar) {
    border-radius: 0;
    background: var(--m-a);
    color: var(--m-aOn);
  }
}
.more__me-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 3px;
}
.more__me-name {
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.more__me-sub {
  font-size: 0.8125rem;
  opacity: 0.95;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.more__me-go {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 36px;
  height: 36px;
  flex: none;
  border-radius: 50%;
  background: var(--m-pOn);
  color: var(--m-p);
  :deep(svg) {
    width: 16px;
    height: 16px;
    stroke-width: 2.4;
  }
}
.more__node {
  position: relative;
  display: flex;
  min-width: 0;
}

/* ---- 搜索 ---- */
.more__search {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 0 16px;
  border-radius: 999px;
  background: var(--el-bg-color);
  border: 1.5px solid var(--el-border-color-light);
  color: var(--el-text-color-secondary);
}
.more__search:focus-within {
  border-color: var(--m-p);
}
.more__search-ic {
  width: 20px;
  height: 20px;
  flex: none;
}
.more__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  background: transparent;
  font: inherit;
  font-size: 0.9375rem;
  color: var(--el-text-color-primary);
  outline: none;
}
.more__search input::placeholder {
  color: var(--el-text-color-secondary);
}
.more__search input::-webkit-search-cancel-button {
  -webkit-appearance: none;
  appearance: none;
}

/* ---- 入口 ---- */
.more__groups {
  display: flex;
  flex-wrap: wrap;
  gap: 20px 12px;
}
.more__group {
  min-width: 0;
}
.more__h {
  margin: 0 0 8px 6px;
  font-size: 0.8125rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.more__tiles {
  display: grid;
  gap: 6px 2px;
  padding: 10px 8px;
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.more__tile {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 8px 2px 6px;
  border: 0;
  border-radius: 18px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.75rem;
  font-weight: 700;
  line-height: 1.25;
  text-align: center;
  cursor: pointer;
  overflow-wrap: anywhere;
}
.more__tile:hover {
  background: var(--el-fill-color-light);
}
.more__stk {
  position: relative;
  width: 52px;
  height: 52px;
  border-radius: 18px;
  :deep(svg) {
    width: 22px;
    height: 22px;
  }
}
.more__bdg {
  position: absolute;
  top: -6px;
  right: -8px;
  min-width: 22px;
  height: 22px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 0.6875rem;
  font-weight: 800;
  display: flex;
  align-items: center;
  justify-content: center;
  border: 2px solid var(--el-bg-color);
}
.more__empty {
  padding: 24px 0;
  text-align: center;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}

/* ---- 我的 ---- */
.more__grp {
  display: flex;
  flex-direction: column;
  border-radius: var(--app-radius-lg);
  overflow: hidden;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  /* .more__row 自身 border: 0，分隔线要比它更具体才不会被覆盖 */
  > * + *,
  > * + .more__row {
    border-top: 1px solid var(--el-border-color-light);
  }
}
.more__row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 56px;
  padding: 8px 14px;
  border: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.more__row:hover {
  background: var(--el-fill-color-light);
}
.more__row--bad {
  color: var(--m-bad);
}
.more__row--bad .more__row-t {
  font-weight: 800;
}
.more__sk {
  width: 34px;
  height: 34px;
  border-radius: 11px;
  :deep(svg) {
    width: 18px;
    height: 18px;
  }
}
.more__row-t {
  flex: 1;
  min-width: 0;
  font-size: 0.9375rem;
  font-weight: 700;
}
.more__row-v {
  max-width: 45%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.more__chev {
  width: 16px;
  height: 16px;
  flex: none;
  color: var(--el-text-color-secondary);
}
.more__langs {
  display: flex;
  flex-direction: column;
  padding: 6px;
}
.more__lang {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 44px;
  padding: 0 12px;
  border: 0;
  border-radius: 12px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
  cursor: pointer;
  :deep(svg) {
    width: 18px;
    height: 18px;
    color: var(--m-pInk);
  }
}
.more__lang:hover {
  background: var(--el-fill-color-light);
}
.more__lang.is-on {
  color: var(--m-pInk);
  font-weight: 700;
}
.more__foot {
  margin: -6px 0 0;
  text-align: center;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}

.more-panel-enter-active,
.more-panel-leave-active {
  transition: opacity 0.16s ease, transform 0.16s ease;
}
.more-panel-enter-from,
.more-panel-leave-to {
  opacity: 0;
  transform: translateY(8px);
}
</style>
