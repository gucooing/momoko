<!-- 偶像卡：按获得时间从新到旧的卡面网格（当前皮肤方图、稀有度、皮肤名与角色、才能开花、上限解放、Prima Stella、碎片）。
     稀有度的数量来自玩家概览；名称按角色分组查 MGP 目录（一个角色一次）。按卡名或角色名模糊搜索时读出全部卡在本地筛选。 -->
<template>
  <div class="pv">
    <ListToolbar
      v-model:status="rarity"
      v-model:keyword="keyword"
      :chips="chips"
      :chips-label="t('gmData.cards.rarity')"
      :placeholder="t('gmData.cards.searchIdol')"
    />

    <div v-if="list.error.value || searchError" class="pv__error">{{ list.error.value || searchError }}</div>
    <div v-if="searchLoading" class="pv__hint">{{ t('gmData.common.searching') }}</div>
    <div v-else-if="list.loading.value && !list.items.value.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState
      v-else-if="!shown.length"
      icon="HOutline:Squares2X2Icon"
      :title="list.items.value.length || searching ? t('gmData.common.noMatch') : t('gmData.cards.noIdol')"
    />
    <div v-else class="pv__grid">
      <article v-for="c in shown" :key="c.idolCardId" class="pv__card">
        <span class="pv__card-art">
          <GameImage :src="image(c)" :alt="cardName(c)" class="pv__card-img" />
          <RarityBadge :rarity="c.rarity" class="pv__card-corner" />
        </span>
        <span class="pv__card-body">
          <span class="pv__strong pv__ellipsis" :title="cardName(c)">{{ cardName(c) }}</span>
          <span class="pv__dim pv__ellipsis">{{ charaName(c) }}</span>
        </span>
        <span class="pv__tags">
          <span class="pv__tag">{{ t('gmData.cards.potential', { n: c.potentialRank }) }}</span>
          <span class="pv__tag">{{ t('gmData.cards.limit', { n: c.levelLimitRank }) }}</span>
          <span v-if="c.primaStella" class="pv__tag is-g">Prima Stella</span>
          <span v-if="c.pieces" class="pv__tag">{{ t('gmData.cards.pieces', { n: num(c.pieces) }) }}</span>
          <span v-if="(c.skinIds?.length || 0) > 1" class="pv__tag">{{ t('gmData.cards.skins', { n: c.skinIds?.length || 0 }) }}</span>
        </span>
      </article>
    </div>

    <div class="pv__foot">
      <span v-if="searching">{{ t('gmData.common.matched', { n: shown.length }) }}</span>
      <span v-else>{{ t('gmData.common.shown', { n: shown.length, total: total }) }}</span>
      <button v-if="!searching && list.nextCursor.value" type="button" class="pv__more" :disabled="list.loading.value" @click="list.more()">
        {{ list.loading.value ? t('gmData.common.loading') : t('gmData.common.loadMore') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmPlayerView, mgpQuery } from '@/api/gameGm'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import type { ListPage, ManifestView } from '@/games/mgp'
import type { IdolCard } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import RarityBadge from '../shared/RarityBadge.vue'
import { catalogImage, rarityLabel } from '../shared/game'
import { useGmFormat } from '../shared/format'
import { useCatalogLookup } from '../shared/lookup'
import { fetchAllPages, usePagedList } from '../shared/paging'
import { useNameSearch } from '../shared/search'

const props = defineProps<{ view?: ManifestView }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()
const { num } = useGmFormat()

const fetchPage = (q: { cursor: string; limit: number }) =>
  getGmPlayerView<ListPage<IdolCard>>(ctx.serverId.value, page.pid.value, props.view?.id || 'idolCards', mgpQuery({ ...q }))
const list = usePagedList<IdolCard>(fetchPage, { pageSize: 200, append: true })
watch(page.version, () => void list.reload(), { immediate: true })

const rarity = ref<number>(0)
const {
  keyword,
  active: searching,
  results,
  loading: searchLoading,
  error: searchError,
} = useNameSearch<IdolCard>({
  fetchAll: () => fetchAllPages(fetchPage),
  names: (c) => [
    ...lookup.names('idolCardSkin', c.skinId),
    ...lookup.names('idolCard', c.idolCardId),
    ...lookup.names('character', c.characterId),
  ],
  version: page.version,
})

const counts = computed(() => page.player.value?.overview?.idolCardRarities || {})
const total = computed(() => page.player.value?.overview?.idolCardCount ?? list.total.value)
const chips = computed<ToolbarChip<number>[]>(() => [
  { label: t('gmData.common.all'), value: 0, count: total.value },
  ...[3, 2, 1].filter((r) => counts.value[String(r)]).map((r) => ({ label: rarityLabel(r), value: r, count: counts.value[String(r)] ?? 0 })),
])

const skinEntry = (c: IdolCard) => lookup.group('idolCardSkin', c.characterId)?.find((e) => e.id === c.skinId)
const charaName = (c: IdolCard) => (c.characterId ? lookup.name('character', c.characterId) : '')
const cardName = (c: IdolCard) => (c.skinId ? lookup.name('idolCardSkin', c.skinId) : lookup.name('idolCard', c.idolCardId))
const image = (c: IdolCard) => catalogImage(ctx.game.value, 'idolCardSkin', skinEntry(c) || { id: c.skinId || c.idolCardId })

const shown = computed(() =>
  (searching.value ? results.value : list.items.value).filter((c) => !rarity.value || c.rarity === rarity.value),
)
</script>

<style scoped lang="scss">
@use '../shared/views';
</style>
