<!-- 道具：未过期且数量大于 0 的道具栈（名称与图片来自 MGP 道具目录，只查当前页）、数量、期限；
     有 item.adjust 指令时每行可以「调整」（预填这个道具，走通用指令对话框，只需填数量）。手机端为卡片。
     按名称模糊搜索：输入后读出全部道具在本地筛选（当前语言的译名与原文名都能搜）。 -->
<template>
  <div class="pv">
    <ListToolbar v-model:keyword="keyword" :placeholder="t('gmData.items.search')" />
    <div v-if="list.error.value || searchError" class="pv__error">{{ list.error.value || searchError }}</div>

    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="rows"
      row-key="key"
      :loading="list.loading.value || searchLoading"
      :empty-text="searching ? t('gmData.common.noMatch') : t('gmData.items.empty')"
    >
      <template #cell-item="{ row }">
        <span class="pv__cell">
          <GameImage :src="lookup.image('item', it(row).itemId)" class="pv__thumb" width="36" height="36" />
          <span class="pv__strong pv__ellipsis">{{ lookup.name('item', it(row).itemId) }}</span>
        </span>
      </template>
      <template #cell-quantity="{ row }">
        <span class="pv__strong m-num">{{ num(it(row).quantity) }}</span>
      </template>
      <template #cell-expiry="{ row }">
        <span class="pv__dim">{{ it(row).expiryTime ? t('gmData.items.expires', { time: dateTime(it(row).expiryTime) }) : t('gmData.items.noExpiry') }}</span>
      </template>
      <template #cell-op="{ row }">
        <div class="pv__acts">
          <UButton v-if="canAdjust" size="sm" color="neutral" variant="outline" @click="adjust(it(row).itemId)">
            {{ t('gmData.items.adjust') }}
          </UButton>
        </div>
      </template>
    </DataTable>

    <template v-else>
      <div v-if="searchLoading" class="pv__hint">{{ t('gmData.common.searching') }}</div>
      <EmptyState
        v-else-if="!list.loading.value && !shownItems.length"
        icon="HOutline:CubeIcon"
        :title="searching ? t('gmData.common.noMatch') : t('gmData.items.empty')"
      />
      <div v-else class="pv__mcards">
        <article v-for="item in shownItems" :key="`${item.itemId}:${item.expiryTime}`" class="pv__mcard">
          <GameImage :src="lookup.image('item', item.itemId)" class="pv__thumb" width="36" height="36" />
          <div class="pv__mbody">
            <span class="pv__strong pv__ellipsis">{{ lookup.name('item', item.itemId) }}</span>
            <span class="pv__dim">{{ item.expiryTime ? t('gmData.items.expires', { time: dateTime(item.expiryTime) }) : t('gmData.items.noExpiry') }}</span>
          </div>
          <div class="pv__stack items-m__right">
            <span class="pv__strong m-num">×{{ num(item.quantity) }}</span>
            <AppIconButton
              v-if="canAdjust"
              icon="HOutline:AdjustmentsHorizontalIcon"
              :label="t('gmData.items.adjust')"
              :box="44"
              @click="adjust(item.itemId)"
            />
          </div>
        </article>
      </div>
    </template>

    <div v-if="searching" class="pv__foot">
      <span class="m-num">{{ t('gmData.common.matched', { n: shownItems.length }) }}</span>
    </div>
    <div v-else-if="list.total.value" class="pv__foot">
      <span class="m-num">{{ t('gmData.items.total', { n: list.total.value }) }}</span>
      <Pagination v-model:page="list.page.value" v-model:page-size="list.pageSize.value" :total="list.total.value" @change="list.load()" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmPlayerView, mgpQuery } from '@/api/gameGm'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ListPage, ManifestView } from '@/games/mgp'
import type { Item } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
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
const { num, dateTime } = useGmFormat()

const fetchPage = (q: { cursor: string; limit: number }) =>
  getGmPlayerView<ListPage<Item>>(ctx.serverId.value, page.pid.value, props.view?.id || 'items', mgpQuery({ ...q }))
const list = usePagedList<Item>(fetchPage, { pageSize: 20 })
watch(page.version, () => void list.load(), { immediate: true })

const {
  keyword,
  active: searching,
  results,
  loading: searchLoading,
  error: searchError,
} = useNameSearch<Item>({
  fetchAll: () => fetchAllPages(fetchPage),
  names: (x) => lookup.names('item', x.itemId),
  version: page.version,
})
const shownItems = computed(() => (searching.value ? results.value : list.items.value))

const columns = computed<DataTableColumn[]>(() => [
  { key: 'item', title: t('gmData.items.item'), minWidth: 220 },
  { key: 'quantity', title: t('gmData.items.quantity'), width: 130, align: 'right' },
  { key: 'expiry', title: t('gmData.items.expiry'), width: 200 },
  { key: 'op', title: '', width: 100, align: 'right' },
])
const rows = computed(() => shownItems.value.map((x) => ({ ...x, key: `${x.itemId}:${x.expiryTime}` })) as unknown as Record<string, unknown>[])
const it = (row: Record<string, unknown>) => row as unknown as Item

const canAdjust = computed(() => ctx.hasCommand('item.adjust'))
const adjust = (itemId: string) => page.run('item.adjust', { itemId })
</script>

<style scoped lang="scss">
@use '../shared/views';
.items-m__right {
  align-items: flex-end;
  gap: 4px;
}
</style>
