<!-- 状态胶囊（带圆点）。 -->
<template>
  <span class="pp" :class="`pp--${tone}`">
    <span v-if="dot" class="pp__dot" aria-hidden="true" />
    <slot />
  </span>
</template>

<script setup lang="ts">
withDefaults(defineProps<{ tone?: 'ok' | 'warn' | 'bad' | 'off' | 'p' | 'g'; dot?: boolean }>(), { tone: 'off', dot: true })
</script>

<style scoped>
.pp {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 24px;
  padding: 0 10px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 800;
  white-space: nowrap;
  color: var(--pp-fg);
  background: var(--pp-bg);
}
.pp__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: currentColor;
}
.pp--ok {
  --pp-fg: var(--m-ok);
  --pp-bg: var(--m-okSoft);
}
.pp--warn {
  --pp-fg: var(--m-warn);
  --pp-bg: var(--m-warnSoft);
}
.pp--bad {
  --pp-fg: var(--m-bad);
  --pp-bg: var(--m-badSoft);
}
.pp--off {
  --pp-fg: var(--m-ink2);
  --pp-bg: var(--m-s2);
}
.pp--p {
  --pp-fg: var(--m-pInk);
  --pp-bg: var(--m-pSoft);
}
.pp--g {
  --pp-fg: var(--m-gInk);
  --pp-bg: var(--m-gSoft);
}
</style>
