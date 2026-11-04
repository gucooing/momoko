<!-- 全局搜索/快速跳转（⌘K）。触发器本身就是可输入搜索框，结果下拉锚定在同一框下方。 -->
<template>
  <div ref="rootRef" class="cmd-search">
    <div class="search-box" :class="{ 'is-open': open, 'is-focused': focused }">
      <component
        :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']"
        class="search-box__icon"
      />
      <input
        ref="inputRef"
        v-model="query"
        class="search-box__input"
        type="search"
        :placeholder="searchPlaceholder"
        :aria-label="t('shell.search')"
        autocomplete="off"
        @focus="onFocus"
        @keydown="onKey"
      />
      <kbd v-if="!open && !query" class="search-box__kbd">{{ isMac ? '⌘' : 'Ctrl' }} K</kbd>
      <button
        v-else-if="query"
        type="button"
        class="search-box__clear"
        :aria-label="t('common.close')"
        @mousedown.prevent
        @click="clearQuery"
      >
        ×
      </button>
    </div>

    <Teleport to="body">
      <div v-if="open" ref="panelRef" class="cmd-panel" :style="panelStyle" role="listbox">
        <div class="cmd-panel__list">
          <template v-for="group in groupedResults" :key="group.key">
            <div v-if="group.title" class="cmd-panel__group">{{ group.title }}</div>
            <button
              v-for="item in group.items"
              :key="item.key"
              type="button"
              class="cmd-panel__item"
              :class="{ 'is-hl': item.index === highlight }"
              role="option"
              :aria-selected="item.index === highlight"
              @click="jump(item.to)"
              @mouseenter="highlight = item.index"
            >
              <span class="cmd-panel__item-icon">
                <component :is="menuStore.iconComponents[item.icon]" v-if="item.icon" />
              </span>
              <span class="cmd-panel__item-title">{{ item.title }}</span>
              <span class="cmd-panel__item-path">{{ item.hint }}</span>
            </button>
          </template>
          <div v-if="searching" class="cmd-panel__empty">{{ t('shell.cmd.searching') }}</div>
          <div v-else-if="results.length === 0" class="cmd-panel__empty">
            {{ t('common.noData') }}
          </div>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import type { RouteLocationRaw } from 'vue-router'
import { MenuType, type MenuInfo } from '@/types/v1/system'
import { getInstances } from '@/api/instance'
import { listDockerContainers } from '@/api/docker'
import { translateKnownText } from '@/locales'
import { useI18n } from 'vue-i18n'

const router = useRouter()
const menuStore = useMenuStore()
const { t } = useI18n()

const rootRef = ref<HTMLElement | null>(null)
const inputRef = ref<HTMLInputElement | null>(null)
const panelRef = ref<HTMLElement | null>(null)
const query = ref('')
const open = ref(false)
const focused = ref(false)
const highlight = ref(0)
const panelStyle = ref<Record<string, string>>({})
const isMac = typeof navigator !== 'undefined' && /Mac|iPhone|iPad/.test(navigator.platform)
const searchPlaceholder = computed(() => t('layout.searchPlaceholder'))

interface FlatItem {
  key: string
  group: 'pages' | 'apps' | 'containers'
  title: string
  hint: string
  to: RouteLocationRaw
  icon?: string
}
const flatItems = computed(() => {
  const out: FlatItem[] = []
  const walk = (nodes?: MenuInfo[]) => {
    nodes?.forEach((n) => {
      if (n.type === MenuType.MenuType_Menu && n.path) {
        out.push({
          key: `p:${n.path}`,
          group: 'pages',
          title: translateKnownText(n.title),
          hint: n.path,
          to: n.path,
          icon: n.icon,
        })
      }
      if (n.children?.length) walk(n.children)
    })
  }
  walk(menuStore.navMenuList)
  return out
})

