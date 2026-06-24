<template>
  <div class="docker-page">
    <el-card shadow="never">
      <el-form :model="queryForm" label-width="auto" @keyup.enter="search">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item :label="t('docker.common.name')">
              <el-input v-model="queryForm.name" :placeholder="t('docker.network.namePlaceholder')" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item :label="t('docker.common.mode')">
              <el-select v-model="queryForm.driver" :placeholder="t('docker.common.all')" clearable style="width: 100%">
                <el-option label="bridge" value="bridge" />
                <el-option label="host" value="host" />
                <el-option label="overlay" value="overlay" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item>
              <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="search">{{ t('docker.common.search') }}</el-button>
              <el-button :icon="menuStore.iconComponents.Refresh" @click="reset">{{ t('docker.common.reset') }}</el-button>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <el-card shadow="never" class="card-mt-16">
      <div class="operation-container">
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" :disabled="!canManage" @click="openCreate">{{ t('docker.network.createNetwork') }}</el-button>
        <el-button :icon="menuStore.iconComponents['HOutline:ClockIcon']" @click="openTasks">{{ t('docker.common.tasks') }}</el-button>
      </div>

      <div v-loading="loading">
        <VxeGrid v-if="!menuStore.isMobile" v-bind="gridConfig">
          <template #column-driver="{ row }: { row: DockerNetworkInfo }">
            <BaseTag :text="row.driver || '-'" type="info" />
          </template>
          <template #column-subnet="{ row }: { row: DockerNetworkInfo }">
            <span>{{ networkSubnets(row).join(', ') || '-' }}</span>
          </template>
          <template #column-gateway="{ row }: { row: DockerNetworkInfo }">
            <span>{{ networkGateways(row).join(', ') || '-' }}</span>
          </template>
          <template #column-labels="{ row }: { row: DockerNetworkInfo }">
            <div v-if="labelEntries(row.labels).length" class="label-cell">
              <span
                v-for="[key, value] in labelEntries(row.labels).slice(0, 2)"
                :key="key"
                class="label-chip"
                :title="`${key}=${value}`"
              >{{ key }}={{ value }}</span>
              <span v-if="labelEntries(row.labels).length > 2" class="text-muted">+{{ labelEntries(row.labels).length - 2 }}</span>
            </div>
            <span v-else class="text-muted">-</span>
          </template>
          <template #column-created="{ row }: { row: DockerNetworkInfo }">
            <span>{{ formatTime(row.created) }}</span>
          </template>
          <template #column-operation="{ row }: { row: DockerNetworkInfo }">
            <el-button type="primary" link size="small" @click="openDetail(row)">{{ t('docker.common.detail') }}</el-button>
            <template v-if="canManage">
              <el-button type="primary" link size="small" @click="openEdit(row)">{{ t('docker.common.edit') }}</el-button>
              <el-button v-if="canDeleteNetwork(row)" type="danger" link size="small" @click="handleDelete(row)">{{ t('docker.common.delete') }}</el-button>
            </template>
          </template>
        </VxeGrid>

        <div v-else-if="list.length" class="mobile-card-list">
          <div v-for="row in list" :key="row.id" class="mobile-card">
            <div class="mobile-card-body">
              <div class="mobile-card-header">
                <span class="mobile-card-title">{{ row.name }}</span>
                <BaseTag :text="row.driver || '-'" type="info" />
              </div>
              <div class="mobile-card-meta"><span>{{ t('docker.network.subnet') }}：{{ networkSubnets(row).join(', ') || '-' }}</span></div>
              <div class="mobile-card-meta"><span>{{ t('docker.network.containerMeta', { count: networkContainerCount(row) }) }}</span></div>
            </div>
            <div class="mobile-card-actions">
              <el-button size="small" plain type="primary" @click="openDetail(row)">{{ t('docker.common.detail') }}</el-button>
              <el-button v-if="canManage && canDeleteNetwork(row)" size="small" plain type="danger" @click="handleDelete(row)">{{ t('docker.common.delete') }}</el-button>
            </div>
          </div>
        </div>
        <el-empty v-else-if="!loading" :description="t('docker.network.noNetworks')" />
      </div>

      <TablePagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :is-mobile="menuStore.isMobile"
        @change="getList"
      />
    </el-card>

    <!-- Create Dialog -->
    <BaseDialog v-model="createVisible" :title="t('docker.network.createNetwork')" width="550">
      <el-form :model="createForm" label-position="top">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item :label="t('docker.common.name')" required>
              <el-input v-model="createForm.name" :placeholder="t('docker.network.namePlaceholder')" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('docker.common.mode')">
              <el-select v-model="createForm.driver" style="width: 100%">
                <el-option label="bridge" value="bridge" />
                <el-option label="host" value="host" />
                <el-option label="overlay" value="overlay" />
              </el-select>
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="8">
            <el-form-item :label="t('docker.network.internal')"><el-switch v-model="createForm.internal" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="IPv6"><el-switch v-model="createForm.enableIpv6" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item :label="t('docker.network.attachable')"><el-switch v-model="createForm.attachable" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item :label="t('docker.network.subnet')">
              <el-input v-model="createForm.subnet" :placeholder="t('docker.network.subnetPlaceholder')" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item :label="t('docker.network.gateway')">
              <el-input v-model="createForm.gateway" :placeholder="t('docker.network.gatewayPlaceholder')" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">{{ t('docker.common.cancel') }}</el-button>
        <el-button type="primary" :loading="createSubmitting" @click="submitCreate">{{ t('docker.common.create') }}</el-button>
      </template>
    </BaseDialog>

    <!-- Detail Dialog -->
    <BaseDialog v-model="detailVisible" :title="t('docker.network.networkDetail')" width="820">
      <div v-if="detail" v-loading="detailLoading" class="network-detail">
        <div class="detail-hero">
          <div class="detail-hero__main">
            <div class="detail-hero__title">
              <span>{{ detail.name || '-' }}</span>
              <BaseTag :text="detail.driver || '-'" type="info" />
              <BaseTag :text="detail.scope || '-'" :type="detail.scope === 'local' ? 'success' : 'warning'" />
            </div>
            <div class="detail-hero__sub">
              <span>{{ shortId(detail.id) }}</span>
              <span>{{ formatTime(detail.created) }}</span>
            </div>
          </div>
          <div class="detail-hero__flags">
            <span v-if="detail.internal" class="flag-tag">{{ t('docker.network.internal') }}</span>
            <span v-if="detail.attachable" class="flag-tag">{{ t('docker.network.attachable') }}</span>
            <span v-if="detail.enableIpv6" class="flag-tag">IPv6</span>
          </div>
        </div>

        <div class="detail-section">
          <div class="detail-section__title">{{ t('docker.common.basicInfo') }}</div>
          <div class="detail-kv">
            <div><span>{{ t('docker.common.mode') }}</span><strong>{{ detail.driver || '-' }}</strong></div>
            <div><span>{{ t('docker.common.scope') }}</span><strong>{{ detail.scope || '-' }}</strong></div>
            <div><span>{{ t('docker.network.internal') }}</span><strong>{{ detail.internal ? t('docker.common.yes') : t('docker.common.no') }}</strong></div>
            <div><span>{{ t('docker.network.attachable') }}</span><strong>{{ detail.attachable ? t('docker.common.yes') : t('docker.common.no') }}</strong></div>
            <div><span>IPv6</span><strong>{{ detail.enableIpv6 ? t('docker.common.enabled') : t('docker.common.notEnabled') }}</strong></div>
            <div><span>{{ t('docker.common.createdAt') }}</span><strong>{{ formatTime(detail.created) }}</strong></div>
          </div>
        </div>

        <div v-if="detail.ipam?.config?.length" class="detail-section">
          <div class="detail-section__title">{{ t('docker.network.ipamConfig') }}</div>
          <div class="detail-list detail-list--ipam">
            <div class="detail-list__head">
              <span>{{ t('docker.network.subnet') }}</span>
              <span>{{ t('docker.network.gateway') }}</span>
              <span>{{ t('docker.network.ipRange') }}</span>
            </div>
            <div v-for="(item, index) in detail.ipam.config" :key="index" class="detail-list__row">
              <span>{{ item.subnet || '-' }}</span>
              <span>{{ item.gateway || '-' }}</span>
              <span>{{ item.ipRange || '-' }}</span>
            </div>
          </div>
        </div>

        <div v-if="detailLabels.length" class="detail-section">
          <div class="detail-section__title">{{ t('docker.common.labels') }}</div>
          <div class="label-chips">
            <span v-for="[key, value] in detailLabels" :key="key" class="label-chip label-chip--full">{{ key }}={{ value }}</span>
          </div>
        </div>

        <div class="detail-section">
          <div class="detail-section__title">{{ t('docker.network.connectedContainers') }} ({{ detailContainers.length }})</div>
          <div v-if="detailContainers.length" class="detail-list detail-list--conn" :class="{ 'detail-list--conn-manage': canManage }">
            <div class="detail-list__head">
              <span>{{ t('docker.common.name') }}</span>
              <span>IPv4</span>
              <span>MAC</span>
              <span v-if="canManage"></span>
            </div>
            <div v-for="item in detailContainers" :key="item.containerId" class="detail-list__row">
              <span class="detail-list__name" :title="item.containerId">{{ item.name || '-' }}</span>
              <span>{{ item.ipv4Address || '-' }}</span>
              <span>{{ item.macAddress || '-' }}</span>
              <span v-if="canManage">
                <el-button type="danger" link size="small" @click="handleDisconnect(detail, item.containerId)">{{ t('docker.network.disconnect') }}</el-button>
              </span>
            </div>
          </div>
          <div v-else class="detail-empty">{{ t('docker.common.noData') }}</div>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">{{ t('docker.common.close') }}</el-button>
      </template>
    </BaseDialog>

    <!-- Edit Dialog -->
    <BaseDialog v-model="editVisible" :title="t('docker.network.editNetwork')" width="550">
      <el-form label-position="top">
        <el-form-item :label="t('docker.common.labels')">
          <el-input v-model="editLabelsText" :placeholder="t('docker.common.lineKvPlaceholder')" type="textarea" :rows="2" />
        </el-form-item>
        <el-divider />
        <el-checkbox v-model="editRecreate" style="margin-bottom: 8px">{{ t('docker.network.recreateTip') }}</el-checkbox>
        <template v-if="editRecreate">
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item :label="t('docker.common.mode')">
                <el-select v-model="editDriver" style="width: 100%">
                  <el-option label="bridge" value="bridge" />
                  <el-option label="host" value="host" />
                  <el-option label="overlay" value="overlay" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item :label="t('docker.common.forceRecreate')">
                <el-switch v-model="editForce" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item :label="t('docker.network.subnet')">
                <el-input v-model="editSubnet" :placeholder="t('docker.network.subnetPlaceholder')" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item :label="t('docker.network.gateway')">
                <el-input v-model="editGateway" :placeholder="t('docker.network.gatewayPlaceholder')" />
              </el-form-item>
            </el-col>
          </el-row>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">{{ t('docker.common.cancel') }}</el-button>
        <el-button type="primary" :loading="editSubmitting" @click="submitEdit">{{ t('docker.common.save') }}</el-button>
      </template>
    </BaseDialog>

    <DockerTaskDialogs v-model="taskDialogsVisible" :active-task="activeTask" @finished="handleTaskFinished" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  createDockerNetwork, deleteDockerNetwork,
  disconnectDockerNetwork, getDockerNetwork, listDockerNetworks,
  updateDockerNetwork,
} from '@/api/docker'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import DockerTaskDialogs from '@/views/docker/components/DockerTaskDialogs.vue'
import BaseTag from '@/components/tag/BaseTag.vue'
import TablePagination from '@/components/pagination/TablePagination.vue'
import { VxeGrid } from '@/plugins/vxeGrid'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import type { DockerNetworkInfo, DockerTaskInfo } from '@/types/v1/docker'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'DockerNetworkView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const canManage = useButtonPermission([PERM.DOCKER_NETWORK_MANAGE], [])

