<!-- 游戏标记：素材包里没有游戏图标，用游戏名首字在游戏皮肤色上做一个贴纸（data-game 让 --m-g* 生效）。 -->
<template>
  <span
    class="gm"
    :data-game="game"
    :style="{ width: `${size}px`, height: `${size}px`, fontSize: `${Math.round(size * 0.4)}px` }"
    :title="name || game"
    role="img"
    :aria-label="name || game"
  >
    {{ letter }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = withDefaults(defineProps<{ game: string; name?: string; size?: number; tilt?: boolean }>(), {
  name: '',
  size: 40,
})

const letter = computed(() => (props.name || props.game || '?').trim().slice(0, 1).toUpperCase())
</script>

<style scoped>
.gm {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  border-radius: 30%;
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  border: 1.5px solid var(--m-gLine, var(--m-line));
  font-family: var(--app-font-display);
  font-weight: 800;
  line-height: 1;
}
</style>
