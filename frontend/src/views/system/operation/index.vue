<template>
  <div>
    <el-card shadow="never" class="mb-4">
      <el-form ref="queryFormRef" :model="queryForm" :inline="true">
        <el-form-item label="用户" prop="userId">
          <el-select
            v-model="queryForm.userId"
            filterable
            remote
            clearable
            reserve-keyword
            placeholder="请输入用户名搜索"
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
        <el-form-item label="操作类型" prop="operationType">
          <el-select
            v-model="queryForm.operationType"
            placeholder="请选择操作类型"
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
        <el-form-item label="结果" prop="success">
          <el-select
            v-model="queryForm.success"
            placeholder="全部"
            clearable
            style="width: 120px"
          >
            <el-option label="成功" :value="true" />
            <el-option label="失败" :value="false" />
          </el-select>
        </el-form-item>
        <el-form-item label="路径" prop="path">
          <el-input
            v-model="queryForm.path"
            placeholder="请输入请求路径"
            clearable
            style="width: 200px"
          />
        </el-form-item>
        <el-form-item>
          <el-button type="primary" @click="search">
            <template #icon>
              <component :is="menuStore.iconComponents.Search" />
            </template>
            查询
          </el-button>
          <el-button @click="reset">重置</el-button>
        </el-form-item>
      </el-form>
    </el-card>

    <el-card shadow="never">
      <!-- desktop: table -->
      <VxeGrid v-if="!menuStore.isMobile" v-bind="gridConfig">
        <template #column-operationType="{ row }">
          <span class="op-type-tag">{{ operationTypeLabels[row.operationType] || row.operationType }}</span>
        </template>
        <template #column-success="{ row }">
          <BaseTag :type="row.success ? 'success' : 'danger'" :text="row.success ? '成功' : '失败'" />
        </template>
        <template #column-detail="{ row }">
          <span class="detail-preview" @click="showDetailDialog(row.detail)">{{ row.detail || '-' }}</span>
        </template>
        <template #column-duration="{ row }">{{ `${row.durationMs}ms` }}</template>
        <template #column-operationTime="{ row }">{{ formatTime(row.operationTime) }}</template>
      </VxeGrid>

      <!-- mobile: cards -->
      <div v-else class="mobile-card-list">
        <div v-if="!logs.length" class="mobile-empty"><el-empty description="暂无数据" /></div>
        <div v-for="(row, idx) in logs" :key="idx" class="mobile-card" @click="row.detail && showDetailDialog(row.detail)">
          <div class="mobile-card-body">
            <div class="mobile-card-header">
              <span class="mobile-card-title">{{ operationTypeLabels[row.operationType] || row.operationType }}</span>
              <BaseTag :type="row.success ? 'success' : 'danger'" :text="row.success ? '成功' : '失败'" />
            </div>
            <div class="mobile-card-meta">
              <span>用户: {{ row.userId }}</span>
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

    <BaseDialog v-model="detailVisible" title="操作详情" width="560">
      <el-scrollbar max-height="60vh">
        <pre v-if="isDetailJson" class="detail-json">{{ detailContent }}</pre>
        <div v-else class="detail-dialog-body">{{ detailContent }}</div>
      </el-scrollbar>
      <template #footer>
        <el-button @click="copyDetail">
          <template #icon>
            <component :is="menuStore.iconComponents['HOutline:ClipboardDocumentIcon']" />
          </template>
          复制
        </el-button>
        <el-button type="primary" @click="detailVisible = false">关闭</el-button>
      </template>
    </BaseDialog>
  </div>
</template>

<script setup lang="ts">
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

