<!-- GM 运营 · 卡池（?tab=gasha）：配置域 gasha（data/Gasha.json）的表单编辑。
     左边是卡池列表（状态按开始 / 结束时间与服务器时间计算），右边是表单（基本信息、UP、概率、按钮、交换点）、
     模拟抽卡与发布前检查；底部草稿条自动保存草稿、查看改动、放弃、发布（发布前服务端校验，必须填原因）。 -->
<template>
  <div class="gk game-skin">
    <PageHeader :title="t('gmConfig.gasha.title')" :back="false" :theme-toggle="false">
      <template #description>{{ countLine }}</template>
      <template #actions>
        <UButton
          class="gmc-btn"
          color="neutral"
          variant="outline"
          icon="i-lucide-history"
          :aria-label="t('gmConfig.revisions.title')"
          :disabled="!state.info.value"
          @click="revisionsOpen = true"
        >
          <template v-if="!menuStore.isMobile">{{ t('gmConfig.revisions.title') }}</template>
        </UButton>
        <UButton class="gmc-btn" icon="i-lucide-plus" :aria-label="t('gmConfig.gasha.create')" :disabled="!cfg?.gashas.length" @click="openCreate()">
          <template v-if="!menuStore.isMobile">{{ t('gmConfig.gasha.create') }}</template>
        </UButton>
      </template>
    </PageHeader>

    <EmptyState
      v-if="unsupported"
      icon="HOutline:PuzzlePieceIcon"
      :title="t('gmConfig.common.unsupportedTitle')"
      :description="t('gmConfig.common.unsupportedDesc', { game: ctx.gameName.value })"
    />
    <EmptyState
      v-else-if="state.error.value && !cfg"
      icon="HOutline:ExclamationTriangleIcon"
      :title="t('gmConfig.common.loadFailed')"
      :description="state.error.value"
    >
      <template #action>
        <UButton class="gmc-btn" color="neutral" variant="outline" icon="i-lucide-refresh-cw" :loading="state.loading.value" @click="state.load()">
          {{ t('gmConfig.common.retry') }}
        </UButton>
      </template>
    </EmptyState>
    <div v-else-if="!cfg" class="gk__loading">{{ t('gmConfig.common.loading') }}</div>
    <template v-else>
      <div class="gk__body">
        <section class="gk__list" :aria-label="t('gmConfig.gasha.listLabel')">
          <div class="gmc-chips" role="tablist" :aria-label="t('gmConfig.common.filterStatus')">
            <button
              v-for="f in filters"
              :key="f.key"
              type="button"
              role="tab"
              class="gmc-chip"
              :class="{ 'is-on': filter === f.key }"
              :aria-selected="filter === f.key"
              @click="filter = f.key"
            >
              {{ f.label }} <span class="gmc-chip__n m-num">{{ f.n }}</span>
            </button>
          </div>
          <button
            v-for="row in rows"
            :key="row.g.id"
            type="button"
            class="gk__row"
            :class="{ 'is-on': row.g.id === selectedId }"
            :aria-pressed="row.g.id === selectedId"
            @click="select(row.g.id)"
          >
            <GameImage v-if="row.g.bannerAssetId" :src="gakumas.gashaBanner(row.g.bannerAssetId)" class="gk__thumb" width="120" height="60" />
            <span v-else class="gk__thumb gk__thumb--empty"><UIcon name="i-lucide-image" /></span>
            <span class="gk__row-text">
              <span class="gk__row-name">{{ labels.gasha(row.g) }}</span>
              <span class="gk__row-meta">
                <StatusPill :variant="statusVariant(row.status)" :label="t(`gmConfig.gasha.status.${row.status}`)" />
                <span class="m-num gk__period">{{ row.period }}</span>
                <span v-if="row.isNew" class="gmc-tag gmc-tag--g">{{ t('gmConfig.common.new') }}</span>
                <span v-else-if="row.changed" class="gmc-tag gmc-tag--warn">{{ t('gmConfig.common.changed') }}</span>
                <span v-if="row.bad" class="gmc-tag gmc-tag--bad">{{ t('gmConfig.gasha.needsCheck') }}</span>
              </span>
            </span>
          </button>
          <EmptyState v-if="!rows.length" icon="HOutline:SparklesIcon" :title="t('gmConfig.gasha.emptyFilter')" />
        </section>

        <section ref="mainRef" class="gk__main">
          <EmptyState v-if="!gasha" icon="HOutline:SparklesIcon" :title="t('gmConfig.gasha.pick')" />
          <template v-else>
            <div class="gk__editor">
              <div class="gk__editor-menu">
                <ActionMenu :items="menuItems" @select="onMenu" />
              </div>
              <GashaBasics />
              <GashaUps />
              <GashaRates />
              <GashaButtons />
              <GashaPoint />
            </div>
            <div class="gk__side">
              <GashaSimulator class="gk__side-item" />
              <GashaChecks class="gk__side-item" />
            </div>
          </template>
          <ConfigDraftBar :state="state" :zone="clock.zone.value" :blockers="blockers" :group-label="groupLabel" :ref-label="refLabel" />
        </section>
      </div>
    </template>

    <FormDialog v-model="createOpen" :title="t('gmConfig.gasha.create')" :width="520" :show-footer="false">
      <div class="gk__dialog game-skin" :data-game="ctx.game.value || undefined">
        <label class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.gasha.newName') }}</span>
          <input v-model.trim="createName" class="gmc-input" :class="{ 'is-invalid': !createName }" />
          <span class="gmc-hint">{{ t('gmConfig.gasha.newNameHint') }}</span>
        </label>
        <div class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.gasha.template') }}</span>
          <AppSelect v-model="createFrom" :options="templateOptions" searchable />
          <span class="gmc-hint">{{ t('gmConfig.gasha.templateHint') }}</span>
        </div>
        <div class="gk__dialog-actions">
          <UButton class="gmc-btn" color="neutral" variant="ghost" @click="createOpen = false">{{ t('gmConfig.common.cancel') }}</UButton>
          <UButton class="gmc-btn" :disabled="!createName || !createFrom" @click="create">{{ t('gmConfig.common.ok') }}</UButton>
        </div>
      </div>
    </FormDialog>

    <ConfigRevisionsDrawer v-model="revisionsOpen" :state="state" :zone="clock.zone.value" :group-label="groupLabel" :ref-label="refLabel" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute } from 'vue-router'
