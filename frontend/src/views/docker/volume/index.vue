<template>
  <div class="docker-page">
    <el-card shadow="never">
      <el-form :model="queryForm" label-width="auto" @keyup.enter="search">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item :label="t('docker.common.name')">
              <el-input v-model="queryForm.name" :placeholder="t('docker.volume.namePlaceholder')" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item :label="t('docker.common.driver')">
              <el-select v-model="queryForm.driver" :placeholder="t('docker.common.all')" clearable style="width: 100%">
                <el-option label="local" value="local" />
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
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" :disabled="!canManage" @click="openCreate">{{ t('docker.volume.createVolume') }}</el-button>
        <el-button type="warning" :disabled="!canManage" @click="handlePrune">{{ t('docker.volume.pruneUnused') }}</el-button>
        <el-button :icon="menuStore.iconComponents['HOutline:ClockIcon']" @click="openTasks">{{ t('docker.common.tasks') }}</el-button>
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
            <el-button type="primary" link size="small" @click="openDetail(row)">{{ t('docker.common.detail') }}</el-button>
            <el-button type="primary" link size="small" @click="openEdit(row)">{{ t('docker.common.edit') }}</el-button>
            <el-button type="primary" link size="small" @click="openExport(row)">{{ t('docker.volume.export') }}</el-button>
            <el-button type="primary" link size="small" @click="openRestore()">{{ t('docker.volume.restore') }}</el-button>
            <el-button type="danger" link size="small" @click="handleDelete(row)">{{ t('docker.common.delete') }}</el-button>
          </template>
        </VxeGrid>

        <div v-else-if="list.length" class="mobile-card-list">
          <div v-for="row in list" :key="row.name" class="mobile-card">
            <div class="mobile-card-body">
              <div class="mobile-card-header">
                <span class="mobile-card-title">{{ row.name }}</span>
                <BaseTag :text="row.driver" type="info" />
              </div>
              <div class="mobile-card-meta"><span>{{ t('docker.volume.mountpointMeta', { mountpoint: row.mountpoint }) }}</span></div>
              <div class="mobile-card-meta"><span>{{ t('docker.volume.sizeMeta', { size: formatBytes(row.usageSize), count: row.refCount }) }}</span></div>
            </div>
            <div class="mobile-card-actions">
              <el-button size="small" plain type="primary" @click="openDetail(row)">{{ t('docker.common.detail') }}</el-button>
              <el-button size="small" plain type="danger" @click="handleDelete(row)">{{ t('docker.common.delete') }}</el-button>
            </div>
          </div>
        </div>
        <el-empty v-else-if="!loading" :description="t('docker.volume.noVolumes')" />
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
    <BaseDialog v-model="createVisible" :title="t('docker.volume.createVolume')" width="450">
      <el-form :model="createForm" label-position="top">
        <el-form-item :label="t('docker.common.name')" required>
          <el-input v-model="createForm.name" :placeholder="t('docker.volume.namePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('docker.common.driver')">
          <el-select v-model="createForm.driver" style="width: 100%">
            <el-option label="local" value="local" />
          </el-select>
        </el-form-item>
        <el-form-item :label="t('docker.common.labels')">
          <el-input v-model="createForm.labelsText" placeholder="key=value" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item :label="t('docker.common.driverOptions')">
          <el-input v-model="createForm.driverOptsText" placeholder="key=value" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="createVisible = false">{{ t('docker.common.cancel') }}</el-button>
        <el-button type="primary" :loading="createSubmitting" @click="submitCreate">{{ t('docker.common.create') }}</el-button>
      </template>
    </BaseDialog>

    <!-- Detail Dialog -->
    <BaseDialog v-model="detailVisible" :title="t('docker.volume.volumeDetail')" width="650">
      <div v-if="detail" class="detail-content" v-loading="detailLoading">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="t('docker.common.name')">{{ detail.name }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.driver')">{{ detail.driver }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.mountpoint')">{{ detail.mountpoint }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.scope')">{{ detail.scope }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.size')">{{ formatBytes(detail.usageSize) }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.referenceCount')">{{ detail.refCount }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.createdAt')">{{ detail.createdAt }}</el-descriptions-item>
        </el-descriptions>
        <div v-if="detail.labels && Object.keys(detail.labels).length" style="margin-top: 12px">
          <h4>{{ t('docker.common.labels') }}</h4>
          <div class="kv-list">
            <template v-for="(v, k) in detail.labels" :key="k">
              <BaseTag :text="`${k}=${v}`" type="info" />
            </template>
          </div>
        </div>
        <div v-if="detail.options && Object.keys(detail.options).length" style="margin-top: 12px">
          <h4>{{ t('docker.common.driverOptions') }}</h4>
          <div class="kv-list">
            <span v-for="(v, k) in detail.options" :key="k" class="text-mono kv-item">{{ k }}={{ v }}</span>
          </div>
        </div>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">{{ t('docker.common.close') }}</el-button>
      </template>
    </BaseDialog>

    <!-- Edit Dialog -->
    <BaseDialog v-model="editVisible" :title="t('docker.volume.editVolume')" width="450">
      <el-form label-position="top">
        <el-form-item :label="t('docker.common.labels')">
          <el-input v-model="editLabelsText" :placeholder="t('docker.common.lineKvPlaceholder')" type="textarea" :rows="2" />
        </el-form-item>
        <el-form-item :label="t('docker.common.driverOptions')">
          <el-input v-model="editDriverOptsText" :placeholder="t('docker.common.lineKvPlaceholder')" type="textarea" :rows="2" />
        </el-form-item>
        <el-divider />
        <el-checkbox v-model="editRecreate" style="margin-bottom: 8px">{{ t('docker.volume.recreateTip') }}</el-checkbox>
        <template v-if="editRecreate">
          <el-row :gutter="12">
            <el-col :span="12">
              <el-form-item :label="t('docker.common.driver')">
                <el-select v-model="editDriver" style="width: 100%">
                  <el-option label="local" value="local" />
                </el-select>
              </el-form-item>
            </el-col>
            <el-col :span="12">
              <el-form-item :label="t('docker.common.forceRecreate')">
                <el-switch v-model="editForce" />
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

    <!-- Export Dialog -->
    <BaseDialog v-model="exportVisible" :title="t('docker.volume.exportVolume')" width="450">
      <el-form label-position="top">
        <el-form-item :label="t('docker.volume.exportPath')" required>
          <el-input v-model="exportPath" :placeholder="t('docker.volume.exportPathPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="exportVisible = false">{{ t('docker.common.cancel') }}</el-button>
        <el-button type="primary" :loading="exportSubmitting" @click="submitExport">{{ t('docker.volume.export') }}</el-button>
      </template>
    </BaseDialog>

    <!-- Restore Dialog -->
    <BaseDialog v-model="restoreVisible" :title="t('docker.volume.restoreVolume')" width="450">
      <el-form label-position="top">
        <el-form-item :label="t('docker.volume.volumeName')" required>
          <el-input v-model="restoreName" :placeholder="t('docker.volume.targetVolumeName')" />
        </el-form-item>
        <el-form-item :label="t('docker.volume.archivePath')" required>
          <el-input v-model="restorePath" :placeholder="t('docker.volume.archivePathPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="restoreVisible = false">{{ t('docker.common.cancel') }}</el-button>
        <el-button type="primary" :loading="restoreSubmitting" @click="submitRestore">{{ t('docker.volume.restore') }}</el-button>
      </template>
    </BaseDialog>

    <DockerTaskDialogs v-model="taskDialogsVisible" :active-task="activeTask" @finished="handleTaskFinished" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  createDockerVolume, deleteDockerVolume, exportDockerVolume,
  getDockerVolume, listDockerVolumes, pruneDockerVolumes, restoreDockerVolume,
  updateDockerVolume,
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
import type { DockerTaskInfo, DockerVolumeInfo } from '@/types/v1/docker'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'DockerVolumeView' })

