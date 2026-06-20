import dayjs from 'dayjs'
import { defineStore } from 'pinia'
import {
  createSub2APIAnnouncement,
  createSub2APITimelineItem,
  deleteSub2APIAnnouncement,
  deleteSub2APITimelineItem,
  getPublicSub2APIHome,
  getPublicSub2APIStats,
  getSub2APIConfig,
  getSub2APISnapshot,
  listSub2APIAnnouncements,
  listSub2APITimeline,
  syncSub2APIUsage,
  testSub2APIConnection,
  updateSub2APIAnnouncement,
  updateSub2APIConfig,
  updateSub2APITimelineItem,
} from '@/api/sub2api'
import { useThemeStore } from '@/stores/theme'
import { showRequestError } from '@/utils/request'
import type {
  CreateSub2APIAnnouncementRequest,
  CreateSub2APITimelineItemRequest,
  Sub2APIAnnouncement,
  Sub2APIConfig,
  Sub2APIHome,
  Sub2APIStats,
  Sub2APITimelineItem,
  Sub2APITopItem,
  Sub2APITrendPoint,
  Sub2APIUsageSnapshot,
  TestSub2APIConnectionResponse,
  UpdateSub2APIAnnouncementRequest,
  UpdateSub2APITimelineItemRequest,
} from '@/types/v1/sub2api'

type DateLike = Date | string | number | undefined | null

export interface Sub2APIMetricCard {
  label: string
  value: string
  detail: string
  icon: string
  tone: 'green' | 'blue' | 'amber' | 'red'
}

const createDefaultConfig = (): Sub2APIConfig => ({
  homeEnabled: false,
  syncEnabled: true,
  baseUrl: '',
  adminApiKey: '',
  consoleUrl: '',
  title: 'Sub2API',
  subtitle: '统一订阅转换与模型调用看板',
  introduction: '',
  syncIntervalMinutes: 10,
  historyDays: 30,
  pageSize: 500,
})

const numberFormatter = new Intl.NumberFormat('zh-CN')

const toNumber = (value: unknown) => {
  const num = Number(value)
  return Number.isFinite(num) ? num : 0
}

const emptyChartOption = (text = '暂无数据') => ({
  title: { text, left: 'center', top: 'middle', textStyle: { color: '#94a3b8', fontSize: 13 } },
  xAxis: { show: false },
  yAxis: { show: false },
  series: [],
})

