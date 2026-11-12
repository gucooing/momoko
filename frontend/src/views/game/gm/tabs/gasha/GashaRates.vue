<!-- 「3 概率」：卡池用的抽选表（通常枠 lotteryId / 确定枠 fixLotteryId）与每个槽位的概率、卡组范围、UP 占比。
     概率在配置里是十万分率，这里按百分比（3 位小数）编辑；改一项时不会自动改别的，合计不是 100% 时提示差多少。 -->
<template>
  <section v-if="g && cfg" class="gmc-sec">
    <div class="gmc-sh"><span class="gmc-sn">3</span><span class="gmc-sh__title">{{ t('gmConfig.gasha.sec.rates') }}</span></div>

    <div class="gmc-grid">
      <div class="gmc-field">
        <span class="gmc-label">{{ t('gmConfig.gasha.lotteryId') }}</span>
        <AppSelect v-model="g.lotteryId" :options="lotteryOptions" searchable />
      </div>
      <div class="gmc-field">
        <span class="gmc-label">{{ t('gmConfig.gasha.fixLotteryId') }}</span>
        <AppSelect v-model="g.fixLotteryId" :options="lotteryOptions" searchable />
      </div>
    </div>
    <p class="gmc-hint">{{ t('gmConfig.gasha.lotteryHint') }}</p>

    <div class="gr__panels">
      <div v-for="panel in panels" :key="panel.id" class="gr__panel">
        <header class="gr__head">
          <span class="gr__title">{{ panel.label }}</span>
          <span class="gmc-tag" :class="panel.total === PROBABILITY_TOTAL ? 'gmc-tag--ok' : 'gmc-tag--bad'">
            {{ t('gmConfig.gasha.total', { v: percent(panel.total) }) }}
          </span>
        </header>
        <div v-if="panel.users.length" class="gmc-note">
          <UIcon name="i-lucide-triangle-alert" class="gmc-note__icon" />
          <span class="gmc-note__text">{{ t('gmConfig.gasha.lotteryShared', { gashas: panel.users.join('、') }) }}</span>
          <UButton class="gmc-btn" size="xs" color="neutral" variant="outline" icon="i-lucide-copy-plus" @click="clone(panel.id)">{{ t('gmConfig.gasha.cloneLottery') }}</UButton>
        </div>

        <div v-for="row in panel.rows" :key="row.index" class="gr__slot">
          <div class="gr__slot-main">
            <span class="gmc-rar" :class="`gmc-rar--${row.rarity || 0}`">{{ rarityLabel(row.rarity) }}</span>
            <span class="gr__kind">{{ row.kindLabel }}</span>
            <span class="gr__cards">{{ t('gmConfig.gasha.slotCards', { n: row.cards }) }}</span>
            <span class="gr__prob">
              <input
                class="m-num"
                type="number"
                min="0"
                step="0.001"
                :value="(row.probability / 1000).toFixed(3)"
                :aria-label="`${row.kindLabel} ${rarityLabel(row.rarity)}`"
                @change="setProbability(panel.id, row.index, ($event.target as HTMLInputElement).valueAsNumber)"
              />
              <span>%</span>
            </span>
            <AppIconButton
              icon="HOutline:TrashIcon"
              :label="t('gmConfig.gasha.removeSlot')"
              :box="40"
              :disabled="panel.rows.length <= 1"
              @click="removeSlot(panel.id, row.index)"
            />
          </div>
          <div class="gr__pools">
            <span class="gr__pools-label">{{ t('gmConfig.gasha.slotPools') }}</span>
            <button
              v-for="p in row.poolChoices"
              :key="p.id"
              type="button"
              class="gmc-chip is-soft gr__pool"
              :class="{ 'is-on': p.on }"
              :aria-pressed="p.on"
              @click="togglePool(panel.id, row.index, p.id)"
            >
              <UIcon v-if="p.on" name="i-lucide-check" />
              <span>{{ p.label }}</span>
              <span class="gmc-chip__n m-num">{{ p.size }}</span>
            </button>
            <span v-for="m in row.missing" :key="m" class="gmc-tag gmc-tag--bad">{{ t('gmConfig.gasha.poolMissing') }}</span>
          </div>
          <div v-if="row.pickups.length || row.rest !== null" class="gr__ups">
            <span v-for="p in row.pickups" :key="p.id" class="gmc-tag gmc-tag--g">
              UP {{ p.name }} {{ percent(p.probability) }}% · {{ t('gmConfig.gasha.shareOf', { rarity: rarityLabel(row.rarity), share: p.share }) }}
            </span>
            <span v-if="row.rest !== null" class="gmc-tag" :class="{ 'gmc-tag--bad': row.rest < 0 }">
              {{ t('gmConfig.gasha.restShare', { n: row.others, v: percent(Math.max(0, row.rest)) }) }}
            </span>
          </div>
        </div>

        <div class="gr__foot">
          <span v-if="panel.total !== PROBABILITY_TOTAL" class="gr__diff">
            {{ t('gmConfig.gasha.totalDiff', { v: percent(PROBABILITY_TOTAL - panel.total) }) }}
          </span>
          <span class="gr__spacer" />
          <AppSelect
            v-if="panel.addable.length"
            :model-value="''"
            :options="panel.addable"
            :placeholder="t('gmConfig.gasha.addSlot')"
            fit
            @update:model-value="addSlot(panel.id, String($event))"
          />
        </div>
      </div>
    </div>
    <p class="gmc-hint">{{ t('gmConfig.gasha.ratesHint') }}</p>
  </section>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { GashaLottery } from '@/games/gakumas/types'
