<!-- 通行证头像：上传的图片 → 游戏素材头像 → 首字母贴纸。 -->
<template>
  <span class="ppav" :style="{ '--pa-size': size + 'px' }">
    <GameImage v-if="src" :src="src" :alt="name" class="ppav__img" />
    <span v-else class="ppav__letter" aria-hidden="true">{{ letter }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import GameImage from '@/components/game/GameImage.vue'
import { passportAvatarSrc } from '../shared'

const props = withDefaults(
  defineProps<{ avatar?: string; avatarAsset?: string; name?: string; size?: number }>(),
  { avatar: '', avatarAsset: '', name: '', size: 40 },
)

const src = computed(() => passportAvatarSrc(props.avatar, props.avatarAsset))
const letter = computed(() => Array.from(props.name.trim())[0]?.toUpperCase() || '?')
</script>

<style scoped>
.ppav {
  flex: none;
  display: inline-flex;
  width: var(--pa-size);
  height: var(--pa-size);
  border-radius: 50%;
  overflow: hidden;
  background: var(--m-pSoft);
}
.ppav__img {
  width: 100%;
  height: 100%;
}
.ppav__letter {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
  background: var(--m-p);
  color: var(--m-pOn);
  font-family: var(--app-font-display);
  font-size: calc(var(--pa-size) * 0.42);
  font-weight: 800;
}
</style>
