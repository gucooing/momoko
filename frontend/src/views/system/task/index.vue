<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <div class="task-header">
        <el-input
          v-model="keywords"
          class="task-search"
          clearable
          :placeholder="t('taskManager.searchPlaceholder')"
          @keyup.enter="reload"
          @clear="reload"
        />
        <el-select v-model="statusFilter" class="task-filter" clearable @change="reload">
          <el-option :label="t('taskManager.statusRunning')" value="running" />
          <el-option :label="t('taskManager.statusPending')" value="pending" />
          <el-option :label="t('taskManager.statusSuccess')" value="success" />
          <el-option :label="t('taskManager.statusFailed')" value="failed" />
          <el-option :label="t('taskManager.statusCanceled')" value="canceled" />
        </el-select>
        <el-button @click="reload">{{ t('taskManager.refresh') }}</el-button>
      </div>
    </el-card>

    <el-card shadow="never" class="card-mt-16">
      <el-table v-loading="loading" :data="items" stripe>
        <el-table-column :label="t('taskManager.title')" prop="title" min-width="180">
          <template #default="{ row }">
            <div>{{ row.title || row.type }}</div>
            <div v-if="row.message" class="task-muted">{{ row.message }}</div>
            <div v-if="row.error" class="task-error">{{ row.error }}</div>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskManager.type')" prop="type" width="150" />
        <el-table-column :label="t('taskManager.kind')" width="90">
          <template #default="{ row }">
            <el-tag size="small" type="info">{{ kindLabel(row.kind) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskManager.status')" width="100">
          <template #default="{ row }">
            <el-tag size="small" :type="statusTagType(row.status)">{{ statusLabel(row.status) }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskManager.progress')" width="160">
          <template #default="{ row }">
            <el-progress
              v-if="Number(row.total) > 0"
              :percentage="progressPercent(row)"
              :status="progressStatus(row.status)"
            />
            <span v-else class="task-muted">{{ Number(row.finished) || 0 }}</span>
          </template>
        </el-table-column>
        <el-table-column :label="t('taskManager.creator')" width="120">
          <template #default="{ row }">{{ row.userName || t('taskManager.system') }}</template>
        </el-table-column>
        <el-table-column :label="t('taskManager.createTime')" width="170">
          <template #default="{ row }">{{ formatDateTime(row.createTime) }}</template>
        </el-table-column>
        <el-table-column :label="t('system.common.operation')" width="200" fixed="right">
          <template #default="{ row }">
            <el-button v-if="isActive(row.status)" type="warning" link @click="cancel(row)">
              {{ t('taskManager.cancel') }}
            </el-button>
            <el-button v-if="row.status === 'TASK_STATUS_FAILED'" type="primary" link @click="retry(row)">
              {{ t('taskManager.retry') }}
            </el-button>
            <el-popconfirm
              v-if="isTerminal(row.status)"
              :title="t('taskManager.confirmDelete')"
              @confirm="remove(row)"
            >
              <template #reference>
                <el-button type="danger" link>{{ t('system.common.delete') }}</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
        <template #empty>{{ t('taskManager.empty') }}</template>
      </el-table>

      <div class="task-pager">
        <el-pagination
          v-model:current-page="page"
          v-model:page-size="pageSize"
          layout="total, prev, pager, next"
          :total="total"
          @current-change="() => getList()"
        />
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
defineOptions({ name: 'TaskManagerView' })
import { useI18n } from 'vue-i18n'
import { showRequestError } from '@/utils/request'
import { formatDateTime } from '@/utils/file'
import { listTasks, cancelTask, retryTask, deleteTask } from '@/api/task'
import type { TaskInfo } from '@/types/v1/task'

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
    case 'TASK_STATUS_PENDING':
      return t('taskManager.statusPending')
    case 'TASK_STATUS_RUNNING':
      return t('taskManager.statusRunning')
    case 'TASK_STATUS_SUCCESS':
      return t('taskManager.statusSuccess')
    case 'TASK_STATUS_FAILED':
      return t('taskManager.statusFailed')
    case 'TASK_STATUS_CANCELED':
      return t('taskManager.statusCanceled')
    default:
      return status
  }
}

const statusTagType = (status: string) => {
  switch (status) {
    case 'TASK_STATUS_RUNNING':
      return 'primary'
    case 'TASK_STATUS_SUCCESS':
      return 'success'
    case 'TASK_STATUS_FAILED':
      return 'danger'
    default:
      return 'info'
  }
}

const isActive = (status: string) =>
  status === 'TASK_STATUS_RUNNING' || status === 'TASK_STATUS_PENDING'
const isTerminal = (status: string) =>
  status === 'TASK_STATUS_SUCCESS' ||
  status === 'TASK_STATUS_FAILED' ||
  status === 'TASK_STATUS_CANCELED'

const progressPercent = (row: TaskInfo) => {
  const totalNum = Number(row.total) || 0
  if (totalNum <= 0) return 0
  return Math.min(100, Math.round((Number(row.finished) / totalNum) * 100))
}
const progressStatus = (status: string) => {
  if (status === 'TASK_STATUS_SUCCESS') return 'success'
  if (status === 'TASK_STATUS_FAILED') return 'exception'
  return ''
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

const cancel = async (row: TaskInfo) => {
  try {
    await cancelTask(row.id)
    ElMessage.success(t('taskManager.cancelSuccess'))
    getList(true)
  } catch (error) {
    showRequestError(error, t('taskManager.actionFailed'))
  }
}

const retry = async (row: TaskInfo) => {
  try {
    await retryTask(row.id)
    ElMessage.success(t('taskManager.retrySuccess'))
    getList(true)
  } catch (error) {
    showRequestError(error, t('taskManager.actionFailed'))
  }
}

const remove = async (row: TaskInfo) => {
  try {
    await deleteTask(row.id)
    ElMessage.success(t('taskManager.deleteSuccess'))
    getList(true)
  } catch (error) {
    showRequestError(error, t('taskManager.actionFailed'))
  }
}

onMounted(() => {
  getList()
  timer = setInterval(() => getList(true), REFRESH_INTERVAL_MS)
})

onBeforeUnmount(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped>
.task-header {
  display: flex;
  align-items: center;
  gap: 0.75rem;
}
.task-search {
  width: 240px;
}
.task-filter {
  width: 140px;
}
.task-pager {
  display: flex;
  justify-content: flex-end;
  margin-top: 16px;
}
.task-muted {
  color: var(--el-text-color-placeholder);
  font-size: 12px;
}
.task-error {
  color: var(--el-color-danger);
  font-size: 12px;
}
</style>
