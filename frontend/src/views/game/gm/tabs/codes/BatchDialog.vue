<!-- 新建 / 编辑兑换码批次（对话框）。
     新建：类型（通用码 / 一次性码）、码或前缀与数量、名称、礼物内容（GmGiftContent：奖励与礼物箱文案，与发送到礼物箱共用）、
     次数、有效期、适用玩家、兑换入口。
     编辑：名称、礼物箱文案、次数、结束时间与兑换入口（码、类型与奖励不能改）。 -->
<template>
  <FormDialog
    :model-value="modelValue"
    :title="batch ? t('gmOps.codes.editTitle') : t('gmOps.codes.newTitle')"
    :width="640"
    :close-on-overlay="false"
    @update:model-value="emit('update:modelValue', $event)"
  >
    <div class="bd">
      <template v-if="!batch">
        <div class="bd__field">
          <span class="bd__label">{{ t('gmOps.codes.kind') }}</span>
          <div class="bd__seg" role="radiogroup" :aria-label="t('gmOps.codes.kind')">
            <button
              v-for="k in kinds"
              :key="k"
              type="button"
              role="radio"
              class="bd__seg-btn"
              :class="{ 'is-on': form.kind === k }"
              :aria-checked="form.kind === k"
              @click="form.kind = k"
            >
              {{ t(`gmOps.codes.kinds.${k}`) }}
            </button>
          </div>
          <span class="bd__hint">{{ t(`gmOps.codes.kindHint.${form.kind}`) }}</span>
        </div>
        <label v-if="form.kind === 'universal'" class="bd__field">
          <span class="bd__label">{{ t('gmOps.codes.code') }}</span>
          <input v-model="form.code" type="text" maxlength="16" class="bd__input m-mono" :placeholder="t('gmOps.codes.codePlaceholder')" @input="form.code = upper(form.code)" />
          <span class="bd__hint">{{ t('gmOps.codes.codeHint') }}</span>
        </label>
        <template v-else>
          <div class="bd__row">
            <label class="bd__field">
              <span class="bd__label">{{ t('gmOps.codes.prefix') }}</span>
              <input v-model="form.prefix" type="text" maxlength="12" class="bd__input m-mono" placeholder="LIVE11" @input="form.prefix = upper(form.prefix)" />
            </label>
            <label class="bd__field">
              <span class="bd__label">{{ t('gmOps.codes.quantity') }}</span>
              <input v-model.number="form.quantity" type="number" min="1" max="10000" class="bd__input m-num" />
            </label>
          </div>
          <span class="bd__hint">{{ t('gmOps.codes.uniqueHint', { sample: sample }) }}</span>
        </template>
      </template>
      <div v-else class="bd__fixed">
        <span class="bd__code m-mono">{{ codeText }}</span>
        <span class="bd__hint">{{ t('gmOps.codes.fixedHint') }}</span>
      </div>

      <label class="bd__field">
        <span class="bd__label">{{ t('gmOps.codes.name') }}</span>
        <input v-model="form.name" type="text" maxlength="64" class="bd__input" :placeholder="t('gmOps.codes.namePlaceholder')" />
      </label>

      <GmGiftContent v-model:rewards="form.rewards" v-model:message="form.message" :max-rewards="rewardMax" :max-message="messageMax" :rewards-locked="!!batch" />

      <div class="bd__row">
        <label v-if="form.kind === 'universal'" class="bd__field">
          <span class="bd__label">{{ t('gmOps.codes.totalLimit') }}</span>
          <input v-model.number="form.totalLimit" type="number" min="0" class="bd__input m-num" :placeholder="t('gmOps.codes.unlimited')" />
          <span class="bd__hint">{{ t('gmOps.codes.totalLimitHint') }}</span>
        </label>
        <label class="bd__field">
          <span class="bd__label">{{ t('gmOps.codes.perPlayer') }}</span>
          <input v-model.number="form.perPlayerLimit" type="number" min="1" max="1000" class="bd__input m-num" />
        </label>
      </div>

      <div class="bd__row">
        <label class="bd__field">
          <span class="bd__label">{{ t('gmOps.codes.startAt') }}</span>
          <input v-model="form.startAt" type="datetime-local" class="bd__input" :disabled="!!batch" />
          <span class="bd__hint">{{ t('gmOps.codes.startHint') }}</span>
        </label>
        <label class="bd__field">
          <span class="bd__label">{{ t('gmOps.codes.endAt') }}</span>
          <input v-model="form.endAt" type="datetime-local" class="bd__input" />
          <span class="bd__hint">{{ t('gmOps.codes.endHint', { zone }) }}</span>
        </label>
      </div>

      <div class="bd__field">
        <span class="bd__label">{{ t('gmOps.codes.eligibility') }}</span>
        <div class="bd__seg" role="radiogroup" :aria-label="t('gmOps.codes.eligibility')">
          <button
            v-for="e in eligibilities"
            :key="e"
            type="button"
            role="radio"
            class="bd__seg-btn"
            :class="{ 'is-on': form.eligibility === e }"
            :aria-checked="form.eligibility === e"
            :disabled="!!batch"
            @click="form.eligibility = e"
          >
            {{ t(`gmOps.codes.eligible.${e}`) }}
          </button>
        </div>
        <label v-if="form.eligibility === 'condition'" class="bd__inline">
          {{ t('gmOps.codes.registeredWithinPre') }}
          <input v-model.number="form.registeredWithinDays" type="number" min="1" max="3650" class="bd__input bd__input--short m-num" :disabled="!!batch" />
          {{ t('gmOps.codes.registeredWithinPost') }}
        </label>
        <span class="bd__hint">{{ t('gmOps.codes.eligibilityHint') }}</span>
      </div>

      <div class="bd__field">
        <span class="bd__label">{{ t('gmOps.codes.channels') }}</span>
        <label class="bd__check">
          <input v-model="form.channels" type="checkbox" value="passport" />
          {{ t('gmOps.codes.channel.passport') }}
        </label>
        <label class="bd__check">
          <input v-model="form.channels" type="checkbox" value="game" />
          {{ t('gmOps.codes.channel.game') }}
        </label>
        <div v-if="form.channels.includes('game')" class="bd__note">
          <UIcon name="i-lucide-info" class="bd__note-icon" />
          <span>{{ t('gmOps.codes.gameUnsupported') }}</span>
        </div>
      </div>
    </div>

    <template #footer>
      <GmRiskPill v-if="!batch && giftRisk" :risk="giftRisk" />
      <span class="bd__spacer" />
      <UButton color="neutral" variant="outline" @click="emit('update:modelValue', false)">{{ t('gmOps.common.cancel') }}</UButton>
      <UButton color="primary" :loading="saving" :disabled="!valid" @click="submit">
        {{ batch ? t('gmOps.codes.saveEdit') : t('gmOps.codes.create') }}
      </UButton>
    </template>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { createGiftCodeBatch, updateGiftCodeBatch } from '@/api/gameOps'
