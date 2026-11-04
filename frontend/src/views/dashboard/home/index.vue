<!-- 工作台（DashboardPC / DashboardMobile）
     PC：主色问候卡（日期 / 问候 / 状态 / 主机 chips / 快捷操作）+「需要你看一眼」→ 指标卡 → 实时监控 + 运行中 → 快捷入口 + Docker + 后台任务。
     手机：问候页头（头像 / 搜索 / 主题 / 通知）+ 节点胶囊 → 主色状态卡 → 待办 → 2×2 指标 → 实时监控 → 运行中 → 快捷入口。
     监控数据与轮询沿用 useDashboardHomeStore；Docker / 任务 / 更新见 useDashboardExtras（各自按权限加载）。 -->
<template>
  <div class="dash">
    <!-- 手机页头 -->
    <template v-if="isMobile">
      <header class="dash-mh">
        <span class="dash-mh__av">
          <AppAvatar :src="userStore.resolvedUserAvatar" :size="44" :name="displayName" />
        </span>
        <span class="dash-mh__text">
          <span class="dash-mh__greet">{{ greeting }}</span>
          <span class="dash-mh__name">{{ displayName }}</span>
        </span>
        <AppIconButton icon="HOutline:MagnifyingGlassIcon" :label="t('shell.search')" :box="44" @click="more.open(true)" />
        <AppIconButton
          icon="HOutline:SwatchIcon"
          :label="t('shell.themeAndAppearance')"
          :box="44"
          @click="router.push({ path: '/profile', query: { tab: 'appearance' } })"
        />
        <NotificationMenu />
      </header>
      <div class="dash-mnode">
        <NodeSwitcher variant="pill" />
      </div>
    </template>

    <div class="dash-row">
      <!-- 问候卡：主色实底 + 黄色圆与第二强调色圆环装饰 -->
      <section class="dash-hero">
        <span class="dash-hero__blob" aria-hidden="true" />
        <span class="dash-hero__ring" aria-hidden="true" />
        <svg class="dash-hero__star" viewBox="0 0 24 24" aria-hidden="true">
          <path d="M12 1.5l2.6 7.9 7.9 2.6-7.9 2.6L12 22.5l-2.6-7.9L1.5 12l7.9-2.6z" />
        </svg>
        <div class="dash-hero__body">
          <div class="dash-hero__date">{{ heroDate }}</div>
          <h1 class="dash-hero__title">{{ isMobile ? headline : `${greeting}，${displayName}` }}</h1>
          <p class="dash-hero__sub">{{ heroSub }}</p>
          <div v-if="!isMobile" class="dash-hero__chips">
            <span v-if="nodeStore.isRemote" class="dash-chip">
              {{ t('dashboard.home.remoteNode', { name: nodeStore.contextInfo?.name || '' }) }}
            </span>
            <span class="dash-chip">{{ hostname }}</span>
            <span v-if="overview?.uptimeSeconds != null" class="dash-chip">
              {{ t('dashboard.home.upFor', { time: formatUptime(overview?.uptimeSeconds) }) }}
            </span>
            <span v-if="osText !== '—'" class="dash-chip">{{ osText }}</span>
          </div>
          <div v-if="quickActions.length" class="dash-hero__actions">
            <button
              v-for="(qa, i) in quickActions"
              :key="qa.path"
              type="button"
              class="dash-hero__btn"
              :class="{ 'is-primary': i === 0 }"
              @click="router.push(qa.to)"
            >
              <component :is="menuStore.iconComponents[qa.icon]" />
              <span>{{ t(isMobile ? qa.short : qa.label) }}</span>
            </button>
          </div>
        </div>
      </section>

      <AttentionCard class="dash-row__side" :items="attention" :compact="isMobile" />
    </div>

    <!-- 远程节点不可用或未授予机器状态权限：不轮询，给出说明 -->
    <AppPanel v-if="!canViewStatus" :title="t('dashboard.home.systemStatus')" title-icon="HOutline:ChartBarIcon">
      <EmptyState
        v-if="!nodeStore.isOnline"
        icon="HOutline:ExclamationTriangleIcon"
        :title="t('dashboard.home.nodeOffline')"
        :description="nodeStore.contextInfo?.status?.message || t('dashboard.home.nodeOfflineDesc')"
      />
      <EmptyState
        v-else
        icon="HOutline:NoSymbolIcon"
        :title="t('dashboard.home.statusForbidden')"
        :description="t('dashboard.home.statusForbiddenDesc')"
      />
    </AppPanel>

    <!-- 指标卡：贴纸图标 + 大号数字 + 圆环 / 迷你曲线 -->
    <div v-if="canViewStatus" class="dash-stats">
      <div v-for="st in statCards" :key="st.key" class="dash-stat">
        <div class="dash-stat__head">
          <span class="dash-stat__label">
            <span class="dash-stat__icon m-sticker" :class="`m-sticker--${st.tone}`" aria-hidden="true">
              <component :is="menuStore.iconComponents[st.icon]" />
            </span>
            {{ isMobile ? st.shortLabel : st.label }}
          </span>
          <svg
            v-if="st.percent != null"
            class="dash-ring"
            viewBox="0 0 64 64"
            role="img"
            :aria-label="`${st.label} ${st.value}`"
          >
            <circle cx="32" cy="32" r="26" class="dash-ring__track" />
            <circle
              cx="32"
              cy="32"
              r="26"
              class="dash-ring__bar"
              :style="{ stroke: st.color }"
              :stroke-dasharray="`${ringLen(st.percent)} 163.4`"
              transform="rotate(-90 32 32)"
            />
          </svg>
          <svg v-else class="dash-spark" viewBox="0 0 120 40" aria-hidden="true">
            <path :d="netSpark" />
          </svg>
        </div>
        <div class="dash-stat__value m-num">{{ isMobile ? st.shortValue : st.value }}</div>
        <div class="dash-stat__sub">{{ isMobile ? st.shortSub : st.sub }}</div>
      </div>
    </div>

    <!-- 实时监控 + 运行中 -->
    <div v-if="canViewStatus || canViewInstances" class="dash-row">
      <SystemRealtimeCharts
        v-if="canViewStatus"
        class="dash-row__main"
        :cpu-history="cpuHistory"
        :memory-history="memoryHistory"
        :network-history="networkHistory"
        :disk-history="diskHistory"
        :refresh-interval="refreshInterval"
        :selected-interface="selectedInterface"
        :selected-disk="selectedDisk"
        :overview="chartOverview"
        @interval-change="setRefreshInterval"
        @interface-change="setSelectedInterface"
        @disk-change="setSelectedDisk"
      />
      <RunningInstanceSection v-if="canViewInstances" class="dash-row__side" />
    </div>

    <!-- 快捷入口 + Docker + 后台任务（手机只保留快捷入口） -->
    <div class="dash-row dash-row--3">
      <ShortcutSection class="dash-row__quick" />
      <template v-if="!isMobile">
        <DockerCard
          v-if="extras.canDocker.value && extras.dockerLoaded.value"
          class="dash-row__card"
          :status="extras.docker.value"
          :counts="extras.dockerCounts.value"
        />
        <TasksCard v-if="extras.canTasks.value" class="dash-row__card" :tasks="extras.tasks.value" />
      </template>
    </div>
  </div>
