<!-- GM 运营 · 工作台：开始干活的地方（设计见 design/game/GkHomePC）。
     页头一行服务器状态（「详情」弹窗看版本、资源、存档队列等）→ 今天的数字一行（在线、活跃、新增账号、抽卡，点进对应页）
     → 待处理（要人去做的事：待审批、今天失败的 GM 操作、未发布的配置草稿、维护与依赖异常、存档队列快满，每条一个去处理的按钮；
     没有时只一行）+ 接下来（7 天内到点的定时邮件、卡池结束、兑换码到期、赛季结束，按时间排）
     → 常用操作（找玩家 + 邮件补偿 / 兑换码 / 公告 / GM 操作入口，只跳转）+ 最近 GM 操作（没有时不显示）。
     趋势图和卡池表现不在这里：抽卡在「卡池」页、活跃在「玩家」页的趋势弹窗里。每 5 分钟自动刷新（标签可见时）。 -->
<template>
  <div class="gh game-skin">
    <PageHeader :back="false" :theme-toggle="false">
      <template #title>{{ t('gmData.home.title') }}</template>
      <template #description>
        <span v-if="report" class="gh__status">
          <RouterLink class="gh__status-srv" :to="serverLink">{{ ctx.server.value?.name || '' }}</RouterLink>
          <StatusPill
            :variant="report.maintenanceEnabled ? 'warning' : 'success'"
            :label="report.maintenanceEnabled ? t('gmData.home.maintenance') : t('gmData.home.running')"
          />
          <span v-for="f in serverLine" :key="f">{{ f }}</span>
          <UButton size="xs" color="neutral" variant="soft" icon="i-lucide-info" @click="serverOpen = true">{{ t('gmData.home.serverMore') }}</UButton>
        </span>
        <template v-else>{{ t('gmData.home.subtitleLoading') }}</template>
      </template>
    </PageHeader>

    <EmptyState
      v-if="error && !report"
      icon="HOutline:ExclamationTriangleIcon"
      :title="t('gmData.home.loadFailed')"
      :description="error"
    >
      <template #action>
        <UButton color="neutral" variant="outline" @click="load(true)">{{ t('gmData.common.retry') }}</UButton>
      </template>
    </EmptyState>

    <template v-else-if="report">
      <!-- 今天：只放数字 -->
      <section class="gh__today" :aria-label="t('gmData.home.today')">
        <span class="gh__today-label">{{ t('gmData.home.today') }}</span>
        <component :is="k.to ? RouterLink : 'span'" v-for="k in today" :key="k.key" :to="k.to" class="gh__today-item">
          <span v-if="k.live" class="gh__live" aria-hidden="true" />
          <span class="gh__dim">{{ k.title }}</span>
          <b class="m-num">{{ k.value }}</b>
          <span v-if="k.sub" class="gh__dim" :class="k.tone">{{ k.sub }}</span>
        </component>
      </section>

      <section class="gh__row">
        <div class="gh__main">
          <!-- 待处理 -->
          <AppPanel :title="todos.length ? `${t('gmData.home.todo')} ${todos.length}` : t('gmData.home.todo')">
            <p v-if="!todos.length" class="gh__calm">
              <UIcon name="i-lucide-circle-check" />{{ t('gmData.home.todoNone') }}
            </p>
            <div v-for="a in todos" :key="a.key" class="gh__todo">
              <span class="gh__todo-icon m-sticker" :class="`m-sticker--${a.tone}`" aria-hidden="true">
                <component :is="menuStore.iconComponents[a.icon]" />
              </span>
              <span class="gh__todo-text">
                <span class="gh__strong">{{ a.title }}</span>
                <span v-if="a.sub" class="gh__dim gh__ellipsis">{{ a.sub }}</span>
              </span>
              <UButton v-if="a.to" size="sm" color="neutral" variant="outline" :to="a.to">{{ a.btn }}</UButton>
            </div>
          </AppPanel>

          <!-- 接下来 -->
          <AppPanel v-if="upcoming.length" :title="t('gmData.home.upcoming')">
            <template v-if="tabLink('calendar')" #actions>
              <RouterLink class="gh__more" :to="tabLink('calendar')!">{{ t('gm.tabs.calendar') }} →</RouterLink>
            </template>
            <component :is="u.to ? RouterLink : 'div'" v-for="u in upcoming" :key="u.key" :to="u.to" class="gh__next">
              <span class="gh__next-at m-num">{{ when(u.at) }}</span>
              <span class="gh__next-t gh__ellipsis">{{ u.title }}</span>
              <span class="gh__tag">{{ u.where }}</span>
            </component>
          </AppPanel>
        </div>

        <div class="gh__side">
          <!-- 常用操作 -->
          <AppPanel :title="t('gmData.home.quick')">
            <form class="gh__find" role="search" @submit.prevent="findPlayer">
              <UInput v-model="playerQ" icon="i-lucide-search" size="lg" :placeholder="t('gmData.home.findPlayer')" :aria-label="t('gmData.home.findPlayer')" />
            </form>
            <div v-if="quick.length" class="gh__quick">
              <RouterLink v-for="q in quick" :key="q.key" class="gh__qa" :to="q.to">
                <component :is="menuStore.iconComponents[q.icon]" class="gh__qa-icon" />{{ q.title }}
              </RouterLink>
            </div>
          </AppPanel>

          <!-- 最近 GM 操作 -->
          <AppPanel v-if="recent.length" :title="t('gmData.home.recent')">
            <template v-if="tabLink('logs')" #actions>
              <RouterLink class="gh__more" :to="tabLink('logs')!">{{ t('gmData.home.recentMore') }}</RouterLink>
            </template>
            <component :is="tabLink('logs') ? RouterLink : 'div'" v-for="e in recent.slice(0, 4)" :key="e.id" :to="tabLink('logs')" class="gh__log">
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
      <span v-for="i in 3" :key="i" class="gh__skel" />
    </div>

    <!-- 弹窗：服务器详情 -->
    <FormDialog v-model="serverOpen" :title="ctx.server.value?.name || t('gmData.home.server')" :width="460">
      <dl class="gh__facts">
        <div v-for="f in serverFacts" :key="f.key">
          <dt>{{ f.label }}</dt>
          <dd :class="f.mono ? 'm-mono' : 'm-num'" :title="f.value">{{ f.value }}</dd>
        </div>
      </dl>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('gmData.common.close') }}</UButton>
        <UButton color="primary" trailing-icon="i-lucide-arrow-right" :to="serverLink">{{ t('gmData.home.openServer') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { RouterLink, useRouter } from 'vue-router'
import dayjs from 'dayjs'
import { gashaConfig, noticeConfig } from '@/api/gakumas'
import { listGmExecutions } from '@/api/gameGm'
import { listGameMails, listGiftCodeBatches } from '@/api/gameOps'
import { getRequestErrorMessage } from '@/utils/request'
import type { Dashboard, DrawTotals } from '@/games/gakumas/types'
import type { GmConfigDomain, GmExecution } from '@/types/v1/game_gm'
import type { GameMail, GiftCodeBatch } from '@/types/v1/game_ops'
import { useGmContext } from '../../context'
import { fetchDashboard } from '../../dashboard'
import { GM_TABS } from '../../tabs'
import GmRiskPill from '../../components/GmRiskPill.vue'
import { gmExecutionsQuery } from '../../player/shared/executions'
import { toMs, useGmFormat, zonedDayStart } from '../../player/shared/format'

defineOptions({ name: 'GmHomeTab' })

const REFRESH_MS = 5 * 60_000
const DAY = 86_400_000
/** 「接下来」只看这么远（7 天），最多列这么多条 */
const UPCOMING_WINDOW = 7 * DAY
const UPCOMING_MAX = 6

const { t } = useI18n()
const router = useRouter()
const ctx = useGmContext()
const menuStore = useMenuStore()
const { num, when, duration, delta } = useGmFormat()

const ZERO: DrawTotals = { requests: 0, draws: 0, ssr: 0, pickup: 0, jewelFree: 0, jewelPaid: 0 }
const report = shallowRef<Dashboard>()
const error = ref('')
const recent = shallowRef<GmExecution[]>([])
const pending = shallowRef<GmExecution[]>([])
const failedToday = ref(0)
const lastFailed = shallowRef<GmExecution>()
const scheduledMails = shallowRef<GameMail[]>([])
const codeBatches = shallowRef<GiftCodeBatch[]>([])
const drafts = shallowRef<GmConfigDomain[]>([])

// ---------- 读取 ----------
let seq = 0
const load = async (force = false) => {
  const sid = ctx.serverId.value
  if (!sid) return
  const id = ++seq
  error.value = ''
  const since = new Date(zonedDayStart(ctx.timezone.value, Date.now()))
  const jobs: Promise<unknown>[] = [
    fetchDashboard(sid, force)
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
    listGameMails({ serverId: sid, status: 'scheduled', q: '', page: 1, pageSize: 20 }).then(({ data }) => {
      if (id === seq) scheduledMails.value = data.items || []
    }),
    listGiftCodeBatches({ serverId: sid, status: 'active', q: '', page: 1, pageSize: 50 }).then(({ data }) => {
      if (id === seq) codeBatches.value = data.items || []
    }),
    Promise.all([gashaConfig.get(sid), noticeConfig.get(sid)]).then((docs) => {
      if (id === seq) drafts.value = docs.map((d) => d.domain).filter((d): d is GmConfigDomain => !!d?.draft)
    }),
  ]
  await Promise.allSettled(jobs)
}

let timer: ReturnType<typeof setInterval> | undefined
const startTimer = () => {
  stopTimer()
  timer = setInterval(() => void load(true), REFRESH_MS)
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

const clock = (ms: number) => (ms ? dayjs(ms).format('HH:mm') : '—')

// ---------- 页头：服务器状态一行 + 详情弹窗 ----------
const serverOpen = ref(false)
const serverLine = computed(() => {
  const r = report.value
  if (!r) return []
  const out = [`${t('gmData.home.uptime')} ${duration(r.uptimeSeconds * 1000)}`]
  if (ctx.server.value?.mgpLatencyMs) out.push(`${t('gmData.home.latency')} ${Math.round(ctx.server.value.mgpLatencyMs)} ms`)
  out.push(t('gmData.home.asOf', { time: clock(r.asOf) }))
  return out
})
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

// ---------- 今天 ----------
const tone = (v: number) => (v === 0 ? '' : v > 0 ? 'is-up' : 'is-down')
const today = computed(() => {
  const r = report.value
  if (!r) return []
  const players = tabLink('players')
  const drawsToday = (r.drawsToday || ZERO).draws
  const activeDelta = r.activeToday - r.activeYesterday
  return [
    { key: 'online', title: t('gmData.home.kpiOnline'), value: num(r.online), sub: t('gmData.home.peak', { n: num(r.onlinePeak) }), tone: '', live: true, to: players },
    {
      key: 'active',
      title: t('gmData.home.kpiActive'),
      value: num(r.activeToday),
      sub: `${t('gmData.home.vsYesterday')} ${delta(activeDelta)}`,
      tone: tone(activeDelta),
      live: false,
      to: players,
    },
    { key: 'new', title: t('gmData.home.kpiNew'), value: num(r.newAccountsToday), sub: '', tone: '', live: false, to: players },
    // 今天还没过完，不和昨天整天比；趋势在「卡池」页的弹窗里看
    { key: 'draws', title: t('gmData.home.kpiDraws'), value: num(drawsToday), sub: '', tone: '', live: false, to: tabLink('gasha') },
  ]
})

// ---------- 待处理：要人去做的事，每条一个去处理的按钮 ----------
interface Todo {
  key: string
  title: string
  sub: string
  icon: string
  tone: 'p' | 'a' | 'a2' | 'ok' | 'bad' | 'warn'
  btn: string
  to?: ReturnType<typeof tabLink> | string
}
const configTab = (domain: string) => (domain === 'gasha' ? 'gasha' : domain === 'notice' ? 'notices' : 'calendar')

const todos = computed<Todo[]>(() => {
  const out: Todo[] = []
  const r = report.value
  const status = ctx.status.value
  const view = t('gmData.home.goView')
  if (ctx.pendingApprovals.value > 0) {
    const first = pending.value[0]
    out.push({
      key: 'pending',
      title: t('gmData.home.alertPending', { n: ctx.pendingApprovals.value }),
      sub: first ? `${first.actorName || '—'} · ${first.commandTitle || first.command} · ${when(first.createTime)}` : '',
      icon: 'HOutline:IdentificationIcon',
      tone: 'p',
      btn: t('gmData.home.goApprove'),
      to: tabLink('logs'),
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
      btn: t('gmData.home.goPublish'),
      to: tabLink(tab),
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
      btn: view,
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
      btn: view,
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
      btn: view,
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
      btn: view,
      to: serverLink.value,
    })
  }
  return out
})

// ---------- 接下来：7 天内到点的事，按时间排 ----------
interface Upcoming {
  key: string
  at: number
  title: string
  where: string
  to?: ReturnType<typeof tabLink>
}
const mailTarget = (m: GameMail) =>
  m.target === 'all' ? t('gmData.home.mailAll') : m.target === 'players' ? t('gmData.home.mailPlayers', { n: m.players?.length || 0 }) : t('gmData.home.mailCondition')

const upcoming = computed<Upcoming[]>(() => {
  const now = Date.now()
  const soon = (at: number) => at > now && at - now <= UPCOMING_WINDOW
  const out: Upcoming[] = []
  for (const m of scheduledMails.value) {
    const at = toMs(m.scheduledAt)
    if (soon(at)) out.push({ key: `mail-${m.id}`, at, title: t('gmData.home.upMail', { title: m.title, target: mailTarget(m) }), where: t('gm.tabs.mail'), to: tabLink('mail') })
  }
  for (const g of report.value?.gashas || []) {
    if (g.open && g.endTime && soon(g.endTime)) {
      out.push({ key: `gasha-${g.gashaId}`, at: g.endTime, title: t('gmData.home.upGashaEnd', { name: g.name || ctx.label('gasha', g.gashaId) }), where: t('gm.tabs.gasha'), to: tabLink('gasha') })
    }
  }
  for (const b of codeBatches.value) {
    const at = toMs(b.endAt)
    if (!soon(at)) continue
    const used = b.totalLimit > 0 ? t('gmData.home.codeUsed', { n: num(b.usedCount), limit: num(b.totalLimit) }) : t('gmData.home.codeUsedNoLimit', { n: num(b.usedCount) })
    out.push({ key: `code-${b.id}`, at, title: t('gmData.home.upCodeEnd', { name: b.name || b.code || b.prefix, used }), where: t('gm.tabs.codes'), to: tabLink('codes') })
  }
  const pvp = report.value?.pvp
  if (pvp && soon(pvp.endTime)) out.push({ key: 'pvp', at: pvp.endTime, title: t('gmData.home.upPvpEnd'), where: t('gm.tabs.modes'), to: tabLink('modes') })
  return out.sort((a, b) => a.at - b.at).slice(0, UPCOMING_MAX)
})

// ---------- 常用操作 ----------
const playerQ = ref('')
const findPlayer = () => {
  const q = playerQ.value.trim()
  void router.push({ path: gmPath.value, query: { tab: 'players', ...(q ? { q } : {}) } })
}
const quick = computed(() =>
  [
    { key: 'mail', title: t('gmData.home.quickMail'), icon: 'HOutline:EnvelopeIcon' },
    { key: 'codes', title: t('gmData.home.quickCodes'), icon: 'HOutline:TicketIcon' },
    { key: 'notices', title: t('gmData.home.quickNotice'), icon: 'HOutline:MegaphoneIcon' },
    { key: 'operations', title: t('gmData.home.quickOps'), icon: 'HOutline:CommandLineIcon' },
  ].flatMap((q) => {
    const to = tabLink(q.key)
    return to ? [{ ...q, to }] : []
  }),
)

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
.is-up {
  color: var(--m-ok);
}
.is-down {
  color: var(--m-bad);
}

/* 页头的服务器状态行 */
.gh__status {
  display: inline-flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px 12px;
}
.gh__status-srv {
  font-weight: 700;
  color: var(--el-text-color-primary);
  text-decoration: none;
}
.gh__status-srv:hover {
  text-decoration: underline;
}

/* 今天 */
.gh__today {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px 28px;
  padding: 14px 22px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.gh__today-label {
  font-size: 13px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gh__today-item {
  display: inline-flex;
  align-items: baseline;
  gap: 8px;
  color: var(--el-text-color-primary);
  text-decoration: none;
}
.gh__today-item b {
  font-size: 22px;
  font-weight: 800;
}
a.gh__today-item:hover b {
  color: var(--m-pInk);
}
.gh__live {
  align-self: center;
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: var(--m-ok);
  box-shadow: 0 0 0 3px var(--m-okSoft);
}

/* 下半区 */
.gh__row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}
.gh__main {
  flex: 3 1 520px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gh__side {
  flex: 2 1 320px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gh__calm {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  padding: 4px 0;
  font-size: 14px;
  color: var(--el-text-color-secondary);
}
.gh__calm :deep(svg),
.gh__calm .iconify {
  color: var(--m-ok);
}

/* 待处理 */
.gh__todo {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 60px;
  padding: 10px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.gh__todo:first-of-type {
  border-top: 0;
}
.gh__todo-icon {
  width: 36px;
  height: 36px;
}
.gh__todo-icon svg {
  width: 18px;
  height: 18px;
}
.gh__todo-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

/* 接下来 */
.gh__next {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  min-height: 44px;
  border-top: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-primary);
  text-decoration: none;
}
.gh__next:first-of-type {
  border-top: 0;
}
a.gh__next:hover .gh__next-t {
  color: var(--m-pInk);
}
.gh__next-at {
  font-size: 13px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gh__next-t {
  font-size: 14px;
  font-weight: 600;
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

/* 常用操作 */
.gh__find {
  margin-bottom: 12px;
}
.gh__find :deep(> div) {
  width: 100%;
}
.gh__quick {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
.gh__qa {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 48px;
  padding: 0 14px;
  border-radius: 16px;
  background: var(--m-s2);
  color: var(--el-text-color-primary);
  font-size: 14px;
  font-weight: 700;
  text-decoration: none;
}
.gh__qa:hover {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.gh__qa-icon {
  flex: none;
  width: 18px;
  height: 18px;
}

/* 最近 GM 操作 */
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

/* 服务器详情弹窗 */
.gh__facts {
  margin: 0;
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 12px 20px;
  font-size: 0.875rem;
}
.gh__facts > div {
  display: contents;
}
.gh__facts dt {
  color: var(--el-text-color-secondary);
}
.gh__facts dd {
  margin: 0;
  min-width: 0;
  font-weight: 700;
  overflow-wrap: anywhere;
}

.gh__skeleton {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.gh__skel {
  height: 96px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
}
@media (max-width: 760px) {
  .gh__today {
    gap: 8px 18px;
    padding: 12px 16px;
  }
  .gh__next {
    grid-template-columns: minmax(0, 1fr) auto;
    grid-template-areas:
      'at tag'
      't t';
    gap: 2px 12px;
    padding: 8px 0;
  }
  .gh__next-at {
    grid-area: at;
  }
  .gh__next-t {
    grid-area: t;
  }
  .gh__tag {
    grid-area: tag;
    justify-self: end;
  }
}
</style>
