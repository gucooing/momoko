<!-- 玩法 · 塔：塔列表来自塔目录（ID 与层数），选一座塔看当前排行季每个玩家到达的最高层与到达时间（塔排名接口，搜索玩家、分页）。 -->
<template>
  <div class="pv">
    <div class="pv__line">
      <div v-if="towers.length" class="tw__chips" role="tablist" :aria-label="t('gmData.tower.towers')">
        <button
          v-for="tw in towers"
          :key="tw.id"
          type="button"
          role="tab"
          class="tw__chip"
          :class="{ 'is-on': tw.id === towerId }"
          :aria-selected="tw.id === towerId"
          @click="towerId = tw.id"
        >
          {{ tw.name || t('game.unnamed') }}
          <span v-if="tw.type" class="tw__count m-num">{{ t('gmData.tower.layers', { n: tw.type }) }}</span>
        </button>
      </div>
      <span v-if="list.summary.value" class="pv__hint">
        {{ t('gmData.tower.season', { date: date(list.summary.value) }) }}
      </span>
      <span class="pv__spacer" />
      <label class="tw__search">
        <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" class="tw__search-icon" />
        <input v-model="keyword" type="search" :placeholder="t('gmData.modes.searchPlayer')" :aria-label="t('gmData.modes.searchPlayer')" />
      </label>
    </div>

    <div v-if="towersError" class="pv__error">{{ towersError }}</div>
    <div v-if="list.error.value" class="pv__error">{{ list.error.value }}</div>
    <EmptyState v-if="towersLoaded && !towers.length" icon="HOutline:ChartBarIcon" :title="t('gmData.tower.noTowers')" />

    <section v-else-if="towerId" class="tw">
      <div class="tw__row tw__row--head" aria-hidden="true">
        <span>{{ t('gmData.pvp.rank') }}</span>
        <span />
        <span>{{ t('gmData.modes.player') }}</span>
        <span>{{ t('gmData.tower.layer') }}</span>
        <span class="is-right">{{ t('gmData.tower.reachedAt') }}</span>
      </div>
      <div v-if="list.loading.value && !list.items.value.length" class="pv__hint tw__note">{{ t('gmData.common.loading') }}</div>
      <EmptyState
        v-else-if="!list.items.value.length"
        icon="HOutline:ChartBarIcon"
        :title="keyword ? t('gmData.common.noMatch') : t('gmData.tower.empty')"
      />
      <RouterLink v-for="r in list.items.value" :key="r.pid" class="tw__row" :to="playerLink(r.pid)">
        <span class="tw__no m-num" :class="{ 'is-top': r.rank <= 3 }">{{ r.rank }}</span>
        <PlayerAvatar :idol-card-id="r.avatarIdolCardId" :name="r.name" :size="40" />
        <span class="pv__strong pv__ellipsis">{{ r.name || t('gm.player.noName') }}</span>
        <span class="tw__layer m-num">{{ t('gmData.tower.layerN', { n: r.layer }) }}</span>
        <span class="pv__dim m-num is-right">{{ when(r.reachedAt) }}</span>
      </RouterLink>
      <div v-if="list.total.value" class="tw__pager">
        <span class="pv__dim m-num">{{ t('gmData.modes.range', { from: rangeFrom, to: rangeTo, total: num(list.total.value) }) }}</span>
        <Pagination v-model:page="list.page.value" v-model:page-size="list.pageSize.value" :total="list.total.value" @change="list.load()" />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { getTowerRanking } from '@/api/gakumas'
import { getRequestErrorMessage } from '@/utils/request'
import type { CatalogEntry } from '@/games/mgp'
import type { TowerRank } from '@/types/v1/gakumas/types'
import { useGmContext } from '../../context'
import PlayerAvatar from '../../player/shared/PlayerAvatar.vue'
import { useGmFormat } from '../../player/shared/format'
import { usePagedList } from '../../player/shared/paging'

const { t } = useI18n()
const ctx = useGmContext()
const menuStore = useMenuStore()
const { num, when, date } = useGmFormat()

// 塔目录（只有几座塔，读一页）
const towers = shallowRef<CatalogEntry[]>([])
const towersLoaded = ref(false)
const towersError = ref('')
const towerId = ref('')
const loadTowers = async () => {
  try {
    const page = await ctx.catalog('tower', { limit: 100 })
    towers.value = page.items || []
    if (!towerId.value) towerId.value = towers.value[0]?.id || ''
  } catch (e) {
    towersError.value = getRequestErrorMessage(e)
  } finally {
    towersLoaded.value = true
  }
}
void loadTowers()

const keyword = ref('')
// summary 为排行季开始时间（不在排行季为 0）
const list = usePagedList<TowerRank, number>(
  async (q) => {
    const page = await getTowerRanking({ serverId: ctx.serverId.value, towerId: towerId.value, q: keyword.value.trim(), cursor: q.cursor, limit: q.limit })
    return { items: page.items, nextCursor: page.nextCursor, total: page.total, summary: page.seasonStart }
  },
  { pageSize: 20 },
)
watch(towerId, (id) => {
  if (id) void list.reload()
})
const search = useDebounceFn(() => {
  if (towerId.value) void list.reload()
}, 300)
watch(keyword, () => void search())

const rangeFrom = computed(() => (list.total.value ? (list.page.value - 1) * list.pageSize.value + 1 : 0))
const rangeTo = computed(() => (list.page.value - 1) * list.pageSize.value + list.items.value.length)
const playerLink = (pid: string) => `/game/gm/${encodeURIComponent(ctx.serverId.value)}/players/${encodeURIComponent(pid)}`
</script>

<style scoped lang="scss">
@use '../../player/shared/views';
.tw__chips {
  display: flex;
  gap: 8px;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.tw__chips::-webkit-scrollbar {
  display: none;
}
.tw__chip {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 14px;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.tw__chip.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.tw__count {
  font-size: 12px;
  opacity: 0.75;
}
.tw__search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 260px;
  min-width: 180px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.tw__search:focus-within {
  border-color: var(--m-p);
}
.tw__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
}
.tw__search-icon {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.tw {
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-card);
  overflow: hidden;
}
.tw__row {
  display: grid;
  grid-template-columns: 36px 40px minmax(0, 1fr) 120px 150px;
  gap: 12px;
  align-items: center;
  min-height: 60px;
  padding: 0 14px;
  border-top: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-primary);
  text-decoration: none;
}
a.tw__row:hover {
  background: var(--el-fill-color-light);
}
.tw__row--head {
  min-height: 40px;
  border-top: 0;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.tw__note {
  padding: 12px 14px;
}
.is-right {
  text-align: right;
}
.tw__no {
  text-align: center;
  font-size: 17px;
  font-weight: 800;
  color: var(--el-text-color-secondary);
}
.tw__no.is-top {
  color: var(--m-gInk);
}
.tw__layer {
  font-weight: 800;
}
.tw__pager {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 14px;
  border-top: 1px solid var(--el-border-color-lighter);
}
@media (max-width: 760px) {
  .tw__row {
    grid-template-columns: 28px 40px minmax(0, 1fr) 76px;
    gap: 10px;
  }
  .tw__row > :last-child {
    display: none;
  }
  .tw__search {
    flex: 1 1 100%;
  }
}
</style>
