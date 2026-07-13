<!-- Docker 镜像（重写 · P1 列表）：PageHeader(拉取镜像/任务) + FilterBar(关键字/悬空) + DataTable/移动卡 + Pagination。
     行内 详情/编辑标签/打标签/历史/删除（ActionMenu，管理权限门控）。弹窗全部走 FormDialog；拉取进度沿用 DockerTaskDialogs。 -->
<template>
  <div class="dk-page">
    <PageHeader :title="t('docker.image.title')" :description="t('docker.image.pageDesc')">
      <template #actions>
        <UButton color="neutral" variant="soft" icon="i-lucide-clock" @click="openTasks">
          {{ t('docker.common.tasks') }}
        </UButton>
        <UButton v-if="canManage" color="primary" icon="i-lucide-download" @click="openPull">
          {{ t('docker.image.pullImage') }}
        </UButton>
      </template>
    </PageHeader>

    <FilterBar @search="search" @reset="reset">
      <template #fields>
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.keyword') }}</label>
          <input
            v-model="queryForm.keyword"
            class="app-input"
            :placeholder="t('docker.image.keywordPlaceholder')"
            @keyup.enter="search"
          />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.image.dangling') }}</label>
          <AppSelect v-model="queryForm.dangling" :options="danglingOptions" />
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
        :empty-text="t('docker.image.noImages')"
      >
        <template #cell-id="{ row }">
          <span class="dk-mono">{{ shortId(row.id) }}</span>
        </template>
        <template #cell-repoTags="{ row }">
          <div v-if="tagsOf(row).length" class="dk-tags">
            <span v-for="tag in tagsOf(row).slice(0, 3)" :key="tag" class="dk-tag">{{ tag }}</span>
            <span v-if="tagsOf(row).length > 3" class="dk-dim">+{{ tagsOf(row).length - 3 }}</span>
          </div>
          <span v-else class="dk-dim">&lt;none&gt;</span>
        </template>
        <template #cell-size="{ row }">{{ formatBytes(row.size) }}</template>
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
          icon="HOutline:CubeIcon"
          :title="t('docker.image.noImages')"
          :description="t('docker.image.emptyDesc')"
        />
        <div v-else class="dk-cards">
          <EntityCard v-for="row in list" :key="row.id">
            <template #title>{{ tagsOf(row)[0] || '<none>' }}</template>
            <template #meta>
              <span class="dk-mono">{{ shortId(row.id) }}</span>
              <span>{{ formatBytes(row.size) }} · {{ t('docker.image.containerCount') }} {{ row.containers }}</span>
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

    <!-- 拉取镜像 -->
    <FormDialog v-model="pullVisible" :title="t('docker.image.pullImage')" :width="460" :loading="pullSubmitting" @confirm="submitPull">
      <div class="dk-form">
        <div class="app-field">
          <label class="app-label app-label--required">{{ t('docker.image.imageReference') }}</label>
          <input v-model="pullForm.reference" class="app-input" :placeholder="t('docker.image.referencePlaceholder')" @keyup.enter="submitPull" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.common.platform') }}</label>
          <input v-model="pullForm.platform" class="app-input" placeholder="linux/amd64" />
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="pullSubmitting" @click="submitPull">{{ t('docker.image.pull') }}</UButton>
      </template>
    </FormDialog>

    <!-- 打标签 -->
    <FormDialog v-model="tagVisible" :title="t('docker.image.tagImage')" :width="420" :loading="tagSubmitting" @confirm="submitTag">
      <div class="app-field">
        <label class="app-label app-label--required">{{ t('docker.image.newTag') }}</label>
        <input v-model="tagTarget" class="app-input" :placeholder="t('docker.image.newTagPlaceholder')" @keyup.enter="submitTag" />
      </div>
    </FormDialog>

    <!-- 编辑标签 -->
    <FormDialog v-model="editVisible" :title="t('docker.image.editImageTags')" :width="500" :loading="editSubmitting" @confirm="submitEdit">
      <div class="dk-form">
        <div class="app-field">
          <label class="app-label">{{ t('docker.image.addTags') }}</label>
          <textarea v-model="editAddTagsText" class="app-textarea" rows="3" :placeholder="t('docker.image.oneTagPerLine')" />
        </div>
        <div class="app-field">
          <label class="app-label">{{ t('docker.image.deleteTags') }}</label>
          <textarea v-model="editDelTagsText" class="app-textarea" rows="3" :placeholder="t('docker.image.oneTagPerLine')" />
        </div>
      </div>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('docker.common.cancel') }}</UButton>
        <UButton color="primary" :loading="editSubmitting" @click="submitEdit">{{ t('docker.common.save') }}</UButton>
      </template>
    </FormDialog>

    <!-- 镜像详情 -->
    <FormDialog v-model="detailVisible" :title="t('docker.image.imageDetail')" :width="720" :show-footer="false">
      <div v-if="detail" class="dk-detail">
        <div class="dk-detail__hero">
          <div class="dk-detail__title">
            <span>{{ (detail.repoTags || [])[0] || '<none>' }}</span>
            <StatusPill variant="info" :label="formatBytes(detail.size)" :dot="false" />
          </div>
          <div class="dk-detail__sub">{{ shortId(detail.id) }} · {{ detail.architecture || '-' }} / {{ detail.os || '-' }}</div>
        </div>
        <div class="dk-kv">
          <div><span>ID</span><strong>{{ shortId(detail.id) }}</strong></div>
          <div><span>{{ t('docker.common.size') }}</span><strong>{{ formatBytes(detail.size) }}</strong></div>
          <div><span>{{ t('docker.common.architecture') }}</span><strong>{{ detail.architecture || '-' }}</strong></div>
          <div><span>{{ t('docker.common.os') }}</span><strong>{{ detail.os || '-' }}</strong></div>
          <div><span>{{ t('docker.common.author') }}</span><strong>{{ detail.author || '-' }}</strong></div>
          <div><span>{{ t('docker.common.createdAt') }}</span><strong>{{ formatTime(detail.created) }}</strong></div>
        </div>
        <div v-if="detail.repoTags?.length" class="dk-detail__block">
          <div class="dk-detail__label">{{ t('docker.image.repoTags') }}</div>
          <div class="dk-tags">
            <span v-for="tag in detail.repoTags" :key="tag" class="dk-tag">{{ tag }}</span>
          </div>
        </div>
        <div v-if="detail.repoDigests?.length" class="dk-detail__block">
          <div class="dk-detail__label">{{ t('docker.common.summary') }}</div>
          <div class="dk-code">
            <div v-for="digest in detail.repoDigests" :key="digest">{{ digest }}</div>
          </div>
        </div>
      </div>
      <div v-else class="dk-detail__loading">{{ t('docker.common.noData') }}</div>
    </FormDialog>

    <!-- 镜像历史 -->
    <FormDialog v-model="historyVisible" :title="t('docker.image.imageHistory')" :width="820" :show-footer="false">
      <DataTable
        :columns="historyColumns"
        :rows="historyRows"
        row-key="_idx"
        :loading="historyLoading"
        :empty-text="t('docker.common.noData')"
      >
        <template #cell-id="{ row }"><span class="dk-mono">{{ shortId(String(row.id || '')) }}</span></template>
        <template #cell-size="{ row }">{{ formatBytes(row.size) }}</template>
        <template #cell-created="{ row }">{{ formatTime(row.created) }}</template>
      </DataTable>
    </FormDialog>

    <DockerTaskDialogs v-model="taskDialogsVisible" :active-task="activeTask" @finished="handleTaskFinished" />
  </div>
