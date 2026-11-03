<!-- Docker 后台任务：列表弹窗（状态 chips 带数量 + 刷新；任务行 = 类型贴纸 + 标题 + 状态 + 开始时间/用时 + 消息）
     → 点任务打开日志面板（DockerTermSheet：PC 居中大面板 / 手机底部面板，固定终端色 xterm 只读输出）。 -->
<template>
  <BaseDialog v-model="visible" :title="t('docker.task.title')" width="760" :show-footer="false" @open="loadTasks">
    <div class="tk">
      <p class="tk-sub">{{ t('docker.task.sub') }}</p>
      <div class="tk-toolbar">
        <div class="tk-chips" role="tablist" :aria-label="t('docker.task.filterLabel')">
          <button
            v-for="chip in statusChips"
            :key="chip.value"
            type="button"
            role="tab"
            class="tk-chip"
            :class="{ 'is-on': statusFilter === chip.value }"
            :aria-selected="statusFilter === chip.value"
            @click="statusFilter = chip.value"
          >{{ chip.label }} <span class="tk-chip__n m-num">{{ chip.count }}</span></button>
        </div>
        <AppIconButton icon="HOutline:ArrowPathIcon" :label="t('docker.common.refresh')" :box="40" :size="18" @click="loadTasks" />
      </div>

      <div class="tk-list" :class="{ 'is-loading': listLoading }">
        <div v-if="listLoading && !tasks.length" class="tk-list__loading">{{ t('system.common.loading') }}</div>
        <template v-else>
          <button
            v-for="item in filteredTasks"
            :key="item.id"
            type="button"
            class="tk-item"
            @click="openLog(item)"
          >
            <span class="m-sticker tk-item__ico" :class="`m-sticker--${typeTone(item.type)}`">
              <component :is="menuStore.iconComponents[typeIcon(item.type)]" />
            </span>
            <span class="tk-item__main">
              <span class="tk-item__row">
                <span class="tk-item__title">{{ displayTaskTitle(item) }}</span>
                <StatusPill :variant="statusPillVariant(item.status)" :label="statusLabel(item.status)" />
              </span>
              <span class="tk-item__meta m-num">
                {{ t('docker.task.started', { time: formatTime(item.startTime) }) }}
                <template v-if="durationText(item)"> · {{ t('docker.task.duration', { d: durationText(item) }) }}</template>
              </span>
              <span v-if="item.error" class="tk-item__msg is-error">{{ item.error }}</span>
              <span v-else-if="item.message" class="tk-item__msg">{{ item.message }}</span>
            </span>
            <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" class="tk-item__chev" />
          </button>
          <EmptyState
            v-if="!filteredTasks.length"
            icon="HOutline:InboxIcon"
            :title="t('docker.task.noTasks')"
          />
        </template>
      </div>
    </div>
  </BaseDialog>

  <DockerTermSheet
    v-model="logVisible"
    :title="logTitle"
    :subtitle="t('docker.task.logSub')"
    :icon="selectedTask ? typeIcon(selectedTask.type) : 'HOutline:CommandLineIcon'"
    :tone="selectedTask ? typeTone(selectedTask.type) : 'a2'"
  >
    <template #head-extra>
      <StatusPill
        v-if="selectedTask"
        :variant="statusPillVariant(selectedTask.status)"
        :label="statusLabel(selectedTask.status)"
      />
    </template>
    <div class="tk-term">
      <div ref="terminalRef" class="tk-term__screen" />
    </div>
    <div class="tk-term-foot">
      <button type="button" class="tk-back" @click="backToList">
        <component :is="menuStore.iconComponents['HOutline:ArrowLeftIcon']" />{{ t('docker.task.backToList') }}
      </button>
    </div>
  </DockerTermSheet>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { FitAddon } from '@xterm/addon-fit'
import { Terminal } from '@xterm/xterm'
import '@xterm/xterm/css/xterm.css'
import { listDockerTasks } from '@/api/docker'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import DockerTermSheet from '@/views/docker/components/DockerTermSheet.vue'
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
const { t } = useI18n()
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

const STATUS_PILL_MAP: Partial<
  Record<DockerTaskStatus, 'success' | 'info' | 'warning' | 'error' | 'neutral'>
> = {
  [DockerTaskStatus.DOCKER_TASK_STATUS_PENDING]: 'info',
  [DockerTaskStatus.DOCKER_TASK_STATUS_RUNNING]: 'warning',
  [DockerTaskStatus.DOCKER_TASK_STATUS_SUCCESS]: 'success',
  [DockerTaskStatus.DOCKER_TASK_STATUS_FAILED]: 'error',
  [DockerTaskStatus.DOCKER_TASK_STATUS_CANCELED]: 'neutral',
}

