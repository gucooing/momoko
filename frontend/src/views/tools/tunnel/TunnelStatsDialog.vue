<!-- 隧道统计：类型贴纸 + 路由 + 状态/客户端 → 时间范围分段 + 实时开关 + 刷新 → 指标格
     → 流量速率曲线 + 活动连接柱 → 导出 CSV。速率由相邻采样的累计流量差分；ECharts 颜色取自当前配色。 -->
<template>
  <FormDialog
    v-model="visible"
    :title="dialogTitle"
    :width="860"
    :show-footer="false"
    overflow-visible
    @close="handleClosed"
  >
    <div class="tns">
      <div class="tns-head">
        <span class="m-sticker tns-head__sticker m-mono" :class="stickerClass">{{ typeLabel }}</span>
        <div class="tns-head__text">
          <div class="tns-head__route m-mono">
            {{ localText }}
            <component :is="menuStore.iconComponents['HOutline:ArrowRightIcon']" />
            <b>{{ remoteText }}</b>
          </div>
          <div class="tns-head__meta">
            <StatusPill
              :variant="online ? 'success' : 'neutral'"
              :label="online ? t('tools.tunnel.stats.online') : t('tools.tunnel.stats.offline')"
            />
            <span v-if="row?.client" class="tns-muted m-mono">
              {{ row.client.hostname }} · {{ [row.client.os, row.client.arch].filter(Boolean).join('/') }} · frpc {{ row.client.version }}
            </span>
            <span v-else class="tns-muted">{{ t('tools.tunnel.statsDialog.noClient') }}</span>
            <span v-if="lastText" class="tns-muted m-num">{{ lastText }}</span>
          </div>
        </div>
      </div>

      <div class="tns-toolbar">
        <div class="tns-seg" role="radiogroup" :aria-label="t('tools.tunnel.statsDialog.rangeLabel')">
          <button
            v-for="opt in rangeOptions"
            :key="opt.value"
            type="button"
            role="radio"
            class="tns-seg__btn"
            :class="{ 'is-on': rangeKey === opt.value }"
            :aria-checked="rangeKey === opt.value"
            @click="setRange(opt.value)"
          >{{ opt.label }}</button>
        </div>
        <span class="tns-spacer" />
        <label class="tns-live">
          <input v-model="autoRefresh" type="checkbox" @change="restartPolling" />
          <span class="tns-live__dot" :class="{ 'is-on': autoRefresh }" />
          {{ t('tools.tunnel.statsDialog.live') }}
        </label>
        <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('tools.tunnel.stats.refresh')" :box="36" :size="18" @click="refresh" />
      </div>

      <div v-if="rangeKey === 'custom'" class="tns-custom">
        <template v-if="!isMobile">
          <input
            v-model="customStartLocal"
            type="datetime-local"
            class="app-input tns-dt"
            :aria-label="t('tools.tunnel.stats.rangeStart')"
            @change="onCustomChange"
          />
          <span class="tns-muted">{{ t('tools.tunnel.statsDialog.to') }}</span>
          <input
            v-model="customEndLocal"
            type="datetime-local"
            class="app-input tns-dt"
            :aria-label="t('tools.tunnel.stats.rangeEnd')"
            @change="onCustomChange"
          />
        </template>
        <template v-else>
          <div class="tns-custom__row">
            <span>{{ t('tools.tunnel.stats.rangeStart') }}</span>
            <input v-model="customStartDate" type="date" class="app-input" @change="syncMobileCustom" />
            <input v-model="customStartTime" type="time" class="app-input" @change="syncMobileCustom" />
          </div>
          <div class="tns-custom__row">
            <span>{{ t('tools.tunnel.stats.rangeEnd') }}</span>
            <input v-model="customEndDate" type="date" class="app-input" @change="syncMobileCustom" />
            <input v-model="customEndTime" type="time" class="app-input" @change="syncMobileCustom" />
          </div>
        </template>
      </div>

      <dl class="tns-stats">
        <div>
          <dt>{{ t('tools.tunnel.statsDialog.rateIn') }}</dt>
          <dd class="m-num is-p">{{ formatRate(lastRate.in, '0 B/s') }}</dd>
        </div>
        <div>
          <dt>{{ t('tools.tunnel.statsDialog.rateOut') }}</dt>
          <dd class="m-num">{{ formatRate(lastRate.out, '0 B/s') }}</dd>
        </div>
        <div>
          <dt>{{ t('tools.tunnel.stats.activeConnections') }}</dt>
          <dd class="m-num">{{ activeConnections }}</dd>
        </div>
        <div>
          <dt>{{ t('tools.tunnel.statsDialog.rangeTraffic') }}</dt>
          <dd class="m-num">{{ formatBytes(rangeTraffic) }}</dd>
        </div>
        <div>
          <dt>{{ t('tools.tunnel.statsDialog.todayTraffic') }}</dt>
          <dd class="m-num">{{ formatBytes(snapshotIn + snapshotOut) }}<small> ↓{{ formatBytes(snapshotIn) }} ↑{{ formatBytes(snapshotOut) }}</small></dd>
        </div>
      </dl>

      <section class="tns-card">
        <div class="tns-card__head">
          <span class="tns-card__title">{{ t('tools.tunnel.statsDialog.rateChart') }}</span>
          <span class="tns-legend"><i class="is-in" />{{ t('tools.tunnel.stats.inbound') }}</span>
          <span class="tns-legend"><i class="is-out" />{{ t('tools.tunnel.stats.outbound') }}</span>
        </div>
        <div class="tns-chart-wrap">
          <div ref="rateEl" class="tns-chart" />
          <div v-if="!points.length && !loading" class="tns-empty">{{ t('tools.tunnel.stats.noData') }}</div>
        </div>
      </section>

      <section class="tns-card">
        <div class="tns-card__head">
          <span class="tns-card__title">{{ t('tools.tunnel.statsDialog.connChart') }}</span>
          <span class="tns-muted m-num">{{ t('tools.tunnel.statsDialog.peak', { n: connPeak }) }}</span>
        </div>
        <div class="tns-chart-wrap is-small">
          <div ref="connEl" class="tns-chart" />
        </div>
      </section>

      <div class="tns-foot">
        <span v-if="sampleSeconds" class="tns-muted">{{ t('tools.tunnel.statsDialog.sample', { sec: sampleSeconds }) }}</span>
        <span class="tns-spacer" />
        <UButton color="neutral" variant="soft" icon="i-lucide-download" :disabled="!points.length" @click="exportCsv">
          {{ t('tools.tunnel.statsDialog.exportCsv') }}
        </UButton>
      </div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import * as echarts from 'echarts'
