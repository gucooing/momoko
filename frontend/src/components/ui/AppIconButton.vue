<!-- 图标按钮：安静方形，quiet；hover 淡填充；可选角标。用于顶栏动作/触发器。
     命名 AppIconButton 以避开项目已存在的 components/button/IconButton.vue。 -->
<template>
  <button
    type="button"
    class="app-icon-btn"
    :class="{ 'app-icon-btn--active': active }"
    :style="{ width: box + 'px', height: box + 'px' }"
    :aria-label="label"
    :title="label"
  >
    <component
      :is="menuStore.iconComponents[icon]"
      v-if="icon"
      class="app-icon-btn__icon"
      :style="{ width: size + 'px', height: size + 'px' }"
    />
    <slot />
    <span
      v-if="badge !== undefined && badge !== null && badge !== ''"
      class="app-icon-btn__badge"
    >
      {{ badge }}
    </span>
  </button>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{
    icon?: string
    label?: string
    /** 图标尺寸（px） */
    size?: number
    /** 按钮命中盒尺寸（px），行内/卡片动作可传更紧凑值 */
    box?: number
    active?: boolean
    badge?: number | string
  }>(),
  { size: 18, box: 36 },
)
const menuStore = useMenuStore()
</script>

<style scoped lang="scss">
.app-icon-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-secondary);
  background: transparent;
  border: none;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.app-icon-btn:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.app-icon-btn--active {
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
}
.app-icon-btn__icon {
  display: block;
}
.app-icon-btn__badge {
  position: absolute;
  top: 2px;
  right: 2px;
  min-width: 16px;
  height: 16px;
  padding: 0 4px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 999px;
  background: var(--el-color-danger, #ef4444);
  color: #fff;
  font-size: 0.625rem;
  font-weight: 600;
  line-height: 1;
  font-variant-numeric: tabular-nums;
  border: 2px solid var(--el-bg-color);
}
</style>
