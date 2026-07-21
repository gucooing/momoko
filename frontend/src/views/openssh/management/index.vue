<!-- SSH 管理（重写 · P1 列表）：PageHeader + FilterBar(名称/主机) + 批量条(测试/删除) + DataTable(可选) / 移动卡 + Pagination。
     行内 连接/测试（全部）+ 编辑/分享/删除（仅创建者）走 ActionMenu；分享走 FormDialog + UserPicker。
     保留 getSshHosts/delete/share/test/batchTest 契约。 -->
<template>
  <div class="ssh-page">
    <PageHeader :title="t('ssh.common.title')" :description="t('ssh.common.pageDesc')">
      <template #actions>
        <UButton color="neutral" variant="soft" icon="i-lucide-rotate-cw" @click="getList">
          {{ t('ssh.common.reset') }}
        </UButton>
        <UButton color="primary" icon="i-lucide-plus" @click="openCreateDialog()">
          {{ t('ssh.common.addConnection') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="reload" @reset="resetFilter">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('ssh.common.name') }}</label>
          <input
            v-model="queryForm.keywords"
            class="app-input"
            :placeholder="t('ssh.common.keywordPlaceholder')"
            @keyup.enter="reload"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('ssh.common.host') }}</label>
          <input
            v-model="queryForm.host"
            class="app-input"
            :placeholder="t('ssh.common.hostPlaceholder')"
            @keyup.enter="reload"
          />
        </div>
      </template>
    </FilterBar>

    <!-- 批量条 -->
    <div v-if="selectedIds.length" class="ssh-page__batch">
      <span class="ssh-page__batch-count">{{ t('ssh.common.selectedCount', { count: selectedIds.length }) }}</span>
      <div class="ssh-page__batch-actions">
        <UButton
          color="primary"
          variant="soft"
          size="sm"
          icon="i-lucide-activity"
          :loading="batchTesting"
          @click="batchTest"
        >
          {{ t('ssh.common.batchTest') }}
        </UButton>
        <UButton color="error" variant="soft" size="sm" icon="i-lucide-trash-2" @click="batchDelete">
          {{ t('ssh.common.batchDelete') }}
        </UButton>
        <UButton color="neutral" variant="ghost" size="sm" @click="selectedIds = []">
          {{ t('ssh.common.clearSelection') }}
        </UButton>
      </div>
    </div>

    <div class="ssh-page__body">
      <div class="ssh-page__bar">
        <span class="ssh-page__hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
        <button
          v-if="menuStore.isMobile && list.length"
          type="button"
          class="ssh-page__selall"
          @click="toggleSelectAll"
        >
          {{ isAllSelected ? t('ssh.common.clearSelection') : t('ssh.common.selectAll') }}
        </button>
      </div>

      <!-- 桌面：表格 -->
      <DataTable
        v-if="!menuStore.isMobile"
        v-model="selectedIds"
        :columns="columns"
        :rows="list"
        row-key="id"
        selectable
        :loading="loading"
        :empty-text="t('ssh.common.noData')"
      >
        <template #cell-host="{ row }">{{ row.host }}:{{ row.port }}</template>
        <template #cell-authType="{ row }">
          <StatusPill
            :variant="row.authType === 'SSH_AUTH_TYPE_KEY' ? 'warning' : 'info'"
            :label="row.authType === 'SSH_AUTH_TYPE_KEY' ? t('ssh.common.key') : t('ssh.common.passwordAuth')"
          />
        </template>
        <template #cell-status="{ row }">
          <StatusPill :variant="statusVariant(row.status)" :label="statusText(row.status)" />
        </template>
        <template #cell-accessRole="{ row }">
          <StatusPill
            :variant="isOwner(row) ? 'primary' : 'info'"
            :label="isOwner(row) ? t('ssh.common.owner') : t('ssh.common.shared')"
          />
        </template>
        <template #cell-createTime="{ row }">{{ formatTime(row.createTime) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="rowActionsFor(row)" @select="(key) => onRowAction(key, row)" />
        </template>
      </DataTable>

      <!-- 移动：卡片 -->
      <template v-else>
        <div v-if="loading" class="ssh-cards">
          <div v-for="i in 4" :key="i" class="ssh-skeleton" />
        </div>
        <EmptyState
          v-else-if="!list.length"
          icon="HOutline:CommandLineIcon"
          :title="t('ssh.common.noData')"
          :description="t('ssh.common.emptyDesc')"
        />
        <div v-else class="ssh-cards">
          <EntityCard v-for="row in list" :key="row.id">
            <template #title>
              <label class="ssh-card__title">
                <input
                  type="checkbox"
                  class="ssh-card__check"
                  :checked="selectedIdSet.has(row.id)"
                  @change="toggleSelect(row.id)"
                />
                <span class="ssh-card__name">{{ row.name }}</span>
              </label>
            </template>
            <template #status>
              <StatusPill :variant="statusVariant(row.status)" :label="statusText(row.status)" />
            </template>
            <template #meta>
              <span class="ssh-card__full">{{ row.host }}:{{ row.port }} · {{ row.username }}</span>
              <StatusPill
                :variant="row.authType === 'SSH_AUTH_TYPE_KEY' ? 'warning' : 'info'"
                :label="row.authType === 'SSH_AUTH_TYPE_KEY' ? t('ssh.common.key') : t('ssh.common.passwordAuth')"
              />
              <StatusPill
                :variant="isOwner(row) ? 'primary' : 'info'"
                :label="isOwner(row) ? t('ssh.common.owner') : t('ssh.common.shared')"
              />
            </template>
            <template #footer>
              <span>{{ formatTime(row.createTime) }}</span>
              <ActionMenu :items="rowActionsFor(row)" @select="(key) => onRowAction(key, row)" />
            </template>
          </EntityCard>
        </div>
      </template>

      <Pagination
        v-model:page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        @change="getList"
      />
    </div>

    <SshConnectionCreate ref="createRef" @refresh="refresh" />

    <!-- 分享连接 -->
    <FormDialog
      v-model="shareDialogOpen"
      :title="t('ssh.common.shareConnection')"
      :width="520"
      :loading="shareLoading"
      @confirm="confirmShare"
    >
      <div class="app-field">
        <label class="app-label">{{ t('ssh.common.sharedUsers') }}</label>
        <UserPicker v-model="shareUserIds" :placeholder="t('ssh.common.shareUsersPlaceholder')" />
      </div>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import SshConnectionCreate from '@/views/openssh/management/create.vue'
