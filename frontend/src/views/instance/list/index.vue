<!-- 应用列表：页头（标题 + 共 N 个 · N 个运行中 + 应用类型 / 新建应用）+ 状态 chips（带数量，后端统计）+ 搜索 + 类型 + 卡/表切换
     + InstanceCard 卡片流 / DataTable 表视图（行内小图标 + ⋯）+ Pagination + 底部深色批量条（启动 / 停止 / 重启 / 删除）
     + 手机悬浮「新建应用」+ 受控 InstanceEditor。页面为 store(useInstanceListStore) 的薄壳。 -->
<template>
  <div class="inst-page">
    <PageHeader :title="t('instance.ui.listTitle')">
      <template #description>
        {{ t('instance.ui.listCount', { n: statusCounts.all }) }} ·
        <span class="inst-page__running">{{ t('instance.ui.runningCount', { n: statusCounts.running }) }}</span>
      </template>
      <template #actions>
        <UButton
          color="neutral"
          variant="ghost"
          icon="i-lucide-rotate-cw"
          class="inst-ico-btn"
          :aria-label="t('common.refresh')"
          :title="t('common.refresh')"
          :loading="loading"
          @click="handleRefreshStatus"
        />
        <template v-if="canSeeTypes">
          <UButton
            v-if="menuStore.isMobile"
            color="neutral"
            variant="ghost"
            icon="i-lucide-layers"
            class="inst-ico-btn"
            :aria-label="t('instance.ui.appTypes')"
            :title="t('instance.ui.appTypes')"
            to="/instance/type"
          />
          <UButton v-else color="neutral" variant="outline" icon="i-lucide-layers" to="/instance/type">
            {{ t('instance.ui.appTypes') }}
          </UButton>
        </template>
        <UButton v-if="!menuStore.isMobile" color="primary" icon="i-lucide-plus" @click="handleOpenCreateEditor">
          {{ t('instance.ui.newApp') }}
        </UButton>
      </template>
    </PageHeader>

    <!-- 工具条：状态 chips（带数量）+ 搜索 + 类型 + 视图切换 -->
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
          {{ opt.label }} <span class="inst-chip__n m-num">{{ opt.count }}</span>
        </button>
      </div>
      <span class="inst-tools__spacer" />
      <label class="inst-search">
        <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" />
        <input
          v-model="queryForm.keyword"
          type="search"
          :placeholder="t('instance.ui.searchApps')"
          :aria-label="t('instance.ui.searchApps')"
          @keyup.enter="applyFilters"
        />
      </label>
      <div class="inst-type">
        <AppSelect v-model="queryForm.type" :options="typeFilterOptions" />
      </div>
      <div v-if="!menuStore.isMobile" class="seg" :aria-label="t('instance.ui.viewMode')">
        <button
          type="button"
          class="seg__btn"
          :class="{ 'is-active': viewMode === 'card' }"
          :aria-label="t('system.common.card')"
          :aria-pressed="viewMode === 'card'"
          @click="viewMode = 'card'"
        >
          <component :is="menuStore.iconComponents['HOutline:Squares2X2Icon']" />
        </button>
        <button
          type="button"
          class="seg__btn"
          :class="{ 'is-active': viewMode === 'table' }"
          :aria-label="t('system.common.table')"
          :aria-pressed="viewMode === 'table'"
          @click="viewMode = 'table'"
        >
          <component :is="menuStore.iconComponents['HOutline:Bars3Icon']" />
        </button>
      </div>
    </div>

    <!-- 主体：卡/表切换（桌面）；手机强制卡片 -->
    <div class="inst-page__body">
      <div v-if="isCardView && pagedInstances.length" class="inst-page__bar">
        <span class="inst-page__bar-hint m-num">{{ t('instance.ui.filteredCount', { n: pagination.total }) }}</span>
        <button type="button" class="inst-page__selall" @click="toggleCurrentPageSelection">
          {{ isCurrentPageAllSelected ? t('instance.deselectAll') : t('instance.selectCurrentPage') }}
        </button>
      </div>

      <!-- 卡片流 -->
      <template v-if="isCardView">
        <div v-if="loading && !pagedInstances.length" class="inst-grid">
          <div v-for="i in 6" :key="i" class="inst-skeleton" />
        </div>
        <EmptyState
          v-else-if="!pagedInstances.length"
          icon="HOutline:ServerStackIcon"
          :title="hasFilter ? t('instance.noMatchedInstances') : t('instance.ui.emptyTitle')"
          :description="hasFilter ? t('instance.ui.emptyFilteredDesc') : t('instance.ui.emptyDesc')"
        >
          <template #action>
            <UButton color="primary" variant="soft" icon="i-lucide-plus" @click="handleOpenCreateEditor">
              {{ t('instance.ui.newApp') }}
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
            :now="now"
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
        :empty-text="hasFilter ? t('instance.noMatchedInstances') : t('instance.ui.emptyTitle')"
      >
        <template #cell-name="{ row }">
          <span class="inst-cell-name">
            <span class="inst-cell-name__av m-sticker" :class="`m-sticker--${toneOf(String(row.id))}`">
              {{ initialOf(String(row.name || '')) }}
            </span>
            <span class="inst-cell-name__text">
              <RouterLink class="inst-cell-name__n" :to="consoleRoute(row)">{{ row.name }}</RouterLink>
              <span class="inst-cell-name__r">{{ row.remark || t('instance.noRemark') }}</span>
            </span>
          </span>
        </template>
        <template #cell-type="{ row }">{{ typeText(row) }}</template>
        <template #cell-status="{ row }">
          <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
        </template>
        <template #cell-uptime="{ row }">
          <span class="m-num">{{ uptimeCell(row) }}</span>
        </template>
        <template #cell-instancePath="{ row }">
          <span class="m-mono inst-cell-mono" :title="String(row.instancePath || '')">{{ row.instancePath || '—' }}</span>
        </template>
        <template #cell-operation="{ row }">
          <span class="inst-row-acts">
            <AppIconButton
              icon="HOutline:CommandLineIcon"
              :label="t('instance.console')"
              :size="18"
              :box="36"
              @click="openById(String(row.id))"
            />
            <AppIconButton
              :icon="isRunningRow(row) ? 'HOutline:StopIcon' : 'HOutline:PlayIcon'"
              :label="isRunningRow(row) ? t('instance.stop') : t('instance.start')"
              :class="isRunningRow(row) ? 'is-stop' : 'is-start'"
              :size="18"
              :box="36"
              @click="togglePowerById(String(row.id))"
            />
            <ActionMenu :items="rowActionsFor(row)" :label="t('instance.moreActions')" @select="(key) => onRowAction(key, row)" />
          </span>
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

    <!-- 手机：悬浮新建（有勾选时让位给批量条） -->
    <div v-if="menuStore.isMobile && !selectedIds.length" class="inst-fab">
      <UButton color="primary" size="xl" icon="i-lucide-plus" class="inst-fab__btn" @click="handleOpenCreateEditor">
        {{ t('instance.ui.newApp') }}
      </UButton>
    </div>

    <!-- 批量操作条：选中后浮在底部 -->
    <Transition name="batch-pop">
      <div v-if="selectedIds.length" class="inst-batch" role="toolbar" :aria-label="t('instance.ui.bulkActions')">
        <span class="inst-batch__count">{{ t('instance.ui.selectedApps', { n: selectedIds.length }) }}</span>
        <button
          type="button"
          class="inst-batch__btn"
          :disabled="!canBatchStart"
          @click="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_RUNNING)"
        >
          <component :is="menuStore.iconComponents['HOutline:PlayIcon']" />{{ t('instance.start') }}
        </button>
        <button
          type="button"
          class="inst-batch__btn"
          :disabled="!canBatchStop"
          @click="handleBatchChangeStatus(InstanceStatus.INSTANCE_STATUS_STOPPED)"
        >
          <component :is="menuStore.iconComponents['HOutline:StopIcon']" />{{ t('instance.stop') }}
        </button>
        <button type="button" class="inst-batch__btn" :disabled="!canBatchRestart" @click="handleBatchRestart">
          <component :is="menuStore.iconComponents['HOutline:ArrowPathIcon']" />{{ t('instance.restart') }}
        </button>
        <button type="button" class="inst-batch__btn is-danger" :disabled="!canBatchDelete" @click="handleBatchDelete">
          <component :is="menuStore.iconComponents['HOutline:TrashIcon']" />{{ t('common.delete') }}
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
      allow-run
      :running="editingRunning"
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
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import InstanceCard from '@/views/instance/list/instanceCard.vue'
import InstanceEditor from '@/views/instance/list/instanceEditor.vue'
import { fmtDuration, initialOf, toneOf } from '@/views/instance/shared'

