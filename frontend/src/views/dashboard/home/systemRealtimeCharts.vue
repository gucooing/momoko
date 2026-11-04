<!-- 实时监控（DashboardPC / Mobile）：指标分段切换 + 刷新间隔，当前 / 平均 / 峰值，单图曲线。
     手机端刷新间隔收成「每 N 秒」，点按切换。 -->
<template>
  <section class="rt" aria-labelledby="rt-h">
    <div class="rt__head">
      <h2 id="rt-h" class="rt__title">{{ t('dashboard.home.realtimeMonitor') }}</h2>
      <div v-if="!compact" class="rt-seg" role="tablist" :aria-label="t('dashboard.home.metrics')">
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
      <span class="rt__spacer" />
      <div v-if="!compact" class="rt-seg" role="group" :aria-label="t('dashboard.home.refreshEvery')">
        <button
          v-for="opt in INTERVALS"
          :key="opt"
          type="button"
          :class="{ 'is-on': refreshInterval === opt }"
          :aria-pressed="refreshInterval === opt"
          @click="onIntervalChange(opt)"
        >
          {{ opt }}s
        </button>
      </div>
      <button
        v-else
        type="button"
        class="rt__every m-num"
        :aria-label="t('dashboard.home.refreshEvery')"
        @click="cycleInterval"
      >
        {{ t('dashboard.home.everySeconds', { n: refreshInterval }) }}
      </button>
    </div>

    <div v-if="compact" class="rt-seg rt-seg--full" role="tablist" :aria-label="t('dashboard.home.metrics')">
      <button
        v-for="m in metricTabs"
        :key="m.key"
        type="button"
        role="tab"
        :aria-selected="metric === m.key"
        :class="{ 'is-on': metric === m.key }"
        @click="metric = m.key"
      >
        {{ t(m.shortLabel || m.label) }}
      </button>
    </div>

    <div v-if="metric === 'network' || metric === 'disk'" class="rt__bar">
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
        v-else
        class="app-select rt-select"
        :value="selectedDisk"
        :aria-label="t('dashboard.home.diskIoRate')"
        @change="onDiskChange(($event.target as HTMLSelectElement).value)"
      >
        <option value="">{{ t('dashboard.home.allSummary') }}</option>
        <option v-for="d in diskOptions" :key="d.value" :value="d.value">{{ d.label }}</option>
      </select>
    </div>

    <div v-if="compact" class="rt__now">
      <span class="rt__now-v m-num">{{ summary.now }}</span>
      <span class="rt__now-p">{{ t('dashboard.home.peak') }} {{ summary.peak }}</span>
    </div>
    <div v-else class="rt-stats">
      <div class="rt-stat">
        <span class="rt-stat__label">{{ t('dashboard.home.current') }}</span>
        <b class="m-num">{{ summary.now }}</b>
      </div>
      <div class="rt-stat">
        <span class="rt-stat__label">{{ t('dashboard.home.average') }}</span>
        <b class="m-num">{{ summary.avg }}</b>
      </div>
      <div class="rt-stat">
        <span class="rt-stat__label">{{ t('dashboard.home.peak') }}</span>
        <b class="m-num">{{ summary.peak }}</b>
      </div>
      <div v-if="summary.second" class="rt-stat">
        <span class="rt-stat__label">
          <i :style="{ background: pc.a2 }" />
          {{ summary.second.label }}
        </span>
        <b class="m-num">{{ summary.second.value }}</b>
      </div>
    </div>

    <div class="rt-chart">
      <VChart v-if="metric === 'cpu'" :option="cpuChartOption" autoresize />
      <VChart v-else-if="metric === 'memory'" :option="memoryChartOption" autoresize />
      <VChart v-else-if="metric === 'network'" :option="networkChartOption" autoresize />
      <VChart v-else :option="diskChartOption" autoresize />
    </div>
  </section>
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
const metricTabs: { key: Metric; label: string; shortLabel?: string }[] = [
  { key: 'cpu', label: 'dashboard.home.tabCpu' },
  { key: 'memory', label: 'dashboard.home.tabMemory' },
  { key: 'network', label: 'dashboard.home.tabNetwork' },
  { key: 'disk', label: 'dashboard.home.tabDisk', shortLabel: 'dashboard.home.tabDiskShort' },
]
const INTERVALS: RefreshInterval[] = [1, 3, 5, 10]
const menuStore = useMenuStore()
const compact = computed(() => menuStore.isMobile)

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
const cycleInterval = () => {
  const i = INTERVALS.indexOf(props.refreshInterval)
  emit('intervalChange', INTERVALS[(i + 1) % INTERVALS.length]!)
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

const baseChartGrid = computed(() =>
  compact.value ? { left: 44, right: 6, top: 8, bottom: 20 } : { left: 52, right: 12, top: 14, bottom: 26 },
)

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
    grid: baseChartGrid.value,
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
  grid: baseChartGrid.value,
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
  grid: baseChartGrid.value,
  xAxis: { type: 'category', data: props.networkHistory.map((p) => p.time), ...axisStyle.value, splitLine: { show: false } },
  yAxis: { type: 'value', ...axisStyle.value, axisLabel: { ...axisStyle.value.axisLabel, formatter: (v: number) => formatBytes(v) } },
  tooltip: { ...tooltipTheme.value, ...tooltipZ, trigger: 'axis', valueFormatter: (v: number) => formatBytes(v) },
  series: [
    { ...makeLineSeries(t('dashboard.home.download'), pc.value.p), data: props.networkHistory.map((p) => p.download) },
    { ...makeLineSeries(t('dashboard.home.upload'), pc.value.a2), data: props.networkHistory.map((p) => p.upload) },
  ],
}))

