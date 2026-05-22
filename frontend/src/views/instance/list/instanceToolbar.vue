<template>
  <div class="operation-container toolbar-row">
    <div class="toolbar-left">
      <el-button
        class="toolbar-button toolbar-button--primary"
        :icon="menuStore.iconComponents['HOutline:PlusCircleIcon']"
        @click="emit('create')"
      >
        新建实例
      </el-button>
      <el-button
        :class="[
          'toolbar-button',
          canBatchStart ? 'toolbar-button--available' : 'toolbar-button--soft',
        ]"
        :icon="menuStore.iconComponents['HOutline:PlayIcon']"
        :disabled="!canBatchStart"
        @click="emit('batchStart')"
      >
        批量启动
      </el-button>
      <el-button
        :class="[
          'toolbar-button',
          canBatchStop ? 'toolbar-button--available' : 'toolbar-button--soft',
        ]"
        :icon="menuStore.iconComponents['HOutline:StopIcon']"
        :disabled="!canBatchStop"
        @click="emit('batchStop')"
      >
        批量停止
      </el-button>
      <el-button
        class="toolbar-button toolbar-button--danger"
        :icon="menuStore.iconComponents['HOutline:TrashIcon']"
        :disabled="!canBatchDelete"
        @click="emit('batchDelete')"
      >
        批量删除
      </el-button>
    </div>
    <div class="toolbar-right">
      <span class="selection-text">已选 {{ selectedCount }} 项</span>
      <el-button link class="toolbar-text-button" @click="emit('toggleCurrentPage')">
        {{ isCurrentPageAllSelected ? '取消全选当前页' : '全选当前页' }}
      </el-button>
      <el-button text class="toolbar-text-button" @click="emit('refresh')">刷新状态</el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
defineProps<{
  selectedCount: number
  canBatchStart: boolean
  canBatchStop: boolean
  canBatchDelete: boolean
  isCurrentPageAllSelected: boolean
}>()

const emit = defineEmits<{
  create: []
  batchStart: []
  batchStop: []
  batchDelete: []
  toggleCurrentPage: []
  refresh: []
}>()

const menuStore = useMenuStore()
</script>

<style scoped lang="scss">
.toolbar-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
}

.toolbar-right {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  flex-wrap: wrap;
}

.toolbar-left {
  display: flex;
  flex-wrap: wrap;
  gap: 0.75rem;
}

