<template>
  <div class="instance-list-page">
    <div class="instance-surface">
      <section class="instance-section instance-section--overview">
        <OverviewStats :cards="summaryCards" />
      </section>

      <section class="instance-section instance-section--filters">
        <InstanceFilters
          v-model="queryForm"
          :type-options="typeOptions"
          :status-options="statusOptions"
          @search="applyFilters"
          @reset="resetFilters"
        />
      </section>

      <section class="instance-section instance-section--workspace" v-loading="loading">
        <InstanceToolbar
          :selected-count="selectedIds.length"
          :can-batch-start="canBatchStart"
          :can-batch-stop="canBatchStop"
          :can-batch-delete="canBatchDelete"
          :is-current-page-all-selected="isCurrentPageAllSelected"
          @create="handleOpenCreateEditor"
          @batch-start="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_RUNNING)"
          @batch-stop="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_STOPPED)"
          @batch-delete="handleBatchDelete"
          @toggle-current-page="toggleCurrentPageSelection"
          @refresh="handleRefreshStatus"
        />

        <Transition name="instance-page-switch" mode="out-in">
          <div v-if="pagedInstances.length" :key="gridTransitionKey" class="instance-grid">
            <div
              v-for="(item, index) in pagedInstances"
              :key="item.id"
              class="instance-grid-item"
              :style="{ '--instance-card-order': index }"
            >
              <InstanceCard
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
          </div>

          <el-empty v-else key="instance-empty" class="instance-empty" description="暂无匹配实例" />
        </Transition>

        <div class="pagination-container instance-pagination">
          <TablePagination
            v-model:current-page="pagination.page"
            v-model:page-size="pagination.pageSize"
            :total="pagination.total"
            :is-mobile="menuStore.isMobile"
            :page-sizes="INSTANCE_PAGE_SIZES"
            @change="handlePageChange"
          />
        </div>
      </section>
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
import InstanceFilters from '@/views/instance/list/instanceFilters.vue'
import InstanceToolbar from '@/views/instance/list/instanceToolbar.vue'
import OverviewStats from '@/views/instance/list/overviewStats.vue'

defineOptions({ name: 'ListView' })

const menuStore = useMenuStore()
const router = useRouter()
const instanceListStore = useInstanceListStore()
const userStore = useUserStore()
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
    query: {
      tabTitle: item.name,
    },
  })
}

const openInstanceFileManager = (
  item: Pick<InstanceRecord, 'id' | 'name' | 'status' | 'instancePath'>,
) => {
  router.push({
    path: `/instance/files/${item.id}`,
    query: {
      tabTitle: `${item.name} 文件`,
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
    showRequestError(error, '加载实例配置失败')
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
    ElMessage.success(mode === 'create' ? '实例创建成功' : '实例配置保存成功')
  } catch (error) {
    showRequestError(error, mode === 'create' ? '实例创建失败' : '实例配置保存失败')
  }
}

const handleRefreshStatus = async () => {
  await refreshStatus()
  ElMessage.success('实例状态已刷新')
}

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
      ? `已批量启动 ${successCount} 个实例${failedCount ? `，失败 ${failedCount} 个` : ''}`
      : `已批量停止 ${successCount} 个实例${failedCount ? `，失败 ${failedCount} 个` : ''}`,
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
      title: '确认批量停止实例',
      content: `确定要停止当前选中的 ${selectedIds.value.length} 个实例吗？`,
      cancelText: '取消',
      confirmText: '确认停止',
      onConfirm: async () => {
        await executeBatchChangeStatus(targetStatus)
      },
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
    targetStatus === InstanceStatus.INSTANCE_STATUS_RUNNING ? '实例已启动' : '实例已停止',
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
      title: '确认停止实例',
      content: `确定要停止实例「${current.name}」吗？`,
      cancelText: '取消',
      confirmText: '确认停止',
      onConfirm: async () => {
        await executeChangeInstanceStatus(id, targetStatus)
      },
    })
    return
  }

  await executeChangeInstanceStatus(id, targetStatus)
}

