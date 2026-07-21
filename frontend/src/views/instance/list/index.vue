<!-- 实例管理（重写 · P1 列表）：概览指标带 + PageHeader + FilterBar + 批量条 + 卡/表切换
     + InstanceCard 卡片流 / DataTable 表视图 + Pagination + 受控 InstanceEditor。
     页面为 store(useInstanceListStore) 的薄壳：保留全部状态/动作契约与业务处理逻辑。 -->
<template>
  <div class="inst-page">
    <PageHeader :title="t('instance.title')" :description="t('instance.pageDesc')">
      <template #actions>
        <UButton
          color="neutral"
          variant="soft"
          icon="i-lucide-rotate-cw"
          @click="handleRefreshStatus"
        >
          {{ t('common.refresh') }}
        </UButton>
        <UButton color="primary" icon="i-lucide-plus" @click="handleOpenCreateEditor">
          {{ t('instance.createInstance') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="applyFilters" @reset="resetFilters">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('instance.searchPlaceholder') }}</label>
          <input
            v-model="queryForm.keyword"
            class="app-input"
            :placeholder="t('instance.searchPlaceholder')"
            @keyup.enter="applyFilters"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('common.type') }}</label>
          <AppSelect v-model="queryForm.type" :options="typeFilterOptions" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('common.status') }}</label>
          <AppSelect v-model="queryForm.status" :options="statusFilterOptions" />
        </div>
      </template>
    </FilterBar>

    <!-- 批量操作条（选中后出现） -->
    <div v-if="selectedIds.length" class="inst-page__batch">
      <span class="inst-page__batch-count">{{ t('instance.selectedItems', { count: selectedIds.length }) }}</span>
      <div class="inst-page__batch-actions">
        <UButton
          color="primary"
          variant="soft"
          size="sm"
          icon="i-lucide-play"
          :disabled="!canBatchStart"
          @click="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_RUNNING)"
        >
          {{ t('instance.batchStart') }}
        </UButton>
        <UButton
          color="neutral"
          variant="soft"
          size="sm"
          icon="i-lucide-square"
          :disabled="!canBatchStop"
          @click="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_STOPPED)"
        >
          {{ t('instance.batchStop') }}
        </UButton>
        <UButton
          color="error"
          variant="soft"
          size="sm"
          icon="i-lucide-trash-2"
          :disabled="!canBatchDelete"
          @click="handleBatchDelete"
        >
          {{ t('instance.batchDelete') }}
        </UButton>
        <UButton color="neutral" variant="ghost" size="sm" @click="selectedIds = []">
          {{ t('instance.deselectAll') }}
        </UButton>
      </div>
    </div>

    <!-- 主体：卡/表切换（桌面）；移动强制卡片 -->
    <div class="inst-page__body">
      <div class="inst-page__bar">
        <div class="inst-page__stats">
          <span class="inst-page__bar-hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
          <span class="inst-page__stat">
            <span class="inst-page__dot is-run" />{{ t('instance.running') }} {{ runningCount }}
          </span>
          <span class="inst-page__stat">
            <span class="inst-page__dot is-stop" />{{ t('instance.stopped') }} {{ stoppedCount }}
          </span>
        </div>
        <div class="inst-page__bar-right">
          <button
            v-if="menuStore.isMobile || viewMode === 'card'"
            type="button"
            class="inst-page__selall"
            @click="toggleCurrentPageSelection"
          >
            {{ isCurrentPageAllSelected ? t('instance.deselectAll') : t('instance.selectCurrentPage') }}
          </button>
          <div v-if="!menuStore.isMobile" class="seg">
            <button
              type="button"
              class="seg__btn"
              :class="{ 'is-active': viewMode === 'card' }"
              @click="viewMode = 'card'"
            >
              <component :is="menuStore.iconComponents['HOutline:Squares2X2Icon']" />
              {{ t('system.common.card') }}
            </button>
            <button
              type="button"
              class="seg__btn"
              :class="{ 'is-active': viewMode === 'table' }"
              @click="viewMode = 'table'"
            >
              <component :is="menuStore.iconComponents['HOutline:Bars3Icon']" />
              {{ t('system.common.table') }}
            </button>
          </div>
        </div>
      </div>

      <!-- 卡片流 -->
      <template v-if="menuStore.isMobile || viewMode === 'card'">
        <div v-if="loading" class="inst-grid">
          <div v-for="i in 6" :key="i" class="inst-skeleton" />
        </div>
        <EmptyState
          v-else-if="!pagedInstances.length"
          icon="HOutline:ServerStackIcon"
          :title="t('instance.noMatchedInstances')"
          :description="t('instance.emptyDesc')"
        >
          <template #action>
            <UButton color="primary" variant="soft" icon="i-lucide-plus" @click="handleOpenCreateEditor">
              {{ t('instance.createInstance') }}
            </UButton>
          </template>
        </EmptyState>
        <div v-else class="inst-grid">
          <InstanceCard
            v-for="item in pagedInstances"
            :key="item.id"
            :item="item"
            :type-label="resolveTypeLabel(item.type)"
            :can-delete="item.userId === currentUserId"
            :selected="selectedIdSet.has(item.id)"
            @toggle-select="setSelection(item.id, $event)"
            @console="openInstanceConsole(item)"
            @config="handleOpenEditEditor(item.id)"
            @change-status="handleChangeInstanceStatus(item.id, $event)"
            @more-action="handleMoreAction(item.id, $event)"
          />
        </div>
      </template>

      <!-- 表视图（桌面） -->
      <DataTable
        v-else
        v-model="selectedIds"
        :columns="columns"
        :rows="pagedInstances"
        row-key="id"
        selectable
        :loading="loading"
        :empty-text="t('instance.noMatchedInstances')"
      >
        <template #cell-type="{ row }">{{ resolveTypeLabel(String(row.type || '')) }}</template>
        <template #cell-status="{ row }">
          <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
        </template>
        <template #cell-createTime="{ row }">{{ fmtTime(row.createTime) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="rowActionsFor(row)" @select="(key) => onRowAction(key, row)" />
        </template>
      </DataTable>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :page-sizes="INSTANCE_PAGE_SIZES"
        @change="handlePageChange"
      />
    </div>

    <InstanceEditor
      :visible="instanceEditorVisible"
      :mode="instanceEditorMode"
      :loading="instanceEditorLoading"
      :submitting="instanceEditorSubmitting"
      :form="instanceEditorForm"
      :type-options="typeOptions"
      @close="handleEditorClose"
      @submit="handleEditorSubmit"
    />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import dayjs from 'dayjs'