</template>

<script setup lang="ts">
import { useI18n } from 'vue-i18n'
import {
  deleteDockerImage, getDockerImage, imageHistory,
  listDockerImages, pullDockerImage, tagDockerImage, updateDockerImageTags,
} from '@/api/docker'
import DockerTaskDialogs from '@/views/docker/components/DockerTaskDialogs.vue'
import { PERM } from '@/config/permission'
import { useButtonPermission } from '@/composables/useButtonPermission'
import { Dialog } from '@/utils/dialog'
import { showRequestError } from '@/utils/request'
import type { DockerImageHistoryItem, DockerImageInfo, DockerImageSummary, DockerTaskInfo } from '@/types/v1/docker'
import type { DataTableColumn } from '@/components/ui/DataTable.vue'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'

defineOptions({ name: 'DockerImageView' })

const menuStore = useMenuStore()
const { t } = useI18n()
const canManage = useButtonPermission([PERM.DOCKER_IMAGE_MANAGE], [])

const list = ref<DockerImageSummary[]>([])
const loading = ref(false)
type DanglingFilter = '' | 'true' | 'false'
const queryForm = reactive({ keyword: '', dangling: '' as DanglingFilter })
const pagination = ref({ page: 1, pageSize: 10, total: 0 })

const danglingOptions = computed<{ label: string; value: DanglingFilter }[]>(() => [
  { label: t('docker.common.all'), value: '' },
  { label: t('docker.common.yes'), value: 'true' },
  { label: t('docker.common.no'), value: 'false' },
])

