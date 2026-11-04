<!-- 端口转发实时统计（按原型）：PC 右侧抽屉 / 手机底部面板（ToolSheet）。
     头部：协议贴纸 + 规则名 + 协议标签 + 状态 + 路由；时间范围分段 + 实时开关 + 刷新；2×2 指标格；
     流量速率曲线 + 并发连接柱；活动连接列表（可单条断开、查看全部）；导出 CSV / 断开全部连接。
     ECharts 颜色取自当前配色（usePaletteColors）。 -->
<template>
  <ToolSheet v-model="visible" :label="dialogTitle" :width="520">
    <template #default="{ close }">
      <header class="pfs-head">
        <span class="m-sticker pfs-head__sticker" :class="isUdp ? 'm-sticker--a' : 'm-sticker--a2'">
          <component :is="menuStore.iconComponents['HOutline:ArrowsRightLeftIcon']" class="pfs-head__ico" />
          <span class="pfs-head__proto m-mono">{{ protoText }}</span>
        </span>
        <div class="pfs-head__text">
          <div class="pfs-head__titlerow">
            <h2 class="pfs-head__name">{{ row?.name || t('tools.portForward.stats.title') }}</h2>
            <span class="pfs-proto m-mono" :class="isUdp ? 'is-udp' : 'is-tcp'">{{ protoText }}</span>
            <StatusPill :variant="statusVariant" :label="statusText" />
          </div>
          <div class="pfs-head__route m-mono">
            <span>{{ row?.listenAddress }}:{{ row?.listenPort }}</span>
            <component :is="menuStore.iconComponents['HOutline:ArrowRightIcon']" />
            <span class="pfs-head__target">{{ row?.targetAddress }}:{{ row?.targetPort }}</span>
          </div>
        </div>
        <AppIconButton icon="HOutline:XMarkIcon" :label="t('system.common.close')" :box="44" @click="close" />
      </header>

      <div class="pfs-toolbar">
        <div class="pfs-seg" role="radiogroup" :aria-label="t('tools.portForward.stats.rangeLabel')">
          <button
            v-for="opt in rangeOptions"
            :key="opt.value"
            type="button"
            role="radio"
            class="pfs-seg__btn"
            :class="{ 'is-on': rangeKey === opt.value }"
            :aria-checked="rangeKey === opt.value"
            @click="setRange(opt.value)"
          >{{ opt.label }}</button>
        </div>
        <span class="pfs-spacer" />
        <label class="pfs-live" :class="{ 'is-on': autoRefresh }">
          <input v-model="autoRefresh" type="checkbox" @change="restartPolling" />
          <span class="pfs-live__dot" />
          {{ t('tools.portForward.stats.live') }}
        </label>
        <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('tools.portForward.stats.refresh')" :box="isMobile ? 44 : 40" :size="18" @click="refresh" />
      </div>

      <div v-if="rangeKey === 'custom'" class="pfs-custom">
        <template v-if="!isMobile">
          <label class="pfs-fld">
            <component :is="menuStore.iconComponents['HOutline:CalendarIcon']" />
            <input
              v-model="customStartLocal"
              type="datetime-local"
              class="m-num"
              :aria-label="t('tools.portForward.stats.rangeStart')"
              @change="onCustomChange"
            />
          </label>
          <span class="pfs-muted is-strong">{{ t('tools.portForward.stats.to') }}</span>
          <label class="pfs-fld">
            <input
              v-model="customEndLocal"
              type="datetime-local"
              class="m-num"
              :aria-label="t('tools.portForward.stats.rangeEnd')"
              @change="onCustomChange"
            />
          </label>
        </template>
        <template v-else>
          <div class="pfs-custom__row">
            <span>{{ t('tools.portForward.stats.rangeStart') }}</span>
            <input v-model="customStartDate" type="date" class="app-input" @change="syncMobileCustom" />
            <input v-model="customStartTime" type="time" class="app-input" @change="syncMobileCustom" />
          </div>
          <div class="pfs-custom__row">
            <span>{{ t('tools.portForward.stats.rangeEnd') }}</span>
            <input v-model="customEndDate" type="date" class="app-input" @change="syncMobileCustom" />
            <input v-model="customEndTime" type="time" class="app-input" @change="syncMobileCustom" />
          </div>
        </template>
      </div>

      <dl class="pfs-stats">
        <div>
          <dt>{{ t('tools.portForward.stats.rateIn') }}</dt>
          <dd class="m-num is-p">{{ formatRate(rateIn, '0 B/s') }}</dd>
        </div>
        <div>
          <dt>{{ t('tools.portForward.stats.rateOut') }}</dt>
          <dd class="m-num">{{ formatRate(rateOut, '0 B/s') }}</dd>
        </div>
        <div>
          <dt>{{ t('tools.portForward.stats.activeConnections') }}</dt>
          <dd class="m-num">
            {{ activeConnections }}<small>{{ t('tools.portForward.stats.totalShort', { n: totalConnections }) }}</small>
          </dd>
        </div>
        <div>
          <dt>{{ t('tools.portForward.stats.rangeTraffic') }}</dt>
          <dd class="m-num">
            {{ formatBytes(rangeTraffic) }}<small>{{ t('tools.portForward.stats.totalShort', { n: formatBytes(totalIn + totalOut) }) }}</small>
          </dd>
        </div>
      </dl>

      <section class="pfs-block">
        <div class="pfs-block__head">
          <span class="pfs-block__title">{{ t('tools.portForward.stats.rateChart') }}</span>
          <span class="pfs-legend"><i class="is-in" />{{ t('tools.portForward.stats.inbound') }}</span>
          <span class="pfs-legend"><i class="is-out" />{{ t('tools.portForward.stats.outbound') }}</span>
        </div>
        <div class="pfs-chart-wrap">
          <div ref="rateEl" class="pfs-chart" />
          <div v-if="!points.length && !loading" class="pfs-empty">{{ t('tools.portForward.stats.noData') }}</div>
        </div>
      </section>

      <section class="pfs-block">
        <div class="pfs-block__head">
          <span class="pfs-block__title">{{ t('tools.portForward.stats.connectionsChart') }}</span>
          <span class="pfs-muted m-num is-strong">{{ t('tools.portForward.stats.peak', { n: connPeak }) }}</span>
        </div>
        <div class="pfs-chart-wrap is-small">
          <div ref="connEl" class="pfs-chart" />
        </div>
      </section>

      <section class="pfs-conns">
        <div class="pfs-block__head">
          <span class="pfs-block__title">{{ t('tools.portForward.stats.activeConnTitle', { n: connections.length }) }}</span>
          <UButton
            v-if="connections.length > CONN_LIMIT"
            color="neutral"
            variant="soft"
            size="sm"
            @click="showAllConns = !showAllConns"
          >
            {{ showAllConns ? t('tools.portForward.stats.collapse') : t('tools.portForward.stats.viewAll') }}
          </UButton>
        </div>
        <div v-for="c in visibleConnections" :key="c.id" class="pfs-conn">
          <span class="pfs-conn__dot" aria-hidden="true" />
          <span class="pfs-conn__main">
            <span class="m-mono pfs-conn__addr">{{ c.remoteAddr }}</span>
            <span class="pfs-muted">
              {{ c.lan ? t('tools.portForward.stats.lan') : t('tools.portForward.stats.public') }} ·
              {{ t('tools.portForward.stats.connectedFor', { d: formatDuration(Date.now() - (toDate(c.startTime)?.getTime() || Date.now()), t) }) }}
            </span>
          </span>
          <span class="pfs-conn__bytes m-num">↓ {{ formatBytes(c.bytesIn) }}<small>↑ {{ formatBytes(c.bytesOut) }}</small></span>
          <AppIconButton
            class="pfs-conn__kill"
            :label="t('tools.portForward.stats.disconnect', { addr: c.remoteAddr })"
            :box="isMobile ? 44 : 36"
            @click="closeConns([c.id])"
          >
            <UIcon name="i-lucide-unplug" />
          </AppIconButton>
        </div>
        <div v-if="!connections.length" class="pfs-conns__empty">{{ t('tools.portForward.stats.noConnections') }}</div>
      </section>

      <footer class="pfs-foot">
        <UButton color="neutral" variant="outline" size="lg" icon="i-lucide-download" :disabled="!points.length" @click="exportCsv">
          {{ t('tools.portForward.stats.exportCsv') }}
        </UButton>
        <UButton color="error" variant="soft" size="lg" icon="i-lucide-unplug" :disabled="!connections.length" @click="closeConns()">
          {{ t('tools.portForward.stats.disconnectAll') }}
        </UButton>
        <UButton v-if="isMobile" class="pfs-foot__close" color="neutral" variant="soft" size="xl" @click="close">
          {{ t('system.common.close') }}
        </UButton>
      </footer>
    </template>
  </ToolSheet>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import * as echarts from 'echarts'