const menuStore = useMenuStore()
const { t } = useI18n()
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
    { field: 'name', title: t('docker.common.name'), minWidth: 160 },
    { field: 'driver', title: t('docker.common.driver'), width: 100, slots: { default: 'column-driver' } },
    { field: 'mountpoint', title: t('docker.common.mountpoint'), minWidth: 220, slots: { default: 'column-mountpoint' } },
    { field: 'usageSize', title: t('docker.common.size'), width: 100, slots: { default: 'column-size' } },
    { field: 'refCount', title: t('docker.common.references'), width: 60 },
    { title: t('docker.common.operation'), width: 260, fixed: 'right', slots: { default: 'column-operation' } },
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
  if (!name) { ElMessage.error(t('docker.volume.enterName')); return }
  createSubmitting.value = true
  try {
    await createDockerVolume({
      options: {
        name, driver: createForm.driver,
        labels: parseKvText(createForm.labelsText),
        driverOpts: parseKvText(createForm.driverOptsText),
      },
    })
    ElMessage.success(t('docker.volume.createSuccess'))
    createVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, t('docker.volume.createFailed')) }
  finally { createSubmitting.value = false }
}

// -- delete --
const handleDelete = async (row: DockerVolumeInfo) => {
  try {
    await Dialog.confirm({ title: t('docker.volume.confirmDeleteTitle'), content: t('docker.volume.confirmDeleteContent', { name: row.name }), confirmText: t('docker.volume.confirmDeleteText'), cancelText: t('docker.common.cancel') })
  } catch { return }
  try {
    await deleteDockerVolume({ name: row.name, force: false })
    ElMessage.success(t('docker.common.deletedName', { name: row.name }))
    await getList()
  } catch (e) { showRequestError(e, t('docker.volume.deleteFailed')) }
}

// -- prune --
const handlePrune = async () => {
  try {
    await Dialog.confirm({ title: t('docker.volume.pruneTitle'), content: t('docker.volume.pruneContent'), confirmText: t('docker.volume.pruneConfirm'), cancelText: t('docker.common.cancel') })
  } catch { return }
  try {
    const { data } = await pruneDockerVolumes({})
    ElMessage.success(t('docker.common.taskCreated'))
    openTask(data?.task)
  } catch (e) { showRequestError(e, t('docker.volume.pruneFailed')) }
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
  } catch (e) { showRequestError(e, t('docker.volume.getDetailFailed')) }
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
  if (!archivePath) { ElMessage.error(t('docker.volume.enterExportPath')); return }
  exportSubmitting.value = true
  try {
    const { data } = await exportDockerVolume({ name: exportVolumeName.value, archivePath })
    ElMessage.success(t('docker.volume.exportTaskCreated'))
    openTask(data?.task)
    exportVisible.value = false
  } catch (e) { showRequestError(e, t('docker.volume.exportFailed')) }
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
  if (!name || !archivePath) { ElMessage.error(t('docker.volume.enterCompleteInfo')); return }
  restoreSubmitting.value = true
  try {
    const { data } = await restoreDockerVolume({ name, archivePath })
    ElMessage.success(t('docker.volume.restoreTaskCreated'))
    openTask(data?.task)
    restoreVisible.value = false
  } catch (e) { showRequestError(e, t('docker.volume.restoreFailed')) }
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
    ElMessage.success(editRecreate.value ? t('docker.volume.recreateTaskCreated') : t('docker.volume.updateTaskCreated'))
    editVisible.value = false
  } catch (e) { showRequestError(e, t('docker.volume.updateFailed')) }
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
