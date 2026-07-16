<!-- Sub2API 活动配置（单页分区）：每日抽奖为其中一个活动。
     累计中 + 报名中 合并展示；历史/设置为按钮弹窗；
     已报名人数可点开报名名单，点报名者按 user_id 实时拉 Sub2API 用户详情。 -->
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

    <!-- 活动切换：同「配置页面」el-tabs 风格；选中的活动占据主区 -->
    <el-tabs v-model="activeActivity" class="activity-tabs">
      <el-tab-pane :label="t('sub2api.activity.brand')" name="lottery">
        <div class="activity-pane">
          <div class="pane-toolbar">
            <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-history" @click="openHistory">
              {{ t('sub2api.activity.historyRecords') }}
            </UButton>
            <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-settings" @click="openSettings">
              {{ t('sub2api.activity.settingsBtn') }}
            </UButton>
          </div>

          <!-- 段一：累计中 = 正在为下一期攒奖池 -->
          <AppPanel>
            <template #header>
              <div class="seg-head">
                <div class="seg-head__main">
                  <span class="seg-head__tag">{{ t('sub2api.activity.tabAccum') }}</span>
                  <h2 class="seg-head__period">
                    {{ t('sub2api.activity.periodLabel') }}
                    <b>{{ accumPeriodDate || '—' }}</b>
                  </h2>
                </div>
                <span class="seg-head__cap">{{ t('sub2api.activity.accumCaption') }}</span>
              </div>
            </template>
            <MetricStrip :columns="3">
              <MetricItem :label="t('sub2api.activity.accumPool')" :value="fmtMoney(overview?.accumPool)" />
              <MetricItem :label="t('sub2api.activity.accumSpend')" :value="fmtMoney(overview?.accumSpend)" />
              <MetricItem :label="t('sub2api.activity.nextSettle')" :value="fmtTime(overview?.nextSettleTime)" />
            </MetricStrip>
          </AppPanel>

          <!-- 段二：报名中 = 指标 + 本轮详情 合成一张卡，期号是标题 -->
          <AppPanel>
            <template #header>
              <div class="seg-head">
                <div class="seg-head__main">
                  <span class="seg-head__tag">{{ t('sub2api.activity.tabRegistering') }}</span>
                  <h2 v-if="overview?.current" class="seg-head__period">
                    {{ t('sub2api.activity.periodLabel') }}
                    <b>{{ overview.current.id }}</b>
                  </h2>
                  <h2 v-else class="seg-head__period is-muted">{{ t('sub2api.activity.noRegistering') }}</h2>
                </div>
                <span class="seg-head__cap">{{ t('sub2api.activity.registeringCaption') }}</span>
              </div>
            </template>

            <template v-if="overview?.current">
              <MetricStrip :columns="3">
                <MetricItem :label="t('sub2api.activity.poolAmount')" :value="fmtMoney(overview.current.poolAmount)" />
                <MetricItem :label="t('sub2api.activity.registeredCount')">
                  <template #value>
                    <button type="button" class="reg-link" @click="openRegistrants(overview.current!.id)">
                      {{ overview.current.registeredCount }}
                      <UIcon name="i-lucide-users" />
                    </button>
                  </template>
                </MetricItem>
                <MetricItem :label="t('sub2api.activity.drawTime')" :value="fmtTime(overview.current.drawTime)" />
              </MetricStrip>
              <div class="round-meta">
                <div class="round-meta__item">
                  <span>{{ t('sub2api.activity.sourceDate') }}</span>
                  <b>{{ overview.current.sourceDate }}</b>
                </div>
                <div class="round-meta__item">
                  <span>{{ t('sub2api.activity.threshold') }}</span>
                  <b>{{ fmtMoney(overview.current.threshold) }}</b>
                </div>
                <div class="round-meta__item">
                  <span>{{ t('sub2api.activity.carryIn') }}</span>
                  <b>{{ fmtMoney(overview.current.carryIn) }}</b>
                </div>
              </div>
            </template>
            <EmptyState
              v-else
              :title="t('sub2api.activity.noRegistering')"
              :description="t('sub2api.activity.noRegisteringDesc')"
            />

            <template v-if="overview?.current && canEdit" #footer>
              <UButton color="primary" variant="soft" :loading="drawing" @click="handleDraw">
                {{ t('sub2api.activity.triggerDraw') }}
              </UButton>
            </template>
          </AppPanel>
        </div>
      </el-tab-pane>
    </el-tabs>

    <!-- 历史弹窗 -->
    <FormDialog v-model="historyOpen" :title="t('sub2api.activity.historyTitle')" :show-footer="false" :width="860">
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
            :label="t('sub2api.activity.statusDrawn')"
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
      <div class="dialog-foot">
        <Pagination
          v-model:page="historyPage"
          v-model:page-size="historyPageSize"
          :total="historyTotal"
          @change="loadHistory"
        />
      </div>
    </FormDialog>

    <!-- 设置弹窗 -->
    <FormDialog v-model="settingsOpen" :title="t('sub2api.activity.settingsTitle')" :width="560">
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
          <input v-model.number="form.poolRatio" type="number" min="0" max="1" step="0.01" class="app-input set-num" :disabled="!canEdit" />
        </div>
        <div class="set-row">
          <div class="set-row__info">
            <span class="set-row__label">{{ t('sub2api.activity.threshold') }}</span>
            <span class="set-row__desc">{{ t('sub2api.activity.thresholdDesc') }}</span>
          </div>
          <input v-model.number="form.threshold" type="number" min="0" step="0.01" class="app-input set-num" :disabled="!canEdit" />
        </div>
        <div class="set-row">
          <div class="set-row__info">
            <span class="set-row__label">{{ t('sub2api.activity.baseWinners') }}</span>
            <span class="set-row__desc">{{ t('sub2api.activity.baseWinnersDesc') }}</span>
          </div>
          <input v-model.number="form.baseWinners" type="number" min="1" step="1" class="app-input set-num" :disabled="!canEdit" />
        </div>
        <div class="set-row">
          <div class="set-row__info">
            <span class="set-row__label">{{ t('sub2api.activity.maxWinners') }}</span>
            <span class="set-row__desc">{{ t('sub2api.activity.maxWinnersDesc') }}</span>
          </div>
          <input v-model.number="form.maxWinners" type="number" min="0" step="1" class="app-input set-num" :disabled="!canEdit" />
        </div>
      </div>
      <template #footer>
        <UButton
          v-if="canEdit"
          color="neutral"
          variant="soft"
          :loading="settling"
          @click="handleSettle"
        >
          {{ t('sub2api.activity.triggerSettle') }}
        </UButton>
        <UButton color="primary" :loading="saving" :disabled="!canEdit" @click="handleSave">
          {{ t('sub2api.activity.save') }}
        </UButton>
      </template>
    </FormDialog>

    <!-- 报名名单弹窗（点击已报名人数） -->
    <FormDialog
      v-model="registrantsOpen"
      :title="`${t('sub2api.activity.registrantsTitle')}${registrantsRound ? ' · ' + registrantsRound : ''}`"
      :show-footer="false"
      :width="560"
    >
      <div v-if="registrantsLoading" class="hint">{{ t('sub2api.activity.loading') }}</div>
      <EmptyState v-else-if="!registrants.length" :title="t('sub2api.activity.noRegistrants')" />
      <div v-else class="reg-list">
        <div class="reg-head">
          <span>{{ t('sub2api.activity.registrant') }}</span>
          <span class="reg-head__spend">{{ t('sub2api.activity.spendAmount') }}</span>
        </div>
        <div v-for="r in registrants" :key="r.id" class="reg-item">
          <button type="button" class="reg-row" @click="toggleUser(r.sub2apiUserId)">
            <span class="reg-row__name">{{ r.userName || `#${r.sub2apiUserId}` }}</span>
            <span class="reg-row__id">#{{ r.sub2apiUserId }}</span>
            <span class="reg-row__spend num">{{ fmtMoney(r.spendSnapshot) }}</span>
            <UIcon
              :name="expandedUserId === r.sub2apiUserId ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
              class="reg-row__chev"
            />
          </button>
          <div v-if="expandedUserId === r.sub2apiUserId" class="reg-detail">
            <div v-if="userLoading" class="hint">{{ t('sub2api.activity.loading') }}</div>
            <div v-else-if="userCache[r.sub2apiUserId]" class="reg-detail__grid">
              <div><span>{{ t('sub2api.activity.userEmail') }}</span><b>{{ userCache[r.sub2apiUserId]?.email || '—' }}</b></div>
              <div><span>{{ t('sub2api.activity.userBalance') }}</span><b>{{ fmtMoney(userCache[r.sub2apiUserId]?.balance) }}</b></div>
              <div><span>{{ t('sub2api.activity.status') }}</span><b>{{ userCache[r.sub2apiUserId]?.status || '—' }}</b></div>
              <div><span>{{ t('sub2api.activity.userRole') }}</span><b>{{ userCache[r.sub2apiUserId]?.role || '—' }}</b></div>
              <div><span>{{ t('sub2api.activity.userCreatedAt') }}</span><b>{{ fmtTime(userCache[r.sub2apiUserId]?.createdAt) }}</b></div>
            </div>
            <div v-else class="warn-text">{{ userError || t('sub2api.activity.userLoadFailed') }}</div>
          </div>
        </div>
      </div>
    </FormDialog>

    <!-- 中奖名单弹窗（从历史表打开） -->
    <FormDialog
      v-model="detailOpen"
      :title="t('sub2api.activity.winnersTitle', { date: detailRound?.id || '' })"
      :show-footer="false"
      :width="640"
    >
      <div v-if="detailLoading" class="hint">{{ t('sub2api.activity.loading') }}</div>
      <EmptyState v-else-if="!detailWinners.length" :title="t('sub2api.activity.noWinners')" />
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
  listLotteryRegistrants,
  getSub2APIUser,
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
  type Sub2APIUserInfo,
} from '@/types/v1/sub2api'

