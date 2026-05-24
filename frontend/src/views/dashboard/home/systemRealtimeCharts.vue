<template>
  <BaseCard>
    <template #header-right>
      <div class="chart-header-right">
        <span class="chart-interval-label">刷新间隔:</span>
        <el-radio-group
          :model-value="refreshInterval"
          size="small"
          @change="onIntervalChange"
        >
          <el-radio-button :value="1">1s</el-radio-button>
          <el-radio-button :value="3">3s</el-radio-button>
          <el-radio-button :value="5">5s</el-radio-button>
          <el-radio-button :value="10">10s</el-radio-button>
        </el-radio-group>
      </div>
    </template>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5">
      <div class="chart-panel">
        <div class="chart-title">CPU 使用率 (%)</div>
        <div class="chart-box"><VChart :option="cpuChartOption" autoresize /></div>
      </div>
      <div class="chart-panel">
        <div class="chart-title">内存 / Swap 使用率 (%)</div>
        <div class="chart-box"><VChart :option="memoryChartOption" autoresize /></div>
      </div>
      <div class="chart-panel">
        <div class="flex items-center justify-between gap-2 mb-2">
          <div class="chart-title" style="margin-bottom:0">网络 速率</div>
          <el-select
            :model-value="selectedInterface"
            size="small"
            class="selector"
            @change="onInterfaceChange"
          >
            <el-option label="全部（汇总）" value="" />
            <el-option
              v-for="iface in networkOptions"
              :key="iface.value"
              :label="iface.label"
              :value="iface.value"
            />
          </el-select>
        </div>
        <div class="chart-box"><VChart :option="networkChartOption" autoresize /></div>
      </div>
      <div class="chart-panel">
        <div class="flex items-center justify-between gap-2 mb-2">
          <div class="chart-title" style="margin-bottom:0">磁盘 IO 速率</div>
          <el-select
            :model-value="selectedDisk"
            size="small"
            class="selector"
            @change="onDiskChange"
          >
            <el-option label="全部（汇总）" value="" />
            <el-option
              v-for="d in diskOptions"
              :key="d.value"
              :label="d.label"
              :value="d.value"
            />
          </el-select>
        </div>
        <div class="chart-box"><VChart :option="diskChartOption" autoresize /></div>
      </div>
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import type { RefreshInterval, CpuPoint, MemoryPoint, NetworkPoint, DiskPoint } from '@/stores/dashboard/home'
import type { NetworkInterfaceOverview, DiskPartitionOverview } from '@/types/v1/system'
import { useThemeStore } from '@/stores/theme'

const themeStore = useThemeStore()

const tooltipTheme = computed(() => ({
  backgroundColor: themeStore.isDarkTheme ? '#1d1e1f' : '#fff',
  borderColor: themeStore.isDarkTheme ? '#363637' : '#e4e7ed',
  textStyle: { color: themeStore.isDarkTheme ? '#cfd3dc' : '#303133' },
}))

const props = defineProps<{
  cpuHistory: CpuPoint[]
  memoryHistory: MemoryPoint[]
  networkHistory: NetworkPoint[]
  diskHistory: DiskPoint[]
  refreshInterval: RefreshInterval
  selectedInterface: string
  selectedDisk: string
  overview: {
    networkInterfaces: NetworkInterfaceOverview[]
    diskPartitions: DiskPartitionOverview[]
  } | null
}>()

const emit = defineEmits<{
  intervalChange: [value: RefreshInterval]
  interfaceChange: [value: string]
  diskChange: [value: string]
}>()

const isRefreshInterval = (value: unknown): value is RefreshInterval =>
  value === 1 || value === 3 || value === 5 || value === 10

const onIntervalChange = (value: string | number | boolean | undefined) => {
  if (isRefreshInterval(value)) {
    emit('intervalChange', value)
  }
}
const onInterfaceChange = (value: string) => emit('interfaceChange', value)
const onDiskChange = (value: string) => emit('diskChange', value)

const networkOptions = computed(() =>
  (props.overview?.networkInterfaces || []).map((iface) => ({
    label: `${iface.name}${iface.isUp ? '' : ' (down)'}`,
    value: iface.name,
  })),
)

