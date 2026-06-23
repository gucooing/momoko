<template>
  <div class="instance-list-page">
    <OverviewStats :cards="summaryCards" />

    <BaseCard :title="t('instance.listTitle')" title-icon="HOutline:ServerStackIcon">
      <div v-loading="loading">
        <div class="list-toolbar">
          <div class="toolbar-filters">
            <el-input
              :model-value="queryForm.keyword"
              :placeholder="t('instance.searchPlaceholder')"
              clearable
              style="width: 180px"
              @update:model-value="(v: string) => queryForm.keyword = v"
              @keyup.enter="applyFilters"
            />
            <el-select
              :model-value="queryForm.type"
              :placeholder="t('common.type')"
              clearable
              style="width: 130px"
              @update:model-value="(v: string) => queryForm.type = v"
            >
              <el-option v-for="item in typeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-select
              :model-value="queryForm.status"
              :placeholder="t('common.status')"
              clearable
              style="width: 110px"
              @update:model-value="(v: any) => queryForm.status = v"
            >
              <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-button type="primary" size="small" @click="applyFilters">{{ t('common.search') }}</el-button>
            <el-button size="small" @click="resetFilters">{{ t('common.reset') }}</el-button>
          </div>

          <div class="toolbar-actions">
            <span v-if="selectedIds.length" class="selection-text">{{ t('instance.selectedItems', { count: selectedIds.length }) }}</span>
            <el-button
              type="primary"
              :icon="menuStore.iconComponents['HOutline:PlusCircleIcon']"
              @click="handleOpenCreateEditor"
            >
              {{ t('common.create') }}
            </el-button>
            <el-button
              :icon="menuStore.iconComponents['HOutline:PlayIcon']"
              :disabled="!canBatchStart"
              @click="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_RUNNING)"
            >
              {{ t('instance.batchStart') }}
            </el-button>
            <el-button
              :icon="menuStore.iconComponents['HOutline:StopIcon']"
              :disabled="!canBatchStop"
              @click="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_STOPPED)"
            >
              {{ t('instance.batchStop') }}
            </el-button>
            <el-button
              type="danger"
              :icon="menuStore.iconComponents['HOutline:TrashIcon']"
              :disabled="!canBatchDelete"
              @click="handleBatchDelete"
            >
              {{ t('instance.batchDelete') }}
            </el-button>
            <el-button link @click="toggleCurrentPageSelection">
              {{ isCurrentPageAllSelected ? t('instance.deselectAll') : t('instance.selectCurrentPage') }}
            </el-button>
            <el-button link :icon="menuStore.iconComponents['HOutline:ArrowPathRoundedSquareIcon']" @click="handleRefreshStatus">
              {{ t('common.refresh') }}
            </el-button>
          </div>
        </div>

        <Transition name="instance-page-switch" mode="out-in">
          <div v-if="pagedInstances.length" :key="gridTransitionKey" class="instance-grid">
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

          <el-empty v-else key="instance-empty" :description="t('instance.noMatchedInstances')" />
        </Transition>

        <div class="instance-pagination">
          <TablePagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            :is-mobile="menuStore.isMobile"
            :page-sizes="INSTANCE_PAGE_SIZES"
            @change="handlePageChange"
          />
        </div>
      </div>
    </BaseCard>

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
import BaseCard from '@/components/card/BaseCard.vue'
import TablePagination from '@/components/pagination/TablePagination.vue'
import { useInstanceListStore } from '@/stores/instance/list'
import {
  InstanceStatus,
  type InstanceEditorFormValue,
  type InstanceRecord,
} from '@/stores/instance/types'
import { useUserStore } from '@/stores/user'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import InstanceCard from '@/views/instance/list/instanceCard.vue'
import InstanceEditor from '@/views/instance/list/instanceEditor.vue'
import OverviewStats from '@/views/instance/list/overviewStats.vue'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'ListView' })

const menuStore = useMenuStore()
const router = useRouter()
const instanceListStore = useInstanceListStore()
const userStore = useUserStore()
const { t } = useI18n()
const INSTANCE_PAGE_SIZES = [9, 18, 27]

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
  gridTransitionKey,
  isCurrentPageAllSelected,
  summaryCards,
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

const openInstanceConsole = (item: InstanceRecord) => {
  router.push({
    path: `/instance/console/${item.id}`,
    query: { tabTitle: item.name },
  })
}

const openInstanceFileManager = (
  item: Pick<InstanceRecord, 'id' | 'name' | 'status' | 'instancePath'>,
) => {
  router.push({
    path: `/instance/files/${item.id}`,
    query: {
      tabTitle: t('instance.instanceFileTab', { name: item.name }),
      from: 'instance',
      status: item.status,
      workdir: item.instancePath,
    },
  })
}

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
    ElMessage.success(mode === 'create' ? t('instance.createSuccess') : t('instance.configSaveSuccess'))
  } catch (error) {
    showRequestError(error, mode === 'create' ? t('instance.createFailed') : t('instance.configSaveFailed'))
  }
}

const handleRefreshStatus = async () => {
  await refreshStatus()
  ElMessage.success(t('instance.statusRefreshed'))
}

const failedSuffix = (count: number) => count ? t('instance.failedCountSuffix', { count }) : ''

const executeBatchChangeStatus = async (
  targetStatus:
    | typeof InstanceStatus.INSTANCE_STATUS_RUNNING
    | typeof InstanceStatus.INSTANCE_STATUS_STOPPED,
) => {
  if (!selectedIds.value.length) return
  const { successCount, failedCount } = await batchChangeStatus(targetStatus)
  if (!successCount && failedCount) return
  ElMessage.success(
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
  ElMessage.success(
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
      ElMessage.success(t('instance.batchDeleteSuccess', { success: successCount, failed: failedSuffix(failedCount) }))
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
        ElMessage.success(t('instance.forceRestarting'))
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
        ElMessage.success(t('instance.instanceDeleted', { name: current.name, failed: failedSuffix(failedCount) }))
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
.instance-list-page {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.list-toolbar {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.toolbar-filters {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.toolbar-actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-wrap: wrap;
}

.selection-text {
  font-size: 0.8rem;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}

.instance-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 0.8rem;
  margin-top: 0.8rem;
}

.instance-pagination {
  margin-top: 1rem;
}

.instance-page-switch-enter-active,
.instance-page-switch-leave-active {
  transition: opacity 0.2s ease;
}

.instance-page-switch-enter-from,
.instance-page-switch-leave-to {
  opacity: 0;
}

@media (width <= 992px) {
  .list-toolbar {
    flex-direction: column;
  }

  .toolbar-filters {
    width: 100%;
  }

  .toolbar-actions {
    width: 100%;
  }
}

@media (width <= 768px) {
  .instance-grid {
    grid-template-columns: 1fr;
  }
}
</style>
