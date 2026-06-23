<template>
  <BaseDialog
    v-model="visible"
    :title="dialogTitle"
    width="860"
    :show-footer="false"
    @opened="handleOpened"
    @close="handleClosed"
  >
    <div class="docker-stats-dialog">
      <div class="docker-stats-toolbar">
        <el-button
          size="small"
          :icon="menuStore.iconComponents.Refresh"
          :loading="loading"
          @click="refresh"
        >
          刷新
        </el-button>
        <div class="docker-stats-interval">
          <span>轮询间隔</span>
          <el-input-number
            v-model="pollIntervalSeconds"
            size="small"
            :min="1"
            :max="60"
            :step="1"
            controls-position="right"
            @change="restartPolling"
          />
          <span>秒</span>
        </div>
        <span class="docker-stats-status" :class="statusClass">
          <span class="docker-stats-status__dot" />
          {{ statusLabel }}
        </span>
      </div>

      <div class="docker-stats-summary">
        <div v-for="item in summaryItems" :key="item.label" class="docker-stats-summary__item">
          <span>{{ item.label }}</span>
          <strong>{{ item.value }}</strong>
        </div>
      </div>

      <div ref="chartEl" class="docker-stats-chart" />
      <el-empty v-if="!history.length && !loading" description="暂无统计数据" />
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import * as echarts from 'echarts'

import { getDockerContainerStats } from '@/api/docker'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
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

const menuStore = useMenuStore()
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
  return name ? `容器统计 - ${name}` : '容器统计'
})

const statusLabel = computed(() => {
  if (status.value === 'polling') return '轮询中'
  if (status.value === 'error') return '请求失败'
  return '已停止'
})

const statusClass = computed(() => ({
  'docker-stats-status--connected': status.value === 'polling',
  'docker-stats-status--error': status.value === 'error',
}))

const currentSample = computed(() => history.value[history.value.length - 1])

const summaryItems = computed(() => {
  const sample = currentSample.value
  return [
    { label: 'CPU 使用率', value: sample ? toPercent(sample.cpuPercent) : '-' },
    { label: '内存使用率', value: sample ? toPercent(sample.memoryPercent) : '-' },
    { label: '内存使用', value: sample ? formatBytes(sample.memoryUsage) : '-' },
    { label: '内存限制', value: sample ? formatBytes(sample.memoryLimit) : '-' },
    { label: '网络接收', value: sample ? formatRate(sample.networkRxRate) : '-' },
    { label: '网络发送', value: sample ? formatRate(sample.networkTxRate) : '-' },
    { label: '块读取', value: sample ? formatRate(sample.blockReadRate) : '-' },
    { label: '块写入', value: sample ? formatRate(sample.blockWriteRate) : '-' },
  ]
})

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
  if (seriesName === 'CPU' || seriesName === '内存') return toPercent(value)
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
    return `<div class="docker-stats-tooltip__row"><span>${marker}${name}</span><strong>${value}</strong></div>`
  }).join('')
  return `<div class="docker-stats-tooltip"><div class="docker-stats-tooltip__title">${title}</div>${rows}</div>`
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
    .filter(item => now - item.timestamp <= HISTORY_WINDOW_MS)
    .slice(-MAX_HISTORY_POINTS)
}

const ensureChart = () => {
  if (!chartEl.value) return
  if (!chart) chart = echarts.init(chartEl.value)
}

const updateChart = () => {
  ensureChart()
  if (!chart) return

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
      data: ['CPU', '内存', '网络接收', '网络发送', '块读取', '块写入'],
    },
    grid: { top: 58, left: 52, right: 64, bottom: 34 },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: history.value.map(item => item.time),
      axisTick: { alignWithLabel: true },
    },
    yAxis: [
      {
        type: 'value',
        min: 0,
        max: 100,
        axisLabel: { formatter: '{value}%' },
        splitLine: { lineStyle: { color: '#e5e7eb' } },
      },
      {
        type: 'value',
        axisLabel: {
          formatter: (value: number) => formatRate(value),
        },
        splitLine: { show: false },
      },
    ],
    series: [
      lineSeries('CPU', history.value.map(item => item.cpuPercent), 0),
      lineSeries('内存', history.value.map(item => item.memoryPercent), 0),
      lineSeries('网络接收', history.value.map(item => item.networkRxRate), 1),
      lineSeries('网络发送', history.value.map(item => item.networkTxRate), 1),
      lineSeries('块读取', history.value.map(item => item.blockReadRate), 1),
      lineSeries('块写入', history.value.map(item => item.blockWriteRate), 1),
    ],
  }
  chart.setOption(option, true)
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
    showRequestError(error, '获取容器统计失败')
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

watch([() => props.containerId], () => {
  if (!visible.value) return
  reset()
  refresh()
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
.docker-stats-dialog {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  min-height: 26rem;
}

.docker-stats-toolbar {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}

.docker-stats-interval {
  display: flex;
  align-items: center;
  gap: 0.4rem;
  color: var(--el-text-color-regular);
  font-size: 0.82rem;

  :deep(.el-input-number) {
    width: 5.75rem;
  }
}

.docker-stats-status {
  display: inline-flex;
  align-items: center;
  gap: 0.35rem;
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 0.78rem;
}

.docker-stats-status__dot {
  width: 0.45rem;
  height: 0.45rem;
  border-radius: 50%;
  background: var(--el-text-color-placeholder);
}

.docker-stats-status--connected .docker-stats-status__dot {
  background: var(--el-color-success);
}

.docker-stats-status--error .docker-stats-status__dot {
  background: var(--el-color-danger);
}

.docker-stats-summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  border: 1px solid var(--el-border-color-lighter);
}

.docker-stats-summary__item {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
  min-width: 0;
  padding: 0.65rem 0.75rem;
  border-right: 1px solid var(--el-border-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);

  span {
    color: var(--el-text-color-secondary);
    font-size: 0.78rem;
  }

  strong {
    color: var(--el-text-color-primary);
    font-size: 0.95rem;
    font-weight: 600;
  }
}

.docker-stats-summary__item:nth-child(4n) {
  border-right: 0;
}

.docker-stats-summary__item:nth-last-child(-n + 4) {
  border-bottom: 0;
}

.docker-stats-chart {
  width: 100%;
  height: 22rem;
  border: 1px solid var(--el-border-color-lighter);
}

:global(.docker-stats-tooltip) {
  min-width: 12rem;
}

:global(.docker-stats-tooltip__title) {
  margin-bottom: 0.35rem;
  color: var(--el-text-color-regular);
  font-weight: 600;
}

:global(.docker-stats-tooltip__row) {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  line-height: 1.6;
}

:global(.docker-stats-tooltip__row span) {
  color: var(--el-text-color-secondary);
}

:global(.docker-stats-tooltip__row strong) {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

@media (max-width: 768px) {
  .docker-stats-toolbar {
    align-items: flex-start;
    flex-direction: column;
  }

  .docker-stats-status {
    margin-left: 0;
  }

  .docker-stats-summary {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .docker-stats-summary__item:nth-child(2n) {
    border-right: 0;
  }

  .docker-stats-summary__item:nth-last-child(-n + 4) {
    border-bottom: 1px solid var(--el-border-color-lighter);
  }

  .docker-stats-summary__item:nth-last-child(-n + 2) {
    border-bottom: 0;
  }
}
</style>
