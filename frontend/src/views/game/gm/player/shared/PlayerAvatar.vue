<!-- 玩家头像：出借记忆的偶像卡方图（圆形，按需加载）；没有时显示昵称首字。可带在线圆点。 -->
<template>
  <span class="pav" :style="{ width: `${size}px`, height: `${size}px` }">
    <GameImage v-if="src" :src="src" :alt="name" class="pav__img" :width="size" :height="size" />
    <span v-else class="pav__initial" :style="{ fontSize: `${Math.round(size * 0.4)}px` }">{{ initial }}</span>
    <span v-if="online" class="pav__dot" aria-hidden="true" />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useGmContext } from '../../context'
import { catalogImage } from './game'

const props = withDefaults(defineProps<{ idolCardId?: string; name?: string; size?: number; online?: boolean }>(), {
  idolCardId: '',
  name: '',
  size: 40,
  online: false,
})

const ctx = useGmContext()
const src = computed(() => (props.idolCardId ? catalogImage(ctx.game.value, 'idolCardSkin', { id: props.idolCardId }) : ''))
const initial = computed(() => (props.name || '?').trim().slice(0, 1).toUpperCase())
</script>

<style scoped>
.pav {
  position: relative;
  flex: none;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  background: var(--m-gSoft, var(--m-pSoft));
  color: var(--m-gInk, var(--m-pInk));
}
.pav__img {
  width: 100%;
  height: 100%;
  border-radius: 50%;
}
.pav__initial {
  font-family: var(--app-font-display);
  font-weight: 800;
}
.pav__dot {
  position: absolute;
  right: 0;
  bottom: 0;
  width: 28%;
  height: 28%;
  min-width: 10px;
  min-height: 10px;
  border-radius: 50%;
  background: var(--m-ok);
  border: 2px solid var(--m-s);
}
</style>
