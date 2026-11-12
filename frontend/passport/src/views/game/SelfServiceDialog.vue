<!-- 执行一个自助操作（学マス 各操作对应自己的接口）：按操作显示参数（文本、数字、开关、时间、奖励、道具、要撤回的礼物），
     需要审批时填申请理由，高风险且开启了两步验证时填验证码；提交后显示结果（数值变化或处理摘要）。 -->
<template>
  <PDialog :model-value="!!item" :title="title" :width="hasPicker ? 600 : 480" @update:model-value="onClose">
    <template v-if="item">
      <div v-if="!result" class="sd">
        <p v-if="item.playerNote" class="sd__note">{{ item.playerNote }}</p>
        <template v-for="f in visibleFields" :key="f.key">
          <div v-if="f.type === 'rewards' || f.type === 'item' || f.type === 'gifts'" class="sd__field">
            <span class="sd__label">{{ t(f.labelKey) }}</span>
            <SelfRewardPicker v-if="f.type === 'rewards'" :model-value="model[f.key] as GameReward[]" @update:model-value="(v) => (model[f.key] = v)" />
            <template v-else-if="f.type === 'item'">
              <div v-if="model[f.key]" class="sd__picked">
                <span class="sd__picked-img"><GameImage :src="ctx.resourceImage(RESOURCE_ITEM, String(model[f.key]))" /></span>
                <span class="sd__picked-name">{{ ctx.resourceName(RESOURCE_ITEM, String(model[f.key])) || t('game.unnamed') }}</span>
                <PButton size="sm" variant="ghost" @click="model[f.key] = ''">{{ t('game.self.change') }}</PButton>
              </div>
              <SelfCatalogSearch v-else kind="item" @pick="(e) => (model[f.key] = e.id)" />
            </template>
            <SelfGiftPicker v-else :model-value="model[f.key] as GiftPick" @update:model-value="(v) => (model[f.key] = v)" />
            <span v-if="errors[f.key]" class="sd__error">{{ errors[f.key] }}</span>
          </div>
          <label v-else-if="f.type === 'checkbox'" class="sd__switch">
            <input v-model="model[f.key]" type="checkbox" />
            <span>{{ t(f.labelKey) }}</span>
          </label>
          <PField v-else :label="t(f.labelKey)" :hint="f.hintKey ? t(f.hintKey) : ''" :error="errors[f.key]">
            <input
              v-if="f.type === 'integer'"
              v-model="model[f.key]"
              type="number"
              inputmode="numeric"
              step="1"
              :min="f.min"
              :max="f.max"
              class="pp-num"
            />
            <input v-else-if="f.type === 'datetime'" v-model="model[f.key]" type="datetime-local" />
            <input v-else-if="f.type === 'password'" v-model="model[f.key]" type="password" autocomplete="new-password" />
            <input v-else v-model="model[f.key]" type="text" :maxlength="f.maxLength" />
          </PField>
        </template>
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
          <template v-if="result.changes.length">
            <span class="pp-hint">{{ t('game.self.changes') }}</span>
            <ul>
              <li v-for="(c, i) in result.changes" :key="i">
                <span class="sd__ck">{{ changeLabel(c.kind, c.resourceId) }}</span>
                <span class="pp-num">{{ valueText(c.before) }} → {{ valueText(c.after) }}</span>
              </li>
            </ul>
          </template>
          <span v-else-if="result.summary" class="sd__summary">{{ result.summary }}</span>
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
import GameImage from '@/components/game/GameImage.vue'
import PButton from '@passport/components/PButton.vue'
import PDialog from '@passport/components/PDialog.vue'
import PField from '@passport/components/PField.vue'
import * as gk from '@passport/api/gakumas'
import { errorText, fmtTime } from '@passport/utils/format'
import type { Change } from '@/games/gakumas/types'
import type { GameReward } from '@/types/v1/game_common'
import type { SelfServiceResult } from '@/types/v1/gakumas/passport'
import type { PassportSelfService } from '@/types/v1/passport'
import { useGameContext, valueText } from './context'
import SelfCatalogSearch from './SelfCatalogSearch.vue'
import SelfGiftPicker, { type GiftPick } from './SelfGiftPicker.vue'
import SelfRewardPicker from './SelfRewardPicker.vue'

