<template>
  <BaseDialog v-model="visible" title="Docker 任务" width="760" :show-footer="false" @open="loadTasks">
    <div class="task-dialog">
      <div class="task-toolbar">
        <el-button size="small" :icon="menuStore.iconComponents.Refresh" :loading="listLoading" @click="loadTasks">
          刷新
        </el-button>
      </div>

      <div v-loading="listLoading" class="task-list">
        <div
          v-for="item in tasks"
          :key="item.id"
          class="task-list-item"
          @click="openLog(item)"
        >
          <div class="task-list-item__main">
            <span class="task-list-item__title">{{ displayTaskTitle(item) }}</span>
            <BaseTag :text="statusLabel(item.status)" :type="statusTagType(item.status)" />
          </div>
          <div class="task-list-item__meta">
            <span>{{ taskTimeRange(item) }}</span>
            <span v-if="item.message">{{ item.message }}</span>
            <span v-else-if="item.error" class="task-error">{{ item.error }}</span>
          </div>
        </div>
        <el-empty v-if="!listLoading && !tasks.length" description="暂无任务" />
      </div>
    </div>
  </BaseDialog>

  <BaseDialog
    v-model="logVisible"
    :title="logTitle"
    width="900"
    :show-footer="false"
    @opened="handleLogOpened"
    @close="handleLogClosed"
  >
    <div class="task-terminal-shell">
      <div ref="terminalRef" class="task-terminal" />
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { listDockerTasks } from '@/api/docker'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import BaseTag from '@/components/tag/BaseTag.vue'
import { showRequestError } from '@/utils/request'
import { buildBackendWebSocketUrl } from '@/utils/websocket'
import { DockerTaskStatus, DockerTaskType } from '@/types/v1/docker'
import type { DockerTaskInfo } from '@/types/v1/docker'

const props = defineProps<{
  modelValue: boolean
  activeTask?: DockerTaskInfo | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  finished: [task: DockerTaskInfo]
}>()

const menuStore = useMenuStore()
const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value),
})

const tasks = ref<DockerTaskInfo[]>([])
const listLoading = ref(false)
const selectedTask = ref<DockerTaskInfo | null>(null)
const logVisible = ref(false)
const terminalRef = ref<HTMLElement>()
const socket = shallowRef<WebSocket | null>(null)
const finishedTaskIds = ref(new Set<string>())

let terminal: Terminal | null = null
let fitAddon: FitAddon | null = null
let resizeObserver: ResizeObserver | null = null
let manualSocketClose = false

const STATUS_MAP: Partial<Record<DockerTaskStatus, string>> = {
  [DockerTaskStatus.DOCKER_TASK_STATUS_PENDING]: '等待中',
  [DockerTaskStatus.DOCKER_TASK_STATUS_RUNNING]: '运行中',
  [DockerTaskStatus.DOCKER_TASK_STATUS_SUCCESS]: '成功',
  [DockerTaskStatus.DOCKER_TASK_STATUS_FAILED]: '失败',
  [DockerTaskStatus.DOCKER_TASK_STATUS_CANCELED]: '已取消',
}

const STATUS_TAG_MAP: Partial<Record<DockerTaskStatus, 'success' | 'info' | 'warning' | 'danger'>> = {
  [DockerTaskStatus.DOCKER_TASK_STATUS_PENDING]: 'info',
  [DockerTaskStatus.DOCKER_TASK_STATUS_RUNNING]: 'warning',
  [DockerTaskStatus.DOCKER_TASK_STATUS_SUCCESS]: 'success',
  [DockerTaskStatus.DOCKER_TASK_STATUS_FAILED]: 'danger',
  [DockerTaskStatus.DOCKER_TASK_STATUS_CANCELED]: 'info',
}

const TASK_TYPE_MAP: Partial<Record<DockerTaskType, string>> = {
  [DockerTaskType.DOCKER_TASK_TYPE_CONTAINER_RECREATE]: '重建容器',
  [DockerTaskType.DOCKER_TASK_TYPE_IMAGE_PULL]: '拉取镜像',
  [DockerTaskType.DOCKER_TASK_TYPE_NETWORK_RECREATE]: '重建网络',
  [DockerTaskType.DOCKER_TASK_TYPE_NETWORK_PRUNE]: '清理网络',
  [DockerTaskType.DOCKER_TASK_TYPE_VOLUME_RECREATE]: '重建储存卷',
  [DockerTaskType.DOCKER_TASK_TYPE_VOLUME_PRUNE]: '清理储存卷',
  [DockerTaskType.DOCKER_TASK_TYPE_VOLUME_EXPORT]: '导出储存卷',
  [DockerTaskType.DOCKER_TASK_TYPE_VOLUME_RESTORE]: '恢复储存卷',
}

const logTitle = computed(() => selectedTask.value ? displayTaskTitle(selectedTask.value) : '任务日志')
const statusLabel = (status: DockerTaskStatus) => STATUS_MAP[status] || '-'
const statusTagType = (status: DockerTaskStatus) => STATUS_TAG_MAP[status] || 'info'
const displayTaskTitle = (task: DockerTaskInfo) => task.title || TASK_TYPE_MAP[task.type] || task.id

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

