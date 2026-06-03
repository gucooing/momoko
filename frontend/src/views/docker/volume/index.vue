<template>
  <div class="docker-page">
    <el-card shadow="never">
      <el-form :model="queryForm" label-width="auto" @keyup.enter="search">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="名称">
              <el-input v-model="queryForm.name" placeholder="卷名称" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="驱动">
              <el-select v-model="queryForm.driver" placeholder="全部" clearable style="width: 100%">
                <el-option label="local" value="local" />
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
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" :disabled="!canManage" @click="openCreate">创建储存卷</el-button>
        <el-button type="warning" :disabled="!canManage" @click="handlePrune">清理未使用卷</el-button>
        <el-button :icon="menuStore.iconComponents['HOutline:ClockIcon']" @click="openTasks">任务</el-button>
      </div>

      <div v-loading="loading">
        <VxeGrid v-if="!menuStore.isMobile" v-bind="gridConfig">
          <template #column-driver="{ row }: { row: DockerVolumeInfo }">
            <BaseTag :text="row.driver" type="info" />
          </template>
          <template #column-mountpoint="{ row }: { row: DockerVolumeInfo }">
            <span class="text-mono">{{ row.mountpoint }}</span>
          </template>
          <template #column-size="{ row }: { row: DockerVolumeInfo }">
            {{ formatBytes(row.usageSize) }}
          </template>
          <template #column-operation="{ row }: { row: DockerVolumeInfo }">
            <el-button type="primary" link size="small" @click="openDetail(row)">详情</el-button>
            <el-button type="primary" link size="small" @click="openEdit(row)">编辑</el-button>
            <el-button type="primary" link size="small" @click="openExport(row)">导出</el-button>
            <el-button type="primary" link size="small" @click="openRestore()">恢复</el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">删除</el-button>
          </template>
        </VxeGrid>

        <div v-else-if="list.length" class="mobile-card-list">
          <div v-for="row in list" :key="row.name" class="mobile-card">
            <div class="mobile-card-body">
              <div class="mobile-card-header">
                <span class="mobile-card-title">{{ row.name }}</span>
                <BaseTag :text="row.driver" type="info" />
              </div>
              <div class="mobile-card-meta"><span>挂载点：{{ row.mountpoint }}</span></div>
              <div class="mobile-card-meta"><span>大小：{{ formatBytes(row.usageSize) }} / 引用：{{ row.refCount }}</span></div>
            </div>
            <div class="mobile-card-actions">
              <el-button size="small" plain type="primary" @click="openDetail(row)">详情</el-button>
              <el-button size="small" plain type="danger" @click="handleDelete(row)">删除</el-button>
            </div>
          </div>
        </div>
        <el-empty v-else-if="!loading" description="暂无储存卷" />
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
    <BaseDialog v-model="createVisible" title="创建储存卷" width="450">
      <el-form :model="createForm" label-position="top">
        <el-form-item label="名称" required>
          <el-input v-model="createForm.name" placeholder="卷名称" />
        </el-form-item>
        <el-form-item label="驱动">
          <el-select v-model="createForm.driver" style="width: 100%">
            <el-option label="local" value="local" />
          </el-select>
        </el-form-item>
        <el-form-item label="标签">
          <el-input v-model="createForm.labelsText" placeholder="key=value" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="驱动参数">
          <el-input v-model="createForm.driverOptsText" placeholder="key=value" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">取消</el-button>
        <el-button type="primary" :loading="createSubmitting" @click="submitCreate">创建</el-button>
      </template>
    </BaseDialog>

    <!-- Detail Dialog -->
    <BaseDialog v-model="detailVisible" title="储存卷详情" width="650">
      <div v-if="detail" class="detail-content" v-loading="detailLoading">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="名称">{{ detail.name }}</el-descriptions-item>
          <el-descriptions-item label="驱动">{{ detail.driver }}</el-descriptions-item>
          <el-descriptions-item label="挂载点">{{ detail.mountpoint }}</el-descriptions-item>
          <el-descriptions-item label="作用域">{{ detail.scope }}</el-descriptions-item>
          <el-descriptions-item label="大小">{{ formatBytes(detail.usageSize) }}</el-descriptions-item>
          <el-descriptions-item label="引用数">{{ detail.refCount }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ detail.createdAt }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="detail.labels && Object.keys(detail.labels).length" style="margin-top: 12px">
          <h4>标签</h4>
          <div class="kv-list">
            <template v-for="(v, k) in detail.labels" :key="k">
              <BaseTag :text="`${k}=${v}`" type="info" />
            </template>
          </div>
        </div>
        <div v-if="detail.options && Object.keys(detail.options).length" style="margin-top: 12px">
          <h4>驱动参数</h4>
          <div class="kv-list">
            <span v-for="(v, k) in detail.options" :key="k" class="text-mono kv-item">{{ k }}={{ v }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </BaseDialog>

    <!-- Edit Dialog -->
    <BaseDialog v-model="editVisible" title="编辑储存卷" width="450">
      <el-form label-position="top">
        <el-form-item label="标签">
          <el-input v-model="editLabelsText" placeholder="key=value&#10;每行一个" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item label="驱动参数">
          <el-input v-model="editDriverOptsText" placeholder="key=value&#10;每行一个" type="textarea" :rows="2" />
        </el-form-item>
        <el-divider />
        <el-checkbox v-model="editRecreate" style="margin-bottom: 8px">重建储存卷（需要修改驱动等时勾选）</el-checkbox>
        <template v-if="editRecreate">
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item label="驱动">
                <el-select v-model="editDriver" style="width: 100%">
                  <el-option label="local" value="local" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item label="强制重建">
                <el-switch v-model="editForce" />
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

    <!-- Export Dialog -->
    <BaseDialog v-model="exportVisible" title="导出储存卷" width="450">
      <el-form label-position="top">
        <el-form-item label="导出路径" required>
          <el-input v-model="exportPath" placeholder="服务端导出路径，如 /tmp/backup.tar.gz" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="exportVisible = false">取消</el-button>
        <el-button type="primary" :loading="exportSubmitting" @click="submitExport">导出</el-button>
      </template>
    </BaseDialog>

    <!-- Restore Dialog -->
    <BaseDialog v-model="restoreVisible" title="恢复储存卷" width="450">
      <el-form label-position="top">
        <el-form-item label="卷名称" required>
          <el-input v-model="restoreName" placeholder="目标卷名称" />
        </el-form-item>
        <el-form-item label="归档路径" required>
          <el-input v-model="restorePath" placeholder="服务端归档文件路径，如 /tmp/backup.tar.gz" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="restoreVisible = false">取消</el-button>
        <el-button type="primary" :loading="restoreSubmitting" @click="submitRestore">恢复</el-button>
      </template>
    </BaseDialog>

    <DockerTaskDrawer v-model="taskDrawerVisible" :active-task="activeTask" @finished="handleTaskFinished" />
  </div>
</template>

<script setup lang="ts">
import {
  createDockerVolume, deleteDockerVolume, exportDockerVolume,
  getDockerVolume, listDockerVolumes, pruneDockerVolumes, restoreDockerVolume,
  updateDockerVolume,
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
import type { DockerTaskInfo, DockerVolumeInfo } from '@/types/v1/docker'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'DockerVolumeView' })

