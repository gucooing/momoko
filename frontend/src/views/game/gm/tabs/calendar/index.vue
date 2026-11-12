<!-- GM 运营 · 运营日历（?tab=calendar）：把卡池、竞赛赛季、活动、登录奖励、公告放在一条月度时间线上。
     泳道只来自这台服务器 manifest 里有的配置域，每个配置域在一次访问里只读一次（GET /configs/{domain}）；
     时间按服务器时区显示，竖线是服务器当前时间。草稿里新增 / 改了时间的条目用虚线标出；下方是从数据算出的冲突与提醒。 -->
<template>
  <div class="gcal game-skin">
    <PageHeader :title="t('gmConfig.calendar.title')" :back="false" :theme-toggle="false">
      <template #description>{{ t('gmConfig.calendar.desc', { zone: zoneLabel }) }}</template>
      <template #actions>
        <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('gmConfig.calendar.reload')" :box="40" @click="load" />
        <UButton
          v-if="tabVisible('notices')"
          class="gmc-btn"
          color="neutral"
          variant="outline"
          icon="i-lucide-megaphone"
          :aria-label="t('gmConfig.notice.create')"
          @click="go({ tab: 'notices', query: { new: '1' } })"
        >
          <template v-if="!menuStore.isMobile">{{ t('gmConfig.notice.create') }}</template>
        </UButton>
      </template>
    </PageHeader>

    <EmptyState
      v-if="unsupported"
      icon="HOutline:PuzzlePieceIcon"
      :title="t('gmConfig.common.unsupportedTitle')"
      :description="t('gmConfig.common.unsupportedDesc', { game: ctx.gameName.value })"
    />
    <div v-else-if="loading && !domains.size" class="gcal__loading">{{ t('gmConfig.common.loading') }}</div>
    <EmptyState
      v-else-if="!lanes.length"
      icon="HOutline:CalendarDaysIcon"
      :title="t('gmConfig.calendar.emptyTitle')"
      :description="t('gmConfig.calendar.emptyDesc')"
    />
    <template v-else>
      <div class="gcal__toolbar">
        <div class="gcal__month" role="group" :aria-label="t('gmConfig.calendar.month')">
          <AppIconButton icon="HOutline:ChevronLeftIcon" :label="t('gmConfig.calendar.prev')" :box="40" @click="shift(-1)" />
          <button type="button" class="gcal__month-label m-num" :title="t('gmConfig.calendar.thisMonth')" @click="resetMonth">
            {{ t('gmConfig.calendar.monthLabel', { y: year, m: month }) }}
          </button>
          <AppIconButton icon="HOutline:ChevronRightIcon" :label="t('gmConfig.calendar.next')" :box="40" @click="shift(1)" />
        </div>
        <div class="gmc-chips gcal__lanes" role="group" :aria-label="t('gmConfig.calendar.lanes')">
          <button
            v-for="l in lanes"
            :key="l.key"
            type="button"
            class="gmc-chip"
            :class="{ 'is-on': !hidden.has(l.key) }"
            :aria-pressed="!hidden.has(l.key)"
            @click="toggleLane(l.key)"
          >
            <span class="gcal__swatch" :class="`is-${l.key}`" />
            {{ t(`gmConfig.calendar.lane.${l.key}`) }}
            <span class="gmc-chip__n m-num">{{ l.items.length }}</span>
          </button>
        </div>
      </div>

      <!-- PC：月度时间线 -->
      <section v-if="!menuStore.isMobile" class="gcal__board">
        <div class="gcal__inner" :style="{ '--days': days.length }">
          <div class="gcal__row gcal__row--head">
            <span class="gcal__label gcal__label--month">{{ t('gmConfig.calendar.monthShort', { m: month }) }}</span>
            <div class="gcal__days">
              <span
                v-for="d in days"
                :key="d.day"
                class="gcal__day"
                :class="{ 'is-weekend': d.weekend, 'is-today': d.today }"
              >
                <span class="gcal__day-n m-num">{{ d.day }}</span>
                <span class="gcal__day-w">{{ t(`gmConfig.calendar.weekdays.${d.weekday}`) }}</span>
              </span>
            </div>
          </div>

          <div v-for="l in visibleLanes" :key="l.key" class="gcal__lane">
            <div v-for="(item, i) in l.items" :key="item.key" class="gcal__row">
              <span class="gcal__label">
                <template v-if="i === 0">
                  <span class="gcal__swatch" :class="`is-${l.key}`" />
                  {{ t(`gmConfig.calendar.lane.${l.key}`) }}
                </template>
              </span>
              <div class="gcal__track">
                <button
                  type="button"
                  class="gcal__bar"
                  :class="[`is-${l.key}`, { 'is-perm': item.permanent, 'is-draft': !!item.draft, 'is-removed': item.draft === 'removed', 'is-on': selected?.key === item.key }]"
                  :style="barStyle(item)"
                  :title="barTitle(item)"
                  @click="selected = item"
                >
                  <GameImage v-if="item.image" :src="item.image" class="gcal__bar-img" />
                  <span class="gcal__bar-text">
                    <span v-if="item.draft" class="gcal__draft">{{ t(`gmConfig.calendar.draft.${item.draft}`) }}</span>
                    {{ item.title }}
                  </span>
                </button>
              </div>
            </div>
            <div v-if="!l.items.length" class="gcal__row">
              <span class="gcal__label">
                <span class="gcal__swatch" :class="`is-${l.key}`" />
                {{ t(`gmConfig.calendar.lane.${l.key}`) }}
              </span>
              <span class="gcal__none">{{ t('gmConfig.calendar.laneEmpty') }}</span>
            </div>
          </div>

          <div v-if="nowPos !== null" class="gcal__now" :style="{ '--x': nowPos }">
            <span class="gcal__now-tag m-num">{{ nowLabel }}</span>
          </div>
        </div>
      </section>

      <!-- 手机：按泳道列出本月的条目 -->
      <section v-else class="gcal__agenda">
        <div v-for="l in visibleLanes" :key="l.key" class="gcal__agenda-lane">
          <h3 class="gcal__agenda-title">
            <span class="gcal__swatch" :class="`is-${l.key}`" />
            {{ t(`gmConfig.calendar.lane.${l.key}`) }}
            <span class="gmc-chip__n m-num">{{ l.items.length }}</span>
          </h3>
          <button
            v-for="item in l.items"
            :key="item.key"
            type="button"
            class="gcal__agenda-item"
            :class="{ 'is-draft': !!item.draft, 'is-removed': item.draft === 'removed' }"
            @click="selected = item"
          >
            <GameImage v-if="item.image" :src="item.image" class="gcal__agenda-img" />
            <span class="gcal__agenda-text">
              <span class="gcal__agenda-name">
                <span v-if="item.draft" class="gcal__draft">{{ t(`gmConfig.calendar.draft.${item.draft}`) }}</span>
                {{ item.title }}
              </span>
              <span class="gcal__agenda-period m-num">{{ periodText(item) }}</span>
            </span>
          </button>
          <p v-if="!l.items.length" class="gmc-hint">{{ t('gmConfig.calendar.laneEmpty') }}</p>
        </div>
      </section>

      <section v-if="selected" class="gcal__detail">
        <GameImage v-if="selected.image" :src="selected.image" class="gcal__detail-img" />
        <span class="gcal__detail-text">
          <span class="gcal__detail-title">
            <span class="gcal__swatch" :class="`is-${selected.lane}`" />
            {{ selected.title }}
            <span v-if="selected.draft" class="gcal__draft">{{ t(`gmConfig.calendar.draft.${selected.draft}`) }}</span>
          </span>
          <span class="gmc-hint m-num">{{ periodText(selected) }}</span>
          <span v-if="selected.sub" class="gmc-hint">{{ selected.sub }}</span>
          <span class="gmc-hint">{{ sourceTitle(selected.source) }}</span>
        </span>
        <UButton class="gmc-btn" v-if="selected.link && tabVisible(selected.link.tab)" color="neutral" variant="outline" icon="i-lucide-arrow-right" @click="go(selected.link)">
          {{ t('gmConfig.calendar.open') }}
        </UButton>
        <AppIconButton icon="HOutline:XMarkIcon" :label="t('gmConfig.common.close')" :box="40" @click="selected = null" />
      </section>

      <div class="gcal__cards">
        <AppPanel :title="t('gmConfig.calendar.hintsTitle')" title-icon="HOutline:ExclamationTriangleIcon" class="gcal__card">
          <ul v-if="hints.length" class="gcal__hints">
            <li v-for="(h, i) in hints" :key="i" class="gcal__hint" :class="`is-${h.level}`">
              <UIcon :name="h.level === 'warn' ? 'i-lucide-triangle-alert' : 'i-lucide-info'" class="gcal__hint-icon" />
              <span class="gcal__hint-text">{{ t(`gmConfig.calendar.hints.${h.key}`, h.params) }}</span>
              <UButton class="gmc-btn" v-if="h.link && tabVisible(h.link.tab)" size="xs" color="neutral" variant="ghost" @click="go(h.link)">{{ t('gmConfig.calendar.open') }}</UButton>
            </li>
          </ul>
          <EmptyState v-else icon="HOutline:CheckCircleIcon" :title="t('gmConfig.calendar.noHints')" />
        </AppPanel>
        <AppPanel :title="t('gmConfig.calendar.draftsTitle')" title-icon="HOutline:PencilSquareIcon" class="gcal__card">
          <ul v-if="drafts.length" class="gcal__drafts">
            <li v-for="d in drafts" :key="d.domain" class="gcal__draft-row">
              <span class="gcal__draft-text">
                <span class="gcal__draft-name">{{ d.title }}</span>
                <span class="gmc-hint">{{ t('gmConfig.calendar.draftMeta', { n: d.changes, name: d.author || '—', time: d.time }) }}</span>
              </span>
              <StatusPill variant="warning" :label="t('gmConfig.calendar.unpublished')" />
              <UButton class="gmc-btn" v-if="d.tab && tabVisible(d.tab)" size="xs" color="neutral" variant="outline" @click="go({ tab: d.tab, query: {} })">{{ t('gmConfig.calendar.open') }}</UButton>
            </li>
          </ul>
          <EmptyState v-else icon="HOutline:CheckCircleIcon" :title="t('gmConfig.calendar.noDrafts')" />
        </AppPanel>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRoute, useRouter } from 'vue-router'
