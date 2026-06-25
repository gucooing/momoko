<template>
  <div>
    <el-card shadow="never" class="card-clear-mb">
      <el-form ref="queryFormRef" :model="queryForm" label-width="auto" @keyup.enter="getList">
        <el-row :gutter="12">
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="6">
            <el-form-item :label="t('tools.tunnel.keyword')" prop="keywords">
              <el-input
                v-model="queryForm.keywords"
                :placeholder="t('tools.tunnel.keywordPlaceholder')"
                :prefix-icon="menuStore.iconComponents.Search"
                clearable
              />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="6">
            <el-form-item :label="t('tools.tunnel.proxyType')" prop="type">
              <el-select v-model="queryForm.type" :placeholder="t('system.common.select')" clearable>
                <el-option v-for="opt in typeOptions" :key="opt" :label="typeLabel(opt)" :value="opt" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="6">
            <el-form-item :label="t('tools.tunnel.enableStatus')" prop="isEnable">
              <el-select v-model="queryForm.isEnable" :placeholder="t('system.common.select')" clearable>
                <el-option :label="t('tools.tunnel.enabled')" :value="true" />
                <el-option :label="t('tools.tunnel.disabled')" :value="false" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="12" :lg="6" :xl="6">
            <el-form-item>
              <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="getList">
                {{ t('system.common.search') }}
              </el-button>
              <el-button :icon="menuStore.iconComponents.Refresh" @click="reset">
                {{ t('system.common.reset') }}
              </el-button>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <el-card shadow="never" class="card-mt-16">
      <div class="operation-container">
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" @click="openCreateDialog()">
          {{ t('tools.tunnel.addTunnel') }}
        </el-button>
        <AdaptiveConfirm
          :title="t('tools.tunnel.confirmDeleteSelected')"
          :placement="POPCONFIRM_CONFIG.placement"
          :width="POPCONFIRM_CONFIG.width"
          @confirm="batchDelete"
        >
          <template #reference>
            <el-button type="danger" :icon="menuStore.iconComponents.Delete" :disabled="!deleteIds.length">
              {{ t('tools.tunnel.batchDelete') }}
            </el-button>
          </template>
        </AdaptiveConfirm>
        <el-button :icon="menuStore.iconComponents['HOutline:Cog6ToothIcon']" @click="frpsVisible = true">
          {{ t('tools.tunnel.frpsConfig') }}
        </el-button>
      </div>

      <!-- desktop: table -->
      <VxeGrid
        v-if="!menuStore.isMobile"
        v-bind="gridConfig"
        @checkbox-change="selectionChange"
        @checkbox-all="selectionChange"
      >
        <template #column-type="{ row }">
          <el-tag size="small">{{ typeLabel(row.type) }}</el-tag>
        </template>
        <template #column-status="{ row }">
          <el-tag :type="statusInfo(row.status).type" size="small">{{ statusInfo(row.status).label }}</el-tag>
        </template>
        <template #column-remote="{ row }">
          <span class="mono-text">{{ remoteText(row) }}</span>
        </template>
        <template #column-target="{ row }">
          <span class="mono-text">{{ row.localIp }}:{{ row.localPort }}</span>
        </template>
        <template #column-isEnable="{ row }">
          <el-switch
            :model-value="row.isEnable"
            size="small"
            @change="(val: string | number | boolean) => toggleEnable(row, !!val)"
          />
        </template>
        <template #column-connections="{ row }">
          <span>{{ Number(row.activeConnections || 0) }}</span>
        </template>
        <template #column-traffic="{ row }">
          <span class="tn-traffic">
            <span class="tn-traffic-item">↓ {{ formatBytes(row.bytesIn) }}</span>
            <span class="tn-traffic-item">↑ {{ formatBytes(row.bytesOut) }}</span>
          </span>
        </template>
        <template #column-operation="{ row }">
          <div class="tn-actions">
            <el-button type="primary" :icon="menuStore.iconComponents['HOutline:ChartBarIcon']" link @click="openStatsDialog(row)">
              {{ t('tools.tunnel.stats.details') }}
            </el-button>
            <el-button type="primary" :icon="menuStore.iconComponents['HOutline:DocumentTextIcon']" link @click="openFrpcDialog(row)">
              {{ t('tools.tunnel.frpcConfig') }}
            </el-button>
            <el-button type="primary" :icon="menuStore.iconComponents.Edit" link @click="openEditDialog(row)">
              {{ t('system.common.edit') }}
            </el-button>
            <el-popconfirm :title="t('tools.tunnel.confirmDeleteRule')" :width="POPCONFIRM_CONFIG.width" @confirm="confirmDelete(row)">
              <template #reference>
                <el-button type="danger" :icon="menuStore.iconComponents.Delete" link>{{ t('system.common.delete') }}</el-button>
              </template>
            </el-popconfirm>
          </div>
        </template>
      </VxeGrid>

      <!-- mobile: cards -->
      <div v-else v-loading="loading" class="mobile-card-list">
        <div v-if="!list.length" class="mobile-empty">
          <el-empty :description="t('system.common.noData')" />
        </div>
        <div v-for="row in list" :key="row.id" class="tn-card" :class="{ 'is-selected': deleteIds.includes(row.id) }">
          <div class="tn-card-check" @click.stop="toggleMobileSelect(row.id)">
            <el-checkbox :model-value="deleteIds.includes(row.id)" size="small" />
          </div>

          <div class="tn-card-header">
            <span class="tn-card-name">{{ row.name }}</span>
            <el-tag :type="statusInfo(row.status).type" size="small">{{ statusInfo(row.status).label }}</el-tag>
            <el-switch
              :model-value="row.isEnable"
              size="small"
              @change="(val: string | number | boolean) => toggleEnable(row, !!val)"
            />
          </div>

          <div class="tn-card-body">
            <div class="tn-card-meta">
              <el-tag size="small">{{ typeLabel(row.type) }}</el-tag>
              <span class="mono-text">{{ remoteText(row) }}</span>
            </div>
            <div class="tn-card-meta">
              <span>{{ t('tools.tunnel.target') }}:</span>
              <span class="mono-text">{{ row.localIp }}:{{ row.localPort }}</span>
            </div>
            <div class="tn-card-meta">
              <span>{{ t('tools.tunnel.stats.connections') }}: {{ Number(row.activeConnections || 0) }}</span>
              <span class="mono-text">↓ {{ formatBytes(row.bytesIn) }} ↑ {{ formatBytes(row.bytesOut) }}</span>
            </div>
            <div class="tn-card-time">{{ row.createTime }}</div>
          </div>

          <div class="tn-card-footer">
            <el-button size="small" plain @click="openStatsDialog(row)">{{ t('tools.tunnel.stats.details') }}</el-button>
            <el-button size="small" plain @click="openFrpcDialog(row)">{{ t('tools.tunnel.frpcConfig') }}</el-button>
            <el-button size="small" plain type="primary" @click="openEditDialog(row)">{{ t('system.common.edit') }}</el-button>
            <el-popconfirm :title="t('tools.tunnel.confirmDeleteRule')" :width="POPCONFIRM_CONFIG.width" @confirm="confirmDelete(row)">
              <template #reference>
                <el-button size="small" plain type="danger">{{ t('system.common.delete') }}</el-button>
              </template>
            </el-popconfirm>
          </div>
        </div>
      </div>

      <div v-if="menuStore.isMobile && deleteIds.length" class="mobile-batch-bar">
        <span>{{ t('tools.tunnel.selectedCount', { count: deleteIds.length }) }}</span>
        <el-button size="small" type="danger" @click="batchDelete">{{ t('tools.tunnel.batchDelete') }}</el-button>
      </div>

      <TablePagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :is-mobile="menuStore.isMobile"
        @change="getList"
      />
    </el-card>

    <TunnelCreate ref="createRef" @refresh="refresh" />
    <TunnelStatsDialog v-model="statsVisible" :row="statsRow" />
    <FrpcConfigDialog v-model="frpcVisible" :row="frpcRow" :frps="frpsConfig" />
    <FrpsConfigDialog v-model="frpsVisible" @saved="onFrpsSaved" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import TablePagination from '@/components/pagination/TablePagination.vue'
