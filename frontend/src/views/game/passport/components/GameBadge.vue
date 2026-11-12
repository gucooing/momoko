<!-- 游戏标记：游戏名称的首字，配色取这款游戏的游戏皮肤（没有皮肤的游戏用主色）。
     manifest 的图标只是资源名，素材包里没有对应的图，所以不用图片。 -->
<template>
  <span class="gbadge" :data-game="game || undefined" :style="{ '--gb-size': size + 'px' }" :title="label" :aria-label="label" role="img">
    {{ letter }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useGameName } from '../shared'

const props = withDefaults(defineProps<{ game: string; name?: string; size?: number }>(), { name: '', size: 26 })

const gameName = useGameName()
const label = computed(() => gameName(props.game, props.name))
const letter = computed(() => Array.from(label.value.trim())[0]?.toUpperCase() || '?')
</script>

<style scoped>
.gbadge {
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: var(--gb-size);
  height: var(--gb-size);
  border-radius: calc(var(--gb-size) * 0.32);
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
  border: 2px solid var(--m-s);
  font-family: var(--app-font-display);
  font-size: calc(var(--gb-size) * 0.46);
  font-weight: 800;
  line-height: 1;
}
</style>
