<!-- GM 运营 · GM 记录：待审批卡片（批准 / 驳回）、筛选（执行者类型、待审批、失败）、时间范围、风险、搜索、
     分页表格、详情抽屉（不含请求参数）与按当前筛选导出 CSV。 -->
<template>
  <div class="gml game-skin">
    <div class="gml__head">
      <div class="gml__head-text">
        <h2 class="gml__h">{{ t('gmOps.logs.title') }}</h2>
        <p class="gml__sub">{{ t('gmOps.logs.subline') }}</p>
      </div>
      <UButton color="neutral" variant="outline" icon="i-lucide-download" :loading="exporting" :disabled="exporting || !total" @click="exportCsv">
        {{ exporting && progress ? progress : t('gmOps.logs.export') }}
      </UButton>
    </div>

    <!-- 待审批 -->
    <section v-if="pending.length" class="gml__pending" :aria-label="t('gmOps.logs.pendingTitle')">
      <span class="gml__pending-count">{{ t('gmOps.logs.pendingTitle') }} <span class="m-num">{{ pending.length }}</span></span>
      <div v-for="p in pending" :key="p.id" class="gml__pending-row">
        <span class="gml__avatar" aria-hidden="true">{{ (p.actorName || '?').slice(0, 1) }}</span>
        <span class="gml__pending-text">
          <span class="gml__pending-title">{{ t('gmOps.logs.applies', { who: actorName(p), title: commandTitle(p) }) }}</span>
          <span class="gml__hint">{{ pendingMeta(p) }}</span>
        </span>
        <GmRiskPill :risk="p.risk" />
        <AppIconButton icon="HOutline:EyeIcon" :label="t('gmOps.logs.view')" @click="openDetail(p)" />
        <UButton color="neutral" variant="outline" size="sm" @click="openReject(p)">{{ t('gmOps.logs.reject') }}</UButton>
        <UButton color="primary" size="sm" icon="i-lucide-check" @click="openApprove(p)">{{ t('gmOps.logs.approve') }}</UButton>
      </div>
    </section>

    <div class="gml__filters">
      <div class="gml__chips" role="tablist" :aria-label="t('gmOps.logs.filterLabel')">
        <button
          v-for="c in chips"
          :key="c.value"
          type="button"
          role="tab"
          class="gml__chip"
          :class="{ 'is-on': filter === c.value }"
          :aria-selected="filter === c.value"
          @click="pickFilter(c.value)"
        >
          {{ c.label }}
          <span class="gml__chip-n m-num">{{ c.count }}</span>
        </button>
      </div>
      <span class="gml__spacer" />
      <AppDropdown align="end" :width="300">
        <template #trigger="{ open }">
          <button type="button" class="gml__chip" :aria-expanded="open">
            <UIcon name="i-lucide-calendar" />
            {{ rangeLabel }}
            <UIcon name="i-lucide-chevron-down" />
          </button>
        </template>
        <template #default="{ close }">
          <div class="gml__range">
            <button
              v-for="r in ranges"
              :key="r"
              type="button"
              class="gml__range-opt"
              :class="{ 'is-on': range === r }"
              @click="pickRange(r, close)"
            >
              {{ t(`gmOps.logs.range.${r}`) }}
            </button>
            <div class="gml__range-custom">
              <span class="gml__hint">{{ t('gmOps.logs.range.custom') }}</span>
              <div class="gml__range-dates">
                <input v-model="customFrom" type="date" class="gml__input" :aria-label="t('gmOps.logs.range.from')" />
                <span>—</span>
                <input v-model="customTo" type="date" class="gml__input" :aria-label="t('gmOps.logs.range.to')" />
              </div>
              <UButton color="primary" size="sm" :disabled="!customFrom" @click="pickRange('custom', close)">{{ t('gmOps.logs.range.apply') }}</UButton>
            </div>
          </div>
        </template>
      </AppDropdown>
      <div class="gml__risk">
        <AppSelect v-model="risk" :options="riskOptions" />
      </div>
      <label class="gml__search">
        <UIcon name="i-lucide-search" />
        <input v-model="keyword" type="search" :placeholder="t('gmOps.logs.search')" :aria-label="t('gmOps.logs.search')" @input="onSearch" />
      </label>
    </div>

    <section class="gml__card">
      <template v-if="!menuStore.isMobile">
        <DataTable
          :columns="columns"
          :rows="rows"
          row-key="id"
          :model-value="detailId ? [detailId] : []"
          :loading="loading"
          :empty-text="t('gmOps.logs.empty')"
          @row-click="(row) => openDetail(asExec(row))"
        >
          <template #cell-time="{ row }">
            <span class="m-num gml__hint gml__time" :title="fmtDateTime(asExec(row).createTime, true)">{{ fmtShort(asExec(row).createTime, today) }}</span>
          </template>
          <template #cell-actor="{ row }">
            <span class="gml__actor">
              <span class="gml__avatar gml__avatar--sm" :class="{ 'is-player': asExec(row).actorKind === 'player' }" aria-hidden="true">{{ (asExec(row).actorName || '?').slice(0, 1) }}</span>
              <span class="gml__actor-text">
                <span class="gml__actor-name">{{ actorName(asExec(row)) }}</span>
                <span class="gml__hint">{{ kindText(asExec(row).actorKind) }}</span>
              </span>
            </span>
          </template>
          <template #cell-command="{ row }">
            <span class="gml__cmd">
              <span class="gml__cmd-title">{{ commandTitle(asExec(row)) }}</span>
              <span class="gml__cmd-sub">{{ objectText(asExec(row)) }}</span>
            </span>
          </template>
          <template #cell-risk="{ row }">
            <GmRiskPill :risk="asExec(row).risk" />
          </template>
          <template #cell-status="{ row }">
            <StatusPill :variant="statusPill(asExec(row).status)" :label="statusText(asExec(row).status)" />
          </template>
        </DataTable>
      </template>
      <template v-else>
        <EmptyState v-if="!items.length && !loading" icon="HOutline:ClockIcon" :title="t('gmOps.logs.empty')" />
        <button v-for="e in items" :key="e.id" type="button" class="gml__mcard" @click="openDetail(e)">
          <span class="gml__mcard-top">
            <span class="gml__cmd-title">{{ commandTitle(e) }}</span>
            <StatusPill :variant="statusPill(e.status)" :label="statusText(e.status)" />
          </span>
          <span class="gml__cmd-sub">{{ objectText(e) }}</span>
          <span class="gml__mcard-meta">
            <span class="gml__hint">{{ e.actorName || '—' }} · {{ kindText(e.actorKind) }} · {{ fmtShort(e.createTime, today) }}</span>
            <GmRiskPill :risk="e.risk" />
          </span>
        </button>
      </template>
      <div class="gml__pager">
        <span class="gml__hint">{{ t('gmOps.logs.totalLine', { range: rangeLabel, n: total }) }}</span>
        <Pagination v-if="total > pageSize" v-model:page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[20, 50, 100]" @change="reload()" />
      </div>
    </section>

    <SideDrawer v-model="detailOpen" :title="t('gmOps.logs.detailTitle')" :width="520" :game="ctx.game.value">
      <LogDetail v-if="detailId" :id="detailId" :preview="detailPreview" />
    </SideDrawer>

    <FormDialog
      v-model="approveOpen"
      :title="t('gmOps.logs.approveTitle')"
      :width="480"
      :loading="acting"
      :confirm-text="t('gmOps.logs.approve')"
      @confirm="approve"
    >
      <div v-if="acted" class="gml__dialog">
        <p class="gml__dialog-text">{{ t('gmOps.logs.approveDesc', { who: actorName(acted), title: commandTitle(acted) }) }}</p>
        <p v-if="acted.risk === 'danger'" class="gml__dialog-text gml__hint">{{ t('gmOps.logs.dangerHint') }}</p>
        <label class="gml__field">
          <span class="gml__label">{{ t('gmOps.logs.note') }}</span>
          <textarea v-model="note" rows="2" maxlength="500" class="gml__input gml__textarea" />
        </label>
      </div>
    </FormDialog>

    <FormDialog
      v-model="rejectOpen"
      :title="t('gmOps.logs.rejectTitle')"
      :width="480"
      :loading="acting"
      :confirm-text="t('gmOps.logs.reject')"
      @confirm="reject"
    >
      <div v-if="acted" class="gml__dialog">
        <p class="gml__dialog-text">{{ t('gmOps.logs.rejectDesc', { who: actorName(acted), title: commandTitle(acted) }) }}</p>
        <label class="gml__field">
          <span class="gml__label">{{ t('gmOps.logs.rejectNote') }} <span class="gml__req">*</span></span>
          <textarea v-model="note" rows="3" maxlength="500" class="gml__input gml__textarea" />
          <span class="gml__hint">{{ t('gmOps.logs.rejectNoteHint') }}</span>
        </label>
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onActivated, onMounted, ref, watch } from 'vue'
import { useI18n } from 'vue-i18n'
import { useDebounceFn } from '@vueuse/core'
import dayjs from 'dayjs'
import { approveGmExecution, listGmExecutions, rejectGmExecution } from '@/api/gameGm'
import type { GmExecution, ListGmExecutionsRequest } from '@/types/v1/game_gm'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import { feedback } from '@/utils/feedback'
import { useGmContext } from '../../context'
import GmRiskPill from '../../components/GmRiskPill.vue'
import SideDrawer from '../codes/SideDrawer.vue'
import { downloadCsv, fetchAllPages, fmtDateTime, fmtShort } from '../mail/shared'
import LogDetail from './LogDetail.vue'
import { useExecText } from './useExecText'

