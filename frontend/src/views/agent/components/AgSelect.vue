<!-- Agent 页面的紧凑下拉：表单字段样子的触发器 + 菜单。选项可以带分组标题（group）。 -->
<template>
  <AppDropdown align="start" block :panel-class="'ag ag-menu ag-select__panel'">
    <template #trigger>
      <button type="button" class="ag-fld" :class="{ 'ag-fld--plain': plain }" :disabled="disabled" :aria-label="label">
        <slot name="prefix" />
        <span class="ag-select__v ag-ellipsis" :class="{ 'ag-mono': mono, 'ag-muted': !current }">
          {{ current?.label ?? placeholder }}
        </span>
        <UIcon name="i-lucide-chevron-down" />
      </button>
    </template>
    <template #default="{ close }">
      <template v-for="(opt, idx) in options" :key="`${opt.group || ''}/${String(opt.value)}`">
        <div v-if="opt.group && opt.group !== options[idx - 1]?.group" class="ag-mi-label">{{ opt.group }}</div>
        <button
          type="button"
          class="ag-mi"
          role="option"
          :class="{ 'is-on': opt.value === modelValue }"
          :aria-selected="opt.value === modelValue"
          :disabled="opt.disabled"
          @click="pick(opt.value, close)"
        >
          <span class="ag-ellipsis" :class="{ 'ag-mono': mono }" style="flex: 1">{{ opt.label }}</span>
          <span v-if="opt.hint" class="ag-muted" style="font-size: 12px">{{ opt.hint }}</span>
          <UIcon v-if="opt.value === modelValue" name="i-lucide-check" style="color: var(--m-pInk)" />
        </button>
      </template>
      <div v-if="!options.length" class="ag-mi-label">{{ emptyText }}</div>
    </template>
  </AppDropdown>
</template>

<script setup lang="ts" generic="T extends string | number | boolean">
export interface AgSelectOption<V> {
  value: V
  label: string
  /** 分组标题：相邻的同组选项只显示一次 */
  group?: string
  hint?: string
  disabled?: boolean
}

const props = withDefaults(
  defineProps<{
    modelValue: T
    options: AgSelectOption<T>[]
    placeholder?: string
    label?: string
    mono?: boolean
    plain?: boolean
    disabled?: boolean
    emptyText?: string
  }>(),
  { placeholder: '', label: '', emptyText: '—' },
)
const emit = defineEmits<{ 'update:modelValue': [value: T] }>()

const current = computed(() => props.options.find((o) => o.value === props.modelValue))
const pick = (value: T, close: () => void) => {
  emit('update:modelValue', value)
  close()
}
</script>

<style lang="scss">
.ag-select__v {
  flex: 1;
  font-size: 13px;
  color: var(--m-ink);
}
.ag-select__v.ag-mono {
  font-size: 12.5px;
}
// 选项多时在浮层里滚动。AppDropdown 的浮层自带 overflow: hidden（带 scoped 属性，优先级更高），
// 这里要用更高的优先级盖过去，不然超出高度的选项被裁掉、选不到；滚动条要看得见，才知道下面还有
.ag.ag-menu.ag-select__panel {
  max-height: min(360px, 60vh);
  overflow-y: auto;
  scrollbar-width: thin;
}
</style>