const list = ref<DockerNetworkInfo[]>([])
const loading = ref(false)
const queryForm = reactive({ name: '', driver: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const pad = (n: number) => String(n).padStart(2, '0')
const formatTime = (value?: string) => {
  if (!value || value.startsWith('0001-')) return '-'
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}
const shortId = (id?: string) => (id ? id.slice(0, 12) : '-')
const labelEntries = (labels?: Record<string, string>) => Object.entries(labels || {})
const networkSubnets = (row: DockerNetworkInfo) => (row.ipam?.config || []).map(item => item.subnet).filter(Boolean)
const networkGateways = (row: DockerNetworkInfo) => (row.ipam?.config || []).map(item => item.gateway).filter(Boolean)
const networkContainerCount = (row: DockerNetworkInfo) => (row.containers ? Object.keys(row.containers).length : 0)
const canDeleteNetwork = (row: DockerNetworkInfo) => networkContainerCount(row) === 0

const gridConfig = computed<VxeGridProps>(() => ({
  border: true, showOverflow: true, rowConfig: { isHover: true },
  data: list.value,
  columns: [
    { type: 'seq', title: '#', width: 50, fixed: 'left' },
    { field: 'name', title: t('docker.common.name'), minWidth: 160, showOverflow: true },
    { field: 'driver', title: t('docker.common.mode'), width: 110, slots: { default: 'column-driver' } },
    { field: 'subnet', title: t('docker.network.subnet'), minWidth: 150, slots: { default: 'column-subnet' } },
    { field: 'gateway', title: t('docker.network.gateway'), minWidth: 140, slots: { default: 'column-gateway' } },
    { field: 'labels', title: t('docker.common.labels'), minWidth: 200, showOverflow: false, slots: { default: 'column-labels' } },
    { field: 'created', title: t('docker.common.createdAt'), width: 170, slots: { default: 'column-created' } },
    { title: t('docker.common.operation'), width: canManage.value ? 170 : 80, fixed: 'right', slots: { default: 'column-operation' } },
  ],
}))

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listDockerNetworks({
      page: pagination.value.page, pageSize: pagination.value.pageSize,
      name: queryForm.name || '', driver: queryForm.driver || '',
      scope: '', labels: {},
    })
    list.value = data?.items || []
    pagination.value.total = Number(data?.total || 0)
  } catch {
    list.value = []
    pagination.value.total = 0
  } finally { loading.value = false }
}

