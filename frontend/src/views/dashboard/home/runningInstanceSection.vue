<template>
  <BaseCard :title="t('dashboard.home.runningInstances')" title-icon="HOutline:ServerStackIcon">
    <div v-loading="runningInstancesLoading">
      <el-empty
        v-if="!runningInstancesLoading && !runningInstances.length"
        :description="t('dashboard.home.noRunningInstances')"
      />
      <el-scrollbar v-else :height="520" class="running-scrollbar">
        <div class="running-grid">
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
import { useI18n } from 'vue-i18n'

const router = useRouter()
const dashboardHomeStore = useDashboardHomeStore()
const { t } = useI18n()
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

<style scoped lang="scss">
.running-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 1rem;
  padding-top: 0.75rem;

  @media (width >= 640px) {
    grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
    gap: 1.25rem;
  }
}

.running-scrollbar {
  @media (width <= 640px) {
    max-height: 360px !important;
    height: auto;
  }
}
</style>
