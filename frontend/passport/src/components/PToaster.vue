<!-- 轻提示。 -->
<template>
  <div class="pt" aria-live="polite">
    <TransitionGroup name="pt">
      <div v-for="x in ui.toasts" :key="x.id" class="pt__item" :class="`pt__item--${x.tone}`" role="status">
        {{ x.text }}
      </div>
    </TransitionGroup>
  </div>
</template>

<script setup lang="ts">
import { useUiStore } from '@passport/stores/ui'

const ui = useUiStore()
</script>

<style scoped>
.pt {
  position: fixed;
  top: 16px;
  left: 50%;
  z-index: 2000;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  width: min(420px, calc(100vw - 32px));
  transform: translateX(-50%);
  pointer-events: none;
}
.pt__item {
  padding: 12px 18px;
  border-radius: 16px;
  background: var(--m-ink);
  color: var(--m-s);
  font-size: 14px;
  font-weight: 700;
  box-shadow: var(--app-shadow-lg);
}
.pt__item--ok {
  background: var(--m-ok);
  color: var(--m-s);
}
.pt__item--bad {
  background: var(--m-bad);
  color: var(--m-s);
}
.pt-enter-active,
.pt-leave-active {
  transition:
    opacity 0.2s,
    transform 0.2s;
}
.pt-enter-from,
.pt-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
