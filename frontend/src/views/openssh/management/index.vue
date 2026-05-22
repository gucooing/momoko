<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <el-form :model="queryForm" label-width="auto" ref="queryFormRef" @keyup.enter="getList">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item label="名称" prop="keywords">
              <el-input v-model="queryForm.keywords" placeholder="搜索名称/标签/主机地址" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item label="主机地址" prop="host">
              <el-input v-model="queryForm.host" placeholder="请输入主机地址" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item>
              <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="getList">
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
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" @click="openCreateDialog()">
          新增连接
        </el-button>
        <el-button
          :icon="menuStore.iconComponents['HOutline:SignalIcon']"
          :disabled="!deleteIds.length"
          :loading="batchTesting"
          @click="batchTest"
        >
          批量测试
        </el-button>
        <AdaptiveConfirm
          title="确定要删除选中的连接吗？"
          :placement="POPCONFIRM_CONFIG.placement"
          :width="POPCONFIRM_CONFIG.width"
          @confirm="batchDelete"
        >
          <template #reference>
            <el-button
              type="danger"
              :icon="menuStore.iconComponents.Delete"
              :disabled="!deleteIds.length"
            >
              批量删除
            </el-button>
          </template>
        </AdaptiveConfirm>
      </div>

      <VxeGrid
        v-bind="gridConfig"
        @checkbox-change="selectionChange"
        @checkbox-all="selectionChange"
      >
        <template #column-authType="{ row }">
          <el-tag :type="row.authType === 'SSH_AUTH_TYPE_KEY' ? 'warning' : 'info'" size="small">
            {{ row.authType === 'SSH_AUTH_TYPE_KEY' ? '密钥' : '密码' }}
          </el-tag>
        </template>

        <template #column-status="{ row }">
          <el-tag :type="statusTagType(row.status)" size="small">
            {{ statusText(row.status) }}
          </el-tag>
        </template>

        <template #column-accessRole="{ row }">
          <el-tag :type="row.accessRole === 'SSH_HOST_ACCESS_ROLE_OWNER' ? 'primary' : 'info'" size="small">
            {{ row.accessRole === 'SSH_HOST_ACCESS_ROLE_OWNER' ? '创建者' : '被分享' }}
          </el-tag>
        </template>

        <template #column-operation="{ row }">
          <div class="ssh-actions">
            <el-button
              type="primary"
              :icon="menuStore.iconComponents['HOutline:GlobeAltIcon']"
              link
              @click="connectSsh(row)"
            >
              连接
            </el-button>
            <el-button
              :icon="menuStore.iconComponents['HOutline:SignalIcon']"
              link
              @click="testConnect(row)"
            >
              测试
            </el-button>
            <el-dropdown
              v-if="row.accessRole === 'SSH_HOST_ACCESS_ROLE_OWNER'"
              trigger="click"
            >
              <el-button :icon="menuStore.iconComponents['HOutline:EllipsisHorizontalIcon']" link>
                更多
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item :icon="menuStore.iconComponents.Edit" @click="openEditDialog(row)">
                    编辑
                  </el-dropdown-item>
                  <el-dropdown-item :icon="menuStore.iconComponents['HOutline:ShareIcon']" @click="openShareDialog(row)">
                    分享
                  </el-dropdown-item>
                  <el-dropdown-item
                    :icon="menuStore.iconComponents.Delete"
                    divided
                    @click="confirmDelete(row)"
                  >
                    删除
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </template>
      </VxeGrid>

      <TablePagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :is-mobile="menuStore.isMobile"
        @change="getList"
      />
    </el-card>

    <SshConnectionCreate ref="createRef" @refresh="refresh" />

    <BaseDialog v-model="shareDialogOpen" title="分享连接" width="520" @close="shareDialogOpen = false">
      <el-form label-width="100px" label-position="right">
        <el-form-item label="当前已分享">
          <div v-if="shareForm.sharedUsers.length" class="share-user-tags">
            <el-tag
              v-for="user in shareForm.sharedUsers"
              :key="user.userId"
              closable
              size="small"
              @close="removeSharedUser(user.userId)"
            >
              {{ user.name || user.userId }}
            </el-tag>
          </div>
          <span v-else class="text-placeholder">暂无分享用户</span>
        </el-form-item>
        <el-form-item label="添加用户">
          <el-select
            v-model="shareForm.newUserIds"
            multiple
            filterable
            remote
            reserve-keyword
            placeholder="搜索用户"
            :remote-method="searchUsers"
            :loading="userSearchLoading"
            style="width: 100%"
          >
            <el-option
              v-for="user in userOptions"
              :key="user.userId"
              :label="`${user.name || user.username} (${user.username})`"
              :value="user.userId"
            />
          </el-select>
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="shareDialogOpen = false">取消</el-button>
        <el-button type="primary" :loading="shareLoading" @click="confirmShare">确定</el-button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import TablePagination from '@/components/pagination/TablePagination.vue'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import { POPCONFIRM_CONFIG } from '@/config/elementConfig'