const handleBatchDelete = async () => {
  Dialog.confirm({
    title: '确认批量删除实例',
    content: `确定要删除当前选中的实例吗？删除后不可恢复。`,
    cancelText: '取消',
    confirmText: '确认删除',
    onConfirm: async () => {
      const { successCount, failedCount } = await batchDeleteInstances()
      if (!successCount && failedCount) return

      ElMessage.success(`已批量删除 ${successCount} 个实例${failedCount ? `，失败 ${failedCount} 个` : ''}`)
    },
  })
}

const handleMoreAction = async (id: string, action: 'forceRestart' | 'delete' | 'fileManager') => {
  const current = findInstanceById(id)
  if (!current) return

  if (action === 'forceRestart') {
    Dialog.confirm({
      title: '确认强制重启实例',
      content: `确定要强制重启实例「${current.name}」吗？`,
      cancelText: '取消',
      confirmText: '确认重启',
      onConfirm: async () => {
        const success = await restartInstance(id)
        if (!success) return
        ElMessage.success('实例强制重启中')
      },
    })
    return
  }

  if (action === 'delete') {
    if (current.userId !== currentUserId.value) {
      return
    }

    Dialog.confirm({
      title: '确认删除实例',
      content: `确定要删除实例「${current.name}」吗？删除后不可恢复。`,
      cancelText: '取消',
      confirmText: '确认删除',
      onConfirm: async () => {
        const { successCount, failedCount } = await deleteInstances([id])
        if (!successCount && failedCount) return

        ElMessage.success(`${current.name} 已删除${failedCount ? `（失败 ${failedCount} 个）` : ''}`)
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
  flex: 1;
  flex-direction: column;
  min-height: 100%;
  margin: -1rem;
  padding: 1rem;
  background: var(--el-bg-color-page);
  box-sizing: border-box;
}

.instance-surface {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.instance-section {
  padding: 1rem 1.05rem;
  border-radius: 1rem;
  background: var(--el-bg-color);
  box-shadow: 0 1px 2px rgb(15 23 42 / 4%);
}

.instance-grid {
  display: grid;
  grid-auto-flow: row;
  grid-template-columns: repeat(auto-fill, minmax(360px, 1fr));
  gap: 0.9rem;
  margin-top: 0.9rem;
}

.instance-grid-item {
  min-width: 0;
}

.instance-empty {
  padding-block: 1.25rem;
}

.instance-pagination {
  padding-top: 0.6rem;
}

.instance-page-switch-enter-active,
.instance-page-switch-leave-active {
  transition:
    opacity 0.24s ease,
    transform 0.24s ease;
}

.instance-page-switch-enter-active .instance-grid-item {
  animation: instance-card-rise 0.36s cubic-bezier(0.22, 1, 0.36, 1) both;
  animation-delay: calc(var(--instance-card-order) * 42ms);
}

.instance-page-switch-enter-from,
.instance-page-switch-leave-to {
  opacity: 0;
  transform: translateY(12px);
}

@keyframes instance-card-rise {
  from {
    opacity: 0;
    transform: translateY(16px) scale(0.985);
  }

  to {
    opacity: 1;
    transform: translateY(0) scale(1);
  }
}

@media (prefers-reduced-motion: reduce) {
  .instance-page-switch-enter-active,
  .instance-page-switch-leave-active {
    transition-duration: 0.01ms;
  }

  .instance-page-switch-enter-active .instance-grid-item {
    animation: none;
  }
}

@media (width <= 768px) {
  .instance-list-page {
    margin: -1rem;
    padding: 1rem;
  }

  .instance-section {
    padding: 0.92rem;
  }

  .instance-grid {
    grid-template-columns: 1fr;
  }
}
</style>