import { POPCONFIRM_CONFIG } from '@/config/elementConfig'
import { VxeGrid } from '@/plugins/vxeGrid'
import TunnelCreate from '@/views/tools/tunnel/create.vue'
import TunnelStatsDialog from '@/views/tools/tunnel/TunnelStatsDialog.vue'
import FrpcConfigDialog from '@/views/tools/tunnel/FrpcConfigDialog.vue'
import FrpsConfigDialog from '@/views/tools/tunnel/FrpsConfigDialog.vue'
import { listTunnels, deleteTunnel, updateTunnel, getFrpsConfig } from '@/api/tunnel'
import { TunnelType, TunnelStatus, type TunnelInfo, type FrpsConfig } from '@/types/v1/tunnel'
import type { FormInstance } from 'element-plus'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'TunnelView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const queryFormRef = useTemplateRef<FormInstance>('queryFormRef')
const createRef = useTemplateRef<InstanceType<typeof TunnelCreate> | null>('createRef')

const deleteIds = ref<string[]>([])
const list = ref<TunnelInfo[]>([])
const loading = ref(false)
const statsVisible = ref(false)
const statsRow = ref<TunnelInfo | null>(null)
const frpcVisible = ref(false)
const frpcRow = ref<TunnelInfo | null>(null)
const frpsVisible = ref(false)
const frpsConfig = ref<FrpsConfig | null>(null)

