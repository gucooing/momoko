<!-- 性能监控（MonitorPC / MonitorMobile）：页头（节点 · 采样间隔 · 保留时长 · 已记录天数 + 实时开关 + 监控设置）
     → 时间范围（1 小时 / 6 小时 / 24 小时 / 7 天 / 30 天 / 自定义，图表上拖动框选也会变成自定义时段）
     → 五个概览块（CPU、GPU、内存、磁盘读写、网络：当前值 + 迷你走势 + 时段平均 / 峰值）
     → 曲线卡（PC 全部铺开；手机点概览块切换下面的一张）→ 磁盘容量与写满预测 → 占用最高的应用。
     数据来自后端分级降采样的历史（查询时按时段自动选精度），服务停止的空档画成斜纹「无数据」。 -->
<template>
  <div class="mon" :class="{ 'mon--m': compact }">
    <PageHeader :title="t('monitor.title')">
      <template #description>
        <template v-if="compact">
          <span class="mon-pill" :class="live && !custom ? 'is-live' : 'is-off'">
            {{ live && !custom ? t('monitor.liveShort') : t('monitor.pausedShort') }}
          </span>
          <span>{{ t('monitor.mobileSummary', { interval: intervalText, retain: retainText }) }}</span>
        </template>
        <template v-else>
          <i18n-t keypath="monitor.summary" tag="span" scope="global">
            <template #host>{{ summaryHost }}</template>
            <template #interval><b class="m-num mon-strong">{{ intervalText }}</b></template>
            <template #retain><b class="m-num mon-strong">{{ retainText }}</b></template>
          </i18n-t>
          <template v-if="settings && !settings.recordEnabled"> · <span class="mon-warn">{{ t('monitor.notRecording') }}</span></template>
          <template v-else-if="recordedText"> · {{ recordedText }}</template>
        </template>
      </template>
      <template #actions>
        <template v-if="!compact">
          <button
            type="button"
            class="mon-live"
            :class="{ 'is-on': live && !custom }"
            :aria-pressed="live && !custom"
            @click="toggleLive"
          >
            <span class="mon-live__dot" />{{ live && !custom ? t('monitor.live') : t('monitor.paused') }}
          </button>
          <UButton color="neutral" variant="outline" @click="settingsOpen = true">
            <MonitorGlyph name="settings" class="mon-btn-ic" />{{ t('monitor.settings') }}
          </UButton>
        </template>
        <AppIconButton
          v-else
          icon="HOutline:Cog6ToothIcon"
          :label="t('monitor.settings')"
          :box="44"
          @click="settingsOpen = true"
        />
      </template>
    </PageHeader>

    <!-- 时间范围 -->
    <div v-if="!compact" class="mon-bar">
      <div class="mon-seg" role="tablist" :aria-label="t('monitor.rangeLabel')">
        <button
          v-for="r in RANGES"
          :key="r.key"
          type="button"
          role="tab"
          :class="{ 'is-on': !custom && range === r.key }"
          :aria-selected="!custom && range === r.key"
          @click="pickRange(r.key)"
        >
          {{ t(`monitor.ranges.${r.key}`) }}
        </button>
      </div>
      <AppDropdown align="start" :width="340">
        <template #trigger>
          <button type="button" class="mon-custom" :class="{ 'is-on': !!custom }" :aria-label="t('monitor.customTitle')" @click="prefillCustom">
            <MonitorGlyph name="cal" class="mon-btn-ic" />{{ custom ? customText : t('monitor.custom') }}
          </button>
        </template>
        <template #default="{ close }">
          <div class="mon-pop" role="dialog" :aria-label="t('monitor.customTitle')">
            <span class="mon-pop__t">{{ t('monitor.customTitle') }}</span>
            <label class="mon-pop__f">
              {{ t('monitor.customStart') }}
              <input v-model="customForm.start" type="datetime-local" class="mon-pop__in" />
            </label>
            <label class="mon-pop__f">
              {{ t('monitor.customEnd') }}
              <input v-model="customForm.end" type="datetime-local" class="mon-pop__in" />
            </label>
            <span class="mon-pop__hint">{{ t('monitor.customHint', { retain: retainText }) }}</span>
            <div class="mon-pop__acts">
              <UButton size="sm" color="neutral" variant="soft" @click="close">{{ t('monitor.cancel') }}</UButton>
              <UButton size="sm" color="primary" @click="applyCustom(close)">{{ t('monitor.view') }}</UButton>
            </div>
          </div>
        </template>
      </AppDropdown>
      <button v-if="custom" type="button" class="mon-back" @click="pickRange(range)">
        {{ t('monitor.backTo', { range: t(`monitor.ranges.${range}`) }) }}
      </button>
      <span class="mon-bar__sp" />
      <span class="mon-bar__hint">{{ t('monitor.hint') }}</span>
    </div>
    <div v-else class="mon-chips" role="tablist" :aria-label="t('monitor.rangeLabel')">
      <button
        v-for="r in RANGES"
        :key="r.key"
        type="button"
        role="tab"
        class="mon-chip"
        :class="{ 'is-on': !custom && range === r.key }"
        :aria-selected="!custom && range === r.key"
        @click="pickRange(r.key)"
      >
        {{ t(`monitor.ranges.${r.key}`) }}
      </button>
      <AppDropdown align="end" :width="340">
        <template #trigger>
          <button type="button" class="mon-chip" :class="{ 'is-on': !!custom }" @click="prefillCustom">
            <MonitorGlyph name="cal" class="mon-btn-ic" />{{ custom ? customText : t('monitor.custom') }}
          </button>
        </template>
        <template #default="{ close }">
          <div class="mon-pop" role="dialog" :aria-label="t('monitor.customTitle')">
            <span class="mon-pop__t">{{ t('monitor.customTitle') }}</span>
            <label class="mon-pop__f">
              {{ t('monitor.customStart') }}
              <input v-model="customForm.start" type="datetime-local" class="mon-pop__in" />
            </label>
            <label class="mon-pop__f">
              {{ t('monitor.customEnd') }}
              <input v-model="customForm.end" type="datetime-local" class="mon-pop__in" />
            </label>
            <span class="mon-pop__hint">{{ t('monitor.customHint', { retain: retainText }) }}</span>
            <div class="mon-pop__acts">
              <UButton size="sm" color="neutral" variant="soft" @click="close">{{ t('monitor.cancel') }}</UButton>
              <UButton size="sm" color="primary" @click="applyCustom(close)">{{ t('monitor.view') }}</UButton>
            </div>
          </div>
        </template>
      </AppDropdown>
    </div>

    <!-- 概览 -->
    <div class="mon-tiles">
      <component
        :is="compact ? 'button' : 'a'"
        v-for="k in tiles"
        :key="k.id"
        class="mon-tile"
        :class="{ 'is-on': compact && sel === k.id }"
        :href="compact ? undefined : `#mon-${k.id}`"
        :type="compact ? 'button' : undefined"
        :aria-pressed="compact ? sel === k.id : undefined"
        @click="onTile($event, k.id)"
      >
        <span class="mon-tile__h">
          <span class="mon-tile__stk m-sticker" :class="`m-sticker--${k.tone}`" :style="{ transform: `rotate(${k.rot})` }">
            <MonitorGlyph :name="k.glyph" />
          </span>
          <span class="mon-tile__t">{{ k.title }}</span>
        </span>
        <span class="mon-tile__v">
          <span class="mon-tile__n m-num">{{ k.value }}</span>
          <span class="mon-tile__u m-num">{{ k.unit }}</span>
        </span>
        <span class="mon-tile__sub">{{ k.sub }}</span>
        <svg class="mon-tile__spark" viewBox="0 0 200 40" preserveAspectRatio="none" aria-hidden="true">
          <path :d="k.spark.area" :style="{ fill: k.color, opacity: 0.14 }" />
          <path :d="k.spark.line" vector-effect="non-scaling-stroke" :style="{ fill: 'none', stroke: k.color, strokeWidth: compact ? 1.8 : 2 }" />
        </svg>
        <span v-if="!compact" class="mon-tile__ap m-num">
          <span>{{ t('monitor.avg', { v: k.avg }) }}</span><span>{{ t('monitor.peak', { v: k.peak }) }}</span>
        </span>
      </component>
    </div>

    <!-- 曲线 -->
    <div v-if="!compact" class="mon-charts">
      <MonitorChartCard
        v-for="c in cards"
        :key="c.id"
        :card-id="`mon-${c.id}`"
        :class="{ 'mon-charts__wide': c.id === 'disk' }"
        v-bind="cardProps(c)"
        :zoomable="true"
        @toggle="toggleLine"
        @pick="(v) => onPick(c.id, v)"
        @zoom="onZoom"
        @action="settingsOpen = true"
      />
    </div>
    <MonitorChartCard
      v-else-if="selectedCard"
      :key="selectedCard.id"
      v-bind="cardProps(selectedCard)"
      compact
      @toggle="toggleLine"
      @pick="(v) => onPick(selectedCard!.id, v)"
      @action="settingsOpen = true"
    />

    <div class="mon-bottom">
      <!-- 磁盘容量与趋势 -->
      <section class="mon-panel" :aria-label="t('monitor.disks.title')">
        <div class="mon-panel__h">
          <span class="mon-panel__t">{{ t('monitor.disks.title') }}</span>
          <span v-if="!compact" class="mon-panel__hint">{{ t('monitor.disks.hint') }}</span>
        </div>
        <template v-if="settings && !settings.collectDisk">
          <EmptyState icon="HOutline:CircleStackIcon" :title="t('monitor.off.title', { name: t('monitor.panel.disk') })" :description="t('monitor.off.desc')" />
        </template>
        <EmptyState v-else-if="!disks.length" icon="HOutline:CircleStackIcon" :title="t('monitor.disks.empty')" />
        <div v-for="d in disks" v-else :key="d.mountpoint" class="mon-disk">
          <div class="mon-disk__h">
            <span class="mon-disk__m m-mono">{{ d.mountpoint }}</span>
            <span v-if="!compact" class="mon-disk__fs">{{ d.fs }}</span>
            <span class="mon-disk__sp" />
            <span class="mon-disk__u m-num">{{ d.used }} / {{ d.total }}</span>
          </div>
          <div class="mon-disk__bar"><span :style="{ width: d.pctW, background: d.color }" /></div>
          <div class="mon-disk__f">
            <span class="mon-disk__pct m-num" :style="{ color: d.color }">{{ d.pct }}</span>
            <span class="mon-disk__grow">{{ d.grow }}</span>
            <span class="mon-disk__sp" />
            <span class="mon-disk__eta" :style="{ color: d.etaColor }">{{ compact ? d.etaShort : d.eta }}</span>
          </div>
        </div>
      </section>

      <!-- 占用最高的应用 -->
      <section class="mon-panel mon-panel--apps" :aria-label="t('monitor.apps.title')">
        <div class="mon-panel__h">
          <span class="mon-panel__t">{{ t('monitor.apps.title') }}</span>
          <div class="mon-seg mon-seg--sm" role="radiogroup" :aria-label="t('monitor.apps.sortLabel')">
            <button
              v-for="s in (['cpu', 'mem'] as const)"
              :key="s"
              type="button"
              role="radio"
              :class="{ 'is-on': topSort === s }"
              :aria-checked="topSort === s"
              @click="topSort = s"
            >
              {{ s === 'cpu' ? t('monitor.apps.sortCpu') : t('monitor.apps.sortMem') }}
            </button>
          </div>
        </div>
        <span v-if="!compact" class="mon-panel__hint mon-panel__hint--row">{{ t('monitor.apps.hint') }}</span>
        <EmptyState
          v-if="settings && !settings.collectApps"
          icon="HOutline:Squares2X2Icon"
          :title="t('monitor.off.title', { name: t('monitor.panel.app') })"
          :description="t('monitor.apps.offDesc')"
        />
        <EmptyState v-else-if="!apps.length" icon="HOutline:Squares2X2Icon" :title="t('monitor.apps.empty')" />
        <RouterLink
          v-for="a in apps"
          v-else
          :key="a.id"
          class="mon-app"
          :to="{ path: `/instance/app/${a.id}`, query: { tabTitle: a.name } }"
        >
          <span class="mon-app__av m-sticker" :class="`m-sticker--${a.tone}`" :style="{ transform: `rotate(${a.rot})` }">{{ a.letter }}</span>
          <span class="mon-app__main">
            <span class="mon-app__n">
              {{ a.name }}
              <span v-if="!a.running" class="mon-app__stop">{{ t('monitor.apps.stopped') }}</span>
            </span>
            <span class="mon-app__bar"><span :style="{ width: a.w }" /></span>
          </span>
          <template v-if="compact">
            <span class="mon-app__vals">
              <span class="mon-app__v m-num">{{ topSort === 'cpu' ? a.cpu : a.mem }}</span>
              <span class="mon-app__v2 m-num">{{ topSort === 'cpu' ? a.mem : a.cpu }}</span>
            </span>
          </template>
          <template v-else>
            <span class="mon-app__cpu m-num">{{ a.cpu }}</span>
            <span class="mon-app__mem m-num">{{ a.mem }}</span>
            <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" class="mon-app__go" />
          </template>
        </RouterLink>
      </section>
    </div>

    <MonitorSettings
      v-model="settingsOpen"
      :node-name="hostName"
      :can-edit="canEdit"
      :compact="compact"
      :range="exportRange"
      @saved="onSaved"
      @cleared="load()"
    />
  </div>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { useDocumentVisibility } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { getMonitorData } from '@/api/monitor'