</template>

<script lang="ts">
// 自动检查更新每个浏览器标签页只做一次：切换节点会整页重载，不应反复弹出更新提示。
const AUTO_UPDATE_CHECKED_KEY = 'momoko-update-auto-checked'
const hasAutoCheckedUpdate = () => {
  try {
    return sessionStorage.getItem(AUTO_UPDATE_CHECKED_KEY) === '1'
  } catch {
    return false
  }
}
const markAutoCheckedUpdate = () => {
  try {
    sessionStorage.setItem(AUTO_UPDATE_CHECKED_KEY, '1')
  } catch {
    // 无法持久化时退化为每次加载检查一次
  }
}
</script>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import RunningInstanceSection from '@/views/dashboard/home/runningInstanceSection.vue'
import ShortcutSection from '@/views/dashboard/home/shortcutSection.vue'
import SystemRealtimeCharts from '@/views/dashboard/home/systemRealtimeCharts.vue'
import AttentionCard from '@/views/dashboard/home/attentionCard.vue'
import DockerCard from '@/views/dashboard/home/dockerCard.vue'
import TasksCard from '@/views/dashboard/home/tasksCard.vue'
import NodeSwitcher from '@/layouts/app/NodeSwitcher.vue'
import NotificationMenu from '@/layouts/app/NotificationMenu.vue'
import { useMorePanel } from '@/layouts/app/tabbar'
import { pageTabTo } from '@/config/pageGroups'
import { useDashboardExtras, type AttentionItem } from '@/views/dashboard/home/useDashboardExtras'
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import { TaskStatus } from '@/types/v1/task'
import { checkForUpdate } from '@/utils/updateCheck'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import { useProfileFormat } from '@/views/profile/components/useProfileFormat'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'HomeView' })