const typeOptions = [
  TunnelType.TUNNEL_TYPE_TCP,
  TunnelType.TUNNEL_TYPE_UDP,
  TunnelType.TUNNEL_TYPE_HTTP,
  TunnelType.TUNNEL_TYPE_HTTPS,
  TunnelType.TUNNEL_TYPE_STCP,
  TunnelType.TUNNEL_TYPE_XTCP,
  TunnelType.TUNNEL_TYPE_TCPMUX,
]

const typeLabel = (type: TunnelType) => type.replace('TUNNEL_TYPE_', '')

const queryForm = ref({
  keywords: '',
  type: undefined as TunnelType | undefined,
  isEnable: undefined as boolean | undefined,
})

const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const formatBytes = (bytes?: number | string): string => {
  const n = Number(bytes)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

const statusInfo = (status: TunnelStatus): { type: 'success' | 'warning' | 'info'; label: string } => {
  switch (status) {
    case TunnelStatus.TUNNEL_STATUS_ONLINE:
      return { type: 'success', label: t('tools.tunnel.status.online') }
    case TunnelStatus.TUNNEL_STATUS_PENDING:
      return { type: 'warning', label: t('tools.tunnel.status.pending') }
    default:
      return { type: 'info', label: t('tools.tunnel.status.offline') }
  }
}

const remoteText = (row: TunnelInfo): string => {
  if ([TunnelType.TUNNEL_TYPE_HTTP, TunnelType.TUNNEL_TYPE_HTTPS].includes(row.type)) {
    return row.customDomains || row.subdomain || '-'
  }
  return row.remotePort ? `:${row.remotePort}` : '-'
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
    { field: 'name', title: t('tools.tunnel.name'), minWidth: 100, fixed: 'left' },
    { field: 'type', title: t('tools.tunnel.proxyType'), width: 90, slots: { default: 'column-type' } },
    { field: 'status', title: t('system.common.status'), width: 90, slots: { default: 'column-status' } },
    { title: t('tools.tunnel.remotePort'), minWidth: 140, slots: { default: 'column-remote' } },
    { title: t('tools.tunnel.target'), minWidth: 160, slots: { default: 'column-target' } },
    { field: 'isEnable', title: t('system.common.enabled'), width: 80, slots: { default: 'column-isEnable' } },
    { title: t('tools.tunnel.stats.connections'), width: 90, slots: { default: 'column-connections' } },
    { title: t('tools.tunnel.stats.traffic'), minWidth: 170, slots: { default: 'column-traffic' } },
    { field: 'createTime', title: t('system.common.createTime'), minWidth: 180 },
    { title: t('system.common.operation'), width: 280, fixed: 'right', showOverflow: false, slots: { default: 'column-operation' } },
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
    const { data } = await listTunnels({
      page: pagination.value.page,
      pageSize: pagination.value.pageSize,
      keywords: queryForm.value.keywords || undefined,
      type: queryForm.value.type || undefined,
      isEnable: queryForm.value.isEnable,
    })
    list.value = data?.infos || []
    pagination.value.total = Number(data?.total) || 0
  } finally {
    loading.value = false
  }
}

