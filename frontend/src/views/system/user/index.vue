<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <el-form :model="queryForm" label-width="auto" ref="queryFormRef" @keyup.enter="getUserList">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item label="用户名" prop="username">
              <el-input v-model="queryForm.username" placeholder="请输入用户名" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item label="状态" prop="status">
              <el-select v-model="queryForm.status" placeholder="请选择">
                <el-option label="启用" :value="UserStatus.Active" />
                <el-option label="停用" :value="UserStatus.InActive" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item>
              <el-button
                type="primary"
                :icon="menuStore.iconComponents.Search"
                @click="getUserList"
              >
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
          @click="userCreateRef?.showDialog()"
          v-permission="[PERM.USER_ADD]"
        >
          新增用户
        </el-button>
        <AdaptiveConfirm
          title="确定要删除选中的用户吗？"
          :placement="POPCONFIRM_CONFIG.placement"
          :width="POPCONFIRM_CONFIG.width"
          @confirm="deleteUserHandle(deleteUserIds)"
        >
          <template #reference>
            <el-button
              type="danger"
              :icon="menuStore.iconComponents.Delete"
              :disabled="
                !useButtonPermission([PERM.USER_DELETE], [() => !!deleteUserIds.length]).value
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
        v-bind="userGridConfig"
        @checkbox-change="tableSelectionChange"
        @checkbox-all="tableSelectionChange"
      >
        <template #column-role="{ row }: { row: UserInfo }">
          <BaseTag v-if="row.roleName" :text="row.roleName" />
          <span v-else>-</span>
        </template>
        <template #column-status="{ row }: { row: UserInfo }">
          <BaseTag :type="getStatusTagType(row.status)" :text="getStatusLabel(row.status)" />
        </template>
        <template #column-operation="{ row }: { row: UserInfo }">
          <el-button type="primary" :icon="menuStore.iconComponents.Edit" link @click="openUserEdit(row)" v-permission="[PERM.USER_EDIT]">编辑</el-button>
          <AdaptiveConfirm title="确定要删除选中的用户吗？" :placement="POPCONFIRM_CONFIG.placement" :width="POPCONFIRM_CONFIG.width" @confirm="deleteUserHandle([row.userId])">
            <template #reference>
              <el-button type="danger" :icon="menuStore.iconComponents.Delete" link v-permission="[PERM.USER_DELETE]">删除</el-button>
            </template>
          </AdaptiveConfirm>
        </template>
      </VxeGrid>

      <!-- mobile: cards -->
      <div v-else class="mobile-card-list">
        <div v-if="!userList.length" class="mobile-empty"><el-empty description="暂无数据" /></div>
        <div v-for="row in userList" :key="row.userId" class="mobile-card" :class="{ 'is-selected': deleteUserIds.includes(row.userId) }">
          <div class="mobile-card-check" @click.stop>
            <el-checkbox :model-value="deleteUserIds.includes(row.userId)" size="small" @change="(v) => toggleMobileUserSelect(row.userId, v as boolean)" />
          </div>
          <div class="mobile-card-body">
            <div class="mobile-card-header">
              <span class="mobile-card-title">{{ row.username }}</span>
              <BaseTag :type="getStatusTagType(row.status)" :text="getStatusLabel(row.status)" />
            </div>
            <div class="mobile-card-meta">
              <span>{{ row.name || '-' }}</span>
              <span class="meta-sep">·</span>
              <span>{{ row.email || '-' }}</span>
            </div>
            <div class="mobile-card-meta">
              <BaseTag v-if="row.roleName" :text="row.roleName" />
              <span v-else class="text-muted">无角色</span>
            </div>
          </div>
          <div class="mobile-card-actions">
            <el-button size="small" plain type="primary" @click.stop="openUserEdit(row)" v-permission="[PERM.USER_EDIT]">编辑</el-button>
            <AdaptiveConfirm title="确定要删除选中的用户吗？" :placement="POPCONFIRM_CONFIG.placement" :width="POPCONFIRM_CONFIG.width" @confirm="deleteUserHandle([row.userId])">
              <template #reference>
                <el-button size="small" plain type="danger" v-permission="[PERM.USER_DELETE]">删除</el-button>
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
        @change="getUserList"
      />
    </el-card>

    <UserCreate ref="userCreateRef" @refresh="refresh" />
  </div>
</template>

<script setup lang="ts">
import { deleteUser, userPage } from '@/api/user'
import TablePagination from '@/components/pagination/TablePagination.vue'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { POPCONFIRM_CONFIG } from '@/config/elementConfig'
import { PERM } from '@/config/permission'
import { VxeGrid } from '@/plugins/vxeGrid'
import { UserStatus, type UserInfo } from '@/types/v1/user'
import UserCreate from '@/views/system/user/create.vue'
import type { FormInstance } from 'element-plus'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'UserView' })

