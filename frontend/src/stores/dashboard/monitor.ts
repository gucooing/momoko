import dayjs from 'dayjs'
import { defineStore } from 'pinia'
import type {
  DashboardLogItem,
  DashboardLogLevel,
  DashboardResourceStatItem,
} from '@/stores/dashboard/types'

const LOG_PRESET_KEYS = [
  'dashboard.monitor.logs.gateway',
  'dashboard.monitor.logs.auth',
  'dashboard.monitor.logs.service',
  'dashboard.monitor.logs.db',
  'dashboard.monitor.logs.cache',
  'dashboard.monitor.logs.security',
]

const createResourceStats = (): DashboardResourceStatItem[] => [
  { labelKey: 'dashboard.monitor.cpuUsage', value: 38.01, unit: '%', color: '--el-color-primary', trend: '+2.1%' },
  { labelKey: 'dashboard.monitor.memory', value: 62.45, unit: '%', color: '--el-color-success', trend: '-0.5%' },
  { labelKey: 'dashboard.monitor.network', value: 75.01, unit: 'Mbps', color: '--el-color-warning', trend: '+12.4' },
  { labelKey: 'dashboard.monitor.activeTasks', value: 12, unit: 'Proc', color: '--el-color-danger', trend: '', trendKey: 'dashboard.monitor.stable' },
]

const createThroughputData = () => new Array(20).fill(0).map(() => Math.floor(Math.random() * 400 + 100))

export const useDashboardMonitorStore = defineStore('dashboard-monitor', () => {
  const logs = ref<DashboardLogItem[]>([])
  const resourceStats = ref<DashboardResourceStatItem[]>(createResourceStats())
  const throughputData = ref<number[]>(createThroughputData())

  const appendLog = () => {
    const level = (Math.random() > 0.85 ? (Math.random() > 0.5 ? 'WARN' : 'ERROR') : 'INFO') as DashboardLogLevel
    const contentKey = LOG_PRESET_KEYS[Math.floor(Math.random() * LOG_PRESET_KEYS.length)] || ''

    logs.value.unshift({
      time: dayjs().format('HH:mm:ss.SSS'),
      level,
      contentKey,
    })

    if (logs.value.length > 50) {
      logs.value.pop()
    }
  }

  const refreshResourceStats = () => {
    resourceStats.value = resourceStats.value.map((item) => {
      const previousValue = item.value
      const nextValue = Number((Math.random() * 100).toFixed(2))
      const diff = Number((nextValue - previousValue).toFixed(2))
      const trend = diff > 0 ? `+${diff}%` : diff < 0 ? `${diff}%` : ''

      return {
        ...item,
        value: nextValue,
        trend,
        trendKey: diff === 0 ? 'dashboard.monitor.stable' : undefined,
      }
    })
  }

  const pushThroughput = () => {
    throughputData.value.push(Math.floor(Math.random() * 400 + 100))
    throughputData.value.shift()
  }

  return {
    logs,
    resourceStats,
    throughputData,
    appendLog,
    refreshResourceStats,
    pushThroughput,
  }
})