const menuStore = useMenuStore()
const canManage = useButtonPermission([PERM.DOCKER_VOLUME_MANAGE], [])

const list = ref<DockerVolumeInfo[]>([])
const loading = ref(false)
const queryForm = reactive({ name: '', driver: '' })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const formatBytes = (bytes?: number | string) => {
  const n = Number(bytes)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0; let v = n
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}

const gridConfig = computed<VxeGridProps>(() => ({
  border: true, showOverflow: true, rowConfig: { isHover: true },
  data: list.value,
  columns: [
    { type: 'seq', title: '#', width: 50, fixed: 'left' },
    { field: 'name', title: '名称', minWidth: 160 },
    { field: 'driver', title: '驱动', width: 100, slots: { default: 'column-driver' } },
    { field: 'mountpoint', title: '挂载点', minWidth: 220, slots: { default: 'column-mountpoint' } },
    { field: 'usageSize', title: '大小', width: 100, slots: { default: 'column-size' } },
    { field: 'refCount', title: '引用', width: 60 },
    { title: '操作', width: 260, fixed: 'right', slots: { default: 'column-operation' } },
  ],
}))

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listDockerVolumes({
      page: pagination.value.page, pageSize: pagination.value.pageSize,
      name: queryForm.name || '', driver: queryForm.driver || '',
      labels: {},
    })
    list.value = data?.items || []
    pagination.value.total = Number(data?.total || 0)
  } catch {
    list.value = []
    pagination.value.total = 0
  } finally {
    loading.value = false
  }
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
const createForm = reactive({ name: '', driver: 'local', labelsText: '', driverOptsText: '' })
const openCreate = () => {
  Object.assign(createForm, { name: '', driver: 'local', labelsText: '', driverOptsText: '' })
  createVisible.value = true
}
const parseKvText = (text: string): Record<string, string> => {
  const obj: Record<string, string> = {}
  text.split('\n').filter(Boolean).forEach(line => {
    const idx = line.indexOf('=')
    if (idx > 0) obj[line.slice(0, idx).trim()] = line.slice(idx + 1).trim()
  })
  return obj
}
const submitCreate = async () => {
  const name = createForm.name.trim()
  if (!name) { ElMessage.error('请输入卷名称'); return }
  createSubmitting.value = true
  try {
    await createDockerVolume({
      options: {
        name, driver: createForm.driver,
        labels: parseKvText(createForm.labelsText),
        driverOpts: parseKvText(createForm.driverOptsText),
      },
    })
    ElMessage.success('储存卷创建成功')
    createVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, '创建储存卷失败') }
  finally { createSubmitting.value = false }
}

