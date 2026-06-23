<!-- 热销商品类目 TOP 5 -->
<template>
  <BaseCard :title="t('dashboard.analysis.topCategoriesTitle')">
    <div class="h-65 w-full">
      <VChart :option="topCategoriesOption" autoresize />
    </div>
  </BaseCard>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import VChart from '@/components/chart/VChart.vue'
import { useDashboardAnalysisStore } from '@/stores/dashboard/analysis'
import { useI18n } from 'vue-i18n'

// 触发器变量（仅仅用来主题或者颜色变化时触发revenueProfitOption 更新的变量）
const colorTrigger = ref(0)
const dashboardAnalysisStore = useDashboardAnalysisStore()
const { topCategories } = storeToRefs(dashboardAnalysisStore)
const { t } = useI18n()

// 4. 热销品类柱状图
const topCategoriesOption = computed(() => {
  void colorTrigger.value
  const style = getComputedStyle(document.documentElement)
  const categoryData = topCategories.value.map((item) => ({
    ...item,
    name: t(item.nameKey),
  }))

  return {
    tooltip: {
      trigger: 'axis',
      axisPointer: {
        type: 'shadow',
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
    grid: { left: '3%', right: '8%', bottom: '3%', top: '5%', containLabel: true },
    xAxis: {
      type: 'value',
      axisLine: { show: false },
      splitLine: {
        lineStyle: { type: 'dashed', color: style.getPropertyValue('--el-border-color') },
      },
      axisLabel: { color: style.getPropertyValue('--el-text-color-regular') },
    },
    yAxis: {
      type: 'category',
      data: categoryData.map((item) => item.name),
      axisLine: { show: false },
      axisTick: { show: false },
      axisLabel: { color: style.getPropertyValue('--el-text-color-regular') },
    },
    series: [
      {
        name: t('dashboard.analysis.salesK'),
        type: 'bar',
        barWidth: '40%',
        itemStyle: {
          borderRadius: [0, 20, 20, 0],
          color: style.getPropertyValue('--el-color-primary'),
        },
        data: categoryData.map((item) => item.value),
      },
    ],
  }
})

// 触发颜色改变
const updateColorTrigger = () => colorTrigger.value++

defineExpose({ updateColorTrigger })
</script>

<style scoped lang="scss"></style>
