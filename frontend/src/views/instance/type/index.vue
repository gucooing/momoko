<!-- 实例类型（重写 · P1 列表）：PageHeader + FilterBar(关键字/状态) + 内联计数条 + DataTable / 移动卡 + 编辑弹窗。
     内置类型（isSystem）锁定：不可编辑/删除，操作列显示「—」。store 薄壳，保留 getInstanceTypeList/deleteTypeById 契约。 -->
<template>
  <div class="itype-page">
    <PageHeader :title="t('instance.typeTitle')" :description="t('instance.typePageDesc')">
      <template #actions>
        <UButton color="neutral" variant="soft" icon="i-lucide-rotate-cw" @click="getInstanceTypeList">
          {{ t('common.refresh') }}
        </UButton>
        <UButton color="primary" icon="i-lucide-plus" @click="openEditor()">
          {{ t('instance.addInstanceType') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @reset="resetFilter">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('instance.typeName') }}</label>
          <input v-model="keyword" class="app-input" :placeholder="t('instance.searchTypePlaceholder')" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('common.status') }}</label>
          <AppSelect v-model="statusFilter" :options="statusFilterOptions" />
        </div>
      </template>
    </FilterBar>

    <div class="itype-page__body">
      <div class="itype-page__bar">
        <span class="itype-page__hint">{{ t('system.common.total', { total: filteredList.length }) }}</span>
        <span class="itype-page__stat">
          <span class="itype-page__dot is-on" />{{ t('common.enabled') }} {{ enabledCount }}
        </span>
        <span class="itype-page__stat">
          <span class="itype-page__dot" />{{ t('common.disabled') }} {{ disabledCount }}
        </span>
      </div>

      <!-- 桌面：表格 -->
      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="filteredList"
        row-key="id"
        :loading="loading"
        :empty-text="t('instance.noInstanceTypeData')"
      >
        <template #cell-name="{ row }">
          <span class="itype-name">{{ row.name }}</span>
        </template>
        <template #cell-id="{ row }">
          <span class="itype-mono">{{ row.id }}</span>
        </template>
        <template #cell-isSystem="{ row }">
          <StatusPill
            :variant="row.isSystem ? 'warning' : 'success'"
            :label="row.isSystem ? t('instance.builtIn') : t('instance.custom')"
          />
        </template>
        <template #cell-isEnable="{ row }">
          <StatusPill
            :variant="row.isEnable ? 'success' : 'neutral'"
            :label="row.isEnable ? t('common.enabled') : t('common.disabled')"
          />
        </template>
        <template #cell-operation="{ row }">
          <ActionMenu v-if="!row.isSystem" :items="rowActions" @select="(key) => onRowAction(key, row)" />
          <span v-else class="itype-dim">—</span>
        </template>
      </DataTable>

      <!-- 移动：卡片 -->
      <template v-else>
        <div v-if="loading" class="itype-cards">
          <div v-for="i in 4" :key="i" class="itype-skeleton" />
        </div>
        <EmptyState
          v-else-if="!filteredList.length"
          icon="HOutline:CubeIcon"
          :title="t('instance.noInstanceTypeData')"
          :description="t('instance.typeEmptyDesc')"
        />
        <div v-else class="itype-cards">
          <EntityCard v-for="row in filteredList" :key="row.id">
            <template #title>{{ row.name }}</template>
            <template #status>
              <StatusPill
                :variant="row.isSystem ? 'warning' : 'success'"
                :label="row.isSystem ? t('instance.builtIn') : t('instance.custom')"
              />
            </template>
            <template #meta>
              <span class="itype-card__id itype-mono">{{ row.id }}</span>
            </template>
            <template #footer>
              <StatusPill
                :variant="row.isEnable ? 'success' : 'neutral'"
                :label="row.isEnable ? t('common.enabled') : t('common.disabled')"
              />
              <ActionMenu v-if="!row.isSystem" :items="rowActions" @select="(key) => onRowAction(key, row)" />
              <span v-else class="itype-dim">—</span>
            </template>
          </EntityCard>
        </div>
      </template>
    </div>

    <InstanceTypeCreate ref="editorRef" @refresh="getInstanceTypeList" />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { useI18n } from 'vue-i18n'
