<!-- GM 运营 · 兑换码：统计卡片、批次表（状态 chips、搜索）、批次详情与最近兑换、导出、新建 / 编辑对话框、停用。
     兑换码在 Momoko 校验，兑换成功以玩家身份执行 gift.send（sourceKind=code）把奖励放进礼物箱。 -->
<template>
  <div class="gmc game-skin">
    <div class="gmc__head">
      <div class="gmc__head-text">
        <h2 class="gmc__h">{{ t('gmOps.codes.title') }}</h2>
        <p class="gmc__sub">{{ t('gmOps.codes.subline') }}</p>
      </div>
      <UButton color="neutral" variant="outline" icon="i-lucide-download" :loading="exporting === 'all'" :disabled="!!exporting" @click="exportRedeems()">
        {{ exporting === 'all' && progress ? progress : t('gmOps.codes.exportAll') }}
      </UButton>
      <UButton color="primary" icon="i-lucide-plus" @click="openCreate">{{ t('gmOps.codes.newBatch') }}</UButton>
    </div>

    <MetricStrip :columns="4">
      <MetricItem :label="t('gmOps.codes.stats.active')" :value="stats?.activeBatches ?? '—'" :caption="t('gmOps.codes.stats.activeCaption', { n: stats?.endedBatches ?? 0 })" />
      <MetricItem :label="t('gmOps.codes.stats.today')" :value="stats?.todayRedeems ?? '—'" :caption="todayCaption" />
      <MetricItem :label="t('gmOps.codes.stats.total')" :value="stats?.totalRedeems ?? '—'" :caption="totalCaption" />
      <MetricItem :label="t('gmOps.codes.stats.failures')" :value="stats?.todayFailures ?? '—'" :caption="failureCaption" />
    </MetricStrip>

    <div class="gmc__body">
      <section class="gmc__card gmc__list" :aria-label="t('gmOps.codes.listLabel')">
        <div class="gmc__toolbar">
          <ListToolbar
            v-model:status="status"
            v-model:keyword="keyword"
            :chips="chips"
            :chips-label="t('gmOps.codes.statusLabel')"
            :placeholder="t('gmOps.codes.search')"
            @search="reload(1)"
          />
        </div>

        <template v-if="!menuStore.isMobile">
          <DataTable
            :columns="columns"
            :rows="rows"
            row-key="id"
            :model-value="selectedId ? [selectedId] : []"
            :loading="loading"
            :empty-text="t('gmOps.codes.empty')"
            @row-click="(row) => select(asBatch(row))"
          >
            <template #cell-batch="{ row }">
              <span class="gmc__batch">
                <span class="gmc__codebox m-mono">{{ codeOf(asBatch(row)) }}</span>
                <span class="gmc__hint">{{ asBatch(row).name }} · {{ kindText(asBatch(row)) }}</span>
              </span>
            </template>
            <template #cell-rewards="{ row }">
              <span class="gmc__rewards">
                <template v-for="(r, i) in asBatch(row).rewards.slice(0, 3)" :key="i">
                  <GameImage v-if="label(r).image" :src="label(r).image" class="gmc__reward-img" width="26" height="26" />
                </template>
                <span class="gmc__reward-sum">{{ summary(asBatch(row).rewards) }}</span>
              </span>
            </template>
            <template #cell-used="{ row }">
              <span class="gmc__used">
                <span class="m-num gmc__used-n">{{ usedText(asBatch(row)) }}</span>
                <span v-if="asBatch(row).totalLimit" class="gmc__bar"><span :style="{ width: `${usedPct(asBatch(row))}%` }" /></span>
              </span>
            </template>
            <template #cell-until="{ row }">
              <span class="m-num gmc__hint">{{ untilText(asBatch(row)) }}</span>
            </template>
            <template #cell-status="{ row }">
              <StatusPill :variant="pillOf(asBatch(row).status)" :label="t(`gmOps.codes.status.${asBatch(row).status}`)" />
            </template>
          </DataTable>
        </template>
        <template v-else>
          <EmptyState v-if="!items.length && !loading" icon="HOutline:TicketIcon" :title="t('gmOps.codes.empty')" />
          <button
            v-for="b in items"
            :key="b.id"
            type="button"
            class="gmc__mcard"
            :class="{ 'is-on': selectedId === b.id }"
            @click="select(b)"
          >
            <span class="gmc__mcard-top">
              <span class="gmc__codebox m-mono">{{ codeOf(b) }}</span>
              <StatusPill :variant="pillOf(b.status)" :label="t(`gmOps.codes.status.${b.status}`)" />
            </span>
            <span class="gmc__mcard-name">{{ b.name }} · {{ kindText(b) }}</span>
            <dl class="gmc__mkv">
              <dt>{{ t('gmOps.codes.col.rewards') }}</dt>
              <dd>{{ summary(b.rewards) }}</dd>
              <dt>{{ t('gmOps.codes.col.used') }}</dt>
              <dd class="m-num">{{ usedText(b) }}</dd>
              <dt>{{ t('gmOps.codes.col.until') }}</dt>
              <dd class="m-num">{{ untilText(b) }}</dd>
            </dl>
          </button>
        </template>

        <div v-if="total > pageSize" class="gmc__pager">
          <Pagination v-model:page="page" v-model:page-size="pageSize" :total="total" :page-sizes="[20, 50, 100]" @change="reload()" />
        </div>
      </section>

      <section ref="detailRef" class="gmc__card gmc__detail">
        <EmptyState v-if="!detail" icon="HOutline:TicketIcon" :title="t('gmOps.codes.pick')" :description="t('gmOps.codes.pickDesc')" />
        <template v-else>
          <div class="gmc__dhead">
            <span class="gmc__dhead-text">
              <span class="gmc__codebox gmc__codebox--lg m-mono">{{ codeOf(detail.batch) }}</span>
              <span class="gmc__dname">{{ detail.batch.name }}</span>
            </span>
            <AppIconButton v-if="detail.batch.kind === 'universal'" icon="HOutline:ClipboardDocumentIcon" :label="t('gmOps.codes.copy')" @click="copy(detail.batch.code)" />
            <AppIconButton icon="HOutline:PencilSquareIcon" :label="t('gmOps.codes.edit')" @click="openEdit" />
          </div>
          <dl class="gmc__kv">
            <dt>{{ t('gmOps.codes.kv.status') }}</dt>
            <dd><StatusPill :variant="pillOf(detail.batch.status)" :label="t(`gmOps.codes.status.${detail.batch.status}`)" /></dd>
            <dt>{{ t('gmOps.codes.kv.kind') }}</dt>
            <dd>{{ kindText(detail.batch) }}</dd>
            <dt>{{ t('gmOps.codes.kv.rewards') }}</dt>
            <dd>{{ summary(detail.batch.rewards) }}</dd>
            <dt>{{ t('gmOps.codes.kv.message') }}</dt>
            <dd>{{ detail.batch.message || t('gmOps.codes.defaultMessage') }}</dd>
            <dt>{{ t('gmOps.codes.kv.limit') }}</dt>
            <dd>{{ limitText(detail.batch) }}</dd>
            <dt>{{ t('gmOps.codes.kv.used') }}</dt>
            <dd class="m-num">{{ t('gmOps.codes.usedToday', { used: usedText(detail.batch), today: detail.batch.todayCount }) }}</dd>
            <dt>{{ t('gmOps.codes.kv.who') }}</dt>
            <dd>{{ whoText(detail.batch) }}</dd>
            <dt>{{ t('gmOps.codes.kv.where') }}</dt>
            <dd>{{ whereText(detail.batch) }}</dd>
            <dt>{{ t('gmOps.codes.kv.span') }}</dt>
            <dd class="m-num">{{ spanText(detail.batch) }}</dd>
            <dt>{{ t('gmOps.codes.kv.by') }}</dt>
            <dd>{{ detail.batch.createdByName || '—' }} · {{ fmtDateTime(detail.batch.createTime) }}</dd>
            <template v-if="detail.batch.disabledAt">
              <dt>{{ t('gmOps.codes.kv.disabled') }}</dt>
              <dd>{{ detail.batch.disabledByName || '—' }} · {{ fmtDateTime(detail.batch.disabledAt) }}</dd>
            </template>
          </dl>

          <div class="gmc__recent">
            <div class="gmc__recent-head">
              <span class="gmc__recent-title">{{ t('gmOps.codes.recent') }}</span>
              <span class="gmc__hint">{{ t('gmOps.codes.recentCounts', { ok: detail.successCount, bad: detail.failureCount }) }}</span>
            </div>
            <div v-if="!detail.recent.length" class="gmc__hint gmc__recent-empty">{{ t('gmOps.codes.noRedeems') }}</div>
            <div v-for="r in detail.recent" :key="r.id" class="gmc__redeem">
              <span class="gmc__ravatar" aria-hidden="true">{{ (r.nickname || '?').slice(0, 1).toUpperCase() }}</span>
              <span class="gmc__rtext">
                <span class="gmc__rname">{{ r.nickname || t('gm.player.noName') }}</span>
                <span class="gmc__hint">{{ t(`gmOps.codes.channel.${r.channel}Short`) }} · {{ fmtShort(r.createTime, today) }}<template v-if="detail.batch.kind === 'unique'"> · <span class="m-mono">{{ r.code }}</span></template></span>
              </span>
              <StatusPill :variant="r.result === 'success' ? 'success' : r.result === 'already_redeemed' ? 'warning' : 'error'" :label="t(`gmOps.codes.result.${r.result}`)" />
            </div>
          </div>

          <div class="gmc__dfoot">
            <UButton color="neutral" variant="outline" size="sm" icon="i-lucide-download" :loading="exporting === 'batch'" :disabled="!!exporting" @click="exportRedeems(detail.batch)">
              {{ t('gmOps.codes.exportRedeems') }}
            </UButton>
            <UButton v-if="detail.batch.kind === 'unique'" color="neutral" variant="outline" size="sm" icon="i-lucide-file-down" :loading="exporting === 'codes'" :disabled="!!exporting" @click="exportCodes(detail.batch)">
              {{ t('gmOps.codes.exportCodes') }}
            </UButton>
            <span class="gmc__spacer" />
            <UButton v-if="!detail.batch.disabledAt" color="error" variant="soft" size="sm" icon="i-lucide-square" @click="disable(detail.batch)">
              {{ t('gmOps.codes.disable') }}
            </UButton>
          </div>
        </template>
      </section>
    </div>

    <BatchDialog v-model="dialogOpen" :batch="editing" @saved="onSaved" />
  </div>
