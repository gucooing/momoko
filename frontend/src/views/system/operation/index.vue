<!-- 操作记录（重写 · P1 只读列表）：PageHeader + FilterBar(可搜索用户/类型 AppSelect) + DataTable / 移动卡片 + Pagination + 详情 FormDialog。
     只读审计页，无 CRUD；用户预加载一次用于筛选下拉 + userId→用户名 映射。保留 listOperationLogs 接口与筛选契约。 -->
<template>
  <div class="op-page">
    <PageHeader :title="t('system.operation.title')" :description="t('system.operation.pageDesc')" />

    <FilterBar @search="search" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('system.operation.user') }}</label>
          <AppSelect
            v-model="queryForm.userId"
            :options="userOptions"
            searchable
            :search-placeholder="t('system.operation.searchUserPlaceholder')"
            :no-match-text="t('system.common.noData')"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('system.operation.operationType') }}</label>
          <AppSelect
            v-model="queryForm.operationType"
            :options="typeSelectOptions"
            searchable
            :search-placeholder="t('system.operation.selectOperationType')"
            :no-match-text="t('system.common.noData')"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('system.operation.result') }}</label>
          <AppSelect v-model="queryForm.success" :options="successOptions" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('system.operation.requestPath') }}</label>
          <input
            v-model="queryForm.path"
            class="app-input"
            :placeholder="t('system.operation.requestPathPlaceholder')"
            @keyup.enter="search"
          />
        </div>
      </template>
    </FilterBar>

    <div class="op-page__body">
      <div class="op-page__bar">
        <span class="op-page__bar-hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
      </div>

      <!-- 桌面：表格 -->
      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="logs"
        :loading="loading"
        :empty-text="t('system.operation.emptyDesc')"
      >
        <template #cell-userId="{ row }">{{ userLabel(row.userId) }}</template>
        <template #cell-operationType="{ row }">
          <StatusPill variant="primary" :dot="false" :label="getOperationTypeLabel(String(row.operationType))" />
        </template>
        <template #cell-durationMs="{ row }">{{ row.durationMs }}ms</template>
        <template #cell-success="{ row }">
          <StatusPill
            :variant="row.success ? 'success' : 'error'"
            :label="row.success ? t('system.common.success') : t('system.common.failed')"
          />
        </template>
        <template #cell-operationTime="{ row }">{{ formatTime(row.operationTime) }}</template>
        <template #cell-detail="{ row }">
          <span v-if="row.detail" class="op-detail-link" @click="showDetailDialog(String(row.detail))">
            {{ row.detail }}
          </span>
          <span v-else class="text-dim">—</span>
        </template>
      </DataTable>

      <!-- 移动：卡片 -->
      <template v-else>
        <div v-if="loading" class="op-cards">
          <div v-for="i in 6" :key="i" class="op-skeleton" />
        </div>
        <EmptyState
          v-else-if="!logs.length"
          icon="HOutline:ClipboardDocumentListIcon"
          :title="t('system.common.noData')"
          :description="t('system.operation.emptyDesc')"
        />
        <div v-else class="op-cards">
          <EntityCard
            v-for="(row, idx) in logs"
            :key="idx"
            :clickable="!!row.detail"
            @click="row.detail && showDetailDialog(row.detail)"
          >
            <template #title>{{ getOperationTypeLabel(row.operationType) }}</template>
            <template #status>
              <StatusPill
                :variant="row.success ? 'success' : 'error'"
                :label="row.success ? t('system.common.success') : t('system.common.failed')"
              />
            </template>
            <template #meta>
              <span>{{ userLabel(row.userId) }}</span>
              <span>IP: {{ row.ip || '—' }}</span>
              <span>{{ row.durationMs }}ms</span>
            </template>
            <template #footer>
              <span>{{ formatTime(row.operationTime) }}</span>
              <span v-if="row.detail" class="op-card__view">{{ t('system.operation.detail') }}</span>
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

    <FormDialog v-model="detailVisible" :title="t('system.operation.detailTitle')" :width="560">
      <pre v-if="isDetailJson" class="op-detail__json">{{ detailContent }}</pre>
      <div v-else class="op-detail__text">{{ detailContent }}</div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" icon="i-lucide-copy" @click="copyDetail">
          {{ t('system.common.copy') }}
        </UButton>
        <UButton color="primary" @click="close">{{ t('system.common.close') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import dayjs from 'dayjs'
import { useClipboard } from '@vueuse/core'
import { listOperationLogs } from '@/api/system'
import { userPage } from '@/api/user'
import { OperationType, type OperationLogInfo } from '@/types/v1/system'
import type { UserInfo } from '@/types/v1/user'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'

defineOptions({ name: 'OperationLogView' })

const menuStore = useMenuStore()
const { t } = useI18n()

const loading = ref(false)
const logs = ref<OperationLogInfo[]>([])
const users = ref<UserInfo[]>([])

const queryForm = ref({
  userId: '' as string,
  operationType: '' as string,
  success: undefined as boolean | undefined,
  path: '',
})
const pagination = ref({ page: 1, pageSize: 20, total: 0 })

// —— 用户预加载：筛选下拉 + userId→用户名 映射 ——
const userMap = computed<Record<string, string>>(() => {
  const m: Record<string, string> = {}
  for (const u of users.value) m[u.userId] = `${u.username}${u.name ? ` (${u.name})` : ''}`
  return m
})
const userOptions = computed<{ label: string; value: string }[]>(() => [
  { label: t('system.operation.allUsers'), value: '' },
  ...users.value.map((u) => ({ label: userMap.value[u.userId] || u.userId, value: u.userId })),
])
const userLabel = (id: unknown) => {
  const key = id == null ? '' : String(id)
  return key ? userMap.value[key] || key : '—'
}

const successOptions = computed<{ label: string; value: boolean | undefined }[]>(() => [
  { label: t('system.common.all'), value: undefined },
  { label: t('system.common.success'), value: true },
  { label: t('system.common.failed'), value: false },
])

const operationTypeLabelKeys: Record<string, string> = {
  [OperationType.OperationTypeUncategorized]: 'system.operation.types.uncategorized',
  [OperationType.OperationTypeAuthLogin]: 'system.operation.types.authLogin',
  [OperationType.OperationTypeAuthRegister]: 'system.operation.types.authRegister',
  [OperationType.OperationTypeAuthLogout]: 'system.operation.types.authLogout',
  [OperationType.OperationTypeAuthUpdatePassword]: 'system.operation.types.authUpdatePassword',
  [OperationType.OperationTypeAuthRegisterEmailCode]: 'system.operation.types.authRegisterEmailCode',
  [OperationType.OperationTypeAuthLoginEmailCode]: 'system.operation.types.authLoginEmailCode',
  [OperationType.OperationTypeUserUpdateMe]: 'system.operation.types.userUpdateMe',
  [OperationType.OperationTypeUserCreate]: 'system.operation.types.userCreate',
  [OperationType.OperationTypeUserUpdate]: 'system.operation.types.userUpdate',
  [OperationType.OperationTypeUserDelete]: 'system.operation.types.userDelete',
  [OperationType.OperationTypeSystemPermissionCreate]: 'system.operation.types.systemPermissionCreate',
  [OperationType.OperationTypeSystemPermissionUpdate]: 'system.operation.types.systemPermissionUpdate',
  [OperationType.OperationTypeSystemPermissionDelete]: 'system.operation.types.systemPermissionDelete',
  [OperationType.OperationTypeSystemRoleCreate]: 'system.operation.types.systemRoleCreate',
  [OperationType.OperationTypeSystemRoleUpdate]: 'system.operation.types.systemRoleUpdate',
  [OperationType.OperationTypeSystemRoleDelete]: 'system.operation.types.systemRoleDelete',
  [OperationType.OperationTypeSystemLoginConfigUpdate]: 'system.operation.types.systemLoginConfigUpdate',
  [OperationType.OperationTypeFileCreate]: 'system.operation.types.fileCreate',
  [OperationType.OperationTypeFileRename]: 'system.operation.types.fileRename',
  [OperationType.OperationTypeFileCopy]: 'system.operation.types.fileCopy',
  [OperationType.OperationTypeFileMove]: 'system.operation.types.fileMove',
  [OperationType.OperationTypeFileDelete]: 'system.operation.types.fileDelete',
  [OperationType.OperationTypeFileCompress]: 'system.operation.types.fileCompress',
  [OperationType.OperationTypeFileDecompress]: 'system.operation.types.fileDecompress',
  [OperationType.OperationTypeFileUploadComplete]: 'system.operation.types.fileUploadComplete',
  [OperationType.OperationTypeFileUploadCancel]: 'system.operation.types.fileUploadCancel',
  [OperationType.OperationTypeFileEdit]: 'system.operation.types.fileEdit',
  [OperationType.OperationTypeInstanceTypeCreate]: 'system.operation.types.instanceTypeCreate',
  [OperationType.OperationTypeInstanceTypeUpdate]: 'system.operation.types.instanceTypeUpdate',
  [OperationType.OperationTypeInstanceTypeDelete]: 'system.operation.types.instanceTypeDelete',
  [OperationType.OperationTypeInstanceCreate]: 'system.operation.types.instanceCreate',
  [OperationType.OperationTypeInstanceStart]: 'system.operation.types.instanceStart',
  [OperationType.OperationTypeInstanceStop]: 'system.operation.types.instanceStop',
  [OperationType.OperationTypeInstanceRestart]: 'system.operation.types.instanceRestart',
  [OperationType.OperationTypeInstanceDelete]: 'system.operation.types.instanceDelete',
  [OperationType.OperationTypeInstanceUpdate]: 'system.operation.types.instanceUpdate',
  [OperationType.OperationTypeInstanceLogDelete]: 'system.operation.types.instanceLogDelete',
  [OperationType.OperationTypeInstanceFileCreate]: 'system.operation.types.instanceFileCreate',
  [OperationType.OperationTypeInstanceFileRename]: 'system.operation.types.instanceFileRename',
  [OperationType.OperationTypeInstanceFileCopy]: 'system.operation.types.instanceFileCopy',
  [OperationType.OperationTypeInstanceFileMove]: 'system.operation.types.instanceFileMove',
  [OperationType.OperationTypeInstanceFileDelete]: 'system.operation.types.instanceFileDelete',
  [OperationType.OperationTypeInstanceFileCompress]: 'system.operation.types.instanceFileCompress',
  [OperationType.OperationTypeInstanceFileDecompress]: 'system.operation.types.instanceFileDecompress',
  [OperationType.OperationTypeInstanceFileUploadPreSign]: 'system.operation.types.instanceFileUploadPreSign',
  [OperationType.OperationTypeInstanceFileEdit]: 'system.operation.types.instanceFileEdit',
  [OperationType.OperationTypeSSHHostCreate]: 'system.operation.types.sshHostCreate',
  [OperationType.OperationTypeSSHHostUpdate]: 'system.operation.types.sshHostUpdate',
  [OperationType.OperationTypeSSHHostDelete]: 'system.operation.types.sshHostDelete',
  [OperationType.OperationTypeSSHHostShare]: 'system.operation.types.sshHostShare',
  [OperationType.OperationTypeSSHHostTest]: 'system.operation.types.sshHostTest',
  [OperationType.OperationTypeSSHHostBatchTest]: 'system.operation.types.sshHostBatchTest',
  [OperationType.OperationTypeSystemEmailConfigUpdate]: 'system.operation.types.systemEmailConfigUpdate',
  [OperationType.OperationTypeSystemEmailConfigTest]: 'system.operation.types.systemEmailConfigTest',
  [OperationType.OperationTypeSystemEmailTemplateUpdate]: 'system.operation.types.systemEmailTemplateUpdate',
  [OperationType.OperationTypeNodeAPIKeyCreate]: 'system.operation.types.nodeAPIKeyCreate',
  [OperationType.OperationTypeNodeAPIKeyCopy]: 'system.operation.types.nodeAPIKeyCopy',
  [OperationType.OperationTypeNodeAPIKeyUpdate]: 'system.operation.types.nodeAPIKeyUpdate',
  [OperationType.OperationTypeNodeAPIKeyRefresh]: 'system.operation.types.nodeAPIKeyRefresh',
  [OperationType.OperationTypeDockerConfigUpdate]: 'system.operation.types.dockerConfigUpdate',
  [OperationType.OperationTypeDockerContainerCreate]: 'system.operation.types.dockerContainerCreate',
  [OperationType.OperationTypeDockerContainerUpdate]: 'system.operation.types.dockerContainerUpdate',
  [OperationType.OperationTypeDockerContainerDelete]: 'system.operation.types.dockerContainerDelete',
  [OperationType.OperationTypeDockerContainerStart]: 'system.operation.types.dockerContainerStart',
  [OperationType.OperationTypeDockerContainerStop]: 'system.operation.types.dockerContainerStop',
  [OperationType.OperationTypeDockerContainerRestart]: 'system.operation.types.dockerContainerRestart',
  [OperationType.OperationTypeDockerContainerKill]: 'system.operation.types.dockerContainerKill',
  [OperationType.OperationTypeDockerContainerPause]: 'system.operation.types.dockerContainerPause',
  [OperationType.OperationTypeDockerContainerUnpause]: 'system.operation.types.dockerContainerUnpause',
  [OperationType.OperationTypeDockerContainerRename]: 'system.operation.types.dockerContainerRename',
  [OperationType.OperationTypeDockerContainerRecreate]: 'system.operation.types.dockerContainerRecreate',
  [OperationType.OperationTypeDockerImagePull]: 'system.operation.types.dockerImagePull',
  [OperationType.OperationTypeDockerImageTagsUpdate]: 'system.operation.types.dockerImageTagsUpdate',
  [OperationType.OperationTypeDockerImageTag]: 'system.operation.types.dockerImageTag',
  [OperationType.OperationTypeDockerImageDelete]: 'system.operation.types.dockerImageDelete',
  [OperationType.OperationTypeDockerNetworkCreate]: 'system.operation.types.dockerNetworkCreate',
  [OperationType.OperationTypeDockerNetworkUpdate]: 'system.operation.types.dockerNetworkUpdate',
  [OperationType.OperationTypeDockerNetworkRecreate]: 'system.operation.types.dockerNetworkRecreate',
  [OperationType.OperationTypeDockerNetworkDelete]: 'system.operation.types.dockerNetworkDelete',
  [OperationType.OperationTypeDockerNetworkConnect]: 'system.operation.types.dockerNetworkConnect',
  [OperationType.OperationTypeDockerNetworkDisconnect]: 'system.operation.types.dockerNetworkDisconnect',
  [OperationType.OperationTypeDockerNetworkPrune]: 'system.operation.types.dockerNetworkPrune',
  [OperationType.OperationTypeDockerVolumeCreate]: 'system.operation.types.dockerVolumeCreate',
  [OperationType.OperationTypeDockerVolumeUpdate]: 'system.operation.types.dockerVolumeUpdate',
  [OperationType.OperationTypeDockerVolumeRecreate]: 'system.operation.types.dockerVolumeRecreate',
  [OperationType.OperationTypeDockerVolumeDelete]: 'system.operation.types.dockerVolumeDelete',
  [OperationType.OperationTypeDockerVolumePrune]: 'system.operation.types.dockerVolumePrune',
  [OperationType.OperationTypeDockerVolumeExport]: 'system.operation.types.dockerVolumeExport',
  [OperationType.OperationTypeDockerVolumeRestore]: 'system.operation.types.dockerVolumeRestore',
  [OperationType.OperationTypeDockerConfigTest]: 'system.operation.types.dockerConfigTest',
  [OperationType.OperationTypeOIDCConfigUpdate]: 'system.operation.types.oidcConfigUpdate',
  [OperationType.OperationTypeOIDCClientCreate]: 'system.operation.types.oidcClientCreate',
  [OperationType.OperationTypeOIDCClientUpdate]: 'system.operation.types.oidcClientUpdate',
  [OperationType.OperationTypeOIDCClientDelete]: 'system.operation.types.oidcClientDelete',
  [OperationType.OperationTypeOIDCClientSecretRefresh]: 'system.operation.types.oidcClientSecretRefresh',
  [OperationType.OperationTypeOIDCAuthorizationCodeCreate]: 'system.operation.types.oidcAuthorizationCodeCreate',
  [OperationType.UNRECOGNIZED]: 'system.operation.types.unrecognized',
}

const getOperationTypeLabel = (operationType: string) => {
  const labelKey = operationTypeLabelKeys[operationType]
  return labelKey ? t(labelKey) : operationType
}

const typeSelectOptions = computed<{ label: string; value: string }[]>(() => [
  { label: t('system.operation.allTypes'), value: '' },
  ...Object.entries(operationTypeLabelKeys)
    .filter(([value]) => value !== OperationType.UNRECOGNIZED)
    .map(([value, labelKey]) => ({ label: t(labelKey), value })),
])

const columns = computed<DataTableColumn[]>(() => [
  { key: 'userId', title: t('system.operation.user'), minWidth: 150 },
  { key: 'operationType', title: t('system.operation.operationType'), minWidth: 160 },
  { key: 'ip', title: t('system.operation.ipAddress'), width: 140 },
  { key: 'durationMs', title: t('system.operation.duration'), width: 90, align: 'right' },
  { key: 'success', title: t('system.operation.result'), width: 90 },
  { key: 'operationTime', title: t('system.operation.operationTime'), width: 170 },
  { key: 'detail', title: t('system.operation.detail'), minWidth: 220 },
  { key: 'userAgent', title: t('system.operation.userAgent'), minWidth: 180 },
])

// —— 详情弹窗 ——
const { copy } = useClipboard()
const detailVisible = ref(false)
const detailContent = ref('')
const isDetailJson = ref(false)

const showDetailDialog = (detail: string) => {
  if (!detail) {
    detailContent.value = '—'
    isDetailJson.value = false
    detailVisible.value = true
    return
  }
  try {
    detailContent.value = JSON.stringify(JSON.parse(detail), null, 2)
    isDetailJson.value = true
  } catch {
    detailContent.value = detail
    isDetailJson.value = false
  }
  detailVisible.value = true
}

const copyDetail = async () => {
  try {
    await copy(detailContent.value)
    feedback.success(t('system.operation.copied'))
  } catch {
    feedback.error(t('system.operation.copyFailed'))
  }
}

const formatTime = (value: unknown) => {
  if (!value) return '—'
  const d = dayjs(value as string | number | Date)
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : String(value)
}

const loadUsers = async () => {
  try {
    const { data } = await userPage({ page: 1, pageSize: 1000 })
    users.value = data?.users || []
  } catch {
    users.value = []
  }
}

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listOperationLogs({
      userId: queryForm.value.userId || undefined,
      operationType: (queryForm.value.operationType || undefined) as OperationType | undefined,
      success: queryForm.value.success,
      path: queryForm.value.path || undefined,
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      startTime: undefined,
      endTime: undefined,
    })
    logs.value = data?.logs || []
    pagination.value.total = Number(data?.total || 0)
    pagination.value.page = Number(data?.page || pagination.value.page)
    pagination.value.pageSize = Number(data?.pageSize || pagination.value.pageSize)
  } catch {
    logs.value = []
  } finally {
    loading.value = false
  }
}

