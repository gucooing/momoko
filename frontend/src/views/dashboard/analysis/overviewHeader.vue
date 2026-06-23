<!-- 业务运营中心 -->
<template>
  <div>
    <el-row :gutter="20">
      <el-col :xs="24" :lg="9">
        <BaseCard>
          <div class="flex items-center justify-between gap-4 px-3">
            <div class="flex flex-col gap-4 py-2.5">
              <div class="text-sm font-semibold text-(--el-color-primary)">{{ t('dashboard.analysis.businessOverview') }}</div>
              <h2 class="text-2xl font-bold">{{ t('dashboard.analysis.operationsCenter') }}</h2>
              <p class="text-sm text-(--el-text-color-regular) md:max-w-80">
                {{ t('dashboard.analysis.overviewText', { percentage: goalProgress.percentage }) }}
              </p>
              <div class="flex gap-4 items-center">
                <el-button type="primary" round>{{ t('dashboard.analysis.generateMonthlyReport') }}</el-button>
                <el-button link>{{ t('dashboard.analysis.performanceForecast') }}</el-button>
              </div>
            </div>
            <div
              class="hidden md:block max-w-full mx-auto w-40 h-40 sm:w-50 sm:h-50 md:w-60 md:h-50 lg:w-70 lg:h-55 xl:w-60 xl:h-50"
            >
              <LottieAnimation :path="analysisLottieUrl" width="100%" height="100%" />
            </div>
          </div>
        </BaseCard>
      </el-col>
      <el-col
        :xs="24"
        :sm="8"
        :lg="5"
        v-for="item in businessStats"
        :key="item.labelKey"
        class="mt-4 min-[1200px]:mt-0"
      >
        <BaseCard :class="item.type" style="height: 100%">
          <div class="h-full flex flex-col gap-2 justify-center p-5">
            <div class="w-12 h-12 bg-white/20 rounded-xl flex items-center justify-center">
              <el-icon size="24">
                <component :is="menuStore.iconComponents[item.icon]" class="text-white" />
              </el-icon>
            </div>
            <div class="text-white break-all">
              <span class="text-xl font-bold">{{ item.value }}</span>
              <span class="text-xs opacity-80 ml-2">{{ item.trend }}</span>
            </div>
            <div class="text-sm opacity-80 text-white">{{ t(item.labelKey) }}</div>
          </div>
        </BaseCard>
      </el-col>
    </el-row>
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import analysisLottieUrl from '@/assets/lotties/colheita.json?url'
import { useDashboardAnalysisStore } from '@/stores/dashboard/analysis'
import { useI18n } from 'vue-i18n'

const menuStore = useMenuStore()
const dashboardAnalysisStore = useDashboardAnalysisStore()
const { businessStats, goalProgress } = storeToRefs(dashboardAnalysisStore)
const { t } = useI18n()
</script>

<style scoped lang="scss">
:deep(.el-card__body) {
  height: 100%;
}

.blue {
  background: linear-gradient(135deg, #5bbff9 0%, #2563eb 100%);
}
.orange {
  background: linear-gradient(135deg, #f99c7d 0%, #ea580c 100%);
}
.indigo {
  background: linear-gradient(135deg, #818cf8 0%, #4f46e5 100%);
}
</style>
