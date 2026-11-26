<!-- GM 运营 · 运营管理（?tab=ops）：一张运营日历，卡池、活动、竞赛赛季、登录奖励、公告各一条泳道（点泳道名收起 / 展开，记在本机）。
     点日历上的条目直接打开它的编辑窗口；改动直接进草稿、日历立即按草稿显示，页面底部的发布条一起查看改动与发布。
     ?item=<种类>:<ID> 打开某一项的编辑窗口，?new=notice 新建公告（工作台跳过来）。 -->
<template>
  <div class="gcal game-skin">
    <PageHeader :title="t('gm.tabs.ops')">
      <template #description>{{ t('gmConfig.ops.desc', { zone: zoneLabel }) }}</template>
      <template #actions>
        <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('gmConfig.calendar.reload')" :box="32" @click="reload" />
        <ActionMenu :items="moreItems" :label="t('gmConfig.ops.more')" @select="onMore" />
        <AppDropdown align="end" :width="200">
          <template #trigger>
            <UButton class="gmc-btn" icon="i-lucide-plus" :disabled="unsupported">{{ t('gmConfig.ops.create') }}</UButton>
          </template>
          <template #default="{ close }">
            <div class="gcal__menu">
              <button v-for="c in createItems" :key="c.key" type="button" class="gcal__menu-item" :disabled="!c.ready" @click="create(c.key, close)">
                <UIcon :name="c.icon" class="gcal__menu-icon" />
                {{ c.label }}
              </button>
            </div>
          </template>
        </AppDropdown>
      </template>
    </PageHeader>

    <EmptyState
      v-if="unsupported"
      icon="HOutline:PuzzlePieceIcon"
      :title="t('gmConfig.common.unsupportedTitle')"
      :description="t('gmConfig.common.unsupportedDesc', { game: ctx.gameName.value })"
    />
    <div v-else-if="loading && !loaded" class="gcal__loading">{{ t('gmConfig.common.loading') }}</div>
    <template v-else>
      <div class="gcal__toolbar">
        <div class="gcal__month" role="group" :aria-label="t('gmConfig.calendar.month')">
          <AppIconButton icon="HOutline:ChevronLeftIcon" :label="t('gmConfig.calendar.prev')" :box="32" @click="shift(-1)" />
          <button type="button" class="gcal__month-label m-num" :title="t('gmConfig.calendar.thisMonth')" @click="resetMonth">
            {{ t('gmConfig.calendar.monthLabel', { y: year, m: month }) }}
          </button>
          <AppIconButton icon="HOutline:ChevronRightIcon" :label="t('gmConfig.calendar.next')" :box="32" @click="shift(1)" />
        </div>
        <span class="gmc-hint">{{ t('gmConfig.ops.boardHint') }}</span>
      </div>

      <!-- PC：月度时间线 -->
      <section v-if="!menuStore.isMobile" class="gcal__board">
        <div class="gcal__inner" :style="{ '--days': days.length }">
          <div class="gcal__row gcal__row--head">
            <span class="gcal__label gcal__label--month">{{ t('gmConfig.calendar.monthShort', { m: month }) }}</span>
            <div class="gcal__days">
              <span v-for="d in days" :key="d.day" class="gcal__day" :class="{ 'is-weekend': d.weekend, 'is-today': d.today }">
                <span class="gcal__day-n m-num">{{ d.day }}</span>
                <span class="gcal__day-w">{{ t(`gmConfig.calendar.weekdays.${d.weekday}`) }}</span>
              </span>
            </div>
          </div>

          <div v-for="l in lanes" :key="l.key" class="gcal__lane">
            <template v-if="!collapsed.has(l.key) && l.items.length">
              <div v-for="(item, i) in l.items" :key="item.key" class="gcal__row">
                <span class="gcal__label">
                  <button v-if="i === 0" type="button" class="gcal__toggle" :aria-expanded="true" @click="toggleLane(l.key)">
                    <UIcon name="i-lucide-chevron-down" class="gcal__chev" />
                    <span class="gcal__swatch" :class="`is-${l.key}`" />
                    {{ t(`gmConfig.calendar.lane.${l.key}`) }}
                    <span class="gcal__count m-num">{{ l.items.length }}</span>
                  </button>
                </span>
                <div class="gcal__track">
                  <button
                    type="button"
                    class="gcal__bar"
                    :class="[`is-${l.key}`, { 'is-perm': item.permanent, 'is-draft': !!item.draft, 'is-removed': item.draft === 'removed', 'is-on': itemOpen && openKey === refKey(item.ref) }]"
                    :style="barStyle(item)"
                    :title="barTitle(item)"
                    @click="openItem(item.ref)"
                  >
                    <GameImage v-if="item.image" :src="item.image" class="gcal__bar-img" />
                    <span class="gcal__bar-text">
                      <span v-if="item.draft" class="gcal__draft">{{ t(`gmConfig.ops.draft.${item.draft}`) }}</span>
                      {{ item.title }}
                    </span>
                  </button>
                </div>
              </div>
            </template>
            <!-- 收起时（或这个月没有）：一行，缩略显示条目覆盖的日子 -->
            <div v-else class="gcal__row gcal__row--folded">
              <span class="gcal__label">
                <button type="button" class="gcal__toggle" :aria-expanded="false" :disabled="!l.items.length" @click="toggleLane(l.key)">
                  <UIcon :name="l.items.length ? 'i-lucide-chevron-right' : 'i-lucide-minus'" class="gcal__chev" />
                  <span class="gcal__swatch" :class="`is-${l.key}`" />
                  {{ t(`gmConfig.calendar.lane.${l.key}`) }}
                  <span class="gcal__count m-num">{{ l.items.length }}</span>
                </button>
              </span>
              <div v-if="l.items.length" class="gcal__track gcal__track--mini" role="button" tabindex="0" :aria-label="t('gmConfig.ops.expand')" @click="toggleLane(l.key)" @keydown.enter="toggleLane(l.key)">
                <span v-for="item in l.items" :key="item.key" class="gcal__mini" :class="`is-${l.key}`" :style="barStyle(item)" />
              </div>
              <span v-else class="gcal__none">{{ t('gmConfig.calendar.laneEmpty') }}</span>
            </div>
          </div>

          <div v-if="nowPos !== null" class="gcal__now" :style="{ '--x': nowPos }">
            <span class="gcal__now-tag m-num">{{ nowLabel }}</span>
          </div>
        </div>
      </section>

      <!-- 手机：按泳道列出本月的条目 -->
      <section v-else class="gcal__agenda">
        <div v-for="l in lanes" :key="l.key" class="gcal__agenda-lane">
          <button type="button" class="gcal__agenda-title" :aria-expanded="!collapsed.has(l.key)" @click="toggleLane(l.key)">
            <UIcon :name="collapsed.has(l.key) ? 'i-lucide-chevron-right' : 'i-lucide-chevron-down'" class="gcal__chev" />
            <span class="gcal__swatch" :class="`is-${l.key}`" />
            {{ t(`gmConfig.calendar.lane.${l.key}`) }}
            <span class="gcal__count m-num">{{ l.items.length }}</span>
          </button>
          <template v-if="!collapsed.has(l.key)">
            <button
              v-for="item in l.items"
              :key="item.key"
              type="button"
              class="gcal__agenda-item"
              :class="{ 'is-draft': !!item.draft, 'is-removed': item.draft === 'removed' }"
              @click="openItem(item.ref)"
            >
              <GameImage v-if="item.image" :src="item.image" class="gcal__agenda-img" />
              <span class="gcal__agenda-text">
                <span class="gcal__agenda-name">
                  <span v-if="item.draft" class="gcal__draft">{{ t(`gmConfig.ops.draft.${item.draft}`) }}</span>
                  {{ item.title }}
                </span>
                <span class="gcal__agenda-period m-num">{{ periodOf(item) }}</span>
              </span>
            </button>
            <p v-if="!l.items.length" class="gmc-hint">{{ t('gmConfig.calendar.laneEmpty') }}</p>
          </template>
        </div>
      </section>

      <AppPanel v-if="hints.length" :title="t('gmConfig.calendar.hintsTitle')" title-icon="HOutline:ExclamationTriangleIcon">
        <ul class="gcal__hints">
          <li v-for="(h, i) in hints" :key="i" class="gcal__hint" :class="`is-${h.level}`">
            <UIcon :name="h.level === 'warn' ? 'i-lucide-triangle-alert' : 'i-lucide-info'" class="gcal__hint-icon" />
            <span class="gcal__hint-text">{{ t(`gmConfig.calendar.hints.${h.key}`, h.params) }}</span>
            <UButton v-if="h.ref" class="gmc-btn" size="xs" color="neutral" variant="ghost" @click="openItem(h.ref)">{{ t('gmConfig.calendar.open') }}</UButton>
          </li>
        </ul>
      </AppPanel>

      <PublishBar :zone="zone" />
    </template>

    <!-- 编辑窗口（点条目直接打开） -->
    <template v-if="opened">
      <GashaDialog v-if="opened.kind === 'gasha'" :key="openKey" v-model="itemOpen" :gasha-id="opened.id" @open="openItem" />
      <NoticeDialog v-else-if="opened.kind === 'notice'" :key="openKey" v-model="itemOpen" :notice-id="opened.id" @open="openItem" />
      <SeasonDialog v-else-if="opened.kind === 'pvp'" :key="openKey" v-model="itemOpen" :id="opened.id" @open="openItem" @rules="rulesOpen = true" />
      <LoginDialog v-else-if="opened.kind === 'login'" :key="openKey" v-model="itemOpen" :id="opened.id" @open="openItem" />
      <EventDialog v-else :key="openKey" v-model="itemOpen" :kind="opened.kind === 'research' ? 'research' : 'event'" :id="opened.id" @open="openItem" />
    </template>

    <GashaCreateDialog v-model="gashaCreateOpen" @created="openItem({ kind: 'gasha', id: $event })" />
    <PvpRulesDialog v-model="rulesOpen" />
    <ConfigRevisionsDrawer
      v-if="revisionsDomain"
      v-model="revisionsOpen"
      :state="revisionsDomain.state"
      :zone="zone"
      :group-label="revisionsDomain.groupLabel"
      :ref-label="revisionsDomain.refLabel"
    />
    <TrendDialog v-model="trendOpen" field="draws" :report="dash" :color="chartColor" />
    <span ref="rootEl" class="gcal__color-probe" aria-hidden="true" />
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onMounted, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import type { Dashboard } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import TrendDialog from '../../components/TrendDialog.vue'
import { fetchDashboard, useGameChartColor } from '../../dashboard'
import { computeHints, laneItems, LANES, parseRef, refKey, type CalItem, type Contents, type Hint, type ItemRef, type LaneKey } from './calendar'
import { provideOpsDomains, type OpsDomain } from './domains'
import EventDialog from './events/EventDialog.vue'
import GashaDialog from './gasha/GashaDialog.vue'
import LoginDialog from './login/LoginDialog.vue'
import NoticeDialog from './notices/NoticeDialog.vue'
import SeasonDialog from './pvp/SeasonDialog.vue'
import GashaCreateDialog from './gasha/GashaCreateDialog.vue'
import { addLoginBonus } from './login/login'
import { addNotice } from './notices/notice'
import PublishBar from './PublishBar.vue'
import PvpRulesDialog from './pvp/PvpRulesDialog.vue'
import { addSeason, seasonLabel } from './pvp/pvp'
import ConfigRevisionsDrawer from './shared/ConfigRevisionsDrawer.vue'
import { nextHour } from './shared/entry'
import { useServerClock } from './shared/serverClock'
import { formatShort, formatZoned, offsetLabel, zonedParts, zonedToMs } from './shared/time'

