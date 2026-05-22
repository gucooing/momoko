<!-- 全球市场份额分布 -->
<template>
  <BaseCard title="全球市场份额分布">
    <div class="h-65 w-full">
      <VChart :option="marketShareOption" autoresize />
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
const { marketShares } = storeToRefs(dashboardAnalysisStore)

// 全球市场份额饼图
const marketShareOption = computed(() => {
  void colorTrigger.value
  const style = getComputedStyle(document.documentElement)
  const shareData = marketShares.value.map((item) => ({
    ...item,
    itemStyle: {
      color: style.getPropertyValue(item.colorVar),
    },
  }))

  return {
    tooltip: {
      trigger: 'item',
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
      bottom: '5%',
      icon: 'circle',
      itemGap: 15,
      textStyle: { color: style.getPropertyValue('--el-text-color-regular') },
    },
    series: [
      {
        name: '市场份额',
        type: 'pie',
        radius: ['45%', '70%'],
        center: ['50%', '45%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 8,
          borderColor: style.getPropertyValue('--el-bg-color-overlay'),
          borderWidth: 4,
        },
        label: { show: false },
        emphasis: {
          label: {
            show: true,
            fontSize: '14',
            fontWeight: 'bold',
            color: style.getPropertyValue('--el-text-color-regular'),
          },
        },
        data: shareData,
      },
    ],
  }
})

// 触发颜色改变
const updateColorTrigger = () => colorTrigger.value++

defineExpose({ updateColorTrigger })
</script>

<style scoped lang="scss"></style>

