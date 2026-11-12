<!-- 支援卡：按获得时间从新到旧的卡面网格（稀有度、等级、上限解放、未使用的重复张数、是否资料出借）。
     稀有度的数量来自玩家概览；按名称模糊搜索时读出全部卡在本地筛选。 -->
<template>
  <div class="pv">
    <ListToolbar
      v-model:status="rarity"
      v-model:keyword="keyword"
      :chips="chips"
      :chips-label="t('gmData.cards.rarity')"
      :placeholder="t('gmData.cards.searchSupport')"
    />

    <div v-if="list.error.value || searchError" class="pv__error">{{ list.error.value || searchError }}</div>
    <div v-if="searchLoading" class="pv__hint">{{ t('gmData.common.searching') }}</div>
    <div v-else-if="list.loading.value && !list.items.value.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState
      v-else-if="!shown.length"
      icon="HOutline:Squares2X2Icon"
      :title="list.items.value.length || searching ? t('gmData.common.noMatch') : t('gmData.cards.noSupport')"
    />
    <div v-else class="pv__grid">
      <article v-for="c in shown" :key="c.supportCardId" class="pv__card">
        <span class="pv__card-art">
          <GameImage :src="image(c)" :alt="cardName(c)" class="pv__card-img" />
          <RarityBadge :rarity="c.rarity" class="pv__card-corner" />
        </span>
        <span class="pv__card-body">
          <span class="pv__strong pv__ellipsis" :title="cardName(c)">{{ cardName(c) }}</span>
        </span>
        <span class="pv__tags">
          <span class="pv__tag">Lv {{ c.level }}</span>
          <span class="pv__tag">{{ t('gmData.cards.limit', { n: c.levelLimitRank }) }}</span>
          <span v-if="c.stockQuantity" class="pv__tag">{{ t('gmData.cards.stock', { n: c.stockQuantity }) }}</span>
          <span v-if="c.lent" class="pv__tag is-g">{{ t('gmData.cards.lent') }}</span>
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
import type { SupportCard } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import RarityBadge from '../shared/RarityBadge.vue'
import { catalogImage, rarityLabel } from '../shared/game'
import { fetchAllPages, usePagedList } from '../shared/paging'
import { useNameSearch } from '../shared/search'
import { useCatalogLookup } from '../shared/lookup'

const props = defineProps<{ view?: ManifestView }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()

const fetchPage = (q: { cursor: string; limit: number }) =>
  getGmPlayerView<ListPage<SupportCard>>(ctx.serverId.value, page.pid.value, props.view?.id || 'supportCards', mgpQuery({ ...q }))
const list = usePagedList<SupportCard>(fetchPage, { pageSize: 100, append: true })
watch(page.version, () => void list.reload(), { immediate: true })

const rarity = ref<number>(0)

const counts = computed(() => page.player.value?.overview?.supportCardRarities || {})
const total = computed(() => page.player.value?.overview?.supportCardCount ?? list.total.value)
const chips = computed<ToolbarChip<number>[]>(() => [
  { label: t('gmData.common.all'), value: 0, count: total.value },
  ...[3, 2, 1].filter((r) => counts.value[String(r)]).map((r) => ({ label: rarityLabel(r), value: r, count: counts.value[String(r)] ?? 0 })),
])

const lookup = useCatalogLookup()
const cardName = (c: SupportCard) => lookup.name('supportCard', c.supportCardId)
const {
  keyword,
  active: searching,
  results,
  loading: searchLoading,
  error: searchError,
} = useNameSearch<SupportCard>({
  fetchAll: () => fetchAllPages(fetchPage),
  names: (c) => lookup.names('supportCard', c.supportCardId),
  version: page.version,
})
const image = (c: SupportCard) => catalogImage(ctx.game.value, 'supportCard', { id: c.supportCardId })

const shown = computed(() =>
  (searching.value ? results.value : list.items.value).filter((c) => !rarity.value || c.rarity === rarity.value),
)
</script>

<style scoped lang="scss">
@use '../shared/views';
</style>
