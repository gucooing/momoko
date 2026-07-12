<!-- 隧道统计（重写）：FormDialog + 摘要 + 时间段 seg + echarts。逻辑保留（速率差分、轮询）。 -->
<template>
  <FormDialog
    v-model="visible"
    :title="dialogTitle"
    :width="820"
    :show-footer="false"
    overflow-visible
    @close="handleClosed"
  >
    <div class="tn-stats">
      <div class="tn-stats-bar">
        <StatusPill variant="primary" :label="typeLabel" :dot="false" />
        <span class="tn-stats-route mono">{{ routeText }}</span>
        <StatusPill :variant="online ? 'success' : 'neutral'" :label="online ? t('tools.tunnel.stats.online') : t('tools.tunnel.stats.offline')" />
      </div>
      <div class="tn-stats-metrics">
        <span><i>{{ t('tools.tunnel.stats.activeConnections') }}</i><b>{{ activeConnections }}</b></span>
        <span><i>{{ t('tools.tunnel.stats.inbound') }}</i><b>{{ formatBytes(snapshotIn) }}</b></span>
        <span><i>{{ t('tools.tunnel.stats.outbound') }}</i><b>{{ formatBytes(snapshotOut) }}</b></span>
      </div>

      <div class="tn-stats-toolbar">
        <div class="seg">
          <button
            v-for="opt in rangeOptions"
            :key="opt.value"
            type="button"
            class="seg__btn"
            :class="{ 'is-active': rangeKey === opt.value }"
            @click="setRange(opt.value)"
          >{{ opt.label }}</button>
        </div>
        <template v-if="rangeKey === 'custom'">
          <input
            v-model="customStartLocal"
            type="datetime-local"
            class="app-input tn-stats-dt"
            :aria-label="t('tools.tunnel.stats.rangeStart')"
            @change="onCustomChange"
          />
          <span class="tn-stats-muted">~</span>
          <input
            v-model="customEndLocal"
            type="datetime-local"
            class="app-input tn-stats-dt"
            :aria-label="t('tools.tunnel.stats.rangeEnd')"
            @change="onCustomChange"
          />
        </template>
        <span class="tn-stats-spacer" />
        <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-refresh-cw" :loading="loading" @click="refresh">
          {{ t('tools.tunnel.stats.refresh') }}
        </UButton>
        <label class="tn-stats-check">
          <input v-model="autoRefresh" type="checkbox" @change="restartPolling" />
          <span>{{ t('tools.tunnel.stats.autoRefresh') }}</span>
        </label>
      </div>

      <div class="tn-stats-chart-wrap">
        <div ref="chartEl" class="tn-stats-chart" />
        <div v-if="!points.length && !loading" class="tn-stats-empty">
          <EmptyState icon="HOutline:ChartBarIcon" :title="t('tools.tunnel.stats.noData')" />
        </div>
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

type RangeKey = '5m' | '30m' | '1h' | '6h' | '24h' | 'custom'
type ChartPoint = { time: string; activeConnections: number; inboundRate: number; outboundRate: number }
type ChartTooltipParam = { axisValueLabel?: string; marker?: string; seriesName?: string; value?: number | string }

const POLL_INTERVAL_MS = 10000
const PRESET_SECONDS: Record<Exclude<RangeKey, 'custom'>, number> = {
  '5m': 300, '30m': 1800, '1h': 3600, '6h': 21600, '24h': 86400,
}