defineOptions({ name: 'GmOpsTab' })

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()
const ctx = useGmContext()
const clock = useServerClock(ctx)
const domains = provideOpsDomains()

const unsupported = computed(() => !!ctx.game.value && ctx.game.value !== 'gakumas')
const zone = computed(() => clock.zone.value)
const zoneLabel = computed(() => `${zone.value} · ${offsetLabel(zone.value, clock.now.value)}`)
const fmt = (ms: number) => formatZoned(ms, zone.value)

// ---- 读取：7 份配置各读一次；标签页切回来时，没有改动的重新读 ----
const loading = ref(false)
const loaded = ref(false)
const reload = async () => {
  loading.value = true
  try {
    await domains.loadAll()
    loaded.value = true
  } finally {
    loading.value = false
  }
}
onMounted(() => {
  if (unsupported.value) return
  void reload().then(applyQuery)
})
let activated = false
onActivated(() => {
  if (activated && !unsupported.value) {
    for (const d of domains.list) if (!d.state.dirty.value && !d.state.info.value?.draft) void d.state.load()
    applyQuery()
  }
  activated = true
})

const contents = computed<Contents>(() => {
  const out: Contents = {}
  for (const d of domains.list) {
    if (d.state.published.value === undefined) continue
    out[d.key] = { published: d.state.published.value, working: d.state.working.value }
  }
  return out
})

