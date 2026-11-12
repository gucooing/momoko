<!-- 头像：上传的图片 > 游戏素材头像（<game>/<素材 key>）> 名字首字母贴纸。 -->
<template>
  <span class="pa" :class="[`pa--${shape}`, { 'pa--tilt': tilt }]" :style="{ width: `${size}px`, height: `${size}px` }">
    <img v-if="src && !failed" :src="src" :alt="name" class="pa__img" @error="failed = true" />
    <GameImage v-else-if="assetUrl" :src="assetUrl" :alt="name" class="pa__img" />
    <span v-else class="pa__letter" :style="{ fontSize: `${Math.round(size * 0.42)}px` }">{{ letter }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import GameImage from '@/components/game/GameImage.vue'
import { gameAsset } from '@/games'

const props = withDefaults(
  defineProps<{ src?: string; asset?: string; name?: string; size?: number; shape?: 'circle' | 'rounded'; tilt?: boolean }>(),
  { src: '', asset: '', name: '', size: 40, shape: 'circle' },
)

const failed = ref(false)
watch(
  () => props.src,
  () => {
    failed.value = false
  },
)

const assetUrl = computed(() => {
  const i = props.asset.indexOf('/')
  if (i <= 0) return ''
  return gameAsset(props.asset.slice(0, i), props.asset.slice(i + 1))
})

const letter = computed(() => (props.name.trim()[0] || '?').toUpperCase())
</script>

<style scoped>
.pa {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  overflow: hidden;
  background: var(--m-p);
  color: var(--m-pOn);
}
.pa--circle {
  border-radius: 50%;
}
.pa--rounded {
  border-radius: 30%;
}
.pa--tilt {
  transform: rotate(-5deg);
  box-shadow: 4px 4px 0 var(--m-a);
}
.pa__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.pa__letter {
  font-family: var(--app-font-display);
  font-weight: 800;
  line-height: 1;
}
</style>
