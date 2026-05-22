import dayjs from 'dayjs'
import { defineStore } from 'pinia'
import type {
  DashboardLogItem,
  DashboardLogLevel,
  DashboardResourceStatItem,
} from '@/stores/dashboard/types'

const LOG_PRESETS = [
  'Gateway: Incoming request from 182.16.4.21',
  'Auth: Token validation successful',
  'Service: Node-14 reported latency > 200ms',
  'DB: Slow query detected (152ms)',
  'Cache: Purge successful for key: user_profile_882',
  'Security: Multiple login attempts detected',
]

const createResourceStats = (): DashboardResourceStatItem[] => [
  { label: 'CPU Usage', value: 38.01, unit: '%', color: '--el-color-primary', trend: '+2.1%' },
  { label: 'Memory', value: 62.45, unit: '%', color: '--el-color-success', trend: '-0.5%' },
  { label: 'Network', value: 75.01, unit: 'Mbps', color: '--el-color-warning', trend: '+12.4' },
  { label: 'Active Tasks', value: 12, unit: 'Proc', color: '--el-color-danger', trend: 'Stable' },
]

const createThroughputData = () => new Array(20).fill(0).map(() => Math.floor(Math.random() * 400 + 100))

export const useDashboardMonitorStore = defineStore('dashboard-monitor', () => {
  const logs = ref<DashboardLogItem[]>([])
  const resourceStats = ref<DashboardResourceStatItem[]>(createResourceStats())
  const throughputData = ref<number[]>(createThroughputData())

  const appendLog = () => {
    const level = (Math.random() > 0.85 ? (Math.random() > 0.5 ? 'WARN' : 'ERROR') : 'INFO') as DashboardLogLevel
    const content = LOG_PRESETS[Math.floor(Math.random() * LOG_PRESETS.length)] || ''

    logs.value.unshift({
      time: dayjs().format('HH:mm:ss.SSS'),
      level,
      content,
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
      const trend = diff > 0 ? `+${diff}%` : diff < 0 ? `${diff}%` : 'Stable'

      return {
        ...item,
        value: nextValue,
        trend,
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
