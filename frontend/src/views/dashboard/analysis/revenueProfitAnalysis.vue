<!-- 年度营收与净利润增长深度分析 -->
<template>
  <BaseCard>
    <template #header>
      <div class="flex items-center justify-between gap-2">
        <div class="flex flex-1 items-center">
          <div class="text-[18px] font-bold">
            <TextEllipsis text="年度营收与净利润增长深度分析" :clickable="false" />
          </div>
          <div class="ml-4 hidden sm:block">
            <BaseTag text="数据已脱敏" />
          </div>
        </div>
        <div class="flex shrink-0 items-center sm:gap-2">
          <el-radio-group v-model="analysisTimeRange" size="small">
            <el-radio-button label="1y">近1年</el-radio-button>
            <el-radio-button label="2y">近2年</el-radio-button>
          </el-radio-group>
          <el-divider direction="vertical" />
          <IconButton icon="HOutline:ArrowPathIcon" type="primary" @click="refreshData" />
        </div>
      </div>
    </template>
    <div class="h-113 w-full">
      <VChart :option="revenueProfitOption" autoresize />
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import VChart from '@/components/chart/VChart.vue'
import { useDashboardAnalysisStore } from '@/stores/dashboard/analysis'

// 触发器变量（仅仅用来主题或者颜色变化时触发revenueProfitOption 更新的变量）
const colorTrigger = ref(0)
const dashboardAnalysisStore = useDashboardAnalysisStore()
const { analysisTimeRange, analysisData } = storeToRefs(dashboardAnalysisStore)
const { refreshData } = dashboardAnalysisStore

const revenueProfitOption = computed(() => {
  void colorTrigger.value
  const style = getComputedStyle(document.documentElement)
  const data = analysisData.value[analysisTimeRange.value]

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'cross',
        label: { backgroundColor: style.getPropertyValue('--el-color-primary') },
      },
      backgroundColor: style.getPropertyValue('--el-text-color-primary'),
      borderWidth: 0,
      borderRadius: 8,
      boxShadow: '0 8px 24px rgba(0,0,0,0.1)',
      textStyle: {
        color: style.getPropertyValue('--el-bg-color'),
        fontWeight: 400,
        fontSize: 12,
      },
    },
    legend: {
      data: ['年度营收', '净利润', '去年同期', '利润率'],
      bottom: 0,
      itemGap: 20,
      textStyle: { color: style.getPropertyValue('--el-text-color-regular') },
    },
    grid: { left: '3%', right: '4%', bottom: 60, top: '10%', containLabel: true },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: data?.months,
      axisLine: { lineStyle: { color: style.getPropertyValue('--el-text-color-regular') } },
      axisTick: { show: false },
      axisLabel: { color: style.getPropertyValue('--el-text-color-regular') },
    },
    // dataZoom: [
    //   {
    //     type: 'inside',
    //     xAxisIndex: 0,
    //     zoomOnMouseWheel: true,
    //     moveOnMouseMove: true,
    //     moveOnMouseWheel: true,
    //   },
    //   {
    //     type: 'slider',
    //     height: 20,
    //     // 滑块选中区域
    //     fillerColor: style.getPropertyValue('--el-color-primary-light-5'),
    //     // 文字
    //     textStyle: {
    //       color: style.getPropertyValue('--el-text-color-secondary'),
    //       fontSize: 12,
    //     },
    //   },
    // ],
    yAxis: [
      {
        type: 'value',
        name: '金额 (k)',
        axisLine: { show: false },
        splitLine: {
          lineStyle: { type: 'dashed', color: style.getPropertyValue('--el-border-color') },
        },
        axisLabel: { color: style.getPropertyValue('--el-text-color-regular') },
        nameTextStyle: { color: style.getPropertyValue('--el-text-color-regular') },
      },
      {
        type: 'value',
        name: '利润率 (%)',
        min: 0,
        max: 100,
        axisLine: { show: false },
        splitLine: { show: false },
        axisLabel: { color: style.getPropertyValue('--el-text-color-regular') },
        nameTextStyle: { color: style.getPropertyValue('--el-text-color-regular') },
      },
    ],
    series: [
      {
        name: '年度营收',
        type: 'line',
        smooth: true,
        lineStyle: { width: 0 },
        showSymbol: false,
        areaStyle: {
          opacity: 0.8,
          color: {
            type: 'linear',
            x: 0,
            y: 0,
            x2: 0,
            y2: 1,
            colorStops: [
              { offset: 0, color: style.getPropertyValue('--el-color-info') },
              { offset: 1, color: style.getPropertyValue('--el-color-primary') },
            ],
          },
        },
        data: data?.revenue,
      },
      {
        name: '去年同期',
        type: 'line',
        symbol: 'none',
        lineStyle: {
          type: 'dashed',
          width: 2,
          color: style.getPropertyValue('--el-text-color-placeholder'),
        },
        data: data?.lastYear,
      },
      {
        name: '净利润',
        type: 'line',
        smooth: true,
        lineStyle: { width: 4, color: style.getPropertyValue('--el-color-warning') },
        symbol: 'circle',
        symbolSize: 8,
        itemStyle: {
          color: style.getPropertyValue('--el-color-warning'),
          borderColor: style.getPropertyValue('--el-color-white'),
          borderWidth: 2,
        },
        data: data?.profit,
      },
      {
        name: '利润率',
        type: 'line',
        yAxisIndex: 1,
        lineStyle: {
          type: 'dashed',
          color: style.getPropertyValue('--el-color-success'),
          width: 2,
        },
        symbol: 'none',
        data: data?.profitRate,
      },
    ],
  }
})

// 触发颜色改变
const updateColorTrigger = () => colorTrigger.value++

defineExpose({ updateColorTrigger })
</script>

<style scoped lang="scss"></style>
