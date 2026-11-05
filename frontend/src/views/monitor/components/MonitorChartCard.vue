<!-- 一张曲线卡（MonitorPC 的详细曲线 / MonitorMobile 选中指标的曲线）：
     贴纸 + 标题 / 副标题 + 网卡、设备筛选 chips → 图例（可点掉，带当前 / 平均 / 峰值）→ 曲线 → 信息格；
     没有显卡、采集项关闭或这段时间没有数据时显示空状态。 -->
<template>
  <section :id="cardId" class="mcard" :class="{ 'mcard--compact': compact }" :aria-label="title">
    <div class="mcard__head">
      <span class="mcard__stk m-sticker" :class="`m-sticker--${tone}`" :style="{ transform: `rotate(${rot})` }">
        <MonitorGlyph :name="glyph" />
      </span>
      <span class="mcard__ttl">
        <span class="mcard__title">{{ title }}</span>
        <span class="mcard__sub">{{ sub }}</span>
      </span>
      <div v-if="picks.length && !compact" class="mcard__picks">
        <button
          v-for="p in picks"
          :key="p.value"
          type="button"
          class="mcard__chip"
          :class="{ 'is-on': p.value === pick }"
          :aria-pressed="p.value === pick"
          @click="emit('pick', p.value)"
        >
          {{ p.label }}
        </button>
      </div>
    </div>
    <div v-if="picks.length && compact" class="mcard__picks mcard__picks--row">
      <button
        v-for="p in picks"
        :key="p.value"
        type="button"
        class="mcard__chip"
        :class="{ 'is-on': p.value === pick }"
        :aria-pressed="p.value === pick"
        @click="emit('pick', p.value)"
      >
        {{ p.label }}
      </button>
    </div>

    <div v-if="empty" class="mcard__empty">
      <span class="mcard__empty-stk m-sticker m-sticker--a2"><MonitorGlyph :name="empty.glyph || glyph" /></span>
      <span class="mcard__empty-t">{{ empty.title }}</span>
      <span class="mcard__empty-d">{{ empty.desc }}</span>
      <UButton v-if="empty.action" size="sm" color="neutral" variant="outline" class="mcard__empty-btn" @click="emit('action')">
        {{ empty.action }}
      </UButton>
    </div>

    <template v-else>
      <div class="mcard__legend">
        <button
          v-for="l in lines"
          :key="l.key"
          type="button"
          class="mcard__lg"
          :class="{ 'is-off': hidden.includes(l.key) }"
          :aria-pressed="!hidden.includes(l.key)"
          @click="emit('toggle', l.key)"
        >
          <i :style="{ background: l.color }" />{{ l.label }}
          <b class="m-num">{{ l.now }}</b>
          <span v-if="compact" class="mcard__lg-sp" />
          <span class="mcard__lg-st m-num">{{ t('monitor.legendStat', { avg: l.avg, peak: l.peak }) }}</span>
        </button>
      </div>
      <MonitorChart
        :times="times"
        :lines="visibleLines"
        :kind="kind"
        :div="div"
        :gaps="gaps"
        :step="step"
        :height="compact ? 180 : 214"
        :compact="compact"
        :zoomable="zoomable"
        :gap-label="t('monitor.gap')"
        :gap-tip="t('monitor.gapTip')"
        @zoom="(s, e) => emit('zoom', s, e)"
      />
      <dl v-if="facts.length" class="mcard__facts">
        <div v-for="f in facts" :key="f.label" class="mcard__fact">
          <dt>{{ f.label }}</dt>
          <dd class="m-num" :class="f.tone ? `is-${f.tone}` : ''">{{ f.value }}</dd>
        </div>
      </dl>
    </template>
  </section>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import MonitorChart from './MonitorChart.vue'
import MonitorGlyph, { type MonitorGlyphName } from './MonitorGlyph.vue'

export interface CardLine {
  key: string
  label: string
  color: string
  fill: number
  values: (number | null)[]
  now: string
  avg: string
  peak: string
}
export interface CardFact {
  label: string
  value: string
  tone?: 'ok' | 'warn' | 'bad'
}
export interface CardEmpty {
  title: string
  desc: string
  action?: string
  glyph?: MonitorGlyphName
}

const props = withDefaults(
  defineProps<{
    cardId?: string
    title: string
    sub: string
    glyph: MonitorGlyphName
    tone: string
    rot?: string
    lines: CardLine[]
    hidden: string[]
    picks?: { value: string; label: string }[]
    pick?: string
    facts?: CardFact[]
    empty?: CardEmpty | null
    times: number[]
    kind: 'pct' | 'rate'
    div?: number
    gaps?: { start: number; end: number }[]
    step: number
    compact?: boolean
    zoomable?: boolean
  }>(),
  { cardId: undefined, rot: '0deg', picks: () => [], pick: '', facts: () => [], empty: null, div: 1, gaps: () => [], compact: false, zoomable: false },
)

