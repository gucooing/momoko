<!-- 端口转发：标题+计数说明 → 子导航 → 概览（主色流量 hero + 规则/连接/速率卡）→ 状态 chips + 协议分段 + 搜索
     → 表格（PC）/ 信息卡（手机）→ 底部悬浮批量条。统计弹窗查看曲线与活动连接。 -->
<template>
  <div class="pf-page">
    <PageHeader :title="t('tools.portForward.title')">
      <template #description>
        {{ t('tools.portForward.summary', { total: overview.total }) }}
        <template v-if="overview.running">
          · <span class="pf-tone-ok">{{ t('tools.portForward.summaryRunning', { count: overview.running }) }}</span>
        </template>
        <template v-if="overview.error">
          · <span class="pf-tone-bad">{{ t('tools.portForward.summaryError', { count: overview.error }) }}</span>
        </template>
      </template>
      <template #actions>
        <AppDropdown align="end" :width="200">
          <template #trigger>
            <UButton color="neutral" variant="soft" icon="i-lucide-arrow-up-down">
              {{ t('tools.portForward.importExport') }}
            </UButton>
          </template>
          <template #default="{ close }">
            <div class="pf-menu">
              <button type="button" class="pf-menu__item" @click="close(); pickImportFile()">
                <component :is="menuStore.iconComponents['HOutline:ArrowUpTrayIcon']" />{{ t('tools.portForward.importRules') }}
              </button>
              <button type="button" class="pf-menu__item" @click="close(); exportRules()">
                <component :is="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" />{{ t('tools.portForward.exportAll') }}
              </button>
            </div>
          </template>
        </AppDropdown>
        <UButton color="primary" icon="i-lucide-plus" @click="openCreateDialog()">
          {{ t('tools.portForward.addRule') }}
        </UButton>
      </template>
    </PageHeader>


    <!-- 概览 -->
    <section class="pf-overview">
      <div class="pf-hero">
        <span class="pf-hero__blob" aria-hidden="true" />
        <span class="pf-hero__ring" aria-hidden="true" />
        <span class="pf-hero__label">{{ t('tools.portForward.todayTraffic') }}</span>
        <span class="pf-hero__value m-num">
          {{ todayTraffic[0] }}<small>{{ todayTraffic[1] }}</small>
        </span>
        <span class="pf-hero__sub m-num">
          {{ t('tools.portForward.overviewTrafficSub', { in: formatBytes(overview.todayBytesIn), out: formatBytes(overview.todayBytesOut) }) }}
          <template v-if="trafficDelta"> · {{ trafficDelta }}</template>
        </span>
      </div>
      <div class="pf-tile">
        <span class="pf-tile__label"><span class="m-sticker m-sticker--a2 pf-tile__ico" style="--rot: -4deg"><component :is="menuStore.iconComponents['HOutline:ArrowsRightLeftIcon']" /></span>{{ t('tools.portForward.overviewRules') }}</span>
        <span class="pf-tile__value m-num">{{ overview.total }}</span>
        <span class="pf-tile__sub">{{ t('tools.portForward.overviewRulesSub', { run: overview.running, stop: overview.stopped, err: overview.error }) }}</span>
      </div>
      <div class="pf-tile">
        <span class="pf-tile__label"><span class="m-sticker m-sticker--ok pf-tile__ico" style="--rot: 3deg"><component :is="menuStore.iconComponents['HOutline:LinkIcon']" /></span>{{ t('tools.portForward.overviewConns') }}</span>
        <span class="pf-tile__value m-num">{{ overview.activeConnections }}</span>
        <span class="pf-tile__sub m-num">
          {{ t('tools.portForward.peakToday', { n: overview.todayPeakConnections }) }}<template v-if="peakTime"> · {{ peakTime }}</template>
        </span>
      </div>
      <div class="pf-tile pf-tile--rate">
        <span class="pf-tile__label"><span class="m-sticker m-sticker--a pf-tile__ico" style="--rot: -2deg"><component :is="menuStore.iconComponents['HOutline:BoltIcon']" /></span>{{ t('tools.portForward.overviewRate') }}</span>
        <span class="pf-tile__value m-num">{{ formatRate(overview.rateIn + overview.rateOut, '0 B/s') }}</span>
        <span class="pf-tile__sub m-num">↓ {{ formatRate(overview.rateIn, '0 B/s') }} · ↑ {{ formatRate(overview.rateOut, '0 B/s') }}</span>
      </div>
    </section>

    <ListToolbar
      v-model:status="queryForm.status"
      v-model:keyword="queryForm.keywords"
      :chips="statusChips"
      :chips-label="t('tools.portForward.statusCol')"
      :placeholder="t('tools.portForward.searchPlaceholder')"
      @search="search"
    >
      <div class="pf-seg" role="radiogroup" :aria-label="t('tools.portForward.forwardType')">
        <button
          v-for="opt in protoOptions"
          :key="opt.value"
          type="button"
          role="radio"
          class="pf-seg__btn"
          :class="{ 'is-on': queryForm.type === opt.value }"
          :aria-checked="queryForm.type === opt.value"
          @click="setProto(opt.value)"
        >{{ opt.label }}</button>
      </div>
    </ListToolbar>

    <div class="pf-page__body">
      <DataTable
        v-if="!menuStore.isMobile"
        v-model="selectedIds"
        :columns="columns"
        :rows="list"
        row-key="id"
        selectable
        :loading="loading"
        :empty-text="t('tools.portForward.noData')"
      >
        <template #cell-name="{ row }">
          <div class="pf-name">
            <span class="pf-name__main">{{ row.name }}</span>
            <span v-if="row.remark" class="pf-name__sub">{{ row.remark }}</span>
          </div>
        </template>
        <template #cell-type="{ row }">
          <span class="pf-proto" :class="isUdp(row) ? 'is-udp' : 'is-tcp'">{{ protoLabel(row) }}</span>
        </template>
        <template #cell-route="{ row }">
          <span class="pf-route m-mono">
            <span>{{ row.listenAddress }}:{{ row.listenPort }}</span>
            <component :is="menuStore.iconComponents['HOutline:ArrowRightIcon']" class="pf-route__arrow" />
            <span class="pf-route__target">{{ row.targetAddress }}:{{ row.targetPort }}</span>
          </span>
        </template>
        <template #cell-status="{ row }">
          <div class="pf-status">
            <StatusPill :variant="statusVariant(row)" :label="statusLabel(row)" />
            <span v-if="row.error" class="pf-status__err" :title="String(row.error)">{{ row.error }}</span>
          </div>
        </template>
        <template #cell-connections="{ row }">
          <span class="m-num pf-strong">{{ isRunning(row) ? Number(row.activeConnections || 0) : '—' }}</span>
        </template>
        <template #cell-rate="{ row }">
          <div class="pf-rate">
            <svg class="pf-spark" viewBox="0 0 80 28" preserveAspectRatio="none" aria-hidden="true">
              <path v-if="spark(row).area" :d="spark(row).area" class="pf-spark__area" />
              <path :d="spark(row).line" class="pf-spark__line" :class="{ 'is-idle': !isRunning(row) }" />
            </svg>
            <div class="pf-rate__nums m-num">
              <span><i class="pf-up">↑</i> {{ formatRate(row.rateOut) }}</span>
              <span><i class="pf-down">↓</i> {{ formatRate(row.rateIn) }}</span>
            </div>
          </div>
        </template>
        <template #cell-total="{ row }">
          <span class="m-num pf-strong">{{ formatBytes(Number(row.totalBytesIn || 0) + Number(row.totalBytesOut || 0)) }}</span>
        </template>
        <template #cell-operation="{ row }">
          <div class="pf-acts">
            <AppIconButton
              icon="HOutline:ChartBarIcon"
              :label="t('tools.portForward.stats.details')"
              :box="36"
              :size="18"
              :active="statsVisible && statsRow?.id === row.id"
              @click="openStatsDialog(findRow(String(row.id))!)"
            />
            <AppIconButton
              :icon="toggleIcon(row)"
              :label="toggleLabel(row)"
              :box="36"
              :size="18"
              :class="row.isEnable ? 'pf-act--stop' : 'pf-act--start'"
              @click="toggleEnable(row, !row.isEnable || isError(row))"
            />
            <AppIconButton
              icon="HOutline:PencilSquareIcon"
              :label="t('tools.portForward.edit')"
              :box="36"
              :size="18"
              @click="openEditDialog(findRow(String(row.id))!)"
            />
            <ActionMenu :items="rowActions" @select="(key) => onRowAction(key, row)" />
          </div>
        </template>
      </DataTable>

      <template v-else>
        <div v-if="loading && !list.length" class="pf-cards">
          <div v-for="i in 3" :key="i" class="pf-skeleton" />
        </div>
        <EmptyState
          v-else-if="!list.length"
          icon="HOutline:ArrowsRightLeftIcon"
          :title="t('tools.portForward.noData')"
          :description="t('tools.portForward.emptyDesc')"
        >
          <template #action>
            <UButton color="primary" variant="soft" icon="i-lucide-plus" @click="openCreateDialog()">
              {{ t('tools.portForward.addRule') }}
            </UButton>
          </template>
        </EmptyState>
        <div v-else class="pf-cards">
          <article
            v-for="row in list"
            :key="row.id"
            class="pf-card"
            :class="{ 'is-selected': selectedIds.includes(row.id) }"
          >
            <header class="pf-card__head">
              <span class="m-sticker pf-card__sticker m-mono" :class="isUdp(row) ? 'm-sticker--a' : 'm-sticker--a2'">{{ protoLabel(row) }}</span>
              <span class="pf-card__title">
                <span class="pf-card__name">{{ row.name }}</span>
                <span v-if="row.remark" class="pf-card__remark">{{ row.remark }}</span>
              </span>
              <StatusPill :variant="statusVariant(row)" :label="statusLabel(row)" />
              <label class="pf-card__check" :aria-label="t('tools.portForward.selectRule', { name: row.name })">
                <input type="checkbox" :checked="selectedIds.includes(row.id)" @change="toggleSelect(row.id)" />
              </label>
            </header>
            <div class="pf-card__route m-mono">
              <span>{{ row.listenAddress }}:{{ row.listenPort }}</span>
              <component :is="menuStore.iconComponents['HOutline:ArrowRightIcon']" class="pf-route__arrow" />
              <span class="pf-route__target">{{ row.targetAddress }}:{{ row.targetPort }}</span>
            </div>
            <dl class="pf-card__grid">
              <div>
                <dt>{{ t('tools.portForward.rateCol') }}</dt>
                <dd class="m-num"><i class="pf-up">↑</i> {{ formatRate(row.rateOut) }}<br /><i class="pf-down">↓</i> {{ formatRate(row.rateIn) }}</dd>
              </div>
              <div class="pf-card__spark">
                <svg class="pf-spark" viewBox="0 0 80 28" preserveAspectRatio="none" aria-hidden="true">
                  <path v-if="spark(row).area" :d="spark(row).area" class="pf-spark__area" />
                  <path :d="spark(row).line" class="pf-spark__line" :class="{ 'is-idle': !isRunning(row) }" />
                </svg>
              </div>
              <div>
                <dt>{{ t('tools.portForward.overviewConns') }}</dt>
                <dd class="m-num">{{ isRunning(row) ? Number(row.activeConnections || 0) : '—' }}</dd>
              </div>
              <div>
                <dt>{{ t('tools.portForward.totalCol') }}</dt>
                <dd class="m-num">{{ formatBytes(Number(row.totalBytesIn || 0) + Number(row.totalBytesOut || 0)) }}</dd>
              </div>
            </dl>
            <div v-if="row.error" class="pf-card__err">
              <component :is="menuStore.iconComponents['HOutline:ExclamationTriangleIcon']" />
              <span>{{ row.error }}</span>
            </div>
            <footer class="pf-card__foot">
              <span class="pf-card__meta">{{ t('tools.portForward.createdAt', { time: formatDateTime(row.createTime) }) }}</span>
              <AppIconButton icon="HOutline:ChartBarIcon" :label="t('tools.portForward.stats.details')" :box="44" :size="20" @click="openStatsDialog(row)" />
              <AppIconButton
                :icon="toggleIcon(row)"
                :label="toggleLabel(row)"
                :box="44"
                :size="20"
                :class="row.isEnable ? 'pf-act--stop' : 'pf-act--start'"
                @click="toggleEnable(row, !row.isEnable || isError(row))"
              />
              <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('tools.portForward.edit')" :box="44" :size="20" @click="openEditDialog(row)" />
              <ActionMenu :items="rowActions" @select="(key) => onRowAction(key, row)" />
            </footer>
          </article>
        </div>
      </template>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="getList()"
      />

      <p class="pf-note">
        <component :is="menuStore.iconComponents['HOutline:InformationCircleIcon']" />
        {{ t('tools.portForward.footNote') }}
      </p>
    </div>

    <Transition name="pf-pop">
      <div v-if="selectedIds.length" class="pf-batch" role="toolbar">
        <span class="pf-batch__count">{{ t('tools.portForward.selectedRules', { count: selectedIds.length }) }}</span>
        <button type="button" class="pf-batch__btn" @click="batchSetEnable(true)">
          <component :is="menuStore.iconComponents['HOutline:PlayIcon']" />{{ t('tools.portForward.actionEnable') }}
        </button>
        <button type="button" class="pf-batch__btn" @click="batchSetEnable(false)">
          <component :is="menuStore.iconComponents['HOutline:StopIcon']" />{{ t('tools.portForward.actionDisable') }}
        </button>
        <button type="button" class="pf-batch__btn" @click="exportRules(selectedIds)">
          <component :is="menuStore.iconComponents['HOutline:ArrowDownTrayIcon']" />{{ t('tools.portForward.export') }}
        </button>
        <button type="button" class="pf-batch__btn is-danger" @click="batchDelete">
          <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />{{ t('tools.portForward.delete') }}
        </button>
        <button type="button" class="pf-batch__btn is-icon" :aria-label="t('tools.portForward.clearSelection')" @click="selectedIds = []">
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
        </button>
      </div>
    </Transition>

    <input ref="importInput" type="file" accept="application/json,.json" hidden @change="onImportFile" />

    <PortForwardCreate ref="createRef" @refresh="refresh" />
    <PortForwardStatsDialog v-model="statsVisible" :row="statsRow" @changed="getList(true)" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import PortForwardCreate from '@/views/tools/port-forward/create.vue'