import type { GetMonitorDataResponse, MonitorSettings as MonitorSettingsInfo } from '@/types/v1/monitor'
import { usePaletteColors } from '@/composables/usePaletteColors'
import MonitorChartCard, { type CardEmpty, type CardFact, type CardLine } from './components/MonitorChartCard.vue'
import MonitorGlyph, { type MonitorGlyphName } from './components/MonitorGlyph.vue'
import MonitorSettings from './components/MonitorSettings.vue'
import {
  durationParts,
  fmtCount,
  fmtPct,
  fmtPerSec,
  fmtRate,
  fmtSignedSize,
  fmtSize,
  num,
  pointTime,
  rangeText,
  rateUnit,
  sparkPaths,
  splitSize,
} from './format'

defineOptions({ name: 'MonitorView' })

const { t } = useI18n()
const menuStore = useMenuStore()
const pc = usePaletteColors()
const compact = computed(() => menuStore.isMobile)
const canEdit = computed(() => menuStore.hasButtonPermission('monitor:edit'))

type RangeKey = '1h' | '6h' | '24h' | '7d' | '30d'
type TileId = 'cpu' | 'gpu' | 'mem' | 'disk' | 'net'
const RANGES: { key: RangeKey; sec: number }[] = [
  { key: '1h', sec: 3600 },
  { key: '6h', sec: 6 * 3600 },
  { key: '24h', sec: 86400 },
  { key: '7d', sec: 7 * 86400 },
  { key: '30d', sec: 30 * 86400 },
]

