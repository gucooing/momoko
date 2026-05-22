<template>
  <!-- 营销目标和今日之星 -->
  <base-card>
    <div class="flex flex-col gap-2">
      <div class="flex items-center justify-center w-10 h-10 rounded-full bg-[#f97316]">
        <el-icon size="20">
          <component :is="menuStore.iconComponents['HSolid:TrophyIcon']" class="text-white" />
        </el-icon>
      </div>
      <div class="flex items-center justify-between">
        <div class="font-semibold text-sm">{{ goalProgress.title }}</div>
        <div class="font-bold text-[18px]">{{ goalProgress.percentage }}%</div>
      </div>
      <el-progress
        :percentage="goalProgress.percentage"
        :stroke-width="15"
        :color="goalProgress.color"
        striped
        striped-flow
        :duration="50"
        :show-text="false"
      />
    </div>
  </base-card>
  <base-card class="mt-2">
    <div class="flex flex-col items-center justify-center gap-4">
      <div class="text-[#f97316] text-xs font-semibold">{{ operationManager.title }}</div>
      <el-avatar :size="64" :src="operationManager.avatar" />
      <div class="text-[18px] font-bold">{{ operationManager.name }}</div>
      <div class="text-[13px] text-(--el-text-color-regular)">{{ operationManager.role }}</div>
    </div>
  </base-card>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useDashboardAnalysisStore } from '@/stores/dashboard/analysis'

const menuStore = useMenuStore()
const dashboardAnalysisStore = useDashboardAnalysisStore()
const { goalProgress, operationManager } = storeToRefs(dashboardAnalysisStore)
</script>
<style scoped lang="scss"></style>