import PortForwardStatsDialog from '@/views/tools/port-forward/PortForwardStatsDialog.vue'
import { formatBytes, formatRate, formatDateTime, formatClock, splitBytes, sparkPath, downloadText } from '@/views/tools/components/format'
import {
  listPortForwards,
  deletePortForward,
  updatePortForward,
  createPortForward,
  resetPortForwardTraffic,
} from '@/api/network'
import {
  PortForwardType,
  PortForwardStatus,
  type PortForwardInfo,
  type PortForwardOverview,
} from '@/types/v1/network'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import { Dialog } from '@/utils/dialog'

defineOptions({ name: 'PortForwardView' })

type Row = Record<string, unknown>
type StatusFilter = '' | PortForwardStatus
type ProtoFilter = '' | PortForwardType

const POLL_MS = 4000

const menuStore = useMenuStore()
const { t } = useI18n()
const createRef = useTemplateRef<InstanceType<typeof PortForwardCreate> | null>('createRef')
const importInput = ref<HTMLInputElement | null>(null)

const selectedIds = ref<string[]>([])
const list = ref<PortForwardInfo[]>([])
const loading = ref(false)
const statsVisible = ref(false)
const statsRow = ref<PortForwardInfo | null>(null)

const emptyOverview = (): PortForwardOverview => ({
  total: 0, running: 0, stopped: 0, error: 0, tcp: 0, udp: 0,
  activeConnections: 0, rateIn: 0, rateOut: 0, todayBytesIn: 0, todayBytesOut: 0,
  yesterdayBytes: 0, todayPeakConnections: 0, todayPeakTime: undefined,
})
const overview = ref<PortForwardOverview>(emptyOverview())

