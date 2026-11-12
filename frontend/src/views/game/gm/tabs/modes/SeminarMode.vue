<!-- 玩法 · 研修：各研修组（研修统计接口，按顺序）：试题数、全部试题的钻石奖励、全部通关的玩家占比；
     试题名来自研修目录（按组 ID 一次查询）。 -->
<template>
  <div class="pv">
    <div v-if="error" class="pv__error">{{ error }}</div>
    <div v-if="loading && !groups" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState v-else-if="groups && !groups.length" icon="HOutline:BuildingLibraryIcon" :title="t('gmData.seminar.noGroups')" />
    <template v-else-if="groups">
      <div class="pv__line">
        {{ t('gmData.seminar.overview', { groups: groups.length, exams: totalExams, jewel: num(totalJewel) }) }}
      </div>
      <div class="sm">
        <article v-for="(g, i) in groups" :key="g.groupId" class="sm__card">
          <span class="sm__head">
            <span class="sm__order m-sticker" :class="STICKERS[i % STICKERS.length]">{{ g.order || i + 1 }}</span>
            <span class="sm__title">
              <span class="sm__name">{{ g.name || ctx.label('seminar', g.groupId) }}</span>
            </span>
          </span>
          <span class="pv__tags sm__tags">
            <span class="pv__tag m-num">{{ t('gmData.seminar.exams', { n: g.exams }) }}</span>
            <span class="pv__tag sm__jewel">
              <GameImage v-if="jewelImage" :src="jewelImage" class="sm__jewel-img" width="16" height="16" />
              <span class="m-num">{{ num(g.jewelReward) }}</span>
            </span>
          </span>
          <span v-if="examNames(g.groupId)" class="pv__dim sm__exams" :title="examNames(g.groupId)">{{ examNames(g.groupId) }}</span>
          <span class="sm__progress">
            <span class="sm__bar" role="progressbar" :aria-valuenow="pct(g.clearedPlayers, g.players)" aria-valuemin="0" aria-valuemax="100">
              <span :style="{ width: `${pct(g.clearedPlayers, g.players)}%` }" />
            </span>
            <span class="pv__dim m-num">{{ t('gmData.seminar.cleared', { pct: pct(g.clearedPlayers, g.players), n: num(g.clearedPlayers), total: num(g.players) }) }}</span>
          </span>
        </article>
      </div>
    </template>
  </div>
</template>

<script setup lang="ts">
import { computed, reactive, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { getSeminarStats } from '@/api/gakumas'
import { getRequestErrorMessage } from '@/utils/request'
import type { CatalogEntry } from '@/games/mgp'
import type { SeminarGroupStat } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { catalogImage } from '../../player/shared/game'
import { useGmFormat } from '../../player/shared/format'

const STICKERS = ['m-sticker--p', 'm-sticker--a2', 'm-sticker--a', 'm-sticker--ok']

const { t } = useI18n()
const ctx = useGmContext()
const { num, pct } = useGmFormat()

const groups = shallowRef<SeminarGroupStat[]>()
const loading = ref(false)
const error = ref('')

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    const data = await getSeminarStats(ctx.serverId.value)
    groups.value = [...data.groups].sort((a, b) => a.order - b.order)
    void loadExams(groups.value.map((g) => g.groupId))
  } catch (e) {
    error.value = getRequestErrorMessage(e)
  } finally {
    loading.value = false
  }
}

const totalExams = computed(() => (groups.value || []).reduce((s, g) => s + g.exams, 0))
const totalJewel = computed(() => (groups.value || []).reduce((s, g) => s + g.jewelReward, 0))
const jewelImage = computed(() => catalogImage(ctx.game.value, 'currency', { id: 'jewelFree' }))

// 试题名：研修目录按组 ID 一次查出这些组，其中 parentId 为组 ID 的条目是该组的试题
const exams = reactive(new Map<string, CatalogEntry[]>())
const loadExams = async (groupIds: string[]) => {
  if (!groupIds.length) return
  try {
    const page = await ctx.catalog('seminar', { ids: groupIds, limit: groupIds.length })
    for (const id of groupIds) exams.set(id, page.items.filter((e) => e.parentId === id))
  } catch {
    for (const id of groupIds) exams.set(id, [])
  }
}
const examNames = (groupId: string) => (exams.get(groupId) || []).map((e) => e.name || t('game.unnamed')).join('、')

void load()
</script>

<style scoped lang="scss">
@use '../../player/shared/views';
.sm {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.sm__card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 18px 20px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  box-shadow: var(--app-shadow-card);
  min-width: 0;
}
.sm__head {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.sm__order {
  width: 38px;
  height: 38px;
  font-size: 14px;
  transform: rotate(-4deg);
}
.sm__title {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.sm__name {
  font-size: 16px;
  font-weight: 800;
}
.sm__tags {
  padding: 0;
}
.sm__jewel {
  gap: 4px;
}
.sm__jewel-img {
  width: 16px;
  height: 16px;
  object-fit: contain;
  background: transparent;
}
.sm__exams {
  display: -webkit-box;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
  overflow: hidden;
}
.sm__progress {
  display: flex;
  flex-direction: column;
  gap: 6px;
  margin-top: auto;
}
.sm__bar {
  height: 8px;
  border-radius: 999px;
  background: var(--m-s2);
  overflow: hidden;
}
.sm__bar span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--m-g);
}
</style>