const range = ref<RangeKey>('24h')
const custom = ref<{ start: number; end: number } | null>(null)
const live = ref(true)
const resp = shallowRef<GetMonitorDataResponse | null>(null)
const settingsOpen = ref(false)
const hidden = ref<string[]>([])
const nic = ref('all')
const dev = ref('all')
const gpuIdx = ref('0')
const sel = ref<TileId>('cpu')
const topSort = ref<'cpu' | 'mem'>('cpu')

// —— 加载与实时刷新 ——
let timer: ReturnType<typeof setTimeout> | undefined
let seq = 0
let active = true
const visibility = useDocumentVisibility()

const queryRange = () => {
  if (custom.value) return { start: custom.value.start, end: custom.value.end }
  const sec = RANGES.find((r) => r.key === range.value)!.sec
  return { start: Math.floor(Date.now() / 1000) - sec, end: 0 }
}

const schedule = (failed = false) => {
  clearTimeout(timer)
  if (!active || !live.value || custom.value || visibility.value === 'hidden') return
  const step = num(resp.value?.step) || 10
  const ms = failed ? 60_000 : Math.min(15, Math.max(5, step)) * 1000
  timer = setTimeout(() => void load(), ms)
}

const load = async () => {
  const my = ++seq
  clearTimeout(timer)
  let failed = false
  try {
    const { data } = await getMonitorData(queryRange())
    if (my !== seq) return
    resp.value = data
  } catch {
    failed = true
  } finally {
    if (my === seq) schedule(failed)
  }
}

const pickRange = (key: RangeKey) => {
  range.value = key
  custom.value = null
  void load()
}
const toggleLive = () => {
  if (custom.value) {
    custom.value = null
    live.value = true
  } else {
    live.value = !live.value
  }
  void load()
}
const onZoom = (start: number, end: number) => {
  custom.value = { start, end }
  void load()
}

watch(visibility, (v) => {
  if (v === 'visible') void load()
  else clearTimeout(timer)
})
onMounted(() => void load())
let activatedOnce = false
onActivated(() => {
  active = true
  if (!activatedOnce) {
    activatedOnce = true
    return
  }
  void load()
})
onDeactivated(() => {
  active = false
  clearTimeout(timer)
})
onBeforeUnmount(() => {
  active = false
  clearTimeout(timer)
})

// —— 自定义时段 ——
const LOCAL_FMT = 'YYYY-MM-DDTHH:mm'
const customForm = reactive({ start: '', end: '' })
const prefillCustom = () => {
  const r = custom.value || { start: Math.floor(Date.now() / 1000) - 86400, end: Math.floor(Date.now() / 1000) }
  customForm.start = dayjs.unix(r.start).format(LOCAL_FMT)
  customForm.end = dayjs.unix(r.end || Math.floor(Date.now() / 1000)).format(LOCAL_FMT)
}
const applyCustom = (close: () => void) => {
  const s = dayjs(customForm.start)
  const e = dayjs(customForm.end)
  if (!s.isValid() || !e.isValid() || !s.isBefore(e)) {
    feedback.error(t('monitor.customInvalid'))
    return
  }
  custom.value = { start: s.unix(), end: Math.min(e.unix(), Math.floor(Date.now() / 1000)) }
  close()
  void load()
}
const customText = computed(() => (custom.value ? rangeText(custom.value.start, custom.value.end) : ''))
const exportRange = computed(() => {
  if (resp.value) return { start: num(resp.value.start), end: num(resp.value.end) }
  const q = queryRange()
  return { start: q.start, end: q.end || Math.floor(Date.now() / 1000) }
})

// —— 数据整理 ——
const settings = computed<MonitorSettingsInfo | undefined>(() => resp.value?.settings)
const host = computed(() => resp.value?.host)
const hostName = computed(() => host.value?.hostname || '')
const buckets = computed(() => num(resp.value?.buckets))
const stepSec = computed(() => num(resp.value?.step) || 10)
const times = computed(() => {
  const s = num(resp.value?.start)
  return Array.from({ length: buckets.value }, (_, i) => (s + i * stepSec.value) * 1000)
})
const gaps = computed(() => (resp.value?.gaps || []).map((g) => ({ start: num(g.start), end: num(g.end) })))

interface Ser {
  values: (number | null)[]
  mean: number
  peak: number
  peakTime: number
  total: number
}
const series = computed(() => {
  const m = new Map<string, Ser>()
  const n = buckets.value
  for (const s of resp.value?.series || []) {
    const values: (number | null)[] = new Array(n).fill(null)
    s.idx.forEach((b, i) => {
      if (b >= 0 && b < n) values[b] = num(s.avg[i])
    })
    m.set(s.key, { values, mean: num(s.mean), peak: num(s.peak), peakTime: num(s.peakTime), total: num(s.total) })
  }
  return m
})
const ser = (k: string) => series.value.get(k)
const current = computed<Record<string, number>>(() => {
  const out: Record<string, number> = {}
  for (const [k, v] of Object.entries(resp.value?.current || {})) out[k] = num(v)
  return out
})
const cur = (k: string): number | undefined => (k in current.value ? current.value[k] : undefined)
const lastOf = (k: string): number | undefined => {
  const v = ser(k)?.values
  if (!v) return undefined
  for (let i = v.length - 1; i >= 0; i--) if (v[i] != null) return v[i] as number
  return undefined
}
/** 「当前」：时段截止到现在时用最新一次采样，否则用时段内最后一个点 */
const nowOf = (k: string) => (custom.value ? lastOf(k) : (cur(k) ?? lastOf(k)))
const sumValues = (a?: (number | null)[], b?: (number | null)[]) =>
  Array.from({ length: buckets.value }, (_, i) => {
    const x = a?.[i]
    const y = b?.[i]
    return x == null && y == null ? null : (x ?? 0) + (y ?? 0)
  })