const BUILTIN_SUPER_ADMIN_ROLE_ID = 'role_1'

const dashboardHomeStore = useDashboardHomeStore()
const menuStore = useMenuStore()
const userStore = useUserStore()
const nodeStore = useNodeStore()
const router = useRouter()
const accessible = useAccessibleRoutes()
const more = useMorePanel()
const extras = useDashboardExtras()
const { t, locale } = useI18n()
const { relTime } = useProfileFormat()

const isMobile = computed(() => menuStore.isMobile)

// 远程节点上下文按节点授权分块展示；本机保持原有行为。
const canViewStatus = computed(() => !nodeStore.isRemote || menuStore.hasButtonPermission('system:status'))
const canViewInstances = computed(
  () => !nodeStore.isRemote || menuStore.hasButtonPermission('instance:manage'),
)

const {
  cpuHistory,
  memoryHistory,
  networkHistory,
  diskHistory,
  refreshInterval,
  selectedInterface,
  selectedDisk,
  overview,
  status,
  currentDate,
  runningInstanceTotal,
} = storeToRefs(dashboardHomeStore)
const {
  fetchOverview,
  startAutoRefresh,
  stopAutoRefresh,
  setRefreshInterval,
  setSelectedInterface,
  setSelectedDisk,
  startCurrentDateTicker,
  stopCurrentDateTicker,
} = dashboardHomeStore

const chartOverview = computed(() => {
  if (!overview.value) return null
  return {
    networkInterfaces: overview.value.networkInterfaces || [],
    diskPartitions: overview.value.diskPartitions || [],
  }
})

const displayName = computed(() => userStore.userInfo?.name || userStore.userInfo?.username || '')

// currentDate 每秒更新，问候语与日期随之刷新
const now = computed(() => (currentDate.value ? new Date(currentDate.value.replace(' ', 'T')) : new Date()))
const greeting = computed(() => {
  const h = now.value.getHours()
  if (h < 6) return t('dashboard.home.greeting.lateNight')
  if (h < 9) return t('dashboard.home.greeting.morning')
  if (h < 12) return t('dashboard.home.greeting.forenoon')
  if (h < 14) return t('dashboard.home.greeting.noon')
  if (h < 18) return t('dashboard.home.greeting.afternoon')
  return t('dashboard.home.greeting.evening')
})
/** 「周二 · 11 月 3 日」 */
const heroDate = computed(() => {
  const d = now.value
  const weekday = new Intl.DateTimeFormat(locale.value, { weekday: 'short' }).format(d)
  const day = t('dashboard.home.monthDay', { month: d.getMonth() + 1, day: d.getDate() })
  return `${weekday} · ${day}`
})

const runningCount = computed(() => runningInstanceTotal.value)
const hostname = computed(
  () => overview.value?.version?.hostname || nodeStore.contextInfo?.status?.remote?.hostname || '—',
)

