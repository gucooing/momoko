<!-- 监控设置（MonitorSettingsPC 右侧抽屉 / MonitorSettingsMobile 底部面板，外壳复用 ToolSheet）：
     记录历史开关 → 采样间隔 → 历史保留时长（预设 + 自定义天数）与分级精度说明、当前 / 预计占用 → 采集项目 → 导出 / 清空。
     没有 monitor:edit 时只读（导出仍可用）。 -->
<template>
  <ToolSheet :model-value="modelValue" :label="t('monitor.panel.title')" :width="520" flush @update:model-value="emit('update:modelValue', $event)">
    <template #default="{ close }">
      <header class="mset__head">
        <span class="mset__ttl">
          <span class="mset__title">{{ t('monitor.panel.title') }}</span>
          <span class="mset__sub">{{ compact ? t('monitor.panel.subShort') : t('monitor.panel.sub', { node: nodeName }) }}</span>
        </span>
        <AppIconButton icon="HOutline:XMarkIcon" :label="t('common.close')" :box="44" @click="close" />
      </header>

      <div class="mset__body">
        <p v-if="!canEdit" class="mset__readonly">{{ t('monitor.panel.readonly') }}</p>

        <div class="mset__row">
          <span class="mset__row-t">
            <span class="mset__row-title">{{ t('monitor.panel.record') }}</span>
            <span class="mset__row-desc">{{ t('monitor.panel.recordDesc') }}</span>
          </span>
          <AppSwitch v-model="draft.recordEnabled" :disabled="!canEdit" :aria-label="t('monitor.panel.record')" />
        </div>

        <div class="mset__group">
          <span class="mset__lbl">{{ t('monitor.panel.interval') }}</span>
          <div class="mset__seg" role="radiogroup" :aria-label="t('monitor.panel.interval')">
            <button
              v-for="iv in INTERVALS"
              :key="iv"
              type="button"
              role="radio"
              :class="{ 'is-on': draft.intervalSeconds === iv }"
              :aria-checked="draft.intervalSeconds === iv"
              :disabled="!canEdit"
              @click="draft.intervalSeconds = iv"
            >
              {{ t('monitor.seconds', { n: iv }) }}
            </button>
          </div>
          <span v-if="!compact" class="mset__hint">{{ t('monitor.panel.intervalHint') }}</span>
        </div>

        <div class="mset__group">
          <span class="mset__lbl">{{ t('monitor.panel.retention') }}</span>
          <div class="mset__chips">
            <button
              v-for="r in RETAINS"
              :key="r"
              type="button"
              class="mset__chip"
              :class="{ 'is-on': retainChoice === r }"
              :aria-pressed="retainChoice === r"
              :disabled="!canEdit"
              @click="retainChoice = r"
            >
              {{ retainLabel(r) }}
            </button>
          </div>
          <label v-if="retainChoice === 'custom'" class="mset__days">
            <input
              v-model.number="customDays"
              type="number"
              min="1"
              max="3650"
              inputmode="numeric"
              :disabled="!canEdit"
              :aria-label="t('monitor.panel.customDays')"
            />
            <span>{{ t('monitor.panel.daysUnit') }}</span>
          </label>
          <div class="mset__tiers">
            <span v-if="!compact" class="mset__tiers-t">{{ t('monitor.panel.tiersTitle') }}</span>
            <div v-for="tr in tiers" :key="tr.label" class="mset__tier">
              <i :style="{ background: tr.color }" />
              <span class="mset__tier-l">{{ tr.label }}</span>
              <span class="mset__tier-v m-num">{{ tr.keep }}</span>
            </div>
            <div class="mset__store">
              <MonitorGlyph v-if="!compact" name="db" class="mset__store-ic" />
              <span class="mset__store-l">{{ t('monitor.panel.usage') }} <b class="m-num">{{ fmtSize(info?.storageBytes) }}</b></span>
              <span class="mset__store-r m-num">
                {{ compact ? t('monitor.panel.projectedShort') : t('monitor.panel.projected') }}
                <b>{{ t('monitor.panel.about', { v: fmtSize(projected) }) }}</b>
              </span>
            </div>
          </div>
        </div>

        <div class="mset__group mset__group--metrics">
          <span class="mset__lbl">{{ t('monitor.panel.metrics') }}</span>
          <div v-for="m in metrics" :key="m.key" class="mset__metric">
            <span v-if="!compact" class="mset__metric-stk m-sticker" :class="`m-sticker--${m.tone}`"><MonitorGlyph :name="m.glyph" /></span>
            <span class="mset__row-t">
              <span class="mset__row-title">{{ m.title }}</span>
              <span class="mset__row-desc">{{ m.desc }}</span>
            </span>
            <AppSwitch
              :model-value="m.on"
              :disabled="m.lock || !canEdit"
              :aria-label="m.title"
              @update:model-value="(v: boolean) => setMetric(m.key, v)"
            />
          </div>
        </div>

        <div class="mset__group">
          <span v-if="!compact" class="mset__lbl">{{ t('monitor.panel.data') }}</span>
          <div class="mset__data">
            <UButton
              color="neutral"
              :variant="compact ? 'soft' : 'outline'"
              :loading="exporting"
              class="mset__btn"
              @click="onExport"
            >
              <MonitorGlyph name="download" class="mset__btn-ic" />
              {{ compact ? t('monitor.panel.exportShort') : t('monitor.panel.export') }}
            </UButton>
            <UButton v-if="canEdit" color="error" variant="soft" class="mset__btn" @click="onClear">
              <MonitorGlyph name="trash" class="mset__btn-ic" />
              {{ compact ? t('monitor.panel.clearShort') : t('monitor.panel.clear') }}
            </UButton>
          </div>
        </div>
      </div>

      <footer class="mset__foot">
        <UButton color="neutral" variant="soft" class="mset__foot-btn" @click="close">{{ t('monitor.panel.cancel') }}</UButton>
        <UButton v-if="canEdit" color="primary" icon="i-lucide-check" :loading="saving" class="mset__foot-btn mset__foot-btn--main" @click="onSave(close)">
          {{ t('monitor.panel.save') }}
        </UButton>
      </footer>
    </template>
  </ToolSheet>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import ToolSheet from '@/views/tools/components/ToolSheet.vue'
