<template>
  <el-drawer v-model="visible" :size="drawerSize" title="Docker 任务" @open="loadTasks" @closed="handleClosed">
    <div class="task-drawer">
      <div class="task-toolbar">
        <el-button size="small" :icon="menuStore.iconComponents.Refresh" :loading="listLoading" @click="loadTasks">
          刷新
        </el-button>
      </div>

      <div class="task-layout">
        <div class="task-list" v-loading="listLoading">
          <div
            v-for="item in tasks"
            :key="item.id"
            class="task-list-item"
            :class="{ 'task-list-item--active': item.id === selectedTaskId }"
            @click="selectTask(item.id)"
          >
            <div class="task-list-item__main">
              <span class="task-list-item__name">{{ taskTypeLabel(item.type) }}</span>
              <BaseTag :text="statusLabel(item.status)" :type="statusTagType(item.status)" />
            </div>
          </div>
          <el-empty v-if="!listLoading && !tasks.length" description="暂无任务" />
        </div>

        <div class="task-detail">
          <template v-if="selectedTask">
            <div class="task-detail__header">
              <div>
                <div class="task-detail__title">{{ taskTypeLabel(selectedTask.type) }}</div>
                <div class="task-detail__id">{{ selectedTask.id }}</div>
              </div>
              <BaseTag :text="statusLabel(selectedTask.status)" :type="statusTagType(selectedTask.status)" />
            </div>

            <div class="task-meta">
              <div class="task-meta__row">
                <span>ID</span>
                <span>{{ selectedTask.id }}</span>
              </div>
              <div class="task-meta__row">
                <span>消息</span>
                <span>{{ selectedTask.message || '-' }}</span>
              </div>
              <div class="task-meta__row">
                <span>进度</span>
                <span>{{ selectedTask.progress || '-' }}</span>
              </div>
              <div class="task-meta__row">
                <span>结果</span>
                <span>{{ selectedTask.resultPath || '-' }}</span>
              </div>
              <div class="task-meta__row">
                <span>时间</span>
                <span>{{ taskTimeRange(selectedTask) }}</span>
              </div>
              <div v-if="selectedTask.error" class="task-meta__row">
                <span>错误</span>
                <span class="task-error">{{ selectedTask.error }}</span>
              </div>
            </div>

            <div class="event-header">
              <span>事件</span>
              <el-button size="small" link :loading="detailLoading" @click="refreshSelectedTask">刷新</el-button>
            </div>
            <div class="event-list">
              <div v-for="event in selectedTask.events" :key="eventKey(event)" class="event-item">
                <div class="event-time">{{ formatTime(event.time) }}</div>
                <div class="event-body">
                  <div class="event-message">{{ event.message || event.status || '-' }}</div>
                  <div v-if="event.progress" class="event-progress">{{ event.progress }}</div>
                  <div v-if="event.error" class="task-error">{{ event.error }}</div>
                </div>
              </div>
              <el-empty v-if="!selectedTask.events?.length" description="暂无事件" />
            </div>
          </template>
          <el-empty v-else description="请选择任务" />
        </div>
      </div>
    </div>
  </el-drawer>
</template>

<script setup lang="ts">
import { getDockerTask, listDockerTasks } from '@/api/docker'
import BaseTag from '@/components/tag/BaseTag.vue'
import { normalizeAuthToken, showRequestError, toBearerAuthHeader } from '@/utils/request'
import type { DockerTaskEvent, DockerTaskInfo } from '@/types/v1/docker'

const props = defineProps<{
  modelValue: boolean
  activeTask?: DockerTaskInfo | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  finished: [task: DockerTaskInfo]
}>()

const menuStore = useMenuStore()
const drawerSize = computed(() => (menuStore.isMobile ? '100%' : '680px'))
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const tasks = ref<DockerTaskInfo[]>([])
const selectedTaskId = ref('')
const selectedTask = ref<DockerTaskInfo | null>(null)
const listLoading = ref(false)
const detailLoading = ref(false)
const socket = shallowRef<WebSocket | null>(null)
const finishedTaskIds = ref(new Set<string>())

const TASK_TYPE_MAP: Record<string, string> = {
  container_recreate: '重建容器',
  image_pull: '拉取镜像',
  image_build: '构建镜像',
  image_prune: '清理镜像',
  network_recreate: '重建网络',
  network_prune: '清理网络',
  volume_recreate: '重建储存卷',
  volume_prune: '清理储存卷',
  volume_export: '导出储存卷',
  volume_restore: '恢复储存卷',
}

const STATUS_MAP: Record<string, string> = {
  pending: '等待中',
  running: '运行中',
  success: '成功',
  failed: '失败',
  canceled: '已取消',
}

const STATUS_TAG_MAP: Record<string, 'success' | 'info' | 'warning' | 'danger'> = {
  pending: 'info',
  running: 'warning',
  success: 'success',
  failed: 'danger',
  canceled: 'info',
}