const intervalText = computed(() => t('monitor.seconds', { n: num(settings.value?.intervalSeconds) || 10 }))
const retainText = computed(() => {
  const d = num(settings.value?.retentionDays) || 30
  return d === 365 ? t('monitor.oneYear') : t('monitor.days', { n: d })
})
const summaryHost = computed(() => hostName.value || '—')
const recordedText = computed(() => {
  const first = num(resp.value?.firstTime)
  if (!first) return ''
  const days = Math.floor((Date.now() / 1000 - first) / 86400)
  return days < 1 ? t('monitor.recordedToday') : t('monitor.recorded', { days })
})

const cleanModel = (s?: string) =>
  (s || '')
    .replace(/\((R|TM)\)/gi, '')
    .replace(/\bCPU\b/g, '')
    .replace(/@.*$/, '')
    .replace(/\d+-Core Processor/i, '')
    .replace(/\s+/g, ' ')
    .trim()

const gpus = computed(() => host.value?.gpus || [])
const gpu = computed(() => gpus.value.find((g) => String(g.index) === gpuIdx.value) || gpus.value[0])
watch(gpus, (list) => {
  if (list.length && !list.some((g) => String(g.index) === gpuIdx.value)) gpuIdx.value = String(list[0]!.index)
})

const fmt1 = (v?: number) => (v == null ? '—' : v.toFixed(1))
const pctLine = (key: string, label: string, color: string, fill: number, dataKey: string): CardLine => {
  const s = ser(dataKey)
  return {
    key,
    label,
    color,
    fill,
    values: s?.values || [],
    now: fmtPct(nowOf(dataKey)),
    avg: fmtPct(s?.mean),
    peak: fmtPct(s?.peak),
  }
}
const rateLine = (key: string, label: string, color: string, fill: number, dataKey: string): CardLine => {
  const s = ser(dataKey)
  return {
    key,
    label,
    color,
    fill,
    values: s?.values || [],
    now: fmtRate(nowOf(dataKey)),
    avg: fmtRate(s?.mean),
    peak: fmtRate(s?.peak),
  }
}
const hasData = (lines: CardLine[]) => lines.some((l) => l.values.some((v) => v != null))

/** 网卡 / 磁盘设备 chips：按时段累计量从大到小，最多 6 个 */
const deviceChips = (prefix: string, other: string) => {
  const totals = new Map<string, number>()
  for (const [k, s] of series.value) {
    for (const p of [prefix, other]) {
      if (k.startsWith(`${p}:`)) {
        const name = k.slice(p.length + 1)
        totals.set(name, (totals.get(name) || 0) + s.total)
      }
    }
  }
  const names = [...totals.entries()].sort((a, b) => b[1] - a[1]).slice(0, 6).map(([n]) => n)
  if (names.length < 2) return []
  return [{ value: 'all', label: t('monitor.all') }, ...names.map((n) => ({ value: n, label: n }))]
}
const nicChips = computed(() => deviceChips('rx', 'tx'))
const devChips = computed(() => deviceChips('rd', 'wr'))
watch(nicChips, (c) => {
  if (nic.value !== 'all' && !c.some((x) => x.value === nic.value)) nic.value = 'all'
})
watch(devChips, (c) => {
  if (dev.value !== 'all' && !c.some((x) => x.value === dev.value)) dev.value = 'all'
})
const suffix = (v: string) => (v === 'all' ? '' : `:${v}`)

interface CardDef {
  id: TileId
  title: string
  sub: string
  glyph: MonitorGlyphName
  tone: string
  rot: string
  kind: 'pct' | 'rate'
  div: number
  lines: CardLine[]
  picks: { value: string; label: string }[]
  pick: string
  facts: CardFact[]
  empty: CardEmpty | null
}

const offEmpty = (name: string, glyph: MonitorGlyphName): CardEmpty => ({
  title: t('monitor.off.title', { name }),
  desc: t('monitor.off.desc'),
  action: canEdit.value ? t('monitor.off.action') : undefined,
  glyph,
})
const noDataEmpty = (glyph: MonitorGlyphName): CardEmpty => ({ title: t('monitor.noData.title'), desc: t('monitor.noData.desc'), glyph })