defineOptions({ name: 'GmLogsTab' })

type Filter = 'all' | 'admin' | 'player' | 'pending' | 'failed'
type Range = 'today' | '7d' | '30d' | 'custom'

const { t } = useI18n()
const ctx = useGmContext()
const menuStore = useMenuStore()
const { statusPill, statusText, kindText, channelText, sourceText, objectText, actorName, commandTitle } = useExecText()
const today = computed(() => t('gmOps.common.today'))

const filter = ref<Filter>('all')
const range = ref<Range>('7d')
const customFrom = ref('')
const customTo = ref('')
const risk = ref('')
const keyword = ref('')
const page = ref(1)
const pageSize = ref(20)
const items = ref<GmExecution[]>([])
const pending = ref<GmExecution[]>([])
const total = ref(0)
const counts = ref<Record<string, number>>({})
const loading = ref(false)

const ranges: Range[] = ['today', '7d', '30d']
const chips = computed(() =>
  (['all', 'admin', 'player', 'pending', 'failed'] as Filter[]).map((f) => ({
    value: f,
    label: t(`gmOps.logs.filter.${f}`),
    count: counts.value[f] ?? 0,
  })),
)
const riskOptions = computed(() => [
  { value: '', label: t('gmOps.logs.riskAll') },
  ...['low', 'medium', 'high', 'danger'].map((r) => ({ value: r, label: t('gmOps.logs.riskIs', { risk: t(`gm.risk.${r}`) }) })),
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'time', title: t('gmOps.logs.col.time'), width: 110 },
  { key: 'actor', title: t('gmOps.logs.col.actor'), width: 170 },
  { key: 'command', title: t('gmOps.logs.col.command'), minWidth: 260 },
  { key: 'risk', title: t('gmOps.logs.col.risk'), width: 90 },
  { key: 'status', title: t('gmOps.logs.col.status'), width: 110 },
])
const rows = computed(() => items.value as unknown as Record<string, unknown>[])
const asExec = (row: Record<string, unknown>) => row as unknown as GmExecution