const formatBytes = (bytes?: number | string | unknown) => {
  const n = Number(bytes)
  if (!n) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) { v /= 1024; i++ }
  return `${v.toFixed(i > 0 ? 1 : 0)} ${units[i]}`
}
const pad = (n: number) => String(n).padStart(2, '0')
const formatTime = (value: unknown): string => {
  if (!value) return '—'
  const d = new Date(value as string | Date)
  if (Number.isNaN(d.getTime())) return '—'
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}
const shortId = (id?: unknown) => String(id || '').replace(/^sha256:/, '').slice(0, 12) || '-'
const tagsOf = (row: Record<string, unknown>) => (row.repoTags as string[] | undefined) || []

const columns = computed<DataTableColumn[]>(() => [
  { key: 'id', title: 'ID', width: 130 },
  { key: 'repoTags', title: t('docker.image.repoTags'), minWidth: 220 },
  { key: 'size', title: t('docker.common.size'), width: 100 },
  { key: 'containers', title: t('docker.image.containerCount'), width: 80 },
  { key: 'created', title: t('docker.common.createdAt'), width: 150 },
  { key: 'operation', title: t('docker.common.operation'), width: 80, align: 'center' },
])

const rowActionsFor = (row: Record<string, unknown>): ActionMenuItem[] => {
  const manage = canManage.value
  const containers = Number(row.containers) || 0
  return [
    { key: 'detail', label: t('docker.common.detail'), icon: 'HOutline:InformationCircleIcon' },
    { key: 'editTags', label: t('docker.image.editTags'), icon: 'HOutline:PencilSquareIcon', hidden: !manage },
    { key: 'tag', label: t('docker.image.tagImage'), icon: 'HOutline:TagIcon', hidden: !manage },
    { key: 'history', label: t('docker.image.history'), icon: 'HOutline:ClockIcon', hidden: !manage },
    { key: 'delete', label: t('docker.common.delete'), icon: 'HOutline:TrashIcon', danger: true, hidden: !manage || containers > 0 },
  ]
}

const findRow = (id: string) => list.value.find((x) => x.id === id)
const onRowAction = (key: string, row: Record<string, unknown>) => {
  const record = findRow(String(row.id))
  if (!record) return
  if (key === 'detail') openDetail(record)
  else if (key === 'editTags') openEdit(record)
  else if (key === 'tag') openTag(record)
  else if (key === 'history') openHistory(record)
  else if (key === 'delete') handleDelete(record)
}

