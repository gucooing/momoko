<!-- GM 运营 · 工作台（只读观测）：今日关键数据 → 活跃玩家 / 抽卡次数（近 7 / 14 天单系列图，悬停看数值）
     → 卡池表现 · 今天 → 需要留意（待审批、今天失败的 GM 操作、定时邮件、未发布的配置草稿、即将结束的卡池与赛季、维护与依赖异常）
     + 玩法概况 + 服务器 + 最近 GM 操作。数据：MGP dashboard 报表（days=14）、status、Momoko 的 GM 记录 / 邮件 / 配置草稿；
     每块只链接到对应页面，不放操作入口。每 5 分钟自动刷新（标签可见时）。 -->
<template>
  <div ref="root" class="gh game-skin">
    <PageHeader :back="false" :theme-toggle="false">
      <template #title>{{ t('gmData.home.title') }}</template>
      <template #description>
        {{ report ? t('gmData.home.subtitle', { server: ctx.server.value?.name || '', time: clock(report.asOf) }) : t('gmData.home.subtitleLoading') }}
      </template>
      <template #actions>
        <div class="gh__seg" role="radiogroup" :aria-label="t('gmData.home.range')">
          <button
            v-for="d in RANGES"
            :key="d"
            type="button"
            role="radio"
            :aria-checked="days === d"
            :class="{ 'is-on': days === d }"
            @click="days = d"
          >
            {{ t('gmData.home.lastDays', { n: d }) }}
          </button>
        </div>
      </template>
    </PageHeader>

    <EmptyState
      v-if="error && !report"
      icon="HOutline:ExclamationTriangleIcon"
      :title="t('gmData.home.loadFailed')"
      :description="error"
    >
      <template #action>
        <UButton color="neutral" variant="outline" @click="load">{{ t('gmData.common.retry') }}</UButton>
      </template>
    </EmptyState>

    <template v-else-if="report">
      <!-- 今日关键数据 -->
      <section class="gh__kpis" :aria-label="t('gmData.home.kpis')">
        <component :is="k.to ? RouterLink : 'div'" v-for="k in kpis" :key="k.key" :to="k.to" class="gh__kpi">
          <span class="gh__kpi-head">
            <span v-if="k.live" class="gh__live" aria-hidden="true" />
            <GameImage v-else-if="k.image" :src="k.image" class="gh__kpi-img" width="24" height="24" />
            <component :is="menuStore.iconComponents[k.icon]" v-else-if="k.icon" class="gh__kpi-icon" />
            <span class="gh__kpi-title">{{ k.title }}</span>
          </span>
          <span class="gh__kpi-value m-num">{{ k.value }}</span>
          <span class="gh__kpi-sub">
            <span v-if="k.delta" class="m-num gh__kpi-delta" :class="k.tone">{{ k.delta }}</span>
            {{ k.sub }}
          </span>
        </component>
      </section>

      <!-- 趋势 -->
      <section class="gh__charts">
        <AppPanel :title="t('gmData.home.activeChart', { range: t('gmData.home.lastDays', { n: days }) })" :caption="t('gmData.home.activeHint')">
          <TrendChart
            kind="line"
            :points="activePoints"
            :color="chartColor"
            :format="(v) => t('gmData.home.people', { n: num(v) })"
            :aria-label="activeAria"
          />
        </AppPanel>
        <AppPanel :title="t('gmData.home.drawChart', { range: t('gmData.home.lastDays', { n: days }) })" :caption="t('gmData.home.drawHint')">
          <TrendChart
            kind="bar"
            :points="drawPoints"
            :color="chartColor"
            :format="(v) => t('gmData.home.times', { n: num(v) })"
            :aria-label="drawAria"
          />
        </AppPanel>
      </section>

      <!-- 卡池表现 -->
      <AppPanel v-if="gashas.length" :title="t('gmData.home.gashaTitle')">
        <template v-if="tabLink('gasha')" #actions>
          <RouterLink class="gh__more" :to="tabLink('gasha')!">{{ t('gmData.home.gashaMore') }}</RouterLink>
        </template>
        <div class="gh__pools">
          <div class="gh__pool gh__pool--head" aria-hidden="true">
            <span>{{ t('gmData.home.gashaPool') }}</span>
            <span />
            <span>{{ t('gmData.home.gashaToday') }}</span>
            <span class="is-right">{{ t('gmData.home.gashaTotal') }}</span>
            <span class="is-right">SSR</span>
            <span>{{ t('gmData.home.gashaState') }}</span>
          </div>
          <component :is="tabLink('gasha') ? RouterLink : 'div'" v-for="g in gashas" :key="g.gashaId" :to="tabLink('gasha')" class="gh__pool">
            <GameImage :src="gashaImage(g)" class="gh__banner" width="160" height="56" />
            <span class="gh__pool-name">
              <span class="gh__strong gh__ellipsis" :title="g.name">{{ g.name || ctx.label('gasha', g.gashaId) }}</span>
              <span class="gh__dim">{{ g.endTime ? t('gmData.home.until', { time: dateShort(g.endTime) }) : t('gmData.home.permanent') }}</span>
            </span>
            <span class="gh__pool-today">
              <span class="gh__track"><span :style="{ width: `${barWidth(g)}%` }" /></span>
              <span class="gh__pool-n m-num">
                {{ num(g.today.draws) }}
                <span class="gh__dim">{{ pct(g.today.draws, report.drawsToday.draws) }}%</span>
              </span>
            </span>
            <span class="gh__pool-total m-num is-right"><span class="gh__mlabel">{{ t('gmData.home.gashaTotal') }} </span>{{ num(g.total.draws) }}</span>
            <span class="gh__pool-ssr m-num is-right"><span class="gh__mlabel">SSR </span>{{ num(g.total.ssr) }}</span>
            <StatusPill class="gh__pool-state" :variant="gashaState(g).variant" :label="gashaState(g).label" />
          </component>
        </div>
      </AppPanel>

      <section class="gh__row">
        <!-- 需要留意 -->
        <AppPanel class="gh__alerts" :title="t('gmData.home.alerts')" :caption="t('gmData.home.alertsHint')">
          <EmptyState v-if="!alerts.length" icon="HOutline:CheckBadgeIcon" :title="t('gmData.home.allGood')" />
          <component :is="a.to ? RouterLink : 'div'" v-for="a in alerts" :key="a.key" :to="a.to" class="gh__alert">
            <span class="gh__alert-icon m-sticker" :class="`m-sticker--${a.tone}`" aria-hidden="true">
              <component :is="menuStore.iconComponents[a.icon]" />
            </span>
            <span class="gh__alert-text">
              <span class="gh__strong">{{ a.title }}</span>
              <span class="gh__dim">{{ a.sub }}</span>
            </span>
            <span v-if="a.where" class="gh__tag">{{ a.where }}</span>
            <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" v-if="a.to" class="gh__chev" />
          </component>
        </AppPanel>

        <div class="gh__side">
          <!-- 玩法 -->
          <AppPanel v-if="modes.length" :title="t('gm.tabs.modes')">
            <template v-if="tabLink('modes')" #actions>
              <RouterLink class="gh__more" :to="tabLink('modes')!">{{ t('gmData.home.view') }}</RouterLink>
            </template>
            <component :is="tabLink('modes') ? RouterLink : 'div'" v-for="m in modes" :key="m.id" :to="tabLink('modes')" class="gh__mode">
              <span class="gh__mode-icon" aria-hidden="true">
                <component :is="menuStore.iconComponents[m.icon]" />
              </span>
              <span class="gh__mode-text">
                <span class="gh__strong">{{ m.title }}</span>
                <span class="gh__dim gh__ellipsis">{{ m.sub }}</span>
              </span>
              <span v-if="m.value" class="gh__mode-value">
                <span class="m-num">{{ m.value }}</span>
                <span class="gh__dim">{{ m.unit }}</span>
              </span>
            </component>
          </AppPanel>

          <!-- 服务器 -->
          <RouterLink class="gh__server" :to="serverLink">
            <span class="gh__server-head">
              <span class="gh__server-title">{{ t('gmData.home.server') }}</span>
              <StatusPill
                :variant="report.maintenanceEnabled ? 'warning' : 'success'"
                :label="report.maintenanceEnabled ? t('gmData.home.maintenance') : t('gmData.home.running')"
              />
            </span>
            <dl class="gh__server-facts">
              <div v-for="f in serverFacts" :key="f.key">
                <dt>{{ f.label }}</dt>
                <dd :class="f.mono ? 'm-mono' : 'm-num'">{{ f.value }}</dd>
              </div>
            </dl>
          </RouterLink>

          <!-- 最近 GM 操作 -->
          <AppPanel :title="t('gmData.home.recent')">
            <template v-if="tabLink('logs')" #actions>
              <RouterLink class="gh__more" :to="tabLink('logs')!">{{ t('gmData.home.recentMore') }}</RouterLink>
            </template>
            <EmptyState v-if="!recent.length" icon="HOutline:ClockIcon" :title="t('gmData.home.recentEmpty')" />
            <component :is="tabLink('logs') ? RouterLink : 'div'" v-for="e in recent" :key="e.id" :to="tabLink('logs')" class="gh__log">
              <span class="gh__log-av" :class="{ 'is-player': e.actorKind === 'player' }">{{ (e.actorName || '?').slice(0, 1) }}</span>
              <span class="gh__log-text">
                <span class="gh__strong gh__ellipsis">{{ logTitle(e) }}</span>
                <span class="gh__dim gh__ellipsis">{{ logSub(e) }}</span>
              </span>
              <GmRiskPill :risk="e.risk" />
            </component>
          </AppPanel>
        </div>
      </section>
    </template>

    <div v-else class="gh__skeleton" aria-busy="true">
      <span v-for="i in 5" :key="i" class="gh__skel-kpi" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink } from 'vue-router'
