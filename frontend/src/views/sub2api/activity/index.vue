<!-- Sub2API 活动配置（每日抽奖）：PageHeader + 令牌 Tab + AppPanel。
     四段：累计中 / 报名中 / 历史 / 配置。严格按 redesign 规范，EP-free。 -->
<template>
  <div class="s2a-act">
    <PageHeader :title="t('sub2api.activity.pageTitle')" :description="t('sub2api.activity.pageDesc')">
      <template #actions>
        <UButton
          color="neutral"
          variant="soft"
          icon="i-lucide-refresh-cw"
          :loading="loading"
          @click="reload"
        >
          {{ t('sub2api.activity.refresh') }}
        </UButton>
      </template>
    </PageHeader>

    <div class="settings-tabs" role="tablist">
      <button
        v-for="tab in TABS"
        :key="tab.name"
        type="button"
        role="tab"
        class="settings-tabs__btn"
        :class="{ 'is-active': activeTab === tab.name }"
        :aria-selected="activeTab === tab.name"
        @click="setTab(tab.name)"
      >
        <component :is="menuStore.iconComponents[tab.icon]" />
        {{ t(tab.labelKey) }}
      </button>
    </div>

    <!-- 累计中 -->
    <div v-show="activeTab === 'accum'" class="settings-tab">
      <MetricStrip :columns="4">
        <MetricItem :label="t('sub2api.activity.accumPool')" :value="fmtMoney(overview?.accumPool)" />
        <MetricItem :label="t('sub2api.activity.accumSpend')" :value="fmtMoney(overview?.accumSpend)" />
        <MetricItem :label="t('sub2api.activity.accumEligible')" :value="overview?.accumEligible ?? 0" />
        <MetricItem :label="t('sub2api.activity.nextSettle')" :value="fmtTime(overview?.nextSettleTime)" />
      </MetricStrip>
      <AppPanel :title="t('sub2api.activity.accumHintTitle')" :padded="false">
        <div class="hint">{{ t('sub2api.activity.accumHint') }}</div>
      </AppPanel>
    </div>

    <!-- 报名中 -->
    <div v-show="activeTab === 'registering'" class="settings-tab">
      <template v-if="overview?.current">
        <MetricStrip :columns="4">
          <MetricItem :label="t('sub2api.activity.poolAmount')" :value="fmtMoney(overview.current.poolAmount)" />
          <MetricItem :label="t('sub2api.activity.registered')" :value="overview.current.registeredCount" />
          <MetricItem :label="t('sub2api.activity.eligible')" :value="overview.current.eligibleCount" />
          <MetricItem :label="t('sub2api.activity.drawTime')" :value="fmtTime(overview.current.drawTime)" />
        </MetricStrip>
        <AppPanel :title="t('sub2api.activity.roundInfo')" :padded="false">
          <div class="set-rows">
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('sub2api.activity.roundDate') }}</span></div>
              <span class="set-value">{{ overview.current.id }}</span>
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('sub2api.activity.sourceDate') }}</span></div>
              <span class="set-value">{{ overview.current.sourceDate }}</span>
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('sub2api.activity.carryIn') }}</span></div>
              <span class="set-value">{{ fmtMoney(overview.current.carryIn) }}</span>
            </div>
            <div class="set-row">
              <div class="set-row__info"><span class="set-row__label">{{ t('sub2api.activity.threshold') }}</span></div>
              <span class="set-value">{{ fmtMoney(overview.current.threshold) }}</span>
            </div>
          </div>
          <template #footer>
            <UButton
              v-if="canEdit"
              color="primary"
              variant="soft"
              :loading="drawing"
              @click="handleDraw"
            >
              {{ t('sub2api.activity.triggerDraw') }}
            </UButton>
          </template>
        </AppPanel>
      </template>
      <EmptyState
        v-else
        :title="t('sub2api.activity.noRegistering')"
        :description="t('sub2api.activity.noRegisteringDesc')"
      />
    </div>

    <!-- 历史 -->
    <div v-show="activeTab === 'history'" class="settings-tab">
      <AppPanel :title="t('sub2api.activity.historyTitle')" :padded="false">
        <DataTable
          :columns="historyColumns"
          :rows="historyList"
          row-key="id"
          :loading="historyLoading"
          :empty-text="t('sub2api.activity.historyEmpty')"
        >
          <template #cell-status="{ row }">
            <StatusPill
              :variant="row.status === LotteryRoundStatus.LOTTERY_ROUND_STATUS_DRAWN ? 'success' : 'primary'"
              :label="statusLabel(row.status as LotteryRoundStatus)"
            />
          </template>
          <template #cell-poolAmount="{ row }">
            <span class="num">{{ fmtMoney(row.poolAmount as number) }}</span>
          </template>
          <template #cell-perWinnerAmount="{ row }">
            <span class="num">{{ fmtMoney(row.perWinnerAmount as number) }}</span>
          </template>
          <template #cell-distributed="{ row }">
            <StatusPill
              :variant="row.distributed ? 'success' : 'warning'"
              :label="row.distributed ? t('sub2api.activity.distributed') : t('sub2api.activity.notDistributed')"
            />
          </template>
          <template #cell-actions="{ row }">
            <div class="row-acts">
              <UButton size="xs" color="neutral" variant="ghost" @click="openDetail(String(row.id))">
                {{ t('sub2api.activity.winners') }}
              </UButton>
              <UButton
                v-if="canEdit && !row.distributed && row.status === LotteryRoundStatus.LOTTERY_ROUND_STATUS_DRAWN"
                size="xs"
                color="primary"
                variant="soft"
                :loading="distributingId === row.id"
                @click="handleDistribute(String(row.id))"
              >
                {{ t('sub2api.activity.distribute') }}
              </UButton>
            </div>
          </template>
        </DataTable>
        <template #footer>
          <Pagination
            v-model:page="historyPage"
            v-model:page-size="historyPageSize"
            :total="historyTotal"
            @change="loadHistory"
          />
        </template>
      </AppPanel>
    </div>

    <!-- 配置 -->
    <div v-show="activeTab === 'settings'" class="settings-tab">
      <AppPanel :title="t('sub2api.activity.settingsTitle')" :padded="false">
        <div class="set-rows">
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.activity.enabled') }}</span>
              <span class="set-row__desc">{{ t('sub2api.activity.enabledDesc') }}</span>
            </div>
            <AppSwitch v-model="form.enabled" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.activity.autoPayout') }}</span>
              <span class="set-row__desc">{{ t('sub2api.activity.autoPayoutDesc') }}</span>
            </div>
            <AppSwitch v-model="form.autoPayout" :disabled="!canEdit" />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.activity.poolRatio') }}</span>
              <span class="set-row__desc">{{ t('sub2api.activity.poolRatioDesc') }}</span>
            </div>
            <input
              v-model.number="form.poolRatio"
              type="number"
              min="0"
              max="1"
              step="0.01"
              class="app-input set-num"
              :disabled="!canEdit"
            />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.activity.threshold') }}</span>
              <span class="set-row__desc">{{ t('sub2api.activity.thresholdDesc') }}</span>
            </div>
            <input
              v-model.number="form.threshold"
              type="number"
              min="0"
              step="0.01"
              class="app-input set-num"
              :disabled="!canEdit"
            />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.activity.baseWinners') }}</span>
              <span class="set-row__desc">{{ t('sub2api.activity.baseWinnersDesc') }}</span>
            </div>
            <input
              v-model.number="form.baseWinners"
              type="number"
              min="1"
              step="1"
              class="app-input set-num"
              :disabled="!canEdit"
            />
          </div>
          <div class="set-row">
            <div class="set-row__info">
              <span class="set-row__label">{{ t('sub2api.activity.maxWinners') }}</span>
              <span class="set-row__desc">{{ t('sub2api.activity.maxWinnersDesc') }}</span>
            </div>
            <input
              v-model.number="form.maxWinners"
              type="number"
              min="0"
              step="1"
              class="app-input set-num"
              :disabled="!canEdit"
            />
          </div>
        </div>
        <template #footer>
          <UButton color="primary" :loading="saving" :disabled="!canEdit" @click="handleSave">
            {{ t('sub2api.activity.save') }}
          </UButton>
          <UButton
            v-if="canEdit"
            color="neutral"
            variant="soft"
            :loading="settling"
            @click="handleSettle"
          >
            {{ t('sub2api.activity.triggerSettle') }}
          </UButton>
        </template>
      </AppPanel>
    </div>

    <!-- 中奖名单弹窗 -->
    <FormDialog
      v-model="detailOpen"
      :title="t('sub2api.activity.winnersTitle', { date: detailRound?.id || '' })"
      :show-footer="false"
      :width="640"
    >
      <div v-if="detailLoading" class="hint">{{ t('sub2api.activity.loading') }}</div>
      <EmptyState
        v-else-if="!detailWinners.length"
        :title="t('sub2api.activity.noWinners')"
      />
      <div v-else class="winner-list">
        <div v-for="w in detailWinners" :key="w.id" class="winner-row">
          <div class="winner-row__main">
            <span class="winner-row__name">{{ w.userName || w.sub2apiUserId }}</span>
            <span class="winner-row__id">#{{ w.sub2apiUserId }}</span>
          </div>
          <div class="winner-row__meta">
            <span class="num">{{ fmtMoney(w.prizeAmount) }}</span>
            <StatusPill :variant="payoutVariant(w.payoutStatus)" :label="payoutLabel(w.payoutStatus)" />
          </div>
        </div>
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  getLotteryOverview,
  updateLotterySettings,
  listLotteryRounds,
  getLotteryRoundDetail,
  distributeLotteryRound,
  triggerLotterySettle,
  triggerLotteryDraw,
} from '@/api/sub2api-lottery'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { showRequestError } from '@/utils/request'
import { useFeedback } from '@/utils/feedback'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import {
  LotteryRoundStatus,
  LotteryPayoutStatus,
  type GetLotteryOverviewResponse,
  type LotterySettings,
  type LotteryRound,
  type LotteryParticipant,
} from '@/types/v1/sub2api'

