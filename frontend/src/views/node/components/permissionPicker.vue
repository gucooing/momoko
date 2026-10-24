<!-- API Key 节点权限选择：按功能分组的复选框（v-model 为权限标识数组）。
     远程主控只能使用这里勾选、且 API Key 所属用户本身拥有的功能。 -->
<template>
  <div class="perm-picker">
    <div class="perm-picker__bar">
      <span class="perm-picker__count">{{ t('node.permission.selected', { count: modelValue.length }) }}</span>
      <button type="button" class="perm-picker__link" @click="selectAll">{{ t('node.permission.selectAll') }}</button>
      <button type="button" class="perm-picker__link" @click="clear">{{ t('node.permission.clear') }}</button>
    </div>
    <div class="perm-picker__groups">
      <fieldset v-for="group in NODE_PERMISSION_GROUPS" :key="group.key" class="perm-group">
        <legend class="perm-group__title">{{ t(group.labelKey) }}</legend>
        <label v-for="item in group.items" :key="item.code" class="perm-item" :class="{ 'is-checked': selected.has(item.code) }">
          <input type="checkbox" :checked="selected.has(item.code)" @change="toggle(item.code)" />
          <span class="perm-item__label">{{ t(item.labelKey) }}</span>
          <code class="perm-item__code">{{ item.code }}</code>
        </label>
      </fieldset>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { NODE_PERMISSION_GROUPS, NODE_PERMISSION_ITEMS } from '@/config/nodePermission'

const props = defineProps<{ modelValue: string[] }>()
const emit = defineEmits<{ 'update:modelValue': [value: string[]] }>()
const { t } = useI18n()

const selected = computed(() => new Set(props.modelValue))

// 按目录顺序输出，便于与后端规范化结果一致
const emitOrdered = (set: Set<string>) => {
  emit(
    'update:modelValue',
    NODE_PERMISSION_ITEMS.filter((item) => set.has(item.code)).map((item) => item.code),
  )
}

const toggle = (code: string) => {
  const next = new Set(selected.value)
  if (next.has(code)) next.delete(code)
  else next.add(code)
  emitOrdered(next)
}
const selectAll = () => emitOrdered(new Set(NODE_PERMISSION_ITEMS.map((item) => item.code)))
const clear = () => emitOrdered(new Set())
</script>

<style scoped lang="scss">
.perm-picker {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.perm-picker__bar {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 0.75rem;
}
.perm-picker__count {
  flex: 1;
  color: var(--el-text-color-secondary);
}
.perm-picker__link {
  padding: 0;
  border: none;
  background: transparent;
  color: var(--el-color-primary);
  cursor: pointer;
  font-size: 0.75rem;
}
.perm-picker__groups {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}
@media (width <= 640px) {
  .perm-picker__groups {
    grid-template-columns: 1fr;
  }
}
.perm-group {
  margin: 0;
  padding: 8px 10px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-sm);
  min-width: 0;
}
.perm-group__title {
  padding: 0 4px;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.perm-item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 2px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.perm-item.is-checked {
  color: var(--el-text-color-primary);
}
.perm-item__label {
  flex: 1;
  min-width: 0;
}
.perm-item__code {
  font-size: 0.6875rem;
  color: var(--el-text-color-placeholder);
}
</style>
