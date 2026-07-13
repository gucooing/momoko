<!-- 内网穿透（重写 · P1 列表）：PageHeader(新增/frps) + FilterBar + 批量删除 + DataTable/移动卡 + Pagination。
     行内 统计/frpc/编辑/删除；启用 AppSwitch；统计/frpc/frps 弹窗。 -->
<template>
  <div class="tn-page">
    <PageHeader :title="t('tools.tunnel.title')" :description="t('tools.tunnel.pageDesc')">
      <template #actions>
        <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-settings" @click="frpsVisible = true">
          {{ t('tools.tunnel.frpsConfig') }}
        </UButton>
        <UButton color="primary" size="sm" icon="i-lucide-plus" @click="openCreateDialog()">
          {{ t('tools.tunnel.addTunnel') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="search" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('tools.tunnel.keyword') }}</label>
          <input v-model="queryForm.keywords" class="app-input" :placeholder="t('tools.tunnel.keywordPlaceholder')" @keyup.enter="search" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('tools.tunnel.proxyType') }}</label>
          <AppSelect v-model="queryForm.type" :options="typeFilterOptions" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('tools.tunnel.enableStatus') }}</label>
          <AppSelect v-model="queryForm.isEnable" :options="enableFilterOptions" />
        </div>
      </template>
    </FilterBar>

    <div v-if="selectedIds.length" class="tn-page__batch">
      <span class="tn-page__batch-count">{{ t('tools.tunnel.selectedCount', { count: selectedIds.length }) }}</span>
      <div class="tn-page__batch-actions">
        <UButton color="error" variant="soft" size="sm" icon="i-lucide-trash-2" @click="batchDelete">
          {{ t('tools.tunnel.batchDelete') }}
        </UButton>
        <UButton color="neutral" variant="ghost" size="sm" @click="selectedIds = []">
          {{ t('tools.tunnel.clearSelection') }}
        </UButton>
      </div>
    </div>

    <div class="tn-page__body">
      <div class="tn-page__bar">
        <span class="tn-page__hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
        <button v-if="menuStore.isMobile && list.length" type="button" class="tn-page__selall" @click="toggleSelectAll">
          {{ isAllSelected ? t('tools.tunnel.clearSelection') : t('tools.tunnel.selectAll') }}
        </button>
      </div>

      <DataTable
        v-if="!menuStore.isMobile"
        v-model="selectedIds"
        :columns="columns"
        :rows="list"
        row-key="id"
        selectable
        :loading="loading"
        :empty-text="t('tools.tunnel.noData')"
      >
        <template #cell-type="{ row }">
          <StatusPill variant="info" :label="typeLabel(String(row.type || ''))" :dot="false" />
        </template>
        <template #cell-status="{ row }">
          <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(String(row.status || ''))" />
        </template>
        <template #cell-remote="{ row }">
          <span class="tn-mono">{{ remoteText(row) }}</span>
        </template>
        <template #cell-target="{ row }">
          <span class="tn-mono">{{ row.localIp }}:{{ row.localPort }}</span>
        </template>
        <template #cell-isEnable="{ row }">
          <AppSwitch :model-value="!!row.isEnable" @update:model-value="(v) => toggleEnable(row, v)" />
        </template>
        <template #cell-connections="{ row }">{{ Number(row.activeConnections || 0) }}</template>
        <template #cell-traffic="{ row }">
          <span class="tn-traffic">↓ {{ formatBytes(row.bytesIn) }} · ↑ {{ formatBytes(row.bytesOut) }}</span>
        </template>
        <template #cell-createTime="{ row }">{{ formatTime(row.createTime) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="rowActions" @select="(key) => onRowAction(key, row)" />
        </template>
      </DataTable>

      <template v-else>
        <div v-if="loading" class="tn-cards">
          <div v-for="i in 4" :key="i" class="tn-skeleton" />
        </div>
        <EmptyState
          v-else-if="!list.length"
          icon="HOutline:GlobeAltIcon"
          :title="t('tools.tunnel.noData')"
          :description="t('tools.tunnel.emptyDesc')"
        />
        <div v-else class="tn-cards">
          <div
            v-for="row in list"
            :key="row.id"
            class="tn-card"
            :class="{ 'is-selected': selectedIds.includes(row.id) }"
            @click="toggleMobileSelect(row.id)"
          >
            <div class="tn-card__top">
              <label class="tn-card__title" @click.stop>
                <input
                  type="checkbox"
                  class="tn-card__check"
                  :checked="selectedIds.includes(row.id)"
                  @change="toggleMobileSelect(row.id)"
                />
                <span class="tn-card__name">{{ row.name }}</span>
              </label>
              <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
            </div>
            <div class="tn-card__route tn-mono">
              <StatusPill variant="info" :label="typeLabel(row.type)" :dot="false" />
              <span>{{ remoteText(row as unknown as Record<string, unknown>) }} → {{ row.localIp }}:{{ row.localPort }}</span>
            </div>
            <div class="tn-card__foot" @click.stop>
              <span class="tn-card__meta">
                {{ Number(row.activeConnections || 0) }} 连接 · ↓{{ formatBytes(row.bytesIn) }} ↑{{ formatBytes(row.bytesOut) }}
              </span>
              <AppSwitch :model-value="row.isEnable" @update:model-value="(v) => toggleEnable(row as unknown as Record<string, unknown>, v)" />
              <ActionMenu :items="rowActions" @select="(key) => onRowAction(key, row as unknown as Record<string, unknown>)" />
            </div>
          </div>
        </div>
      </template>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="getList"
      />
    </div>

    <TunnelCreate ref="createRef" @refresh="refresh" />
    <TunnelStatsDialog v-model="statsVisible" :row="statsRow" />
    <FrpcConfigDialog v-model="frpcVisible" :row="frpcRow" :frps="frpsConfig" />
    <FrpsConfigDialog v-model="frpsVisible" @saved="onFrpsSaved" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import TunnelCreate from '@/views/tools/tunnel/create.vue'
