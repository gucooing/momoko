<!-- Sub2API 活动配置（重写 · P2/P4 去 EP）：PageHeader + 令牌活动 Tab（当前仅每日抽奖）
     + 累计中 AppPanel(MetricStrip + #actions 设置) + 报名中 AppPanel + 历史 DataTable(行点详情)。
     设置/报名名单/轮次详情走 FormDialog；toast=useFeedback。保留 lottery API + PERM.SUB2API_EDIT。 -->
<template>
  <div class="s2a-act" :class="{ 'is-loading': loading }">
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

    <!-- 活动切换：令牌 Tab 条（与 config/home 同款；后续活动追加 TABS 项） -->
    <div class="s2a-tabs" role="tablist">
      <button
        v-for="tab in TABS"
        :key="tab.name"
        type="button"
        role="tab"
        class="s2a-tabs__btn"
        :class="{ 'is-active': activeActivity === tab.name }"
        :aria-selected="activeActivity === tab.name"
        @click="activeActivity = tab.name"
      >
        <component :is="menuStore.iconComponents[tab.icon]" />
        {{ t(tab.labelKey) }}
      </button>
    </div>

    <div v-show="activeActivity === 'lottery'" class="s2a-tab">
      <!-- 段一：累计中 —— 设置入口放本板块 #actions -->
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
        <template #actions>
          <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-settings" @click="openSettings">
            {{ t('sub2api.activity.settingsBtn') }}
          </UButton>
        </template>
        <MetricStrip :columns="3">
          <MetricItem :label="t('sub2api.activity.accumPool')" :value="fmtMoney(overview?.accumPool)" />
          <MetricItem :label="t('sub2api.activity.accumSpend')" :value="fmtMoney(overview?.accumSpend)" />
          <MetricItem :label="t('sub2api.activity.nextSettle')" :value="fmtTime(overview?.nextSettleTime)" />
        </MetricStrip>
      </AppPanel>

      <!-- 段二：报名中 -->
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

      <!-- 段三：历史记录 —— 页底表格；整行点击开详情（不设「历史」按钮弹窗） -->
      <AppPanel :title="t('sub2api.activity.historyTitle')">
        <DataTable
          :columns="historyColumns"
          :rows="historyList"
          row-key="id"
          :loading="historyLoading"
          :empty-text="t('sub2api.activity.historyEmpty')"
          row-clickable
          @row-click="onHistoryRowClick"
        >
          <template #cell-status="{ row }">
            <StatusPill :variant="roundStatusVariant(row.status)" :label="roundStatusLabel(row.status)" />
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
        <div class="history-foot">
          <Pagination
            v-model:page="historyPage"
            v-model:page-size="historyPageSize"
            :total="historyTotal"
            @change="loadHistory"
          />
        </div>
      </AppPanel>
    </div>

    <!-- 设置弹窗（入口在「累计中」#actions；内容走 P3 set-row） -->
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
        <UButton v-if="canEdit" color="neutral" variant="soft" :loading="settling" @click="handleSettle">
          {{ t('sub2api.activity.triggerSettle') }}
        </UButton>
        <UButton color="primary" :loading="saving" :disabled="!canEdit" @click="handleSave">
          {{ t('sub2api.activity.save') }}
        </UButton>
      </template>
    </FormDialog>

    <!-- 报名名单（点已报名人数） -->
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
              <div>
                <span>{{ t('sub2api.activity.userEmail') }}</span>
                <b>{{ userCache[r.sub2apiUserId]?.email || '—' }}</b>
              </div>
              <div>
                <span>{{ t('sub2api.activity.userBalance') }}</span>
                <b>{{ fmtMoney(userCache[r.sub2apiUserId]?.balance) }}</b>
              </div>
              <div>
                <span>{{ t('sub2api.activity.status') }}</span>
                <b>{{ userCache[r.sub2apiUserId]?.status || '—' }}</b>
              </div>
              <div>
                <span>{{ t('sub2api.activity.userRole') }}</span>
                <b>{{ userCache[r.sub2apiUserId]?.role || '—' }}</b>
              </div>
              <div>
                <span>{{ t('sub2api.activity.userCreatedAt') }}</span>
                <b>{{ fmtTime(userCache[r.sub2apiUserId]?.createdAt) }}</b>
              </div>
            </div>
            <div v-else class="warn-text">{{ userError || t('sub2api.activity.userLoadFailed') }}</div>
          </div>
        </div>
      </div>
    </FormDialog>

    <!-- 历史轮次详情（点击表行）：奖池构成 + 中奖人消费/奖金，不是列表列放大 -->
    <FormDialog
      v-model="detailOpen"
      :title="t('sub2api.activity.detailTitle', { date: detailRound?.id || '' })"
      :show-footer="false"
      :width="720"
    >
      <div v-if="detailLoading" class="hint">{{ t('sub2api.activity.loading') }}</div>
      <template v-else-if="detailRound">
        <!-- 奖池怎么来的 -->
        <section class="detail-sec">
          <h4 class="detail-sec__title">{{ t('sub2api.activity.detailPoolSection') }}</h4>
          <div class="detail-grid">
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.sourceDate') }}</span>
              <b>{{ detailRound.sourceDate || '—' }}</b>
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.groupSpendTotal') }}</span>
              <b class="num">{{ fmtMoney(detailRound.groupSpendTotal) }}</b>
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.poolRatio') }}</span>
              <b class="num">{{ fmtRatio(detailRound.poolRatio) }}</b>
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.carryIn') }}</span>
              <b class="num">{{ fmtMoney(detailRound.carryIn) }}</b>
            </div>
            <div class="detail-kv is-em">
              <span>{{ t('sub2api.activity.poolAmount') }}</span>
              <b class="num">{{ fmtMoney(detailRound.poolAmount) }}</b>
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.carryOut') }}</span>
              <b class="num">{{ fmtMoney(detailRound.carryOut) }}</b>
            </div>
          </div>
          <p class="detail-formula">
            {{
              t('sub2api.activity.poolFormula', {
                spend: fmtMoney(detailRound.groupSpendTotal),
                ratio: fmtRatio(detailRound.poolRatio),
                carry: fmtMoney(detailRound.carryIn),
                pool: fmtMoney(detailRound.poolAmount),
              })
            }}
          </p>
        </section>

        <!-- 轮次规则快照 + 时间 -->
        <section class="detail-sec">
          <h4 class="detail-sec__title">{{ t('sub2api.activity.detailRoundSection') }}</h4>
          <div class="detail-grid">
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.status') }}</span>
              <StatusPill :variant="roundStatusVariant(detailRound.status)" :label="roundStatusLabel(detailRound.status)" />
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.payout') }}</span>
              <StatusPill
                :variant="detailRound.distributed ? 'success' : 'warning'"
                :label="
                  detailRound.distributed ? t('sub2api.activity.distributed') : t('sub2api.activity.notDistributed')
                "
              />
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.threshold') }}</span>
              <b class="num">{{ fmtMoney(detailRound.threshold) }}</b>
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.eligible') }}</span>
              <b class="num">{{ detailRound.eligibleCount ?? '—' }}</b>
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.registered') }}</span>
              <b class="num">{{ detailRound.registeredCount ?? '—' }}</b>
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.winnersCount') }}</span>
              <b class="num">{{ detailRound.winnerCount ?? detailWinners.length }}</b>
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.perWinner') }}</span>
              <b class="num">{{ fmtMoney(detailRound.perWinnerAmount) }}</b>
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.baseWinners') }}</span>
              <b class="num">{{ detailRound.baseWinners ?? '—' }}</b>
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.settleTime') }}</span>
              <b>{{ fmtTime(detailRound.settleTime) }}</b>
            </div>
            <div class="detail-kv">
              <span>{{ t('sub2api.activity.drawTime') }}</span>
              <b>{{ fmtTime(detailRound.drawTime) }}</b>
            </div>
          </div>
        </section>

        <!-- 中奖人：消费快照 + 奖金 + 发放 -->
        <section class="detail-sec">
          <h4 class="detail-sec__title">
            {{ t('sub2api.activity.winners') }}
            <span class="detail-sec__count">{{ detailWinners.length }}</span>
          </h4>
          <EmptyState v-if="!detailWinners.length" :title="t('sub2api.activity.noWinners')" />
          <div v-else class="winner-list">
            <div class="winner-head">
              <span>{{ t('sub2api.activity.registrant') }}</span>
              <span class="winner-head__num">{{ t('sub2api.activity.winnerSpend') }}</span>
              <span class="winner-head__num">{{ t('sub2api.activity.prizeAmount') }}</span>
              <span class="winner-head__pay">{{ t('sub2api.activity.payout') }}</span>
            </div>
            <div v-for="w in detailWinners" :key="w.id" class="winner-item">
              <button type="button" class="winner-row" @click="toggleUser(w.sub2apiUserId)">
                <div class="winner-row__main">
                  <span class="winner-row__name">{{ w.userName || `#${w.sub2apiUserId}` }}</span>
                  <span class="winner-row__id">#{{ w.sub2apiUserId }}</span>
                </div>
                <span class="winner-row__num num">{{ fmtMoney(w.spendSnapshot) }}</span>
                <span class="winner-row__num num is-prize">{{ fmtMoney(w.prizeAmount) }}</span>
                <div class="winner-row__pay">
                  <StatusPill :variant="payoutVariant(w.payoutStatus)" :label="payoutLabel(w.payoutStatus)" />
                  <UIcon
                    :name="expandedUserId === w.sub2apiUserId ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'"
                    class="winner-row__chev"
                  />
                </div>
              </button>
              <p v-if="w.payoutError" class="winner-row__err" :title="w.payoutError">{{ w.payoutError }}</p>
              <!-- 与报名名单同款：点用户展开实时拉 Sub2API 用户详情（缓存） -->
              <div v-if="expandedUserId === w.sub2apiUserId" class="reg-detail">
                <div v-if="userLoading" class="hint">{{ t('sub2api.activity.loading') }}</div>
                <div v-else-if="userCache[w.sub2apiUserId]" class="reg-detail__grid">
                  <div>
                    <span>{{ t('sub2api.activity.userEmail') }}</span>
                    <b>{{ userCache[w.sub2apiUserId]?.email || '—' }}</b>
                  </div>
                  <div>
                    <span>{{ t('sub2api.activity.userBalance') }}</span>
                    <b>{{ fmtMoney(userCache[w.sub2apiUserId]?.balance) }}</b>
                  </div>
                  <div>
                    <span>{{ t('sub2api.activity.status') }}</span>
                    <b>{{ userCache[w.sub2apiUserId]?.status || '—' }}</b>
                  </div>
                  <div>
                    <span>{{ t('sub2api.activity.userRole') }}</span>
                    <b>{{ userCache[w.sub2apiUserId]?.role || '—' }}</b>
                  </div>
                  <div>
                    <span>{{ t('sub2api.activity.userCreatedAt') }}</span>
                    <b>{{ fmtTime(userCache[w.sub2apiUserId]?.createdAt) }}</b>
                  </div>
                </div>
                <div v-else class="warn-text">{{ userError || t('sub2api.activity.userLoadFailed') }}</div>
              </div>
            </div>
          </div>
        </section>
      </template>
      <EmptyState v-else :title="t('sub2api.activity.loadFailed')" />
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
const menuStore = useMenuStore()
const fb = useFeedback()
const canEdit = useButtonPermission([PERM.SUB2API_EDIT], [])

const TABS = [
  { name: 'lottery', labelKey: 'sub2api.activity.brand', icon: 'HOutline:GiftIcon' },
] as const

const activeActivity = ref<(typeof TABS)[number]['name']>('lottery')

const loading = ref(false)
const saving = ref(false)
const settling = ref(false)
const drawing = ref(false)
const overview = ref<GetLotteryOverviewResponse | null>(null)

const form = reactive<LotterySettings>({
  enabled: false,
  poolRatio: 0.05,
  threshold: 2,
  baseWinners: 10,
  maxWinners: 0,
  autoPayout: true,
})

// —— 历史 ——
const historyLoading = ref(false)
const historyPage = ref(1)
const historyPageSize = ref(20)
const historyTotal = ref(0)
const historyList = ref<LotteryRound[]>([])
const distributingId = ref('')

// —— 设置 ——
const settingsOpen = ref(false)

// —— 中奖/轮次详情 ——
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

const historyColumns = computed<DataTableColumn[]>(() => {
  const cols: DataTableColumn[] = [
    { key: 'id', title: t('sub2api.activity.roundDate'), minWidth: 110 },
    { key: 'status', title: t('sub2api.activity.status'), width: 100 },
    { key: 'poolAmount', title: t('sub2api.activity.poolAmount'), width: 110, align: 'right' },
    { key: 'registeredCount', title: t('sub2api.activity.registered'), width: 90, align: 'right' },
    { key: 'winnerCount', title: t('sub2api.activity.winnersCount'), width: 90, align: 'right' },
    { key: 'perWinnerAmount', title: t('sub2api.activity.perWinner'), width: 110, align: 'right' },
    { key: 'distributed', title: t('sub2api.activity.payout'), width: 100 },
  ]
  // 仅编辑权限展示「发放」；行点击看详情，不再放「中奖名单」按钮
  if (canEdit.value) {
    cols.push({ key: 'actions', title: t('sub2api.common.operation'), width: 100 })
  }
  return cols
})

const fmtMoney = (n?: number | null) => `$${(Number(n) || 0).toFixed(4)}`

const fmtRatio = (n?: number | null) => {
  const v = Number(n)
  if (!Number.isFinite(v)) return '—'
  return `${(v * 100).toFixed(v * 100 % 1 === 0 ? 0 : 2)}%`
}

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

const roundStatusLabel = (s: unknown) => {
  switch (s) {
    case LotteryRoundStatus.LOTTERY_ROUND_STATUS_DRAWN:
      return t('sub2api.activity.statusDrawn')
    case LotteryRoundStatus.LOTTERY_ROUND_STATUS_REGISTERING:
      return t('sub2api.activity.statusRegistering')
    default:
      return t('sub2api.activity.statusUnknown')
  }
}

const roundStatusVariant = (s: unknown): 'success' | 'primary' | 'neutral' => {
  switch (s) {
    case LotteryRoundStatus.LOTTERY_ROUND_STATUS_DRAWN:
      return 'success'
    case LotteryRoundStatus.LOTTERY_ROUND_STATUS_REGISTERING:
      return 'primary'
    default:
      return 'neutral'
  }
}

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
  await Promise.all([loadOverview(), loadHistory()])
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
  expandedUserId.value = null
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

const onHistoryRowClick = (row: Record<string, unknown>) => {
  const id = row.id
  if (id == null || id === '') return
  openDetail(String(id))
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
  reload()
})
</script>

