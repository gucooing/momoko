<!-- 游戏服务器：标题 + 统计（服务器、游戏、运行中、在线玩家）+ 适配器 / 接入服务器；
     接入方式 / 游戏 / 状态 chips（带数量，后端按其他筛选统计）+ 标签 + 搜索；卡片流（分页，几百台也只渲染一页）；
     内置的游戏适配器在弹窗里查看（能力由已实现的驱动推出）。 -->
<template>
  <div class="gsl-page">
    <PageHeader :title="t('gameServers.list.title')">
      <template #description>
        {{ t('gameServers.list.summary', { total: summary.total, games: summary.games }) }} ·
        <span class="gsl-page__running">{{ t('gameServers.list.running', { n: summary.running }) }}</span> ·
        {{ t('gameServers.list.online') }} <b class="m-num">{{ summary.online }}</b>
      </template>
      <template #actions>
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-lucide-rotate-cw"
          class="gsl-ico"
          :aria-label="t('gameServers.common.refresh')"
          :title="t('gameServers.common.refresh')"
          :loading="loading"
          @click="load()"
        />
        <UButton color="neutral" variant="outline" icon="i-lucide-plug" @click="adaptersOpen = true">
          {{ t('gameServers.list.adapters') }}
        </UButton>
        <UButton color="primary" icon="i-lucide-plus" to="/game/servers/new">
          {{ t('gameServers.list.create') }}
        </UButton>
      </template>
    </PageHeader>

    <div class="gsl-tools">
      <div class="gsl-chips" role="tablist" :aria-label="t('gameServers.list.modes')">
        <button
          v-for="c in modeChips"
          :key="c.value"
          type="button"
          role="tab"
          class="gsl-chip"
          :class="{ 'is-on': query.mode === c.value }"
          :aria-selected="query.mode === c.value"
          @click="setFilter('mode', c.value)"
        >
          {{ c.label }} <span class="gsl-chip__n m-num">{{ c.count }}</span>
        </button>
      </div>
      <span v-if="gameChips.length > 2" class="gsl-sep" aria-hidden="true" />
      <div v-if="gameChips.length > 2" class="gsl-chips" role="tablist" :aria-label="t('gameServers.list.games')">
        <button
          v-for="c in gameChips"
          :key="c.value"
          type="button"
          role="tab"
          class="gsl-chip"
          :class="{ 'is-on': query.game === c.value }"
          :aria-selected="query.game === c.value"
          @click="setFilter('game', c.value)"
        >
          {{ c.label }} <span class="gsl-chip__n m-num">{{ c.count }}</span>
        </button>
      </div>
      <span class="gsl-sep" aria-hidden="true" />
      <div class="gsl-chips" role="tablist" :aria-label="t('gameServers.list.states')">
        <button
          v-for="c in stateChips"
          :key="c.value"
          type="button"
          role="tab"
          class="gsl-chip"
          :class="{ 'is-on': query.state === c.value }"
          :aria-selected="query.state === c.value"
          @click="setFilter('state', c.value)"
        >
          {{ c.label }} <span class="gsl-chip__n m-num">{{ c.count }}</span>
        </button>
      </div>
      <span class="gsl-tools__spacer" />
      <div v-if="tagOptions.length > 1" class="gsl-tag">
        <AppSelect v-model="query.tag" :options="tagOptions" searchable @update:model-value="onFilterChange" />
      </div>
      <label class="gsl-search">
        <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" />
        <input
          v-model="keyword"
          type="search"
          :placeholder="t('gameServers.list.search')"
          :aria-label="t('gameServers.list.search')"
          @keyup.enter="applySearch"
          @search="applySearch"
        />
      </label>
    </div>

    <div v-if="loading && !items.length" class="gsl-grid">
      <div v-for="i in 3" :key="i" class="gsl-skeleton" />
    </div>
    <EmptyState
      v-else-if="!items.length && summary.total === 0"
      icon="HOutline:ServerStackIcon"
      :title="t('gameServers.list.emptyTitle')"
      :description="t('gameServers.list.emptyDesc')"
    >
      <template #action>
        <UButton color="primary" icon="i-lucide-plus" to="/game/servers/new">{{ t('gameServers.list.create') }}</UButton>
      </template>
    </EmptyState>
    <EmptyState
      v-else-if="!items.length"
      icon="HOutline:MagnifyingGlassIcon"
      :title="t('gameServers.list.noMatchTitle')"
      :description="t('gameServers.list.noMatchDesc')"
    >
      <template #action>
        <UButton color="neutral" variant="soft" @click="clearFilters">{{ t('gameServers.list.clearFilters') }}</UButton>
      </template>
    </EmptyState>
    <template v-else>
      <div class="gsl-grid">
        <ServerCard
          v-for="s in items"
          :key="s.brief?.id"
          :server="s"
          :can-gm="canGm(s)"
          :busy="pendingId === s.brief?.id"
          @control="(a) => control(s, a)"
          @menu="(k) => onMenu(s, k)"
        />
      </div>
      <Pagination
        v-if="total > pageSize"
        v-model:page="page"
        v-model:page-size="pageSize"
        :total="total"
        :page-sizes="[12, 24, 48, 96]"
        @change="load()"
      />
    </template>

    <AdaptersDialog v-model="adaptersOpen" :adapters="adapters" :loaded="adaptersLoaded" />
    <ServerEditDialog v-model="editOpen" :server="active" @saved="load()" />
    <ServerDeleteDialog v-model="deleteOpen" :server="active" @deleted="load()" />
    <RotateKeyDialog v-model="rotateOpen" :server="active" @done="load()" />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import '@/games'
