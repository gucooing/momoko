<!-- 抽卡记录：玩家全部抽卡的合计（MGP summary），每次抽卡的时间、卡池横幅与名称（MGP 卡池目录）、抽数、消耗、抽到的卡
     （稀有度描边，NEW / UP 标记）。可按卡池筛选（从 MGP 卡池目录搜索）；按卡池名或抽到的卡名模糊搜索时读出全部记录在本地筛选。 -->
<template>
  <div class="pv">
    <div class="pv__line">
      <span v-if="totals">
        <i18n-t keypath="gmData.gasha.totals" tag="span" scope="global">
          <template #draws><b class="m-num">{{ num(totals.draws) }}</b></template>
          <template #ssr><b class="m-num">{{ num(totals.ssr) }}</b></template>
          <template #pickup><b class="m-num">{{ num(totals.pickup) }}</b></template>
          <template #jewel><b class="m-num">{{ num(totals.jewelFree + totals.jewelPaid) }}</b></template>
          <template #paid><b class="m-num">{{ num(totals.jewelPaid) }}</b></template>
        </i18n-t>
      </span>
      <span class="pv__spacer" />
      <AppDropdown align="end" :width="360">
        <template #trigger>
          <button type="button" class="gh__chip">
            {{ gashaId ? t('gmData.gasha.filterOne', { name: lookup.name('gasha', gashaId) }) : t('gmData.gasha.filterAll') }}
            <component :is="menuStore.iconComponents['HOutline:ChevronUpDownIcon']" class="gh__chip-icon" />
          </button>
        </template>
        <template #default="{ close }">
          <div class="gh__pick">
            <button v-if="gashaId" type="button" class="gh__pick-all" @click="pick(''); close()">{{ t('gmData.gasha.filterAll') }}</button>
            <GmCatalogSearch kind="gasha" :selected-id="gashaId" @pick="(e) => (pick(e.id), close())" />
          </div>
        </template>
      </AppDropdown>
    </div>
    <ListToolbar v-model:keyword="keyword" :placeholder="t('gmData.gasha.search')" />

    <div v-if="list.error.value || searchError" class="pv__error">{{ list.error.value || searchError }}</div>
    <div v-if="searchLoading" class="pv__hint">{{ t('gmData.common.searching') }}</div>
    <div v-else-if="list.loading.value && !list.items.value.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState
      v-else-if="!shownDraws.length"
      icon="HOutline:SparklesIcon"
      :title="searching ? t('gmData.common.noMatch') : t('gmData.gasha.empty')"
    />

    <div v-else class="gh__list">
      <article v-for="d in shownDraws" :key="d.id" class="gh__row">
        <span class="gh__when pv__dim m-num">{{ when(d.createdAt) }}</span>
        <span class="gh__gasha">
          <GameImage :src="lookup.image('gasha', d.gashaId)" class="gh__banner" width="160" height="56" />
          <span class="pv__dim pv__ellipsis" :title="lookup.name('gasha', d.gashaId)">{{ lookup.name('gasha', d.gashaId) }}</span>
        </span>
        <span class="gh__count pv__strong">{{ t('gmData.gasha.draws', { n: d.drawCount }) }}</span>
        <span class="gh__cost">
          <GameImage v-if="costImage(d)" :src="costImage(d)" class="gh__cost-img" width="22" height="22" />
          <span class="pv__strong m-num">{{ num(d.costQuantity) }}</span>
        </span>
        <span class="gh__cards">
          <span
            v-for="(c, i) in d.cards || []"
            :key="i"
            class="gh__card"
            :class="`is-r${c.rarity}`"
            :title="cardTitle(c)"
          >
            <GameImage :src="cardImage(c)" class="gh__card-img" width="34" height="34" />
            <span v-if="c.isPickup" class="gh__mark is-up">UP</span>
            <span v-else-if="c.isNew" class="gh__mark">NEW</span>
          </span>
        </span>
      </article>
    </div>

    <div class="pv__foot">
      <span v-if="searching">{{ t('gmData.common.matched', { n: shownDraws.length }) }}</span>
      <span v-else>{{ t('gmData.common.loadedN', { n: list.items.value.length }) }}</span>
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
import type { ManifestView } from '@/games/mgp'
import type { DrawTotals, GashaDraw, GashaDrawnCard, GashaHistoryPage } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import GmCatalogSearch from '../../components/GmCatalogSearch.vue'
import { usePlayerPage } from '../context'
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
const menuStore = useMenuStore()
const { num, when } = useGmFormat()