const queryForm = reactive({ keywords: '', type: '' as ProtoFilter, status: '' as StatusFilter })
const pagination = ref({ page: 1, pageSize: 20, total: 0 })

// ---- 概览 ----
const num = (v: unknown) => Number(v || 0)
const todayTraffic = computed(() => splitBytes(num(overview.value.todayBytesIn) + num(overview.value.todayBytesOut)))
const trafficDelta = computed(() => {
  const today = num(overview.value.todayBytesIn) + num(overview.value.todayBytesOut)
  const y = num(overview.value.yesterdayBytes)
  if (!y) return ''
  const pct = Math.round(((today - y) / y) * 100)
  return pct >= 0 ? t('tools.portForward.vsYesterdayUp', { pct }) : t('tools.portForward.vsYesterdayDown', { pct: -pct })
})
const peakTime = computed(() => (overview.value.todayPeakConnections ? formatClock(overview.value.todayPeakTime) : ''))

// ---- 筛选 ----
const statusChips = computed<ToolbarChip<StatusFilter>[]>(() => [
  { label: t('tools.portForward.allStatus'), value: '', count: num(overview.value.total) },
  { label: t('tools.portForward.statusRunning'), value: PortForwardStatus.PORT_FORWARD_STATUS_RUNNING, count: num(overview.value.running) },
  { label: t('tools.portForward.statusStopped'), value: PortForwardStatus.PORT_FORWARD_STATUS_STOPPED, count: num(overview.value.stopped) },
  { label: t('tools.portForward.statusError'), value: PortForwardStatus.PORT_FORWARD_STATUS_ERROR, count: num(overview.value.error) },
])
const protoOptions = computed(() => [
  { label: t('tools.portForward.allProtocols'), value: '' as ProtoFilter },
  { label: 'TCP', value: PortForwardType.PORT_FORWARD_TYPE_TCP as ProtoFilter },
  { label: 'UDP', value: PortForwardType.PORT_FORWARD_TYPE_UDP as ProtoFilter },
])
const setProto = (v: ProtoFilter) => {
  queryForm.type = v
  search()
}

