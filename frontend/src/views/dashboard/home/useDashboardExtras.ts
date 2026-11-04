import { getDockerStatus, listDockerContainers, restartDockerContainer } from '@/api/docker'
import { checkUpdateRequest } from '@/api/login'
import { listTasks } from '@/api/task'
import type {
  DockerContainerListStats,
  DockerContainerSummary,
  DockerStatusResponse,
} from '@/types/v1/docker'
import type { CheckUpdateResponse } from '@/types/v1/system'
import type { TaskInfo } from '@/types/v1/task'

// 工作台的补充数据：Docker 概况与异常退出的容器、后台任务、版本更新。
// 每块各自按权限加载，失败只影响自己那张卡片。

const UPDATE_CACHE_KEY = 'momoko-update-result'

/** 「需要你看一眼」里的一条待办 */
export interface AttentionItem {
  key: string
  icon: string
  tone: 'bad' | 'warn' | 'a' | 'a2' | 'p' | 'ok'
  title: string
  sub: string
  loading?: boolean
  action?: { label: string; run: () => void }
}

export interface ExitedContainer {
  id: string
  name: string
  image: string
  code: number
  finishedAt?: Date | string
}

const containerName = (c: DockerContainerSummary) =>
  (c.names?.[0] || c.id.slice(0, 12)).replace(/^\//, '')

const readUpdateCache = (): CheckUpdateResponse | null => {
  try {
    const raw = sessionStorage.getItem(UPDATE_CACHE_KEY)
    return raw ? (JSON.parse(raw) as CheckUpdateResponse) : null
  } catch {
    return null
  }
}
const writeUpdateCache = (data: CheckUpdateResponse) => {
  try {
    sessionStorage.setItem(UPDATE_CACHE_KEY, JSON.stringify(data))
  } catch {
    // 无法缓存时每次进入工作台查询一次
  }
}

export const useDashboardExtras = () => {
  const menuStore = useMenuStore()

  const canDocker = computed(() => menuStore.hasButtonPermission('docker:view'))
  const canManageContainer = computed(() => menuStore.hasButtonPermission('docker_container:manage'))
  const canTasks = computed(() => menuStore.hasButtonPermission('task:manage'))
  const canUpdate = computed(() => menuStore.hasButtonPermission('system:update'))

  // ---- Docker ----
  const docker = ref<DockerStatusResponse | null>(null)
  const exited = ref<ExitedContainer[]>([])
  const stats = ref<DockerContainerListStats | null>(null)
  const dockerLoaded = ref(false)
  const restarting = ref<Set<string>>(new Set())

  const loadDocker = async () => {
    if (!canDocker.value) return
    try {
      const { data } = await getDockerStatus()
      docker.value = data
      if (data?.connected) {
        // status=exited 只含非零退出码 / 已死亡的容器；stats 是不受筛选影响的全量统计
        const { data: list } = await listDockerContainers({
          status: 'exited',
          keyword: '',
          network: '',
          labels: {},
          page: 1,
          pageSize: 10,
        })
        stats.value = list?.stats ?? null
        exited.value = (list?.items || []).map((c) => ({
          id: c.id,
          name: containerName(c),
          image: c.image,
          code: c.exitCode,
          finishedAt: c.finishedAt,
        }))
      } else {
        stats.value = null
        exited.value = []
      }
    } catch {
      docker.value = null
      stats.value = null
      exited.value = []
    } finally {
      dockerLoaded.value = true
    }
  }

  const dockerCounts = computed(() => {
    const st = stats.value
    const info = docker.value?.info
    const running = st?.running ?? info?.containersRunning ?? 0
    const stopped = st ? st.stopped + st.paused : (info?.containersStopped ?? 0) + (info?.containersPaused ?? 0)
    const exitedN = st?.exited ?? 0
    return { running, stopped, exited: exitedN, total: running + stopped + exitedN }
  })

  const restartContainer = async (id: string) => {
    const next = new Set(restarting.value)
    next.add(id)
    restarting.value = next
    try {
      await restartDockerContainer({ id, timeoutSeconds: 10 })
      await loadDocker()
    } finally {
      const done = new Set(restarting.value)
      done.delete(id)
      restarting.value = done
    }
  }

  // ---- 后台任务 ----
  const tasks = ref<TaskInfo[]>([])
  const runningTaskCount = ref(0)

  const loadTasks = async () => {
    if (!canTasks.value) return
    try {
      const { data } = await listTasks({ page: 1, pageSize: 3, kind: 'oneshot' })
      tasks.value = data?.items || []
      runningTaskCount.value = Number(data?.counts?.running ?? 0) + Number(data?.counts?.pending ?? 0)
    } catch {
      tasks.value = []
    }
  }

  // ---- 版本更新（同一标签页只查询一次） ----
  const update = ref<CheckUpdateResponse | null>(readUpdateCache())

  const loadUpdate = async (fresh?: CheckUpdateResponse | null) => {
    if (fresh) {
      update.value = fresh
      writeUpdateCache(fresh)
      return
    }
    if (!canUpdate.value || update.value) return
    try {
      const { data } = await checkUpdateRequest()
      if (data) {
        update.value = data
        writeUpdateCache(data)
      }
    } catch {
      // 检查失败不提示
    }
  }

  const loadAll = () => Promise.allSettled([loadDocker(), loadTasks()])

  return {
    canDocker,
    canManageContainer,
    canTasks,
    canUpdate,
    docker,
    dockerLoaded,
    dockerCounts,
    exited,
    restarting,
    restartContainer,
    tasks,
    runningTaskCount,
    update,
    loadDocker,
    loadTasks,
    loadUpdate,
    loadAll,
  }
}
