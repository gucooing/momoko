<!-- 客户端日志查看器（「客户端日志」页与 GM 玩家详情共用）：打开即读出日志全文，不用下载。
     头部是这份日志的信息（服务器、玩家、通行证、原因、设备、版本、时间、大小、摘要与说明）；
     正文等宽、保留换行，按行虚拟滚动（只渲染看得见的行，几十 MB 的日志也不卡），带行号；
     「在日志里查找」标出全部匹配并在匹配之间跳转（Enter 下一个、Shift+Enter 上一个）；复制全部与下载 .log。 -->
<template>
  <FormDialog
    :model-value="modelValue"
    :title="title"
    :width="1180"
    :show-footer="false"
    @update:model-value="(v: boolean) => emit('update:modelValue', v)"
  >
    <div v-if="log" class="clv">
      <dl class="clv__facts">
        <div>
          <dt>{{ t('clientLogs.facts.server') }}</dt>
          <dd>{{ clientLogServer(log) }}</dd>
        </div>
        <div>
          <dt>{{ t('clientLogs.facts.player') }}</dt>
          <dd class="m-mono">{{ log.playerId || '—' }}</dd>
        </div>
        <div>
          <dt>{{ t('clientLogs.facts.passport') }}</dt>
          <dd>{{ clientLogPassport(log) || t('clientLogs.noLink') }}</dd>
        </div>
        <div>
          <dt>{{ t('clientLogs.facts.reason') }}</dt>
          <dd><StatusPill :variant="clientLogReasonVariant(log.reason)" :label="clientLogReasonLabel(t, log.reason)" /></dd>
        </div>
        <div>
          <dt>{{ t('clientLogs.facts.device') }}</dt>
          <dd>{{ log.device || '—' }}</dd>
        </div>
        <div>
          <dt>{{ t('clientLogs.facts.version') }}</dt>
          <dd class="m-mono">{{ log.clientVersion || '—' }}</dd>
        </div>
        <div>
          <dt>{{ t('clientLogs.facts.occurredAt') }}</dt>
          <dd class="m-num">{{ dateTime(log.occurredAt) }}</dd>
        </div>
        <div>
          <dt>{{ t('clientLogs.facts.createdAt') }}</dt>
          <dd class="m-num">{{ dateTime(log.createdAt) }}</dd>
        </div>
        <div>
          <dt>{{ t('clientLogs.facts.size') }}</dt>
          <dd class="m-num">{{ formatFileSize(log.size) }}</dd>
        </div>
        <div v-if="log.summary" class="is-wide">
          <dt>{{ t('clientLogs.facts.summary') }}</dt>
          <dd>{{ log.summary }}</dd>
        </div>
        <div v-if="log.note" class="is-wide">
          <dt>{{ t('clientLogs.facts.note') }}</dt>
          <dd>{{ log.note }}</dd>
        </div>
      </dl>

      <div class="clv__bar">
        <label class="clv__find">
          <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" class="clv__find-icon" />
          <input
            v-model="query"
            type="search"
            :placeholder="t('clientLogs.viewer.find')"
            :aria-label="t('clientLogs.viewer.find')"
            :disabled="!lines.length"
            @keydown.enter.prevent="onEnter($event.shiftKey)"
          />
          <span v-if="needle" class="clv__count m-num">{{ matchText }}</span>
          <AppIconButton icon="HOutline:ChevronUpIcon" :label="t('clientLogs.viewer.prev')" :box="32" :size="16" @click="step(-1)" />
          <AppIconButton icon="HOutline:ChevronDownIcon" :label="t('clientLogs.viewer.next')" :box="32" :size="16" @click="step(1)" />
        </label>
        <span class="clv__grow" />
        <UButton color="primary" variant="soft" icon="i-lucide-copy" :disabled="!content" @click="copyAll">{{ t('clientLogs.copyAll') }}</UButton>
        <UButton color="neutral" variant="ghost" icon="i-lucide-download" :disabled="!content" @click="download">
          {{ t('clientLogs.actions.download') }}
        </UButton>
      </div>

      <div v-if="loading" class="clv__state">{{ t('clientLogs.viewer.loading') }}</div>
      <div v-else-if="error" class="clv__state is-error">{{ error }}</div>
      <div v-else ref="scrollRef" class="clv__scroll m-mono" @scroll="onScroll">
        <div class="clv__track" :style="{ height: `${trackHeight}px` }">
          <div class="clv__lines" :style="{ transform: `translateY(${blockTop}px)` }">
            <div v-for="row in rows" :key="row.index" class="clv__line" :class="{ 'is-current': row.current }">
              <span class="clv__ln">{{ row.index + 1 }}</span>
              <span class="clv__text"><template v-for="(p, i) in row.parts" :key="i"><mark v-if="p.hit" :class="{ 'is-current': p.current }">{{ p.text }}</mark><template v-else>{{ p.text }}</template></template></span>
            </div>
          </div>
        </div>
      </div>
      <div v-if="!loading && !error" class="clv__foot m-num">{{ t('clientLogs.viewer.lines', { n: lines.length.toLocaleString() }) }}</div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, nextTick, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn, useResizeObserver } from '@vueuse/core'
