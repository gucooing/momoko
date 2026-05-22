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

      <VxeGrid v-bind="instanceTypeGridConfig" :loading="loading">
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

<style scoped></style>
