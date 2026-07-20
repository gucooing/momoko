<!-- Sub2API 管理首页（重写 · P4/P2 去 EP）：PageHeader(#actions StatusPill+同步) + 令牌三 Tab(概览/公告/时间线)。
     概览=令牌 range seg + 自定义 datetime-local + MetricStrip(单色) + AppPanel 图表(保留 ECharts) + 最近请求 DataTable/卡 + Pagination + 详情 FormDialog。
     公告/时间线=DataTable/卡 + ActionMenu + FormDialog CRUD。toast=useFeedback；删除确认=Dialog.confirm(迁移期)。
     保留全部 store 契约(loadAdmin/loadAdminStats/loadAdminRecent/syncUsage/save*/remove* + PERM.SUB2API_EDIT)。 -->
<template>
  <div class="s2a-home">
    <PageHeader :title="t('sub2api.admin.title')" :description="t('sub2api.admin.subtitle')">
      <template #actions>
        <StatusPill :variant="statusVariant" :label="store.statusText(snapshot?.status)" />
        <UButton icon="i-lucide-external-link" color="neutral" variant="ghost" @click="openPublicHome">
          {{ t('sub2api.common.publicHome') }}
        </UButton>
        <UButton v-if="canEdit" color="neutral" variant="soft" :loading="store.syncing" @click="onSync(false)">
          {{ t('sub2api.common.incrementalSync') }}
        </UButton>
        <UButton v-if="canEdit" color="primary" :loading="store.syncing" @click="onSync(true)">
          {{ t('sub2api.common.fullSync') }}
        </UButton>
      </template>
    </PageHeader>

    <!-- Tab 条 -->
    <div class="s2a-tabs" role="tablist">
      <button
        v-for="tab in TABS"
        :key="tab.name"
        type="button"
        role="tab"
        class="s2a-tabs__btn"
        :class="{ 'is-active': activeTab === tab.name }"
        :aria-selected="activeTab === tab.name"
        @click="activeTab = tab.name"
      >
        <component :is="menuStore.iconComponents[tab.icon]" />
        {{ t(tab.labelKey) }}
      </button>
    </div>

    <!-- ========== 概览 ========== -->
    <div v-show="activeTab === 'overview'" class="s2a-tab">
      <!-- 时间区间 -->
      <div class="ov-range">
        <div class="seg" role="tablist">
          <button
            v-for="opt in rangeOptions"
            :key="opt.value"
            type="button"
            class="seg__btn"
            :class="{ 'is-active': rangeKey === opt.value }"
            @click="selectRange(opt.value)"
          >
            {{ opt.label }}
          </button>
        </div>
        <template v-if="rangeKey === 'custom'">
          <input v-model="customStart" type="datetime-local" class="app-input ov-dt" @change="onCustomChange" />
          <span class="ov-sep">~</span>
          <input v-model="customEnd" type="datetime-local" class="app-input ov-dt" @change="onCustomChange" />
        </template>
        <span class="ov-current">{{ store.adminStats?.rangeLabel }}</span>
      </div>

      <!-- 同步元信息 -->
      <div class="ov-meta">
        <span>{{ t('sub2api.common.lastSync', { time: store.formatDateTime(snapshot?.lastSyncTime) }) }}</span>
        <span>{{ t('sub2api.common.nextSync', { time: store.formatDateTime(snapshot?.nextSyncTime) }) }}</span>
        <span>{{ t('sub2api.common.latestRecord', { time: store.formatDateTime(snapshot?.latestRecordTime) }) }}</span>
        <span>{{ t('sub2api.common.dataRange', { range: snapshot?.dataRange || '-' }) }}</span>
      </div>

      <div class="ov-body" :class="{ 'is-loading': store.adminStatsLoading }">
        <!-- 指标带（单色，禁彩色边框） -->
        <MetricStrip :columns="5">
          <MetricItem
            v-for="card in store.adminStatsMetricCards"
            :key="card.label"
            :label="card.label"
            :value="card.value"
            :caption="card.detail"
          />
        </MetricStrip>

        <!-- 趋势图 -->
        <AppPanel :title="t('sub2api.common.usageTrend')">
          <VChart class="chart" :option="store.adminTrendOption" :update-options="chartUpdate" autoresize />
        </AppPanel>

        <!-- Top 图表 -->
        <div class="chart-grid">
          <AppPanel :title="t('sub2api.common.modelRequestTop')">
            <VChart class="chart chart--sm" :option="store.adminModelOption" :update-options="chartUpdate" autoresize />
          </AppPanel>
          <AppPanel :title="t('sub2api.common.groupRequestTop')">
            <VChart class="chart chart--sm" :option="store.adminGroupOption" :update-options="chartUpdate" autoresize />
          </AppPanel>
        </div>

        <!-- 最近请求 -->
        <AppPanel :title="t('sub2api.common.recentRequests')" :padded="false">
          <div class="dt-desk">
            <DataTable
              :columns="recentColumns"
              :rows="recentRows"
              row-key="_rk"
              :loading="store.recentLoading"
              :empty-text="t('sub2api.common.noRequestRecords')"
            >
              <template #cell-status="{ row }">
                <StatusPill :variant="rec(row).success ? 'success' : 'error'" :label="statusLabel(rec(row))" />
              </template>
              <template #cell-latencyMs="{ row }">{{ store.formatLatency(rec(row).latencyMs) }}</template>
              <template #cell-tokenCount="{ row }">{{ store.formatToken(rec(row).tokenCount) }}</template>
              <template #cell-requestTime="{ row }">{{ store.formatDateTime(rec(row).requestTime) }}</template>
              <template #cell-_act="{ row }">
                <UButton color="primary" variant="link" size="xs" @click="openDetail(rec(row))">
                  {{ t('sub2api.common.detail') }}
                </UButton>
              </template>
            </DataTable>
          </div>
          <div class="dt-mob">
            <article
              v-for="row in recentRows"
              :key="row._rk"
              class="req-card"
              @click="openDetail(row)"
            >
              <div class="req-card__main">
                <strong>{{ row.model || t('sub2api.common.unknownModel') }}</strong>
                <small>{{ recentSub(row) }}</small>
                <small>{{ store.formatLatency(row.latencyMs) }} · {{ store.formatToken(row.tokenCount) }} · {{ store.formatDateTime(row.requestTime) }}</small>
              </div>
              <div class="req-card__side">
                <StatusPill :variant="row.success ? 'success' : 'error'" :label="statusLabel(row)" />
                <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" class="req-card__chevron" />
              </div>
            </article>
            <EmptyState v-if="!recentRows.length && !store.recentLoading" :title="t('sub2api.common.noRequestRecords')" />
          </div>
          <template v-if="store.adminRecentTotal > store.recentPageSize" #footer>
            <Pagination
              :page="store.recentPage"
              :page-size="store.recentPageSize"
              :total="store.adminRecentTotal"
              @update:page="onRecentPageChange"
            />
          </template>
        </AppPanel>
      </div>
    </div>

    <!-- ========== 公告 ========== -->
    <div v-show="activeTab === 'announcements'" class="s2a-tab">
      <div class="list-toolbar">
        <span class="list-toolbar__count">{{ t('sub2api.common.totalItems', { count: store.announcements.length }) }}</span>
        <UButton v-if="canEdit" color="primary" icon="i-lucide-plus" @click="openAnnouncement()">
          {{ t('sub2api.admin.addAnnouncement') }}
        </UButton>
      </div>
      <AppPanel :padded="false">
        <div class="dt-desk">
          <DataTable
            :columns="annColumns"
            :rows="store.announcements"
            :loading="store.listLoading"
            :empty-text="t('sub2api.home.noAnnouncements')"
          >
            <template #cell-level="{ row }">
              <StatusPill :variant="levelVariant(ann(row).level)" :label="levelText(ann(row).level)" />
            </template>
            <template #cell-pinned="{ row }">
              <StatusPill v-if="ann(row).pinned" variant="warning" :dot="false" :label="t('sub2api.common.pinned')" />
              <span v-else class="muted">—</span>
            </template>
            <template #cell-publishedAt="{ row }">{{ store.formatDateTime(ann(row).publishedAt) }}</template>
            <template #cell-_act="{ row }">
              <ActionMenu :items="annActions" @select="(key) => onAnnAction(key, ann(row))" />
            </template>
          </DataTable>
        </div>
        <div class="dt-mob">
          <article v-for="row in store.announcements" :key="row.id" class="edit-card">
            <div class="edit-card__head">
              <strong>{{ row.title || t('sub2api.common.announcements') }}</strong>
              <span class="edit-card__pills">
                <StatusPill :variant="levelVariant(row.level)" :label="levelText(row.level)" />
                <StatusPill v-if="row.pinned" variant="warning" :dot="false" :label="t('sub2api.common.pinned')" />
              </span>
            </div>
            <p>{{ row.content || '-' }}</p>
            <div class="edit-card__foot">
              <time>{{ store.formatDateTime(row.publishedAt) }}</time>
              <span v-if="canEdit" class="edit-card__acts">
                <UButton color="primary" variant="link" size="xs" @click="openAnnouncement(row)">{{ t('sub2api.common.edit') }}</UButton>
                <UButton color="error" variant="link" size="xs" @click="onDeleteAnnouncement(row)">{{ t('sub2api.common.delete') }}</UButton>
              </span>
            </div>
          </article>
          <EmptyState v-if="!store.announcements.length && !store.listLoading" :title="t('sub2api.home.noAnnouncements')" />
        </div>
      </AppPanel>
    </div>

    <!-- ========== 时间线 ========== -->
    <div v-show="activeTab === 'timeline'" class="s2a-tab">
      <div class="list-toolbar">
        <span class="list-toolbar__count">{{ t('sub2api.common.totalItems', { count: store.timeline.length }) }}</span>
        <UButton v-if="canEdit" color="primary" icon="i-lucide-plus" @click="openTimeline()">
          {{ t('sub2api.admin.addTimeline') }}
        </UButton>
      </div>
      <AppPanel :padded="false">
        <div class="dt-desk">
          <DataTable
            :columns="tlColumns"
            :rows="store.timeline"
            :loading="store.listLoading"
            :empty-text="t('sub2api.common.noData')"
          >
            <template #cell-publishedAt="{ row }">{{ store.formatDateTime(tl(row).publishedAt) }}</template>
            <template #cell-_act="{ row }">
              <ActionMenu :items="tlActions" @select="(key) => onTlAction(key, tl(row))" />
            </template>
          </DataTable>
        </div>
        <div class="dt-mob">
          <article v-for="row in store.timeline" :key="row.id" class="edit-card">
            <div class="edit-card__head">
              <strong>{{ row.title || t('sub2api.common.update') }}</strong>
              <StatusPill :variant="'info'" :dot="false" :label="row.category || t('sub2api.common.update')" />
            </div>
            <p>{{ row.content || '-' }}</p>
            <div class="edit-card__foot">
              <time>{{ store.formatDateTime(row.publishedAt) }}</time>
              <span v-if="canEdit" class="edit-card__acts">
                <UButton color="primary" variant="link" size="xs" @click="openTimeline(row)">{{ t('sub2api.common.edit') }}</UButton>
                <UButton color="error" variant="link" size="xs" @click="onDeleteTimeline(row)">{{ t('sub2api.common.delete') }}</UButton>
              </span>
            </div>
          </article>
          <EmptyState v-if="!store.timeline.length && !store.listLoading" :title="t('sub2api.common.noData')" />
        </div>
      </AppPanel>
    </div>

    <!-- 公告弹窗 -->
    <FormDialog v-model="announcementDialog" :title="annForm.id ? t('sub2api.admin.editAnnouncement') : t('sub2api.admin.addAnnouncement')" :width="520">
      <div class="dlg-form">
        <div class="app-field">
          <label class="app-label">{{ t('sub2api.common.title') }}</label>
          <input v-model="annForm.title" class="app-input" :disabled="!canEdit" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('sub2api.common.content') }}</label>
          <textarea v-model="annForm.content" class="app-textarea" rows="4" :disabled="!canEdit" />
        </div>
        <div class="dlg-form__grid">
          <div class="app-field">
            <label class="app-label">{{ t('sub2api.common.level') }}</label>
            <AppSelect v-model="annForm.level" :options="levelOptions" :disabled="!canEdit" />
          </div>
          <div class="app-field app-field--row">
            <label class="app-label">{{ t('sub2api.common.pinned') }}</label>
            <AppSwitch v-model="annForm.pinned" :disabled="!canEdit" />
          </div>
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('sub2api.common.publishTime') }}</label>
          <input v-model="annForm.publishedAt" type="datetime-local" class="app-input" :disabled="!canEdit" />
          <span class="app-hint">{{ t('sub2api.common.defaultNow') }}</span>
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('sub2api.common.cancel') }}</UButton>
        <UButton v-if="canEdit" color="primary" :loading="store.listLoading" @click="submitAnnouncement">{{ t('sub2api.common.save') }}</UButton>
      </template>
    </FormDialog>

    <!-- 时间线弹窗 -->
    <FormDialog v-model="timelineDialog" :title="tlForm.id ? t('sub2api.admin.editTimeline') : t('sub2api.admin.addTimeline')" :width="520">
      <div class="dlg-form">
        <div class="app-field">
          <label class="app-label">{{ t('sub2api.common.title') }}</label>
          <input v-model="tlForm.title" class="app-input" :disabled="!canEdit" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('sub2api.common.content') }}</label>
          <textarea v-model="tlForm.content" class="app-textarea" rows="4" :disabled="!canEdit" />
        </div>
        <div class="dlg-form__grid">
          <div class="app-field">
            <label class="app-label">{{ t('sub2api.common.category') }}</label>
            <input v-model="tlForm.category" class="app-input" :placeholder="t('sub2api.common.update')" :disabled="!canEdit" />
          </div>
          <div class="app-field">
            <label class="app-label">{{ t('sub2api.common.publishTime') }}</label>
            <input v-model="tlForm.publishedAt" type="datetime-local" class="app-input" :disabled="!canEdit" />
          </div>
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('sub2api.common.cancel') }}</UButton>
        <UButton v-if="canEdit" color="primary" :loading="store.listLoading" @click="submitTimeline">{{ t('sub2api.common.save') }}</UButton>
      </template>
    </FormDialog>

    <!-- 最近请求详情 -->
    <FormDialog v-model="detailDialog" :title="t('sub2api.common.requestDetail')" :width="540">
      <div v-if="detailRow" class="kv">
        <div class="kv__row">
          <span class="kv__k">{{ t('sub2api.common.status') }}</span>
          <span class="kv__v"><StatusPill :variant="detailRow.success ? 'success' : 'error'" :label="statusLabel(detailRow)" /></span>
        </div>
        <div class="kv__row"><span class="kv__k">{{ t('sub2api.common.model') }}</span><span class="kv__v">{{ detailRow.model || '-' }}</span></div>
        <div class="kv__row"><span class="kv__k">{{ t('sub2api.common.endpoint') }}</span><span class="kv__v">{{ detailRow.endpoint || '-' }}</span></div>
        <div class="kv__row"><span class="kv__k">{{ t('sub2api.common.accountName') }}</span><span class="kv__v">{{ detailRow.accountName || '-' }}</span></div>
        <div class="kv__row"><span class="kv__k">{{ t('sub2api.common.group') }}</span><span class="kv__v">{{ detailRow.groupName || '-' }}</span></div>
        <div class="kv__row"><span class="kv__k">{{ t('sub2api.common.cost') }}</span><span class="kv__v">{{ store.formatCost(detailRow.cost) }}</span></div>
        <div class="kv__row"><span class="kv__k">{{ t('sub2api.common.token') }}</span><span class="kv__v">{{ store.formatToken(detailRow.tokenCount) }}</span></div>
        <div class="kv__row"><span class="kv__k">{{ t('sub2api.common.duration') }}</span><span class="kv__v">{{ store.formatLatency(detailRow.latencyMs) }}</span></div>
        <div class="kv__row"><span class="kv__k">{{ t('sub2api.common.firstToken') }}</span><span class="kv__v">{{ firstTokenText(detailRow.firstTokenMs) }}</span></div>
        <div class="kv__row"><span class="kv__k">{{ t('sub2api.common.reasoningEffort') }}</span><span class="kv__v">{{ detailRow.reasoningEffort || '-' }}</span></div>
        <div class="kv__row"><span class="kv__k">{{ t('sub2api.common.time') }}</span><span class="kv__v">{{ store.formatDateTime(detailRow.requestTime) }}</span></div>
        <template v-if="!detailRow.success">
          <div class="kv__row"><span class="kv__k">{{ t('sub2api.common.httpCode') }}</span><span class="kv__v">{{ detailRow.httpStatus || '-' }}</span></div>
          <div class="kv__row kv__row--block">
            <span class="kv__k">{{ t('sub2api.common.errorDetail') }}</span>
            <span class="kv__v kv__v--err">{{ detailRow.errorMessage || '-' }}</span>
          </div>
        </template>
        <div class="kv__row kv__row--block">
          <span class="kv__k">{{ t('sub2api.common.requestId') }}</span>
          <span class="kv__v kv__v--mono">{{ detailRow.requestId || '-' }}</span>
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('sub2api.common.close') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import VChart from '@/components/chart/VChart.vue'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { useSub2APIStore } from '@/stores/sub2api'
import { useFeedback } from '@/utils/feedback'
import { Dialog } from '@/utils/dialog'
import type {
  Sub2APIAnnouncement,
  Sub2APIRecentRequest,
  Sub2APITimelineItem,
} from '@/types/v1/sub2api'

