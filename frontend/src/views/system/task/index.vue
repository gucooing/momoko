<!-- 任务管理（重写 · P1 只读列表 + 行内操作）：PageHeader + FilterBar + DataTable / 移动卡片 + Pagination。
     通用任务管理器（一次性/定时/常驻）；2.5s 静默轮询刷新；行内 取消/重试/删除（删除走 Dialog.info 确认）。保留 listTasks/cancel/retry/deleteTask 契约。 -->
<template>
  <div class="task-page">
    <PageHeader :title="t('taskManager.pageTitle')" :description="t('taskManager.pageDesc')">
      <template #actions>
        <UButton color="neutral" variant="soft" icon="i-lucide-rotate-cw" @click="reload">
          {{ t('taskManager.refresh') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="reload" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('taskManager.searchPlaceholder') }}</label>
          <input
            v-model="keywords"
            class="app-input"
            :placeholder="t('taskManager.searchPlaceholder')"
            @keyup.enter="reload"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('taskManager.status') }}</label>
          <AppSelect v-model="statusFilter" :options="statusOptions" />
        </div>
      </template>
    </FilterBar>

    <div class="task-page__body">
      <div class="task-page__bar">
        <span class="task-page__bar-hint">{{ t('system.common.total', { total }) }}</span>
      </div>

      <!-- 桌面：表格 -->
      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="items"
        :loading="loading"
        :empty-text="t('taskManager.empty')"
      >
        <template #cell-title="{ row }">
          <div class="task-title">
            <span class="task-title__main">{{ row.title || row.type }}</span>
            <span v-if="row.message" class="task-title__msg">{{ row.message }}</span>
            <span v-if="row.error" class="task-title__err">{{ row.error }}</span>
          </div>
        </template>
        <template #cell-kind="{ row }">
          <StatusPill variant="neutral" :dot="false" :label="kindLabel(String(row.kind))" />
        </template>
        <template #cell-status="{ row }">
          <StatusPill :variant="statusVariant(String(row.status))" :label="statusLabel(String(row.status))" />
        </template>
        <template #cell-progress="{ row }">
          <div v-if="Number(row.total) > 0" class="task-progress">
            <div class="task-progress__track">
              <div
                class="task-progress__fill"
                :class="`is-${progressTone(String(row.status))}`"
                :style="{ width: `${percent(row)}%` }"
              />
            </div>
            <span class="task-progress__pct">{{ percent(row) }}%</span>
          </div>
          <span v-else class="text-dim">{{ Number(row.finished) || 0 }}</span>
        </template>
        <template #cell-creator="{ row }">{{ row.userName || t('taskManager.system') }}</template>
        <template #cell-createTime="{ row }">{{ formatTime(row.createTime) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu
            v-if="hasRowActions(String(row.status))"
            :items="rowActions(String(row.status))"
            @select="(key) => onRowAction(key, row)"
          />
          <span v-else class="text-dim">—</span>
        </template>
      </DataTable>

      <!-- 移动：卡片 -->
      <template v-else>
        <div v-if="loading" class="task-cards">
          <div v-for="i in 6" :key="i" class="task-skeleton" />
        </div>
        <EmptyState
          v-else-if="!items.length"
          icon="HOutline:QueueListIcon"
          :title="t('taskManager.empty')"
          :description="t('taskManager.emptyDesc')"
        />
        <div v-else class="task-cards">
          <EntityCard v-for="row in items" :key="row.id">
            <template #title>{{ row.title || row.type }}</template>
            <template #status>
              <StatusPill :variant="statusVariant(row.status)" :label="statusLabel(row.status)" />
            </template>
            <template #meta>
              <StatusPill variant="neutral" :dot="false" :label="kindLabel(row.kind)" />
              <span class="task-card__type">{{ row.type }}</span>
              <span>{{ row.userName || t('taskManager.system') }}</span>
              <span class="task-card__full">{{ formatTime(row.createTime) }}</span>
              <span v-if="row.message" class="task-card__full task-title__msg">{{ row.message }}</span>
              <span v-if="row.error" class="task-card__full task-title__err">{{ row.error }}</span>
            </template>
            <template #footer>
              <div v-if="Number(row.total) > 0" class="task-progress">
                <div class="task-progress__track">
                  <div
                    class="task-progress__fill"
                    :class="`is-${progressTone(row.status)}`"
                    :style="{ width: `${percent(row)}%` }"
                  />
                </div>
                <span class="task-progress__pct">{{ percent(row) }}%</span>
              </div>
              <span v-else class="text-dim">—</span>
              <ActionMenu
                v-if="hasRowActions(row.status)"
                :items="rowActions(row.status)"
                @select="(key) => onRowAction(key, row)"
              />
            </template>
          </EntityCard>
        </div>
      </template>

      <Pagination
        v-model:page="page"
        v-model:page-size="pageSize"
        :total="total"
        @change="() => getList()"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { showRequestError } from '@/utils/request'
