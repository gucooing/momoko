<!-- 培育记录：玩家保有的培育结果记忆（MGP produceHistory，从新到旧）；培育结算时交换掉或之后放弃的记忆不在其中。
     按偶像、卡名或剧本模糊搜索时读出全部记录在本地筛选。 -->
<template>
  <div class="pv">
    <ListToolbar v-model:keyword="keyword" :placeholder="t('gmData.produce.search')" />
    <div v-if="list.error.value || searchError" class="pv__error">{{ list.error.value || searchError }}</div>
    <ProduceTable
      :rows="searching ? results : list.items.value"
      :loading="list.loading.value || searchLoading"
      :empty-text="searching ? t('gmData.common.noMatch') : t('gmData.produce.empty')"
    />
    <div v-if="searching" class="pv__foot">
      <span class="m-num">{{ t('gmData.common.matched', { n: results.length }) }}</span>
    </div>
    <div v-else-if="list.total.value" class="pv__foot">
      <span class="m-num">{{ t('gmData.produce.total', { n: list.total.value }) }}</span>
      <Pagination v-model:page="list.page.value" v-model:page-size="list.pageSize.value" :total="list.total.value" @change="list.load()" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmPlayerView, mgpQuery } from '@/api/gameGm'
import type { ListPage, ManifestView } from '@/games/mgp'
import type { Memory } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import ProduceTable from '../shared/ProduceTable.vue'
import { useCatalogLookup } from '../shared/lookup'
import { fetchAllPages, usePagedList } from '../shared/paging'
import { useNameSearch } from '../shared/search'

const props = defineProps<{ view?: ManifestView }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()

const fetchPage = (q: { cursor: string; limit: number }) =>
  getGmPlayerView<ListPage<Memory>>(ctx.serverId.value, page.pid.value, props.view?.id || 'produceHistory', mgpQuery({ ...q }))
const list = usePagedList<Memory>(fetchPage, { pageSize: 20 })
watch(page.version, () => void list.load(), { immediate: true })

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
</script>

<style scoped lang="scss">
@use '../shared/views';
</style>
