<template>
  <div class="flex-1 h-full flex flex-col gap-6">
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
    <el-row :gutter="20">
      <el-col :xs="24" :lg="16">
        <RunningInstanceSection />
      </el-col>
      <el-col :xs="24" :lg="8" class="mt-5 lg:mt-0">
        <ShortcutSection class="mt-4 min-[1200px]:mt-0" />
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

<style></style>
