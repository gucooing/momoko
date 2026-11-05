<!-- 性能监控页的线性图标（与原型 MonitorPC / MonitorMobile 同一套路径），描边随文字颜色。 -->
<template>
  <svg class="mglyph" viewBox="0 0 24 24" aria-hidden="true"><path :d="PATHS[name]" /></svg>
</template>

<script setup lang="ts">
export type MonitorGlyphName = 'cpu' | 'gpu' | 'mem' | 'disk' | 'net' | 'cal' | 'pulse' | 'settings' | 'download' | 'trash' | 'db' | 'apps'

defineProps<{ name: MonitorGlyphName }>()

const PATHS: Record<MonitorGlyphName, string> = {
  cpu: 'M7 7h10v10H7zM10 10h4v4h-4zM10 2v5M14 2v5M10 17v5M14 17v5M2 10h5M2 14h5M17 10h5M17 14h5',
  gpu: 'M2 7h17a2 2 0 0 1 2 2v6a2 2 0 0 1-2 2H2zM6 10.5a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3zM12 10.5a1.5 1.5 0 1 1 0 3 1.5 1.5 0 0 1 0-3zM5 17v3M16 10h2M16 13h2',
  mem: 'M3 7h18v10H3zM7 7v10M11 7v10M15 7v10M3 20v-3M21 20v-3M8 20v-3M16 20v-3',
  disk: 'M12 3c4.4 0 8 1.3 8 3v12c0 1.7-3.6 3-8 3s-8-1.3-8-3V6c0-1.7 3.6-3 8-3zM4 6c0 1.7 3.6 3 8 3s8-1.3 8-3M17 15h.01',
  net: 'M9 3h6v5H9zM3 16h6v5H3zM15 16h6v5h-6zM12 8v4M6 16v-4h12v4',
  cal: 'M4 6h16v14H4zM4 10h16M8 3v5M16 3v5',
  pulse: 'M3 12h4l2.5-6 4 12 2.5-6H21',
  settings:
    'M12 9a3 3 0 1 1 0 6 3 3 0 0 1 0-6zM19.4 15a1.7 1.7 0 0 0 .3 1.8l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-2.9 1.2V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-2.9-1.2l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1A1.7 1.7 0 0 0 3.1 14H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.2-2.9l-.1-.1A2 2 0 1 1 7 4.2l.1.1A1.7 1.7 0 0 0 10 3.1V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 2.9 1.2l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0 1.2 2.9H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z',
  download: 'M12 4v12M7 11l5 5 5-5M4 20h16',
  trash: 'M4 7h16M9 7V4h6v3M6 7l1 13h10l1-13',
  db: 'M4 6c0-1.7 3.6-3 8-3s8 1.3 8 3-3.6 3-8 3-8-1.3-8-3zM4 6v12c0 1.7 3.6 3 8 3s8-1.3 8-3V6M4 12c0 1.7 3.6 3 8 3s8-1.3 8-3',
  apps: 'M21 8 12 3 3 8v8l9 5 9-5zM3 8l9 5 9-5M12 13v8',
}
</script>

<style scoped>
.mglyph {
  width: 1em;
  height: 1em;
  flex: none;
  fill: none;
  stroke: currentColor;
  stroke-width: 1.9;
  stroke-linecap: round;
  stroke-linejoin: round;
}
</style>