import { Dialog } from '@/utils/dialog'
import { listTasks, cancelTask, retryTask, deleteTask } from '@/api/task'
import { TaskStatus, type TaskInfo } from '@/types/v1/task'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

defineOptions({ name: 'TaskManagerView' })

const menuStore = useMenuStore()
const { t } = useI18n()

const REFRESH_INTERVAL_MS = 2500

const loading = ref(false)
const items = ref<TaskInfo[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const keywords = ref('')
const statusFilter = ref('')
let timer: ReturnType<typeof setInterval> | undefined

// 过滤状态值沿用原页面（后端小写状态过滤契约）
const statusOptions = computed<{ label: string; value: string }[]>(() => [
  { label: t('taskManager.allStatus'), value: '' },
  { label: t('taskManager.statusRunning'), value: 'running' },
  { label: t('taskManager.statusPending'), value: 'pending' },
  { label: t('taskManager.statusSuccess'), value: 'success' },
  { label: t('taskManager.statusFailed'), value: 'failed' },
  { label: t('taskManager.statusCanceled'), value: 'canceled' },
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'title', title: t('taskManager.title'), minWidth: 220, ellipsis: false },
  { key: 'type', title: t('taskManager.type'), minWidth: 150 },
  { key: 'kind', title: t('taskManager.kind'), width: 90 },
  { key: 'status', title: t('taskManager.status'), width: 100 },
  { key: 'progress', title: t('taskManager.progress'), width: 160 },
  { key: 'creator', title: t('taskManager.creator'), width: 130 },
  { key: 'createTime', title: t('taskManager.createTime'), width: 170 },
  { key: 'operation', title: t('system.common.operation'), width: 70, align: 'center' },
])

const kindLabel = (kind: string) => {
  switch (kind) {
    case 'TASK_KIND_ONESHOT':
      return t('taskManager.kindOneshot')
    case 'TASK_KIND_SCHEDULED':
      return t('taskManager.kindScheduled')
    case 'TASK_KIND_DAEMON':
      return t('taskManager.kindDaemon')
    default:
      return kind
  }
}

const statusLabel = (status: string) => {
  switch (status) {
    case TaskStatus.TASK_STATUS_PENDING:
      return t('taskManager.statusPending')
    case TaskStatus.TASK_STATUS_RUNNING:
      return t('taskManager.statusRunning')
    case TaskStatus.TASK_STATUS_SUCCESS:
      return t('taskManager.statusSuccess')
    case TaskStatus.TASK_STATUS_FAILED:
      return t('taskManager.statusFailed')
    case TaskStatus.TASK_STATUS_CANCELED:
      return t('taskManager.statusCanceled')
    default:
      return status
  }
}

const statusVariant = (status: string) => {
  switch (status) {
    case TaskStatus.TASK_STATUS_RUNNING:
      return 'primary' as const
    case TaskStatus.TASK_STATUS_SUCCESS:
      return 'success' as const
    case TaskStatus.TASK_STATUS_FAILED:
      return 'error' as const
    case TaskStatus.TASK_STATUS_PENDING:
      return 'info' as const
    default:
      return 'neutral' as const
  }
}

const progressTone = (status: string) => {
  if (status === TaskStatus.TASK_STATUS_SUCCESS) return 'success'
  if (status === TaskStatus.TASK_STATUS_FAILED) return 'error'
  return 'primary'
}

const percent = (row: Record<string, unknown>) => {
  const totalNum = Number(row.total) || 0
  if (totalNum <= 0) return 0
  return Math.min(100, Math.round((Number(row.finished) / totalNum) * 100))
}

const formatTime = (v: unknown) => (v ? dayjs(v as string | Date).format('YYYY-MM-DD HH:mm:ss') : '—')

const isActive = (status: string) =>
  status === TaskStatus.TASK_STATUS_RUNNING || status === TaskStatus.TASK_STATUS_PENDING
const isTerminal = (status: string) =>
  status === TaskStatus.TASK_STATUS_SUCCESS ||
  status === TaskStatus.TASK_STATUS_FAILED ||
  status === TaskStatus.TASK_STATUS_CANCELED

