<template>
  <div class="docker-page">
    <el-card shadow="never">
      <el-form :model="queryForm" label-width="auto" @keyup.enter="search">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="名称">
              <el-input v-model="queryForm.name" placeholder="容器名称" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="状态">
              <el-select v-model="queryForm.status" placeholder="全部" clearable style="width: 100%">
                <el-option label="运行中" value="running" />
                <el-option label="已停止" value="exited" />
                <el-option label="暂停" value="paused" />
              </el-select>
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="镜像">
              <el-input v-model="queryForm.image" placeholder="镜像名" clearable />
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
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" :disabled="!canManage" @click="openCreate">创建容器</el-button>
      </div>

      <div v-loading="loading">
        <VxeGrid v-if="!menuStore.isMobile" v-bind="gridConfig">
          <template #column-state="{ row }: { row: DockerContainerSummary }">
            <BaseTag :text="stateLabel(row.state)" :type="stateTagType(row.state)" />
          </template>
          <template #column-names="{ row }: { row: DockerContainerSummary }">
            <span>{{ displayNames(row) }}</span>
          </template>
          <template #column-ports="{ row }: { row: DockerContainerSummary }">
            <template v-if="row.ports?.length">
              <span v-for="p in row.ports.slice(0, 3)" :key="`${p.privatePort}-${p.publicPort}`" class="port-tag">
                {{ p.publicPort ? `${p.publicPort}:${p.privatePort}` : p.privatePort }}/{{ p.type }}
              </span>
              <span v-if="row.ports.length > 3" class="text-muted">+{{ row.ports.length - 3 }}</span>
            </template>
            <span v-else class="text-muted">-</span>
          </template>
          <template #column-operation="{ row }: { row: DockerContainerSummary }">
            <template v-if="canManage">
              <el-button v-if="row.state === 'running'" type="primary" link size="small" @click="handleAction(row, 'pause')">暂停</el-button>
              <el-button v-if="row.state === 'running'" type="primary" link size="small" @click="handleAction(row, 'stop')">停止</el-button>
              <el-button v-if="row.state === 'running'" type="primary" link size="small" @click="handleAction(row, 'restart')">重启</el-button>
              <el-button v-if="row.state === 'paused'" type="primary" link size="small" @click="handleAction(row, 'unpause')">恢复</el-button>
              <el-button v-if="row.state === 'exited' || row.state === 'created'" type="primary" link size="small" @click="handleAction(row, 'start')">启动</el-button>
              <el-button v-if="row.state === 'running'" type="primary" link size="small" @click="openLogs(row)">日志</el-button>
              <el-button type="primary" link size="small" @click="openEdit(row)">编辑</el-button>
            </template>
            <el-button type="primary" link size="small" @click="openDetail(row)">详情</el-button>
            <el-button v-if="canManage" type="danger" link size="small" @click="handleAction(row, 'delete')">删除</el-button>
          </template>
        </VxeGrid>

        <div v-else-if="list.length" class="mobile-card-list">
          <div v-for="row in list" :key="row.id" class="mobile-card">
            <div class="mobile-card-body">
              <div class="mobile-card-header">
                <span class="mobile-card-title">{{ displayNames(row) }}</span>
                <BaseTag :text="stateLabel(row.state)" :type="stateTagType(row.state)" />
              </div>
              <div class="mobile-card-meta"><span>镜像：{{ row.image }}</span></div>
              <div class="mobile-card-meta"><span>状态：{{ row.status }}</span></div>
            </div>
            <div class="mobile-card-actions">
              <el-button v-if="canManage && row.state === 'running'" size="small" plain type="primary" @click="handleAction(row, 'stop')">停止</el-button>
              <el-button v-if="canManage && row.state === 'exited'" size="small" plain type="primary" @click="handleAction(row, 'start')">启动</el-button>
              <el-button size="small" plain type="primary" @click="openDetail(row)">详情</el-button>
            </div>
          </div>
        </div>
        <el-empty v-else-if="!loading" description="暂无容器" />
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
    <BaseDialog v-model="createVisible" title="创建容器" width="700">
      <el-form :model="createForm" label-position="top">
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="容器名称">
              <el-input v-model="createForm.name" placeholder="可选" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="镜像" required>
              <el-input v-model="createForm.image" placeholder="如 nginx:latest" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="启动命令">
          <el-input v-model="createForm.cmdText" placeholder="如 /bin/bash -c 'echo hello'" />
        </el-form-item>
        <el-form-item label="环境变量">
          <el-input v-model="createForm.envText" type="textarea" :rows="2" placeholder="KEY=VALUE" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="端口映射">
              <el-input v-model="createForm.portsText" type="textarea" :rows="2" placeholder="8080:80" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="卷挂载">
              <el-input v-model="createForm.mountsText" type="textarea" :rows="2" placeholder="vol_name:/data" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="网络">
              <el-input v-model="createForm.network" placeholder="默认 bridge" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="重启策略">
              <el-select v-model="createForm.restartPolicy" style="width: 100%">
                <el-option label="不重启" value="" />
                <el-option label="总是重启" value="always" />
                <el-option label="失败时重启" value="on-failure" />
                <el-option label="除非停止" value="unless-stopped" />
              </el-select>
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
    <BaseDialog v-model="detailVisible" title="容器详情" width="800">
      <div v-if="detail" v-loading="detailLoading">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="ID">{{ detail.id }}</el-descriptions-item>
          <el-descriptions-item label="名称">{{ detail.name }}</el-descriptions-item>
          <el-descriptions-item label="镜像">{{ detail.image }}</el-descriptions-item>
          <el-descriptions-item label="状态">{{ detail.state?.status }}</el-descriptions-item>
          <el-descriptions-item label="平台">{{ detail.platform || '-' }}</el-descriptions-item>
          <el-descriptions-item label="重启次数">{{ detail.restartCount }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ detail.created }}</el-descriptions-item>
          <el-descriptions-item label="驱动">{{ detail.driver || '-' }}</el-descriptions-item>
          <el-descriptions-item label="日志路径" :span="2">{{ detail.logPath || '-' }}</el-descriptions-item>
          <el-descriptions-item label="启动命令" :span="2">{{ (detail.args || []).join(' ') || detail.path || '-' }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="detail.mounts?.length" style="margin-top: 12px">
          <h4>挂载点</h4>
          <el-table :data="detail.mounts" size="small" border>
            <el-table-column prop="type" label="类型" width="80" />
            <el-table-column prop="source" label="来源" />
            <el-table-column prop="destination" label="目标" />
            <el-table-column prop="mode" label="模式" width="100" />
            <el-table-column label="读写" width="60">
              <template #default="{ row: m }"><BaseTag :text="m.rw ? 'RW' : 'RO'" :type="m.rw ? 'success' : 'warning'" /></template>
            </el-table-column>
          </el-table>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </BaseDialog>

    <!-- Logs Dialog -->
    <BaseDialog v-model="logsVisible" title="容器日志" width="900">
      <div class="logs-options">
        <el-input-number v-model="logsTail" :min="10" :max="10000" size="small" style="width: 120px" />
        <el-checkbox v-model="logsTimestamps" size="small">时间戳</el-checkbox>
        <el-button size="small" :loading="logsLoading" @click="loadLogs">刷新</el-button>
      </div>
      <pre class="logs-content">{{ logsText || '暂无日志' }}</pre>
      <template #footer>
        <el-button @click="logsVisible = false">关闭</el-button>
      </template>
    </BaseDialog>

    <!-- Edit Dialog -->
    <BaseDialog v-model="editVisible" title="编辑容器" width="700">
      <el-form :model="editForm" label-position="top">
        <el-form-item label="容器名称">
          <el-input v-model="editForm.name" placeholder="容器名称" />
        </el-form-item>
        <el-form-item label="重启策略">
          <el-select v-model="editForm.restartPolicy" style="width: 100%">
            <el-option label="不重启" value="" />
            <el-option label="总是重启" value="always" />
            <el-option label="失败时重启" value="on-failure" />
            <el-option label="除非停止" value="unless-stopped" />
          </el-select>
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="12">
            <el-form-item label="内存限制 (MB)">
              <el-input-number v-model="editForm.memoryMB" :min="0" :step="64" style="width: 100%" />
            </el-form-item>
          </el-col>
          <el-col :span="12">
            <el-form-item label="CPU 限制 (核)">
              <el-input-number v-model="editForm.nanoCpuCores" :min="0" :step="0.5" :precision="1" style="width: 100%" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-divider />
        <el-checkbox v-model="editForm.recreate" style="margin-bottom: 8px">重建容器（需要修改镜像/端口/挂载等时勾选）</el-checkbox>
        <template v-if="editForm.recreate">
          <el-form-item label="镜像" required>
            <el-input v-model="editForm.image" placeholder="如 nginx:latest" />
          </el-form-item>
          <el-form-item label="端口映射">
            <el-input v-model="editForm.portsText" type="textarea" :rows="2" placeholder="8080:80" />
          </el-form-item>
          <el-form-item label="卷挂载">
            <el-input v-model="editForm.mountsText" type="textarea" :rows="2" placeholder="vol_name:/data" />
          </el-form-item>
          <el-form-item label="环境变量">
            <el-input v-model="editForm.envText" type="textarea" :rows="2" placeholder="KEY=VALUE" />
          </el-form-item>
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item label="网络">
                <el-input v-model="editForm.network" placeholder="默认 bridge" />
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="强制删除旧容器">
                <el-switch v-model="editForm.force" />
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
  </div>
</template>

<script setup lang="ts">
import {
  containerLogs, createDockerContainer, deleteDockerContainer,
  getDockerContainer, killDockerContainer, listDockerContainers,
  pauseDockerContainer, restartDockerContainer, startDockerContainer,
  stopDockerContainer, unpauseDockerContainer, updateDockerContainer,
} from '@/api/docker'
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import BaseTag from '@/components/tag/BaseTag.vue'
import TablePagination from '@/components/pagination/TablePagination.vue'
import { VxeGrid } from '@/plugins/vxeGrid'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import type { DockerContainerInfo, DockerContainerSummary, DockerPortBinding, DockerMount } from '@/types/v1/docker'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'DockerContainerView' })