const props = defineProps<{ item: PassportSelfService | null }>()
const emit = defineEmits<{ close: []; done: [status: string] }>()

const { t, te } = useI18n()
const ctx = useGameContext()

/** 道具的资源类型。 */
const RESOURCE_ITEM = 2

/** 撤回礼物的来源：礼物的 source_kind → 接口的 source_kind。 */
const GIFT_SOURCE: Record<number, string> = { 1: 'gm', 2: 'mail', 3: 'code' }

type FieldType = 'text' | 'password' | 'integer' | 'checkbox' | 'datetime' | 'rewards' | 'item' | 'gifts'

/** 一个自助操作的表单字段。 */
interface FieldDef {
  key: string
  type: FieldType
  labelKey: string
  hintKey?: string
  required?: boolean
  /** 文本长度上限（昵称长度由游戏服务端的配置校验） */
  maxLength?: number
  /** 整数的范围 */
  min?: number
  max?: number
  /** 只在条件满足时显示并提交（封禁到：没勾选永久封禁时） */
  showIf?: (m: Record<string, unknown>) => boolean
}

/** 自助操作的响应：处理情况与我的游戏账号上的结果（失败与待审批时没有 data）。 */
interface SelfServiceResponse {
  result: SelfServiceResult | undefined
  data?: object
}

interface SelfServiceBase {
  serverId: string
  reason: string
  totpCode: string
}

type Params = Record<string, unknown>

const str = (v: unknown) => (typeof v === 'string' ? v.trim() : '')
const int = (v: unknown) => {
  const n = typeof v === 'number' ? v : Number(String(v ?? '').trim())
  return Number.isFinite(n) ? Math.trunc(n) : 0
}
const messageField: FieldDef = { key: 'message', type: 'text', labelKey: 'game.self.giftMessage', maxLength: 64 }

/** 学マス 的自助操作：表单字段与对应的接口（passport/src/api/gakumas.ts）。 */
const OPERATIONS: Record<string, { fields: FieldDef[]; run: (base: SelfServiceBase, p: Params) => Promise<SelfServiceResponse> }> = {
  'gift.send': {
    fields: [
      { key: 'rewards', type: 'rewards', labelKey: 'game.self.rewards', required: true },
      messageField,
      { key: 'expireDays', type: 'integer', labelKey: 'game.self.expireDays', hintKey: 'game.self.expireDaysHint', min: 0, max: 365 },
    ],
    run: (b, p) => gk.sendMyGift({ ...b, rewards: p.rewards as GameReward[], message: str(p.message), expireDays: int(p.expireDays) }),
  },
  'gift.revoke': {
    fields: [{ key: 'gifts', type: 'gifts', labelKey: 'game.self.revokeGifts', required: true }],
    run: (b, p) => {
      const pick = p.gifts as GiftPick
      return gk.revokeMyGifts({ ...b, sourceKind: GIFT_SOURCE[pick.kind] || '', sourceId: '', giftIds: pick.ids })
    },
  },
  'idolCard.grantAll': { fields: [messageField], run: (b, p) => gk.grantMyIdolCards({ ...b, message: str(p.message) }) },
  'supportCard.grantAll': { fields: [messageField], run: (b, p) => gk.grantMySupportCards({ ...b, message: str(p.message) }) },
  'jewel.adjust': {
    fields: [
      { key: 'free', type: 'integer', labelKey: 'game.self.jewelFree', hintKey: 'game.self.deltaHint' },
      { key: 'paid', type: 'integer', labelKey: 'game.self.jewelPaid', hintKey: 'game.self.deltaHint' },
    ],
    run: (b, p) => gk.adjustMyJewel({ ...b, free: int(p.free), paid: int(p.paid) }),
  },
  'item.adjust': {
    fields: [
      { key: 'itemId', type: 'item', labelKey: 'game.self.item', required: true },
      { key: 'quantity', type: 'integer', labelKey: 'game.self.quantityDelta', hintKey: 'game.self.deltaHint', required: true },
    ],
    run: (b, p) => gk.adjustMyItem({ ...b, itemId: str(p.itemId), quantity: int(p.quantity) }),
  },
  'actionPoint.refill': { fields: [], run: (b) => gk.refillMyActionPoint(b) },
  'exp.add': {
    fields: [{ key: 'exp', type: 'integer', labelKey: 'game.self.exp', required: true, min: 1 }],
    run: (b, p) => gk.addMyExp({ ...b, exp: int(p.exp) }),
  },
  'player.kick': { fields: [], run: (b) => gk.kickMe(b) },
  'player.ban': {
    fields: [
      { key: 'permanent', type: 'checkbox', labelKey: 'game.self.banPermanent' },
      { key: 'until', type: 'datetime', labelKey: 'game.self.banUntil', required: true, showIf: (m) => !m.permanent },
      { key: 'banReason', type: 'text', labelKey: 'game.self.banReason', maxLength: 256 },
    ],
    run: (b, p) =>
      gk.banMe({
        ...b,
        permanent: !!p.permanent,
        until: p.permanent ? 0 : new Date(str(p.until)).getTime() || 0,
        banReason: str(p.banReason),
      }),
  },
  'player.unban': { fields: [], run: (b) => gk.unbanMe(b) },
  'player.password': {
    fields: [{ key: 'password', type: 'password', labelKey: 'game.self.password', required: true }],
    run: (b, p) => gk.setMyPassword({ ...b, password: String(p.password ?? '') }),
  },
  'player.rename': {
    fields: [{ key: 'name', type: 'text', labelKey: 'game.self.newName', required: true }],
    run: (b, p) => gk.renameMe({ ...b, name: str(p.name) }),
  },
  'pvp.setRate': {
    fields: [{ key: 'rate', type: 'integer', labelKey: 'game.self.rate', required: true, min: 0 }],
    run: (b, p) => gk.setMyPvpRate({ ...b, rate: int(p.rate) }),
  },
  'pvp.resetDaily': { fields: [], run: (b) => gk.resetMyPvpDaily(b) },
  'seminar.clear': { fields: [], run: (b) => gk.clearMySeminar(b) },
  'seminar.reset': { fields: [], run: (b) => gk.resetMySeminar(b) },
  'archive.export': { fields: [], run: (b) => gk.exportMyArchive(b) },
}