const taskTypeLabel = (type: string) => TASK_TYPE_MAP[type] || type || '-'
const statusLabel = (status: string) => STATUS_MAP[status] || status || '-'
const statusTagType = (status: string) => STATUS_TAG_MAP[status] || 'info'

const toDateValue = (value: unknown): Date | undefined => {
  if (!value) return undefined
  if (value instanceof Date) return value
  const date = new Date(String(value))
  return Number.isNaN(date.getTime()) ? undefined : date
}

const pad = (value: number) => String(value).padStart(2, '0')
const formatTime = (value: Date | string | undefined) => {
  const date = toDateValue(value)
  if (!date) return '-'
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}:${pad(date.getSeconds())}`
}

const taskTimeRange = (task: DockerTaskInfo) => {
  const start = formatTime(task.startTime)
  const end = formatTime(task.endTime)
  return end === '-' ? start : `${start} / ${end}`
}

const eventKey = (event: DockerTaskEvent) => {
  return [
    formatTime(event.time),
    event.status,
    event.progress,
    event.id,
    event.message,
    event.error,
  ].join('|')
}

const sortTasks = (items: DockerTaskInfo[]) => {
  return [...items].sort((a, b) => {
    const left = toDateValue(a.startTime)?.getTime() || 0
    const right = toDateValue(b.startTime)?.getTime() || 0
    return right - left
  })
}

const upsertTask = (task: DockerTaskInfo) => {
  const index = tasks.value.findIndex((item) => item.id === task.id)
  if (index >= 0) tasks.value.splice(index, 1, task)
  else tasks.value.unshift(task)
  tasks.value = sortTasks(tasks.value)
}

const isDoneStatus = (status: string) => ['success', 'failed', 'canceled'].includes(status)

const closeSocket = () => {
  if (!socket.value) return
  socket.value.close()
  socket.value = null
}

const getAccessToken = () => normalizeAuthToken(localStorage.getItem('accessToken') || '')

const buildTaskSocketUrl = (taskId: string) => {
  const apiBaseUrl = new URL(
    import.meta.env.VITE_API_BASE_URL || '/api/v1',
    window.location.origin,
  )
  const wsOrigin = `${apiBaseUrl.protocol === 'https:' ? 'wss:' : 'ws:'}//${apiBaseUrl.host}`
  const apiPath = apiBaseUrl.pathname.replace(/\/$/, '')
  const url = new URL(`${apiPath}/docker/task/ws`, wsOrigin)
  url.searchParams.set('task_id', taskId)

  const token = getAccessToken()
  if (token) {
    url.searchParams.set('accessToken', toBearerAuthHeader(token))
  }

  return url.toString()
}

type RawTaskEvent = Partial<DockerTaskEvent> & {
  Time?: string
  Status?: string
  Progress?: string
  ID?: string
  Message?: string
  Error?: string
}

const normalizeEvent = (raw: RawTaskEvent): DockerTaskEvent => ({
  time: toDateValue(raw.time || raw.Time),
  status: raw.status || raw.Status || '',
  progress: raw.progress || raw.Progress || '',
  id: raw.id || raw.ID || '',
  message: raw.message || raw.Message || '',
  error: raw.error || raw.Error || '',
})

const appendTaskEvent = (event: DockerTaskEvent) => {
  if (!selectedTask.value) return

  const exists = selectedTask.value.events.some((item) => eventKey(item) === eventKey(event))
  const nextEvents = exists ? selectedTask.value.events : [...selectedTask.value.events, event]
  const nextTask: DockerTaskInfo = {
    ...selectedTask.value,
    events: nextEvents,
    status: event.status || selectedTask.value.status,
    progress: event.progress || selectedTask.value.progress,
    message: event.message || selectedTask.value.message,
    error: event.error || selectedTask.value.error,
  }

  if (isDoneStatus(nextTask.status) && !nextTask.endTime) {
    nextTask.endTime = event.time || new Date()
  }

  selectedTask.value = nextTask
  upsertTask(nextTask)

  if (isDoneStatus(nextTask.status) && !finishedTaskIds.value.has(nextTask.id)) {
    finishedTaskIds.value.add(nextTask.id)
    emit('finished', nextTask)
    closeSocket()
  }
}

const connectTaskSocket = (taskId: string) => {
  closeSocket()
  if (!taskId) return

  const nextSocket = new WebSocket(buildTaskSocketUrl(taskId))
  socket.value = nextSocket

  nextSocket.onmessage = (event) => {
    if (socket.value !== nextSocket) return
    try {
      appendTaskEvent(normalizeEvent(JSON.parse(String(event.data)) as RawTaskEvent))
    } catch {
      appendTaskEvent({ time: new Date(), status: '', progress: '', id: '', message: String(event.data), error: '' })
    }
  }

  nextSocket.onerror = () => {
    if (socket.value !== nextSocket) return
    ElMessage.error('任务事件连接异常')
  }

  nextSocket.onclose = () => {
    if (socket.value === nextSocket) socket.value = null
  }
}

