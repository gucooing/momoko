<!-- 概览：在线 / 注册 / CPU·内存 / 存档队列（MGP status 与进程占用）、状态探测（端口、MGP、数据库、资源服务器）、
     实时动态（转发 MGP /events 的 SSE）。没有的数据不显示。 -->
<template>
  <div class="ov">
    <div class="ov__stats">
      <div class="ov__stat">
        <span class="ov__label">{{ t('gameServers.overview.online') }}</span>
        <span class="ov__value"><b class="m-num">{{ st?.online ?? '—' }}</b><small v-if="st?.onlinePeak != null">{{ t('gameServers.overview.peak', { n: st.onlinePeak }) }}</small></span>
        <div v-if="st?.onlinePeak" class="ov__bar"><span :style="{ width: pct(st.online || 0, st.onlinePeak), background: 'var(--m-gInk, var(--m-p))' }" /></div>
      </div>
      <div class="ov__stat">
        <span class="ov__label">{{ t('gameServers.overview.registered') }}</span>
        <span class="ov__value"><b class="m-num">{{ st?.registered ?? '—' }}</b><small v-if="st?.newToday != null">{{ t('gameServers.overview.newToday', { n: st.newToday }) }}</small></span>
      </div>
      <div v-if="hosted" class="ov__stat">
        <span class="ov__label">{{ t('gameServers.overview.cpuMem') }}</span>
        <span class="ov__value">
          <b class="m-num">{{ cpu.toFixed(1) }}%</b>
          <small>{{ fmtBytes(mem) }}<template v-if="overview?.cpuCores"> · {{ t('gameServers.overview.cpuOf', { n: overview.cpuCores }) }}</template></small>
        </span>
        <div v-if="overview?.cpuCores" class="ov__bar"><span :style="{ width: pct(cpu, overview.cpuCores * 100), background: 'var(--m-a2)' }" /></div>
      </div>
      <div v-if="st?.saves" class="ov__stat">
        <span class="ov__label">{{ t('gameServers.overview.saves') }}</span>
        <span class="ov__value"><b class="m-num">{{ st.saves.waiting ?? 0 }}</b><small>{{ t('gameServers.overview.savesSub', { n: st.saves.capacity ?? 0 }) }}</small></span>
        <div class="ov__bar"><span :style="{ width: pct(st.saves.waiting || 0, st.saves.capacity || 1), background: 'var(--m-ok)' }" /></div>
      </div>
    </div>
    <p v-if="st?.maintenance?.enabled" class="ov__notice">{{ t('gameServers.overview.maintenance') }}</p>

    <div class="ov__row">
      <section class="ov__card ov__probes">
        <h2>{{ t('gameServers.overview.probes') }}</h2>
        <div v-for="p in probes" :key="p.key" class="ov__probe">
          <StatusPill :variant="checkVariant(p.state)" :label="t(`gameServers.overview.probeState.${p.state}`, p.state)" class="ov__probe-pill" />
          <span class="ov__probe-text">
            <b>{{ t(`gameServers.overview.probe.${p.key}`, p.key) }}</b>
            <small>{{ probeDesc(p) }}</small>
          </span>
          <span v-if="p.latencyMs" class="ov__probe-lat m-mono">{{ p.latencyMs.toFixed(p.latencyMs < 10 ? 1 : 0) }} ms</span>
        </div>
        <template v-if="hosted && overview?.processes.length">
          <h3>{{ t('gameServers.overview.processes') }}</h3>
          <div v-for="p in overview.processes" :key="p.role" class="ov__probe">
            <StatusPill :variant="stateVariant(p.state)" :label="t(`gameServers.state.${p.state}`, p.state)" class="ov__probe-pill" />
            <span class="ov__probe-text">
              <b>{{ t(`gameServers.role.${p.role}`, p.role) }}</b>
              <small v-if="p.pid" class="m-num">CPU {{ p.cpuPercent.toFixed(1) }}% · {{ fmtBytes(p.memoryBytes) }}</small>
            </span>
            <span v-if="p.startTime" class="ov__probe-lat m-num">{{ fmtDuration(t, now - toMs(p.startTime)) }}</span>
          </div>
        </template>
      </section>

      <section class="ov__card ov__feed">
        <div class="ov__feed-head">
          <h2>{{ t('gameServers.overview.feed') }}</h2>
          <template v-if="server.brief?.mgpConfigured">
            <StatusPill :variant="feedVariant" :label="feedLabel" />
            <UButton v-if="feedState === 'off'" size="sm" color="neutral" variant="ghost" @click="connect">{{ t('gameServers.overview.feedReconnect') }}</UButton>
          </template>
        </div>
        <p v-if="!server.brief?.mgpConfigured" class="ov__muted">{{ t('gameServers.overview.feedNoMgp') }}</p>
        <p v-else-if="!events.length" class="ov__muted">{{ t('gameServers.overview.feedEmpty') }}</p>
        <ul v-else class="ov__events">
          <li v-for="ev in events" :key="ev.seq" class="ov__event">
            <GameImage v-if="ev.image" :src="ev.image" class="ov__event-img" />
            <span v-else class="ov__event-ico" aria-hidden="true"><UIcon :name="ev.icon" /></span>
            <span class="ov__event-text">{{ ev.text }}</span>
            <span class="ov__event-at m-num">{{ fmtAgo(t, ev.time, now) }}</span>
          </li>
        </ul>
      </section>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onBeforeUnmount, onDeactivated, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { gakumas } from '@/games'
