<!-- 工作台：今天的数字、近 14 天的新增与活跃、待处理、常用操作（只跳转）、最近 GM 操作。 -->
<template>
  <div class="azh">
    <MetricStrip :columns="5">
      <MetricItem :label="t('azurGm.home.online')" :value="num(status?.online)" icon="HOutline:SignalIcon" />
      <MetricItem :label="t('azurGm.home.onlinePeak')" :value="num(status?.onlinePeak)" icon="HOutline:ArrowTrendingUpIcon" />
      <MetricItem :label="t('azurGm.home.active')" :value="num(status?.activeToday)" icon="HOutline:UserGroupIcon" />
      <MetricItem :label="t('azurGm.home.newAccounts')" :value="num(status?.newToday)" icon="HOutline:UserPlusIcon" />
      <MetricItem :label="t('azurGm.home.registered')" :value="num(status?.registered)" icon="HOutline:IdentificationIcon" />
    </MetricStrip>

    <div class="azh__grid">
      <AppPanel :title="t('azurGm.home.trend')">
        <p v-if="trendError" class="azh__hint">{{ t('azurGm.common.loadFailed', { msg: trendError }) }}</p>
        <div v-else class="azh__trend">
          <div v-for="d in days" :key="d.day" class="azh__day" :title="dayTitle(d)">
            <span class="azh__bars">
              <span class="azh__bar azh__bar--active" :style="{ height: `${pct(d.active)}%` }" />
              <span class="azh__bar azh__bar--new" :style="{ height: `${pct(d.newAccounts)}%` }" />
            </span>
            <span class="azh__axis m-num">{{ d.day.slice(5) }}</span>
          </div>
        </div>
        <div class="azh__legend">
          <span><i class="azh__dot azh__dot--active" />{{ t('azurGm.home.trendActive') }}</span>
          <span><i class="azh__dot azh__dot--new" />{{ t('azurGm.home.trendNew') }}</span>
        </div>
      </AppPanel>

      <div class="azh__side">
        <AppPanel :title="t('azurGm.home.pending')">
          <div v-if="ctx.pendingApprovals.value" class="azh__todo">
            <span>{{ t('azurGm.home.pendingApprovals', { n: ctx.pendingApprovals.value }) }}</span>
            <UButton size="sm" :to="{ query: { tab: 'operations', view: 'logs' } }">{{ t('azurGm.home.goApprove') }}</UButton>
          </div>
          <p v-else class="azh__hint">{{ t('azurGm.home.pendingNone') }}</p>
        </AppPanel>

        <AppPanel :title="t('azurGm.home.quick')">
          <form class="azh__find" @submit.prevent="findPlayer">
            <input v-model="q" type="search" :placeholder="t('azurGm.home.findPlayer')" />
            <UButton type="submit" size="sm" icon="i-heroicons-magnifying-glass">{{ t('azurGm.common.search') }}</UButton>
          </form>
          <div class="azh__links">
            <UButton size="sm" color="neutral" variant="soft" :to="{ query: { tab: 'mail' } }">{{ t('azurGm.home.goMail') }}</UButton>
            <UButton size="sm" color="neutral" variant="soft" :to="{ query: { tab: 'codes' } }">{{ t('azurGm.home.goCodes') }}</UButton>
            <UButton size="sm" color="neutral" variant="soft" :to="{ query: { tab: 'operations' } }">{{ t('azurGm.home.goOperations') }}</UButton>
          </div>
        </AppPanel>

        <AppPanel :title="t('azurGm.home.health')">
          <DescriptionList :items="health" :columns="1" />
        </AppPanel>
      </div>
    </div>

    <AppPanel v-if="recent.length" :title="t('azurGm.home.recent')">
      <ul class="azh__recent">
        <li v-for="e in recent" :key="e.id">
          <GmRiskPill :risk="e.risk" />
          <span class="azh__recent-title">{{ e.commandTitle }}</span>
          <span class="azh__hint">{{ e.actorName }} · {{ e.successCount }}/{{ e.successCount + e.failedCount }}</span>
          <span class="azh__hint m-num">{{ when(e.createTime) }}</span>
        </li>
      </ul>
    </AppPanel>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onMounted, ref, shallowRef } from 'vue'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import dayjs from 'dayjs'
