<template>
  <div class="docker-page">
    <el-card shadow="never">
      <el-form :model="queryForm" label-width="auto" @keyup.enter="search">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item :label="t('docker.common.keyword')">
              <el-input v-model="queryForm.keyword" :placeholder="t('docker.image.keywordPlaceholder')" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item :label="t('docker.image.dangling')">
              <el-select v-model="queryForm.dangling" :placeholder="t('docker.common.all')" clearable style="width: 100%">
                <el-option :label="t('docker.common.yes')" :value="true" />
                <el-option :label="t('docker.common.no')" :value="false" />
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
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" :disabled="!canManage" @click="openPull">{{ t('docker.image.pullImage') }}</el-button>
        <el-button :icon="menuStore.iconComponents['HOutline:ClockIcon']" @click="openTasks">{{ t('docker.common.tasks') }}</el-button>
      </div>

      <div v-loading="loading">
        <VxeGrid v-if="!menuStore.isMobile" v-bind="gridConfig">
          <template #column-repoTags="{ row }: { row: DockerImageSummary }">
            <template v-if="row.repoTags?.length">
              <BaseTag v-for="tag in row.repoTags.slice(0, 3)" :key="tag" :text="tag" type="info" />
              <span v-if="row.repoTags.length > 3" class="text-muted">+{{ row.repoTags.length - 3 }}</span>
            </template>
            <span v-else class="text-muted">&lt;none&gt;</span>
          </template>
          <template #column-size="{ row }: { row: DockerImageSummary }">
            {{ formatBytes(row.size) }}
          </template>
          <template #column-created="{ row }: { row: DockerImageSummary }">
            {{ formatTime(row.created) }}
          </template>
          <template #column-operation="{ row }: { row: DockerImageSummary }">
            <el-button type="primary" link size="small" @click="openDetail(row)">{{ t('docker.common.detail') }}</el-button>
            <template v-if="canManage">
              <el-button type="primary" link size="small" @click="openEdit(row)">{{ t('docker.image.editTags') }}</el-button>
              <el-button type="primary" link size="small" @click="openTag(row)">{{ t('docker.image.tagImage') }}</el-button>
              <el-button type="primary" link size="small" @click="openHistory(row)">{{ t('docker.image.history') }}</el-button>
              <el-button v-if="canDeleteImage(row)" type="danger" link size="small" @click="handleDelete(row)">{{ t('docker.common.delete') }}</el-button>
            </template>
          </template>
        </VxeGrid>

        <div v-else-if="list.length" class="mobile-card-list">
          <div v-for="row in list" :key="row.id" class="mobile-card">
            <div class="mobile-card-body">
              <div class="mobile-card-header">
                <span class="mobile-card-title">{{ (row.repoTags || ['<none>'])[0] }}</span>
              </div>
              <div class="mobile-card-meta"><span>{{ t('docker.image.idMeta', { id: row.id?.slice(7, 19) || '-' }) }}</span></div>
              <div class="mobile-card-meta"><span>{{ t('docker.image.sizeMeta', { size: formatBytes(row.size), count: row.containers }) }}</span></div>
            </div>
            <div class="mobile-card-actions">
              <el-button size="small" plain type="primary" @click="openDetail(row)">{{ t('docker.common.detail') }}</el-button>
              <el-button v-if="canManage && canDeleteImage(row)" size="small" plain type="danger" @click="handleDelete(row)">{{ t('docker.common.delete') }}</el-button>
            </div>
          </div>
        </div>
        <el-empty v-else-if="!loading" :description="t('docker.image.noImages')" />
      </div>

      <TablePagination
        v-model:current-page="pagination.page"
        v-model:page-size="pagination.pageSize"
        :total="pagination.total"
        :is-mobile="menuStore.isMobile"
        @change="getList"
      />
    </el-card>

    <!-- Pull Dialog -->
    <BaseDialog v-model="pullVisible" :title="t('docker.image.pullImage')" width="450">
      <el-form label-position="top">
        <el-form-item :label="t('docker.image.imageReference')" required>
          <el-input v-model="pullForm.reference" :placeholder="t('docker.image.referencePlaceholder')" />
        </el-form-item>
        <el-form-item :label="t('docker.common.platform')">
          <el-input v-model="pullForm.platform" placeholder="linux/amd64" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pullVisible = false">{{ t('docker.common.cancel') }}</el-button>
        <el-button type="primary" :loading="pullSubmitting" @click="submitPull">{{ t('docker.image.pull') }}</el-button>
      </template>
    </BaseDialog>

    <!-- Tag Dialog -->
    <BaseDialog v-model="tagVisible" :title="t('docker.image.tagImage')" width="400">
      <el-form label-position="top">
        <el-form-item :label="t('docker.image.newTag')" required>
          <el-input v-model="tagTarget" :placeholder="t('docker.image.newTagPlaceholder')" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="tagVisible = false">{{ t('docker.common.cancel') }}</el-button>
        <el-button type="primary" :loading="tagSubmitting" @click="submitTag">{{ t('docker.common.confirm') }}</el-button>
      </template>
    </BaseDialog>

    <!-- Detail Dialog -->
    <BaseDialog v-model="detailVisible" :title="t('docker.image.imageDetail')" width="700">
      <div v-if="detail" v-loading="detailLoading">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item :label="t('docker.common.id')">{{ detail.id }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.size')">{{ formatBytes(detail.size) }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.architecture')">{{ detail.architecture }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.os')">{{ detail.os }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.author')">{{ detail.author || '-' }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.createdAt')">{{ detail.created }}</el-descriptions-item>
          <el-descriptions-item :label="t('docker.image.repoTags')" :span="2">
            <template v-if="detail.repoTags?.length">
              <BaseTag v-for="tag in detail.repoTags" :key="tag" :text="tag" type="info" />
            </template>
            <span v-else>-</span>
          </el-descriptions-item>
          <el-descriptions-item :label="t('docker.common.summary')" :span="2">
            <span v-if="detail.repoDigests?.length">{{ detail.repoDigests.join(', ') }}</span>
            <span v-else>-</span>
          </el-descriptions-item>
        </el-descriptions>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">{{ t('docker.common.close') }}</el-button>
      </template>
    </BaseDialog>

    <!-- Edit Tags Dialog -->
    <BaseDialog v-model="editVisible" :title="t('docker.image.editImageTags')" width="500">
      <el-form label-position="top">
        <el-form-item :label="t('docker.image.addTags')">
          <el-input v-model="editAddTagsText" :placeholder="t('docker.image.oneTagPerLine')" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item :label="t('docker.image.deleteTags')">
          <el-input v-model="editDelTagsText" :placeholder="t('docker.image.oneTagPerLine')" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">{{ t('docker.common.cancel') }}</el-button>
        <el-button type="primary" :loading="editSubmitting" @click="submitEdit">{{ t('docker.common.save') }}</el-button>
      </template>
    </BaseDialog>

    <!-- History Dialog -->
    <BaseDialog v-model="historyVisible" :title="t('docker.image.imageHistory')" width="800">
      <el-table :data="historyItems" size="small" border v-loading="historyLoading">
        <el-table-column type="index" label="#" width="50" />
        <el-table-column prop="id" :label="t('docker.common.id')" width="140" show-overflow-tooltip />
        <el-table-column prop="createdBy" :label="t('docker.image.createdBy')" min-width="300" show-overflow-tooltip />
        <el-table-column :label="t('docker.common.size')" width="100">
          <template #default="{ row: h }">{{ formatBytes(h.size) }}</template>
        </el-table-column>
        <el-table-column :label="t('docker.common.createdAt')" width="180">
          <template #default="{ row: h }">{{ formatTime(h.created) }}</template>
        </el-table-column>
        <el-table-column :label="t('docker.common.labels')" width="150">
          <template #default="{ row: h }">
            <BaseTag v-for="tag in h.tags || []" :key="tag" :text="tag" type="info" />
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="historyVisible = false">{{ t('docker.common.close') }}</el-button>
      </template>
    </BaseDialog>

    <DockerTaskDialogs v-model="taskDialogsVisible" :active-task="activeTask" @finished="handleTaskFinished" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  deleteDockerImage, getDockerImage, imageHistory,
  listDockerImages, pullDockerImage, tagDockerImage,
  updateDockerImageTags,
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
import type { DockerImageHistoryItem, DockerImageInfo, DockerImageSummary, DockerTaskInfo } from '@/types/v1/docker'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'DockerImageView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const canManage = useButtonPermission([PERM.DOCKER_IMAGE_MANAGE], [])