import type { GmClientLog } from '@/types/v1/game_gm'
import { formatFileSize } from '@/utils/file'
import { getRequestErrorMessage } from '@/utils/request'
import { useGmFormat } from '@/views/game/gm/player/shared/format'
import {
  clientLogPassport,
  clientLogReasonLabel,
  clientLogReasonVariant,
  clientLogServer,
  copyClientLogText,
  fetchClientLogContent,
  saveClientLog,
} from '../clientLogs'

const props = defineProps<{ modelValue: boolean; log: GmClientLog | null }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()

/** 每行高度（px，与样式一致）。 */
const LINE_HEIGHT = 18
/** 视口上下多渲染的行数。 */
const OVERSCAN = 30
/** 滚动轨道的最大高度：行数太多时按比例换算滚动位置（浏览器对元素高度有上限）。 */
const MAX_TRACK = 8_000_000
/** 最多标出的匹配数。 */
const MAX_HITS = 10_000

const { t } = useI18n()
const menuStore = useMenuStore()
const { dateTime } = useGmFormat()

const title = computed(() => (props.log ? `${clientLogReasonLabel(t, props.log.reason)} · ${dateTime(props.log.occurredAt)}` : ''))

// ---------- 读取 ----------
const content = ref('')
const lines = shallowRef<string[]>([])
let lowerLines: string[] | null = null
const loading = ref(false)
const error = ref('')
let seq = 0

const load = async (log: GmClientLog) => {
  const id = ++seq
  content.value = ''
  lines.value = []
  lowerLines = null
  error.value = ''
  query.value = ''
  needle.value = ''
  hits.value = []
  capped.value = false
  current.value = 0
  scrollTop.value = 0
  loading.value = true
  try {
    const text = await fetchClientLogContent(log.id)
    if (id !== seq) return
    content.value = text
    lines.value = text.split(/\r?\n/)
  } catch (e) {
    if (id === seq) error.value = getRequestErrorMessage(e)
  } finally {
    if (id === seq) {
      loading.value = false
      await nextTick()
      measure()
    }
  }
}

// ---------- 虚拟滚动 ----------
const scrollRef = ref<HTMLElement>()
const scrollTop = ref(0)
const viewHeight = ref(600)

const measure = () => {
  const el = scrollRef.value
  if (!el) return
  viewHeight.value = el.clientHeight || 600
  scrollTop.value = el.scrollTop
}
useResizeObserver(scrollRef, measure)

const onScroll = () => {
  const el = scrollRef.value
  if (el) scrollTop.value = el.scrollTop
}