import { getSshHosts, deleteSshHost, shareSshHost, testSshHost, batchTestSshHosts } from '@/api/openssh'
import { SSHHostStatus, type SSHHostInfo } from '@/types/v1/openssh'
import { Dialog } from '@/utils/dialog'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

defineOptions({ name: 'SshManagementView' })

const router = useRouter()
const menuStore = useMenuStore()
const { t } = useI18n()
const createRef = useTemplateRef<InstanceType<typeof SshConnectionCreate> | null>('createRef')

const OWNER_ROLE = 'SSH_HOST_ACCESS_ROLE_OWNER'

const loading = ref(false)
const list = ref<SSHHostInfo[]>([])
const selectedIds = ref<string[]>([])
const selectedIdSet = computed(() => new Set(selectedIds.value))
const batchTesting = ref(false)

const queryForm = ref({ keywords: '', host: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const isOwner = (row: Record<string, unknown>) => row.accessRole === OWNER_ROLE

const formatTime = (value: unknown): string => {
  if (!value) return '—'
  const d = new Date(value as string | Date)
  if (Number.isNaN(d.getTime())) return String(value)
  const pad = (n: number) => String(n).padStart(2, '0')
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

const statusVariant = (status: unknown) =>
  status === SSHHostStatus.SSH_HOST_STATUS_ONLINE ? 'success'
    : status === SSHHostStatus.SSH_HOST_STATUS_OFFLINE ? 'error'
    : 'neutral'
const statusText = (status: unknown) =>
  status === SSHHostStatus.SSH_HOST_STATUS_ONLINE ? t('ssh.common.online')
    : status === SSHHostStatus.SSH_HOST_STATUS_OFFLINE ? t('ssh.common.offline')
    : t('ssh.common.unchecked')

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('ssh.common.name'), minWidth: 140 },
  { key: 'host', title: t('ssh.common.host'), minWidth: 170 },
  { key: 'username', title: t('ssh.common.username'), minWidth: 110 },
  { key: 'authType', title: t('ssh.common.authType'), width: 100 },
  { key: 'status', title: t('ssh.common.status'), width: 90 },
  { key: 'accessRole', title: t('ssh.common.permission'), width: 100 },
  { key: 'remark', title: t('ssh.common.remark'), minWidth: 150 },
  { key: 'createTime', title: t('ssh.common.createTime'), width: 170 },
  { key: 'operation', title: t('ssh.common.operation'), width: 80, align: 'center' },
])

const rowActionsFor = (row: Record<string, unknown>): ActionMenuItem[] => {
  const owner = isOwner(row)
  return [
    { key: 'connect', label: t('ssh.common.connect'), icon: 'HOutline:GlobeAltIcon' },
    { key: 'test', label: t('ssh.common.test'), icon: 'HOutline:SignalIcon' },
    { key: 'edit', label: t('ssh.common.edit'), icon: 'HOutline:PencilSquareIcon', hidden: !owner },
    { key: 'share', label: t('ssh.common.share'), icon: 'HOutline:ShareIcon', hidden: !owner },
    { key: 'delete', label: t('ssh.common.delete'), icon: 'HOutline:TrashIcon', danger: true, hidden: !owner },
  ]
}

const findRow = (id: string) => list.value.find((x) => x.id === id)