import { gameServerEventsPath, getGameServerOverview } from '@/api/gameServer'
import { normalizeAuthToken, toBearerAuthHeader } from '@/utils/request'
import type { GameAdapter, GameServerInfo, GameServerOverview, GameServerProbe } from '@/types/v1/game_server'
import { checkVariant, fmtAgo, fmtBytes, fmtDuration, stateVariant, toMs, type PillVariant, type StatusDoc } from '../../shared'

const props = defineProps<{ server: GameServerInfo; adapter?: GameAdapter | null }>()
const { t } = useI18n()

const overview = ref<GameServerOverview | null>(null)
const now = ref(Date.now())
const id = computed(() => props.server.brief?.id || '')
const hosted = computed(() => props.server.brief?.mode === 'hosted')
const st = computed<StatusDoc | null>(() => {
  const v = (overview.value?.status ?? props.server.status) as unknown
  return v && typeof v === 'object' ? (v as StatusDoc) : null
})
const cpu = computed(() => (overview.value?.processes || []).reduce((n, p) => n + p.cpuPercent, 0))
const mem = computed(() => (overview.value?.processes || []).reduce((n, p) => n + Number(p.memoryBytes || 0), 0))
const probes = computed(() => overview.value?.probes || [])
const pct = (v: number, max: number) => `${Math.min(100, Math.max(0, max > 0 ? (v / max) * 100 : 0)).toFixed(1)}%`

const probeDesc = (p: GameServerProbe) => {
  const params = p.params || {}
  switch (p.key) {
    case 'grpc':
    case 'http':
      return t(`gameServers.overview.probeDesc.${p.key}`, params)
    case 'mgp':
      if (p.state === 'pending') return t('gameServers.overview.probeDesc.mgpNone')
      return p.error || t('gameServers.overview.probeDesc.mgp', { endpoint: params.endpoint })
    case 'database':
      return p.error || t('gameServers.overview.probeDesc.database', params)
    case 'saves':
      return t('gameServers.overview.probeDesc.saves', params)
    default:
      return p.error || params.detail || ''
  }
}

const load = async () => {
  if (!id.value) return
  try {
    overview.value = (await getGameServerOverview(id.value)).data
  } catch {
    // 下次轮询再试
  }
}