const TASK_TYPE_KEY_MAP: Partial<Record<DockerTaskType, string>> = {
  [DockerTaskType.DOCKER_TASK_TYPE_CONTAINER_RECREATE]: 'docker.task.type.containerRecreate',
  [DockerTaskType.DOCKER_TASK_TYPE_IMAGE_PULL]: 'docker.task.type.imagePull',
  [DockerTaskType.DOCKER_TASK_TYPE_NETWORK_RECREATE]: 'docker.task.type.networkRecreate',
  [DockerTaskType.DOCKER_TASK_TYPE_NETWORK_PRUNE]: 'docker.task.type.networkPrune',
}

const logTitle = computed(() => selectedTask.value ? displayTaskTitle(selectedTask.value) : t('docker.task.logTitle'))
const statusLabel = (status: DockerTaskStatus) => {
  if (status === DockerTaskStatus.DOCKER_TASK_STATUS_PENDING) return t('docker.task.status.pending')
  if (status === DockerTaskStatus.DOCKER_TASK_STATUS_RUNNING) return t('docker.task.status.running')
  if (status === DockerTaskStatus.DOCKER_TASK_STATUS_SUCCESS) return t('docker.task.status.success')
  if (status === DockerTaskStatus.DOCKER_TASK_STATUS_FAILED) return t('docker.task.status.failed')
  if (status === DockerTaskStatus.DOCKER_TASK_STATUS_CANCELED) return t('docker.task.status.canceled')
  return '-'
}
const statusPillVariant = (status: DockerTaskStatus) => STATUS_PILL_MAP[status] || 'info'

type StatusFilter = 'all' | DockerTaskStatus
const statusFilter = ref<StatusFilter>('all')
const STATUS_ORDER: DockerTaskStatus[] = [
  DockerTaskStatus.DOCKER_TASK_STATUS_RUNNING,
  DockerTaskStatus.DOCKER_TASK_STATUS_PENDING,
  DockerTaskStatus.DOCKER_TASK_STATUS_SUCCESS,
  DockerTaskStatus.DOCKER_TASK_STATUS_FAILED,
  DockerTaskStatus.DOCKER_TASK_STATUS_CANCELED,
]
const statusChips = computed(() => {
  const chips: { value: StatusFilter; label: string; count: number }[] = [
    { value: 'all', label: t('docker.task.all'), count: tasks.value.length },
  ]
  for (const st of STATUS_ORDER) {
    const count = tasks.value.filter((x) => x.status === st).length
    if (count || statusFilter.value === st) chips.push({ value: st, label: statusLabel(st), count })
  }
  return chips
})
const filteredTasks = computed(() =>
  statusFilter.value === 'all' ? tasks.value : tasks.value.filter((x) => x.status === statusFilter.value),
)

const TYPE_ICON: Partial<Record<DockerTaskType, string>> = {
  [DockerTaskType.DOCKER_TASK_TYPE_CONTAINER_RECREATE]: 'HOutline:ArrowPathRoundedSquareIcon',
  [DockerTaskType.DOCKER_TASK_TYPE_IMAGE_PULL]: 'HOutline:ArrowDownTrayIcon',
  [DockerTaskType.DOCKER_TASK_TYPE_NETWORK_RECREATE]: 'HOutline:GlobeAltIcon',
  [DockerTaskType.DOCKER_TASK_TYPE_NETWORK_PRUNE]: 'HOutline:TrashIcon',
}
const TYPE_TONE: Partial<Record<DockerTaskType, 'p' | 'a' | 'a2' | 'ok'>> = {
  [DockerTaskType.DOCKER_TASK_TYPE_CONTAINER_RECREATE]: 'p',
  [DockerTaskType.DOCKER_TASK_TYPE_IMAGE_PULL]: 'a2',
  [DockerTaskType.DOCKER_TASK_TYPE_NETWORK_RECREATE]: 'ok',
  [DockerTaskType.DOCKER_TASK_TYPE_NETWORK_PRUNE]: 'a',
}
const typeIcon = (type: DockerTaskType) => TYPE_ICON[type] || 'HOutline:CommandLineIcon'
const typeTone = (type: DockerTaskType) => TYPE_TONE[type] || 'a2'

