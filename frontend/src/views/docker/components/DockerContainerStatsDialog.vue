<!-- 容器统计：FormDialog 壳 + 贴纸头（容器名 / 状态 / 刷新间隔分段 / 刷新）→ 4 张指标卡（CPU、内存带进度条、网络、磁盘 IO）
     → 「CPU 与内存」百分比曲线 + 「网络与磁盘 IO」速率曲线。轮询 getDockerContainerStats，保留 5 分钟窗口；ECharts 颜色取自当前配色。 -->
<template>
  <FormDialog v-model="visible" :title="dialogTitle" :width="900" :show-footer="false" @close="handleClosed">
    <div class="dks">
      <div class="dks-head">
        <span class="m-sticker m-sticker--a2 dks-head__sticker"><component :is="menuStore.iconComponents['HOutline:ChartBarIcon']" /></span>
        <span class="dks-head__text">
          <span class="dks-head__name m-mono">{{ containerName || containerId }}</span>
          <span class="dks-head__sub">{{ t('docker.statsDialog.sub') }}</span>
        </span>
        <StatusPill :variant="status === 'polling' ? 'success' : status === 'error' ? 'error' : 'neutral'" :label="statusLabel" />
      </div>

      <div class="dks-toolbar">
        <span class="dks-toolbar__label">{{ t('docker.statsDialog.intervalLabel') }}</span>
        <div class="dks-seg" role="radiogroup" :aria-label="t('docker.statsDialog.intervalLabel')">
          <button
            v-for="n in INTERVALS"
            :key="n"
            type="button"
            role="radio"
            class="dks-seg__btn"
            :class="{ 'is-on': pollIntervalSeconds === n }"
            :aria-checked="pollIntervalSeconds === n"
            @click="setPollInterval(n)"
          >{{ t('docker.statsDialog.secShort', { n }) }}</button>
        </div>
        <span class="dks-spacer" />
        <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('docker.common.refresh')" :box="36" :size="18" @click="refresh" />
      </div>

      <div class="dks-tiles">
        <div class="dks-tile">
          <span class="dks-tile__label">{{ t('docker.statsDialog.cpuUsage') }}</span>
          <span class="dks-tile__value m-num">{{ sample ? toPercent(sample.cpuPercent) : '—' }}</span>
          <span class="dks-bar"><i :style="{ width: barWidth(sample?.cpuPercent), background: 'var(--m-p)' }" /></span>
        </div>
        <div class="dks-tile">
          <span class="dks-tile__label">{{ t('docker.statsDialog.memoryUsage') }}</span>
          <span class="dks-tile__value m-num">{{ sample ? formatBytes(sample.memoryUsage) : '—' }}</span>
          <span class="dks-bar"><i :style="{ width: barWidth(sample?.memoryPercent), background: 'var(--m-a2)' }" /></span>
          <span class="dks-tile__sub m-num">
            {{ sample ? toPercent(sample.memoryPercent) : '—' }} ·
            {{ sample?.memoryLimit ? t('docker.statsDialog.ofLimit', { limit: formatBytes(sample.memoryLimit) }) : t('docker.statsDialog.noLimit') }}
          </span>
        </div>
        <div class="dks-tile">
          <span class="dks-tile__label">{{ t('docker.statsDialog.netRate') }}</span>
          <span class="dks-tile__pair m-num">
            <span><i class="dks-down">↓</i> {{ sample ? formatRate(sample.networkRxRate) : '—' }}</span>
            <span><i class="dks-up">↑</i> {{ sample ? formatRate(sample.networkTxRate) : '—' }}</span>
          </span>
          <span class="dks-tile__sub m-num">{{ sample ? `${formatBytes(sample.networkRx)} / ${formatBytes(sample.networkTx)}` : '' }}</span>
        </div>
        <div class="dks-tile">
          <span class="dks-tile__label">{{ t('docker.statsDialog.blockRate') }}</span>
          <span class="dks-tile__pair m-num">
            <span><i class="dks-down">R</i> {{ sample ? formatRate(sample.blockReadRate) : '—' }}</span>
            <span><i class="dks-up">W</i> {{ sample ? formatRate(sample.blockWriteRate) : '—' }}</span>
          </span>
          <span class="dks-tile__sub m-num">{{ sample ? `${formatBytes(sample.blockRead)} / ${formatBytes(sample.blockWrite)}` : '' }}</span>
        </div>
      </div>

      <section class="dks-card">
        <div class="dks-card__head">
          <span class="dks-card__title">{{ t('docker.statsDialog.cpuChart') }}</span>
          <span class="dks-legend"><i style="background: var(--m-p)" />{{ chartLabels.cpu }}</span>
          <span class="dks-legend"><i style="background: var(--m-a2)" />{{ chartLabels.memory }}</span>
        </div>
        <div class="dks-chart-wrap">
          <div ref="cpuEl" class="dks-chart" />
          <div v-if="!history.length && !loading" class="dks-empty">{{ t('docker.statsDialog.noData') }}</div>
        </div>
      </section>

      <section class="dks-card">
        <div class="dks-card__head">
          <span class="dks-card__title">{{ t('docker.statsDialog.ioChart') }}</span>
          <span class="dks-legend"><i style="background: var(--m-p)" />{{ chartLabels.networkRx }}</span>
          <span class="dks-legend"><i style="background: var(--m-a2)" />{{ chartLabels.networkTx }}</span>
          <span class="dks-legend"><i style="background: var(--m-a)" />{{ chartLabels.blockRead }}</span>
          <span class="dks-legend"><i style="background: var(--m-ok)" />{{ chartLabels.blockWrite }}</span>
        </div>
        <div class="dks-chart-wrap">
          <div ref="ioEl" class="dks-chart" />
          <div v-if="!history.length && !loading" class="dks-empty">{{ t('docker.statsDialog.noData') }}</div>
        </div>
      </section>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import * as echarts from 'echarts'

