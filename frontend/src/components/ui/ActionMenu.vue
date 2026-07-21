<!-- 行内操作菜单：⋯ 触发 + AppDropdown 面板。用于表格行/卡片的多操作收纳（03b §5）。 -->
<template>
  <AppDropdown align="end" :width="160">
    <template #trigger>
      <!-- 触发器也走 UIcon，避免 Heroicons 异步组件未就绪时空白 -->
      <button type="button" class="action-menu__trigger" :aria-label="label" :title="label">
        <UIcon name="i-lucide-ellipsis" class="action-menu__trigger-ico" />
      </button>
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
          <!-- icon 支持 i-lucide-*（新）或 HOutline:*（存量页兼容） -->
          <UIcon v-if="item.icon && isLucide(item.icon)" :name="item.icon" class="action-menu__ico" />
          <component
            :is="menuStore.iconComponents[item.icon]"
            v-else-if="item.icon"
            class="action-menu__ico"
          />
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
  /** Lucide iconify 名（i-lucide-*）或存量 Heroicons key（HOutline:*） */
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
const isLucide = (icon: string) => icon.startsWith('i-')

const onSelect = (item: ActionMenuItem, close: () => void) => {
  if (item.disabled) return
  close()
  emit('select', item.key)
}
</script>

<style scoped lang="scss">
.action-menu__trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.action-menu__trigger:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.action-menu__trigger:focus-visible {
  outline: none;
  box-shadow: var(--app-focus-ring, 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 25%, transparent));
}
@media (pointer: coarse) {
  .action-menu__trigger {
    width: 40px;
    height: 40px;
  }
}
.action-menu__trigger-ico {
  width: 18px;
  height: 18px;
}
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
  transition: background 0.15s, color 0.15s, box-shadow 0.15s;
}
.action-menu__item:focus-visible {
  outline: none;
  box-shadow: var(--app-focus-ring, 0 0 0 3px color-mix(in srgb, var(--el-color-primary) 25%, transparent));
}
.action-menu__ico,
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
