<!-- 列表工具条：状态胶囊（单选）+ 搜索框（输入即搜，防抖）+ 额外控件插槽。
     替代表单式 FilterBar：v-model:status / v-model:keyword，变化后触发 search。 -->
<template>
  <div class="ltb">
    <div v-if="chips?.length" class="ltb__chips" role="tablist" :aria-label="chipsLabel">
      <button
        v-for="opt in chips"
        :key="String(opt.value)"
        type="button"
        role="tab"
        class="ltb__chip"
        :class="{ 'is-on': status === opt.value }"
        :aria-selected="status === opt.value"
        @click="pick(opt.value)"
      >
        {{ opt.label }}
        <span v-if="opt.count != null" class="ltb__count m-num">{{ opt.count }}</span>
      </button>
    </div>
    <span class="ltb__spacer" />
    <label v-if="searchable" class="ltb__search">
      <component :is="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']" />
      <input
        :value="keyword"
        type="search"
        :placeholder="placeholder"
        :aria-label="placeholder"
        @input="onInput(($event.target as HTMLInputElement).value)"
        @keyup.enter="emit('search')"
      />
    </label>
    <slot />
  </div>
</template>

<script setup lang="ts" generic="T extends string | number">
import { useDebounceFn } from '@vueuse/core'

export interface ToolbarChip<V> {
  label: string
  value: V
  count?: number | null
}

withDefaults(
  defineProps<{
    chips?: ToolbarChip<T>[]
    chipsLabel?: string
    placeholder?: string
    searchable?: boolean
  }>(),
  { searchable: true },
)
const status = defineModel<T>('status')
const keyword = defineModel<string>('keyword', { default: '' })
const emit = defineEmits<{ search: [] }>()
const menuStore = useMenuStore()

const debouncedSearch = useDebounceFn(() => emit('search'), 350)
const onInput = (value: string) => {
  keyword.value = value
  debouncedSearch()
}
const pick = (value: T) => {
  status.value = value
  emit('search')
}
</script>

<style scoped lang="scss">
.ltb {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
}
.ltb__spacer {
  flex: 1;
}
.ltb__chips {
  display: flex;
  gap: 8px;
  max-width: 100%;
  overflow-x: auto;
  scrollbar-width: none;
}
.ltb__chips::-webkit-scrollbar {
  display: none;
}
.ltb__chip {
  flex: none;
  display: inline-flex;
  align-items: center;
  gap: 6px;
  min-height: 40px;
  padding: 0 16px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition: border-color 0.15s, color 0.15s, background 0.15s;
}
.ltb__chip:hover {
  border-color: var(--m-p);
  color: var(--m-pInk);
}
.ltb__chip.is-on {
  background: var(--m-ink);
  border-color: var(--m-ink);
  color: var(--m-bg);
}
.ltb__count {
  font-size: 0.75rem;
  opacity: 0.75;
}
.ltb__search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 0 1 280px;
  min-width: 180px;
  min-height: 40px;
  padding: 0 14px;
  border-radius: 999px;
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  color: var(--el-text-color-secondary);
  transition: border-color 0.15s;
}
.ltb__search:focus-within {
  border-color: var(--m-p);
}
.ltb__search :deep(svg) {
  width: 16px;
  height: 16px;
  flex: none;
}
.ltb__search input {
  flex: 1;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-size: 0.875rem;
}
@media (width < 768px) {
  .ltb__spacer {
    display: none;
  }
  .ltb__chips {
    order: 3;
    width: 100%;
  }
  .ltb__search {
    flex: 1 1 0;
    min-width: 0;
  }
}
</style>
