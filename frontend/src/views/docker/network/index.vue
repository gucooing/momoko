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
            <el-form-item :label="t('docker.common.driver')">
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
        <el-button type="warning" :disabled="!canManage" @click="handlePrune">{{ t('docker.network.pruneUnused') }}</el-button>
        <el-button :icon="menuStore.iconComponents['HOutline:ClockIcon']" @click="openTasks">{{ t('docker.common.tasks') }}</el-button>
      </div>

      <div v-loading="loading">
        <VxeGrid v-if="!menuStore.isMobile" v-bind="gridConfig">
          <template #column-driver="{ row }: { row: DockerNetworkInfo }">
            <BaseTag :text="row.driver" type="info" />
          </template>
          <template #column-scope="{ row }: { row: DockerNetworkInfo }">
            <BaseTag :text="row.scope" :type="row.scope === 'local' ? 'success' : 'warning'" />
          </template>
          <template #column-containers="{ row }: { row: DockerNetworkInfo }">
            {{ row.containers ? Object.keys(row.containers).length : 0 }}
          </template>
          <template #column-operation="{ row }: { row: DockerNetworkInfo }">
            <el-button type="primary" link size="small" @click="openDetail(row)">{{ t('docker.common.detail') }}</el-button>
            <template v-if="canManage">
              <el-button type="primary" link size="small" @click="openEdit(row)">{{ t('docker.common.edit') }}</el-button>
              <el-button type="primary" link size="small" @click="openConnect(row)">{{ t('docker.network.connectContainer') }}</el-button>
              <el-button type="danger" link size="small" @click="handleDelete(row)">{{ t('docker.common.delete') }}</el-button>
            </template>
          </template>
        </VxeGrid>

        <div v-else-if="list.length" class="mobile-card-list">
          <div v-for="row in list" :key="row.id" class="mobile-card">
            <div class="mobile-card-body">
              <div class="mobile-card-header">
                <span class="mobile-card-title">{{ row.name }}</span>
                <BaseTag :text="row.driver" type="info" />
              </div>
              <div class="mobile-card-meta"><span>{{ t('docker.image.idMeta', { id: row.id?.slice(0, 12) || '-' }) }}</span></div>
              <div class="mobile-card-meta"><span>{{ t('docker.network.containerMeta', { count: row.containers ? Object.keys(row.containers).length : 0 }) }}</span></div>
            </div>
            <div class="mobile-card-actions">
              <el-button size="small" plain type="primary" @click="openDetail(row)">{{ t('docker.common.detail') }}</el-button>
              <el-button v-if="canManage" size="small" plain type="danger" @click="handleDelete(row)">{{ t('docker.common.delete') }}</el-button>
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
            <el-form-item :label="t('docker.common.driver')">
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
    <BaseDialog v-model="detailVisible" :title="t('docker.network.networkDetail')" width="800">
      <div v-if="detail" v-loading="detailLoading">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="t('docker.common.id')">{{ detail.id }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.name')">{{ detail.name }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.driver')">{{ detail.driver }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.scope')">{{ detail.scope }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.network.internal')">{{ detail.internal ? t('docker.common.yes') : t('docker.common.no') }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.network.attachable')">{{ detail.attachable ? t('docker.common.yes') : t('docker.common.no') }}</el-descriptions-item>
          <el-descriptions-item label="IPv6">{{ detail.enableIpv6 ? t('docker.common.enabled') : t('docker.common.notEnabled') }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.createdAt')">{{ detail.created }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="detail.ipam?.config?.length" style="margin-top: 12px">
          <h4>{{ t('docker.network.ipamConfig') }}</h4>
          <el-table :data="detail.ipam.config" size="small" border>
            <el-table-column prop="subnet" :label="t('docker.network.subnet')" />
            <el-table-column prop="gateway" :label="t('docker.network.gateway')" />
            <el-table-column prop="ipRange" :label="t('docker.network.ipRange')" />
          </el-table>
        </div>
        <div v-if="detail.containers && Object.keys(detail.containers).length" style="margin-top: 12px">
          <h4>{{ t('docker.network.connectedContainers') }}</h4>
          <el-table :data="Object.entries(detail.containers).map(([k, v]) => ({ ...v, containerId: k }))" size="small" border>
            <el-table-column prop="containerId" :label="t('docker.network.containerId')" width="150" show-overflow-tooltip />
            <el-table-column prop="name" :label="t('docker.common.name')" />
            <el-table-column prop="ipv4Address" label="IPv4" width="160" />
            <el-table-column prop="macAddress" label="MAC" width="150" />
            <el-table-column v-if="canManage" :label="t('docker.common.operation')" width="80">
              <template #default="{ row: c }">
                <el-button type="danger" link size="small" @click="handleDisconnect(detail, c.containerId)">{{ t('docker.network.disconnect') }}</el-button>
              </template>
            </el-table-column>
          </el-table>
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
              <el-form-item :label="t('docker.common.driver')">
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

    <!-- Connect Dialog -->
    <BaseDialog v-model="connectVisible" :title="t('docker.network.connectContainerDialog')" width="450">
      <el-form label-position="top">
        <el-form-item :label="t('docker.network.containerIdOrName')" required>
          <el-input v-model="connectForm.containerId" :placeholder="t('docker.network.containerIdOrNamePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('docker.network.ipv4Address')">
          <el-input v-model="connectForm.ipv4Address" :placeholder="t('docker.network.ipv4Placeholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="connectVisible = false">{{ t('docker.common.cancel') }}</el-button>
        <el-button type="primary" :loading="connectSubmitting" @click="submitConnect">{{ t('docker.common.connect') }}</el-button>
      </template>
    </BaseDialog>

    <DockerTaskDialogs v-model="taskDialogsVisible" :active-task="activeTask" @finished="handleTaskFinished" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  connectDockerNetwork, createDockerNetwork, deleteDockerNetwork,
  disconnectDockerNetwork, getDockerNetwork, listDockerNetworks, pruneDockerNetworks,
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

const gridConfig = computed<VxeGridProps>(() => ({
  border: true, showOverflow: true, rowConfig: { isHover: true },
  data: list.value,
  columns: [
    { type: 'seq', title: '#', width: 50, fixed: 'left' },
    { field: 'id', title: 'ID', width: 150, showOverflow: true },
    { field: 'name', title: t('docker.common.name'), minWidth: 160 },
    { field: 'driver', title: t('docker.common.driver'), width: 100, slots: { default: 'column-driver' } },
    { field: 'scope', title: t('docker.common.scope'), width: 90, slots: { default: 'column-scope' } },
    { field: 'containers', title: t('docker.image.containerCount'), width: 80, slots: { default: 'column-containers' } },
    { title: t('docker.common.operation'), width: canManage.value ? 230 : 90, fixed: 'right', slots: { default: 'column-operation' } },
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

// -- prune --
const handlePrune = async () => {
  try { await Dialog.confirm({ title: t('docker.network.pruneTitle'), content: t('docker.network.pruneContent'), confirmText: t('docker.network.pruneConfirm'), cancelText: t('docker.common.cancel') }) }
  catch { return }
  try {
    const { data } = await pruneDockerNetworks({})
    ElMessage.success(t('docker.common.taskCreated'))
    openTask(data?.task)
  }
  catch (e) { showRequestError(e, t('docker.network.pruneFailed')) }
}

// -- detail --
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<DockerNetworkInfo | null>(null)
const openDetail = async (row: DockerNetworkInfo) => {
  detailVisible.value = true; detail.value = null; detailLoading.value = true
  try { const { data } = await getDockerNetwork({ id: row.id }); detail.value = data?.info || null }
  catch (e) { showRequestError(e, t('docker.network.getDetailFailed')) }
  finally { detailLoading.value = false }
}

// -- connect --
const connectVisible = ref(false)
const connectSubmitting = ref(false)
const connectNetworkId = ref('')
const connectForm = reactive({ containerId: '', ipv4Address: '' })
const openConnect = (row: DockerNetworkInfo) => {
  connectNetworkId.value = row.id
  Object.assign(connectForm, { containerId: '', ipv4Address: '' })
  connectVisible.value = true
}
const submitConnect = async () => {
  const containerId = connectForm.containerId.trim()
  if (!containerId) { ElMessage.error(t('docker.network.enterContainerId')); return }
  connectSubmitting.value = true
  try {
    await connectDockerNetwork({
      networkId: connectNetworkId.value, containerId,
      aliases: [], ipv4Address: connectForm.ipv4Address.trim(), ipv6Address: '',
    })
    ElMessage.success(t('docker.network.connectSuccess'))
    connectVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, t('docker.network.connectFailed')) }
  finally { connectSubmitting.value = false }
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
h4 { margin: 0 0 8px; font-size: 0.9rem; }

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
