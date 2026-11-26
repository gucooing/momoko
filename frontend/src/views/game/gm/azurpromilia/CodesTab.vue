<!-- 兑换码：批次列表（搜索批次名或完整的码）、新建批次（通用码 / 一次性码、次数、等级、有效期、兑换后的邮件）、
     批次详情（码与兑换记录，停用 / 恢复，导出全部码）。 -->
<template>
  <div class="azc">
    <ListToolbar v-model:keyword="q" :placeholder="t('azurGm.codes.search')" searchable @search="pages.reset()">
      <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('azurGm.common.refresh')" :box="32" @click="pages.load()" />
      <UButton icon="i-heroicons-plus" @click="startCreate">{{ t('azurGm.codes.create') }}</UButton>
    </ListToolbar>

    <p v-if="pages.error.value" class="azc__error">{{ t('azurGm.common.loadFailed', { msg: pages.error.value }) }}</p>
    <DataTable :columns="columns" :rows="rows" row-key="id" :loading="pages.loading.value" :empty-text="t('azurGm.common.empty')" @row-click="openBatch">
      <template #cell-state="{ row }">
        <StatusPill :variant="row.state === 'active' ? 'success' : row.state === 'pending' ? 'warning' : 'neutral'" :label="t(`azurGm.codes.state.${row.state}`)" />
      </template>
    </DataTable>
    <CursorPager
      :page="pages.page.value"
      :total="pages.total.value"
      :has-prev="pages.hasPrev.value"
      :has-next="pages.hasNext.value"
      :loading="pages.loading.value"
      @prev="pages.prev()"
      @next="pages.next()"
    />

    <FormDialog v-model="creating" :title="t('azurGm.codes.create')" :width="680" :show-footer="false">
      <div class="azc__form">
        <label class="azc__field">
          <span>{{ t('azurGm.codes.name') }}</span>
          <input v-model="form.name" maxlength="64" />
        </label>
        <div class="azc__seg" role="radiogroup">
          <button v-for="k in ['unique', 'universal']" :key="k" type="button" :class="{ 'is-on': form.kind === k }" @click="setKind(k as Kind)">
            {{ t(`azurGm.codes.kind.${k}`) }}
          </button>
        </div>
        <div class="azc__row">
          <label v-if="form.kind === 'universal'" class="azc__field">
            <span>{{ t('azurGm.codes.code') }}</span>
            <input v-model="form.code" maxlength="32" />
          </label>
          <label v-else class="azc__field">
            <span>{{ t('azurGm.codes.count') }}</span>
            <input v-model.number="form.count" type="number" min="1" max="10000" />
          </label>
          <label class="azc__field">
            <span>{{ t('azurGm.codes.codeUses') }}</span>
            <input v-model.number="form.codeUses" type="number" min="0" />
          </label>
          <label class="azc__field">
            <span>{{ t('azurGm.codes.minLevel') }}</span>
            <input v-model.number="form.minLevel" type="number" min="0" />
          </label>
        </div>
        <div class="azc__row">
          <label class="azc__field">
            <span>{{ t('azurGm.codes.startAt') }}</span>
            <input v-model="form.startAt" type="datetime-local" />
          </label>
          <label class="azc__field">
            <span>{{ t('azurGm.codes.endAt') }}</span>
            <input v-model="form.endAt" type="datetime-local" />
          </label>
        </div>
        <p class="azc__hint">{{ t('azurGm.codes.mailHint') }}</p>
        <label class="azc__field">
          <span>{{ t('azurGm.form.title') }}</span>
          <input v-model="form.title" maxlength="64" />
        </label>
        <label class="azc__field">
          <span>{{ t('azurGm.form.sender') }}</span>
          <input v-model="form.sender" maxlength="32" />
        </label>
        <label class="azc__field">
          <span>{{ t('azurGm.form.content') }}</span>
          <textarea v-model="form.content" rows="3" />
        </label>
        <div class="azc__field">
          <span>{{ t('azurGm.form.rewards') }}</span>
          <RewardEditor v-model="form.rewards" />
        </div>
        <label class="azc__field azc__field--short">
          <span>{{ t('azurGm.form.effectDays') }}</span>
          <input v-model.number="form.days" type="number" min="0" max="3650" />
        </label>
        <p v-if="formError" class="azc__error">{{ formError }}</p>
        <div v-if="createdCodes.length" class="azc__created">
          <span>{{ t('azurGm.codes.created', { n: createdCodes.length }) }}</span>
          <UButton size="sm" color="neutral" variant="soft" @click="downloadCodes(form.name, createdCodes)">{{ t('azurGm.codes.export') }}</UButton>
        </div>
        <div class="azc__actions">
          <UButton :loading="saving" :disabled="saving" @click="create">{{ t('azurGm.codes.create') }}</UButton>
        </div>
      </div>
    </FormDialog>

    <FormDialog v-model="viewing" :title="batch?.name || ''" :width="720" :show-footer="false">
      <div v-if="batch" class="azc__form">
        <DescriptionList :items="batchDetail" :columns="2" />
        <div class="azc__actions">
          <UButton color="neutral" variant="soft" :loading="toggling" @click="toggle">
            {{ batch.disabled ? t('azurGm.codes.enable') : t('azurGm.codes.disable') }}
          </UButton>
          <UButton color="neutral" variant="soft" :loading="exporting" @click="exportAll">{{ t('azurGm.codes.export') }}</UButton>
        </div>
        <div class="azc__seg" role="tablist">
          <button type="button" :class="{ 'is-on': detailTab === 'codes' }" @click="detailTab = 'codes'">{{ t('azurGm.codes.codesTab') }}</button>
          <button type="button" :class="{ 'is-on': detailTab === 'redemptions' }" @click="detailTab = 'redemptions'">{{ t('azurGm.codes.redemptionsTab') }}</button>
        </div>
        <template v-if="detailTab === 'codes'">
          <DataTable :columns="codeColumns" :rows="codeRows" row-key="code" :loading="codes.loading.value" :empty-text="t('azurGm.common.empty')" />
          <CursorPager
            :page="codes.page.value"
            :total="codes.total.value"
            :has-prev="codes.hasPrev.value"
            :has-next="codes.hasNext.value"
            :loading="codes.loading.value"
            @prev="codes.prev()"
            @next="codes.next()"
          />
        </template>
        <template v-else>
          <DataTable :columns="redemptionColumns" :rows="redemptionRows" row-key="key" :loading="redemptions.loading.value" :empty-text="t('azurGm.common.empty')" />
          <CursorPager
            :page="redemptions.page.value"
            :total="redemptions.total.value"
            :has-prev="redemptions.hasPrev.value"
            :has-next="redemptions.hasNext.value"
            :loading="redemptions.loading.value"
            @prev="redemptions.prev()"
            @next="redemptions.next()"
          />
        </template>
        <p v-if="formError" class="azc__error">{{ formError }}</p>
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import * as api from '@/api/azurpromilia'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { Code, CodeBatch, Redemption, Reward } from '@/types/v1/azurpromilia/types'
import { feedback } from '@/utils/feedback'
import { useGmContext } from '../context'
import CursorPager from './CursorPager.vue'
import RewardEditor from './RewardEditor.vue'
import { errorMessage, fmtTime, fromInput, rewardsText, useCursorPages } from './shared'