// -- delete --
const handleDelete = async (row: DockerVolumeInfo) => {
  try {
    await Dialog.confirm({ title: '确认删除储存卷', content: `确定要删除储存卷「${row.name}」吗？`, confirmText: '确认删除', cancelText: '取消' })
  } catch { return }
  try {
    await deleteDockerVolume({ name: row.name, force: false })
    ElMessage.success(`${row.name} 已删除`)
    await getList()
  } catch (e) { showRequestError(e, '删除储存卷失败') }
}

// -- prune --
const handlePrune = async () => {
  try {
    await Dialog.confirm({ title: '清理未使用储存卷', content: '确定要清理所有未使用的储存卷吗？', confirmText: '确认清理', cancelText: '取消' })
  } catch { return }
  try {
    const { data } = await pruneDockerVolumes({})
    ElMessage.success('清理任务已创建')
    openTask(data?.task)
  } catch (e) { showRequestError(e, '清理储存卷失败') }
}

// -- detail --
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<DockerVolumeInfo | null>(null)
const openDetail = async (row: DockerVolumeInfo) => {
  detailVisible.value = true; detail.value = null; detailLoading.value = true
  try {
    const { data } = await getDockerVolume({ name: row.name })
    detail.value = data?.info || null
  } catch (e) { showRequestError(e, '获取储存卷详情失败') }
  finally { detailLoading.value = false }
}

// -- export --
const exportVisible = ref(false)
const exportSubmitting = ref(false)
const exportVolumeName = ref('')
const exportPath = ref('')
const openExport = (row: DockerVolumeInfo) => {
  exportVolumeName.value = row.name; exportPath.value = ''; exportVisible.value = true
}
const submitExport = async () => {
  const archivePath = exportPath.value.trim()
  if (!archivePath) { ElMessage.error('请输入导出路径'); return }
  exportSubmitting.value = true
  try {
    const { data } = await exportDockerVolume({ name: exportVolumeName.value, archivePath })
    ElMessage.success('导出任务已创建')
    openTask(data?.task)
    exportVisible.value = false
  } catch (e) { showRequestError(e, '导出储存卷失败') }
  finally { exportSubmitting.value = false }
}

// -- restore --
const restoreVisible = ref(false)
const restoreSubmitting = ref(false)
const restoreName = ref('')
const restorePath = ref('')
const openRestore = () => {
  restoreName.value = ''; restorePath.value = ''; restoreVisible.value = true
}
const submitRestore = async () => {
  const name = restoreName.value.trim()
  const archivePath = restorePath.value.trim()
  if (!name || !archivePath) { ElMessage.error('请填写完整信息'); return }
  restoreSubmitting.value = true
  try {
    const { data } = await restoreDockerVolume({ name, archivePath })
    ElMessage.success('恢复任务已创建')
    openTask(data?.task)
    restoreVisible.value = false
  } catch (e) { showRequestError(e, '恢复储存卷失败') }
  finally { restoreSubmitting.value = false }
}

// -- edit --
const editVisible = ref(false)
const editSubmitting = ref(false)
const editName = ref('')
const editLabelsText = ref('')
const editDriverOptsText = ref('')
const editRecreate = ref(false)
const editForce = ref(false)
const editDriver = ref('local')
const openEdit = async (row: DockerVolumeInfo) => {
  editName.value = row.name; editRecreate.value = false; editForce.value = false; editDriver.value = 'local'
  editVisible.value = true
  try {
    const { data } = await getDockerVolume({ name: row.name })
    const info = data?.info
    editLabelsText.value = info?.labels ? Object.entries(info.labels).map(([k, v]) => `${k}=${v}`).join('\n') : ''
    editDriverOptsText.value = info?.options ? Object.entries(info.options).map(([k, v]) => `${k}=${v}`).join('\n') : ''
    editDriver.value = info?.driver || 'local'
  } catch { editLabelsText.value = ''; editDriverOptsText.value = '' }
}
const parseKV = (text: string) => {
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
      const { data } = await updateDockerVolume({
        name: editName.value, labels: parseKV(editLabelsText.value),
        driverOpts: parseKV(editDriverOptsText.value), force: editForce.value,
        options: { name: editName.value, driver: editDriver.value, labels: parseKV(editLabelsText.value), driverOpts: parseKV(editDriverOptsText.value) },
      })
      openTask(data?.task)
    } else {
      const { data } = await updateDockerVolume({ name: editName.value, labels: parseKV(editLabelsText.value), driverOpts: parseKV(editDriverOptsText.value), force: false, options: undefined })
      openTask(data?.task)
    }
    ElMessage.success(editRecreate.value ? '储存卷重建任务已创建' : '储存卷更新任务已创建')
    editVisible.value = false
  } catch (e) { showRequestError(e, '更新储存卷失败') }
  finally { editSubmitting.value = false }
}

onMounted(() => { getList() })
</script>

<style scoped lang="scss">
.docker-page { .card-mt-16 { margin-top: 16px; } }
.operation-container { margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap; }
.detail-content h4 { margin: 0 0 8px; font-size: 0.9rem; }
.text-mono { font-size: 0.78rem; font-family: monospace; color: var(--el-text-color-secondary); }
.kv-list { display: flex; flex-wrap: wrap; gap: 4px; }
.kv-item { margin-right: 8px; }

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
