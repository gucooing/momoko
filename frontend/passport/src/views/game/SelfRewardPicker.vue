<!-- 自助发礼物的奖励：从可发放的目录（货币、道具、支援卡、偶像卡、歌曲、服装）里挑条目并设数量，最多 max 项。
     货币作为奖励时 resourceId 为空串。 -->
<template>
  <div class="srp">
    <div v-if="modelValue.length" class="srp__chosen">
      <div v-for="(r, i) in modelValue" :key="`${r.resourceType}:${r.resourceId}`" class="srp__item">
        <span class="srp__img"><GameImage :src="ctx.resourceImage(r.resourceType, r.resourceId)" /></span>
        <span class="srp__name">{{ ctx.resourceName(r.resourceType, r.resourceId) || t('game.unnamed') }}</span>
        <input
          class="srp__qty pp-num"
          type="number"
          min="1"
          :value="r.quantity"
          :aria-label="t('game.self.quantity')"
          @input="setQuantity(i, ($event.target as HTMLInputElement).valueAsNumber)"
        />
        <button type="button" class="srp__rm" :aria-label="t('game.self.remove')" @click="remove(i)">
          <XMarkIcon aria-hidden="true" />
        </button>
      </div>
    </div>
    <p v-else class="pp-hint srp__none">{{ t('game.self.noRewards') }}</p>

    <template v-if="modelValue.length < max">
      <div class="pp-scroll-x srp__kinds" role="tablist" :aria-label="t('game.self.rewardKinds')">
        <button
          v-for="k in rewardCatalogs"
          :key="k"
          type="button"
          role="tab"
          class="pp-chip"
          :class="{ 'is-on': k === kind }"
          :aria-selected="k === kind"
          @click="kind = k"
        >
          {{ t(`game.self.kinds.${k}`) }}
        </button>
      </div>
      <SelfCatalogSearch :kind="kind" :exclude-ids="pickedIds" @pick="add" />
    </template>
    <p class="pp-hint srp__hint">{{ t('game.self.rewardsHint', { max }) }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { XMarkIcon } from '@heroicons/vue/24/outline'
import GameImage from '@/components/game/GameImage.vue'
import { currencyResourceTypes, rewardCatalogs, type GakumasCatalogKind } from '@/games/gakumas/catalog'
import type { CatalogEntry } from '@/games/mgp'
import type { GameReward } from '@/types/v1/game_common'
import { useGameContext } from './context'
import SelfCatalogSearch from './SelfCatalogSearch.vue'

const props = withDefaults(defineProps<{ modelValue: GameReward[]; max?: number }>(), { max: 20 })
const emit = defineEmits<{ 'update:modelValue': [value: GameReward[]] }>()

const { t } = useI18n()
const ctx = useGameContext()

const CURRENCY = 'currency'
const kind = ref<GakumasCatalogKind>(rewardCatalogs[0]!)

const keyOf = (r: { resourceType: number; resourceId: string }) => `${r.resourceType}:${r.resourceId}`

// 当前分类里已经选上的条目不再列出（货币条目的 id 是货币名）
const pickedIds = computed(() =>
  props.modelValue
    .map((r) => ctx.resourceRef(r.resourceType, r.resourceId))
    .filter((ref) => ref.kind === kind.value)
    .map((ref) => ref.id),
)

const add = (entry: CatalogEntry) => {
  const resourceType = kind.value === CURRENCY ? currencyResourceTypes[entry.id] || entry.resourceType : entry.resourceType
  if (!resourceType) return
  const reward: GameReward = { resourceType, resourceId: kind.value === CURRENCY ? '' : entry.id, quantity: 1 }
  const i = props.modelValue.findIndex((r) => keyOf(r) === keyOf(reward))
  if (i >= 0) {
    setQuantity(i, props.modelValue[i]!.quantity + 1)
    return
  }
  emit('update:modelValue', [...props.modelValue, reward])
}

const setQuantity = (i: number, value: number) => {
  const next = props.modelValue.slice()
  const cur = next[i]
  if (!cur) return
  next[i] = { ...cur, quantity: Number.isFinite(value) && value >= 1 ? Math.floor(value) : 1 }
  emit('update:modelValue', next)
}

const remove = (i: number) => {
  const next = props.modelValue.slice()
  next.splice(i, 1)
  emit('update:modelValue', next)
}
</script>

<style scoped>
.srp {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.srp__chosen {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.srp__item {
  display: grid;
  grid-template-columns: 36px minmax(0, 1fr) 88px 32px;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  border-radius: 12px;
  background: var(--m-s2);
}
.srp__img {
  width: 36px;
  height: 36px;
}
.srp__img :deep(img) {
  width: 100%;
  height: 100%;
  object-fit: contain;
}
.srp__name {
  font-size: 13px;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.srp__qty {
  width: 100%;
  min-height: 34px;
  padding: 0 8px;
  text-align: right;
}
.srp__rm {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: 0;
  border-radius: 10px;
  background: transparent;
  color: var(--m-ink2);
  cursor: pointer;
}
.srp__rm:hover {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.srp__rm svg {
  width: 18px;
  height: 18px;
}
.srp__kinds {
  display: flex;
  gap: 6px;
}
.srp__none,
.srp__hint {
  margin: 0;
}
</style>