/** 提交后显示的结果。 */
interface Result {
  status: string
  errorMessage: string
  changes: Change[]
  /** 没有数值变化的操作的处理摘要 */
  summary: string
  /** archive.export 导出的存档（解析后的 JSON） */
  export?: unknown
}

const model = reactive<Params>({})
const errors = reactive<Record<string, string>>({})
const reason = ref('')
const totpCode = ref('')
const submitting = ref(false)
const submitError = ref('')
const result = ref<Result | null>(null)

const title = computed(() => {
  const cmd = props.item?.command || ''
  return cmd && te(`game.self.ops.${cmd}`) ? t(`game.self.ops.${cmd}`) : props.item?.title || ''
})
const operation = computed(() => (props.item ? OPERATIONS[props.item.command] : undefined))
const fields = computed(() => operation.value?.fields ?? [])
const visibleFields = computed(() => fields.value.filter((f) => !f.showIf || f.showIf(model)))
const hasPicker = computed(() => fields.value.some((f) => f.type === 'rewards' || f.type === 'item' || f.type === 'gifts'))

const initial = (f: FieldDef): unknown => {
  switch (f.type) {
    case 'checkbox':
      return false
    case 'rewards':
      return []
    case 'gifts':
      return { kind: 0, ids: [] } satisfies GiftPick
    default:
      return ''
  }
}