// 应用与容器：输入后去后端按关键字查（各取前 5 条），没有权限的类别不查
const remoteItems = ref<FlatItem[]>([])
const searching = ref(false)
let searchSeq = 0
let searchTimer: ReturnType<typeof setTimeout> | null = null
const nodeStore = useNodeStore()
const searchRemote = async (q: string) => {
  const seq = ++searchSeq
  const jobs: Promise<FlatItem[]>[] = []
  if (menuStore.hasButtonPermission('instance:manage') && nodeStore.canAccessRoute('/instance/list')) {
    jobs.push(
      getInstances({ page: 1, pageSize: 5, keywords: q }).then(({ data }) =>
        (data?.infos || []).map((it) => ({
          key: `a:${it.id}`,
          group: 'apps' as const,
          title: it.name,
          hint: it.type || '',
          to: { path: `/instance/app/${it.id}`, query: { tabTitle: it.name } },
          icon: 'HOutline:CubeIcon',
        })),
      ),
    )
  }
  if (menuStore.hasButtonPermission('docker:view') && nodeStore.canAccessRoute('/docker')) {
    jobs.push(
      listDockerContainers({ status: '', keyword: q, network: '', labels: {}, page: 1, pageSize: 5 }).then(
        ({ data }) =>
          (data?.items || []).map((c) => {
            const name = (c.names?.[0] || c.id.slice(0, 12)).replace(/^\//, '')
            return {
              key: `c:${c.id}`,
              group: 'containers' as const,
              title: name,
              hint: c.image,
              to: { path: '/docker', query: { tab: 'container', keyword: name } },
              icon: 'HOutline:ServerStackIcon',
            }
          }),
      ),
    )
  }
  if (!jobs.length) {
    remoteItems.value = []
    return
  }
  searching.value = true
  const settled = await Promise.allSettled(jobs)
  if (seq !== searchSeq) return
  remoteItems.value = settled.flatMap((r) => (r.status === 'fulfilled' ? r.value : []))
  searching.value = false
}

const results = computed(() => {
  const q = query.value.trim().toLowerCase()
  if (!q) return flatItems.value.slice(0, 8)
  const pages = flatItems.value.filter((i) => i.title.toLowerCase().includes(q)).slice(0, 8)
  return [...pages, ...remoteItems.value]
})

const GROUP_TITLES: Record<FlatItem['group'], string> = {
  pages: 'shell.cmd.pages',
  apps: 'shell.cmd.apps',
  containers: 'shell.cmd.containers',
}
const groupedResults = computed(() => {
  const showTitles = !!query.value.trim()
  const out: { key: string; title: string; items: (FlatItem & { index: number })[] }[] = []
  results.value.forEach((item, index) => {
    let g = out.find((x) => x.key === item.group)
    if (!g) {
      g = { key: item.group, title: showTitles ? t(GROUP_TITLES[item.group]) : '', items: [] }
      out.push(g)
    }
    g.items.push({ ...item, index })
  })
  return out
})

const place = () => {
  const el = rootRef.value
  if (!el) return
  const r = el.getBoundingClientRect()
  panelStyle.value = {
    position: 'fixed',
    top: `${r.bottom + 4}px`,
    left: `${r.left}px`,
    width: `${Math.max(r.width, 320)}px`,
    zIndex: '2300',
  }
}

const onOutside = (e: MouseEvent) => {
  const t = e.target as Node
  if (rootRef.value?.contains(t) || panelRef.value?.contains(t)) return
  close()
}

const openPanel = () => {
  place()
  open.value = true
  highlight.value = 0
  nextTick(() => {
    place()
    document.addEventListener('mousedown', onOutside, true)
    window.addEventListener('resize', place)
  })
}

const close = () => {
  if (!open.value) return
  open.value = false
  document.removeEventListener('mousedown', onOutside, true)
  window.removeEventListener('resize', place)
}

const onFocus = () => {
  focused.value = true
  openPanel()
}

const clearQuery = () => {
  query.value = ''
  highlight.value = 0
  inputRef.value?.focus()
}

const jump = (to: RouteLocationRaw) => {
  void router.push(to)
  query.value = ''
  close()
  inputRef.value?.blur()
  focused.value = false
}

const moveHighlight = (dir: 1 | -1) => {
  const n = results.value.length
  if (!n) return
  highlight.value = (highlight.value + dir + n) % n
}

const onKey = (e: KeyboardEvent) => {
  if (e.key === 'Escape') {
    e.preventDefault()
    close()
    inputRef.value?.blur()
    focused.value = false
    return
  }
  if (!open.value) openPanel()
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    moveHighlight(1)
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    moveHighlight(-1)
  } else if (e.key === 'Enter') {
    e.preventDefault()
    const item = results.value[highlight.value]
    if (item) jump(item.to)
  }
}

