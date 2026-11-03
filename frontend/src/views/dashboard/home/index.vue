<!-- 工作台：主色问候卡（含快捷操作）+ 彩色指标卡 + 实时监控（单图切换）+ 运行中应用 / 快捷入口。
     数据与轮询逻辑全部沿用 useDashboardHomeStore。 -->
<template>
  <div class="dash">
    <!-- 问候卡：主色实底 + 黄色圆与第二强调色圆环装饰 -->
    <section class="dash-hero">
      <span class="dash-hero__blob" aria-hidden="true" />
      <span class="dash-hero__ring" aria-hidden="true" />
      <svg class="dash-hero__star" viewBox="0 0 24 24" aria-hidden="true">
        <path d="M12 1.5l2.6 7.9 7.9 2.6-7.9 2.6L12 22.5l-2.6-7.9L1.5 12l7.9-2.6z" />
      </svg>
      <div class="dash-hero__body">
        <div class="dash-hero__date m-num">{{ currentDate }}</div>
        <h2 class="dash-hero__title">{{ greeting }}，{{ displayName }}</h2>
        <p class="dash-hero__sub">
          <template v-if="canViewInstances">
            {{ runningCount > 0 ? t('dashboard.home.heroRunning', { count: runningCount }) : t('dashboard.home.heroNoRunning') }}
          </template>
          <template v-else>{{ t('dashboard.home.heroAllGood') }}</template>
        </p>
        <div class="dash-hero__chips">
          <span v-if="nodeStore.isRemote" class="dash-chip">
            {{ t('dashboard.home.remoteNode', { name: nodeStore.contextInfo?.name || '' }) }}
          </span>
          <span class="dash-chip">{{ hostname }}</span>
          <span v-if="osText !== '—'" class="dash-chip">{{ osText }}</span>
          <span class="dash-chip">{{ t('dashboard.home.uptime') }} {{ formatUptime(overview?.uptimeSeconds) }}</span>
        </div>
        <div v-if="quickActions.length" class="dash-hero__actions">
          <button
            v-for="(qa, i) in quickActions"
            :key="qa.path"
            type="button"
            class="dash-hero__btn"
            :class="{ 'is-primary': i === 0 }"
            @click="router.push(qa.path)"
          >
            <component :is="menuStore.iconComponents[qa.icon]" />
            <span>{{ t(qa.label) }}</span>
          </button>
        </div>
      </div>
    </section>

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

    <!-- 指标卡：贴纸图标 + 大号数字 + 圆环/迷你曲线 -->
    <div v-if="canViewStatus" class="dash-stats">
      <div v-for="st in statCards" :key="st.key" class="dash-stat">
        <div class="dash-stat__head">
          <span class="dash-stat__label">
            <span class="dash-stat__icon m-sticker" :class="`m-sticker--${st.tone}`">
              <component :is="menuStore.iconComponents[st.icon]" />
            </span>
            {{ st.label }}
          </span>
          <svg v-if="st.percent != null" class="dash-ring" viewBox="0 0 64 64" role="img" :aria-label="`${st.label} ${st.value}`">
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
        <div class="dash-stat__value m-num">{{ st.value }}</div>
        <div class="dash-stat__sub">{{ st.sub }}</div>
      </div>
    </div>

    <!-- 实时监控 -->
    <SystemRealtimeCharts
      v-if="canViewStatus"
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

    <!-- 底部两栏：运行中实例 / 快捷入口 -->
    <div class="dash-cols">
      <RunningInstanceSection v-if="canViewInstances" class="dash-cols__main" />
      <ShortcutSection class="dash-cols__side" />
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
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import { checkForUpdate } from '@/utils/updateCheck'
import { useAccessibleRoutes } from '@/composables/useAccessibleRoutes'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'HomeView' })

const BUILTIN_SUPER_ADMIN_ROLE_ID = 'role_1'

const dashboardHomeStore = useDashboardHomeStore()
const menuStore = useMenuStore()
const userStore = useUserStore()
const nodeStore = useNodeStore()
const router = useRouter()
const accessible = useAccessibleRoutes()
const { t } = useI18n()

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
const greeting = computed(() => {
  const h = new Date().getHours()
  if (h < 6) return t('dashboard.home.greeting.lateNight')
  if (h < 9) return t('dashboard.home.greeting.morning')
  if (h < 12) return t('dashboard.home.greeting.forenoon')
  if (h < 14) return t('dashboard.home.greeting.noon')
  if (h < 18) return t('dashboard.home.greeting.afternoon')
  return t('dashboard.home.greeting.evening')
})

