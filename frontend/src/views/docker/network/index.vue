<!-- Docker 网络（重写 · P1 列表）：PageHeader(创建网络/任务) + FilterBar(名称/驱动) + DataTable/移动卡 + Pagination。
     行内 详情/编辑/删除（ActionMenu，管理权限门控）。弹窗全部 FormDialog；重建走 DockerTaskDialogs。 -->
<template>
  <div class="dk-page">
    <PageHeader :title="t('docker.network.title')" :description="t('docker.network.pageDesc')">
      <template #actions>
        <UButton color="neutral" variant="soft" icon="i-lucide-clock" @click="openTasks">
          {{ t('docker.common.tasks') }}
        </UButton>
        <UButton v-if="canManage" color="primary" icon="i-lucide-plus" @click="openCreate">
          {{ t('docker.network.createNetwork') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="search" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.name') }}</label>
          <input v-model="queryForm.name" class="app-input" :placeholder="t('docker.network.namePlaceholder')" @keyup.enter="search" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.mode') }}</label>
          <AppSelect v-model="queryForm.driver" :options="driverFilterOptions" />
        </div>
      </template>
    </FilterBar>

    <div class="dk-page__body">
      <div class="dk-page__bar">
        <span class="dk-page__hint">{{ t('system.common.total', { total: pagination.total }) }}</span>
      </div>

      <!-- 桌面：表格 -->
      <DataTable
        v-if="!menuStore.isMobile"
        :columns="columns"
        :rows="list"
        row-key="id"
        :loading="loading"
        :empty-text="t('docker.network.noNetworks')"
      >
        <template #cell-driver="{ row }"><StatusPill variant="info" :label="String(row.driver || '-')" :dot="false" /></template>
        <template #cell-subnet="{ row }">{{ subnetsOf(row).join(', ') || '—' }}</template>
        <template #cell-gateway="{ row }">{{ gatewaysOf(row).join(', ') || '—' }}</template>
        <template #cell-labels="{ row }">
          <div v-if="labelEntries(row.labels).length" class="dk-tags">
            <span v-for="[k, v] in labelEntries(row.labels).slice(0, 2)" :key="k" class="dk-tag" :title="`${k}=${v}`">{{ k }}={{ v }}</span>
            <span v-if="labelEntries(row.labels).length > 2" class="dk-dim">+{{ labelEntries(row.labels).length - 2 }}</span>
          </div>
          <span v-else class="dk-dim">—</span>
        </template>
        <template #cell-created="{ row }">{{ formatTime(row.created) }}</template>
        <template #cell-operation="{ row }">
          <ActionMenu :items="rowActionsFor(row)" @select="(key) => onRowAction(key, row)" />
        </template>
      </DataTable>

      <!-- 移动：卡片 -->
      <template v-else>
        <div v-if="loading" class="dk-cards">
          <div v-for="i in 4" :key="i" class="dk-skeleton" />
        </div>
        <EmptyState
          v-else-if="!list.length"
          icon="HOutline:GlobeAltIcon"
          :title="t('docker.network.noNetworks')"
          :description="t('docker.network.emptyDesc')"
        />
        <div v-else class="dk-cards">
          <EntityCard v-for="row in list" :key="row.id">
            <template #title>{{ row.name }}</template>
            <template #status><StatusPill variant="info" :label="row.driver || '-'" :dot="false" /></template>
            <template #meta>
              <span class="dk-card__full">{{ t('docker.network.subnet') }}: {{ subnetsOf(row).join(', ') || '—' }}</span>
              <span>{{ t('docker.network.containerMeta', { count: containerCountOf(row) }) }}</span>
            </template>
            <template #footer>
              <span>{{ formatTime(row.created) }}</span>
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

    <!-- 创建网络 -->
    <FormDialog v-model="createVisible" :title="t('docker.network.createNetwork')" :width="560" :loading="createSubmitting" @confirm="submitCreate">
      <div class="dk-grid">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('docker.common.name') }}</label>
          <input v-model="createForm.name" class="app-input" :placeholder="t('docker.network.namePlaceholder')" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.mode') }}</label>
          <AppSelect v-model="createForm.driver" :options="driverOptions" />
        </div>
        <div class="app-field dk-switch"><label class="app-label">{{ t('docker.network.internal') }}</label><AppSwitch v-model="createForm.internal" /></div>
        <div class="app-field dk-switch"><label class="app-label">IPv6</label><AppSwitch v-model="createForm.enableIpv6" /></div>
        <div class="app-field dk-switch"><label class="app-label">{{ t('docker.network.attachable') }}</label><AppSwitch v-model="createForm.attachable" /></div>
        <div class="app-field dk-grid__full-2">
          <label class="app-label">{{ t('docker.network.subnet') }}</label>
          <input v-model="createForm.subnet" class="app-input" :placeholder="t('docker.network.subnetPlaceholder')" />
        </div>
        <div class="app-field dk-grid__full-2">
          <label class="app-label">{{ t('docker.network.gateway') }}</label>
          <input v-model="createForm.gateway" class="app-input" :placeholder="t('docker.network.gatewayPlaceholder')" />
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="createSubmitting" @click="submitCreate">{{ t('docker.common.create') }}</UButton>
      </template>
    </FormDialog>

    <!-- 网络详情 -->
    <FormDialog v-model="detailVisible" :title="t('docker.network.networkDetail')" :width="820" :show-footer="false">
      <div v-if="detail" class="dk-detail">
        <div class="dk-detail__hero">
          <div class="dk-detail__title">
            <span>{{ detail.name || '-' }}</span>
            <StatusPill variant="info" :label="detail.driver || '-'" :dot="false" />
            <StatusPill :variant="detail.scope === 'local' ? 'success' : 'warning'" :label="detail.scope || '-'" :dot="false" />
          </div>
          <div class="dk-detail__sub">{{ shortId(detail.id) }} · {{ formatTime(detail.created) }}</div>
        </div>
        <div class="dk-kv">
          <div><span>{{ t('docker.common.mode') }}</span><strong>{{ detail.driver || '-' }}</strong></div>
          <div><span>{{ t('docker.common.scope') }}</span><strong>{{ detail.scope || '-' }}</strong></div>
          <div><span>{{ t('docker.network.internal') }}</span><strong>{{ detail.internal ? t('docker.common.yes') : t('docker.common.no') }}</strong></div>
          <div><span>{{ t('docker.network.attachable') }}</span><strong>{{ detail.attachable ? t('docker.common.yes') : t('docker.common.no') }}</strong></div>
          <div><span>IPv6</span><strong>{{ detail.enableIpv6 ? t('docker.common.enabled') : t('docker.common.notEnabled') }}</strong></div>
          <div><span>{{ t('docker.common.createdAt') }}</span><strong>{{ formatTime(detail.created) }}</strong></div>
        </div>

        <div v-if="detail.ipam?.config?.length" class="dk-detail__block">
          <div class="dk-detail__label">{{ t('docker.network.ipamConfig') }}</div>
          <div class="dk-list dk-list--ipam">
            <div class="dk-list__head"><span>{{ t('docker.network.subnet') }}</span><span>{{ t('docker.network.gateway') }}</span><span>{{ t('docker.network.ipRange') }}</span></div>
            <div v-for="(item, index) in detail.ipam.config" :key="index" class="dk-list__row">
              <span>{{ item.subnet || '-' }}</span><span>{{ item.gateway || '-' }}</span><span>{{ item.ipRange || '-' }}</span>
            </div>
          </div>
        </div>

        <div v-if="detailLabels.length" class="dk-detail__block">
          <div class="dk-detail__label">{{ t('docker.common.labels') }}</div>
          <div class="dk-tags"><span v-for="[k, v] in detailLabels" :key="k" class="dk-tag">{{ k }}={{ v }}</span></div>
        </div>

        <div class="dk-detail__block">
          <div class="dk-detail__label">{{ t('docker.network.connectedContainers') }} ({{ detailContainers.length }})</div>
          <div v-if="detailContainers.length" class="dk-list" :class="canManage ? 'dk-list--conn-manage' : 'dk-list--conn'">
            <div class="dk-list__head"><span>{{ t('docker.common.name') }}</span><span>IPv4</span><span>MAC</span><span v-if="canManage" /></div>
            <div v-for="item in detailContainers" :key="item.containerId" class="dk-list__row">
              <span class="dk-list__name" :title="item.containerId">{{ item.name || '-' }}</span>
              <span>{{ item.ipv4Address || '-' }}</span>
              <span>{{ item.macAddress || '-' }}</span>
              <span v-if="canManage">
                <button type="button" class="dk-link-danger" @click="handleDisconnect(detail, item.containerId)">{{ t('docker.network.disconnect') }}</button>
              </span>
            </div>
          </div>
          <div v-else class="dk-empty">{{ t('docker.common.noData') }}</div>
        </div>
      </div>
      <div v-else class="dk-detail__loading">{{ t('docker.common.noData') }}</div>
    </FormDialog>

    <!-- 编辑网络 -->
    <FormDialog v-model="editVisible" :title="t('docker.network.editNetwork')" :width="560" :loading="editSubmitting" @confirm="submitEdit">
      <div class="dk-form">
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.labels') }}</label>
          <textarea v-model="editLabelsText" class="app-textarea" rows="2" :placeholder="t('docker.common.lineKvPlaceholder')" />
        </div>
        <div class="app-field dk-switch">
          <label class="app-label">{{ t('docker.network.recreateTip') }}</label>
          <AppSwitch v-model="editRecreate" />
        </div>
        <template v-if="editRecreate">
          <div class="dk-grid">
            <div class="app-field">
              <label class="app-label">{{ t('docker.common.mode') }}</label>
              <AppSelect v-model="editDriver" :options="driverOptions" />
            </div>
            <div class="app-field dk-switch">
              <label class="app-label">{{ t('docker.common.forceRecreate') }}</label>
              <AppSwitch v-model="editForce" />
            </div>
            <div class="app-field">
              <label class="app-label">{{ t('docker.network.subnet') }}</label>
              <input v-model="editSubnet" class="app-input" :placeholder="t('docker.network.subnetPlaceholder')" />
            </div>
            <div class="app-field">
              <label class="app-label">{{ t('docker.network.gateway') }}</label>
              <input v-model="editGateway" class="app-input" :placeholder="t('docker.network.gatewayPlaceholder')" />
            </div>
          </div>
        </template>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="editSubmitting" @click="submitEdit">{{ t('docker.common.save') }}</UButton>
      </template>
    </FormDialog>

    <DockerTaskDialogs v-model="taskDialogsVisible" :active-task="activeTask" @finished="handleTaskFinished" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  createDockerNetwork, deleteDockerNetwork, disconnectDockerNetwork,
  getDockerNetwork, listDockerNetworks, updateDockerNetwork,
} from '@/api/docker'
import DockerTaskDialogs from '@/views/docker/components/DockerTaskDialogs.vue'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import type { DockerNetworkInfo, DockerTaskInfo } from '@/types/v1/docker'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

