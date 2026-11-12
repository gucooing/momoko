<!-- 游戏素材图片：src 用 gameAsset() / gakumas.*Image() 得到的 URL。
     默认 loading="lazy"（只有滚动到可见区域才请求），解码异步；加载失败换成占位图，不出现破图。 -->
<template>
  <img
    :src="current"
    :alt="alt"
    :width="width"
    :height="height"
    :loading="eager ? 'eager' : 'lazy'"
    decoding="async"
    draggable="false"
    class="game-image"
    @error="onError"
  />
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { gamePlaceholder } from '@/games'

const props = withDefaults(
  defineProps<{
    src: string
    alt?: string
    width?: number | string
    height?: number | string
    /** 首屏关键图（如横幅）可以立即加载 */
    eager?: boolean
  }>(),
  { alt: '', width: undefined, height: undefined, eager: false },
)

const failed = ref(false)
watch(
  () => props.src,
  () => {
    failed.value = false
  },
)

const current = computed(() => (failed.value || !props.src ? gamePlaceholder : props.src))

const onError = () => {
  failed.value = true
}
</script>

<style scoped>
.game-image {
  display: block;
  object-fit: cover;
  background: var(--el-fill-color-light);
}
</style>
