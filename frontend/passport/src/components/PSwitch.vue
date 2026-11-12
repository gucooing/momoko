<!-- 开关。 -->
<template>
  <button
    type="button"
    role="switch"
    class="ps"
    :class="{ 'is-on': modelValue }"
    :aria-checked="modelValue"
    :aria-label="label"
    :disabled="disabled"
    @click="emit('update:modelValue', !modelValue)"
  >
    <span class="ps__knob" />
  </button>
</template>

<script setup lang="ts">
defineProps<{ modelValue: boolean; disabled?: boolean; label?: string }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
</script>

<style scoped>
.ps {
  position: relative;
  flex: none;
  width: 48px;
  height: 28px;
  padding: 0;
  border: 0;
  border-radius: 999px;
  background: var(--m-line);
  cursor: pointer;
  transition: background 0.15s;
}
.ps::after {
  content: '';
  position: absolute;
  inset: -8px -2px;
}
.ps.is-on {
  background: var(--m-p);
}
.ps:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.ps__knob {
  position: absolute;
  top: 3px;
  left: 3px;
  width: 22px;
  height: 22px;
  border-radius: 50%;
  background: var(--m-s);
  transition: transform 0.15s;
}
.ps.is-on .ps__knob {
  transform: translateX(20px);
}
</style>