import { clearMonitorHistory, exportMonitorData, getMonitorSettings, updateMonitorSettings } from '@/api/monitor'
import type { GetMonitorSettingsResponse, MonitorSettings } from '@/types/v1/monitor'
import { Dialog } from '@/utils/dialog'
import { downloadText } from '@/views/tools/components/format'
import { usePaletteColors } from '@/composables/usePaletteColors'
import MonitorGlyph, { type MonitorGlyphName } from './MonitorGlyph.vue'
import { fmtSize, num } from '../format'

const props = defineProps<{
  modelValue: boolean
  nodeName: string
  canEdit: boolean
  compact: boolean
  /** 当前查看的时段（unix 秒），导出用 */
  range: { start: number; end: number }
}>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [settings: MonitorSettings]; cleared: [] }>()

const { t } = useI18n()
const pc = usePaletteColors()

const INTERVALS = [5, 10, 30, 60]
const RETAINS = ['7', '30', '90', '180', '365', 'custom'] as const
type RetainChoice = (typeof RETAINS)[number]

const info = ref<GetMonitorSettingsResponse | null>(null)
const draft = reactive<MonitorSettings>({
  recordEnabled: true,
  intervalSeconds: 10,
  retentionDays: 30,
  collectGpu: true,
  collectDisk: true,
  collectNetwork: true,
  collectApps: true,
})
const retainChoice = ref<RetainChoice>('30')
const customDays = ref<number>(45)
const saving = ref(false)
const exporting = ref(false)

const load = async () => {
  const { data } = await getMonitorSettings()
  info.value = data
  Object.assign(draft, data.settings)
  const days = num(data.settings?.retentionDays) || 30
  const preset = RETAINS.find((r) => r !== 'custom' && Number(r) === days)
  retainChoice.value = preset || 'custom'
  customDays.value = days
}
watch(
  () => props.modelValue,
  (open) => {
    if (open) void load()
  },
)

const retainLabel = (r: RetainChoice) =>
  r === 'custom' ? t('monitor.custom') : r === '365' ? t('monitor.oneYear') : t('monitor.days', { n: r })

const retentionDays = computed(() => (retainChoice.value === 'custom' ? Math.round(num(customDays.value)) : Number(retainChoice.value)))
const daysText = (d: number) => (d >= 365 && d % 365 === 0 ? (d === 365 ? t('monitor.oneYear') : t('monitor.days', { n: d })) : t('monitor.days', { n: d }))

// 与后端分级一致：原始 24 小时 → 1 分钟 7 天 → 10 分钟 90 天 → 1 小时到期满（保留更短时同步截短）
const tiers = computed(() => {
  const days = Math.max(1, retentionDays.value || 1)
  const c = pc.value
  const iv = t('monitor.seconds', { n: draft.intervalSeconds })
  const out = [
    { label: t('monitor.panel.tierRaw', { iv }), keep: t('monitor.panel.keep24h'), color: c.p },
    { label: t('monitor.panel.tier1m'), keep: t('monitor.panel.keep', { d: daysText(Math.min(days, 7)) }), color: c.a2 },
  ]
  if (days > 7) out.push({ label: t('monitor.panel.tier10m'), keep: t('monitor.panel.keep', { d: daysText(Math.min(days, 90)) }), color: c.a })
  if (days > 90) out.push({ label: t('monitor.panel.tier1h'), keep: t('monitor.panel.keep', { d: daysText(days) }), color: c.ok })
  return out
})

