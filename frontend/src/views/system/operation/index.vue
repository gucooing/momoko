<template>
  <div>
    <el-card shadow="never" class="mb-4">
      <el-form ref="queryFormRef" :model="queryForm" :inline="true">
        <el-form-item :label="t('system.operation.user')" prop="userId">
          <el-select
            v-model="queryForm.userId"
            filterable
            remote
            clearable
            reserve-keyword
            :placeholder="t('system.operation.searchUserPlaceholder')"
            :remote-method="searchUser"
            :loading="userLoading"
            style="width: 200px"
          >
            <el-option
              v-for="item in userOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('system.operation.operationType')" prop="operationType">
          <el-select
            v-model="queryForm.operationType"
            :placeholder="t('system.operation.selectOperationType')"
            clearable
            style="width: 200px"
          >
            <el-option
              v-for="item in operationTypeOptions"
              :key="item.value"
              :label="item.label"
              :value="item.value"
            />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('system.operation.result')" prop="success">
          <el-select
            v-model="queryForm.success"
            :placeholder="t('system.common.all')"
            clearable
            style="width: 120px"
          >
            <el-option :label="t('system.common.success')" :value="true" />
            <el-option :label="t('system.common.failed')" :value="false" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('system.operation.requestPath')" prop="path">
          <el-input
            v-model="queryForm.path"
            :placeholder="t('system.operation.requestPathPlaceholder')"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="search">
            <template #icon>
              <component :is="menuStore.iconComponents.Search" />
            </template>
            {{ t('system.operation.query') }}
          </el-button>
          <el-button @click="reset">{{ t('system.common.reset') }}</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <!-- desktop: table -->
      <VxeGrid v-if="!menuStore.isMobile" v-bind="gridConfig">
        <template #column-operationType="{ row }">
          <span class="op-type-tag">{{ getOperationTypeLabel(row.operationType) }}</span>
        </template>
        <template #column-success="{ row }">
          <BaseTag :type="row.success ? 'success' : 'danger'" :text="row.success ? t('system.common.success') : t('system.common.failed')" />
        </template>
        <template #column-detail="{ row }">
          <span class="detail-preview" @click="showDetailDialog(row.detail)">{{ row.detail || '-' }}</span>
        </template>
        <template #column-duration="{ row }">{{ `${row.durationMs}ms` }}</template>
        <template #column-operationTime="{ row }">{{ formatTime(row.operationTime) }}</template>
      </VxeGrid>

      <!-- mobile: cards -->
      <div v-else class="mobile-card-list">
        <div v-if="!logs.length" class="mobile-empty"><el-empty :description="t('system.common.noData')" /></div>
        <div v-for="(row, idx) in logs" :key="idx" class="mobile-card" @click="row.detail && showDetailDialog(row.detail)">
          <div class="mobile-card-body">
            <div class="mobile-card-header">
              <span class="mobile-card-title">{{ getOperationTypeLabel(row.operationType) }}</span>
              <BaseTag :type="row.success ? 'success' : 'danger'" :text="row.success ? t('system.common.success') : t('system.common.failed')" />
            </div>
            <div class="mobile-card-meta">
              <span>{{ t('system.operation.userMeta', { userId: row.userId }) }}</span>
              <span class="meta-sep">·</span>
              <span>IP: {{ row.ip || '-' }}</span>
              <span class="meta-sep">·</span>
              <span>{{ row.durationMs }}ms</span>
            </div>
            <div class="mobile-card-meta">{{ formatTime(row.operationTime) }}</div>
            <div v-if="row.detail" class="mobile-card-detail">{{ row.detail }}</div>
          </div>
        </div>
      </div>

      <TablePagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :is-mobile="menuStore.isMobile"
        @change="getList"
      />
    </el-card>

    <BaseDialog v-model="detailVisible" :title="t('system.operation.detailTitle')" width="560">
      <el-scrollbar max-height="60vh">
        <pre v-if="isDetailJson" class="detail-json">{{ detailContent }}</pre>
        <div v-else class="detail-dialog-body">{{ detailContent }}</div>
      </el-scrollbar>
      <template #footer>
        <el-button @click="copyDetail">
          <template #icon>
            <component :is="menuStore.iconComponents['HOutline:ClipboardDocumentIcon']" />
          </template>
          {{ t('system.common.copy') }}
        </el-button>
        <el-button type="primary" @click="detailVisible = false">{{ t('system.common.close') }}</el-button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import { listOperationLogs } from '@/api/system'
