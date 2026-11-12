<!-- 记忆：玩家的全部记忆（MGP memories，从新到旧）：卡面、评价与评价值、角色、Vo / Da / Vi、编成力、援助与保护标记；
     点开看记忆详情与培育记录，并可设为援助回忆。
     按稀有度（R / SR / SSR）、评价（S、SS、SSS……）与偶像或卡名筛选，都由服务端筛选分页：关键字解析成偶像卡 ID，
     解析不出时关键字交给服务端模糊匹配；胶囊上的条数来自列表的 summary。 -->
<template>
  <div class="pv">
    <ListToolbar
      v-model:status="rarity"
      v-model:keyword="search.keyword.value"
      :chips="rarityChips"
      :chips-label="t('gmData.cards.rarity')"
      :placeholder="t('gmData.memories.search')"
    />
    <ListToolbar v-model:status="grade" :chips="gradeChips" :chips-label="t('gmData.memories.grade')" :searchable="false" />
    <div v-if="list.error.value || search.error.value" class="pv__error">{{ list.error.value || search.error.value }}</div>
    <div v-if="search.loading.value" class="pv__hint">{{ t('gmData.common.searching') }}</div>
    <div v-else-if="list.loading.value && !list.items.value.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState
      v-else-if="!shownMemories.length"
      icon="HOutline:Squares2X2Icon"
      :title="search.active.value || filters.active.value ? t('gmData.common.noMatch') : t('gmData.memories.empty')"
    />
    <div v-else class="pv__grid">
      <button v-for="m in shownMemories" :key="m.memoryId" type="button" class="pv__card pv__card--btn" @click="memoryId = m.memoryId">
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

    <MemoryDetailDialog v-model:memory-id="memoryId" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listPlayerMemories } from '@/api/gakumas'
import type { Memory } from '@/games/gakumas/types'
import type { MemoryCounts } from '@/types/v1/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import GradeBadge from '../shared/GradeBadge.vue'
import MemoryDetailDialog from '../shared/MemoryDetailDialog.vue'
import { catalogImage } from '../shared/game'
import { useGmFormat } from '../shared/format'
import { useCatalogLookup } from '../shared/lookup'
import { useMemoryFilters } from '../shared/memoryFilters'
import { usePagedList, type PageQuery } from '../shared/paging'
import { idolCardIds, useIdSearch } from '../shared/search'

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()
const { num } = useGmFormat()

const search = useIdSearch((q) => idolCardIds(ctx, q), { serverQuery: true })
const filters = useMemoryFilters(() => list.summary.value)
const { rarity, grade, rarityChips, gradeChips } = filters
const memoryId = ref('')

const fetchPage = async (q: PageQuery) => {
  if (search.none.value) return { items: [], nextCursor: '', total: 0 }
  return listPlayerMemories({
    serverId: ctx.serverId.value,
    pid: page.pid.value,
    produced: false,
    q: search.q.value,
    rarity: rarity.value,
    grades: filters.grades.value,
    characterId: '',
    idolCardIds: search.ids.value || [],
    cursor: q.cursor,
    limit: q.limit,
  })
}
const list = usePagedList<Memory, MemoryCounts>(fetchPage, { pageSize: 20 })
watch([page.version, search.ids, search.q, rarity, grade], () => void list.reload(), { immediate: true })

const shownMemories = computed(() => list.items.value)

const image = (m: Memory) => catalogImage(ctx.game.value, 'idolCardSkin', { id: m.idolCardId })
const charaName = (m: Memory) => (m.characterId ? lookup.name('character', m.characterId) : lookup.name('idolCard', m.idolCardId))
</script>

<style scoped lang="scss">
@use '../shared/views';
</style>
