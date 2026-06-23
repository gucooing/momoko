<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <el-form :model="queryForm" label-width="auto" ref="queryFormRef" @keyup.enter="getList">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item :label="t('ssh.common.name')" prop="keywords">
              <el-input v-model="queryForm.keywords" :placeholder="t('ssh.common.keywordPlaceholder')" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item :label="t('ssh.common.host')" prop="host">
              <el-input v-model="queryForm.host" :placeholder="t('ssh.common.hostPlaceholder')" />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="8" :xl="8">
            <el-form-item>
              <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="getList">
                {{ t('ssh.common.search') }}
              </el-button>
              <el-button :icon="menuStore.iconComponents.Refresh" @click="reset">{{ t('ssh.common.reset') }}</el-button>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <el-card shadow="never" class="card-mt-16">
      <div class="operation-container">
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" @click="openCreateDialog()">
          {{ t('ssh.common.addConnection') }}
        </el-button>
        <el-button
          :icon="menuStore.iconComponents['HOutline:SignalIcon']"
          :disabled="!deleteIds.length"
          :loading="batchTesting"
          @click="batchTest"
        >
          {{ t('ssh.common.batchTest') }}
        </el-button>
        <AdaptiveConfirm
          :title="t('ssh.common.confirmDeleteSelected')"
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
              {{ t('ssh.common.batchDelete') }}
            </el-button>
          </template>
        </AdaptiveConfirm>
      </div>

      <!-- desktop: table -->
      <VxeGrid
        v-if="!menuStore.isMobile"
        v-bind="gridConfig"
        @checkbox-change="selectionChange"
        @checkbox-all="selectionChange"
      >
        <template #column-authType="{ row }">
          <el-tag :type="row.authType === 'SSH_AUTH_TYPE_KEY' ? 'warning' : 'info'" size="small">
            {{ row.authType === 'SSH_AUTH_TYPE_KEY' ? t('ssh.common.key') : t('ssh.common.passwordAuth') }}
          </el-tag>
        </template>
        <template #column-status="{ row }">
          <el-tag :type="statusTagType(row.status)" size="small">
            {{ statusText(row.status) }}
          </el-tag>
        </template>
        <template #column-accessRole="{ row }">
          <el-tag :type="row.accessRole === 'SSH_HOST_ACCESS_ROLE_OWNER' ? 'primary' : 'info'" size="small">
            {{ row.accessRole === 'SSH_HOST_ACCESS_ROLE_OWNER' ? t('ssh.common.owner') : t('ssh.common.shared') }}
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
              {{ t('ssh.common.connect') }}
            </el-button>
            <el-button
              :icon="menuStore.iconComponents['HOutline:SignalIcon']"
              link
              @click="testConnect(row)"
            >
              {{ t('ssh.common.test') }}
            </el-button>
            <el-dropdown
              v-if="row.accessRole === 'SSH_HOST_ACCESS_ROLE_OWNER'"
              trigger="click"
            >
              <el-button :icon="menuStore.iconComponents['HOutline:EllipsisHorizontalIcon']" link>
                {{ t('ssh.common.more') }}
              </el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item :icon="menuStore.iconComponents.Edit" @click="openEditDialog(row)">
                    {{ t('ssh.common.edit') }}
                  </el-dropdown-item>
                  <el-dropdown-item :icon="menuStore.iconComponents['HOutline:ShareIcon']" @click="openShareDialog(row)">
                    {{ t('ssh.common.share') }}
                  </el-dropdown-item>
                  <el-dropdown-item
                    :icon="menuStore.iconComponents.Delete"
                    divided
                    @click="confirmDelete(row)"
                  >
                    {{ t('ssh.common.delete') }}
                  </el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </template>
      </VxeGrid>

      <!-- mobile: cards -->
      <div v-else v-loading="loading" class="mobile-card-list">
        <div v-if="!list.length" class="mobile-empty">
          <el-empty :description="t('ssh.common.noData')" />
        </div>
        <div v-for="row in list" :key="row.id" class="ssh-card" :class="{ 'is-selected': deleteIds.includes(row.id) }">
          <div class="ssh-card-check" @click.stop="toggleMobileSelect(row.id)">
            <el-checkbox :model-value="deleteIds.includes(row.id)" size="small" />
          </div>

          <div class="ssh-card-header">
            <span class="ssh-card-name">{{ row.name }}</span>
            <el-tag :type="statusTagType(row.status)" size="small">
              {{ statusText(row.status) }}
            </el-tag>
          </div>

          <div class="ssh-card-body">
            <div class="ssh-card-meta">
              <el-icon size="12"><component :is="menuStore.iconComponents['HOutline:GlobeAltIcon']" /></el-icon>
              <span>{{ row.host }}:{{ row.port }}</span>
            </div>
            <div class="ssh-card-meta">
              <el-icon size="12"><component :is="menuStore.iconComponents['Element:User']" /></el-icon>
              <span>{{ row.username }}</span>
            </div>
            <div class="ssh-card-tags">
              <el-tag :type="row.authType === 'SSH_AUTH_TYPE_KEY' ? 'warning' : 'info'" size="small">
                {{ row.authType === 'SSH_AUTH_TYPE_KEY' ? t('ssh.common.key') : t('ssh.common.passwordAuth') }}
              </el-tag>
              <el-tag :type="row.accessRole === 'SSH_HOST_ACCESS_ROLE_OWNER' ? 'primary' : 'info'" size="small">
                {{ row.accessRole === 'SSH_HOST_ACCESS_ROLE_OWNER' ? t('ssh.common.owner') : t('ssh.common.shared') }}
              </el-tag>
            </div>
            <div v-if="row.remark" class="ssh-card-remark">{{ row.remark }}</div>
            <div class="ssh-card-time">{{ row.createTime }}</div>
          </div>

          <div class="ssh-card-footer">
            <el-button size="small" plain type="primary" @click="connectSsh(row)">{{ t('ssh.common.connect') }}</el-button>
            <el-button size="small" plain @click="testConnect(row)">{{ t('ssh.common.test') }}</el-button>
            <el-dropdown v-if="row.accessRole === 'SSH_HOST_ACCESS_ROLE_OWNER'" trigger="click">
              <el-button size="small" plain>{{ t('ssh.common.more') }}</el-button>
              <template #dropdown>
                <el-dropdown-menu>
                  <el-dropdown-item @click="openEditDialog(row)">{{ t('ssh.common.edit') }}</el-dropdown-item>
                  <el-dropdown-item @click="openShareDialog(row)">{{ t('ssh.common.share') }}</el-dropdown-item>
                  <el-dropdown-item divided @click="confirmDelete(row)">{{ t('ssh.common.delete') }}</el-dropdown-item>
                </el-dropdown-menu>
              </template>
            </el-dropdown>
          </div>
        </div>
      </div>

      <!-- mobile: batch bar -->
      <div v-if="menuStore.isMobile && deleteIds.length" class="mobile-batch-bar">
        <span>{{ t('ssh.common.selectedCount', { count: deleteIds.length }) }}</span>
        <el-button size="small" :loading="batchTesting" @click="batchTest">{{ t('ssh.common.batchTest') }}</el-button>
        <el-button size="small" type="danger" @click="batchDelete">{{ t('ssh.common.batchDelete') }}</el-button>
      </div>

      <TablePagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :is-mobile="menuStore.isMobile"
        @change="getList"
      />
    </el-card>

    <SshConnectionCreate ref="createRef" @refresh="refresh" />

    <BaseDialog v-model="shareDialogOpen" :title="t('ssh.common.shareConnection')" width="520" @close="shareDialogOpen = false">
      <el-form label-width="100px" label-position="right">
        <el-form-item :label="t('ssh.common.currentShared')">
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
          <span v-else class="text-placeholder">{{ t('ssh.common.noSharedUsers') }}</span>
        </el-form-item>
        <el-form-item :label="t('ssh.common.addUser')">
          <el-select
            v-model="shareForm.newUserIds"
            multiple
            filterable
            remote
            reserve-keyword
            :placeholder="t('ssh.common.searchUser')"
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
        <el-button @click="shareDialogOpen = false">{{ t('ssh.common.cancel') }}</el-button>
        <el-button type="primary" :loading="shareLoading" @click="confirmShare">{{ t('ssh.common.confirm') }}</el-button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
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
const { t } = useI18n()
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
  return status === SSHHostStatus.SSH_HOST_STATUS_ONLINE ? t('ssh.common.online')
    : status === SSHHostStatus.SSH_HOST_STATUS_OFFLINE ? t('ssh.common.offline')
    : t('ssh.common.unchecked')
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
    { type: 'seq', title: t('ssh.common.serialNumber'), width: 55, fixed: 'left' },
    { field: 'name', title: t('ssh.common.name'), minWidth: 140, fixed: 'left' },
    { field: 'host', title: t('ssh.common.host'), minWidth: 140 },
    { field: 'port', title: t('ssh.common.port'), width: 90 },
    { field: 'username', title: t('ssh.common.username'), minWidth: 120 },
    { field: 'authType', title: t('ssh.common.authType'), width: 110, slots: { default: 'column-authType' } },
    { field: 'status', title: t('ssh.common.status'), width: 90, slots: { default: 'column-status' } },
    { field: 'accessRole', title: t('ssh.common.permission'), width: 100, slots: { default: 'column-accessRole' } },
    { field: 'remark', title: t('ssh.common.remark'), minWidth: 180 },
    { field: 'createTime', title: t('ssh.common.createTime'), minWidth: 180 },
    { title: t('ssh.common.operation'), width: 220, fixed: 'right', showOverflow: false, slots: { default: 'column-operation' } },
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

