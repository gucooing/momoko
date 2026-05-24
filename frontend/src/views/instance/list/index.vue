<template>
  <div class="instance-list-page">
    <OverviewStats :cards="summaryCards" />

    <BaseCard title="实例列表" title-icon="HOutline:ServerStackIcon">
      <div v-loading="loading">
        <div class="list-toolbar">
          <div class="toolbar-filters">
            <el-input
              :model-value="queryForm.keyword"
              placeholder="搜索实例名 / 标签"
              clearable
              style="width: 180px"
              @update:model-value="(v: string) => queryForm.keyword = v"
              @keyup.enter="applyFilters"
            />
            <el-select
              :model-value="queryForm.type"
              placeholder="类型"
              clearable
              style="width: 130px"
              @update:model-value="(v: string) => queryForm.type = v"
            >
              <el-option v-for="item in typeOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-select
              :model-value="queryForm.status"
              placeholder="状态"
              clearable
              style="width: 110px"
              @update:model-value="(v: any) => queryForm.status = v"
            >
              <el-option v-for="item in statusOptions" :key="item.value" :label="item.label" :value="item.value" />
            </el-select>
            <el-button type="primary" size="small" @click="applyFilters">搜索</el-button>
            <el-button size="small" @click="resetFilters">重置</el-button>
          </div>

          <div class="toolbar-actions">
            <span v-if="selectedIds.length" class="selection-text">已选 {{ selectedIds.length }} 项</span>
            <el-button
              type="primary"
              :icon="menuStore.iconComponents['HOutline:PlusCircleIcon']"
              @click="handleOpenCreateEditor"
            >
              新建
            </el-button>
            <el-button
              :icon="menuStore.iconComponents['HOutline:PlayIcon']"
              :disabled="!canBatchStart"
              @click="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_RUNNING)"
            >
              批量启动
            </el-button>
            <el-button
              :icon="menuStore.iconComponents['HOutline:StopIcon']"
              :disabled="!canBatchStop"
              @click="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_STOPPED)"
            >
              批量停止
            </el-button>
            <el-button
              type="danger"
              :icon="menuStore.iconComponents['HOutline:TrashIcon']"
              :disabled="!canBatchDelete"
              @click="handleBatchDelete"
            >
              批量删除
            </el-button>
            <el-button link @click="toggleCurrentPageSelection">
              {{ isCurrentPageAllSelected ? '取消全选' : '全选当前页' }}
            </el-button>
            <el-button link :icon="menuStore.iconComponents['HOutline:ArrowPathRoundedSquareIcon']" @click="handleRefreshStatus">
              刷新
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

          <el-empty v-else key="instance-empty" description="暂无匹配实例" />
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
    query: { tabTitle: item.name },
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
      onConfirm: async () => { await executeChangeInstanceStatus(id, targetStatus) },
    })
    return
  }
  await executeChangeInstanceStatus(id, targetStatus)
}

const handleBatchDelete = async () => {
  Dialog.confirm({
    title: '确认批量删除实例',
    content: '确定要删除当前选中的实例吗？删除后不可恢复。',
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
    if (current.userId !== currentUserId.value) return
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
