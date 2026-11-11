<!-- 自定义背景层：position: fixed 铺在整个界面下面（z-index -1），图片 / 渐变 / 图案 / 纯色 + 模糊 + 遮罩。
     深色模式下「自动加深遮罩」；开启「减少动效」时渐变不流动（全局动效开关会停掉 animation）。 -->
<template>
  <div v-if="visible" class="theme-backdrop" aria-hidden="true">
    <div class="theme-backdrop__layer" :class="{ 'is-blur': bg.blur > 0, 'is-flow': flow }" :style="layerStyle" />
    <div v-if="dim > 0" class="theme-backdrop__layer" :style="dimStyle" />
  </div>
</template>

<script setup lang="ts">
import { backgroundCss, effectiveDim } from '@/theme/content'

defineOptions({ name: 'ThemeBackdrop' })

const themeStore = useThemeStore()

const bg = computed(() => themeStore.background.bg)
const visible = computed(() => bg.value.type !== 'theme')
const flow = computed(() => bg.value.type === 'gradient' && bg.value.gradientFlow && !themeStore.reduceMotion)
const dim = computed(() => effectiveDim(bg.value, themeStore.scheme))

const layerStyle = computed(() => ({
  background: backgroundCss(bg.value, themeStore.activeResolver),
  filter: bg.value.blur > 0 ? `blur(${bg.value.blur}px)` : undefined,
}))
const dimStyle = computed(() => ({
  background: `color-mix(in srgb, var(--m-bg) ${dim.value}%, transparent)`,
}))
</script>
