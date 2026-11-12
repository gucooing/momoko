<!-- 偶像卡：按获得时间从新到旧的卡面网格（当前皮肤方图、稀有度、皮肤名与角色、才能开花、上限解放、Prima Stella、皮肤数），
     点开看详情（持有的皮肤与这张卡的培育记录，培育记录再点开看记忆详情）。名称按角色分组查目录（一个角色一次）。
     稀有度、角色（头像快捷筛选）与搜索都由服务端筛选分页：稀有度与角色是表里的列，胶囊与头像上的张数来自列表的 summary；
     卡名、皮肤名、角色名解析成偶像卡 ID，解析不出时关键字交给服务端模糊匹配。 -->
<template>
  <div class="pv">
    <ListToolbar
      v-model:status="rarity"
      v-model:keyword="search.keyword.value"
      :chips="chips"
      :chips-label="t('gmData.cards.rarity')"
      :placeholder="t('gmData.cards.searchIdol')"
    />
    <div v-if="charaChips.length" class="ic__charas" role="group" :aria-label="t('gmData.cards.character')">
      <button
        v-for="c in charaChips"
        :key="c.id"
        type="button"
        class="ic__chara"
        :class="{ 'is-on': chara === c.id }"
        :aria-pressed="chara === c.id"
        :title="c.name"
        @click="chara = chara === c.id ? '' : c.id"
      >
        <GameImage :src="c.image" :alt="c.name" class="ic__chara-img" />
        <span class="ic__chara-n m-num">{{ c.count }}</span>
      </button>
    </div>

    <div v-if="list.error.value || search.error.value" class="pv__error">{{ list.error.value || search.error.value }}</div>
    <div v-if="search.loading.value" class="pv__hint">{{ t('gmData.common.searching') }}</div>
    <div v-else-if="list.loading.value && !list.items.value.length" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState
      v-else-if="!shown.length"
      icon="HOutline:Squares2X2Icon"
      :title="filtered ? t('gmData.common.noMatch') : t('gmData.cards.noIdol')"
    />
    <div v-else class="pv__grid">
      <button v-for="c in shown" :key="c.idolCardId" type="button" class="pv__card pv__card--btn" @click="opened = c">
        <span class="pv__card-art">
          <GameImage :src="image(c)" :alt="cardName(c)" class="pv__card-img" />
          <RarityBadge :rarity="c.rarity" class="pv__card-corner" />
        </span>
        <span class="pv__card-body">
          <span class="pv__strong pv__ellipsis" :title="cardName(c)">{{ cardName(c) }}</span>
          <span class="pv__dim pv__ellipsis">{{ charaName(c) }}</span>
        </span>
        <span class="pv__tags">
          <span class="pv__tag">{{ t('gmData.cards.potential', { n: c.potentialRank }) }}</span>
          <span class="pv__tag">{{ t('gmData.cards.limit', { n: c.levelLimitRank }) }}</span>
          <span v-if="c.primaStella" class="pv__tag is-g">Prima Stella</span>
          <span v-if="(c.skinIds?.length || 0) > 1" class="pv__tag">{{ t('gmData.cards.skins', { n: c.skinIds?.length || 0 }) }}</span>
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

    <IdolCardDetailDialog v-model:card="opened" @memory="openMemory" />
    <MemoryDetailDialog v-model:memory-id="memoryId" />
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { listPlayerIdolCards } from '@/api/gakumas'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import type { IdolCard } from '@/games/gakumas/types'
import type { IdolCardCounts } from '@/types/v1/gakumas/types'
import { useGmContext } from '../../context'
import { usePlayerPage } from '../context'
import IdolCardDetailDialog from '../shared/IdolCardDetailDialog.vue'
import MemoryDetailDialog from '../shared/MemoryDetailDialog.vue'
import RarityBadge from '../shared/RarityBadge.vue'
import { catalogImage, rarityLabel } from '../shared/game'
import { useCatalogLookup } from '../shared/lookup'
import { usePagedList, type PageQuery } from '../shared/paging'
import { catalogEntries, idolCardIds, useIdSearch } from '../shared/search'

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const lookup = useCatalogLookup()

const rarity = ref<number>(0)
const chara = ref('')
const search = useIdSearch((q) => idolCardIds(ctx, q), { serverQuery: true })
const opened = ref<IdolCard | null>(null)
const memoryId = ref('')
const openMemory = (id: string) => {
  opened.value = null
  memoryId.value = id
}

const fetchPage = async (q: PageQuery) => {
  if (search.none.value) return { items: [], nextCursor: '', total: 0 }
  return listPlayerIdolCards({
    serverId: ctx.serverId.value,
    pid: page.pid.value,
    q: search.q.value,
    characterId: chara.value,
    rarity: rarity.value,
    idolCardIds: search.ids.value || [],
    cursor: q.cursor,
    limit: q.limit,
  })
}
const list = usePagedList<IdolCard, IdolCardCounts>(fetchPage, { pageSize: 20 })
watch([page.version, rarity, chara, search.ids, search.q], () => void list.reload(), { immediate: true })

// 稀有度与角色的张数来自列表的 summary（各自带上另一项与搜索的筛选）
const rarityCounts = computed(() => list.summary.value?.rarities || {})
const filtered = computed(() => search.active.value || !!rarity.value || !!chara.value)
const chips = computed<ToolbarChip<number>[]>(() => {
  const counts = rarityCounts.value
  const all = Object.values(counts).reduce((n, v) => n + v, 0)
  return [
    { label: t('gmData.common.all'), value: 0, count: list.summary.value ? all : null },
    ...[3, 2, 1].filter((r) => counts[r] || rarity.value === r).map((r) => ({ label: rarityLabel(r), value: r, count: counts[r] ?? 0 })),
  ]
})

// 角色快捷筛选：角色目录的顺序，只列出有卡的角色（与选中的角色）
const characters = ref<{ id: string }[]>([])
void catalogEntries(ctx, 'character', {}).then((list) => (characters.value = list))
const charaChips = computed(() => {
  const counts = list.summary.value?.characters || {}
  return characters.value
    .filter((c) => counts[c.id] || chara.value === c.id)
    .map((c) => ({
      id: c.id,
      name: lookup.name('character', c.id),
      image: catalogImage(ctx.game.value, 'character', { id: c.id }),
      count: counts[c.id] ?? 0,
    }))
})

const skinEntry = (c: IdolCard) => lookup.group('idolCardSkin', c.characterId)?.find((e) => e.id === c.skinId)
const charaName = (c: IdolCard) => (c.characterId ? lookup.name('character', c.characterId) : '')
const cardName = (c: IdolCard) => (c.skinId ? lookup.name('idolCardSkin', c.skinId) : lookup.name('idolCard', c.idolCardId))
const image = (c: IdolCard) => catalogImage(ctx.game.value, 'idolCardSkin', skinEntry(c) || { id: c.skinId || c.idolCardId })

const shown = computed(() => list.items.value)
</script>

<style scoped lang="scss">
@use '../shared/views';

.ic__charas {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.ic__chara {
  position: relative;
  width: 44px;
  height: 44px;
  padding: 0;
  border: 2px solid transparent;
  border-radius: 50%;
  background: var(--el-fill-color-light);
  cursor: pointer;
}
.ic__chara.is-on {
  border-color: var(--el-color-primary);
}
.ic__chara-img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
}
.ic__chara-n {
  position: absolute;
  right: -6px;
  bottom: -4px;
  min-width: 18px;
  padding: 0 4px;
  border-radius: 999px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
  font-size: 11px;
  line-height: 16px;
  text-align: center;
}
</style>