import { useInstanceTypeStore } from '@/stores/instance/type'
import { Dialog } from '@/utils/dialog'
import type { InstanceTypeInfo } from '@/types/v1/instance'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import InstanceTypeCreate from '@/views/instance/type/create.vue'

defineOptions({ name: 'TypeView' })

const menuStore = useMenuStore()
const instanceTypeStore = useInstanceTypeStore()
const { t } = useI18n()

const editorRef = useTemplateRef<InstanceType<typeof InstanceTypeCreate> | null>('editorRef')

const { loading, instanceTypeList } = storeToRefs(instanceTypeStore)
const { getInstanceTypeList, deleteTypeById } = instanceTypeStore

// —— 客户端筛选（store 只加载全量，视图侧过滤，保留 store 契约不变）——
type StatusFilter = '' | 'enabled' | 'disabled'
const keyword = ref('')
const statusFilter = ref<StatusFilter>('')
const statusFilterOptions = computed<{ label: string; value: StatusFilter }[]>(() => [
  { label: t('instance.allStatus'), value: '' },
  { label: t('common.enabled'), value: 'enabled' },
  { label: t('common.disabled'), value: 'disabled' },
])

const filteredList = computed(() => {
  const kw = keyword.value.trim().toLowerCase()
  return instanceTypeList.value.filter((row) => {
    if (kw && !`${row.name} ${row.id}`.toLowerCase().includes(kw)) return false
    if (statusFilter.value === 'enabled' && !row.isEnable) return false
    if (statusFilter.value === 'disabled' && row.isEnable) return false
    return true
  })
})

const enabledCount = computed(() => filteredList.value.filter((r) => r.isEnable).length)
const disabledCount = computed(() => filteredList.value.filter((r) => !r.isEnable).length)

const resetFilter = () => {
  keyword.value = ''
  statusFilter.value = ''
}

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('instance.typeName'), minWidth: 200 },
  { key: 'id', title: t('instance.typeId'), minWidth: 240 },
  { key: 'isSystem', title: t('common.type'), width: 120 },
  { key: 'isEnable', title: t('common.status'), width: 120 },
  { key: 'operation', title: t('common.operation'), width: 80, align: 'center' },
])

const rowActions = computed<ActionMenuItem[]>(() => [
  { key: 'edit', label: t('common.edit'), icon: 'HOutline:PencilSquareIcon' },
  { key: 'delete', label: t('common.delete'), icon: 'HOutline:TrashIcon', danger: true },
])

const openEditor = (row?: InstanceTypeInfo) => editorRef.value?.showDialog(row)

const onRowAction = (key: string, row: Record<string, unknown>) => {
  const item = instanceTypeList.value.find((x) => x.id === String(row.id))
  if (!item) return
  if (key === 'edit') openEditor(item)
  else if (key === 'delete') confirmDelete(item)
}

const confirmDelete = (item: InstanceTypeInfo) => {
  Dialog.info({
    showCancelButton: true,
    content: t('instance.deleteInstanceTypeConfirm'),
    confirmText: t('common.delete'),
    cancelText: t('common.cancel'),
    onConfirm: async () => {
      await deleteTypeById(item.id)
      ElMessage.success(t('common.deleteSuccess'))
    },
  })
}

onMounted(() => {
  void getInstanceTypeList()
})
</script>

<style scoped lang="scss">
.itype-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.itype-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.itype-page__bar {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 6px 16px;
}
.itype-page__hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.itype-page__stat {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.itype-page__dot {
  width: 7px;
  height: 7px;
  border-radius: 999px;
  flex-shrink: 0;
  background: var(--el-text-color-placeholder);
}
.itype-page__dot.is-on {
  background: var(--el-color-success, #16a34a);
}

.itype-name {
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.itype-mono {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
}
.itype-dim {
  color: var(--el-text-color-placeholder);
}

/* 移动卡片 */
.itype-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.itype-card__id {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.itype-skeleton {
  height: 96px;
  border-radius: var(--app-radius);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: itype-shimmer 1.4s ease-in-out infinite;
}
@keyframes itype-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