const fmtPercent = (v?: number) => `${(v ?? 0).toFixed(1)}%`

const runningCount = computed(() => runningInstanceTotal.value)
const hostname = computed(
  () => overview.value?.version?.hostname || nodeStore.contextInfo?.status?.remote?.hostname || '—',
)

// 问候卡快捷操作：只展示能进入的页面
const QUICK_ACTIONS = [
  { path: '/instance/list', label: 'dashboard.home.quickNewApp', icon: 'HOutline:Squares2X2Icon' },
  { path: '/openssh/management', label: 'dashboard.home.quickTerminal', icon: 'HOutline:CommandLineIcon' },
  { path: '/file/index', label: 'dashboard.home.quickFiles', icon: 'HOutline:FolderIcon' },
]
const quickActions = computed(() =>
  QUICK_ACTIONS.filter((a) => accessible.value.has(a.path) && nodeStore.canAccessRoute(a.path)),
)

// 圆环周长 2πr = 163.4（r=26）
const ringLen = (percent: number) => ((Math.min(100, Math.max(0, percent)) / 100) * 163.4).toFixed(1)

const statCards = computed(() => {
  const st = status.value
  return [
    {
      key: 'cpu',
      label: t('dashboard.home.statCpu'),
      icon: 'HOutline:CpuChipIcon',
      tone: 'p',
      color: 'var(--m-p)',
      percent: st?.cpu?.totalPercent ?? 0,
      value: fmtPercent(st?.cpu?.totalPercent),
      sub: overview.value?.cpu?.modelName || '—',
    },
    {
      key: 'memory',
      label: t('dashboard.home.statMemory'),
      icon: 'HOutline:CircleStackIcon',
      tone: 'a2',
      color: 'var(--m-a2)',
      percent: st?.memory?.physicalMemory?.usedPercent ?? 0,
      value: fmtPercent(st?.memory?.physicalMemory?.usedPercent),
      sub: `${formatBytes(st?.memory?.physicalMemory?.usedBytes)} / ${formatBytes(st?.memory?.physicalMemory?.totalBytes)}`,
    },
    {
      key: 'disk',
      label: t('dashboard.home.statDisk'),
      icon: 'HOutline:ServerIcon',
      tone: 'a',
      color: 'var(--m-a)',
      percent: st?.disk?.total?.usedPercent ?? 0,
      value: fmtPercent(st?.disk?.total?.usedPercent),
      sub: `${formatBytes(st?.disk?.total?.usedBytes)} / ${formatBytes(st?.disk?.total?.totalBytes)}`,
    },
    {
      key: 'network',
      label: t('dashboard.home.statNetwork'),
      icon: 'HOutline:ArrowsUpDownIcon',
      tone: 'ok',
      color: 'var(--m-ok)',
      percent: null,
      value: `↓ ${formatBytes(st?.network?.total?.downloadRateBytesPerSecond)}/s`,
      sub: `↑ ${formatBytes(st?.network?.total?.uploadRateBytesPerSecond)}/s · ${t('dashboard.home.totalReceived')} ${formatBytes(st?.network?.total?.bytesRecv)}`,
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


const formatBytes = (bytes?: number | string) => {
  const num = Number(bytes)
  if (!num || num <= 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const base = 1024
  const i = Math.min(Math.floor(Math.log(num) / Math.log(base)), units.length - 1)
  return (num / base ** i).toFixed(i === 0 ? 0 : 1) + ' ' + units[i]
}

const formatUptime = (seconds?: number) => {
  if (seconds == null) return '—'
  const d = Math.floor(seconds / 86400)
  const h = Math.floor((seconds % 86400) / 3600)
  const m = Math.floor((seconds % 3600) / 60)
  if (d > 0) return t('dashboard.home.dayHourMinute', { days: d, hours: h, minutes: m })
  if (h > 0) return t('dashboard.home.hourMinute', { hours: h, minutes: m })
  return t('dashboard.home.minute', { minutes: m })
}

const osText = computed(() => {
  const v = overview.value?.version
  if (!v) return '—'
  const os = `${v.os || ''} ${v.platformVersion || ''}`.trim()
  return [os, v.kernelArch].filter(Boolean).join(' · ') || '—'
})

const autoCheckUpdate = async () => {
  if (hasAutoCheckedUpdate()) return
  if (!userStore.userInfo) await userStore.getUserInfo()
  if (!menuStore.hasLoadedPermissions) await menuStore.getUserPermissions()

  const isBuiltinSuperAdmin = userStore.userInfo?.roleId === BUILTIN_SUPER_ADMIN_ROLE_ID
  if (!isBuiltinSuperAdmin || !menuStore.hasButtonPermission('system:update')) return

  markAutoCheckedUpdate()
  await checkForUpdate({ silentNoUpdate: true })
}

onMounted(() => {
  if (canViewStatus.value) {
    fetchOverview()
    startAutoRefresh()
  }
  startCurrentDateTicker()
  void autoCheckUpdate()
})

// 页面被缓存时暂停实时刷新，回来时立即刷新一次再继续（首次挂载也会触发 onActivated，用标记跳过）
let activatedOnce = false
onActivated(() => {
  if (!activatedOnce) {
    activatedOnce = true
    return
  }
  if (canViewStatus.value) {
    fetchOverview()
    startAutoRefresh()
  }
  startCurrentDateTicker()
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
  gap: 22px;
}

/* ---- 问候卡 ---- */
.dash-hero {
  position: relative;
  overflow: hidden;
  padding: 30px 34px;
  border-radius: var(--app-radius-xl);
  background: var(--m-p);
  color: var(--m-pOn);
}
.dash-hero__blob {
  position: absolute;
  right: -70px;
  bottom: -120px;
  width: 300px;
  height: 300px;
  border-radius: 50%;
  background: var(--m-a);
}
.dash-hero__ring {
  position: absolute;
  right: 200px;
  top: -44px;
  width: 120px;
  height: 120px;
  border-radius: 50%;
  border: 18px solid var(--m-a2);
}
.dash-hero__star {
  position: absolute;
  right: 64px;
  top: 34px;
  width: 42px;
  height: 42px;
  fill: var(--m-pOn);
}
.dash-hero__body {
  position: relative;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 640px;
}
.dash-hero__date {
  font-size: 0.875rem;
  font-weight: 600;
  opacity: 0.9;
}
.dash-hero__title {
  font-family: var(--app-font-display);
  font-size: 2.25rem;
  line-height: 1.15;
  font-weight: 800;
  letter-spacing: -0.01em;
}
.dash-hero__sub {
  font-size: 0.9375rem;
  opacity: 0.95;
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
  font-size: 0.875rem;
  font-weight: 700;
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
  width: 18px;
  height: 18px;
}
@media (width < 640px) {
  .dash-hero {
    padding: 22px 20px;
    border-radius: 26px;
  }
  .dash-hero__blob {
    right: -60px;
    top: -70px;
    bottom: auto;
    width: 170px;
    height: 170px;
  }
  .dash-hero__ring {
    right: -18px;
    top: auto;
    bottom: 96px;
    width: 58px;
    height: 58px;
    border-width: 10px;
  }
  .dash-hero__star {
    display: none;
  }
  .dash-hero__title {
    font-size: 1.625rem;
  }
  .dash-hero__actions {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: 8px;
  }
  .dash-hero__btn {
    flex-direction: column;
    justify-content: center;
    gap: 4px;
    padding: 10px 4px;
    border-radius: 18px;
    font-size: 0.75rem;
  }
}

/* ---- 指标卡 ---- */
.dash-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}
@media (width >= 1100px) {
  .dash-stats {
    grid-template-columns: repeat(4, minmax(0, 1fr));
  }
}
.dash-stat {
  display: flex;
  flex-direction: column;
  gap: 10px;
  min-width: 0;
  padding: 20px;
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
  width: 17px;
  height: 17px;
}
.dash-ring {
  width: 56px;
  height: 56px;
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
  font-size: 2.125rem;
  line-height: 1;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.dash-stat__sub {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
@media (width < 640px) {
  .dash-stat {
    padding: 16px;
    gap: 8px;
  }
  .dash-stat__label {
    font-size: 0.8125rem;
  }
  .dash-stat__icon {
    display: none;
  }
  .dash-ring {
    width: 38px;
    height: 38px;
  }
  .dash-spark {
    width: 60px;
    height: 28px;
  }
  .dash-stat__value {
    font-size: 1.5rem;
  }
}

/* ---- 底部两栏 ---- */
.dash-cols {
  display: grid;
  grid-template-columns: 1fr;
  gap: 22px;
}
@media (width >= 1100px) {
  .dash-cols {
    grid-template-columns: 3fr 2fr;
  }
}
</style>