defineOptions({ name: 'ListView' })

type SwitchStatus = typeof InstanceStatus.INSTANCE_STATUS_RUNNING | typeof InstanceStatus.INSTANCE_STATUS_STOPPED
type MoreAction = 'forceRestart' | 'restart' | 'delete' | 'fileManager' | 'maintenanceOn' | 'maintenanceOff'

const menuStore = useMenuStore()
const router = useRouter()
const instanceListStore = useInstanceListStore()
const userStore = useUserStore()
const accessible = useAccessibleRoutes()
const { t } = useI18n()

const INSTANCE_PAGE_SIZES = [9, 18, 27]
const viewMode = ref<'card' | 'table'>('card')
const isCardView = computed(() => menuStore.isMobile || viewMode.value === 'card')
const canSeeTypes = computed(() => accessible.value.has('/instance/type'))

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
  statusCounts,
  selectedIdSet,
  typeOptions,
  statusOptions,
  pagedInstances,
  isCurrentPageAllSelected,
  canBatchStart,
  canBatchStop,
  canBatchRestart,
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
  batchRestart,
  batchDeleteInstances,
  changeInstanceStatus,
  deleteInstances,
  restartInstance,
  setMaintenance,
  endMaintenanceAndStart,
  findInstanceById,
  resolveTypeLabel,
  initialize,
} = instanceListStore

