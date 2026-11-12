<!-- 支援卡：按获得时间从新到旧的卡面网格（稀有度、等级、上限解放、未使用的重复张数、是否援助中），点开看详情并可设为援助。
     稀有度的数量来自玩家概览；稀有度与名称搜索由服务端筛选分页：名称解析成支援卡 ID，解析不出时关键字交给服务端模糊匹配。 -->
<template>
  <div class="pv">
    <ListToolbar
      v-model:status="rarity"
      v-model:keyword="search.keyword.value"
      :chips="chips"
      :chips-label="t('gmData.cards.rarity')"
      :placeholder="t('gmData.cards.searchSupport')"
    />

    <div v-if="list.error.value || search.error.value" class="pv__error">{{ list.error.value || search.error.value }}</div>
    <div v-if="search.loading.value" class="pv__hint">{{ t('gmData.common.searching') }}</div>
    <div v-else-if="list.loading.value && !list.items.value.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState
      v-else-if="!shown.length"
      icon="HOutline:Squares2X2Icon"
      :title="filtered ? t('gmData.common.noMatch') : t('gmData.cards.noSupport')"
    />
    <div v-else class="pv__grid">
      <button v-for="c in shown" :key="c.supportCardId" type="button" class="pv__card pv__card--btn" @click="opened = c">
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
      </button>
    </div>

    <div v-if="list.total.value" class="pv__foot">
      <span class="m-num">{{ t('gmData.common.total', { n: list.total.value }) }}</span>
      <Pagination
        v-model:page="list.page.value"
        v-model:page-size="list.pageSize.value"
        :total="list.total.value"
        @change="list.load()"
      />
    </div>

    <SupportCardDetailDialog v-model:card="opened" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listPlayerSupportCards } from '@/api/gakumas'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import type { SupportCard } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import RarityBadge from '../shared/RarityBadge.vue'
import SupportCardDetailDialog from '../shared/SupportCardDetailDialog.vue'
import { catalogImage, rarityLabel } from '../shared/game'
import { usePagedList, type PageQuery } from '../shared/paging'
import { catalogEntries, useIdSearch } from '../shared/search'
import { useCatalogLookup } from '../shared/lookup'

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()

const rarity = ref<number>(0)
const search = useIdSearch(async (q) => (await catalogEntries(ctx, 'supportCard', { q })).map((e) => e.id), { serverQuery: true })
const opened = ref<SupportCard | null>(null)

const fetchPage = async (q: PageQuery) => {
  if (search.none.value) return { items: [], nextCursor: '', total: 0 }
  return listPlayerSupportCards({
    serverId: ctx.serverId.value,
    pid: page.pid.value,
    q: search.q.value,
    rarity: rarity.value,
    supportCardIds: search.ids.value || [],
    cursor: q.cursor,
    limit: q.limit,
  })
}
const list = usePagedList<SupportCard>(fetchPage, { pageSize: 20 })
watch([page.version, rarity, search.ids, search.q], () => void list.reload(), { immediate: true })

const counts = computed(() => page.player.value?.overview?.supportCardRarities || {})
const total = computed(() => page.player.value?.overview?.supportCardCount ?? list.total.value)
const filtered = computed(() => search.active.value || !!rarity.value)
const chips = computed<ToolbarChip<number>[]>(() => [
  { label: t('gmData.common.all'), value: 0, count: total.value },
  ...[3, 2, 1].filter((r) => counts.value[r]).map((r) => ({ label: rarityLabel(r), value: r, count: counts.value[r] ?? 0 })),
])

const cardName = (c: SupportCard) => lookup.name('supportCard', c.supportCardId)
const image = (c: SupportCard) => catalogImage(ctx.game.value, 'supportCard', { id: c.supportCardId })

const shown = computed(() => list.items.value)
</script>

<style scoped lang="scss">
@use '../shared/views';
</style>
