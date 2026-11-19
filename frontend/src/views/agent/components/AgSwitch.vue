<!-- Agent 页面的紧凑开关（34×20）。 -->
<template>
  <button
    type="button"
    class="ag-tg"
    role="switch"
    :class="{ 'is-on': modelValue }"
    :aria-checked="modelValue"
    :aria-label="label"
    :disabled="disabled"
    @click.stop="emit('update:modelValue', !modelValue)"
  />
</template>

<script setup lang="ts">
defineProps<{ modelValue: boolean; label?: string; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>

<style lang="scss">
.ag-tg {
  position: relative;
  width: 34px;
  height: 20px;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: var(--m-line);
  cursor: pointer;
  flex: none;
  transition: background 0.15s;
}
.ag-tg::after {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #fff;
  box-shadow: 0 1px 3px rgba(20, 10, 30, 0.25);
  transition: transform 0.15s;
}
.ag-tg.is-on {
  background: var(--m-p);
}
.ag-tg.is-on::after {
  transform: translateX(14px);
}
.ag-tg:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.ag-tg:focus-visible {
  outline: 3px solid var(--m-a2);
  outline-offset: 2px;
}
</style>
