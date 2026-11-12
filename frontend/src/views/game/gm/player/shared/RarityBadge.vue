<!-- 稀有度角标（R / SR / SSR）。 -->
<template>
  <span v-if="label" class="rbadge m-num" :class="`is-r${rarity}`">{{ label }}</span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { rarityLabel } from './game'

const props = defineProps<{ rarity?: number }>()
const label = computed(() => rarityLabel(props.rarity))
</script>

<style scoped>
.rbadge {
  display: inline-flex;
  align-items: center;
  height: 20px;
  padding: 0 7px;
  border-radius: 7px;
  background: var(--m-s2);
  color: var(--m-ink);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0.02em;
  box-shadow: 0 0 0 1.5px var(--m-s);
}
.rbadge.is-r3 {
  background: var(--m-a);
  color: var(--m-aOn);
}
.rbadge.is-r2 {
  background: var(--m-a2Soft);
  color: var(--m-ink);
}
</style>
