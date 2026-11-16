<!-- 趋势弹窗：近 14 天的单系列图（活跃玩家折线 / 抽卡次数柱状），数据是 dashboard 的 series。
     今天还没过完：柱子用浅色，说明行和提示框里写「截至 HH:mm」。页面上只放今天的数字和打开它的按钮。 -->
<template>
  <FormDialog
    :model-value="modelValue"
    :title="title"
    :width="760"
    :show-footer="false"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
  >
    <div class="td">
      <p class="td__caption">{{ caption }}</p>
      <TrendChart v-if="points.length" :kind="kind" :points="points" :color="color" :format="format" :aria-label="aria" :height="260" />
      <p v-else class="td__empty">{{ loading ? t('gmData.common.loading') : t('gmData.home.trendEmpty') }}</p>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import type { Dashboard } from '@/games/gakumas/types'
import { useGmFormat } from '../player/shared/format'
import TrendChart, { type TrendPoint } from './TrendChart.vue'

const props = defineProps<{
  modelValue: boolean
  /** active：活跃玩家（折线）；draws：抽卡次数（柱状） */
  field: 'active' | 'draws'
  report?: Dashboard
  loading?: boolean
  /** 游戏皮肤色（页面用 useGameChartColor 读好传进来） */
  color: string
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

const { t } = useI18n()
const { num, delta } = useGmFormat()

const kind = computed(() => (props.field === 'active' ? 'line' : 'bar'))
const range = computed(() => t('gmData.home.lastDays', { n: props.report?.series.length || 14 }))
const title = computed(() =>
  props.field === 'active' ? t('gmData.home.activeChart', { range: range.value }) : t('gmData.home.drawChart', { range: range.value }),
)
const asOf = computed(() => (props.report?.asOf ? dayjs(props.report.asOf).format('HH:mm') : '—'))
const caption = computed(() => {
  const hint = props.field === 'active' ? t('gmData.home.activeHint') : t('gmData.home.drawHint')
  return `${hint}；${t('gmData.home.today')}${t('gmData.home.asOf', { time: asOf.value })}`
})
const format = (v: number) => (props.field === 'active' ? t('gmData.home.people', { n: num(v) }) : t('gmData.home.times', { n: num(v) }))

const points = computed<TrendPoint[]>(() => {
  const series = props.report?.series || []
  return series.map((s, i) => {
    const last = i === series.length - 1
    const d = dayjs(s.day)
    const label = `${d.format('MM-DD')}（${t(`gmData.weekday.${d.day()}`)}）${last ? ` · ${t('gmData.home.today')}` : ''}`
    const value = props.field === 'active' ? s.active : s.draws
    const prev = series[i - 1]
    const note = last
      ? t('gmData.home.asOf', { time: asOf.value })
      : props.field === 'active' && prev
        ? t('gmData.home.vsPrev', { d: delta(s.active - prev.active) })
        : ''
    return { axis: last ? t('gmData.home.today') : d.format('MM-DD'), label, value, note, partial: last && props.field === 'draws' }
  })
})

const aria = computed(() => {
  const series = props.report?.series || []
  const key = props.field
  const peak = series.reduce((best, s) => (best && best[key] >= s[key] ? best : s), series[0])
  const last = series[series.length - 1]
  if (!peak || !last) return ''
  return key === 'active'
    ? t('gmData.home.activeAria', { day: peak.day, peak: peak.active, today: last.active })
    : t('gmData.home.drawAria', { day: peak.day, peak: peak.draws, today: last.draws })
})
</script>

<style scoped lang="scss">
.td {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.td__caption,
.td__empty {
  margin: 0;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
}
.td__empty {
  padding: 40px 0;
  text-align: center;
}
</style>
