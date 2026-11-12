<!-- 玩法 · 竞赛：当前赛季全部段位的排名（MGP pvp 报表：段位高、Rate 高者在前；搜索玩家、分页）+ 选中玩家当前配置的阵容
     （MGP 玩家视图 pvp-unit，只读）。不显示段位规则、奖励与结算。 -->
<template>
  <div class="pv">
    <div class="pv__line">
      <template v-if="season">
        <StatusPill :variant="seasonState.variant" :label="seasonState.label" />
        <span class="m-num">{{ dateShort(season.startTime) }} → {{ dateShort(season.endTime) }}</span>
        <i18n-t keypath="gmData.pvp.seasonLine" tag="span" scope="global">
          <template #n><b class="m-num">{{ num(season.participants) }}</b></template>
          <template #battles><b class="m-num">{{ num(season.battlesToday) }}</b></template>
        </i18n-t>
      </template>
      <span class="pv__spacer" />
      <label class="pm__search">
        <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" class="pm__search-icon" />
        <input v-model="keyword" type="search" :placeholder="t('gmData.modes.searchPlayer')" :aria-label="t('gmData.modes.searchPlayer')" />
      </label>
    </div>

    <div v-if="list.error.value" class="pv__error">{{ list.error.value }}</div>
    <div v-if="list.loading.value && !list.loaded.value" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState v-else-if="list.loaded.value && !season && !keyword" icon="HOutline:TrophyIcon" :title="t('gmData.pvp.noSeason')" />

    <div v-else class="pm">
      <section class="pm__rank">
        <div class="pm__row pm__row--head" aria-hidden="true">
          <span>{{ t('gmData.pvp.rank') }}</span>
          <span />
          <span>{{ t('gmData.modes.player') }}</span>
          <span class="is-right">Rate</span>
          <span class="is-right">{{ t('gmData.pvp.winLose') }}</span>
        </div>
        <EmptyState v-if="!list.items.value.length && !list.loading.value" icon="HOutline:UsersIcon" :title="keyword ? t('gmData.common.noMatch') : t('gmData.pvp.noEntries')" />
        <button
          v-for="r in list.items.value"
          :key="r.pid"
          type="button"
          class="pm__row"
          :class="{ 'is-on': r.pid === selected?.pid }"
          :aria-pressed="r.pid === selected?.pid"
          @click="select(r)"
        >
          <span class="pm__no m-num" :class="{ 'is-top': r.rank <= 3 }">{{ r.rank }}</span>
          <PlayerAvatar :idol-card-id="r.avatarIdolCardId" :name="r.name" :size="40" />
          <span class="pm__who">
            <span class="pv__strong pv__ellipsis">{{ r.name || t('gm.player.noName') }}</span>
            <span class="pv__dim">{{ t('gmData.pvp.grade', { n: r.grade }) }}</span>
          </span>
          <span class="pm__rate m-num is-right">{{ num(r.rate) }}</span>
          <span class="pv__dim m-num is-right">{{ num(r.winCount) }} / {{ num(r.playCount - r.winCount) }}</span>
        </button>
        <div v-if="list.total.value" class="pm__pager">
          <span class="pv__dim m-num">{{ t('gmData.modes.range', { from: rangeFrom, to: rangeTo, total: num(list.total.value) }) }}</span>
          <Pagination v-model:page="list.page.value" v-model:page-size="list.pageSize.value" :total="list.total.value" @change="list.load()" />
        </div>
      </section>

      <AppPanel class="pm__unit">
        <template #header>
          <div v-if="selected" class="pm__unit-head">
            <PlayerAvatar :idol-card-id="selected.avatarIdolCardId" :name="selected.name" :size="52" />
            <span class="pm__unit-title">
              <span class="pm__unit-name">{{ t('gmData.pvp.lineupOf', { name: selected.name || t('gm.player.noName') }) }}</span>
              <span class="pv__dim">
                {{ t('gmData.pvp.lineupSub', { rank: selected.rank, rate: num(selected.rate) }) }}
                <template v-if="unit?.unitUpdatedTime"> · {{ t('gmData.pvp.edited', { time: when(unit.unitUpdatedTime) }) }}</template>
              </span>
            </span>
            <span class="pm__power">
              <span class="pv__dim">{{ t('gmData.pvp.powerLabel') }}</span>
              <span class="m-num">{{ num(unit?.unitPower ?? selected.power) }}</span>
            </span>
            <AppIconButton icon="HOutline:ChevronRightIcon" :label="t('gmData.modes.viewPlayer', { name: selected.name })" :box="40" @click="openPlayer(selected.pid)" />
          </div>
        </template>
        <EmptyState v-if="!selected" icon="HOutline:UserGroupIcon" :title="t('gmData.pvp.pickPlayer')" />
        <template v-else>
          <div v-if="unitError" class="pv__error">{{ unitError }}</div>
          <div v-if="unitLoading && !unit" class="pv__hint">{{ t('gmData.common.loading') }}</div>
          <PvpStages v-else-if="unit?.stages?.length" :stages="unit.stages" />
          <EmptyState v-else-if="unit" icon="HOutline:UserGroupIcon" :title="t('gmData.pvp.noLineup')" />
          <p class="pv__hint pm__note">{{ t('gmData.pvp.lineupNote') }}</p>
        </template>
      </AppPanel>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { useDebounceFn } from '@vueuse/core'