watch(query, (value) => {
  if (!open.value) openPanel()
  highlight.value = 0
  if (searchTimer) clearTimeout(searchTimer)
  const q = value.trim()
  if (!q) {
    searchSeq += 1
    remoteItems.value = []
    searching.value = false
    return
  }
  searchTimer = setTimeout(() => void searchRemote(q), 250)
})

const onGlobalKey = (e: KeyboardEvent) => {
  if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'K')) {
    e.preventDefault()
    inputRef.value?.focus()
    openPanel()
  }
}

onMounted(() => window.addEventListener('keydown', onGlobalKey))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onGlobalKey)
  close()
})
</script>

<style scoped lang="scss">
.cmd-search {
  position: relative;
}
.search-box {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-width: 220px;
  height: 44px;
  padding: 0 14px;
  border: 1.5px solid var(--el-border-color-light);
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  transition:
    border-color 0.15s,
    box-shadow 0.15s;
}
.search-box.is-focused,
.search-box.is-open {
  border-color: var(--m-p);
  color: var(--el-text-color-primary);
}
.search-box__icon {
  width: 20px;
  height: 20px;
  flex-shrink: 0;
}
.search-box__input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  font-size: 0.875rem;
  font-family: inherit;
  color: var(--el-text-color-primary);
}
.search-box__input::placeholder {
  color: var(--el-text-color-secondary);
}
/* 去掉部分浏览器 search 输入的原生清除按钮，避免双 × */
.search-box__input::-webkit-search-cancel-button {
  -webkit-appearance: none;
  appearance: none;
}
.search-box__kbd {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
  border-radius: 8px;
  padding: 3px 8px;
  font-family: inherit;
  flex-shrink: 0;
}
.search-box__clear {
  border: none;
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  font-size: 16px;
  line-height: 1;
  padding: 0 2px;
  flex-shrink: 0;
}
</style>

<style lang="scss">
/* teleport 到 body 的结果面板 */
.cmd-panel {
  background: var(--el-bg-color-overlay);
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius);
  box-shadow: var(--app-shadow-lg);
  overflow: hidden;
}
.cmd-panel__list {
  max-height: 20rem;
  overflow-y: auto;
  overscroll-behavior: contain;
  padding: 6px;
}
.cmd-panel__item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  cursor: pointer;
  text-align: left;
  transition: background 0.15s;
  font-family: inherit;
}
.cmd-panel__item:hover,
.cmd-panel__item.is-hl {
  background: var(--el-fill-color-light);
}
.cmd-panel__group {
  padding: 8px 10px 4px;
  font-size: 0.6875rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.cmd-panel__item-icon {
  display: flex;
  width: 18px;
  height: 18px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}
.cmd-panel__item-icon svg {
  width: 18px;
  height: 18px;
}
.cmd-panel__item-title {
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}
.cmd-panel__item-path {
  margin-left: auto;
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
}
.cmd-panel__empty {
  padding: 1.5rem;
  text-align: center;
  font-size: 0.8125rem;
  color: var(--el-text-color-placeholder);
}
</style>
