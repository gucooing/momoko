<!-- 奖励选择：从可发放的目录（games/<游戏>/catalog.ts 的 rewardCatalogs）里挑条目，设数量。
     货币在 currency 目录里（作为奖励时 resourceId 为空串）。 -->
<template>
  <div class="grp">
    <div v-if="modelValue.length" class="grp__chosen">
      <div v-for="(r, i) in modelValue" :key="`${r.resourceType}:${r.resourceId}:${i}`" class="grp__item">
        <GameImage v-if="imageOf(r)" :src="imageOf(r)" class="grp__img" width="36" height="36" />
        <span class="grp__text">
          <span class="grp__name">{{ nameOf(r) }}</span>
          <span v-if="kindTitle(r)" class="grp__sub">{{ kindTitle(r) }}</span>
        </span>
        <input
          class="grp__qty m-num"
          type="number"
          min="1"
          :value="r.quantity"
          :aria-label="t('gm.reward.quantity')"
          :disabled="disabled"
          @input="setQuantity(i, ($event.target as HTMLInputElement).valueAsNumber)"
        />
        <AppIconButton icon="HOutline:XMarkIcon" :label="t('gm.reward.remove')" :box="36" :disabled="disabled" @click="remove(i)" />
      </div>
    </div>
    <div v-else class="grp__empty">{{ t('gm.reward.none') }}</div>

    <template v-if="!disabled && modelValue.length < max">
      <div v-if="kinds.length > 1" class="grp__kinds" role="tablist" :aria-label="t('gm.reward.kinds')">
        <button
          v-for="k in kinds"
          :key="k.id"
          type="button"
          role="tab"
          class="grp__kind"
          :class="{ 'is-on': k.id === kind }"
          :aria-selected="k.id === kind"
          @click="kind = k.id"
        >
          {{ k.title }}
        </button>
      </div>
      <GmCatalogSearch v-if="kind" :kind="kind" grantable-only :exclude-ids="pickedIds" @pick="add" />
      <div class="grp__hint">{{ t('gm.reward.hint', { max }) }}</div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { gameCatalogImage } from '@/games'
import { catalogTitles, currencyResourceTypes, resourceRef, rewardCatalogs } from '@/games/gakumas/catalog'
import type { CatalogEntry } from '@/games/mgp'
import { useGmContext } from '../context'
import { catalogLookupFor } from '../player/shared/lookup'
import GmCatalogSearch from './GmCatalogSearch.vue'
import type { GmReward } from './types'

const props = withDefaults(
  defineProps<{
    modelValue: GmReward[]
    /** 最多几项 */
    max?: number
    /** 不允许的资源类型（例如邮件不能发付费钻石 1101） */
    exclude?: number[]
    disabled?: boolean
  }>(),
  { max: 20, exclude: () => [], disabled: false },
)
const emit = defineEmits<{ 'update:modelValue': [value: GmReward[]] }>()

const { t } = useI18n()
const ctx = useGmContext()

const CURRENCY = 'currency'

const kinds = rewardCatalogs.map((id) => ({ id, title: catalogTitles[id] }))
const kind = ref<string>(kinds[0]!.id)

// 刚选上的条目的名称与图片（已有奖励的名称由名称表 / 目录查）
const labels = reactive(new Map<string, { name: string; image: string }>())
const keyOf = (r: { resourceType: number; resourceId: string }) => `${r.resourceType}:${r.resourceId}`
const labelOf = (r: GmReward) => labels.get(keyOf(r)) || { name: '', image: '' }

const lookup = catalogLookupFor(ctx)
const imageOf = (r: GmReward) => {
  const ref = resourceRef(r.resourceType, r.resourceId)
  return labelOf(r).image || (ref ? lookup.image(ref.kind, ref.id) : '')
}

const nameOf = (r: GmReward) => {
  const ref = resourceRef(r.resourceType, r.resourceId)
  if (!ref) return t('game.unnamed')
  return ctx.name(ref.kind, ref.id, labelOf(r).name) || t('game.unnamed')
}

// 当前分类里已经选上的条目：从下面的搜索列表里隐藏（货币条目的 id 是货币名）
const pickedIds = computed(() =>
  props.modelValue
    .map((r) => resourceRef(r.resourceType, r.resourceId))
    .filter((ref) => ref?.kind === kind.value)
    .map((ref) => ref!.id),
)

const kindTitle = (r: GmReward) => {
  const ref = resourceRef(r.resourceType, r.resourceId)
  return ref ? catalogTitles[ref.kind] : ''
}

const resourceTypeOf = (entry: CatalogEntry) => (kind.value === CURRENCY ? currencyResourceTypes[entry.id] || entry.resourceType : entry.resourceType)

const add = (entry: CatalogEntry) => {
  const resourceType = resourceTypeOf(entry)
  if (!resourceType || props.exclude.includes(resourceType)) return
  const reward: GmReward = {
    resourceType,
    resourceId: kind.value === CURRENCY ? '' : entry.id,
    quantity: 1,
  }
  labels.set(keyOf(reward), { name: entry.name, image: gameCatalogImage(ctx.game.value, kind.value, entry) })
  const existing = props.modelValue.findIndex((r) => keyOf(r) === keyOf(reward))
  if (existing >= 0) {
    setQuantity(existing, props.modelValue[existing]!.quantity + 1)
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
.grp {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
}
.grp__chosen {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.grp__item {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 6px 8px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
}
.grp__img {
  flex: none;
  width: 36px;
  height: 36px;
  border-radius: 8px;
}
.grp__text {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.grp__name {
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.grp__sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.grp__qty {
  width: 96px;
  min-height: 36px;
  padding: 0 10px;
  border-radius: var(--app-radius-sm);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
}
.grp__empty,
.grp__hint {
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.grp__kinds {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  scrollbar-width: none;
}
.grp__kinds::-webkit-scrollbar {
  display: none;
}
.grp__kind {
  flex: none;
  min-height: 36px;
  padding: 0 12px;
  border-radius: 999px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
.grp__kind.is-on {
  border-color: transparent;
  background: var(--m-p);
  color: var(--m-pOn);
}
</style>
