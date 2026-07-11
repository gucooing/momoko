<!-- 工作台（重写）：安静问候 + 单色指标带 + 实时监控 + 运行实例/快捷入口 + 系统信息。
     保留 useDashboardHomeStore 全部数据与轮询逻辑（06a）。 -->
<template>
  <div class="dash">
    <!-- 问候区（安静，非大卡） -->
    <header class="dash-hi">
      <div class="dash-hi__text">
        <h2 class="dash-hi__title">
          {{ greeting }}，{{ displayName }}<span class="dash-hi__wave">👋</span>
        </h2>
        <div class="dash-hi__meta">
          <span>{{ overview?.version?.hostname || '—' }}</span>
          <i class="dash-hi__sep" />
          <span>{{ osText }}</span>
          <i class="dash-hi__sep" />
          <span>{{ t('dashboard.home.uptime') }} {{ formatUptime(overview?.uptimeSeconds) }}</span>
          <i class="dash-hi__sep" />
          <span class="tnum">{{ currentDate }}</span>
        </div>
      </div>
    </header>

    <!-- 指标带（单色，无彩色图标盒） -->
    <MetricStrip :columns="4">
      <MetricItem
        :label="t('dashboard.home.cpuUsage')"
        :value="fmtPercent(status?.cpu?.totalPercent)"
        :percent="status?.cpu?.totalPercent ?? 0"
        :caption="overview?.cpu?.modelName || '—'"
      />
      <MetricItem
        :label="t('dashboard.home.memoryInfo')"
        :value="fmtPercent(status?.memory?.physicalMemory?.usedPercent)"
        :percent="status?.memory?.physicalMemory?.usedPercent ?? 0"
        :caption="`${formatBytes(status?.memory?.physicalMemory?.usedBytes)} / ${formatBytes(status?.memory?.physicalMemory?.totalBytes)}`"
      />
      <MetricItem
        :label="t('dashboard.home.diskUsage')"
        :value="fmtPercent(status?.disk?.total?.usedPercent)"
        :percent="status?.disk?.total?.usedPercent ?? 0"
        :caption="`${formatBytes(status?.disk?.total?.usedBytes)} / ${formatBytes(status?.disk?.total?.totalBytes)}`"
      />
      <MetricItem :label="t('dashboard.home.networkInfo')">
        <template #value>
          <div class="dash-net">
            <span class="dash-net__down">↓ {{ formatBytes(status?.network?.total?.downloadRateBytesPerSecond) }}/s</span>
            <span class="dash-net__up">↑ {{ formatBytes(status?.network?.total?.uploadRateBytesPerSecond) }}/s</span>
          </div>
        </template>
        <template #caption>
          {{ t('dashboard.home.totalReceived') }} {{ formatBytes(status?.network?.total?.bytesRecv) }}
          · {{ t('dashboard.home.totalSent') }} {{ formatBytes(status?.network?.total?.bytesSent) }}
        </template>
      </MetricItem>
    </MetricStrip>

    <!-- 实时监控 -->
    <SystemRealtimeCharts
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
      <RunningInstanceSection class="dash-cols__main" />
      <ShortcutSection class="dash-cols__side" />
    </div>
  </div>
</template>

<script lang="ts">
let hasAutoCheckedUpdate = false
</script>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import RunningInstanceSection from '@/views/dashboard/home/runningInstanceSection.vue'
import ShortcutSection from '@/views/dashboard/home/shortcutSection.vue'
import SystemRealtimeCharts from '@/views/dashboard/home/systemRealtimeCharts.vue'
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import { checkForUpdate } from '@/utils/updateCheck'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'HomeView' })

const BUILTIN_SUPER_ADMIN_ROLE_ID = 'role_1'

const dashboardHomeStore = useDashboardHomeStore()
const menuStore = useMenuStore()
const userStore = useUserStore()
const { t } = useI18n()

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
  if (h < 6) return '凌晨好'
  if (h < 9) return '早上好'
  if (h < 12) return '上午好'
  if (h < 14) return '中午好'
  if (h < 18) return '下午好'
  return '晚上好'
})

const fmtPercent = (v?: number) => `${(v ?? 0).toFixed(1)}%`

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
  if (hasAutoCheckedUpdate) return
  if (!userStore.userInfo) await userStore.getUserInfo()
  if (!menuStore.hasLoadedPermissions) await menuStore.getUserPermissions()

  const isBuiltinSuperAdmin = userStore.userInfo?.roleId === BUILTIN_SUPER_ADMIN_ROLE_ID
  if (!isBuiltinSuperAdmin || !menuStore.hasButtonPermission('system:update')) return

  hasAutoCheckedUpdate = true
  await checkForUpdate({ silentNoUpdate: true })
}

onMounted(() => {
  fetchOverview()
  startAutoRefresh()
  startCurrentDateTicker()
  void autoCheckUpdate()
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
  gap: 20px;
}

/* 问候区 */
.dash-hi {
  display: flex;
  align-items: center;
  gap: 1rem;
  padding: 4px 2px;
}
.dash-hi__title {
  font-size: 1.375rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
  display: flex;
  align-items: center;
}
.dash-hi__wave {
  margin-left: 0.4rem;
  font-weight: 400;
}
.dash-hi__meta {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 6px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.dash-hi__meta .tnum {
  font-variant-numeric: tabular-nums;
}
.dash-hi__sep {
  width: 3px;
  height: 3px;
  border-radius: 999px;
  background: var(--el-text-color-placeholder);
  opacity: 0.6;
}

/* 网络指标双行 */
.dash-net {
  display: flex;
  flex-direction: column;
  gap: 2px;
  font-size: 1rem;
  font-weight: 700;
  line-height: 1.2;
  font-variant-numeric: tabular-nums;
}
.dash-net__down {
  color: var(--el-color-success, #16a34a);
}
.dash-net__up {
  color: var(--el-color-primary);
}

/* 底部两栏 */
.dash-cols {
  display: grid;
  grid-template-columns: 1fr;
  gap: 20px;
}
@media (width >= 1024px) {
  .dash-cols {
    grid-template-columns: 2fr 1fr;
  }
}
</style>
