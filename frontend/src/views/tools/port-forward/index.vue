<!-- 端口转发（重写 · P1 列表）：PageHeader + FilterBar + 批量删除 + DataTable/移动卡 + Pagination。
     行内 统计/编辑/删除；启用 AppSwitch；错误 tip；统计 FormDialog。 -->
<template>
  <div class="pf-page">
    <PageHeader :title="t('tools.portForward.title')" :description="t('tools.portForward.pageDesc')">
      <template #actions>
        <UButton color="primary" size="sm" icon="i-lucide-plus" @click="openCreateDialog()">
          {{ t('tools.portForward.addRule') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="search" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('tools.portForward.keyword') }}</label>
          <input v-model="queryForm.keywords" class="app-input" :placeholder="t('tools.portForward.keywordPlaceholder')" @keyup.enter="search" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('tools.portForward.forwardType') }}</label>
          <AppSelect v-model="queryForm.type" :options="typeFilterOptions" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('tools.portForward.enableStatus') }}</label>
          <AppSelect v-model="queryForm.isEnable" :options="enableFilterOptions" />
        </div>
      </template>
    </FilterBar>

    <div v-if="selectedIds.length" class="pf-page__batch">
      <span class="pf-page__batch-count">{{ t('tools.portForward.selectedCount', { count: selectedIds.length }) }}</span>
      <div class="pf-page__batch-actions">
        <UButton color="error" variant="soft" size="sm" icon="i-lucide-trash-2" @click="batchDelete">
          {{ t('tools.portForward.batchDelete') }}
        </UButton>
        <UButton color="neutral" variant="ghost" size="sm" @click="selectedIds = []">
          {{ t('tools.portForward.clearSelection') }}
        </UButton>
      </div>
    </div>

    <div class="pf-page__body">
      <div class="pf-page__bar">
        <span class="pf-page__hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
        <button v-if="menuStore.isMobile && list.length" type="button" class="pf-page__selall" @click="toggleSelectAll">
          {{ isAllSelected ? t('tools.portForward.clearSelection') : t('tools.portForward.selectAll') }}
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
        :empty-text="t('tools.portForward.noData')"
      >
        <template #cell-name="{ row }">
          <span>{{ row.name }}</span>
          <span v-if="row.error" class="pf-err" :title="String(row.error)">!</span>
        </template>
        <template #cell-type="{ row }">
          <StatusPill
            :variant="row.type === 'PORT_FORWARD_TYPE_UDP' ? 'warning' : 'primary'"
            :label="row.type === 'PORT_FORWARD_TYPE_UDP' ? 'UDP' : 'TCP'"
            :dot="false"
          />
        </template>
        <template #cell-route="{ row }">
          <span class="pf-route">
            <span class="pf-mono">{{ row.listenAddress }}:{{ row.listenPort }}</span>
            <span class="pf-route__arrow">→</span>
            <span class="pf-mono">{{ row.targetAddress }}:{{ row.targetPort }}</span>
          </span>
        </template>
        <template #cell-isEnable="{ row }">
          <AppSwitch :model-value="!!row.isEnable" @update:model-value="(v) => toggleEnable(row, v)" />
        </template>
        <template #cell-connections="{ row }">{{ Number(row.activeConnections || 0) }}</template>
        <template #cell-traffic="{ row }">
          <span class="pf-traffic">↓ {{ formatBytes(row.bytesIn) }} · ↑ {{ formatBytes(row.bytesOut) }}</span>
        </template>
        <template #cell-createTime="{ row }">{{ formatTime(row.createTime) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="rowActions" @select="(key) => onRowAction(key, row)" />
        </template>
      </DataTable>

      <template v-else>
        <div v-if="loading" class="pf-cards">
          <div v-for="i in 4" :key="i" class="pf-skeleton" />
        </div>
        <EmptyState
          v-else-if="!list.length"
          icon="HOutline:ArrowPathRoundedSquareIcon"
          :title="t('tools.portForward.noData')"
          :description="t('tools.portForward.emptyDesc')"
        />
        <div v-else class="pf-cards">
          <div
            v-for="row in list"
            :key="row.id"
            class="pf-card"
            :class="{ 'is-selected': selectedIds.includes(row.id) }"
            @click="toggleMobileSelect(row.id)"
          >
            <div class="pf-card__top">
              <label class="pf-card__title" @click.stop>
                <input
                  type="checkbox"
                  class="pf-card__check"
                  :checked="selectedIds.includes(row.id)"
                  @change="toggleMobileSelect(row.id)"
                />
                <span class="pf-card__name">{{ row.name }}</span>
                <span v-if="row.error" class="pf-err" :title="row.error">!</span>
              </label>
              <StatusPill
                :variant="row.type === 'PORT_FORWARD_TYPE_UDP' ? 'warning' : 'primary'"
                :label="row.type === 'PORT_FORWARD_TYPE_UDP' ? 'UDP' : 'TCP'"
                :dot="false"
              />
            </div>
            <div class="pf-card__route pf-mono">
              {{ row.listenAddress }}:{{ row.listenPort }} → {{ row.targetAddress }}:{{ row.targetPort }}
            </div>
            <div class="pf-card__foot" @click.stop>
              <span class="pf-card__meta">
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

    <PortForwardCreate ref="createRef" @refresh="refresh" />
    <PortForwardStatsDialog v-model="statsVisible" :row="statsRow" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import PortForwardCreate from '@/views/tools/port-forward/create.vue'