type Kind = 'unique' | 'universal'

const { t } = useI18n()
const ctx = useGmContext()
const q = ref('')

const pages = useCursorPages<CodeBatch>((cursor) => api.listCodeBatches({ serverId: ctx.serverId.value, q: q.value.trim(), cursor, limit: 20 }))

const stateOf = (b: CodeBatch) => {
  const now = Date.now()
  if (b.disabled) return 'disabled'
  if (b.endAt && b.endAt <= now) return 'expired'
  if (b.startAt && b.startAt > now) return 'pending'
  return 'active'
}

const range = (b: CodeBatch) => `${fmtTime(b.startAt)} ~ ${fmtTime(b.endAt)}`

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('azurGm.codes.cols.name'), minWidth: 160, ellipsis: true },
  { key: 'codes', title: t('azurGm.codes.cols.codes'), width: 80, align: 'right' },
  { key: 'used', title: t('azurGm.codes.cols.used'), width: 80, align: 'right' },
  { key: 'redeemed', title: t('azurGm.codes.cols.redeemed'), width: 100, align: 'right' },
  { key: 'time', title: t('azurGm.codes.cols.time'), minWidth: 240 },
  { key: 'state', title: t('azurGm.codes.cols.state'), width: 90 },
  { key: 'operator', title: t('azurGm.codes.cols.operator'), width: 120, ellipsis: true },
])

