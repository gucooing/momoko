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
  gap: 0.5rem;
  padding: 2.5rem 1.5rem;
}
.empty-state__icon {
  width: 60px;
  height: 60px;
  display: flex;
  align-items: center;
  justify-content: center;
  border-radius: 20px;
  margin-bottom: 0.4rem;
  color: var(--m-pInk);
  background: var(--m-pSoft);
  transform: rotate(-6deg);
}
.empty-state__icon :deep(svg) {
  width: 28px;
  height: 28px;
}
.empty-state__title {
  font-size: 1rem;
  font-weight: 800;
  color: var(--el-text-color-primary);
}
.empty-state__desc {
  font-size: 0.875rem;
  color: var(--el-text-color-secondary);
  max-width: 24rem;
}
.empty-state__action {
  margin-top: 0.75rem;
}
</style>
