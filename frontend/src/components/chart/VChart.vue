<template>
  <VueECharts ref="inner" v-bind="$attrs" />
</template>

<script setup lang="ts">
import { use } from 'echarts/core'
import { CanvasRenderer } from 'echarts/renderers'
import {
  LineChart,
  RadarChart,
  PieChart,
  BarChart,
  GaugeChart,
  EffectScatterChart,
} from 'echarts/charts'
import {
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
  GeoComponent,
  MarkAreaComponent,
  ToolboxComponent,
} from 'echarts/components'
import VueECharts from 'vue-echarts'

defineOptions({
  name: 'VChart',
  inheritAttrs: false,
})

use([
  CanvasRenderer,
  LineChart,
  RadarChart,
  PieChart,
  BarChart,
  GaugeChart,
  EffectScatterChart,
  TitleComponent,
  TooltipComponent,
  GridComponent,
  LegendComponent,
  DataZoomComponent,
  GeoComponent,
  MarkAreaComponent,
  ToolboxComponent,
])

// 暴露内部 vue-echarts 实例（dispatchAction / convertFromPixel 等），供需要命令式控制的图表使用
const inner = ref<InstanceType<typeof VueECharts>>()
defineExpose({ inner })
</script>
