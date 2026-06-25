<template>
  <BaseDialog
    v-model="visible"
    :title="dialogTitle"
    width="820"
    :show-footer="false"
    @opened="handleOpened"
    @close="handleClosed"
  >
    <div class="pf-stats">
      <!-- 详情 + 实时快照（紧凑双行，避免占用过多高度） -->
      <div class="pf-stats-bar">
        <el-tag size="small" :type="row?.type === PortForwardType.PORT_FORWARD_TYPE_UDP ? 'warning' : 'primary'">
          {{ row?.type === PortForwardType.PORT_FORWARD_TYPE_UDP ? 'UDP' : 'TCP' }}
        </el-tag>
        <span class="pf-stats-route mono">{{ row?.listenAddress }}:{{ row?.listenPort }} → {{ row?.targetAddress }}:{{ row?.targetPort }}</span>
        <span class="pf-stats-state" :class="running ? 'is-running' : 'is-stopped'">
          <span class="pf-stats-dot" />{{ running ? t('tools.portForward.stats.running') : t('tools.portForward.stats.stopped') }}
        </span>
        <span v-if="running && startTimeText" class="pf-stats-muted">{{ t('tools.portForward.stats.startTime') }} {{ startTimeText }}</span>
      </div>
      <div class="pf-stats-metrics">
        <span><i>{{ t('tools.portForward.stats.activeConnections') }}</i><b>{{ activeConnections }}</b></span>
        <span><i>{{ t('tools.portForward.stats.totalConnections') }}</i><b>{{ totalConnections }}</b></span>
        <span><i>{{ t('tools.portForward.stats.inbound') }}</i><b>{{ formatBytes(snapshotIn) }}</b></span>
        <span><i>{{ t('tools.portForward.stats.outbound') }}</i><b>{{ formatBytes(snapshotOut) }}</b></span>
      </div>

      <!-- 工具栏：预设时间段 + 自定义 -->
      <div class="pf-stats-toolbar">
        <el-radio-group v-model="rangeKey" size="small" @change="onRangeChange">
          <el-radio-button value="5m">{{ t('tools.portForward.stats.range5m') }}</el-radio-button>
          <el-radio-button value="30m">{{ t('tools.portForward.stats.range30m') }}</el-radio-button>
          <el-radio-button value="1h">{{ t('tools.portForward.stats.range1h') }}</el-radio-button>
          <el-radio-button value="6h">{{ t('tools.portForward.stats.range6h') }}</el-radio-button>
          <el-radio-button value="24h">{{ t('tools.portForward.stats.range24h') }}</el-radio-button>
          <el-radio-button value="custom">{{ t('tools.portForward.stats.rangeCustom') }}</el-radio-button>
        </el-radio-group>
        <template v-if="rangeKey === 'custom'">
          <el-date-picker
            v-model="customStart"
            type="datetime"
            size="small"
            format="MM-DD HH:mm"
            :placeholder="t('tools.portForward.stats.rangeStart')"
            :clearable="false"
            class="pf-stats-dt"
            @change="onCustomChange"
          />
          <span class="pf-stats-muted">~</span>
          <el-date-picker
            v-model="customEnd"
            type="datetime"
            size="small"
            format="MM-DD HH:mm"
            :placeholder="t('tools.portForward.stats.rangeEnd')"
            :clearable="false"
            class="pf-stats-dt"
            @change="onCustomChange"
          />
        </template>
        <span class="pf-stats-spacer" />
        <el-button size="small" :icon="menuStore.iconComponents.Refresh" :loading="loading" @click="refresh">
          {{ t('tools.portForward.stats.refresh') }}
        </el-button>
        <el-checkbox v-model="autoRefresh" size="small" @change="restartPolling">
          {{ t('tools.portForward.stats.autoRefresh') }}
        </el-checkbox>
      </div>

      <!-- 图表：固定高度容器 + 覆盖式空状态，避免刷新/切换时高度抖动 -->
      <div class="pf-stats-chart-wrap">
        <div ref="chartEl" class="pf-stats-chart" />
        <div v-if="!points.length && !loading" class="pf-stats-empty">
          <el-empty :description="t('tools.portForward.stats.noData')" :image-size="60" />
        </div>
      </div>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import * as echarts from 'echarts'

import { getPortForwardStats } from '@/api/network'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { PortForwardType, type PortForwardInfo, type PortForwardStatPoint } from '@/types/v1/network'
import { showRequestError } from '@/utils/request'

type RangeKey = '5m' | '30m' | '1h' | '6h' | '24h' | 'custom'

type ChartPoint = {
  time: string
  activeConnections: number
  inboundRate: number
  outboundRate: number
}

type ChartTooltipParam = {
  axisValueLabel?: string
  marker?: string
  seriesName?: string
  value?: number | string
}

const POLL_INTERVAL_MS = 10000
const PRESET_SECONDS: Record<Exclude<RangeKey, 'custom'>, number> = {
  '5m': 300,
  '30m': 1800,
  '1h': 3600,
  '6h': 21600,
  '24h': 86400,
}