const props = defineProps<{ modelValue: boolean; row: TunnelInfo | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const { t, locale } = useI18n()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const chartEl = ref<HTMLDivElement | null>(null)
const rangeKey = ref<RangeKey>('1h')
const customStart = ref<Date | null>(null)
const customEnd = ref<Date | null>(null)
const loading = ref(false)
const autoRefresh = ref(true)
const online = ref(false)
const snapshotIn = ref(0)
const snapshotOut = ref(0)
const activeConnections = ref(0)
const points = ref<ChartPoint[]>([])

let chart: echarts.ECharts | null = null
let pollTimer: ReturnType<typeof setTimeout> | null = null

const toLocalInput = (d: Date) => {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`
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
const typeLabel = computed(() => (props.row?.type || TunnelType.TUNNEL_TYPE_TCP).replace('TUNNEL_TYPE_', ''))
const routeText = computed(() => {
  const row = props.row
  if (!row) return ''
  if ([TunnelType.TUNNEL_TYPE_HTTP, TunnelType.TUNNEL_TYPE_HTTPS].includes(row.type)) {
    return row.customDomains || row.subdomain || '-'
  }
  const local = `${row.localIp || '127.0.0.1'}:${row.localPort || 0}`
  return row.remotePort ? `:${row.remotePort} → ${local}` : local
})
const chartLabels = computed(() => ({
  connections: t('tools.tunnel.stats.connectionsChart'),
  inbound: t('tools.tunnel.stats.inbound'),
  outbound: t('tools.tunnel.stats.outbound'),
}))

const toNumber = (value?: number | string) => Number(value || 0)
const formatBytes = (bytes?: number | string) => {
  const n = Number(bytes)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}
const formatRate = (bytesPerSecond?: number | string) => `${formatBytes(bytesPerSecond)}/s`
const formatAxisTime = (date: Date) => {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

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

const buildChartPoints = (raw: TunnelStatPoint[]): ChartPoint[] =>
  raw.map((point, index) => {
    const time = point.time ? new Date(point.time) : new Date()
    const bytesIn = toNumber(point.bytesIn)
    const bytesOut = toNumber(point.bytesOut)
    let inboundRate = 0
    let outboundRate = 0
    const prev = index > 0 ? raw[index - 1] : undefined
    if (prev) {
      const prevTime = prev.time ? new Date(prev.time).getTime() : time.getTime()
      const dt = Math.max(1, (time.getTime() - prevTime) / 1000)
      inboundRate = Math.max(0, (bytesIn - toNumber(prev.bytesIn)) / dt)
      outboundRate = Math.max(0, (bytesOut - toNumber(prev.bytesOut)) / dt)
    }
    return { time: formatAxisTime(time), activeConnections: toNumber(point.activeConnections), inboundRate, outboundRate }
  })

const tooltipFormatter = (params: unknown) => {
  const items = (Array.isArray(params) ? params : [params]) as ChartTooltipParam[]
  const title = items[0]?.axisValueLabel || ''
  const rows = items.map((item) => {
    const name = item.seriesName || ''
    const value = Number(item.value ?? 0)
    const text = name === chartLabels.value.connections ? String(value) : formatRate(value)
    return `<div class="tn-stats-tooltip__row"><span>${item.marker || ''}${name}</span><strong>${text}</strong></div>`
  }).join('')
  return `<div class="tn-stats-tooltip"><div class="tn-stats-tooltip__title">${title}</div>${rows}</div>`
}

const themeColors = () => {
  const isDark = document.documentElement.classList.contains('dark')
  return {
    isDark,
    text: isDark ? '#c9d1d9' : '#4b5563',
    muted: isDark ? '#8b949e' : '#9ca3af',
    split: isDark ? 'rgba(148,163,184,0.18)' : '#e5e7eb',
    bg: isDark ? '#0d1117' : '#ffffff',
  }
}

const ensureChart = () => {
  if (!chartEl.value) return
  if (!chart) chart = echarts.init(chartEl.value, undefined, { renderer: 'canvas' })
}

const updateChart = (animate = false) => {
  ensureChart()
  if (!chart) return
  const labels = chartLabels.value
  const theme = themeColors()
  const showSymbol = points.value.length > 0 && points.value.length <= 8
  chart.setOption({
    backgroundColor: theme.bg,
    animation: animate,
    animationDuration: 400,
    animationDurationUpdate: 300,
    color: ['#2f7cff', '#13b981', '#f59e0b'],
    tooltip: {
      trigger: 'axis',
      formatter: tooltipFormatter,
      confine: true,
      backgroundColor: theme.isDark ? '#161b22' : '#fff',
      borderColor: theme.split,
      textStyle: { color: theme.text, fontSize: 12 },
    },
    legend: {
      top: 4, left: 8, itemWidth: 10, itemHeight: 10, itemGap: 12,
      textStyle: { color: theme.text, fontSize: 11 },
      data: [labels.connections, labels.inbound, labels.outbound],
    },
    grid: { top: 40, left: 48, right: 72, bottom: 30 },
    xAxis: {
      type: 'category', boundaryGap: false,
      data: points.value.map((item) => item.time),
      axisTick: { alignWithLabel: true },
      axisLabel: { color: theme.muted, fontSize: 11 },
      axisLine: { lineStyle: { color: theme.split } },
    },
    yAxis: [
      { type: 'value', min: 0, minInterval: 1, splitLine: { lineStyle: { color: theme.split } }, axisLabel: { color: theme.muted, fontSize: 11 } },
      { type: 'value', min: 0, axisLabel: { formatter: (value: number) => formatRate(value), color: theme.muted, fontSize: 11 }, splitLine: { show: false } },
    ],
    series: [
      { name: labels.connections, type: 'line', yAxisIndex: 0, data: points.value.map((i) => i.activeConnections), showSymbol, symbolSize: 6, smooth: true, lineStyle: { width: 2 } },
      { name: labels.inbound, type: 'line', yAxisIndex: 1, data: points.value.map((i) => i.inboundRate), showSymbol, symbolSize: 6, smooth: true, areaStyle: { opacity: theme.isDark ? 0.12 : 0.08 }, lineStyle: { width: 2 } },
      { name: labels.outbound, type: 'line', yAxisIndex: 1, data: points.value.map((i) => i.outboundRate), showSymbol, symbolSize: 6, smooth: true, areaStyle: { opacity: theme.isDark ? 0.12 : 0.08 }, lineStyle: { width: 2 } },
    ],
  })
  requestAnimationFrame(() => chart?.resize())
}

const stopPolling = () => {
  if (pollTimer) { clearTimeout(pollTimer); pollTimer = null }
}
const schedulePolling = () => {
  stopPolling()
  if (!visible.value || !autoRefresh.value) return
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
    activeConnections.value = toNumber(snapshot?.activeConnections)
    snapshotIn.value = toNumber(snapshot?.bytesIn)
    snapshotOut.value = toNumber(snapshot?.bytesOut)
    points.value = buildChartPoints(data?.points || [])
    await nextTick()
    updateChart(animate)
    await nextTick()
    chart?.resize()
  } catch (error) {
    showRequestError(error, t('tools.tunnel.stats.loadFailed'))
  } finally {
    loading.value = false
    schedulePolling()
  }
}

const refresh = () => {
  stopPolling()
  loading.value = false
  void loadStats(true)
}
const setRange = (key: RangeKey) => {
  rangeKey.value = key
  if (key === 'custom' && (!customStart.value || !customEnd.value)) {
    const now = new Date()
    customStart.value = new Date(now.getTime() - 3600 * 1000)
    customEnd.value = now
  }
  refresh()
}
const onCustomChange = () => { if (customStart.value && customEnd.value) refresh() }
const restartPolling = () => { if (autoRefresh.value) schedulePolling(); else stopPolling() }

const handleOpened = async () => {
  await nextTick()
  ensureChart()
  chart?.resize()
  await loadStats(true)
}
const handleClosed = () => { stopPolling(); points.value = [] }

watch(visible, (open) => { if (open) void handleOpened(); else handleClosed() })
watch(locale, () => { if (visible.value) updateChart() })
const resizeChart = () => chart?.resize()
onMounted(() => window.addEventListener('resize', resizeChart))
onBeforeUnmount(() => {
  stopPolling()
  window.removeEventListener('resize', resizeChart)
  chart?.dispose()
  chart = null
})
</script>

<style scoped lang="scss">
.tn-stats { display: flex; flex-direction: column; gap: 10px; }
.tn-stats-bar { display: flex; align-items: center; flex-wrap: wrap; gap: 8px 12px; font-size: 0.8125rem; }
.tn-stats-route { color: var(--el-text-color-primary); font-weight: 600; }
.tn-stats-metrics {
  display: flex; flex-wrap: wrap; gap: 6px 20px; padding: 8px 12px;
  border: 1px solid var(--el-border-color-lighter); border-radius: var(--app-radius);
  span { display: inline-flex; align-items: baseline; gap: 6px; }
  i { color: var(--el-text-color-secondary); font-style: normal; font-size: 0.75rem; }
  b { color: var(--el-text-color-primary); font-size: 0.9rem; font-variant-numeric: tabular-nums; }
}
.tn-stats-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 8px;
}
.seg {
  display: inline-flex;
  flex-direction: row;
  flex-wrap: nowrap;
  align-items: center;
  flex: 0 1 auto;
  padding: 2px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
  max-width: 100%;
  overflow-x: auto;
}
.seg__btn {
  flex: 0 0 auto;
  white-space: nowrap;
  padding: 5px 9px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  line-height: 1.2;
}
.seg__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.tn-stats-dt {
  width: 11.5rem;
  max-width: 100%;
  height: 32px;
  font-variant-numeric: tabular-nums;
  font-size: 0.8125rem;
}
.tn-stats-spacer { flex: 1 1 8px; min-width: 0; }
.tn-stats-muted { color: var(--el-text-color-secondary); }
.tn-stats-check {
  display: inline-flex; align-items: center; gap: 6px; font-size: 0.78rem;
  color: var(--el-text-color-regular); cursor: pointer; user-select: none;
}
.tn-stats-check input { accent-color: var(--el-color-primary); }
.tn-stats-chart-wrap {
  position: relative;
  width: 100%;
  height: 16rem;
  min-height: 256px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  overflow: hidden;
  background: var(--el-bg-color);
}
.tn-stats-chart {
  width: 100%;
  height: 100%;
  min-height: 256px;
}
.tn-stats-empty {
  position: absolute; inset: 0; display: flex; align-items: center; justify-content: center;
  background: var(--el-bg-color); border-radius: var(--app-radius);
}
.mono { font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', Consolas, monospace; }
:global(.tn-stats-tooltip) { min-width: 11rem; }
:global(.tn-stats-tooltip__title) { margin-bottom: 0.35rem; color: var(--el-text-color-regular); font-weight: 600; }
:global(.tn-stats-tooltip__row) { display: flex; justify-content: space-between; gap: 1rem; line-height: 1.6; }
:global(.tn-stats-tooltip__row span) { color: var(--el-text-color-secondary); }
@media (width <= 768px) {
  .tn-stats-spacer { display: none; }
  .tn-stats-dt { flex: 1; width: auto; }
}
</style>
