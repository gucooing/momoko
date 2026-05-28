<template>
  <div class="docker-page">
    <el-card shadow="never">
      <el-form :model="queryForm" label-width="auto" @keyup.enter="search">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="名称">
              <el-input v-model="queryForm.name" placeholder="网络名称" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="驱动">
              <el-select v-model="queryForm.driver" placeholder="全部" clearable style="width: 100%">
                <el-option label="bridge" value="bridge" />
                <el-option label="host" value="host" />
                <el-option label="overlay" value="overlay" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item>
              <el-button type="primary" :icon="menuStore.iconComponents.Search" @click="search">搜索</el-button>
              <el-button :icon="menuStore.iconComponents.Refresh" @click="reset">重置</el-button>
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
    </el-card>

    <el-card shadow="never" class="card-mt-16">
      <div class="operation-container">
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" :disabled="!canManage" @click="openCreate">创建网络</el-button>
        <el-button type="warning" :disabled="!canManage" @click="handlePrune">清理未使用网络</el-button>
        <el-button :icon="menuStore.iconComponents['HOutline:ClockIcon']" @click="openTasks">任务</el-button>
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
            <el-button type="primary" link size="small" @click="openDetail(row)">详情</el-button>
            <template v-if="canManage">
              <el-button type="primary" link size="small" @click="openEdit(row)">编辑</el-button>
              <el-button type="primary" link size="small" @click="openConnect(row)">连接容器</el-button>
              <el-button type="danger" link size="small" @click="handleDelete(row)">删除</el-button>
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
              <div class="mobile-card-meta"><span>ID：{{ row.id?.slice(0, 12) }}</span></div>
              <div class="mobile-card-meta"><span>容器：{{ row.containers ? Object.keys(row.containers).length : 0 }}</span></div>
            </div>
            <div class="mobile-card-actions">
              <el-button size="small" plain type="primary" @click="openDetail(row)">详情</el-button>
              <el-button v-if="canManage" size="small" plain type="danger" @click="handleDelete(row)">删除</el-button>
            </div>
          </div>
        </div>
        <el-empty v-else-if="!loading" description="暂无网络" />
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
    <BaseDialog v-model="createVisible" title="创建网络" width="550">
      <el-form :model="createForm" label-position="top">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="名称" required>
              <el-input v-model="createForm.name" placeholder="网络名称" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="驱动">
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
            <el-form-item label="内部网络"><el-switch v-model="createForm.internal" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="IPv6"><el-switch v-model="createForm.enableIpv6" /></el-form-item>
          </el-col>
          <el-col :span="8">
            <el-form-item label="可附加"><el-switch v-model="createForm.attachable" /></el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="子网">
              <el-input v-model="createForm.subnet" placeholder="如 172.20.0.0/16" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="网关">
              <el-input v-model="createForm.gateway" placeholder="如 172.20.0.1" />
            </el-form-item>
          </el-col>
        </el-row>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="createSubmitting" @click="submitCreate">创建</el-button>
      </template>
    </BaseDialog>

    <!-- Detail Dialog -->
    <BaseDialog v-model="detailVisible" title="网络详情" width="800">
      <div v-if="detail" v-loading="detailLoading">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="ID">{{ detail.id }}</el-descriptions-item>
          <el-descriptions-item label="名称">{{ detail.name }}</el-descriptions-item>
          <el-descriptions-item label="驱动">{{ detail.driver }}</el-descriptions-item>
          <el-descriptions-item label="作用域">{{ detail.scope }}</el-descriptions-item>
          <el-descriptions-item label="内部网络">{{ detail.internal ? '是' : '否' }}</el-descriptions-item>
          <el-descriptions-item label="可附加">{{ detail.attachable ? '是' : '否' }}</el-descriptions-item>
          <el-descriptions-item label="IPv6">{{ detail.enableIpv6 ? '启用' : '未启用' }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ detail.created }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="detail.ipam?.config?.length" style="margin-top: 12px">
          <h4>IPAM 配置</h4>
          <el-table :data="detail.ipam.config" size="small" border>
            <el-table-column prop="subnet" label="子网" />
            <el-table-column prop="gateway" label="网关" />
            <el-table-column prop="ipRange" label="IP范围" />
          </el-table>
        </div>
        <div v-if="detail.containers && Object.keys(detail.containers).length" style="margin-top: 12px">
          <h4>已连接容器</h4>
          <el-table :data="Object.entries(detail.containers).map(([k, v]) => ({ ...v, containerId: k }))" size="small" border>
            <el-table-column prop="containerId" label="容器ID" width="150" show-overflow-tooltip />
            <el-table-column prop="name" label="名称" />
            <el-table-column prop="ipv4Address" label="IPv4" width="160" />
            <el-table-column prop="macAddress" label="MAC" width="150" />
            <el-table-column v-if="canManage" label="操作" width="80">
              <template #default="{ row: c }">
                <el-button type="danger" link size="small" @click="handleDisconnect(detail, c.containerId)">断开</el-button>
              </template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </BaseDialog>

    <!-- Edit Dialog -->
    <BaseDialog v-model="editVisible" title="编辑网络" width="550">
      <el-form label-position="top">
        <el-form-item label="标签">
          <el-input v-model="editLabelsText" placeholder="key=value&#10;每行一个" type="textarea" :rows="2" />
        </el-form-item>
        <el-divider />
        <el-checkbox v-model="editRecreate" style="margin-bottom: 8px">重建网络（需要修改驱动/子网等时勾选）</el-checkbox>
        <template v-if="editRecreate">
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item label="驱动">
                <el-select v-model="editDriver" style="width: 100%">
                  <el-option label="bridge" value="bridge" />
                  <el-option label="host" value="host" />
                  <el-option label="overlay" value="overlay" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="强制重建">
                <el-switch v-model="editForce" />
              </el-form-item>
            </el-col>
          </el-row>
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item label="子网">
                <el-input v-model="editSubnet" placeholder="如 172.20.0.0/16" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="网关">
                <el-input v-model="editGateway" placeholder="如 172.20.0.1" />
              </el-form-item>
            </el-col>
          </el-row>
        </template>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="editSubmitting" @click="submitEdit">保存</el-button>
      </template>
    </BaseDialog>

    <!-- Connect Dialog -->
    <BaseDialog v-model="connectVisible" title="连接容器到网络" width="450">
      <el-form label-position="top">
        <el-form-item label="容器 ID 或名称" required>
          <el-input v-model="connectForm.containerId" placeholder="容器ID 或名称" />
        </el-form-item>
        <el-form-item label="IPv4 地址">
          <el-input v-model="connectForm.ipv4Address" placeholder="如 172.20.0.10" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="connectVisible = false">取消</el-button>
        <el-button type="primary" :loading="connectSubmitting" @click="submitConnect">连接</el-button>
      </template>
    </BaseDialog>

    <DockerTaskDrawer v-model="taskDrawerVisible" :active-task="activeTask" @finished="handleTaskFinished" />
  </div>