// ---- 行展示 ----
const isUdp = (row: Row) => row.type === PortForwardType.PORT_FORWARD_TYPE_UDP
const protoLabel = (row: Row) => (isUdp(row) ? 'UDP' : 'TCP')
const isRunning = (row: Row) => row.status === PortForwardStatus.PORT_FORWARD_STATUS_RUNNING || (row.status === PortForwardStatus.PORT_FORWARD_STATUS_ERROR && !!row.startTime)
const isError = (row: Row) => row.status === PortForwardStatus.PORT_FORWARD_STATUS_ERROR
const statusVariant = (row: Row) => {
  if (row.status === PortForwardStatus.PORT_FORWARD_STATUS_RUNNING) return 'success'
  if (isError(row)) return 'error'
  return 'neutral'
}
const statusLabel = (row: Row) => {
  if (row.status === PortForwardStatus.PORT_FORWARD_STATUS_RUNNING) return t('tools.portForward.statusRunning')
  if (isError(row)) return t('tools.portForward.statusError')
  return t('tools.portForward.statusStopped')
}
const toggleIcon = (row: Row) => {
  if (isError(row)) return 'HOutline:ArrowPathIcon'
  return row.isEnable ? 'HOutline:StopIcon' : 'HOutline:PlayIcon'
}
const toggleLabel = (row: Row) => {
  if (isError(row)) return t('tools.portForward.actionRetry')
  return row.isEnable ? t('tools.portForward.actionDisable') : t('tools.portForward.actionEnable')
}
const spark = (row: Row) => sparkPath(isRunning(row) ? (row.rateHistory as number[] | undefined) : [])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('tools.portForward.ruleName'), minWidth: 150 },
  { key: 'type', title: t('tools.portForward.protocolCol'), width: 76 },
  { key: 'route', title: t('tools.portForward.routeCol'), minWidth: 250 },
  { key: 'status', title: t('tools.portForward.statusCol'), minWidth: 120 },
  { key: 'connections', title: t('tools.portForward.connCol'), width: 70, align: 'right' },
  { key: 'rate', title: t('tools.portForward.rateCol'), width: 190 },
  { key: 'total', title: t('tools.portForward.totalCol'), width: 100, align: 'right' },
  { key: 'operation', title: t('tools.portForward.operation'), width: 176, align: 'right' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  { key: 'copy', label: t('tools.portForward.copyListen'), icon: 'HOutline:ClipboardDocumentIcon' },
  { key: 'export', label: t('tools.portForward.export'), icon: 'HOutline:ArrowDownTrayIcon' },
  { key: 'resetTraffic', label: t('tools.portForward.resetTraffic'), icon: 'HOutline:ArrowUturnLeftIcon' },
  { key: 'delete', label: t('tools.portForward.delete'), icon: 'HOutline:TrashIcon', danger: true },
])