// 运行时长每 30 秒刷新一次
const now = ref(Date.now())
let clockTimer: ReturnType<typeof setInterval> | undefined

const hasFilter = computed(() => !!(queryForm.value.keyword.trim() || queryForm.value.type || queryForm.value.status))

// —— 筛选 ——
const typeFilterOptions = computed(() => [{ label: t('instance.allTypes'), value: '' }, ...typeOptions.value])
const statusChips = computed(() => {
  const c = statusCounts.value
  const label = (s: InstanceStatus) => statusOptions.value.find((o) => o.value === s)?.label || ''
  return [
    { label: t('instance.ui.chipAll'), value: '' as InstanceStatus | '', count: c.all },
    { label: label(InstanceStatus.INSTANCE_STATUS_RUNNING), value: InstanceStatus.INSTANCE_STATUS_RUNNING, count: c.running },
    { label: label(InstanceStatus.INSTANCE_STATUS_STOPPED), value: InstanceStatus.INSTANCE_STATUS_STOPPED, count: c.stopped },
    {
      label: label(InstanceStatus.INSTANCE_STATUS_MAINTENANCE),
      value: InstanceStatus.INSTANCE_STATUS_MAINTENANCE,
      count: c.maintenance,
    },
  ]
})
const setStatusFilter = (value: InstanceStatus | '') => {
  queryForm.value.status = value
  applyFilters()
}
// 输入即搜索（防抖）、切换类型即筛选
const debouncedApply = useDebounceFn(() => applyFilters(), 350)
watch(() => queryForm.value.keyword, () => debouncedApply())
watch(() => queryForm.value.type, () => applyFilters())