defineOptions({ name: 'Sub2APIHome' })

const store = useSub2APIStore()
const router = useRouter()
const menuStore = useMenuStore()
const { snapshot } = storeToRefs(store)
const { t } = useI18n()
const fb = useFeedback()
const canEdit = useButtonPermission([PERM.SUB2API_EDIT], [])

const chartUpdate = { notMerge: true }

const TABS = [
  { name: 'overview', labelKey: 'sub2api.common.usageOverview', icon: 'HOutline:ChartBarIcon' },
  { name: 'announcements', labelKey: 'sub2api.common.announcements', icon: 'HOutline:MegaphoneIcon' },
  { name: 'timeline', labelKey: 'sub2api.common.timeline', icon: 'HOutline:ClockIcon' },
] as const
const activeTab = ref<'overview' | 'announcements' | 'timeline'>('overview')

const STATUS_VARIANT = { warning: 'warning', success: 'success', danger: 'error', info: 'info' } as const
const statusVariant = computed(
  () => STATUS_VARIANT[store.statusType(snapshot.value?.status) as keyof typeof STATUS_VARIANT] ?? 'neutral',
)

// —— 用量概览时间区间：默认近 24h ——
const rangeOptions = computed(() => [
  { label: t('sub2api.common.today'), value: 'today' as const },
  { label: t('sub2api.common.last24h'), value: '24h' as const },
  { label: t('sub2api.common.sevenDays'), value: '7d' as const },
  { label: t('sub2api.common.thirtyDays'), value: '30d' as const },
  { label: t('sub2api.common.all'), value: 'all' as const },
  { label: t('sub2api.common.custom'), value: 'custom' as const },
])
type RangeKey = 'today' | '24h' | '7d' | '30d' | 'all' | 'custom'
const rangeKey = ref<RangeKey>('24h')
// 自定义时间段：datetime-local 字符串（精度到分钟）
const customStart = ref('')
const customEnd = ref('')

