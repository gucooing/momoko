<!-- 奖励选择：从 MGP 目录里挑可发放的条目（只列出可发放类型），设数量。
     分类来自 manifest：resourceTypes 里有对应类型的目录才出现；货币在 currency 目录里（作为奖励时 resourceId 为空串）。 -->
<template>
  <div class="grp">
    <div v-if="modelValue.length" class="grp__chosen">
      <div v-for="(r, i) in modelValue" :key="`${r.resourceType}:${r.resourceId}:${i}`" class="grp__item">
        <GameImage v-if="labelOf(r).image" :src="labelOf(r).image" class="grp__img" width="36" height="36" />
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
      <GmCatalogSearch v-if="kind" :kind="kind" grantable-only @pick="add" />
      <div class="grp__hint">{{ t('gm.reward.hint', { max }) }}</div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { gameCatalogImage } from '@/games'
import type { CatalogEntry } from '@/games/mgp'
import { useGmContext } from '../context'
import GmCatalogSearch from './GmCatalogSearch.vue'
import type { GmReward } from './types'

const props = withDefaults(
  defineProps<{
    modelValue: GmReward[]
    /** 最多几项（manifest 里 gift.send 的 rewards.maxItems） */
    max?: number
    /** 不允许的资源类型（manifest resourceTypes 的键，例如邮件不能发 jewelPaid） */
    exclude?: string[]
    disabled?: boolean
  }>(),
  { max: 20, exclude: () => [], disabled: false },
)
const emit = defineEmits<{ 'update:modelValue': [value: GmReward[]] }>()

const { t } = useI18n()
const ctx = useGmContext()

const CURRENCY = 'currency'

const resourceTypes = computed(() => ctx.manifest.value?.resourceTypes || {})
const excludedTypes = computed(() => new Set(props.exclude.map((k) => resourceTypes.value[k]).filter((v) => v !== undefined)))

// 目录分类：manifest.catalogs 里，id 本身是奖励资源类型的目录，加上货币目录。
const kinds = computed(() =>
  (ctx.manifest.value?.catalogs || []).filter((c) => c.id === CURRENCY || c.id in resourceTypes.value),
)
const kind = ref('')
watch(
  kinds,
  (list) => {
    if (!list.some((k) => k.id === kind.value)) kind.value = list[0]?.id || ''
  },
  { immediate: true },
)

// 资源类型 → 目录 id（货币类型都在 currency 目录里）
const kindOfType = (resourceType: number) => {
  const key = Object.keys(resourceTypes.value).find((k) => resourceTypes.value[k] === resourceType)
  if (!key) return ''
  return kinds.value.some((k) => k.id === key) ? key : CURRENCY
}

const labels = reactive(new Map<string, { name: string; image: string }>())
const keyOf = (r: { resourceType: number; resourceId: string }) => `${r.resourceType}:${r.resourceId}`
const pending = new Set<string>()

const labelOf = (r: GmReward) => labels.get(keyOf(r)) || { name: '', image: '' }

const typeKeyOf = (resourceType: number) => Object.keys(resourceTypes.value).find((k) => resourceTypes.value[k] === resourceType) || ''

// 名称：货币按资源类型名（jewelFree …）查 currency 目录，其余按条目 ID 查名称表 / 目录
const nameOf = (r: GmReward) => {
  const k = kindOfType(r.resourceType)
  if (k === CURRENCY || !r.resourceId) return ctx.name(CURRENCY, typeKeyOf(r.resourceType), labelOf(r).name) || t('game.unnamed')
  return ctx.name(k, r.resourceId, labelOf(r).name) || t('game.unnamed')
}

const kindTitle = (r: GmReward) => {
  const k = kindOfType(r.resourceType)
  return k ? kinds.value.find((x) => x.id === k)?.title || '' : ''
}

// 已有奖励（例如编辑草稿）按需查目录补上名称与图片。
const describe = async (r: GmReward) => {
  const key = keyOf(r)
  if (labels.has(key) || pending.has(key)) return
  const k = kindOfType(r.resourceType)
  if (!k) return
  pending.add(key)
  try {
    const page = await ctx.catalog(k, { q: k === CURRENCY ? undefined : r.resourceId, limit: k === CURRENCY ? 20 : 10 })
    const entry = page.items.find((e) =>
      k === CURRENCY ? e.resourceType === r.resourceType : e.id === r.resourceId,
    )
    if (entry) labels.set(key, { name: entry.name, image: gameCatalogImage(ctx.game.value, k, entry) })
  } catch {
    // 查不到时显示「未命名」
  } finally {
    pending.delete(key)
  }
}
watch(() => props.modelValue, (list) => list.forEach((r) => void describe(r)), { immediate: true, deep: true })

const add = (entry: CatalogEntry) => {
  if (excludedTypes.value.has(entry.resourceType)) return
  const reward: GmReward = {
    resourceType: entry.resourceType,
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
