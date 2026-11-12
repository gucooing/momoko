<!-- 表单项：标签 + 输入 + 提示 / 错误。 -->
<template>
  <label class="pf">
    <span v-if="label" class="pf__label">{{ label }}</span>
    <slot />
    <span v-if="error" class="pf__error">{{ error }}</span>
    <span v-else-if="hint" class="pf__hint">{{ hint }}</span>
  </label>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ label?: string; hint?: string; error?: string }>(), { label: '', hint: '', error: '' })
</script>

<style scoped>
.pf {
  display: flex;
  flex-direction: column;
  gap: 6px;
  min-width: 0;
}
.pf__label {
  font-size: 13px;
  font-weight: 700;
}
.pf__hint {
  font-size: 12px;
  line-height: 1.5;
  color: var(--m-ink2);
}
.pf__error {
  font-size: 12px;
  color: var(--m-bad);
}
.pf :deep(input:not([type='checkbox'])),
.pf :deep(select),
.pf :deep(textarea) {
  width: 100%;
  min-height: 44px;
  padding: 0 14px;
  border-radius: var(--app-radius-sm, 12px);
  border: 1.5px solid var(--m-line);
  background: var(--m-s);
  color: var(--m-ink);
  font: inherit;
  font-size: 14px;
  outline: none;
}
.pf :deep(textarea) {
  min-height: 88px;
  padding: 10px 14px;
  resize: vertical;
}
.pf :deep(input:focus),
.pf :deep(select:focus),
.pf :deep(textarea:focus) {
  border-color: var(--m-p);
}
</style>