.toolbar-button {
  min-width: 7rem;
  height: 2.35rem;
  border-radius: 0.75rem;
  --el-button-bg-color: var(--el-bg-color-page);
  --el-button-text-color: var(--el-text-color-primary);
  --el-button-border-color: var(--el-border-color-extra-light);
  --el-button-hover-bg-color: color-mix(in srgb, #000000 3%, var(--el-bg-color-page));
  --el-button-hover-text-color: var(--el-text-color-primary);
  --el-button-hover-border-color: var(--el-border-color);
  --el-button-active-bg-color: color-mix(in srgb, #000000 6%, var(--el-bg-color-page));
  --el-button-active-text-color: var(--el-text-color-primary);
  --el-button-active-border-color: var(--el-border-color);
  --el-button-disabled-bg-color: color-mix(in srgb, #000000 2%, var(--el-bg-color-page));
  --el-button-disabled-text-color: #a0adbf;
  --el-button-disabled-border-color: var(--el-border-color-extra-light);
  box-shadow: none;
  transition:
    transform 0.15s ease,
    box-shadow 0.2s ease,
    background-color 0.2s ease,
    border-color 0.2s ease;
}

.toolbar-left :deep(.el-button + .el-button) {
  margin-left: 0;
}

.toolbar-right :deep(.el-button + .el-button) {
  margin-left: 0;
}

.toolbar-button:hover:not(:disabled) {
  transform: translateY(-1px);
  box-shadow: 0 8px 16px rgb(15 23 42 / 6%);
}

.toolbar-button:active:not(:disabled) {
  transform: translateY(0);
  box-shadow: 0 4px 8px rgb(15 23 42 / 5%);
}

.toolbar-button--primary {
  --el-button-bg-color: color-mix(in srgb, var(--el-color-primary) 12%, var(--el-bg-color));
  --el-button-text-color: var(--el-text-color-primary);
  --el-button-border-color: color-mix(in srgb, var(--el-color-primary) 26%, var(--el-border-color));
  --el-button-hover-bg-color: color-mix(in srgb, var(--el-color-primary) 16%, var(--el-bg-color));
  --el-button-hover-text-color: var(--el-text-color-primary);
  --el-button-hover-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 32%,
    var(--el-border-color)
  );
  --el-button-active-bg-color: color-mix(in srgb, var(--el-color-primary) 20%, var(--el-bg-color));
  --el-button-active-text-color: var(--el-text-color-primary);
  --el-button-active-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 38%,
    var(--el-border-color)
  );
  box-shadow: none;
}

.toolbar-button--soft {
  --el-button-bg-color: var(--el-bg-color);
  --el-button-text-color: var(--el-text-color-primary);
  --el-button-border-color: var(--el-border-color-extra-light);
  --el-button-hover-bg-color: color-mix(in srgb, #000000 3%, var(--el-bg-color));
  --el-button-hover-text-color: var(--el-text-color-primary);
  --el-button-hover-border-color: var(--el-border-color);
  --el-button-active-bg-color: color-mix(in srgb, #000000 6%, var(--el-bg-color));
  --el-button-active-text-color: var(--el-text-color-primary);
  --el-button-active-border-color: var(--el-border-color);
}

.toolbar-button--available {
  --el-button-bg-color: color-mix(in srgb, var(--el-color-primary) 10%, var(--el-bg-color));
  --el-button-text-color: var(--el-text-color-primary);
  --el-button-border-color: color-mix(in srgb, var(--el-color-primary) 22%, var(--el-border-color));
  --el-button-hover-bg-color: color-mix(in srgb, var(--el-color-primary) 14%, var(--el-bg-color));
  --el-button-hover-text-color: var(--el-text-color-primary);
  --el-button-hover-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 28%,
    var(--el-border-color)
  );
  --el-button-active-bg-color: color-mix(in srgb, var(--el-color-primary) 18%, var(--el-bg-color));
  --el-button-active-text-color: var(--el-text-color-primary);
  --el-button-active-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 34%,
    var(--el-border-color)
  );
}

.toolbar-button--danger {
  --el-button-bg-color: color-mix(in srgb, var(--el-color-danger) 5%, var(--el-bg-color));
  --el-button-text-color: var(--el-text-color-primary);
  --el-button-border-color: color-mix(in srgb, var(--el-color-danger) 16%, var(--el-border-color));
  --el-button-hover-bg-color: color-mix(in srgb, var(--el-color-danger) 8%, var(--el-bg-color));
  --el-button-hover-text-color: var(--el-text-color-primary);
  --el-button-hover-border-color: color-mix(
    in srgb,
    var(--el-color-danger) 22%,
    var(--el-border-color)
  );
  --el-button-active-bg-color: color-mix(in srgb, var(--el-color-danger) 10%, var(--el-bg-color));
  --el-button-active-text-color: var(--el-text-color-primary);
  --el-button-active-border-color: color-mix(
    in srgb,
    var(--el-color-danger) 26%,
    var(--el-border-color)
  );
}

.toolbar-text-button {
  --el-button-text-color: var(--el-text-color-primary);
  --el-button-hover-text-color: var(--el-text-color-primary);
  --el-button-active-text-color: var(--el-text-color-primary);
  font-weight: 500;
}

.toolbar-text-button:hover {
  text-decoration: underline;
  text-underline-offset: 2px;
}

:global(html.dark .toolbar-button) {
  --el-button-bg-color: color-mix(in srgb, var(--el-bg-color-page) 78%, #0f172a);
  --el-button-text-color: #ffffff;
  --el-button-border-color: color-mix(in srgb, var(--el-border-color) 72%, transparent);
  --el-button-hover-bg-color: #242d3b;
  --el-button-hover-text-color: #ffffff;
  --el-button-hover-border-color: color-mix(in srgb, var(--el-border-color) 88%, transparent);
  --el-button-active-bg-color: #2a3442;
  --el-button-active-text-color: #ffffff;
  --el-button-active-border-color: color-mix(in srgb, var(--el-border-color) 96%, transparent);
  --el-button-disabled-bg-color: #161c26;
  --el-button-disabled-text-color: #7f8da2;
  --el-button-disabled-border-color: color-mix(in srgb, var(--el-border-color) 60%, transparent);
}

:global(html.dark .toolbar-button--primary) {
  --el-button-bg-color: color-mix(in srgb, var(--el-color-primary) 24%, #111827);
  --el-button-text-color: #ffffff;
  --el-button-border-color: color-mix(in srgb, var(--el-color-primary) 34%, var(--el-border-color));
  --el-button-hover-bg-color: color-mix(in srgb, var(--el-color-primary) 28%, #111827);
  --el-button-hover-text-color: #ffffff;
  --el-button-hover-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 42%,
    var(--el-border-color)
  );
  --el-button-active-bg-color: color-mix(in srgb, var(--el-color-primary) 32%, #111827);
  --el-button-active-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 48%,
    var(--el-border-color)
  );
}

:global(html.dark .toolbar-button--available) {
  --el-button-bg-color: color-mix(in srgb, var(--el-color-primary) 18%, #111827);
  --el-button-text-color: #ffffff;
  --el-button-border-color: color-mix(in srgb, var(--el-color-primary) 28%, var(--el-border-color));
  --el-button-hover-bg-color: color-mix(in srgb, var(--el-color-primary) 22%, #111827);
  --el-button-hover-text-color: #ffffff;
  --el-button-hover-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 36%,
    var(--el-border-color)
  );
  --el-button-active-bg-color: color-mix(in srgb, var(--el-color-primary) 26%, #111827);
  --el-button-active-text-color: #ffffff;
  --el-button-active-border-color: color-mix(
    in srgb,
    var(--el-color-primary) 42%,
    var(--el-border-color)
  );
}

:global(html.dark .toolbar-button--danger) {
  --el-button-bg-color: color-mix(in srgb, var(--el-color-danger) 12%, #111827);
  --el-button-text-color: #ffffff;
  --el-button-border-color: color-mix(in srgb, var(--el-color-danger) 18%, var(--el-border-color));
  --el-button-hover-bg-color: color-mix(in srgb, var(--el-color-danger) 16%, #111827);
  --el-button-hover-text-color: #ffffff;
  --el-button-hover-border-color: color-mix(
    in srgb,
    var(--el-color-danger) 24%,
    var(--el-border-color)
  );
  --el-button-active-bg-color: color-mix(in srgb, var(--el-color-danger) 20%, #111827);
  --el-button-active-text-color: #ffffff;
  --el-button-active-border-color: color-mix(
    in srgb,
    var(--el-color-danger) 28%,
    var(--el-border-color)
  );
}

:global(html.dark .toolbar-text-button) {
  --el-button-text-color: #ffffff;
  --el-button-hover-text-color: #ffffff;
  --el-button-active-text-color: #ffffff;
}

.selection-text {
  display: inline-flex;
  align-items: center;
  border-radius: 999px;
  background: var(--el-bg-color-page);
  padding: 0.38rem 0.7rem;
  font-size: 0.82rem;
  font-weight: 700;
  color: var(--el-text-color-secondary);
}

@media (width <= 992px) {
  .toolbar-row {
    align-items: stretch;
    flex-direction: column;
  }

  .toolbar-left {
    width: 100%;
  }

  .toolbar-right {
    width: 100%;
    justify-content: space-between;
  }
}

@media (width <= 576px) {
  .toolbar-left {
    flex-direction: column;
  }

  .toolbar-button {
    width: 100%;
  }
}
</style>