import { controlGameServer, listGameAdapters, listGameServers, refreshGameServer } from '@/api/gameServer'
import { getRequestErrorMessage } from '@/utils/request'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import type { GameAdapter, GameServerInfo, GameServerTagCount, GameServersSummary } from '@/types/v1/game_server'
import ServerCard from './components/ServerCard.vue'
import ServerEditDialog from './components/ServerEditDialog.vue'
import ServerDeleteDialog from './components/ServerDeleteDialog.vue'
import RotateKeyDialog from './components/RotateKeyDialog.vue'
import AdaptersDialog from './components/AdaptersDialog.vue'

defineOptions({ name: 'GameServersView' })

const { t, te } = useI18n()
const router = useRouter()
const menuStore = useMenuStore()
const fb = useFeedback()
const accessible = useAccessibleRoutes()

const query = reactive({ mode: '', game: '', state: '', tag: '', q: '' })
const keyword = ref('')
const page = ref(1)
const pageSize = ref(24)
const total = ref(0)
const items = ref<GameServerInfo[]>([])
const counts = ref<Record<string, number>>({})
const tags = ref<GameServerTagCount[]>([])
const summary = ref<GameServersSummary>({ total: 0, games: 0, running: 0, online: 0 })
const loading = ref(false)
const adapters = ref<GameAdapter[]>([])
const adaptersLoaded = ref(false)
const pendingId = ref('')
const adaptersOpen = ref(false)

const adapterName = (id: string) => (te(`game.adapters.${id}`) ? t(`game.adapters.${id}`) : t('game.unnamed'))

const load = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    const { data } = await listGameServers({ ...query, page: page.value, pageSize: pageSize.value })
    items.value = data.items ?? []
    total.value = data.total
    counts.value = data.counts ?? {}
    tags.value = data.tags ?? []
    if (data.summary) summary.value = data.summary
    if (!items.value.length && page.value > 1 && data.total > 0) {
      page.value = 1
      await load(silent)
    }
  } catch (e) {
    if (!silent) fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    loading.value = false
  }
}

const loadAdapters = async () => {
  try {
    const { data } = await listGameAdapters()
    adapters.value = data.items ?? []
  } finally {
    adaptersLoaded.value = true
  }
}

const count = (key: string) => counts.value[key] ?? 0
const modeChips = computed(() => [
  { value: '', label: t('gameServers.list.all'), count: count('mode:all') },
  { value: 'hosted', label: t('gameServers.mode.hosted'), count: count('mode:hosted') },
  { value: 'remote', label: t('gameServers.mode.remote'), count: count('mode:remote') },
])
const gameChips = computed(() => {
  const ids = new Set<string>(adapters.value.map((a) => a.id))
  for (const k of Object.keys(counts.value)) if (k.startsWith('game:') && k !== 'game:all') ids.add(k.slice(5))
  return [
    { value: '', label: t('gameServers.list.allGames'), count: count('game:all') },
    ...[...ids].map((id) => ({ value: id, label: adapterName(id), count: count(`game:${id}`) })),
  ]
})
const STATES = ['running', 'stopped', 'maintenance', 'unreachable', 'starting', 'stopping', 'restarting', 'unknown']
const stateChips = computed(() => [
  { value: '', label: t('gameServers.list.allStates'), count: count('state:all') },
  ...STATES.filter((s) => count(`state:${s}`) > 0 || query.state === s).map((s) => ({
    value: s,
    label: t(`gameServers.state.${s}`),
    count: count(`state:${s}`),
  })),
])
const tagOptions = computed(() => [
  { value: '', label: t('gameServers.list.allTags') },
  ...tags.value.map((x) => ({ value: x.tag, label: `${x.tag} · ${x.count}` })),
])

