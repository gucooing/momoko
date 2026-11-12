<!-- 「5 交换点」：卡池的交换点数（gashaPointId → points）。每抽到一张卡加 1 点（游戏规则，不可配置），
     可兑换的项目、所需点数与期限；结束后剩余点数按每点 expireReward 折算。 -->
<template>
  <section v-if="g && cfg" class="gmc-sec">
    <div class="gmc-sh">
      <span class="gmc-sn">5</span>
      <span class="gmc-sh__title">{{ t('gmConfig.gasha.sec.point') }}</span>
      <AppSwitch :model-value="!!g.gashaPointId" :aria-label="t('gmConfig.gasha.pointEnable')" @update:model-value="toggle" />
    </div>

    <template v-if="!g.gashaPointId">
      <p class="gmc-hint">{{ t('gmConfig.gasha.pointOff') }}</p>
      <div v-if="choosing" class="gp__choose">
        <AppSelect :model-value="''" :options="pointOptions" :placeholder="t('gmConfig.gasha.pointUseExisting')" searchable @update:model-value="usePoint(String($event))" />
        <UButton class="gmc-btn" color="neutral" variant="outline" icon="i-lucide-plus" @click="createPoint">{{ t('gmConfig.gasha.pointCreate') }}</UButton>
      </div>
    </template>

    <template v-else-if="point">
      <div class="gmc-grid">
        <div class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.gasha.pointId') }}</span>
          <AppSelect :model-value="g.gashaPointId" :options="pointOptions" searchable @update:model-value="usePoint(String($event))" />
        </div>
        <label class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.gasha.pointName') }}</span>
          <input v-model="point.name" class="gmc-input" :class="{ 'is-invalid': !point.name }" />
        </label>
        <div class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.common.start') }}</span>
          <DateTimeField v-model="point.startTime" :zone="ed.zone.value" :invalid="!(point.startTime > 0)" />
        </div>
        <div class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.common.end') }}</span>
          <DateTimeField v-model="point.endTime" :zone="ed.zone.value" :invalid="point.endTime <= point.startTime" />
        </div>
      </div>
      <div v-if="sharedWith" class="gmc-note">
        <UIcon name="i-lucide-triangle-alert" class="gmc-note__icon" />
        <span class="gmc-note__text">{{ t('gmConfig.gasha.pointShared', { gashas: sharedWith }) }}</span>
      </div>
      <div class="gmc-grid gmc-grid--3">
        <div class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.gasha.pointPerDraw') }}</span>
          <span class="gmc-readonly">{{ t('gmConfig.gasha.pointPerDrawValue') }}</span>
        </div>
        <div class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.gasha.pointMin') }}</span>
          <span class="gmc-readonly m-num">{{ minRequire ? `${minRequire} Pt` : '—' }}</span>
        </div>
        <div class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.gasha.pointExpire') }}</span>
          <button type="button" class="gp__reward-btn" @click="openExpire">
            <template v-if="expireView">
              <GameImage v-if="expireView.image" :src="expireView.image" class="gp__reward-img" width="28" height="28" />
              <span class="gp__reward-name">{{ t('gmConfig.gasha.perPoint', { name: expireView.name, n: expireView.quantity }) }}</span>
            </template>
            <span v-else class="gp__reward-name is-missing">{{ t('gmConfig.gasha.pointExpireMissing') }}</span>
            <UIcon name="i-lucide-pencil" class="gp__reward-edit" />
          </button>
        </div>
      </div>

      <div class="gp__ex-head">
        <span class="gmc-label">{{ t('gmConfig.gasha.exchanges', { n: point.exchanges.length }) }}</span>
        <span class="gp__spacer" />
        <UButton class="gmc-btn" v-if="missingUps.length" size="sm" color="neutral" variant="soft" icon="i-lucide-sparkles" @click="addUpExchanges">
          {{ t('gmConfig.gasha.exchangeAddUps', { n: missingUps.length }) }}
        </UButton>
        <UButton class="gmc-btn" size="sm" color="neutral" variant="soft" icon="i-lucide-plus" @click="openExchange(-1)">{{ t('gmConfig.gasha.exchangeAdd') }}</UButton>
      </div>
      <div class="gp__list">
        <div v-for="row in shownExchanges" :key="row.index" class="gp__ex">
          <span class="gp__ex-n m-num">{{ row.x.number }}</span>
          <span class="gp__ex-rewards">
            <span v-for="(r, k) in row.rewards" :key="k" class="gp__ex-reward">
              <GameImage v-if="r.image" :src="r.image" class="gp__reward-img" width="32" height="32" />
              <span class="gp__ex-name">{{ r.name }} ×{{ r.quantity }}</span>
            </span>
            <span v-if="!row.rewards.length" class="gmc-tag gmc-tag--bad">{{ t('gmConfig.gasha.exchangeNoReward') }}</span>
            <span v-if="row.period" class="gmc-hint">{{ row.period }}</span>
          </span>
          <span class="gp__ex-qty">
            <input
              class="m-num"
              type="number"
              min="1"
              step="1"
              :value="row.x.requiredQuantity || ''"
              :aria-label="t('gmConfig.gasha.exchangeRequired')"
              :class="{ 'is-invalid': !(row.x.requiredQuantity > 0) }"
              @change="setRequired(row.index, ($event.target as HTMLInputElement).valueAsNumber)"
            />
            <span>Pt</span>
          </span>
          <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('gmConfig.gasha.exchangeEdit')" :box="40" @click="openExchange(row.index)" />
          <AppIconButton icon="HOutline:XMarkIcon" :label="t('gmConfig.gasha.exchangeRemove')" :box="40" @click="removeExchange(row.index)" />
        </div>
      </div>
      <UButton class="gmc-btn" v-if="exchangeLimit < point.exchanges.length" color="neutral" variant="outline" block @click="exchangeLimit += 10">
        {{ t('gmConfig.common.moreCount', { n: point.exchanges.length - exchangeLimit }) }}
      </UButton>
    </template>
    <p v-else class="gmc-note">
      <UIcon name="i-lucide-triangle-alert" class="gmc-note__icon" />
      <span class="gmc-note__text">{{ t('gmConfig.checks.pointMissing') }}</span>
    </p>

    <FormDialog v-model="expireOpen" :title="t('gmConfig.gasha.pointExpire')" :width="560" :show-footer="false">
      <div class="gp__dialog game-skin" :data-game="ed.game.value || undefined">
        <p class="gmc-hint">{{ t('gmConfig.gasha.pointExpireHint') }}</p>
        <GmRewardPicker v-model="expireDraft" :max="1" />
        <div class="gp__dialog-actions">
          <UButton class="gmc-btn" color="neutral" variant="ghost" @click="expireOpen = false">{{ t('gmConfig.common.cancel') }}</UButton>
          <UButton class="gmc-btn" :disabled="expireDraft.length !== 1" @click="saveExpire">{{ t('gmConfig.common.ok') }}</UButton>
        </div>
      </div>
    </FormDialog>

    <FormDialog v-model="exchangeOpen" :title="exchangeIndex < 0 ? t('gmConfig.gasha.exchangeAdd') : t('gmConfig.gasha.exchangeEdit')" :width="600" :show-footer="false" overflow-visible>
      <div class="gp__dialog game-skin" :data-game="ed.game.value || undefined">
        <div class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.gasha.exchangeRewards') }}</span>
          <GmRewardPicker v-model="exchangeDraft.rewards" :max="10" />
        </div>
        <label class="gmc-field">
          <span class="gmc-label">{{ t('gmConfig.gasha.exchangeRequired') }}</span>
          <span class="gmc-unit">
            <input v-model.number="exchangeDraft.requiredQuantity" class="gmc-input m-num" type="number" min="1" step="1" />
            <span class="gmc-unit__u">Pt</span>
          </span>
        </label>
        <div class="gmc-grid">
          <div class="gmc-field">
            <span class="gmc-label">{{ t('gmConfig.common.start') }}</span>
            <DateTimeField v-model="exchangeDraft.startTime" :zone="ed.zone.value" />
          </div>
          <div class="gmc-field">
            <span class="gmc-label">
              {{ t('gmConfig.common.end') }}
              <span class="gp__perm">
                {{ t('gmConfig.gasha.noEnd') }}
                <AppSwitch :model-value="!exchangeDraft.endTime" @update:model-value="exchangeDraft.endTime = $event ? 0 : point?.endTime || exchangeDraft.startTime" />
              </span>
            </span>
            <DateTimeField v-if="exchangeDraft.endTime" v-model="exchangeDraft.endTime" :zone="ed.zone.value" />
            <span v-else class="gmc-readonly">{{ t('gmConfig.gasha.noEndHint') }}</span>
          </div>
        </div>
        <div class="gp__dialog-actions">
          <UButton class="gmc-btn" color="neutral" variant="ghost" @click="exchangeOpen = false">{{ t('gmConfig.common.cancel') }}</UButton>
          <UButton class="gmc-btn" :disabled="!exchangeDraft.rewards.length || !(exchangeDraft.requiredQuantity > 0)" @click="saveExchange">{{ t('gmConfig.common.ok') }}</UButton>
        </div>
      </div>
    </FormDialog>
  </section>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GashaPoint, GashaPointExchange } from '@/games/gakumas/types'
