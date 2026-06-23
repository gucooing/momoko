import dayjs from 'dayjs'
import { defineStore } from 'pinia'
import { getInstances } from '@/api/instance'
import { getSystemOverview, getSystemStatus } from '@/api/system'
import { InstanceStatus, type InstanceInfo } from '@/types/v1/instance'
import type {
  DashboardPendingApproval,
  DashboardShortcutItem,
  DashboardTaskProgress,
} from '@/stores/dashboard/types'
import type {
  SystemOverviewResponse,
  SystemStatusRequest,
  SystemStatusResponse,
} from '@/types/v1/system'
import { translate } from '@/locales'

const MAX_HISTORY = 30

export type RefreshInterval = 1 | 3 | 5 | 10

export interface CpuPoint {
  time: string
  total: number
  cores: number[]
}

export interface MemoryPoint {
  time: string
  physical: number
  physicalUsed: number
  physicalTotal: number
  swap: number
  swapUsed: number
  swapTotal: number
}

export interface NetworkPoint {
  time: string
  download: number
  upload: number
}

export interface DiskPoint {
  time: string
  read: number
  write: number
}

const createHomeShortcuts = (): DashboardShortcutItem[] => [
  { labelKey: 'dashboard.home.shortcuts.analysis', icon: 'HOutline:ChartBarIcon', color: '#ef4444', routePath: '/dashboard/analysis' },
  { labelKey: 'dashboard.home.shortcuts.monitor', icon: 'HOutline:EyeIcon', color: '#4f46e5', routePath: '/dashboard/monitor' },
  { labelKey: 'dashboard.home.shortcuts.users', icon: 'HOutline:UserGroupIcon', color: '#f59e0b', routePath: '/system/user' },
  { labelKey: 'dashboard.home.shortcuts.roles', icon: 'HOutline:IdentificationIcon', color: '#10b981', routePath: '/system/role' },
  { labelKey: 'dashboard.home.shortcuts.apps', icon: 'HOutline:Squares2X2Icon', color: '#ec4899', routePath: '/instance/list' },
  { labelKey: 'dashboard.home.shortcuts.terminal', icon: 'HOutline:CommandLineIcon', color: '#8b5cf6', routePath: '/instance/terminal' },
  { labelKey: 'dashboard.home.shortcuts.fileManager', icon: 'HOutline:FolderIcon', color: '#06b6d4', routePath: '/file/index' },
]

const RUNNING_INSTANCE_LIMIT = 6