const gashaId = ref('')
const fetchPage = (q: { cursor: string; limit: number }) =>
  getGmPlayerView<GashaHistoryPage>(ctx.serverId.value, page.pid.value, props.view?.id || 'gashaHistory', mgpQuery({ gashaId: gashaId.value, ...q }))
const list = usePagedList<GashaDraw, DrawTotals>(fetchPage, { pageSize: 20, append: true })
watch([gashaId, page.version], () => void list.reload(), { immediate: true })

const {
  keyword,
  active: searching,
  results,
  loading: searchLoading,
  error: searchError,
} = useNameSearch<GashaDraw>({
  fetchAll: () => fetchAllPages(fetchPage),
  names: (d) => [
    ...lookup.names('gasha', d.gashaId),
    ...(d.cards || []).flatMap((c) => {
      const rr = cardRef(c)
      return rr ? lookup.names(rr.kind, rr.id) : []
    }),
  ],
  version: [page.version, gashaId],
})
const shownDraws = computed(() => (searching.value ? results.value : list.items.value))

const totals = computed(() => list.summary.value)
const pick = (id: string) => {
  gashaId.value = id
}

const costImage = (d: GashaDraw) => {
  const rr = lookup.rewardRef(d.costResourceType, d.costResourceId)
  return rr ? lookup.image(rr.kind, rr.id) : ''
}
const cardRef = (c: GashaDrawnCard) => lookup.rewardRef(c.resourceType, c.resourceId)
const cardImage = (c: GashaDrawnCard) => {
  const rr = cardRef(c)
  return rr ? catalogImage(ctx.game.value, rr.kind, { id: rr.id }) : ''
}
const cardTitle = (c: GashaDrawnCard) => {
  const rr = cardRef(c)
  return `${rarityLabel(c.rarity)} ${rr ? lookup.name(rr.kind, rr.id) : t('game.unnamed')}`
}
</script>

<style scoped lang="scss">
@use '../shared/views';
.gh__chip {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  max-width: 320px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  cursor: pointer;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gh__chip-icon {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.gh__pick {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
}
.gh__pick-all {
  min-height: 40px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}
.gh__list {
  display: flex;
  flex-direction: column;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  overflow: hidden;
}
.gh__row {
  display: grid;
  grid-template-columns: 110px 176px 80px 100px minmax(0, 1fr);
  gap: 12px;
  align-items: center;
  min-height: 84px;
  padding: 10px 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.gh__row:first-child {
  border-top: 0;
}
.gh__gasha {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.gh__banner {
  width: 160px;
  height: 56px;
  object-fit: contain;
  border-radius: 12px;
  background: var(--m-s2);
}
.gh__cost {
  display: inline-flex;
  align-items: center;
  gap: 6px;
}
.gh__cost-img {
  width: 22px;
  height: 22px;
  object-fit: contain;
  background: transparent;
}
.gh__cards {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.gh__card {
  position: relative;
  width: 34px;
  height: 34px;
  border-radius: 9px;
  overflow: hidden;
  box-shadow: inset 0 0 0 2px var(--m-s2);
}
.gh__card.is-r3 {
  box-shadow: 0 0 0 2px var(--m-a);
}
.gh__card.is-r2 {
  box-shadow: 0 0 0 2px var(--m-a2);
}
.gh__card-img {
  width: 100%;
  height: 100%;
}
.gh__mark {
  position: absolute;
  left: 0;
  right: 0;
  bottom: 0;
  font-size: 8px;
  font-weight: 800;
  line-height: 12px;
  text-align: center;
  background: color-mix(in srgb, var(--m-ink) 70%, transparent);
  color: var(--m-bg);
}
.gh__mark.is-up {
  background: var(--m-g);
  color: var(--m-gOn);
}
@media (max-width: 900px) {
  .gh__row {
    grid-template-columns: 1fr auto;
    grid-template-areas:
      'gasha gasha'
      'count cost'
      'cards cards'
      'when when';
    gap: 8px;
  }
  .gh__gasha {
    grid-area: gasha;
  }
  .gh__count {
    grid-area: count;
  }
  .gh__cost {
    grid-area: cost;
    justify-self: end;
  }
  .gh__cards {
    grid-area: cards;
  }
  .gh__when {
    grid-area: when;
  }
}
</style>
