<!-- 公开页分段切换：浅底胶囊 + 白色选中块（登录方式、注册 / 找回密码）。 -->
<template>
  <div class="aseg" role="tablist" :aria-label="label">
    <button
      v-for="o in options"
      :key="o.value"
      type="button"
      role="tab"
      class="aseg__item"
      :class="{ 'is-on': model === o.value }"
      :aria-selected="model === o.value"
      @click="model = o.value"
    >
      {{ o.label }}
    </button>
  </div>
</template>

<script setup lang="ts" generic="T extends string">
defineProps<{ options: Array<{ value: T; label: string }>; label: string }>()
const model = defineModel<T>({ required: true })
</script>

<style scoped lang="scss">
.aseg {
  display: flex;
  gap: 2px;
  padding: 4px;
  border-radius: 999px;
  background: var(--m-s2);
}
.aseg__item {
  flex: 1;
  min-width: 0;
  min-height: 44px;
  padding: 0 12px;
  border: 0;
  border-radius: 999px;
  background: transparent;
  color: var(--el-text-color-secondary);
  font: inherit;
  font-size: 0.875rem;
  font-weight: 700;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color 0.18s,
    color 0.18s,
    box-shadow 0.18s;
}
.aseg__item:hover:not(.is-on) {
  color: var(--el-text-color-primary);
}
.aseg__item.is-on {
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  box-shadow: 0 1px 3px color-mix(in srgb, var(--m-ink) 14%, transparent);
}
</style>