const rows = computed(() => pages.items.value.map((b) => ({ ...b, time: range(b), state: stateOf(b) })))

// ---------- 新建 ----------
const creating = ref(false)
const saving = ref(false)
const formError = ref('')
const createdCodes = ref<string[]>([])
const form = reactive({
  name: '',
  kind: 'unique' as Kind,
  code: '',
  count: 100,
  codeUses: 1,
  minLevel: 0,
  startAt: '',
  endAt: '',
  title: '',
  sender: '',
  content: '',
  rewards: [] as Reward[],
  days: 30,
})

const setKind = (k: Kind) => {
  form.kind = k
  form.codeUses = k === 'unique' ? 1 : 0
}

const startCreate = () => {
  Object.assign(form, { name: '', kind: 'unique', code: '', count: 100, codeUses: 1, minLevel: 0, startAt: '', endAt: '', title: '', sender: '', content: '', rewards: [], days: 30 })
  formError.value = ''
  createdCodes.value = []
  creating.value = true
}

const create = async () => {
  formError.value = ''
  createdCodes.value = []
  const missing = !form.name.trim() ? t('azurGm.codes.name') : !form.title.trim() ? t('azurGm.form.title') : form.kind === 'universal' && !form.code.trim() ? t('azurGm.codes.code') : ''
  if (missing) {
    formError.value = t('azurGm.form.required', { field: missing })
    return
  }
  saving.value = true
  try {
    const res = await api.createCodes({
      serverId: ctx.serverId.value,
      name: form.name.trim(),
      codes: form.kind === 'universal' ? [form.code.trim()] : [],
      count: form.kind === 'unique' ? Math.floor(form.count || 0) : 0,
      codeUses: Math.max(0, Math.floor(form.codeUses || 0)),
      minLevel: Math.max(0, Math.floor(form.minLevel || 0)),
      startAt: fromInput(form.startAt),
      endAt: fromInput(form.endAt),
      mail: {
        sender: form.sender.trim(),
        title: form.title.trim(),
        content: form.content,
        rewards: form.rewards,
        effect: Math.max(0, Math.floor(form.days || 0)) * 86400,
      },
      dryRun: false,
    })
    createdCodes.value = res.result?.codes || []
    feedback.success(t('azurGm.codes.created', { n: createdCodes.value.length }))
    void pages.reset()
  } catch (e) {
    formError.value = errorMessage(e)
  } finally {
    saving.value = false
  }
}

const downloadCodes = (name: string, list: string[]) => {
  const blob = new Blob([list.join('\n') + '\n'], { type: 'text/plain;charset=utf-8' })
  const a = document.createElement('a')
  a.href = URL.createObjectURL(blob)
  a.download = `${name || 'codes'}.txt`
  a.click()
  URL.revokeObjectURL(a.href)
}

// ---------- 详情 ----------
const viewing = ref(false)
const batch = ref<CodeBatch>()
const detailTab = ref<'codes' | 'redemptions'>('codes')
const toggling = ref(false)
const exporting = ref(false)

const codes = useCursorPages<Code>((cursor) => api.listBatchCodes({ serverId: ctx.serverId.value, id: batch.value?.id || '', cursor, limit: 20 }))
const redemptions = useCursorPages<Redemption>((cursor) =>
  api.listBatchRedemptions({ serverId: ctx.serverId.value, id: batch.value?.id || '', cursor, limit: 20 }),
)

const openBatch = (row: Record<string, unknown>) => {
  batch.value = pages.items.value.find((b) => b.id === row.id)
  detailTab.value = 'codes'
  formError.value = ''
  viewing.value = true
  void codes.reset()
}

watch(detailTab, (v) => {
  if (v === 'redemptions') void redemptions.reset()
})