import { getPortForwardStats, closePortForwardConnections } from '@/api/network'
import {
  PortForwardType,
  PortForwardStatus,
  type PortForwardInfo,
  type PortForwardStatPoint,
  type PortForwardConnection,
} from '@/types/v1/network'
import { showRequestError } from '@/utils/request'
import { usePaletteColors, withAlpha } from '@/composables/usePaletteColors'
import { formatBytes, formatRate, formatDateTime, formatDuration, toDate, downloadText } from '@/views/tools/components/format'
import ToolSheet from '@/views/tools/components/ToolSheet.vue'

type RangeKey = '5m' | '30m' | '1h' | '6h' | '24h' | 'custom'
type ChartPoint = { ts: number; label: string; activeConnections: number; inboundRate: number; outboundRate: number; bytesIn: number; bytesOut: number }
type TooltipParam = { axisValueLabel?: string; marker?: string; seriesName?: string; value?: number | string }

const POLL_INTERVAL_MS = 5000
const CONN_LIMIT = 5
const PRESET_SECONDS: Record<Exclude<RangeKey, 'custom'>, number> = {
  '5m': 300, '30m': 1800, '1h': 3600, '6h': 21600, '24h': 86400,
}

const props = defineProps<{ modelValue: boolean; row: PortForwardInfo | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; changed: [] }>()
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
const running = ref(false)
const activeConnections = ref(0)
const totalConnections = ref(0)
const rateIn = ref(0)
const rateOut = ref(0)
const totalIn = ref(0)
const totalOut = ref(0)
const points = ref<ChartPoint[]>([])
const connections = ref<PortForwardConnection[]>([])
const showAllConns = ref(false)

