<!-- 执行一条自助指令：按参数 JSON Schema 生成简单表单（字符串、数字、开关、枚举），
     需要审批时填申请理由，高风险且开启了两步验证时填验证码；提交后显示结果。 -->
<template>
  <PDialog :model-value="!!item" :title="item?.title || ''" :width="480" @update:model-value="onClose">
    <template v-if="item">
      <div v-if="!result" class="sd">
        <p v-if="item.playerNote" class="sd__note">{{ item.playerNote }}</p>
        <PField
          v-for="f in fields"
          :key="f.key"
          :label="f.label"
          :hint="f.description"
          :error="errors[f.key]"
        >
          <select v-if="f.kind === 'enum'" v-model="model[f.key]">
            <option v-for="o in f.options" :key="String(o)" :value="o">{{ o }}</option>
          </select>
          <span v-else-if="f.kind === 'boolean'" class="sd__switch">
            <PSwitch :model-value="!!model[f.key]" :label="f.label" @update:model-value="(v: boolean) => (model[f.key] = v)" />
            <span class="pp-hint">{{ model[f.key] ? t('game.self.yes') : t('game.self.no') }}</span>
          </span>
          <input
            v-else-if="f.kind === 'number'"
            v-model.number="model[f.key]"
            type="number"
            :min="f.min"
            :max="f.max"
            :step="f.integer ? 1 : 'any'"
          />
          <input v-else v-model="model[f.key]" type="text" :maxlength="f.maxLength" />
        </PField>
        <PField v-if="item.needApproval" :label="t('game.self.reason')" :hint="t('game.self.reasonHint')">
          <textarea v-model="reason" maxlength="200" rows="3" />
        </PField>
        <PField v-if="item.needTotp" :label="t('game.self.totpCode')" :hint="t('game.self.totpHint')" :error="errors.__totp">
          <input v-model="totpCode" type="text" inputmode="numeric" autocomplete="one-time-code" maxlength="10" />
        </PField>
        <p v-if="submitError" class="sd__err">{{ submitError }}</p>
      </div>

      <div v-else class="sd sd--result">
        <span class="sd__badge" :class="`sd__badge--${resultTone}`">
          <component :is="resultIcon" aria-hidden="true" />
        </span>
        <span class="sd__result-t">{{ resultText }}</span>
        <span v-if="result.status === 'failed' && result.errorMessage" class="pp-hint">{{ result.errorMessage }}</span>
        <div v-if="result.status !== 'pending' && result.status !== 'failed'" class="sd__changes">
          <span class="pp-hint">{{ t('game.self.changes') }}</span>
          <ul v-if="result.changes.length">
            <li v-for="(c, i) in result.changes" :key="i">
              <span class="sd__ck">{{ changeLabel(c.kind, c.resourceId) }}</span>
              <span class="pp-num">{{ valueText(c.before) }} → {{ valueText(c.after) }}</span>
            </li>
          </ul>
          <span v-else class="pp-hint">{{ t('game.self.noChanges') }}</span>
        </div>
      </div>
    </template>
    <template #footer>
      <template v-if="item && !result">
        <PButton variant="ghost" @click="onClose(false)">{{ t('common.cancel') }}</PButton>
        <PButton variant="primary" :loading="submitting" @click="submit">
          {{ item.needApproval ? t('game.self.submitApply') : t('game.self.submit') }}
        </PButton>
      </template>
      <template v-else-if="result">
        <PButton v-if="exportData !== undefined" :icon="ArrowDownTrayIcon" @click="download">{{ t('game.self.download') }}</PButton>
        <PButton variant="primary" @click="onClose(false)">{{ t('common.close') }}</PButton>
      </template>
    </template>
  </PDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { ArrowDownTrayIcon, CheckCircleIcon, ClockIcon, XCircleIcon } from '@heroicons/vue/24/outline'
import PButton from '@passport/components/PButton.vue'
import PDialog from '@passport/components/PDialog.vue'
import PField from '@passport/components/PField.vue'
import PSwitch from '@passport/components/PSwitch.vue'
import { executeSelfService } from '@passport/api/passport'
import { errorText } from '@passport/utils/format'
import type { JSONSchema } from '@/games/mgp'
import type { ExecutePassportSelfServiceResponse, PassportSelfService } from '@/types/v1/passport'
import { useGameContext, valueText } from './context'

const props = defineProps<{ item: PassportSelfService | null }>()
const emit = defineEmits<{ close: []; done: [status: string] }>()

const { t, te } = useI18n()
const ctx = useGameContext()

interface FieldDef {
  key: string
  label: string
  description: string
  kind: 'string' | 'number' | 'boolean' | 'enum'
  integer: boolean
  required: boolean
  options: Array<string | number>
  min?: number
  max?: number
  maxLength?: number
}

type FieldValue = string | number | boolean | undefined

const model = reactive<Record<string, FieldValue>>({})
const errors = reactive<Record<string, string>>({})
const reason = ref('')
const totpCode = ref('')
const submitting = ref(false)
const submitError = ref('')
const result = ref<ExecutePassportSelfServiceResponse | null>(null)

const schema = computed(() => (props.item?.params ?? null) as JSONSchema | null)