const loadTasks = async () => {
  listLoading.value = true
  try {
    const { data } = await listDockerTasks()
    tasks.value = sortTasks(data?.tasks || [])
  } catch (error) {
    tasks.value = []
    showRequestError(error, '获取任务列表失败')
  } finally {
    listLoading.value = false
  }
}

const buildTaskSocketUrl = (task: DockerTaskInfo) => {
  if (!task.id) return ''
  return buildBackendWebSocketUrl(task.wsPath || '', (url) => {
    url.searchParams.set('task_id', task.id)
  })
}

const decodeSocketPayload = (payload: ArrayBuffer | string) => {
  if (typeof payload === 'string') return payload
  return new TextDecoder().decode(payload)
}

const mountTerminal = () => {
  const el = terminalRef.value
  if (!el || terminal) return

  terminal = new Terminal({
    convertEol: true,
    cursorBlink: false,
    disableStdin: true,
    fontFamily: 'Consolas, Menlo, Monaco, monospace',
    fontSize: 13,
    theme: {
      background: '#0b1020',
      foreground: '#d5dde8',
    },
  })
  fitAddon = new FitAddon()
  terminal.loadAddon(fitAddon)
  terminal.open(el)
  fitAddon.fit()

  resizeObserver = new ResizeObserver(() => fitAddon?.fit())
  resizeObserver.observe(el)
}

const resetTerminal = () => {
  terminal?.dispose()
  terminal = null
  fitAddon = null
  resizeObserver?.disconnect()
  resizeObserver = null
}

const closeSocket = () => {
  if (!socket.value) return
  manualSocketClose = true
  socket.value.close()
  socket.value = null
}

const connectTaskSocket = (task: DockerTaskInfo) => {
  closeSocket()
  if (!task.id) return

  const socketUrl = buildTaskSocketUrl(task)
  if (!socketUrl) {
    terminal?.writeln('\x1b[31m缺少任务 WS 路径\x1b[0m')
    return
  }

  manualSocketClose = false
  const nextSocket = new WebSocket(socketUrl)
  socket.value = nextSocket

  nextSocket.onmessage = (event) => {
    if (socket.value !== nextSocket) return
    const text = decodeSocketPayload(event.data as ArrayBuffer | string)
    if (text) terminal?.write(text)
  }

  nextSocket.onerror = () => {
    if (socket.value !== nextSocket) return
    terminal?.writeln('\x1b[31m任务日志连接异常\x1b[0m')
  }

  nextSocket.onclose = () => {
    const closedManually = manualSocketClose
    if (socket.value === nextSocket) socket.value = null
    if (closedManually || !selectedTask.value) return

    void loadTasks()
    if (!finishedTaskIds.value.has(selectedTask.value.id)) {
      finishedTaskIds.value.add(selectedTask.value.id)
      emit('finished', selectedTask.value)
    }
  }
}

const openLog = (task: DockerTaskInfo) => {
  selectedTask.value = task
  upsertTask(task)
  visible.value = false
  logVisible.value = true
}

const handleLogOpened = async () => {
  await nextTick()
  mountTerminal()
  terminal?.clear()
  if (selectedTask.value) connectTaskSocket(selectedTask.value)
}

const handleLogClosed = () => {
  closeSocket()
  resetTerminal()
}

const setActiveTask = (task: DockerTaskInfo | null | undefined) => {
  if (!task?.id) return
  openLog(task)
}

watch(
  () => props.activeTask,
  (task) => {
    setActiveTask(task)
  },
)

onBeforeUnmount(() => {
  closeSocket()
  resetTerminal()
})
</script>

<style scoped lang="scss">
.task-dialog { min-height: 420px; }
.task-toolbar { display: flex; justify-content: flex-end; margin-bottom: 0.5rem; }
.task-list { min-height: 340px; max-height: 62vh; overflow: auto; border: 1px solid var(--el-border-color-lighter); border-radius: 6px; }
.task-list-item { padding: 0.55rem 0.65rem; border-bottom: 1px solid var(--el-border-color-extra-light); cursor: pointer; }
.task-list-item:last-child { border-bottom: 0; }
.task-list-item:hover { background: var(--el-fill-color-light); }
.task-list-item__main { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.task-list-item__title { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: var(--el-text-color-primary); font-size: 0.86rem; font-weight: 600; }
.task-list-item__meta { display: flex; gap: 0.6rem; margin-top: 0.28rem; color: var(--el-text-color-secondary); font-size: 0.74rem; }
.task-list-item__meta span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.task-error { color: var(--el-color-danger); }
.task-terminal-shell { height: min(62vh, 560px); min-height: 360px; padding: 0.5rem; border-radius: 6px; background: #0b1020; }
.task-terminal { width: 100%; height: 100%; }
.task-terminal :deep(.xterm) { height: 100%; }
.task-terminal :deep(.xterm-viewport) { overflow-y: auto; }
</style>