import dayjs from 'dayjs'
import { getGmReport, listGmConfigs, listGmExecutions } from '@/api/gameGm'
import { listGameMails } from '@/api/gameOps'
import { getRequestErrorMessage } from '@/utils/request'
import { usePaletteColors } from '@/composables/usePaletteColors'
import type { DashboardReport, GashaStats } from '@/games/gakumas/types'
import type { GmConfigDomain, GmExecution } from '@/types/v1/game_gm'
import type { GameMail } from '@/types/v1/game_ops'
import { useGmContext } from '../../context'
import { GM_TABS } from '../../tabs'
import GmRiskPill from '../../components/GmRiskPill.vue'
import { catalogImage, gradeLabel } from '../../player/shared/game'
import { gmExecutionsQuery } from '../../player/shared/executions'
import { toMs, useGmFormat, zonedDayStart } from '../../player/shared/format'
import TrendChart, { type TrendPoint } from './TrendChart.vue'

defineOptions({ name: 'GmHomeTab' })

const RANGES = [7, 14] as const
const REFRESH_MS = 5 * 60_000
const DAY = 86_400_000

const { t } = useI18n()
const ctx = useGmContext()
const menuStore = useMenuStore()
const { num, when, duration, pct, delta } = useGmFormat()

const days = ref<7 | 14>(7)
const report = shallowRef<DashboardReport>()
const error = ref('')
const recent = shallowRef<GmExecution[]>([])
const pending = shallowRef<GmExecution[]>([])
const failedToday = ref(0)
const lastFailed = shallowRef<GmExecution>()
const scheduledMails = shallowRef<GameMail[]>([])
const drafts = shallowRef<GmConfigDomain[]>([])

