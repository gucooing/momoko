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
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.page-header__text {
  min-width: 0;
  flex: 1;
}
.page-header__title {
  font-family: var(--app-font-display);
  font-size: 1.875rem;
  font-weight: 800;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
  line-height: 1.2;
}
.page-header__desc {
  margin-top: 6px;
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
  line-height: 1.45;
}
.page-header__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
  flex-wrap: wrap;
  justify-content: flex-end;
}
/* 动作区按钮：统一 40px 全圆角药丸 */
.page-header__actions :deep(button) {
  min-height: 40px;
  height: 40px;
  padding-inline: 16px;
  border-radius: 999px;
  font-size: 0.875rem;
  font-weight: 700;
}
.page-header__actions :deep(button svg) {
  width: 16px;
  height: 16px;
}
@media (width <= 768px) {
  .page-header {
    align-items: flex-start;
    gap: 8px;
  }
  .page-header__title {
    font-size: 1.5rem;
  }
  .page-header__desc {
    display: -webkit-box;
    -webkit-line-clamp: 1;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .page-header__actions :deep(button) {
    min-height: 40px;
    height: 40px;
    padding-inline: 14px;
  }
}
</style>
