<!-- 工作台的单系列趋势图（折线或柱状）：无图例，悬停看当天数值；今天还没过完，柱子用浅色。
     颜色由页面传入（游戏皮肤的 --m-gInk，保证与背景 3:1 对比）。 -->
<template>
  <div class="trend" :style="{ height: `${height}px` }" role="img" :aria-label="ariaLabel">
    <VChart :option="option" autoresize />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { usePaletteColors, withAlpha } from '@/composables/usePaletteColors'

export interface TrendPoint {
  /** 横轴标签 */
  axis: string
  /** 提示框标题 */
  label: string
  value: number
  /** 提示框附注 */
  note?: string
  /** 还没过完的一天 */
  partial?: boolean
}

const props = withDefaults(
  defineProps<{
    points: TrendPoint[]
    kind: 'line' | 'bar'
    color: string
    /** 数值格式（提示框里的「N 人」「N 次」） */
    format: (value: number) => string
    ariaLabel?: string
    height?: number
  }>(),
  { ariaLabel: '', height: 220 },
)

const pc = usePaletteColors()

const escape = (s: string) => s.replace(/[&<>"]/g, (c) => ({ '&': '&amp;', '<': '&lt;', '>': '&gt;', '"': '&quot;' })[c] || c)
const compact = (v: number) => (v >= 10000 ? `${Math.round(v / 1000)}k` : v >= 1000 ? `${Math.round(v / 100) / 10}k` : String(v))

const option = computed(() => {
  const c = pc.value
  const color = props.color || c.pInk
  const n = props.points.length
  const step = n > 7 ? Math.ceil(n / 5) : 1
  const series =
    props.kind === 'line'
      ? {
          type: 'line' as const,
          data: props.points.map((p, i) =>
            i === n - 1
              ? {
                  value: p.value,
                  symbolSize: 10,
                  label: { show: true, position: 'top' as const, color: c.ink, fontWeight: 800, fontSize: 13, formatter: () => compact(p.value) },
                }
              : p.value,
          ),
          showSymbol: true,
          symbol: 'circle',
          symbolSize: 0,
          lineStyle: { color, width: 2, join: 'round' as const },
          itemStyle: { color, borderColor: c.s, borderWidth: 2 },
          areaStyle: { color: withAlpha(color, 0.14) },
          emphasis: { scale: false, itemStyle: { color } },
        }
      : {
          type: 'bar' as const,
          barMaxWidth: 28,
          data: props.points.map((p) => ({
            value: p.value,
            itemStyle: { color: p.partial ? withAlpha(color, 0.5) : color, borderRadius: [4, 4, 0, 0] },
          })),
          emphasis: { itemStyle: { opacity: 0.85 } },
        }
  return {
    animation: false,
    grid: { left: 40, right: 14, top: 24, bottom: 24 },
    xAxis: {
      type: 'category' as const,
      data: props.points.map((p) => p.axis),
      boundaryGap: props.kind === 'bar',
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: {
        color: c.ink2,
        fontSize: 11,
        interval: (i: number) => (n - 1 - i) % step === 0,
      },
    },
    yAxis: {
      type: 'value' as const,
      minInterval: 1,
      splitNumber: 3,
      axisLabel: { color: c.ink2, fontSize: 11, formatter: compact },
      splitLine: { lineStyle: { color: c.line, type: 'dashed' as const } },
    },
    tooltip: {
      trigger: 'axis' as const,
      confine: true,
      backgroundColor: c.ink,
      borderWidth: 0,
      borderRadius: 12,
      padding: [8, 10],
      textStyle: { color: c.bg, fontSize: 12 },
      axisPointer: {
        type: props.kind === 'bar' ? ('shadow' as const) : ('line' as const),
        lineStyle: { color: c.ink2, width: 1 },
        shadowStyle: { color: withAlpha(c.ink2, 0.08) },
      },
      formatter: (params: { dataIndex: number }[]) => {
        const p = props.points[params[0]?.dataIndex ?? -1]
        if (!p) return ''
        const note = p.note ? `<div>${escape(p.note)}</div>` : ''
        return `<div>${escape(p.label)}</div><div style="font-size:15px;font-weight:800">${escape(props.format(p.value))}</div>${note}`
      },
    },
    series: [series],
  }
})
</script>

<style scoped>
.trend {
  width: 100%;
  min-width: 0;
}
</style>
