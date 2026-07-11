<!-- 指标带：一块面板 + 竖向 hairline 分隔（1px gap 透出底色），响应式 4→2→1。
     不是 N 张浮卡（01 §8）。 -->
<template>
  <div class="metric-strip" :style="{ '--cols': columns }">
    <slot />
  </div>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ columns?: number }>(), { columns: 4 })
</script>

<style scoped lang="scss">
.metric-strip {
  display: grid;
  grid-template-columns: 1fr;
  gap: 1px;
  background: var(--el-border-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  overflow: hidden;
  box-shadow: var(--app-shadow-card);
}
@media (width >= 640px) {
  .metric-strip {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
}
@media (width >= 1024px) {
  .metric-strip {
    grid-template-columns: repeat(var(--cols, 4), minmax(0, 1fr));
  }
}
</style>