import { useI18n } from 'vue-i18n'
import { useInstanceListStore } from '@/stores/instance/list'
import {
  InstanceStatus,
  statusMeta,
  type InstanceEditorFormValue,
  type InstanceRecord,
} from '@/stores/instance/types'
import { useUserStore } from '@/stores/user'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import InstanceCard from '@/views/instance/list/instanceCard.vue'
import InstanceEditor from '@/views/instance/list/instanceEditor.vue'

defineOptions({ name: 'ListView' })

const menuStore = useMenuStore()
const router = useRouter()
const instanceListStore = useInstanceListStore()
const userStore = useUserStore()
const { t } = useI18n()

const INSTANCE_PAGE_SIZES = [9, 18, 27]
const viewMode = ref<'card' | 'table'>('card')

const currentUserId = computed(() => userStore.userInfo?.userId?.trim() || '')

const {
  queryForm,
  loading,
  selectedIds,
  pagination,
  selectedIdSet,
  typeOptions,
  statusOptions,
  pagedInstances,
  isCurrentPageAllSelected,
  canBatchStart,
  canBatchStop,
  canBatchDelete,
  instanceEditorVisible,
  instanceEditorMode,
  instanceEditorLoading,
  instanceEditorSubmitting,
  instanceEditorForm,
} = storeToRefs(instanceListStore)

const {
  setSelection,
  toggleCurrentPageSelection,
  openCreateEditor,
  openEditEditor,
  closeInstanceEditor,
  submitInstanceEditor,
  applyFilters,
  resetFilters,
  handlePageChange,
  refreshStatus,
  batchChangeStatus,
  batchDeleteInstances,
  changeInstanceStatus,
  deleteInstances,
  restartInstance,
  findInstanceById,
  resolveTypeLabel,
  initialize,
} = instanceListStore

