<!-- 玩法 · 培育：全服玩家的培育结果，从新到旧（MGP produce 报表）。筛选：剧本（MGP 培育组目录）、偶像（MGP 角色目录）、
     日期（按服务器时区的自然日）、搜索玩家。与玩家详情的「培育记录」是同一张表，只是范围是全服。 -->
<template>
  <div class="pv">
    <div class="pv__line pr__filters">
      <div v-if="groups.length" class="pr__chips" role="tablist" :aria-label="t('gmData.produce.scenario')">
        <button type="button" role="tab" class="pr__chip" :class="{ 'is-on': !groupId }" :aria-selected="!groupId" @click="groupId = ''">
          {{ t('gmData.produce.allScenarios') }}
        </button>
        <button
          v-for="g in groups"
          :key="g.id"
          type="button"
          role="tab"
          class="pr__chip"
          :class="{ 'is-on': groupId === g.id }"
          :aria-selected="groupId === g.id"
          @click="groupId = g.id"
        >
          {{ g.name || t('game.unnamed') }}
        </button>
      </div>
      <AppDropdown align="start" :width="320">
        <template #trigger>
          <button type="button" class="pr__chip pr__chip--pick" :class="{ 'is-set': !!characterId }">
            <GameImage v-if="characterId && charaImage" :src="charaImage" class="pr__chip-img" width="22" height="22" />
            {{ characterId ? t('gmData.produce.idolOne', { name: lookup.name('character', characterId) }) : t('gmData.produce.allIdols') }}
            <component :is="menuStore.iconComponents['HOutline:ChevronUpDownIcon']" class="pr__chip-icon" />
          </button>
        </template>
        <template #default="{ close }">
          <div class="pr__pick">
            <button v-if="characterId" type="button" class="pr__pick-all" @click="characterId = ''; close()">
              {{ t('gmData.produce.allIdols') }}
            </button>
            <GmCatalogSearch
              kind="character"
              :selected-id="characterId"
              :placeholder="t('gmData.produce.searchIdol')"
              @pick="(e) => ((characterId = e.id), close())"
            />
          </div>
        </template>
      </AppDropdown>
      <div class="pr__select">
        <AppSelect v-model="range" :options="rangeOptions" />
      </div>
      <template v-if="range === 'custom'">
        <input v-model="from" class="pr__date" type="date" :aria-label="t('gmData.produce.from')" />
        <span class="pv__dim">—</span>
        <input v-model="to" class="pr__date" type="date" :aria-label="t('gmData.produce.to')" />
      </template>
      <span class="pv__spacer" />
      <label class="pr__search">
        <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" class="pr__search-icon" />
        <input v-model="keyword" type="search" :placeholder="t('gmData.modes.searchPlayer')" :aria-label="t('gmData.modes.searchPlayer')" />
      </label>
    </div>

    <div v-if="list.error.value" class="pv__error">{{ list.error.value }}</div>
    <ProduceTable :rows="list.items.value" :loading="list.loading.value" show-player :empty-text="t('gmData.produce.emptyAll')" />

    <div class="pv__foot">
      <span v-if="list.summary.value" class="m-num">
        {{ t('gmData.produce.summary', { today: num(list.summary.value.today), total: num(list.summary.value.total), n: num(list.total.value) }) }}
      </span>
      <Pagination
        v-if="list.total.value"
        v-model:page="list.page.value"
        v-model:page-size="list.pageSize.value"
        :total="list.total.value"
        @change="list.load()"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import { getGmReport, mgpQuery } from '@/api/gameGm'
import type { CatalogEntry, ManifestEntry } from '@/games/mgp'
import type { ProduceReport, ProduceEntry, ProduceReportSummary } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import GmCatalogSearch from '../../components/GmCatalogSearch.vue'
import { catalogImage } from '../../player/shared/game'
import { useCatalogLookup } from '../../player/shared/lookup'
import ProduceTable from '../../player/shared/ProduceTable.vue'
import { useGmFormat, zonedDayStart } from '../../player/shared/format'
import { usePagedList } from '../../player/shared/paging'

