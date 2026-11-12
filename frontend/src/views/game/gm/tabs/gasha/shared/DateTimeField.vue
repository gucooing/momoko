<!-- 时间输入：值是 Unix 毫秒，按服务器时区显示与解释（到秒），右侧标出时区偏移。值为 0 时输入框为空。 -->
<template>
  <span class="dtf" :class="{ 'is-invalid': invalid, 'is-disabled': disabled }">
    <input
      type="datetime-local"
      step="1"
      class="dtf__input m-num"
      :value="inputValue"
      :disabled="disabled"
      :aria-label="label"
      @change="onChange"
    />
    <span class="dtf__zone">{{ offset }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { fromInputValue, offsetLabel, toInputValue } from './time'

const props = withDefaults(
  defineProps<{
    modelValue: number
    zone: string
    label?: string
    disabled?: boolean
    invalid?: boolean
  }>(),
  { label: undefined, disabled: false, invalid: false },
)
const emit = defineEmits<{ 'update:modelValue': [value: number] }>()

const inputValue = computed(() => (props.modelValue > 0 ? toInputValue(props.modelValue, props.zone) : ''))
const offset = computed(() => offsetLabel(props.zone, props.modelValue > 0 ? props.modelValue : Date.now()))

const onChange = (e: Event) => {
  const v = fromInputValue((e.target as HTMLInputElement).value, props.zone)
  if (v !== undefined) emit('update:modelValue', v)
}
</script>

<style scoped lang="scss">
.dtf {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 0 12px;
  border-radius: var(--app-radius);
  border: 1.5px solid var(--el-border-color-light);
  background: var(--el-bg-color);
  min-width: 0;
}
.dtf:focus-within {
  border-color: var(--m-p);
}
.dtf.is-invalid {
  border-color: var(--m-bad);
}
.dtf.is-disabled {
  opacity: 0.6;
}
.dtf__input {
  flex: 1;
  min-width: 0;
  border: 0;
  outline: 0;
  background: transparent;
  color: var(--el-text-color-primary);
  font: inherit;
  font-weight: 600;
}
.dtf__zone {
  flex: none;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}
</style>