const toggleMobileSelect = (id: string) => {
  const idx = deleteIds.value.indexOf(id)
  if (idx === -1) {
    deleteIds.value = [...deleteIds.value, id]
  } else {
    deleteIds.value = deleteIds.value.filter((d) => d !== id)
  }
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
      ElMessage.success(t('ssh.common.connectionSuccess', { message: data.message || t('ssh.common.online') }))
    } else {
      ElMessage.warning(t('ssh.common.connectionFailedWithMessage', { message: data.message || t('ssh.common.offline') }))
    }
    getList()
  } catch {
    // error already handled by interceptor
  }
}

const confirmDelete = (row: SSHHostInfo) => {
  ElMessageBox.confirm(t('ssh.common.confirmDeleteContent'), t('ssh.common.confirmDeleteTitle'), {
    type: 'warning',
    confirmButtonText: t('ssh.common.delete'),
    cancelButtonText: t('ssh.common.cancel'),
  })
    .then(async () => {
      await deleteSshHost({ id: row.id })
      ElMessage.success(t('ssh.common.deleteSuccess'))
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
    ElMessage.success(t('ssh.common.batchTestDone'))
  } finally {
    batchTesting.value = false
  }
}

const batchDelete = async () => {
  if (!deleteIds.value.length) return
  try {
    await Promise.all(deleteIds.value.map((id) => deleteSshHost({ id })))
    ElMessage.success(t('ssh.common.batchDeleteSuccess'))
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
    ElMessage.success(finalUserIds.length ? t('ssh.common.shareSuccess') : t('ssh.common.shareAllCancelled'))
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

<style scoped lang="scss">
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

/* ===== mobile ===== */
.mobile-card-list {
  display: flex;
  flex-direction: column;
  gap: 0.65rem;
}

.mobile-empty {
  padding: 1.5rem 0;
}

.ssh-card {
  position: relative;
  padding: 0.75rem 0.85rem;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.65rem;
  background: var(--el-bg-color);
  transition: border-color 0.15s;
}

.ssh-card.is-selected {
  border-color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 4%, var(--el-bg-color));
}

.ssh-card-check {
  position: absolute;
  top: 0.6rem;
  right: 0.7rem;
  z-index: 1;
}

.ssh-card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding-right: 2rem;
}

.ssh-card-name {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ssh-card-body {
  margin-top: 0.55rem;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.ssh-card-meta {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.76rem;
  color: var(--el-text-color-secondary);
}

.ssh-card-tags {
  display: flex;
  gap: 0.4rem;
  margin-top: 0.15rem;
}

.ssh-card-remark {
  font-size: 0.74rem;
  color: var(--el-text-color-placeholder);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ssh-card-time {
  font-size: 0.7rem;
  color: var(--el-text-color-placeholder);
}

.ssh-card-footer {
  display: flex;
  gap: 0.4rem;
  margin-top: 0.6rem;
  padding-top: 0.5rem;
  border-top: 1px solid var(--el-border-color-extra-light);
}

/* ===== mobile batch bar ===== */
.mobile-batch-bar {
  position: sticky;
  bottom: 0;
  display: flex;
  align-items: center;
  gap: 0.5rem;
  margin: 0.65rem -0.5rem -0.5rem;
  padding: 0.6rem 0.75rem;
  background: var(--el-bg-color);
  border-top: 1px solid var(--el-border-color);
  font-size: 0.8rem;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
</style>