import { getDockerContainerStats } from '@/api/docker'
import { DockerBlkioOperation } from '@/types/v1/docker'
import type { DockerBlkioEntry, DockerContainerStats } from '@/types/v1/docker'
import { showRequestError } from '@/utils/request'
import { usePaletteColors, withAlpha } from '@/composables/usePaletteColors'

type PollStatus = 'stopped' | 'polling' | 'error'

type StatsSample = {
  timestamp: number
  time: string
  cpuPercent: number
  memoryPercent: number
  memoryUsage: number
  memoryLimit: number
  networkRx: number
  networkTx: number
  blockRead: number
  blockWrite: number
  networkRxRate: number
  networkTxRate: number
  blockReadRate: number
  blockWriteRate: number
}

type ChartTooltipParam = {
  axisValueLabel?: string
  data?: number | string
  marker?: string
  name?: string
  seriesName?: string
  value?: number | string
}

const HISTORY_WINDOW_MS = 5 * 60 * 1000
const INTERVALS = [1, 3, 5, 10]
const MAX_HISTORY_POINTS = 240

const props = defineProps<{
  modelValue: boolean
  containerId: string
  containerName?: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
}>()

const { t, locale } = useI18n()
const menuStore = useMenuStore()
const palette = usePaletteColors()
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const cpuEl = ref<HTMLDivElement | null>(null)
const ioEl = ref<HTMLDivElement | null>(null)
const pollIntervalSeconds = ref(3)
const loading = ref(false)
const status = ref<PollStatus>('stopped')
const history = ref<StatsSample[]>([])

let cpuChart: echarts.ECharts | null = null
let ioChart: echarts.ECharts | null = null
let pollTimer: ReturnType<typeof setTimeout> | null = null

const dialogTitle = computed(() => {
  const name = props.containerName || props.containerId
  return name ? t('docker.statsDialog.titleWithName', { name }) : t('docker.statsDialog.title')
})

const statusLabel = computed(() => {
  if (status.value === 'polling') return t('docker.common.polling')
  if (status.value === 'error') return t('docker.common.requestFailed')
  return t('docker.common.stopped')
})

const currentSample = computed(() => history.value[history.value.length - 1])
const sample = currentSample
const barWidth = (v?: number) => `${Math.min(100, Math.max(0, v || 0)).toFixed(1)}%`

const chartLabels = computed(() => ({
  cpu: t('docker.common.cpu'),
  memory: t('docker.statsDialog.memorySeries'),
  networkRx: t('docker.statsDialog.networkRx'),
  networkTx: t('docker.statsDialog.networkTx'),
  blockRead: t('docker.statsDialog.blockRead'),
  blockWrite: t('docker.statsDialog.blockWrite'),
}))

const toNumber = (value?: number | string) => Number(value || 0)

