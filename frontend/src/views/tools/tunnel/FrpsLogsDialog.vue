<!-- frps 日志：贴纸头 + 级别分段 + 实时开关 + 刷新/下载 → 深色日志区（时间 · 级别 · 内容，固定终端色）。 -->
<template>
  <FormDialog v-model="visible" :title="t('tools.tunnel.logs.title')" :width="900" :show-footer="false">
    <div class="fl">
      <div class="fl-head">
        <span class="m-sticker m-sticker--a2 fl-head__sticker"><component :is="menuStore.iconComponents['HOutline:DocumentTextIcon']" /></span>
        <span class="fl-head__sub">{{ t('tools.tunnel.logs.sub') }}</span>
      </div>

      <div class="fl-toolbar">
        <div class="fl-seg" role="radiogroup" :aria-label="t('tools.tunnel.logs.title')">
          <button
            v-for="opt in levelOptions"
            :key="opt.value"
            type="button"
            role="radio"
            class="fl-seg__btn"
            :class="{ 'is-on': level === opt.value }"
            :aria-checked="level === opt.value"
            @click="level = opt.value"
          >{{ opt.label }}<span class="fl-seg__n m-num">{{ opt.count }}</span></button>
        </div>
        <span class="fl-spacer" />
        <label class="fl-live">
          <input v-model="autoRefresh" type="checkbox" @change="schedule" />
          <span class="fl-live__dot" :class="{ 'is-on': autoRefresh }" />
          {{ t('tools.tunnel.logs.live') }}
        </label>
        <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('tools.tunnel.logs.refresh')" :box="36" :size="18" @click="load(true)" />
        <AppIconButton icon="HOutline:ArrowDownTrayIcon" :label="t('tools.tunnel.logs.download')" :box="36" :size="18" @click="download" />
      </div>

      <div ref="boxEl" class="fl-box m-mono" role="log" :aria-label="t('tools.tunnel.logs.title')" tabindex="0">
        <div v-if="!shown.length && !loading" class="fl-empty">
          <b>{{ t('tools.tunnel.logs.empty') }}</b>
          <span>{{ t('tools.tunnel.logs.emptyDesc') }}</span>
        </div>
        <div v-for="(e, i) in shown" :key="i" class="fl-line" :class="`is-${e.level}`">
          <span class="fl-line__time">{{ clock(e.time) }}</span>
          <span class="fl-line__lv">{{ levelTag(e.level) }}</span>
          <span class="fl-line__msg">{{ e.message }}</span>
        </div>
      </div>
      <div class="fl-foot m-num">{{ t('tools.tunnel.logs.lines', { n: shown.length }) }}</div>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { getFrpsLogs } from '@/api/tunnel'
import type { FrpsLogEntry } from '@/types/v1/tunnel'
import { downloadText, toDate, formatDateTime } from '@/views/tools/components/format'

type Level = 'all' | 'warn'

const POLL_MS = 3000

