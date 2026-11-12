<!-- 游戏账号头像：学マス 用出借记忆的偶像卡方图；其他游戏或没有时用昵称首字。 -->
<template>
  <span class="la" :style="{ width: `${size}px`, height: `${size}px` }" :data-game="game">
    <GameImage v-if="url" :src="url" :alt="name" class="la__img" />
    <span v-else class="la__letter" :style="{ fontSize: `${Math.round(size * 0.4)}px` }">{{ (name || '?').slice(0, 1).toUpperCase() }}</span>
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import GameImage from '@/components/game/GameImage.vue'
import { gakumas } from '@/games'

const props = withDefaults(defineProps<{ game: string; asset?: string; name?: string; size?: number }>(), {
  asset: '',
  name: '',
  size: 40,
})

const url = computed(() => (props.game === 'gakumas' && props.asset ? gakumas.idolCardImage(props.asset, 'sq') : ''))
</script>

<style scoped>
.la {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex: none;
  overflow: hidden;
  border-radius: 50%;
  background: var(--m-gSoft, var(--m-s2));
  color: var(--m-gInk, var(--m-ink));
}
.la__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
.la__letter {
  font-family: var(--app-font-display);
  font-weight: 800;
}
</style>