const cards = computed<CardDef[]>(() => {
  const c = pc.value
  const h = host.value
  const st = settings.value

  // CPU
  const cpuLines = [pctLine('cpu', t('monitor.cpu.usage'), c.p, 0.14, 'cpu')]
  if (ser('iow')) cpuLines.push(pctLine('iow', t('monitor.cpu.iowait'), c.warn, 0.08, 'iow'))
  const load = (k: string) => (cur(k) == null ? '—' : (cur(k) as number).toFixed(2))
  const model = cleanModel(h?.cpuModel)
  const cpuSub = [
    model,
    t('monitor.cpu.sub', { cores: num(h?.cpuCores), threads: num(h?.cpuThreads) }),
    t('monitor.cpu.loadSub', { l1: load('load1'), l5: load('load5'), l15: load('load15') }),
  ]
    .filter(Boolean)
    .join(' · ')
  const temp = cur('ctemp')
  const cpu: CardDef = {
    id: 'cpu',
    title: t('monitor.cpu.title'),
    sub: cpuSub,
    glyph: 'cpu',
    tone: 'p',
    rot: '-4deg',
    kind: 'pct',
    div: 1,
    lines: cpuLines,
    picks: [],
    pick: '',
    facts: [
      { label: t('monitor.cpu.load1'), value: load('load1') },
      { label: t('monitor.cpu.procs'), value: fmtCount(cur('procs')) },
      { label: t('monitor.cpu.temp'), value: temp == null ? '—' : `${Math.round(temp)} °C`, tone: temp != null && temp >= 85 ? 'bad' : temp != null && temp >= 75 ? 'warn' : undefined },
      { label: t('monitor.cpu.ctx'), value: fmtPerSec(cur('ctx')) },
    ],
    empty: null,
  }
  cpu.empty = hasData(cpu.lines) ? null : noDataEmpty('cpu')

  // GPU
  const g = gpu.value
  const gi = g ? String(g.index) : '0'
  const gpuLines = g
    ? [pctLine('gpu', t('monitor.gpu.usage'), c.a2, 0.14, `gpu:${gi}`), pctLine('vram', t('monitor.gpu.vram'), c.p, 0.06, `vram:${gi}`)].filter(
        (l, i) => i === 0 || l.values.length,
      )
    : []
  const gtemp = cur(`gtemp:${gi}`)
  const gpow = cur(`gpow:${gi}`)
  const gpowl = cur(`gpowl:${gi}`)
  const gfan = cur(`gfan:${gi}`)
  const vramu = cur(`vramu:${gi}`)
  const vramt = cur(`vramt:${gi}`) ?? (num(g?.memoryTotal) || undefined)
  const gpuCard: CardDef = {
    id: 'gpu',
    title: t('monitor.gpu.title'),
    sub: g
      ? [g.name, num(g.memoryTotal) ? fmtSize(num(g.memoryTotal)) : '', g.driver ? t('monitor.gpu.driver', { v: g.driver }) : ''].filter(Boolean).join(' · ')
      : st && !st.collectGpu
        ? ''
        : t('monitor.noGpuShort'),
    glyph: 'gpu',
    tone: 'a2',
    rot: '3deg',
    kind: 'pct',
    div: 1,
    lines: gpuLines,
    picks: gpus.value.length > 1 ? gpus.value.map((x) => ({ value: String(x.index), label: x.name || `GPU ${x.index}` })) : [],
    pick: gi,
    facts: [
      { label: t('monitor.gpu.temp'), value: gtemp == null ? '—' : `${Math.round(gtemp)} °C`, tone: gtemp != null && gtemp >= 85 ? 'bad' : gtemp != null && gtemp >= 75 ? 'warn' : undefined },
      { label: t('monitor.gpu.power'), value: gpow == null ? '—' : gpowl ? `${Math.round(gpow)} / ${Math.round(gpowl)} W` : `${Math.round(gpow)} W` },
      { label: t('monitor.gpu.fan'), value: gfan == null ? '—' : `${Math.round(gfan)}%` },
      { label: t('monitor.gpu.memory'), value: vramu == null ? '—' : vramt ? `${fmtSize(vramu)} / ${fmtSize(vramt)}` : fmtSize(vramu) },
    ],
    empty: null,
  }
  if (st && !st.collectGpu) gpuCard.empty = offEmpty(t('monitor.panel.gpu'), 'gpu')
  else if (!g) gpuCard.empty = { title: t('monitor.gpu.emptyTitle'), desc: compact.value ? t('monitor.gpu.emptyShort') : t('monitor.gpu.emptyDesc'), glyph: 'gpu' }
  else if (!hasData(gpuLines)) gpuCard.empty = noDataEmpty('gpu')

  // 内存
  const memLines = [pctLine('mem', t('monitor.mem.used'), c.ok, 0.14, 'mem')]
  if (ser('cache')) memLines.push(pctLine('cache', t('monitor.mem.cache'), c.a2, 0.06, 'cache'))
  else if (ser('swap')) memLines.push(pctLine('swap', t('monitor.mem.swap'), c.a2, 0.06, 'swap'))
  const swapTotal = num(h?.swapTotal)
  const memPeak = ser('mem')
  const mem: CardDef = {
    id: 'mem',
    title: t('monitor.mem.title'),
    sub: swapTotal
      ? t('monitor.mem.sub', { total: fmtSize(num(h?.memoryTotal)), swap: fmtSize(swapTotal) })
      : t('monitor.mem.subNoSwap', { total: fmtSize(num(h?.memoryTotal)) }),
    glyph: 'mem',
    tone: 'ok',
    rot: '-2deg',
    kind: 'pct',
    div: 1,
    lines: memLines,
    picks: [],
    pick: '',
    facts: [
      { label: t('monitor.mem.used'), value: fmtSize(cur('memu')) },
      { label: t('monitor.mem.avail'), value: fmtSize(cur('mema')) },
      { label: t('monitor.mem.swapUsed'), value: swapTotal ? fmtSize(cur('swapu') ?? 0) : '—' },
      { label: t('monitor.mem.peakAt'), value: memPeak?.peakTime ? pointTime(memPeak.peakTime * 1000, stepSec.value) : '—' },
    ],
    empty: null,
  }
  mem.empty = hasData(mem.lines) ? null : noDataEmpty('mem')

  // 网络
  const ns = suffix(nic.value)
  const netLines = [rateLine('rx', t('monitor.net.rx'), c.ok, 0.12, `rx${ns}`), rateLine('tx', t('monitor.net.tx'), c.a2, 0.06, `tx${ns}`)]
  const netUnit = rateUnit(Math.max(ser(`rx${ns}`)?.peak || 0, ser(`tx${ns}`)?.peak || 0))
  const net: CardDef = {
    id: 'net',
    title: t('monitor.net.title'),
    sub: `${t('monitor.net.sub')} · ${t('monitor.unit', { u: netUnit.label })}`,
    glyph: 'net',
    tone: 'a',
    rot: '4deg',
    kind: 'rate',
    div: netUnit.div,
    lines: netLines,
    picks: nicChips.value,
    pick: nic.value,
    facts: [
      { label: t('monitor.net.totalRx'), value: fmtSize(ser(`rx${ns}`)?.total ?? 0) },
      { label: t('monitor.net.totalTx'), value: fmtSize(ser(`tx${ns}`)?.total ?? 0) },
      { label: t('monitor.net.peak'), value: fmtRate(ser(`rx${ns}`)?.peak ?? 0) },
      { label: t('monitor.net.tcp'), value: fmtCount(cur('tcp')) },
    ],
    empty: null,
  }
  if (st && !st.collectNetwork) net.empty = offEmpty(t('monitor.panel.net'), 'net')
  else if (!hasData(netLines)) net.empty = noDataEmpty('net')

  // 磁盘读写
  const ds = suffix(dev.value)
  const diskLines = [rateLine('rd', t('monitor.disk.read'), c.p, 0.12, `rd${ds}`), rateLine('wr', t('monitor.disk.write'), c.a, 0.1, `wr${ds}`)]
  const diskUnit = rateUnit(Math.max(ser(`rd${ds}`)?.peak || 0, ser(`wr${ds}`)?.peak || 0))
  const lat = ser(`lat${ds}`)
  const disk: CardDef = {
    id: 'disk',
    title: t('monitor.disk.title'),
    sub: `${t('monitor.disk.sub')} · ${t('monitor.unit', { u: diskUnit.label })}`,
    glyph: 'disk',
    tone: 'p',
    rot: '-3deg',
    kind: 'rate',
    div: diskUnit.div,
    lines: diskLines,
    picks: devChips.value,
    pick: dev.value,
    facts: [
      { label: t('monitor.disk.totalRead'), value: fmtSize(ser(`rd${ds}`)?.total ?? 0) },
      { label: t('monitor.disk.totalWrite'), value: fmtSize(ser(`wr${ds}`)?.total ?? 0) },
      { label: t('monitor.disk.iopsPeak'), value: fmtCount(ser(`iops${ds}`)?.peak) },
      { label: t('monitor.disk.latency'), value: lat ? `${lat.mean.toFixed(1)} ms` : '—' },
    ],
    empty: null,
  }
  if (st && !st.collectDisk) disk.empty = offEmpty(t('monitor.panel.disk'), 'disk')
  else if (!hasData(diskLines)) disk.empty = noDataEmpty('disk')

  return [cpu, gpuCard, mem, net, disk]
})
const selectedCard = computed(() => cards.value.find((c) => c.id === sel.value))
const cardProps = (c: CardDef) => ({
  title: c.title,
  sub: c.sub,
  glyph: c.glyph,
  tone: c.tone,
  rot: c.rot,
  lines: c.lines,
  hidden: hidden.value,
  picks: c.picks,
  pick: c.pick,
  facts: c.facts,
  empty: c.empty,
  times: times.value,
  kind: c.kind,
  div: c.div,
  gaps: gaps.value,
  step: stepSec.value,
})
const toggleLine = (key: string) => {
  hidden.value = hidden.value.includes(key) ? hidden.value.filter((k) => k !== key) : [...hidden.value, key]
}
const onPick = (id: TileId, v: string) => {
  if (id === 'net') nic.value = v
  else if (id === 'disk') dev.value = v
  else if (id === 'gpu') gpuIdx.value = v
}

