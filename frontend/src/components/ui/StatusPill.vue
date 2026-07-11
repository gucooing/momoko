<!-- 状态胶囊：圆点 + 语义色淡底 + 语义文字（01 §1.3）。不做满色块。 -->
<template>
  <span class="status-pill" :style="pillStyle">
    <span v-if="dot" class="status-pill__dot" :style="{ background: tone }" />
    <slot>{{ label }}</slot>
  </span>
</template>

<script setup lang="ts">
type Variant = 'success' | 'warning' | 'error' | 'info' | 'neutral' | 'primary'
const props = withDefaults(
  defineProps<{ variant?: Variant; label?: string; dot?: boolean }>(),
  { variant: 'neutral', dot: true },
)

const toneMap: Record<Variant, string> = {
  success: 'var(--el-color-success, #16a34a)',
  warning: 'var(--el-color-warning, #f59e0b)',
  error: 'var(--el-color-danger, #ef4444)',
  info: '#64748b',
  neutral: 'var(--el-text-color-secondary)',
  primary: 'var(--el-color-primary)',
}
const tone = computed(() => toneMap[props.variant])
const pillStyle = computed(() => ({
  color: tone.value,
  background: `color-mix(in srgb, ${tone.value} 12%, transparent)`,
}))
</script>

<style scoped lang="scss">
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 0.75rem;
  font-weight: 600;
  line-height: 1.5;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
.status-pill__dot {
  width: 6px;
  height: 6px;
  border-radius: 999px;
  flex-shrink: 0;
}
</style>