const fields = computed<FieldDef[]>(() => {
  const s = schema.value
  const required = new Set(s?.required ?? [])
  const out: FieldDef[] = []
  for (const [key, p] of Object.entries(s?.properties ?? {})) {
    if (p['x-picker'] || p.type === 'array' || p.type === 'object') continue
    const kind: FieldDef['kind'] = p.enum?.length
      ? 'enum'
      : p.type === 'boolean'
        ? 'boolean'
        : p.type === 'integer' || p.type === 'number'
          ? 'number'
          : 'string'
    out.push({
      key,
      label: p.title || key,
      description: p.description || '',
      kind,
      integer: p.type === 'integer',
      required: required.has(key),
      options: p.enum ?? [],
      min: p.minimum,
      max: p.maximum,
      maxLength: p.maxLength,
    })
  }
  return out
})

const reset = () => {
  for (const k of Object.keys(model)) delete model[k]
  for (const k of Object.keys(errors)) delete errors[k]
  for (const [key, p] of Object.entries(schema.value?.properties ?? {})) {
    if (p.default !== undefined) model[key] = p.default as FieldValue
    else if (p.type === 'boolean') model[key] = false
    else if (p.enum?.length) model[key] = p.enum[0]
  }
  reason.value = ''
  totpCode.value = ''
  submitError.value = ''
  result.value = null
}

watch(
  () => props.item,
  (item) => {
    if (item) reset()
  },
)

const validate = () => {
  for (const k of Object.keys(errors)) delete errors[k]
  let ok = true
  for (const f of fields.value) {
    const v = model[f.key]
    if (f.required && (v === undefined || v === '' || (typeof v === 'number' && Number.isNaN(v)))) {
      errors[f.key] = t('game.self.required', { field: f.label })
      ok = false
    }
  }
  if (props.item?.needTotp && !totpCode.value.trim()) {
    errors.__totp = t('game.self.required', { field: t('game.self.totpCode') })
    ok = false
  }
  return ok
}

const submit = async () => {
  const item = props.item
  if (!item || !validate()) return
  const params: Record<string, unknown> = {}
  for (const f of fields.value) {
    const v = model[f.key]
    if (v === undefined || v === '') continue
    params[f.key] = v
  }
  submitting.value = true
  submitError.value = ''
  try {
    result.value = await executeSelfService({
      serverId: ctx.serverId.value,
      command: item.command,
      params,
      reason: reason.value.trim(),
      totpCode: totpCode.value.trim(),
    })
    emit('done', result.value.status)
  } catch (e) {
    submitError.value = errorText(e)
  } finally {
    submitting.value = false
  }
}

const resultTone = computed(() => {
  switch (result.value?.status) {
    case 'pending':
      return 'wait'
    case 'failed':
      return 'bad'
    default:
      return 'ok'
  }
})
const resultIcon = computed(() => (resultTone.value === 'wait' ? ClockIcon : resultTone.value === 'bad' ? XCircleIcon : CheckCircleIcon))
const resultText = computed(() => {
  switch (result.value?.status) {
    case 'pending':
      return t('game.self.resultPending')
    case 'failed':
      return t('game.self.resultFailed')
    case 'partial':
      return t('game.self.resultPartial')
    default:
      return t('game.self.resultSuccess')
  }
})

// 一项变化的名称：道具等显示条目名称，货币与等级等显示对应文案，从不显示 ID
const changeLabel = (kind: string, resourceId: string) => {
  if (resourceId) return ctx.nameOf(kind, resourceId) || t('game.unnamed')
  if (te(`game.change.${kind}`)) return t(`game.change.${kind}`)
  return ctx.nameOf('currency', kind) || kind
}

/** 结果 detail 里玩家可以下载的数据（例如 archive.export 的存档）。 */
const exportData = computed<unknown>(() => {
  const detail = result.value?.detail as Record<string, unknown> | null | undefined
  return detail && typeof detail === 'object' && 'export' in detail ? detail.export : undefined
})

const download = () => {
  const blob = new Blob([JSON.stringify(exportData.value, null, 2)], { type: 'application/json' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${props.item?.command || 'export'}-${new Date().toISOString().slice(0, 10)}.json`
  a.click()
  URL.revokeObjectURL(url)
}

const onClose = (open: boolean) => {
  if (!open) emit('close')
}
</script>

<style scoped>
.sd {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.sd__note {
  margin: 0;
  padding: 12px 14px;
  border-radius: 14px;
  background: var(--m-gSoft);
  font-size: 13px;
  line-height: 1.6;
}
.sd__switch {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
}
.sd__err {
  margin: 0;
  color: var(--m-bad);
  font-size: 13px;
}
.sd--result {
  align-items: center;
  text-align: center;
  padding: 8px 0;
}
.sd__badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 56px;
  height: 56px;
  border-radius: 18px;
}
.sd__badge svg {
  width: 30px;
  height: 30px;
}
.sd__badge--ok {
  background: var(--m-okSoft);
  color: var(--m-ok);
}
.sd__badge--bad {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.sd__badge--wait {
  background: var(--m-gSoft);
  color: var(--m-gInk);
}
.sd__result-t {
  font-size: 16px;
  font-weight: 800;
}
.sd__changes {
  align-self: stretch;
  display: flex;
  flex-direction: column;
  gap: 6px;
  text-align: left;
}
.sd__changes ul {
  margin: 0;
  padding: 0;
  list-style: none;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.sd__changes li {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  border-radius: 12px;
  background: var(--m-s2);
  font-size: 13px;
}
.sd__ck {
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
