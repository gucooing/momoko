<!-- 容器统计（重写）：FormDialog 令牌壳 + 摘要卡 + echarts 折线。
     轮询 getDockerContainerStats；保留 CPU/内存/网络/块 IO 计算与 5 分钟窗口。 -->
<template>
  <FormDialog v-model="visible" :title="dialogTitle" :width="900" :show-footer="false" @close="handleClosed">
    <div class="dk-stats">
      <div class="dk-stats__bar">
        <UButton color="neutral" variant="soft" icon="i-lucide-refresh-cw" size="sm" :loading="loading" @click="refresh">
          {{ t('docker.common.refresh') }}
        </UButton>
        <div class="dk-stats__interval">
          <span>{{ t('docker.statsDialog.interval') }}</span>
          <input
            v-model.number="pollIntervalSeconds"
            type="number"
            min="1"
            max="60"
            step="1"
            class="app-input dk-stats__num"
            @change="restartPolling"
          />
          <span>{{ t('docker.common.seconds') }}</span>
        </div>
        <span class="dk-stats__status">
          <StatusPill :variant="status === 'polling' ? 'success' : status === 'error' ? 'error' : 'neutral'" :label="statusLabel" />
        </span>
      </div>

      <div class="dk-stats__summary">
        <div v-for="item in summaryItems" :key="item.label" class="dk-stats__item">
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
        </div>
      </div>

      <div ref="chartEl" class="dk-stats__chart" />
      <EmptyState
        v-if="!history.length && !loading"
        icon="HOutline:ChartBarIcon"
        :title="t('docker.statsDialog.noData')"
      />
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
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const chartEl = ref<HTMLDivElement | null>(null)
const pollIntervalSeconds = ref(3)
const loading = ref(false)
const status = ref<PollStatus>('stopped')
const history = ref<StatsSample[]>([])

let chart: echarts.ECharts | null = null
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

const summaryItems = computed(() => {
  const sample = currentSample.value
  return [
    { label: t('docker.statsDialog.cpuUsage'), value: sample ? toPercent(sample.cpuPercent) : '-' },
    { label: t('docker.statsDialog.memoryUsagePercent'), value: sample ? toPercent(sample.memoryPercent) : '-' },
    { label: t('docker.statsDialog.memoryUsage'), value: sample ? formatBytes(sample.memoryUsage) : '-' },
    { label: t('docker.statsDialog.memoryLimit'), value: sample ? formatBytes(sample.memoryLimit) : '-' },
    { label: t('docker.statsDialog.networkRx'), value: sample ? formatRate(sample.networkRxRate) : '-' },
    { label: t('docker.statsDialog.networkTx'), value: sample ? formatRate(sample.networkTxRate) : '-' },
    { label: t('docker.statsDialog.blockRead'), value: sample ? formatRate(sample.blockReadRate) : '-' },
    { label: t('docker.statsDialog.blockWrite'), value: sample ? formatRate(sample.blockWriteRate) : '-' },
  ]
})

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
  if (!chartEl.value) return
  if (!chart) chart = echarts.init(chartEl.value)
}

const lineSeries = (name: string, data: number[], yAxisIndex: number) => ({
  name,
  type: 'line' as const,
  yAxisIndex,
  data,
  showSymbol: false,
  smooth: true,
  lineStyle: { width: 2 },
})

