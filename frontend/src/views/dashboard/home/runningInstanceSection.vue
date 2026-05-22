<template>
  <BaseCard title="正在运行中的实例" title-icon="HOutline:ServerStackIcon">
    <div v-loading="runningInstancesLoading">
      <el-empty v-if="!runningInstancesLoading && !runningInstances.length" description="暂无运行中的实例" />
      <el-scrollbar v-else :height="520">
        <div class="grid grid-cols-[repeat(auto-fill,minmax(300px,1fr))] gap-5 pt-3">
          <RunningInstanceCard
            v-for="instance in runningInstances"
            :key="instance.id"
            :instance="instance"
            @open="openInstanceConsole"
          />
        </div>
      </el-scrollbar>
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import RunningInstanceCard from '@/components/card/RunningInstanceCard.vue'
import { useDashboardHomeStore } from '@/stores/dashboard/home'
import type { InstanceInfo } from '@/types/v1/instance'

const router = useRouter()
const dashboardHomeStore = useDashboardHomeStore()
const { runningInstancesLoading, runningInstances } = storeToRefs(dashboardHomeStore)
const { getRunningInstances } = dashboardHomeStore

const openInstanceConsole = (instance: InstanceInfo) => {
  router.push({
    path: `/instance/console/${instance.id}`,
    query: {
      tabTitle: instance.name,
    },
  })
}

onMounted(() => {
  void getRunningInstances()
})
</script>

<style scoped lang="scss"></style>