// ---- 需要你看一眼 ----
const attention = computed<AttentionItem[]>(() => {
  const items: AttentionItem[] = []
  if (nodeStore.isRemote && !nodeStore.isOnline) {
    items.push({
      key: 'node-offline',
      icon: 'HOutline:SignalSlashIcon',
      tone: 'bad',
      title: t('dashboard.home.nodeOffline'),
      sub: nodeStore.contextInfo?.status?.message || t('dashboard.home.nodeOfflineDesc'),
    })
  }
  for (const c of extras.exited.value.slice(0, 3)) {
    items.push({
      key: `ctr-${c.id}`,
      icon: 'HOutline:ExclamationTriangleIcon',
      tone: 'bad',
      title: t('dashboard.home.containerExited', { name: c.name }),
      sub: c.finishedAt
        ? `${t('dashboard.home.exitCode', { code: c.code })} · ${relTime(c.finishedAt)}`
        : `${t('dashboard.home.exitCode', { code: c.code })} · ${c.image}`,
      loading: extras.restarting.value.has(c.id),
      action: extras.canManageContainer.value
        ? { label: t('dashboard.home.restart'), run: () => void extras.restartContainer(c.id) }
        : { label: t('dashboard.home.view'), run: () => void router.push('/docker') },
    })
  }
  const up = extras.update.value
  if (up?.hasUpdate) {
    items.push({
      key: 'update',
      icon: 'HOutline:GiftIcon',
      tone: 'a',
      title: t('dashboard.home.newVersion', { version: up.latestVersion }),
      sub: t('dashboard.home.newVersionDesc', { current: up.currentVersion }),
      action: { label: t('dashboard.home.view'), run: () => void checkForUpdate() },
    })
  }
  const busy = extras.runningTaskCount.value
  if (busy > 0) {
    const titles = extras.tasks.value
      .filter((task) => task.status === TaskStatus.TASK_STATUS_RUNNING || task.status === TaskStatus.TASK_STATUS_PENDING)
      .map((task) => task.title || task.type)
    items.push({
      key: 'tasks',
      icon: 'HOutline:ClockIcon',
      tone: 'a2',
      title: t('dashboard.home.tasksRunning', { n: busy }),
      sub: titles.join(' · ') || t('dashboard.home.tasksRunningDesc'),
      action: { label: t('dashboard.home.view'), run: () => void router.push({ path: '/system/logs', query: { tab: 'task' } }) },
    })
  }
  return items
})
const issueCount = computed(() => attention.value.filter((i) => i.tone === 'bad').length)

const headline = computed(() =>
  issueCount.value > 0
    ? t('dashboard.home.heroIssues', { n: issueCount.value })
    : t('dashboard.home.heroAllGood'),
)
const runningText = computed(() =>
  runningCount.value > 0
    ? t('dashboard.home.heroRunning', { count: runningCount.value })
    : t('dashboard.home.heroNoRunning'),
)
const heroSub = computed(() => {
  const parts: string[] = []
  if (isMobile.value) {
    if (canViewInstances.value) parts.push(runningText.value)
    if (overview.value?.uptimeSeconds != null) {
      parts.push(t('dashboard.home.upFor', { time: formatUptime(overview.value.uptimeSeconds) }))
    }
    return parts.join(' · ') || t('dashboard.home.heroAllGood')
  }
  parts.push(headline.value)
  if (canViewInstances.value) parts.push(runningText.value)
  if (attention.value.length) parts.push(t('dashboard.home.heroAttention', { n: attention.value.length }))
  return parts.join(' · ')
})

const fmtPercent = (v?: number) => `${(v ?? 0).toFixed(1)}%`

// 问候卡快捷操作：只展示能进入的页面（path 判断权限，分组标签用虚拟路径；to 是跳转地址）
const QUICK_ACTIONS = [
  { path: '/instance/list', to: '/instance/list', label: 'dashboard.home.quickNewApp', short: 'dashboard.home.quickNewApp', icon: 'HOutline:PlusIcon' },
  { path: '/openssh/management', to: '/openssh/management', label: 'dashboard.home.quickTerminal', short: 'dashboard.home.quickTerminalShort', icon: 'HOutline:CommandLineIcon' },
  { path: '/file/index', to: pageTabTo('file', 'index'), label: 'dashboard.home.quickFiles', short: 'dashboard.home.quickFilesShort', icon: 'HOutline:ArrowUpTrayIcon' },
]
const quickActions = computed(() =>
  QUICK_ACTIONS.filter((a) => accessible.value.has(a.path) && nodeStore.canAccessRoute(a.path)),
)

// 圆环周长 2πr = 163.4（r=26）
const ringLen = (percent: number) => ((Math.min(100, Math.max(0, percent)) / 100) * 163.4).toFixed(1)