const updateChart = () => {
  ensureChart()
  if (!chart) return
  const labels = chartLabels.value
  const isDark = document.documentElement.classList.contains('dark')
    || document.documentElement.getAttribute('data-theme') === 'dark'
  const split = isDark ? 'rgba(148,163,184,0.15)' : '#e5e7eb'

  const option: echarts.EChartsOption = {
    animation: false,
    color: ['#2f7cff', '#13b981', '#f59e0b', '#ef4444', '#8b5cf6', '#06b6d4'],
    tooltip: {
      trigger: 'axis',
      formatter: tooltipFormatter,
      confine: true,
    },
    legend: {
      type: 'scroll',
      top: 8,
      left: 12,
      right: 12,
      itemWidth: 10,
      itemHeight: 10,
      itemGap: 12,
      textStyle: { color: 'var(--el-text-color-regular)', fontSize: 11 },
      data: [labels.cpu, labels.memory, labels.networkRx, labels.networkTx, labels.blockRead, labels.blockWrite],
    },
    grid: { top: 58, left: 52, right: 64, bottom: 34 },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: history.value.map((item) => item.time),
      axisTick: { alignWithLabel: true },
      axisLabel: { color: 'var(--el-text-color-secondary)', fontSize: 11 },
      axisLine: { lineStyle: { color: split } },
    },
    yAxis: [
      {
        type: 'value',
        min: 0,
        max: 100,
        axisLabel: { formatter: '{value}%', color: 'var(--el-text-color-secondary)', fontSize: 11 },
        splitLine: { lineStyle: { color: split } },
      },
      {
        type: 'value',
        axisLabel: {
          formatter: (value: number) => formatRate(value),
          color: 'var(--el-text-color-secondary)',
          fontSize: 11,
        },
        splitLine: { show: false },
      },
    ],
    series: [
      lineSeries(labels.cpu, history.value.map((item) => item.cpuPercent), 0),
      lineSeries(labels.memory, history.value.map((item) => item.memoryPercent), 0),
      lineSeries(labels.networkRx, history.value.map((item) => item.networkRxRate), 1),
      lineSeries(labels.networkTx, history.value.map((item) => item.networkTxRate), 1),
      lineSeries(labels.blockRead, history.value.map((item) => item.blockReadRate), 1),
      lineSeries(labels.blockWrite, history.value.map((item) => item.blockWriteRate), 1),
    ],
  }
  chart.setOption(option, true)
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
}

const resizeChart = () => {
  chart?.resize()
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

watch(locale, () => {
  if (!visible.value) return
  updateChart()
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
.dk-stats {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-height: 26rem;
}
.dk-stats__bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 10px;
}
.dk-stats__interval {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
}
.dk-stats__num {
  width: 72px;
}
.dk-stats__status {
  margin-left: auto;
}
.dk-stats__summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  overflow: hidden;
}
.dk-stats__item {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  padding: 8px 10px;
  border-right: 1px solid var(--el-border-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);
  span {
    color: var(--el-text-color-secondary);
    font-size: 0.72rem;
  }
  strong {
    color: var(--el-text-color-primary);
    font-size: 0.875rem;
    font-weight: 600;
    font-variant-numeric: tabular-nums;
  }
}
.dk-stats__item:nth-child(4n) { border-right: 0; }
.dk-stats__item:nth-last-child(-n + 4) { border-bottom: 0; }
.dk-stats__chart {
  width: 100%;
  height: 22rem;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
}

:global(.dk-stats-tooltip) { min-width: 12rem; }
:global(.dk-stats-tooltip__title) {
  margin-bottom: 0.35rem;
  color: var(--el-text-color-regular);
  font-weight: 600;
}
:global(.dk-stats-tooltip__row) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  line-height: 1.6;
}
:global(.dk-stats-tooltip__row span) { color: var(--el-text-color-secondary); }
:global(.dk-stats-tooltip__row strong) { color: var(--el-text-color-primary); font-weight: 600; }

@media (width <= 768px) {
  .dk-stats__status { margin-left: 0; width: 100%; }
  .dk-stats__summary { grid-template-columns: repeat(2, minmax(0, 1fr)); }
  .dk-stats__item:nth-child(2n) { border-right: 0; }
  .dk-stats__item:nth-last-child(-n + 4) { border-bottom: 1px solid var(--el-border-color-lighter); }
  .dk-stats__item:nth-last-child(-n + 2) { border-bottom: 0; }
}
</style>