// ---------- 读取 ----------
let seq = 0
const load = async () => {
  const sid = ctx.serverId.value
  if (!sid) return
  const id = ++seq
  error.value = ''
  const since = new Date(zonedDayStart(ctx.timezone.value, Date.now()))
  const jobs: Promise<unknown>[] = [
    getGmReport<DashboardReport>(sid, 'dashboard', `days=${RANGES[RANGES.length - 1]}`)
      .then((r) => {
        if (id === seq) report.value = r
      })
      .catch((e) => {
        if (id === seq) error.value = getRequestErrorMessage(e)
      }),
    listGmExecutions(gmExecutionsQuery(sid, { pageSize: 5 })).then(({ data }) => {
      if (id !== seq) return
      recent.value = data.items || []
      pending.value = data.pending || []
    }),
    listGmExecutions(gmExecutionsQuery(sid, { status: 'failed', since, pageSize: 1 })).then(({ data }) => {
      if (id !== seq) return
      failedToday.value = data.total || 0
      lastFailed.value = data.items?.[0]
    }),
  ]
  if (ctx.hasCommand('mail.broadcast')) {
    jobs.push(
      listGameMails({ serverId: sid, status: 'scheduled', q: '', page: 1, pageSize: 20 }).then(({ data }) => {
        if (id === seq) scheduledMails.value = data.items || []
      }),
    )
  }
  if (ctx.manifest.value?.configs?.length) {
    jobs.push(
      listGmConfigs(sid).then(({ data }) => {
        if (id === seq) drafts.value = (data.items || []).filter((d) => !!d.draft)
      }),
    )
  }
  await Promise.allSettled(jobs)
}

let timer: ReturnType<typeof setInterval> | undefined
const startTimer = () => {
  stopTimer()
  timer = setInterval(() => void load(), REFRESH_MS)
}
const stopTimer = () => {
  if (timer) clearInterval(timer)
  timer = undefined
}
let mountedOnce = false
onMounted(() => {
  void load()
  startTimer()
})
onActivated(() => {
  if (mountedOnce) void load()
  mountedOnce = true
  startTimer()
})
onDeactivated(stopTimer)
onBeforeUnmount(stopTimer)