</template>

<script setup lang="ts">
import { computed, nextTick, onActivated, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  disableGiftCodeBatch,
  getGiftCodeBatch,
  listGiftCodeBatches,
  listGiftCodeRedeems,
  listGiftCodes,
} from '@/api/gameOps'
import type { GetGiftCodeBatchResponse, GiftCodeBatch, GiftCodeStats } from '@/types/v1/game_ops'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ToolbarChip } from '@/components/ui/ListToolbar.vue'
import { Dialog } from '@/utils/dialog'
import { feedback } from '@/utils/feedback'
import { useGmContext } from '../../context'
import { downloadCsv, fetchAllPages, fmtDateTime, fmtShort, useRewardLabels } from '../mail/shared'
import BatchDialog from './BatchDialog.vue'

defineOptions({ name: 'GmCodesTab' })

type Pill = 'success' | 'warning' | 'error' | 'info' | 'neutral' | 'primary'

const { t } = useI18n()
const ctx = useGmContext()
const menuStore = useMenuStore()
const { label, summary } = useRewardLabels()
const today = computed(() => t('gmOps.common.today'))

const status = ref('')
const keyword = ref('')
const page = ref(1)
const pageSize = ref(20)
const items = ref<GiftCodeBatch[]>([])
const total = ref(0)
const counts = ref<Record<string, number>>({})
const stats = ref<GiftCodeStats>()
const loading = ref(false)
const selectedId = ref('')
const detail = ref<GetGiftCodeBatchResponse & { batch: GiftCodeBatch }>()
const detailRef = ref<HTMLElement>()