defineOptions({ name: 'Sub2APIActivityView' })

const { t } = useI18n()
const fb = useFeedback()
const canEdit = useButtonPermission([PERM.SUB2API_EDIT], [])

const loading = ref(false)
const saving = ref(false)
const settling = ref(false)
const drawing = ref(false)
const overview = ref<GetLotteryOverviewResponse | null>(null)

// 活动切换（当前仅「每日抽奖」；后续活动追加为新的 el-tab-pane）
const activeActivity = ref('lottery')

const form = reactive<LotterySettings>({
  enabled: false,
  poolRatio: 0.05,
  threshold: 2,
  baseWinners: 10,
  maxWinners: 0,
  autoPayout: true,
})

// —— 历史 ——
const historyOpen = ref(false)
const historyLoading = ref(false)
const historyPage = ref(1)
const historyPageSize = ref(20)
const historyTotal = ref(0)
const historyList = ref<LotteryRound[]>([])
const distributingId = ref('')

// —— 设置 ——
const settingsOpen = ref(false)

// —— 中奖名单 ——
const detailOpen = ref(false)
const detailLoading = ref(false)
const detailRound = ref<LotteryRound | null>(null)
const detailWinners = ref<LotteryParticipant[]>([])

// —— 报名名单 ——
const registrantsOpen = ref(false)
const registrantsLoading = ref(false)
const registrantsRound = ref('')
const registrants = ref<LotteryParticipant[]>([])
const expandedUserId = ref<number | null>(null)
const userLoading = ref(false)
const userError = ref('')
const userCache = reactive<Record<number, Sub2APIUserInfo>>({})

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