import { gakumas } from '@/games'
import type { GashaConfig, GashaDef } from '@/games/gakumas/types'
import { Dialog } from '@/utils/dialog'
import { useGmContext } from '../../context'
import GashaBasics from './GashaBasics.vue'
import GashaButtons from './GashaButtons.vue'
import GashaChecks from './GashaChecks.vue'
import GashaPoint from './GashaPoint.vue'
import GashaRates from './GashaRates.vue'
import GashaSimulator from './GashaSimulator.vue'
import GashaUps from './GashaUps.vue'
import { provideGashaEditor } from './editor'
import { useGashaLabels } from './labels'
import {
  catalogKindOf,
  freeId,
  GASHA_STATUSES,
  gashaChecks,
  gashaStatus,
  rarityInPools,
  statusVariant,
  type GashaStatus,
} from './gasha'
import { useCatalogLookup } from './shared/catalogLookup'
import ConfigDraftBar from './shared/ConfigDraftBar.vue'
import ConfigRevisionsDrawer from './shared/ConfigRevisionsDrawer.vue'
import { deepEqual } from './shared/diff'
import { useServerClock } from './shared/serverClock'
import { formatDay } from './shared/time'
import { useConfigDomain } from './shared/useConfigDomain'

defineOptions({ name: 'GmGashaTab' })

const DOMAIN = 'gasha'

const { t, te } = useI18n()
const route = useRoute()
const menuStore = useMenuStore()
const ctx = useGmContext()
const clock = useServerClock(ctx)
const lookup = useCatalogLookup(ctx)
const labels = useGashaLabels()
const state = useConfigDomain<GashaConfig>(DOMAIN)

const revisionsOpen = ref(false)
const unsupported = computed(() => !!ctx.game.value && ctx.game.value !== 'gakumas')
const cfg = computed(() => state.working.value)
const selectedId = ref('')
const gasha = computed(() => cfg.value?.gashas.find((g) => g.id === selectedId.value))
const publishedGasha = computed(() => state.published.value?.gashas.find((g) => g.id === selectedId.value))

const rarityOf = (resourceType: number, id: string) => {
  const c = cfg.value
  return (c && rarityInPools(c, resourceType, id)) ?? lookup.lookup(catalogKindOf(resourceType), id)?.rarity
}
const cardStatus = (resourceType: number, id: string) => lookup.status(catalogKindOf(resourceType), id)

provideGashaEditor({
  state,
  cfg,
  gasha,
  publishedGasha,
  zone: clock.zone,
  now: clock.now,
  game: ctx.game,
  lookup,
  rarityOf,
  cardStatus,
})

onMounted(async () => {
  if (unsupported.value) return
  await state.load()
  applyRouteGasha()
})

