// Docker 后台任务（拉取镜像 / 创建、重建容器 / 重建网络）的共享列表：各页标题的「任务」角标、拉取进度卡共用。
// 有进行中的任务时每 2.5 秒轮询一次；没有组件在用时停止轮询。
import { listDockerTasks } from '@/api/docker'
import { DockerTaskStatus } from '@/types/v1/docker'
import type { DockerTaskInfo } from '@/types/v1/docker'

const tasks = ref<DockerTaskInfo[]>([])
const loaded = ref(false)
/** 每有一个进行中的任务结束就 +1，页面据此刷新列表 */
const finishedTick = ref(0)
let timer: ReturnType<typeof setTimeout> | null = null
let inflight: Promise<void> | null = null
let users = 0

export const isActiveTask = (task: DockerTaskInfo) =>
  task.status === DockerTaskStatus.DOCKER_TASK_STATUS_RUNNING ||
  task.status === DockerTaskStatus.DOCKER_TASK_STATUS_PENDING

const activeIds = () => new Set(tasks.value.filter(isActiveTask).map((t) => t.id))

const stop = () => {
  if (timer) clearTimeout(timer)
  timer = null
}

const schedule = () => {
  stop()
  if (users > 0 && tasks.value.some(isActiveTask)) timer = setTimeout(() => void refresh(), 2500)
}

const refresh = async () => {
  if (inflight) return inflight
  inflight = (async () => {
    const before = activeIds()
    try {
      const { data } = await listDockerTasks({})
      tasks.value = (data?.tasks || []).slice().sort(
        (a, b) => new Date(String(b.startTime || 0)).getTime() - new Date(String(a.startTime || 0)).getTime(),
      )
      loaded.value = true
    } catch {
      /* 任务列表是辅助信息，失败时静默 */
    }
    const after = activeIds()
    let ended = 0
    before.forEach((id) => {
      if (!after.has(id)) ended++
    })
    if (ended) finishedTick.value += ended
  })()
  try {
    await inflight
  } finally {
    inflight = null
    schedule()
  }
}

export function useDockerTasks() {
  const active = computed(() => tasks.value.filter(isActiveTask))
  onMounted(() => {
    users++
    void refresh()
  })
  onBeforeUnmount(() => {
    users = Math.max(0, users - 1)
    if (!users) stop()
  })
  return { tasks, loaded, active, finishedTick, refresh }
}