const operationTypeLabels: Record<string, string> = {
  [OperationType.OperationTypeUncategorized]: '未分类',
  [OperationType.OperationTypeAuthLogin]: '登录',
  [OperationType.OperationTypeAuthRegister]: '注册',
  [OperationType.OperationTypeAuthLogout]: '退出登录',
  [OperationType.OperationTypeAuthUpdatePassword]: '修改密码',
  [OperationType.OperationTypeAuthDeviceDelete]: '登录设备删除',
  [OperationType.OperationTypeAuthRegisterEmailCode]: '注册邮件验证码发送',
  [OperationType.OperationTypeAuthLoginEmailCode]: '登录邮件验证码发送',
  [OperationType.OperationTypeUserUpdateMe]: '个人信息更新',
  [OperationType.OperationTypeUserCreate]: '用户新增',
  [OperationType.OperationTypeUserUpdate]: '用户编辑',
  [OperationType.OperationTypeUserDelete]: '用户删除',
  [OperationType.OperationTypeSystemPermissionCreate]: '权限菜单新增',
  [OperationType.OperationTypeSystemPermissionUpdate]: '权限菜单编辑',
  [OperationType.OperationTypeSystemPermissionDelete]: '权限菜单删除',
  [OperationType.OperationTypeSystemRoleCreate]: '角色新增',
  [OperationType.OperationTypeSystemRoleUpdate]: '角色编辑',
  [OperationType.OperationTypeSystemRoleDelete]: '角色删除',
  [OperationType.OperationTypeSystemLoginConfigUpdate]: '登录配置更新',
  [OperationType.OperationTypeFileCreate]: '文件创建',
  [OperationType.OperationTypeFileRename]: '文件重命名',
  [OperationType.OperationTypeFileCopy]: '文件复制',
  [OperationType.OperationTypeFileMove]: '文件移动',
  [OperationType.OperationTypeFileDelete]: '文件删除',
  [OperationType.OperationTypeFileCompress]: '文件压缩',
  [OperationType.OperationTypeFileDecompress]: '文件解压',
  [OperationType.OperationTypeFileUploadComplete]: '文件上传完成',
  [OperationType.OperationTypeFileUploadCancel]: '文件上传取消',
  [OperationType.OperationTypeInstanceTypeCreate]: '实例类型新增',
  [OperationType.OperationTypeInstanceTypeUpdate]: '实例类型编辑',
  [OperationType.OperationTypeInstanceTypeDelete]: '实例类型删除',
  [OperationType.OperationTypeInstanceTerminalStart]: '终端启动',
  [OperationType.OperationTypeInstanceTerminalStop]: '终端停止',
  [OperationType.OperationTypeInstanceTerminalRestart]: '终端重启',
  [OperationType.OperationTypeInstanceCreate]: '实例新增',
  [OperationType.OperationTypeInstanceStart]: '实例启动',
  [OperationType.OperationTypeInstanceStop]: '实例停止',
  [OperationType.OperationTypeInstanceRestart]: '实例重启',
  [OperationType.OperationTypeInstanceDelete]: '实例删除',
  [OperationType.OperationTypeInstanceUpdate]: '实例编辑',
  [OperationType.OperationTypeInstanceLogDelete]: '实例日志删除',
  [OperationType.OperationTypeInstanceFileCreate]: '实例文件创建',
  [OperationType.OperationTypeInstanceFileRename]: '实例文件重命名',
  [OperationType.OperationTypeInstanceFileCopy]: '实例文件复制',
  [OperationType.OperationTypeInstanceFileMove]: '实例文件移动',
  [OperationType.OperationTypeInstanceFileDelete]: '实例文件删除',
  [OperationType.OperationTypeInstanceFileCompress]: '实例文件压缩',
  [OperationType.OperationTypeInstanceFileDecompress]: '实例文件解压',
  [OperationType.OperationTypeInstanceFileUploadPreSign]: '实例文件上传签名',
  [OperationType.OperationTypeSSHHostCreate]: 'SSH连接新增',
  [OperationType.OperationTypeSSHHostUpdate]: 'SSH连接编辑',
  [OperationType.OperationTypeSSHHostDelete]: 'SSH连接删除',
  [OperationType.OperationTypeSSHHostShare]: 'SSH连接分享',
  [OperationType.OperationTypeSSHHostTest]: 'SSH连接测试',
  [OperationType.OperationTypeSSHHostBatchTest]: 'SSH连接批量测试',
  [OperationType.OperationTypeSystemEmailConfigUpdate]: '邮件配置更新',
  [OperationType.OperationTypeSystemEmailConfigTest]: '邮件配置测试',
  [OperationType.OperationTypeSystemEmailTemplateUpdate]: '邮件模板更新',
}

const operationTypeOptions = Object.entries(operationTypeLabels).map(([value, label]) => ({
  value,
  label,
}))

const gridConfig = computed<VxeGridProps>(() => ({
  border: true,
  showOverflow: true,
  rowConfig: { isHover: true },
  data: logs.value,
  columns: [
    { field: 'userId', title: '用户ID', minWidth: 140 },
    { field: 'operationType', title: '操作类型', minWidth: 140, slots: { default: 'column-operationType' } },
    { field: 'ip', title: 'IP 地址', minWidth: 140 },
    { field: 'durationMs', title: '耗时', width: 100, slots: { default: 'column-duration' } },
    { field: 'success', title: '结果', width: 80, slots: { default: 'column-success' } },
    { field: 'operationTime', title: '操作时间', minWidth: 170, slots: { default: 'column-operationTime' } },
    { field: 'detail', title: '操作详情', minWidth: 160, slots: { default: 'column-detail' } },
    { field: 'userAgent', title: 'UA', minWidth: 200 },
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
    ElMessage.success('已复制')
  } catch {
    ElMessage.error('复制失败')
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
