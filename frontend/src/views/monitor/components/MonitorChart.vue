<!-- 性能监控曲线：时间轴折线 + 面积，空桶断开不连线；服务停止 / 暂停记录的空档画成斜纹「无数据」区。
     PC 上可拖动框选放大（toolbox dataZoom 选择模式，选中后抛出 zoom 事件由页面改成自定义时段重新查询）。 -->
<template>
  <div class="mchart" :style="{ height: `${height}px` }">
    <VChart ref="chartRef" :option="option" autoresize @datazoom="onZoom" />
  </div>
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { usePaletteColors, withAlpha } from '@/composables/usePaletteColors'
import { axisTimeFormat, fmtRate, pointTime } from '../format'

export interface ChartLine {
  key: string
  label: string
  color: string
  fill: number
  values: (number | null)[]
}

const props = withDefaults(
  defineProps<{
    times: number[]
    lines: ChartLine[]
    kind: 'pct' | 'rate'
    /** 速率类坐标轴的单位除数（与副标题里的单位一致） */
    div?: number
    gaps?: { start: number; end: number }[]
    step: number
    height?: number
    compact?: boolean
    zoomable?: boolean
    gapLabel?: string
    gapTip?: string
  }>(),
  { div: 1, gaps: () => [], height: 210, compact: false, zoomable: false, gapLabel: '', gapTip: '' },
)

const emit = defineEmits<{ zoom: [start: number, end: number] }>()

const pc = usePaletteColors()
const chartRef = ref<{ inner?: { dispatchAction: (payload: Record<string, unknown>) => void } }>()

// 斜纹填充（与原型 repeating-linear-gradient 相同的观感）
const hatch = (color: string) => {
  const cv = document.createElement('canvas')
  cv.width = 10
  cv.height = 10
  const g = cv.getContext('2d')
  if (g) {
    g.strokeStyle = color
    g.lineWidth = 3.5
    g.beginPath()
    for (const o of [-10, 0, 10]) {
      g.moveTo(o, 10)
      g.lineTo(o + 10, 0)
    }
    g.stroke()
  }
  return { image: cv, repeat: 'repeat' as const }
}

const fmtValue = (v: number) => (props.kind === 'pct' ? `${v.toFixed(1)}%` : fmtRate(v * props.div))

