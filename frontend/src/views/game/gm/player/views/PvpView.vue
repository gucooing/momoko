<!-- 竞赛：当前配置的阵容（各阶段编成力、主 / 副记忆）与本赛季（名次、Rate、段位、胜负、今日剩余次数、保留的对战记录）。
     数据来自 MGP 玩家视图 pvp（只有 pvp-unit 时只显示阵容），只读。 -->
<template>
  <div class="pv">
    <div v-if="error" class="pv__error">{{ error }}</div>
    <div v-if="loading && !data" class="pv__hint">{{ t('gmData.common.loading') }}</div>
    <EmptyState v-else-if="data && !data.seasonId" icon="HOutline:TrophyIcon" :title="t('gmData.pvp.noSeason')" />

    <div v-else-if="data" class="pvp">
      <AppPanel class="pvp__unit" :title="t('gmData.pvp.lineup')">
        <template #actions>
          <span class="pv__dim">{{ data.unitUpdatedTime ? t('gmData.pvp.edited', { time: when(data.unitUpdatedTime) }) : '' }}</span>
          <span class="pvp__power m-num">{{ t('gmData.pvp.power', { n: num(data.unitPower) }) }}</span>
        </template>
        <PvpStages v-if="data.stages?.length" :stages="data.stages" />
        <EmptyState v-else icon="HOutline:UserGroupIcon" :title="t('gmData.pvp.noLineup')" />
      </AppPanel>

      <AppPanel class="pvp__season" :title="t('gmData.pvp.season')">
        <template v-if="modesLink" #actions>
          <RouterLink class="pvp__link" :to="modesLink">{{ t('gmData.pvp.ranking') }}</RouterLink>
        </template>
        <div class="pvp__stats">
          <div v-for="s in stats" :key="s.key" class="pvp__stat">
            <span class="pv__dim">{{ s.label }}</span>
            <span class="pvp__stat-v m-num">{{ s.value }}</span>
          </div>
        </div>
        <template v-if="battles">
          <EmptyState v-if="!battles.length" icon="HOutline:TrophyIcon" :title="t('gmData.pvp.noBattles')" />
          <ul v-else class="pvp__battles">
            <li v-for="(b, i) in battles" :key="i" class="pvp__battle">
              <StatusPill :variant="b.isWin ? 'success' : 'error'" :label="b.isWin ? t('gmData.pvp.win') : t('gmData.pvp.lose')" />
              <RouterLink v-if="b.rivalPid" class="pvp__rival" :to="playerLink(b.rivalPid)">
                {{ b.rivalName || (b.rivalDisplayId ? t('gm.player.noName') : t('gmData.pvp.npc')) }}
              </RouterLink>
              <span v-else class="pvp__rival">{{ b.rivalName || t('gmData.pvp.npc') }}</span>
              <span class="pv__dim">{{ t('gmData.pvp.grade', { n: b.grade }) }}</span>
              <span class="pvp__delta m-num" :class="b.earnedRate >= 0 ? 'is-up' : 'is-down'">{{ delta(b.earnedRate) }}</span>
              <span class="pvp__at pv__dim m-num">{{ when(b.playTime) }}</span>
            </li>
          </ul>
        </template>
      </AppPanel>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, shallowRef, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { getGmPlayerView } from '@/api/gameGm'
import { getRequestErrorMessage } from '@/utils/request'
import type { ManifestView } from '@/games/mgp'
import type { PvpView } from '@/games/gakumas/types'
import { useGmContext } from '../../context'
import { GM_TABS } from '../../tabs'
import { usePlayerPage } from '../context'
import PvpStages from '../shared/PvpStages.vue'
import { useGmFormat } from '../shared/format'

const props = defineProps<{ view?: ManifestView }>()

const { t } = useI18n()
const ctx = useGmContext()
const page = usePlayerPage()
const { num, when, delta } = useGmFormat()

// pvp 视图带名次与对战记录；pvp-unit 只有赛季条目与阵容
const data = shallowRef<Partial<PvpView>>()
const loading = ref(false)
const error = ref('')

const load = async () => {
  loading.value = true
  error.value = ''
  try {
    data.value = await getGmPlayerView<PvpView>(ctx.serverId.value, page.pid.value, props.view?.id || 'pvp')
  } catch (e) {
    error.value = getRequestErrorMessage(e)
  } finally {
    loading.value = false
  }
}
watch(page.version, () => void load(), { immediate: true })

const battles = computed(() => data.value?.battles)

const stats = computed(() => {
  const d = data.value
  if (!d) return []
  if (!d.entered) return [{ key: 'entered', label: t('gmData.pvp.status'), value: t('gmData.pvp.notEntered') }]
  const out = [
    ...(d.rank !== undefined ? [{ key: 'rank', label: t('gmData.pvp.rank'), value: d.rank ? num(d.rank) : '—' }] : []),
    { key: 'rate', label: 'Rate', value: num(d.rate) },
    { key: 'wl', label: t('gmData.pvp.winLose'), value: `${num(d.winCount)} / ${num((d.playCount || 0) - (d.winCount || 0))}` },
    { key: 'grade', label: t('gmData.pvp.gradeLabel'), value: t('gmData.pvp.gradeBest', { n: d.grade, best: d.bestGrade }) },
    { key: 'plays', label: t('gmData.pvp.remaining'), value: `${num(d.remainingPlays)} / ${num(d.maxDailyPlays)}` },
  ]
  return out
})

// 「全服排名」链到玩法标签（这台服务器有玩法时）
const modesLink = computed(() => {
  const tab = GM_TABS.find((x) => x.key === 'modes')
  if (!tab || !tab.visible(ctx)) return undefined
  return { path: `/game/gm/${encodeURIComponent(ctx.serverId.value)}`, query: { tab: 'modes' } }
})
const playerLink = (pid: string) => `/game/gm/${encodeURIComponent(ctx.serverId.value)}/players/${encodeURIComponent(pid)}`
</script>

<style scoped lang="scss">
@use '../shared/views';
.pvp {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}
.pvp__unit {
  flex: 3 1 560px;
  min-width: 0;
}
.pvp__season {
  flex: 2 1 340px;
  min-width: 0;
}
.pvp__power {
  font-size: 17px;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.pvp__link {
  font-size: 13px;
  font-weight: 700;
  color: var(--m-pInk);
  text-decoration: none;
}
.pvp__stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(100px, 1fr));
  gap: 8px;
  margin-bottom: 10px;
}
.pvp__stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.pvp__stat-v {
  font-size: 18px;
  font-weight: 800;
}
.pvp__battles {
  margin: 0;
  padding: 0;
  list-style: none;
}
.pvp__battle {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 52px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.pvp__rival {
  flex: 1;
  min-width: 0;
  font-size: 13px;
  font-weight: 700;
  color: var(--el-text-color-primary);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
a.pvp__rival:hover {
  color: var(--m-pInk);
}
.pvp__delta {
  font-weight: 800;
}
.pvp__delta.is-up {
  color: var(--m-ok);
}
.pvp__delta.is-down {
  color: var(--m-bad);
}
.pvp__at {
  width: 84px;
  text-align: right;
}
</style>