defineOptions({ name: 'Sub2APIActivityView' })

type TabName = 'accum' | 'registering' | 'history' | 'settings'
const TABS: { name: TabName; labelKey: string; icon: string }[] = [
  { name: 'accum', labelKey: 'sub2api.activity.tabAccum', icon: 'HOutline:ChartBarIcon' },
  { name: 'registering', labelKey: 'sub2api.activity.tabRegistering', icon: 'HOutline:ClockIcon' },
  { name: 'history', labelKey: 'sub2api.activity.tabHistory', icon: 'HOutline:ArchiveBoxIcon' },
  { name: 'settings', labelKey: 'sub2api.activity.tabSettings', icon: 'HOutline:Cog6ToothIcon' },
]

const menuStore = useMenuStore()
const { t } = useI18n()
const fb = useFeedback()
const canEdit = useButtonPermission([PERM.SUB2API_EDIT], [])

const activeTab = ref<TabName>('accum')
const loading = ref(false)
const saving = ref(false)
const settling = ref(false)
const drawing = ref(false)
const historyLoading = ref(false)
const overview = ref<GetLotteryOverviewResponse | null>(null)

const form = reactive<LotterySettings>({
  enabled: false,
  poolRatio: 0.05,
  threshold: 2,
  baseWinners: 10,
  maxWinners: 0,
  autoPayout: true,
})