const option = computed(() => {
  const c = pc.value
  const t0 = props.times[0] ?? 0
  const t1 = (props.times[props.times.length - 1] ?? t0) + props.step * 1000
  const span = (t1 - t0) / 1000
  const timeFmt = axisTimeFormat(span)
  const gapArea = props.gaps.length
    ? {
        silent: true,
        itemStyle: { color: hatch(c.line), borderColor: c.ink2, borderType: 'dashed' as const, borderWidth: 1 },
        label: {
          show: true,
          position: 'insideTop' as const,
          distance: 4,
          color: c.ink2,
          fontSize: 11,
          fontWeight: 700,
          backgroundColor: c.bg,
          padding: [2, 8],
          borderRadius: 999,
        },
        data: props.gaps.map((g) => {
          const wide = span > 0 && (g.end - g.start) / span > 0.08
          return [
            { xAxis: g.start * 1000, name: wide && !props.compact ? props.gapLabel : '' },
            { xAxis: g.end * 1000 },
          ]
        }),
      }
    : undefined

  const isolated = (vals: (number | null)[], k: number) => vals[k] != null && vals[k - 1] == null && vals[k + 1] == null
  const series = props.lines.map((l, i) => ({
    name: l.label,
    type: 'line' as const,
    showSymbol: true,
    showAllSymbol: true,
    symbol: 'circle',
    // 只有前后都是空桶的孤立点画成小圆点，其余不画点
    symbolSize: (_v: unknown, p: { dataIndex: number }) => (isolated(l.values, p.dataIndex) ? (props.compact ? 4 : 5) : 0),
    connectNulls: false,
    lineStyle: { color: l.color, width: props.compact ? 1.8 : 2, join: 'round' as const },
    itemStyle: { color: l.color },
    areaStyle: { color: withAlpha(l.color, l.fill), origin: 'start' as const },
    emphasis: { disabled: true },
    data: props.times.map((t, k) => {
      const v = l.values[k]
      return [t, v == null ? null : props.kind === 'pct' ? v : v / props.div]
    }),
    markArea: i === 0 ? gapArea : undefined,
    z: 10 - i,
  }))

  return {
    animation: false,
    grid: props.compact ? { left: 34, right: 4, top: 6, bottom: 20 } : { left: 44, right: 8, top: 10, bottom: 22 },
    xAxis: {
      type: 'time' as const,
      min: t0,
      max: t1,
      splitNumber: props.compact ? 3 : 5,
      axisLabel: {
        color: c.ink2,
        fontSize: props.compact ? 10 : 11,
        hideOverlap: true,
        formatter: (v: number) => dayjs(v).format(timeFmt),
      },
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { show: false },
    },
    yAxis: {
      type: 'value' as const,
      min: 0,
      max: props.kind === 'pct' ? 100 : undefined,
      splitNumber: 2,
      axisLabel: {
        color: c.ink2,
        fontSize: props.compact ? 10 : 11,
        formatter: (v: number) => (props.kind === 'pct' ? `${v}%` : `${Math.round(v * 10) / 10}`),
      },
      axisLine: { show: false },
      axisTick: { show: false },
      splitLine: { lineStyle: { color: c.line, type: 'dashed' as const } },
    },
    tooltip: {
      trigger: 'axis' as const,
      confine: true,
      backgroundColor: c.s,
      borderColor: c.line,
      borderRadius: 14,
      padding: [8, 12],
      textStyle: { color: c.ink, fontSize: 12 },
      extraCssText: 'box-shadow: 0 12px 28px -14px rgba(20,10,30,.4); z-index: 99999 !important',
      axisPointer: { type: 'line' as const, lineStyle: { color: c.ink, width: 1.5, type: 'solid' as const } },
      formatter: (params: { value: [number, number | null]; marker: string; seriesName: string }[]) => {
        const first = params[0]
        if (!first) return ''
        let html = `<div style="font-weight:800;margin-bottom:3px">${pointTime(first.value[0], props.step)}</div>`
        const rows = params.filter((p) => p.value[1] != null)
        if (!rows.length) return html + `<div style="color:${c.ink2}">${props.gapTip}</div>`
        for (const p of rows) {
          html += `<div style="display:flex;align-items:center;gap:6px">${p.marker}${p.seriesName} <b>${fmtValue(p.value[1] as number)}</b></div>`
        }
        return html
      },
    },
    // toolbox 必须渲染出来框选才会生效：放到画布外、图标为空，只借用它的「框选缩放」
    toolbox: props.zoomable
      ? {
          show: true,
          left: -200,
          top: -200,
          itemSize: 1,
          showTitle: false,
          feature: { dataZoom: { yAxisIndex: 'none' as const, icon: { zoom: 'path://M0 0', back: 'path://M0 0' } } },
        }
      : undefined,
    series,
  }
})

// 框选放大：开启 toolbox 的选择光标；选中后把时段交给页面重新查询，并复位图表自身的缩放
const enableBrush = () => {
  if (!props.zoomable) return
  chartRef.value?.inner?.dispatchAction({ type: 'takeGlobalCursor', key: 'dataZoomSelect', dataZoomSelectActive: true })
}
watch(option, () => nextTick(enableBrush), { flush: 'post' })
onMounted(() => setTimeout(enableBrush, 0))

const onZoom = (e: { batch?: { startValue?: number; endValue?: number }[] }) => {
  const b = e.batch?.[0]
  if (!b || b.startValue == null || b.endValue == null) return
  const start = Math.floor(Math.min(b.startValue, b.endValue) / 1000)
  const end = Math.ceil(Math.max(b.startValue, b.endValue) / 1000)
  chartRef.value?.inner?.dispatchAction({ type: 'dataZoom', start: 0, end: 100 })
  if (end - start >= 60) emit('zoom', start, end)
}
</script>

<style scoped>
.mchart {
  position: relative;
  width: 100%;
  min-width: 0;
}
.mchart > * {
  width: 100%;
  height: 100%;
}
</style>