import { userPage } from '@/api/user'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import TablePagination from '@/components/pagination/TablePagination.vue'
import { useClipboard } from '@vueuse/core'
import { VxeGrid } from '@/plugins/vxeGrid'
import dayjs from 'dayjs'
import { OperationType, type OperationLogInfo } from '@/types/v1/system'
import type { FormInstance } from 'element-plus'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'OperationLogView' })

const menuStore = useMenuStore()
const { t } = useI18n()

const queryFormRef = useTemplateRef<FormInstance>('queryFormRef')
const logs = ref<OperationLogInfo[]>([])

const queryForm = ref({
  userId: '',
  operationType: undefined as OperationType | undefined,
  success: undefined as boolean | undefined,
  path: '',
})

const pagination = ref({
  page: 1,
  pageSize: 20,
  total: 0,
})

const operationTypeLabelKeys: Record<string, string> = {
  [OperationType.OperationTypeUncategorized]: 'system.operation.types.uncategorized',
  [OperationType.OperationTypeAuthLogin]: 'system.operation.types.authLogin',
  [OperationType.OperationTypeAuthRegister]: 'system.operation.types.authRegister',
  [OperationType.OperationTypeAuthLogout]: 'system.operation.types.authLogout',
  [OperationType.OperationTypeAuthUpdatePassword]: 'system.operation.types.authUpdatePassword',
  [OperationType.OperationTypeAuthDeviceDelete]: 'system.operation.types.authDeviceDelete',
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
  [OperationType.OperationTypeInstanceTerminalStart]: 'system.operation.types.instanceTerminalStart',
  [OperationType.OperationTypeInstanceTerminalStop]: 'system.operation.types.instanceTerminalStop',
  [OperationType.OperationTypeInstanceTerminalRestart]: 'system.operation.types.instanceTerminalRestart',
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
}

const getOperationTypeLabel = (operationType: string) => {
  const labelKey = operationTypeLabelKeys[operationType]
  return labelKey ? t(labelKey) : operationType
}

const operationTypeOptions = computed(() =>
  Object.entries(operationTypeLabelKeys).map(([value, labelKey]) => ({
    value,
    label: t(labelKey),
  })),
)

const gridConfig = computed<VxeGridProps>(() => ({
  border: true,
  showOverflow: true,
  rowConfig: { isHover: true },
  data: logs.value,
  columns: [
    { field: 'userId', title: t('system.operation.userId'), minWidth: 140 },
    { field: 'operationType', title: t('system.operation.operationType'), minWidth: 140, slots: { default: 'column-operationType' } },
    { field: 'ip', title: t('system.operation.ipAddress'), minWidth: 140 },
    { field: 'durationMs', title: t('system.operation.duration'), width: 100, slots: { default: 'column-duration' } },
    { field: 'success', title: t('system.operation.result'), width: 80, slots: { default: 'column-success' } },
    { field: 'operationTime', title: t('system.operation.operationTime'), minWidth: 170, slots: { default: 'column-operationTime' } },
    { field: 'detail', title: t('system.operation.detail'), minWidth: 160, slots: { default: 'column-detail' } },
    { field: 'userAgent', title: t('system.operation.userAgent'), minWidth: 200 },
  ],
}))

const { copy } = useClipboard()
const detailVisible = ref(false)
const detailContent = ref('')
const isDetailJson = ref(false)

