<!-- 空状态：安静的单色图标 + 标题 + 可选说明/操作。不用彩色（01 §8）。 -->
<template>
  <div class="empty-state">
    <div v-if="icon" class="empty-state__icon">
      <component :is="menuStore.iconComponents[icon]" />
    </div>
    <p class="empty-state__title">{{ title }}</p>
    <p v-if="description" class="empty-state__desc">{{ description }}</p>
    <div v-if="$slots.action" class="empty-state__action"><slot name="action" /></div>
  </div>
</template>

<script setup lang="ts">
withDefaults(
  defineProps<{ icon?: string; title?: string; description?: string }>(),
  { icon: 'HOutline:InboxIcon' },
)
const menuStore = useMenuStore()
</script>

<style scoped lang="scss">
.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  gap: 0.375rem;
  padding: 1.75rem 1rem;
}
.empty-state__icon {
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 12px;
  margin-bottom: 0.25rem;
  color: var(--m-ink2);
  background: var(--m-s2);
}
.empty-state__icon :deep(svg) {
  width: 20px;
  height: 20px;
}
.empty-state__title {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.empty-state__desc {
  font-size: 0.75rem;
  color: var(--el-text-color-secondary);
  max-width: 24rem;
}
.empty-state__action {
  margin-top: 0.5rem;
}
</style>