// —— 表视图 ——
const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('instance.ui.name'), minWidth: 220 },
  { key: 'type', title: t('common.type'), width: 130 },
  { key: 'status', title: t('common.status'), width: 110 },
  { key: 'uptime', title: t('instance.ui.colUptime'), width: 170 },
  { key: 'instancePath', title: t('instance.ui.workdir'), minWidth: 180 },
  { key: 'operation', title: t('common.operation'), width: 140, align: 'right' },
])

type PillVariant = 'success' | 'neutral' | 'warning' | 'error'
const statusVariantMap: Record<InstanceStatus, PillVariant> = {
  [InstanceStatus.INSTANCE_STATUS_RUNNING]: 'success',
  [InstanceStatus.INSTANCE_STATUS_STOPPED]: 'neutral',
  [InstanceStatus.INSTANCE_STATUS_MAINTENANCE]: 'warning',
  [InstanceStatus.INSTANCE_STATUS_UNSPECIFIED]: 'neutral',
  [InstanceStatus.UNRECOGNIZED]: 'error',
}
const statusVariant = (status: unknown) => statusVariantMap[status as InstanceStatus] || 'neutral'
const statusLabel = (status: unknown) => {
  const meta = statusMeta[status as InstanceStatus]
  return meta ? t(meta.labelKey) : t('instance.unknownStatus')
}
const typeText = (row: Record<string, unknown>) =>
  row.type ? resolveTypeLabel(String(row.type)) : t('instance.ui.uncategorized')
const isRunningRow = (row: Record<string, unknown>) => row.status === InstanceStatus.INSTANCE_STATUS_RUNNING
const fmtShort = (value: unknown) => {
  if (!value) return ''
  const d = dayjs(value as string | Date)
  return d.isValid() ? d.format('MM-DD HH:mm') : ''
}
const uptimeCell = (row: Record<string, unknown>) => {
  const inst = row as unknown as InstanceRecord
  if (inst.status === InstanceStatus.INSTANCE_STATUS_RUNNING && inst.startTime) {
    return fmtDuration(t, now.value - dayjs(inst.startTime as unknown as string).valueOf())
  }
  if (inst.status === InstanceStatus.INSTANCE_STATUS_MAINTENANCE && inst.maintenanceTime) {
    return `${t('instance.ui.upMaintenance')} ${fmtShort(inst.maintenanceTime)}`
  }
  if (inst.stopTime) return `${t('instance.ui.upStoppedAt')} ${fmtShort(inst.stopTime)}`
  return t('instance.ui.neverRun')
}
const consoleRoute = (row: Record<string, unknown>) => ({
  path: `/instance/app/${String(row.id)}`,
  query: { tabTitle: String(row.name || '') },
})