// —— 概览块 ——
interface Tile {
  id: TileId
  title: string
  glyph: MonitorGlyphName
  tone: string
  rot: string
  value: string
  unit: string
  sub: string
  color: string
  spark: { line: string; area: string }
  avg: string
  peak: string
}
const fullestMount = computed(() => {
  let best: { m: string; pct: number } | null = null
  for (const d of resp.value?.disks || []) {
    const total = num(d.total)
    if (!total) continue
    const pct = (num(d.used) / total) * 100
    if (!best || pct > best.pct) best = { m: d.mountpoint, pct }
  }
  return best
})
const tiles = computed<Tile[]>(() => {
  const c = pc.value
  const h = host.value
  const st = settings.value
  const cpuS = ser('cpu')
  const g = gpu.value
  const gi = g ? String(g.index) : '0'
  const gS = ser(`gpu:${gi}`)
  const memS = ser('mem')
  const rd = ser('rd')
  const wr = ser('wr')
  const rx = ser('rx')
  const tx = ser('tx')
  const ioSum = sumValues(rd?.values, wr?.values)
  const netSum = sumValues(rx?.values, tx?.values)
  const peakOf = (v: (number | null)[]) => v.reduce<number>((a, b) => Math.max(a, b ?? 0), 0)
  const rateTile = (v: number | undefined): [string, string] => {
    if (v == null) return ['—', '']
    const [n, u] = splitSize(v)
    return [n, `${u}/s`]
  }
  const ioNow = nowOf('rd') == null && nowOf('wr') == null ? undefined : (nowOf('rd') ?? 0) + (nowOf('wr') ?? 0)
  const netNow = nowOf('rx') == null && nowOf('tx') == null ? undefined : (nowOf('rx') ?? 0) + (nowOf('tx') ?? 0)
  const [ioV, ioU] = rateTile(ioNow)
  const [netV, netU] = rateTile(netNow)
  const gtemp = cur(`gtemp:${gi}`)
  const gpuName = (g?.name || '').replace(/^(NVIDIA|AMD|Intel\(R\)|Intel)\s+/i, '').replace(/^GeForce\s+/i, '')
  const fm = fullestMount.value
  const tcp = cur('tcp')
  const gpuOff = !!st && !st.collectGpu
  return [
    {
      id: 'cpu',
      title: t('monitor.tiles.cpu'),
      glyph: 'cpu',
      tone: 'p',
      rot: '-4deg',
      value: fmt1(nowOf('cpu')),
      unit: '%',
      sub: [t('monitor.threads', { n: num(h?.cpuThreads) }), cur('load1') != null ? t('monitor.load', { v: (cur('load1') as number).toFixed(2) }) : '']
        .filter(Boolean)
        .join(' · '),
      color: c.p,
      spark: sparkPaths(cpuS?.values || [], 100),
      avg: fmtPct(cpuS?.mean),
      peak: fmtPct(cpuS?.peak),
    },
    {
      id: 'gpu',
      title: t('monitor.tiles.gpu'),
      glyph: 'gpu',
      tone: 'a2',
      rot: '3deg',
      value: g && !gpuOff ? fmt1(nowOf(`gpu:${gi}`)) : '—',
      unit: g && !gpuOff ? '%' : '',
      sub: gpuOff ? t('monitor.off.title', { name: 'GPU' }) : g ? [gpuName, gtemp != null ? `${Math.round(gtemp)} °C` : ''].filter(Boolean).join(' · ') : t('monitor.noGpuShort'),
      color: g ? c.a2 : c.ink2,
      spark: g && !gpuOff ? sparkPaths(gS?.values || [], 100) : { line: '', area: '' },
      avg: g && !gpuOff ? fmtPct(gS?.mean) : '—',
      peak: g && !gpuOff ? fmtPct(gS?.peak) : '—',
    },
    {
      id: 'mem',
      title: t('monitor.tiles.mem'),
      glyph: 'mem',
      tone: 'ok',
      rot: '-2deg',
      value: fmt1(nowOf('mem')),
      unit: '%',
      sub: `${fmtSize(cur('memu'))} / ${fmtSize(num(h?.memoryTotal))}`,
      color: c.ok,
      spark: sparkPaths(memS?.values || [], 100),
      avg: fmtPct(memS?.mean),
      peak: fmtPct(memS?.peak),
    },
    {
      id: 'disk',
      title: t('monitor.tiles.disk'),
      glyph: 'disk',
      tone: 'p',
      rot: '2deg',
      value: st && !st.collectDisk ? '—' : ioV,
      unit: st && !st.collectDisk ? '' : ioU,
      sub: st && !st.collectDisk ? t('monitor.off.title', { name: t('monitor.panel.disk') }) : fm ? t('monitor.mountUsed', { mount: fm.m, pct: `${Math.round(fm.pct)}%` }) : '—',
      color: c.a,
      spark: sparkPaths(ioSum),
      avg: fmtRate((rd?.mean ?? 0) + (wr?.mean ?? 0)),
      peak: fmtRate(peakOf(ioSum)),
    },
    {
      id: 'net',
      title: t('monitor.tiles.net'),
      glyph: 'net',
      tone: 'a',
      rot: '4deg',
      value: st && !st.collectNetwork ? '—' : netV,
      unit: st && !st.collectNetwork ? '' : netU,
      sub:
        st && !st.collectNetwork
          ? t('monitor.off.title', { name: t('monitor.panel.net') })
          : `↓ ${fmtRate(nowOf('rx'))} · ↑ ${fmtRate(nowOf('tx'))}${compact.value && tcp != null ? ` · ${t('monitor.net.connections', { n: fmtCount(tcp) })}` : ''}`,
      color: c.ok,
      spark: sparkPaths(netSum),
      avg: fmtRate((rx?.mean ?? 0) + (tx?.mean ?? 0)),
      peak: fmtRate(peakOf(netSum)),
    },
  ]
})
const onTile = (e: Event, id: TileId) => {
  if (compact.value) {
    sel.value = id
    return
  }
  e.preventDefault()
  document.getElementById(`mon-${id}`)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}

