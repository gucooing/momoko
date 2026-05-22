<template>
  <VueECharts v-bind="$attrs" />
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
])
</script>