const historyPage = ref(1)
const historyPageSize = ref(20)
const historyTotal = ref(0)
const historyList = ref<LotteryRound[]>([])
const distributingId = ref('')

const historyColumns = computed<DataTableColumn[]>(() => [
  { key: 'id', title: t('sub2api.activity.roundDate'), minWidth: 110 },
  { key: 'status', title: t('sub2api.activity.status'), width: 100 },
  { key: 'poolAmount', title: t('sub2api.activity.poolAmount'), width: 110, align: 'right' },
  { key: 'registeredCount', title: t('sub2api.activity.registered'), width: 90, align: 'right' },
  { key: 'winnerCount', title: t('sub2api.activity.winnersCount'), width: 90, align: 'right' },
  { key: 'perWinnerAmount', title: t('sub2api.activity.perWinner'), width: 110, align: 'right' },
  { key: 'distributed', title: t('sub2api.activity.payout'), width: 100 },
  { key: 'actions', title: t('sub2api.common.operation'), width: 160 },
])

const detailOpen = ref(false)
const detailLoading = ref(false)
const detailRound = ref<LotteryRound | null>(null)
const detailWinners = ref<LotteryParticipant[]>([])

const fmtMoney = (n?: number | null) => {
  const v = Number(n) || 0
  return `$${v.toFixed(4)}`
}

const fmtTime = (v?: Date | string | null) => {
  if (!v) return '—'
  const d = v instanceof Date ? v : new Date(v)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString()
}

// 历史列表仅 drawn，状态列直接显示「已开奖」
const statusLabel = (_s: LotteryRoundStatus) => t('sub2api.activity.statusDrawn')

const payoutLabel = (s: LotteryPayoutStatus) => {
  switch (s) {
    case LotteryPayoutStatus.LOTTERY_PAYOUT_STATUS_PAID:
      return t('sub2api.activity.payoutPaid')
    case LotteryPayoutStatus.LOTTERY_PAYOUT_STATUS_PENDING:
      return t('sub2api.activity.payoutPending')
    case LotteryPayoutStatus.LOTTERY_PAYOUT_STATUS_FAILED:
      return t('sub2api.activity.payoutFailed')
    case LotteryPayoutStatus.LOTTERY_PAYOUT_STATUS_NONE:
      return t('sub2api.activity.payoutNone')
    default:
      return t('sub2api.activity.statusUnknown')
  }
}

const payoutVariant = (s: LotteryPayoutStatus): 'success' | 'warning' | 'error' | 'neutral' => {
  switch (s) {
    case LotteryPayoutStatus.LOTTERY_PAYOUT_STATUS_PAID:
      return 'success'
    case LotteryPayoutStatus.LOTTERY_PAYOUT_STATUS_PENDING:
      return 'warning'
    case LotteryPayoutStatus.LOTTERY_PAYOUT_STATUS_FAILED:
      return 'error'
    default:
      return 'neutral'
  }
}

