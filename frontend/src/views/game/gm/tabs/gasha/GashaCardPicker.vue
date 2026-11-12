<!-- 选 UP 卡：偶像卡先选角色、再选这个角色的卡；支援卡按稀有度与名称搜索。
     卡来自 MGP 目录（idolCardSkin / supportCard），每次只取一页（搜索 + 分页），图片懒加载。 -->
<template>
  <div class="gcp">
    <div class="gcp__bar">
      <div class="gmc-seg" role="radiogroup" :aria-label="t('gmConfig.gasha.picker.kind')">
        <button
          v-for="k in KINDS"
          :key="k"
          type="button"
          role="radio"
          class="gmc-seg__btn"
          :class="{ 'is-on': kind === k }"
          :aria-checked="kind === k"
          @click="kind = k"
        >
          {{ t(`gmConfig.gasha.picker.${k}`) }}
        </button>
      </div>
      <div class="gmc-seg" role="radiogroup" :aria-label="t('gmConfig.gasha.picker.rarity')">
        <button
          v-for="r in RARITIES"
          :key="r"
          type="button"
          role="radio"
          class="gmc-seg__btn"
          :class="{ 'is-on': rarity === r }"
          :aria-checked="rarity === r"
          @click="rarity = r"
        >
          {{ rarityLabel(r) }}
        </button>
      </div>
      <label class="gcp__search">
        <UIcon name="i-lucide-search" class="gcp__search-icon" />
        <input v-model="q" type="search" :placeholder="t('gmConfig.gasha.picker.search')" :aria-label="t('gmConfig.gasha.picker.search')" />
      </label>
    </div>

    <div v-if="kind === 'idolCardSkin'" class="gcp__chars" role="radiogroup" :aria-label="t('gmConfig.gasha.picker.character')">
      <button
        v-for="c in characters"
        :key="c.id"
        type="button"
        role="radio"
        class="gcp__char"
        :class="{ 'is-on': c.id === characterId }"
        :aria-checked="c.id === characterId"
        @click="characterId = c.id"
      >
        <GameImage :src="gakumas.charaImage(c.id)" class="gcp__char-img" width="44" height="44" />
        <span class="gcp__char-name">{{ c.name }}</span>
      </button>
      <span v-if="charactersLoading" class="gmc-hint">{{ t('gmConfig.common.loading') }}</span>
      <button v-else-if="charactersCursor" type="button" class="gcp__char gcp__char--more" @click="loadCharacters(true)">
        <UIcon name="i-lucide-plus" class="gcp__char-more" />
        {{ t('gmConfig.common.more') }}
      </button>
    </div>

    <p v-if="kind === 'idolCardSkin' && !characterId" class="gmc-hint">{{ t('gmConfig.gasha.picker.pickCharacter') }}</p>
    <template v-else>
      <div v-if="error" class="gmc-hint">{{ error }}</div>
      <div v-else-if="!items.length && !loading" class="gmc-hint">{{ t('gmConfig.gasha.picker.empty') }}</div>
      <div class="gcp__grid">
        <button
          v-for="e in items"
          :key="e.id"
          type="button"
          class="gcp__card"
          :class="{ 'is-on': isSelected(e) }"
          :aria-pressed="isSelected(e)"
          @click="toggle(e)"
        >
          <GameImage :src="gameCatalogImage(game, kind, e)" class="gcp__card-img" />
          <span v-if="isSelected(e)" class="gcp__check"><UIcon name="i-lucide-check" /></span>
          <span class="gcp__card-name">{{ e.name || t('game.unnamed') }}</span>
          <span v-if="e.originalName" class="gcp__card-sub">{{ e.originalName }}</span>
        </button>
      </div>
      <div v-if="loading" class="gmc-hint">{{ t('gmConfig.common.loading') }}</div>
      <UButton class="gmc-btn" v-else-if="nextCursor" color="neutral" variant="outline" block @click="load(true)">{{ t('gmConfig.common.more') }}</UButton>
    </template>
    <p class="gmc-hint">{{ kind === 'idolCardSkin' ? t('gmConfig.gasha.picker.idolHint') : t('gmConfig.gasha.picker.supportHint') }}</p>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { gakumas, gameCatalogImage } from '@/games'
import type { CatalogEntry } from '@/games/mgp'
import { useGmContext } from '../../context'
import { rarityLabel, RESOURCE_IDOL, RESOURCE_SUPPORT } from './gasha'

const props = defineProps<{
  /** 已选的卡（`${resourceType}:${id}`） */
  selected: string[]
}>()
const emit = defineEmits<{ toggle: [resourceType: number, entry: CatalogEntry] }>()

const KINDS = ['idolCardSkin', 'supportCard'] as const
const RARITIES = [3, 2, 1] as const
const PAGE = 24

const { t } = useI18n()
const ctx = useGmContext()
const game = computed(() => ctx.game.value)

