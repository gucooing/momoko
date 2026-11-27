<!-- 游戏标记：服务器上传了应用图标（src，GameServerBrief.appIcon）就显示图标，没有时用游戏图标（gameIcon）；
     游戏也没有图标（或图片取不到）时用游戏名首字在游戏皮肤色上做一个贴纸（data-game 让 --m-g* 生效）。 -->
<template>
  <span
    class="gm"
    :class="{ 'gm--img': shown }"
    :data-game="game"
    :style="{ width: `${size}px`, height: `${size}px`, fontSize: `${Math.round(size * 0.4)}px` }"
    :title="name || game"
    role="img"
    :aria-label="name || game"
  >
    <img v-if="shown" :src="url" alt="" loading="lazy" class="gm__img" @error="failed = url" />
    <template v-else>{{ letter }}</template>
  </span>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { gameIcon } from '@/games'

const props = withDefaults(defineProps<{ game: string; name?: string; size?: number; tilt?: boolean; src?: string }>(), {
  name: '',
  src: '',
  size: 40,
})

const url = computed(() => props.src || gameIcon(props.game))
// 图片取不到时退回首字（记下失败的地址，换了地址再试）
const failed = ref('')
const shown = computed(() => !!url.value && url.value !== failed.value)
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
  font-weight: 600;
  line-height: 1;
}
.gm--img {
  overflow: hidden;
  background: var(--m-s);
}
.gm__img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}
</style>
