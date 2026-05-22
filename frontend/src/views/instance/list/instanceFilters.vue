<template>
  <el-form
    :model="modelValue"
    label-width="auto"
    class="instance-filter-form"
    @keyup.enter="emit('search')"
  >
    <el-row :gutter="10">
      <el-col :xs="24" :sm="12" :lg="8">
        <el-form-item label="实例关键词" prop="keyword">
          <el-input
            :model-value="modelValue.keyword"
            placeholder="实例名 / 场景标签"
            clearable
            @update:model-value="updateField('keyword', $event)"
          />
        </el-form-item>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-form-item label="实例类型" prop="type">
          <el-select
            :model-value="modelValue.type"
            placeholder="全部类型"
            clearable
            @update:model-value="updateField('type', $event)"
          >
            <el-option
              v-for="item in typeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="6">
        <el-form-item label="运行状态" prop="status">
          <el-select
            :model-value="modelValue.status"
            placeholder="全部状态"
            clearable
            @update:model-value="updateField('status', $event)"
          >
            <el-option
              v-for="item in statusOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
      </el-col>
      <el-col :xs="24" :sm="12" :lg="4">
        <el-form-item class="filter-action-item">
          <div class="filter-actions">
            <el-button
              type="primary"
              :icon="menuStore.iconComponents['HOutline:MagnifyingGlassIcon']"
              @click="emit('search')"
            >
              搜索
            </el-button>
            <el-button
              :icon="menuStore.iconComponents['HOutline:ArrowPathRoundedSquareIcon']"
              @click="emit('reset')"
            >
              重置
            </el-button>
          </div>
        </el-form-item>
      </el-col>
    </el-row>
  </el-form>
</template>

<script setup lang="ts">
import type { InstanceStatus, InstanceTypeOption, QueryFormValue } from '@/stores/instance/types'

const props = defineProps<{
  modelValue: QueryFormValue
  typeOptions: InstanceTypeOption[]
  statusOptions: Array<{ label: string; value: InstanceStatus }>
}>()

const emit = defineEmits<{
  'update:modelValue': [value: QueryFormValue]
  search: []
  reset: []
}>()

const menuStore = useMenuStore()

const updateField = <K extends keyof QueryFormValue>(key: K, value: QueryFormValue[K]) => {
  emit('update:modelValue', {
    ...props.modelValue,
    [key]: value,
  })
}
</script>

<style scoped lang="scss">
.instance-filter-form :deep(.el-form-item) {
  margin-bottom: 0.9rem;
}

.instance-filter-form :deep(.el-form-item__label) {
  font-weight: 700;
  color: var(--el-text-color-regular);
}

.instance-filter-form :deep(.el-input__wrapper),
.instance-filter-form :deep(.el-select__wrapper) {
  border-radius: 0.75rem;
  background: var(--el-bg-color-page);
  box-shadow: none;
}

.instance-filter-form :deep(.el-input__wrapper.is-focus),
.instance-filter-form :deep(.el-select__wrapper.is-focused) {
  box-shadow: 0 0 0 1px color-mix(in srgb, var(--el-color-primary) 24%, transparent);
}

.filter-action-item :deep(.el-form-item__content) {
  width: 100%;
}

.filter-actions {
  display: grid;
  width: 100%;
  gap: 0.6rem;
  grid-template-columns: repeat(2, minmax(0, 1fr));
}

.filter-actions :deep(.el-button) {
  width: 100%;
  margin-left: 0;
  height: 2.35rem;
  border-radius: 0.75rem;
  box-shadow: none;
}

.filter-actions :deep(.el-button--primary) {
  --el-button-bg-color: color-mix(in srgb, var(--el-color-primary) 12%, var(--el-bg-color));
  --el-button-border-color: color-mix(in srgb, var(--el-color-primary) 26%, var(--el-border-color));
  --el-button-text-color: var(--el-text-color-primary);
  --el-button-hover-bg-color: color-mix(in srgb, var(--el-color-primary) 16%, var(--el-bg-color));
  --el-button-hover-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 32%,
    var(--el-border-color)
  );
  --el-button-hover-text-color: var(--el-text-color-primary);
  --el-button-active-bg-color: color-mix(in srgb, var(--el-color-primary) 20%, var(--el-bg-color));
  --el-button-active-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 38%,
    var(--el-border-color)
  );
  --el-button-active-text-color: var(--el-text-color-primary);
}

.filter-actions :deep(.el-button:not(.el-button--primary)) {
  --el-button-bg-color: var(--el-bg-color);
  --el-button-border-color: var(--el-border-color-extra-light);
  --el-button-text-color: var(--el-text-color-primary);
  --el-button-hover-bg-color: color-mix(in srgb, #000000 3%, var(--el-bg-color));
  --el-button-hover-border-color: color-mix(in srgb, #000000 12%, var(--el-border-color));
  --el-button-hover-text-color: var(--el-text-color-primary);
  --el-button-active-bg-color: color-mix(in srgb, #000000 6%, var(--el-bg-color));
  --el-button-active-border-color: color-mix(in srgb, #000000 16%, var(--el-border-color));
  --el-button-active-text-color: var(--el-text-color-primary);
}

:global(html.dark .instance-filter-form .el-input__wrapper),
:global(html.dark .instance-filter-form .el-select__wrapper) {
  background: color-mix(in srgb, var(--el-bg-color-page) 76%, #0f172a);
}

:global(html.dark .filter-actions .el-button--primary) {
  --el-button-bg-color: color-mix(in srgb, var(--el-color-primary) 24%, #111827);
  --el-button-border-color: color-mix(in srgb, var(--el-color-primary) 34%, var(--el-border-color));
  --el-button-text-color: #ffffff;
  --el-button-hover-bg-color: color-mix(in srgb, var(--el-color-primary) 28%, #111827);
  --el-button-hover-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 42%,
    var(--el-border-color)
  );
  --el-button-hover-text-color: #ffffff;
  --el-button-active-bg-color: color-mix(in srgb, var(--el-color-primary) 32%, #111827);
  --el-button-active-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 48%,
    var(--el-border-color)
  );
  --el-button-active-text-color: #ffffff;
}

:global(html.dark .filter-actions .el-button:not(.el-button--primary)) {
  --el-button-text-color: #ffffff;
  --el-button-hover-text-color: #ffffff;
  --el-button-active-text-color: #ffffff;
}

@media (width <= 992px) {
  .filter-actions {
    grid-template-columns: 1fr;
  }
}
</style>
