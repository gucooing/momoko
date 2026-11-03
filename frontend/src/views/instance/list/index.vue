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

    <!-- 工具条：状态胶囊 + 搜索 + 类型 + 视图切换（替代表单式筛选） -->
    <div class="inst-tools">
      <div class="inst-chips" role="tablist" :aria-label="t('common.status')">
        <button
          v-for="opt in statusChips"
          :key="String(opt.value)"
          type="button"
          role="tab"
          class="inst-chip"
          :class="{ 'is-on': queryForm.status === opt.value }"
          :aria-selected="queryForm.status === opt.value"
          @click="setStatusFilter(opt.value)"
        >
          {{ opt.label }}
        </button>
      </div>
      <span class="inst-tools__spacer" />
      <label class="inst-search">
        <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" />
        <input
          v-model="queryForm.keyword"
          type="search"
          :placeholder="t('instance.searchPlaceholder')"
          :aria-label="t('instance.searchPlaceholder')"
          @keyup.enter="applyFilters"
        />
      </label>
      <div class="inst-type">
        <AppSelect v-model="queryForm.type" :options="typeFilterOptions" />
      </div>
      <div v-if="!menuStore.isMobile" class="seg">
        <button
          type="button"
          class="seg__btn"
          :class="{ 'is-active': viewMode === 'card' }"
          :aria-label="t('system.common.card')"
          @click="viewMode = 'card'"
        >
          <component :is="menuStore.iconComponents['HOutline:Squares2X2Icon']" />
        </button>
        <button
          type="button"
          class="seg__btn"
          :class="{ 'is-active': viewMode === 'table' }"
          :aria-label="t('system.common.table')"
          @click="viewMode = 'table'"
        >
          <component :is="menuStore.iconComponents['HOutline:Bars3Icon']" />
        </button>
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
        <button
          v-if="menuStore.isMobile || viewMode === 'card'"
          type="button"
          class="inst-page__selall"
          @click="toggleCurrentPageSelection"
        >
          {{ isCurrentPageAllSelected ? t('instance.deselectAll') : t('instance.selectCurrentPage') }}
        </button>
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

    <!-- 批量操作条：选中后浮在底部 -->
    <Transition name="batch-pop">
      <div v-if="selectedIds.length" class="inst-batch" role="toolbar">
        <span class="inst-batch__count">{{ t('instance.selectedItems', { count: selectedIds.length }) }}</span>
        <button
          type="button"
          class="inst-batch__btn"
          :disabled="!canBatchStart"
          @click="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_RUNNING)"
        >
          <component :is="menuStore.iconComponents['HOutline:PlayIcon']" />{{ t('instance.batchStart') }}
        </button>
        <button
          type="button"
          class="inst-batch__btn"
          :disabled="!canBatchStop"
          @click="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_STOPPED)"
        >
          <component :is="menuStore.iconComponents['HOutline:StopIcon']" />{{ t('instance.batchStop') }}
        </button>
        <button type="button" class="inst-batch__btn is-danger" :disabled="!canBatchDelete" @click="handleBatchDelete">
          <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />{{ t('instance.batchDelete') }}
        </button>
        <button type="button" class="inst-batch__btn is-icon" :aria-label="t('instance.deselectAll')" @click="selectedIds = []">
          <component :is="menuStore.iconComponents['HOutline:XMarkIcon']" />
        </button>
      </div>
    </Transition>

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
import { useDebounceFn } from '@vueuse/core'
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

const nodeStore = useNodeStore()
// 实例归属按所在节点的用户比较：远程节点上为该节点 API Key 所属用户。
const currentUserId = computed(() =>
  nodeStore.isRemote ? nodeStore.remoteUserId : userStore.userInfo?.userId?.trim() || '',
)

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
// 只给用户能理解的三种状态做筛选胶囊（未指定/未知不单列）
const CHIP_STATUSES = [
  InstanceStatus.INSTANCE_STATUS_RUNNING,
  InstanceStatus.INSTANCE_STATUS_STOPPED,
  InstanceStatus.INSTANCE_STATUS_MAINTENANCE,
] as const
const statusChips = computed(() => [
  { label: t('instance.allStatus'), value: '' as InstanceStatus | '' },
  ...statusOptions.value.filter((o) => (CHIP_STATUSES as readonly InstanceStatus[]).includes(o.value)),
])
const setStatusFilter = (value: InstanceStatus | '') => {
  queryForm.value.status = value
  applyFilters()
}
// 输入即搜索（防抖）、切换类型即筛选，省去「搜索」按钮
const debouncedApply = useDebounceFn(() => applyFilters(), 350)
watch(() => queryForm.value.keyword, () => debouncedApply())
watch(() => queryForm.value.type, () => applyFilters())

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
    path: `/instance/app/${item.id}`,
    query: { tabTitle: item.name },
  })
}

