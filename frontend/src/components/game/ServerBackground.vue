<!-- 服务器的应用背景（图片或视频，见 @/games/serverMedia）：视频静音循环播放。
     hoverPlay 时视频只显示首帧，鼠标移上去才播放（列表卡片用，避免一屏几十个视频同时解码）。 -->
<template>
  <video
    v-if="video"
    ref="videoEl"
    :src="hoverPlay ? `${src}#t=0.1` : src"
    class="server-bg"
    muted
    loop
    playsinline
    :autoplay="!hoverPlay"
    :preload="hoverPlay ? 'metadata' : 'auto'"
    aria-hidden="true"
    @mouseenter="hoverPlay && play()"
    @mouseleave="hoverPlay && pause()"
  />
  <img v-else :src="src" :alt="alt" class="server-bg" :loading="eager ? 'eager' : 'lazy'" decoding="async" draggable="false" />
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { isVideoLink } from '@/games/serverMedia'

const props = withDefaults(defineProps<{ src: string; alt?: string; eager?: boolean; hoverPlay?: boolean }>(), {
  alt: '',
  eager: false,
  hoverPlay: false,
})

const video = computed(() => isVideoLink(props.src))
const videoEl = ref<HTMLVideoElement>()
const play = () => void videoEl.value?.play().catch(() => undefined)
const pause = () => videoEl.value?.pause()
</script>
