<template>
  <div>
    <el-card shadow="never">
      <div class="operation-container">
        <el-button
          type="primary"
          :icon="menuStore.iconComponents.Plus"
          @click="instanceTypeCreateRef?.showDialog()"
        >
          新增实例类型
        </el-button>
        <el-button :icon="menuStore.iconComponents.Refresh" @click="getInstanceTypeList">
          刷新
        </el-button>
      </div>

      <!-- desktop: table -->
      <VxeGrid v-if="!menuStore.isMobile" v-bind="instanceTypeGridConfig" :loading="loading">
        <template #column-type="{ row }: { row: InstanceTypeInfo }">
          <BaseTag :type="row.isSystem ? 'warning' : 'success'" :text="row.isSystem ? '内置' : '自定义'" />
        </template>
        <template #column-status="{ row }: { row: InstanceTypeInfo }">
          <BaseTag :type="row.isEnable ? 'success' : 'danger'" :text="row.isEnable ? '启用' : '禁用'" />
        </template>
        <template #column-operation="{ row }: { row: InstanceTypeInfo }">
          <template v-if="!row.isSystem">
            <el-button
              type="primary"
              :icon="menuStore.iconComponents.Edit"
              link
              @click="instanceTypeCreateRef?.showDialog(row)"
            >
              编辑
            </el-button>
            <AdaptiveConfirm
              title="确定要删除这个实例类型吗？"
              :placement="POPCONFIRM_CONFIG.placement"
              :width="POPCONFIRM_CONFIG.width"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <el-button type="danger" :icon="menuStore.iconComponents.Delete" link>
                  删除
                </el-button>
              </template>
            </AdaptiveConfirm>
          </template>
          <span v-else>-</span>
        </template>
      </VxeGrid>

      <!-- mobile: cards -->
      <div v-else v-loading="loading" class="mobile-card-list">
        <div v-if="!instanceTypeList.length" class="mobile-empty">
          <el-empty description="暂无数据" />
        </div>
        <div v-for="row in instanceTypeList" :key="row.id" class="type-card">
          <div class="type-card-header">
            <span class="type-card-name">{{ row.name }}</span>
            <BaseTag :type="row.isSystem ? 'warning' : 'success'" :text="row.isSystem ? '内置' : '自定义'" />
          </div>
          <div class="type-card-body">
            <span class="type-card-id">{{ row.id }}</span>
            <BaseTag :type="row.isEnable ? 'success' : 'danger'" :text="row.isEnable ? '启用' : '禁用'" />
          </div>
          <div v-if="!row.isSystem" class="type-card-footer">
            <el-button type="primary" size="small" plain @click="instanceTypeCreateRef?.showDialog(row)">
              编辑
            </el-button>
            <AdaptiveConfirm
              title="确定要删除这个实例类型吗？"
              :placement="POPCONFIRM_CONFIG.placement"
              :width="POPCONFIRM_CONFIG.width"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <el-button type="danger" size="small" plain>删除</el-button>
              </template>
            </AdaptiveConfirm>
          </div>
        </div>
      </div>
    </el-card>

    <InstanceTypeCreate ref="instanceTypeCreateRef" @refresh="getInstanceTypeList" />
  </div>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { POPCONFIRM_CONFIG } from '@/config/elementConfig'
import { VxeGrid } from '@/plugins/vxeGrid'
import { useInstanceTypeStore } from '@/stores/instance/type'
import type { InstanceTypeInfo } from '@/types/v1/instance'
import InstanceTypeCreate from '@/views/instance/type/create.vue'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'TypeView' })

const menuStore = useMenuStore()
const instanceTypeStore = useInstanceTypeStore()
const instanceTypeCreateRef = useTemplateRef<InstanceType<typeof InstanceTypeCreate> | null>(
  'instanceTypeCreateRef',
)

const { loading, instanceTypeList } = storeToRefs(instanceTypeStore)
const { getInstanceTypeList, deleteTypeById } = instanceTypeStore

const instanceTypeGridConfig = computed<VxeGridProps>(() => ({
  border: true,
  showOverflow: true,
  rowConfig: { isHover: true },
  data: instanceTypeList.value,
  columns: [
    { type: 'seq', title: '序号', width: 60, fixed: 'left' },
    { field: 'name', title: '类型名称', minWidth: 220, fixed: 'left' },
    { field: 'id', title: '类型 ID', minWidth: 260 },
    { field: 'isSystem', title: '类型', width: 120, slots: { default: 'column-type' } },
    { field: 'isEnable', title: '状态', width: 120, slots: { default: 'column-status' } },
    { title: '操作', width: 160, fixed: 'right', slots: { default: 'column-operation' } },
  ],
}))

const handleDelete = async (id: string) => {
  await deleteTypeById(id)
  ElMessage.success('删除成功')
}

onMounted(() => {
  void getInstanceTypeList()
})
</script>

<style scoped lang="scss">
.mobile-card-list {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.mobile-empty {
  padding: 1.5rem 0;
}

.type-card {
  padding: 0.8rem 0.9rem;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.65rem;
  background: var(--el-bg-color);
}

.type-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
}

.type-card-name {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}

.type-card-body {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 0.5rem;
  margin-top: 0.45rem;
}

.type-card-id {
  font-size: 0.72rem;
  color: var(--el-text-color-placeholder);
  font-family: monospace;
}

.type-card-footer {
  display: flex;
  gap: 0.5rem;
  margin-top: 0.6rem;
  padding-top: 0.55rem;
  border-top: 1px solid var(--el-border-color-extra-light);
}
</style>