const batchDetail = computed(() => {
  const b = batch.value
  if (!b) return []
  return [
    { label: t('azurGm.codes.cols.state'), value: t(`azurGm.codes.state.${stateOf(b)}`) },
    { label: t('azurGm.codes.cols.time'), value: range(b) },
    { label: t('azurGm.codes.codeUses'), value: b.codeUses || '∞' },
    { label: t('azurGm.codes.minLevel'), value: b.minLevel || '—' },
    { label: t('azurGm.codes.cols.codes'), value: b.codes },
    { label: t('azurGm.codes.cols.redeemed'), value: b.redeemed },
    { label: t('azurGm.form.title'), value: b.title },
    { label: t('azurGm.form.rewards'), value: rewardsText(b.rewards) },
  ]
})

const codeColumns = computed<DataTableColumn[]>(() => [
  { key: 'code', title: t('azurGm.codes.codesTab'), minWidth: 200 },
  { key: 'usedText', title: t('azurGm.codes.cols.used'), width: 140 },
])
const codeRows = computed(() => codes.items.value.map((c) => ({ ...c, usedText: t('azurGm.codes.usedTimes', { n: c.used }) })))

const redemptionColumns = computed<DataTableColumn[]>(() => [
  { key: 'who', title: t('azurGm.codes.player'), minWidth: 160 },
  { key: 'code', title: t('azurGm.codes.codesTab'), minWidth: 160 },
  { key: 'at', title: t('azurGm.codes.redeemedAt'), width: 160 },
])
const redemptionRows = computed(() =>
  redemptions.items.value.map((r) => ({ key: `${r.pid}:${r.code}`, who: `${r.name || t('azurGm.common.noName')} (${r.displayId})`, code: r.code, at: fmtTime(r.redeemedAt) })),
)

const toggle = async () => {
  if (!batch.value) return
  toggling.value = true
  formError.value = ''
  try {
    const res = await api.setCodeBatchDisabled({ serverId: ctx.serverId.value, id: batch.value.id, disabled: !batch.value.disabled, dryRun: false })
    if (res.batch) batch.value = res.batch
    void pages.load()
  } catch (e) {
    formError.value = errorMessage(e)
  } finally {
    toggling.value = false
  }
}

const exportAll = async () => {
  const b = batch.value
  if (!b) return
  exporting.value = true
  formError.value = ''
  try {
    const all: string[] = []
    let cursor = ''
    do {
      const page = await api.listBatchCodes({ serverId: ctx.serverId.value, id: b.id, cursor, limit: 100 })
      all.push(...page.items.map((c) => c.code))
      cursor = page.nextCursor
    } while (cursor)
    downloadCodes(b.name, all)
  } catch (e) {
    formError.value = errorMessage(e)
  } finally {
    exporting.value = false
  }
}

void pages.reset()
onActivated(() => void pages.load())
</script>

<style scoped lang="scss">
.azc {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.azc__form {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.azc__row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 12px;
}
.azc__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  > span {
    font-size: 13px;
    color: var(--el-text-color-secondary);
  }
  input,
  textarea {
    padding: 6px 10px;
    border: 1px solid var(--el-border-color-light);
    border-radius: var(--app-radius-sm);
    background: var(--el-bg-color);
    color: var(--el-text-color-primary);
    font: inherit;
  }
  input {
    height: 32px;
  }
}
.azc__field--short {
  max-width: 320px;
}
.azc__seg {
  display: inline-flex;
  gap: 4px;
  padding: 3px;
  border-radius: var(--app-radius-pill);
  background: var(--el-fill-color-light);
  align-self: flex-start;
  button {
    padding: 4px 14px;
    border: 0;
    border-radius: var(--app-radius-pill);
    background: transparent;
    color: var(--el-text-color-secondary);
    font: inherit;
    cursor: pointer;
    &.is-on {
      background: var(--el-bg-color);
      color: var(--el-text-color-primary);
    }
  }
}
.azc__actions {
  display: flex;
  gap: 8px;
}
.azc__created {
  display: flex;
  align-items: center;
  gap: 10px;
  font-size: 13px;
}
.azc__hint {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.azc__error {
  margin: 0;
  color: var(--m-bad);
  font-size: 13px;
}
</style>