const onRowAction = (key: string, row: Record<string, unknown>) => {
  const record = findRow(String(row.id))
  if (!record) return
  if (key === 'connect') connectSsh(record)
  else if (key === 'test') testConnect(record)
  else if (key === 'edit') createRef.value?.showDialog(record)
  else if (key === 'share') openShareDialog(record)
  else if (key === 'delete') confirmDelete(record)
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

const reload = () => {
  pagination.value.page = 1
  getList()
}
const resetFilter = () => {
  queryForm.value = { keywords: '', host: '' }
  pagination.value.page = 1
  getList()
}

// —— 移动端选择 ——
const isAllSelected = computed(() => list.value.length > 0 && list.value.every((r) => selectedIdSet.value.has(r.id)))
const toggleSelect = (id: string) => {
  selectedIds.value = selectedIdSet.value.has(id)
    ? selectedIds.value.filter((x) => x !== id)
    : [...selectedIds.value, id]
}
const toggleSelectAll = () => {
  selectedIds.value = isAllSelected.value ? [] : list.value.map((r) => r.id)
}

const openCreateDialog = () => createRef.value?.showDialog()

const connectSsh = (row: SSHHostInfo) => {
  router.push({ path: '/openssh/terminal', query: { id: row.id } })
}

const testConnect = async (row: SSHHostInfo) => {
  const { data } = await testSshHost({ id: row.id })
  if (data?.status === SSHHostStatus.SSH_HOST_STATUS_ONLINE) {
    feedback.success(t('ssh.common.connectionSuccess', { message: data.message || t('ssh.common.online') }))
  } else {
    feedback.warning(t('ssh.common.connectionFailedWithMessage', { message: data?.message || t('ssh.common.offline') }))
  }
  getList()
}

const confirmDelete = (row: SSHHostInfo) => {
  Dialog.confirm({
    title: t('ssh.common.confirmDeleteTitle'),
    content: t('ssh.common.confirmDeleteContent'),
    confirmText: t('ssh.common.delete'),
    cancelText: t('ssh.common.cancel'),
    onConfirm: async () => {
      await deleteSshHost({ id: row.id })
      feedback.success(t('ssh.common.deleteSuccess'))
      selectedIds.value = selectedIds.value.filter((d) => d !== row.id)
      getList()
    },
  })
}

const batchTest = async () => {
  if (!selectedIds.value.length) return
  batchTesting.value = true
  try {
    const { data } = await batchTestSshHosts({ ids: selectedIds.value })
    if (data?.results) {
      for (const result of data.results) {
        const row = list.value.find((item) => item.id === result.id)
        if (row) row.status = result.status
      }
    }
    feedback.success(t('ssh.common.batchTestDone'))
  } finally {
    batchTesting.value = false
  }
}

const batchDelete = () => {
  if (!selectedIds.value.length) return
  Dialog.confirm({
    title: t('ssh.common.confirmDeleteTitle'),
    content: t('ssh.common.confirmDeleteSelected'),
    confirmText: t('ssh.common.batchDelete'),
    cancelText: t('ssh.common.cancel'),
    onConfirm: async () => {
      await Promise.all(selectedIds.value.map((id) => deleteSshHost({ id })))
      feedback.success(t('ssh.common.batchDeleteSuccess'))
      selectedIds.value = []
      getList()
    },
  })
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') pagination.value.page = 1
  getList()
}

// —— 分享 ——
const shareDialogOpen = ref(false)
const shareLoading = ref(false)
const shareHostId = ref('')
const shareUserIds = ref<string[]>([])

const openShareDialog = (row: SSHHostInfo) => {
  shareHostId.value = row.id
  shareUserIds.value = row.sharedUsers?.map((user) => user.userId) || []
  shareDialogOpen.value = true
}

const confirmShare = async () => {
  shareLoading.value = true
  try {
    await shareSshHost({ id: shareHostId.value, userIds: shareUserIds.value })
    feedback.success(shareUserIds.value.length ? t('ssh.common.shareSuccess') : t('ssh.common.shareAllCancelled'))
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
.ssh-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* 批量条 */
.ssh-page__batch {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 16px;
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 30%, var(--el-border-color-lighter));
  border-radius: var(--app-radius-lg);
  background: color-mix(in srgb, var(--el-color-primary) 5%, var(--el-bg-color));
}
.ssh-page__batch-count {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-color-primary);
  white-space: nowrap;
}
.ssh-page__batch-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

.ssh-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.ssh-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.ssh-page__hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.ssh-page__selall {
  border: none;
  background: transparent;
  color: var(--el-color-primary);
  font-size: 0.8125rem;
  font-weight: 500;
  cursor: pointer;
}

/* 移动卡片 */
.ssh-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.ssh-card__title {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  cursor: pointer;
}
.ssh-card__check {
  width: 15px;
  height: 15px;
  accent-color: var(--el-color-primary);
  cursor: pointer;
}
.ssh-card__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.ssh-card__full {
  flex-basis: 100%;
}
.ssh-skeleton {
  height: 130px;
  border-radius: var(--app-radius);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: ssh-shimmer 1.4s ease-in-out infinite;
}
@keyframes ssh-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
