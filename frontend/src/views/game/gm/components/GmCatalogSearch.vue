<!-- 目录搜索：按名称（当前语言的译名或游戏原文名）搜索目录（每次最多 30 条，按需请求，不拉全量），点选一条。 -->
<template>
  <div class="gcs">
    <label class="gcs__search">
      <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" class="gcs__icon" />
      <input v-model="q" type="search" :placeholder="placeholder || t('gm.catalog.search')" :aria-label="placeholder || t('gm.catalog.search')" />
    </label>
    <div class="gcs__list" role="listbox">
      <button
        v-for="entry in visible"
        :key="entry.id"
        type="button"
        role="option"
        class="gcs__row"
        :aria-selected="entry.id === selectedId"
        :class="{ 'is-on': entry.id === selectedId }"
        @click="emit('pick', entry)"
      >
        <GameImage v-if="image(entry)" :src="image(entry)" class="gcs__img" width="36" height="36" />
        <span class="gcs__text">
          <span class="gcs__name">{{ entry.name || t('game.unnamed') }}</span>
          <span v-if="entry.originalName" class="gcs__sub">{{ entry.originalName }}</span>
        </span>
      </button>
      <div v-if="loading" class="gcs__hint">{{ t('gm.catalog.loading') }}</div>
      <div v-else-if="error" class="gcs__hint">{{ error }}</div>
      <div v-else-if="!visible.length" class="gcs__hint">{{ t('gm.catalog.empty') }}</div>
      <button v-if="nextCursor && !loading" type="button" class="gcs__more" @click="load(true)">{{ t('gm.catalog.more') }}</button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { gameCatalogImage } from '@/games'
import type { CatalogEntry } from '@/games/mgp'
import { useGmContext } from '../context'

const props = defineProps<{
  kind: string
  selectedId?: string
  placeholder?: string
  characterId?: string
  rarity?: string | number
  /** 只列出可以作为礼物发放的条目（resourceType 非 0） */
  grantableOnly?: boolean
  /** 不列出的条目（已经选上的，避免在下面重复出现） */
  excludeIds?: string[]
}>()
const emit = defineEmits<{ pick: [entry: CatalogEntry] }>()

const { t } = useI18n()
const menuStore = useMenuStore()
const ctx = useGmContext()

const PAGE = 30
const q = ref('')
const items = ref<CatalogEntry[]>([])
const nextCursor = ref('')
const loading = ref(false)
const error = ref('')
let seq = 0

const visible = computed(() => {
  const ex = new Set(props.excludeIds || [])
  return ex.size ? items.value.filter((e) => !ex.has(e.id)) : items.value
})

const image = (entry: CatalogEntry) => gameCatalogImage(ctx.game.value, props.kind, entry)

const load = async (more = false) => {
  const id = ++seq
  loading.value = true
  error.value = ''
  try {
    const page = await ctx.catalog(props.kind, {
      q: q.value.trim() || undefined,
      characterId: props.characterId,
      rarity: Number(props.rarity) || undefined,
      limit: PAGE,
      cursor: more ? nextCursor.value : undefined,
    })
    if (id !== seq) return
    const list = (page.items || []).filter((e) => !props.grantableOnly || e.resourceType > 0)
    items.value = more ? [...items.value, ...list] : list
    nextCursor.value = page.nextCursor || ''
  } catch (e) {
    if (id === seq) error.value = e instanceof Error ? e.message : String(e)
  } finally {
    if (id === seq) loading.value = false
  }
}

const debounced = useDebounceFn(() => load(), 250)
watch(q, () => void debounced())
watch(() => [props.kind, props.characterId, props.rarity], () => void load(), { immediate: true })
</script>

<style scoped>
.gcs {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-width: 0;
}
.gcs__search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 12px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.gcs__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
}
.gcs__icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.gcs__list {
  display: flex;
  flex-direction: column;
  gap: 2px;
  max-height: 280px;
  overflow-y: auto;
}
.gcs__row {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 44px;
  padding: 4px 8px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gcs__row:hover,
.gcs__row.is-on {
  background: var(--m-pSoft);
}
.gcs__img {
  flex: none;
  width: 36px;
  height: 36px;
  border-radius: 8px;
}
.gcs__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.gcs__name {
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gcs__sub {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.gcs__hint {
  padding: 10px 8px;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.gcs__more {
  min-height: 36px;
  border: 0;
  background: transparent;
  color: var(--m-pInk);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}
</style>