const loadOverview = async () => {
  loading.value = true
  try {
    const { data } = await getLotteryOverview()
    overview.value = data || null
    if (data?.settings) Object.assign(form, data.settings)
  } catch (e) {
    showRequestError(e, t('sub2api.activity.loadFailed'))
  } finally {
    loading.value = false
  }
}

const loadHistory = async () => {
  historyLoading.value = true
  try {
    const { data } = await listLotteryRounds({
      page: historyPage.value,
      pageSize: historyPageSize.value,
    })
    historyList.value = data?.rounds || []
    historyTotal.value = Number(data?.total || 0)
  } catch (e) {
    showRequestError(e, t('sub2api.activity.loadFailed'))
  } finally {
    historyLoading.value = false
  }
}

const reload = async () => {
  await loadOverview()
  if (activeTab.value === 'history') await loadHistory()
}

const handleSave = async () => {
  saving.value = true
  try {
    const { data } = await updateLotterySettings({ settings: { ...form } })
    if (data?.settings) Object.assign(form, data.settings)
    fb.success(t('sub2api.activity.saveSuccess'))
    await loadOverview()
  } catch (e) {
    showRequestError(e, t('sub2api.activity.saveFailed'))
  } finally {
    saving.value = false
  }
}

const handleSettle = async () => {
  settling.value = true
  try {
    await triggerLotterySettle({ date: '' })
    fb.success(t('sub2api.activity.settleSuccess'))
    await reload()
  } catch (e) {
    showRequestError(e, t('sub2api.activity.settleFailed'))
  } finally {
    settling.value = false
  }
}

const handleDraw = async () => {
  drawing.value = true
  try {
    await triggerLotteryDraw({ date: '' })
    fb.success(t('sub2api.activity.drawSuccess'))
    await reload()
  } catch (e) {
    showRequestError(e, t('sub2api.activity.drawFailed'))
  } finally {
    drawing.value = false
  }
}

const handleDistribute = async (id: string) => {
  distributingId.value = id
  try {
    await distributeLotteryRound({ id })
    fb.success(t('sub2api.activity.distributeSuccess'))
    await loadHistory()
  } catch (e) {
    showRequestError(e, t('sub2api.activity.distributeFailed'))
  } finally {
    distributingId.value = ''
  }
}

const openDetail = async (id: string) => {
  detailOpen.value = true
  detailLoading.value = true
  detailRound.value = null
  detailWinners.value = []
  try {
    const { data } = await getLotteryRoundDetail({ id })
    detailRound.value = data?.round || null
    detailWinners.value = data?.winners || []
  } catch (e) {
    showRequestError(e, t('sub2api.activity.loadFailed'))
  } finally {
    detailLoading.value = false
  }
}

const loadedTabs = ref(new Set<string>())
const onTabChange = (name: string) => {
  if (loadedTabs.value.has(name)) return
  loadedTabs.value.add(name)
  if (name === 'history') loadHistory()
  else loadOverview()
}
const setTab = (name: TabName) => {
  activeTab.value = name
  onTabChange(name)
}

onMounted(() => {
  onTabChange(activeTab.value)
})
</script>

<style scoped lang="scss">
.s2a-act {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.settings-tabs {
  display: inline-flex;
  align-self: flex-start;
  padding: 3px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
  flex-wrap: wrap;
}
.settings-tabs__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 14px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.settings-tabs__btn :deep(svg) {
  width: 16px;
  height: 16px;
}
.settings-tabs__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.settings-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.set-rows {
  display: flex;
  flex-direction: column;
}
.set-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  padding: 11px 20px;
}
.set-row + .set-row {
  border-top: 1px solid var(--el-border-color-lighter);
}
.set-row__info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1;
}
.set-row__label {
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}
.set-row__desc {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.set-value {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  text-align: right;
  font-variant-numeric: tabular-nums;
}
.set-num {
  width: 140px;
  flex-shrink: 0;
}
.hint {
  padding: 16px 20px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
.num {
  font-variant-numeric: tabular-nums;
}
.row-acts {
  display: inline-flex;
  gap: 4px;
  flex-wrap: wrap;
}
.winner-list {
  display: flex;
  flex-direction: column;
}
.winner-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 4px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.winner-row__main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}
.winner-row__name {
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
}
.winner-row__id {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
}
.winner-row__meta {
  display: inline-flex;
  align-items: center;
  gap: 10px;
}
@media (width <= 768px) {
  .set-row {
    flex-direction: column;
    align-items: stretch;
    gap: 8px;
  }
  .set-num {
    width: 100%;
  }
  .set-value {
    text-align: left;
  }
}
</style>