import TunnelStatsDialog from '@/views/tools/tunnel/TunnelStatsDialog.vue'
import FrpcConfigDialog from '@/views/tools/tunnel/FrpcConfigDialog.vue'
import FrpsConfigDialog from '@/views/tools/tunnel/FrpsConfigDialog.vue'
import { listTunnels, deleteTunnel, updateTunnel, getFrpsConfig } from '@/api/tunnel'
import { TunnelType, TunnelStatus, type TunnelInfo, type FrpsConfig } from '@/types/v1/tunnel'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import { Dialog } from '@/utils/dialog'

defineOptions({ name: 'TunnelView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const createRef = useTemplateRef<InstanceType<typeof TunnelCreate> | null>('createRef')

const selectedIds = ref<string[]>([])
const list = ref<TunnelInfo[]>([])
const loading = ref(false)
const statsVisible = ref(false)
const statsRow = ref<TunnelInfo | null>(null)
const frpcVisible = ref(false)
const frpcRow = ref<TunnelInfo | null>(null)
const frpsVisible = ref(false)
const frpsConfig = ref<FrpsConfig | null>(null)

const typeOptions = [
  TunnelType.TUNNEL_TYPE_TCP,
  TunnelType.TUNNEL_TYPE_UDP,
  TunnelType.TUNNEL_TYPE_HTTP,
  TunnelType.TUNNEL_TYPE_HTTPS,
  TunnelType.TUNNEL_TYPE_STCP,
  TunnelType.TUNNEL_TYPE_XTCP,
  TunnelType.TUNNEL_TYPE_TCPMUX,
]
const typeLabel = (type: string) => String(type || '').replace('TUNNEL_TYPE_', '') || '-'
const typeFilterOptions = computed(() => [
  { label: t('tools.tunnel.allTypes'), value: '' },
  ...typeOptions.map((opt) => ({ label: typeLabel(opt), value: opt })),
])
const enableFilterOptions = computed(() => [
  { label: t('tools.tunnel.allStatus'), value: '' },
  { label: t('tools.tunnel.enabled'), value: 'true' },
  { label: t('tools.tunnel.disabled'), value: 'false' },
])

const queryForm = reactive({ keywords: '', type: '' as string, isEnable: '' as string })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const formatBytes = (bytes?: unknown): string => {
  const n = Number(bytes)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}
const formatTime = (value: unknown) => {
  const s = value ? String(value) : ''
  if (!s || s.startsWith('0001-')) return '—'
  const date = new Date(s)
  if (Number.isNaN(date.getTime())) return s
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}

type PillVariant = 'success' | 'info' | 'warning' | 'error' | 'neutral'
const statusLabel = (status: string) => {
  if (status === TunnelStatus.TUNNEL_STATUS_ONLINE) return t('tools.tunnel.status.online')
  if (status === TunnelStatus.TUNNEL_STATUS_PENDING) return t('tools.tunnel.status.pending')
  return t('tools.tunnel.status.offline')
}
const statusVariant = (status: unknown): PillVariant => {
  if (status === TunnelStatus.TUNNEL_STATUS_ONLINE) return 'success'
  if (status === TunnelStatus.TUNNEL_STATUS_PENDING) return 'warning'
  return 'neutral'
}

const remoteText = (row: Record<string, unknown>) => {
  const type = String(row.type || '')
  if ([TunnelType.TUNNEL_TYPE_HTTP, TunnelType.TUNNEL_TYPE_HTTPS].includes(type as TunnelType)) {
    return String(row.customDomains || row.subdomain || '-')
  }
  return row.remotePort ? `:${row.remotePort}` : '-'
}

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('tools.tunnel.name'), minWidth: 110 },
  { key: 'type', title: t('tools.tunnel.proxyType'), width: 90 },
  { key: 'status', title: t('tools.tunnel.statusCol'), width: 90 },
  { key: 'remote', title: t('tools.tunnel.remotePort'), minWidth: 130 },
  { key: 'target', title: t('tools.tunnel.target'), minWidth: 150 },
  { key: 'isEnable', title: t('tools.tunnel.enabled'), width: 80 },
  { key: 'connections', title: t('tools.tunnel.stats.connections'), width: 80 },
  { key: 'traffic', title: t('tools.tunnel.stats.traffic'), minWidth: 150 },
  { key: 'createTime', title: t('tools.tunnel.createTime'), width: 140 },
  { key: 'operation', title: t('tools.tunnel.operation'), width: 80, align: 'center' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  { key: 'stats', label: t('tools.tunnel.stats.details'), icon: 'HOutline:ChartBarIcon' },
  { key: 'frpc', label: t('tools.tunnel.frpcConfig'), icon: 'HOutline:DocumentTextIcon' },
  { key: 'edit', label: t('tools.tunnel.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'delete', label: t('tools.tunnel.delete'), icon: 'HOutline:TrashIcon', danger: true },
])

const isAllSelected = computed(() => list.value.length > 0 && list.value.every((r) => selectedIds.value.includes(r.id)))
const toggleSelectAll = () => {
  selectedIds.value = isAllSelected.value ? [] : list.value.map((r) => r.id)
}
const toggleMobileSelect = (id: string) => {
  const idx = selectedIds.value.indexOf(id)
  if (idx === -1) selectedIds.value = [...selectedIds.value, id]
  else selectedIds.value = selectedIds.value.filter((d) => d !== id)
}

const findRow = (id: string) => list.value.find((x) => x.id === id)
const onRowAction = (key: string, row: Record<string, unknown>) => {
  const record = findRow(String(row.id))
  if (!record) return
  if (key === 'stats') openStatsDialog(record)
  else if (key === 'frpc') openFrpcDialog(record)
  else if (key === 'edit') openEditDialog(record)
  else if (key === 'delete') confirmDelete(record)
}

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listTunnels({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.keywords || undefined,
      type: (queryForm.type || undefined) as TunnelType | undefined,
      isEnable: queryForm.isEnable === '' ? undefined : queryForm.isEnable === 'true',
    })
    list.value = data?.infos || []
    pagination.value.total = Number(data?.total) || 0
  } catch {
    list.value = []
    pagination.value.total = 0
  } finally {
    loading.value = false
  }
}
const search = () => {
  pagination.value.page = 1
  getList()
}
const reset = () => {
  queryForm.keywords = ''
  queryForm.type = ''
  queryForm.isEnable = ''
  search()
}