import { useGashaEditor } from './editor'
import { useGashaLabels } from './labels'
import {
  freeId,
  lotteryById,
  lotteryTotal,
  otherLotteryUsers,
  percent,
  PROBABILITY_TOTAL,
  rarityLabel,
  slotInfo,
} from './gasha'

const { t } = useI18n()
const ed = useGashaEditor()
const g = computed(() => ed.gasha.value)
const cfg = computed(() => ed.cfg.value)
const labels = useGashaLabels()

const lotteryOptions = computed(() => {
  const c = cfg.value
  return c ? c.lotteries.map((l) => ({ label: labels.lottery(c, l.id), value: l.id })) : []
})

const kindLabel = labels.kind

const panels = computed(() => {
  const x = g.value
  const c = cfg.value
  if (!x || !c) return []
  const ids: { id: string; label: string }[] = []
  if (x.lotteryId) ids.push({ id: x.lotteryId, label: t('gmConfig.gasha.normalSlot') })
  if (x.fixLotteryId && x.fixLotteryId !== x.lotteryId) ids.push({ id: x.fixLotteryId, label: t('gmConfig.gasha.fixSlot') })
  const out = []
  for (const { id, label } of ids) {
    const l = lotteryById(c, id)
    if (!l) continue
    const infos = (l.slots || []).map((s, i) => slotInfo(c, s, i))
    const usedSig = new Set(infos.map((s) => `${s.resourceType}:${s.rarity}`))
    const rows = infos.map((s) => {
      const pickups = (l.pickups || [])
        .filter((p) => p.resourceType === s.resourceType && ed.rarityOf(p.resourceType, p.resourceId) === s.rarity)
        .map((p) => ({
          id: p.resourceId,
          name: labels.card(p.resourceType, p.resourceId),
          probability: p.probability,
          share: s.slot.probability > 0 ? `${((p.probability / s.slot.probability) * 100).toFixed(1)}%` : '—',
        }))
      const picked = new Set(pickups.map((p) => p.id))
      const others = s.cardIds.filter((cid) => !picked.has(cid)).length
      const rest = pickups.length ? s.slot.probability - pickups.reduce((sum, p) => sum + p.probability, 0) : null
      return {
        index: s.index,
        rarity: s.rarity,
        kindLabel: kindLabel(s.resourceType),
        probability: s.slot.probability || 0,
        cards: s.cardIds.length,
        missing: s.missingPools,
        poolChoices: c.pools
          .filter((p) => s.resourceType === undefined || (p.resourceType === s.resourceType && p.rarity === s.rarity))
          .map((p) => ({ id: p.id, label: labels.pool(c, p), size: (p.ids || []).length, on: (s.slot.poolIds || []).includes(p.id) })),
        pickups,
        others,
        rest,
      }
    })
    // 可以加的槽位：这个抽选表还没有的「资源类型 × 稀有度」的卡组
    const addable = c.pools
      .filter((p) => !usedSig.has(`${p.resourceType}:${p.rarity}`))
      .map((p) => ({ label: labels.pool(c, p), value: p.id }))
    out.push({
      id,
      label,
      total: lotteryTotal(l),
      users: otherLotteryUsers(c, id, x.id).map((u) => labels.gasha(u)),
      rows,
      addable,
    })
  }
  return out
})

