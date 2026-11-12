<!-- 通行证按钮：primary 主色实底 / soft 淡底 / outline 描边 / ghost 透明 / danger 停止色 / ink 深色。 -->
<template>
  <component
    :is="to ? RouterLink : 'button'"
    :to="to"
    :type="to ? undefined : type"
    class="pb"
    :class="[`pb--${variant}`, `pb--${size}`, { 'pb--block': block, 'is-loading': loading }]"
    :disabled="to ? undefined : disabled || loading"
    :aria-busy="loading || undefined"
  >
    <span v-if="loading" class="pb__spin" aria-hidden="true" />
    <component :is="icon" v-else-if="icon" class="pb__icon" aria-hidden="true" />
    <slot />
  </component>
</template>

<script setup lang="ts">
import type { Component } from 'vue'
import { RouterLink, type RouteLocationRaw } from 'vue-router'

withDefaults(
  defineProps<{
    variant?: 'primary' | 'soft' | 'outline' | 'ghost' | 'danger' | 'ink'
    size?: 'sm' | 'md' | 'lg'
    type?: 'button' | 'submit'
    icon?: Component
    loading?: boolean
    disabled?: boolean
    block?: boolean
    to?: RouteLocationRaw
  }>(),
  { variant: 'outline', size: 'md', type: 'button', icon: undefined, to: undefined },
)
</script>

<style scoped>
.pb {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 44px;
  padding: 0 18px;
  border-radius: var(--app-radius-pill, 999px);
  border: 1.5px solid transparent;
  font: inherit;
  font-size: 14px;
  font-weight: 800;
  text-decoration: none;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background 0.15s,
    border-color 0.15s,
    opacity 0.15s;
}
.pb--sm {
  min-height: 40px;
  padding: 0 14px;
  font-size: 13px;
}
.pb--lg {
  min-height: 52px;
  font-size: 16px;
}
.pb--block {
  width: 100%;
}
.pb:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
.pb--primary {
  background: var(--m-p);
  color: var(--m-pOn);
  box-shadow: var(--app-shadow-brand);
}
.pb--primary:hover:not(:disabled) {
  background: color-mix(in srgb, var(--m-p) 88%, var(--m-ink));
}
.pb--soft {
  background: var(--m-pSoft);
  color: var(--m-pInk);
}
.pb--outline {
  background: var(--m-s);
  border-color: var(--m-line);
  color: var(--m-ink);
}
.pb--outline:hover:not(:disabled),
.pb--ghost:hover:not(:disabled) {
  background: var(--m-s2);
}
.pb--ghost {
  background: transparent;
  color: var(--m-ink2);
}
.pb--danger {
  background: var(--m-badSoft);
  color: var(--m-bad);
}
.pb--danger:hover:not(:disabled) {
  border-color: var(--m-bad);
}
.pb--ink {
  background: var(--m-ink);
  color: var(--m-s);
}
.pb__icon {
  width: 18px;
  height: 18px;
  flex: none;
}
.pb__spin {
  width: 16px;
  height: 16px;
  border-radius: 50%;
  border: 2px solid currentColor;
  border-right-color: transparent;
  animation: pb-spin 0.8s linear infinite;
}
@keyframes pb-spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
