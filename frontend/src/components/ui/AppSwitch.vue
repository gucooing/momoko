<!-- 令牌开关（替代 el-switch）：track + thumb，开=主色。用于布尔开关（配置/状态）。 -->
<template>
  <button
    type="button"
    role="switch"
    class="app-switch"
    :class="{ 'is-on': modelValue, 'is-disabled': disabled }"
    :aria-checked="modelValue"
    :disabled="disabled"
    @click="toggle"
  >
    <span class="app-switch__thumb" />
  </button>
</template>

<script setup lang="ts">
const props = defineProps<{ modelValue: boolean; disabled?: boolean }>()
const emit = defineEmits<{ 'update:modelValue': [value: boolean] }>()
const toggle = () => {
  if (!props.disabled) emit('update:modelValue', !props.modelValue)
}
</script>

<style scoped lang="scss">
.app-switch {
  position: relative;
  width: 38px;
  height: 22px;
  flex-shrink: 0;
  padding: 0;
  border: none;
  border-radius: 999px;
  background: var(--el-border-color);
  cursor: pointer;
  transition: background 0.18s;
}
.app-switch.is-on {
  background: var(--el-color-primary);
}
.app-switch.is-disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.app-switch__thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  width: 18px;
  height: 18px;
  border-radius: 999px;
  background: #fff;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.2);
  transition: transform 0.18s;
}
.app-switch.is-on .app-switch__thumb {
  transform: translateX(16px);
}
</style>