const toggleSelect = (id: string) => {
  selectedIds.value = selectedIds.value.includes(id)
    ? selectedIds.value.filter((d) => d !== id)
    : [...selectedIds.value, id]
}

const findRow = (id: string) => list.value.find((x) => x.id === id)
const onRowAction = (key: string, row: Row) => {
  const record = findRow(String(row.id))
  if (!record) return
  if (key === 'delete') confirmDelete(record)
  else if (key === 'export') exportRules([record.id])
  else if (key === 'resetTraffic') confirmResetTraffic(record)
  else if (key === 'copy') copyListen(record)
}

// ---- 数据 ----
let pollTimer: ReturnType<typeof setInterval> | null = null

const getList = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    const { data } = await listPortForwards({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.keywords || undefined,
      type: queryForm.type || undefined,
      status: queryForm.status || undefined,
    })
    list.value = data?.infos || []
    pagination.value.total = Number(data?.total) || 0
    overview.value = { ...emptyOverview(), ...(data?.overview || {}) }
    const ids = new Set(list.value.map((r) => r.id))
    if (selectedIds.value.some((id) => !ids.has(id))) selectedIds.value = selectedIds.value.filter((id) => ids.has(id))
    if (statsRow.value) statsRow.value = findRow(statsRow.value.id) || statsRow.value
  } catch {
    if (!silent) {
      list.value = []
      pagination.value.total = 0
    }
  } finally {
    loading.value = false
  }
}
const search = () => {
  pagination.value.page = 1
  getList()
}

const openCreateDialog = () => createRef.value?.showDialog()
const openEditDialog = (row: PortForwardInfo) => createRef.value?.showDialog(row)
const openStatsDialog = (row: PortForwardInfo) => {
  statsRow.value = row
  statsVisible.value = true
}

const toggleEnable = async (row: Row, val: boolean) => {
  const record = findRow(String(row.id))
  if (!record) return
  try {
    const { data } = await updatePortForward({ id: record.id, isEnable: val })
    if (data?.info) {
      Object.assign(record, data.info)
      if (data.info.error) feedback.error(data.info.error)
      else feedback.success(data.info.isEnable ? t('tools.portForward.enabled') : t('tools.portForward.disabled'))
    }
    getList(true)
  } catch {
    /* interceptor */
  }
}

const batchSetEnable = async (val: boolean) => {
  const ids = [...selectedIds.value]
  const results = await Promise.allSettled(ids.map((id) => updatePortForward({ id, isEnable: val })))
  const ok = results.filter((r) => r.status === 'fulfilled').length
  feedback.success(val ? t('tools.portForward.batchEnableSuccess', { count: ok }) : t('tools.portForward.batchDisableSuccess', { count: ok }))
  getList(true)
}

const copyListen = async (row: PortForwardInfo) => {
  try {
    await navigator.clipboard.writeText(`${row.listenAddress}:${row.listenPort}`)
    feedback.success(t('tools.tunnel.frpc.copied'))
  } catch {
    feedback.error(t('tools.tunnel.frpc.copyFailed'))
  }
}