// ---- 实时动态（SSE）----
type FeedItem = { seq: number; time: number; text: string; icon: string; image: string }
type RawEvent = { kind: string; time?: number; accountId?: number; publicUserId?: string; name?: string; data?: Record<string, unknown> }
const events = ref<FeedItem[]>([])
const feedState = ref<'off' | 'connecting' | 'on'>('off')
let abort: AbortController | null = null
let seq = 0

const feedVariant = computed<PillVariant>(() => (feedState.value === 'on' ? 'success' : feedState.value === 'connecting' ? 'info' : 'neutral'))
const feedLabel = computed(() =>
  t(feedState.value === 'on' ? 'gameServers.overview.feedOn' : feedState.value === 'connecting' ? 'gameServers.overview.feedConnecting' : 'gameServers.overview.feedOff'),
)

const ICONS: Record<string, string> = {
  'session.login': 'i-lucide-log-in',
  'session.disconnect': 'i-lucide-unplug',
  'session.kick': 'i-lucide-user-x',
  'gasha.draw': 'i-lucide-sparkles',
  'produce.finish': 'i-lucide-star',
  'gm.command': 'i-lucide-wand-sparkles',
  'config.publish': 'i-lucide-file-cog',
}
const describe = (ev: RawEvent): FeedItem => {
  const d = ev.data || {}
  const who = ev.name || (d.operator as string) || t('gameServers.overview.someone')
  const game = props.server.brief?.game
  let text = ''
  let image = ''
  switch (ev.kind) {
    case 'session.login':
      text = t('gameServers.overview.event.session.login', { who, platform: d.platform || '—' })
      break
    case 'session.disconnect':
    case 'session.kick':
      text = t(`gameServers.overview.event.${ev.kind}`, { who })
      break
    case 'gasha.draw':
      text = t('gameServers.overview.event.gasha.draw', { who, n: d.drawCount ?? 0, ssr: d.ssrCount ?? 0, pickup: d.pickupCount ?? 0 })
      if (game === 'gakumas' && typeof d.gashaId === 'string') image = gakumas.gashaBanner(d.gashaId)
      break
    case 'produce.finish':
      text = t('gameServers.overview.event.produce.finish', { who, score: d.score ?? '—' })
      if (game === 'gakumas' && typeof d.idolCardId === 'string') image = gakumas.idolCardImage(d.idolCardId)
      break
    case 'gm.command':
      text = t('gameServers.overview.event.gm.command', {
        who: (d.operator as string) || t('gameServers.overview.system'),
        command: d.command ?? '',
        n: Array.isArray(d.targets) ? d.targets.length : 0,
      })
      break
    case 'config.publish':
      text = t('gameServers.overview.event.config.publish', { who: (d.operator as string) || t('gameServers.overview.system'), domain: d.domain ?? '' })
      break
    default:
      text = t('gameServers.overview.event.other', { kind: ev.kind })
  }
  return { seq: ++seq, time: ev.time || Date.now(), text, icon: ICONS[ev.kind] || 'i-lucide-activity', image }
}

