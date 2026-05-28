<template>
  <div class="docker-page">
    <el-card shadow="never">
      <el-form :model="queryForm" label-width="auto" @keyup.enter="search">
        <el-row :gutter="10">
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="关键字">
              <el-input v-model="queryForm.keyword" placeholder="仓库名 / 标签" clearable />
            </el-form-item>
          </el-col>
          <el-col :xs="24" :sm="12" :md="8" :lg="6">
            <el-form-item label="悬空">
              <el-select v-model="queryForm.dangling" placeholder="全部" clearable style="width: 100%">
                <el-option label="是" :value="true" />
                <el-option label="否" :value="false" />
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
        <el-button type="primary" :icon="menuStore.iconComponents.Plus" :disabled="!canManage" @click="openPull">拉取镜像</el-button>
        <el-button type="success" :disabled="!canManage" @click="openBuild">构建镜像</el-button>
        <el-button type="warning" :disabled="!canManage" @click="handlePrune">清理未使用镜像</el-button>
        <el-button :icon="menuStore.iconComponents['HOutline:ClockIcon']" @click="openTasks">任务</el-button>
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
            <el-button type="primary" link size="small" @click="openDetail(row)">详情</el-button>
            <template v-if="canManage">
              <el-button type="primary" link size="small" @click="openEdit(row)">编辑标签</el-button>
              <el-button type="primary" link size="small" @click="openTag(row)">打标签</el-button>
              <el-button type="primary" link size="small" @click="openHistory(row)">历史</el-button>
              <el-button type="danger" link size="small" @click="handleDelete(row)">删除</el-button>
            </template>
          </template>
        </VxeGrid>

        <div v-else-if="list.length" class="mobile-card-list">
          <div v-for="row in list" :key="row.id" class="mobile-card">
            <div class="mobile-card-body">
              <div class="mobile-card-header">
                <span class="mobile-card-title">{{ (row.repoTags || ['<none>'])[0] }}</span>
              </div>
              <div class="mobile-card-meta"><span>ID：{{ row.id?.slice(7, 19) }}</span></div>
              <div class="mobile-card-meta"><span>大小：{{ formatBytes(row.size) }} / 容器数：{{ row.containers }}</span></div>
            </div>
            <div class="mobile-card-actions">
              <el-button size="small" plain type="primary" @click="openDetail(row)">详情</el-button>
              <el-button v-if="canManage" size="small" plain type="danger" @click="handleDelete(row)">删除</el-button>
            </div>
          </div>
        </div>
        <el-empty v-else-if="!loading" description="暂无镜像" />
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
    <BaseDialog v-model="pullVisible" title="拉取镜像" width="450">
      <el-form label-position="top">
        <el-form-item label="镜像引用" required>
          <el-input v-model="pullForm.reference" placeholder="如 nginx:latest" />
        </el-form-item>
        <el-form-item label="平台">
          <el-input v-model="pullForm.platform" placeholder="如 linux/amd64（留空使用默认）" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="pullVisible = false">取消</el-button>
        <el-button type="primary" :loading="pullSubmitting" @click="submitPull">拉取</el-button>
      </template>
    </BaseDialog>

    <!-- Build Dialog -->
    <BaseDialog v-model="buildVisible" title="构建镜像" width="500">
      <el-form label-position="top">
        <el-form-item label="上下文路径" required>
          <el-input v-model="buildForm.contextPath" placeholder="/path/to/context" />
        </el-form-item>
        <el-form-item label="Dockerfile 路径">
          <el-input v-model="buildForm.dockerfile" placeholder="相对上下文路径，默认 Dockerfile" />
        </el-form-item>
        <el-form-item label="镜像标签">
          <el-input v-model="buildForm.tagsText" placeholder="myimage:latest（每行一个标签）" type="textarea" :rows="2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="buildVisible = false">取消</el-button>
        <el-button type="primary" :loading="buildSubmitting" @click="submitBuild">构建</el-button>
      </template>
    </BaseDialog>

    <!-- Tag Dialog -->
    <BaseDialog v-model="tagVisible" title="打标签" width="400">
      <el-form label-position="top">
        <el-form-item label="新标签" required>
          <el-input v-model="tagTarget" placeholder="如 myimage:v2" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="tagVisible = false">取消</el-button>
        <el-button type="primary" :loading="tagSubmitting" @click="submitTag">确认</el-button>
      </template>
    </BaseDialog>

    <!-- Detail Dialog -->
    <BaseDialog v-model="detailVisible" title="镜像详情" width="700">
      <div v-if="detail" v-loading="detailLoading">
        <el-descriptions :column="2" border size="small">
          <el-descriptions-item label="ID">{{ detail.id }}</el-descriptions-item>
          <el-descriptions-item label="大小">{{ formatBytes(detail.size) }}</el-descriptions-item>
          <el-descriptions-item label="架构">{{ detail.architecture }}</el-descriptions-item>
          <el-descriptions-item label="操作系统">{{ detail.os }}</el-descriptions-item>
          <el-descriptions-item label="作者">{{ detail.author || '-' }}</el-descriptions-item>
          <el-descriptions-item label="创建时间">{{ detail.created }}</el-descriptions-item>
          <el-descriptions-item label="仓库标签" :span="2">
            <template v-if="detail.repoTags?.length">
              <BaseTag v-for="tag in detail.repoTags" :key="tag" :text="tag" type="info" />
            </template>
            <span v-else>-</span>
          </el-descriptions-item>
          <el-descriptions-item label="摘要" :span="2">
            <span v-if="detail.repoDigests?.length">{{ detail.repoDigests.join(', ') }}</span>
            <span v-else>-</span>
          </el-descriptions-item>
        </el-descriptions>
      </div>
      <template #footer>
        <el-button @click="detailVisible = false">关闭</el-button>
      </template>
    </BaseDialog>

    <!-- Edit Tags Dialog -->
    <BaseDialog v-model="editVisible" title="编辑镜像标签" width="500">
      <el-form label-position="top">
        <el-form-item label="新增标签">
          <el-input v-model="editAddTagsText" placeholder="每行一个标签" type="textarea" :rows="3" />
        </el-form-item>
        <el-form-item label="删除标签">
          <el-input v-model="editDelTagsText" placeholder="每行一个标签" type="textarea" :rows="3" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editVisible = false">取消</el-button>
        <el-button type="primary" :loading="editSubmitting" @click="submitEdit">保存</el-button>
      </template>
    </BaseDialog>

    <!-- History Dialog -->
    <BaseDialog v-model="historyVisible" title="镜像历史" width="800">
      <el-table :data="historyItems" size="small" border v-loading="historyLoading">
        <el-table-column type="index" label="#" width="50" />
        <el-table-column prop="id" label="ID" width="140" show-overflow-tooltip />
        <el-table-column prop="createdBy" label="创建命令" min-width="300" show-overflow-tooltip />
        <el-table-column label="大小" width="100">
          <template #default="{ row: h }">{{ formatBytes(h.size) }}</template>
        </el-table-column>
        <el-table-column label="创建时间" width="180">
          <template #default="{ row: h }">{{ formatTime(h.created) }}</template>
        </el-table-column>
        <el-table-column label="标签" width="150">
          <template #default="{ row: h }">
            <BaseTag v-for="tag in h.tags || []" :key="tag" :text="tag" type="info" />
          </template>
        </el-table-column>
      </el-table>
      <template #footer>
        <el-button @click="historyVisible = false">关闭</el-button>
      </template>
    </BaseDialog>

    <DockerTaskDrawer v-model="taskDrawerVisible" :active-task="activeTask" @finished="handleTaskFinished" />
  </div>