const diskOptions = computed(() =>
  (props.overview?.diskPartitions || []).map((part) => ({
    label: `${part.device} — ${part.mountpoint}`,
    value: part.device,
  })),
)

const formatBytes = (bytes: number | string) => {
  const num = Number(bytes)
  if (!num || num <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const base = 1024
  const i = Math.min(Math.floor(Math.log(num) / Math.log(base)), units.length - 1)
  return (num / base ** i).toFixed(i === 0 ? 0 : 1) + ' ' + units[i] + '/s'
}

const baseChartGrid = { left: 45, right: 15, top: 10, bottom: 25 }

const makeLineSeries = (name: string, color: string) => ({
  name,
  type: 'line',
  smooth: true,
  symbol: 'none',
  lineStyle: { color, width: 2 },
  areaStyle: {
    color: {
      type: 'linear',
      x: 0, y: 0, x2: 0, y2: 1,
      colorStops: [
        { offset: 0, color: color + '30' },
        { offset: 1, color: 'transparent' },
      ],
    },
  },
})

const coreColors = [
  '#10b981', '#f59e0b', '#ef4444', '#8b5cf6',
  '#06b6d4', '#ec4899', '#84cc16', '#f97316', '#14b8a6',
  '#3b82f6', '#e11d48', '#a855f7', '#0ea5e9', '#eab308',
  '#22c55e', '#fdba74',
]

const tooltipZ = { extraCssText: 'z-index: 99999 !important; pointer-events: auto' }

const cpuChartOption = computed(() => {
  const coreCount = props.cpuHistory[0]?.cores.length ?? 0

  const rebuiltSeries: Record<string, unknown>[] = [
    {
      name: '总计',
      type: 'line',
      smooth: true,
      symbol: 'none',
      lineStyle: { color: '#6366f1', width: 2.5 },
      data: props.cpuHistory.map((p) => p.total),
      areaStyle: {
        color: {
          type: 'linear',
          x: 0, y: 0, x2: 0, y2: 1,
          colorStops: [
            { offset: 0, color: '#6366f140' },
            { offset: 1, color: 'transparent' },
          ],
        },
      },
    },
  ]

  for (let i = 0; i < coreCount; i++) {
    const color = coreColors[i % coreColors.length]
    rebuiltSeries.push({
      name: `核心 ${i}`,
      type: 'line',
      smooth: true,
      symbol: 'none',
      lineStyle: { color, width: 1, opacity: 0.5 },
      data: props.cpuHistory.map((p) => p.cores[i] ?? 0),
    })
  }

  return {
    grid: baseChartGrid,
    xAxis: { type: 'category', data: props.cpuHistory.map((p) => p.time), axisLabel: { fontSize: 10 } },
    yAxis: { type: 'value', max: 100, axisLabel: { formatter: '{value}%', fontSize: 10 } },
    tooltip: {
      ...tooltipTheme.value,
      extraCssText: 'z-index: 99999 !important; pointer-events: auto; max-height: 420px; overflow-y: auto',
      confine: true,
      enterable: true,
      trigger: 'axis',
      formatter: (params: { seriesName: string; value: number; marker: string; axisValue: string }[]) => {
        const [total, ...cores] = params
        if (!total) return ''
        const mid = Math.ceil(cores.length / 2)
        const left = cores.slice(0, mid)
        const right = cores.slice(mid)

        let html = `<div style="font-weight:600;margin-bottom:4px">${total.axisValue}</div>`
        html += `<div style="margin-bottom:4px">${total.marker} 总计: ${total.value.toFixed(1)}%</div>`
        html += '<table style="width:100%"><tbody>'

        for (const [i, l] of left.entries()) {
          const r = right[i]
          html += '<tr>'
          html += `<td style="white-space:nowrap;padding-right:12px">${l.marker} 核心 ${l.seriesName.replace('核心 ', '')}: ${l.value.toFixed(1)}%</td>`
          if (r) {
            html += `<td style="white-space:nowrap">${r.marker} 核心 ${r.seriesName.replace('核心 ', '')}: ${r.value.toFixed(1)}%</td>`
          }
          html += '</tr>'
        }

        html += '</tbody></table>'
        return html
      },
    },
    series: rebuiltSeries,
  }
})

const formatMemoryBytes = (bytes: number | string) => {
  const num = Number(bytes)
  if (!num || num <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const base = 1024
  const i = Math.min(Math.floor(Math.log(num) / Math.log(base)), units.length - 1)
  return (num / base ** i).toFixed(i === 0 ? 0 : 1) + ' ' + units[i]
}

const memoryChartOption = computed(() => ({
  grid: baseChartGrid,
  xAxis: { type: 'category', data: props.memoryHistory.map((p) => p.time), axisLabel: { fontSize: 10 } },
  yAxis: { type: 'value', max: 100, axisLabel: { formatter: '{value}%', fontSize: 10 } },
  tooltip: {
    ...tooltipTheme.value,
    ...tooltipZ,
    trigger: 'axis',
    formatter: (params: { axisValue: string; seriesName: string; value: number; dataIndex: number; marker: string }[]) => {
      const first = params[0]
      if (!first) return ''

      let result = first.axisValue + '<br/>'
      for (const p of params) {
        const idx = p.dataIndex
        const point = props.memoryHistory[idx]
        if (!point) continue
        if (p.seriesName === '物理内存') {
          result += `${p.marker} 物理内存: ${p.value.toFixed(1)}% (${formatMemoryBytes(point.physicalUsed)} / ${formatMemoryBytes(point.physicalTotal)})<br/>`
        } else {
          result += `${p.marker} Swap: ${p.value.toFixed(1)}% (${formatMemoryBytes(point.swapUsed)} / ${formatMemoryBytes(point.swapTotal)})<br/>`
        }
      }
      return result
    },
  },
  series: [
    { ...makeLineSeries('物理内存', '#10b981'), data: props.memoryHistory.map((p) => p.physical) },
    { ...makeLineSeries('Swap', '#f59e0b'), data: props.memoryHistory.map((p) => p.swap) },
  ],
}))

const networkChartOption = computed(() => ({
  grid: baseChartGrid,
  xAxis: { type: 'category', data: props.networkHistory.map((p) => p.time), axisLabel: { fontSize: 10 } },
  yAxis: { type: 'value', axisLabel: { formatter: (v: number) => formatBytes(v), fontSize: 10 } },
  tooltip: { ...tooltipTheme.value, ...tooltipZ, trigger: 'axis', valueFormatter: (v: number) => formatBytes(v) },
  series: [
    { ...makeLineSeries('下载', '#10b981'), data: props.networkHistory.map((p) => p.download) },
    { ...makeLineSeries('上传', '#f59e0b'), data: props.networkHistory.map((p) => p.upload) },
  ],
}))

const diskChartOption = computed(() => ({
  grid: baseChartGrid,
  xAxis: { type: 'category', data: props.diskHistory.map((p) => p.time), axisLabel: { fontSize: 10 } },
  yAxis: { type: 'value', axisLabel: { formatter: (v: number) => formatBytes(v), fontSize: 10 } },
  tooltip: { ...tooltipTheme.value, ...tooltipZ, trigger: 'axis', valueFormatter: (v: number) => formatBytes(v) },
  series: [
    { ...makeLineSeries('读取', '#6366f1'), data: props.diskHistory.map((p) => p.read) },
    { ...makeLineSeries('写入', '#ef4444'), data: props.diskHistory.map((p) => p.write) },
  ],
}))
</script>

<style scoped lang="scss">
.chart-header-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.chart-interval-label {
  font-size: 0.7rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.chart-panel {
  background: var(--el-bg-color-page);
  border-radius: 1rem;
  padding: 12px;

  @media (width >= 640px) {
    padding: 16px;
  }

  .chart-title {
    font-size: 12px;
    font-weight: 600;
    color: var(--el-text-color-secondary);
    margin-bottom: 6px;
    flex-shrink: 0;

    @media (width >= 640px) {
      font-size: 13px;
      margin-bottom: 8px;
    }
  }

  .chart-box {
    width: 100%;
    height: 200px;
    overflow: visible;

    @media (width >= 640px) {
      height: 240px;
    }
  }

  .selector {
    width: 130px;
    flex-shrink: 0;

    @media (width >= 640px) {
      width: 160px;
    }
  }
}
</style>
