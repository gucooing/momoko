<!-- 礼物箱：未领取 / 已领取 / 已过期（MGP state 筛选），按发放时间从新到旧；内容名称与图片来自 MGP 目录。
     按礼物名称、来源或留言模糊搜索：读出这个状态下的全部礼物在本地筛选。
     管理端（GM 指令、邮件、兑换码）发出且未领取的礼物可以撤回（gift.revoke，走通用指令对话框，记进 GM 记录）。 -->
<template>
  <div class="pv">
    <div class="pv__line">
      <div class="pv__seg" role="radiogroup" :aria-label="t('gmData.giftBox.state')">
        <button
          v-for="s in states"
          :key="s.value"
          type="button"
          role="radio"
          :aria-checked="state === s.value"
          :class="{ 'is-on': state === s.value }"
          @click="state = s.value"
        >
          {{ s.label }}
        </button>
      </div>
      <span class="pv__spacer" />
      <span v-if="canRevoke" class="pv__hint">{{ t('gmData.giftBox.revokeHint') }}</span>
    </div>
    <ListToolbar v-model:keyword="keyword" :placeholder="t('gmData.giftBox.search')" />

    <div v-if="list.error.value || searchError" class="pv__error">{{ list.error.value || searchError }}</div>

    <DataTable
      v-if="!menuStore.isMobile"
      :columns="columns"
      :rows="rows"
      row-key="giftId"
      :loading="list.loading.value || searchLoading"
      :empty-text="t('gmData.giftBox.empty')"
    >
      <template #cell-content="{ row }">
        <span class="pv__cell">
          <GameImage :src="giftImage(g(row))" class="pv__thumb" width="36" height="36" />
          <span class="pv__strong pv__ellipsis">{{ giftName(g(row)) }} ×{{ num(g(row).quantity) }}</span>
        </span>
      </template>
      <template #cell-source="{ row }">
        <span class="pv__stack">
          <span class="pv__ellipsis">{{ sourceLabel(g(row)) }}</span>
        </span>
      </template>
      <template #cell-created="{ row }">
        <span class="pv__dim m-num">{{ when(g(row).createdAt) }}</span>
      </template>
      <template #cell-limit="{ row }">
        <StatusPill :variant="limitPill(g(row)).variant" :label="limitPill(g(row)).label" />
      </template>
      <template #cell-op="{ row }">
        <div class="pv__acts">
          <UButton v-if="revocable(g(row))" size="sm" color="neutral" variant="outline" @click="revoke(g(row))">
            {{ t('gmData.giftBox.revoke') }}
          </UButton>
        </div>
      </template>
    </DataTable>

    <template v-else>
      <div v-if="searchLoading" class="pv__hint">{{ t('gmData.common.searching') }}</div>
      <EmptyState
        v-else-if="!list.loading.value && !shownGifts.length"
        icon="HOutline:GiftIcon"
        :title="searching ? t('gmData.common.noMatch') : t('gmData.giftBox.empty')"
      />
      <div v-else class="pv__mcards">
        <article v-for="gift in shownGifts" :key="gift.giftId" class="pv__mcard">
          <GameImage :src="giftImage(gift)" class="pv__thumb" width="36" height="36" />
          <div class="pv__mbody">
            <span class="pv__strong">{{ giftName(gift) }} ×{{ num(gift.quantity) }}</span>
            <span class="pv__dim">{{ sourceLabel(gift) }}</span>
            <span class="pv__mline">
              <StatusPill :variant="limitPill(gift).variant" :label="limitPill(gift).label" />
              <span class="pv__dim m-num">{{ when(gift.createdAt) }}</span>
            </span>
          </div>
          <AppIconButton
            v-if="revocable(gift)"
            icon="HOutline:ArrowUturnLeftIcon"
            :label="t('gmData.giftBox.revoke')"
            :box="44"
            @click="revoke(gift)"
          />
        </article>
      </div>
    </template>

    <div v-if="searching" class="pv__foot">
      <span class="m-num">{{ t('gmData.common.matched', { n: shownGifts.length }) }}</span>
    </div>
    <div v-else-if="list.total.value" class="pv__foot">
      <span class="m-num">{{ t('gmData.giftBox.total', { n: list.total.value }) }}</span>
      <Pagination v-model:page="list.page.value" v-model:page-size="list.pageSize.value" :total="list.total.value" @change="list.load()" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmPlayerView, mgpQuery } from '@/api/gameGm'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ListPage, ManifestView } from '@/games/mgp'
