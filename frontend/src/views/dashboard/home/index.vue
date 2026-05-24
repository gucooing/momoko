<template>
  <div class="dashboard-home">
    <WelcomePanel />
    <system-realtime-charts
      :cpu-history="cpuHistory"
      :memory-history="memoryHistory"
      :network-history="networkHistory"
      :disk-history="diskHistory"
      :refresh-interval="refreshInterval"
      :selected-interface="selectedInterface"
      :selected-disk="selectedDisk"
      :overview="chartOverview"
      @interval-change="onIntervalChange"
      @interface-change="onInterfaceChange"
      @disk-change="onDiskChange"
    />
    <el-row :gutter="20" class="dashboard-bottom-row">
      <el-col :xs="24" :lg="16">
        <RunningInstanceSection />
      </el-col>
      <el-col :xs="24" :lg="8" class="dashboard-bottom-right">
        <ShortcutSection />
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import WelcomePanel from '@/views/dashboard/home/welcomePanel.vue'
import RunningInstanceSection from '@/views/dashboard/home/runningInstanceSection.vue'
import ShortcutSection from '@/views/dashboard/home/shortcutSection.vue'
import SystemRealtimeCharts from '@/views/dashboard/home/systemRealtimeCharts.vue'
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import type { RefreshInterval } from '@/stores/dashboard/home'

defineOptions({ name: 'HomeView' })

const dashboardHomeStore = useDashboardHomeStore()

const { cpuHistory, memoryHistory, networkHistory, diskHistory, refreshInterval, selectedInterface, selectedDisk, overview } =
  storeToRefs(dashboardHomeStore)
const {
  fetchOverview,
  startAutoRefresh,
  stopAutoRefresh,
  setRefreshInterval,
  setSelectedInterface,
  setSelectedDisk,
} = dashboardHomeStore

const chartOverview = computed(() => {
  if (!overview.value) return null
  return {
    networkInterfaces: overview.value.networkInterfaces || [],
    diskPartitions: overview.value.diskPartitions || [],
  }
})

const onIntervalChange = (interval: RefreshInterval) => {
  setRefreshInterval(interval)
}

const onInterfaceChange = (value: string) => {
  setSelectedInterface(value)
}

const onDiskChange = (value: string) => {
  setSelectedDisk(value)
}

onMounted(() => {
  fetchOverview()
  startAutoRefresh()
})

onBeforeUnmount(() => {
  stopAutoRefresh()
})
</script>

<style scoped lang="scss">
.dashboard-home {
  display: flex;
  flex: 1;
  height: 100%;
  flex-direction: column;
  gap: 1rem;

  @media (width >= 640px) {
    gap: 1.5rem;
  }
}

.dashboard-bottom-right {
  margin-top: 1rem;

  @media (width >= 1024px) {
    margin-top: 0;
  }
}
</style>