import type { GameReward } from '@/types/v1/game_common'
import type { GiftCodeBatch } from '@/types/v1/game_ops'
import { feedback } from '@/utils/feedback'
import { useGmContext } from '../../context'
import FormDialog from '@/components/ui/FormDialog.vue'
import GmGiftContent from '../../components/GmGiftContent.vue'
import GmRiskPill from '../../components/GmRiskPill.vue'
import { fromLocalInput, localZone, toLocalInput } from '../mail/shared'

type Kind = 'universal' | 'unique'
type Eligibility = 'all' | 'condition'

const props = defineProps<{ modelValue: boolean; batch?: GiftCodeBatch }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; saved: [batch: GiftCodeBatch] }>()

const { t } = useI18n()
const ctx = useGmContext()
const zone = localZone()
const kinds: Kind[] = ['universal', 'unique']
const eligibilities: Eligibility[] = ['all', 'condition']

const giftRisk = computed(() => ctx.operation('gift.send')?.risk || '')
const rewardMax = computed(() => ctx.operation('gift.send')?.params?.properties?.rewards?.maxItems || 20)
const messageMax = computed(() => ctx.operation('gift.send')?.params?.properties?.message?.maxLength || 64)

const blank = () => ({
  kind: 'universal' as Kind,
  code: '',
  prefix: '',
  quantity: 100,
  name: '',
  rewards: [] as GameReward[],
  message: '',
  totalLimit: 0 as number | '',
  perPlayerLimit: 1,
  startAt: '',
  endAt: '',
  eligibility: 'all' as Eligibility,
  registeredWithinDays: 7,
  channels: ['passport'] as string[],
})
const form = reactive(blank())
const saving = ref(false)

watch(
  () => [props.modelValue, props.batch?.id] as const,
  ([open]) => {
    if (!open) return
    const next = blank()
    const b = props.batch
    if (b) {
      next.kind = b.kind === 'unique' ? 'unique' : 'universal'
      next.code = b.code
      next.prefix = b.prefix
      next.quantity = b.quantity
      next.name = b.name
      next.rewards = (b.rewards || []).map((r) => ({ ...r, quantity: Number(r.quantity) }))
      next.message = b.message
      next.totalLimit = b.totalLimit
      next.perPlayerLimit = b.perPlayerLimit || 1
      next.startAt = toLocalInput(b.startAt)
      next.endAt = toLocalInput(b.endAt)
      next.eligibility = b.eligibility === 'condition' ? 'condition' : 'all'
      next.registeredWithinDays = b.registeredWithinDays || 7
      next.channels = [...(b.channels || [])]
    }
    Object.assign(form, next)
  },
  { immediate: true },
)