// ---------- 链接 ----------
const gmPath = computed(() => `/game/gm/${encodeURIComponent(ctx.serverId.value)}`)
const tabLink = (key: string) => {
  const tab = GM_TABS.find((x) => x.key === key)
  return tab && tab.visible(ctx) ? { path: gmPath.value, query: { tab: key } } : undefined
}
const serverLink = computed(() => `/game/servers/${encodeURIComponent(ctx.serverId.value)}`)

// ---------- 图表颜色（游戏皮肤的 --m-gInk；主题、深浅色变化时重新读取） ----------
const root = ref<HTMLElement>()
const palette = usePaletteColors()
const chartColor = ref('')
const readColor = () => {
  const el = root.value
  chartColor.value = (el && getComputedStyle(el).getPropertyValue('--m-gInk').trim()) || palette.value.pInk
}
watch(palette, () => nextTick(readColor))
onMounted(readColor)

// ---------- KPI ----------
const clock = (ms: number) => (ms ? dayjs(ms).format('HH:mm') : '—')
const dateShort = (ms: number) => dayjs(ms).format('MM-DD HH:mm')
const changePct = (today: number, yesterday: number) => (yesterday > 0 ? Math.round(((today - yesterday) / yesterday) * 100) : undefined)
const tone = (v: number | undefined) => (v === undefined || v === 0 ? '' : v > 0 ? 'is-up' : 'is-down')

const kpis = computed(() => {
  const r = report.value
  if (!r) return []
  const players = tabLink('players')
  const gasha = tabLink('gasha')
  const drawPct = changePct(r.drawsToday.draws, r.drawsYesterday.draws)
  const activeDelta = r.activeToday - r.activeYesterday
  const jewels = r.drawsToday.jewelFree + r.drawsToday.jewelPaid
  return [
    {
      key: 'online',
      title: t('gmData.home.kpiOnline'),
      value: num(r.online),
      delta: '',
      tone: '',
      sub: r.onlinePeakAt ? t('gmData.home.peakAt', { n: num(r.onlinePeak), time: clock(r.onlinePeakAt) }) : t('gmData.home.peak', { n: num(r.onlinePeak) }),
      live: true,
      image: '',
      icon: '',
      to: players,
    },
    {
      key: 'active',
      title: t('gmData.home.kpiActive'),
      value: num(r.activeToday),
      delta: delta(activeDelta),
      tone: tone(activeDelta),
      sub: t('gmData.home.vsYesterday'),
      live: false,
      image: '',
      icon: 'HOutline:UsersIcon',
      to: players,
    },
    {
      key: 'new',
      title: t('gmData.home.kpiNew'),
      value: num(r.newAccountsToday),
      delta: '',
      tone: '',
      sub: t('gmData.home.registered', { n: num(r.registered) }),
      live: false,
      image: '',
      icon: 'HSolid:UserPlusIcon',
      to: players,
    },
    {
      key: 'draws',
      title: t('gmData.home.kpiDraws'),
      value: num(r.drawsToday.draws),
      delta: drawPct === undefined ? '' : `${drawPct > 0 ? '+' : drawPct < 0 ? '−' : '±'}${Math.abs(drawPct)}%`,
      tone: tone(drawPct),
      sub: drawPct === undefined ? t('gmData.home.drawRequests', { n: num(r.drawsToday.requests) }) : t('gmData.home.vsYesterday'),
      live: false,
      image: '',
      icon: 'HOutline:SparklesIcon',
      to: gasha,
    },
    {
      key: 'jewel',
      title: t('gmData.home.kpiJewel'),
      value: num(jewels),
      delta: '',
      tone: '',
      sub: t('gmData.home.paidShare', { n: pct(r.drawsToday.jewelPaid, jewels) }),
      live: false,
      image: catalogImage(ctx.game.value, 'currency', { id: 'jewelFree' }),
      icon: 'HOutline:SparklesIcon',
      to: gasha,
    },
  ]
})