const formatBytes = (bytes?: number | string, digits = 1) => {
  const num = Number(bytes)
  if (!num || num <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const base = 1024
  const i = Math.min(Math.floor(Math.log(num) / Math.log(base)), units.length - 1)
  return (num / base ** i).toFixed(i === 0 ? 0 : digits) + ' ' + units[i]
}
/** 手机指标卡用的紧凑写法：20.6K / 1.2M */
const formatCompact = (bytes?: number | string) => {
  const num = Number(bytes)
  if (!num || num <= 0) return '0'
  const units = ['', 'K', 'M', 'G', 'T']
  const i = Math.min(Math.floor(Math.log(num) / Math.log(1024)), units.length - 1)
  return (num / 1024 ** i).toFixed(i === 0 ? 0 : 1) + units[i]
}

const statCards = computed(() => {
  const st = status.value
  const net = st?.network?.total
  const downs = networkHistory.value.map((p) => p.download)
  const peakDown = Math.max(...downs, 1)
  const cpuModel = overview.value?.cpu?.modelName || '—'
  const mem = st?.memory?.physicalMemory
  const disk = st?.disk?.total
  return [
    {
      key: 'cpu',
      label: t('dashboard.home.statCpu'),
      shortLabel: t('dashboard.home.statCpu'),
      icon: 'HOutline:CpuChipIcon',
      tone: 'p',
      color: 'var(--m-p)',
      percent: st?.cpu?.totalPercent ?? 0,
      value: fmtPercent(st?.cpu?.totalPercent),
      shortValue: fmtPercent(st?.cpu?.totalPercent),
      sub: cpuModel,
      shortSub: cpuModel.replace(/^(Intel\(R\)|AMD)\s*/i, '').replace(/\s*CPU\s*@.*$/i, ''),
    },
    {
      key: 'memory',
      label: t('dashboard.home.statMemory'),
      shortLabel: t('dashboard.home.statMemory'),
      icon: 'HOutline:CircleStackIcon',
      tone: 'a2',
      color: 'var(--m-a2)',
      percent: mem?.usedPercent ?? 0,
      value: fmtPercent(mem?.usedPercent),
      shortValue: fmtPercent(mem?.usedPercent),
      sub: `${formatBytes(mem?.usedBytes)} / ${formatBytes(mem?.totalBytes)}`,
      shortSub: `${formatBytes(mem?.usedBytes)} / ${formatBytes(mem?.totalBytes)}`,
    },
    {
      key: 'disk',
      label: t('dashboard.home.statDisk'),
      shortLabel: t('dashboard.home.statDisk'),
      icon: 'HOutline:ServerIcon',
      tone: 'a',
      color: 'var(--m-a)',
      percent: disk?.usedPercent ?? 0,
      value: fmtPercent(disk?.usedPercent),
      shortValue: fmtPercent(disk?.usedPercent),
      sub: `${formatBytes(disk?.usedBytes)} / ${formatBytes(disk?.totalBytes)}`,
      shortSub: `${formatBytes(disk?.usedBytes)} / ${formatBytes(disk?.totalBytes)}`,
    },
    {
      key: 'network',
      label: t('dashboard.home.statNetwork'),
      shortLabel: `${t('dashboard.home.statNetwork')} ↓`,
      icon: 'HOutline:ArrowsUpDownIcon',
      tone: 'ok',
      color: 'var(--m-ok)',
      // PC 用迷你曲线；手机用圆环（当前下行 / 最近峰值）
      percent: isMobile.value ? ((net?.downloadRateBytesPerSecond ?? 0) / peakDown) * 100 : null,
      value: `↓ ${formatBytes(net?.downloadRateBytesPerSecond)}/s`,
      shortValue: formatCompact(net?.downloadRateBytesPerSecond),
      sub: `↑ ${formatBytes(net?.uploadRateBytesPerSecond)}/s · ${t('dashboard.home.totalReceived')} ${formatBytes(net?.bytesRecv)}`,
      shortSub: `↑ ${formatBytes(net?.uploadRateBytesPerSecond)}/s`,
    },
  ]
})

// 网络下行迷你曲线（最近 30 个点）
const netSpark = computed(() => {
  const pts = networkHistory.value.slice(-30).map((p) => p.download)
  if (pts.length < 2) return 'M0 36 L120 36'
  const max = Math.max(...pts, 1)
  return pts
    .map((v, i) => `${i ? 'L' : 'M'}${((i / (pts.length - 1)) * 120).toFixed(1)} ${(37 - (v / max) * 32).toFixed(1)}`)
    .join(' ')
})

const formatUptime = (seconds?: number) => {
  if (seconds == null) return '—'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return t('dashboard.home.dayHour', { days: d, hours: h })
  if (h > 0) return t('dashboard.home.hourMinute', { hours: h, minutes: m })
  return t('dashboard.home.minute', { minutes: m })
}

const osText = computed(() => {
  const v = overview.value?.version
  if (!v) return '—'
  const os = `${v.os || ''} ${v.platformVersion || ''}`.trim()
  return [os, v.kernelArch].filter(Boolean).join(' · ') || '—'
})

/** 内置超级管理员每个标签页自动弹一次更新提示；其余有更新权限的人只在「需要你看一眼」里看到 */
const autoCheckUpdate = async () => {
  if (!userStore.userInfo) await userStore.getUserInfo()
  if (!menuStore.hasLoadedPermissions) await menuStore.getUserPermissions()
  if (!menuStore.hasButtonPermission('system:update')) return

  const isBuiltinSuperAdmin = !!userStore.userInfo?.roles?.some((r) => r.roleId === BUILTIN_SUPER_ADMIN_ROLE_ID)
  if (isBuiltinSuperAdmin && !hasAutoCheckedUpdate()) {
    markAutoCheckedUpdate()
    const data = await checkForUpdate({ silentNoUpdate: true }).catch(() => undefined)
    await extras.loadUpdate(data ?? null)
    return
  }
  await extras.loadUpdate()
}

const startAll = () => {
  if (canViewStatus.value) {
    fetchOverview()
    startAutoRefresh()
  }
  startCurrentDateTicker()
  void extras.loadAll()
}

onMounted(() => {
  startAll()
  void autoCheckUpdate()
})

// 页面被缓存时暂停实时刷新，回来时立即刷新一次再继续（首次挂载也会触发 onActivated，用标记跳过）
let activatedOnce = false
onActivated(() => {
  if (!activatedOnce) {
    activatedOnce = true
    return
  }
  startAll()
  void dashboardHomeStore.getRunningInstances()
})
onDeactivated(() => {
  stopAutoRefresh()
  stopCurrentDateTicker()
})

onBeforeUnmount(() => {
  stopAutoRefresh()
  stopCurrentDateTicker()
})
</script>

<style scoped lang="scss">
.dash {
  display: flex;
  flex-direction: column;
  gap: 24px;
}
.dash-row {
  display: flex;
  flex-wrap: wrap;
  gap: 20px;
  align-items: stretch;
}
.dash-row > * {
  min-width: 0;
}
.dash-row__main,
.dash-hero {
  flex: 2 1 560px;
}
.dash-row__side {
  flex: 1 1 320px;
}
.dash-row--3 {
  align-items: flex-start;
}
.dash-row__quick {
  flex: 2 1 460px;
}
.dash-row__card {
  flex: 1 1 280px;
}

/* ---- 手机页头 ---- */
.dash-mh {
  display: flex;
  align-items: center;
  gap: 4px;
  margin: 2px -8px -8px 4px;
}
.dash-mh__av {
  display: inline-flex;
  margin-right: 6px;
  border-radius: 50%;
  :deep(.app-avatar) {
    background: var(--m-aSoft);
    color: var(--m-aOn);
  }
}
.dash-mh__text {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
}
.dash-mh__greet {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.dash-mh__name {
  font-family: var(--app-font-display);
  font-size: 1.125rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.dash-mnode {
  display: flex;
  min-width: 0;
  margin-bottom: -10px;
}

/* ---- 问候卡 ---- */
.dash-hero {
  position: relative;
  overflow: hidden;
  min-height: 248px;
  padding: 32px 36px;
  border-radius: var(--app-radius-xl);
  background: var(--m-p);
  color: var(--m-pOn);
}
.dash-hero__blob {
  position: absolute;
  right: -60px;
  bottom: -110px;
  width: 300px;
  height: 300px;
  border-radius: 50%;
  background: var(--m-a);
}
.dash-hero__ring {
  position: absolute;
  right: 190px;
  top: -40px;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  border: 18px solid var(--m-a2);
}
.dash-hero__star {
  position: absolute;
  right: 64px;
  top: 40px;
  width: 44px;
  height: 44px;
  fill: var(--m-pOn);
}
.dash-hero__body {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 10px;
  max-width: 560px;
}
.dash-hero__date {
  font-size: 0.875rem;
  font-weight: 600;
  opacity: 0.9;
}
.dash-hero__title {
  margin: 0;
  font-family: var(--app-font-display);
  font-size: 2.375rem;
  line-height: 1.15;
  font-weight: 800;
  letter-spacing: -0.01em;
  overflow-wrap: anywhere;
}
.dash-hero__sub {
  margin: 0;
  font-size: 0.9375rem;
  opacity: 0.92;
}
.dash-hero__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 2px;
}
.dash-chip {
  display: inline-flex;
  align-items: center;
  height: 30px;
  padding: 0 12px;
  border-radius: 999px;
  font-size: 0.8125rem;
  font-weight: 600;
  background: color-mix(in srgb, var(--m-pOn) 16%, transparent);
}
.dash-hero__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 12px;
}
.dash-hero__btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 18px;
  border: none;
  border-radius: 999px;
  background: color-mix(in srgb, var(--m-pOn) 18%, transparent);
  color: var(--m-pOn);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 600;
  cursor: pointer;
  transition: transform 0.15s, filter 0.15s;
}
.dash-hero__btn:hover {
  filter: brightness(1.06);
}
.dash-hero__btn:active {
  transform: scale(0.97);
}
.dash-hero__btn.is-primary {
  background: var(--m-pOn);
  color: var(--m-p);
}
.dash-hero__btn :deep(svg) {
  width: 20px;
  height: 20px;
}