import dayjs from 'dayjs'
import { getGmPlayerView, getGmReport, mgpQuery } from '@/api/gameGm'
import { getRequestErrorMessage } from '@/utils/request'
import type { ManifestEntry } from '@/games/mgp'
import type { PvpEntry, PvpReport, PvpSeasonSummary, PvpUnit } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import PlayerAvatar from '../../player/shared/PlayerAvatar.vue'
import PvpStages from '../../player/shared/PvpStages.vue'
import { useGmFormat } from '../../player/shared/format'
import { usePagedList } from '../../player/shared/paging'

const props = defineProps<{ mode: ManifestEntry }>()

const { t } = useI18n()
const router = useRouter()
const ctx = useGmContext()
const menuStore = useMenuStore()
const { num, when } = useGmFormat()

const keyword = ref('')
const list = usePagedList<PvpEntry, PvpSeasonSummary>(
  (q) => getGmReport<PvpReport>(ctx.serverId.value, props.mode.id, mgpQuery({ q: keyword.value.trim(), ...q })),
  { pageSize: 20 },
)
void list.load()
const search = useDebounceFn(() => void list.reload(), 300)
watch(keyword, () => void search())

const season = computed(() => list.summary.value)
const dateShort = (ms: number) => (ms ? dayjs(ms).format('MM-DD HH:mm') : '—')
const seasonState = computed((): { variant: 'success' | 'primary' | 'neutral'; label: string } => {
  const s = season.value
  const now = Date.now()
  if (!s) return { variant: 'neutral', label: '' }
  if (now < s.startTime) return { variant: 'primary', label: t('gmData.pvp.preparing') }
  if (now < s.endTime) return { variant: 'success', label: t('gmData.pvp.running') }
  return { variant: 'neutral', label: t('gmData.pvp.ended') }
})

const rangeFrom = computed(() => (list.total.value ? (list.page.value - 1) * list.pageSize.value + 1 : 0))
const rangeTo = computed(() => (list.page.value - 1) * list.pageSize.value + list.items.value.length)

// ---------- 阵容 ----------
const selected = shallowRef<PvpEntry>()
const unit = shallowRef<PvpUnit>()
const unitLoading = ref(false)
const unitError = ref('')
const unitView = computed(() => (ctx.hasView('pvp-unit') ? 'pvp-unit' : ctx.hasView('pvp') ? 'pvp' : ''))
let unitSeq = 0

const select = async (r: PvpEntry) => {
  selected.value = r
  unit.value = undefined
  unitError.value = ''
  if (!unitView.value) return
  const id = ++unitSeq
  unitLoading.value = true
  try {
    const data = await getGmPlayerView<PvpUnit>(ctx.serverId.value, r.pid, unitView.value)
    if (id === unitSeq) unit.value = data
  } catch (e) {
    if (id === unitSeq) unitError.value = getRequestErrorMessage(e)
  } finally {
    if (id === unitSeq) unitLoading.value = false
  }
}

// 第一次读到排名时默认选第一名
watch(
  () => list.items.value,
  (items) => {
    const first = items[0]
    if (!selected.value && first) void select(first)
  },
)

const openPlayer = (pid: string) => {
  void router.push(`/game/gm/${encodeURIComponent(ctx.serverId.value)}/players/${encodeURIComponent(pid)}`)
}
</script>

<style scoped lang="scss">
@use '../../player/shared/views';
.pm__search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 280px;
  min-width: 180px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.pm__search:focus-within {
  border-color: var(--m-p);
}
.pm__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
}
.pm__search-icon {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
.pm {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}
.pm__rank {
  flex: 2 1 420px;
  min-width: 0;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-card);
  overflow: hidden;
}
.pm__row {
  display: grid;
  grid-template-columns: 36px 40px minmax(0, 1fr) 76px 84px;
  gap: 12px;
  align-items: center;
  width: 100%;
  min-height: 60px;
  padding: 0 14px;
  border: 0;
  border-top: 1px solid var(--el-border-color-lighter);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.pm__row:hover {
  background: var(--el-fill-color-light);
}
.pm__row.is-on {
  background: var(--m-gSoft);
  box-shadow: inset 3px 0 0 var(--m-g);
}
.pm__row--head {
  min-height: 40px;
  border-top: 0;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
  cursor: default;
}
.pm__row--head:hover {
  background: transparent;
}
.is-right {
  text-align: right;
}
.pm__no {
  text-align: center;
  font-size: 17px;
  font-weight: 800;
  color: var(--el-text-color-secondary);
}
.pm__no.is-top {
  color: var(--m-gInk);
}
.pm__who {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.pm__rate {
  font-size: 16px;
  font-weight: 800;
}
.pm__pager {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px 14px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.pm__unit {
  flex: 3 1 520px;
  min-width: 0;
}
.pm__unit-head {
  display: flex;
  align-items: center;
  gap: 14px;
  width: 100%;
  min-width: 0;
}
.pm__unit-title {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.pm__unit-name {
  font-size: 20px;
  font-weight: 800;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pm__power {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
}
.pm__power .m-num {
  font-size: 22px;
  font-weight: 800;
}
.pm__note {
  margin: 14px 0 0;
}
@media (max-width: 760px) {
  .pm__row {
    grid-template-columns: 28px 40px minmax(0, 1fr) 64px;
    gap: 10px;
  }
  .pm__row > :last-child {
    display: none;
  }
  .pm__unit-head {
    flex-wrap: wrap;
  }
  .pm__unit-name {
    font-size: 17px;
  }
}
</style>
