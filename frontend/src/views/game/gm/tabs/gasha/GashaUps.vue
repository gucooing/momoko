<!-- 「2 UP 角色与卡牌」：卡池展示的 UP（pickupIdolCardSkinIds / pickupSupportCardIds，卡池公告页也用它），
     以及这些卡在卡池抽选表里的单独概率（抽选表 pickups，通常枠 / 确定枠各一份）。先选角色再选卡。 -->
<template>
  <section v-if="g && cfg" class="gmc-sec">
    <div class="gmc-sh"><span class="gmc-sn">2</span><span class="gmc-sh__title">{{ t('gmConfig.gasha.sec.ups') }}</span></div>

    <div v-if="sharedNote" class="gmc-note">
      <UIcon name="i-lucide-triangle-alert" class="gmc-note__icon" />
      <span class="gmc-note__text">{{ sharedNote }}</span>
    </div>

    <div class="gu__list">
      <div v-for="u in ups" :key="`${u.resourceType}:${u.id}`" class="gu__card">
        <GameImage :src="u.image" class="gu__img" width="56" height="56" />
        <span class="gu__info">
          <span class="gu__line">
            <span class="gmc-rar" :class="`gmc-rar--${u.rarity || 0}`">{{ rarityLabel(u.rarity) }}</span>
            <span class="gu__name">{{ u.name }}</span>
          </span>
          <span class="gu__sub">
            {{ u.kindLabel }}<template v-if="u.character"> · {{ u.character }}</template>
          </span>
          <span class="gu__rates">
            <label v-for="slot in u.rates" :key="slot.lotteryId" class="gu__rate">
              <span class="gu__rate-label">{{ slot.label }}</span>
              <span class="gu__rate-input">
                <input
                  class="m-num"
                  type="number"
                  min="0"
                  step="0.001"
                  :value="slot.value"
                  :disabled="slot.shared"
                  :aria-label="`${u.name} · ${slot.label}`"
                  @change="setRate(u.resourceType, u.id, slot.lotteryId, ($event.target as HTMLInputElement).value)"
                />
                <span>%</span>
              </span>
              <span v-if="slot.share" class="gu__share">{{ t('gmConfig.gasha.upShare', { rarity: rarityLabel(u.rarity), share: slot.share }) }}</span>
            </label>
          </span>
        </span>
        <AppIconButton icon="HOutline:XMarkIcon" :label="t('gmConfig.gasha.removeUp', { name: u.name })" :box="40" @click="remove(u.resourceType, u.id)" />
      </div>
      <button type="button" class="gmc-chip" :class="{ 'is-on': pickerOpen }" :aria-expanded="pickerOpen" @click="pickerOpen = !pickerOpen">
        <UIcon :name="pickerOpen ? 'i-lucide-x' : 'i-lucide-plus'" />
        {{ pickerOpen ? t('gmConfig.gasha.closePicker') : t('gmConfig.gasha.addUp') }}
      </button>
    </div>
    <p v-if="!ups.length" class="gmc-hint">{{ t('gmConfig.gasha.noUps') }}</p>
    <p class="gmc-hint">{{ t('gmConfig.gasha.upHint') }}</p>

    <GashaCardPicker v-if="pickerOpen" :selected="selectedKeys" @toggle="toggle" />
  </section>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { gakumas } from '@/games'
import type { CatalogEntry } from '@/games/mgp'
import type { GashaLottery } from '@/games/gakumas/types'
import GashaCardPicker from './GashaCardPicker.vue'
import { useGashaEditor } from './editor'
import { useGashaLabels } from './labels'
import { useGmContext } from '../../context'
import {
  catalogKindOf,
  lotteryById,
  otherLotteryUsers,
  rarityLabel,
  RESOURCE_IDOL,
  RESOURCE_SUPPORT,
  slotInfo,
} from './gasha'

const { t } = useI18n()
const ed = useGashaEditor()
const g = computed(() => ed.gasha.value)
const cfg = computed(() => ed.cfg.value)
const labels = useGashaLabels()
const ctx = useGmContext()
const pickerOpen = ref(false)

// 通常枠与确定枠（相同时只有一份）
const lotteries = computed(() => {
  const x = g.value
  const c = cfg.value
  if (!x || !c) return [] as { id: string; label: string; lottery: GashaLottery; shared: boolean }[]
  const out: { id: string; label: string; lottery: GashaLottery; shared: boolean }[] = []
  const add = (id: string, label: string) => {
    const l = id ? lotteryById(c, id) : undefined
    if (l && !out.some((o) => o.id === id)) out.push({ id, label, lottery: l, shared: otherLotteryUsers(c, id, x.id).length > 0 })
  }
  add(x.lotteryId, t('gmConfig.gasha.normalSlot'))
  add(x.fixLotteryId, t('gmConfig.gasha.fixSlot'))
  return out
})

const sharedNote = computed(() => {
  const shared = lotteries.value.filter((l) => l.shared)
  if (!shared.length || !cfg.value || !g.value) return ''
  const users = new Set<string>()
  for (const l of shared) for (const u of otherLotteryUsers(cfg.value, l.id, g.value.id)) users.add(labels.gasha(u))
  return t('gmConfig.gasha.upShared', { lotteries: shared.map((l) => l.label).join('、'), gashas: [...users].join('、') })
})

