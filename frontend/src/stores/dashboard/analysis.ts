import { defineStore } from 'pinia'
import type {
  DashboardAnalysisTimeRange,
  DashboardBusinessStatItem,
  DashboardChannelSalesItem,
  DashboardEventItem,
  DashboardEventTab,
  DashboardGoalProgress,
  DashboardMarketShareItem,
  DashboardRevenueProfitDataItem,
  DashboardStarManager,
  DashboardTopCategoryItem,
} from '@/stores/dashboard/types'

const createBusinessStats = (): DashboardBusinessStatItem[] => [
  {
    labelKey: 'dashboard.analysis.totalRevenue',
    value: '￥1,284,500',
    trend: '+15.2%',
    icon: 'HSolid:BanknotesIcon',
    type: 'blue',
  },
  {
    labelKey: 'dashboard.analysis.totalOrders',
    value: '8,429',
    trend: '+8.4%',
    icon: 'HSolid:ShoppingCartIcon',
    type: 'orange',
  },
  {
    labelKey: 'dashboard.analysis.newMembers',
    value: '1,562',
    trend: '+22.1%',
    icon: 'HSolid:UserPlusIcon',
    type: 'indigo',
  },
]

const createChannelSales = (): DashboardChannelSalesItem[] => [
  {
    nameKey: 'dashboard.analysis.channels.mobileApp',
    owner: 'David Fan',
    revenue: '542,800',
    achievement: 92,
    statusKey: 'dashboard.analysis.channelStatus.leadingGrowth',
    statusType: 'success',
    color: '#8b5cf6',
  },
  {
    nameKey: 'dashboard.analysis.channels.tmall',
    owner: 'Alice Zhang',
    revenue: '320,400',
    achievement: 88,
    statusKey: 'dashboard.analysis.channelStatus.stable',
    statusType: 'primary',
    color: '#ef4444',
  },
  {
    nameKey: 'dashboard.analysis.channels.douyin',
    owner: 'Mike Lee',
    revenue: '281,000',
    achievement: 100,
    statusKey: 'dashboard.analysis.channelStatus.booming',
    statusType: 'warning',
    color: '#f59e0b',
  },
  {
    nameKey: 'dashboard.analysis.channels.jd',
    owner: 'Lisa Wang',
    revenue: '240,000',
    achievement: 80,
    statusKey: 'dashboard.analysis.channelStatus.newStore',
    statusType: 'info',
    color: '#3b82f6',
  },
]

const createMarketShares = (): DashboardMarketShareItem[] => [
  { value: 45, nameKey: 'dashboard.analysis.market.northAmerica', colorVar: '--el-color-primary' },
  { value: 25, nameKey: 'dashboard.analysis.market.europe', colorVar: '--el-color-warning' },
  { value: 20, nameKey: 'dashboard.analysis.market.asiaPacific', colorVar: '--el-color-success' },
  { value: 10, nameKey: 'dashboard.analysis.market.others', colorVar: '--el-color-info' },
]

const createTopCategories = (): DashboardTopCategoryItem[] => [
  { nameKey: 'dashboard.analysis.categories.electronics', value: 820 },
  { nameKey: 'dashboard.analysis.categories.outdoor', value: 732 },
  { nameKey: 'dashboard.analysis.categories.home', value: 601 },
  { nameKey: 'dashboard.analysis.categories.beauty', value: 534 },
  { nameKey: 'dashboard.analysis.categories.fashion', value: 490 },
]

const createAnalysisData = (): Record<DashboardAnalysisTimeRange, DashboardRevenueProfitDataItem> => ({
  '1y': {
    months: Array.from({ length: 12 }, (_, i) => i + 1),
    revenue: [120, 132, 101, 134, 290, 230, 210, 250, 220, 280, 310, 330],
    profit: [80, 92, 70, 84, 150, 130, 110, 140, 120, 160, 190, 240],
    lastYear: [100, 110, 95, 120, 200, 180, 190, 210, 180, 230, 260, 300],
    profitRate: [66, 69, 69, 62, 52, 56, 52, 56, 54, 57, 61, 72],
  },
  '2y': {
    months: Array.from({ length: 24 }, (_, i) => i + 1),
    revenue: Array.from({ length: 24 }, () => Math.floor(Math.random() * 300 + 50)),
    profit: Array.from({ length: 24 }, () => Math.floor(Math.random() * 200 + 50)),
    lastYear: Array.from({ length: 24 }, () => Math.floor(Math.random() * 250 + 50)),
    profitRate: Array.from({ length: 24 }, () => Math.floor(Math.random() * 100)),
  },
})