const menuStore = useMenuStore()
const canManage = useButtonPermission([PERM.DOCKER_CONTAINER_MANAGE], [])

const list = ref<DockerContainerSummary[]>([])
const loading = ref(false)
const queryForm = reactive({ name: '', status: '', image: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const STATE_MAP: Record<string, string> = { running: '运行中', exited: '已停止', paused: '暂停', restarting: '重启中', created: '已创建', removing: '删除中', dead: '已死亡' }
const STATE_TAG: Record<string, 'success' | 'info' | 'warning' | 'danger'> = { running: 'success', exited: 'info', paused: 'warning', restarting: 'warning', created: 'info', dead: 'danger' }

const stateLabel = (s: string) => STATE_MAP[s] || s || '-'
const stateTagType = (s: string) => STATE_TAG[s] || 'info'
const displayNames = (row: DockerContainerSummary) => (row.names || []).map(n => n.replace(/^\//, '')).join(', ') || '-'

const gridConfig = computed<VxeGridProps>(() => ({
  border: true, showOverflow: true, rowConfig: { isHover: true },
  data: list.value,
  columns: [
    { type: 'seq', title: '#', width: 50, fixed: 'left' },
    { field: 'id', title: 'ID', width: 140, showOverflow: true },
    { field: 'names', title: '名称', minWidth: 160, slots: { default: 'column-names' } },
    { field: 'image', title: '镜像', minWidth: 180, showOverflow: true },
    { field: 'state', title: '状态', width: 90, slots: { default: 'column-state' } },
    { field: 'status', title: '描述', minWidth: 120, showOverflow: true },
    { field: 'ports', title: '端口', width: 160, slots: { default: 'column-ports' } },
    { title: '操作', width: canManage ? 380 : 140, fixed: 'right', slots: { default: 'column-operation' } },
  ],
}))

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listDockerContainers({
      all: true, page: pagination.value.page, pageSize: pagination.value.pageSize,
      status: queryForm.status || '', name: queryForm.name || '', image: queryForm.image || '',
      network: '', labels: {},
    })
    list.value = data?.items || []
    pagination.value.total = Number(data?.total || 0)
  } catch {
    list.value = []
    pagination.value.total = 0
  } finally { loading.value = false }
}

const search = () => { pagination.value.page = 1; getList() }
const reset = () => { queryForm.name = ''; queryForm.status = ''; queryForm.image = ''; search() }

// -- create --
const createVisible = ref(false)
const createSubmitting = ref(false)
const createForm = reactive({ name: '', image: '', cmdText: '', envText: '', portsText: '', mountsText: '', network: '', restartPolicy: '' })
const openCreate = () => {
  Object.assign(createForm, { name: '', image: '', cmdText: '', envText: '', portsText: '', mountsText: '', network: '', restartPolicy: '' })
  createVisible.value = true
}
const parsePorts = (text: string): DockerPortBinding[] =>
  text.split('\n').filter(Boolean).map(line => {
    const [hostPort, containerPort] = line.trim().split(':')
    return { containerPort: containerPort || hostPort, hostPort: containerPort ? hostPort : '', hostIp: '' }
  })
const parseMounts = (text: string): DockerMount[] =>
  text.split('\n').filter(Boolean).map(line => {
    const [source, target] = line.trim().split(':')
    return { type: !!source ? 'bind' : 'volume', source: source || '', target: target || '', readOnly: false }
  })
const submitCreate = async () => {
  const image = createForm.image.trim()
  if (!image) { ElMessage.error('请输入镜像名'); return }
  createSubmitting.value = true
  try {
    await createDockerContainer({
      options: {
        name: createForm.name.trim() || undefined as any,
        image,
        cmd: createForm.cmdText.trim() ? createForm.cmdText.trim().split(/\s+/) : [],
        env: createForm.envText.trim() ? createForm.envText.trim().split('\n').filter(Boolean) : [],
        ports: parsePorts(createForm.portsText),
        mounts: parseMounts(createForm.mountsText),
        network: createForm.network.trim() || undefined as any,
        restartPolicy: createForm.restartPolicy || undefined as any,
        hostname: '', user: '', entrypoint: [], workingDir: '', labels: {},
        tty: false, openStdin: false, autoRemove: false, privileged: false,
        memory: 0, memorySwap: 0, cpuShares: 0, cpuQuota: 0, cpuPeriod: 0, nanoCpus: 0, platform: '',
      },
    })
    ElMessage.success('容器创建成功')
    createVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, '创建容器失败') }
  finally { createSubmitting.value = false }
}

// -- actions --
const actionMap: Record<string, { label: string; fn: (id: string) => Promise<any>; confirm: boolean }> = {
  start: { label: '启动', fn: (id: string) => startDockerContainer({ id }), confirm: false },
  stop: { label: '停止', fn: (id: string) => stopDockerContainer({ id, timeoutSeconds: 10 }), confirm: true },
  restart: { label: '重启', fn: (id: string) => restartDockerContainer({ id, timeoutSeconds: 10 }), confirm: true },
  kill: { label: '强制停止', fn: (id: string) => killDockerContainer({ id, signal: 'SIGKILL' }), confirm: true },
  pause: { label: '暂停', fn: (id: string) => pauseDockerContainer({ id }), confirm: false },
  unpause: { label: '恢复', fn: (id: string) => unpauseDockerContainer({ id }), confirm: false },
  delete: { label: '删除', fn: (id: string) => deleteDockerContainer({ id, force: false, removeVolumes: false }), confirm: true },
}

const handleAction = async (row: DockerContainerSummary, action: string) => {
  const def = actionMap[action]
  if (!def) return
  const name = displayNames(row)
  if (def.confirm) {
    try { await Dialog.confirm({ title: `确认${def.label}`, content: `确定要${def.label}容器「${name}」吗？`, confirmText: `确认${def.label}`, cancelText: '取消' }) }
    catch { return }
  }
  try {
    await def.fn(row.id)
    ElMessage.success(`${def.label}操作成功`)
    await getList()
  } catch (e) { showRequestError(e, `${def.label}操作失败`) }
}

// -- detail --
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<DockerContainerInfo | null>(null)
const openDetail = async (row: DockerContainerSummary) => {
  detailVisible.value = true; detail.value = null; detailLoading.value = true
  try {
    const { data } = await getDockerContainer({ id: row.id })
    detail.value = data?.info || null
  } catch (e) { showRequestError(e, '获取容器详情失败') }
  finally { detailLoading.value = false }
}

// -- logs --
const logsVisible = ref(false)
const logsLoading = ref(false)
const logsText = ref('')
const logsTail = ref(200)
const logsTimestamps = ref(true)
const logsContainerId = ref('')
const openLogs = (row: DockerContainerSummary) => {
  logsContainerId.value = row.id; logsVisible.value = true
  nextTick(() => loadLogs())
}
const loadLogs = async () => {
  logsLoading.value = true
  try {
    const { data } = await containerLogs({
      id: logsContainerId.value, tail: String(logsTail.value),
      timestamps: logsTimestamps.value, stdout: true, stderr: true,
      since: '', until: '', details: false,
    })
    logsText.value = data?.logs || ''
  } catch (e) { showRequestError(e, '获取日志失败'); logsText.value = '获取日志失败' }
  finally { logsLoading.value = false }
}

// -- edit --
const editVisible = ref(false)
const editSubmitting = ref(false)
const editId = ref('')
const editForm = reactive({ name: '', restartPolicy: '', memoryMB: 0, nanoCpuCores: 0, recreate: false, image: '', portsText: '', mountsText: '', envText: '', network: '', force: false })
const resetEditForm = () => Object.assign(editForm, { name: '', restartPolicy: '', memoryMB: 0, nanoCpuCores: 0, recreate: false, image: '', portsText: '', mountsText: '', envText: '', network: '', force: false })
const openEdit = async (row: DockerContainerSummary) => {
  editId.value = row.id
  resetEditForm()
  editForm.name = displayNames(row)
  editForm.image = row.image
  editVisible.value = true
  try {
    const { data } = await getDockerContainer({ id: row.id })
    const info = data?.info
    if (info) {
      editForm.name = info.name || editForm.name
      editForm.image = info.image || editForm.image
      const hc = (info.hostConfig || {}) as Record<string, any>
      editForm.restartPolicy = (hc.RestartPolicy?.Name as string) || ''
      if (hc.Memory) editForm.memoryMB = Math.round(Number(hc.Memory) / 1024 / 1024)
      if (hc.NanoCpus) editForm.nanoCpuCores = Math.round((Number(hc.NanoCpus) / 1e9) * 10) / 10
    }
  } catch { /* use defaults */ }
}
const submitEdit = async () => {
  editSubmitting.value = true
  try {
    if (editForm.recreate) {
      if (!editForm.image.trim()) { ElMessage.error('请输入镜像名'); editSubmitting.value = false; return }
      await updateDockerContainer({
        id: editId.value,
        name: editForm.name.trim(),
        restartPolicy: editForm.restartPolicy,
        memory: editForm.memoryMB * 1024 * 1024,
        memorySwap: 0, cpuShares: 0, cpuQuota: 0, cpuPeriod: 0,
        nanoCpus: Math.round(editForm.nanoCpuCores * 1e9),
        recreate: true,
        force: editForm.force,
        removeVolumes: false,
        options: {
          name: editForm.name.trim() || undefined as any,
          image: editForm.image.trim(),
          cmd: [], env: editForm.envText.trim().split('\n').filter(Boolean),
          ports: parsePorts(editForm.portsText),
          mounts: parseMounts(editForm.mountsText),
          network: editForm.network.trim() || undefined as any,
          restartPolicy: editForm.restartPolicy || undefined as any,
          hostname: '', user: '', entrypoint: [], workingDir: '', labels: {},
          tty: false, openStdin: false, autoRemove: false, privileged: false,
          memory: editForm.memoryMB * 1024 * 1024, memorySwap: 0,
          cpuShares: 0, cpuQuota: 0, cpuPeriod: 0,
          nanoCpus: Math.round(editForm.nanoCpuCores * 1e9), platform: '',
        },
      })
    } else {
      await updateDockerContainer({
        id: editId.value,
        name: editForm.name.trim(),
        restartPolicy: editForm.restartPolicy,
        memory: editForm.memoryMB * 1024 * 1024,
        memorySwap: 0, cpuShares: 0, cpuQuota: 0, cpuPeriod: 0,
        nanoCpus: Math.round(editForm.nanoCpuCores * 1e9),
        recreate: false, force: false, removeVolumes: false, options: undefined,
      })
    }
    ElMessage.success(editForm.recreate ? '容器重建更新成功' : '容器更新成功')
    editVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, '更新容器失败') }
  finally { editSubmitting.value = false }
}

