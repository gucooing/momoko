<!-- 从游戏素材选图：读前端自带的素材包（分组索引按需加载，图片懒加载，一次只渲染一页）。
     选中后返回配置里要写的客户端资源名（assetId）。素材名不显示：能对应到卡池的显示卡池名称，其余只看图。 -->
<template>
  <FormDialog :model-value="modelValue" :title="title" :width="860" :show-footer="false" @update:model-value="emit('update:modelValue', $event)">
    <div class="apd game-skin" :data-game="game || undefined">
      <div class="apd__bar">
        <div v-if="sources.length > 1" class="apd__tabs" role="tablist">
          <button
            v-for="s in sources"
            :key="s.key"
            type="button"
            role="tab"
            class="apd__tab"
            :class="{ 'is-on': s.key === active }"
            :aria-selected="s.key === active"
            @click="active = s.key"
          >
            {{ s.label }}
          </button>
        </div>
        <label class="apd__search">
          <UIcon name="i-lucide-search" class="apd__search-icon" />
          <input v-model="q" type="search" :placeholder="t('gmConfig.assets.search')" :aria-label="t('gmConfig.assets.search')" />
        </label>
      </div>
      <p class="apd__hint">{{ t('gmConfig.assets.hint', { n: filtered.length }) }}</p>
      <div v-if="loading" class="apd__empty">{{ t('gmConfig.common.loading') }}</div>
      <EmptyState v-else-if="!filtered.length" icon="HOutline:PhotoIcon" :title="t('gmConfig.assets.empty')" />
      <div v-else class="apd__grid">
        <button
          v-for="item in shown"
          :key="item.key"
          type="button"
          class="apd__item"
          :class="{ 'is-on': item.assetId === selected }"
          :aria-pressed="item.assetId === selected"
          @click="pick(item.assetId)"
        >
          <GameImage :src="item.url" class="apd__img" />
          <span v-if="item.label" class="apd__name">{{ item.label }}</span>
        </button>
      </div>
      <UButton class="gmc-btn" v-if="shown.length < filtered.length" color="neutral" variant="outline" block @click="limit += PAGE">
        {{ t('gmConfig.common.moreCount', { n: filtered.length - shown.length }) }}
      </UButton>
    </div>
  </FormDialog>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listGameAssets, type GameAssetEntry } from '@/games'
import { useGmContext } from '../../../context'
import type { AssetSource } from './assets'

const props = withDefaults(
  defineProps<{
    modelValue: boolean
    title: string
    sources: AssetSource[]
    /** 当前的 assetId */
    selected?: string
  }>(),
  { selected: '' },
)
const emit = defineEmits<{ 'update:modelValue': [value: boolean]; pick: [assetId: string] }>()

const { t } = useI18n()
const ctx = useGmContext()
const game = computed(() => ctx.game.value)

const PAGE = 48
const active = ref('')
const q = ref('')
const limit = ref(PAGE)
const loading = ref(false)
const groups = shallowRef(new Map<string, GameAssetEntry[]>())

const source = computed(() => props.sources.find((s) => s.key === active.value) || props.sources[0])

watch(
  () => props.modelValue,
  (open) => {
    if (!open) return
    if (!props.sources.some((s) => s.key === active.value)) active.value = props.sources[0]?.key || ''
    q.value = ''
    limit.value = PAGE
  },
  { immediate: true },
)

watch(
  [() => props.modelValue, source],
  async ([open, s]) => {
    if (!open || !s || groups.value.has(s.group)) return
    loading.value = true
    try {
      const list = await listGameAssets(game.value, s.group)
      const next = new Map(groups.value)
      next.set(s.group, list)
      groups.value = next
    } finally {
      loading.value = false
    }
  },
  { immediate: true },
)

watch([q, active], () => {
  limit.value = PAGE
})

const filtered = computed(() => {
  const s = source.value
  if (!s) return []
  const list = groups.value.get(s.group) || []
  const needle = q.value.trim().toLowerCase()
  const prefix = `${s.group}/`
  const out: { key: string; label: string; url: string; assetId: string }[] = []
  for (const e of list) {
    const name = e.key.startsWith(prefix) ? e.key.slice(prefix.length) : e.key
    if (!s.filter(name)) continue
    const label = s.owner ? ctx.name(s.owner.kind, s.owner.id(name)) : ''
    if (needle && !label.toLowerCase().includes(needle)) continue
    out.push({ key: e.key, label, url: e.url, assetId: s.toAssetId(name) })
  }
  return out
})
const shown = computed(() => filtered.value.slice(0, limit.value))

const pick = (assetId: string) => {
  emit('pick', assetId)
  emit('update:modelValue', false)
}
</script>

<style lang="scss" src="./form.scss"></style>
<style scoped lang="scss">
.apd {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.apd__bar {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  align-items: center;
}
.apd__tabs {
  display: flex;
  gap: 6px;
  overflow-x: auto;
  scrollbar-width: none;
}
.apd__tabs::-webkit-scrollbar {
  display: none;
}
.apd__tab {
  flex: none;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 13px;
  font-weight: 700;
  cursor: pointer;
}
.apd__tab.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.apd__search {
  flex: 1 1 220px;
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.apd__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
}
.apd__search-icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.apd__hint {
  margin: 0;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.apd__empty {
  padding: 24px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}
.apd__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 10px;
  max-height: 56vh;
  overflow-y: auto;
  padding: 2px;
}
.apd__item {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 6px;
  border-radius: var(--app-radius);
  border: 2px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.apd__item:hover {
  border-color: var(--m-g, var(--m-p));
}
.apd__item.is-on {
  border-color: var(--m-g, var(--m-p));
  background: var(--m-gSoft, var(--m-pSoft));
}
.apd__item .apd__img {
  width: 100%;
  aspect-ratio: 920 / 320;
  object-fit: contain;
  border-radius: 10px;
  background: var(--el-fill-color-light);
}
.apd__name {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