import type { Gift, GiftState } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import { giftSourceName } from '../shared/game'
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
const { num, when, dateTime } = useGmFormat()

const state = ref<GiftState>('unreceived')
const fetchPage = (q: { cursor: string; limit: number }) =>
  getGmPlayerView<ListPage<Gift>>(ctx.serverId.value, page.pid.value, props.view?.id || 'giftBox', mgpQuery({ state: state.value, ...q }))
const list = usePagedList<Gift>(fetchPage, { pageSize: 20 })
watch([state, page.version], () => void list.reload(), { immediate: true })

const states = computed(() => {
  const unreceived = page.player.value?.overview?.gifts?.unreceived
  return [
    { value: 'unreceived' as GiftState, label: unreceived ? t('gmData.giftBox.unreceivedN', { n: unreceived }) : t('gmData.giftBox.unreceived') },
    { value: 'received' as GiftState, label: t('gmData.giftBox.received') },
    { value: 'expired' as GiftState, label: t('gmData.giftBox.expired') },
  ]
})

const columns = computed<DataTableColumn[]>(() => [
  { key: 'content', title: t('gmData.giftBox.content'), minWidth: 220 },
  { key: 'source', title: t('gmData.giftBox.source'), minWidth: 220 },
  { key: 'created', title: t('gmData.giftBox.created'), width: 130 },
  { key: 'limit', title: t('gmData.giftBox.limit'), width: 170 },
  { key: 'op', title: '', width: 100, align: 'right' },
])
const {
  keyword,
  active: searching,
  results,
  loading: searchLoading,
  error: searchError,
} = useNameSearch<Gift>({
  fetchAll: () => fetchAllPages(fetchPage),
  names: (gift) => {
    const rr = giftRef(gift)
    return [...(rr ? lookup.names(rr.kind, rr.id) : []), sourceLabel(gift), gift.message]
  },
  version: [page.version, state],
})
const shownGifts = computed(() => (searching.value ? results.value : list.items.value))
const rows = computed(() => shownGifts.value as unknown as Record<string, unknown>[])
const g = (row: Record<string, unknown>) => row as unknown as Gift

const giftRef = (gift: Gift) => lookup.rewardRef(gift.resourceType, gift.resourceId)
const giftName = (gift: Gift) => {
  const rr = giftRef(gift)
  return rr ? lookup.name(rr.kind, rr.id) : t('game.unnamed')
}
const giftImage = (gift: Gift) => {
  const rr = giftRef(gift)
  return rr ? lookup.image(rr.kind, rr.id) : ''
}

const sourceLabel = (gift: Gift) => {
  const kind = t(`gmData.giftBox.sourceKind.${gift.sourceKind}`)
  return gift.message ? `${kind} · ${gift.message}` : kind
}

const DAY = 86_400_000
const limitPill = (gift: Gift): { variant: 'success' | 'warning' | 'error' | 'neutral'; label: string } => {
  if (gift.state === 'received') return { variant: 'success', label: t('gmData.giftBox.receivedAt', { time: when(gift.receivedAt) }) }
  if (gift.state === 'expired') return { variant: 'neutral', label: t('gmData.giftBox.expiredAt', { time: dateTime(gift.limitTime) }) }
  if (!gift.limitTime) return { variant: 'neutral', label: t('gmData.giftBox.noLimit') }
  const left = gift.limitTime - Date.now()
  return {
    variant: left <= DAY ? 'error' : left <= 7 * DAY ? 'warning' : 'neutral',
    label: t('gmData.giftBox.until', { time: when(gift.limitTime) }),
  }
}

const canRevoke = computed(() => ctx.hasCommand('gift.revoke'))
const revocable = (gift: Gift) => canRevoke.value && gift.state === 'unreceived' && !!giftSourceName(gift.sourceKind)
const revoke = (gift: Gift) => {
  const source = giftSourceName(gift.sourceKind)
  if (!source) return
  page.run('gift.revoke', { sourceKind: source, giftIds: [gift.giftId] })
}
</script>

<style scoped lang="scss">
@use '../shared/views';
</style>