const loadFrpsConfig = async () => {
  try {
    const { data } = await getFrpsConfig()
    frpsConfig.value = data?.config || null
  } catch {
    /* non-fatal */
  }
}
const onFrpsSaved = (config: FrpsConfig) => {
  frpsConfig.value = config
  getList()
}

const openCreateDialog = () => createRef.value?.showDialog()
const openEditDialog = (row: TunnelInfo) => createRef.value?.showDialog(row)
const openStatsDialog = (row: TunnelInfo) => {
  statsRow.value = row
  statsVisible.value = true
}
const openFrpcDialog = (row: TunnelInfo) => {
  frpcRow.value = row
  frpcVisible.value = true
}

const toggleEnable = async (row: Record<string, unknown>, val: boolean) => {
  const record = findRow(String(row.id))
  if (!record) return
  try {
    const { data } = await updateTunnel({ id: record.id, isEnable: val })
    if (data?.info) {
      record.isEnable = data.info.isEnable
      record.status = data.info.status
      ElMessage.success(data.info.isEnable ? t('tools.tunnel.enabled') : t('tools.tunnel.disabled'))
    }
  } catch {
    /* interceptor */
  }
}

const confirmDelete = (row: TunnelInfo) => {
  Dialog.confirm({
    title: t('tools.tunnel.confirmDeleteTitle'),
    content: t('tools.tunnel.confirmDeleteRule'),
    confirmText: t('tools.tunnel.confirmDeleteText'),
    cancelText: t('tools.tunnel.cancel'),
    onConfirm: async () => {
      await deleteTunnel({ id: row.id })
      ElMessage.success(t('tools.tunnel.deleteSuccess'))
      selectedIds.value = selectedIds.value.filter((d) => d !== row.id)
      getList()
    },
  })
}