</template>

<script setup lang="ts">
import {
  connectDockerNetwork, createDockerNetwork, deleteDockerNetwork,
  disconnectDockerNetwork, getDockerNetwork, listDockerNetworks, pruneDockerNetworks,
  updateDockerNetwork,
} from '@/api/docker'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import DockerTaskDrawer from '@/views/docker/components/DockerTaskDrawer.vue'
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
    { field: 'name', title: '名称', minWidth: 160 },
    { field: 'driver', title: '驱动', width: 100, slots: { default: 'column-driver' } },
    { field: 'scope', title: '作用域', width: 90, slots: { default: 'column-scope' } },
    { field: 'containers', title: '容器数', width: 80, slots: { default: 'column-containers' } },
    { title: '操作', width: canManage ? 230 : 90, fixed: 'right', slots: { default: 'column-operation' } },
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

const taskDrawerVisible = ref(false)
const activeTask = ref<DockerTaskInfo | null>(null)
const openTasks = () => { taskDrawerVisible.value = true }
const openTask = (task: DockerTaskInfo | undefined) => {
  if (!task?.id) return
  activeTask.value = task
  taskDrawerVisible.value = true
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
  if (!name) { ElMessage.error('请输入网络名称'); return }
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
    ElMessage.success('网络创建成功')
    createVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, '创建网络失败') }
  finally { createSubmitting.value = false }
}

// -- delete --
const handleDelete = async (row: DockerNetworkInfo) => {
  try { await Dialog.confirm({ title: '确认删除网络', content: `确定要删除网络「${row.name}」吗？`, confirmText: '确认删除', cancelText: '取消' }) }
  catch { return }
  try { await deleteDockerNetwork({ id: row.id }); ElMessage.success(`${row.name} 已删除`); await getList() }
  catch (e) { showRequestError(e, '删除网络失败') }
}

// -- prune --
const handlePrune = async () => {
  try { await Dialog.confirm({ title: '清理未使用网络', content: '确定要清理所有未使用的网络吗？', confirmText: '确认清理', cancelText: '取消' }) }
  catch { return }
  try {
    const { data } = await pruneDockerNetworks({})
    ElMessage.success('清理任务已创建')
    openTask(data?.task)
  }
  catch (e) { showRequestError(e, '清理网络失败') }
}

// -- detail --
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<DockerNetworkInfo | null>(null)
const openDetail = async (row: DockerNetworkInfo) => {
  detailVisible.value = true; detail.value = null; detailLoading.value = true
  try { const { data } = await getDockerNetwork({ id: row.id }); detail.value = data?.info || null }
  catch (e) { showRequestError(e, '获取网络详情失败') }
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
  if (!containerId) { ElMessage.error('请输入容器ID'); return }
  connectSubmitting.value = true
  try {
    await connectDockerNetwork({
      networkId: connectNetworkId.value, containerId,
      aliases: [], ipv4Address: connectForm.ipv4Address.trim(), ipv6Address: '',
    })
    ElMessage.success('容器已连接到网络')
    connectVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, '连接容器失败') }
  finally { connectSubmitting.value = false }
}

const handleDisconnect = async (network: DockerNetworkInfo, containerId: string) => {
  try {
    await Dialog.confirm({ title: '断开网络连接', content: `确定要断开容器「${containerId.slice(0, 12)}」吗？`, confirmText: '确认断开', cancelText: '取消' })
  } catch { return }
  try {
    await disconnectDockerNetwork({ networkId: network.id, containerId, force: false })
    ElMessage.success('网络连接已断开')
    const { data } = await getDockerNetwork({ id: network.id })
    detail.value = data?.info || null
    await getList()
  } catch (e) { showRequestError(e, '断开网络连接失败') }
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
    ElMessage.success(editRecreate.value ? '网络重建任务已创建' : '网络更新成功')
    editVisible.value = false
    if (!editRecreate.value) await getList()
  } catch (e) { showRequestError(e, '更新网络失败') }
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