import { gashaConfig, homeConfig, loginBonusConfig, noticeConfig, pvpRateConfig, researchConfig, storyEventConfig, type GakumasConfigApi } from '@/api/gakumas'
import { getRequestErrorMessage } from '@/utils/request'
import { useGmContext } from '../../context'
import { GM_TABS } from '../../tabs'
import { diffJson } from '../gasha/shared/diff'
import { useServerClock } from '../gasha/shared/serverClock'
import { formatShort, formatZoned, offsetLabel, toMs, zonedParts, zonedToMs } from '../gasha/shared/time'
import { computeHints, laneItems, LANES, type CalItem, type DomainContent, type Hint, type LaneKey } from './calendar'

defineOptions({ name: 'GmCalendarTab' })

const { t } = useI18n()
const route = useRoute()
const router = useRouter()
const menuStore = useMenuStore()
const ctx = useGmContext()
const clock = useServerClock(ctx)

const unsupported = computed(() => !!ctx.game.value && ctx.game.value !== 'gakumas')
const zone = clock.zone
const zoneLabel = computed(() => `${zone.value} · ${offsetLabel(zone.value, clock.now.value)}`)
const fmt = (ms: number) => formatZoned(ms, zone.value)

// ---- 读取配置文档（每次访问一次） ----
const CONFIGS: Record<string, GakumasConfigApi> = {
  gasha: gashaConfig,
  notice: noticeConfig,
  pvpRate: pvpRateConfig,
  home: homeConfig,
  research: researchConfig,
  storyEvent: storyEventConfig,
  loginBonus: loginBonusConfig,
}
const domains = shallowRef(new Map<string, DomainContent>())
const failures = ref<{ domain: string; error: string }[]>([])
const loading = ref(false)