// ---- 时间范围（本地时间的自然日）----
const span = computed<{ since?: Date; until?: Date }>(() => {
  const start = dayjs().startOf('day')
  switch (range.value) {
    case 'today':
      return { since: start.toDate() }
    case '30d':
      return { since: start.subtract(29, 'day').toDate() }
    case 'custom': {
      const from = customFrom.value ? dayjs(customFrom.value).startOf('day') : undefined
      const to = customTo.value ? dayjs(customTo.value).startOf('day').add(1, 'day') : undefined
      return { since: from?.toDate(), until: to?.toDate() }
    }
    default:
      return { since: start.subtract(6, 'day').toDate() }
  }
})
const rangeLabel = computed(() => {
  if (range.value !== 'custom') return t(`gmOps.logs.range.${range.value}`)
  return `${customFrom.value || '…'} — ${customTo.value || t('gmOps.logs.range.now')}`
})
const pickRange = (r: Range, close: () => void) => {
  if (r === 'custom' && customFrom.value && customTo.value && customTo.value < customFrom.value) {
    const from = customTo.value
    customTo.value = customFrom.value
    customFrom.value = from
  }
  range.value = r
  close()
  void reload(1)
}

const pickFilter = (f: Filter) => {
  filter.value = f
  void reload(1)
}
const onSearch = useDebounceFn(() => void reload(1), 350)
watch(risk, () => void reload(1))

const query = (p: number, size: number): ListGmExecutionsRequest => ({
  serverId: ctx.serverId.value,
  actorKind: filter.value === 'admin' || filter.value === 'player' ? filter.value : '',
  status: filter.value === 'pending' || filter.value === 'failed' ? filter.value : '',
  risk: risk.value,
  command: '',
  pid: '',
  q: keyword.value.trim(),
  since: span.value.since,
  until: span.value.until,
  includeDryRun: false,
  page: p,
  pageSize: size,
  actorId: '',
})

