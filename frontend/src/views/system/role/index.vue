<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <el-form :model="queryForm" label-width="auto" ref="queryFormRef" @keyup.enter="getRoleList">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item label="角色名称" prop="name">
              <el-input v-model="queryForm.name" placeholder="请输入角色名称" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item label="状态" prop="status">
              <el-select v-model="queryForm.status" placeholder="请选择">
                <el-option label="启用" :value="RoleStatus.RoleStatus_Active" />
                <el-option label="停用" :value="RoleStatus.RoleStatus_InActive" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item>
              <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="getRoleList">
                搜索
              </el-button>
              <el-button :icon="menuStore.iconComponents.Refresh" @click="reset">重置</el-button>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <el-card shadow="never" class="card-mt-16">
      <div class="operation-container">
        <el-button
          type="primary"
          :icon="menuStore.iconComponents.Plus"
          @click="roleCreateRef?.showDialog(undefined)"
          v-permission="[PERM.ROLE_ADD]"
        >
          新增角色
        </el-button>
        <AdaptiveConfirm
          title="确定要删除选中的角色吗？"
          :placement="POPCONFIRM_CONFIG.placement"
          :width="POPCONFIRM_CONFIG.width"
          @confirm="deleteRoleHandle(deleteRoleIds)"
        >
          <template #reference>
            <el-button
              type="danger"
              :icon="menuStore.iconComponents.Delete"
              :disabled="
                !useButtonPermission([PERM.ROLE_DELETE], [() => !!deleteRoleIds.length]).value
              "
            >
              批量删除
            </el-button>
          </template>
        </AdaptiveConfirm>
      </div>

      <!-- desktop: table -->
      <VxeGrid
        v-if="!menuStore.isMobile"
        v-bind="roleGridConfig"
        @checkbox-change="tableSelectionChange"
        @checkbox-all="tableSelectionChange"
      >
        <template #column-type="{ row }: { row: RoleInfo }">
          <BaseTag :type="row.isBuiltin ? 'warning' : 'success'" :text="row.isBuiltin ? '内置' : '自定义'" />
        </template>
        <template #column-status="{ row }: { row: RoleInfo }">
          <BaseTag :type="row.status === RoleStatus.RoleStatus_Active ? 'success' : 'danger'" :text="row.status === RoleStatus.RoleStatus_Active ? '启用' : '停用'" />
        </template>
        <template #column-operation="{ row }: { row: RoleInfo }">
          <el-button v-if="!row.isBuiltin" type="primary" :icon="menuStore.iconComponents.Edit" link @click="openRoleEdit(row)" v-permission="[PERM.ROLE_EDIT]">编辑</el-button>
          <AdaptiveConfirm v-if="!row.isBuiltin" title="确定要删除选中的角色吗？" :placement="POPCONFIRM_CONFIG.placement" :width="POPCONFIRM_CONFIG.width" @confirm="deleteRoleHandle([row.roleId])">
            <template #reference>
              <el-button type="danger" :icon="menuStore.iconComponents.Delete" link v-permission="[PERM.ROLE_DELETE]">删除</el-button>
            </template>
          </AdaptiveConfirm>
        </template>
      </VxeGrid>

      <!-- mobile: cards -->
      <div v-else class="mobile-card-list">
        <div v-if="!roleList.length" class="mobile-empty"><el-empty description="暂无数据" /></div>
        <div v-for="row in roleList" :key="row.roleId" class="mobile-card" :class="{ 'is-selected': deleteRoleIds.includes(row.roleId) && !row.isBuiltin }">
          <div class="mobile-card-check" @click.stop>
            <el-checkbox v-if="!row.isBuiltin" :model-value="deleteRoleIds.includes(row.roleId)" size="small" @change="(v: boolean) => toggleMobileRoleSelect(row.roleId, v)" />
          </div>
          <div class="mobile-card-body">
            <div class="mobile-card-header">
              <span class="mobile-card-title">{{ row.name }}</span>
              <BaseTag :type="row.status === RoleStatus.RoleStatus_Active ? 'success' : 'danger'" :text="row.status === RoleStatus.RoleStatus_Active ? '启用' : '停用'" />
            </div>
            <div class="mobile-card-meta">{{ row.description || '-' }}</div>
            <div class="mobile-card-meta"><BaseTag :type="row.isBuiltin ? 'warning' : 'success'" :text="row.isBuiltin ? '内置' : '自定义'" /></div>
          </div>
          <div v-if="!row.isBuiltin" class="mobile-card-actions">
            <el-button size="small" plain type="primary" @click.stop="openRoleEdit(row)" v-permission="[PERM.ROLE_EDIT]">编辑</el-button>
            <AdaptiveConfirm title="确定要删除选中的角色吗？" :placement="POPCONFIRM_CONFIG.placement" :width="POPCONFIRM_CONFIG.width" @confirm="deleteRoleHandle([row.roleId])">
              <template #reference>
                <el-button size="small" plain type="danger" v-permission="[PERM.ROLE_DELETE]">删除</el-button>
              </template>
            </AdaptiveConfirm>
          </div>
        </div>
      </div>

      <TablePagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :is-mobile="menuStore.isMobile"
        @change="getRoleList"
      />
    </el-card>

    <RoleCreate ref="roleCreateRef" @refresh="refresh" />
  </div>