const onFilterChange = () => {
  page.value = 1
  load()
}
const setFilter = (key: 'mode' | 'game' | 'state', value: string) => {
  query[key] = value
  onFilterChange()
}
const applySearch = () => {
  query.q = keyword.value.trim()
  onFilterChange()
}
const clearFilters = () => {
  Object.assign(query, { mode: '', game: '', state: '', tag: '', q: '' })
  keyword.value = ''
  onFilterChange()
}

const canGm = (s: GameServerInfo) => accessible.value.has('/game/gm') && (s.capabilities || []).includes('gm')

const control = async (s: GameServerInfo, action: 'start' | 'stop' | 'restart') => {
  const id = s.brief?.id
  if (!id) return
  pendingId.value = id
  try {
    await controlGameServer(id, action)
    fb.success(t(action === 'start' ? 'gameServers.actions.started' : action === 'stop' ? 'gameServers.actions.stopping' : 'gameServers.actions.restarting'))
    await load(true)
  } catch (e) {
    fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
  } finally {
    pendingId.value = ''
  }
}

const active = ref<GameServerInfo | null>(null)
const editOpen = ref(false)
const deleteOpen = ref(false)
const rotateOpen = ref(false)
const onMenu = async (s: GameServerInfo, key: string) => {
  active.value = s
  const id = s.brief?.id || ''
  switch (key) {
    case 'edit':
      editOpen.value = true
      break
    case 'delete':
      deleteOpen.value = true
      break
    case 'rotate':
      rotateOpen.value = true
      break
    case 'restart':
      await control(s, 'restart')
      break
    case 'switch':
      router.push({ path: `/game/servers/${encodeURIComponent(id)}`, query: { tab: 'connection' } })
      break
    case 'refresh':
      try {
        await refreshGameServer(id)
        fb.success(t('gameServers.actions.refreshed'))
        await load(true)
      } catch (e) {
        fb.error(getRequestErrorMessage(e, t('gameServers.common.failed')))
      }
      break
  }
}

// 状态由后端每 30 秒探测；列表每 20 秒静默刷新一次
let timer: ReturnType<typeof setInterval> | undefined
onMounted(() => {
  load()
  loadAdapters()
  timer = setInterval(() => {
    if (document.visibilityState === 'visible') load(true)
  }, 20000)
})
onBeforeUnmount(() => clearInterval(timer))
</script>

<style scoped lang="scss">
.gsl-page {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.gsl-page__running {
  font-weight: 700;
  color: var(--m-ok);
}
.gsl-page .gsl-ico {
  width: 40px;
  padding-inline: 0;
  justify-content: center;
}
.gsl-tools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.gsl-tools__spacer {
  flex: 1;
}
.gsl-sep {
  width: 1px;
  height: 28px;
  background: var(--el-border-color-light);
}
.gsl-chips {
  display: flex;
  gap: 8px;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.gsl-chips::-webkit-scrollbar {
  display: none;
}
.gsl-chip {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}
.gsl-chip:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.gsl-chip.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.gsl-chip__n {
  font-size: 0.75rem;
  opacity: 0.7;
}
.gsl-tag {
  width: 180px;
}
.gsl-tag :deep(.app-sel__trigger) {
  height: 40px;
  border-radius: 999px;
  padding-inline: 14px 10px;
}
.gsl-search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 280px;
  min-width: 180px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
}
.gsl-search:focus-within {
  border-color: var(--m-p);
}
.gsl-search :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
}
.gsl-search input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
}
.gsl-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;
}
@media (width >= 560px) {
  .gsl-grid {
    grid-template-columns: repeat(auto-fill, minmax(330px, 1fr));
  }
}
.gsl-skeleton {
  height: 330px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
}
@media (width <= 768px) {
  .gsl-page .gsl-ico {
    width: 44px;
  }
  .gsl-sep {
    display: none;
  }
  .gsl-search,
  .gsl-tag {
    flex: 1 1 100%;
    width: auto;
  }
  .gsl-chip {
    min-height: 44px;
  }
}
</style>