import GmRewardPicker from '../../components/GmRewardPicker.vue'
import type { GmReward } from '../../components/types'
import { useGashaEditor } from './editor'
import { useGashaLabels } from './labels'
import { freeId, otherPointUsers, RESOURCE_IDOL, RESOURCE_SUPPORT } from './gasha'
import { mergeRewards, rewardView, toGmRewards } from './rewards'
import DateTimeField from './shared/DateTimeField.vue'
import { formatShort } from './shared/time'

const { t } = useI18n()
const ed = useGashaEditor()
const labels = useGashaLabels()
const g = computed(() => ed.gasha.value)
const cfg = computed(() => ed.cfg.value)

const choosing = ref(false)
const point = computed(() => (g.value?.gashaPointId ? cfg.value?.points.find((p) => p.id === g.value!.gashaPointId) : undefined))
// 交换点按名称选；同名时用「哪些卡池在用」区分
const pointOptions = computed(() => {
  const c = cfg.value
  if (!c) return []
  return c.points.map((p) => {
    const users = c.gashas.filter((x) => x.gashaPointId === p.id).map((x) => labels.gasha(x))
    return {
      label: p.name || t('game.unnamed'),
      value: p.id,
      description: users.length ? t('gmConfig.gasha.usedBy', { gashas: users.join('、') }) : t('gmConfig.gasha.unused'),
    }
  })
})
const sharedWith = computed(() =>
  point.value && cfg.value && g.value
    ? otherPointUsers(cfg.value, point.value.id, g.value.id)
        .map((x) => labels.gasha(x))
        .join('、')
    : '',
)
const minRequire = computed(() => {
  const list = (point.value?.exchanges || []).map((x) => x.requiredQuantity).filter((v) => v > 0)
  return list.length ? Math.min(...list) : 0
})