const fullHeight = computed(() => lines.value.length * LINE_HEIGHT)
const scaled = computed(() => fullHeight.value > MAX_TRACK)
const trackHeight = computed(() => Math.min(fullHeight.value, MAX_TRACK))
const pageLines = computed(() => Math.max(1, Math.ceil(viewHeight.value / LINE_HEIGHT)))
const maxFirst = computed(() => Math.max(0, lines.value.length - pageLines.value))
const maxScroll = computed(() => Math.max(0, trackHeight.value - viewHeight.value))

/** 视口顶部那一行。 */
const firstLine = computed(() => {
  if (!scaled.value) return Math.floor(scrollTop.value / LINE_HEIGHT)
  return maxScroll.value > 0 ? Math.round((scrollTop.value / maxScroll.value) * maxFirst.value) : 0
})
const start = computed(() => Math.max(0, firstLine.value - OVERSCAN))
const end = computed(() => Math.min(lines.value.length, firstLine.value + pageLines.value + OVERSCAN))
const blockTop = computed(() =>
  scaled.value ? scrollTop.value - (firstLine.value - start.value) * LINE_HEIGHT : start.value * LINE_HEIGHT,
)

/** 滚动到让第 index 行出现在视口中间。 */
const scrollToLine = (index: number) => {
  const el = scrollRef.value
  if (!el) return
  const first = Math.min(maxFirst.value, Math.max(0, index - Math.floor(pageLines.value / 2)))
  el.scrollTop = scaled.value ? (maxFirst.value > 0 ? (first / maxFirst.value) * maxScroll.value : 0) : first * LINE_HEIGHT
  scrollTop.value = el.scrollTop
}

// ---------- 查找 ----------
interface Hit {
  line: number
  col: number
}
interface Part {
  text: string
  hit: boolean
  current: boolean
}

const query = ref('')
const needle = ref('')
const hits = shallowRef<Hit[]>([])
const capped = ref(false)
const current = ref(0)

const runSearch = () => {
  const q = query.value.trim().toLowerCase()
  needle.value = q
  current.value = 0
  if (!q) {
    hits.value = []
    capped.value = false
    return
  }
  if (!lowerLines) lowerLines = lines.value.map((l) => l.toLowerCase())
  const out: Hit[] = []
  let more = false
  for (let i = 0; i < lowerLines.length && !more; i++) {
    const line = lowerLines[i] ?? ''
    for (let j = line.indexOf(q); j >= 0; j = line.indexOf(q, j + q.length)) {
      if (out.length >= MAX_HITS) {
        more = true
        break
      }
      out.push({ line: i, col: j })
    }
  }
  hits.value = out
  capped.value = more
  if (out.length) void jump()
}
const debouncedSearch = useDebounceFn(runSearch, 200)
watch(query, () => void debouncedSearch())

const jump = async () => {
  const h = hits.value[current.value]
  if (!h) return
  scrollToLine(h.line)
  await nextTick()
  scrollRef.value?.querySelector<HTMLElement>('mark.is-current')?.scrollIntoView({ block: 'nearest', inline: 'center' })
}

const step = (delta: number) => {
  const n = hits.value.length
  if (!n) return
  current.value = (current.value + delta + n) % n
  void jump()
}

const onEnter = (back: boolean) => {
  if (query.value.trim().toLowerCase() !== needle.value) {
    runSearch()
    return
  }
  step(back ? -1 : 1)
}

const matchText = computed(() => {
  const n = hits.value.length
  if (!n) return t('clientLogs.viewer.noMatch')
  return t(capped.value ? 'clientLogs.viewer.matchesMore' : 'clientLogs.viewer.matches', { cur: current.value + 1, n })
})

/** 每行的匹配列（只含标出的匹配）。 */
const hitsByLine = computed(() => {
  const m = new Map<number, number[]>()
  for (const h of hits.value) {
    const cols = m.get(h.line)
    if (cols) cols.push(h.col)
    else m.set(h.line, [h.col])
  }
  return m
})