let seq = 0
const reload = async (toPage?: number) => {
  if (toPage) page.value = toPage
  const mine = ++seq
  loading.value = true
  try {
    const { data } = await listGmExecutions(query(page.value, pageSize.value))
    if (mine !== seq) return
    items.value = data.items || []
    total.value = data.total || 0
    counts.value = data.counts || {}
    pending.value = data.pending || []
    if (pending.value.length < 20) ctx.pendingApprovals.value = pending.value.length
  } catch {
    // 错误已由请求层提示
  } finally {
    if (mine === seq) loading.value = false
  }
}

// ---- 待审批 ----
const targetName = (e: GmExecution) => e.targets?.[0]?.name || e.targets?.[0]?.pid || ''
const pendingMeta = (p: GmExecution) => {
  const parts = [fmtShort(p.createTime, today.value)]
  if (p.reason) parts.push(t('gmOps.logs.applyReason', { reason: p.reason }))
  if (p.targets?.length) parts.push(targetName(p))
  parts.push(p.priorUses ? t('gmOps.logs.priorUses', { n: p.priorUses }) : t('gmOps.logs.firstUse'))
  return parts.join(' · ')
}

const acted = ref<GmExecution>()
const approveOpen = ref(false)
const rejectOpen = ref(false)
const acting = ref(false)
const note = ref('')

const openApprove = (p: GmExecution) => {
  acted.value = p
  note.value = ''
  approveOpen.value = true
}
const openReject = (p: GmExecution) => {
  acted.value = p
  note.value = ''
  rejectOpen.value = true
}

const approve = async () => {
  const p = acted.value
  if (!p) return
  acting.value = true
  try {
    const { data } = await approveGmExecution({ serverId: ctx.serverId.value, id: p.id, note: note.value.trim() })
    approveOpen.value = false
    if (data.status === 'success') feedback.success(t('gmOps.logs.approved'))
    else feedback.warning(t('gmOps.logs.approvedFailed', { status: statusText(data.status) }))
    await reload()
  } catch {
    // 错误已由请求层提示
  } finally {
    acting.value = false
  }
}

const reject = async () => {
  const p = acted.value
  if (!p) return
  if (!note.value.trim()) {
    feedback.warning(t('gmOps.logs.rejectNoteRequired'))
    return
  }
  acting.value = true
  try {
    await rejectGmExecution({ serverId: ctx.serverId.value, id: p.id, note: note.value.trim() })
    rejectOpen.value = false
    feedback.success(t('gmOps.logs.rejected'))
    await reload()
  } catch {
    // 错误已由请求层提示
  } finally {
    acting.value = false
  }
}

// ---- 详情 ----
const detailOpen = ref(false)
const detailId = ref('')
const detailPreview = ref<GmExecution>()
const openDetail = (e: GmExecution) => {
  detailPreview.value = e
  detailId.value = e.id
  detailOpen.value = true
}

// ---- 导出（当前筛选，分页拉取；不含请求参数）----
const exporting = ref(false)
const progress = ref('')
const exportCsv = async () => {
  exporting.value = true
  progress.value = ''
  try {
    const list = await fetchAllPages(
      async (p) => {
        const { data } = await listGmExecutions(query(p, 500))
        return { items: data.items || [], total: data.total || 0 }
      },
      (done, all) => (progress.value = `${done} / ${all}`),
    )
    downloadCsv(
      `gm-records-${ctx.server.value?.name || 'server'}-${dayjs().format('YYYYMMDDHHmm')}`,
      [
        t('gmOps.logs.csv.time'),
        t('gmOps.logs.csv.actor'),
        t('gmOps.logs.csv.kind'),
        t('gmOps.logs.csv.channel'),
        t('gmOps.logs.csv.title'),
        t('gmOps.logs.csv.object'),
        t('gmOps.logs.csv.source'),
        t('gmOps.logs.csv.risk'),
        t('gmOps.logs.csv.status'),
        t('gmOps.logs.csv.success'),
        t('gmOps.logs.csv.failed'),
        t('gmOps.logs.csv.reason'),
        t('gmOps.logs.csv.reviewer'),
        t('gmOps.logs.csv.reviewNote'),
        t('gmOps.logs.csv.error'),
      ],
      list.map((e) => [
        fmtDateTime(e.createTime, true),
        actorName(e),
        kindText(e.actorKind),
        [channelText(e.channel), e.platform, e.clientIp].filter(Boolean).join(' · '),
        commandTitle(e),
        objectText(e),
        sourceText(e),
        e.risk ? t(`gm.risk.${e.risk}`) : '',
        statusText(e.status),
        e.successCount,
        e.failedCount,
        e.reason,
        e.reviewerName,
        e.reviewNote,
        e.errorMessage || e.errorCode,
      ]),
    )
  } catch {
    // 错误已由请求层提示
  } finally {
    exporting.value = false
    progress.value = ''
  }
}