const formatTime = (date: Date) => {
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

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
const toPercent = (value: number) => `${Math.max(0, value).toFixed(2)}%`

const formatSeriesValue = (seriesName: string, value: number) => {
  if (seriesName === chartLabels.value.cpu || seriesName === chartLabels.value.memory) return toPercent(value)
  return formatRate(value)
}

const tooltipValue = (item: ChartTooltipParam) => Number(item.value ?? item.data ?? 0)

const tooltipFormatter = (params: unknown) => {
  const items = (Array.isArray(params) ? params : [params]) as ChartTooltipParam[]
  const title = items[0]?.axisValueLabel || items[0]?.name || ''
  const rows = items.map((item) => {
    const name = item.seriesName || ''
    const marker = item.marker || ''
    const value = formatSeriesValue(name, tooltipValue(item))
    return `<div class="dk-stats-tooltip__row"><span>${marker}${name}</span><strong>${value}</strong></div>`
  }).join('')
  return `<div class="dk-stats-tooltip"><div class="dk-stats-tooltip__title">${title}</div>${rows}</div>`
}

const sumNetwork = (networks: DockerContainerStats['networks'], field: 'rxBytes' | 'txBytes') => {
  return (networks || []).reduce((sum, item) => sum + toNumber(item?.[field]), 0)
}

const sumBlockIo = (entries: DockerBlkioEntry[] | undefined, operation: DockerBlkioOperation) => {
  return (entries || []).reduce((sum, item) => {
    return item.op === operation ? sum + toNumber(item.value) : sum
  }, 0)
}

const parseStatsSample = (data: DockerContainerStats, previous?: StatsSample): StatsSample => {
  const now = new Date()
  const cpuStats = data.cpuStats
  const preCpuStats = data.precpuStats
  const cpuDelta = toNumber(cpuStats?.cpuUsage?.totalUsage) - toNumber(preCpuStats?.cpuUsage?.totalUsage)
  const systemDelta = toNumber(cpuStats?.systemCpuUsage) - toNumber(preCpuStats?.systemCpuUsage)
  const onlineCpus = toNumber(cpuStats?.onlineCpus) || 1
  const cpuPercent = systemDelta > 0 ? (cpuDelta / systemDelta) * onlineCpus * 100 : 0
  const memoryStats = data.memoryStats
  const memoryUsage = toNumber(memoryStats?.usage)
  const memoryLimit = toNumber(memoryStats?.limit)
  const networkRx = sumNetwork(data.networks, 'rxBytes')
  const networkTx = sumNetwork(data.networks, 'txBytes')
  const blockRead = sumBlockIo(data.blkioStats?.ioServiceBytesRecursive, DockerBlkioOperation.DOCKER_BLKIO_OPERATION_READ)
  const blockWrite = sumBlockIo(data.blkioStats?.ioServiceBytesRecursive, DockerBlkioOperation.DOCKER_BLKIO_OPERATION_WRITE)
  const interval = previous ? Math.max(1, (now.getTime() - previous.timestamp) / 1000) : Math.max(1, pollIntervalSeconds.value)

  return {
    timestamp: now.getTime(),
    time: formatTime(now),
    cpuPercent,
    memoryPercent: memoryLimit > 0 ? (memoryUsage / memoryLimit) * 100 : 0,
    memoryUsage,
    memoryLimit,
    networkRx,
    networkTx,
    blockRead,
    blockWrite,
    networkRxRate: previous ? Math.max(0, (networkRx - previous.networkRx) / interval) : 0,
    networkTxRate: previous ? Math.max(0, (networkTx - previous.networkTx) / interval) : 0,
    blockReadRate: previous ? Math.max(0, (blockRead - previous.blockRead) / interval) : 0,
    blockWriteRate: previous ? Math.max(0, (blockWrite - previous.blockWrite) / interval) : 0,
  }
}

const pruneHistory = (items: StatsSample[], now: number) => {
  return items
    .filter((item) => now - item.timestamp <= HISTORY_WINDOW_MS)
    .slice(-MAX_HISTORY_POINTS)
}

const ensureChart = () => {
  if (cpuEl.value && !cpuChart) cpuChart = echarts.init(cpuEl.value, undefined, { renderer: 'canvas' })
  if (ioEl.value && !ioChart) ioChart = echarts.init(ioEl.value, undefined, { renderer: 'canvas' })
}

const lineSeries = (name: string, data: number[], color: string, opts: { area?: boolean; dashed?: boolean } = {}) => ({
  name,
  type: 'line' as const,
  data,
  showSymbol: false,
  smooth: true,
  lineStyle: { width: opts.dashed ? 2 : 2.5, color, type: opts.dashed ? ('dashed' as const) : ('solid' as const) },
  itemStyle: { color },
  ...(opts.area ? { areaStyle: { color: withAlpha(color, 0.14) } } : {}),
})

const baseOption = () => {
  const c = palette.value
  return {
    animation: false,
    backgroundColor: 'transparent',
    textStyle: { fontFamily: 'inherit' },
    tooltip: {
      trigger: 'axis' as const,
      formatter: tooltipFormatter,
      confine: true,
      backgroundColor: c.s,
      borderColor: c.line,
      textStyle: { color: c.ink, fontSize: 12 },
    },
    grid: { top: 12, left: 8, right: 22, bottom: 4, containLabel: true },
    xAxis: {
      type: 'category' as const,
      boundaryGap: false,
      data: history.value.map((item) => item.time),
      axisTick: { show: false },
      axisLabel: { color: c.ink2, fontSize: 11, hideOverlap: true },
      axisLine: { lineStyle: { color: c.line } },
    },
  }
}

const updateChart = () => {
  ensureChart()
  const c = palette.value
  const labels = chartLabels.value
  const base = baseOption()
  cpuChart?.setOption({
    ...base,
    yAxis: {
      type: 'value',
      min: 0,
      max: (v: { max: number }) => Math.max(100, Math.ceil(v.max)),
      splitNumber: 2,
      axisLabel: { formatter: '{value}%', color: c.ink2, fontSize: 11 },
      splitLine: { lineStyle: { color: c.line, type: 'dashed' } },
    },
    series: [
      lineSeries(labels.cpu, history.value.map((item) => item.cpuPercent), c.p, { area: true }),
      lineSeries(labels.memory, history.value.map((item) => item.memoryPercent), c.a2, { dashed: true }),
    ],
  }, { notMerge: true })
  ioChart?.setOption({
    ...base,
    yAxis: {
      type: 'value',
      min: 0,
      splitNumber: 2,
      axisLabel: { formatter: (value: number) => formatRate(value), color: c.ink2, fontSize: 11 },
      splitLine: { lineStyle: { color: c.line, type: 'dashed' } },
    },
    series: [
      lineSeries(labels.networkRx, history.value.map((item) => item.networkRxRate), c.p, { area: true }),
      lineSeries(labels.networkTx, history.value.map((item) => item.networkTxRate), c.a2),
      lineSeries(labels.blockRead, history.value.map((item) => item.blockReadRate), c.a, { dashed: true }),
      lineSeries(labels.blockWrite, history.value.map((item) => item.blockWriteRate), c.ok, { dashed: true }),
    ],
  }, { notMerge: true })
  requestAnimationFrame(() => {
    cpuChart?.resize()
    ioChart?.resize()
  })
}

const stopPolling = () => {
  if (pollTimer) {
    clearTimeout(pollTimer)
    pollTimer = null
  }
  status.value = 'stopped'
}

const schedulePolling = () => {
  if (!visible.value) return
  if (pollTimer) clearTimeout(pollTimer)
  pollTimer = setTimeout(() => {
    void loadStats()
  }, pollIntervalSeconds.value * 1000)
}

const loadStats = async () => {
  if (!props.containerId || loading.value) return
  loading.value = true
  try {
    const { data } = await getDockerContainerStats({ id: props.containerId })
    const stats = data?.stats
    if (stats) {
      const sample = parseStatsSample(stats, currentSample.value)
      history.value = pruneHistory([...history.value, sample], sample.timestamp)
      status.value = 'polling'
      updateChart()
    }
  } catch (error) {
    status.value = 'error'
    showRequestError(error, t('docker.statsDialog.loadFailed'))
  } finally {
    loading.value = false
    schedulePolling()
  }
}

const refresh = () => {
  if (pollTimer) clearTimeout(pollTimer)
  void loadStats()
}

const setPollInterval = (n: number) => {
  pollIntervalSeconds.value = n
  restartPolling()
}

const restartPolling = () => {
  if (!visible.value) return
  schedulePolling()
}

const reset = () => {
  history.value = []
  updateChart()
}

const handleOpened = async () => {
  await nextTick()
  ensureChart()
  reset()
  await loadStats()
}

const handleClosed = () => {
  stopPolling()
  cpuChart?.dispose()
  ioChart?.dispose()
  cpuChart = null
  ioChart = null
}

const resizeChart = () => {
  cpuChart?.resize()
  ioChart?.resize()
}

watch(visible, (open) => {
  if (open) void handleOpened()
  else handleClosed()
})

watch([() => props.containerId], () => {
  if (!visible.value) return
  reset()
  refresh()
})

watch([locale, palette], () => {
  if (!visible.value) return
  updateChart()
}, { deep: true })

onMounted(() => {
  window.addEventListener('resize', resizeChart)
})

onBeforeUnmount(() => {
  stopPolling()
  window.removeEventListener('resize', resizeChart)
  cpuChart?.dispose()
  ioChart?.dispose()
  cpuChart = null
  ioChart = null
})
</script>

<style scoped lang="scss">
.dks { display: flex; flex-direction: column; gap: 14px; }
.dks-head { display: flex; align-items: center; gap: 12px; min-width: 0; }
.dks-head__sticker {
  width: 46px;
  height: 46px;
  flex: none;
  border-radius: 15px;
  transform: rotate(-5deg);
  :deep(svg) { width: 22px; height: 22px; }
}
.dks-head__text { display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0; }
.dks-head__name {
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.dks-head__sub { font-size: 0.75rem; color: var(--el-text-color-secondary); }
.dks-toolbar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.dks-toolbar__label { font-size: 0.8125rem; font-weight: 700; color: var(--el-text-color-secondary); }
.dks-spacer { flex: 1; }
.dks-seg { display: inline-flex; gap: 2px; padding: 3px; border-radius: 999px; background: var(--el-fill-color-light); }
.dks-seg__btn {
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
.dks-seg__btn.is-on { background: var(--el-bg-color); color: var(--el-text-color-primary); box-shadow: var(--app-shadow-sm); }
.dks-tiles { display: grid; grid-template-columns: repeat(4, minmax(0, 1fr)); gap: 10px; }
.dks-tile {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.dks-tile__label { font-size: 0.72rem; font-weight: 700; color: var(--el-text-color-secondary); white-space: nowrap; }
.dks-tile__value {
  font-family: var(--app-font-display);
  font-size: 1.375rem;
  font-weight: 800;
  line-height: 1.1;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.dks-tile__pair {
  display: flex;
  flex-direction: column;
  font-family: var(--app-font-display);
  font-size: 0.9375rem;
  font-weight: 800;
  line-height: 1.35;
  color: var(--el-text-color-primary);
  white-space: nowrap;
}
.dks-tile__sub { font-size: 0.6875rem; font-weight: 600; color: var(--el-text-color-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.dks-up, .dks-down { font-style: normal; font-weight: 800; }
.dks-up { color: var(--m-a2); }
.dks-down { color: var(--m-pInk); }
.dks-bar {
  display: block;
  height: 6px;
  border-radius: 999px;
  background: var(--el-border-color-lighter);
  overflow: hidden;
  i { display: block; height: 100%; border-radius: 999px; transition: width 0.3s; }
}
.dks-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px 16px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-lighter);
}
.dks-card__head { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.dks-card__title { flex: 1; font-size: 0.875rem; font-weight: 800; color: var(--el-text-color-primary); }
.dks-legend {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  i { width: 14px; height: 3px; border-radius: 2px; }
}
.dks-chart-wrap { position: relative; height: 180px; }
.dks-chart { width: 100%; height: 100%; }
.dks-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}

:global(.dk-stats-tooltip) { min-width: 12rem; }
:global(.dk-stats-tooltip__title) { margin-bottom: 0.35rem; font-weight: 700; }
:global(.dk-stats-tooltip__row) { display: flex; align-items: center; justify-content: space-between; gap: 1rem; line-height: 1.6; }

@media (width <= 768px) {
  .dks-tiles { grid-template-columns: 1fr 1fr; }
  .dks-toolbar__label { display: none; }
  .dks-seg { flex: 1; }
  .dks-seg__btn { flex: 1; min-height: 38px; }
  .dks-spacer { display: none; }
  .dks-chart-wrap { height: 160px; }
  .dks-card__title { flex-basis: 100%; }
}
</style>