const durationText = (task: DockerTaskInfo) => {
  const start = toDateValue(task.startTime)
  if (!start) return ''
  const end = toDateValue(task.endTime)
  const running = task.status === DockerTaskStatus.DOCKER_TASK_STATUS_RUNNING || task.status === DockerTaskStatus.DOCKER_TASK_STATUS_PENDING
  if (!end && !running) return ''
  const sec = Math.max(0, Math.round(((end?.getTime() ?? Date.now()) - start.getTime()) / 1000))
  if (sec < 60) return t('tools.time.sec', { s: sec })
  if (sec < 3600) return t('tools.time.min', { m: Math.floor(sec / 60) })
  return t('tools.time.hourMin', { h: Math.floor(sec / 3600), m: Math.floor((sec % 3600) / 60) })
}
const displayTaskTitle = (task: DockerTaskInfo) => task.title || (TASK_TYPE_KEY_MAP[task.type] ? t(TASK_TYPE_KEY_MAP[task.type] as string) : task.id)

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
    if (selectedTask.value) selectedTask.value = tasks.value.find((x) => x.id === selectedTask.value?.id) || selectedTask.value
  } catch (error) {
    tasks.value = []
    showRequestError(error, t('docker.task.loadFailed'))
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
      background: '#0f0b14',
      foreground: '#e8e3ee',
      cursor: '#e8e3ee',
      selectionBackground: 'rgba(232, 227, 238, 0.25)',
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
    terminal?.writeln(`\x1b[31m${t('docker.task.missingWsPath')}\x1b[0m`)
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
    terminal?.writeln(`\x1b[31m${t('docker.task.logConnectError')}\x1b[0m`)
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

const backToList = () => {
  logVisible.value = false
  visible.value = true
}

watch(logVisible, (open) => {
  if (open) void handleLogOpened()
  else handleLogClosed()
})

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
.tk { display: flex; flex-direction: column; gap: 12px; min-height: 420px; }
.tk-sub { margin: 0; font-size: 0.8125rem; color: var(--el-text-color-secondary); }
.tk-toolbar { display: flex; align-items: center; gap: 8px; }
.tk-chips {
  display: flex;
  flex: 1;
  gap: 8px;
  min-width: 0;
  overflow-x: auto;
  scrollbar-width: none;
}
.tk-chips::-webkit-scrollbar { display: none; }
.tk-chip {
  display: inline-flex;
  flex: none;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 600;
  white-space: nowrap;
  cursor: pointer;
}
.tk-chip:hover { border-color: var(--m-p); color: var(--m-pInk); }
.tk-chip.is-on { background: var(--m-ink); border-color: var(--m-ink); color: var(--m-bg); }
.tk-chip__n { font-size: 0.75rem; opacity: 0.7; }
.tk-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
  min-height: 320px;
  max-height: 60vh;
  overflow-y: auto;
  scrollbar-width: thin;
}
.tk-list.is-loading { opacity: 0.7; }
.tk-list__loading {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 220px;
  color: var(--el-text-color-secondary);
  font-size: 0.8125rem;
}
.tk-item {
  display: flex;
  align-items: center;
  gap: 12px;
  width: 100%;
  min-height: 64px;
  padding: 12px 12px 12px 14px;
  border-radius: var(--app-radius-lg);
  border: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  text-align: left;
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}
.tk-item:hover { border-color: var(--m-p); background: color-mix(in srgb, var(--m-pSoft) 35%, var(--el-bg-color)); }
.tk-item__ico {
  width: 40px;
  height: 40px;
  flex: none;
  border-radius: 13px;
  transform: rotate(-4deg);
  :deep(svg) { width: 20px; height: 20px; }
}
.tk-item__main { display: flex; flex-direction: column; gap: 3px; flex: 1; min-width: 0; }
.tk-item__row { display: flex; align-items: center; gap: 10px; min-width: 0; }
.tk-item__title {
  flex: 1;
  min-width: 0;
  font-size: 0.9375rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tk-item__meta { font-size: 0.75rem; color: var(--el-text-color-secondary); }
.tk-item__msg {
  font-size: 0.75rem;
  color: var(--el-text-color-regular);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.tk-item__msg.is-error { color: var(--m-bad); }
.tk-item__chev { width: 18px; height: 18px; flex: none; color: var(--el-text-color-secondary); }

/* 任务日志：固定终端色 */
.tk-term {
  flex: 1;
  min-height: 0;
  padding: 12px 8px 8px 14px;
  border-radius: var(--app-radius-lg);
  background: #0f0b14;
}
.tk-term__screen { width: 100%; height: 100%; }
.tk-term__screen :deep(.xterm) { height: 100%; background: #0f0b14; }
.tk-term__screen :deep(.xterm-viewport),
.tk-term__screen :deep(.xterm-scrollable-element) { background-color: #0f0b14 !important; }
.tk-term__screen :deep(.xterm-viewport) { overflow-y: auto; scrollbar-width: thin; scrollbar-color: #3a3142 transparent; }
.tk-term-foot { display: flex; justify-content: flex-start; padding-top: 10px; }
.tk-back {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 14px;
  border: none;
  border-radius: 999px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  font-weight: 700;
  cursor: pointer;
  :deep(svg) { width: 16px; height: 16px; }
}
.tk-back:hover { background: var(--m-pSoft); color: var(--m-pInk); }
@media (width <= 768px) {
  .tk { min-height: 0; }
  .tk-item { min-height: 68px; }
  .tk-term { padding: 10px 4px 6px 10px; }
  .tk-back { width: 100%; justify-content: center; min-height: 44px; }
}
</style>