const openInstanceFileManager = (
  item: Pick<InstanceRecord, 'id' | 'name' | 'status' | 'instancePath'>,
) => {
  // 实例文件 API 以实例目录为根，不能再把 instancePath 当地址（会 500「路径不存在」）
  router.push({
    path: `/instance/app/${item.id}`,
    query: { tab: 'files', tabTitle: item.name },
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
  gap: 16px;
}

/* 工具条 */
.inst-tools {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.inst-tools__spacer {
  flex: 1;
}
.inst-chips {
  display: flex;
  gap: 8px;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.inst-chips::-webkit-scrollbar {
  display: none;
}
.inst-chip {
  flex: none;
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}
.inst-chip:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.inst-chip.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.inst-search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 280px;
  min-width: 180px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  transition: border-color 0.15s;
}
.inst-search:focus-within {
  border-color: var(--m-p);
}
.inst-search :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
}
.inst-search input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
}
.inst-type {
  width: 170px;
}
@media (width < 768px) {
  .inst-tools__spacer {
    display: none;
  }
  .inst-chips {
    order: 3;
    width: 100%;
  }
  .inst-search {
    flex: 1 1 0;
    min-width: 0;
  }
  .inst-type {
    width: 128px;
  }
}

/* 主体 */
.inst-page__body {
  display: flex;
  flex-direction: column;
  gap: 12px;
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
.inst-page__bar-hint,
.inst-page__stat {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.inst-page__dot {
  width: 8px;
  height: 8px;
  border-radius: 999px;
  flex-shrink: 0;
}
.inst-page__dot.is-run {
  background: var(--m-ok);
}
.inst-page__dot.is-stop {
  background: var(--el-text-color-secondary);
  opacity: 0.5;
}
.inst-page__selall {
  min-height: 36px;
  border: none;
  background: transparent;
  color: var(--m-pInk);
  font-size: 0.875rem;
  font-weight: 700;
  cursor: pointer;
  white-space: nowrap;
}

/* 卡/表切换 */
.seg {
  display: inline-flex;
  padding: 4px;
  gap: 2px;
  background: var(--el-fill-color-light);
  border-radius: 999px;
}
.seg__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 40px;
  min-height: 32px;
  border: none;
  background: transparent;
  border-radius: 999px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
}
.seg__btn :deep(svg) {
  width: 17px;
  height: 17px;
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
  gap: 14px;
}
@media (width >= 560px) {
  .inst-grid {
    grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  }
}
.inst-skeleton {
  height: 260px;
  border-radius: var(--app-radius-lg);
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

/* 浮动批量条 */
.inst-batch {
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
.inst-batch__count {
  margin-right: 10px;
  font-size: 0.875rem;
  font-weight: 800;
  white-space: nowrap;
}
.inst-batch__btn {
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
.inst-batch__btn:hover:not(:disabled) {
  background: color-mix(in srgb, var(--m-bg) 14%, transparent);
}
.inst-batch__btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.inst-batch__btn.is-danger {
  color: var(--m-a);
}
.inst-batch__btn.is-icon {
  width: 40px;
  padding: 0;
  justify-content: center;
}
.inst-batch__btn :deep(svg) {
  width: 16px;
  height: 16px;
}
@media (width < 992px) {
  .inst-batch {
    bottom: calc(78px + env(safe-area-inset-bottom));
  }
}
.batch-pop-enter-active,
.batch-pop-leave-active {
  transition: opacity 0.18s, transform 0.18s;
}
.batch-pop-enter-from,
.batch-pop-leave-to {
  opacity: 0;
  transform: translateY(12px);
}
</style>
