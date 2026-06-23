export type DashboardTagType = 'success' | 'primary' | 'warning' | 'info' | 'danger'

export interface DashboardShortcutItem {
  labelKey: string
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
  labelKey: string
  value: string
  trend: string
  icon: string
  type: 'blue' | 'orange' | 'indigo'
}

export interface DashboardChannelSalesItem {
  nameKey: string
  owner: string
  revenue: string
  achievement: number
  statusKey: string
  statusType: DashboardTagType
  color: string
}

export interface DashboardMarketShareItem {
  value: number
  nameKey: string
  colorVar: string
}

export interface DashboardTopCategoryItem {
  nameKey: string
  value: number
}

export type DashboardEventTab = 'toBeOpened' | 'inProgress' | 'review'

export interface DashboardEventItem {
  id: number
  dateKey: string
  titleKey: string
  range: string
  color: string
}

export type DashboardAnalysisTimeRange = '1y' | '2y'

export interface DashboardRevenueProfitDataItem {
  months: number[]
  revenue: number[]
  profit: number[]
  lastYear: number[]
  profitRate: number[]
}

export interface DashboardGoalProgress {
  titleKey: string
  percentage: number
  color: string
}

export interface DashboardStarManager {
  titleKey: string
  name: string
  roleKey: string
  avatar: string
}

export interface DashboardResourceStatItem {
  labelKey: string
  value: number
  unit: string
  color: string
  trend: string
  trendKey?: string
}

export type DashboardLogLevel = 'INFO' | 'WARN' | 'ERROR'

export interface DashboardLogItem {
  time: string
  level: DashboardLogLevel
  contentKey: string
}