const emit = defineEmits<{
  toggle: [key: string]
  pick: [value: string]
  zoom: [start: number, end: number]
  action: []
}>()

const { t } = useI18n()
const visibleLines = computed(() => props.lines.filter((l) => !props.hidden.includes(l.key)))
</script>

<style scoped lang="scss">
.mcard {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
  padding: 18px 20px 16px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  scroll-margin-top: 16px;
}
.mcard__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.mcard__stk {
  width: 36px;
  height: 36px;
  border-radius: 12px;
  font-size: 16px;
}
.mcard__ttl {
  display: flex;
  flex: 1 1 160px;
  flex-direction: column;
  min-width: 0;
}
.mcard__title {
  font-family: var(--app-font-display);
  font-size: 1.0625rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.mcard__sub {
  overflow: hidden;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mcard__picks {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.mcard__picks--row {
  flex-wrap: nowrap;
  overflow-x: auto;
  margin: 0 -14px;
  padding: 0 14px;
  scrollbar-width: none;
}
.mcard__picks--row::-webkit-scrollbar {
  display: none;
}
.mcard__chip {
  flex: none;
  min-height: 34px;
  padding: 0 12px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s;
}
.mcard__chip:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.mcard__chip.is-on,
.mcard__chip.is-on:hover {
  border-color: var(--m-ink);
  background: var(--m-ink);
  color: var(--m-bg);
}

.mcard__legend {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 18px;
}
.mcard__lg {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 32px;
  padding: 0 4px;
  border: 0;
  border-radius: 8px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
}
.mcard__lg:hover {
  background: var(--el-fill-color-light);
}
.mcard__lg i {
  width: 10px;
  height: 10px;
  border-radius: 3px;
}
.mcard__lg b {
  color: var(--el-text-color-primary);
  font-weight: 800;
}
.mcard__lg-st {
  font-weight: 600;
}
.mcard__lg.is-off {
  opacity: 0.45;
  text-decoration: line-through;
}

.mcard__facts {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px 14px;
  margin: 4px 0 0;
  padding: 12px 14px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
}
.mcard__fact {
  min-width: 0;
}
.mcard__fact dt {
  font-size: 0.6875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.mcard__fact dd {
  overflow: hidden;
  margin: 2px 0 0;
  font-size: 0.875rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.mcard__fact dd.is-warn {
  color: var(--m-warn);
}
.mcard__fact dd.is-bad {
  color: var(--m-bad);
}
.mcard__fact dd.is-ok {
  color: var(--m-ok);
}

.mcard__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
  min-height: 230px;
  padding: 24px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
  text-align: center;
}
.mcard__empty-stk {
  width: 48px;
  height: 48px;
  border-radius: 16px;
  font-size: 22px;
  transform: rotate(-5deg);
}
.mcard__empty-t {
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.mcard__empty-d {
  max-width: 380px;
  font-size: 0.8125rem;
  line-height: 1.55;
  color: var(--el-text-color-secondary);
}
.mcard__empty-btn {
  margin-top: 4px;
  border-radius: 999px;
}

/* 手机：图例逐行（当前值靠左、平均 / 峰值靠右），信息格两列 */
.mcard--compact {
  gap: 10px;
  padding: 16px 14px 14px;
}
.mcard--compact .mcard__stk {
  width: 34px;
  height: 34px;
  border-radius: 11px;
}
.mcard--compact .mcard__title {
  font-size: 1rem;
}
.mcard--compact .mcard__sub {
  font-size: 0.6875rem;
}
.mcard--compact .mcard__legend {
  flex-direction: column;
  gap: 2px;
}
.mcard--compact .mcard__lg {
  width: 100%;
  min-height: 36px;
  text-align: left;
}
.mcard--compact .mcard__lg-sp {
  flex: 1;
}
.mcard--compact .mcard__lg-st {
  font-size: 0.75rem;
}
.mcard--compact .mcard__facts {
  grid-template-columns: repeat(2, minmax(0, 1fr));
  margin: 0;
  padding: 10px 12px;
}
.mcard--compact .mcard__empty {
  min-height: 200px;
  padding: 20px;
}
.mcard--compact .mcard__empty-stk {
  width: 44px;
  height: 44px;
  border-radius: 14px;
}
.mcard--compact .mcard__empty-t {
  font-size: 0.875rem;
}
.mcard--compact .mcard__empty-d {
  font-size: 0.75rem;
}
</style>