const search = () => {
  pagination.value.page = 1
  getList()
}
const reset = () => {
  queryForm.value = { userId: '', operationType: '', success: undefined, path: '' }
  pagination.value.page = 1
  getList()
}

onMounted(() => {
  loadUsers()
  getList()
})
</script>

<style scoped lang="scss">
.op-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.op-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.op-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.op-page__bar-hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.op-detail-link {
  color: var(--el-color-primary);
  cursor: pointer;
}
.op-detail-link:hover {
  text-decoration: underline;
}
.text-dim {
  color: var(--el-text-color-placeholder);
}

/* 详情弹窗 */
.op-detail__json {
  margin: 0;
  padding: 12px 14px;
  background: var(--el-fill-color-light);
  border-radius: var(--app-radius-sm);
  font-size: 0.8rem;
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  line-height: 1.55;
  color: var(--el-text-color-primary);
  overflow-x: auto;
  white-space: pre;
}
.op-detail__text {
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 0.8125rem;
  line-height: 1.6;
  color: var(--el-text-color-regular);
}

/* 移动卡片 */
.op-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.op-card__view {
  color: var(--el-color-primary);
}
.op-skeleton {
  height: 96px;
  border-radius: var(--app-radius);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: op-shimmer 1.4s ease-in-out infinite;
}
@keyframes op-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}
</style>