import { getTunnelStats } from '@/api/tunnel'
import { TunnelType, type TunnelInfo, type TunnelStatPoint } from '@/types/v1/tunnel'
import { showRequestError } from '@/utils/request'
import { usePaletteColors, withAlpha } from '@/composables/usePaletteColors'
import { formatBytes, formatRate, formatDateTime, formatAgo, toDate, downloadText } from '@/views/tools/components/format'

type RangeKey = '5m' | '30m' | '1h' | '6h' | '24h' | 'custom'
type ChartPoint = { ts: number; label: string; activeConnections: number; inboundRate: number; outboundRate: number; bytesIn: number; bytesOut: number }
type TooltipParam = { axisValueLabel?: string; marker?: string; seriesName?: string; value?: number | string }

const POLL_INTERVAL_MS = 10000
const PRESET_SECONDS: Record<Exclude<RangeKey, 'custom'>, number> = {
  '5m': 300, '30m': 1800, '1h': 3600, '6h': 21600, '24h': 86400,
}

const props = defineProps<{ modelValue: boolean; row: TunnelInfo | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const { t, locale } = useI18n()
const menuStore = useMenuStore()
const palette = usePaletteColors()
const isMobile = computed(() => menuStore.isMobile)

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const rateEl = ref<HTMLDivElement | null>(null)
const connEl = ref<HTMLDivElement | null>(null)
const rangeKey = ref<RangeKey>('1h')
const customStart = ref<Date | null>(null)
const customEnd = ref<Date | null>(null)
const customStartDate = ref('')
const customStartTime = ref('')
const customEndDate = ref('')
const customEndTime = ref('')
const loading = ref(false)
const autoRefresh = ref(true)
const online = ref(false)
const snapshotIn = ref(0)
const snapshotOut = ref(0)
const activeConnections = ref(0)
const lastStart = ref<Date | null>(null)
const lastClose = ref<Date | null>(null)
const sampleSeconds = ref(0)
const points = ref<ChartPoint[]>([])

let rateChart: echarts.ECharts | null = null
let connChart: echarts.ECharts | null = null
let pollTimer: ReturnType<typeof setTimeout> | null = null

const typeKey = computed(() => (props.row?.type || TunnelType.TUNNEL_TYPE_TCP).replace('TUNNEL_TYPE_', '').toLowerCase())
const typeLabel = computed(() => typeKey.value.toUpperCase())
const stickerClass = computed(() => {
  if (typeKey.value === 'udp') return 'm-sticker--a'
  if (typeKey.value === 'http') return 'm-sticker--p'
  if (typeKey.value === 'https') return 'm-sticker--ok'
  return 'm-sticker--a2'
})
const localText = computed(() => `${props.row?.localIp || '127.0.0.1'}:${props.row?.localPort || 0}`)
const remoteText = computed(() => {
  const row = props.row
  if (!row) return ''
  if (row.type === TunnelType.TUNNEL_TYPE_HTTP || row.type === TunnelType.TUNNEL_TYPE_HTTPS) {
    return row.customDomains || row.subdomain || '—'
  }
  if (row.type === TunnelType.TUNNEL_TYPE_STCP || row.type === TunnelType.TUNNEL_TYPE_XTCP) return t('tools.tunnel.p2pEntry')
  return row.remotePort ? `:${row.remotePort}` : '—'
})
const lastText = computed(() => {
  if (online.value && lastStart.value) return t('tools.tunnel.statsDialog.lastOnline', { time: formatAgo(lastStart.value, t) })
  if (!online.value && lastClose.value) return t('tools.tunnel.statsDialog.lastOffline', { time: formatAgo(lastClose.value, t) })
  return ''
})

const lastRate = computed(() => {
  const p = points.value[points.value.length - 1]
  return { in: p?.inboundRate || 0, out: p?.outboundRate || 0 }
})
const connPeak = computed(() => points.value.reduce((m, p) => Math.max(m, p.activeConnections), activeConnections.value))
const rangeTraffic = computed(() => {
  // 相邻采样累计值之差求和（跨天归零时取新值）
  let sum = 0
  for (let i = 1; i < points.value.length; i++) {
    const a = points.value[i - 1]!
    const b = points.value[i]!
    const dIn = b.bytesIn - a.bytesIn
    const dOut = b.bytesOut - a.bytesOut
    sum += (dIn < 0 ? b.bytesIn : dIn) + (dOut < 0 ? b.bytesOut : dOut)
  }
  return sum
})

const pad2 = (n: number) => String(n).padStart(2, '0')
const toLocalInput = (d: Date) =>
  `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}T${pad2(d.getHours())}:${pad2(d.getMinutes())}`
const toDatePart = (d: Date) => `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())}`
const toTimePart = (d: Date) => `${pad2(d.getHours())}:${pad2(d.getMinutes())}`
const fillMobileCustomFromDates = () => {
  if (customStart.value) {
    customStartDate.value = toDatePart(customStart.value)
    customStartTime.value = toTimePart(customStart.value)
  }
  if (customEnd.value) {
    customEndDate.value = toDatePart(customEnd.value)
    customEndTime.value = toTimePart(customEnd.value)
  }
}
const syncMobileCustom = () => {
  if (customStartDate.value && customStartTime.value) customStart.value = new Date(`${customStartDate.value}T${customStartTime.value}`)
  if (customEndDate.value && customEndTime.value) customEnd.value = new Date(`${customEndDate.value}T${customEndTime.value}`)
  onCustomChange()
}
const customStartLocal = computed({
  get: () => (customStart.value ? toLocalInput(customStart.value) : ''),
  set: (v: string) => { customStart.value = v ? new Date(v) : null },
})
const customEndLocal = computed({
  get: () => (customEnd.value ? toLocalInput(customEnd.value) : ''),
  set: (v: string) => { customEnd.value = v ? new Date(v) : null },
})

const rangeOptions = computed(() => [
  { value: '5m' as RangeKey, label: t('tools.tunnel.stats.range5m') },
  { value: '30m' as RangeKey, label: t('tools.tunnel.stats.range30m') },
  { value: '1h' as RangeKey, label: t('tools.tunnel.stats.range1h') },
  { value: '6h' as RangeKey, label: t('tools.tunnel.stats.range6h') },
  { value: '24h' as RangeKey, label: t('tools.tunnel.stats.range24h') },
  { value: 'custom' as RangeKey, label: t('tools.tunnel.stats.rangeCustom') },
])

const dialogTitle = computed(() => {
  const name = props.row?.name
  return name ? t('tools.tunnel.stats.titleWithName', { name }) : t('tools.tunnel.stats.title')
})

const num = (value?: number | string) => Number(value || 0)
const axisLabel = (d: Date, spanSeconds: number) =>
  spanSeconds > 86400 ? `${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}` : `${pad2(d.getHours())}:${pad2(d.getMinutes())}`

const resolveRange = (): { startTimeMs: number; endTimeMs: number } => {
  if (rangeKey.value === 'custom' && customStart.value && customEnd.value) {
    let start = customStart.value.getTime()
    let end = customEnd.value.getTime()
    if (start > end) [start, end] = [end, start]
    return { startTimeMs: start, endTimeMs: end }
  }
  const end = Date.now()
  const seconds = PRESET_SECONDS[rangeKey.value === 'custom' ? '1h' : rangeKey.value]
  return { startTimeMs: end - seconds * 1000, endTimeMs: end }
}

const buildChartPoints = (raw: TunnelStatPoint[], spanSeconds: number): ChartPoint[] =>
  raw.map((point, index) => {
    const time = toDate(point.time) || new Date()
    const bytesIn = num(point.bytesIn)
    const bytesOut = num(point.bytesOut)
    let inboundRate = 0
    let outboundRate = 0
    const prev = index > 0 ? raw[index - 1] : undefined
    if (prev) {
      const prevTime = toDate(prev.time)?.getTime() ?? time.getTime()
      const dt = Math.max(1, (time.getTime() - prevTime) / 1000)
      const dIn = bytesIn - num(prev.bytesIn)
      const dOut = bytesOut - num(prev.bytesOut)
      inboundRate = Math.max(0, (dIn < 0 ? bytesIn : dIn) / dt)
      outboundRate = Math.max(0, (dOut < 0 ? bytesOut : dOut) / dt)
    }
    return { ts: time.getTime(), label: axisLabel(time, spanSeconds), activeConnections: num(point.activeConnections), inboundRate, outboundRate, bytesIn, bytesOut }
  })

const tooltipFormatter = (params: unknown) => {
  const items = (Array.isArray(params) ? params : [params]) as TooltipParam[]
  const title = items[0]?.axisValueLabel || ''
  const rows = items.map((item) => {
    const value = Number(item.value ?? 0)
    const text = item.seriesName === t('tools.tunnel.statsDialog.connChart') ? String(value) : formatRate(value, '0 B/s')
    return `<div style="display:flex;justify-content:space-between;gap:16px"><span>${item.marker || ''}${item.seriesName || ''}</span><b>${text}</b></div>`
  }).join('')
  return `<div style="min-width:160px"><div style="margin-bottom:4px;font-weight:700">${title}</div>${rows}</div>`
}

const baseOption = () => {
  const c = palette.value
  return {
    backgroundColor: 'transparent',
    textStyle: { fontFamily: 'inherit' },
    tooltip: {
      trigger: 'axis' as const,
      confine: true,
      formatter: tooltipFormatter,
      backgroundColor: c.s,
      borderColor: c.line,
      textStyle: { color: c.ink, fontSize: 12 },
    },
    xAxis: {
      type: 'category' as const,
      boundaryGap: false,
      data: points.value.map((p) => p.label),
      axisTick: { show: false },
      axisLine: { lineStyle: { color: c.line } },
      axisLabel: { color: c.ink2, fontSize: 11, hideOverlap: true },
    },
  }
}

const ensureCharts = () => {
  if (rateEl.value && !rateChart) rateChart = echarts.init(rateEl.value, undefined, { renderer: 'canvas' })
  if (connEl.value && !connChart) connChart = echarts.init(connEl.value, undefined, { renderer: 'canvas' })
}

const updateCharts = (animate = false) => {
  ensureCharts()
  const c = palette.value
  const showSymbol = points.value.length > 0 && points.value.length <= 8
  const base = baseOption()
  rateChart?.setOption({
    ...base,
    animation: animate,
    color: [c.p, c.a2],
    grid: { top: 12, left: 8, right: 22, bottom: 4, containLabel: true },
    yAxis: {
      type: 'value',
      min: 0,
      splitNumber: 3,
      splitLine: { lineStyle: { color: c.line, type: 'dashed' } },
      axisLabel: { color: c.ink2, fontSize: 11, formatter: (v: number) => formatRate(v, '0') },
    },
    series: [
      {
        name: t('tools.tunnel.stats.inbound'),
        type: 'line',
        smooth: true,
        showSymbol,
        symbolSize: 6,
        data: points.value.map((p) => p.inboundRate),
        lineStyle: { width: 2.5, color: c.p },
        itemStyle: { color: c.p },
        areaStyle: { color: withAlpha(c.p, 0.14) },
      },
      {
        name: t('tools.tunnel.stats.outbound'),
        type: 'line',
        smooth: true,
        showSymbol,
        symbolSize: 6,
        data: points.value.map((p) => p.outboundRate),
        lineStyle: { width: 2, color: c.a2, type: 'dashed' },
        itemStyle: { color: c.a2 },
      },
    ],
  }, { notMerge: true })
  connChart?.setOption({
    ...base,
    animation: animate,
    xAxis: { ...base.xAxis, boundaryGap: true },
    grid: { top: 8, left: 8, right: 22, bottom: 4, containLabel: true },
    yAxis: {
      type: 'value',
      min: 0,
      minInterval: 1,
      splitNumber: 2,
      splitLine: { lineStyle: { color: c.line, type: 'dashed' } },
      axisLabel: { color: c.ink2, fontSize: 11 },
    },
    series: [
      {
        name: t('tools.tunnel.statsDialog.connChart'),
        type: 'bar',
        barMaxWidth: 14,
        data: points.value.map((p) => p.activeConnections),
        itemStyle: { color: c.a, borderRadius: [4, 4, 0, 0] },
      },
    ],
  }, { notMerge: true })
  requestAnimationFrame(() => {
    rateChart?.resize()
    connChart?.resize()
  })
}

const stopPolling = () => {
  if (pollTimer) { clearTimeout(pollTimer); pollTimer = null }
}
const schedulePolling = () => {
  stopPolling()
  if (!visible.value || !autoRefresh.value || rangeKey.value === 'custom') return
  pollTimer = setTimeout(() => { void loadStats() }, POLL_INTERVAL_MS)
}

const loadStats = async (animate = false) => {
  if (!props.row?.id) return
  loading.value = true
  try {
    const range = resolveRange()
    const { data } = await getTunnelStats({ id: props.row.id, startTimeMs: range.startTimeMs, endTimeMs: range.endTimeMs })
    const snapshot = data?.snapshot
    online.value = !!snapshot?.online
    activeConnections.value = num(snapshot?.activeConnections)
    snapshotIn.value = num(snapshot?.bytesIn)
    snapshotOut.value = num(snapshot?.bytesOut)
    lastStart.value = toDate(snapshot?.lastStartTime)
    lastClose.value = toDate(snapshot?.lastCloseTime)
    sampleSeconds.value = num(data?.sampleIntervalSeconds)
    points.value = buildChartPoints(data?.points || [], (range.endTimeMs - range.startTimeMs) / 1000)
    await nextTick()
    updateCharts(animate)
  } catch (error) {
    showRequestError(error, t('tools.tunnel.stats.loadFailed'))
  } finally {
    loading.value = false
    schedulePolling()
  }
}

const refresh = () => {
  stopPolling()
  void loadStats(true)
}
const setRange = (key: RangeKey) => {
  rangeKey.value = key
  if (key === 'custom') {
    if (!customStart.value || !customEnd.value) {
      const now = new Date()
      customStart.value = new Date(now.getTime() - 3600 * 1000)
      customEnd.value = now
    }
    fillMobileCustomFromDates()
  }
  refresh()
}
const onCustomChange = () => { if (customStart.value && customEnd.value) refresh() }
const restartPolling = () => { if (autoRefresh.value) schedulePolling(); else stopPolling() }

const exportCsv = () => {
  const header = 'time,active_connections,inbound_bytes_per_sec,outbound_bytes_per_sec,inbound_today_bytes,outbound_today_bytes'
  const lines = points.value.map((p) =>
    [formatDateTime(new Date(p.ts)), p.activeConnections, Math.round(p.inboundRate), Math.round(p.outboundRate), p.bytesIn, p.bytesOut].join(','),
  )
  downloadText(`tunnel-${props.row?.name || 'stats'}.csv`, [header, ...lines].join('\n') + '\n', 'text/csv;charset=utf-8')
}

const handleOpened = async () => {
  await nextTick()
  ensureCharts()
  await loadStats(true)
}
const handleClosed = () => {
  stopPolling()
  points.value = []
  rateChart?.dispose()
  connChart?.dispose()
  rateChart = null
  connChart = null
}

watch(visible, (open) => { if (open) void handleOpened(); else handleClosed() })
watch(() => props.row?.id, (id, prev) => { if (visible.value && id && id !== prev) void loadStats(true) })
watch([locale, palette], () => { if (visible.value) updateCharts() }, { deep: true })
const resizeCharts = () => {
  rateChart?.resize()
  connChart?.resize()
}
onMounted(() => window.addEventListener('resize', resizeCharts))
onBeforeUnmount(() => {
  window.removeEventListener('resize', resizeCharts)
  handleClosed()
})
</script>

<style scoped lang="scss">
.tns { display: flex; flex-direction: column; gap: 14px; }
.tns-muted { font-size: 0.75rem; color: var(--el-text-color-secondary); }
.tns-head { display: flex; align-items: center; gap: 12px; min-width: 0; }
.tns-head__sticker {
  width: 48px;
  height: 48px;
  flex: none;
  border-radius: 16px;
  font-size: 0.75rem;
  transform: rotate(-5deg);
}
.tns-head__text { display: flex; flex-direction: column; gap: 6px; min-width: 0; }
.tns-head__route {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 0.875rem;
  color: var(--el-text-color-regular);
  b { color: var(--el-text-color-primary); }
  :deep(svg) { width: 14px; height: 14px; color: var(--m-pInk); }
}
.tns-head__meta { display: flex; align-items: center; flex-wrap: wrap; gap: 6px 10px; }

.tns-toolbar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.tns-spacer { flex: 1; }
.tns-seg {
  display: inline-flex;
  gap: 2px;
  padding: 3px;
  max-width: 100%;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  overflow-x: auto;
  scrollbar-width: none;
}
.tns-seg::-webkit-scrollbar { display: none; }
.tns-seg__btn {
  flex: none;
  min-height: 34px;
  padding: 0 12px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.tns-seg__btn.is-on { background: var(--el-bg-color); color: var(--el-text-color-primary); box-shadow: var(--app-shadow-sm); }
.tns-live {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 4px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  user-select: none;
  input { position: absolute; opacity: 0; pointer-events: none; }
}
.tns-live__dot { width: 8px; height: 8px; border-radius: 50%; background: var(--el-border-color-light); }
.tns-live__dot.is-on { background: var(--m-ok); box-shadow: 0 0 0 4px var(--m-okSoft); }
.tns-custom { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.tns-dt { width: 12rem; height: 38px; font-variant-numeric: tabular-nums; }
.tns-custom__row {
  display: grid;
  grid-template-columns: 4rem 1fr 6rem;
  gap: 6px;
  align-items: center;
  width: 100%;
  span { font-size: 0.75rem; font-weight: 700; color: var(--el-text-color-secondary); }
  .app-input { min-width: 0; height: 40px; }
}

.tns-stats {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
  margin: 0;
  > div { min-width: 0; padding: 12px 14px; border-radius: var(--app-radius); background: var(--el-fill-color-light); }
  dt { font-size: 0.72rem; font-weight: 700; color: var(--el-text-color-secondary); white-space: nowrap; }
  dd {
    margin: 4px 0 0;
    font-family: var(--app-font-display);
    font-size: 1.25rem;
    font-weight: 800;
    color: var(--el-text-color-primary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
    small { display: block; font-family: inherit; font-size: 0.6875rem; font-weight: 600; color: var(--el-text-color-secondary); }
  }
  dd.is-p { color: var(--m-pInk); }
}

.tns-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-lighter);
}
.tns-card__head { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.tns-card__title { flex: 1; font-size: 0.875rem; font-weight: 800; color: var(--el-text-color-primary); }
.tns-legend {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  i { width: 14px; height: 3px; border-radius: 2px; }
  i.is-in { background: var(--m-p); }
  i.is-out { background: var(--m-a2); }
}
.tns-chart-wrap { position: relative; height: 200px; }
.tns-chart-wrap.is-small { height: 110px; }
.tns-chart { width: 100%; height: 100%; }
.tns-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.tns-foot { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }

@media (width <= 768px) {
  .tns-stats { grid-template-columns: 1fr 1fr; }
  .tns-stats > div:last-child { grid-column: 1 / -1; }
  .tns-spacer { display: none; }
  .tns-seg { width: 100%; }
  .tns-seg__btn { flex: 1; padding: 0 8px; }
  .tns-chart-wrap { height: 170px; }
  .tns-foot > :deep(button) { flex: 1; justify-content: center; }
}
</style>
