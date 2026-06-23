<template>
  <div>
    <el-card shadow="never">
      <div class="operation-container">
        <el-button
          type="primary"
          :icon="menuStore.iconComponents.Plus"
          @click="instanceTypeCreateRef?.showDialog()"
        >
          {{ t('instance.addInstanceType') }}
        </el-button>
        <el-button :icon="menuStore.iconComponents.Refresh" @click="getInstanceTypeList">
          {{ t('common.refresh') }}
        </el-button>
      </div>

      <!-- desktop: table -->
      <VxeGrid v-if="!menuStore.isMobile" v-bind="instanceTypeGridConfig" :loading="loading">
        <template #column-type="{ row }: { row: InstanceTypeInfo }">
          <BaseTag :type="row.isSystem ? 'warning' : 'success'" :text="row.isSystem ? t('instance.builtIn') : t('instance.custom')" />
        </template>
        <template #column-status="{ row }: { row: InstanceTypeInfo }">
          <BaseTag :type="row.isEnable ? 'success' : 'danger'" :text="row.isEnable ? t('common.enabled') : t('common.disabled')" />
        </template>
        <template #column-operation="{ row }: { row: InstanceTypeInfo }">
          <template v-if="!row.isSystem">
            <el-button
              type="primary"
              :icon="menuStore.iconComponents.Edit"
              link
              @click="instanceTypeCreateRef?.showDialog(row)"
            >
              {{ t('common.edit') }}
            </el-button>
            <AdaptiveConfirm
              :title="t('instance.deleteInstanceTypeConfirm')"
              :placement="POPCONFIRM_CONFIG.placement"
              :width="POPCONFIRM_CONFIG.width"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <el-button type="danger" :icon="menuStore.iconComponents.Delete" link>
                  {{ t('common.delete') }}
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
          <el-empty :description="t('instance.noInstanceTypeData')" />
        </div>
        <div v-for="row in instanceTypeList" :key="row.id" class="type-card">
          <div class="type-card-header">
            <span class="type-card-name">{{ row.name }}</span>
            <BaseTag :type="row.isSystem ? 'warning' : 'success'" :text="row.isSystem ? t('instance.builtIn') : t('instance.custom')" />
          </div>
          <div class="type-card-body">
            <span class="type-card-id">{{ row.id }}</span>
            <BaseTag :type="row.isEnable ? 'success' : 'danger'" :text="row.isEnable ? t('common.enabled') : t('common.disabled')" />
          </div>
          <div v-if="!row.isSystem" class="type-card-footer">
            <el-button type="primary" size="small" plain @click="instanceTypeCreateRef?.showDialog(row)">
              {{ t('common.edit') }}
            </el-button>
            <AdaptiveConfirm
              :title="t('instance.deleteInstanceTypeConfirm')"
              :placement="POPCONFIRM_CONFIG.placement"
              :width="POPCONFIRM_CONFIG.width"
              @confirm="handleDelete(row.id)"
            >
              <template #reference>
                <el-button type="danger" size="small" plain>{{ t('common.delete') }}</el-button>
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
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'TypeView' })

const menuStore = useMenuStore()
const instanceTypeStore = useInstanceTypeStore()
const { t } = useI18n()
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
    { type: 'seq', title: t('instance.serialNumber'), width: 60, fixed: 'left' },
    { field: 'name', title: t('instance.typeName'), minWidth: 220, fixed: 'left' },
    { field: 'id', title: t('instance.typeId'), minWidth: 260 },
    { field: 'isSystem', title: t('common.type'), width: 120, slots: { default: 'column-type' } },
    { field: 'isEnable', title: t('common.status'), width: 120, slots: { default: 'column-status' } },
    { title: t('common.operation'), width: 160, fixed: 'right', slots: { default: 'column-operation' } },
  ],
}))

const handleDelete = async (id: string) => {
  await deleteTypeById(id)
  ElMessage.success(t('common.deleteSuccess'))
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