export const useDashboardHomeStore = defineStore('dashboard-home', () => {
  const currentDate = ref('')
  const weatherText = computed(() => translate('dashboard.home.weatherSunny22'))
  const todayTask = ref<DashboardTaskProgress>({
    done: 12,
    total: 16,
    percentage: 75,
  })
  const pendingApproval = ref<DashboardPendingApproval>({
    count: 4,
    avatars: [
      'https://api.dicebear.com/7.x/avataaars/svg?seed=1',
      'https://api.dicebear.com/7.x/avataaars/svg?seed=2',
    ],
    extraCount: 2,
  })
  const shortcuts = ref<DashboardShortcutItem[]>(createHomeShortcuts())
  const runningInstancesLoading = ref(false)
  const runningInstances = ref<InstanceInfo[]>([])
  let currentDateTimer: ReturnType<typeof setInterval> | null = null

  // System monitoring state
  const overviewLoading = ref(false)
  const overview = ref<SystemOverviewResponse | null>(null)
  const statusLoading = ref(false)
  const status = ref<SystemStatusResponse | null>(null)
  const cpuHistory = ref<CpuPoint[]>([])
  const memoryHistory = ref<MemoryPoint[]>([])
  const networkHistory = ref<NetworkPoint[]>([])
  const diskHistory = ref<DiskPoint[]>([])
  const refreshInterval = ref<RefreshInterval>(3)
  const selectedInterface = ref('')
  const selectedDisk = ref('')
  let refreshTimer: ReturnType<typeof setInterval> | null = null

  const updateCurrentDate = () => {
    currentDate.value = dayjs().format('YYYY-MM-DD HH:mm:ss')
  }

  const startCurrentDateTicker = () => {
    updateCurrentDate()
    if (currentDateTimer) return
    currentDateTimer = setInterval(updateCurrentDate, 1000)
  }

  const stopCurrentDateTicker = () => {
    if (!currentDateTimer) return
    clearInterval(currentDateTimer)
    currentDateTimer = null
  }

  const getRunningInstances = async () => {
    runningInstancesLoading.value = true
    try {
      const { data } = await getInstances({
        page: 1,
        pageSize: RUNNING_INSTANCE_LIMIT,
        status: InstanceStatus.INSTANCE_STATUS_RUNNING,
      })
      runningInstances.value = (data?.infos || []).filter(
        (item) => item.status === InstanceStatus.INSTANCE_STATUS_RUNNING,
      )
    } finally {
      runningInstancesLoading.value = false
    }
  }

  const fetchOverview = async () => {
    overviewLoading.value = true
    try {
      const { data } = await getSystemOverview()
      overview.value = data
    } finally {
      overviewLoading.value = false
    }
  }

  const fetchStatus = async () => {
    statusLoading.value = true
    try {
      const params: SystemStatusRequest = {}
      if (selectedInterface.value) params.interfaceName = selectedInterface.value
      if (selectedDisk.value) params.diskName = selectedDisk.value

      const { data } = await getSystemStatus(params)
      status.value = data

      const netData = selectedInterface.value
        ? data.network?.selectedInterface
        : data.network?.total
      const diskIOData = selectedDisk.value
        ? data.disk?.selectedIo
        : data.disk?.totalIo

      const now = dayjs().format('HH:mm:ss')

      cpuHistory.value = [
        ...cpuHistory.value,
        {
          time: now,
          total: data.cpu?.totalPercent ?? 0,
          cores: (data.cpu?.cores || []).map((c) => c.percent),
        },
      ].slice(-MAX_HISTORY)

      memoryHistory.value = [
        ...memoryHistory.value,
        {
          time: now,
          physical: data.memory?.physicalMemory?.usedPercent ?? 0,
          physicalUsed: data.memory?.physicalMemory?.usedBytes ?? 0,
          physicalTotal: data.memory?.physicalMemory?.totalBytes ?? 0,
          swap: data.memory?.virtualMemory?.usedPercent ?? 0,
          swapUsed: data.memory?.virtualMemory?.usedBytes ?? 0,
          swapTotal: data.memory?.virtualMemory?.totalBytes ?? 0,
        },
      ].slice(-MAX_HISTORY)

      networkHistory.value = [
        ...networkHistory.value,
        {
          time: now,
          download: netData?.downloadRateBytesPerSecond ?? 0,
          upload: netData?.uploadRateBytesPerSecond ?? 0,
        },
      ].slice(-MAX_HISTORY)

      diskHistory.value = [
        ...diskHistory.value,
        {
          time: now,
          read: diskIOData?.readRateBytesPerSecond ?? 0,
          write: diskIOData?.writeRateBytesPerSecond ?? 0,
        },
      ].slice(-MAX_HISTORY)
    } finally {
      statusLoading.value = false
    }
  }

  const setSelectedInterface = (name: string) => {
    if (selectedInterface.value === name) return
    selectedInterface.value = name
    networkHistory.value = []
  }

  const setSelectedDisk = (name: string) => {
    if (selectedDisk.value === name) return
    selectedDisk.value = name
    diskHistory.value = []
  }

  const startAutoRefresh = () => {
    stopAutoRefresh()
    fetchStatus()
    refreshTimer = setInterval(fetchStatus, refreshInterval.value * 1000)
  }

  const stopAutoRefresh = () => {
    if (!refreshTimer) return
    clearInterval(refreshTimer)
    refreshTimer = null
  }

  const setRefreshInterval = (interval: RefreshInterval) => {
    refreshInterval.value = interval
    if (refreshTimer) {
      startAutoRefresh()
    }
  }

  return {
    currentDate,
    weatherText,
    todayTask,
    pendingApproval,
    shortcuts,
    runningInstancesLoading,
    runningInstances,
    overviewLoading,
    overview,
    statusLoading,
    status,
    cpuHistory,
    memoryHistory,
    networkHistory,
    diskHistory,
    refreshInterval,
    selectedInterface,
    selectedDisk,
    updateCurrentDate,
    startCurrentDateTicker,
    stopCurrentDateTicker,
    getRunningInstances,
    fetchOverview,
    fetchStatus,
    startAutoRefresh,
    stopAutoRefresh,
    setRefreshInterval,
    setSelectedInterface,
    setSelectedDisk,
  }
})