// —— 当前页运行/停止计数（内联进工具条，替代大号总览带）——
const runningCount = computed(
  () => pagedInstances.value.filter((i) => i.status === InstanceStatus.INSTANCE_STATUS_RUNNING).length,
)
const stoppedCount = computed(
  () => pagedInstances.value.filter((i) => i.status === InstanceStatus.INSTANCE_STATUS_STOPPED).length,
)

// —— 筛选下拉（补“全部”项，值为空串，对应 store 的 keyword/type/status='' 语义）——
const typeFilterOptions = computed(() => [
  { label: t('instance.allTypes'), value: '' },
  ...typeOptions.value,
])
const statusFilterOptions = computed(() => [
  { label: t('instance.allStatus'), value: '' as InstanceStatus | '' },
  ...statusOptions.value,
])

// —— 表视图列 + 单元格渲染 ——
const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('instance.instanceName'), minWidth: 160 },
  { key: 'type', title: t('common.type'), width: 150 },
  { key: 'status', title: t('common.status'), width: 110 },
  { key: 'instancePath', title: t('instance.instancePath'), minWidth: 180 },
  { key: 'createTime', title: t('instance.createTime'), width: 170 },
  { key: 'operation', title: t('common.operation'), width: 80, align: 'center' },
])

type PillVariant = 'success' | 'neutral' | 'warning' | 'error'
const statusVariantMap: Record<InstanceStatus, PillVariant> = {
  [InstanceStatus.INSTANCE_STATUS_RUNNING]: 'success',
  [InstanceStatus.INSTANCE_STATUS_STOPPED]: 'neutral',
  [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: 'warning',
  [InstanceStatus.INSTANCE_STATUS_UNSPECIFIED]: 'neutral',
  [InstanceStatus.UNRECOGNIZED]: 'error',
}
const statusVariant = (status: unknown) =>
  statusVariantMap[status as InstanceStatus] || 'neutral'
const statusLabel = (status: unknown) => {
  const meta = statusMeta[status as InstanceStatus]
  return meta ? t(meta.labelKey) : t('instance.unknownStatus')
}
const fmtTime = (value: unknown) => {
  if (!value) return '—'
  const date = dayjs(value as string | Date)
  return date.isValid() ? date.format('YYYY-MM-DD HH:mm') : '—'
}

const rowActionsFor = (row: Record<string, unknown>): ActionMenuItem[] => {
  const isRunning = row.status === InstanceStatus.INSTANCE_STATUS_RUNNING
  return [
    { key: 'console', label: t('instance.console'), icon: 'HOutline:CommandLineIcon' },
    { key: 'config', label: t('instance.config'), icon: 'HOutline:Cog6ToothIcon' },
    {
      key: isRunning ? 'stop' : 'start',
      label: isRunning ? t('instance.stop') : t('instance.start'),
      icon: isRunning ? 'HOutline:StopIcon' : 'HOutline:PlayIcon',
    },
    { key: 'forceRestart', label: t('instance.forceRestart'), icon: 'HOutline:ArrowPathIcon' },
    { key: 'fileManager', label: t('instance.fileManagerTitle'), icon: 'HOutline:FolderIcon' },
    {
      key: 'delete',
      label: t('common.delete'),
      icon: 'HOutline:TrashIcon',
      danger: true,
      hidden: String(row.userId) !== currentUserId.value,
    },
  ]
}

const onRowAction = (key: string, row: Record<string, unknown>) => {
  const inst = findInstanceById(String(row.id))
  if (!inst) return
  if (key === 'console') openInstanceConsole(inst)
  else if (key === 'config') handleOpenEditEditor(inst.id)
  else if (key === 'start') handleChangeInstanceStatus(inst.id, InstanceStatus.INSTANCE_STATUS_RUNNING)
  else if (key === 'stop') handleChangeInstanceStatus(inst.id, InstanceStatus.INSTANCE_STATUS_STOPPED)
  else handleMoreAction(inst.id, key as 'forceRestart' | 'delete' | 'fileManager')
}

// —— 导航 ——
const openInstanceConsole = (item: InstanceRecord) => {
  router.push({
    path: `/instance/console/${item.id}`,
    query: { tabTitle: item.name },
  })
}

const openInstanceFileManager = (
  item: Pick<InstanceRecord, 'id' | 'name' | 'status' | 'instancePath'>,
) => {
  // 实例文件 API 以实例目录为根，不能再把 instancePath 当地址（会 500「路径不存在」）
  router.push({
    path: `/instance/files/${item.id}`,
    query: {
      tabTitle: t('instance.instanceFileTab', { name: item.name }),
      from: 'instance',
      status: item.status,
    },
  })
}

// —— 编辑器 ——
const handleOpenCreateEditor = () => {
  openCreateEditor()
}

const handleOpenEditEditor = async (instanceId: string) => {
  try {
    await openEditEditor(instanceId)
  } catch (error) {
    showRequestError(error, t('instance.loadConfigFailed'))
  }
}

const handleEditorClose = () => {
  closeInstanceEditor()
}

const handleEditorSubmit = async (form: InstanceEditorFormValue) => {
  const mode = instanceEditorMode.value
  try {
    instanceEditorForm.value = { ...form }
    await submitInstanceEditor()
    feedback.success(mode === 'create' ? t('instance.createSuccess') : t('instance.configSaveSuccess'))
  } catch (error) {
    showRequestError(error, mode === 'create' ? t('instance.createFailed') : t('instance.configSaveFailed'))
  }
}

const handleRefreshStatus = async () => {
  await refreshStatus()
  feedback.success(t('instance.statusRefreshed'))
}

// —— 批量 / 单条 状态与删除 ——
const failedSuffix = (count: number) => (count ? t('instance.failedCountSuffix', { count }) : '')

const executeBatchChangeStatus = async (
  targetStatus:
    | typeof InstanceStatus.INSTANCE_STATUS_RUNNING
    | typeof InstanceStatus.INSTANCE_STATUS_STOPPED,
) => {
  if (!selectedIds.value.length) return
  const { successCount, failedCount } = await batchChangeStatus(targetStatus)
  if (!successCount && failedCount) return
  feedback.success(
    targetStatus === InstanceStatus.INSTANCE_STATUS_RUNNING
      ? t('instance.batchStartSuccess', { success: successCount, failed: failedSuffix(failedCount) })
      : t('instance.batchStopSuccess', { success: successCount, failed: failedSuffix(failedCount) }),
  )
}

const handleBatchChangeStatus = async (
  targetStatus:
    | typeof InstanceStatus.INSTANCE_STATUS_RUNNING
    | typeof InstanceStatus.INSTANCE_STATUS_STOPPED,
) => {
  if (!selectedIds.value.length) return
  if (targetStatus === InstanceStatus.INSTANCE_STATUS_STOPPED) {
    Dialog.confirm({
      title: t('instance.confirmBatchStopTitle'),
      content: t('instance.confirmBatchStopContent', { count: selectedIds.value.length }),
      cancelText: t('common.cancel'),
      confirmText: t('instance.confirmStop'),
      onConfirm: async () => { await executeBatchChangeStatus(targetStatus) },
    })
    return
  }
  await executeBatchChangeStatus(targetStatus)
}

const executeChangeInstanceStatus = async (
  id: string,
  targetStatus:
    | typeof InstanceStatus.INSTANCE_STATUS_RUNNING
    | typeof InstanceStatus.INSTANCE_STATUS_STOPPED,
) => {
  const success = await changeInstanceStatus(id, targetStatus)
  if (!success) return
  feedback.success(
    targetStatus === InstanceStatus.INSTANCE_STATUS_RUNNING ? t('instance.instanceStarted') : t('instance.instanceStopped'),
  )
}

const handleChangeInstanceStatus = async (
  id: string,
  targetStatus:
    | typeof InstanceStatus.INSTANCE_STATUS_RUNNING
    | typeof InstanceStatus.INSTANCE_STATUS_STOPPED,
) => {
  const current = findInstanceById(id)
  if (!current) return
  if (targetStatus === InstanceStatus.INSTANCE_STATUS_STOPPED) {
    Dialog.confirm({
      title: t('instance.confirmStopInstanceTitle'),
      content: t('instance.confirmStopInstanceContent', { name: current.name }),
      cancelText: t('common.cancel'),
      confirmText: t('instance.confirmStop'),
      onConfirm: async () => { await executeChangeInstanceStatus(id, targetStatus) },
    })
    return
  }
  await executeChangeInstanceStatus(id, targetStatus)
}

const handleBatchDelete = async () => {
  Dialog.confirm({
    title: t('instance.confirmBatchDeleteTitle'),
    content: t('instance.confirmBatchDeleteContent'),
    cancelText: t('common.cancel'),
    confirmText: t('common.delete'),
    onConfirm: async () => {
      const { successCount, failedCount } = await batchDeleteInstances()
      if (!successCount && failedCount) return
      feedback.success(t('instance.batchDeleteSuccess', { success: successCount, failed: failedSuffix(failedCount) }))
    },
  })
}

const handleMoreAction = async (id: string, action: 'forceRestart' | 'delete' | 'fileManager') => {
  const current = findInstanceById(id)
  if (!current) return

  if (action === 'forceRestart') {
    Dialog.confirm({
      title: t('instance.confirmForceRestartInstanceTitle'),
      content: t('instance.confirmForceRestartInstanceContent', { name: current.name }),
      cancelText: t('common.cancel'),
      confirmText: t('instance.confirmRestart'),
      onConfirm: async () => {
        const success = await restartInstance(id)
        if (!success) return
        feedback.success(t('instance.forceRestarting'))
      },
    })
    return
  }

  if (action === 'delete') {
    if (current.userId !== currentUserId.value) return
    Dialog.confirm({
      title: t('instance.confirmDeleteInstanceTitle'),
      content: t('instance.confirmDeleteInstanceContent', { name: current.name }),
      cancelText: t('common.cancel'),
      confirmText: t('common.delete'),
      onConfirm: async () => {
        const { successCount, failedCount } = await deleteInstances([id])
        if (!successCount && failedCount) return
        feedback.success(t('instance.instanceDeleted', { name: current.name, failed: failedSuffix(failedCount) }))
      },
    })
    return
  }

  openInstanceFileManager(current)
}

onMounted(() => {
  void initialize()
})
</script>

<style scoped lang="scss">
.inst-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* 批量条 */
.inst-page__batch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 30%, var(--el-border-color-lighter));
  border-radius: var(--app-radius-lg);
  background: color-mix(in srgb, var(--el-color-primary) 5%, var(--el-bg-color));
}
.inst-page__batch-count {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-color-primary);
  white-space: nowrap;
}
.inst-page__batch-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