import { listGmExecutions } from '@/api/gameGm'
import * as api from '@/api/azurpromilia'
import type { GmExecution } from '@/types/v1/game_gm'
import type { DailyStat } from '@/types/v1/azurpromilia/types'
import { useGmContext } from '../context'
import GmRiskPill from '../components/GmRiskPill.vue'
import { errorMessage } from './shared'

const { t } = useI18n()
const router = useRouter()
const ctx = useGmContext()
const status = computed(() => ctx.azurStatus.value)

const num = (v: number | undefined) => (v === undefined ? '—' : new Intl.NumberFormat().format(v))

const days = shallowRef<DailyStat[]>([])
const trendError = ref('')
const recent = shallowRef<GmExecution[]>([])
const q = ref('')

const maxValue = computed(() => Math.max(1, ...days.value.map((d) => Math.max(d.active, d.newAccounts))))
const pct = (v: number) => Math.round((v / maxValue.value) * 100)
const dayTitle = (d: DailyStat) =>
  `${d.day} · ${t('azurGm.home.trendActive')} ${d.active} · ${t('azurGm.home.trendNew')} ${d.newAccounts} · ${t('azurGm.home.trendPeak')} ${d.onlinePeak}`
const when = (v: Date | string | undefined) => (v ? dayjs(v).format('MM-DD HH:mm') : '')

const health = computed(() => {
  const s = status.value
  const rows = [
    { label: t('azurGm.home.version'), value: s?.version || '—' },
    { label: t('azurGm.home.resource'), value: s?.resource?.releaseHot || '—' },
  ]
  for (const h of s?.health || []) rows.push({ label: h.name, value: h.ok ? t('azurGm.home.healthy') : `${t('azurGm.home.unhealthy')} ${h.detail}` })
  return rows
})

const load = async () => {
  const serverId = ctx.serverId.value
  try {
    days.value = (await api.getDailyStats({ serverId, days: 14 })).items
    trendError.value = ''
  } catch (e) {
    trendError.value = errorMessage(e)
  }
  try {
    const { data } = await listGmExecutions({
      serverId,
      actorKind: '',
      status: '',
      risk: '',
      command: '',
      pid: '',
      q: '',
      since: undefined,
      until: undefined,
      includeDryRun: false,
      page: 1,
      pageSize: 4,
      actorId: '',
    })
    recent.value = data.items
  } catch {
    recent.value = []
  }
}

const findPlayer = () => {
  void router.push({ query: { tab: 'players', q: q.value.trim() || undefined } })
}

onMounted(load)
onActivated(() => {
  void ctx.refresh()
  void load()
})
</script>

<style scoped lang="scss">
.azh {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.azh__grid {
  display: grid;
  grid-template-columns: minmax(0, 2fr) minmax(260px, 1fr);
  gap: 14px;
  @media (max-width: 960px) {
    grid-template-columns: 1fr;
  }
}
.azh__side {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.azh__trend {
  display: grid;
  grid-template-columns: repeat(14, minmax(0, 1fr));
  gap: 6px;
  height: 200px;
  align-items: end;
}
.azh__day {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  height: 100%;
}
.azh__bars {
  flex: 1;
  width: 100%;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  gap: 2px;
}
.azh__bar {
  width: 40%;
  min-height: 2px;
  border-radius: 3px 3px 0 0;
}
.azh__bar--active,
.azh__dot--active {
  background: var(--m-p);
}
.azh__bar--new,
.azh__dot--new {
  background: var(--m-a);
}
.azh__axis {
  font-size: 11px;
  color: var(--el-text-color-secondary);
}
.azh__legend {
  display: flex;
  gap: 14px;
  margin-top: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
  span {
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
}
.azh__dot {
  display: inline-block;
  width: 8px;
  height: 8px;
  border-radius: 50%;
}
.azh__hint {
  margin: 0;
  font-size: 13px;
  color: var(--el-text-color-secondary);
}
.azh__todo {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.azh__find {
  display: flex;
  gap: 8px;
  input {
    flex: 1;
    height: 32px;
    padding: 0 10px;
    border: 1px solid var(--el-border-color-light);
    border-radius: var(--app-radius-sm);
    background: var(--el-bg-color);
    color: var(--el-text-color-primary);
  }
}
.azh__links {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 10px;
}
.azh__recent {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  li {
    display: flex;
    align-items: center;
    gap: 10px;
  }
}
.azh__recent-title {
  flex: 1;
  min-width: 0;
}
</style>
