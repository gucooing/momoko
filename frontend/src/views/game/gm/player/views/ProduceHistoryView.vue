<!-- 培育记录：玩家保有的培育结果记忆（从新到旧）；培育结算时交换掉或之后放弃的记忆不在其中。点开一行看记忆详情与培育记录。
     按稀有度（R / SR / SSR）、评价（S、SS、SSS……）与偶像或卡名筛选，都由服务端筛选分页：关键字解析成偶像卡 ID，
     解析不出时关键字交给服务端模糊匹配；胶囊上的条数来自列表的 summary。 -->
<template>
  <div class="pv">
    <ListToolbar
      v-model:status="rarity"
      v-model:keyword="search.keyword.value"
      :chips="rarityChips"
      :chips-label="t('gmData.cards.rarity')"
      :placeholder="t('gmData.produce.search')"
    />
    <ListToolbar v-model:status="grade" :chips="gradeChips" :chips-label="t('gmData.memories.grade')" :searchable="false" />
    <div v-if="list.error.value || search.error.value" class="pv__error">{{ list.error.value || search.error.value }}</div>
    <ProduceTable
      :rows="list.items.value"
      :loading="list.loading.value || search.loading.value"
      :empty-text="search.active.value || filters.active.value ? t('gmData.common.noMatch') : t('gmData.produce.empty')"
      openable
      @open="(row) => (memoryId = row.memoryId)"
    />
    <div v-if="list.total.value" class="pv__foot">
      <span class="m-num">{{ t('gmData.produce.total', { n: list.total.value }) }}</span>
      <Pagination v-model:page="list.page.value" v-model:page-size="list.pageSize.value" :total="list.total.value" @change="list.load()" />
    </div>

    <MemoryDetailDialog v-model:memory-id="memoryId" />
  </div>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listPlayerMemories } from '@/api/gakumas'
import type { Memory } from '@/games/gakumas/types'
import type { MemoryCounts } from '@/types/v1/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import MemoryDetailDialog from '../shared/MemoryDetailDialog.vue'
import ProduceTable from '../shared/ProduceTable.vue'
import { useMemoryFilters } from '../shared/memoryFilters'
import { usePagedList, type PageQuery } from '../shared/paging'
import { idolCardIds, useIdSearch } from '../shared/search'

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()

const search = useIdSearch((q) => idolCardIds(ctx, q), { serverQuery: true })
const filters = useMemoryFilters(() => list.summary.value)
const { rarity, grade, rarityChips, gradeChips } = filters
const memoryId = ref('')

const fetchPage = async (q: PageQuery) => {
  if (search.none.value) return { items: [], nextCursor: '', total: 0 }
  return listPlayerMemories({
    serverId: ctx.serverId.value,
    pid: page.pid.value,
    produced: true,
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
</script>

<style scoped lang="scss">
@use '../shared/views';
</style>