const load = async () => {
  const serverId = ctx.serverId.value
  const needed = [...new Set(LANES.flatMap((l) => l.domains))].filter((d) => !!CONFIGS[d])
  loading.value = true
  try {
    const results = await Promise.allSettled(needed.map((d) => CONFIGS[d]!.get(serverId)))
    const next = new Map<string, DomainContent>()
    const failed: { domain: string; error: string }[] = []
    results.forEach((r, i) => {
      const domain = needed[i]!
      if (r.status === 'rejected') {
        failed.push({ domain, error: getRequestErrorMessage(r.reason) })
        return
      }
      try {
        const data = r.value
        const draft = data.domain?.draft
        next.set(domain, {
          domain,
          title: data.domain?.title || domain,
          published: JSON.parse(data.content || 'null'),
          draft: draft?.content ? JSON.parse(draft.content) : undefined,
          draftAuthor: draft?.authorName || '',
          draftTime: toMs(draft?.updateTime),
        })
      } catch (e) {
        failed.push({ domain, error: e instanceof Error ? e.message : String(e) })
      }
    })
    domains.value = next
    failures.value = failed
  } finally {
    loading.value = false
  }
}
onMounted(() => {
  if (!unsupported.value) void load()
})
// 标签页切回来算一次新的访问：重新读一次（切换月份、泳道不会重复读取）
let activated = false
onActivated(() => {
  if (activated && !unsupported.value) void load()
  activated = true
})