const list = ref<DockerImageSummary[]>([])
const loading = ref(false)
const queryForm = reactive({ keyword: '', dangling: undefined as boolean | undefined })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const formatBytes = (bytes?: number | string) => {
  const n = Number(bytes)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0; let v = n
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}
const pad = (n: number) => String(n).padStart(2, '0')
const formatTime = (t: Date | string | undefined): string => {
  if (!t) return '-'
  const d = new Date(t)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}`
}

const gridConfig = computed<VxeGridProps>(() => ({
  border: true, showOverflow: true, rowConfig: { isHover: true },
  data: list.value,
  columns: [
    { type: 'seq', title: '#', width: 50, fixed: 'left' },
    { field: 'id', title: 'ID', width: 150, showOverflow: true },
    { field: 'repoTags', title: t('docker.image.repoTags'), minWidth: 220, slots: { default: 'column-repoTags' } },
    { field: 'size', title: t('docker.common.size'), width: 100, slots: { default: 'column-size' } },
    { field: 'containers', title: t('docker.image.containerCount'), width: 80 },
    { field: 'created', title: t('docker.common.createdAt'), width: 170, slots: { default: 'column-created' } },
    { title: t('docker.common.operation'), width: canManage.value ? 270 : 90, fixed: 'right', slots: { default: 'column-operation' } },
  ],
}))

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listDockerImages({
      all: false, page: pagination.value.page, pageSize: pagination.value.pageSize,
      keyword: queryForm.keyword || '', dangling: queryForm.dangling, labels: {},
    })
    list.value = data?.items || []
    pagination.value.total = Number(data?.total || 0)
  } catch {
    list.value = []
    pagination.value.total = 0
  } finally { loading.value = false }
}

const search = () => { pagination.value.page = 1; getList() }
const reset = () => { queryForm.keyword = ''; queryForm.dangling = undefined; search() }
const canDeleteImage = (row: DockerImageSummary) => row.containers <= 0

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

// -- pull --
const pullVisible = ref(false)
const pullSubmitting = ref(false)
const pullForm = reactive({ reference: '', platform: '' })
const openPull = () => { pullForm.reference = ''; pullForm.platform = ''; pullVisible.value = true }
const submitPull = async () => {
  const reference = pullForm.reference.trim()
  if (!reference) { ElMessage.error(t('docker.image.enterReference')); return }
  pullSubmitting.value = true
  try {
    const { data } = await pullDockerImage({ reference, platform: pullForm.platform.trim(), registryAuth: undefined })
    ElMessage.success(t('docker.image.pullTaskCreated'))
    openTask(data?.task)
    pullVisible.value = false
  } catch (e) { showRequestError(e, t('docker.image.pullFailed')) }
  finally { pullSubmitting.value = false }
}

// -- tag --
const tagVisible = ref(false)
const tagSubmitting = ref(false)
const tagTarget = ref('')
const tagId = ref('')
const openTag = (row: DockerImageSummary) => { tagId.value = row.id; tagTarget.value = ''; tagVisible.value = true }
const submitTag = async () => {
  const target = tagTarget.value.trim()
  if (!target) { ElMessage.error(t('docker.image.enterNewTag')); return }
  tagSubmitting.value = true
  try { await tagDockerImage({ id: tagId.value, target }); ElMessage.success(t('docker.image.tagSuccess')); tagVisible.value = false; await getList() }
  catch (e) { showRequestError(e, t('docker.image.tagFailed')) }
  finally { tagSubmitting.value = false }
}

// -- delete --
const handleDelete = async (row: DockerImageSummary) => {
  const name = (row.repoTags || ['<none>'])[0]
  try { await Dialog.confirm({ title: t('docker.image.confirmDeleteTitle'), content: t('docker.image.confirmDeleteContent', { name }), confirmText: t('docker.image.confirmDeleteText'), cancelText: t('docker.common.cancel') }) }
  catch { return }
  try { await deleteDockerImage({ id: row.id, force: false, pruneChildren: false }); ElMessage.success(t('docker.common.deletedName', { name })); await getList() }
  catch (e) { showRequestError(e, t('docker.image.deleteFailed')) }
}

// -- detail --
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<DockerImageInfo | null>(null)
const openDetail = async (row: DockerImageSummary) => {
  detailVisible.value = true; detail.value = null; detailLoading.value = true
  try { const { data } = await getDockerImage({ id: row.id }); detail.value = data?.info || null }
  catch (e) { showRequestError(e, t('docker.image.getDetailFailed')) }
  finally { detailLoading.value = false }
}

// -- history --
const historyVisible = ref(false)
const historyLoading = ref(false)
const historyItems = ref<DockerImageHistoryItem[]>([])
const openHistory = async (row: DockerImageSummary) => {
  historyVisible.value = true; historyLoading.value = true; historyItems.value = []
  try { const { data } = await imageHistory({ id: row.id }); historyItems.value = data?.items || [] }
  catch (e) { showRequestError(e, t('docker.image.getHistoryFailed')) }
  finally { historyLoading.value = false }
}

// -- edit tags --
const editVisible = ref(false)
const editSubmitting = ref(false)
const editImageId = ref('')
const editAddTagsText = ref('')
const editDelTagsText = ref('')
const openEdit = (row: DockerImageSummary) => {
  editImageId.value = row.id; editAddTagsText.value = ''; editDelTagsText.value = ''; editVisible.value = true
}
const submitEdit = async () => {
  const addTags = editAddTagsText.value.trim().split('\n').filter(Boolean)
  const deleteTags = editDelTagsText.value.trim().split('\n').filter(Boolean)
  if (!addTags.length && !deleteTags.length) { ElMessage.error(t('docker.image.enterTagsToEdit')); return }
  editSubmitting.value = true
  try {
    await updateDockerImageTags({ imageId: editImageId.value, addTags, deleteTags, forceDelete: false })
    ElMessage.success(t('docker.image.tagUpdateSuccess'))
    editVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, t('docker.image.tagUpdateFailed')) }
  finally { editSubmitting.value = false }
}

onMounted(() => { getList() })
</script>

<style scoped lang="scss">
.docker-page { .card-mt-16 { margin-top: 16px; } }
.operation-container { margin-bottom: 12px; display: flex; gap: 8px; flex-wrap: wrap; }
.text-muted { color: var(--el-text-color-placeholder); font-size: 0.82rem; }
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