const lottery = (id: string): GashaLottery | undefined => (cfg.value ? lotteryById(cfg.value, id) : undefined)

const setProbability = (lotteryId: string, index: number, value: number) => {
  const slot = lottery(lotteryId)?.slots?.[index]
  if (!slot) return
  slot.probability = Number.isFinite(value) && value > 0 ? Math.round(value * 1000) : 0
}

const togglePool = (lotteryId: string, index: number, poolId: string) => {
  const slot = lottery(lotteryId)?.slots?.[index]
  if (!slot) return
  if (!slot.poolIds) slot.poolIds = []
  const i = slot.poolIds.indexOf(poolId)
  if (i >= 0) {
    if (slot.poolIds.length > 1) slot.poolIds.splice(i, 1)
  } else slot.poolIds.push(poolId)
}

const addSlot = (lotteryId: string, poolId: string) => {
  const l = lottery(lotteryId)
  if (!l || !poolId) return
  if (!l.slots) l.slots = []
  l.slots.push({ poolIds: [poolId], probability: 0 })
}

const removeSlot = (lotteryId: string, index: number) => {
  const l = lottery(lotteryId)
  if (!l?.slots || l.slots.length <= 1) return
  l.slots.splice(index, 1)
}

// 共用的抽选表复制一份给这个卡池单独用（新抽选表的 ID 用卡池 ID，确定枠加 -fix；只写进配置，界面上不显示）
const clone = (lotteryId: string) => {
  const x = g.value
  const c = cfg.value
  const src = lottery(lotteryId)
  if (!x || !c || !src) return
  const isFix = x.fixLotteryId === lotteryId && x.lotteryId !== lotteryId
  const id = freeId((v) => c.lotteries.some((l) => l.id === v), isFix ? `${x.id}-fix` : x.id)
  const copy = JSON.parse(JSON.stringify(src)) as GashaLottery
  copy.id = id
  c.lotteries.splice(c.lotteries.indexOf(src) + 1, 0, copy)
  if (x.lotteryId === lotteryId) x.lotteryId = id
  if (x.fixLotteryId === lotteryId) x.fixLotteryId = id
}
</script>

<style lang="scss" src="./shared/form.scss"></style>
<style scoped lang="scss">
.gr__panels {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(320px, 100%), 1fr));
  gap: 16px;
}
.gr__panel {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 14px;
  border-radius: 18px;
  background: var(--m-s2);
  min-width: 0;
}
.gr__head {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.gr__title {
  font-size: 14px;
  font-weight: 800;
}
.gr__slot {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
  border-radius: 14px;
  background: var(--el-bg-color);
}
.gr__slot-main {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.gr__kind {
  font-size: 13px;
  font-weight: 700;
}
.gr__cards {
  flex: 1;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gr__prob {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 40px;
  padding: 0 10px;
  border-radius: 12px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  font-weight: 700;
}
.gr__prob:focus-within {
  border-color: var(--m-p);
}
.gr__prob input {
  width: 76px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: right;
}
.gr__pools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 6px;
}
.gr__pools-label {
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.gr__pool {
  padding: 0 12px;
  font-size: 12px;
}
.gr__ups {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.gr__foot {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}
.gr__diff {
  font-size: 12px;
  font-weight: 700;
  color: var(--m-bad);
}
.gr__spacer {
  flex: 1;
}
</style>