const HOUR = 3_600_000
const MINUTE = 60_000
const floorMinute = (ms: number) => ms - (ms % MINUTE)
const startOfTodayMs = () => {
  const d = new Date()
  d.setHours(0, 0, 0, 0)
  return d.getTime()
}
// 本地 datetime-local 字符串 <-> 时间戳
const toLocalInput = (v?: string | number | Date | null) => {
  if (!v) return ''
  const d = new Date(v)
  if (Number.isNaN(d.getTime())) return ''
  const p = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}`
}

// 当前选择对应的 [startTime, endTime]（Unix 毫秒，精度到分钟）；startTime=0=不限起点。
const resolveRange = (): { startTime: number; endTime: number } => {
  const end = floorMinute(Date.now())
  switch (rangeKey.value) {
    case 'today':
      return { startTime: floorMinute(startOfTodayMs()), endTime: end }
    case '24h':
      return { startTime: end - 24 * HOUR, endTime: end }
    case '7d':
      return { startTime: end - 7 * 24 * HOUR, endTime: end }
    case '30d':
      return { startTime: end - 30 * 24 * HOUR, endTime: end }
    case 'all':
      return { startTime: 0, endTime: end }
    case 'custom':
      if (customStart.value && customEnd.value) {
        let s = floorMinute(new Date(customStart.value).getTime())
        let e = floorMinute(new Date(customEnd.value).getTime())
        if (Number.isNaN(s) || Number.isNaN(e)) return { startTime: end - 24 * HOUR, endTime: end }
        if (s > e) [s, e] = [e, s]
        return { startTime: s, endTime: e }
      }
      return { startTime: end - 24 * HOUR, endTime: end }
    default:
      return { startTime: end - 24 * HOUR, endTime: end }
  }
}

const lastRange = ref<{ startTime: number; endTime: number }>({ startTime: 0, endTime: 0 })
const reloadStats = () => {
  const range = resolveRange()
  lastRange.value = range
  store.loadAdminStats(range.startTime, range.endTime)
  store.loadAdminRecent(range.startTime, range.endTime, 1)
}
const onRecentPageChange = (page: number) => {
  store.loadAdminRecent(lastRange.value.startTime, lastRange.value.endTime, page)
}
const selectRange = (v: RangeKey) => {
  rangeKey.value = v
  if (v === 'custom' && (!customStart.value || !customEnd.value)) {
    const now = new Date()
    customStart.value = toLocalInput(new Date(now.getTime() - 24 * HOUR))
    customEnd.value = toLocalInput(now)
  }
  reloadStats()
}
const onCustomChange = () => {
  if (customStart.value && customEnd.value) reloadStats()
}

// —— 最近请求 ——
const recentColumns = computed<DataTableColumn[]>(() => [
  { key: 'model', title: t('sub2api.common.model'), minWidth: 140 },
  { key: 'endpoint', title: t('sub2api.common.endpoint'), minWidth: 130 },
  { key: 'accountName', title: t('sub2api.common.account'), minWidth: 110 },
  { key: 'groupName', title: t('sub2api.common.group'), minWidth: 110 },
  { key: 'status', title: t('sub2api.common.status'), width: 112 },
  { key: 'latencyMs', title: t('sub2api.common.latency'), width: 92, align: 'right' },
  { key: 'tokenCount', title: 'Token', width: 92, align: 'right' },
  { key: 'requestTime', title: t('sub2api.common.time'), minWidth: 160 },
  { key: '_act', title: t('sub2api.common.operation'), width: 72, align: 'right' },
])
// DataTable 需要稳定 rowKey：requestId 可能为空/重复 → 合成 _rk
const recentRows = computed(() =>
  store.adminRecent.map((r, i) => ({ ...r, _rk: `${r.requestId || ''}-${r.requestTime || ''}-${i}` })),
)
// DataTable 单元格插槽的 row 为 Record<string, unknown>，按列语义收窄
const rec = (r: Record<string, unknown>) => r as unknown as Sub2APIRecentRequest
const ann = (r: Record<string, unknown>) => r as unknown as Sub2APIAnnouncement
const tl = (r: Record<string, unknown>) => r as unknown as Sub2APITimelineItem

const statusLabel = (row: Sub2APIRecentRequest) => {
  if (row.success) return t('sub2api.common.success')
  return row.httpStatus
    ? t('sub2api.common.failedWithCode', { code: row.httpStatus })
    : t('sub2api.common.failed')
}
const recentSub = (row: Sub2APIRecentRequest) =>
  [row.accountName, row.groupName, row.endpoint || '-'].filter(Boolean).join(' · ')
const firstTokenText = (ms?: number) => (ms && ms > 0 ? store.formatLatency(ms) : '-')

// 详情
const detailDialog = ref(false)
const detailRow = ref<Sub2APIRecentRequest>()
const openDetail = (row: Sub2APIRecentRequest) => {
  detailRow.value = row
  detailDialog.value = true
}

const openPublicHome = () => {
  window.open(router.resolve('/public/sub2api/home').href, '_blank')
}
const onSync = async (full: boolean) => {
  if (!canEdit.value) return
  const ok = await store.syncUsage(full)
  if (ok) {
    fb.success(t('sub2api.common.synced'))
    reloadStats()
  }
}

// —— 级别（公告） ——
const levelOptions = computed<{ label: string; value: string }[]>(() => [
  { label: t('sub2api.common.info'), value: 'info' },
  { label: t('sub2api.common.success'), value: 'success' },
  { label: t('sub2api.common.warning'), value: 'warning' },
  { label: t('sub2api.common.danger'), value: 'danger' },
])
const levelText = (level: string) =>
  ({
    info: t('sub2api.common.info'),
    success: t('sub2api.common.success'),
    warning: t('sub2api.common.warning'),
    danger: t('sub2api.common.danger'),
  })[level] || t('sub2api.common.info')
const levelVariant = (level: string): 'info' | 'success' | 'warning' | 'error' =>
  (({ info: 'info', success: 'success', warning: 'warning', danger: 'error' }) as const)[
    level as 'info' | 'success' | 'warning' | 'danger'
  ] || 'info'

// —— 公告 CRUD ——
const annColumns = computed<DataTableColumn[]>(() => {
  const cols: DataTableColumn[] = [
    { key: 'title', title: t('sub2api.common.title'), minWidth: 160 },
    { key: 'content', title: t('sub2api.common.content'), minWidth: 220 },
    { key: 'level', title: t('sub2api.common.level'), width: 110 },
    { key: 'pinned', title: t('sub2api.common.pinned'), width: 90 },
    { key: 'publishedAt', title: t('sub2api.common.publishTime'), width: 180 },
  ]
  if (canEdit.value) cols.push({ key: '_act', title: t('sub2api.common.operation'), width: 70, align: 'right' })
  return cols
})
const annActions = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('sub2api.common.edit'), icon: 'i-lucide-pencil' },
  { key: 'delete', label: t('sub2api.common.delete'), icon: 'i-lucide-trash-2', danger: true },
])
const onAnnAction = (key: string, row: Sub2APIAnnouncement) => {
  if (key === 'edit') openAnnouncement(row)
  else if (key === 'delete') onDeleteAnnouncement(row)
}

const announcementDialog = ref(false)
const annForm = reactive<{ id?: string; title: string; content: string; level: string; pinned: boolean; publishedAt: string }>({
  title: '',
  content: '',
  level: 'info',
  pinned: false,
  publishedAt: '',
})
const openAnnouncement = (row?: Sub2APIAnnouncement) => {
  if (!canEdit.value) return
  Object.assign(annForm, {
    id: row?.id,
    title: row?.title || '',
    content: row?.content || '',
    level: row?.level || 'info',
    pinned: row?.pinned || false,
    publishedAt: toLocalInput(row?.publishedAt),
  })
  announcementDialog.value = true
}
const submitAnnouncement = async () => {
  if (!canEdit.value) return
  const ok = await store.saveAnnouncement({
    id: annForm.id,
    title: annForm.title,
    content: annForm.content,
    level: annForm.level,
    pinned: annForm.pinned,
    publishedAt: annForm.publishedAt ? new Date(annForm.publishedAt) : undefined,
  })
  if (ok) {
    announcementDialog.value = false
    fb.success(t('sub2api.common.saved'))
  }
}
const onDeleteAnnouncement = (row: Sub2APIAnnouncement) => {
  if (!canEdit.value) return
  Dialog.confirm({
    title: t('sub2api.common.tip'),
    content: t('sub2api.admin.confirmDeleteAnnouncement'),
    onConfirm: async () => {
      if (await store.removeAnnouncement(row.id)) fb.success(t('sub2api.common.deleted'))
    },
  }).catch(() => undefined)
}

// —— 时间线 CRUD ——
const tlColumns = computed<DataTableColumn[]>(() => {
  const cols: DataTableColumn[] = [
    { key: 'title', title: t('sub2api.common.title'), minWidth: 160 },
    { key: 'content', title: t('sub2api.common.content'), minWidth: 220 },
    { key: 'category', title: t('sub2api.common.category'), width: 130 },
    { key: 'publishedAt', title: t('sub2api.common.publishTime'), width: 180 },
  ]
  if (canEdit.value) cols.push({ key: '_act', title: t('sub2api.common.operation'), width: 70, align: 'right' })
  return cols
})
const tlActions = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('sub2api.common.edit'), icon: 'i-lucide-pencil' },
  { key: 'delete', label: t('sub2api.common.delete'), icon: 'i-lucide-trash-2', danger: true },
])
const onTlAction = (key: string, row: Sub2APITimelineItem) => {
  if (key === 'edit') openTimeline(row)
  else if (key === 'delete') onDeleteTimeline(row)
}

const timelineDialog = ref(false)
const tlForm = reactive<{ id?: string; title: string; content: string; category: string; publishedAt: string }>({
  title: '',
  content: '',
  category: t('sub2api.common.update'),
  publishedAt: '',
})
const openTimeline = (row?: Sub2APITimelineItem) => {
  if (!canEdit.value) return
  Object.assign(tlForm, {
    id: row?.id,
    title: row?.title || '',
    content: row?.content || '',
    category: row?.category || t('sub2api.common.update'),
    publishedAt: toLocalInput(row?.publishedAt),
  })
  timelineDialog.value = true
}
const submitTimeline = async () => {
  if (!canEdit.value) return
  const ok = await store.saveTimelineItem({
    id: tlForm.id,
    title: tlForm.title,
    content: tlForm.content,
    category: tlForm.category,
    publishedAt: tlForm.publishedAt ? new Date(tlForm.publishedAt) : undefined,
  })
  if (ok) {
    timelineDialog.value = false
    fb.success(t('sub2api.common.saved'))
  }
}
const onDeleteTimeline = (row: Sub2APITimelineItem) => {
  if (!canEdit.value) return
  Dialog.confirm({
    title: t('sub2api.common.tip'),
    content: t('sub2api.admin.confirmDeleteTimeline'),
    onConfirm: async () => {
      if (await store.removeTimelineItem(row.id)) fb.success(t('sub2api.common.deleted'))
    },
  }).catch(() => undefined)
}

onMounted(() => {
  store.loadAdmin()
  reloadStats()
})
</script>

<style scoped lang="scss">
.s2a-home {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* Tab 条（与 config 一致） */
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
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
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
  gap: 14px;
}

/* range 区间 */
.ov-range {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.seg {
  display: inline-flex;
  padding: 3px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.seg::-webkit-scrollbar {
  display: none;
}
.seg__btn {
  flex: none;
  padding: 5px 12px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.seg__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}
.ov-dt {
  width: 190px;
}
.ov-sep {
  color: var(--el-text-color-secondary);
}
.ov-current {
  margin-left: auto;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
}

/* 同步元信息 */
.ov-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px 22px;
  color: var(--el-text-color-secondary);
  font-size: 0.78rem;
  font-variant-numeric: tabular-nums;
}

.ov-body {
  display: flex;
  flex-direction: column;
  gap: 14px;
  transition: opacity 0.15s;
}
.ov-body.is-loading {
  opacity: 0.6;
  pointer-events: none;
}

.chart {
  width: 100%;
  height: 320px;
}
.chart--sm {
  height: 280px;
}
.chart-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

/* 列表工具条 */
.list-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.list-toolbar__count {
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
}

/* 表格 / 卡片切换 */
.dt-desk {
  display: block;
}
.dt-mob {
  display: none;
}

/* 最近请求卡 */
.req-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 11px 16px;
  cursor: pointer;
  border-bottom: 1px solid var(--el-border-color-lighter);
  transition: background 0.12s;
}
.req-card:last-child {
  border-bottom: 0;
}
.req-card:hover {
  background: var(--el-fill-color-light);
}
.req-card__main {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.req-card__main strong {
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.req-card__main small {
  color: var(--el-text-color-secondary);
  font-size: 0.72rem;
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.req-card__side {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  flex: none;
}
.req-card__chevron {
  width: 14px;
  height: 14px;
  color: var(--el-text-color-placeholder);
}

/* 公告 / 时间线 卡 */
.edit-card {
  padding: 12px 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.edit-card:last-child {
  border-bottom: 0;
}
.edit-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}
.edit-card__head strong {
  min-width: 0;
  color: var(--el-text-color-primary);
  font-size: 0.875rem;
  line-height: 1.4;
  overflow-wrap: anywhere;
}
.edit-card__pills {
  display: inline-flex;
  flex: none;
  gap: 6px;
}
.edit-card p {
  margin: 7px 0;
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  line-height: 1.55;
  white-space: pre-wrap;
  overflow-wrap: anywhere;
}
.edit-card__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.edit-card__foot time {
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
}
.edit-card__acts {
  display: inline-flex;
  gap: 4px;
}

.muted {
  color: var(--el-text-color-placeholder);
}

/* 弹窗表单 */
.dlg-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.dlg-form__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}
.app-field {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.app-field--row {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
}
.app-hint {
  color: var(--el-text-color-placeholder);
  font-size: 0.72rem;
}

/* 详情 KV */
.kv {
  display: flex;
  flex-direction: column;
}
.kv__row {
  display: grid;
  grid-template-columns: 92px minmax(0, 1fr);
  gap: 10px;
  padding: 8px 0;
  border-bottom: 1px solid var(--el-border-color-lighter);
  font-size: 0.8125rem;
}
.kv__row:last-child {
  border-bottom: 0;
}
.kv__k {
  color: var(--el-text-color-secondary);
}
.kv__v {
  min-width: 0;
  color: var(--el-text-color-primary);
  overflow-wrap: anywhere;
  font-variant-numeric: tabular-nums;
}
.kv__row--block {
  grid-template-columns: 1fr;
  gap: 4px;
}
.kv__v--err {
  color: var(--el-color-danger);
  white-space: pre-wrap;
}
.kv__v--mono {
  font-family: var(--el-font-family-mono, monospace);
  font-size: 0.75rem;
  color: var(--el-text-color-regular);
}

@media (width <= 1024px) {
  .chart-grid {
    grid-template-columns: 1fr;
  }
}

@media (width <= 768px) {
  .s2a-tabs {
    align-self: stretch;
    display: flex;
  }
  .s2a-tabs__btn {
    flex: 1;
    justify-content: center;
  }
  .dt-desk {
    display: none;
  }
  .dt-mob {
    display: flex;
    flex-direction: column;
  }
  .ov-dt {
    width: 100%;
  }
  .ov-sep {
    display: none;
  }
  .ov-current {
    width: 100%;
    margin-left: 0;
  }
  .chart {
    height: 240px;
  }
  .chart--sm {
    height: 220px;
  }
  .dlg-form__grid {
    grid-template-columns: 1fr;
  }
}
</style>
