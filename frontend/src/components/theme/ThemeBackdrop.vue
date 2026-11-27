<!-- 自定义背景层：position: fixed 铺在整个界面下面（z-index -1），图片 / 渐变 / 图案 / 纯色 + 模糊 + 遮罩。
     深色模式下「自动加深遮罩」；开启「减少动效」时渐变不流动（全局动效开关会停掉 animation）。
     管理后台（主题 store）和用户通行证（全站默认主题）共用：背景、深浅色与资源解析由使用方传入。 -->
<template>
  <div v-if="visible" class="theme-backdrop" aria-hidden="true">
    <div class="theme-backdrop__layer" :class="{ 'is-blur': bg.blur > 0, 'is-flow': flow }" :style="layerStyle" />
    <div v-if="dim > 0" class="theme-backdrop__layer" :style="dimStyle" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { ThemeBackground } from '@/types/v1/theme'
import { backgroundCss, effectiveDim, type AssetResolver } from '@/theme/content'
import type { SchemeKey } from '@/theme/palettes'

defineOptions({ name: 'ThemeBackdrop' })

const props = defineProps<{ bg: ThemeBackground; scheme: SchemeKey; resolver?: AssetResolver; reduceMotion?: boolean }>()

const visible = computed(() => props.bg.type !== 'theme')
const flow = computed(() => props.bg.type === 'gradient' && props.bg.gradientFlow && !props.reduceMotion)
const dim = computed(() => effectiveDim(props.bg, props.scheme))

const layerStyle = computed(() => ({
  background: backgroundCss(props.bg, props.resolver),
  filter: props.bg.blur > 0 ? `blur(${props.bg.blur}px)` : undefined,
}))
const dimStyle = computed(() => ({
  background: `color-mix(in srgb, var(--m-bg) ${dim.value}%, transparent)`,
}))
</script>
