<!-- 页头：标题 + 一句描述 + 右侧主操作（05 P1/P2）。页面第一焦点是标题，不再包大卡。 -->
<template>
  <header class="page-header">
    <div class="page-header__text">
      <h1 class="page-header__title"><slot name="title">{{ title }}</slot></h1>
      <p v-if="description || $slots.description" class="page-header__desc">
        <slot name="description">{{ description }}</slot>
      </p>
    </div>
    <div v-if="$slots.actions" class="page-header__actions"><slot name="actions" /></div>
  </header>
</template>

<script setup lang="ts">
defineProps<{ title?: string; description?: string }>()
</script>

<style scoped lang="scss">
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 1rem;
}
.page-header__text {
  min-width: 0;
}
.page-header__title {
  font-size: 1.1875rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
  line-height: 1.3;
}
.page-header__desc {
  margin-top: 2px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.page-header__actions {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  flex-shrink: 0;
}
@media (width <= 768px) {
  .page-header__title {
    font-size: 1.125rem;
  }
}
</style>