const connect = async () => {
  if (!props.server.brief?.mgpConfigured || abort) return
  abort = new AbortController()
  feedState.value = 'connecting'
  const base = String(import.meta.env.VITE_API_BASE_URL || '/api/v1').replace(/\/$/, '')
  const token = normalizeAuthToken(localStorage.getItem('accessToken') || '')
  try {
    const res = await fetch(`${base}${gameServerEventsPath(id.value)}`, {
      headers: { Authorization: toBearerAuthHeader(token), Accept: 'text/event-stream' },
      signal: abort.signal,
    })
    if (!res.ok || !res.body) throw new Error(String(res.status))
    feedState.value = 'on'
    const reader = res.body.getReader()
    const decoder = new TextDecoder()
    let buf = ''
    for (;;) {
      const { done, value } = await reader.read()
      if (done) break
      buf += decoder.decode(value, { stream: true })
      let i: number
      while ((i = buf.indexOf('\n\n')) >= 0) {
        const block = buf.slice(0, i)
        buf = buf.slice(i + 2)
        let kind = ''
        const data: string[] = []
        for (const line of block.split('\n')) {
          if (line.startsWith('event:')) kind = line.slice(6).trim()
          else if (line.startsWith('data:')) data.push(line.slice(5).trimStart())
        }
        if (!data.length || kind === 'error') continue
        try {
          const ev = JSON.parse(data.join('\n')) as RawEvent
          if (!ev.kind) ev.kind = kind
          events.value = [describe(ev), ...events.value].slice(0, 30)
        } catch {
          // 跳过无法解析的一条
        }
      }
    }
  } catch {
    // 断开或被取消
  } finally {
    abort = null
    feedState.value = 'off'
  }
}
const disconnect = () => {
  abort?.abort()
  abort = null
}

let timer: ReturnType<typeof setInterval> | undefined
const startTimers = () => {
  clearInterval(timer)
  timer = setInterval(() => {
    now.value = Date.now()
    if (document.visibilityState === 'visible') load()
  }, 5000)
}
onMounted(() => {
  load()
  connect()
  startTimers()
})
onActivated(() => {
  load()
  connect()
  startTimers()
})
onDeactivated(() => {
  clearInterval(timer)
  disconnect()
})
onBeforeUnmount(() => {
  clearInterval(timer)
  disconnect()
})
</script>

<style scoped lang="scss">
.ov {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.ov__stats {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: 12px;
}
.ov__stat,
.ov__card {
  border-radius: var(--app-radius-lg);
  border: var(--cmp-card-border-width, 1px) solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.ov__stat {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 16px 18px;
}
.ov__label {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.ov__value {
  display: flex;
  align-items: baseline;
  flex-wrap: wrap;
  gap: 4px 8px;
}
.ov__value b {
  font-family: var(--app-font-display);
  font-size: 1.75rem;
  font-weight: 800;
}
.ov__value small {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.ov__bar {
  height: 6px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}
.ov__bar span {
  display: block;
  height: 100%;
  border-radius: 999px;
}
.ov__notice {
  margin: 0;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 0.875rem;
  font-weight: 700;
}
.ov__row {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}
.ov__card {
  min-width: 0;
  display: flex;
  flex-direction: column;
  padding: 18px 20px;
}
.ov__card h2 {
  margin: 0 0 8px;
  font-size: 1.0625rem;
  font-weight: 800;
}
.ov__card h3 {
  margin: 14px 0 4px;
  font-size: 0.875rem;
  font-weight: 800;
}
.ov__probes {
  flex: 3 1 480px;
}
.ov__feed {
  flex: 2 1 320px;
}
.ov__probe {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 52px;
  border-top: 1px solid var(--el-border-color-light);
}
.ov__probe-pill {
  flex: none;
  min-width: 72px;
  justify-content: center;
}
.ov__probe-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
  font-size: 0.875rem;
}
.ov__probe-text small {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.ov__probe-lat {
  flex: none;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.ov__feed-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}
.ov__feed-head h2 {
  flex: 1;
  margin: 0;
}
.ov__muted {
  margin: 0;
  padding: 12px 0;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
}
.ov__events {
  margin: 0;
  padding: 0;
  list-style: none;
  max-height: 520px;
  overflow-y: auto;
}
.ov__event {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 52px;
  border-top: 1px solid var(--el-border-color-light);
}
.ov__event-img,
.ov__event-ico {
  flex: none;
  width: 34px;
  height: 34px;
  border-radius: 50%;
  object-fit: cover;
}
.ov__event-ico {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
}
.ov__event-text {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  line-height: 1.5;
  overflow-wrap: anywhere;
}
.ov__event-at {
  flex: none;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
</style>