// ---------- 趋势 ----------
const series = computed(() => (report.value?.series || []).slice(-days.value))
const dayLabel = (day: string, last: boolean) => {
  const d = dayjs(day)
  const base = `${d.format('MM-DD')}（${t(`gmData.weekday.${d.day()}`)}）`
  return last ? `${base} · ${t('gmData.home.today')}` : base
}
const activePoints = computed<TrendPoint[]>(() =>
  series.value.map((s, i, arr) => {
    const last = i === arr.length - 1
    const prev = arr[i - 1]
    return {
      axis: last ? t('gmData.home.today') : dayjs(s.day).format('MM-DD'),
      label: dayLabel(s.day, last),
      value: s.active,
      note: last
        ? t('gmData.home.asOf', { time: clock(report.value?.asOf || 0) })
        : prev
          ? t('gmData.home.vsPrev', { d: delta(s.active - prev.active) })
          : '',
    }
  }),
)
const drawPoints = computed<TrendPoint[]>(() =>
  series.value.map((s, i, arr) => {
    const last = i === arr.length - 1
    return {
      axis: last ? t('gmData.home.today') : dayjs(s.day).format('MM-DD'),
      label: dayLabel(s.day, last),
      value: s.draws,
      note: last ? t('gmData.home.asOf', { time: clock(report.value?.asOf || 0) }) : '',
      partial: last,
    }
  }),
)
const peakOf = (key: 'active' | 'draws') =>
  series.value.reduce((best, s) => (best && best[key] >= s[key] ? best : s), series.value[0])
const activeAria = computed(() => {
  const peak = peakOf('active')
  const last = series.value[series.value.length - 1]
  return peak && last ? t('gmData.home.activeAria', { day: peak.day, peak: peak.active, today: last.active }) : ''
})
const drawAria = computed(() => {
  const peak = peakOf('draws')
  const last = series.value[series.value.length - 1]
  return peak && last ? t('gmData.home.drawAria', { day: peak.day, peak: peak.draws, today: last.draws }) : ''
})

// ---------- 卡池 ----------
const GASHA_ROWS = 6
const gashas = computed(() =>
  [...(report.value?.gashas || [])]
    .sort((a, b) => b.today.draws - a.today.draws || Number(b.open) - Number(a.open) || b.total.draws - a.total.draws)
    .slice(0, GASHA_ROWS),
)
const maxToday = computed(() => Math.max(1, ...gashas.value.map((g) => g.today.draws)))
const barWidth = (g: GashaStats) => Math.round((g.today.draws / maxToday.value) * 100)
const gashaImage = (g: GashaStats) => catalogImage(ctx.game.value, 'gasha', { id: g.gashaId, assetId: g.bannerAssetId })
const gashaState = (g: GashaStats): { variant: 'success' | 'info' | 'primary' | 'neutral'; label: string } => {
  if (g.open) return g.endTime ? { variant: 'success', label: t('gmData.home.gashaOpen') } : { variant: 'info', label: t('gmData.home.permanent') }
  if (g.startTime > Date.now()) return { variant: 'primary', label: t('gmData.home.gashaSoon') }
  return { variant: 'neutral', label: t('gmData.home.gashaEnded') }
}

// ---------- 需要留意 ----------
interface Alert {
  key: string
  title: string
  sub: string
  icon: string
  tone: 'p' | 'a' | 'a2' | 'ok' | 'bad' | 'warn'
  where: string
  to?: ReturnType<typeof tabLink> | string
}

const configTab = (domain: string) => (domain === 'gasha' ? 'gasha' : domain === 'notice' ? 'notices' : 'calendar')