const chips = computed<ToolbarChip<string>[]>(() => [
  { label: t('gmOps.codes.chips.all'), value: '', count: counts.value.all ?? null },
  { label: t('gmOps.codes.chips.active'), value: 'active', count: counts.value.active ?? null },
  { label: t('gmOps.codes.chips.ended'), value: 'ended', count: counts.value.ended ?? null },
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'batch', title: t('gmOps.codes.col.batch'), minWidth: 200 },
  { key: 'rewards', title: t('gmOps.codes.col.rewards'), minWidth: 180 },
  { key: 'used', title: t('gmOps.codes.col.used'), width: 150 },
  { key: 'until', title: t('gmOps.codes.col.until'), width: 130 },
  { key: 'status', title: t('gmOps.codes.col.status'), width: 110 },
])
const rows = computed(() => items.value as unknown as Record<string, unknown>[])
const asBatch = (row: Record<string, unknown>) => row as unknown as GiftCodeBatch

// ---- 统计卡片 ----
const todayCaption = computed(() => {
  const s = stats.value
  if (!s || !s.todayTopBatch || !s.todayTopCount) return t('gmOps.codes.stats.todayNone')
  return t('gmOps.codes.stats.todayTop', { name: s.todayTopBatch, n: s.todayTopCount })
})
const totalCaption = computed(() => {
  const s = stats.value
  return s?.firstRedeemAt ? t('gmOps.codes.stats.since', { date: fmtDateTime(s.firstRedeemAt).slice(0, 10) }) : t('gmOps.codes.stats.noneYet')
})
const failureCaption = computed(() => {
  const reasons = stats.value?.todayFailureReasons || {}
  const parts = Object.entries(reasons)
    .sort((a, b) => b[1] - a[1])
    .map(([k, n]) => `${t(`gmOps.codes.result.${k}`)} ${n}`)
  return parts.length ? parts.join(' · ') : t('gmOps.codes.stats.noFailures')
})

