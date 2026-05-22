import dayjs from 'dayjs'
import { defineStore } from 'pinia'
import { getInstances } from '@/api/instance'
import { InstanceStatus, type InstanceInfo } from '@/types/v1/instance'
import type {
  DashboardPendingApproval,
  DashboardShortcutItem,
  DashboardTaskProgress,
  DashboardWelcomeStatCard,
} from '@/stores/dashboard/types'

const createHomeShortcuts = (): DashboardShortcutItem[] => [
  { label: '分析', icon: 'HOutline:ChartBarIcon', color: '#ef4444', routePath: '/dashboard/analysis' },
  { label: '监控', icon: 'HOutline:EyeIcon', color: '#4f46e5', routePath: '/dashboard/monitor' },
  { label: '用户', icon: 'HOutline:UserGroupIcon', color: '#f59e0b', routePath: '/system/user' },
  { label: '角色', icon: 'HOutline:IdentificationIcon', color: '#10b981', routePath: '/system/role' },
  { label: '应用列表', icon: 'HOutline:Squares2X2Icon', color: '#ec4899', routePath: '/instance/list' },
  { label: '终端', icon: 'HOutline:CommandLineIcon', color: '#8b5cf6', routePath: '/instance/terminal' },
  { label: '文件管理', icon: 'HOutline:FolderIcon', color: '#06b6d4', routePath: '/file/index' },
]

const createWelcomeCards = (): DashboardWelcomeStatCard[] => [
  {
    label: '本周任务完成',
    value: '52',
    trend: '+12%',
    trendType: 'up',
    color: '#6366f1',
    icon: 'HOutline:CheckCircleIcon',
    chartData: [30, 40, 35, 50, 49, 60, 52],
  },
  {
    label: '项目活跃度',
    value: '84%',
    trend: '+5%',
    trendType: 'up',
    color: '#10b981',
    icon: 'HOutline:ArrowTrendingUpIcon',
    chartData: [70, 75, 72, 80, 78, 85, 84],
  },
  {
    label: '待办处理率',
    value: '92%',
    trend: '-2%',
    trendType: 'down',
    color: '#f59e0b',
    icon: 'HOutline:ClipboardDocumentListIcon',
    chartData: [95, 94, 96, 92, 93, 91, 92],
  },
  {
    label: '团队协作值',
    value: '76',
    trend: '+18%',
    trendType: 'up',
    color: '#ef4444',
    icon: 'HOutline:UserGroupIcon',
    chartData: [50, 55, 60, 65, 70, 75, 76],
  },
]

const RUNNING_INSTANCE_LIMIT = 6

export const useDashboardHomeStore = defineStore('dashboard-home', () => {
  const currentDate = ref('')
  const weatherText = ref('晴 22℃')
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
  const welcomeCards = ref<DashboardWelcomeStatCard[]>(createWelcomeCards())
  const runningInstancesLoading = ref(false)
  const runningInstances = ref<InstanceInfo[]>([])
  let currentDateTimer: ReturnType<typeof setInterval> | null = null

  const updateCurrentDate = () => {
    currentDate.value = dayjs().format('YYYY-MM-DD HH:mm:ss')
  }

  const startCurrentDateTicker = () => {
    updateCurrentDate()

    if (currentDateTimer) return

    currentDateTimer = setInterval(() => {
      updateCurrentDate()
    }, 1000)
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

  return {
    currentDate,
    weatherText,
    todayTask,
    pendingApproval,
    shortcuts,
    welcomeCards,
    runningInstancesLoading,
    runningInstances,
    updateCurrentDate,
    startCurrentDateTicker,
    stopCurrentDateTicker,
    getRunningInstances,
  }
})