const menuStore = useMenuStore()
const queryFormRef = useTemplateRef<FormInstance>('queryFormRef')
const userCreateRef = useTemplateRef<InstanceType<typeof UserCreate> | null>('userCreateRef')

const deleteUserIds = ref<string[]>([])
const userList = ref<UserInfo[]>([])

const queryForm = ref({
  username: '',
  status: undefined as UserStatus | undefined,
})

const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0,
})

const getStatusLabel = (status: UserStatus): string => {
  if (status === UserStatus.Active) return '启用'
  if (status === UserStatus.InActive) return '停用'
  return status
}

const getStatusTagType = (status: UserStatus): 'success' | 'danger' => {
  if (status === UserStatus.Active) return 'success'
  return 'danger'
}

const userGridConfig = computed<VxeGridProps>(() => ({
  border: true,
  showOverflow: true,
  rowConfig: { isHover: true },
  checkboxConfig: { highlight: true },
  data: userList.value,
  columns: [
    { type: 'checkbox', width: 55, fixed: 'left' },
    { type: 'seq', title: '序号', width: 55, fixed: 'left' },
    { field: 'username', title: '用户名', minWidth: 160, fixed: 'left' },
    { field: 'name', title: '姓名', minWidth: 140 },
    { field: 'email', title: '邮箱', minWidth: 180 },
    { field: 'roleName', title: '角色', minWidth: 150, slots: { default: 'column-role' } },
    { field: 'status', title: '状态', width: 110, slots: { default: 'column-status' } },
    { field: 'createTime', title: '创建时间', minWidth: 180 },
    { field: 'updateTime', title: '更新时间', minWidth: 180 },
    { title: '操作', width: 150, fixed: 'right', slots: { default: 'column-operation' } },
  ],
}))

const reset = () => {
  queryFormRef.value?.resetFields()
  pagination.value.page = 1
  getUserList()
}

const getUserList = async () => {
  const { data: res } = await userPage({
    page: pagination.value.page,
    pageSize: pagination.value.pageSize,
    username: queryForm.value.username || undefined,
    status: queryForm.value.status,
  })

  userList.value = res?.users || []
  pagination.value.total = Number(res?.total || 0)
  pagination.value.page = Number(res?.page || pagination.value.page)
  pagination.value.pageSize = Number(res?.pageSize || pagination.value.pageSize)
}

const tableSelectionChange = ({ records }: { records: UserInfo[] }) => {
  deleteUserIds.value = records.map((item) => item.userId)
}

const toggleMobileUserSelect = (userId: string, checked: boolean) => {
  if (checked) {
    deleteUserIds.value = [...deleteUserIds.value, userId]
  } else {
    deleteUserIds.value = deleteUserIds.value.filter((id) => id !== userId)
  }
}

const openUserEdit = (row: UserInfo) => {
  userCreateRef.value?.showDialog({
    userId: row.userId,
    roleName: row.roleName,
  })
}

const deleteUserHandle = async (ids: string[]) => {
  if (!ids.length) return
  await deleteUser({ userIds: ids })
  deleteUserIds.value = []
  ElMessage.success('删除成功')
  getUserList()
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') {
    pagination.value.page = 1
  }
  getUserList()
}

onMounted(() => {
  getUserList()
})
</script>

<style lang="scss" scoped>
.mobile-card-list { display: flex; flex-direction: column; gap: 0.55rem; }
.mobile-empty { padding: 1.5rem 0; }
.mobile-card {
  position: relative; display: flex; align-items: flex-start; gap: 0.6rem;
  padding: 0.7rem 0.8rem; border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.6rem; background: var(--el-bg-color);
}
.mobile-card.is-selected { border-color: var(--el-color-primary); background: color-mix(in srgb, var(--el-color-primary) 3%, var(--el-bg-color)); }
.mobile-card-check { flex-shrink: 0; padding-top: 0.1rem; }
.mobile-card-body { flex: 1; min-width: 0; }
.mobile-card-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.mobile-card-title { font-size: 0.88rem; font-weight: 700; color: var(--el-text-color-primary); }
.mobile-card-meta { display: flex; align-items: center; gap: 0.3rem; margin-top: 0.25rem; font-size: 0.74rem; color: var(--el-text-color-secondary); }
.meta-sep { color: var(--el-text-color-placeholder); }
.text-muted { color: var(--el-text-color-placeholder); }
.mobile-card-actions { display: flex; flex-direction: column; gap: 0.3rem; flex-shrink: 0; }
</style>
