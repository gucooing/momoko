<!-- 道具：未过期且数量大于 0 的道具栈（名称与图片来自 MGP 道具目录，只查当前页）、数量、期限；
     有 item.adjust 指令时每行可以「调整」（预填这个道具，走通用指令对话框，只需填数量）。手机端为卡片。
     按名称搜索：关键字解析成道具 ID（当前语言的译名与原文名都能搜）后由服务端筛选分页。 -->
<template>
  <div class="pv">
    <ListToolbar v-model:keyword="search.keyword.value" :placeholder="t('gmData.items.search')" />
    <div v-if="list.error.value || search.error.value" class="pv__error">{{ list.error.value || search.error.value }}</div>

    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="rows"
      row-key="key"
      :loading="list.loading.value || search.loading.value"
      :empty-text="search.active.value ? t('gmData.common.noMatch') : t('gmData.items.empty')"
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
      <div v-if="search.loading.value" class="pv__hint">{{ t('gmData.common.searching') }}</div>
      <EmptyState
        v-else-if="!list.loading.value && !shownItems.length"
        icon="HOutline:CubeIcon"
        :title="search.active.value ? t('gmData.common.noMatch') : t('gmData.items.empty')"
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

    <div v-if="list.total.value" class="pv__foot">
      <span class="m-num">{{ t('gmData.items.total', { n: list.total.value }) }}</span>
      <Pagination v-model:page="list.page.value" v-model:page-size="list.pageSize.value" :total="list.total.value" @change="list.load()" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listPlayerItems } from '@/api/gakumas'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { Item } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import { useGmFormat } from '../shared/format'
import { useCatalogLookup } from '../shared/lookup'
import { usePagedList, type PageQuery } from '../shared/paging'
import { catalogEntries, useIdSearch } from '../shared/search'

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()
const menuStore = useMenuStore()
const { num, dateTime } = useGmFormat()

const search = useIdSearch(async (q) => (await catalogEntries(ctx, 'item', { q })).map((e) => e.id))

const fetchPage = async (q: PageQuery) => {
  if (search.none.value) return { items: [], nextCursor: '', total: 0 }
  return listPlayerItems({
    serverId: ctx.serverId.value,
    pid: page.pid.value,
    type: 0,
    itemIds: search.ids.value || [],
    cursor: q.cursor,
    limit: q.limit,
  })
}
const list = usePagedList<Item>(fetchPage, { pageSize: 20 })
watch([page.version, search.ids], () => void list.reload(), { immediate: true })

const shownItems = computed(() => list.items.value)

const columns = computed<DataTableColumn[]>(() => [
  { key: 'item', title: t('gmData.items.item'), minWidth: 220 },
  { key: 'quantity', title: t('gmData.items.quantity'), width: 130, align: 'right' },
  { key: 'expiry', title: t('gmData.items.expiry'), width: 200 },
  { key: 'op', title: '', width: 100, align: 'right' },
])
const rows = computed(() => shownItems.value.map((x) => ({ ...x, key: `${x.itemId}:${x.expiryTime}` })) as unknown as Record<string, unknown>[])
const it = (row: Record<string, unknown>) => row as unknown as Item

const canAdjust = computed(() => !!ctx.operation('item.adjust'))
const adjust = (itemId: string) => page.run('item.adjust', { itemId })
</script>

<style scoped lang="scss">
@use '../shared/views';
.items-m__right {
  align-items: flex-end;
  gap: 4px;
}
</style>
