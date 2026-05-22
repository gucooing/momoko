<!-- 资源面板 -->
<template>
  <el-row :gutter="20">
    <el-col :sm="12" :lg="6" v-for="item in resourceStatsWithOption" :key="item.label" class="mt-4">
      <BaseCard>
        <div class="flex items-center justify-between">
          <div class="flex flex-col gap-3">
            <div class="text-sm font-medium text-(--el-text-color-regular)">
              {{ item.label }}
            </div>
            <div class="flex items-baseline gap-1">
              <span class="text-2xl font-extrabold">{{ item.value }}</span>
              <span class="text-sm font-semibold text-(--el-text-color-secondary)">{{
                item.unit
              }}</span>
            </div>
            <div class="flex gap-2 text-xs">
              <span
                class="font-bold"
                :class="
                  item.trend.startsWith('+')
                    ? 'text-(--el-color-success)'
                    : 'text-(--el-color-danger)'
                "
                >{{ item.trend }}</span
              >
              <span class="text-(--el-text-color-secondary)">vs last hour</span>
            </div>
          </div>
          <div class="h-23 w-23">
            <VChart :option="item.option" />
          </div>
        </div>
      </BaseCard>
    </el-col>
  </el-row>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import VChart from '@/components/chart/VChart.vue'
import { useDashboardMonitorStore } from '@/stores/dashboard/monitor'

// 触发器变量（仅仅用来主题或者颜色变化时触发revenueProfitOption 更新的变量）
const colorTrigger = ref(0)
let timer: ReturnType<typeof setInterval> | null = null
const dashboardMonitorStore = useDashboardMonitorStore()
const { resourceStats } = storeToRefs(dashboardMonitorStore)
const { refreshResourceStats } = dashboardMonitorStore

/**
 * 创建迷你仪表盘图表配置
 * @param value  值
 * @param color  颜色
 */
const createMiniGauge = (value: number, color: string) => {
  const style = getComputedStyle(document.documentElement)
  return {
    series: [
      {
        type: 'gauge',
        startAngle: 90,
        endAngle: -270,
        pointer: { show: false },
        progress: {
          show: true,
          overlap: false,
          roundCap: true,
          clip: false,
          itemStyle: { color: style.getPropertyValue(color) },
        },
        axisLine: {
          lineStyle: { width: 8, color: [[1, style.getPropertyValue('--el-bg-color-page')]] },
        },
        splitLine: { show: false },
        axisTick: { show: false },
        axisLabel: { show: false },
        data: [{ value }],
        detail: { show: false },
      },
    ],
  }
}

//  动态创建图表配置
const resourceStatsWithOption = computed(() => {
  void colorTrigger.value
  return resourceStats.value.map((item) => ({
    ...item,
    option: createMiniGauge(item.value, item.color),
  }))
})

// 触发颜色改变
const updateColorTrigger = () => colorTrigger.value++

defineExpose({ updateColorTrigger })

onMounted(() => {
  timer = setInterval(refreshResourceStats, 5000)
})

onUnmounted(() => {
  if (timer) clearInterval(timer)
})
</script>

<style scoped lang="scss"></style>
