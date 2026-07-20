import dayjs from 'dayjs'
import { defineStore } from 'pinia'
import { useWindowSize } from '@vueuse/core'
import {
  createSub2APIAnnouncement,
  createSub2APITimelineItem,
  deleteSub2APIAnnouncement,
  deleteSub2APITimelineItem,
  getPublicSub2APIHome,
  getPublicSub2APIOverview,
  getPublicSub2APIStats,
  getSub2APIAdminTop,
  getSub2APIAdminTotals,
  getSub2APIAdminTrend,
  getSub2APIConfig,
  getSub2APIRecentRequests,
  getSub2APISnapshot,
  listSub2APIAnnouncements,
  listSub2APITimeline,
  syncSub2APIUsage,
  testSub2APIConnection,
  updateSub2APIAnnouncement,
  updateSub2APIConfig,
  updateSub2APITimelineItem,
} from '@/api/sub2api'
import { getCurrentLocale, translate as t } from '@/locales'
import { useThemeStore } from '@/stores/theme'
import { showRequestError } from '@/utils/request'
import type {
  CreateSub2APIAnnouncementRequest,
  CreateSub2APITimelineItemRequest,
  GetPublicSub2APIOverviewResponse,
  GetSub2APIAdminTotalsResponse,
  Sub2APIAnnouncement,
  Sub2APIConfig,
  Sub2APIGroup,
  Sub2APIHome,
  Sub2APIRecentRequest,
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

// 最近请求多维度筛选：字段缺省 = 该维度不参与过滤（对应后端 proto3 optional）
export interface Sub2APIRecentFilter {
  model?: string
  groupName?: string
  accountName?: string
  outcome?: string
}

const createDefaultConfig = (): Sub2APIConfig => ({
  homeEnabled: false,
  syncEnabled: true,
  baseUrl: '',
  adminApiKey: '',
  consoleUrl: '',
  title: 'Sub2API',
  subtitle: t('sub2api.store.defaultSubtitle'),
  introduction: '',
  syncIntervalMinutes: 10,
  historyDays: 30,
  pageSize: 500,
  allowedSrcHosts: [],
  imageEnabled: true,
  srcHostWhitelistEnabled: false,
  publicGroups: [],
})

const toNumber = (value: unknown) => {
  const num = Number(value)
  return Number.isFinite(num) ? num : 0
}

const emptyChartOption = (text = t('sub2api.store.noData')) => ({
  title: { text, left: 'center', top: 'middle', textStyle: { color: '#94a3b8', fontSize: 13 } },
  xAxis: { show: false },
  yAxis: { show: false },
  series: [],
})

export const useSub2APIStore = defineStore('sub2api', () => {
  const themeStore = useThemeStore()
  const { width: viewportWidth } = useWindowSize()
  const isCompactChart = computed(() => viewportWidth.value <= 640)

  const config = ref<Sub2APIConfig>(createDefaultConfig())
  const configForm = reactive<Sub2APIConfig>(createDefaultConfig())
  const groups = ref<Sub2APIGroup[]>([])
  const snapshot = ref<Sub2APIUsageSnapshot>()
  const home = ref<Sub2APIHome>()
  // 公开首页今日概览：独立于 home 元信息，前端并行拉取、渐进填充
  const publicOverview = ref<GetPublicSub2APIOverviewResponse>()
  const stats = ref<Sub2APIStats>()
  // 管理端概览：三个面板独立拉取（totals/trend/top），各自 loading（禁单请求大聚合）
  const adminTotals = ref<GetSub2APIAdminTotalsResponse>()
  const adminTrend = ref<Sub2APITrendPoint[]>([])
  const adminModels = ref<Sub2APITopItem[]>([])
  const adminGroups = ref<Sub2APITopItem[]>([])
  // 最近请求：按时间区间分页 + 多维度筛选（独立于概览统计，翻页只刷新本列表）
  const adminRecent = ref<Sub2APIRecentRequest[]>([])
  const adminRecentTotal = ref(0)
  const recentPage = ref(1)
  const recentPageSize = ref(10)
  const announcements = ref<Sub2APIAnnouncement[]>([])
  const timeline = ref<Sub2APITimelineItem[]>([])

  const configLoading = ref(false)
  const snapshotLoading = ref(false)
  const publicLoading = ref(false)
  const publicOverviewLoading = ref(false)
  const statsLoading = ref(false)
  const adminTotalsLoading = ref(false)
  const adminTrendLoading = ref(false)
  const adminTopLoading = ref(false)
  const recentLoading = ref(false)
  const saving = ref(false)
  const testing = ref(false)
  const syncing = ref(false)
  const listLoading = ref(false)

  const applyConfig = (next?: Sub2APIConfig) => {
    const normalized = { ...createDefaultConfig(), ...(next || {}) }
    if (!Array.isArray(normalized.publicGroups)) normalized.publicGroups = []
    if (!Array.isArray(normalized.allowedSrcHosts)) normalized.allowedSrcHosts = []
    config.value = normalized
    Object.assign(configForm, normalized)
  }

  const loadConfig = async () => {
    configLoading.value = true
    try {
      const { data } = await getSub2APIConfig({})
      const list = data?.groups || []
      groups.value = list
      const cfg = { ...(data?.config || {}) } as Sub2APIConfig
      const activeEnabled = list
        .filter((g) => g && !g.deleted && g.publicEnabled && g.name)
        .map((g) => g.name)
      const deletedEnabled = list.some((g) => g?.deleted && g.publicEnabled)
      cfg.publicGroups = deletedEnabled
        ? [...activeEnabled, '__deleted__']
        : activeEnabled
      applyConfig(cfg)
      return config.value
    } catch (error) {
      showRequestError(error, t('sub2api.store.loadConfigFailed'))
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
      showRequestError(error, t('sub2api.store.loadSnapshotFailed'))
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
      showRequestError(error, t('sub2api.store.loadHomeFailed'))
    } finally {
      publicLoading.value = false
    }
  }

  // 公开首页今日概览：状态 + 今日标量 + 今日曲线（独立请求，供页面并行渐进渲染）
  const loadPublicOverview = async () => {
    publicOverviewLoading.value = true
    try {
      const { data } = await getPublicSub2APIOverview({})
      publicOverview.value = data
      return data
    } catch (error) {
      showRequestError(error, t('sub2api.store.loadOverviewFailed'))
    } finally {
      publicOverviewLoading.value = false
    }
  }

  const loadStats = async (rangeDays: number) => {
    statsLoading.value = true
    try {
      const { data } = await getPublicSub2APIStats({ rangeDays })
      stats.value = data?.stats
      return stats.value
    } catch (error) {
      showRequestError(error, t('sub2api.store.loadStatsFailed'))
    } finally {
      statsLoading.value = false
    }
  }

  // 管理端概览按模块拆分：totals / trend / top 各自独立请求 + loading（时间段 Unix 毫秒，精度到分钟）
  const loadAdminTotals = async (startTime: number, endTime: number) => {
    adminTotalsLoading.value = true
    try {
      const { data } = await getSub2APIAdminTotals({ startTime, endTime })
      adminTotals.value = data
      return data
    } catch (error) {
      showRequestError(error, t('sub2api.store.loadAdminStatsFailed'))
    } finally {
      adminTotalsLoading.value = false
    }
  }

  const loadAdminTrend = async (startTime: number, endTime: number) => {
    adminTrendLoading.value = true
    try {
      const { data } = await getSub2APIAdminTrend({ startTime, endTime })
      adminTrend.value = data?.trend || []
      return adminTrend.value
    } catch (error) {
      showRequestError(error, t('sub2api.store.loadAdminStatsFailed'))
    } finally {
      adminTrendLoading.value = false
    }
  }

  // limit<=0 返回全部维度行（DB 侧 GroupBy，行数受不同模型/分组数约束）；图表侧再截 TopN。
  const loadAdminTop = async (startTime: number, endTime: number, limit = 0) => {
    adminTopLoading.value = true
    try {
      const { data } = await getSub2APIAdminTop({ startTime, endTime, limit })
      adminModels.value = data?.models || []
      adminGroups.value = data?.groups || []
      return data
    } catch (error) {
      showRequestError(error, t('sub2api.store.loadAdminStatsFailed'))
    } finally {
      adminTopLoading.value = false
    }
  }

  // 管理端最近请求：按时间区间分页 + 多维度筛选（page 从 1 起；filter 指针字段缺省=该维度不过滤）
  const loadAdminRecent = async (
    startTime: number,
    endTime: number,
    page = recentPage.value,
    filter: Sub2APIRecentFilter = {},
  ) => {
    recentLoading.value = true
    try {
      const { data } = await getSub2APIRecentRequests({
        startTime,
        endTime,
        page,
        pageSize: recentPageSize.value,
        ...filter,
      })
      adminRecent.value = data?.recentRequests || []
      adminRecentTotal.value = Number(data?.total || 0)
      recentPage.value = page
      return adminRecent.value
    } catch (error) {
      showRequestError(error, t('sub2api.store.loadRecentFailed'))
    } finally {
      recentLoading.value = false
    }
  }

  const loadAdmin = async () => {
    await Promise.all([loadConfig(), loadSnapshot(), loadAnnouncements(), loadTimeline()])
  }

  const saveConfig = async () => {
    saving.value = true
    try {
      await updateSub2APIConfig({ config: { ...configForm } })
      // 回读完整配置 + 分组表状态（public_enabled / deleted），避免仅回写 config 导致 groups 过期
      await loadConfig()
      return true
    } catch (error) {
      showRequestError(error, t('sub2api.store.saveConfigFailed'))
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
      showRequestError(error, t('sub2api.store.testConnectionFailed'))
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
      showRequestError(error, t('sub2api.store.syncFailed'))
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
      showRequestError(error, t('sub2api.store.loadAnnouncementsFailed'))
    }
  }

  const loadTimeline = async () => {
    try {
      const { data } = await listSub2APITimeline()
      timeline.value = data?.timeline || []
    } catch (error) {
      showRequestError(error, t('sub2api.store.loadTimelineFailed'))
    }
  }

  const saveAnnouncement = async (payload: CreateSub2APIAnnouncementRequest & { id?: string }) => {
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
      showRequestError(error, t('sub2api.store.saveAnnouncementFailed'))
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
      showRequestError(error, t('sub2api.store.deleteAnnouncementFailed'))
      return false
    }
  }

  const saveTimelineItem = async (payload: CreateSub2APITimelineItemRequest & { id?: string }) => {
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
      showRequestError(error, t('sub2api.store.saveTimelineFailed'))
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
      showRequestError(error, t('sub2api.store.deleteTimelineFailed'))
      return false
    }
  }

  // ---------- 格式化 ----------
  const formatNumber = (value: unknown) => new Intl.NumberFormat(getCurrentLocale()).format(toNumber(value))
  const formatPercent = (value: unknown) => `${toNumber(value).toFixed(1)}%`
  const formatLatency = (value: unknown) => {
    const ms = toNumber(value)
    if (ms >= 1000) return `${(ms / 1000).toFixed(2)} s`
    return `${Math.round(ms)} ms`
  }
  const formatToken = (value: unknown) => {
    const count = toNumber(value)
    const abs = Math.abs(count)
    if (abs >= 1e9) return `${(count / 1e9).toFixed(2)}B`
    if (abs >= 1e6) return `${(count / 1e6).toFixed(2)}M`
    if (abs >= 1e3) return `${(count / 1e3).toFixed(1)}K`
    return formatNumber(count)
  }
  // 请求量紧凑显示（与 formatToken 同口径：K/M/B），用于图表轴与 tooltip
  const formatCompactCount = (value: unknown) => formatToken(value)

  // ECharts axis tooltip 参数（不依赖 echarts 类型包）
  type ChartTooltipParam = {
    axisValueLabel?: string
    axisValue?: string | number
    seriesName?: string
    value?: unknown
    marker?: string
  }
  const formatAxisTooltip = (
    params: ChartTooltipParam | ChartTooltipParam[],
    formatValue: (name: string, raw: unknown) => string,
  ) => {
    const list = Array.isArray(params) ? params : [params]
    if (!list.length) return ''
    const title = String(list[0]?.axisValueLabel ?? list[0]?.axisValue ?? '')
    const lines = list.map((p) => {
      const name = p.seriesName || ''
      const raw = Array.isArray(p.value) ? p.value[p.value.length - 1] : p.value
      return `${p.marker || ''}${name}: ${formatValue(name, raw)}`
    })
    return [title, ...lines].join('<br/>')
  }
  const formatThroughput = (value: unknown) => {
    const tps = toNumber(value)
    if (tps >= 10000) {
      return new Intl.NumberFormat(getCurrentLocale(), {
        notation: 'compact',
        maximumFractionDigits: 1,
      }).format(tps)
    }
    if (tps >= 100) return `${Math.round(tps)}`
    return tps.toFixed(1)
  }
  // 费用展示（USD）：金额越小保留越多位，避免小额请求显示为 $0
  const formatCost = (value: unknown) => {
    const cost = toNumber(value)
    if (cost <= 0) return '$0'
    if (cost < 0.01) return `$${cost.toFixed(6)}`
    if (cost < 1) return `$${cost.toFixed(4)}`
    return `$${cost.toFixed(2)}`
  }
  const formatDateTime = (value: DateLike) => {
    if (!value) return '-'
    const parsed = dayjs(value)
    return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm:ss') : '-'
  }

  const statusText = (value?: string) => {
    switch (value) {
      case 'syncing':
        return t('sub2api.store.syncing')
      case 'success':
        return t('sub2api.store.normal')
      case 'error':
        return t('sub2api.store.error')
      default:
        return t('sub2api.store.pendingSync')
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

  const trendOption = (trend: Sub2APITrendPoint[] = []) => {
    if (!trend.length) return emptyChartOption()
    const isDark = themeStore.isDarkTheme
    const axisColor = isDark ? '#94a3b8' : '#64748b'
    const compact = isCompactChart.value
    return {
      color: ['#3b82f6', '#8b5cf6', '#10b981'],
      tooltip: {
        trigger: 'axis',
        confine: true,
        formatter: (params: ChartTooltipParam | ChartTooltipParam[]) =>
          formatAxisTooltip(params, (name, raw) => {
            if (name === t('sub2api.common.successRate')) return formatPercent(raw)
            if (name === 'Token') return formatToken(raw)
            return formatCompactCount(raw)
          }),
      },
      legend: {
        top: 0,
        left: 0,
        right: 0,
        type: compact ? 'scroll' : 'plain',
        textStyle: { color: axisColor },
      },
      grid: {
        left: compact ? 36 : 44,
        right: compact ? 32 : 48,
        top: compact ? 50 : 36,
        bottom: 28,
      },
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
        // 左侧可见轴：Token 数（用 K/M/B 紧凑显示）
        {
          type: 'value',
          splitLine: { lineStyle: { color: isDark ? '#1f2937' : '#eef2f7' } },
          axisLabel: {
            color: axisColor,
            fontSize: 11,
            formatter: (val: number) => formatToken(val),
          },
        },
        {
          type: 'value',
          max: 100,
          axisLabel: { formatter: '{value}%', color: axisColor, fontSize: 11 },
          splitLine: { show: false },
        },
        // 请求量独立缩放轴：不在侧边显示，仅用于让折线按自身量级铺开
        { type: 'value', show: false },
      ],
      series: [
        {
          name: t('sub2api.common.requestCount'),
          type: 'line',
          yAxisIndex: 2,
          smooth: true,
          symbol: 'none',
          data: trend.map((item) => item.requestCount),
        },
        {
          name: 'Token',
          type: 'line',
          yAxisIndex: 0,
          smooth: true,
          symbol: 'none',
          data: trend.map((item) => item.tokenCount),
        },
        {
          name: t('sub2api.common.successRate'),
          type: 'line',
          yAxisIndex: 1,
          smooth: true,
          symbol: 'none',
          data: trend.map((item) => Number(toNumber(item.successRate).toFixed(2))),
        },
      ],
    }
  }

  // Top 图表按 token 用量呈现（轴/tooltip 用 K/M/B 紧凑格式；数据已由后端按 token 降序）
  const topOption = (items: Sub2APITopItem[] = [], title: string) => {
    if (!items.length) return emptyChartOption()
    const isDark = themeStore.isDarkTheme
    const axisColor = isDark ? '#94a3b8' : '#64748b'
    const compact = isCompactChart.value
    const chartItems = [...items].reverse()
    return {
      color: ['#6366f1'],
      tooltip: {
        trigger: 'axis',
        confine: true,
        formatter: (params: ChartTooltipParam | ChartTooltipParam[]) =>
          formatAxisTooltip(params, (_name, raw) => formatToken(raw)),
      },
      grid: { left: compact ? 72 : 96, right: compact ? 8 : 18, top: 16, bottom: 20 },
      xAxis: {
        type: 'value',
        axisLabel: {
          color: axisColor,
          fontSize: 11,
          formatter: (val: number) => formatToken(val),
        },
        splitLine: { lineStyle: { color: isDark ? '#1f2937' : '#eef2f7' } },
      },
      yAxis: {
        type: 'category',
        data: chartItems.map((item) => item.name || t('sub2api.common.unlabeled')),
        axisLabel: { color: axisColor, fontSize: 11 },
      },
      series: [
        {
          name: title,
          type: 'bar',
          barMaxWidth: 16,
          itemStyle: { borderRadius: [0, 4, 4, 0] },
          data: chartItems.map((item) => item.tokenCount),
        },
      ],
    }
  }

  // 概览 Top 图表只展示前 N 项（筛选下拉用全量枚举 adminModels/adminGroups）
  const TOP_CHART_LIMIT = 10
  const adminTrendOption = computed(() => trendOption(adminTrend.value))
  const adminModelOption = computed(() =>
    topOption(adminModels.value.slice(0, TOP_CHART_LIMIT), t('sub2api.common.modelTokenTop')),
  )
  const adminGroupOption = computed(() =>
    topOption(adminGroups.value.slice(0, TOP_CHART_LIMIT), t('sub2api.common.groupTokenTop')),
  )
  const statsTrendOption = computed(() => trendOption(stats.value?.trend))

  // 当日“成功率 + 生成速度”随时间移动曲线（time 轴，短时段也能铺满）；数据来自公开概览接口
  const todaySeriesOption = computed(() => {
    const series = publicOverview.value?.todaySeries || []
    if (!series.length) return emptyChartOption(t('sub2api.store.todayNoData'))
    const isDark = themeStore.isDarkTheme
    const axisColor = isDark ? '#94a3b8' : '#64748b'
    const compact = isCompactChart.value
    const toMs = (t: unknown) => (t ? new Date(t as string | Date).getTime() : 0)
    return {
      color: ['#10b981', '#3b82f6', '#f59e0b'],
      tooltip: {
        trigger: 'axis',
        confine: true,
        formatter: (params: ChartTooltipParam | ChartTooltipParam[]) =>
          formatAxisTooltip(params, (name, raw) => {
            if (name === t('sub2api.common.successRate')) return formatPercent(raw)
            if (name === t('sub2api.common.generationSpeed')) return formatThroughput(raw)
            return formatCompactCount(raw)
          }),
      },
      legend: {
        top: 0,
        left: 0,
        right: 0,
        type: compact ? 'scroll' : 'plain',
        textStyle: { color: axisColor },
      },
      grid: {
        left: compact ? 36 : 48,
        right: compact ? 36 : 78,
        top: compact ? 50 : 58,
        bottom: 28,
      },
      xAxis: {
        type: 'time',
        axisLabel: { color: axisColor, fontSize: 11, hideOverlap: true },
      },
      yAxis: [
        {
          type: 'value',
          name: compact ? '' : 'token/s',
          position: 'left',
          nameTextStyle: { color: axisColor },
          axisLabel: { color: axisColor, fontSize: 11 },
          splitLine: { lineStyle: { color: isDark ? '#1f2937' : '#eef2f7' } },
        },
        {
          type: 'value',
          name: compact ? '' : t('sub2api.common.successRate'),
          position: 'right',
          max: 100,
          axisLabel: { formatter: '{value}%', color: axisColor, fontSize: 11 },
          splitLine: { show: false },
        },
        {
          type: 'value',
          name: t('sub2api.common.requestCount'),
          position: 'right',
          offset: 46,
          show: !compact,
          axisLabel: {
            color: axisColor,
            fontSize: 11,
            formatter: (val: number) => formatCompactCount(val),
          },
          splitLine: { show: false },
        },
      ],
      series: [
        {
          name: t('sub2api.common.successRate'),
          type: 'line',
          yAxisIndex: 1,
          smooth: true,
          symbol: 'none',
          areaStyle: { opacity: 0.06 },
          data: series.map((p) => [toMs(p.time), Number(toNumber(p.successRate).toFixed(2))]),
        },
        {
          name: t('sub2api.common.generationSpeed'),
          type: 'line',
          yAxisIndex: 0,
          smooth: true,
          symbol: 'none',
          data: series.map((p) => [toMs(p.time), Number(toNumber(p.avgTps).toFixed(1))]),
        },
        {
          name: t('sub2api.common.requestCount'),
          type: 'line',
          yAxisIndex: 2,
          smooth: true,
          symbol: 'none',
          data: series.map((p) => [toMs(p.time), toNumber(p.requestCount)]),
        },
      ],
    }
  })

  // 汇总指标卡：公开 stats 与管理端 totals 结构上共享这组标量字段
  type StatsCardSource = {
    requestCount?: number
    successCount?: number
    successRate?: number
    tokenCount?: number
    averageLatencyMs?: number
    averageTps?: number
    rangeLabel?: string
  }
  const buildStatsCards = (s?: StatsCardSource): Sub2APIMetricCard[] => [
    {
      label: t('sub2api.common.requestCount'),
      value: formatNumber(s?.requestCount),
      detail: s?.rangeLabel || '',
      icon: 'HOutline:ArrowPathRoundedSquareIcon',
      tone: 'blue',
    },
    {
      label: 'Token',
      value: formatToken(s?.tokenCount),
      detail: t('sub2api.store.intervalTotal'),
      icon: 'HOutline:CircleStackIcon',
      tone: 'blue',
    },
    {
      label: t('sub2api.common.successRate'),
      value: formatPercent(s?.successRate),
      detail: t('sub2api.store.successCount', { count: formatNumber(s?.successCount) }),
      icon: 'HOutline:CheckCircleIcon',
      tone: 'green',
    },
    {
      label: t('sub2api.store.averageLatency'),
      value: formatLatency(s?.averageLatencyMs),
      detail: t('sub2api.store.successfulRequestAverage'),
      icon: 'HOutline:ClockIcon',
      tone: 'amber',
    },
    {
      label: t('sub2api.store.tokenSpeed'),
      value: `${formatThroughput(s?.averageTps)} token/s`,
      detail: t('sub2api.store.requestAverageNoCache'),
      icon: 'HOutline:BoltIcon',
      tone: 'red',
    },
  ]

  const statsMetricCards = computed(() => buildStatsCards(stats.value))
  const adminStatsMetricCards = computed(() => buildStatsCards(adminTotals.value))

  return {
    config,
    configForm,
    groups,
    snapshot,
    home,
    stats,
    adminTotals,
    adminTrend,
    adminModels,
    adminGroups,
    adminRecent,
    adminRecentTotal,
    recentPage,
    recentPageSize,
    announcements,
    timeline,
    publicOverview,
    configLoading,
    snapshotLoading,
    publicLoading,
    publicOverviewLoading,
    statsLoading,
    adminTotalsLoading,
    adminTrendLoading,
    adminTopLoading,
    recentLoading,
    saving,
    testing,
    syncing,
    listLoading,
    statsMetricCards,
    adminStatsMetricCards,
    adminTrendOption,
    adminModelOption,
    adminGroupOption,
    statsTrendOption,
    todaySeriesOption,
    loadConfig,
    loadSnapshot,
    loadPublicHome,
    loadPublicOverview,
    loadStats,
    loadAdminTotals,
    loadAdminTrend,
    loadAdminTop,
    loadAdminRecent,
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
    formatCost,
    formatDateTime,
    statusText,
    statusType,
  }
})