type Range = 'all' | 'today' | 'week' | 'month' | 'custom'

const props = defineProps<{ mode: ManifestEntry }>()

const { t } = useI18n()
const ctx = useGmContext()
const menuStore = useMenuStore()
const { num } = useGmFormat()
const lookup = useCatalogLookup()

const DAY = 86_400_000

const groupId = ref('')
const characterId = ref('')
const range = ref<Range>('all')
const from = ref('')
const to = ref('')
const keyword = ref('')

const bounds = computed((): { since?: number; until?: number } => {
  const tz = ctx.timezone.value
  const now = Date.now()
  switch (range.value) {
    case 'today':
      return { since: zonedDayStart(tz, now) }
    case 'week':
      return { since: zonedDayStart(tz, now - 6 * DAY) }
    case 'month':
      return { since: zonedDayStart(tz, now - 29 * DAY) }
    case 'custom':
      return {
        since: from.value ? zonedDayStart(tz, from.value) : undefined,
        until: to.value ? zonedDayStart(tz, to.value) + DAY : undefined,
      }
    default:
      return {}
  }
})

const list = usePagedList<ProduceEntry, ProduceReportSummary>(
  (q) =>
    getGmReport<ProduceReport>(
      ctx.serverId.value,
      props.mode.id,
      mgpQuery({
        produceGroupId: groupId.value,
        characterId: characterId.value,
        since: bounds.value.since,
        until: bounds.value.until,
        q: keyword.value.trim(),
        ...q,
      }),
    ),
  { pageSize: 20 },
)
watch([groupId, characterId, bounds], () => void list.reload(), { immediate: true })
const search = useDebounceFn(() => void list.reload(), 300)
watch(keyword, () => void search())

// 剧本 chips 来自 MGP 培育组目录（只有几个，读一页）；偶像从 MGP 角色目录搜索选择
const groups = shallowRef<CatalogEntry[]>([])
const loadGroups = async () => {
  try {
    groups.value = (await ctx.catalog('produceGroup', { limit: 50 })).items || []
  } catch {
    groups.value = []
  }
}
void loadGroups()
const charaImage = computed(() => (characterId.value ? catalogImage(ctx.game.value, 'character', { id: characterId.value }) : ''))
const rangeOptions = computed<{ label: string; value: Range }[]>(() => [
  { label: t('gmData.produce.rangeAll'), value: 'all' },
  { label: t('gmData.produce.rangeToday'), value: 'today' },
  { label: t('gmData.produce.rangeWeek'), value: 'week' },
  { label: t('gmData.produce.rangeMonth'), value: 'month' },
  { label: t('gmData.produce.rangeCustom'), value: 'custom' },
])
</script>

<style scoped lang="scss">
@use '../../player/shared/views';
.pr__chips {
  display: flex;
  gap: 8px;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.pr__chips::-webkit-scrollbar {
  display: none;
}
.pr__chip {
  flex: none;
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
.pr__chip:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.pr__chip.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.pr__select {
  width: 170px;
}
.pr__chip--pick {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  max-width: 240px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pr__chip--pick.is-set {
  border-color: var(--m-ink);
  color: var(--el-text-color-primary);
}
.pr__chip-img {
  flex: none;
  width: 22px;
  height: 22px;
  border-radius: 50%;
}
.pr__chip-icon {
  flex: none;
  width: 16px;
  height: 16px;
}
.pr__pick {
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 10px;
}
.pr__pick-all {
  min-height: 40px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
  font: inherit;
  font-weight: 700;
  cursor: pointer;
}
.pr__date {
  min-height: 40px;
  padding: 0 12px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 13px;
}
.pr__search {
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
.pr__search:focus-within {
  border-color: var(--m-p);
}
.pr__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
}
.pr__search-icon {
  flex: none;
  width: 16px;
  height: 16px;
  color: var(--el-text-color-secondary);
}
@media (max-width: 760px) {
  .pr__select {
    flex: 1 1 140px;
    width: auto;
  }
  .pr__search {
    flex: 1 1 100%;
  }
  .pr__date {
    flex: 1 1 120px;
    min-height: 44px;
  }
}
</style>
