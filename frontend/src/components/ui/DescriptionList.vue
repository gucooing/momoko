<!-- 描述列表：两列 term/description，靠字色分层而非加框（01 §2.3）。 -->
<template>
  <dl class="desc-list" :style="{ '--dl-cols': columns }">
    <div v-for="(it, i) in items" :key="i" class="desc-list__row">
      <dt>{{ it.label }}</dt>
      <dd :title="displayValue(it.value)">{{ displayValue(it.value) }}</dd>
    </div>
  </dl>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{ items: { label: string; value?: string | number | null }[]; columns?: number }>(),
  { columns: 2 },
)
const displayValue = (v?: string | number | null) =>
  v === null || v === undefined || v === '' ? '--' : String(v)
</script>

<style scoped lang="scss">
.desc-list {
  display: grid;
  grid-template-columns: 1fr;
  column-gap: 2rem;
  row-gap: 0.65rem;
}
@media (width >= 640px) {
  .desc-list {
    grid-template-columns: repeat(var(--dl-cols, 2), minmax(0, 1fr));
  }
}
.desc-list__row {
  display: flex;
  gap: 0.75rem;
  align-items: baseline;
  min-width: 0;
  font-size: 0.8125rem;
}
.desc-list__row dt {
  flex-shrink: 0;
  min-width: 5rem;
  color: var(--el-text-color-secondary);
  font-weight: 500;
}
.desc-list__row dd {
  min-width: 0;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-variant-numeric: tabular-nums;
}
</style>