import PortForwardStatsDialog from '@/views/tools/port-forward/PortForwardStatsDialog.vue'
import { listPortForwards, deletePortForward, updatePortForward } from '@/api/network'
import { PortForwardType, type PortForwardInfo } from '@/types/v1/network'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import { Dialog } from '@/utils/dialog'

defineOptions({ name: 'PortForwardView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const createRef = useTemplateRef<InstanceType<typeof PortForwardCreate> | null>('createRef')

const selectedIds = ref<string[]>([])
const list = ref<PortForwardInfo[]>([])
const loading = ref(false)
const statsVisible = ref(false)
const statsRow = ref<PortForwardInfo | null>(null)

const typeFilterOptions = computed(() => [
  { label: t('tools.portForward.allTypes'), value: '' },
  { label: 'TCP', value: PortForwardType.PORT_FORWARD_TYPE_TCP },
  { label: 'UDP', value: PortForwardType.PORT_FORWARD_TYPE_UDP },
])
const enableFilterOptions = computed(() => [
  { label: t('tools.portForward.allStatus'), value: '' },
  { label: t('tools.portForward.enabled'), value: 'true' },
  { label: t('tools.portForward.disabled'), value: 'false' },
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

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('tools.portForward.ruleName'), minWidth: 110 },
  { key: 'type', title: t('tools.portForward.forwardType'), width: 72 },
  { key: 'route', title: t('tools.portForward.routeCol'), minWidth: 280 },
  { key: 'isEnable', title: t('tools.portForward.enabled'), width: 80 },
  { key: 'connections', title: t('tools.portForward.stats.connections'), width: 72 },
  { key: 'traffic', title: t('tools.portForward.stats.traffic'), minWidth: 140 },
  { key: 'createTime', title: t('tools.portForward.createTime'), width: 130 },
  { key: 'operation', title: t('tools.portForward.operation'), width: 72, align: 'center' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  { key: 'stats', label: t('tools.portForward.stats.details'), icon: 'HOutline:ChartBarIcon' },
  { key: 'edit', label: t('tools.portForward.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'delete', label: t('tools.portForward.delete'), icon: 'HOutline:TrashIcon', danger: true },
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
  else if (key === 'edit') openEditDialog(record)
  else if (key === 'delete') confirmDelete(record)
}

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listPortForwards({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.keywords || undefined,
      type: (queryForm.type || undefined) as PortForwardType | undefined,
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

const openCreateDialog = () => createRef.value?.showDialog()
const openEditDialog = (row: PortForwardInfo) => createRef.value?.showDialog(row)
const openStatsDialog = (row: PortForwardInfo) => {
  statsRow.value = row
  statsVisible.value = true
}

const toggleEnable = async (row: Record<string, unknown>, val: boolean) => {
  const record = findRow(String(row.id))
  if (!record) return
  try {
    const { data } = await updatePortForward({ id: record.id, isEnable: val })
    if (data?.info) {
      record.isEnable = data.info.isEnable
      record.error = data.info.error
      if (data.info.error) feedback.error(data.info.error)
      else feedback.success(data.info.isEnable ? t('tools.portForward.enabled') : t('tools.portForward.disabled'))
    }
  } catch {
    /* interceptor */
  }
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

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') pagination.value.page = 1
  getList()
}

onMounted(() => {
  getList()
})
</script>

<style scoped lang="scss">
.pf-page { display: flex; flex-direction: column; gap: 10px; }
.pf-page__body { display: flex; flex-direction: column; gap: 8px; }
@media (width <= 768px) {
  .pf-page { gap: 8px; }
  .pf-page__body { gap: 6px; }
  .pf-page__batch { padding: 6px 10px; }
}
.pf-page__bar { display: flex; align-items: center; justify-content: space-between; gap: 12px; }
.pf-page__hint { font-size: 0.8125rem; color: var(--el-text-color-secondary); font-variant-numeric: tabular-nums; }
.pf-page__selall {
  border: none; background: transparent; color: var(--el-color-primary);
  font-size: 0.8125rem; cursor: pointer;
}
.pf-page__batch {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 8px 12px; border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius); background: color-mix(in srgb, var(--el-color-primary) 6%, transparent);
}
.pf-page__batch-count { font-size: 0.8125rem; font-weight: 600; color: var(--el-text-color-primary); }
.pf-page__batch-actions { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.pf-mono {
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', Consolas, monospace;
  font-size: 0.78rem;
}
.pf-route {
  display: inline-flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px;
}
.pf-route__arrow {
  color: var(--el-color-primary);
  font-weight: 700;
  font-size: 0.85rem;
}
.pf-traffic { font-size: 0.75rem; color: var(--el-text-color-secondary); font-variant-numeric: tabular-nums; }
.pf-err {
  display: inline-flex; align-items: center; justify-content: center;
  width: 14px; height: 14px; margin-left: 4px;
  border-radius: 50%; border: 1px solid var(--el-color-danger);
  color: var(--el-color-danger); font-size: 10px; font-weight: 700; cursor: help;
}
/* 移动紧凑行卡：三行密度，禁止 EntityCard 大留白 */
.pf-cards { display: flex; flex-direction: column; gap: 6px; }
.pf-card {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-bg-color);
}
.pf-card.is-selected {
  border-color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 6%, var(--el-bg-color));
}
.pf-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  min-width: 0;
}
.pf-card__title {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  flex: 1;
}
.pf-card__check {
  width: 14px;
  height: 14px;
  flex-shrink: 0;
  accent-color: var(--el-color-primary);
}
.pf-card__name {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pf-card__route {
  font-size: 0.75rem;
  color: var(--el-text-color-regular);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pf-card__foot {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.pf-card__meta {
  flex: 1;
  min-width: 0;
  font-size: 0.7rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.pf-skeleton {
  height: 72px;
  border-radius: var(--app-radius);
  background: linear-gradient(100deg, var(--el-fill-color-light) 30%, var(--el-fill-color) 50%, var(--el-fill-color-light) 70%);
  background-size: 200% 100%;
  animation: pf-shimmer 1.4s ease-in-out infinite;
}
@keyframes pf-shimmer { from { background-position: 200% 0; } to { background-position: -200% 0; } }
</style>