const batchDelete = () => {
  if (!selectedIds.value.length) return
  Dialog.confirm({
    title: t('tools.tunnel.confirmDeleteTitle'),
    content: t('tools.tunnel.confirmDeleteSelected'),
    confirmText: t('tools.tunnel.confirmDeleteText'),
    cancelText: t('tools.tunnel.cancel'),
    onConfirm: async () => {
      await Promise.all(selectedIds.value.map((id) => deleteTunnel({ id })))
      ElMessage.success(t('tools.tunnel.batchDeleteSuccess'))
      selectedIds.value = []
      getList()
    },
  })
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') pagination.value.page = 1
  getList()
}

onMounted(() => {
  getList()
  loadFrpsConfig()
})
</script>

<style scoped lang="scss">
.tn-page { display: flex; flex-direction: column; gap: 10px; }
.tn-page__body { display: flex; flex-direction: column; gap: 8px; }
@media (width <= 768px) {
  .tn-page { gap: 8px; }
  .tn-page__body { gap: 6px; }
  .tn-page__batch { padding: 6px 10px; }
}
.tn-page__bar { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.tn-page__hint { font-size: 0.8125rem; color: var(--el-text-color-secondary); font-variant-numeric: tabular-nums; }
.tn-page__selall {
  border: none; background: transparent; color: var(--el-color-primary);
  font-size: 0.8125rem; cursor: pointer;
}
.tn-page__batch {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 8px 12px; border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius); background: color-mix(in srgb, var(--el-color-primary) 6%, transparent);
}
.tn-page__batch-count { font-size: 0.8125rem; font-weight: 600; color: var(--el-text-color-primary); }
.tn-page__batch-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.tn-mono {
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', Consolas, monospace;
  font-size: 0.78rem;
}
.tn-traffic { font-size: 0.75rem; color: var(--el-text-color-secondary); font-variant-numeric: tabular-nums; }
/* 移动紧凑行卡：三行密度 */
.tn-cards { display: flex; flex-direction: column; gap: 6px; }
.tn-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
}
.tn-card.is-selected {
  border-color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 6%, var(--el-bg-color));
}
.tn-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}
.tn-card__title {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  flex: 1;
}
.tn-card__check {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  accent-color: var(--el-color-primary);
}
.tn-card__name {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tn-card__route {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  font-size: 0.75rem;
  color: var(--el-text-color-regular);
  overflow: hidden;
  span {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
}
.tn-card__foot {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.tn-card__meta {
  flex: 1;
  min-width: 0;
  font-size: 0.7rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.tn-skeleton {
  height: 72px;
  border-radius: var(--app-radius);
  background: linear-gradient(100deg, var(--el-fill-color-light) 30%, var(--el-fill-color) 50%, var(--el-fill-color-light) 70%);
  background-size: 200% 100%;
  animation: tn-shimmer 1.4s ease-in-out infinite;
}
@keyframes tn-shimmer { from { background-position: 200% 0; } to { background-position: -200% 0; } }
</style>