const fmtMoney = (n?: number | null) => `$${(Number(n) || 0).toFixed(4)}`

const fmtTime = (v?: Date | string | null) => {
  if (!v) return '—'
  const d = v instanceof Date ? v : new Date(v)
  if (Number.isNaN(d.getTime())) return '—'
  return d.toLocaleString()
}

// 累计中对应的期号 = 下次结算日（今日累计 → 次日 0 点开出那一期）
const accumPeriodDate = computed(() => {
  const v = overview.value?.nextSettleTime
  if (!v) return ''
  const d = v instanceof Date ? v : new Date(v)
  if (Number.isNaN(d.getTime())) return ''
  const y = d.getFullYear()
  const m = String(d.getMonth() + 1).padStart(2, '0')
  const day = String(d.getDate()).padStart(2, '0')
  return `${y}-${m}-${day}`
})

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
  if (historyOpen.value) await loadHistory()
}

const openHistory = async () => {
  historyOpen.value = true
  await loadHistory()
}

const openSettings = () => {
  settingsOpen.value = true
}

const handleSave = async () => {
  saving.value = true
  try {
    const { data } = await updateLotterySettings({ settings: { ...form } })
    if (data?.settings) Object.assign(form, data.settings)
    fb.success(t('sub2api.activity.saveSuccess'))
    settingsOpen.value = false
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
    await loadOverview()
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

const openRegistrants = async (roundId: string) => {
  registrantsOpen.value = true
  registrantsRound.value = roundId
  registrants.value = []
  expandedUserId.value = null
  registrantsLoading.value = true
  try {
    const { data } = await listLotteryRegistrants({ id: roundId })
    registrants.value = data?.registrants || []
  } catch (e) {
    showRequestError(e, t('sub2api.activity.loadFailed'))
  } finally {
    registrantsLoading.value = false
  }
}

// 点击报名者：按 user_id 实时拉取 Sub2API 用户详情（缓存，避免重复请求）
const toggleUser = async (userId: number) => {
  if (expandedUserId.value === userId) {
    expandedUserId.value = null
    return
  }
  expandedUserId.value = userId
  userError.value = ''
  if (userCache[userId]) return
  userLoading.value = true
  try {
    const { data } = await getSub2APIUser({ userId })
    if (data?.user) userCache[userId] = data.user
    else userError.value = t('sub2api.activity.userLoadFailed')
  } catch (e) {
    userError.value = e instanceof Error ? e.message : t('sub2api.activity.userLoadFailed')
  } finally {
    userLoading.value = false
  }
}

onMounted(() => {
  loadOverview()
})
</script>

<style scoped lang="scss">
.s2a-act {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.activity-tabs {
  margin-top: 4px;
}
.activity-pane {
  display: flex;
  flex-direction: column;
  gap: 16px;
  padding-top: 4px;
}
.pane-toolbar {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
.seg-head {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}
.seg-head__main {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-wrap: wrap;
  min-width: 0;
}
.seg-head__tag {
  flex: none;
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
  border-radius: var(--app-radius-sm);
  padding: 2px 8px;
}
.seg-head__period {
  margin: 0;
  font-size: 1.125rem;
  font-weight: 600;
  letter-spacing: -0.01em;
  color: var(--el-text-color-primary);
  b {
    font-variant-numeric: tabular-nums;
    font-weight: 700;
  }
  &.is-muted {
    font-size: 0.9375rem;
    font-weight: 500;
    color: var(--el-text-color-secondary);
  }
}
.seg-head__cap {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  line-height: 1.4;
}
.reg-link {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  border: none;
  background: none;
  padding: 0;
  cursor: pointer;
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 1.1;
  letter-spacing: -0.02em;
  color: var(--el-color-primary);
  font-variant-numeric: tabular-nums;
}
.reg-link :deep(svg) {
  width: 16px;
  height: 16px;
}
.reg-link:hover {
  text-decoration: underline;
}
.round-meta {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 0;
  margin-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.round-meta__item {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
  gap: 12px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  padding: 12px 4px 0;
  b {
    color: var(--el-text-color-primary);
    font-variant-numeric: tabular-nums;
    font-weight: 600;
  }
}
.dialog-foot {
  margin-top: 12px;
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
  padding: 11px 0;
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
.set-num {
  width: 140px;
  flex-shrink: 0;
}
.hint {
  padding: 12px 4px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.num {
  font-variant-numeric: tabular-nums;
}
.row-acts {
  display: inline-flex;
  gap: 4px;
  flex-wrap: wrap;
}
.reg-list {
  display: flex;
  flex-direction: column;
}
.reg-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 4px 8px;
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--el-text-color-secondary);
}
.reg-head__spend {
  margin-left: auto;
}
.reg-item {
  border-top: 1px solid var(--el-border-color-lighter);
}
.reg-row {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 10px 4px;
  border: none;
  background: none;
  cursor: pointer;
  text-align: left;
}
.reg-row:hover {
  background: var(--el-fill-color-lighter);
}
.reg-row__name {
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}
.reg-row__id {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
  flex-shrink: 0;
}
.reg-row__spend {
  margin-left: auto;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-color-primary);
  flex-shrink: 0;
}
.reg-row__chev {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}
.reg-detail {
  padding: 4px 10px 12px;
}
.reg-detail__grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 6px 20px;
  padding: 10px 12px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-lighter);
  div {
    display: flex;
    align-items: baseline;
    justify-content: space-between;
    gap: 12px;
    font-size: 0.8125rem;
    color: var(--el-text-color-secondary);
  }
  b {
    color: var(--el-text-color-primary);
    font-variant-numeric: tabular-nums;
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }
}
.warn-text {
  padding: 10px 12px;
  font-size: 0.8125rem;
  color: var(--el-color-warning);
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
}
</style>