// ---- 月份（服务器时区） ----
const initial = zonedParts(clock.now.value, zone.value)
const year = ref(initial.year)
const month = ref(initial.month)
const shift = (delta: number) => {
  const m = month.value - 1 + delta
  year.value += Math.floor(m / 12)
  month.value = ((m % 12) + 12) % 12 + 1
}
const resetMonth = () => {
  const p = zonedParts(clock.now.value, zone.value)
  year.value = p.year
  month.value = p.month
}
const monthStart = computed(() => zonedToMs(zone.value, year.value, month.value, 1))
const monthEnd = computed(() => (month.value === 12 ? zonedToMs(zone.value, year.value + 1, 1, 1) : zonedToMs(zone.value, year.value, month.value + 1, 1)))

const days = computed(() => {
  const today = zonedParts(clock.now.value, zone.value)
  const out: { day: number; start: number; weekday: number; weekend: boolean; today: boolean }[] = []
  for (let d = 1; d <= 31; d++) {
    const start = zonedToMs(zone.value, year.value, month.value, d)
    if (start >= monthEnd.value) break
    const p = zonedParts(start + 12 * 3_600_000, zone.value)
    if (p.month !== month.value) break
    out.push({ day: d, start, weekday: p.weekday, weekend: p.weekday === 0 || p.weekday === 6, today: today.year === year.value && today.month === month.value && today.day === d })
  }
  return out
})