<style scoped lang="scss">
.s2a-act {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.s2a-act.is-loading .s2a-tab {
  opacity: 0.55;
  pointer-events: none;
}

/* Tab 条 —— 令牌分段（与 sub2api/config · home 同款） */
.s2a-tabs {
  display: inline-flex;
  align-self: flex-start;
  padding: 3px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
}
.s2a-tabs__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 6px 16px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition:
    background 0.15s,
    color 0.15s,
    box-shadow 0.15s;
}
.s2a-tabs__btn :deep(svg) {
  width: 16px;
  height: 16px;
}
.s2a-tabs__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}

.s2a-tab {
  display: flex;
  flex-direction: column;
  gap: 12px;
  transition: opacity 0.15s;
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
.history-foot {
  margin-top: 12px;
}
.detail-sec {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.detail-sec + .detail-sec {
  margin-top: 18px;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}
.detail-sec__title {
  margin: 0;
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.detail-sec__count {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
}
.detail-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 10px 14px;
}
.detail-kv {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  b {
    font-size: 0.875rem;
    color: var(--el-text-color-primary);
    font-weight: 600;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  &.is-em b {
    color: var(--el-color-primary);
    font-size: 1rem;
  }
}
.detail-formula {
  margin: 0;
  padding: 8px 10px;
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-lighter);
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  line-height: 1.5;
  font-variant-numeric: tabular-nums;
}

/* 设置行（弹窗内，对齐 P3 set-row 密度） */
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
  line-height: 1.45;
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
.winner-head {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) 100px 100px 120px;
  gap: 10px;
  align-items: center;
  padding: 0 4px 8px;
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.04em;
  text-transform: uppercase;
  color: var(--el-text-color-secondary);
}
.winner-head__num,
.winner-head__pay {
  text-align: right;
}
.winner-item {
  border-top: 1px solid var(--el-border-color-lighter);
}
.winner-row {
  display: grid;
  grid-template-columns: minmax(0, 1.4fr) 100px 100px 120px;
  gap: 10px;
  align-items: center;
  width: 100%;
  padding: 10px 4px;
  border: none;
  background: none;
  cursor: pointer;
  text-align: left;
  color: inherit;
}
.winner-row:hover {
  background: var(--el-fill-color-lighter);
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
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.winner-row__id {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
}
.winner-row__num {
  text-align: right;
  font-size: 0.8125rem;
  color: var(--el-text-color-primary);
  font-weight: 500;
  &.is-prize {
    color: var(--el-color-primary);
    font-weight: 700;
  }
}
.winner-row__pay {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 6px;
  min-width: 0;
}
.winner-row__chev {
  width: 16px;
  height: 16px;
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}
.winner-row__err {
  margin: 0;
  padding: 0 4px 8px;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 0.6875rem;
  color: var(--el-color-danger);
}
@media (width <= 640px) {
  .winner-head {
    display: none;
  }
  .winner-row {
    grid-template-columns: 1fr 1fr;
    gap: 6px 12px;
  }
  .winner-row__main {
    grid-column: 1 / -1;
  }
  .winner-row__num {
    text-align: left;
  }
  .winner-row__pay {
    justify-content: flex-start;
  }
}
@media (width <= 768px) {
  .s2a-tabs {
    align-self: stretch;
  }
  .s2a-tabs__btn {
    flex: 1;
    justify-content: center;
  }
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