// ---- 月份（服务器时区） ----
const initial = zonedParts(clock.now.value, zone.value)
const year = ref(initial.year)
const month = ref(initial.month)
const shift = (delta: number) => {
  const m = month.value - 1 + delta
  year.value += Math.floor(m / 12)
  month.value = ((m % 12) + 12) % 12 + 1
  selected.value = null
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
    out.push({
      day: d,
      start,
      weekday: p.weekday,
      weekend: p.weekday === 0 || p.weekday === 6,
      today: today.year === year.value && today.month === month.value && today.day === d,
    })
  }
  return out
})

/** 时刻在本月时间线上的位置（0–1） */
const position = (ms: number) => {
  const list = days.value
  const n = list.length
  if (!n) return 0
  if (ms <= monthStart.value) return 0
  if (ms >= monthEnd.value) return 1
  for (let i = 0; i < n; i++) {
    const start = list[i]!.start
    const end = i + 1 < n ? list[i + 1]!.start : monthEnd.value
    if (ms < end) return (i + (ms - start) / (end - start)) / n
  }
  return 1
}

const nowPos = computed(() => {
  const now = clock.now.value
  return now >= monthStart.value && now < monthEnd.value ? position(now) : null
})
const nowLabel = computed(() => t('gmConfig.calendar.now', { time: formatShort(clock.now.value, zone.value) }))

// ---- 泳道 ----
const hidden = ref(new Set<LaneKey>())
const toggleLane = (key: LaneKey) => {
  const next = new Set(hidden.value)
  if (next.has(key)) next.delete(key)
  else next.add(key)
  hidden.value = next
}

const overlaps = (x: CalItem) => (x.permanent && !x.start) || (x.start < monthEnd.value && (!x.end || x.end > monthStart.value))

const lanes = computed(() => {
  const input = { domains: domains.value, t: (k: string, p?: Record<string, unknown>) => t(k, p || {}), fmt }
  return LANES.filter((l) => l.domains.some((d) => domains.value.has(d))).map((l) => ({
    key: l.key,
    items: laneItems(l.key, input)
      .filter(overlaps)
      .sort((a, b) => a.start - b.start || a.title.localeCompare(b.title)),
  }))
})
const visibleLanes = computed(() => lanes.value.filter((l) => !hidden.value.has(l.key)))

const MIN_WIDTH = 0.012
const barStyle = (x: CalItem) => {
  const left = x.start ? position(x.start) : 0
  const right = x.end && !x.permanent ? position(x.end) : 1
  return { '--x': left, '--w': Math.max(MIN_WIDTH, right - left) }
}
const periodText = (x: CalItem) => {
  const from = x.start ? formatShort(x.start, zone.value) : ''
  if (x.permanent) return from ? t('gmConfig.calendar.permanentFrom', { from }) : t('gmConfig.calendar.permanent')
  return `${from} → ${formatShort(x.end, zone.value)}`
}
const barTitle = (x: CalItem) => [x.title, periodText(x), x.sub].filter(Boolean).join('\n')

