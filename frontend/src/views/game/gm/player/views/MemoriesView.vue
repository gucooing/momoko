<!-- 记忆：玩家的全部记忆（MGP memories，从新到旧）：卡面、评价与评价值、角色、Vo / Da / Vi、编成力、出借与保护标记。
     按偶像、卡名或剧本模糊搜索时读出全部记忆在本地筛选。 -->
<template>
  <div class="pv">
    <ListToolbar v-model:keyword="keyword" :placeholder="t('gmData.memories.search')" />
    <div v-if="list.error.value || searchError" class="pv__error">{{ list.error.value || searchError }}</div>
    <div v-if="searchLoading" class="pv__hint">{{ t('gmData.common.searching') }}</div>
    <div v-else-if="list.loading.value && !list.items.value.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState
      v-else-if="!shownMemories.length"
      icon="HOutline:Squares2X2Icon"
      :title="searching ? t('gmData.common.noMatch') : t('gmData.memories.empty')"
    />
    <div v-else class="pv__grid">
      <article v-for="m in shownMemories" :key="m.memoryId" class="pv__card">
        <span class="pv__card-art">
          <GameImage :src="image(m)" :alt="charaName(m)" class="pv__card-img" />
          <GradeBadge :grade="m.grade" class="pv__card-corner" />
        </span>
        <span class="pv__card-body">
          <span class="pv__strong pv__ellipsis">{{ charaName(m) }}</span>
          <span v-if="m.score" class="pv__dim m-num">{{ t('gmData.memories.score', { n: num(m.score) }) }}</span>
          <span class="pv__dim m-num">Vo {{ num(m.vocal) }} · Da {{ num(m.dance) }} · Vi {{ num(m.visual) }}</span>
        </span>
        <span class="pv__tags">
          <span v-if="m.power" class="pv__tag">{{ t('gmData.memories.power', { n: num(m.power) }) }}</span>
          <span v-if="m.lent" class="pv__tag is-g">{{ t('gmData.memories.lent') }}</span>
          <span v-if="m.protected" class="pv__tag">{{ t('gmData.memories.protected') }}</span>
          <span v-if="!m.produceId" class="pv__tag">{{ m.researchId ? t('gmData.memories.research') : t('gmData.memories.other') }}</span>
        </span>
      </article>
    </div>
    <div class="pv__foot">
      <span v-if="searching">{{ t('gmData.common.matched', { n: shownMemories.length }) }}</span>
      <span v-else>{{ t('gmData.common.shown', { n: list.items.value.length, total: list.total.value }) }}</span>
      <button v-if="!searching && list.nextCursor.value" type="button" class="pv__more" :disabled="list.loading.value" @click="list.more()">
        {{ list.loading.value ? t('gmData.common.loading') : t('gmData.common.loadMore') }}
      </button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmPlayerView, mgpQuery } from '@/api/gameGm'
import type { ListPage, ManifestView } from '@/games/mgp'
import type { Memory } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import GradeBadge from '../shared/GradeBadge.vue'
import { catalogImage } from '../shared/game'
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
  getGmPlayerView<ListPage<Memory>>(ctx.serverId.value, page.pid.value, props.view?.id || 'memories', mgpQuery({ ...q }))
const list = usePagedList<Memory>(fetchPage, { pageSize: 60, append: true })
watch(page.version, () => void list.reload(), { immediate: true })

const {
  keyword,
  active: searching,
  results,
  loading: searchLoading,
  error: searchError,
} = useNameSearch<Memory>({
  fetchAll: () => fetchAllPages(fetchPage),
  names: (m) => [
    ...lookup.names('character', m.characterId),
    ...lookup.names('idolCard', m.idolCardId),
    ...lookup.names('produce', m.produceId),
  ],
  version: page.version,
})
const shownMemories = computed(() => (searching.value ? results.value : list.items.value))

const image = (m: Memory) => catalogImage(ctx.game.value, 'idolCardSkin', { id: m.idolCardId })
const charaName = (m: Memory) => (m.characterId ? lookup.name('character', m.characterId) : lookup.name('idolCard', m.idolCardId))
</script>

<style scoped lang="scss">
@use '../shared/views';
</style>