// —— 磁盘容量 ——
const durationText = (sec: number) => {
  const p = durationParts(sec)
  return t(`monitor.disks.${p.key}`, { n: p.n })
}
const disks = computed(() => {
  const c = pc.value
  return (resp.value?.disks || []).map((d) => {
    const used = num(d.used)
    const total = num(d.total)
    const pct = total ? (used / total) * 100 : 0
    const color = pct >= 85 ? c.bad : pct >= 60 ? c.warn : c.ok
    const span = num(d.growthSpan)
    const fullIn = num(d.fullIn)
    let eta = t('monitor.disks.collecting')
    let etaShort = eta
    let etaColor = c.ink2
    if (fullIn >= 0) {
      const tt = durationText(fullIn)
      eta = t('monitor.disks.fullIn', { t: tt })
      etaShort = t('monitor.disks.fullInShort', { t: tt })
      etaColor = fullIn < 30 * 86400 ? c.bad : fullIn < 180 * 86400 ? c.warn : c.ok
    } else if (span >= 6 * 3600) {
      eta = t('monitor.disks.flat')
      etaShort = eta
      etaColor = c.ok
    }
    return {
      mountpoint: d.mountpoint,
      fs: [d.fstype, d.device && d.device !== d.mountpoint ? d.device : ''].filter(Boolean).join(' · '),
      used: fmtSize(used),
      total: fmtSize(total),
      pct: `${Math.round(pct)}%`,
      pctW: `${Math.min(100, pct).toFixed(1)}%`,
      color,
      grow: span > 0 ? t('monitor.disks.grow', { span: durationText(span), size: fmtSignedSize(num(d.growth)) }) : '',
      eta,
      etaShort,
      etaColor,
    }
  })
})

// —— 应用排行 ——
const TONES = ['p', 'a2', 'ok', 'a', 'p']
const ROTS = ['-4deg', '3deg', '-2deg', '4deg', '-3deg']
const apps = computed(() => {
  const list = [...(resp.value?.apps || [])].sort((a, b) => (topSort.value === 'cpu' ? num(b.cpu) - num(a.cpu) : num(b.memory) - num(a.memory)))
  const top = list.slice(0, compact.value ? 4 : 5)
  const max = Math.max(1e-9, ...top.map((a) => (topSort.value === 'cpu' ? num(a.cpu) : num(a.memory))))
  return top.map((a, i) => ({
    id: a.id,
    name: a.name,
    running: a.running,
    letter: (a.name || '?').trim().charAt(0).toUpperCase(),
    tone: TONES[i % TONES.length]!,
    rot: ROTS[i % ROTS.length]!,
    cpu: fmtPct(num(a.cpu)),
    mem: fmtSize(num(a.memory)),
    w: `${(((topSort.value === 'cpu' ? num(a.cpu) : num(a.memory)) / max) * 100).toFixed(1)}%`,
  }))
})

const onSaved = (s: MonitorSettingsInfo) => {
  if (resp.value) resp.value = { ...resp.value, settings: s }
  setTimeout(() => void load(), 600)
}
</script>