const search = () => { pagination.value.page = 1; getList() }
const reset = () => { queryForm.name = ''; queryForm.driver = ''; search() }

const taskDialogsVisible = ref(false)
const activeTask = ref<DockerTaskInfo | null>(null)
const openTasks = () => { taskDialogsVisible.value = true }
const openTask = (task: DockerTaskInfo | undefined) => {
  if (!task?.id) return
  activeTask.value = task
  taskDialogsVisible.value = true
}
const handleTaskFinished = async () => {
  await getList()
}

// -- create --
const createVisible = ref(false)
const createSubmitting = ref(false)
const createForm = reactive({ name: '', driver: 'bridge', internal: false, enableIpv6: false, attachable: false, subnet: '', gateway: '' })
const openCreate = () => {
  Object.assign(createForm, { name: '', driver: 'bridge', internal: false, enableIpv6: false, attachable: false, subnet: '', gateway: '' })
  createVisible.value = true
}
const submitCreate = async () => {
  const name = createForm.name.trim()
  if (!name) { ElMessage.error(t('docker.network.enterName')); return }
  createSubmitting.value = true
  try {
    const ipamConfig = createForm.subnet.trim() ? [{ subnet: createForm.subnet.trim(), gateway: createForm.gateway.trim(), ipRange: '', auxAddress: {} }] : []
    await createDockerNetwork({
      options: {
        name, driver: createForm.driver, internal: createForm.internal,
        enableIpv6: createForm.enableIpv6, attachable: createForm.attachable,
        ingress: false, scope: '', enableIpv4: undefined,
        ipam: ipamConfig.length ? { driver: 'default', options: {}, config: ipamConfig } : undefined,
        options: {}, labels: {},
      },
    })
    ElMessage.success(t('docker.network.createSuccess'))
    createVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, t('docker.network.createFailed')) }
  finally { createSubmitting.value = false }
}