const getList = async () => {
  loading.value = true
  try {
    const { data } = await listDockerImages({
      all: false, page: pagination.value.page, pageSize: pagination.value.pageSize,
      keyword: queryForm.keyword || '',
      dangling: queryForm.dangling === '' ? undefined : queryForm.dangling === 'true',
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
const reset = () => { queryForm.keyword = ''; queryForm.dangling = ''; search() }

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

// —— 拉取 ——
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

// —— 打标签 ——
const tagVisible = ref(false)
const tagSubmitting = ref(false)
const tagTarget = ref('')
const tagId = ref('')
const openTag = (row: DockerImageSummary) => { tagId.value = row.id; tagTarget.value = ''; tagVisible.value = true }
const submitTag = async () => {
  const target = tagTarget.value.trim()
  if (!target) { ElMessage.error(t('docker.image.enterNewTag')); return }
  tagSubmitting.value = true
  try {
    await tagDockerImage({ id: tagId.value, target })
    ElMessage.success(t('docker.image.tagSuccess'))
    tagVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, t('docker.image.tagFailed')) }
  finally { tagSubmitting.value = false }
}

// —— 删除 ——
const handleDelete = (row: DockerImageSummary) => {
  const name = (row.repoTags || ['<none>'])[0]
  Dialog.confirm({
    title: t('docker.image.confirmDeleteTitle'),
    content: t('docker.image.confirmDeleteContent', { name }),
    confirmText: t('docker.image.confirmDeleteText'),
    cancelText: t('docker.common.cancel'),
    onConfirm: async () => {
      try {
        await deleteDockerImage({ id: row.id, force: false, pruneChildren: false })
        ElMessage.success(t('docker.common.deletedName', { name }))
        await getList()
      } catch (e) { showRequestError(e, t('docker.image.deleteFailed')) }
    },
  })
}

// —— 详情 ——
const detailVisible = ref(false)
const detail = ref<DockerImageInfo | null>(null)
const openDetail = async (row: DockerImageSummary) => {
  detailVisible.value = true
  detail.value = null
  try { const { data } = await getDockerImage({ id: row.id }); detail.value = data?.info || null }
  catch (e) { showRequestError(e, t('docker.image.getDetailFailed')) }
}

// —— 历史 ——
const historyVisible = ref(false)
const historyLoading = ref(false)
const historyItems = ref<DockerImageHistoryItem[]>([])
const historyRows = computed(() =>
  historyItems.value.map((item, idx) => ({ ...item, _idx: idx })),
)
const historyColumns = computed<DataTableColumn[]>(() => [
  { key: 'id', title: t('docker.common.id'), width: 130 },
  { key: 'createdBy', title: t('docker.image.createdBy'), minWidth: 280 },
  { key: 'size', title: t('docker.common.size'), width: 100 },
  { key: 'created', title: t('docker.common.createdAt'), width: 150 },
])
const openHistory = async (row: DockerImageSummary) => {
  historyVisible.value = true
  historyLoading.value = true
  historyItems.value = []
  try { const { data } = await imageHistory({ id: row.id }); historyItems.value = data?.items || [] }
  catch (e) { showRequestError(e, t('docker.image.getHistoryFailed')) }
  finally { historyLoading.value = false }
}

// —— 编辑标签 ——
const editVisible = ref(false)
const editSubmitting = ref(false)
const editImageId = ref('')
const editAddTagsText = ref('')
const editDelTagsText = ref('')
const openEdit = (row: DockerImageSummary) => {
  editImageId.value = row.id
  editAddTagsText.value = ''
  editDelTagsText.value = ''
  editVisible.value = true
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
.dk-mono {
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: 0.8125rem;
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
  max-width: 240px;
  padding: 1px 8px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  border-radius: var(--app-radius-xs);
  background: var(--el-fill-color-light);
  color: var(--el-text-color-regular);
  font-size: 0.75rem;
}

.dk-form {
  display: flex;
  flex-direction: column;
  gap: 14px;
}

/* 详情 */
.dk-detail {
  display: flex;
  flex-direction: column;
  gap: 14px;
}
.dk-detail__hero {
  padding: 12px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-fill-color-lighter);
}
.dk-detail__title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-weight: 700;
  color: var(--el-text-color-primary);
}
.dk-detail__title > span:first-child {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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
.dk-kv span {
  display: block;
  color: var(--el-text-color-secondary);
  font-size: 0.72rem;
}
.dk-kv strong {
  display: block;
  margin-top: 2px;
  color: var(--el-text-color-primary);
  font-size: 0.8rem;
  font-weight: 600;
  word-break: break-word;
}
.dk-detail__label {
  margin-bottom: 6px;
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-text-color-primary);
}
.dk-code {
  padding: 8px 10px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius);
  background: var(--el-fill-color-lighter);
  color: var(--el-text-color-regular);
  font-size: 0.75rem;
  line-height: 1.6;
  word-break: break-all;
}
.dk-detail__loading {
  padding: 40px 0;
  text-align: center;
  color: var(--el-text-color-secondary);
}

/* 移动卡片 */
.dk-cards {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.dk-skeleton {
  height: 108px;
  border-radius: var(--app-radius);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: dk-shimmer 1.4s ease-in-out infinite;
}
@keyframes dk-shimmer {
  from { background-position: 200% 0; }
  to { background-position: -200% 0; }
}
</style>