defineOptions({ name: 'DockerNetworkView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const canManage = useButtonPermission([PERM.DOCKER_NETWORK_MANAGE], [])

const list = ref<DockerNetworkInfo[]>([])
const loading = ref(false)
const queryForm = reactive({ name: '', driver: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const DRIVERS = ['bridge', 'host', 'overlay']
const driverOptions = DRIVERS.map((d) => ({ label: d, value: d }))
const driverFilterOptions = computed(() => [{ label: t('docker.common.all'), value: '' }, ...driverOptions])

const pad = (n: number) => String(n).padStart(2, '0')
const formatTime = (value: unknown): string => {
  const s = value ? String(value) : ''
  if (!s || s.startsWith('0001-')) return '—'
  const date = new Date(s)
  if (Number.isNaN(date.getTime())) return s
  return `${date.getFullYear()}-${pad(date.getMonth() + 1)}-${pad(date.getDate())} ${pad(date.getHours())}:${pad(date.getMinutes())}`
}
const shortId = (id?: unknown) => { const s = String(id || ''); return s ? s.slice(0, 12) : '-' }
const labelEntries = (labels?: unknown) => Object.entries((labels as Record<string, string>) || {})
const subnetsOf = (row: Record<string, unknown>) =>
  (((row.ipam as DockerNetworkInfo['ipam'])?.config) || []).map((i) => i.subnet).filter(Boolean)
const gatewaysOf = (row: Record<string, unknown>) =>
  (((row.ipam as DockerNetworkInfo['ipam'])?.config) || []).map((i) => i.gateway).filter(Boolean)
const containerCountOf = (row: Record<string, unknown>) =>
  row.containers ? Object.keys(row.containers as object).length : 0

const columns = computed<DataTableColumn[]>(() => [
  { key: 'name', title: t('docker.common.name'), minWidth: 160 },
  { key: 'driver', title: t('docker.common.mode'), width: 110 },
  { key: 'subnet', title: t('docker.network.subnet'), minWidth: 150 },
  { key: 'gateway', title: t('docker.network.gateway'), minWidth: 140 },
  { key: 'labels', title: t('docker.common.labels'), minWidth: 180 },
  { key: 'created', title: t('docker.common.createdAt'), width: 150 },
  { key: 'operation', title: t('docker.common.operation'), width: 80, align: 'center' },
])

const rowActionsFor = (row: Record<string, unknown>): ActionMenuItem[] => {
  const manage = canManage.value
  return [
    { key: 'detail', label: t('docker.common.detail'), icon: 'HOutline:InformationCircleIcon' },
    { key: 'edit', label: t('docker.common.edit'), icon: 'HOutline:PencilSquareIcon', hidden: !manage },
    { key: 'delete', label: t('docker.common.delete'), icon: 'HOutline:TrashIcon', danger: true, hidden: !manage || containerCountOf(row) > 0 },
  ]
}

const findRow = (id: string) => list.value.find((x) => x.id === id)
const onRowAction = (key: string, row: Record<string, unknown>) => {
  const record = findRow(String(row.id))
  if (!record) return
  if (key === 'detail') openDetail(record)
  else if (key === 'edit') openEdit(record)
  else if (key === 'delete') handleDelete(record)
}

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listDockerNetworks({
      page: pagination.value.page, pageSize: pagination.value.pageSize,
      name: queryForm.name || '', driver: queryForm.driver || '', scope: '', labels: {},
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

// —— 任务 ——
const taskDialogsVisible = ref(false)
const activeTask = ref<DockerTaskInfo | null>(null)
const openTasks = () => { taskDialogsVisible.value = true }
const openTask = (task: DockerTaskInfo | undefined) => {
  if (!task?.id) return
  activeTask.value = task
  taskDialogsVisible.value = true
}
const handleTaskFinished = async () => { await getList() }

// —— 创建 ——
const createVisible = ref(false)
const createSubmitting = ref(false)
const createForm = reactive({ name: '', driver: 'bridge', internal: false, enableIpv6: false, attachable: false, subnet: '', gateway: '' })
const openCreate = () => {
  Object.assign(createForm, { name: '', driver: 'bridge', internal: false, enableIpv6: false, attachable: false, subnet: '', gateway: '' })
  createVisible.value = true
}
const submitCreate = async () => {
  const name = createForm.name.trim()
  if (!name) { feedback.error(t('docker.network.enterName')); return }
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
    feedback.success(t('docker.network.createSuccess'))
    createVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, t('docker.network.createFailed')) }
  finally { createSubmitting.value = false }
}

// —— 删除 ——
const handleDelete = (row: DockerNetworkInfo) => {
  Dialog.confirm({
    title: t('docker.network.confirmDeleteTitle'),
    content: t('docker.network.confirmDeleteContent', { name: row.name }),
    confirmText: t('docker.network.confirmDeleteText'),
    cancelText: t('docker.common.cancel'),
    onConfirm: async () => {
      try {
        await deleteDockerNetwork({ id: row.id })
        feedback.success(t('docker.common.deletedName', { name: row.name }))
        await getList()
      } catch (e) { showRequestError(e, t('docker.network.deleteFailed')) }
    },
  })
}

// —— 详情 ——
const detailVisible = ref(false)
const detail = ref<DockerNetworkInfo | null>(null)
const detailLabels = computed(() => Object.entries(detail.value?.labels || {}))
const detailContainers = computed(() =>
  Object.entries(detail.value?.containers || {}).map(([id, item]) => ({ ...item, containerId: id })),
)
const openDetail = async (row: DockerNetworkInfo) => {
  detailVisible.value = true
  detail.value = null
  try { const { data } = await getDockerNetwork({ id: row.id }); detail.value = data?.info || null }
  catch (e) { showRequestError(e, t('docker.network.getDetailFailed')) }
}

const handleDisconnect = (network: DockerNetworkInfo, containerId: string) => {
  Dialog.confirm({
    title: t('docker.network.disconnectTitle'),
    content: t('docker.network.disconnectContent', { id: containerId.slice(0, 12) }),
    confirmText: t('docker.network.disconnectConfirm'),
    cancelText: t('docker.common.cancel'),
    onConfirm: async () => {
      try {
        await disconnectDockerNetwork({ networkId: network.id, containerId, force: false })
        feedback.success(t('docker.network.disconnectSuccess'))
        const { data } = await getDockerNetwork({ id: network.id })
        detail.value = data?.info || null
        await getList()
      } catch (e) { showRequestError(e, t('docker.network.disconnectFailed')) }
    },
  })
}

// —— 编辑 ——
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
  editId.value = row.id
  editRecreate.value = false
  editForce.value = false
  editDriver.value = 'bridge'
  editSubnet.value = ''
  editGateway.value = ''
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
  text.trim().split('\n').filter(Boolean).forEach((line) => {
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
    feedback.success(editRecreate.value ? t('docker.network.recreateTaskCreated') : t('docker.network.updateSuccess'))
    editVisible.value = false
    if (!editRecreate.value) await getList()
  } catch (e) { showRequestError(e, t('docker.network.updateFailed')) }
  finally { editSubmitting.value = false }
}

onMounted(() => { getList() })
</script>

<style scoped lang="scss">
.dk-page {
  display: flex;
  flex-direction: column;
  gap: 12px;
}
.dk-page__body {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.dk-page__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}
.dk-page__hint {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}
.dk-dim {
  color: var(--el-text-color-placeholder);
}
.dk-tags {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 4px;
}
.dk-tag {
  display: inline-block;
  max-width: 200px;
  padding: 1px 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  border-radius: var(--app-radius-xs);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 0.72rem;
}

/* 表单 */
.dk-form { display: flex; flex-direction: column; gap: 14px; }
.dk-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 14px;
}
.dk-grid__full-2 { grid-column: 1 / -1; }
.dk-switch {
  flex-direction: row;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

/* 详情 */
.dk-detail { display: flex; flex-direction: column; gap: 14px; }
.dk-detail__hero {
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-fill-color-lighter);
}
.dk-detail__title {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.dk-detail__sub {
  margin-top: 4px;
  font-size: 0.78rem;
  color: var(--el-text-color-secondary);
  word-break: break-all;
}
.dk-kv {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  overflow: hidden;
}
.dk-kv > div {
  padding: 8px 10px;
  border-right: 1px solid var(--el-border-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);
}
.dk-kv > div:nth-child(3n) { border-right: 0; }
.dk-kv > div:nth-last-child(-n + 3) { border-bottom: 0; }
.dk-kv span { display: block; color: var(--el-text-color-secondary); font-size: 0.72rem; }
.dk-kv strong { display: block; margin-top: 2px; color: var(--el-text-color-primary); font-size: 0.8rem; font-weight: 600; word-break: break-word; }
.dk-detail__label { margin-bottom: 6px; font-size: 0.8125rem; font-weight: 600; color: var(--el-text-color-primary); }
.dk-list {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  overflow: hidden;
  font-size: 0.78rem;
}
.dk-list__head,
.dk-list__row {
  display: grid;
  gap: 8px;
  align-items: center;
  padding: 7px 10px;
}
.dk-list__head { color: var(--el-text-color-secondary); font-size: 0.72rem; background: var(--el-fill-color-lighter); }
.dk-list__row { border-top: 1px solid var(--el-border-color-lighter); color: var(--el-text-color-regular); }
.dk-list--ipam .dk-list__head,
.dk-list--ipam .dk-list__row { grid-template-columns: 1.4fr 1fr 1fr; }
.dk-list--conn .dk-list__head,
.dk-list--conn .dk-list__row { grid-template-columns: 1.4fr 1.2fr 1.4fr; }
.dk-list--conn-manage .dk-list__head,
.dk-list--conn-manage .dk-list__row { grid-template-columns: 1.4fr 1.2fr 1.4fr 64px; }
.dk-list__name { font-weight: 600; color: var(--el-text-color-primary); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dk-list span { min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.dk-link-danger {
  border: none;
  background: transparent;
  color: var(--el-color-danger, #ef4444);
  font-size: 0.78rem;
  cursor: pointer;
  padding: 0;
}
.dk-link-danger:hover { text-decoration: underline; }
.dk-empty {
  padding: 12px;
  border: 1px dashed var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  color: var(--el-text-color-placeholder);
  font-size: 0.78rem;
  text-align: center;
}
.dk-detail__loading { padding: 40px 0; text-align: center; color: var(--el-text-color-secondary); }

/* 移动卡片 */
.dk-cards { display: flex; flex-direction: column; gap: 10px; }
.dk-card__full { flex-basis: 100%; }
.dk-skeleton {
  height: 120px;
  border-radius: var(--app-radius);
  background: linear-gradient(100deg, var(--el-fill-color-light) 30%, var(--el-fill-color) 50%, var(--el-fill-color-light) 70%);
  background-size: 200% 100%;
  animation: dk-shimmer 1.4s ease-in-out infinite;
}
@keyframes dk-shimmer {
  from { background-position: 200% 0; }
  to { background-position: -200% 0; }
}
@media (width <= 768px) {
  .dk-grid { grid-template-columns: 1fr; }
  .dk-grid__full-2 { grid-column: 1; }
}
</style>
