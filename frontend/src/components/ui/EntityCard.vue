<!-- 实体卡：hairline 静态卡；可点时 hover 抬升(-2px)+边框转主色淡+轻阴影（01 §5）。 -->
<template>
  <component
    :is="clickable ? 'button' : 'div'"
    class="entity-card"
    :class="{ 'entity-card--clickable': clickable }"
    :type="clickable ? 'button' : undefined"
    @click="clickable ? $emit('click') : undefined"
  >
    <div class="entity-card__head">
      <div class="entity-card__title"><slot name="title">{{ title }}</slot></div>
      <div v-if="$slots.status" class="entity-card__status"><slot name="status" /></div>
    </div>
    <div v-if="$slots.meta" class="entity-card__meta"><slot name="meta" /></div>
    <div v-if="$slots.footer" class="entity-card__foot"><slot name="footer" /></div>
  </component>
</template>

<script setup lang="ts">
defineProps<{ title?: string; clickable?: boolean }>()
defineEmits<{ click: [] }>()
</script>

<style scoped lang="scss">
.entity-card {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
  width: 100%;
  text-align: left;
  padding: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-lg);
  color: inherit;
  font: inherit;
}
.entity-card--clickable {
  cursor: pointer;
  transition:
    transform 0.18s cubic-bezier(0.4, 0, 0.2, 1),
    border-color 0.18s,
    box-shadow 0.18s;
}
.entity-card--clickable:hover {
  transform: translateY(-2px);
  border-color: color-mix(in srgb, var(--el-color-primary) 40%, var(--el-border-color));
  box-shadow: var(--app-shadow-md);
}
.entity-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
}
.entity-card__title {
  min-width: 0;
  font-size: 0.9375rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.entity-card__status {
  flex-shrink: 0;
}
.entity-card__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 0.4rem 0.75rem;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.entity-card__foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  padding-top: 0.75rem;
  border-top: 1px solid var(--el-border-color-lighter);
  font-size: 0.75rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
}
</style>