import { VxeGrid } from '@/plugins/vxeGrid'
import SshConnectionCreate from '@/views/openssh/management/create.vue'
import { getSshHosts, deleteSshHost, shareSshHost, testSshHost, batchTestSshHosts } from '@/api/openssh'
import { userPage } from '@/api/user'
import { SSHHostStatus, type SSHHostInfo } from '@/types/v1/openssh'
import type { UserInfo } from '@/types/v1/user'
import type { FormInstance } from 'element-plus'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'SshManagementView' })

const router = useRouter()
const menuStore = useMenuStore()
const queryFormRef = useTemplateRef<FormInstance>('queryFormRef')
const createRef = useTemplateRef<InstanceType<typeof SshConnectionCreate> | null>('createRef')

const deleteIds = ref<string[]>([])
const list = ref<SSHHostInfo[]>([])
const loading = ref(false)

const queryForm = ref({
  keywords: '',
  host: '',
})

const pagination = ref({
  page: 1,
  pageSize: 10,
  total: 0,
})

const statusTagType = (status: SSHHostStatus) => {
  return status === SSHHostStatus.SSH_HOST_STATUS_ONLINE ? 'success'
    : status === SSHHostStatus.SSH_HOST_STATUS_OFFLINE ? 'danger'
    : 'info'
}

const statusText = (status: SSHHostStatus) => {
  return status === SSHHostStatus.SSH_HOST_STATUS_ONLINE ? '在线'
    : status === SSHHostStatus.SSH_HOST_STATUS_OFFLINE ? '离线'
    : '未检测'
}

const gridConfig = computed<VxeGridProps>(() => ({
  border: true,
  showOverflow: true,
  rowConfig: { isHover: true },
  checkboxConfig: { highlight: true },
  loading: loading.value,
  data: list.value,
  columns: [
    { type: 'checkbox', width: 55, fixed: 'left' },
    { type: 'seq', title: '序号', width: 55, fixed: 'left' },
    { field: 'name', title: '名称', minWidth: 140, fixed: 'left' },
    { field: 'host', title: '主机地址', minWidth: 140 },
    { field: 'port', title: '端口', width: 90 },
    { field: 'username', title: '用户名', minWidth: 120 },
    { field: 'authType', title: '认证方式', width: 110, slots: { default: 'column-authType' } },
    { field: 'status', title: '状态', width: 90, slots: { default: 'column-status' } },
    { field: 'accessRole', title: '权限', width: 100, slots: { default: 'column-accessRole' } },
    { field: 'remark', title: '备注', minWidth: 180 },
    { field: 'createTime', title: '创建时间', minWidth: 180 },
    { title: '操作', width: 220, fixed: 'right', showOverflow: false, slots: { default: 'column-operation' } },
  ],
}))

const reset = () => {
  queryFormRef.value?.resetFields()
  pagination.value.page = 1
  getList()
}

const getList = async () => {
  loading.value = true
  try {
    const { data } = await getSshHosts({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.value.keywords || undefined,
      host: queryForm.value.host || undefined,
    })
    list.value = data?.infos || []
    pagination.value.total = Number(data?.total) || 0
  } finally {
    loading.value = false
  }
}

const selectionChange = ({ records }: { records: SSHHostInfo[] }) => {
  deleteIds.value = records.map((item) => item.id)
}