// ---- 文本 ----
const codeOf = (b: GiftCodeBatch) => (b.kind === 'unique' ? `${b.prefix ? `${b.prefix}-` : ''}····-····` : b.code)
const kindText = (b: GiftCodeBatch) => (b.kind === 'unique' ? t('gmOps.codes.kindUnique', { n: b.quantity }) : t('gmOps.codes.kinds.universal'))
const usedText = (b: GiftCodeBatch) => (b.totalLimit ? `${b.usedCount} / ${b.totalLimit}` : t('gmOps.codes.times', { n: b.usedCount }))
const usedPct = (b: GiftCodeBatch) => (b.totalLimit ? Math.min(100, Math.round((b.usedCount / b.totalLimit) * 100)) : 0)
const untilText = (b: GiftCodeBatch) => {
  if (b.disabledAt) return t('gmOps.codes.disabledAt', { time: fmtShort(b.disabledAt, today.value) })
  return b.endAt ? fmtDateTime(b.endAt) : t('gmOps.codes.forever')
}
const limitText = (b: GiftCodeBatch) => {
  const per = t('gmOps.codes.perPlayerN', { n: b.perPlayerLimit })
  if (b.kind === 'unique') return `${t('gmOps.codes.uniqueLimit', { n: b.quantity })} · ${per}`
  return `${b.totalLimit ? t('gmOps.codes.totalN', { n: b.totalLimit }) : t('gmOps.codes.noTotal')} · ${per}`
}
const whoText = (b: GiftCodeBatch) =>
  b.eligibility === 'condition' ? t('gmOps.codes.whoCondition', { n: b.registeredWithinDays }) : t('gmOps.codes.eligible.all')
const whereText = (b: GiftCodeBatch) =>
  (b.channels || []).map((c) => t(c === 'game' ? 'gmOps.codes.channel.gameUnsupportedShort' : `gmOps.codes.channel.${c}Short`)).join(' · ') || '—'
const spanText = (b: GiftCodeBatch) => {
  const start = fmtDateTime(b.startAt || b.createTime)
  return b.endAt ? `${start} → ${fmtDateTime(b.endAt)}` : t('gmOps.codes.fromForever', { start })
}
const pillOf = (s: string): Pill => {
  switch (s) {
    case 'active':
      return 'success'
    case 'not_started':
      return 'info'
    case 'exhausted':
      return 'warning'
    case 'disabled':
      return 'error'
    default:
      return 'neutral'
  }
}

// ---- 数据 ----
let seq = 0
const reload = async (toPage?: number) => {
  if (toPage) page.value = toPage
  const mine = ++seq
  loading.value = true
  try {
    const { data } = await listGiftCodeBatches({
      serverId: ctx.serverId.value,
      status: status.value,
      q: keyword.value.trim(),
      page: page.value,
      pageSize: pageSize.value,
    })
    if (mine !== seq) return
    items.value = data.items || []
    total.value = data.total || 0
    counts.value = data.counts || {}
    stats.value = data.stats
    if (!selectedId.value && items.value[0] && !menuStore.isMobile) void select(items.value[0], false)
  } catch {
    // 错误已由请求层提示
  } finally {
    if (mine === seq) loading.value = false
  }
}

const loadDetail = async (id: string) => {
  try {
    const { data } = await getGiftCodeBatch(ctx.serverId.value, id)
    if (selectedId.value !== id || !data.batch) return
    detail.value = { ...data, batch: data.batch, recent: data.recent || [] }
  } catch {
    // 错误已由请求层提示
  }
}

