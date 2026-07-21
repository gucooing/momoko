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
  font-size: 1.125rem;
  font-weight: 700;
  letter-spacing: -0.02em;
  color: var(--el-text-color-primary);
  line-height: 1.25;
}
.page-header__desc {
  margin-top: 2px;
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  line-height: 1.35;
}
.page-header__actions {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-shrink: 0;
}
/* 动作区按钮默认压到 sm 高度，避免默认 UButton 在移动端显得「超级大」 */
.page-header__actions :deep(button) {
  min-height: 28px;
  height: 28px;
  padding-inline: 10px;
  font-size: 0.8125rem;
}
.page-header__actions :deep(button svg) {
  width: 14px;
  height: 14px;
}
@media (width <= 768px) {
  .page-header {
    align-items: flex-start;
    gap: 8px;
  }
  .page-header__title {
    font-size: 1rem;
  }
  .page-header__desc {
    display: -webkit-box;
    -webkit-line-clamp: 1;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
  .page-header__actions :deep(button) {
    min-height: 36px;
    height: 36px;
  }
}
</style>