const confirmResetTraffic = (row: PortForwardInfo) => {
  Dialog.confirm({
    title: t('tools.portForward.resetTraffic'),
    content: t('tools.portForward.resetTrafficConfirm', { name: row.name }),
    confirmText: t('tools.portForward.resetTrafficOk'),
    cancelText: t('tools.portForward.cancel'),
    onConfirm: async () => {
      await resetPortForwardTraffic({ id: row.id })
      feedback.success(t('tools.portForward.resetTrafficDone'))
      getList(true)
    },
  })
}

const confirmDelete = (row: PortForwardInfo) => {
  Dialog.confirm({
    title: t('tools.portForward.confirmDeleteTitle'),
    content: t('tools.portForward.confirmDeleteRule'),
    confirmText: t('tools.portForward.confirmDeleteText'),
    cancelText: t('tools.portForward.cancel'),
    onConfirm: async () => {
      await deletePortForward({ id: row.id })
      feedback.success(t('tools.portForward.deleteSuccess'))
      selectedIds.value = selectedIds.value.filter((d) => d !== row.id)
      getList()
    },
  })
}

const batchDelete = () => {
  if (!selectedIds.value.length) return
  Dialog.confirm({
    title: t('tools.portForward.confirmDeleteTitle'),
    content: t('tools.portForward.confirmDeleteSelected'),
    confirmText: t('tools.portForward.confirmDeleteText'),
    cancelText: t('tools.portForward.cancel'),
    onConfirm: async () => {
      await Promise.all(selectedIds.value.map((id) => deletePortForward({ id })))
      feedback.success(t('tools.portForward.batchDeleteSuccess'))
      selectedIds.value = []
      getList()
    },
  })
}

// ---- 导入 / 导出（JSON） ----
type ExportedRule = {
  name: string
  type: 'TCP' | 'UDP'
  listenAddress: string
  listenPort: number
  targetAddress: string
  targetPort: number
  isEnable: boolean
  remark: string
}

const exportRules = async (ids?: string[]) => {
  try {
    const { data } = await listPortForwards({ page: 1, pageSize: 0 })
    const all = data?.infos || []
    const picked = ids?.length ? all.filter((r) => ids.includes(r.id)) : all
    const rules: ExportedRule[] = picked.map((r) => ({
      name: r.name,
      type: r.type === PortForwardType.PORT_FORWARD_TYPE_UDP ? 'UDP' : 'TCP',
      listenAddress: r.listenAddress,
      listenPort: r.listenPort,
      targetAddress: r.targetAddress,
      targetPort: r.targetPort,
      isEnable: r.isEnable,
      remark: r.remark,
    }))
    const stamp = formatDateTime(new Date()).replace(/[-: ]/g, '')
    downloadText(`port-forwards-${stamp}.json`, JSON.stringify({ version: 1, rules }, null, 2), 'application/json')
    feedback.success(t('tools.portForward.exportDone', { count: rules.length }))
  } catch {
    /* interceptor */
  }
}

const pickImportFile = () => importInput.value?.click()
const onImportFile = async (e: Event) => {
  const input = e.target as HTMLInputElement
  const file = input.files?.[0]
  input.value = ''
  if (!file) return
  let rules: Partial<ExportedRule>[] = []
  try {
    const parsed = JSON.parse(await file.text()) as { rules?: Partial<ExportedRule>[] } | Partial<ExportedRule>[]
    rules = Array.isArray(parsed) ? parsed : parsed.rules || []
  } catch {
    feedback.error(t('tools.portForward.importInvalid'))
    return
  }
  const valid = rules.filter((r) => r.name && r.listenPort && r.targetPort && r.targetAddress)
  if (!valid.length) {
    feedback.error(t('tools.portForward.importInvalid'))
    return
  }
  const results = await Promise.allSettled(
    valid.map((r) =>
      createPortForward({
        name: String(r.name),
        type: String(r.type).toUpperCase() === 'UDP' ? PortForwardType.PORT_FORWARD_TYPE_UDP : PortForwardType.PORT_FORWARD_TYPE_TCP,
        listenAddress: r.listenAddress || '0.0.0.0',
        listenPort: Number(r.listenPort),
        targetAddress: String(r.targetAddress),
        targetPort: Number(r.targetPort),
        isEnable: r.isEnable ?? false,
        remark: r.remark || '',
      }),
    ),
  )
  const ok = results.filter((r) => r.status === 'fulfilled').length
  feedback.success(t('tools.portForward.importDone', { ok, total: valid.length }))
  search()
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') pagination.value.page = 1
  getList()
}

const startPoll = () => {
  pollTimer = setInterval(() => {
    if (!document.hidden && !loading.value) void getList(true)
  }, POLL_MS)
}
const stopPoll = () => {
  if (pollTimer) clearInterval(pollTimer)
  pollTimer = null
}
onMounted(() => {
  getList()
  startPoll()
})
// 页面被缓存（切到同组其它页）时暂停轮询；回来时静默刷新一次再继续
onActivated(() => {
  if (pollTimer) return
  void getList(true)
  startPoll()
})
onDeactivated(stopPoll)
onBeforeUnmount(stopPoll)
</script>