// 条目来自哪份配置：显示配置的标题
const sourceTitle = (domain: string) => domains.value.get(domain)?.title || ''

const selected = ref<CalItem | null>(null)

// ---- 冲突与提醒、草稿 ----
const hints = computed<Hint[]>(() => {
  const out = computeHints(domains.value, clock.now.value, monthStart.value, monthEnd.value, fmt)
  for (const f of failures.value) out.unshift({ level: 'warn', key: 'loadFailed', params: { domain: f.domain, error: f.error } })
  return out
})

const TAB_OF_DOMAIN: Record<string, string> = { gasha: 'gasha', notice: 'notices' }
const drafts = computed(() =>
  [...domains.value.values()]
    .filter((d) => d.draft !== undefined)
    .map((d) => ({
      domain: d.domain,
      title: d.title,
      author: d.draftAuthor,
      time: d.draftTime ? formatShort(d.draftTime, zone.value) : '—',
      changes: diffJson(d.published, d.draft).length,
      tab: TAB_OF_DOMAIN[d.domain] || '',
    })),
)

const tabVisible = (key: string) => !!GM_TABS.find((tab) => tab.key === key)?.visible(ctx)
const go = (link: { tab: string; query: Record<string, string> }) => {
  void router.replace({ path: route.path, query: { tab: link.tab, ...link.query } })
}
</script>

<style lang="scss" src="../gasha/shared/form.scss"></style>
<style scoped lang="scss">
.gcal {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.gcal__loading {
  padding: 40px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
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
  font-weight: 800;
  cursor: pointer;
}
.gcal__toolbar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.gcal__lanes {
  flex: 1 1 320px;
  flex-wrap: wrap;
}
.gcal__swatch {
  flex: none;
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 4px;
}
.gcal__swatch.is-gasha {
  background: var(--m-g, var(--m-p));
}
.gcal__swatch.is-pvp {
  background: var(--m-g3, var(--m-a2));
}
.gcal__swatch.is-event {
  background: var(--m-g2, var(--m-a));
}
.gcal__swatch.is-login {
  background: var(--m-a2);
}
.gcal__swatch.is-notice {
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
.gcal__label {
  display: flex;
  align-items: center;
  gap: 8px;
  padding-left: 20px;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gcal__label--month {
  font-size: 13px;
  font-weight: 800;
  color: var(--el-text-color-primary);
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
  font-weight: 800;
}
.gcal__day-w {
  font-size: 10px;
  color: var(--el-text-color-secondary);
}
.gcal__lane {
  padding: 6px 0;
  border-bottom: 1px solid var(--el-border-color-light);
}
.gcal__track {
  position: relative;
  height: 48px;
  background-image: linear-gradient(to right, var(--el-border-color-lighter) 1px, transparent 1px);
  background-size: calc(100% / var(--days)) 100%;
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
  font-weight: 700;
  text-align: left;
  white-space: nowrap;
  overflow: hidden;
  cursor: pointer;
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
  background: var(--el-bg-color);
  border-style: dashed;
  border-color: var(--m-warn);
}
.gcal__bar.is-removed {
  opacity: 0.6;
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
  font-weight: 800;
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
  font-weight: 800;
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
  margin: 0 0 4px;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 800;
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
  font-weight: 800;
  overflow-wrap: anywhere;
}
.gcal__agenda-period {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

/* 选中条目 */
.gcal__detail {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 14px;
  padding: 14px 16px;
  border-radius: var(--app-radius-xl);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.gcal__detail .gcal__detail-img {
  flex: none;
  width: 184px;
  max-width: 100%;
  height: 64px;
  object-fit: contain;
  background: transparent;
}
.gcal__detail-text {
  flex: 1 1 240px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gcal__detail-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 15px;
  font-weight: 800;
  overflow-wrap: anywhere;
}

/* 提醒与草稿 */
.gcal__cards {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}
.gcal__card {
  flex: 1 1 420px;
  min-width: 0;
}
.gcal__hints,
.gcal__drafts {
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
.gcal__draft-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: 16px;
  background: var(--el-fill-color-light);
}
.gcal__draft-text {
  flex: 1 1 200px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gcal__draft-name {
  font-weight: 800;
}
</style>