const props = defineProps<{
  modelValue: boolean
  row: PortForwardInfo | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const menuStore = useMenuStore()
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
const running = ref(false)
const startTime = ref<Date | null>(null)
const snapshotIn = ref(0)
const snapshotOut = ref(0)
const activeConnections = ref(0)
const totalConnections = ref(0)
const points = ref<ChartPoint[]>([])

let chart: echarts.ECharts | null = null
let pollTimer: ReturnType<typeof setTimeout> | null = null

const dialogTitle = computed(() => {
  const name = props.row?.name
  return name
    ? t('tools.portForward.stats.titleWithName', { name })
    : t('tools.portForward.stats.title')
})

const startTimeText = computed(() => {
  if (!running.value || !startTime.value) return ''
  const d = startTime.value
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
})

const chartLabels = computed(() => ({
  connections: t('tools.portForward.stats.connectionsChart'),
  inbound: t('tools.portForward.stats.inbound'),
  outbound: t('tools.portForward.stats.outbound'),
}))

const toNumber = (value?: number | string) => Number(value || 0)

const formatBytes = (bytes?: number | string) => {
  const n = Number(bytes)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

const formatRate = (bytesPerSecond?: number | string) => `${formatBytes(bytesPerSecond)}/s`

const formatAxisTime = (date: Date) => {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(date.getHours())}:${pad(date.getMinutes())}`
}

// 解析当前生效的时间段（Unix 毫秒）。预设按当前时间滑动，自定义用选择器的绝对区间。
const resolveRange = (): { startTimeMs: number; endTimeMs: number } => {
  if (rangeKey.value === 'custom' && customStart.value && customEnd.value) {
    let start = customStart.value.getTime()
    let end = customEnd.value.getTime()
    if (start > end) [start, end] = [end, start]
    return { startTimeMs: start, endTimeMs: end }
  }
  const end = Date.now()
  const seconds = PRESET_SECONDS[(rangeKey.value === 'custom' ? '1h' : rangeKey.value)]
  return { startTimeMs: end - seconds * 1000, endTimeMs: end }
}

// 累计流量按相邻采样点差值换算为速率（重启会使累计值归零，负值钳为 0）。
const buildChartPoints = (raw: PortForwardStatPoint[]): ChartPoint[] => {
  return raw.map((point, index) => {
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
    return {
      time: formatAxisTime(time),
      activeConnections: toNumber(point.activeConnections),
      inboundRate,
      outboundRate,
    }
  })
}

const tooltipFormatter = (params: unknown) => {
  const items = (Array.isArray(params) ? params : [params]) as ChartTooltipParam[]
  const title = items[0]?.axisValueLabel || ''
  const rows = items
    .map((item) => {
      const name = item.seriesName || ''
      const value = Number(item.value ?? 0)
      const text = name === chartLabels.value.connections ? String(value) : formatRate(value)
      return `<div class="pf-stats-tooltip__row"><span>${item.marker || ''}${name}</span><strong>${text}</strong></div>`
    })
    .join('')
  return `<div class="pf-stats-tooltip"><div class="pf-stats-tooltip__title">${title}</div>${rows}</div>`
}

const ensureChart = () => {
  if (!chartEl.value) return
  if (!chart) chart = echarts.init(chartEl.value)
}

// animate=true：用户切换时间段/手动刷新时播放过渡动画；轮询刷新传 false 原地更新，避免抖动
const updateChart = (animate = false) => {
  ensureChart()
  if (!chart) return
  const labels = chartLabels.value

  const option: echarts.EChartsOption = {
    animation: animate,
    animationDuration: 600,
    animationDurationUpdate: 500,
    animationEasing: 'cubicOut',
    animationEasingUpdate: 'cubicOut',
    color: ['#2f7cff', '#13b981', '#f59e0b'],
    tooltip: { trigger: 'axis', formatter: tooltipFormatter, confine: true },
    legend: {
      top: 4,
      left: 8,
      itemWidth: 10,
      itemHeight: 10,
      itemGap: 12,
      data: [labels.connections, labels.inbound, labels.outbound],
    },
    // 固定边距，避免 y 轴标签宽度随数值变化导致绘图区横向抖动
    grid: { top: 38, left: 44, right: 70, bottom: 28 },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: points.value.map((item) => item.time),
      axisTick: { alignWithLabel: true },
    },
    yAxis: [
      {
        type: 'value',
        min: 0,
        minInterval: 1,
        splitLine: { lineStyle: { color: '#e5e7eb' } },
      },
      {
        type: 'value',
        min: 0,
        axisLabel: { formatter: (value: number) => formatRate(value) },
        splitLine: { show: false },
      },
    ],
    series: [
      {
        name: labels.connections,
        type: 'line',
        yAxisIndex: 0,
        data: points.value.map((item) => item.activeConnections),
        showSymbol: false,
        smooth: true,
        lineStyle: { width: 2 },
      },
      {
        name: labels.inbound,
        type: 'line',
        yAxisIndex: 1,
        data: points.value.map((item) => item.inboundRate),
        showSymbol: false,
        smooth: true,
        areaStyle: { opacity: 0.08 },
        lineStyle: { width: 2 },
      },
      {
        name: labels.outbound,
        type: 'line',
        yAxisIndex: 1,
        data: points.value.map((item) => item.outboundRate),
        showSymbol: false,
        smooth: true,
        areaStyle: { opacity: 0.08 },
        lineStyle: { width: 2 },
      },
    ],
  }
  // 合并更新（非 notMerge），原地刷新数据，避免重建导致的闪烁/抖动
  chart.setOption(option)
}

const loadStats = async (animate = false) => {
  if (!props.row?.id || loading.value) return
  loading.value = true
  try {
    const range = resolveRange()
    const { data } = await getPortForwardStats({
      id: props.row.id,
      startTimeMs: range.startTimeMs,
      endTimeMs: range.endTimeMs,
    })
    const snapshot = data?.snapshot
    running.value = !!snapshot?.running
    activeConnections.value = toNumber(snapshot?.activeConnections)
    totalConnections.value = toNumber(snapshot?.totalConnections)
    snapshotIn.value = toNumber(snapshot?.bytesIn)
    snapshotOut.value = toNumber(snapshot?.bytesOut)
    startTime.value = snapshot?.startTime ? new Date(snapshot.startTime) : null
    points.value = buildChartPoints(data?.points || [])
    await nextTick()
    updateChart(animate)
  } catch (error) {
    showRequestError(error, t('tools.portForward.stats.loadFailed'))
  } finally {
    loading.value = false
    schedulePolling()
  }
}

const stopPolling = () => {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
}

const schedulePolling = () => {
  stopPolling()
  if (!visible.value || !autoRefresh.value) return
  pollTimer = setTimeout(() => {
    void loadStats()
  }, POLL_INTERVAL_MS)
}

const refresh = () => {
  stopPolling()
  void loadStats(true)
}

const onRangeChange = () => {
  if (rangeKey.value === 'custom' && (!customStart.value || !customEnd.value)) {
    const now = new Date()
    customStart.value = new Date(now.getTime() - 3600 * 1000)
    customEnd.value = now
  }
  refresh()
}

const onCustomChange = () => {
  if (customStart.value && customEnd.value) refresh()
}

const restartPolling = () => {
  if (autoRefresh.value) schedulePolling()
  else stopPolling()
}

const handleOpened = async () => {
  await nextTick()
  ensureChart()
  chart?.resize()
  await loadStats(true)
}

const handleClosed = () => {
  stopPolling()
  points.value = []
}

const resizeChart = () => chart?.resize()

watch(locale, () => {
  if (visible.value) updateChart()
})

onMounted(() => {
  window.addEventListener('resize', resizeChart)
})

onBeforeUnmount(() => {
  stopPolling()
  window.removeEventListener('resize', resizeChart)
  chart?.dispose()
  chart = null
})
</script>

<style scoped lang="scss">
.pf-stats {
  display: flex;
  flex-direction: column;
  gap: 0.6rem;
}

.pf-stats-bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem 0.75rem;
  font-size: 0.82rem;
}

.pf-stats-route {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.pf-stats-state {
  display: inline-flex;
  align-items: center;
  gap: 0.3rem;
}

.pf-stats-state.is-running {
  color: var(--el-color-success);
}

.pf-stats-state.is-stopped {
  color: var(--el-text-color-secondary);
}

.pf-stats-dot {
  width: 0.45rem;
  height: 0.45rem;
  border-radius: 50%;
  background: currentColor;
}

.pf-stats-muted {
  color: var(--el-text-color-secondary);
}

.pf-stats-metrics {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem 1.4rem;
  padding: 0.5rem 0.75rem;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;

  span {
    display: inline-flex;
    align-items: baseline;
    gap: 0.4rem;
  }

  i {
    color: var(--el-text-color-secondary);
    font-style: normal;
    font-size: 0.78rem;
  }

  b {
    color: var(--el-text-color-primary);
    font-size: 0.95rem;
    font-variant-numeric: tabular-nums;
  }
}

.pf-stats-toolbar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.pf-stats-dt {
  width: 9.5rem;
}

.pf-stats-spacer {
  flex: 1;
}

.pf-stats-chart-wrap {
  position: relative;
  width: 100%;
  height: 16rem;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 8px;
}

.pf-stats-chart {
  width: 100%;
  height: 100%;
}

.pf-stats-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--el-bg-color);
  border-radius: 8px;
}

:global(.pf-stats-tooltip) {
  min-width: 11rem;
}

:global(.pf-stats-tooltip__title) {
  margin-bottom: 0.35rem;
  color: var(--el-text-color-regular);
  font-weight: 600;
}

:global(.pf-stats-tooltip__row) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  line-height: 1.6;
}

:global(.pf-stats-tooltip__row span) {
  color: var(--el-text-color-secondary);
}

.mono {
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', monospace;
}

@media (max-width: 768px) {
  .pf-stats-spacer {
    display: none;
  }

  .pf-stats-dt {
    flex: 1;
    width: auto;
  }
}
</style>