<style scoped lang="scss">
.pf-page {
  display: flex;
  flex-direction: column;
  gap: 18px;
}
.pf-page__body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.pf-tone-ok { color: var(--m-ok); font-weight: 700; }
.pf-tone-bad { color: var(--m-bad); font-weight: 700; }

/* 导入/导出 菜单 */
.pf-menu { padding: 6px; }
.pf-menu__item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  min-height: 40px;
  padding: 8px 12px;
  border: none;
  border-radius: 12px;
  background: transparent;
  color: var(--el-text-color-regular);
  font-size: 0.875rem;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.pf-menu__item:hover { background: var(--el-fill-color-light); color: var(--el-text-color-primary); }
.pf-menu__item :deep(svg) { width: 16px; height: 16px; flex: none; }

/* 概览 */
.pf-overview {
  display: grid;
  grid-template-columns: 1.3fr 1fr 1fr 1fr;
  gap: 16px;
}
.pf-hero {
  position: relative;
  overflow: hidden;
  display: flex;
  flex-direction: column;
  gap: 6px;
  padding: 20px 22px;
  border-radius: var(--app-radius-xl);
  background: var(--m-p);
  color: var(--m-pOn);
}
.pf-hero__blob {
  position: absolute;
  right: -34px;
  bottom: -48px;
  width: 130px;
  height: 130px;
  border-radius: 50%;
  background: var(--m-a);
}
.pf-hero__ring {
  position: absolute;
  right: 92px;
  top: -18px;
  width: 48px;
  height: 48px;
  border-radius: 50%;
  border: 8px solid var(--m-a2);
}
.pf-hero__label,
.pf-hero__value,
.pf-hero__sub { position: relative; }
.pf-hero__label { font-size: 0.8125rem; font-weight: 700; }
.pf-hero__value {
  font-family: var(--app-font-display);
  font-size: 2.375rem;
  font-weight: 800;
  line-height: 1.05;
  small { margin-left: 4px; font-size: 1.125rem; }
}
.pf-hero__sub { font-size: 0.8125rem; font-weight: 600; }
.pf-tile {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
  padding: 20px 22px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.pf-tile__label {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 0.8125rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.pf-tile__ico {
  transform: rotate(var(--rot, 0deg));
  width: 28px;
  height: 28px;
  border-radius: 9px;
  :deep(svg) { width: 16px; height: 16px; }
}
.pf-tile__value {
  font-family: var(--app-font-display);
  font-size: 1.875rem;
  font-weight: 800;
  line-height: 1.1;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pf-tile__sub {
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* 协议分段 */
.pf-seg {
  display: inline-flex;
  padding: 3px;
  gap: 2px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
}
.pf-seg__btn {
  min-height: 34px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.pf-seg__btn.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}

/* 表格单元 */
.pf-name { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
.pf-name__main { font-weight: 800; color: var(--el-text-color-primary); }
.pf-name__sub {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pf-proto {
  display: inline-flex;
  align-items: center;
  height: 24px;
  padding: 0 9px;
  border-radius: 8px;
  font-family: var(--app-font-mono);
  font-size: 0.72rem;
  font-weight: 800;
}
.pf-proto.is-tcp { background: var(--m-a2Soft); color: var(--m-ink); }
.pf-proto.is-udp { background: var(--m-aSoft); color: var(--m-ink); }
.pf-route {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
}
.pf-route__arrow { width: 14px; height: 14px; flex: none; color: var(--m-p); }
.pf-route__target { font-weight: 700; color: var(--el-text-color-primary); }
.pf-status { display: flex; flex-direction: column; align-items: flex-start; gap: 4px; min-width: 0; }
.pf-status__err {
  max-width: 220px;
  font-size: 0.72rem;
  color: var(--m-bad);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pf-strong { font-weight: 700; color: var(--el-text-color-primary); }
.pf-rate { display: flex; align-items: center; gap: 10px; }
.pf-rate__nums {
  display: flex;
  flex-direction: column;
  font-size: 0.75rem;
  line-height: 1.35;
  color: var(--el-text-color-regular);
  white-space: nowrap;
}
.pf-up, .pf-down { font-style: normal; font-weight: 800; }
.pf-up { color: var(--m-a2); }
.pf-down { color: var(--m-pInk); }
.pf-spark { width: 80px; height: 28px; flex: none; overflow: visible; }
.pf-spark__area { fill: var(--m-pSoft); opacity: 0.8; }
.pf-spark__line { fill: none; stroke: var(--m-p); stroke-width: 2; stroke-linecap: round; vector-effect: non-scaling-stroke; }
.pf-spark__line.is-idle { stroke: var(--el-border-color-light); }
.pf-acts { display: inline-flex; align-items: center; justify-content: flex-end; gap: 2px; }
.pf-acts :deep(.pf-act--stop), .pf-card__foot :deep(.pf-act--stop) { color: var(--m-bad); }
.pf-acts :deep(.pf-act--start), .pf-card__foot :deep(.pf-act--start) { color: var(--m-ok); }
.pf-acts :deep(.pf-act--stop:hover), .pf-card__foot :deep(.pf-act--stop:hover) { color: var(--m-bad); background: var(--m-badSoft); }
.pf-acts :deep(.pf-act--start:hover), .pf-card__foot :deep(.pf-act--start:hover) { color: var(--m-ok); background: var(--m-okSoft); }

.pf-note {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  :deep(svg) { width: 16px; height: 16px; flex: none; }
}

/* 手机卡片 */
.pf-cards { display: flex; flex-direction: column; gap: 12px; }
.pf-card {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 16px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
}
.pf-card.is-selected { border-color: var(--m-p); box-shadow: 0 0 0 2px var(--m-pSoft); }
.pf-card__head { display: flex; align-items: center; gap: 10px; min-width: 0; }
.pf-card__sticker {
  width: 44px;
  height: 44px;
  flex: none;
  border-radius: 14px;
  font-size: 0.75rem;
  font-weight: 800;
}
.pf-card__title { display: flex; flex-direction: column; gap: 2px; flex: 1; min-width: 0; }
.pf-card__name {
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pf-card__remark {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pf-card__check {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 44px;
  flex: none;
  input { width: 18px; height: 18px; accent-color: var(--m-p); }
}
.pf-card__route {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  span { white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
}
.pf-card__grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 10px 14px;
  margin: 0;
  dt { font-size: 0.72rem; font-weight: 700; color: var(--el-text-color-secondary); }
  dd { margin: 2px 0 0; font-size: 0.875rem; font-weight: 700; color: var(--el-text-color-primary); line-height: 1.4; }
}
.pf-card__spark { display: flex; align-items: center; .pf-spark { width: 100%; height: 36px; } }
.pf-card__err {
  display: flex;
  align-items: flex-start;
  gap: 8px;
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--m-badSoft);
  color: var(--m-bad);
  font-size: 0.8125rem;
  font-weight: 600;
  :deep(svg) { width: 16px; height: 16px; flex: none; margin-top: 2px; }
}
.pf-card__foot { display: flex; align-items: center; gap: 2px; }
.pf-card__meta {
  flex: 1;
  min-width: 0;
  font-size: 0.72rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.pf-skeleton {
  height: 220px;
  border-radius: var(--app-radius-xl);
  background: var(--el-fill-color-light);
  animation: pf-pulse 1.4s ease-in-out infinite;
}
@keyframes pf-pulse { 50% { opacity: 0.55; } }

/* 批量条 */
.pf-batch {
  position: sticky;
  bottom: 18px;
  z-index: 20;
  align-self: center;
  display: flex;
  align-items: center;
  gap: 2px;
  max-width: 100%;
  padding: 6px 6px 6px 18px;
  border-radius: 999px;
  background: var(--m-ink);
  color: var(--m-bg);
  box-shadow: var(--app-shadow-lg);
  overflow-x: auto;
  scrollbar-width: none;
}
.pf-batch::-webkit-scrollbar { display: none; }
.pf-batch__count { margin-right: 10px; font-size: 0.875rem; font-weight: 800; white-space: nowrap; }
.pf-batch__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: transparent;
  color: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
}
.pf-batch__btn:hover { background: color-mix(in srgb, var(--m-bg) 14%, transparent); }
.pf-batch__btn.is-danger { color: var(--m-a); }
.pf-batch__btn.is-icon { width: 40px; padding: 0; justify-content: center; }
.pf-batch__btn :deep(svg) { width: 16px; height: 16px; }
.pf-pop-enter-active, .pf-pop-leave-active { transition: opacity 0.18s, transform 0.18s; }
.pf-pop-enter-from, .pf-pop-leave-to { opacity: 0; transform: translateY(12px); }

@media (width <= 1180px) {
  .pf-overview { grid-template-columns: 1fr 1fr; }
}
@media (width <= 768px) {
  .pf-page { gap: 14px; }
  .pf-overview { grid-template-columns: 1fr 1fr; gap: 10px; }
  .pf-hero { grid-column: 1 / -1; padding: 18px 20px; }
  .pf-hero__value { font-size: 2.125rem; }
  .pf-tile { padding: 14px 16px; border-radius: var(--app-radius-lg); }
  .pf-tile__value { font-size: 1.5rem; }
  .pf-tile__ico { display: none; }
  .pf-tile--rate { display: none; }
  .pf-seg { width: 100%; }
  .pf-seg__btn { flex: 1; min-height: 38px; }
  .pf-batch { bottom: calc(78px + env(safe-area-inset-bottom)); padding-left: 14px; }
  .pf-batch__btn { padding: 0 10px; }
  .pf-batch__btn.is-icon { width: 40px; padding: 0; }
}
</style>
