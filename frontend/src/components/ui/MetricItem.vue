<!-- 单个指标：可选单色 outline 图标 + 大写小标签 + tabular 大数 + 可选细进度线 + caption。
     单色，禁彩色图标盒（01 §8）。icon 走 menuStore Heroicons（HOutline:*）。 -->
<template>
  <div class="metric">
    <div class="metric__head">
      <component
        :is="menuStore.iconComponents[icon]"
        v-if="icon"
        class="metric__icon"
        aria-hidden="true"
      />
      <div class="metric__label">{{ label }}</div>
    </div>
    <div class="metric__value"><slot name="value">{{ value }}</slot></div>
    <div v-if="percent != null" class="metric__bar" role="progressbar">
      <span :style="{ width: clamped + '%' }" />
    </div>
    <div v-if="caption || $slots.caption" class="metric__caption">
      <slot name="caption">{{ caption }}</slot>
    </div>
  </div>
</template>

<script setup lang="ts">
const props = defineProps<{
  label?: string
  value?: string | number
  percent?: number | null
  caption?: string
  /** Heroicons key，如 HOutline:BoltIcon；单色继承，无彩色底 */
  icon?: string
}>()
const menuStore = useMenuStore()
const clamped = computed(() => Math.min(100, Math.max(0, Number(props.percent) || 0)))
</script>

<style scoped lang="scss">
.metric {
  background: var(--el-bg-color);
  padding: 16px 18px;
  display: flex;
  flex-direction: column;
  gap: 0.4rem;
  min-width: 0;
}
.metric__head {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
}
.metric__icon {
  flex: none;
  width: 14px;
  height: 14px;
  color: var(--el-text-color-secondary);
}
.metric__label {
  font-size: 0.6875rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--el-text-color-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.metric__value {
  font-size: 1.5rem;
  font-weight: 700;
  line-height: 1.1;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
  font-variant-numeric: tabular-nums;
}
.metric__bar {
  height: 4px;
  border-radius: 999px;
  background: var(--el-fill-color);
  overflow: hidden;
  margin-top: 0.15rem;
}
.metric__bar > span {
  display: block;
  height: 100%;
  border-radius: 999px;
  background: var(--el-color-primary);
  transition: width 0.5s cubic-bezier(0.4, 0, 0.2, 1);
}
.metric__caption {
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
  line-height: 1.35;
}
</style>