const diskChartOption = computed(() => ({
  grid: baseChartGrid.value,
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

const avgOf = (vals: number[]) => (vals.length ? vals.reduce((a, b) => a + b, 0) / vals.length : 0)

/** 当前指标主曲线的「当前 / 平均 / 峰值」，双曲线指标另附第二条曲线的当前值 */
const summary = computed(() => {
  let vals: number[]
  let fmt: (v: number) => string
  let second: { label: string; value: string } | null = null
  if (metric.value === 'cpu') {
    vals = props.cpuHistory.map((p) => p.total)
    fmt = pct
  } else if (metric.value === 'memory') {
    vals = props.memoryHistory.map((p) => p.physical)
    fmt = pct
    second = { label: 'Swap', value: pct(last(props.memoryHistory)?.swap) }
  } else if (metric.value === 'network') {
    vals = props.networkHistory.map((p) => p.download)
    fmt = formatBytes
    second = { label: t('dashboard.home.upload'), value: formatBytes(last(props.networkHistory)?.upload ?? 0) }
  } else {
    vals = props.diskHistory.map((p) => p.read)
    fmt = formatBytes
    second = { label: t('dashboard.home.write'), value: formatBytes(last(props.diskHistory)?.write ?? 0) }
  }
  return {
    now: fmt(last(vals) ?? 0),
    avg: fmt(avgOf(vals)),
    peak: fmt(peakOf(vals)),
    second,
  }
})
</script>

<style scoped lang="scss">
.rt {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
  padding: 22px 24px;
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.rt__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 12px;
}
.rt__title {
  margin: 0 8px 0 0;
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.rt__spacer {
  flex: 1;
}
.rt-seg {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.rt-seg button {
  min-height: 34px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
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
.rt-seg--full {
  display: flex;
}
.rt-seg--full button {
  flex: 1;
  min-height: 40px;
  padding: 0 6px;
}
.rt__every {
  min-height: 36px;
  padding: 0 4px;
  border: 0;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 700;
  cursor: pointer;
}
.rt__bar {
  display: flex;
  margin-top: -4px;
}
.rt-select {
  width: auto;
  max-width: 100%;
  min-width: 180px;
}
.rt-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 24px;
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
  font-weight: 600;
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
.rt__now {
  display: flex;
  align-items: baseline;
  gap: 8px;
}
.rt__now-v {
  font-size: 1.875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.rt__now-p {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.rt-chart {
  width: 100%;
  height: 220px;
}
@media (width < 992px) {
  .rt {
    padding: 16px;
    gap: 12px;
  }
  .rt__title {
    font-size: 1rem;
  }
  .rt-chart {
    height: 150px;
  }
}
</style>