const loadFrpsConfig = async () => {
  try {
    const { data } = await getFrpsConfig()
    frpsConfig.value = data?.config || null
  } catch {
    // 非致命：frpc 配置弹窗会回退到占位地址
  }
}

const onFrpsSaved = (config: FrpsConfig) => {
  frpsConfig.value = config
  getList()
}

const selectionChange = ({ records }: { records: TunnelInfo[] }) => {
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

const openCreateDialog = () => createRef.value?.showDialog()
const openEditDialog = (row: TunnelInfo) => createRef.value?.showDialog(row)

const openStatsDialog = (row: TunnelInfo) => {
  statsRow.value = row
  statsVisible.value = true
}

const openFrpcDialog = (row: TunnelInfo) => {
  frpcRow.value = row
  frpcVisible.value = true
}

const toggleEnable = async (row: TunnelInfo, val: boolean) => {
  try {
    const { data } = await updateTunnel({ id: row.id, isEnable: val })
    if (data?.info) {
      row.isEnable = data.info.isEnable
      row.status = data.info.status
      ElMessage.success(data.info.isEnable ? t('tools.tunnel.enabled') : t('tools.tunnel.disabled'))
    }
  } catch {
    // error handled by interceptor
  }
}

const confirmDelete = (row: TunnelInfo) => {
  ElMessageBox.confirm(t('tools.tunnel.confirmDeleteRule'), t('tools.tunnel.confirmDeleteTitle'), {
    type: 'warning',
    confirmButtonText: t('tools.tunnel.confirmDeleteText'),
    cancelButtonText: t('system.common.cancel'),
  })
    .then(async () => {
      await deleteTunnel({ id: row.id })
      ElMessage.success(t('system.common.deleteSuccess'))
      deleteIds.value = deleteIds.value.filter((d) => d !== row.id)
      getList()
    })
    .catch(() => { /* cancelled */ })
}

const batchDelete = async () => {
  if (!deleteIds.value.length) return
  try {
    await Promise.all(deleteIds.value.map((id) => deleteTunnel({ id })))
    ElMessage.success(t('tools.tunnel.batchDeleteSuccess'))
    deleteIds.value = []
    getList()
  } catch {
    // error handled by interceptor
  }
}

const refresh = (type: 'create' | 'update') => {
  if (type === 'create') pagination.value.page = 1
  getList()
}

onMounted(() => {
  getList()
  loadFrpsConfig()
})
</script>

<style scoped lang="scss">
.tn-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  white-space: nowrap;
}

.tn-actions :deep(.el-button) {
  margin-left: 0;
}

.mono-text {
  font-family: 'Cascadia Code', 'Fira Code', 'JetBrains Mono', monospace;
  font-size: 0.88rem;
}

.tn-traffic {
  display: inline-flex;
  flex-direction: column;
  line-height: 1.3;
  font-size: 0.8rem;
}

.tn-traffic-item {
  white-space: nowrap;
  color: var(--el-text-color-regular);
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

.tn-card {
  position: relative;
  padding: 0.75rem 0.85rem;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.65rem;
  background: var(--el-bg-color);
  transition: border-color 0.15s;
}

.tn-card.is-selected {
  border-color: var(--el-color-primary);
  background: color-mix(in srgb, var(--el-color-primary) 4%, var(--el-bg-color));
}

.tn-card-check {
  position: absolute;
  top: 0.6rem;
  right: 0.7rem;
  z-index: 1;
}

.tn-card-header {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding-right: 2rem;
}

.tn-card-name {
  font-size: 0.9rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.tn-card-body {
  margin-top: 0.55rem;
  display: flex;
  flex-direction: column;
  gap: 0.3rem;
}

.tn-card-meta {
  display: flex;
  align-items: center;
  gap: 0.3rem;
  font-size: 0.76rem;
  color: var(--el-text-color-secondary);
}

.tn-card-time {
  font-size: 0.7rem;
  color: var(--el-text-color-placeholder);
}

.tn-card-footer {
  display: flex;
  flex-wrap: wrap;
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
