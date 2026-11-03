<template>
  <AppPanel :title="t('dashboard.home.realtimeMonitor')" title-icon="HOutline:ChartBarSquareIcon">
    <template #actions>
      <div class="rt-seg" role="group" :aria-label="t('dashboard.home.refreshEvery')">
        <button
          v-for="opt in [1, 3, 5, 10]"
          :key="opt"
          type="button"
          :class="{ 'is-on': refreshInterval === opt }"
          @click="onIntervalChange(opt)"
        >
          {{ opt }}s
        </button>
      </div>
    </template>

    <div class="rt-bar">
      <div class="rt-seg rt-seg--metrics" role="tablist">
        <button
          v-for="m in metricTabs"
          :key="m.key"
          type="button"
          role="tab"
          :aria-selected="metric === m.key"
          :class="{ 'is-on': metric === m.key }"
          @click="metric = m.key"
        >
          {{ t(m.label) }}
        </button>
      </div>
      <select
        v-if="metric === 'network'"
        class="app-select rt-select"
        :value="selectedInterface"
        :aria-label="t('dashboard.home.networkRate')"
        @change="onInterfaceChange(($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ t('dashboard.home.allSummary') }}</option>
        <option v-for="iface in networkOptions" :key="iface.value" :value="iface.value">
          {{ iface.label }}
        </option>
      </select>
      <select
        v-if="metric === 'disk'"
        class="app-select rt-select"
        :value="selectedDisk"
        :aria-label="t('dashboard.home.diskIoRate')"
        @change="onDiskChange(($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ t('dashboard.home.allSummary') }}</option>
        <option v-for="d in diskOptions" :key="d.value" :value="d.value">{{ d.label }}</option>
      </select>
    </div>

    <div class="rt-stats">
      <div v-for="st in currentStats" :key="st.label" class="rt-stat">
        <span class="rt-stat__label">
          <i :style="{ background: st.color }" />
          {{ st.label }}
        </span>
        <b class="m-num">{{ st.value }}</b>
      </div>
    </div>

    <div class="rt-chart">
      <VChart v-if="metric === 'cpu'" :option="cpuChartOption" autoresize />
      <VChart v-else-if="metric === 'memory'" :option="memoryChartOption" autoresize />
      <VChart v-else-if="metric === 'network'" :option="networkChartOption" autoresize />
      <VChart v-else :option="diskChartOption" autoresize />
    </div>
  </AppPanel>
</template>

<script setup lang="ts">
import type { RefreshInterval, CpuPoint, MemoryPoint, NetworkPoint, DiskPoint } from '@/stores/dashboard/home'
import type { NetworkInterfaceOverview, DiskPartitionOverview } from '@/types/v1/system'
import { usePaletteColors, withAlpha } from '@/composables/usePaletteColors'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()
const pc = usePaletteColors()

type Metric = 'cpu' | 'memory' | 'network' | 'disk'
const metric = ref<Metric>('cpu')
const metricTabs: { key: Metric; label: string }[] = [
  { key: 'cpu', label: 'dashboard.home.tabCpu' },
  { key: 'memory', label: 'dashboard.home.tabMemory' },
  { key: 'network', label: 'dashboard.home.tabNetwork' },
  { key: 'disk', label: 'dashboard.home.tabDisk' },
]

const tooltipTheme = computed(() => ({
  backgroundColor: pc.value.s,
  borderColor: pc.value.line,
  borderRadius: 14,
  padding: [10, 12],
  textStyle: { color: pc.value.ink },
}))