// -- delete --
const handleDelete = async (row: DockerNetworkInfo) => {
  try { await Dialog.confirm({ title: t('docker.network.confirmDeleteTitle'), content: t('docker.network.confirmDeleteContent', { name: row.name }), confirmText: t('docker.network.confirmDeleteText'), cancelText: t('docker.common.cancel') }) }
  catch { return }
  try { await deleteDockerNetwork({ id: row.id }); ElMessage.success(t('docker.common.deletedName', { name: row.name })); await getList() }
  catch (e) { showRequestError(e, t('docker.network.deleteFailed')) }
}

// -- detail --
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<DockerNetworkInfo | null>(null)
const detailLabels = computed(() => Object.entries(detail.value?.labels || {}))
const detailContainers = computed(() =>
  Object.entries(detail.value?.containers || {}).map(([id, item]) => ({ ...item, containerId: id })),
)
const openDetail = async (row: DockerNetworkInfo) => {
  detailVisible.value = true; detail.value = null; detailLoading.value = true
  try { const { data } = await getDockerNetwork({ id: row.id }); detail.value = data?.info || null }
  catch (e) { showRequestError(e, t('docker.network.getDetailFailed')) }
  finally { detailLoading.value = false }
}

const handleDisconnect = async (network: DockerNetworkInfo, containerId: string) => {
  try {
    await Dialog.confirm({ title: t('docker.network.disconnectTitle'), content: t('docker.network.disconnectContent', { id: containerId.slice(0, 12) }), confirmText: t('docker.network.disconnectConfirm'), cancelText: t('docker.common.cancel') })
  } catch { return }
  try {
    await disconnectDockerNetwork({ networkId: network.id, containerId, force: false })
    ElMessage.success(t('docker.network.disconnectSuccess'))
    const { data } = await getDockerNetwork({ id: network.id })
    detail.value = data?.info || null
    await getList()
  } catch (e) { showRequestError(e, t('docker.network.disconnectFailed')) }
}