// 稳定后的行数 × 每行平均占用
const projected = computed(() => {
  const days = Math.max(1, retentionDays.value || 1)
  const rows =
    86400 / Math.max(1, draft.intervalSeconds) +
    Math.min(days, 7) * 1440 +
    (days > 7 ? Math.min(days, 90) * 144 : 0) +
    (days > 90 ? days * 24 : 0)
  return rows * (num(info.value?.rowBytes) || 700)
})

type MetricKey = 'base' | 'gpu' | 'disk' | 'net' | 'app'
const metrics = computed(() => {
  const gpu = info.value?.gpus?.[0]
  const list: { key: MetricKey; title: string; desc: string; glyph: MonitorGlyphName; tone: string; on: boolean; lock?: boolean }[] = [
    { key: 'base', title: t('monitor.panel.base'), desc: t('monitor.panel.baseDesc'), glyph: 'cpu', tone: 'p', on: true, lock: true },
    {
      key: 'gpu',
      title: t('monitor.panel.gpu'),
      desc: gpu ? t('monitor.panel.gpuFound', { name: gpu.name }) : t('monitor.panel.gpuNone'),
      glyph: 'gpu',
      tone: 'a2',
      on: draft.collectGpu,
    },
    {
      key: 'disk',
      title: t('monitor.panel.disk'),
      desc: props.compact ? t('monitor.panel.diskDescShort') : t('monitor.panel.diskDesc'),
      glyph: 'disk',
      tone: 'ok',
      on: draft.collectDisk,
    },
    { key: 'net', title: t('monitor.panel.net'), desc: t('monitor.panel.netDesc'), glyph: 'net', tone: 'a', on: draft.collectNetwork },
    {
      key: 'app',
      title: t('monitor.panel.app'),
      desc: props.compact ? t('monitor.panel.appDescShort') : t('monitor.panel.appDesc'),
      glyph: 'apps',
      tone: 'p',
      on: draft.collectApps,
    },
  ]
  return list
})
const setMetric = (key: MetricKey, v: boolean) => {
  if (key === 'gpu') draft.collectGpu = v
  else if (key === 'disk') draft.collectDisk = v
  else if (key === 'net') draft.collectNetwork = v
  else if (key === 'app') draft.collectApps = v
}

const onSave = async (close: () => void) => {
  const days = retentionDays.value
  if (!Number.isInteger(days) || days < 1 || days > 3650) {
    feedback.error(t('monitor.panel.customInvalid'))
    return
  }
  saving.value = true
  try {
    const { data } = await updateMonitorSettings({ settings: { ...draft, retentionDays: days } })
    feedback.success(t('monitor.panel.saved'))
    if (data.settings) emit('saved', data.settings)
    close()
  } finally {
    saving.value = false
  }
}

const onExport = async () => {
  exporting.value = true
  try {
    const { data } = await exportMonitorData({ start: props.range.start, end: props.range.end })
    downloadText(data.filename || 'momoko-monitor.csv', data.content || '', 'text/csv;charset=utf-8')
    feedback.success(t('monitor.panel.exported', { n: num(data.rows) }))
  } finally {
    exporting.value = false
  }
}

const onClear = () => {
  Dialog.confirm({
    title: t('monitor.panel.clearTitle'),
    content: t('monitor.panel.clearConfirm'),
    confirmText: t('monitor.panel.clearOk'),
    cancelText: t('monitor.panel.cancel'),
    onConfirm: async () => {
      const { data } = await clearMonitorHistory()
      feedback.success(t('monitor.panel.cleared', { n: num(data.deleted) }))
      await load()
      emit('cleared')
    },
  })
}
</script>

