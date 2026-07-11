<template>
  <AppPanel title="实时监控" title-icon="HOutline:ChartBarSquareIcon">
    <template #actions>
      <div class="rt-interval" role="group">
        <button
          v-for="opt in [1, 3, 5, 10]"
          :key="opt"
          type="button"
          class="rt-interval__btn"
          :class="{ 'is-active': refreshInterval === opt }"
          @click="onIntervalChange(opt)"
        >
          {{ opt }}s
        </button>
      </div>
    </template>

    <div class="rt-grid">
      <div class="rt-chart">
        <div class="rt-chart__head">
          <span class="rt-chart__title">{{ t('dashboard.home.cpuUsageChart') }}</span>
        </div>
        <div class="rt-chart__box"><VChart :option="cpuChartOption" autoresize /></div>
      </div>
      <div class="rt-chart">
        <div class="rt-chart__head">
          <span class="rt-chart__title">{{ t('dashboard.home.memorySwapUsageChart') }}</span>
        </div>
        <div class="rt-chart__box"><VChart :option="memoryChartOption" autoresize /></div>
      </div>
      <div class="rt-chart">
        <div class="rt-chart__head">
          <span class="rt-chart__title">{{ t('dashboard.home.networkRate') }}</span>
          <select
            class="rt-select"
            :value="selectedInterface"
            @change="onInterfaceChange(($event.target as HTMLSelectElement).value)"
          >
            <option value="">{{ t('dashboard.home.allSummary') }}</option>
            <option v-for="iface in networkOptions" :key="iface.value" :value="iface.value">
              {{ iface.label }}
            </option>
          </select>
        </div>
        <div class="rt-chart__box"><VChart :option="networkChartOption" autoresize /></div>
      </div>
      <div class="rt-chart">
        <div class="rt-chart__head">
          <span class="rt-chart__title">{{ t('dashboard.home.diskIoRate') }}</span>
          <select
            class="rt-select"
            :value="selectedDisk"
            @change="onDiskChange(($event.target as HTMLSelectElement).value)"
          >
            <option value="">{{ t('dashboard.home.allSummary') }}</option>
            <option v-for="d in diskOptions" :key="d.value" :value="d.value">{{ d.label }}</option>
          </select>
        </div>
        <div class="rt-chart__box"><VChart :option="diskChartOption" autoresize /></div>
      </div>
    </div>
  </AppPanel>
</template>

<script setup lang="ts">
import type { RefreshInterval, CpuPoint, MemoryPoint, NetworkPoint, DiskPoint } from '@/stores/dashboard/home'
import type { NetworkInterfaceOverview, DiskPartitionOverview } from '@/types/v1/system'
import { useThemeStore } from '@/stores/theme'
import { useI18n } from 'vue-i18n'

const themeStore = useThemeStore()
const { t } = useI18n()

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
    label: `${iface.name}${iface.isUp ? '' : t('dashboard.home.interfaceDown')}`,
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
      name: t('dashboard.home.total'),
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
      name: t('dashboard.home.core', { index: i }),
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
        html += `<div style="margin-bottom:4px">${total.marker} ${t('dashboard.home.total')}: ${total.value.toFixed(1)}%</div>`
        html += '<table style="width:100%"><tbody>'

        for (const [i, l] of left.entries()) {
          const r = right[i]
          html += '<tr>'
          html += `<td style="white-space:nowrap;padding-right:12px">${l.marker} ${l.seriesName}: ${l.value.toFixed(1)}%</td>`
          if (r) {
            html += `<td style="white-space:nowrap">${r.marker} ${r.seriesName}: ${r.value.toFixed(1)}%</td>`
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
        if (p.seriesName === t('dashboard.home.physicalMemorySeries')) {
          result += `${p.marker} ${t('dashboard.home.physicalMemorySeries')}: ${p.value.toFixed(1)}% (${formatMemoryBytes(point.physicalUsed)} / ${formatMemoryBytes(point.physicalTotal)})<br/>`
        } else {
          result += `${p.marker} Swap: ${p.value.toFixed(1)}% (${formatMemoryBytes(point.swapUsed)} / ${formatMemoryBytes(point.swapTotal)})<br/>`
        }
      }
      return result
    },
  },
  series: [
    { ...makeLineSeries(t('dashboard.home.physicalMemorySeries'), '#10b981'), data: props.memoryHistory.map((p) => p.physical) },
    { ...makeLineSeries('Swap', '#f59e0b'), data: props.memoryHistory.map((p) => p.swap) },
  ],
}))

const networkChartOption = computed(() => ({
  grid: baseChartGrid,
  xAxis: { type: 'category', data: props.networkHistory.map((p) => p.time), axisLabel: { fontSize: 10 } },
  yAxis: { type: 'value', axisLabel: { formatter: (v: number) => formatBytes(v), fontSize: 10 } },
  tooltip: { ...tooltipTheme.value, ...tooltipZ, trigger: 'axis', valueFormatter: (v: number) => formatBytes(v) },
  series: [
    { ...makeLineSeries(t('dashboard.home.download'), '#10b981'), data: props.networkHistory.map((p) => p.download) },
    { ...makeLineSeries(t('dashboard.home.upload'), '#f59e0b'), data: props.networkHistory.map((p) => p.upload) },
  ],
}))

const diskChartOption = computed(() => ({
  grid: baseChartGrid,
  xAxis: { type: 'category', data: props.diskHistory.map((p) => p.time), axisLabel: { fontSize: 10 } },
  yAxis: { type: 'value', axisLabel: { formatter: (v: number) => formatBytes(v), fontSize: 10 } },
  tooltip: { ...tooltipTheme.value, ...tooltipZ, trigger: 'axis', valueFormatter: (v: number) => formatBytes(v) },
  series: [
    { ...makeLineSeries(t('dashboard.home.read'), '#6366f1'), data: props.diskHistory.map((p) => p.read) },
    { ...makeLineSeries(t('dashboard.home.write'), '#ef4444'), data: props.diskHistory.map((p) => p.write) },
  ],
}))
</script>

<style scoped lang="scss">
.rt-interval {
  display: inline-flex;
  gap: 2px;
  padding: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius-sm);
}
.rt-interval__btn {
  padding: 3px 10px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 500;
  cursor: pointer;
  font-variant-numeric: tabular-nums;
  transition: background 0.15s, color 0.15s;
}
.rt-interval__btn:hover {
  color: var(--el-text-color-primary);
}
.rt-interval__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-color-primary);
  box-shadow: var(--app-shadow-card);
}

.rt-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 16px;
}
@media (width >= 1024px) {
  .rt-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
.rt-chart {
  padding: 14px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
}
.rt-chart__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-height: 28px;
  margin-bottom: 8px;
}
.rt-chart__title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-regular);
}
.rt-chart__box {
  width: 100%;
  height: 220px;
  overflow: visible;
}
.rt-select {
  height: 28px;
  max-width: 160px;
  padding: 0 8px;
  border: 1px solid var(--el-border-color-light);
  border-radius: var(--app-radius-sm);
  background: var(--el-bg-color);
  color: var(--el-text-color-regular);
  font-size: 0.75rem;
  cursor: pointer;
}
</style>