const axisStyle = computed(() => ({
  axisLabel: { fontSize: 11, color: pc.value.ink2 },
  axisLine: { lineStyle: { color: pc.value.line } },
  axisTick: { show: false },
  splitLine: { lineStyle: { color: pc.value.line, type: 'dashed' as const } },
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

const baseChartGrid = { left: 52, right: 12, top: 14, bottom: 26 }

const makeLineSeries = (name: string, color: string) => ({
  name,
  type: 'line',
  smooth: true,
  symbol: 'none',
  itemStyle: { color },
  lineStyle: { color, width: 3 },
  areaStyle: { color: withAlpha(color, 0.14) },
})


const tooltipZ = { extraCssText: 'z-index: 99999 !important; pointer-events: auto' }

const cpuChartOption = computed(() => {
  const coreCount = props.cpuHistory[0]?.cores.length ?? 0

  const rebuiltSeries: Record<string, unknown>[] = [
    {
      name: t('dashboard.home.total'),
      type: 'line',
      smooth: true,
      symbol: 'none',
      itemStyle: { color: pc.value.p },
      lineStyle: { color: pc.value.p, width: 3 },
      data: props.cpuHistory.map((p) => p.total),
      areaStyle: { color: withAlpha(pc.value.p, 0.14) },
      z: 3,
    },
  ]
  // 各核心用强调色系的细线，退居背景
  const coreColors = [pc.value.a2, pc.value.a, pc.value.ok, pc.value.ink2]

  for (let i = 0; i < coreCount; i++) {
    const color = coreColors[i % coreColors.length]
    rebuiltSeries.push({
      name: t('dashboard.home.core', { index: i }),
      type: 'line',
      smooth: true,
      symbol: 'none',
      itemStyle: { color },
      lineStyle: { color, width: 1, opacity: 0.35 },
      data: props.cpuHistory.map((p) => p.cores[i] ?? 0),
    })
  }

  return {
    grid: baseChartGrid,
    xAxis: { type: 'category', data: props.cpuHistory.map((p) => p.time), ...axisStyle.value, splitLine: { show: false } },
    yAxis: { type: 'value', max: 100, ...axisStyle.value, axisLabel: { ...axisStyle.value.axisLabel, formatter: '{value}%' } },
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
  xAxis: { type: 'category', data: props.memoryHistory.map((p) => p.time), ...axisStyle.value, splitLine: { show: false } },
  yAxis: { type: 'value', max: 100, ...axisStyle.value, axisLabel: { ...axisStyle.value.axisLabel, formatter: '{value}%' } },
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
    { ...makeLineSeries(t('dashboard.home.physicalMemorySeries'), pc.value.p), data: props.memoryHistory.map((p) => p.physical) },
    { ...makeLineSeries('Swap', pc.value.a2), data: props.memoryHistory.map((p) => p.swap) },
  ],
}))

const networkChartOption = computed(() => ({
  grid: baseChartGrid,
  xAxis: { type: 'category', data: props.networkHistory.map((p) => p.time), ...axisStyle.value, splitLine: { show: false } },
  yAxis: { type: 'value', ...axisStyle.value, axisLabel: { ...axisStyle.value.axisLabel, formatter: (v: number) => formatBytes(v) } },
  tooltip: { ...tooltipTheme.value, ...tooltipZ, trigger: 'axis', valueFormatter: (v: number) => formatBytes(v) },
  series: [
    { ...makeLineSeries(t('dashboard.home.download'), pc.value.p), data: props.networkHistory.map((p) => p.download) },
    { ...makeLineSeries(t('dashboard.home.upload'), pc.value.a2), data: props.networkHistory.map((p) => p.upload) },
  ],
}))

const diskChartOption = computed(() => ({
  grid: baseChartGrid,
  xAxis: { type: 'category', data: props.diskHistory.map((p) => p.time), ...axisStyle.value, splitLine: { show: false } },
  yAxis: { type: 'value', ...axisStyle.value, axisLabel: { ...axisStyle.value.axisLabel, formatter: (v: number) => formatBytes(v) } },
  tooltip: { ...tooltipTheme.value, ...tooltipZ, trigger: 'axis', valueFormatter: (v: number) => formatBytes(v) },
  series: [
    { ...makeLineSeries(t('dashboard.home.read'), pc.value.p), data: props.diskHistory.map((p) => p.read) },
    { ...makeLineSeries(t('dashboard.home.write'), pc.value.a2), data: props.diskHistory.map((p) => p.write) },
  ],
}))

const pct = (v?: number) => `${(v ?? 0).toFixed(1)}%`
const last = <T,>(arr: T[]) => arr[arr.length - 1]
const peakOf = (vals: number[]) => (vals.length ? Math.max(...vals) : 0)

/** 当前指标的「当前 / 峰值」摘要，色点与曲线一致 */
const currentStats = computed(() => {
  const c = pc.value
  if (metric.value === 'cpu') {
    const vals = props.cpuHistory.map((p) => p.total)
    return [
      { label: t('dashboard.home.current'), value: pct(last(vals)), color: c.p },
      { label: t('dashboard.home.peak'), value: pct(peakOf(vals)), color: c.a },
    ]
  }
  if (metric.value === 'memory') {
    const pt = last(props.memoryHistory)
    return [
      { label: t('dashboard.home.physicalMemorySeries'), value: pct(pt?.physical), color: c.p },
      { label: 'Swap', value: pct(pt?.swap), color: c.a2 },
    ]
  }
  if (metric.value === 'network') {
    const pt = last(props.networkHistory)
    return [
      { label: t('dashboard.home.download'), value: formatBytes(pt?.download ?? 0), color: c.p },
      { label: t('dashboard.home.upload'), value: formatBytes(pt?.upload ?? 0), color: c.a2 },
    ]
  }
  const pt = last(props.diskHistory)
  return [
    { label: t('dashboard.home.read'), value: formatBytes(pt?.read ?? 0), color: c.p },
    { label: t('dashboard.home.write'), value: formatBytes(pt?.write ?? 0), color: c.a2 },
  ]
})
</script>

<style scoped lang="scss">
.rt-seg {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.rt-seg button {
  min-height: 32px;
  padding: 0 12px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.rt-seg button:hover {
  color: var(--el-text-color-primary);
}
.rt-seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.rt-bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.rt-seg--metrics button {
  min-height: 36px;
  padding: 0 16px;
  font-size: 0.875rem;
}
@media (width < 640px) {
  .rt-seg--metrics {
    display: flex;
    width: 100%;
  }
  .rt-seg--metrics button {
    flex: 1;
    padding: 0 6px;
  }
}
.rt-select {
  width: auto;
  max-width: 240px;
}
.rt-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 28px;
  margin: 18px 0 6px;
}
.rt-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.rt-stat__label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.rt-stat__label i {
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.rt-stat b {
  font-size: 1.625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.rt-chart {
  width: 100%;
  height: 260px;
}
@media (width < 640px) {
  .rt-chart {
    height: 200px;
  }
}
</style>
