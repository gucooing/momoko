<!-- 分页：总数 + 每页 + 页码（省略号）。移动端简化为 上一页/页码/下一页（03/M-1）。令牌驱动。 -->
<template>
  <div class="pagination">
    <span class="pagination__total">{{ t('system.common.total', { total }) }}</span>

    <div class="pagination__spacer" />

    <select
      v-if="!menuStore.isMobile"
      class="app-select pagination__size"
      :value="pageSize"
      @change="onSizeChange"
    >
      <option v-for="s in pageSizes" :key="s" :value="s">{{ t('system.common.perPage', { size: s }) }}</option>
    </select>

    <nav class="pagination__pages" aria-label="pagination">
      <button
        type="button"
        class="pagination__btn"
        :disabled="page <= 1"
        @click="go(page - 1)"
      >
        <component :is="menuStore.iconComponents['HOutline:ChevronLeftIcon']" />
      </button>

      <template v-if="!menuStore.isMobile">
        <button
          v-for="(item, i) in pageItems"
          :key="i"
          type="button"
          class="pagination__btn"
          :class="{ 'is-active': item === page, 'is-ellipsis': item === '...' }"
          :disabled="item === '...'"
          @click="typeof item === 'number' && go(item)"
        >
          {{ item }}
        </button>
      </template>
      <span v-else class="pagination__current">{{ page }} / {{ pageCount }}</span>

      <button
        type="button"
        class="pagination__btn"
        :disabled="page >= pageCount"
        @click="go(page + 1)"
      >
        <component :is="menuStore.iconComponents['HOutline:ChevronRightIcon']" />
      </button>
    </nav>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'AppPagination' })

const props = withDefaults(
  defineProps<{ page: number; pageSize: number; total: number; pageSizes?: number[] }>(),
  { pageSizes: () => [10, 20, 50, 100] },
)
const emit = defineEmits<{
  'update:page': [value: number]
  'update:pageSize': [value: number]
  change: []
}>()

const menuStore = useMenuStore()
const { t } = useI18n()

const pageCount = computed(() => Math.max(1, Math.ceil(props.total / props.pageSize)))

const pageItems = computed<(number | '...')[]>(() => {
  const count = pageCount.value
  const cur = props.page
  if (count <= 7) return Array.from({ length: count }, (_, i) => i + 1)
  const items: (number | '...')[] = [1]
  const left = Math.max(2, cur - 1)
  const right = Math.min(count - 1, cur + 1)
  if (left > 2) items.push('...')
  for (let i = left; i <= right; i++) items.push(i)
  if (right < count - 1) items.push('...')
  items.push(count)
  return items
})

const go = (target: number) => {
  const next = Math.min(Math.max(1, target), pageCount.value)
  if (next === props.page) return
  emit('update:page', next)
  emit('change')
}

const onSizeChange = (e: Event) => {
  const size = Number((e.target as HTMLSelectElement).value)
  emit('update:pageSize', size)
  emit('update:page', 1)
  emit('change')
}
</script>

<style scoped lang="scss">
.pagination {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-wrap: wrap;
}
.pagination__total {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.pagination__spacer {
  flex: 1;
}
.pagination__size {
  width: auto;
  height: 32px;
  padding: 0 30px 0 10px;
  font-size: 0.8125rem;
}
.pagination__pages {
  display: flex;
  align-items: center;
  gap: 4px;
}
.pagination__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px;
  padding: 0 6px;
  border: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  font-variant-numeric: tabular-nums;
  cursor: pointer;
  transition: background 0.15s, border-color 0.15s, color 0.15s;
}
.pagination__btn :deep(svg) {
  width: 16px;
  height: 16px;
}
.pagination__btn:hover:not(:disabled):not(.is-ellipsis) {
  border-color: color-mix(in srgb, var(--el-color-primary) 40%, var(--el-border-color));
  color: var(--el-color-primary);
}
.pagination__btn.is-active {
  background: var(--el-color-primary);
  border-color: var(--el-color-primary);
  color: #fff;
}
.pagination__btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.pagination__btn.is-ellipsis {
  border: none;
  background: transparent;
  cursor: default;
  opacity: 1;
}
.pagination__current {
  padding: 0 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  font-variant-numeric: tabular-nums;
}
</style>