const openCreateDialog = () => {
  createRef.value?.showDialog()
}

const openEditDialog = (row: SSHHostInfo) => {
  createRef.value?.showDialog(row)
}

const connectSsh = (row: SSHHostInfo) => {
  router.push({ path: '/openssh/terminal', query: { id: row.id } })
}

const testConnect = async (row: SSHHostInfo) => {
  try {
    const { data } = await testSshHost({ id: row.id })
    if (data?.status === SSHHostStatus.SSH_HOST_STATUS_ONLINE) {
      ElMessage.success(`连接成功: ${data.message || '在线'}`)
    } else {
      ElMessage.warning(`连接失败: ${data.message || '离线'}`)
    }
    getList()
  } catch {
    // error already handled by interceptor
  }
}

const confirmDelete = (row: SSHHostInfo) => {
  ElMessageBox.confirm('确定要删除该连接吗？', '确认删除', {
    type: 'warning',
    confirmButtonText: '删除',
    cancelButtonText: '取消',
  })
    .then(async () => {
      await deleteSshHost({ id: row.id })
      ElMessage.success('删除成功')
      deleteIds.value = deleteIds.value.filter((d) => d !== row.id)
      getList()
    })
    .catch(() => { /* cancelled */ })
}

const batchTesting = ref(false)

const batchTest = async () => {
  if (!deleteIds.value.length) return
  batchTesting.value = true
  try {
    const { data } = await batchTestSshHosts({ ids: deleteIds.value })
    if (data?.results) {
      for (const result of data.results) {
        const row = list.value.find((item) => item.id === result.id)
        if (row) {
          row.status = result.status
        }
      }
    }
    ElMessage.success('批量测试完成')
  } finally {
    batchTesting.value = false
  }
}

const batchDelete = async () => {
  if (!deleteIds.value.length) return
  try {
    await Promise.all(deleteIds.value.map((id) => deleteSshHost({ id })))
    ElMessage.success('批量删除成功')
    deleteIds.value = []
    getList()
  } catch {
    // error handled by interceptor
  }
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') {
    pagination.value.page = 1
  }
  getList()
}

// ---- share dialog ----
const shareDialogOpen = ref(false)
const shareLoading = ref(false)
const shareForm = ref({
  hostId: '',
  sharedUsers: [] as { userId: string; name: string }[],
  newUserIds: [] as string[],
})
const userOptions = ref<UserInfo[]>([])
const userSearchLoading = ref(false)

const openShareDialog = async (row: SSHHostInfo) => {
  shareForm.value = {
    hostId: row.id,
    sharedUsers: row.sharedUsers.map((user) => ({ userId: user.userId, name: user.name })),
    newUserIds: [],
  }
  userOptions.value = []
  shareDialogOpen.value = true
}

const removeSharedUser = (userId: string) => {
  shareForm.value.sharedUsers = shareForm.value.sharedUsers.filter((user) => user.userId !== userId)
}

const searchUsers = async (query: string) => {
  if (!query) {
    userOptions.value = []
    return
  }
  userSearchLoading.value = true
  try {
    const { data } = await userPage({ page: 1, pageSize: 20, username: query })
    userOptions.value = data?.users || []
  } finally {
    userSearchLoading.value = false
  }
}

const confirmShare = async () => {
  const finalUserIds = [
    ...shareForm.value.sharedUsers.map((user) => user.userId),
    ...shareForm.value.newUserIds,
  ].filter((id, index, arr) => arr.indexOf(id) === index)

  shareLoading.value = true
  try {
    await shareSshHost({ id: shareForm.value.hostId, userIds: finalUserIds })
    ElMessage.success(finalUserIds.length ? '分享成功' : '已取消全部分享')
    shareDialogOpen.value = false
    getList()
  } finally {
    shareLoading.value = false
  }
}

onMounted(() => {
  getList()
})
</script>

<style scoped>
.ssh-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.ssh-actions :deep(.el-button) {
  margin-left: 0;
}

.share-user-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.text-placeholder {
  color: var(--el-text-color-placeholder);
  font-size: 0.88rem;
}
</style>