const toggle = (on: boolean) => {
  const x = g.value
  if (!x) return
  if (!on) {
    x.gashaPointId = ''
    choosing.value = false
    return
  }
  choosing.value = true
}

const usePoint = (id: string) => {
  if (!g.value || !id) return
  g.value.gashaPointId = id
  choosing.value = false
}

const createPoint = () => {
  const x = g.value
  const c = cfg.value
  if (!x || !c) return
  const id = freeId((v) => c.points.some((p) => p.id === v), `gasha_point-${x.id}`)
  const p: GashaPoint = {
    id,
    name: '',
    conditionSetId: '',
    startTime: x.startTime,
    endTime: x.endTime,
    expireReward: { resourceType: 0, resourceId: '', quantity: 0 },
    exchanges: [],
  }
  c.points.push(p)
  x.gashaPointId = id
  choosing.value = false
}

// ---- 折算奖励 ----
const expireView = computed(() => {
  const r = point.value?.expireReward
  return r && (r.resourceType || 0) > 0 ? rewardView(r, ed.lookup) : undefined
})
const expireOpen = ref(false)
const expireDraft = ref<GmReward[]>([])
const openExpire = () => {
  const r = point.value?.expireReward
  expireDraft.value = r && (r.resourceType || 0) > 0 ? toGmRewards([r]) : []
  expireOpen.value = true
}
const saveExpire = () => {
  const p = point.value
  if (!p) return
  const next = mergeRewards(p.expireReward && (p.expireReward.resourceType || 0) > 0 ? [p.expireReward] : [], expireDraft.value)[0]
  if (next) p.expireReward = next
  expireOpen.value = false
}

// ---- 兑换项 ----
const exchangeLimit = ref(10)
const shownExchanges = computed(() => {
  const p = point.value
  if (!p) return []
  return (p.exchanges || []).slice(0, exchangeLimit.value).map((x, index) => ({
    index,
    x,
    rewards: (x.rewards || []).map((r) => rewardView(r, ed.lookup)),
    period:
      x.startTime === p.startTime && x.endTime === p.endTime
        ? ''
        : `${formatShort(x.startTime, ed.zone.value)} → ${x.endTime ? formatShort(x.endTime, ed.zone.value) : t('gmConfig.gasha.noEnd')}`,
  }))
})

const renumber = (p: GashaPoint) => p.exchanges.forEach((x, i) => (x.number = i + 1))

const setRequired = (index: number, value: number) => {
  const x = point.value?.exchanges?.[index]
  if (x) x.requiredQuantity = Number.isFinite(value) && value > 0 ? Math.trunc(value) : 0
}

const removeExchange = (index: number) => {
  const p = point.value
  if (!p) return
  p.exchanges.splice(index, 1)
  renumber(p)
}