// 卡池按显示顺序（order，相同时按 ID）
const ordered = computed(() =>
  [...(cfg.value?.gashas || [])].sort((a, b) => (a.order || 0) - (b.order || 0) || a.id.localeCompare(b.id)),
)

// ?gasha=<id> 打开某个卡池（运营日历跳过来）
const applyRouteGasha = () => {
  const wanted = route.query.gasha
  if (typeof wanted === 'string' && cfg.value?.gashas.some((g) => g.id === wanted)) selectedId.value = wanted
}
watch(() => route.query.gasha, applyRouteGasha)
// 选中的卡池不存在了（删除、重新加载）时选第一个
watch(
  ordered,
  (list) => {
    if (!list.some((g) => g.id === selectedId.value)) selectedId.value = list[0]?.id || ''
  },
  { immediate: true },
)

const filter = ref<'all' | GashaStatus>('all')
const statusOf = (g: GashaDef) => gashaStatus(g, clock.now.value)

const filters = computed(() => {
  const list = ordered.value
  return [
    { key: 'all' as const, label: t('gmConfig.common.all'), n: list.length },
    ...GASHA_STATUSES.map((s) => ({ key: s, label: t(`gmConfig.gasha.status.${s}`), n: list.filter((g) => statusOf(g) === s).length })),
  ]
})

const localBad = (g: GashaDef) => {
  const c = cfg.value
  return !!c && gashaChecks(c, g, { rarityOf }).some((x) => x.level === 'bad')
}

const periodOf = (g: GashaDef, zone: string) =>
  g.endTime
    ? `${formatDay(g.startTime, zone)}–${formatDay(g.endTime, zone)}`
    : t('gmConfig.gasha.permanentFrom', { day: formatDay(g.startTime, zone) })

const rows = computed(() => {
  const zone = clock.zone.value
  const published = state.published.value?.gashas || []
  return ordered.value
    .filter((g) => filter.value === 'all' || statusOf(g) === filter.value)
    .map((g) => {
      const before = published.find((p) => p.id === g.id)
      return {
        g,
        status: statusOf(g),
        period: periodOf(g, zone),
        isNew: !before,
        changed: !!before && !deepEqual(before, g),
        bad: localBad(g),
      }
    })
})

const countLine = computed(() => {
  const list = ordered.value
  const parts = [
    t('gmConfig.gasha.count', { n: list.length }),
    t('gmConfig.gasha.countOpen', { n: list.filter((g) => { const s = statusOf(g); return s === 'open' || s === 'permanent' }).length }),
  ]
  if (state.dirty.value) parts.push(t('gmConfig.gasha.countDraft', { n: state.changes.value.length }))
  return parts.join(' · ')
})