const kind = ref<(typeof KINDS)[number]>('idolCardSkin')
const rarity = ref<number>(3)
const q = ref('')
const characterId = ref('')
const characters = ref<CatalogEntry[]>([])
const charactersLoading = ref(false)
const charactersCursor = ref('')
const items = ref<CatalogEntry[]>([])
const nextCursor = ref('')
const loading = ref(false)
const error = ref('')
let seq = 0

const resourceType = computed(() => (kind.value === 'idolCardSkin' ? RESOURCE_IDOL : RESOURCE_SUPPORT))
const isSelected = (e: CatalogEntry) => props.selected.includes(`${resourceType.value}:${e.id}`)
const toggle = (e: CatalogEntry) => emit('toggle', resourceType.value, e)

const loadCharacters = async (more = false) => {
  if (!more && characters.value.length) return
  charactersLoading.value = true
  try {
    const page = await ctx.catalog('character', { limit: 50, cursor: more ? charactersCursor.value : undefined })
    characters.value = more ? [...characters.value, ...(page.items || [])] : page.items || []
    charactersCursor.value = page.nextCursor || ''
  } catch (e) {
    error.value = e instanceof Error ? e.message : String(e)
  } finally {
    charactersLoading.value = false
  }
}

const load = async (more = false) => {
  if (kind.value === 'idolCardSkin' && !characterId.value) {
    items.value = []
    nextCursor.value = ''
    return
  }
  const id = ++seq
  loading.value = true
  error.value = ''
  try {
    const page = await ctx.catalog(kind.value, {
      q: q.value.trim() || undefined,
      characterId: kind.value === 'idolCardSkin' ? characterId.value : undefined,
      rarity: rarity.value,
      limit: PAGE,
      cursor: more ? nextCursor.value : undefined,
    })
    if (id !== seq) return
    items.value = more ? [...items.value, ...(page.items || [])] : page.items || []
    nextCursor.value = page.nextCursor || ''
  } catch (e) {
    if (id === seq) error.value = e instanceof Error ? e.message : String(e)
  } finally {
    if (id === seq) loading.value = false
  }
}

const debounced = useDebounceFn(() => load(), 250)
watch(q, () => void debounced())
watch([kind, rarity, characterId], () => void load(), { immediate: true })
watch(
  kind,
  (k) => {
    if (k === 'idolCardSkin') void loadCharacters()
  },
  { immediate: true },
)
</script>

<style lang="scss" src="./shared/form.scss"></style>
<style scoped lang="scss">
.gcp {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  border-radius: 20px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--m-bg);
  min-width: 0;
}
.gcp__bar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.gcp__search {
  flex: 1 1 200px;
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  background: var(--el-bg-color);
  border: 1.5px solid var(--el-border-color-light);
}
.gcp__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
}
.gcp__search-icon {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.gcp__chars {
  display: flex;
  gap: 2px;
  overflow-x: auto;
  scrollbar-width: none;
}
.gcp__chars::-webkit-scrollbar {
  display: none;
}
.gcp__char {
  flex: none;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  width: 68px;
  padding: 6px 0;
  border: 0;
  border-radius: 14px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 11px;
  font-weight: 700;
  cursor: pointer;
}
.gcp__char:hover {
  background: var(--m-s2);
}
.gcp__char .gcp__char-img {
  width: 44px;
  height: 44px;
  border-radius: 50%;
  border: 2px solid transparent;
}
.gcp__char.is-on {
  color: var(--el-text-color-primary);
}
.gcp__char.is-on .gcp__char-img {
  border-color: var(--m-g, var(--m-p));
}
.gcp__char-more {
  width: 44px;
  height: 44px;
  padding: 12px;
  border-radius: 50%;
  background: var(--m-s2);
}
.gcp__char-name {
  max-width: 64px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gcp__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(118px, 1fr));
  gap: 10px;
}
.gcp__card {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 6px;
  border-radius: 16px;
  border: 2px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
  min-width: 0;
}
.gcp__card:hover {
  border-color: var(--m-g, var(--m-p));
}
.gcp__card.is-on {
  border-color: var(--m-g, var(--m-p));
  background: var(--m-gSoft, var(--m-pSoft));
}
.gcp__card .gcp__card-img {
  width: 100%;
  aspect-ratio: 1;
  border-radius: 12px;
  object-fit: cover;
}
.gcp__check {
  position: absolute;
  right: 10px;
  top: 10px;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--m-g, var(--m-p));
  color: var(--m-gOn, var(--m-pOn));
  display: flex;
  align-items: center;
  justify-content: center;
}
.gcp__card-name {
  font-size: 12px;
  font-weight: 800;
  padding: 0 2px;
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.gcp__card-sub {
  font-size: 11px;
  color: var(--el-text-color-secondary);
  padding: 0 2px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
</style>