let rateChart: echarts.ECharts | null = null
let connChart: echarts.ECharts | null = null
let pollTimer: ReturnType<typeof setTimeout> | null = null

const isUdp = computed(() => props.row?.type === PortForwardType.PORT_FORWARD_TYPE_UDP)
const protoText = computed(() => (isUdp.value ? 'UDP' : 'TCP'))
// 状态取列表行（父页轮询更新）；行缺失时退回快照的运行状态
const statusVariant = computed(() => {
  const st = props.row?.status
  if (st === PortForwardStatus.PORT_FORWARD_STATUS_RUNNING || (!st && running.value)) return 'success'
  if (st === PortForwardStatus.PORT_FORWARD_STATUS_ERROR) return 'error'
  return 'neutral'
})
const statusText = computed(() => {
  if (statusVariant.value === 'success') return t('tools.portForward.statusRunning')
  if (statusVariant.value === 'error') return t('tools.portForward.statusError')
  return t('tools.portForward.statusStopped')
})
const visibleConnections = computed(() => (showAllConns.value ? connections.value : connections.value.slice(0, CONN_LIMIT)))
const connPeak = computed(() => points.value.reduce((m, p) => Math.max(m, p.activeConnections), activeConnections.value))
const rangeTraffic = computed(() => {
  // 区间流量：相邻采样累计值之差（计数归零时取新值）
  let sum = 0
  for (let i = 1; i < points.value.length; i++) {
    const a = points.value[i - 1]!
    const b = points.value[i]!
    const dIn = b.bytesIn - a.bytesIn
    const dOut = b.bytesOut - a.bytesOut
    sum += dIn < 0 || dOut < 0 ? b.bytesIn + b.bytesOut : dIn + dOut
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
  if (customStartDate.value && customStartTime.value) {
    customStart.value = new Date(`${customStartDate.value}T${customStartTime.value}`)
  }
  if (customEndDate.value && customEndTime.value) {
    customEnd.value = new Date(`${customEndDate.value}T${customEndTime.value}`)
  }
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
  { value: '5m' as RangeKey, label: t('tools.portForward.stats.range5m') },
  { value: '30m' as RangeKey, label: t('tools.portForward.stats.range30m') },
  { value: '1h' as RangeKey, label: t('tools.portForward.stats.range1h') },
  { value: '6h' as RangeKey, label: t('tools.portForward.stats.range6h') },
  { value: '24h' as RangeKey, label: t('tools.portForward.stats.range24h') },
  { value: 'custom' as RangeKey, label: t('tools.portForward.stats.rangeCustom') },
])

const dialogTitle = computed(() => {
  const name = props.row?.name
  return name ? t('tools.portForward.stats.titleWithName', { name }) : t('tools.portForward.stats.title')
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

const buildChartPoints = (raw: PortForwardStatPoint[], spanSeconds: number): ChartPoint[] =>
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
    return {
      ts: time.getTime(),
      label: axisLabel(time, spanSeconds),
      activeConnections: num(point.activeConnections),
      inboundRate,
      outboundRate,
      bytesIn,
      bytesOut,
    }
  })

const tooltipFormatter = (params: unknown) => {
  const items = (Array.isArray(params) ? params : [params]) as TooltipParam[]
  const title = items[0]?.axisValueLabel || ''
  const rows = items.map((item) => {
    const value = Number(item.value ?? 0)
    const text = item.seriesName === t('tools.portForward.stats.connectionsChart') ? String(value) : formatRate(value, '0 B/s')
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
    grid: { top: 12, left: 8, right: 12, bottom: 4, containLabel: true },
    yAxis: {
      type: 'value',
      min: 0,
      splitNumber: 3,
      // 速率很小时避免出现 0.25 / 0.5 B/s 之类刻度被格式化成重复的「1 B/s」
      minInterval: 1,
      splitLine: { lineStyle: { color: c.line, type: 'dashed' } },
      axisLabel: { color: c.ink2, fontSize: 11, formatter: (v: number) => formatRate(v, '0') },
    },
    series: [
      {
        name: t('tools.portForward.stats.inbound'),
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
        name: t('tools.portForward.stats.outbound'),
        type: 'line',
        smooth: true,
        showSymbol,
        symbolSize: 6,
        data: points.value.map((p) => p.outboundRate),
        lineStyle: { width: 2.5, color: c.a2 },
        itemStyle: { color: c.a2 },
      },
    ],
  }, { notMerge: true })
  connChart?.setOption({
    ...base,
    animation: animate,
    // 并发柱只看形状：横轴时间与上方速率曲线一致，这里不再重复标注
    xAxis: { ...base.xAxis, boundaryGap: true, axisLabel: { show: false } },
    grid: { top: 6, left: 8, right: 12, bottom: 2, containLabel: true },
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
        name: t('tools.portForward.stats.connectionsChart'),
        type: 'bar',
        barMaxWidth: 14,
        data: points.value.map((p) => p.activeConnections),
        itemStyle: { color: c.a2, borderRadius: [4, 4, 0, 0] },
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
    const { data } = await getPortForwardStats({ id: props.row.id, startTimeMs: range.startTimeMs, endTimeMs: range.endTimeMs })
    const snapshot = data?.snapshot
    running.value = !!snapshot?.running
    activeConnections.value = num(snapshot?.activeConnections)
    totalConnections.value = num(snapshot?.totalConnections)
    rateIn.value = num(snapshot?.rateIn)
    rateOut.value = num(snapshot?.rateOut)
    totalIn.value = num(snapshot?.totalBytesIn)
    totalOut.value = num(snapshot?.totalBytesOut)
    connections.value = data?.connections || []
    points.value = buildChartPoints(data?.points || [], (range.endTimeMs - range.startTimeMs) / 1000)
    await nextTick()
    updateCharts(animate)
  } catch (error) {
    showRequestError(error, t('tools.portForward.stats.loadFailed'))
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

const closeConns = async (ids?: string[]) => {
  if (!props.row?.id) return
  try {
    const { data } = await closePortForwardConnections({ id: props.row.id, connIds: ids || [], all: !ids })
    feedback.success(t('tools.portForward.stats.disconnected', { n: Number(data?.closed || 0) }))
    emit('changed')
    setTimeout(() => void loadStats(), 300)
  } catch {
    /* interceptor */
  }
}

const exportCsv = () => {
  const header = 'time,active_connections,inbound_bytes_per_sec,outbound_bytes_per_sec,inbound_total_bytes,outbound_total_bytes'
  const lines = points.value.map((p) =>
    [formatDateTime(new Date(p.ts)), p.activeConnections, Math.round(p.inboundRate), Math.round(p.outboundRate), p.bytesIn, p.bytesOut].join(','),
  )
  downloadText(`port-forward-${props.row?.name || 'stats'}.csv`, [header, ...lines].join('\n') + '\n', 'text/csv;charset=utf-8')
}

const handleOpened = async () => {
  showAllConns.value = false
  await nextTick()
  ensureCharts()
  await loadStats(true)
}
const handleClosed = () => {
  stopPolling()
  points.value = []
  connections.value = []
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
.pfs-muted { font-size: 0.75rem; color: var(--el-text-color-secondary); }
.pfs-muted.is-strong { font-weight: 700; }

/* 头部 */
.pfs-head { display: flex; align-items: flex-start; gap: 12px; min-width: 0; }
.pfs-head__sticker {
  width: 48px;
  height: 48px;
  flex: none;
  border-radius: 16px;
  transform: rotate(-4deg);
}
.pfs-head__ico { width: 24px; height: 24px; }
.pfs-head__proto { display: none; font-size: 0.75rem; font-weight: 800; letter-spacing: 0.04em; }
.pfs-head__text { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 4px; padding-top: 2px; }
.pfs-head__titlerow { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; min-width: 0; }
.pfs-head__name {
  margin: 0;
  min-width: 0;
  font-family: var(--app-font-display);
  font-size: 1.3125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pfs-proto {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 8px;
  border-radius: 8px;
  font-size: 0.75rem;
  font-weight: 800;
  letter-spacing: 0.04em;
}
.pfs-proto.is-tcp { background: var(--m-a2Soft); color: var(--m-ink); }
.pfs-proto.is-udp { background: var(--m-aSoft); color: var(--m-ink); }
.pfs-head__route {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  span { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
  :deep(svg) { width: 14px; height: 14px; flex: none; color: var(--m-pInk); }
}
.pfs-head__target { color: var(--el-text-color-primary); font-weight: 700; }

/* 时间范围 */
.pfs-toolbar { display: flex; align-items: center; gap: 6px; flex-wrap: wrap; }
.pfs-spacer { flex: 1; }
.pfs-seg {
  display: inline-flex;
  gap: 2px;
  padding: 4px;
  max-width: 100%;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  overflow-x: auto;
  scrollbar-width: none;
}
.pfs-seg::-webkit-scrollbar { display: none; }
.pfs-seg__btn {
  flex: none;
  min-height: 34px;
  /* 480px 抽屉里 6 个时段 + 实时 + 刷新要排在一行 */
  padding: 0 9px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.pfs-seg__btn.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.pfs-live {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 4px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  user-select: none;
  input { position: absolute; opacity: 0; pointer-events: none; }
}
.pfs-live.is-on { color: var(--m-ok); }
.pfs-live__dot { width: 8px; height: 8px; border-radius: 50%; background: var(--el-border-color-light); }
.pfs-live.is-on .pfs-live__dot { background: var(--m-ok); box-shadow: 0 0 0 4px var(--m-okSoft); }
.pfs-live:focus-within { outline: 3px solid var(--m-a2); outline-offset: 2px; border-radius: 8px; }
.pfs-custom { display: flex; align-items: center; flex-wrap: wrap; gap: 8px; }
.pfs-fld {
  flex: 1;
  min-width: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 12px;
  border-radius: 12px;
  background: var(--el-fill-color-light);
  border: 1.5px solid transparent;
  color: var(--el-text-color-secondary);
  :deep(svg) { width: 16px; height: 16px; flex: none; }
  input {
    flex: 1;
    min-width: 0;
    border: 0;
    background: transparent;
    font: inherit;
    font-size: 0.8125rem;
    font-weight: 600;
    color: var(--el-text-color-primary);
    outline: none;
  }
}
.pfs-fld:focus-within { border-color: var(--m-p); }
.pfs-custom__row {
  display: grid;
  grid-template-columns: 4rem 1fr 6rem;
  gap: 6px;
  align-items: center;
  width: 100%;
  span { font-size: 0.75rem; font-weight: 700; color: var(--el-text-color-secondary); }
  .app-input { min-width: 0; height: 44px; }
}

/* 指标格 */
.pfs-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin: 0;
  > div {
    min-width: 0;
    padding: 12px 14px;
    border-radius: var(--app-radius);
    background: var(--el-fill-color-light);
  }
  dt { font-size: 0.75rem; font-weight: 700; color: var(--el-text-color-secondary); white-space: nowrap; }
  dd {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 2px 8px;
    margin: 4px 0 0;
    font-family: var(--app-font-display);
    font-size: 1.375rem;
    font-weight: 800;
    color: var(--el-text-color-primary);
    min-width: 0;
    small { font-family: inherit; font-size: 0.75rem; font-weight: 600; color: var(--el-text-color-secondary); }
  }
  dd.is-p { color: var(--m-pInk); }
}

/* 图表 */
.pfs-block { display: flex; flex-direction: column; gap: 8px; }
.pfs-block__head { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; min-height: 32px; }
.pfs-block__title { flex: 1; font-size: 0.9375rem; font-weight: 800; color: var(--el-text-color-primary); }
.pfs-legend {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  i { width: 10px; height: 10px; border-radius: 3px; }
  i.is-in { background: var(--m-p); }
  i.is-out { background: var(--m-a2); }
}
.pfs-chart-wrap { position: relative; height: 180px; }
.pfs-chart-wrap.is-small { height: 92px; }
.pfs-chart { width: 100%; height: 100%; }
.pfs-empty {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}

/* 活动连接 */
.pfs-conns { display: flex; flex-direction: column; }
.pfs-conns .pfs-block__head { padding-bottom: 6px; }
.pfs-conn {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 56px;
  padding: 8px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.pfs-conn__dot { width: 8px; height: 8px; flex: none; border-radius: 50%; background: var(--m-ok); }
.pfs-conn__main { display: flex; flex-direction: column; gap: 1px; flex: 1; min-width: 0; }
.pfs-conn__addr {
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pfs-conn__bytes {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  line-height: 1.35;
  white-space: nowrap;
  small { font-size: inherit; font-weight: 700; color: var(--el-text-color-secondary); }
}
.pfs-conns :deep(.pfs-conn__kill) { color: var(--m-bad); }
.pfs-conns :deep(.pfs-conn__kill:hover) { color: var(--m-bad); background: var(--m-badSoft); }
.pfs-conns :deep(.pfs-conn__kill .iconify) { width: 16px; height: 16px; }
.pfs-conns__empty {
  padding: 18px 0;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  text-align: center;
}

.pfs-foot { display: flex; gap: 10px; margin-top: auto; padding-top: 4px; }
.pfs-foot > * { flex: 1; justify-content: center; }

@media (width <= 768px) {
  .pfs-head { align-items: center; }
  .pfs-head__sticker { width: 46px; height: 46px; border-radius: 15px; }
  .pfs-head__ico { display: none; }
  .pfs-head__proto { display: inline; }
  .pfs-head__titlerow .pfs-proto { display: none; }
  .pfs-head__name { font-size: 1.125rem; }
  .pfs-head__route { font-size: 0.72rem; }
  .pfs-spacer { display: none; }
  .pfs-seg { width: 100%; }
  .pfs-seg__btn { flex: 1; min-height: 44px; padding: 0 6px; }
  .pfs-toolbar { justify-content: space-between; }
  .pfs-stats { gap: 8px; }
  .pfs-stats > div { padding: 10px 12px; }
  .pfs-stats dd { font-size: 1.1875rem; }
  .pfs-chart-wrap { height: 150px; }
  .pfs-foot { flex-wrap: wrap; }
  .pfs-foot__close { flex-basis: 100%; min-height: 52px; }
}
</style>