const alerts = computed<Alert[]>(() => {
  const out: Alert[] = []
  const r = report.value
  const status = ctx.status.value
  const now = Date.now()
  const logs = t('gm.tabs.logs')
  if (ctx.pendingApprovals.value > 0) {
    const first = pending.value[0]
    out.push({
      key: 'pending',
      title: t('gmData.home.alertPending', { n: ctx.pendingApprovals.value }),
      sub: first ? `${first.actorName || '—'} · ${first.commandTitle || first.command} · ${when(first.createTime)}` : '',
      icon: 'HOutline:IdentificationIcon',
      tone: 'p',
      where: logs,
      to: tabLink('logs'),
    })
  }
  if (failedToday.value > 0) {
    const e = lastFailed.value
    out.push({
      key: 'failed',
      title: t('gmData.home.alertFailed', { n: failedToday.value }),
      sub: e ? `${e.commandTitle || e.command}${e.errorCode ? ` · ${e.errorCode}` : ''} · ${when(e.createTime)}` : '',
      icon: 'HOutline:ExclamationTriangleIcon',
      tone: 'bad',
      where: logs,
      to: tabLink('logs'),
    })
  }
  if (r?.maintenanceEnabled) {
    out.push({
      key: 'maintenance',
      title: t('gmData.home.alertMaintenance'),
      sub: t('gmData.home.alertMaintenanceSub', { n: status?.maintenance?.whitelist?.length || 0 }),
      icon: 'HOutline:WrenchScrewdriverIcon',
      tone: 'warn',
      where: t('gmData.home.server'),
      to: serverLink.value,
    })
  }
  for (const h of status?.health || []) {
    if (h.ok) continue
    out.push({
      key: `health-${h.name}`,
      title: t('gmData.home.alertHealth', { name: h.name }),
      sub: h.detail,
      icon: 'HOutline:SignalIcon',
      tone: 'bad',
      where: t('gmData.home.server'),
      to: serverLink.value,
    })
  }
  const saves = status?.saves
  if (saves && saves.capacity > 0 && saves.waiting / saves.capacity >= 0.8) {
    out.push({
      key: 'saves',
      title: t('gmData.home.alertSaves'),
      sub: t('gmData.home.alertSavesSub', { waiting: saves.waiting, capacity: saves.capacity }),
      icon: 'HOutline:ArchiveBoxIcon',
      tone: 'warn',
      where: t('gmData.home.server'),
      to: serverLink.value,
    })
  }
  const mails = scheduledMails.value
    .map((m) => ({ m, at: toMs(m.scheduledAt) }))
    .filter((x) => x.at)
    .sort((a, b) => a.at - b.at)
  if (mails.length) {
    const next = mails[0]!
    out.push({
      key: 'mails',
      title: t('gmData.home.alertMail', { n: mails.length, time: when(next.at) }),
      sub: `${next.m.title} · ${mailTarget(next.m)}`,
      icon: 'HOutline:EnvelopeIcon',
      tone: 'a2',
      where: t('gm.tabs.mail'),
      to: tabLink('mail'),
    })
  }
  for (const d of drafts.value) {
    const tab = configTab(d.id)
    out.push({
      key: `draft-${d.id}`,
      title: t('gmData.home.alertDraft', { name: d.title || t('game.unnamed') }),
      sub: [d.draft?.authorName, d.draft?.updateTime ? when(d.draft.updateTime) : ''].filter(Boolean).join(' · '),
      icon: 'HOutline:PencilSquareIcon',
      tone: 'a',
      where: t(`gm.tabs.${tab}`),
      to: tabLink(tab),
    })
  }
  for (const g of r?.gashas || []) {
    if (!g.open || !g.endTime || g.endTime - now > DAY || g.endTime < now) continue
    out.push({
      key: `gasha-${g.gashaId}`,
      title: t('gmData.home.alertGashaEnding', { name: g.name || ctx.label('gasha', g.gashaId) }),
      sub: t('gmData.home.alertEndsAt', { time: when(g.endTime) }),
      icon: 'HOutline:SparklesIcon',
      tone: 'a',
      where: t('gm.tabs.gasha'),
      to: tabLink('gasha'),
    })
  }
  if (r?.pvp && r.pvp.endTime > now && r.pvp.endTime - now <= DAY) {
    out.push({
      key: 'pvp',
      title: t('gmData.home.alertPvpEnding'),
      sub: t('gmData.home.alertEndsAt', { time: when(r.pvp.endTime) }),
      icon: 'HOutline:TrophyIcon',
      tone: 'ok',
      where: t('gm.tabs.modes'),
      to: tabLink('modes'),
    })
  }
  return out
})

const mailTarget = (m: GameMail) =>
  m.target === 'all' ? t('gmData.home.mailAll') : m.target === 'players' ? t('gmData.home.mailPlayers', { n: m.players?.length || 0 }) : t('gmData.home.mailCondition')

// ---------- 玩法 ----------
const MODE_ICONS: Record<string, string> = {
  pvp: 'HOutline:TrophyIcon',
  produce: 'HOutline:SparklesIcon',
  seminar: 'HOutline:BuildingLibraryIcon',
  tower: 'HOutline:ChartBarIcon',
}
const modes = computed(() => {
  const r = report.value
  return (ctx.manifest.value?.modes || []).map((m) => {
    const out = { id: m.id, title: m.title || t('game.unnamed'), icon: MODE_ICONS[m.id] || 'HOutline:PuzzlePieceIcon', sub: '', value: '', unit: '' }
    if (!r) return out
    if (m.id === 'pvp') {
      if (r.pvp) {
        out.value = num(r.pvp.participants)
        out.unit = t('gmData.home.participants')
        out.sub = r.pvpTop
          ? t('gmData.home.pvpSubTop', { time: dateShort(r.pvp.endTime), name: r.pvpTop.name || t('gm.player.noName') })
          : t('gmData.home.pvpSub', { time: dateShort(r.pvp.endTime) })
      } else {
        out.sub = t('gmData.home.noSeason')
      }
    } else if (m.id === 'produce') {
      out.value = num(r.producesToday)
      out.unit = t('gmData.home.producesToday')
      const best = r.bestProduceToday
      out.sub = best
        ? t('gmData.home.produceBest', { grade: gradeLabel(best.grade), name: best.name || t('gm.player.noName') })
        : t('gmData.home.producesTotal', { n: num(r.producesTotal) })
    } else if (m.id === 'seminar') {
      out.value = num(r.seminarClearsToday)
      out.unit = t('gmData.home.clearsToday')
      const best = [...(r.seminar || [])].filter((g) => g.players > 0).sort((a, b) => a.order - b.order)[0]
      out.sub = best ? t('gmData.home.seminarSub', { name: best.name, n: pct(best.clearedPlayers, best.players) }) : ''
    } else if (m.id === 'tower') {
      out.sub = t('gmData.home.towerSub')
    }
    return out
  })
})