const createEvents = (): Record<DashboardEventTab, DashboardEventItem[]> => ({
  toBeOpened: [
    { id: 1, dateKey: 'dashboard.analysis.events.jan20', titleKey: 'dashboard.analysis.events.newYearFestival', range: '01.20 - 02.10', color: '#ef4444' },
    { id: 2, dateKey: 'dashboard.analysis.events.feb14', titleKey: 'dashboard.analysis.events.valentine', range: '02.10 - 02.15', color: '#f99c7d' },
    { id: 5, dateKey: 'dashboard.analysis.events.mar01', titleKey: 'dashboard.analysis.events.springLaunch', range: '03.01 - 03.05', color: '#3b82f6' },
  ],
  inProgress: [
    { id: 3, dateKey: 'dashboard.analysis.events.mar01', titleKey: 'dashboard.analysis.events.springLaunch', range: '03.01 - 03.05', color: '#3b82f6' },
  ],
  review: [
    {
      id: 4,
      dateKey: 'dashboard.analysis.events.dec25',
      titleKey: 'dashboard.analysis.events.christmasReview',
      range: '12.20 - 12.26',
      color: '#10b981',
    },
  ],
})

export const useDashboardAnalysisStore = defineStore('dashboard-analysis', () => {
  const businessStats = ref<DashboardBusinessStatItem[]>(createBusinessStats())
  const channelSales = ref<DashboardChannelSalesItem[]>(createChannelSales())
  const marketShares = ref<DashboardMarketShareItem[]>(createMarketShares())
  const topCategories = ref<DashboardTopCategoryItem[]>(createTopCategories())

  const goalProgress = ref<DashboardGoalProgress>({
    titleKey: 'dashboard.analysis.goalTitle',
    percentage: 76.4,
    color: '#f97316',
  })

  const operationManager = ref<DashboardStarManager>({
    titleKey: 'dashboard.analysis.operationManagerTitle',
    name: 'David Fan',
    roleKey: 'dashboard.analysis.operationManagerRole',
    avatar: 'https://api.dicebear.com/7.x/avataaars/svg?seed=Operation',
  })

  const analysisTimeRange = ref<DashboardAnalysisTimeRange>('1y')
  const analysisData = ref<Record<DashboardAnalysisTimeRange, DashboardRevenueProfitDataItem>>(
    createAnalysisData(),
  )

  const currentEventTab = ref<DashboardEventTab>('toBeOpened')
  const eventsByTab = ref<Record<DashboardEventTab, DashboardEventItem[]>>(createEvents())

  const events = computed(() => eventsByTab.value[currentEventTab.value])

  const setAnalysisTimeRange = (range: DashboardAnalysisTimeRange) => {
    analysisTimeRange.value = range
  }

  const setCurrentEventTab = (tab: DashboardEventTab) => {
    currentEventTab.value = tab
  }

  const refreshData = () => {
    const range = analysisTimeRange.value
    analysisData.value[range] = {
      months: analysisData.value[range].months,
      revenue: analysisData.value[range].revenue.map((item) => item + Math.floor(Math.random() * 20 - 10)),
      profit: analysisData.value[range].profit.map((item) => item + Math.floor(Math.random() * 10 - 5)),
      lastYear: analysisData.value[range].lastYear.map((item) => item + Math.floor(Math.random() * 10 - 5)),
      profitRate: analysisData.value[range].profitRate.map((item) =>
        Math.min(100, Math.max(0, item + Math.floor(Math.random() * 5 - 2))),
      ),
    }
  }

  return {
    businessStats,
    channelSales,
    marketShares,
    topCategories,
    goalProgress,
    operationManager,
    analysisTimeRange,
    analysisData,
    currentEventTab,
    eventsByTab,
    events,
    setAnalysisTimeRange,
    setCurrentEventTab,
    refreshData,
  }
})