const showDetailDialog = (detail: string) => {
  if (!detail) {
    detailContent.value = '-'
    isDetailJson.value = false
    detailVisible.value = true
    return
  }

  try {
    const parsed = JSON.parse(detail)
    detailContent.value = JSON.stringify(parsed, null, 2)
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
    ElMessage.success(t('system.operation.copied'))
  } catch {
    ElMessage.error(t('system.operation.copyFailed'))
  }
}

const userOptions = ref<{ label: string; value: string }[]>([])
const userLoading = ref(false)

const searchUser = async (query: string) => {
  if (!query) {
    userOptions.value = []
    return
  }
  userLoading.value = true
  try {
    const { data } = await userPage({ username: query, page: 1, pageSize: 10 })
    userOptions.value = (data?.users || []).map((u) => ({
      label: `${u.username}${u.name ? ` (${u.name})` : ''}`,
      value: u.userId,
    }))
  } catch {
    userOptions.value = []
  } finally {
    userLoading.value = false
  }
}

const getList = async () => {
  try {
    const { data } = await listOperationLogs({
      userId: queryForm.value.userId || undefined,
      operationType: queryForm.value.operationType,
      success: queryForm.value.success,
      path: queryForm.value.path || undefined,
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      startTime: undefined,
      endTime: undefined,
    })
    logs.value = data?.logs || []
    pagination.value = {
      page: Number(data?.page || 1),
      pageSize: Number(data?.pageSize || 10),
      total: Number(data?.total || 0),
    }
  } catch {
    logs.value = []
  }
}

const search = () => {
  pagination.value.page = 1
  getList()
}

const reset = () => {
  queryFormRef.value?.resetFields()
  pagination.value.page = 1
  getList()
}

const formatTime = (value: unknown) => {
  if (!value) return '-'
  const d = dayjs(value as string | number | Date)
  return d.isValid() ? d.format('YYYY-MM-DD HH:mm:ss') : String(value)
}

onMounted(() => {
  getList()
})
</script>

<style scoped lang="scss">
.op-type-tag {
  display: inline-block;
  padding: 0.1rem 0.45rem;
  font-size: 0.8rem;
  color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 30%, transparent);
  border-radius: 0.25rem;
}

.detail-preview {
  display: -webkit-box;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
  line-clamp: 2;
  overflow: hidden;
  cursor: pointer;
  word-break: break-word;
}

.detail-dialog-body {
  white-space: pre-wrap;
  word-break: break-word;
}

.detail-json {
  margin: 0;
  padding: 0.75rem 1rem;
  background: color-mix(in srgb, var(--el-fill-color) 60%, transparent);
  border-radius: 0.5rem;
  font-size: 0.8rem;
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  line-height: 1.55;
  color: var(--el-text-color-primary);
  overflow-x: auto;
}

/* mobile */
.mobile-card-list { display: flex; flex-direction: column; gap: 0.5rem; }
.mobile-empty { padding: 1.5rem 0; }
.mobile-card { padding: 0.65rem 0.75rem; border: 1px solid var(--el-border-color-extra-light); border-radius: 0.6rem; background: var(--el-bg-color); }
.mobile-card-body { min-width: 0; }
.mobile-card-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
.mobile-card-title { font-size: 0.85rem; font-weight: 700; color: var(--el-text-color-primary); }
.mobile-card-meta { display: flex; align-items: center; gap: 0.25rem; margin-top: 0.2rem; font-size: 0.72rem; color: var(--el-text-color-secondary); flex-wrap: wrap; }
.meta-sep { color: var(--el-text-color-placeholder); }
.mobile-card-detail { margin-top: 0.3rem; font-size: 0.7rem; color: var(--el-text-color-placeholder); overflow: hidden; text-overflow: ellipsis; display: -webkit-box; -webkit-box-orient: vertical; -webkit-line-clamp: 2; line-clamp: 2; }
</style>