// UP 卡还不能用交换点换的，各加一项（所需点数需要填）
const missingUps = computed(() => {
  const x = g.value
  const p = point.value
  if (!x || !p) return []
  const has = (type: number, id: string) => p.exchanges.some((e) => (e.rewards || []).some((r) => r.resourceType === type && r.resourceId === id))
  return [
    ...(x.pickupIdolCardSkinIds || []).map((id) => ({ type: RESOURCE_IDOL, id })),
    ...(x.pickupSupportCardIds || []).map((id) => ({ type: RESOURCE_SUPPORT, id })),
  ].filter((u) => !has(u.type, u.id))
})
const addUpExchanges = () => {
  const p = point.value
  if (!p) return
  for (const u of missingUps.value) {
    p.exchanges.push({
      number: p.exchanges.length + 1,
      requiredQuantity: 0,
      rewards: [{ resourceType: u.type, resourceId: u.id, quantity: 1 }],
      startTime: p.startTime,
      endTime: p.endTime,
    })
  }
  exchangeLimit.value = Math.max(exchangeLimit.value, p.exchanges.length)
}

const exchangeOpen = ref(false)
const exchangeIndex = ref(-1)
const exchangeDraft = reactive({ rewards: [] as GmReward[], requiredQuantity: 0, startTime: 0, endTime: 0 })
const openExchange = (index: number) => {
  const p = point.value
  if (!p) return
  const x = index >= 0 ? p.exchanges[index] : undefined
  exchangeIndex.value = index
  exchangeDraft.rewards = toGmRewards(x?.rewards)
  exchangeDraft.requiredQuantity = x?.requiredQuantity || 0
  exchangeDraft.startTime = x?.startTime || p.startTime
  exchangeDraft.endTime = x ? x.endTime : p.endTime
  exchangeOpen.value = true
}
const saveExchange = () => {
  const p = point.value
  if (!p) return
  const x: GashaPointExchange | undefined = exchangeIndex.value >= 0 ? p.exchanges[exchangeIndex.value] : undefined
  if (x) {
    x.rewards = mergeRewards(x.rewards, exchangeDraft.rewards)
    x.requiredQuantity = Math.trunc(exchangeDraft.requiredQuantity)
    x.startTime = exchangeDraft.startTime
    x.endTime = exchangeDraft.endTime
  } else {
    p.exchanges.push({
      number: p.exchanges.length + 1,
      requiredQuantity: Math.trunc(exchangeDraft.requiredQuantity),
      rewards: mergeRewards([], exchangeDraft.rewards),
      startTime: exchangeDraft.startTime,
      endTime: exchangeDraft.endTime,
    })
    exchangeLimit.value = Math.max(exchangeLimit.value, p.exchanges.length)
  }
  exchangeOpen.value = false
}
</script>

<style lang="scss" src="./shared/form.scss"></style>
<style scoped lang="scss">
.gp__choose {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}
.gp__choose > :first-child {
  flex: 1 1 260px;
}
.gp__reward-btn {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 12px;
  border-radius: var(--app-radius);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  min-width: 0;
}
.gp__reward-btn:hover {
  border-color: var(--m-p);
}
.gp__reward-btn .gp__reward-img,
.gp__ex .gp__reward-img {
  flex: none;
  width: 28px;
  height: 28px;
  border-radius: 6px;
  object-fit: contain;
}
.gp__reward-name {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gp__reward-name.is-missing {
  color: var(--m-bad);
}
.gp__reward-edit {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.gp__ex-head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.gp__spacer {
  flex: 1;
}
.gp__list {
  display: flex;
  flex-direction: column;
}
.gp__ex {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 0;
  border-top: 1px solid var(--el-border-color-lighter);
}
.gp__ex-n {
  flex: none;
  width: 28px;
  font-weight: 800;
  color: var(--el-text-color-secondary);
}
.gp__ex-rewards {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px 12px;
}
.gp__ex-reward {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.gp__ex-name {
  font-size: 13px;
  font-weight: 700;
  overflow-wrap: anywhere;
}
.gp__ex-qty {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 40px;
  padding: 0 10px;
  border-radius: 12px;
  border: 1.5px solid var(--el-border-color-light);
  font-size: 12px;
  font-weight: 700;
}
.gp__ex-qty input {
  width: 64px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: right;
}
.gp__ex-qty input.is-invalid {
  color: var(--m-bad);
}
.gp__dialog {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.gp__dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
.gp__perm {
  margin-left: auto;
  display: inline-flex;
  align-items: center;
  gap: 8px;
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
@media (max-width: 720px) {
  .gp__ex {
    flex-wrap: wrap;
  }
  .gp__ex-rewards {
    flex-basis: calc(100% - 40px);
  }
}
</style>