</template>

<script setup lang="ts">
import {
  buildDockerImage, deleteDockerImage, getDockerImage, imageHistory,
  listDockerImages, pruneDockerImages, pullDockerImage, tagDockerImage,
  updateDockerImageTags,
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
import type { DockerImageHistoryItem, DockerImageInfo, DockerImageSummary, DockerTaskInfo } from '@/types/v1/docker'
import type { VxeGridProps } from 'vxe-table'

defineOptions({ name: 'DockerImageView' })

const menuStore = useMenuStore()
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
    { field: 'repoTags', title: '仓库标签', minWidth: 220, slots: { default: 'column-repoTags' } },
    { field: 'size', title: '大小', width: 100, slots: { default: 'column-size' } },
    { field: 'containers', title: '容器数', width: 80 },
    { field: 'created', title: '创建时间', width: 170, slots: { default: 'column-created' } },
    { title: '操作', width: canManage ? 270 : 90, fixed: 'right', slots: { default: 'column-operation' } },
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

// -- pull --
const pullVisible = ref(false)
const pullSubmitting = ref(false)
const pullForm = reactive({ reference: '', platform: '' })
const openPull = () => { pullForm.reference = ''; pullForm.platform = ''; pullVisible.value = true }
const submitPull = async () => {
  const reference = pullForm.reference.trim()
  if (!reference) { ElMessage.error('请输入镜像引用'); return }
  pullSubmitting.value = true
  try {
    const { data } = await pullDockerImage({ reference, platform: pullForm.platform.trim(), registryAuth: undefined })
    ElMessage.success('拉取任务已创建')
    openTask(data?.task)
    pullVisible.value = false
  } catch (e) { showRequestError(e, '拉取镜像失败') }
  finally { pullSubmitting.value = false }
}

// -- build --
const buildVisible = ref(false)
const buildSubmitting = ref(false)
const buildForm = reactive({ contextPath: '', dockerfile: '', tagsText: '' })
const openBuild = () => { Object.assign(buildForm, { contextPath: '', dockerfile: '', tagsText: '' }); buildVisible.value = true }
const submitBuild = async () => {
  const contextPath = buildForm.contextPath.trim()
  if (!contextPath) { ElMessage.error('请输入上下文路径'); return }
  buildSubmitting.value = true
  try {
    const tags = buildForm.tagsText.trim() ? buildForm.tagsText.trim().split('\n').filter(Boolean) : []
    const { data } = await buildDockerImage({
      contextPath, dockerfile: buildForm.dockerfile.trim() || 'Dockerfile',
      tags, buildArgs: {}, labels: {}, platform: '', noCache: false,
      pullParent: false, remove: true, forceRemove: false,
    })
    ElMessage.success('构建任务已创建')
    openTask(data?.task)
    buildVisible.value = false
  } catch (e) { showRequestError(e, '构建镜像失败') }
  finally { buildSubmitting.value = false }
}

// -- tag --
const tagVisible = ref(false)
const tagSubmitting = ref(false)
const tagTarget = ref('')
const tagId = ref('')
const openTag = (row: DockerImageSummary) => { tagId.value = row.id; tagTarget.value = ''; tagVisible.value = true }
const submitTag = async () => {
  const target = tagTarget.value.trim()
  if (!target) { ElMessage.error('请输入新标签'); return }
  tagSubmitting.value = true
  try { await tagDockerImage({ id: tagId.value, target }); ElMessage.success('标签创建成功'); tagVisible.value = false; await getList() }
  catch (e) { showRequestError(e, '打标签失败') }
  finally { tagSubmitting.value = false }
}

// -- delete --
const handleDelete = async (row: DockerImageSummary) => {
  const name = (row.repoTags || ['<none>'])[0]
  try { await Dialog.confirm({ title: '确认删除镜像', content: `确定要删除镜像「${name}」吗？`, confirmText: '确认删除', cancelText: '取消' }) }
  catch { return }
  try { await deleteDockerImage({ id: row.id, force: false, pruneChildren: false }); ElMessage.success(`${name} 已删除`); await getList() }
  catch (e) { showRequestError(e, '删除镜像失败') }
}

// -- prune --
const handlePrune = async () => {
  try { await Dialog.confirm({ title: '清理未使用镜像', content: '确定要清理所有未使用的镜像吗？', confirmText: '确认清理', cancelText: '取消' }) }
  catch { return }
  try {
    const { data } = await pruneDockerImages({ danglingOnly: false })
    ElMessage.success('清理任务已创建')
    openTask(data?.task)
  }
  catch (e) { showRequestError(e, '清理镜像失败') }
}

// -- detail --
const detailVisible = ref(false)
const detailLoading = ref(false)
const detail = ref<DockerImageInfo | null>(null)
const openDetail = async (row: DockerImageSummary) => {
  detailVisible.value = true; detail.value = null; detailLoading.value = true
  try { const { data } = await getDockerImage({ id: row.id }); detail.value = data?.info || null }
  catch (e) { showRequestError(e, '获取镜像详情失败') }
  finally { detailLoading.value = false }
}

// -- history --
const historyVisible = ref(false)
const historyLoading = ref(false)
const historyItems = ref<DockerImageHistoryItem[]>([])
const openHistory = async (row: DockerImageSummary) => {
  historyVisible.value = true; historyLoading.value = true; historyItems.value = []
  try { const { data } = await imageHistory({ id: row.id }); historyItems.value = data?.items || [] }
  catch (e) { showRequestError(e, '获取镜像历史失败') }
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
  if (!addTags.length && !deleteTags.length) { ElMessage.error('请输入要新增或删除的标签'); return }
  editSubmitting.value = true
  try {
    await updateDockerImageTags({ imageId: editImageId.value, addTags, deleteTags, forceDelete: false })
    ElMessage.success('标签更新成功')
    editVisible.value = false
    await getList()
  } catch (e) { showRequestError(e, '更新标签失败') }
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