<style scoped lang="scss">
.mon {
  display: flex;
  flex-direction: column;
  gap: 20px;
  min-width: 0;
}
.mon :deep(.mon-strong) {
  color: var(--el-text-color-primary);
  font-weight: 700;
}
.mon-warn {
  color: var(--m-warn);
  font-weight: 700;
}
.mon-btn-ic {
  font-size: 18px;
}
.mon-pill {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 22px;
  margin-right: 8px;
  padding: 0 9px;
  border-radius: 999px;
  font-size: 0.6875rem;
  font-weight: 700;
  vertical-align: middle;
}
.mon-pill::before {
  content: '';
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
}
.mon-pill.is-live {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.mon-pill.is-off {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}

/* 实时开关：更新中为绿色淡底，暂停为描边 */
.mon-live {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s;
}
.mon-live:hover {
  border-color: var(--m-p);
}
.mon-live.is-on {
  border-color: transparent;
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.mon-live__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: currentColor;
}
.mon-live.is-on .mon-live__dot {
  animation: mon-blink 1.2s ease-in-out infinite;
}
@keyframes mon-blink {
  50% {
    opacity: 0.25;
  }
}

/* 时间范围 */
.mon-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
  margin-top: -4px;
}
.mon-bar__sp {
  flex: 1;
}
.mon-bar__hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.mon-seg {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.mon-seg button {
  min-height: 34px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}
.mon-seg button:hover {
  color: var(--el-text-color-primary);
}
.mon-seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: 0 1px 3px rgba(20, 10, 30, 0.12);
}
.mon-custom {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 18px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}
.mon-custom:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.mon-custom.is-on,
.mon-custom.is-on:hover {
  border-color: var(--m-ink);
  background: var(--m-ink);
  color: var(--m-bg);
}
.mon-back {
  min-height: 40px;
  padding: 0 6px;
  border: 0;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
}
.mon-pop {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 18px;
}
.mon-pop__t {
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.mon-pop__f {
  display: flex;
  flex-direction: column;
  gap: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.mon-pop__in {
  min-height: 44px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
  outline: none;
  color-scheme: light dark;
}
.mon-pop__in:focus {
  border-color: var(--m-p);
}
.mon-pop__hint {
  font-size: 0.75rem;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.mon-pop__acts {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
.mon-pop__acts :deep(button) {
  min-height: 36px;
  border-radius: 999px;
  padding-inline: 14px;
}

/* 概览块 */
.mon-tiles {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 14px;
}
.mon-tile {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
  padding: calc(var(--cmp-card-unit) * 16) calc(var(--cmp-card-unit) * 16) calc(var(--cmp-card-unit) * 12);
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  text-decoration: none;
  cursor: pointer;
  transition: border-color 0.15s, transform 0.15s;
}
.mon-tile:hover {
  border-color: var(--m-p);
  transform: translateY(-2px);
}
.mon-tile__h {
  display: flex;
  align-items: center;
  gap: 10px;
}
.mon-tile__stk {
  width: 34px;
  height: 34px;
  border-radius: 11px;
  font-size: 16px;
}
.mon-tile__t {
  flex: 1;
  min-width: 0;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.mon-tile__v {
  display: flex;
  align-items: baseline;
  gap: 6px;
}
.mon-tile__n {
  font-size: 1.75rem;
  font-weight: 800;
  line-height: 1;
}
.mon-tile__u {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.mon-tile__sub {
  overflow: hidden;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mon-tile__spark {
  display: block;
  width: 100%;
  height: 34px;
}
.mon-tile__ap {
  display: flex;
  justify-content: space-between;
  gap: 6px;
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

/* 曲线卡 */
.mon-charts {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}
.mon-charts__wide {
  grid-column: 1 / -1;
}

/* 磁盘容量 + 应用排行 */
.mon-bottom {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 16px;
}
.mon-panel {
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: calc(var(--cmp-card-unit) * 18) calc(var(--cmp-card-unit) * 20);
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.mon-panel--apps {
  gap: 8px;
  padding: 18px 12px 12px 20px;
}
.mon-panel__h {
  display: flex;
  align-items: center;
  gap: 10px;
}
.mon-panel--apps .mon-panel__h,
.mon-panel__hint--row {
  padding-right: 8px;
}
.mon-panel__t {
  flex: 1;
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.mon-panel__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.mon-seg--sm button {
  min-height: 32px;
  padding: 0 12px;
}
.mon-disk {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 12px 14px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
}
.mon-disk__h,
.mon-disk__f {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.mon-disk__m {
  overflow: hidden;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mon-disk__fs {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.mon-disk__sp {
  flex: 1;
}
.mon-disk__u {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}
.mon-disk__bar {
  height: 10px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.mon-disk__bar span {
  display: block;
  height: 100%;
  border-radius: 999px;
}
.mon-disk__f {
  gap: 8px;
  font-size: 0.75rem;
}
.mon-disk__pct {
  font-weight: 800;
}
.mon-disk__grow {
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.mon-disk__eta {
  overflow: hidden;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mon-app {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 52px;
  padding: 6px 8px;
  border-radius: var(--app-radius);
  color: var(--el-text-color-primary);
  text-decoration: none;
}
.mon-app:hover {
  background: var(--el-fill-color-light);
}
.mon-app__av {
  width: 34px;
  height: 34px;
  border-radius: 11px;
  font-size: 0.9375rem;
}
.mon-app__main {
  display: flex;
  flex: 1;
  flex-direction: column;
  min-width: 0;
}
.mon-app__n {
  display: flex;
  align-items: center;
  gap: 6px;
  overflow: hidden;
  font-size: 0.875rem;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mon-app__stop {
  flex: none;
  padding: 1px 7px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
  font-size: 0.6875rem;
  font-weight: 700;
}
.mon-app__bar {
  height: 6px;
  margin-top: 6px;
  overflow: hidden;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.mon-app__bar span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--m-p);
}
.mon-app__cpu {
  width: 64px;
  text-align: right;
  font-size: 0.875rem;
  font-weight: 800;
}
.mon-app__mem {
  width: 76px;
  text-align: right;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.mon-app__go {
  width: 16px;
  height: 16px;
  flex: none;
  color: var(--el-text-color-secondary);
}

@media (width <= 1280px) {
  .mon-tiles {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}
@media (width <= 1100px) {
  .mon-charts,
  .mon-bottom {
    grid-template-columns: minmax(0, 1fr);
  }
}

/* —— 手机（MonitorMobile）—— */
.mon--m {
  gap: 14px;
}
.mon-chips {
  display: flex;
  gap: 8px;
  margin: 0 -16px;
  padding: 2px 16px 0;
  overflow-x: auto;
  scrollbar-width: none;
}
.mon-chips::-webkit-scrollbar {
  display: none;
}
.mon-chip {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}
.mon-chip.is-on {
  border-color: var(--m-ink);
  background: var(--m-ink);
  color: var(--m-bg);
}
.mon--m .mon-chips :deep(.app-dropdown__trigger) {
  flex: none;
}
.mon--m .mon-tiles {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}
.mon--m .mon-tile {
  gap: 6px;
  padding: 12px 12px 10px;
  border-width: 1.5px;
}
.mon--m .mon-tile:hover {
  transform: none;
}
.mon--m .mon-tile:last-child {
  grid-column: 1 / -1;
}
.mon--m .mon-tile.is-on {
  border-color: var(--m-p);
  box-shadow: inset 0 0 0 1px var(--m-p);
}
.mon--m .mon-tile__h {
  gap: 8px;
}
.mon--m .mon-tile__stk {
  width: 28px;
  height: 28px;
  border-radius: 9px;
  font-size: 14px;
}
.mon--m .mon-tile__t {
  font-size: 0.75rem;
}
.mon--m .mon-tile__n {
  font-size: 1.5rem;
}
.mon--m .mon-tile__u {
  font-size: 0.75rem;
}
.mon--m .mon-tile__sub {
  font-size: 0.6875rem;
}
.mon--m .mon-tile__spark {
  height: 26px;
}
.mon--m .mon-bottom {
  grid-template-columns: minmax(0, 1fr);
  gap: 14px;
}
.mon--m .mon-panel {
  gap: 12px;
  padding: 16px 14px;
}
.mon--m .mon-panel--apps {
  gap: 4px;
  padding: 14px 8px 8px 14px;
}
.mon--m .mon-panel__t {
  font-size: 1rem;
}
.mon--m .mon-disk {
  gap: 6px;
  padding: 0;
  border: 0;
}
.mon--m .mon-disk__m {
  font-size: 0.8125rem;
}
.mon--m .mon-disk__u {
  font-size: 0.75rem;
}
.mon--m .mon-disk__bar {
  height: 8px;
}
.mon--m .mon-disk__f {
  font-size: 0.6875rem;
}
.mon--m .mon-disk__grow {
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
}
.mon--m .mon-disk__f .mon-disk__sp {
  display: none;
}
.mon--m .mon-app {
  gap: 10px;
  padding: 4px 6px;
}
.mon--m .mon-app__av {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  font-size: 0.875rem;
}
.mon--m .mon-app__bar {
  height: 5px;
  margin-top: 5px;
}
.mon-app__vals {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  min-width: 64px;
}
.mon-app__v {
  font-size: 0.875rem;
  font-weight: 800;
}
.mon-app__v2 {
  font-size: 0.6875rem;
  color: var(--el-text-color-secondary);
}
</style>