const pickupOf = (l: GashaLottery, resourceType: number, id: string) =>
  (l.pickups || []).find((p) => p.resourceType === resourceType && p.resourceId === id)

const ups = computed(() => {
  const x = g.value
  const c = cfg.value
  if (!x || !c) return []
  const list = [
    ...(x.pickupIdolCardSkinIds || []).map((id) => ({ resourceType: RESOURCE_IDOL, id })),
    ...(x.pickupSupportCardIds || []).map((id) => ({ resourceType: RESOURCE_SUPPORT, id })),
  ]
  return list.map((u) => {
    const kind = catalogKindOf(u.resourceType)
    const entry = ed.lookup.lookup(kind, u.id)
    const rarity = ed.rarityOf(u.resourceType, u.id)
    return {
      ...u,
      rarity,
      name: labels.card(u.resourceType, u.id),
      character: entry?.characterId ? ctx.label('character', entry.characterId) : '',
      kindLabel: t(`gmConfig.gasha.picker.${kind}`),
      image: u.resourceType === RESOURCE_IDOL ? gakumas.idolCardImage(u.id) : gakumas.supportCardImage(u.id),
      rates: lotteries.value.map((l) => {
        const p = pickupOf(l.lottery, u.resourceType, u.id)
        const slot = (l.lottery.slots || [])
          .map((s, i) => slotInfo(c, s, i))
          .find((s) => s.resourceType === u.resourceType && s.rarity === rarity)
        return {
          lotteryId: l.id,
          label: l.label,
          shared: l.shared,
          value: p ? (p.probability / 1000).toFixed(3) : '',
          share: p && slot && slot.slot.probability > 0 ? `${((p.probability / slot.slot.probability) * 100).toFixed(1)}%` : '',
        }
      }),
    }
  })
})

const selectedKeys = computed(() => ups.value.map((u) => `${u.resourceType}:${u.id}`))

const idsOf = (resourceType: number) => {
  const x = g.value!
  if (resourceType === RESOURCE_IDOL) {
    if (!x.pickupIdolCardSkinIds) x.pickupIdolCardSkinIds = []
    return x.pickupIdolCardSkinIds
  }
  if (!x.pickupSupportCardIds) x.pickupSupportCardIds = []
  return x.pickupSupportCardIds
}

const setRate = (resourceType: number, id: string, lotteryId: string, raw: string) => {
  const c = cfg.value
  if (!c) return
  const l = lotteryById(c, lotteryId)
  if (!l) return
  if (!l.pickups) l.pickups = []
  const value = raw.trim() === '' ? NaN : Number(raw)
  const i = l.pickups.findIndex((p) => p.resourceType === resourceType && p.resourceId === id)
  if (!Number.isFinite(value) || value <= 0) {
    if (i >= 0) l.pickups.splice(i, 1)
    return
  }
  const probability = Math.round(value * 1000)
  if (i >= 0) l.pickups[i]!.probability = probability
  else l.pickups.push({ resourceType: resourceType as 4 | 5, resourceId: id, probability })
}

const remove = (resourceType: number, id: string) => {
  const list = idsOf(resourceType)
  const i = list.indexOf(id)
  if (i >= 0) list.splice(i, 1)
  // 同时去掉本卡池抽选表里这张卡的单独概率（共用的抽选表不动，避免改到别的卡池）
  for (const l of lotteries.value) {
    if (l.shared) continue
    const k = (l.lottery.pickups || []).findIndex((p) => p.resourceType === resourceType && p.resourceId === id)
    if (k >= 0) l.lottery.pickups.splice(k, 1)
  }
}

const toggle = (resourceType: number, entry: CatalogEntry) => {
  ed.lookup.remember(catalogKindOf(resourceType), entry)
  const list = idsOf(resourceType)
  if (list.includes(entry.id)) remove(resourceType, entry.id)
  else list.push(entry.id)
}
</script>

<style lang="scss" src="./shared/form.scss"></style>
<style scoped lang="scss">
.gu__list {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}
.gu__card {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 8px 6px 8px 8px;
  border-radius: 16px;
  background: var(--m-gSoft, var(--m-pSoft));
  border: 1.5px solid var(--m-gLine, var(--el-border-color-light));
  min-width: 0;
  max-width: 100%;
}
.gu__card .gu__img {
  flex: none;
  width: 56px;
  height: 56px;
  border-radius: 12px;
  object-fit: cover;
}
.gu__info {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.gu__line {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.gu__name {
  font-size: 14px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gu__sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  overflow-wrap: anywhere;
}
.gu__rates {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.gu__rate {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 700;
}
.gu__rate-label {
  color: var(--el-text-color-secondary);
}
.gu__rate-input {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 40px;
  padding: 0 10px;
  border-radius: 12px;
  border: 1.5px solid var(--m-gLine, var(--el-border-color-light));
  background: var(--el-bg-color);
}
.gu__rate-input input {
  width: 72px;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-weight: 700;
}
.gu__rate-input input:disabled {
  opacity: 0.6;
}
.gu__share {
  color: var(--el-text-color-secondary);
  font-weight: 600;
}
</style>