const upper = (s: string) => s.toUpperCase().replace(/[^A-Z0-9]/g, '')
const sample = computed(() => `${form.prefix ? `${form.prefix}-` : ''}XXXX-XXXX`)
const codeText = computed(() => {
  const b = props.batch
  if (!b) return ''
  return b.kind === 'unique' ? `${b.prefix ? `${b.prefix}-` : ''}····-····` : b.code
})

const valid = computed(() => {
  if (!form.name.trim() || !form.channels.length) return false
  if (!(Number(form.perPlayerLimit) >= 1)) return false
  if (props.batch) return true
  if (!form.rewards.length) return false
  if (form.kind === 'universal' && form.code && !/^[A-Z0-9]{4,16}$/.test(form.code)) return false
  if (form.kind === 'unique' && !(Number(form.quantity) >= 1 && Number(form.quantity) <= 10000)) return false
  if (form.eligibility === 'condition' && !(Number(form.registeredWithinDays) >= 1)) return false
  const start = fromLocalInput(form.startAt)
  const end = fromLocalInput(form.endAt)
  if (start && end && end <= start) return false
  return true
})

const submit = async () => {
  if (!valid.value) return
  saving.value = true
  try {
    const b = props.batch
    if (b) {
      const { data } = await updateGiftCodeBatch({
        serverId: ctx.serverId.value,
        id: b.id,
        name: form.name.trim(),
        endAt: fromLocalInput(form.endAt),
        totalLimit: Number(form.totalLimit) || 0,
        perPlayerLimit: Number(form.perPlayerLimit) || 1,
        channels: form.channels,
        message: form.message.trim(),
      })
      feedback.success(t('gmOps.codes.updated'))
      emit('saved', data)
    } else {
      const { data } = await createGiftCodeBatch({
        serverId: ctx.serverId.value,
        name: form.name.trim(),
        kind: form.kind,
        code: form.kind === 'universal' ? form.code : '',
        prefix: form.kind === 'unique' ? form.prefix : '',
        quantity: form.kind === 'unique' ? Number(form.quantity) : 0,
        rewards: form.rewards,
        message: form.message.trim(),
        totalLimit: form.kind === 'universal' ? Number(form.totalLimit) || 0 : 0,
        perPlayerLimit: Number(form.perPlayerLimit) || 1,
        startAt: fromLocalInput(form.startAt),
        endAt: fromLocalInput(form.endAt),
        eligibility: form.eligibility,
        registeredWithinDays: form.eligibility === 'condition' ? Number(form.registeredWithinDays) : 0,
        channels: form.channels,
      })
      feedback.success(t('gmOps.codes.created', { code: data.kind === 'unique' ? t('gmOps.codes.uniqueCreated', { n: data.quantity }) : data.code }))
      emit('saved', data)
    }
    emit('update:modelValue', false)
  } catch {
    // 错误已由请求层提示
  } finally {
    saving.value = false
  }
}
</script>

<style scoped>
.bd {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.bd__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  flex: 1 1 0;
}
.bd__row {
  display: flex;
  gap: 12px;
  flex-wrap: wrap;
}
.bd__row > .bd__field {
  flex: 1 1 200px;
}
.bd__label {
  font-size: 13px;
  font-weight: 700;
}
.bd__req {
  color: var(--m-bad);
}
.bd__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.bd__input {
  width: 100%;
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius-sm);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 14px;
  outline: none;
  min-width: 0;
}
.bd__input:focus {
  border-color: var(--m-p);
}
.bd__input:disabled {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-secondary);
}
.bd__input--short {
  width: 96px;
}
.bd__textarea {
  padding: 10px 12px;
  resize: vertical;
  line-height: 1.6;
}
.bd__seg {
  display: inline-flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 4px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  align-self: flex-start;
  max-width: 100%;
}
.bd__seg-btn {
  min-height: 36px;
  padding: 0 14px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
.bd__seg-btn.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: 0 1px 3px color-mix(in srgb, var(--m-ink) 12%, transparent);
}
.bd__seg-btn:disabled {
  cursor: default;
}
.bd__inline {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}
.bd__check {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 40px;
  font-size: 14px;
  cursor: pointer;
}
.bd__check input {
  width: 18px;
  height: 18px;
  accent-color: var(--m-g, var(--m-p));
}
.bd__note {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--m-warnSoft);
  font-size: 13px;
  line-height: 1.6;
}
.bd__note-icon {
  flex: none;
  margin-top: 3px;
  color: var(--m-warn);
}
.bd__fixed {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.bd__code {
  align-self: flex-start;
  padding: 6px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--m-gSoft, var(--el-fill-color-light));
  color: var(--m-gInk, var(--el-text-color-primary));
  font-size: 16px;
  font-weight: 800;
  letter-spacing: 0.04em;
}
.bd__spacer {
  flex: 1;
}
@media (width < 768px) {
  .bd__seg-btn,
  .bd__check {
    min-height: 44px;
  }
}
</style>