const select = async (b: GiftCodeBatch, scroll = true) => {
  selectedId.value = b.id
  if (detail.value?.batch.id !== b.id) detail.value = { batch: b, recent: [], successCount: 0, failureCount: 0 }
  if (scroll && menuStore.isMobile) {
    await nextTick()
    detailRef.value?.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
  await loadDetail(b.id)
}

// ---- 新建 / 编辑 / 停用 ----
const dialogOpen = ref(false)
const editing = ref<GiftCodeBatch>()
const openCreate = () => {
  editing.value = undefined
  dialogOpen.value = true
}
const openEdit = () => {
  editing.value = detail.value?.batch
  dialogOpen.value = true
}
const onSaved = async (b: GiftCodeBatch) => {
  await reload()
  void select(b, false)
}

const disable = (b: GiftCodeBatch) => {
  Dialog.confirm({
    title: t('gmOps.codes.disable'),
    content: t('gmOps.codes.disableDesc', { name: b.name }),
    cancelText: t('gmOps.common.cancel'),
    onConfirm: async () => {
      try {
        await disableGiftCodeBatch(ctx.serverId.value, b.id)
        feedback.success(t('gmOps.codes.disabled'))
        await reload()
        await loadDetail(b.id)
      } catch {
        // 错误已由请求层提示
      }
    },
  })
}

const copy = async (text: string) => {
  try {
    await navigator.clipboard.writeText(text)
    feedback.success(t('gmOps.codes.copied'))
  } catch {
    feedback.error(t('gmOps.codes.copyFailed'))
  }
}

// ---- 导出（分页拉取）----
const exporting = ref<'' | 'all' | 'batch' | 'codes'>('')
const progress = ref('')
const stamp = () => fmtDateTime(new Date()).replace(/[-: ]/g, '')

const exportRedeems = async (b?: GiftCodeBatch) => {
  exporting.value = b ? 'batch' : 'all'
  progress.value = ''
  try {
    const list = await fetchAllPages(
      async (p) => {
        const { data } = await listGiftCodeRedeems({ serverId: ctx.serverId.value, batchId: b?.id || '', result: '', page: p, pageSize: 500 })
        return { items: data.items || [], total: data.total || 0 }
      },
      (done, all) => (progress.value = `${done} / ${all}`),
    )
    downloadCsv(
      `redeems-${b ? b.name : ctx.server.value?.name || 'server'}-${stamp()}`,
      [
        t('gmOps.codes.csv.time'),
        t('gmOps.codes.csv.batch'),
        t('gmOps.codes.csv.code'),
        t('gmOps.codes.csv.nickname'),
        t('gmOps.codes.csv.channel'),
        t('gmOps.codes.csv.platform'),
        t('gmOps.codes.csv.result'),
        t('gmOps.codes.csv.detail'),
      ],
      list.map((r) => [
        fmtDateTime(r.createTime, true),
        r.batchName,
        r.code,
        r.nickname,
        t(`gmOps.codes.channel.${r.channel}Short`),
        r.platform,
        t(`gmOps.codes.result.${r.result}`),
        r.errorMessage,
      ]),
    )
  } catch {
    // 错误已由请求层提示
  } finally {
    exporting.value = ''
    progress.value = ''
  }
}

const exportCodes = async (b: GiftCodeBatch) => {
  exporting.value = 'codes'
  try {
    const list = await fetchAllPages(async (p) => {
      const { data } = await listGiftCodes({ serverId: ctx.serverId.value, batchId: b.id, state: '', page: p, pageSize: 1000 })
      return { items: data.items || [], total: data.total || 0 }
    })
    downloadCsv(
      `codes-${b.name}-${stamp()}`,
      [t('gmOps.codes.csv.code'), t('gmOps.codes.csv.state'), t('gmOps.codes.csv.nickname'), t('gmOps.codes.csv.usedAt')],
      list.map((c) => [
        c.code,
        c.usedByPlayer ? t('gmOps.codes.csv.used') : t('gmOps.codes.csv.unused'),
        c.usedByName,
        fmtDateTime(c.usedAt, true),
      ]),
    )
  } catch {
    // 错误已由请求层提示
  } finally {
    exporting.value = ''
  }
}

let activatedOnce = false
onMounted(() => void reload(1))
onActivated(() => {
  if (activatedOnce) {
    void reload()
    if (selectedId.value) void loadDetail(selectedId.value)
  }
  activatedOnce = true
})
</script>

<style scoped>
.gmc {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.gmc__head {
  display: flex;
  flex-wrap: wrap;
  align-items: flex-end;
  gap: 12px;
}
.gmc__head-text {
  flex: 1 1 320px;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.gmc__h {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 30px;
  font-weight: 800;
  letter-spacing: -0.01em;
}
.gmc__sub {
  margin: 0;
  font-size: 14px;
  color: var(--el-text-color-secondary);
}
.gmc__body {
  display: flex;
  flex-wrap: wrap;
  gap: 16px;
  align-items: flex-start;
}
.gmc__card {
  min-width: 0;
  border-radius: var(--app-radius-xl);
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.gmc__list {
  flex: 3 1 620px;
  overflow: hidden;
}
.gmc__detail {
  flex: 2 1 380px;
  padding: 20px 22px;
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.gmc__toolbar {
  padding: 14px 16px;
}
.gmc__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--el-text-color-secondary);
}
.gmc__batch {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 5px;
  min-width: 0;
}
.gmc__codebox {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  padding: 3px 10px;
  border-radius: var(--app-radius-sm);
  background: var(--m-gSoft, var(--el-fill-color-light));
  color: var(--m-gInk, var(--el-text-color-primary));
  font-size: 13px;
  font-weight: 800;
  letter-spacing: 0.04em;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gmc__codebox--lg {
  align-self: flex-start;
  padding: 6px 12px;
  font-size: 16px;
}
.gmc__rewards {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.gmc__reward-img {
  flex: none;
  width: 26px;
  height: 26px;
  object-fit: contain;
}
.gmc__reward-sum {
  font-size: 13px;
  font-weight: 600;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gmc__used {
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gmc__used-n {
  font-size: 13px;
  font-weight: 700;
}
.gmc__bar {
  display: block;
  height: 6px;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  overflow: hidden;
}
.gmc__bar span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--m-gInk, var(--m-pInk));
}
.gmc__pager {
  padding: 12px 16px;
  border-top: 1px solid var(--el-border-color-light);
}
.gmc__mcard {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
  padding: 14px 16px;
  border: 0;
  border-top: 1px solid var(--el-border-color-light);
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  text-align: left;
  cursor: pointer;
}
.gmc__mcard.is-on {
  background: var(--m-gSoft, var(--m-pSoft));
  box-shadow: inset 3px 0 0 var(--m-g, var(--m-p));
}
.gmc__mcard-top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}
.gmc__mcard-name {
  font-size: 13px;
  font-weight: 700;
}
.gmc__mkv,
.gmc__kv {
  display: grid;
  grid-template-columns: 84px minmax(0, 1fr);
  gap: 8px 12px;
  margin: 0;
  font-size: 13px;
}
.gmc__mkv {
  padding: 10px 12px;
  border-radius: var(--app-radius);
  background: var(--el-fill-color-light);
}
.gmc__mkv dt,
.gmc__kv dt {
  color: var(--el-text-color-secondary);
}
.gmc__mkv dd,
.gmc__kv dd {
  margin: 0;
  font-weight: 600;
  min-width: 0;
  overflow-wrap: anywhere;
}
.gmc__dhead {
  display: flex;
  align-items: flex-start;
  gap: 8px;
}
.gmc__dhead-text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.gmc__dname {
  font-size: 16px;
  font-weight: 800;
}
.gmc__recent {
  display: flex;
  flex-direction: column;
}
.gmc__recent-head {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 4px;
}
.gmc__recent-title {
  flex: 1;
  font-size: 15px;
  font-weight: 800;
}
.gmc__recent-empty {
  padding: 12px 0;
}
.gmc__redeem {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 9px 0;
  border-top: 1px solid var(--el-border-color-light);
}
.gmc__ravatar {
  flex: none;
  width: 32px;
  height: 32px;
  border-radius: 50%;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  font-size: 13px;
  font-weight: 800;
}
.gmc__rtext {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.gmc__rname {
  font-size: 13px;
  font-weight: 700;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.gmc__dfoot {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: center;
}
.gmc__spacer {
  flex: 1;
}
@media (width < 768px) {
  .gmc__h {
    font-size: 24px;
  }
  .gmc__detail {
    padding: 16px;
  }
  .gmc__dfoot :deep(button) {
    min-height: 44px;
  }
}
</style>