const rowActionsFor = (row: Record<string, unknown>): ActionMenuItem[] => {
  const maintenance = row.maintenance === true
  return [
    { key: 'config', label: t('instance.ui.editApp'), icon: 'HOutline:PencilSquareIcon' },
    { key: 'fileManager', label: t('instance.ui.appFiles'), icon: 'HOutline:FolderIcon' },
    { key: 'restart', label: t('instance.restart'), icon: 'HOutline:ArrowPathIcon', disabled: maintenance },
    { key: 'forceRestart', label: t('instance.forceRestart'), icon: 'HOutline:BoltIcon', disabled: maintenance },
    maintenance
      ? { key: 'maintenanceOff', label: t('instance.ui.endMaintenance'), icon: 'HOutline:CheckBadgeIcon' }
      : { key: 'maintenanceOn', label: t('instance.ui.enterMaintenance'), icon: 'HOutline:WrenchScrewdriverIcon' },
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
  if (key === 'config') handleOpenEditEditor(inst.id)
  else handleMoreAction(inst.id, key as MoreAction)
}

const openById = (id: string) => {
  const inst = findInstanceById(id)
  if (inst) openInstanceConsole(inst)
}
const togglePowerById = (id: string) => {
  const inst = findInstanceById(id)
  if (!inst) return
  handleChangeInstanceStatus(
    id,
    inst.status === InstanceStatus.INSTANCE_STATUS_RUNNING
      ? InstanceStatus.INSTANCE_STATUS_STOPPED
      : InstanceStatus.INSTANCE_STATUS_RUNNING,
  )
}

// —— 导航 ——
const openInstanceConsole = (item: InstanceRecord) => {
  router.push({ path: `/instance/app/${item.id}`, query: { tabTitle: item.name } })
}
// 实例文件 API 以实例目录为根：进详情页的「文件」标签
const openInstanceFileManager = (item: Pick<InstanceRecord, 'id' | 'name'>) => {
  router.push({ path: `/instance/app/${item.id}`, query: { tab: 'files', tabTitle: item.name } })
}

// —— 编辑器 ——
const editingRunning = computed(() => {
  const id = instanceEditorForm.value.id
  return !!id && findInstanceById(id)?.status === InstanceStatus.INSTANCE_STATUS_RUNNING
})
const handleOpenCreateEditor = () => openCreateEditor()

const handleOpenEditEditor = async (instanceId: string) => {
  try {
    await openEditEditor(instanceId)
  } catch (error) {
    showRequestError(error, t('instance.loadConfigFailed'))
  }
}

const handleEditorClose = () => closeInstanceEditor()

const handleEditorSubmit = async (form: InstanceEditorFormValue, options?: { run: boolean }) => {
  const mode = instanceEditorMode.value
  const wasRunning = editingRunning.value
  try {
    instanceEditorForm.value = { ...form }
    const saved = await submitInstanceEditor()
    feedback.success(mode === 'create' ? t('instance.createSuccess') : t('instance.configSaveSuccess'))
    if (!options?.run || !saved?.id) return
    // 保存并启动 / 保存并重启
    const ok =
      wasRunning && mode === 'edit'
        ? await restartInstance(saved.id)
        : await changeInstanceStatus(saved.id, InstanceStatus.INSTANCE_STATUS_RUNNING)
    if (ok) feedback.success(wasRunning && mode === 'edit' ? t('instance.ui.restarted') : t('instance.instanceStarted'))
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

const executeBatchChangeStatus = async (targetStatus: SwitchStatus) => {
  if (!selectedIds.value.length) return
  const { successCount, failedCount } = await batchChangeStatus(targetStatus)
  if (!successCount && failedCount) return
  feedback.success(
    targetStatus === InstanceStatus.INSTANCE_STATUS_RUNNING
      ? t('instance.batchStartSuccess', { success: successCount, failed: failedSuffix(failedCount) })
      : t('instance.batchStopSuccess', { success: successCount, failed: failedSuffix(failedCount) }),
  )
}

const handleBatchChangeStatus = async (targetStatus: SwitchStatus) => {
  if (!selectedIds.value.length) return
  if (targetStatus === InstanceStatus.INSTANCE_STATUS_STOPPED) {
    Dialog.confirm({
      title: t('instance.confirmBatchStopTitle'),
      content: t('instance.confirmBatchStopContent', { count: selectedIds.value.length }),
      cancelText: t('common.cancel'),
      confirmText: t('instance.confirmStop'),
      onConfirm: async () => {
        await executeBatchChangeStatus(targetStatus)
      },
    })
    return
  }
  await executeBatchChangeStatus(targetStatus)
}

const handleBatchRestart = () => {
  if (!selectedIds.value.length) return
  Dialog.confirm({
    title: t('instance.ui.confirmBatchRestartTitle'),
    content: t('instance.ui.confirmBatchRestartContent', { count: selectedIds.value.length }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.confirmRestart'),
    onConfirm: async () => {
      const { successCount, failedCount } = await batchRestart()
      if (!successCount && failedCount) return
      feedback.success(t('instance.ui.batchRestartSuccess', { success: successCount, failed: failedSuffix(failedCount) }))
    },
  })
}

const executeChangeInstanceStatus = async (id: string, targetStatus: SwitchStatus) => {
  const success = await changeInstanceStatus(id, targetStatus)
  if (!success) return
  feedback.success(
    targetStatus === InstanceStatus.INSTANCE_STATUS_RUNNING ? t('instance.instanceStarted') : t('instance.instanceStopped'),
  )
}

const handleChangeInstanceStatus = async (id: string, targetStatus: SwitchStatus) => {
  const current = findInstanceById(id)
  if (!current) return
  if (targetStatus === InstanceStatus.INSTANCE_STATUS_RUNNING && current.maintenance) {
    Dialog.confirm({
      title: t('instance.ui.endMaintenanceStartTitle'),
      content: t('instance.ui.endMaintenanceStartContent', { name: current.name }),
      cancelText: t('common.cancel'),
      confirmText: t('instance.ui.endAndStart'),
      onConfirm: async () => {
        try {
          if (await endMaintenanceAndStart(id)) feedback.success(t('instance.instanceStarted'))
        } catch (error) {
          showRequestError(error, t('instance.configSaveFailed'))
        }
      },
    })
    return
  }
  if (targetStatus === InstanceStatus.INSTANCE_STATUS_STOPPED) {
    Dialog.confirm({
      title: t('instance.confirmStopInstanceTitle'),
      content: t('instance.confirmStopInstanceContent', { name: current.name }),
      cancelText: t('common.cancel'),
      confirmText: t('instance.confirmStop'),
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

const toggleMaintenance = (current: InstanceRecord, on: boolean) => {
  const run = async () => {
    try {
      await setMaintenance(current.id, on)
      feedback.success(on ? t('instance.ui.maintenanceOnDone') : t('instance.ui.maintenanceOffDone'))
    } catch (error) {
      showRequestError(error, t('instance.configSaveFailed'))
    }
  }
  if (!on) {
    void run()
    return
  }
  Dialog.confirm({
    title: t('instance.ui.enterMaintenance'),
    content: t('instance.ui.maintenanceOnContent', { name: current.name }),
    cancelText: t('common.cancel'),
    confirmText: t('instance.ui.enterMaintenance'),
    onConfirm: run,
  })
}

const handleMoreAction = async (id: string, action: MoreAction) => {
  const current = findInstanceById(id)
  if (!current) return

  switch (action) {
    case 'restart':
      Dialog.confirm({
        title: t('instance.confirmRestartInstanceTitle'),
        content: t('instance.confirmRestartInstanceContent', { name: current.name }),
        cancelText: t('common.cancel'),
        confirmText: t('instance.confirmRestart'),
        onConfirm: async () => {
          if (await restartInstance(id)) feedback.success(t('instance.ui.restarted'))
        },
      })
      return
    case 'forceRestart':
      Dialog.confirm({
        title: t('instance.confirmForceRestartInstanceTitle'),
        content: t('instance.confirmForceRestartInstanceContent', { name: current.name }),
        cancelText: t('common.cancel'),
        confirmText: t('instance.confirmRestart'),
        onConfirm: async () => {
          if (await restartInstance(id)) feedback.success(t('instance.forceRestarting'))
        },
      })
      return
    case 'maintenanceOn':
    case 'maintenanceOff':
      toggleMaintenance(current, action === 'maintenanceOn')
      return
    case 'delete':
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
    case 'fileManager':
      openInstanceFileManager(current)
  }
}

onMounted(() => {
  void initialize()
  clockTimer = setInterval(() => (now.value = Date.now()), 30000)
})
onBeforeUnmount(() => {
  if (clockTimer) clearInterval(clockTimer)
})
</script>

<style scoped lang="scss">
.inst-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}
.inst-page__running {
  font-weight: 700;
  color: var(--m-ok);
}
.inst-page .inst-ico-btn {
  width: 40px;
  padding-inline: 0;
  justify-content: center;
}
@media (width <= 768px) {
  .inst-page .inst-ico-btn {
    width: 44px;
  }
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
  display: inline-flex;
  align-items: center;
  gap: 6px;
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
.inst-chip__n {
  font-size: 0.75rem;
  opacity: 0.7;
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
/* 与搜索胶囊同高 */
.inst-type :deep(.app-sel__trigger) {
  height: 40px;
  border-radius: 999px;
  padding-inline: 14px 10px;
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
  margin-bottom: -4px;
}
.inst-page__bar-hint {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
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

/* 表格单元格 */
.inst-cell-name {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}
.inst-cell-name__av {
  width: 34px;
  height: 34px;
  border-radius: 11px;
  font-size: 0.9375rem;
  flex: none;
}
.inst-cell-name__text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.inst-cell-name__n {
  font-weight: 800;
  color: var(--el-text-color-primary);
  text-decoration: none;
  overflow: hidden;
  text-overflow: ellipsis;
}
.inst-cell-name__n:hover {
  color: var(--m-pInk);
}
.inst-cell-name__r {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
}
.inst-cell-mono {
  font-size: 0.78rem;
}
.inst-row-acts {
  display: inline-flex;
  align-items: center;
  justify-content: flex-end;
  gap: 2px;
}
.inst-row-acts :deep(.is-stop) {
  color: var(--m-bad);
}
.inst-row-acts :deep(.is-start) {
  color: var(--m-ok);
}

/* 手机悬浮新建 */
.inst-fab {
  position: sticky;
  bottom: 0;
  display: flex;
  justify-content: flex-end;
  pointer-events: none;
  z-index: 5;
}
.inst-fab__btn {
  pointer-events: auto;
  min-height: 56px;
  padding: 0 22px;
  border-radius: var(--app-radius-pill);
  font-weight: 700;
  box-shadow: var(--app-shadow-brand);
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
.inst-batch::-webkit-scrollbar {
  display: none;
}
.inst-batch__count {
  margin-right: 10px;
  font-size: 0.875rem;
  font-weight: 800;
  white-space: nowrap;
}
.inst-batch__btn {
  display: inline-flex;
  flex: none;
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
.batch-pop-enter-active,
.batch-pop-leave-active {
  transition: opacity 0.18s, transform 0.18s;
}
.batch-pop-enter-from,
.batch-pop-leave-to {
  opacity: 0;
  transform: translateY(12px);
}

@media (width < 992px) {
  .inst-batch {
    bottom: calc(78px + env(safe-area-inset-bottom));
  }
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
    min-height: 44px;
  }
  .inst-type {
    width: 128px;
  }
  .inst-type :deep(.app-sel__trigger) {
    height: 44px;
  }
  .inst-chip {
    min-height: 44px;
  }
  .inst-page__selall {
    min-height: 44px;
    padding-inline: 4px;
    margin-right: -4px;
  }
  /* 手机：图标在上、文字在下，整条放进 358px 宽，不用横向滚动 */
  .inst-batch {
    padding: 4px 4px 4px 14px;
  }
  .inst-batch__count {
    margin-right: 4px;
  }
  .inst-batch__btn {
    flex-direction: column;
    justify-content: center;
    gap: 2px;
    min-width: 48px;
    min-height: 48px;
    padding: 0 6px;
    font-size: 0.6875rem;
  }
  .inst-batch__btn.is-icon {
    width: 44px;
    min-width: 44px;
  }
  .inst-page {
    padding-bottom: 72px;
  }
}
</style>