export const useSub2APIStore = defineStore('sub2api', () => {
  const themeStore = useThemeStore()

  const config = ref<Sub2APIConfig>(createDefaultConfig())
  const configForm = reactive<Sub2APIConfig>(createDefaultConfig())
  const snapshot = ref<Sub2APIUsageSnapshot>()
  const home = ref<Sub2APIHome>()
  const stats = ref<Sub2APIStats>()
  const announcements = ref<Sub2APIAnnouncement[]>([])
  const timeline = ref<Sub2APITimelineItem[]>([])

  const configLoading = ref(false)
  const snapshotLoading = ref(false)
  const publicLoading = ref(false)
  const statsLoading = ref(false)
  const saving = ref(false)
  const testing = ref(false)
  const syncing = ref(false)
  const listLoading = ref(false)

  const applyConfig = (next?: Sub2APIConfig) => {
    const normalized = { ...createDefaultConfig(), ...(next || {}) }
    config.value = normalized
    Object.assign(configForm, normalized)
  }

  const loadConfig = async () => {
    configLoading.value = true
    try {
      const { data } = await getSub2APIConfig({})
      applyConfig(data?.config)
      return config.value
    } catch (error) {
      showRequestError(error, '获取 Sub2API 配置失败')
    } finally {
      configLoading.value = false
    }
  }

  const loadSnapshot = async () => {
    snapshotLoading.value = true
    try {
      const { data } = await getSub2APISnapshot({})
      snapshot.value = data?.snapshot
      return snapshot.value
    } catch (error) {
      showRequestError(error, '获取 Sub2API 快照失败')
    } finally {
      snapshotLoading.value = false
    }
  }

  const loadPublicHome = async () => {
    publicLoading.value = true
    try {
      const { data } = await getPublicSub2APIHome({})
      home.value = data?.home
      return home.value
    } catch (error) {
      showRequestError(error, '获取 Sub2API 首页失败')
    } finally {
      publicLoading.value = false
    }
  }

  const loadStats = async (rangeDays: number) => {
    statsLoading.value = true
    try {
      const { data } = await getPublicSub2APIStats({ rangeDays })
      stats.value = data?.stats
      return stats.value
    } catch (error) {
      showRequestError(error, '获取 Sub2API 统计失败')
    } finally {
      statsLoading.value = false
    }
  }

  const loadAdmin = async () => {
    await Promise.all([loadConfig(), loadSnapshot(), loadAnnouncements(), loadTimeline()])
  }

  const saveConfig = async () => {
    saving.value = true
    try {
      const { data } = await updateSub2APIConfig({ config: { ...configForm } })
      applyConfig(data?.config)
      return true
    } catch (error) {
      showRequestError(error, '保存 Sub2API 配置失败')
      return false
    } finally {
      saving.value = false
    }
  }

  const testConfig = async (): Promise<TestSub2APIConnectionResponse | undefined> => {
    testing.value = true
    try {
      const { data } = await testSub2APIConnection({ config: { ...configForm } })
      return data
    } catch (error) {
      showRequestError(error, '测试 Sub2API 连接失败')
    } finally {
      testing.value = false
    }
  }

  const syncUsage = async (full = false) => {
    syncing.value = true
    try {
      const { data } = await syncSub2APIUsage({ full })
      snapshot.value = data?.snapshot
      return true
    } catch (error) {
      showRequestError(error, '同步 Sub2API 使用记录失败')
      return false
    } finally {
      syncing.value = false
    }
  }

  // ---------- 公告 / 时间线 ----------
  const loadAnnouncements = async () => {
    try {
      const { data } = await listSub2APIAnnouncements()
      announcements.value = data?.announcements || []
    } catch (error) {
      showRequestError(error, '获取公告失败')
    }
  }

  const loadTimeline = async () => {
    try {
      const { data } = await listSub2APITimeline()
      timeline.value = data?.timeline || []
    } catch (error) {
      showRequestError(error, '获取时间线失败')
    }
  }

  const saveAnnouncement = async (
    payload: CreateSub2APIAnnouncementRequest & { id?: string },
  ) => {
    listLoading.value = true
    try {
      if (payload.id) {
        await updateSub2APIAnnouncement(payload as UpdateSub2APIAnnouncementRequest)
      } else {
        await createSub2APIAnnouncement(payload)
      }
      await loadAnnouncements()
      return true
    } catch (error) {
      showRequestError(error, '保存公告失败')
      return false
    } finally {
      listLoading.value = false
    }
  }

  const removeAnnouncement = async (id: string) => {
    try {
      await deleteSub2APIAnnouncement(id)
      await loadAnnouncements()
      return true
    } catch (error) {
      showRequestError(error, '删除公告失败')
      return false
    }
  }

  const saveTimelineItem = async (
    payload: CreateSub2APITimelineItemRequest & { id?: string },
  ) => {
    listLoading.value = true
    try {
      if (payload.id) {
        await updateSub2APITimelineItem(payload as UpdateSub2APITimelineItemRequest)
      } else {
        await createSub2APITimelineItem(payload)
      }
      await loadTimeline()
      return true
    } catch (error) {
      showRequestError(error, '保存时间线失败')
      return false
    } finally {
      listLoading.value = false
    }
  }

  const removeTimelineItem = async (id: string) => {
    try {
      await deleteSub2APITimelineItem(id)
      await loadTimeline()
      return true
    } catch (error) {
      showRequestError(error, '删除时间线失败')
      return false
    }
  }

  // ---------- 格式化 ----------
  const formatNumber = (value: unknown) => numberFormatter.format(toNumber(value))
  const formatPercent = (value: unknown) => `${toNumber(value).toFixed(1)}%`
  const formatLatency = (value: unknown) => {
    const ms = toNumber(value)
    if (ms >= 1000) return `${(ms / 1000).toFixed(2)} s`
    return `${Math.round(ms)} ms`
  }
  const formatToken = (value: unknown) => {
    const count = toNumber(value)
    if (count >= 100000000) return `${(count / 100000000).toFixed(2)} 亿`
    if (count >= 10000) return `${(count / 10000).toFixed(1)} 万`
    return formatNumber(count)
  }
  const formatThroughput = (value: unknown) => {
    const tps = toNumber(value)
    if (tps >= 10000) return `${(tps / 10000).toFixed(1)}万`
    if (tps >= 100) return `${Math.round(tps)}`
    return tps.toFixed(1)
  }
  const formatDateTime = (value: DateLike) => {
    if (!value) return '-'
    const time = dayjs(value)
    return time.isValid() ? time.format('YYYY-MM-DD HH:mm:ss') : '-'
  }

  const statusText = (value?: string) => {
    switch (value) {
      case 'syncing':
        return '同步中'
      case 'success':
        return '正常'
      case 'error':
        return '异常'
      default:
        return '待同步'
    }
  }
  const statusType = (value?: string) => {
    switch (value) {
      case 'syncing':
        return 'warning'
      case 'success':
        return 'success'
      case 'error':
        return 'danger'
      default:
        return 'info'
    }
  }

  const buildMetricCards = (data?: Sub2APIUsageSnapshot): Sub2APIMetricCard[] => {
    return [
      {
        label: 'Token 生成速度',
        value: `${formatThroughput(data?.recentTps)} token/s`,
        detail: '按请求平均 · 不含缓存',
        icon: 'HOutline:BoltIcon',
        tone: 'blue',
      },
      {
        label: '平均延迟',
        value: formatLatency(data?.todayAverageLatencyMs),
        detail: `全量 ${formatLatency(data?.averageLatencyMs)}`,
        icon: 'HOutline:ClockIcon',
        tone: 'amber',
      },
      {
        label: '今日成功率',
        value: formatPercent(data?.todaySuccessRate),
        detail: `${formatNumber(data?.todaySuccessCount)} / ${formatNumber(data?.todayRequestCount)} 次`,
        icon: 'HOutline:CheckCircleIcon',
        tone: 'green',
      },
    ]
  }

  const adminMetricCards = computed(() => buildMetricCards(snapshot.value))
  const publicMetricCards = computed(() => buildMetricCards(home.value?.snapshot))

  const trendOption = (trend: Sub2APITrendPoint[] = []) => {
    if (!trend.length) return emptyChartOption()
    const isDark = themeStore.isDarkTheme
    const axisColor = isDark ? '#94a3b8' : '#64748b'
    return {
      color: ['#3b82f6', '#8b5cf6', '#10b981'],
      tooltip: { trigger: 'axis', confine: true },
      legend: { top: 0, textStyle: { color: axisColor } },
      grid: { left: 44, right: 48, top: 36, bottom: 28 },
      xAxis: {
        type: 'category',
        data: trend.map((item) => {
          const date = item.date || ''
          // 按天的日期(YYYY-MM-DD)截掉年份；日内标签(HH:MM)原样展示
          return date.length >= 10 ? date.slice(5) : date
        }),
        axisLabel: { color: axisColor, fontSize: 11 },
      },
      yAxis: [
        { type: 'value', splitLine: { lineStyle: { color: isDark ? '#1f2937' : '#eef2f7' } }, axisLabel: { color: axisColor, fontSize: 11 } },
        { type: 'value', max: 100, axisLabel: { formatter: '{value}%', color: axisColor, fontSize: 11 }, splitLine: { show: false } },
      ],
      series: [
        { name: '请求量', type: 'bar', barMaxWidth: 18, itemStyle: { borderRadius: [4, 4, 0, 0] }, data: trend.map((item) => item.requestCount) },
        { name: 'Token', type: 'line', smooth: true, symbol: 'none', data: trend.map((item) => item.tokenCount) },
        { name: '成功率', type: 'line', yAxisIndex: 1, smooth: true, symbol: 'none', data: trend.map((item) => Number(toNumber(item.successRate).toFixed(2))) },
      ],
    }
  }

  const topOption = (items: Sub2APITopItem[] = [], title = '请求量') => {
    if (!items.length) return emptyChartOption()
    const isDark = themeStore.isDarkTheme
    const axisColor = isDark ? '#94a3b8' : '#64748b'
    const chartItems = [...items].reverse()
    return {
      color: ['#6366f1'],
      tooltip: { trigger: 'axis' },
      grid: { left: 96, right: 18, top: 16, bottom: 20 },
      xAxis: { type: 'value', axisLabel: { color: axisColor, fontSize: 11 }, splitLine: { lineStyle: { color: isDark ? '#1f2937' : '#eef2f7' } } },
      yAxis: { type: 'category', data: chartItems.map((item) => item.name || '未标记'), axisLabel: { color: axisColor, fontSize: 11 } },
      series: [
        { name: title, type: 'bar', barMaxWidth: 16, itemStyle: { borderRadius: [0, 4, 4, 0] }, data: chartItems.map((item) => item.requestCount) },
      ],
    }
  }

  const adminTrendOption = computed(() => trendOption(snapshot.value?.trend))
  const adminModelOption = computed(() => topOption(snapshot.value?.modelUsage, '模型请求量'))
  const adminEndpointOption = computed(() => topOption(snapshot.value?.endpointUsage, '接口请求量'))
  const publicTrendOption = computed(() => trendOption(home.value?.snapshot?.trend))
  const statsTrendOption = computed(() => trendOption(stats.value?.trend))

  // 当日“成功率 + 生成速度”随时间移动曲线（time 轴，短时段也能铺满）
  const todaySeriesOption = computed(() => {
    const series = home.value?.snapshot?.todaySeries || []
    if (!series.length) return emptyChartOption('今日暂无数据')
    const isDark = themeStore.isDarkTheme
    const axisColor = isDark ? '#94a3b8' : '#64748b'
    const toMs = (t: unknown) => (t ? new Date(t as string | Date).getTime() : 0)
    return {
      color: ['#10b981', '#3b82f6'],
      tooltip: {
        trigger: 'axis',
        confine: true,
      },
      legend: { top: 0, textStyle: { color: axisColor } },
      grid: { left: 48, right: 52, top: 36, bottom: 28 },
      xAxis: {
        type: 'time',
        axisLabel: { color: axisColor, fontSize: 11, hideOverlap: true },
      },
      yAxis: [
        { type: 'value', name: 'token/s', nameTextStyle: { color: axisColor }, axisLabel: { color: axisColor, fontSize: 11 }, splitLine: { lineStyle: { color: isDark ? '#1f2937' : '#eef2f7' } } },
        { type: 'value', name: '成功率', max: 100, position: 'right', axisLabel: { formatter: '{value}%', color: axisColor, fontSize: 11 }, splitLine: { show: false } },
      ],
      series: [
        { name: '成功率', type: 'line', yAxisIndex: 1, smooth: true, symbol: 'none', areaStyle: { opacity: 0.06 }, data: series.map((p) => [toMs(p.time), Number(toNumber(p.successRate).toFixed(2))]) },
        { name: '生成速度', type: 'line', yAxisIndex: 0, smooth: true, symbol: 'none', data: series.map((p) => [toMs(p.time), Number(toNumber(p.avgTps).toFixed(1))]) },
      ],
    }
  })

  const statsMetricCards = computed<Sub2APIMetricCard[]>(() => {
    const s = stats.value
    return [
      { label: '请求数', value: formatNumber(s?.requestCount), detail: s?.rangeLabel || '', icon: 'HOutline:ArrowPathRoundedSquareIcon', tone: 'blue' },
      { label: 'Token', value: formatToken(s?.tokenCount), detail: '区间累计', icon: 'HOutline:CircleStackIcon', tone: 'blue' },
      { label: '成功率', value: formatPercent(s?.successRate), detail: `成功 ${formatNumber(s?.successCount)}`, icon: 'HOutline:CheckCircleIcon', tone: 'green' },
      { label: '平均延迟', value: formatLatency(s?.averageLatencyMs), detail: '成功请求均值', icon: 'HOutline:ClockIcon', tone: 'amber' },
      { label: 'Token 生成速度', value: `${formatThroughput(s?.averageTps)} token/s`, detail: '按请求平均 · 不含缓存/mini', icon: 'HOutline:BoltIcon', tone: 'red' },
    ]
  })

  return {
    config,
    configForm,
    snapshot,
    home,
    stats,
    announcements,
    timeline,
    configLoading,
    snapshotLoading,
    publicLoading,
    statsLoading,
    saving,
    testing,
    syncing,
    listLoading,
    adminMetricCards,
    publicMetricCards,
    statsMetricCards,
    adminTrendOption,
    adminModelOption,
    adminEndpointOption,
    publicTrendOption,
    statsTrendOption,
    todaySeriesOption,
    loadConfig,
    loadSnapshot,
    loadPublicHome,
    loadStats,
    loadAdmin,
    saveConfig,
    testConfig,
    syncUsage,
    loadAnnouncements,
    loadTimeline,
    saveAnnouncement,
    removeAnnouncement,
    saveTimelineItem,
    removeTimelineItem,
    formatNumber,
    formatPercent,
    formatLatency,
    formatToken,
    formatThroughput,
    formatDateTime,
    statusText,
    statusType,
  }
})