/* 主体 */
.inst-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.inst-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.inst-page__stats {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 16px;
}
.inst-page__bar-hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.inst-page__stat {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.inst-page__dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  flex-shrink: 0;
  background: var(--el-text-color-placeholder);
}
.inst-page__dot.is-run {
  background: var(--el-color-success, #16a34a);
}
.inst-page__dot.is-stop {
  background: var(--el-text-color-placeholder);
}
.inst-page__bar-right {
  display: flex;
  align-items: center;
  gap: 12px;
}
.inst-page__selall {
  border: none;
  background: transparent;
  color: var(--el-color-primary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  white-space: nowrap;
}
.inst-page__selall:hover {
  text-decoration: underline;
}

/* 卡/表切换 分段控件 */
.seg {
  display: inline-flex;
  padding: 3px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius);
}
.seg__btn {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  padding: 5px 12px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.seg__btn :deep(svg) {
  width: 15px;
  height: 15px;
}
.seg__btn.is-active {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: var(--app-shadow-sm);
}

/* 卡片流 */
.inst-grid {
  display: grid;
  grid-template-columns: 1fr;
  gap: 12px;
}
@media (width >= 560px) {
  .inst-grid {
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  }
}

/* 卡片骨架 */
.inst-skeleton {
  height: 168px;
  border-radius: var(--app-radius);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: inst-shimmer 1.4s ease-in-out infinite;
}
@keyframes inst-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