const rowActions = (status: string): ActionMenuItem[] => [
  {
    key: 'cancel',
    label: t('taskManager.cancel'),
    icon: 'HOutline:XCircleIcon',
    hidden: !isActive(status),
  },
  {
    key: 'retry',
    label: t('taskManager.retry'),
    icon: 'HOutline:ArrowPathIcon',
    hidden: status !== TaskStatus.TASK_STATUS_FAILED,
  },
  {
    key: 'delete',
    label: t('system.common.delete'),
    icon: 'HOutline:TrashIcon',
    danger: true,
    hidden: !isTerminal(status),
  },
]
const hasRowActions = (status: string) => rowActions(status).some((a) => !a.hidden)

const onRowAction = (key: string, row: Record<string, unknown>) => {
  const id = String(row.id)
  if (key === 'cancel') cancel(id)
  else if (key === 'retry') retry(id)
  else if (key === 'delete') confirmDelete(id)
}

const getList = async (silent = false) => {
  if (!silent) loading.value = true
  try {
    const { data } = await listTasks({
      page: page.value,
      pageSize: pageSize.value,
      keywords: keywords.value || undefined,
      status: statusFilter.value || undefined,
    })
    items.value = data.items ?? []
    total.value = Number(data.total) || 0
  } catch (error) {
    if (!silent) showRequestError(error, t('taskManager.loadFailed'))
  } finally {
    if (!silent) loading.value = false
  }
}

const reload = () => {
  page.value = 1
  getList()
}
const reset = () => {
  keywords.value = ''
  statusFilter.value = ''
  page.value = 1
  getList()
}

const cancel = async (id: string) => {
  try {
    await cancelTask(id)
    ElMessage.success(t('taskManager.cancelSuccess'))
    getList(true)
  } catch (error) {
    showRequestError(error, t('taskManager.actionFailed'))
  }
}

const retry = async (id: string) => {
  try {
    await retryTask(id)
    ElMessage.success(t('taskManager.retrySuccess'))
    getList(true)
  } catch (error) {
    showRequestError(error, t('taskManager.actionFailed'))
  }
}

const confirmDelete = (id: string) => {
  Dialog.info({
    showCancelButton: true,
    content: t('taskManager.confirmDelete'),
    confirmText: t('system.common.delete'),
    cancelText: t('system.common.cancel'),
    onConfirm: async () => {
      try {
        await deleteTask(id)
        ElMessage.success(t('taskManager.deleteSuccess'))
        getList(true)
      } catch (error) {
        showRequestError(error, t('taskManager.actionFailed'))
      }
    },
  })
}

onMounted(() => {
  getList()
  timer = setInterval(() => getList(true), REFRESH_INTERVAL_MS)
})
onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped lang="scss">
.task-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.task-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.task-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.task-page__bar-hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.text-dim {
  color: var(--el-text-color-placeholder);
}

/* 标题多行单元格（含消息/错误副文） */
.task-title {
  display: flex;
  flex-direction: column;
  gap: 2px;
  white-space: normal;
  min-width: 0;
}
.task-title__main {
  color: var(--el-text-color-primary);
  font-weight: 500;
}
.task-title__msg {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
}
.task-title__err {
  font-size: 0.75rem;
  color: var(--el-color-danger, #ef4444);
  word-break: break-word;
}

/* 进度条 */
.task-progress {
  display: flex;
  align-items: center;
  gap: 8px;
}
.task-progress__track {
  flex: 1;
  min-width: 60px;
  height: 6px;
  border-radius: 999px;
  background: var(--el-fill-color);
  overflow: hidden;
}
.task-progress__fill {
  height: 100%;
  border-radius: 999px;
  transition: width 0.3s ease;
}
.task-progress__fill.is-primary {
  background: var(--el-color-primary);
}
.task-progress__fill.is-success {
  background: var(--el-color-success, #16a34a);
}
.task-progress__fill.is-error {
  background: var(--el-color-danger, #ef4444);
}
.task-progress__pct {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
  min-width: 32px;
  text-align: right;
}

/* 移动卡片 */
.task-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.task-card__type {
  font-variant-numeric: tabular-nums;
}
.task-card__full {
  flex-basis: 100%;
}
.task-skeleton {
  height: 108px;
  border-radius: var(--app-radius);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: task-shimmer 1.4s ease-in-out infinite;
}
@keyframes task-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