// -- edit --
const editVisible = ref(false)
const editSubmitting = ref(false)
const editId = ref('')
const editLabelsText = ref('')
const editRecreate = ref(false)
const editForce = ref(false)
const editDriver = ref('bridge')
const editSubnet = ref('')
const editGateway = ref('')
const openEdit = async (row: DockerNetworkInfo) => {
  editId.value = row.id; editRecreate.value = false; editForce.value = false
  editDriver.value = 'bridge'; editSubnet.value = ''; editGateway.value = ''
  editVisible.value = true
  try {
    const { data } = await getDockerNetwork({ id: row.id })
    const info = data?.info
    editLabelsText.value = info?.labels ? Object.entries(info.labels).map(([k, v]) => `${k}=${v}`).join('\n') : ''
    editDriver.value = info?.driver || 'bridge'
  } catch { editLabelsText.value = '' }
}
const parseLabels = (text: string) => {
  const obj: Record<string, string> = {}
  text.trim().split('\n').filter(Boolean).forEach(line => {
    const idx = line.indexOf('=')
    if (idx > 0) obj[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
  })
  return obj
}
const submitEdit = async () => {
  editSubmitting.value = true
  try {
    if (editRecreate.value) {
      const ipamConfig = editSubnet.value.trim() ? [{ subnet: editSubnet.value.trim(), gateway: editGateway.value.trim(), ipRange: '', auxAddress: {} }] : []
      const { data } = await updateDockerNetwork({
        id: editId.value, labels: parseLabels(editLabelsText.value), force: editForce.value,
        options: {
          name: '', driver: editDriver.value, scope: '', enableIpv4: undefined, enableIpv6: undefined,
          internal: false, attachable: false, ingress: false,
          ipam: ipamConfig.length ? { driver: 'default', options: {}, config: ipamConfig } : undefined,
          options: {}, labels: parseLabels(editLabelsText.value),
        },
      })
      openTask(data?.task)
    } else {
      await updateDockerNetwork({ id: editId.value, labels: parseLabels(editLabelsText.value), force: false, options: undefined })
    }
    ElMessage.success(editRecreate.value ? t('docker.network.recreateTaskCreated') : t('docker.network.updateSuccess'))
    editVisible.value = false
    if (!editRecreate.value) await getList()
  } catch (e) { showRequestError(e, t('docker.network.updateFailed')) }
  finally { editSubmitting.value = false }
}

onMounted(() => { getList() })
</script>

<style scoped lang="scss">
.docker-page { .card-mt-16 { margin-top: 16px; } }
.operation-container { margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap; }
.text-muted { color: var(--el-text-color-placeholder); font-size: 0.82rem; }

.label-cell {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}
.label-chip {
  display: inline-block;
  max-width: 150px;
  padding: 1px 6px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  border-radius: 4px;
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 0.72rem;
}

/* -- detail -- */
.network-detail {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}
.detail-hero {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 0.75rem;
  padding: 0.75rem;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  background: var(--el-fill-color-lighter);
}
.detail-hero__main { min-width: 0; flex: 1; }
.detail-hero__title {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.detail-hero__title > span:first-child {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.detail-hero__sub {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.35rem;
  font-size: 0.78rem;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.detail-hero__flags {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 0.35rem;
}
.flag-tag {
  padding: 1px 8px;
  border-radius: 10px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-size: 0.72rem;
  font-weight: 600;
}
.detail-section { min-width: 0; }
.detail-section__title {
  margin-bottom: 0.4rem;
  font-size: 0.86rem;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.detail-kv {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  overflow: hidden;
}
.detail-kv > div {
  min-width: 0;
  padding: 0.45rem 0.55rem;
  border-right: 1px solid var(--el-border-color-extra-light);
  border-bottom: 1px solid var(--el-border-color-extra-light);
}
.detail-kv > div:nth-child(3n) { border-right: 0; }
.detail-kv > div:nth-last-child(-n + 3) { border-bottom: 0; }
.detail-kv span {
  display: block;
  color: var(--el-text-color-secondary);
  font-size: 0.72rem;
}
.detail-kv strong {
  display: block;
  margin-top: 0.12rem;
  color: var(--el-text-color-primary);
  font-size: 0.8rem;
  font-weight: 600;
  word-break: break-word;
}
.detail-list {
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 6px;
  overflow: hidden;
  font-size: 0.78rem;
}
.detail-list__head,
.detail-list__row {
  display: grid;
  gap: 0.5rem;
  align-items: center;
  padding: 0.45rem 0.6rem;
}
.detail-list__head {
  color: var(--el-text-color-secondary);
  font-size: 0.72rem;
  background: var(--el-fill-color-lighter);
}
.detail-list__row { border-top: 1px solid var(--el-border-color-extra-light); color: var(--el-text-color-regular); }
.detail-list--ipam .detail-list__head,
.detail-list--ipam .detail-list__row { grid-template-columns: 1.4fr 1fr 1fr; }
.detail-list--conn .detail-list__head,
.detail-list--conn .detail-list__row { grid-template-columns: 1.4fr 1.2fr 1.4fr; }
.detail-list--conn-manage .detail-list__head,
.detail-list--conn-manage .detail-list__row { grid-template-columns: 1.4fr 1.2fr 1.4fr 64px; }
.detail-list__name {
  font-weight: 600;
  color: var(--el-text-color-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.detail-list span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.label-chips { display: flex; flex-wrap: wrap; gap: 0.35rem; }
.label-chip--full { max-width: 100%; }
.detail-empty {
  padding: 0.6rem;
  border: 1px dashed var(--el-border-color-extra-light);
  border-radius: 6px;
  color: var(--el-text-color-placeholder);
  font-size: 0.78rem;
  text-align: center;
}

.mobile-card-list { display: flex; flex-direction: column; gap: 0.55rem; }
.mobile-card {
  display: flex; align-items: flex-start; gap: 0.6rem;
  padding: 0.7rem 0.8rem; border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.6rem; background: var(--el-bg-color);
  .mobile-card-body { flex: 1; min-width: 0; }
  .mobile-card-header { display: flex; align-items: center; gap: 0.5rem; }
  .mobile-card-title { font-size: 0.88rem; font-weight: 700; }
  .mobile-card-meta { margin-top: 0.25rem; font-size: 0.74rem; color: var(--el-text-color-secondary); }
  .mobile-card-actions { display: flex; flex-direction: column; gap: 0.3rem; flex-shrink: 0; }
  .mobile-card-actions .el-button + .el-button { margin-left: 0; }
}
</style>