/* ---- 指标卡 ---- */
.dash-stats {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
}
.dash-stat {
  container-type: inline-size;
  display: flex;
  flex-direction: column;
  gap: 14px;
  min-width: 0;
  padding: 22px;
  border-radius: var(--app-radius-lg);
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-light);
}
.dash-stat__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
}
.dash-stat__label {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  font-size: 0.875rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
.dash-stat__icon {
  width: 32px;
  height: 32px;
  border-radius: 10px;
}
.dash-stat__icon :deep(svg) {
  width: 16px;
  height: 16px;
}
.dash-ring {
  width: 58px;
  height: 58px;
  flex: none;
}
.dash-ring__track {
  fill: none;
  stroke: var(--el-fill-color-light);
  stroke-width: 8;
}
.dash-ring__bar {
  fill: none;
  stroke-width: 8;
  stroke-linecap: round;
  transition: stroke-dasharray 0.5s ease;
}
.dash-spark {
  width: 96px;
  height: 40px;
  flex: none;
}
.dash-spark path {
  fill: none;
  stroke: var(--m-ok);
  stroke-width: 2.5;
  stroke-linecap: round;
  stroke-linejoin: round;
}
.dash-stat__value {
  font-size: 2.375rem;
  line-height: 1;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
@container (width < 200px) {
  .dash-stat__value {
    font-size: 1.875rem;
  }
}
.dash-stat__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ---- 手机 ---- */
@media (width < 992px) {
  .dash {
    gap: 14px;
  }
  .dash-row {
    gap: 14px;
  }
  .dash-row > * {
    flex: 1 1 100%;
  }
  .dash-hero {
    min-height: 0;
    padding: 22px;
    border-radius: var(--app-radius-xl);
  }
  .dash-hero__blob {
    right: -50px;
    top: -60px;
    bottom: auto;
    width: 170px;
    height: 170px;
  }
  .dash-hero__ring {
    right: 90px;
    top: 40px;
    width: 48px;
    height: 48px;
    border-width: 9px;
  }
  .dash-hero__star {
    display: none;
  }
  .dash-hero__body {
    gap: 6px;
  }
  .dash-hero__date {
    font-size: 0.8125rem;
  }
  .dash-hero__title {
    font-size: 1.75rem;
  }
  .dash-hero__sub {
    font-size: 0.875rem;
  }
  .dash-hero__actions {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
    margin-top: 14px;
  }
  .dash-hero__btn {
    flex-direction: column;
    justify-content: center;
    gap: 6px;
    padding: 12px 4px;
    border-radius: 18px;
    font-size: 0.8125rem;
    font-weight: 700;
  }
  .dash-stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 12px;
  }
  .dash-stat {
    gap: 10px;
    padding: 16px;
    border-radius: var(--app-radius-lg);
  }
  .dash-stat__label {
    font-size: 0.8125rem;
  }
  .dash-stat__icon {
    display: none;
  }
  .dash-ring {
    width: 36px;
    height: 36px;
  }
  .dash-ring__track,
  .dash-ring__bar {
    stroke-width: 10;
  }
  .dash-stat__value {
    font-size: 1.625rem;
  }
  .dash-stat__sub {
    font-size: 0.75rem;
  }
}
</style>