/** 时刻在本月时间线上的位置（0–1） */
const position = (ms: number) => {
  const list = days.value
  const n = list.length
  if (!n || ms <= monthStart.value) return 0
  if (ms >= monthEnd.value) return 1
  for (let i = 0; i < n; i++) {
    const start = list[i]!.start
    const end = i + 1 < n ? list[i + 1]!.start : monthEnd.value
    if (ms < end) return (i + (ms - start) / (end - start)) / n
  }
  return 1
}
const nowPos = computed(() => (clock.now.value >= monthStart.value && clock.now.value < monthEnd.value ? position(clock.now.value) : null))
const nowLabel = computed(() => t('gmConfig.calendar.now', { time: formatShort(clock.now.value, zone.value) }))

// ---- 泳道（收起 / 展开记在本机） ----
const COLLAPSE_KEY = 'momoko.gm.ops.collapsed'
const collapsed = ref(new Set<LaneKey>((() => {
  try {
    return JSON.parse(localStorage.getItem(COLLAPSE_KEY) || '[]') as LaneKey[]
  } catch {
    return []
  }
})()))
const toggleLane = (key: LaneKey) => {
  const next = new Set(collapsed.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  collapsed.value = next
  localStorage.setItem(COLLAPSE_KEY, JSON.stringify([...next]))
}

const seasonName = (id: string) => {
  const s = domains.pvpRate.state.working.value?.seasons.find((x) => x.pvpRateConfigId === id) || domains.pvpRate.state.published.value?.seasons.find((x) => x.pvpRateConfigId === id)
  return s ? seasonLabel(t, s, ctx.name) : id
}
const tt = (k: string, p?: Record<string, unknown>) => t(k, p || {})
const overlaps = (x: CalItem) => (x.permanent && !x.start) || (x.start < monthEnd.value && (!x.end || x.end > monthStart.value))
const lanes = computed(() =>
  LANES.filter((l) => l.domains.some((d) => contents.value[d])).map((l) => ({
    key: l.key,
    items: laneItems(l.key, contents.value, tt, seasonName)
      .filter(overlaps)
      .sort((a, b) => a.start - b.start || a.title.localeCompare(b.title)),
  })),
)

const MIN_WIDTH = 0.012
const barStyle = (x: CalItem) => {
  const left = x.start ? position(x.start) : 0
  const right = x.end && !x.permanent ? position(x.end) : 1
  return { '--x': left, '--w': Math.max(MIN_WIDTH, right - left) }
}
const periodOf = (x: CalItem) => {
  const from = x.start ? formatShort(x.start, zone.value) : ''
  if (x.permanent) return from ? t('gmConfig.ops.permanentFrom', { from }) : t('gmConfig.ops.status.permanent')
  return `${from} → ${formatShort(x.end, zone.value)}`
}
const barTitle = (x: CalItem) => [x.title, periodOf(x), x.sub].filter(Boolean).join('\n')

// ---- 冲突与提醒 ----
const hints = computed<Hint[]>(() => {
  const out = computeHints(contents.value, clock.now.value, monthStart.value, monthEnd.value, fmt, tt)
  for (const d of domains.list) if (d.state.error.value) out.unshift({ level: 'warn', key: 'loadFailed', params: { domain: t(`gmConfig.ops.domains.${d.key}`), error: d.state.error.value } })
  return out
})

// ---- 编辑窗口：点日历上的条目直接打开 ----
const opened = ref<ItemRef>()
const itemOpen = ref(false)
const openKey = computed(() => (opened.value ? refKey(opened.value) : ''))
const openItem = (r: ItemRef) => {
  opened.value = r
  itemOpen.value = true
}

// ?item=<种类>:<ID> 打开编辑窗口、?new=notice 新建公告（工作台、别的页面跳过来）；处理完从地址里去掉
const applyQuery = () => {
  if (!loaded.value) return
  const r = parseRef(route.query.item)
  const wanted = typeof route.query.new === 'string' ? route.query.new : ''
  if (!r && !wanted) return
  void router.replace({ path: route.path, query: { ...route.query, item: undefined, new: undefined } })
  if (r) openItem(r)
  if (wanted) create(wanted)
}
watch(() => [route.query.item, route.query.new], applyQuery)

// ---- 新建 ----
const gashaCreateOpen = ref(false)
const createItems = computed(() => [
  { key: 'gasha', label: t('gmConfig.ops.new.gasha'), icon: 'i-lucide-sparkles', ready: !!domains.gasha.state.working.value?.gashas.length },
  { key: 'loginEvent', label: t('gmConfig.ops.new.loginEvent'), icon: 'i-lucide-gift', ready: !!domains.loginBonus.state.working.value },
  { key: 'loginSelect', label: t('gmConfig.ops.new.loginSelect'), icon: 'i-lucide-list-checks', ready: !!domains.loginBonus.state.working.value },
  { key: 'season', label: t('gmConfig.ops.new.season'), icon: 'i-lucide-trophy', ready: !!domains.pvpRate.state.working.value },
  { key: 'notice', label: t('gmConfig.ops.new.notice'), icon: 'i-lucide-megaphone', ready: !!domains.notice.state.working.value },
])
const create = (key: string, close?: () => void) => {
  close?.()
  const start = nextHour(clock.now.value)
  if (key === 'gasha') gashaCreateOpen.value = true
  if (key === 'notice') {
    const c = domains.notice.state.working.value
    if (c) openItem({ kind: 'notice', id: addNotice(c, clock.now.value, t('gmConfig.ops.new.noticeTitle')).id })
  }
  if (key === 'loginEvent' || key === 'loginSelect') {
    const c = domains.loginBonus.state.working.value
    if (c) openItem({ kind: 'login', id: addLoginBonus(c, key === 'loginEvent' ? 'event' : 'select', start, (from) => (from ? t('gmConfig.ops.copyName', { name: from }) : '')) })
  }
  if (key === 'season') {
    const c = domains.pvpRate.state.working.value
    if (c) openItem({ kind: 'pvp', id: addSeason(c, clock.now.value) })
  }
}

// ---- 更多：竞赛规则、抽卡趋势、各份配置的版本记录 ----
const rulesOpen = ref(false)
const trendOpen = ref(false)
const revisionsOpen = ref(false)
const revisionsDomain = shallowRef<OpsDomain>()
const moreItems = computed(() => [
  { key: 'rules', label: t('gmConfig.pvp.rulesAndRewards'), icon: 'i-lucide-list-ordered' },
  { key: 'trend', label: t('gmData.home.drawTrend'), icon: 'i-lucide-chart-column' },
  ...domains.list.map((d) => ({ key: `rev:${d.key}`, label: t('gmConfig.ops.revisionsOf', { name: t(`gmConfig.ops.domains.${d.key}`) }), icon: 'i-lucide-history' })),
])
const dash = shallowRef<Dashboard>()
const rootEl = ref<HTMLElement>()
const chartColor = useGameChartColor(rootEl)
const onMore = (key: string) => {
  if (key === 'rules') rulesOpen.value = true
  if (key === 'trend') {
    trendOpen.value = true
    fetchDashboard(ctx.serverId.value)
      .then((r) => (dash.value = r))
      .catch(() => {
        // 没有报表时弹窗显示空
      })
  }
  if (key.startsWith('rev:')) {
    revisionsDomain.value = domains.list.find((d) => `rev:${d.key}` === key)
    revisionsOpen.value = true
  }
}
</script>

<style lang="scss" src="./shared/form.scss"></style>
<style scoped lang="scss">
.gcal {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.gcal__loading {
  padding: 30px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}
.gcal__menu {
  display: flex;
  flex-direction: column;
  padding: 6px;
}
.gcal__menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  padding: 0 12px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.gcal__menu-item:hover:not(:disabled) {
  background: var(--el-fill-color-light);
}
.gcal__menu-item:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.gcal__menu-icon {
  width: 16px;
  height: 16px;
}
.gcal__month {
  display: inline-flex;
  align-items: center;
  gap: 2px;
  padding: 2px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.gcal__month-label {
  min-height: 40px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-weight: 600;
  cursor: pointer;
}
.gcal__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px;
}
.gcal__swatch {
  flex: none;
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 4px;
}
.is-gasha.gcal__swatch,
.gcal__mini.is-gasha {
  background: var(--m-g, var(--m-p));
}
.is-pvp.gcal__swatch,
.gcal__mini.is-pvp {
  background: var(--m-g3, var(--m-a2));
}
.is-event.gcal__swatch,
.gcal__mini.is-event {
  background: var(--m-g2, var(--m-a));
}
.is-login.gcal__swatch,
.gcal__mini.is-login {
  background: var(--m-a2);
}
.is-notice.gcal__swatch,
.gcal__mini.is-notice {
  background: var(--m-warn);
}

/* 时间线 */
.gcal__board {
  overflow-x: auto;
  padding: 8px 0 14px;
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.gcal__inner {
  --label: 180px;
  position: relative;
  min-width: 1040px;
}
.gcal__row {
  display: grid;
  grid-template-columns: var(--label) minmax(0, 1fr);
  align-items: center;
  min-height: 48px;
}
.gcal__row--head {
  min-height: 52px;
  border-bottom: 1px solid var(--el-border-color-light);
}
.gcal__row--folded {
  min-height: 44px;
}
.gcal__label {
  display: flex;
  align-items: center;
  padding-left: 12px;
  min-width: 0;
}
.gcal__label--month {
  padding-left: 14px;
  font-size: 13px;
  font-weight: 600;
}
.gcal__toggle {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 30px;
  padding: 0 10px 0 6px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  cursor: pointer;
}
.gcal__toggle:hover:not(:disabled) {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.gcal__toggle:disabled {
  cursor: default;
}
.gcal__chev {
  width: 16px;
  height: 16px;
}
.gcal__count {
  font-size: 12px;
  opacity: 0.7;
}
.gcal__days {
  display: grid;
  grid-template-columns: repeat(var(--days), minmax(0, 1fr));
}
.gcal__day {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 1px;
  height: 52px;
}
.gcal__day.is-weekend {
  background: var(--m-s2);
}
.gcal__day.is-today {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.gcal__day-n {
  font-size: 13px;
  font-weight: 600;
}
.gcal__day-w {
  font-size: 10px;
  color: var(--el-text-color-secondary);
}
.gcal__lane {
  padding: 4px 0;
  border-bottom: 1px solid var(--el-border-color-light);
}
.gcal__track {
  position: relative;
  height: 48px;
  background-image: linear-gradient(to right, var(--el-border-color-lighter) 1px, transparent 1px);
  background-size: calc(100% / var(--days)) 100%;
}
.gcal__track--mini {
  height: 36px;
  cursor: pointer;
}
.gcal__mini {
  position: absolute;
  top: 14px;
  height: 8px;
  left: calc(var(--x) * 100%);
  width: calc(var(--w) * 100%);
  border-radius: 999px;
  opacity: 0.35;
}
.gcal__bar {
  position: absolute;
  top: 4px;
  bottom: 4px;
  left: calc(var(--x) * 100%);
  width: calc(var(--w) * 100%);
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 10px 0 4px;
  border-radius: 999px;
  border: 1.5px solid transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 12px;
  font-weight: 600;
  text-align: left;
  white-space: nowrap;
  overflow: hidden;
  cursor: pointer;
}
.gcal__bar:hover {
  filter: brightness(0.97);
  box-shadow: 0 0 0 2px var(--el-border-color);
}
.gcal__bar.is-gasha {
  background: var(--m-gSoft, var(--m-pSoft));
  border-color: var(--m-gLine, var(--el-border-color-light));
}
.gcal__bar.is-pvp {
  background: var(--m-g3Soft, var(--m-a2Soft));
}
.gcal__bar.is-event {
  background: var(--m-g2Soft, var(--m-aSoft));
}
.gcal__bar.is-login {
  background: var(--m-a2Soft);
}
.gcal__bar.is-notice {
  background: var(--m-warnSoft);
}
.gcal__bar.is-perm {
  border-style: dashed;
  border-color: var(--el-border-color);
}
.gcal__bar.is-draft {
  border-style: dashed;
  border-color: var(--m-warn);
}
.gcal__bar.is-removed {
  opacity: 0.55;
  text-decoration: line-through;
}
.gcal__bar.is-on {
  box-shadow: 0 0 0 2px var(--m-ink);
}
.gcal__bar .gcal__bar-img {
  flex: none;
  height: 32px;
  width: 92px;
  object-fit: contain;
  border-radius: 999px;
  background: transparent;
}
.gcal__bar-text {
  overflow: hidden;
  text-overflow: ellipsis;
}
.gcal__draft {
  display: inline-flex;
  margin-right: 4px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 10px;
  font-weight: 600;
}
.gcal__none {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  padding-left: 8px;
}
.gcal__now {
  position: absolute;
  top: 0;
  bottom: 0;
  left: calc(var(--label) + (100% - var(--label)) * var(--x));
  width: 2px;
  margin-left: -1px;
  background: var(--m-p);
  z-index: 2;
  pointer-events: none;
}
.gcal__now-tag {
  position: absolute;
  bottom: -2px;
  left: 4px;
  padding: 0 6px;
  border-radius: 999px;
  background: var(--m-p);
  color: var(--m-pOn);
  font-size: 10px;
  font-weight: 600;
  white-space: nowrap;
}

/* 手机 */
.gcal__agenda {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.gcal__agenda-lane {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 14px;
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.gcal__agenda-title {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  border: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 13px;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.gcal__agenda-item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  padding: 6px 8px;
  border: 1.5px solid transparent;
  border-radius: 14px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gcal__agenda-item.is-draft {
  border-style: dashed;
  border-color: var(--m-warn);
  background: var(--el-bg-color);
}
.gcal__agenda-item.is-removed {
  opacity: 0.6;
  text-decoration: line-through;
}
.gcal__agenda-item .gcal__agenda-img {
  flex: none;
  width: 84px;
  height: 30px;
  object-fit: contain;
  background: transparent;
}
.gcal__agenda-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gcal__agenda-name {
  font-size: 13px;
  font-weight: 600;
  overflow-wrap: anywhere;
}
.gcal__agenda-period {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* 提醒 */
.gcal__hints {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.gcal__hint {
  display: flex;
  align-items: flex-start;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 16px;
  font-size: 13px;
  line-height: 1.6;
}
.gcal__hint.is-warn {
  background: var(--m-warnSoft);
}
.gcal__hint.is-info {
  background: var(--m-a2Soft);
}
.gcal__hint-icon {
  flex: none;
  width: 18px;
  height: 18px;
  margin-top: 2px;
}
.gcal__hint.is-warn .gcal__hint-icon {
  color: var(--m-warn);
}
.gcal__hint-text {
  flex: 1;
  min-width: 0;
  overflow-wrap: anywhere;
}
.gcal__color-probe {
  display: none;
}
</style>
