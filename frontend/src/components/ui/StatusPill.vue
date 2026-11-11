<!-- 状态胶囊：圆点 + 语义色淡底 + 语义文字（01 §1.3）。主题的「状态标签：实心」会改成语义色实底、去掉圆点（html.theme-pill-fill）。 -->
<template>
  <span class="status-pill" :style="pillStyle">
    <span v-if="dot" class="status-pill__dot" />
    <slot>{{ label }}</slot>
  </span>
</template>

<script setup lang="ts">
type Variant = 'success' | 'warning' | 'error' | 'info' | 'neutral' | 'primary'
const props = withDefaults(
  defineProps<{ variant?: Variant; label?: string; dot?: boolean }>(),
  { variant: 'neutral', dot: true },
)

// 文字色 + 淡底色均取自当前配色令牌，保证各配色下对比度达标
const toneMap: Record<Variant, [string, string]> = {
  success: ['var(--m-ok)', 'var(--m-okSoft)'],
  warning: ['var(--m-warn)', 'var(--m-warnSoft)'],
  error: ['var(--m-bad)', 'var(--m-badSoft)'],
  info: ['var(--m-ink)', 'var(--m-a2Soft)'],
  neutral: ['var(--m-ink2)', 'var(--m-s2)'],
  primary: ['var(--m-pInk)', 'var(--m-pSoft)'],
}
const pillStyle = computed(() => ({
  '--pill-fg': toneMap[props.variant][0],
  '--pill-bg': toneMap[props.variant][1],
}))
</script>

<style scoped lang="scss">
.status-pill {
  display: inline-flex;
  align-items: center;
  gap: 0.375rem;
  padding: 3px var(--cmp-pill-padding-x);
  border-radius: 999px;
  font-size: var(--cmp-pill-font-size);
  font-weight: 700;
  line-height: 1.5;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
  color: var(--pill-fg);
  background: var(--pill-bg);
}
.status-pill__dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  flex-shrink: 0;
  background: var(--pill-fg);
}
:global(html.theme-pill-fill) .status-pill {
  color: var(--m-s);
  background: var(--pill-fg);
}
:global(html.theme-pill-fill) .status-pill__dot {
  display: none;
}
</style>