const splitLine = (text: string, cols: number[], len: number, currentCol: number): Part[] => {
  const parts: Part[] = []
  let i = 0
  for (const c of cols) {
    if (c < i || c >= text.length) continue
    if (c > i) parts.push({ text: text.slice(i, c), hit: false, current: false })
    parts.push({ text: text.slice(c, c + len), hit: true, current: c === currentCol })
    i = c + len
  }
  if (i < text.length) parts.push({ text: text.slice(i), hit: false, current: false })
  return parts
}

const rows = computed(() => {
  const out: { index: number; current: boolean; parts: Part[] }[] = []
  const cur = hits.value[current.value]
  const len = needle.value.length
  for (let i = start.value; i < end.value; i++) {
    const text = lines.value[i] ?? ''
    const cols = hitsByLine.value.get(i)
    const isCurrent = cur?.line === i
    out.push({
      index: i,
      current: isCurrent,
      parts: cols ? splitLine(text, cols, len, isCurrent && cur ? cur.col : -1) : [{ text, hit: false, current: false }],
    })
  }
  return out
})

// ---------- 复制与下载 ----------
const copyAll = () => void copyClientLogText(t, content.value, t('clientLogs.copiedAll'))

const download = () => {
  if (props.log && content.value) saveClientLog(props.log, content.value)
}

// 打开或换一份日志时读取全文（重新打开同一份时沿用已读出的内容）。
watch(
  () => [props.modelValue, props.log?.id] as const,
  ([open, id], prev) => {
    if (!open || !props.log || !id) return
    if (prev && prev[0] && prev[1] === id && (content.value || loading.value)) return
    void load(props.log)
  },
  { immediate: true },
)
</script>

<style scoped lang="scss">
.clv {
  display: flex;
  flex-direction: column;
  gap: 12px;
  min-width: 0;
}
.clv__facts {
  margin: 0;
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(170px, 1fr));
  gap: 10px 16px;
  padding: 12px 14px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.clv__facts > div {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.clv__facts > div.is-wide {
  grid-column: 1 / -1;
}
.clv__facts dt {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.clv__facts dd {
  margin: 0;
  font-size: 13px;
  font-weight: 600;
  word-break: break-all;
}
.clv__bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
}
.clv__grow {
  flex: 1;
}
.clv__find {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1 1 360px;
  min-width: 0;
  max-width: 560px;
  min-height: 40px;
  padding: 0 4px 0 12px;
  border-radius: 999px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.clv__find:focus-within {
  border-color: var(--m-p);
}
.clv__find-icon {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.clv__find input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 13px;
}
.clv__count {
  flex: none;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.clv__state {
  padding: 16px 0;
  text-align: center;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.clv__state.is-error {
  color: var(--m-bad);
}
.clv__scroll {
  height: 62vh;
  min-height: 280px;
  overflow: auto;
  border-radius: var(--app-radius);
  border: 1px solid var(--el-border-color-light);
  background: var(--m-s2);
  color: var(--el-text-color-primary);
  font-size: 12px;
}
.clv__track {
  width: max-content;
  min-width: 100%;
}
.clv__lines {
  padding: 0 14px 0 0;
  will-change: transform;
}
.clv__line {
  display: flex;
  height: 18px;
  line-height: 18px;
  white-space: pre;
  tab-size: 4;
}
.clv__line.is-current {
  background: var(--m-warnSoft);
}
.clv__ln {
  position: sticky;
  left: 0;
  flex: none;
  min-width: 7ch;
  padding: 0 12px 0 8px;
  text-align: right;
  color: var(--el-text-color-secondary);
  background: var(--m-s2);
  user-select: none;
}
.clv__text mark {
  border-radius: 3px;
  background: var(--m-warnSoft);
  color: var(--m-ink);
  outline: 1px solid var(--m-warn);
}
.clv__text mark.is-current {
  background: var(--m-warn);
  color: var(--m-ink);
}
.clv__foot {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
@media (max-width: 760px) {
  .clv__scroll {
    height: 58vh;
    font-size: 11px;
  }
}
</style>
