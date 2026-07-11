<!-- 行内操作菜单：⋯ 触发 + AppDropdown 面板。用于表格行/卡片的多操作收纳（03b §5）。 -->
<template>
  <AppDropdown align="end" :width="160">
    <template #trigger>
      <AppIconButton icon="HOutline:EllipsisHorizontalIcon" :label="label" :size="18" />
    </template>
    <template #default="{ close }">
      <div class="action-menu">
        <button
          v-for="item in visibleItems"
          :key="item.key"
          type="button"
          class="action-menu__item"
          :class="{ 'action-menu__item--danger': item.danger }"
          :disabled="item.disabled"
          @click="onSelect(item, close)"
        >
          <component :is="menuStore.iconComponents[item.icon]" v-if="item.icon" />
          <span>{{ item.label }}</span>
        </button>
      </div>
    </template>
  </AppDropdown>
</template>

<script setup lang="ts">
export interface ActionMenuItem {
  key: string
  label: string
  icon?: string
  danger?: boolean
  disabled?: boolean
  hidden?: boolean
}

const props = withDefaults(defineProps<{ items: ActionMenuItem[]; label?: string }>(), {
  label: '操作',
})
const emit = defineEmits<{ select: [key: string] }>()

const menuStore = useMenuStore()
const visibleItems = computed(() => props.items.filter((i) => !i.hidden))

const onSelect = (item: ActionMenuItem, close: () => void) => {
  if (item.disabled) return
  close()
  emit('select', item.key)
}
</script>

<style scoped lang="scss">
.action-menu {
  padding: 6px;
}
.action-menu__item {
  display: flex;
  align-items: center;
  gap: 10px;
  width: 100%;
  padding: 8px 10px;
  border: none;
  background: transparent;
  border-radius: var(--app-radius-sm);
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  text-align: left;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.action-menu__item :deep(svg) {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
.action-menu__item:hover:not(:disabled) {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.action-menu__item:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}
.action-menu__item--danger {
  color: var(--el-color-danger, #ef4444);
}
.action-menu__item--danger:hover:not(:disabled) {
  background: color-mix(in srgb, var(--el-color-danger, #ef4444) 10%, transparent);
  color: var(--el-color-danger, #ef4444);
}
</style>