<style scoped lang="scss">
.mset__head {
  display: flex;
  flex: none;
  align-items: center;
  gap: 12px;
  padding: 22px 20px 16px 26px;
  border-bottom: 1px solid var(--el-border-color-light);
}
.mset__ttl {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.mset__title {
  font-family: var(--app-font-display);
  font-size: 1.25rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.mset__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.mset__body {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 22px;
  min-height: 0;
  padding: 20px 26px;
  overflow-y: auto;
  overscroll-behavior: contain;
  scrollbar-width: none;
}
.mset__body::-webkit-scrollbar {
  display: none;
}
.mset__readonly {
  margin: 0;
  padding: 10px 14px;
  border-radius: var(--app-radius);
  background: var(--m-warnSoft);
  color: var(--m-warn);
  font-size: 0.8125rem;
  font-weight: 600;
}
.mset__row,
.mset__metric {
  display: flex;
  align-items: center;
  gap: 12px;
}
.mset__row-t {
  display: flex;
  flex: 1;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.mset__row-title {
  font-size: 0.9375rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.mset__row-desc {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.mset__group {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.mset__lbl {
  font-size: 0.75rem;
  font-weight: 700;
  letter-spacing: 0.08em;
  color: var(--el-text-color-secondary);
}
.mset__hint {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
}
.mset__seg {
  display: flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.mset__seg button {
  flex: 1;
  min-height: 36px;
  padding: 0 10px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}
.mset__seg button:hover:not(:disabled) {
  color: var(--el-text-color-primary);
}
.mset__seg button.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: 0 1px 3px rgba(20, 10, 30, 0.12);
}
.mset__seg button:disabled,
.mset__chip:disabled {
  cursor: not-allowed;
}
.mset__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.mset__chip {
  min-height: 36px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.8125rem;
  font-weight: 600;
  cursor: pointer;
}
.mset__chip:hover:not(:disabled) {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.mset__chip.is-on,
.mset__chip.is-on:hover {
  border-color: var(--m-ink);
  background: var(--m-ink);
  color: var(--m-bg);
}
.mset__days {
  display: flex;
  align-items: center;
  gap: 10px;
  max-width: 220px;
  min-height: 44px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.mset__days:focus-within {
  border-color: var(--m-p);
}
.mset__days input::-webkit-outer-spin-button,
.mset__days input::-webkit-inner-spin-button {
  margin: 0;
  -webkit-appearance: none;
}
.mset__days input {
  flex: 1;
  -moz-appearance: textfield;
  appearance: textfield;
  min-width: 0;
  border: 0;
  background: transparent;
  font: inherit;
  font-size: 0.875rem;
  color: var(--el-text-color-primary);
  outline: none;
}
.mset__tiers {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 14px;
  border-radius: var(--app-radius-lg);
  background: var(--el-fill-color-light);
}
.mset__tiers-t {
  font-size: 0.75rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.mset__tier {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}
.mset__tier i {
  width: 10px;
  height: 10px;
  border-radius: 3px;
}
.mset__tier-l {
  flex: 1;
}
.mset__tier-v {
  font-weight: 700;
}
.mset__store {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-top: 4px;
  padding-top: 10px;
  border-top: 1px solid var(--el-border-color-light);
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}
.mset__store-ic {
  font-size: 16px;
  color: var(--el-text-color-secondary);
}
.mset__store-l {
  flex: 1;
}
.mset__store-r {
  color: var(--el-text-color-secondary);
}
.mset__store-r b {
  color: var(--el-text-color-primary);
}
.mset__group--metrics {
  gap: 0;
}
.mset__group--metrics .mset__lbl {
  margin-bottom: 8px;
}
.mset__metric {
  padding: 10px 0;
  border-top: 1px solid var(--el-border-color-light);
}
.mset__metric-stk {
  width: 32px;
  height: 32px;
  border-radius: 10px;
  font-size: 15px;
}
.mset__metric .mset__row-title {
  font-size: 0.875rem;
}
.mset__metric .mset__row-desc {
  font-size: 0.75rem;
}
.mset__data {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.mset__btn {
  min-height: 36px;
  gap: 6px;
  border-radius: 999px;
  font-weight: 600;
}
.mset__btn-ic {
  font-size: 16px;
}
.mset__foot {
  display: flex;
  flex: none;
  justify-content: flex-end;
  gap: 10px;
  padding: 14px 20px;
  border-top: 1px solid var(--el-border-color-light);
}
.mset__foot-btn {
  min-height: 44px;
  justify-content: center;
  padding-inline: 18px;
  border-radius: 999px;
  font-weight: 600;
}

@media (width <= 768px) {
  .mset__head {
    gap: 8px;
    padding: 6px 8px 10px 20px;
    border-bottom: 0;
  }
  .mset__title {
    font-size: 1.125rem;
  }
  .mset__sub {
    font-size: 0.75rem;
  }
  .mset__body {
    gap: 18px;
    padding: 4px 20px 16px;
  }
  .mset__tiers {
    gap: 7px;
    padding: 12px;
  }
  .mset__tier,
  .mset__store {
    font-size: 0.75rem;
  }
  .mset__metric {
    min-height: 56px;
    padding: 0;
  }
  .mset__metric .mset__row-desc {
    font-size: 0.6875rem;
  }
  .mset__data {
    flex-wrap: nowrap;
    gap: 10px;
  }
  .mset__btn {
    flex: 1;
    min-height: 44px;
    justify-content: center;
  }
  .mset__foot {
    padding: 12px 16px calc(26px + env(safe-area-inset-bottom));
  }
  .mset__foot-btn {
    flex: 1;
  }
  .mset__foot-btn--main {
    flex: 2;
  }
}
</style>