const mainRef = ref<HTMLElement>()
const select = (id: string) => {
  selectedId.value = id
  if (menuStore.isMobile) void nextTick(() => mainRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' }))
}

// 阻止发布的问题：所有卡池的本地规则检查（整份文件一起发布）
const blockers = computed(() => {
  const c = cfg.value
  if (!c) return []
  const out: string[] = []
  for (const g of c.gashas) {
    for (const x of gashaChecks(c, g, { rarityOf })) {
      if (x.level === 'bad') out.push(`${labels.gasha(g)}：${labels.problem(x, c, g)}`)
    }
  }
  return out.length > 6 ? [...out.slice(0, 6), t('gmConfig.common.andMore', { n: out.length - 6 })] : out
})

const groupLabel = (path: string[]) => {
  const [section, id] = path
  if (!id) return section && te(`gmConfig.gasha.sections.${section}`) ? t(`gmConfig.gasha.sections.${section}`) : section || ''
  const c = cfg.value
  const p = state.published.value
  const title = te(`gmConfig.gasha.sections.${section}`) ? t(`gmConfig.gasha.sections.${section}`) : ''
  if (section === 'gashas') {
    const g = c?.gashas.find((x) => x.id === id) || p?.gashas.find((x) => x.id === id)
    return `${title} · ${labels.gasha(g)}`
  }
  if (section === 'points') {
    const x = c?.points.find((v) => v.id === id) || p?.points.find((v) => v.id === id)
    return `${title} · ${x?.name || t('game.unnamed')}`
  }
  if (section === 'lotteries') {
    const src = c?.lotteries.some((l) => l.id === id) ? c : p
    return src ? `${title} · ${labels.lottery(src, id)}` : title
  }
  if (section === 'pools') {
    const src = c?.pools.some((l) => l.id === id) ? c : p
    const pool = src?.pools.find((x) => x.id === id)
    return src && pool ? `${title} · ${labels.pool(src, pool)}` : title
  }
  return title
}

const refLabel = (collection: string, id: string) => labels.ref([cfg.value, state.published.value], collection, id)

// ---- 新建 / 复制 / 删除 ----
// 新卡池只填名称；配置里的卡池 ID 由模板的 ID 自动生成（不重复），界面上不出现。
const createOpen = ref(false)
const createName = ref('')
const createFrom = ref('')
const templateOptions = computed(() =>
  ordered.value.map((g) => ({ label: labels.gasha(g), value: g.id, description: periodOf(g, clock.zone.value) })),
)
const openCreate = (from?: string) => {
  createFrom.value = typeof from === 'string' && from ? from : selectedId.value || ordered.value[0]?.id || ''
  const src = cfg.value?.gashas.find((g) => g.id === createFrom.value)
  createName.value = src?.name ? t('gmConfig.gasha.copyName', { name: src.name }) : ''
  createOpen.value = true
}
const create = () => {
  const c = cfg.value
  const src = c?.gashas.find((g) => g.id === createFrom.value)
  if (!c || !src || !createName.value) return
  const copy = JSON.parse(JSON.stringify(src)) as GashaDef
  copy.id = freeId((v) => c.gashas.some((g) => g.id === v), `${src.id}-copy`)
  copy.name = createName.value
  // 交换点按卡池各自计数，复制出来的卡池不沿用模板的交换点
  copy.gashaPointId = ''
  c.gashas.splice(c.gashas.indexOf(src) + 1, 0, copy)
  createOpen.value = false
  filter.value = 'all'
  select(copy.id)
}

const menuItems = computed(() => [
  { key: 'copy', label: t('gmConfig.gasha.copy'), icon: 'i-lucide-copy' },
  { key: 'delete', label: t('gmConfig.gasha.delete'), icon: 'i-lucide-trash-2', danger: true },
])
const onMenu = (key: string) => {
  const g = gasha.value
  if (!g) return
  if (key === 'copy') openCreate(g.id)
  if (key === 'delete') {
    Dialog.confirm({
      title: t('gmConfig.gasha.deleteTitle', { name: labels.gasha(g) }),
      content: t('gmConfig.gasha.deleteContent'),
      cancelText: t('gmConfig.common.cancel'),
      confirmText: t('gmConfig.gasha.delete'),
      onConfirm: () => {
        const c = cfg.value
        if (!c) return
        const i = c.gashas.indexOf(g)
        if (i >= 0) c.gashas.splice(i, 1)
        selectedId.value = ordered.value[0]?.id || ''
      },
    })
  }
}
</script>

<style lang="scss" src="./shared/form.scss"></style>
<style scoped lang="scss">
.gk {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 0;
}
.gk__loading {
  padding: 40px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}
.gk__body {
  display: flex;
  flex-wrap: wrap;
  gap: 18px;
  align-items: flex-start;
}
.gk__list {
  flex: 1 1 340px;
  max-width: 420px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 12px;
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.gk__list > .gmc-chips {
  padding: 2px 2px 6px;
}
.gk__row {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  padding: 10px;
  border-radius: 20px;
  border: 2px solid transparent;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gk__row:hover {
  background: var(--m-s2);
}
.gk__row.is-on {
  border-color: var(--m-g, var(--m-p));
  background: var(--m-gSoft, var(--m-pSoft));
}
.gk__row .gk__thumb {
  flex: none;
  width: 120px;
  height: 60px;
  object-fit: contain;
  border-radius: 12px;
  background: transparent;
}
.gk__thumb--empty {
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--m-s2) !important;
  color: var(--el-text-color-secondary);
}
.gk__row-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.gk__row-name {
  font-size: 13px;
  font-weight: 800;
  line-height: 1.35;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.gk__row-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  align-items: center;
}
.gk__period {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gk__main {
  flex: 2 1 620px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gk__editor {
  position: relative;
  padding: 20px 24px 4px;
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  min-width: 0;
}
.gk__editor-menu {
  position: absolute;
  top: 14px;
  right: 14px;
  z-index: 1;
}
.gk__side {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: stretch;
}
.gk__side-item {
  flex: 1 1 320px;
  min-width: 0;
}
.gk__dialog {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.gk__dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
@media (max-width: 991px) {
  .gk__list {
    max-width: none;
  }
  .gk__editor {
    padding: 16px 14px 4px;
  }
  .gk__row .gk__thumb {
    width: 96px;
    height: 48px;
  }
}
</style>