const props = defineProps<{ modelValue: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const { t } = useI18n()
const menuStore = useMenuStore()

const visible = computed({
  get: () => props.modelValue,
  set: (v: boolean) => emit('update:modelValue', v),
})

const entries = ref<FrpsLogEntry[]>([])
const loading = ref(false)
const autoRefresh = ref(true)
const level = ref<Level>('all')
const boxEl = ref<HTMLDivElement | null>(null)
let timer: ReturnType<typeof setTimeout> | null = null

const isWarn = (e: FrpsLogEntry) => e.level === 'warn' || e.level === 'error'
const shown = computed(() => (level.value === 'warn' ? entries.value.filter(isWarn) : entries.value))
const levelOptions = computed(() => [
  { value: 'all' as Level, label: t('tools.tunnel.logs.levelAll'), count: entries.value.length },
  { value: 'warn' as Level, label: t('tools.tunnel.logs.levelWarn'), count: entries.value.filter(isWarn).length },
])

const pad2 = (n: number) => String(n).padStart(2, '0')
const clock = (v: unknown) => {
  const d = toDate(v)
  return d ? `${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}:${pad2(d.getSeconds())}` : ''
}
const levelTag = (lv: string) => (lv || 'info').slice(0, 4).toUpperCase()

const atBottom = () => {
  const el = boxEl.value
  return !el || el.scrollHeight - el.scrollTop - el.clientHeight < 24
}

const load = async (scroll = false) => {
  loading.value = true
  const stick = scroll || atBottom()
  try {
    const { data } = await getFrpsLogs({ limit: 1000 })
    entries.value = data?.entries || []
    if (stick) {
      await nextTick()
      if (boxEl.value) boxEl.value.scrollTop = boxEl.value.scrollHeight
    }
  } catch {
    /* interceptor */
  } finally {
    loading.value = false
    schedule()
  }
}

const stop = () => {
  if (timer) { clearTimeout(timer); timer = null }
}
const schedule = () => {
  stop()
  if (visible.value && autoRefresh.value) timer = setTimeout(() => void load(), POLL_MS)
}

const download = () => {
  const text = entries.value.map((e) => `${formatDateTime(e.time)} [${levelTag(e.level)}] ${e.message}`).join('\n') + '\n'
  downloadText('frps.log', text)
}

watch(visible, (open) => {
  if (open) void load(true)
  else stop()
})
onBeforeUnmount(stop)
</script>

<style scoped lang="scss">
.fl { display: flex; flex-direction: column; gap: 12px; }
.fl-head { display: flex; align-items: center; gap: 12px; }
.fl-head__sticker {
  width: 44px;
  height: 44px;
  flex: none;
  border-radius: 14px;
  transform: rotate(-5deg);
  :deep(svg) { width: 22px; height: 22px; }
}
.fl-head__sub { font-size: 0.8125rem; color: var(--el-text-color-secondary); }
.fl-toolbar { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.fl-spacer { flex: 1; }
.fl-seg { display: inline-flex; gap: 2px; padding: 3px; border-radius: 999px; background: var(--el-fill-color-light); }
.fl-seg__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 34px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.fl-seg__btn.is-on { background: var(--el-bg-color); color: var(--el-text-color-primary); box-shadow: var(--app-shadow-sm); }
.fl-seg__n { font-size: 0.75rem; opacity: 0.7; }
.fl-live {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 36px;
  padding: 0 4px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  user-select: none;
  input { position: absolute; opacity: 0; pointer-events: none; }
}
.fl-live__dot { width: 8px; height: 8px; border-radius: 50%; background: var(--el-border-color-light); }
.fl-live__dot.is-on { background: var(--m-ok); box-shadow: 0 0 0 4px var(--m-okSoft); }

/* 日志区：固定终端色 */
.fl-box {
  height: min(56vh, 480px);
  overflow: auto;
  padding: 12px 0;
  border-radius: var(--app-radius-lg);
  background: #0f0b14;
  color: #e8e3ee;
  font-size: 0.75rem;
  line-height: 1.7;
  scrollbar-width: thin;
  scrollbar-color: #3a3142 transparent;
}
.fl-box:focus-visible { outline: 3px solid var(--m-a2); outline-offset: 2px; }
.fl-line { display: flex; gap: 10px; padding: 0 14px; }
.fl-line:hover { background: rgba(255, 255, 255, 0.04); }
.fl-line__time { flex: none; color: #8a8197; }
.fl-line__lv { flex: none; width: 3.2em; font-weight: 800; color: #79c0ff; }
.fl-line__msg { min-width: 0; white-space: pre-wrap; word-break: break-all; }
.fl-line.is-warn .fl-line__lv { color: #ffc14d; }
.fl-line.is-error .fl-line__lv,
.fl-line.is-error .fl-line__msg { color: #ff7b8a; }
.fl-line.is-debug .fl-line__lv,
.fl-line.is-trace .fl-line__lv { color: #8a8197; }
.fl-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 6px;
  height: 100%;
  padding: 0 24px;
  font-family: var(--app-font-display, inherit);
  text-align: center;
  b { font-size: 1rem; color: #e8e3ee; }
  span { font-size: 0.8125rem; color: #b8b0c2; }
}
.fl-foot { font-size: 0.75rem; color: var(--el-text-color-secondary); text-align: right; }
@media (width <= 768px) {
  .fl-seg { width: 100%; }
  .fl-seg__btn { flex: 1; justify-content: center; min-height: 40px; }
  .fl-spacer { display: none; }
  .fl-box { height: 56vh; font-size: 0.6875rem; }
  .fl-line { flex-wrap: wrap; gap: 0 8px; padding: 2px 12px; }
  .fl-line__msg { flex-basis: 100%; }
}
</style>
