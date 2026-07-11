<!-- 头像：圆形 + 可选在线圆点。图片失败时回退到首字母。 -->
<template>
  <span class="app-avatar" :style="{ width: size + 'px', height: size + 'px' }">
    <img v-if="src && !broken" :src="src" :alt="alt" @error="broken = true" />
    <span v-else class="app-avatar__fallback">{{ initials }}</span>
    <span v-if="online" class="app-avatar__dot" :style="dotStyle" />
  </span>
</template>

<script setup lang="ts">
const props = withDefaults(
  defineProps<{ src?: string; size?: number; alt?: string; name?: string; online?: boolean }>(),
  { size: 40 },
)
const broken = ref(false)
watch(
  () => props.src,
  () => (broken.value = false),
)
const initials = computed(() => (props.name || props.alt || '?').trim().charAt(0).toUpperCase())
const dotStyle = computed(() => {
  const d = Math.max(8, Math.round(props.size * 0.26))
  return { width: d + 'px', height: d + 'px' }
})
</script>

<style scoped lang="scss">
.app-avatar {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  border-radius: 999px;
  overflow: visible;
  background: var(--el-fill-color-light);
}
.app-avatar img {
  width: 100%;
  height: 100%;
  border-radius: 999px;
  object-fit: cover;
}
.app-avatar__fallback {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color);
}
.app-avatar__dot {
  position: absolute;
  right: -1px;
  bottom: -1px;
  border-radius: 999px;
  background: var(--el-color-success, #16a34a);
  border: 2px solid var(--el-bg-color);
}
</style>