onMounted(() => { getList() })
</script>

<style scoped lang="scss">
.docker-page { .card-mt-16 { margin-top: 16px; } }
.operation-container { margin-bottom: 12px; }
.port-tag {
  display: inline-block; font-size: 0.72rem; background: var(--el-fill-color-light);
  padding: 1px 6px; border-radius: 4px; margin-right: 4px; margin-bottom: 2px;
}
.text-muted { color: var(--el-text-color-placeholder); font-size: 0.82rem; }
.logs-options { display: flex; align-items: center; gap: 10px; margin-bottom: 10px; }
.logs-content {
  background: #1e1e1e; color: #d4d4d4; padding: 12px; border-radius: 6px;
  max-height: 450px; overflow: auto; font-size: 0.82rem; white-space: pre-wrap; word-break: break-all;
  line-height: 1.5; margin: 0;
}
h4 { margin: 0 0 8px; font-size: 0.9rem; }

.mobile-card-list { display: flex; flex-direction: column; gap: 0.55rem; }
.mobile-card {
  display: flex; align-items: flex-start; gap: 0.6rem;
  padding: 0.7rem 0.8rem; border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.6rem; background: var(--el-bg-color);
  .mobile-card-body { flex: 1; min-width: 0; }
  .mobile-card-header { display: flex; align-items: center; justify-content: space-between; gap: 0.5rem; }
  .mobile-card-title { font-size: 0.88rem; font-weight: 700; }
  .mobile-card-meta { margin-top: 0.25rem; font-size: 0.74rem; color: var(--el-text-color-secondary); }
  .mobile-card-actions { display: flex; flex-direction: column; gap: 0.3rem; flex-shrink: 0; }
  .mobile-card-actions .el-button + .el-button { margin-left: 0; }
}
</style>