let activatedOnce = false
onMounted(() => void reload(1))
onActivated(() => {
  if (activatedOnce) void reload()
  activatedOnce = true
})
</script>

<style scoped>
.gml {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.gml__head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 12px 16px;
}
.gml__head-text {
  flex: 1 1 320px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.gml__h {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 30px;
  font-weight: 800;
  letter-spacing: -0.01em;
}
.gml__sub {
  margin: 0;
  font-size: 14px;
  color: var(--el-text-color-secondary);
}
.gml__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.gml__pending {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 14px 18px;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--m-gLine, var(--el-border-color-light));
  background: var(--m-gSoft, var(--m-pSoft));
}
.gml__pending-count {
  font-size: 15px;
  font-weight: 800;
}
.gml__pending-count .m-num {
  color: var(--m-gInk, var(--m-pInk));
}
.gml__pending-row {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px 14px;
  padding: 8px 0;
}
.gml__pending-row + .gml__pending-row {
  border-top: 1px solid var(--m-gLine, var(--el-border-color-light));
}
.gml__pending-text {
  flex: 1 1 320px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gml__pending-title {
  font-size: 14px;
  font-weight: 800;
  overflow-wrap: anywhere;
}
.gml__avatar {
  flex: none;
  width: 40px;
  height: 40px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--m-g, var(--m-p));
  color: var(--m-gOn, var(--m-pOn));
  font-weight: 800;
}
.gml__avatar--sm {
  width: 30px;
  height: 30px;
  font-size: 12px;
  background: var(--m-a2Soft);
  color: var(--el-text-color-primary);
}
.gml__avatar--sm.is-player {
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
}
.gml__filters {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.gml__chips {
  display: flex;
  gap: 8px;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.gml__chips::-webkit-scrollbar {
  display: none;
}
.gml__spacer {
  flex: 1;
}
.gml__chip {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
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
.gml__chip:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.gml__chip.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.gml__chip-n {
  font-size: 12px;
  opacity: 0.75;
}
.gml__range {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px;
}
.gml__range-opt {
  min-height: 40px;
  padding: 0 12px;
  border: 0;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-weight: 600;
  text-align: left;
  cursor: pointer;
}
.gml__range-opt:hover {
  background: var(--el-fill-color-light);
}
.gml__range-opt.is-on {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.gml__range-custom {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: 8px 4px 4px;
  border-top: 1px solid var(--el-border-color-light);
}
.gml__range-dates {
  display: flex;
  align-items: center;
  gap: 6px;
}
.gml__risk {
  width: 150px;
}
.gml__search {
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
  color: var(--el-text-color-secondary);
}
.gml__search:focus-within {
  border-color: var(--m-p);
}
.gml__search input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 14px;
}
.gml__card {
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  overflow: hidden;
}
.gml__time {
  white-space: nowrap;
}
.gml__actor {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.gml__actor-text {
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.gml__actor-name {
  font-size: 13px;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gml__cmd {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}
.gml__cmd-title {
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gml__cmd-sub {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gml__mcard {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  min-height: 44px;
  padding: 14px 16px;
  border: 0;
  border-top: 1px solid var(--el-border-color-light);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gml__mcard:first-child {
  border-top: 0;
}
.gml__mcard-top,
.gml__mcard-meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}
.gml__pager {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  padding: 12px 16px;
  border-top: 1px solid var(--el-border-color-light);
}
.gml__pager > .gml__hint {
  flex: 1;
}
.gml__dialog {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.gml__dialog-text {
  margin: 0;
  font-size: 14px;
  line-height: 1.6;
}
.gml__field {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gml__label {
  font-size: 13px;
  font-weight: 700;
}
.gml__req {
  color: var(--m-bad);
}
.gml__input {
  width: 100%;
  min-height: 40px;
  padding: 0 12px;
  border-radius: var(--app-radius-sm);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 14px;
  outline: none;
  min-width: 0;
}
.gml__input:focus {
  border-color: var(--m-p);
}
.gml__textarea {
  padding: 10px 12px;
  resize: vertical;
  line-height: 1.6;
}
@media (width < 768px) {
  .gml__h {
    font-size: 24px;
  }
  .gml__spacer {
    display: none;
  }
  .gml__chips {
    width: 100%;
  }
  .gml__chip {
    min-height: 44px;
  }
  .gml__risk {
    flex: 1 1 140px;
    width: auto;
  }
  .gml__search {
    flex: 1 1 100%;
    min-height: 44px;
  }
  .gml__pending :deep(button) {
    min-height: 44px;
  }
}
</style>