</template>

<script setup lang="ts">
import { deleteRole, rolePage } from '@/api/role'
import TablePagination from '@/components/pagination/TablePagination.vue'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { POPCONFIRM_CONFIG } from '@/config/elementConfig'
import { PERM } from '@/config/permission'
import { VxeGrid } from '@/plugins/vxeGrid'
import { RoleStatus, type RoleInfo } from '@/types/v1/system'
import RoleCreate from '@/views/system/role/create.vue'
import type { FormInstance } from 'element-plus'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'RoleView' })

const menuStore = useMenuStore()
const queryFormRef = useTemplateRef<FormInstance>('queryFormRef')
const roleCreateRef = useTemplateRef<InstanceType<typeof RoleCreate> | null>('roleCreateRef')

const deleteRoleIds = ref<string[]>([])
const roleList = ref<RoleInfo[]>([])

const queryForm = ref({
  name: '',
  status: undefined as RoleStatus | undefined,
})

const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0,
})

const roleGridConfig = computed<VxeGridProps>(() => ({
  border: true,
  showOverflow: true,
  rowConfig: { isHover: true },
  checkboxConfig: {
    highlight: true,
    checkMethod: ({ row }: { row: RoleInfo }) => !row.isBuiltin,
  },
  data: roleList.value,
  columns: [
    { type: 'checkbox', width: 55, fixed: 'left' },
    { type: 'seq', title: '序号', width: 55, fixed: 'left' },
    { field: 'name', title: '角色名称', minWidth: 160, fixed: 'left' },
    { field: 'description', title: '角色描述', minWidth: 200 },
    { field: 'isBuiltin', title: '类型', width: 110, slots: { default: 'column-type' } },
    { field: 'status', title: '状态', width: 110, slots: { default: 'column-status' } },
    { field: 'createTime', title: '创建时间', minWidth: 180 },
    { field: 'updateTime', title: '更新时间', minWidth: 180 },
    { title: '操作', width: 150, fixed: 'right', slots: { default: 'column-operation' } },
  ],
}))

const reset = () => {
  queryFormRef.value?.resetFields()
  getRoleList()
}

const getRoleList = async () => {
  const { data: res } = await rolePage({
    page: pagination.value.page,
    pageSize: pagination.value.pageSize,
    name: queryForm.value.name || undefined,
    status: queryForm.value.status,
  })

  roleList.value = res?.roles || []
  pagination.value.total = Number(res?.total || 0)
}

const tableSelectionChange = ({ records }: { records: RoleInfo[] }) => {
  deleteRoleIds.value = records.map((item) => item.roleId)
}

const toggleMobileRoleSelect = (roleId: string, checked: boolean) => {
  if (checked) {
    deleteRoleIds.value = [...deleteRoleIds.value, roleId]
  } else {
    deleteRoleIds.value = deleteRoleIds.value.filter((id) => id !== roleId)
  }
}

const openRoleEdit = (role: RoleInfo) => {
  if (role.isBuiltin) return
  roleCreateRef.value?.showDialog(role.roleId)
}

const deleteRoleHandle = async (ids: string[]) => {
  if (!ids.length) return

  const canDeleteIds = ids.filter((id) => {
    const role = roleList.value.find((item) => item.roleId === id)
    return !role?.isBuiltin
  })
  if (!canDeleteIds.length) return

  await deleteRole({ roleIds: canDeleteIds })

  deleteRoleIds.value = []
  ElMessage.success('删除成功')
  getRoleList()
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') {
    pagination.value.page = 1
  }
  getRoleList()
}

onMounted(() => {
  getRoleList()
})
</script>

<style lang="scss" scoped>
.mobile-card-list { display: flex; flex-direction: column; gap: 0.55rem; }
.mobile-empty { padding: 1.5rem 0; }
.mobile-card { position: relative; display: flex; align-items: flex-start; gap: 0.6rem; padding: 0.7rem 0.8rem; border: 1px solid var(--el-border-color-extra-light); border-radius: 0.6rem; background: var(--el-bg-color); }
.mobile-card.is-selected { border-color: var(--el-color-primary); background: color-mix(in srgb, var(--el-color-primary) 3%, var(--el-bg-color)); }
.mobile-card-check { flex-shrink: 0; padding-top: 0.1rem; min-width: 18px; }
.mobile-card-body { flex: 1; min-width: 0; }
.mobile-card-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.mobile-card-title { font-size: 0.88rem; font-weight: 700; color: var(--el-text-color-primary); }
.mobile-card-meta { margin-top: 0.25rem; font-size: 0.74rem; color: var(--el-text-color-secondary); }
.mobile-card-actions { display: flex; flex-direction: column; gap: 0.3rem; flex-shrink: 0; }
</style>