// ---------- 服务器 ----------
const serverFacts = computed(() => {
  const r = report.value
  const st = ctx.status.value
  const out: { key: string; label: string; value: string; mono?: boolean }[] = []
  if (!r) return out
  out.push({ key: 'uptime', label: t('gmData.home.uptime'), value: duration(r.uptimeSeconds * 1000) })
  if (ctx.server.value?.mgpLatencyMs) out.push({ key: 'latency', label: t('gmData.home.latency'), value: `${Math.round(ctx.server.value.mgpLatencyMs)} ms` })
  out.push({ key: 'version', label: t('gmData.home.version'), value: r.version || '—', mono: true })
  if (st?.resource?.app) out.push({ key: 'resource', label: t('gmData.home.resource'), value: `${st.resource.app} · ${st.resource.master}`, mono: true })
  if (st?.saves) out.push({ key: 'saves', label: t('gmData.home.saves'), value: `${num(st.saves.waiting)} / ${num(st.saves.capacity)}` })
  out.push({ key: 'registered', label: t('gmData.home.registeredLabel'), value: num(r.registered) })
  return out
})

// ---------- 最近 GM 操作 ----------
const logTitle = (e: GmExecution) => {
  const title = e.commandTitle || e.command
  const target = e.targets?.length === 1 ? e.targets[0]?.name || e.targets[0]?.pid : e.targets?.length ? t('gmData.home.targets', { n: e.targets.length }) : ''
  return target ? `${title} · ${target}` : title
}
const logSub = (e: GmExecution) =>
  `${e.actorName || '—'}${e.actorKind === 'player' ? ` · ${t('gmData.gmLog.kindPlayer')}` : ''} · ${when(e.createTime)}`
</script>