const reset = () => {
  for (const k of Object.keys(model)) delete model[k]
  for (const k of Object.keys(errors)) delete errors[k]
  for (const f of fields.value) model[f.key] = initial(f)
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

/** 一个字段的错误（没有为空串）。 */
const fieldError = (f: FieldDef): string => {
  const v = model[f.key]
  const name = t(f.labelKey)
  switch (f.type) {
    case 'rewards':
      return f.required && !(v as GameReward[]).length ? t('game.self.required', { field: name }) : ''
    case 'gifts':
      return f.required && !(v as GiftPick).ids.length ? t('game.self.required', { field: name }) : ''
    case 'checkbox':
      return ''
    case 'integer': {
      const raw = String(v ?? '').trim()
      if (!raw) return f.required ? t('game.self.required', { field: name }) : ''
      const n = Number(raw)
      if (!Number.isInteger(n)) return t('game.self.integer', { field: name })
      if (f.min !== undefined && n < f.min) return t('game.self.min', { field: name, n: f.min })
      if (f.max !== undefined && n > f.max) return t('game.self.max', { field: name, n: f.max })
      return ''
    }
    case 'datetime': {
      const raw = str(v)
      if (!raw) return f.required ? t('game.self.required', { field: name }) : ''
      return Number.isFinite(new Date(raw).getTime()) ? '' : t('game.self.required', { field: name })
    }
    case 'password':
      return f.required && !String(v ?? '') ? t('game.self.required', { field: name }) : ''
    default:
      return f.required && !str(v) ? t('game.self.required', { field: name }) : ''
  }
}

const validate = () => {
  for (const k of Object.keys(errors)) delete errors[k]
  let ok = true
  for (const f of visibleFields.value) {
    const e = fieldError(f)
    if (e) {
      errors[f.key] = e
      ok = false
    }
  }
  if (props.item?.needTotp && !totpCode.value.trim()) {
    errors.__totp = t('game.self.required', { field: t('game.self.totpCode') })
    ok = false
  }
  return ok
}

const parseExport = (raw: string): unknown => {
  try {
    return JSON.parse(raw)
  } catch {
    return raw
  }
}

/** 没有数值变化的操作：按返回的内容写一句处理摘要。 */
const summarize = (command: string, data: Record<string, unknown> | undefined): string => {
  if (!data) return ''
  const n = (v: unknown) => Number(v) || 0
  const len = (v: unknown) => (Array.isArray(v) ? v.length : 0)
  switch (command) {
    case 'gift.send':
      return t('game.self.sum.giftSent', { n: len(data.gifts) })
    case 'gift.revoke':
      return t('game.self.sum.giftRevoked', { n: len(data.gifts) })
    case 'idolCard.grantAll':
      return t('game.self.sum.idolCards', { skins: n(data.skinGifts), pieces: n(data.pieceGifts) })
    case 'supportCard.grantAll':
      return t('game.self.sum.supportCards', { n: n(data.gifts) })
    case 'player.kick':
      return data.sessionEnded ? t('game.self.sum.kicked') : t('game.self.sum.notOnline')
    case 'player.ban':
    case 'player.unban':
      return n(data.after) ? t('game.self.sum.bannedUntil', { time: fmtTime(n(data.after)) }) : t('game.self.sum.unbanned')
    case 'player.password':
      return t('game.self.sum.password')
    case 'player.rename':
      return t('game.self.sum.renamed', { before: String(data.before ?? ''), after: String(data.after ?? '') })
    case 'seminar.clear':
      return t('game.self.sum.seminarCleared', { n: len(data.cleared) })
    case 'seminar.reset':
      return t('game.self.sum.seminarReset', { n: n(data.removed) })
    default:
      return ''
  }
}

const toResult = (command: string, r: SelfServiceResponse): Result => {
  const data = r.data as (Record<string, unknown> & { changes?: Change[]; archive?: string }) | undefined
  return {
    status: r.result?.status || '',
    errorMessage: r.result?.errorMessage || '',
    changes: data?.changes ?? [],
    summary: summarize(command, data),
    export: data?.archive !== undefined ? parseExport(data.archive) : undefined,
  }
}

const submit = async () => {
  const item = props.item
  const op = operation.value
  if (!item || !op || !validate()) return
  const params: Params = {}
  for (const f of visibleFields.value) params[f.key] = model[f.key]
  submitting.value = true
  submitError.value = ''
  try {
    const r = await op.run({ serverId: ctx.serverId.value, reason: reason.value.trim(), totpCode: totpCode.value.trim() }, params)
    result.value = toResult(item.command, r)
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

/** 玩家可以下载的数据（archive.export 的存档）。 */
const exportData = computed<unknown>(() => result.value?.export)

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
.sd__field {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.sd__label {
  font-size: 13px;
  font-weight: 700;
}
.sd__error {
  font-size: 12px;
  color: var(--m-bad);
}
.sd__picked {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 10px;
  border-radius: 12px;
  background: var(--m-s2);
}
.sd__picked-img {
  width: 40px;
  height: 40px;
  flex: none;
}
.sd__picked-img :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.sd__picked-name {
  flex: 1;
  min-width: 0;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.sd__switch {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
}
.sd__switch input {
  width: 18px;
  height: 18px;
  accent-color: var(--m-p);
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
.sd__summary {
  font-size: 14px;
  text-align: center;
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