const loadTasks = async () => {
  listLoading.value = true
  try {
    const { data } = await listDockerTasks()
    tasks.value = sortTasks(data?.tasks || [])
    const firstTask = tasks.value[0]
    if (!selectedTaskId.value && firstTask) {
      await selectTask(firstTask.id)
    }
  } catch (error) {
    tasks.value = []
    showRequestError(error, '获取任务列表失败')
  } finally {
    listLoading.value = false
  }
}

const refreshSelectedTask = async () => {
  if (!selectedTaskId.value) return
  detailLoading.value = true
  try {
    const { data } = await getDockerTask({ taskId: selectedTaskId.value })
    selectedTask.value = data?.task || null
    if (selectedTask.value) upsertTask(selectedTask.value)
  } catch (error) {
    showRequestError(error, '获取任务详情失败')
  } finally {
    detailLoading.value = false
  }
}

const selectTask = async (taskId: string) => {
  selectedTaskId.value = taskId
  await refreshSelectedTask()
  if (selectedTask.value && !isDoneStatus(selectedTask.value.status)) {
    connectTaskSocket(taskId)
  } else {
    closeSocket()
  }
}

const setActiveTask = async (task: DockerTaskInfo | null | undefined) => {
  if (!task?.id) return
  selectedTaskId.value = task.id
  selectedTask.value = {
    ...task,
    events: task.events || [],
  }
  upsertTask(selectedTask.value)
  visible.value = true
  await refreshSelectedTask()
  if (selectedTask.value && !isDoneStatus(selectedTask.value.status)) {
    connectTaskSocket(task.id)
  }
}

const handleClosed = () => {
  closeSocket()
}

watch(
  () => props.activeTask,
  (task) => {
    void setActiveTask(task)
  },
)

onBeforeUnmount(() => {
  closeSocket()
})
</script>

<style scoped lang="scss">
.task-drawer {
  height: 100%;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.task-drawer :deep(.el-tag) {
  height: 22px;
  padding: 0 0.42rem;
  font-size: 0.72rem;
}

.task-toolbar {
  display: flex;
  justify-content: flex-end;
  margin-bottom: 0.5rem;
}

.task-layout {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr);
  gap: 0.6rem;
}

.task-list,
.task-detail {
  min-height: 0;
  overflow: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  background: var(--el-bg-color);
}

.task-list-item {
  padding: 0.45rem 0.55rem;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  cursor: pointer;
}

.task-list-item--active {
  background: var(--el-fill-color-light);
}

.task-list-item__main,
.task-detail__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.4rem;
}

.task-detail__title,
.event-message {
  color: var(--el-text-color-primary);
  font-weight: 600;
}

.task-list-item__name {
  min-width: 0;
  color: var(--el-text-color-regular);
  font-size: 0.78rem;
  font-weight: 400;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-detail {
  padding: 0.55rem;
}

.task-detail__header {
  margin-bottom: 0.5rem;
}

.task-detail__title {
  font-size: 0.95rem;
}

.task-detail__id,
.event-time,
.event-progress {
  margin-top: 0.15rem;
  color: var(--el-text-color-secondary);
  font-size: 0.72rem;
}

.task-meta {
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  overflow: hidden;
}

.task-meta__row {
  display: grid;
  grid-template-columns: 54px minmax(0, 1fr);
  border-bottom: 1px solid var(--el-border-color-extra-light);
  font-size: 0.76rem;
  line-height: 1.45;
}

.task-meta__row:last-child {
  border-bottom: 0;
}

.task-meta__row > span {
  min-width: 0;
  padding: 0.3rem 0.42rem;
  word-break: break-word;
}

.task-meta__row > span:first-child {
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-lighter);
  font-weight: 600;
}

.event-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin: 0.65rem 0 0.35rem;
  font-size: 0.9rem;
  font-weight: 600;
}

.event-list {
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  overflow: hidden;
}

.event-item {
  display: grid;
  grid-template-columns: 120px minmax(0, 1fr);
  gap: 0.5rem;
  padding: 0.38rem 0.5rem;
  border-bottom: 1px solid var(--el-border-color-extra-light);
  font-size: 0.78rem;
}

.event-item:last-child {
  border-bottom: 0;
}

.event-body {
  min-width: 0;
  word-break: break-word;
}

.event-message {
  font-size: 0.78rem;
  line-height: 1.45;
}

.task-error {
  color: var(--el-color-danger);
  word-break: break-word;
}

@media (width <= 768px) {
  .task-layout {
    grid-template-columns: 1fr;
  }

  .task-list {
    max-height: 240px;
  }

  .event-item {
    grid-template-columns: 1fr;
    gap: 0.25rem;
  }
}
</style>