<style scoped lang="scss">
.gh {
  display: flex;
  flex-direction: column;
  gap: 18px;
  min-width: 0;
}
.gh__seg {
  display: inline-flex;
  gap: 4px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.gh__seg button {
  min-height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
.gh__seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: 0 1px 4px color-mix(in srgb, var(--m-ink) 14%, transparent);
}
.gh__strong {
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.gh__dim {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gh__ellipsis {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.gh__more {
  font-size: 14px;
  font-weight: 700;
  color: var(--m-pInk);
  text-decoration: none;
  white-space: nowrap;
}

/* KPI */
.gh__kpis {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
.gh__kpi {
  flex: 1 1 180px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 16px 18px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-card);
  color: var(--el-text-color-primary);
  text-decoration: none;
}
a.gh__kpi:hover {
  border-color: var(--m-gLine);
}
.gh__kpi-head {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 24px;
}
.gh__live {
  width: 10px;
  height: 10px;
  margin: 0 7px;
  border-radius: 50%;
  background: var(--m-ok);
  box-shadow: 0 0 0 4px var(--m-okSoft);
}
.gh__kpi-img {
  width: 24px;
  height: 24px;
  object-fit: contain;
  background: transparent;
}
.gh__kpi-icon {
  width: 22px;
  height: 22px;
  color: var(--m-gInk);
}
.gh__kpi-title {
  font-size: 13px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gh__kpi-value {
  font-size: 30px;
  font-weight: 800;
  line-height: 1.1;
}
.gh__kpi-sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gh__kpi-delta {
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.gh__kpi-delta.is-up {
  color: var(--m-ok);
}
.gh__kpi-delta.is-down {
  color: var(--m-bad);
}

/* 图表 */
.gh__charts {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
}
.gh__charts > * {
  flex: 1 1 420px;
  min-width: 0;
}

/* 卡池 */
.gh__pools {
  overflow-x: auto;
  scrollbar-width: none;
}
.gh__pools::-webkit-scrollbar {
  display: none;
}
.gh__pool {
  display: grid;
  grid-template-columns: 168px minmax(0, 1.4fr) minmax(0, 1.3fr) 90px 70px 100px;
  gap: 12px;
  align-items: center;
  min-width: 860px;
  min-height: 76px;
  padding: 0 4px;
  border-top: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-primary);
  text-decoration: none;
}
a.gh__pool:hover {
  background: var(--el-fill-color-light);
}
.gh__pool--head {
  min-height: 36px;
  border-top: 0;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.is-right {
  text-align: right;
}
.gh__banner {
  width: 160px;
  height: 56px;
  border-radius: 999px;
  object-fit: contain;
  background: var(--m-s2);
}
.gh__pool-name {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.gh__pool-today {
  display: flex;
  align-items: center;
  gap: 10px;
}
.gh__track {
  flex: 1;
  height: 10px;
  border-radius: 999px;
  background: var(--m-s2);
  overflow: hidden;
}
.gh__track span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--m-gInk);
}
.gh__pool-n {
  width: 96px;
  font-weight: 700;
}
.gh__pool-total,
.gh__pool-ssr {
  font-weight: 600;
}
.gh__pool-state {
  justify-self: start;
}

/* 下半区 */
.gh__row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}
.gh__alerts {
  flex: 3 1 520px;
  min-width: 0;
}
.gh__side {
  flex: 2 1 340px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gh__alert {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 60px;
  padding: 10px 4px;
  border-top: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-primary);
  text-decoration: none;
}
.gh__alert:first-of-type {
  border-top: 0;
}
a.gh__alert:hover {
  background: var(--el-fill-color-light);
}
.gh__alert-icon {
  width: 36px;
  height: 36px;
}
.gh__alert-icon svg {
  width: 18px;
  height: 18px;
}
.gh__alert-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gh__tag {
  flex: none;
  padding: 2px 10px;
  border-radius: 999px;
  background: var(--m-s2);
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gh__chev {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.gh__mode {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 56px;
  color: var(--el-text-color-primary);
  text-decoration: none;
}
.gh__mode-icon {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 40px;
  height: 40px;
  border-radius: 13px;
  background: var(--m-gSoft);
  color: var(--m-gInk);
}
.gh__mode-icon svg {
  width: 20px;
  height: 20px;
}
.gh__mode-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.gh__mode-value {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}
.gh__mode-value .m-num {
  font-size: 18px;
  font-weight: 800;
}
.gh__server {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 18px 20px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-card);
  color: var(--el-text-color-primary);
  text-decoration: none;
}
.gh__server:hover {
  border-color: var(--m-gLine);
}
.gh__server-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.gh__server-title {
  flex: 1;
  font-size: 18px;
  font-weight: 800;
}
.gh__server-facts {
  margin: 0;
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 8px 12px;
  font-size: 13px;
}
.gh__server-facts > div {
  display: contents;
}
.gh__server-facts dt {
  color: var(--el-text-color-secondary);
}
.gh__server-facts dd {
  margin: 0;
  font-weight: 700;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gh__log {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  border-top: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-primary);
  text-decoration: none;
}
.gh__log:first-of-type {
  border-top: 0;
}
.gh__log-av {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  border-radius: 50%;
  background: var(--m-a2Soft);
  color: var(--m-ink);
  font-size: 12px;
  font-weight: 800;
}
.gh__log-av.is-player {
  background: var(--m-gSoft);
  color: var(--m-gInk);
}
.gh__log-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.gh__skeleton {
  display: flex;
  flex-wrap: wrap;
  gap: 12px;
}
.gh__skel-kpi {
  flex: 1 1 180px;
  height: 110px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
}
.gh__mlabel {
  display: none;
}
@media (max-width: 760px) {
  .gh__pool {
    min-width: 0;
    grid-template-columns: minmax(0, 1fr) auto;
    grid-template-areas:
      'banner banner'
      'name state'
      'today today'
      'total ssr';
    gap: 8px;
    padding: 12px 4px;
  }
  .gh__pool--head {
    display: none;
  }
  .gh__banner {
    grid-area: banner;
  }
  .gh__pool-name {
    grid-area: name;
  }
  .gh__pool-state {
    grid-area: state;
    justify-self: end;
  }
  .gh__pool-today {
    grid-area: today;
  }
  .gh__pool-total {
    grid-area: total;
    text-align: left;
  }
  .gh__pool-ssr {
    grid-area: ssr;
  }
  .gh__mlabel {
    display: inline;
    font-family: inherit;
    font-size: 12px;
    font-weight: 500;
    color: var(--el-text-color-secondary);
  }
  .gh__kpi {
    flex-basis: calc(50% - 6px);
    padding: 14px;
  }
  .gh__kpi-value {
    font-size: 24px;
  }
  .gh__tag {
    display: none;
  }
  .gh__seg button {
    min-height: 44px;
  }
}
</style>
