export type DashboardTagType = 'success' | 'primary' | 'warning' | 'info' | 'danger'

export interface DashboardShortcutItem {
  label: string
  icon: string
  color: string
  routePath: string
}

export interface DashboardWelcomeStatCard {
  label: string
  value: string
  trend: string
  trendType: 'up' | 'down'
  color: string
  icon: string
  chartData: number[]
}

export interface DashboardTaskProgress {
  done: number
  total: number
  percentage: number
}

export interface DashboardPendingApproval {
  count: number
  avatars: string[]
  extraCount: number
}

export interface DashboardBusinessStatItem {
  label: string
  value: string
  trend: string
  icon: string
  type: 'blue' | 'orange' | 'indigo'
}

export interface DashboardChannelSalesItem {
  name: string
  owner: string
  revenue: string
  achievement: number
  status: string
  statusType: DashboardTagType
  color: string
}

export interface DashboardMarketShareItem {
  value: number
  name: string
  colorVar: string
}

export interface DashboardTopCategoryItem {
  name: string
  value: number
}

export type DashboardEventTab = 'toBeOpened' | 'inProgress' | 'review'

export interface DashboardEventItem {
  id: number
  date: string
  title: string
  range: string
  color: string
}

export type DashboardAnalysisTimeRange = '1y' | '2y'

export interface DashboardRevenueProfitDataItem {
  months: string[]
  revenue: number[]
  profit: number[]
  lastYear: number[]
  profitRate: number[]
}

export interface DashboardGoalProgress {
  title: string
  percentage: number
  color: string
}

export interface DashboardStarManager {
  title: string
  name: string
  role: string
  avatar: string
}

export interface DashboardResourceStatItem {
  label: string
  value: number
  unit: string
  color: string
  trend: string
}

export type DashboardLogLevel = 'INFO' | 'WARN' | 'ERROR'

export interface DashboardLogItem {
  time: string
  level: DashboardLogLevel
  content: string
}

