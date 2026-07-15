<!-- 文件管理（真·重写 · P6 全屏工具页）：app 设计令牌 + 令牌组件（UButton/AppSelect/ActionMenu/
     AppDropdown/EmptyState/Pagination/StatusPill）+ heroicons（menuStore.iconComponents）。
     外壳=导航条(来源/前进后退上级/面包屑/刷新) + 双栏(左目录树 + 右主区)。
     右主区=工具栏(上传/新建/更多 · 搜索/筛选/视图切换) + 选择条 + 列表·网格 + 页脚(汇总+分页)。
     逻辑全部复用 fileClient（scope 无关，系统级/实例级同构），只重画观感。移动端单栏 + 面包屑下钻。 -->
<template>
  <div class="fb">
    <!-- 顶部导航：来源切换 + 返回/前进/上级 + 面包屑地址 + 刷新 -->
    <div class="fb-nav">
      <div v-if="isSystemScope" class="fb-source">
        <AppSelect
          :model-value="currentSourceId"
          :options="sourceOptions"
          :placeholder="t('fileSource.localDisk')"
          @update:model-value="onSourceChange"
        />
      </div>

      <div class="fb-nav__group">
        <button type="button" class="fb-ico" :disabled="!canGoBack" :title="t('fileManager.back')" @click="goBack">
          <UIcon name="i-lucide-arrow-left" />
        </button>
        <button type="button" class="fb-ico" :disabled="!canGoForward" :title="t('fileManager.forward')" @click="goForward">
          <UIcon name="i-lucide-arrow-right" />
        </button>
        <button type="button" class="fb-ico" :disabled="!canGoUp" :title="t('fileManager.up')" @click="goUp">
          <UIcon name="i-lucide-arrow-up" />
        </button>
      </div>

      <div class="fb-crumb" @click="startEditPath">
        <input
          v-if="editingPath"
          ref="pathInputRef"
          v-model="pathDraft"
          class="app-input fb-crumb__input"
          @keyup.enter="commitPath"
          @keyup.esc="editingPath = false"
          @blur="editingPath = false"
          @click.stop
        />
        <template v-else>
          <button type="button" class="fb-crumb__seg fb-crumb__root" :title="t('fileManager.rootDir')" @click.stop="navigateTo('')">
            <UIcon name="i-lucide-home" />
          </button>
          <template v-for="segment in pathSegments" :key="segment.path">
            <UIcon name="i-lucide-chevron-right" class="fb-crumb__sep" />
            <button type="button" class="fb-crumb__seg" @click.stop="navigateTo(segment.path)">{{ segment.name }}</button>
          </template>
        </template>
      </div>

      <button
        type="button"
        class="fb-ico"
        :class="{ 'is-spin': loading }"
        :title="t('fileManager.refresh')"
        @click="refresh"
      >
        <UIcon name="i-lucide-refresh-cw" />
      </button>
    </div>

    <!-- 主体：左目录树 + 右列表 -->
    <div class="fb-body">
      <aside v-if="!menuStore.isMobile" class="fb-aside">
        <FileTree
          ref="treeRef"
          :client="client"
          root-path=""
          :active-path="currentPath"
          selectable="all"
          @select="onTreeSelect"
        />
      </aside>

      <div class="fb-main">
        <!-- 工具栏 -->
        <div class="fb-tools">
          <div class="fb-tools__left">
            <UButton color="primary" size="sm" icon="i-lucide-upload" @click="uploadOpen = true">
              {{ t('fileManager.upload') }}
            </UButton>
            <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-folder-plus" @click="openCreateFolder">
              {{ t('fileManager.createFolder') }}
            </UButton>

            <AppDropdown align="start" :width="200">
              <template #trigger>
                <!-- 点击由 AppDropdown 外壳接管，勿再 @click=toggle（会开→关） -->
                <UButton color="neutral" variant="soft" size="sm" trailing-icon="i-lucide-chevron-down">
                  {{ t('fileManager.moreActions') }}
                </UButton>
              </template>
              <template #default="{ close }">
                <div class="fb-menu">
                  <template v-for="item in moreActions" :key="item.key">
                    <div v-if="item.divider" class="fb-menu__div" />
                    <button
                      v-else
                      type="button"
                      class="fb-menu__item"
                      :disabled="item.disabled"
                      @click="() => { onMoreAction(item.key); close() }"
                    >
                      <UIcon :name="item.icon" class="fb-menu__ico" />
                      <span>{{ item.label }}</span>
                    </button>
                  </template>
                </div>
              </template>
            </AppDropdown>
          </div>

          <div class="fb-tools__right">
            <div class="fb-search">
              <UIcon name="i-lucide-search" class="fb-search__ico" />
              <input v-model="keywords" class="app-input fb-search__input" :placeholder="t('fileManager.searchPlaceholder')" />
            </div>

            <AppDropdown align="end" :width="200">
              <template #trigger>
                <button
                  type="button"
                  class="fb-ico"
                  :class="{ 'is-active': includeSubDir }"
                  :title="t('fileManager.filter')"
                >
                  <UIcon name="i-lucide-funnel" />
                </button>
              </template>
              <template #default>
                <div class="fb-filter">
                  <label class="fb-filter__item">
                    <input v-model="includeSubDir" type="checkbox" @change="onFilterChange" />
                    <span>{{ t('fileManager.subdirectories') }}</span>
                  </label>
                </div>
              </template>
            </AppDropdown>

            <div class="fb-seg">
              <button
                type="button"
                class="fb-seg__btn"
                :class="{ 'is-active': viewMode === 'list' }"
                :title="t('fileManager.viewList')"
                @click="viewMode = 'list'"
              >
                <UIcon name="i-lucide-list" />
              </button>
              <button
                type="button"
                class="fb-seg__btn"
                :class="{ 'is-active': viewMode === 'grid' }"
                :title="t('fileManager.viewGrid')"
                @click="viewMode = 'grid'"
              >
                <UIcon name="i-lucide-layout-grid" />
              </button>
            </div>
          </div>
        </div>

        <!-- 选择条（选中出现，批量 下载/删除/清空，同 app 批量条范式） -->
        <div v-if="hasSelection" class="fb-selbar">
          <span class="fb-selbar__count">{{ t('fileManager.selectedCount', { count: selectedPaths.size }) }}</span>
          <div class="fb-selbar__actions">
            <UButton color="neutral" variant="soft" size="sm" icon="i-lucide-download" @click="downloadSelected">
              {{ t('fileManager.download') }}
            </UButton>
            <UButton color="error" variant="soft" size="sm" icon="i-lucide-trash-2" @click="deleteSelected">
              {{ t('fileManager.delete') }}
            </UButton>
            <UButton color="neutral" variant="ghost" size="sm" @click="clearSelection">
              {{ t('fileManager.clearSelection') }}
            </UButton>
          </div>
        </div>

        <!-- 内容区：列表 或 网格 -->
        <div class="fb-content">
          <!-- 加载骨架 -->
          <div v-if="loading" class="fb-skeleton">
            <div v-for="i in 8" :key="i" class="fb-skeleton__row" />
          </div>

          <!-- 空态 -->
          <EmptyState
            v-else-if="!items.length"
            icon="HOutline:FolderOpenIcon"
            :title="t('fileManager.directoryEmpty')"
            :description="keywords ? t('system.common.noData') : t('fileManager.uploadDropzoneTitle')"
          />

          <!-- 列表视图 -->
          <div v-else-if="viewMode === 'list'" class="fb-table">
            <div class="fb-thead">
              <div class="fb-col fb-col--check">
                <input type="checkbox" :checked="allSelected" :indeterminate.prop="someSelected" @change="toggleSelectAll" />
              </div>
              <button type="button" class="fb-col fb-col--name fb-sort" @click="sortBy('name')">
                {{ t('fileManager.tableName') }}
                <UIcon
                  v-if="sortField === FileSortField.FILE_SORT_FIELD_NAME"
                  :name="isDesc ? 'i-lucide-chevron-down' : 'i-lucide-chevron-up'"
                  class="fb-sort__ico"
                />
              </button>
              <div class="fb-col fb-col--type">{{ t('fileManager.type') }}</div>
              <div class="fb-col fb-col--size">{{ t('fileManager.size') }}</div>
              <button type="button" class="fb-col fb-col--time fb-sort" @click="sortBy('time')">
                {{ t('fileManager.updatedAt') }}
                <UIcon
                  v-if="sortField === FileSortField.FILE_SORT_FIELD_UPDATE_TIME"
                  :name="isDesc ? 'i-lucide-chevron-down' : 'i-lucide-chevron-up'"
                  class="fb-sort__ico"
                />
              </button>
              <div class="fb-col fb-col--ops">{{ t('fileManager.operation') }}</div>
            </div>

            <div class="fb-tbody">
              <div
                v-for="row in items"
                :key="row.path"
                class="fb-row"
                :class="{ 'is-selected': selectedPaths.has(row.path) }"
                @click="onRowClick(row, $event)"
              >
                <div class="fb-col fb-col--check" @click.stop>
                  <input type="checkbox" :checked="selectedPaths.has(row.path)" @change="toggleSelect(row)" />
                </div>
                <div class="fb-col fb-col--name">
                  <UIcon :name="rowIcon(row)" class="fb-row__ico" :class="row.isDir ? 'is-folder' : 'is-file'" />
                  <span class="fb-row__name" :title="row.name">{{ row.name }}</span>
                </div>
                <div class="fb-col fb-col--type">{{ typeLabel(row) }}</div>
                <div class="fb-col fb-col--size">{{ row.isDir ? '—' : formatFileSize(row.size) }}</div>
                <div class="fb-col fb-col--time">{{ formatDateTime(row.updateTime) }}</div>
                <div class="fb-col fb-col--ops" @click.stop>
                  <button
                    type="button"
                    class="fb-ico fb-ico--sm"
                    :title="t('fileManager.download')"
                    :disabled="row.isDir"
                    @click="downloadOne(row)"
                  >
                    <UIcon name="i-lucide-download" />
                  </button>
                  <ActionMenu :items="rowActions(row)" @select="(key) => onRowAction(key, row)" />
                </div>
              </div>
            </div>
          </div>

          <!-- 网格视图 -->
          <div v-else class="fb-grid">
            <div
              v-for="row in items"
              :key="row.path"
              class="fb-tile"
              :class="{ 'is-selected': selectedPaths.has(row.path) }"
              @click="onRowClick(row, $event)"
            >
              <input
                type="checkbox"
                class="fb-tile__check"
                :checked="selectedPaths.has(row.path)"
                @click.stop
                @change="toggleSelect(row)"
              />
              <div class="fb-tile__more" @click.stop>
                <ActionMenu :items="rowActions(row)" @select="(key) => onRowAction(key, row)" />
              </div>
              <UIcon :name="rowIcon(row)" class="fb-tile__ico" :class="row.isDir ? 'is-folder' : 'is-file'" />
              <span class="fb-tile__name" :title="row.name">{{ row.name }}</span>
              <span class="fb-tile__meta">{{ row.isDir ? typeLabel(row) : formatFileSize(row.size) }}</span>
            </div>
          </div>
        </div>

        <!-- 底部：汇总 + 分页 -->
        <div class="fb-foot">
          <span class="fb-foot__summary">
            {{ t('fileManager.footerSummary', { folders: folderCount, files: fileCount }) }}
          </span>
          <Pagination
            v-model:page="page"
            v-model:page-size="pageSize"
            :total="total"
            @change="loadList"
          />
        </div>
      </div>
    </div>

    <!-- 子弹窗 -->
    <FilePromptDialog
      v-model="prompt.open"
      :title="prompt.title"
      :label="prompt.label"
      :description="prompt.description"
      :placeholder="prompt.placeholder"
      :initial-value="prompt.initialValue"
      :confirming="prompt.confirming"
      @confirm="onPromptConfirm"
    />

    <FileMediaDialog v-model="media.open" :name="media.name" :kind="media.kind" :url="media.url" />

    <FileUploadDialog v-model="uploadOpen" :client="client" :target-path="currentPath" @uploaded="afterMutation" />

    <FileEditor
      v-model="editor.open"
      :client="client"
      :path="editor.path"
      root-path=""
      @saved="loadList"
      @renamed="afterMutation"
      @deleted="afterMutation"
    />

    <ShareFormDialog v-model="shareOpen" :items="shareItems" />

    <!-- 删除确认（令牌 FormDialog，替代 EP 命令式确认） -->
    <FormDialog v-model="del.open" :title="t('fileManager.confirmDelete')" :width="420" :loading="del.pending" @confirm="confirmDelete">
      <p class="fb-confirm">{{ del.message }}</p>
      <template #footer="{ close }">
        <UButton color="neutral" variant="soft" @click="close">{{ t('system.common.cancel') }}</UButton>
        <UButton color="error" :loading="del.pending" @click="confirmDelete">{{ t('system.common.confirm') }}</UButton>
      </template>
    </FormDialog>
  </div>
</template>

<script setup lang="ts">
import { useDebounceFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { getRequestErrorMessage } from '@/utils/request'
import { useFeedback } from '@/utils/feedback'
import {
  FileSortField,
  type FileEntryInfo,
  type FileDirectoryInfo,
  type FileSourceInfo,
  type FileSourceCaps,
} from '@/types/v1/file'
import { listFileSourcesRequest } from '@/api/fileSource'
import {
  formatDateTime,
  formatFileSize,
  getFileExtension,
  getParentPath,
  isMediaFile,
  joinPath,
  resolveFilePreviewKind,
  resolvePreSignedFileUrl,
  splitPathSegments,
  downloadFileFromUrl,
  copyTextToClipboard,
  isFileTooLargeForEditor,
  type FilePreviewKind,
} from '@/utils/file'
import { createFileClient } from './fileClient'
import type { FileClient, FileClipboard, FileScope, PickedFile } from './types'
import type { ActionMenuItem } from '@/components/ui/ActionMenu.vue'
import FileTree from './FileTree.vue'
import FilePromptDialog from './FilePromptDialog.vue'
import FileMediaDialog from './FileMediaDialog.vue'
import FileUploadDialog from './FileUploadDialog.vue'
import FileEditor from './FileEditor.vue'
import ShareFormDialog from '@/components/share/ShareFormDialog.vue'

const props = defineProps<{
  scope: FileScope
  initialPath?: string
}>()

const { t } = useI18n()
const menuStore = useMenuStore()
const fb = useFeedback()

// 请求错误反馈：走统一 toast（getRequestErrorMessage 解析后端消息）。
const notifyError = (error: unknown, fallback: string) => fb.error(getRequestErrorMessage(error, fallback))

// ---- 视图模式（列表/网格，持久化）----
const viewMode = ref<'list' | 'grid'>(
  (localStorage.getItem('fm-view') as 'list' | 'grid') === 'grid' ? 'grid' : 'list',
)
watch(viewMode, (v) => localStorage.setItem('fm-view', v))

// ---- 文件来源（仅系统级显示切换）----
const isSystemScope = props.scope.kind === 'system'
const currentSourceId = ref('')
const sources = ref<FileSourceInfo[]>([])
const localCaps: FileSourceCaps = {
  presign: false,
  copy: true,
  move: true,
  compress: true,
  resumableUpload: true,
}
const currentCaps = computed<FileSourceCaps>(() => {
  if (!currentSourceId.value) return localCaps
  return sources.value.find((s) => s.id === currentSourceId.value)?.caps ?? localCaps
})
const sourceOptions = computed<{ label: string; value: string }[]>(() => [
  { label: t('fileSource.localDisk'), value: '' },
  ...sources.value.map((s) => ({ label: `${s.name}（${s.type.toUpperCase()}）`, value: s.id })),
])

// scope 在组件生命周期内稳定；source_id 动态读取，切换来源无需重建 client。
const client: FileClient = createFileClient(props.scope, () => currentSourceId.value)

// 左侧目录树引用：结构性变更（新建/删除/重命名/粘贴/切换来源）后重载。
const treeRef = ref<InstanceType<typeof FileTree> | null>(null)
const reloadTree = () => treeRef.value?.reload()
// 变更后统一刷新列表 + 目录树
const afterMutation = () => {
  loadList()
  reloadTree()
}

const loadSources = async () => {
  if (!isSystemScope) return
  try {
    const { data } = await listFileSourcesRequest({ enabledOnly: true })
    sources.value = data.items ?? []
  } catch {
    sources.value = []
  }
}

const onSourceChange = (id: string) => {
  // 切换来源：清空历史/选择，回到该来源根目录。
  currentSourceId.value = id
  history.value = []
  historyIndex.value = -1
  selectedPaths.value = new Set()
  clipboard.value = null
  keywords.value = ''
  page.value = 1
  currentPath.value = ''
  afterMutation()
}

// ---- 列表状态 ----
const currentPath = ref(props.initialPath || '')
const directory = ref<FileDirectoryInfo>()
const items = ref<FileEntryInfo[]>([])
const loading = ref(false)
const page = ref(1)
const pageSize = ref(20)
const total = ref(0)
const keywords = ref('')
const includeSubDir = ref(false)
const sortField = ref<FileSortField>(FileSortField.FILE_SORT_FIELD_NAME)
const isDesc = ref(false)
const selectedPaths = ref<Set<string>>(new Set())

// ---- 导航历史 ----
const history = ref<string[]>([])
const historyIndex = ref(-1)
const canGoBack = computed(() => historyIndex.value > 0)
const canGoForward = computed(() => historyIndex.value < history.value.length - 1)
const canGoUp = computed(
  () => !!currentPath.value && getParentPath(currentPath.value) !== currentPath.value,
)

// ---- 面包屑 ----
const pathSegments = computed(() => splitPathSegments(currentPath.value))
const editingPath = ref(false)
const pathDraft = ref('')
const pathInputRef = ref<HTMLInputElement | null>(null)

// ---- 剪贴板 ----
const clipboard = ref<FileClipboard | null>(null)

// ---- 弹窗状态 ----
type PromptMode = 'createFolder' | 'createFile' | 'rename' | 'compress' | 'unzip'
const prompt = reactive({
  open: false,
  mode: 'createFolder' as PromptMode,
  title: '',
  label: '' as string | undefined,
  description: '' as string | undefined,
  placeholder: '' as string | undefined,
  initialValue: '' as string | undefined,
  confirming: false,
  targets: [] as FileEntryInfo[],
})

const media = reactive({
  open: false,
  name: '',
  kind: 'image' as FilePreviewKind,
  url: '',
})

const editor = reactive({ open: false, path: '' })
const uploadOpen = ref(false)
const shareOpen = ref(false)
const shareItems = ref<PickedFile[]>([])

// ---- 选择 ----
const selectedRows = computed(() =>
  items.value.filter((item) => selectedPaths.value.has(item.path)),
)
const hasSelection = computed(() => selectedPaths.value.size > 0)
const allSelected = computed(
  () => items.value.length > 0 && items.value.every((item) => selectedPaths.value.has(item.path)),
)
const someSelected = computed(() => hasSelection.value && !allSelected.value)

const toggleSelect = (row: FileEntryInfo) => {
  const next = new Set(selectedPaths.value)
  if (next.has(row.path)) next.delete(row.path)
  else next.add(row.path)
  selectedPaths.value = next
}
const toggleSelectAll = () => {
  selectedPaths.value = allSelected.value
    ? new Set()
    : new Set(items.value.map((item) => item.path))
}
const clearSelection = () => {
  selectedPaths.value = new Set()
}

// ---- 汇总 ----
const folderCount = computed(() => Number(directory.value?.dirCount ?? 0))
const fileCount = computed(() => Number(directory.value?.fileCount ?? 0))

// ---- 加载 ----
// 返回是否成功：导航失败时调用方必须回滚 currentPath / 历史，避免「报错了却已经切进去」。
const loadList = async (): Promise<boolean> => {
  loading.value = true
  try {
    const res = await client.list({
      path: currentPath.value,
      page: page.value,
      pageSize: pageSize.value,
      keywords: keywords.value.trim() || undefined,
      includeSubDir: includeSubDir.value || undefined,
      sortField: sortField.value,
      isDesc: isDesc.value,
    })
    directory.value = res.directory
    items.value = res.items
    total.value = Number(res.total) || 0
    selectedPaths.value = new Set()
    if (res.directory?.path) currentPath.value = res.directory.path
    // 首次加载后用规范化路径播种历史
    if (historyIndex.value === -1) {
      history.value = [currentPath.value]
      historyIndex.value = 0
    }
    return true
  } catch (error) {
    notifyError(error, t('fileManager.refreshFailed'))
    return false
  } finally {
    loading.value = false
  }
}

const refresh = () => afterMutation()

// ---- 导航（失败回滚路径，禁止“报错却切进去”）----
const navigateTo = async (path: string, pushHistory = true) => {
  const prevPath = currentPath.value
  const prevPage = page.value
  const prevKeywords = keywords.value
  const prevHistory = history.value.slice()
  const prevHistoryIndex = historyIndex.value

  if (pushHistory) {
    history.value = history.value.slice(0, historyIndex.value + 1)
    history.value.push(path)
    historyIndex.value = history.value.length - 1
  }
  currentPath.value = path
  page.value = 1
  keywords.value = ''
  const ok = await loadList()
  if (!ok) {
    currentPath.value = prevPath
    page.value = prevPage
    keywords.value = prevKeywords
    history.value = prevHistory
    historyIndex.value = prevHistoryIndex
  }
}
const goBack = async () => {
  if (!canGoBack.value) return
  const prevIndex = historyIndex.value
  const prevPath = currentPath.value
  const prevPage = page.value
  historyIndex.value -= 1
  currentPath.value = history.value[historyIndex.value] ?? currentPath.value
  page.value = 1
  const ok = await loadList()
  if (!ok) {
    historyIndex.value = prevIndex
    currentPath.value = prevPath
    page.value = prevPage
  }
}
const goForward = async () => {
  if (!canGoForward.value) return
  const prevIndex = historyIndex.value
  const prevPath = currentPath.value
  const prevPage = page.value
  historyIndex.value += 1
  currentPath.value = history.value[historyIndex.value] ?? currentPath.value
  page.value = 1
  const ok = await loadList()
  if (!ok) {
    historyIndex.value = prevIndex
    currentPath.value = prevPath
    page.value = prevPage
  }
}
const goUp = () => {
  if (!canGoUp.value) return
  void navigateTo(getParentPath(currentPath.value))
}

const startEditPath = () => {
  if (editingPath.value) return
  pathDraft.value = currentPath.value
  editingPath.value = true
  nextTick(() => pathInputRef.value?.focus())
}
const commitPath = () => {
  editingPath.value = false
  const target = pathDraft.value.trim()
  if (target && target !== currentPath.value) navigateTo(target)
}

// ---- 目录树点击：目录→进入，文件→打开 ----
const onTreeSelect = (path: string, name: string, isDir: boolean) => {
  if (isDir) {
    if (path !== currentPath.value) navigateTo(path)
    return
  }
  openFileByName(path, name, 0)
}

// ---- 搜索/筛选 ----
const debouncedReload = useDebounceFn(() => {
  page.value = 1
  loadList()
}, 350)
watch(keywords, () => debouncedReload())

const onFilterChange = () => {
  page.value = 1
  loadList()
}

// ---- 排序 ----
const sortBy = (field: 'name' | 'time') => {
  const target =
    field === 'name'
      ? FileSortField.FILE_SORT_FIELD_NAME
      : FileSortField.FILE_SORT_FIELD_UPDATE_TIME
  if (sortField.value === target) {
    isDesc.value = !isDesc.value
  } else {
    sortField.value = target
    isDesc.value = false
  }
  page.value = 1
  loadList()
}

// ---- 行展示（Lucide iconify name，与 UButton 同源）----
const rowIcon = (row: FileEntryInfo) => {
  if (row.isDir) return 'i-lucide-folder'
  const kind = resolveFilePreviewKind(row.name)
  if (kind === 'image') return 'i-lucide-image'
  if (kind === 'video') return 'i-lucide-film'
  if (kind === 'audio') return 'i-lucide-music'
  return 'i-lucide-file'
}
const typeLabel = (row: FileEntryInfo) => {
  if (row.isDir) return t('fileManager.typeFolder')
  const ext = getFileExtension(row.name)
  const extUpper = ext ? ext.toUpperCase() : ''
  const kind = resolveFilePreviewKind(row.name)
  if (kind === 'image') return t('fileManager.typeImage', { ext: extUpper })
  if (kind === 'video') return t('fileManager.typeVideo', { ext: extUpper })
  if (kind === 'audio') return t('fileManager.typeAudio', { ext: extUpper })
  if (ext) return t('fileManager.typeNamed', { ext: extUpper })
  return t('fileManager.typeFile')
}

// ---- 行交互 ----
const onRowClick = (row: FileEntryInfo, event: MouseEvent) => {
  // Ctrl/⌘ 单击 = 多选；普通单击：目录进入、文件打开（单击即打开）
  if (event.ctrlKey || event.metaKey) {
    toggleSelect(row)
    return
  }
  onRowOpen(row)
}
const onRowOpen = (row: FileEntryInfo) => {
  if (row.isDir) {
    navigateTo(row.path)
    return
  }
  if (isMediaFile(row.name)) {
    openMedia(row.path, row.name)
    return
  }
  if (isFileTooLargeForEditor(row.path, row.size)) {
    fb.warning(t('fileManager.fileTooLargeForEditor'))
    return
  }
  editor.path = row.path
  editor.open = true
}
// 目录树点击文件：无 size 信息，媒体走预览，其它进编辑器（大小校验在编辑器内兜底）。
const openFileByName = (path: string, name: string, size: number) => {
  if (isMediaFile(name)) {
    openMedia(path, name)
    return
  }
  if (size && isFileTooLargeForEditor(path, size)) {
    fb.warning(t('fileManager.fileTooLargeForEditor'))
    return
  }
  editor.path = path
  editor.open = true
}

// ---- 媒体预览 ----
const openMedia = async (path: string, name: string) => {
  try {
    const downloadPath = await client.preSignDownload(path, true)
    media.url = resolvePreSignedFileUrl(downloadPath)
    media.name = name
    media.kind = (resolveFilePreviewKind(name) || 'image') as FilePreviewKind
    media.open = true
  } catch (error) {
    notifyError(error, t('fileManager.openFailed'))
  }
}

// ---- 创建 / 重命名 / 压缩 / 解压（共用 prompt） ----
const openCreateFolder = () => {
  Object.assign(prompt, {
    mode: 'createFolder',
    title: t('fileManager.createFolder'),
    label: '',
    description: t('fileManager.currentDirectory', { path: currentPath.value }),
    placeholder: t('fileManager.createFolderPlaceholder'),
    initialValue: '',
    targets: [],
    open: true,
  })
}
const openCreateFile = () => {
  Object.assign(prompt, {
    mode: 'createFile',
    title: t('fileManager.createFile'),
    label: '',
    description: t('fileManager.currentDirectory', { path: currentPath.value }),
    placeholder: t('fileManager.createFilePlaceholder'),
    initialValue: '',
    targets: [],
    open: true,
  })
}
const openRename = (row: FileEntryInfo) => {
  Object.assign(prompt, {
    mode: 'rename',
    title: t('fileManager.renameTitle'),
    label: '',
    description: t('fileManager.renameTarget', { name: row.name }),
    placeholder: t('fileManager.renamePlaceholder'),
    initialValue: row.name,
    targets: [row],
    open: true,
  })
}
const openCompress = (rows: FileEntryInfo[]) => {
  const first = rows[0]
  if (!first) {
    fb.warning(t('fileManager.selectCompressTarget'))
    return
  }
  const description =
    rows.length === 1
      ? t('fileManager.compressOne', { name: first.name })
      : t('fileManager.compressMany', { count: rows.length })
  Object.assign(prompt, {
    mode: 'compress',
    title: t('fileManager.compressTitle'),
    label: '',
    description,
    placeholder: t('fileManager.compressPlaceholder'),
    initialValue: `${first.name}.zip`,
    targets: rows,
    open: true,
  })
}
const openUnzip = (row: FileEntryInfo) => {
  Object.assign(prompt, {
    mode: 'unzip',
    title: t('fileManager.unzipTitle'),
    label: '',
    description: t('fileManager.unzipTo', { name: row.name }),
    placeholder: t('fileManager.unzipPlaceholder'),
    initialValue: row.name.replace(/\.[^.]+$/, ''),
    targets: [row],
    open: true,
  })
}

const onPromptConfirm = async (value: string) => {
  prompt.confirming = true
  try {
    if (prompt.mode === 'createFolder') {
      await client.create({ path: joinPath(currentPath.value, value), isDir: true })
      fb.success(t('fileManager.folderCreateSuccess'))
    } else if (prompt.mode === 'createFile') {
      const newPath = joinPath(currentPath.value, value)
      await client.create({ path: newPath, isDir: false, content: '' })
      fb.success(t('fileManager.fileCreateSuccess'))
      // 新建文件后直接进入编辑器（创建即编辑，符合直觉）
      editor.path = newPath
      editor.open = true
    } else if (prompt.mode === 'rename') {
      const target = prompt.targets[0]
      if (target) await client.rename(target.path, value)
      fb.success(t('fileManager.renameSuccess'))
    } else if (prompt.mode === 'compress') {
      const name = /\.(zip|tar|gz|tgz)$/i.test(value) ? value : `${value}.zip`
      const output = await client.compress(
        prompt.targets.map((item) => item.path),
        joinPath(currentPath.value, name),
      )
      fb.success(
        t('fileManager.compressDone', { path: t('fileManager.outputPath', { path: output }) }),
      )
    } else if (prompt.mode === 'unzip') {
      const target = prompt.targets[0]
      if (!target) return
      const output = await client.unzip(target.path, joinPath(currentPath.value, value))
      fb.success(
        t('fileManager.unzipDone', { path: t('fileManager.outputPath', { path: output }) }),
      )
    }
    prompt.open = false
    afterMutation()
  } catch (error) {
    notifyError(error, t('fileManager.refreshFailed'))
  } finally {
    prompt.confirming = false
  }
}

// ---- 删除 ----
// 删除确认走令牌 FormDialog（内联状态），不用 EP 命令式弹窗。
const del = reactive({ open: false, message: '', pending: false, rows: [] as FileEntryInfo[] })
const deleteRows = (rows: FileEntryInfo[]) => {
  if (!rows.length) {
    fb.warning(t('fileManager.selectDeleteTarget'))
    return
  }
  del.rows = rows
  del.message =
    rows.length === 1
      ? t('fileManager.deleteOneConfirm', { name: rows[0]?.name ?? '' })
      : t('fileManager.deleteManyConfirm', { count: rows.length })
  del.open = true
}
const confirmDelete = async () => {
  del.pending = true
  try {
    await client.remove(del.rows.map((item) => item.path))
    fb.success(t('fileManager.deleteSuccess'))
    del.open = false
    afterMutation()
  } catch (error) {
    notifyError(error, t('fileManager.deleteFailed'))
  } finally {
    del.pending = false
  }
}
const deleteSelected = () => deleteRows(selectedRows.value)

// ---- 下载 / 复制链接 ----
const downloadOne = async (row: FileEntryInfo) => {
  if (row.isDir) return
  try {
    const downloadPath = await client.preSignDownload(row.path)
    downloadFileFromUrl(resolvePreSignedFileUrl(downloadPath), row.name)
    fb.success(t('fileManager.downloadStarted', { name: row.name }))
  } catch (error) {
    notifyError(error, t('fileManager.downloadFailed'))
  }
}
const downloadSelected = async () => {
  const files = selectedRows.value.filter((row) => !row.isDir)
  if (!files.length) {
    fb.warning(t('fileManager.selectDownloadFile'))
    return
  }
  for (const file of files) {
    await downloadOne(file)
  }
}
const copyLink = async (row: FileEntryInfo) => {
  if (row.isDir) {
    fb.warning(t('fileManager.selectCopyLinkFile'))
    return
  }
  try {
    const downloadPath = await client.preSignDownload(row.path)
    await copyTextToClipboard(resolvePreSignedFileUrl(downloadPath))
    fb.success(t('fileManager.copiedTemporaryLink', { name: row.name }))
  } catch (error) {
    notifyError(error, t('fileManager.copyLinkFailed'))
  }
}

// ---- 复制 / 剪切 / 粘贴 ----
const setClipboard = (mode: 'copy' | 'cut', rows: FileEntryInfo[]) => {
  if (!rows.length) {
    fb.warning(
      mode === 'copy' ? t('fileManager.selectCopyTarget') : t('fileManager.selectCutTarget'),
    )
    return
  }
  clipboard.value = { mode, paths: rows.map((item) => item.path) }
  fb.success(
    mode === 'copy'
      ? t('fileManager.copiedItems', { count: rows.length })
      : t('fileManager.cutItems', { count: rows.length }),
  )
}
const paste = async () => {
  if (!clipboard.value || !clipboard.value.paths.length) {
    fb.warning(t('fileManager.copyOrCutFirst'))
    return
  }
  const { mode, paths } = clipboard.value
  const pending = fb.add({ title: t('fileManager.pasteProcessing'), color: 'info', duration: 0 })
  try {
    if (mode === 'copy') await client.copy(paths, currentPath.value)
    else await client.move(paths, currentPath.value)
    fb.success(t('fileManager.pasteDone'))
    if (mode === 'cut') clipboard.value = null
    afterMutation()
  } catch (error) {
    notifyError(error, t('fileManager.pasteFailed'))
  } finally {
    fb.remove(pending.id)
  }
}

// ---- 分享 ----
const openShare = (rows: FileEntryInfo[]) => {
  // 选中项按当前来源打包为分享条目（分享支持跨来源，此处均来自当前浏览来源）。
  shareItems.value = rows.map((row) => ({ sourceId: currentSourceId.value, path: row.path }))
  shareOpen.value = true
}

// ---- 工具栏"更多操作"（含分隔符，走 AppDropdown 自定义面板）----
type MoreItem = { key: string; label: string; icon: string; disabled?: boolean; divider?: boolean }
const moreActions = computed<MoreItem[]>(() => {
  const caps = currentCaps.value
  const actions: MoreItem[] = [
    { key: 'createFile', label: t('fileManager.createFile'), icon: 'i-lucide-file-plus' },
  ]
  if (caps.copy) {
    actions.push({ key: 'copy', label: t('fileManager.copy'), icon: 'i-lucide-copy', disabled: !hasSelection.value })
  }
  if (caps.move) {
    actions.push({ key: 'cut', label: t('fileManager.cut'), icon: 'i-lucide-scissors', disabled: !hasSelection.value })
  }
  if (caps.copy || caps.move) {
    actions.push({ key: 'paste', label: t('fileManager.paste'), icon: 'i-lucide-clipboard-paste', disabled: !clipboard.value })
  }
  actions.push({ key: 'divider-1', label: '', icon: '', divider: true })
  if (caps.compress) {
    actions.push({ key: 'compress', label: t('fileManager.compress'), icon: 'i-lucide-archive', disabled: !hasSelection.value })
  }
  // 分享支持任意系统来源（本地/OSS/FTP/WebDAV）；实例作用域不提供分享。
  if (isSystemScope) {
    actions.push({ key: 'share', label: t('fileManager.share'), icon: 'i-lucide-share-2', disabled: !hasSelection.value })
  }
  return actions
})
const onMoreAction = (key: string) => {
  if (key === 'createFile') openCreateFile()
  else if (key === 'copy') setClipboard('copy', selectedRows.value)
  else if (key === 'cut') setClipboard('cut', selectedRows.value)
  else if (key === 'paste') paste()
  else if (key === 'compress') openCompress(selectedRows.value)
  else if (key === 'share') openShare(selectedRows.value)
}

// ---- 行"更多"（列表 ⋯ 与 网格瓦片 ⋯ 共用；下载/删除并入）----
const rowActions = (row: FileEntryInfo): ActionMenuItem[] => {
  const caps = currentCaps.value
  const actions: ActionMenuItem[] = [
    { key: 'rename', label: t('fileManager.rename'), icon: 'i-lucide-pencil' },
  ]
  if (!row.isDir) actions.push({ key: 'download', label: t('fileManager.download'), icon: 'i-lucide-download' })
  if (!row.isDir) actions.push({ key: 'copyLink', label: t('fileManager.copyLink'), icon: 'i-lucide-link' })
  if (caps.copy) actions.push({ key: 'copy', label: t('fileManager.copy'), icon: 'i-lucide-copy' })
  if (caps.move) actions.push({ key: 'cut', label: t('fileManager.cut'), icon: 'i-lucide-scissors' })
  if (caps.compress) actions.push({ key: 'compress', label: t('fileManager.compress'), icon: 'i-lucide-archive' })
  if (caps.compress && !row.isDir && /\.(zip|tar|gz|tgz|rar|7z)$/i.test(row.name)) {
    actions.push({ key: 'unzip', label: t('fileManager.unzip'), icon: 'i-lucide-folder-archive' })
  }
  // 分享支持任意系统来源（本地/OSS/FTP/WebDAV）；实例作用域不提供分享。
  if (isSystemScope) {
    actions.push({ key: 'share', label: t('fileManager.share'), icon: 'i-lucide-share-2' })
  }
  actions.push({ key: 'delete', label: t('fileManager.delete'), icon: 'i-lucide-trash-2', danger: true })
  return actions
}
const onRowAction = (key: string, row: FileEntryInfo) => {
  if (key === 'rename') openRename(row)
  else if (key === 'download') downloadOne(row)
  else if (key === 'copyLink') copyLink(row)
  else if (key === 'copy') setClipboard('copy', [row])
  else if (key === 'cut') setClipboard('cut', [row])
  else if (key === 'compress') openCompress([row])
  else if (key === 'unzip') openUnzip(row)
  else if (key === 'share') openShare([row])
  else if (key === 'delete') deleteRows([row])
}

onMounted(() => {
  loadSources()
  loadList()
})

defineExpose({ refresh, navigateTo })
</script>

<style scoped lang="scss">
/* 全出血：整页即文件管理，用 app 令牌，跟随浅/暗 + 薄荷主色（不自带主题）。
   填满高度：flex:1 + min-height:0（勿用 height:100% 靠百分比，见终端页 TerminalShell）。 */
.fb {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  overflow: hidden;
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
}

/* ===== 顶部导航栏 ===== */
.fb-nav {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 8px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  flex-shrink: 0;
}
.fb-source {
  width: 180px;
  flex-shrink: 0;
}
.fb-nav__group {
  display: flex;
  gap: 2px;
  flex-shrink: 0;
}
.fb-crumb {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 2px;
  min-width: 0;
  height: 32px;
  padding: 0 8px;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-sm);
  background: var(--el-fill-color-light);
  overflow-x: auto;
  cursor: text;
  white-space: nowrap;
}
.fb-crumb::-webkit-scrollbar {
  height: 0;
}
.fb-crumb__input {
  height: 26px;
}
.fb-crumb__seg {
  flex-shrink: 0;
  display: inline-flex;
  align-items: center;
  padding: 2px 6px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.fb-crumb__seg:hover {
  background: var(--el-fill-color);
  color: var(--el-color-primary);
}
.fb-crumb__seg :deep(svg),
.fb-crumb__seg :deep(.iconify) {
  width: 15px;
  height: 15px;
}
.fb-crumb__sep {
  flex-shrink: 0;
  width: 14px;
  height: 14px;
  color: var(--el-text-color-placeholder);
}

/* ===== 令牌图标按钮（导航/工具条）===== */
.fb-ico {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  flex-shrink: 0;
  border: 1px solid transparent;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-regular);
  cursor: pointer;
  transition: background 0.15s, color 0.15s, border-color 0.15s;
}
.fb-ico:hover:not(:disabled) {
  background: var(--el-fill-color);
  color: var(--el-text-color-primary);
}
.fb-ico:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.fb-ico.is-active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.fb-ico :deep(svg),
.fb-ico :deep(.iconify) {
  width: 17px;
  height: 17px;
}
.fb-ico--sm {
  width: 28px;
  height: 28px;
}
.fb-ico--sm :deep(svg),
.fb-ico--sm :deep(.iconify) {
  width: 16px;
  height: 16px;
}
.fb-ico.is-spin :deep(svg) {
  animation: fb-spin 0.8s linear infinite;
}
@keyframes fb-spin {
  to {
    transform: rotate(360deg);
  }
}

/* ===== 主体双栏 ===== */
.fb-body {
  flex: 1;
  display: flex;
  min-height: 0;
  /* 双栏各自滚动，禁止树变高时把整页/列表一起撑飞 */
  overflow: hidden;
}
.fb-aside {
  width: 240px;
  flex: 0 0 240px;
  min-height: 0;
  border-right: 1px solid var(--el-border-color-lighter);
  overflow: auto;
  overscroll-behavior: contain;
  background: var(--el-bg-color);
}
.fb-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
  overflow: hidden;
}

/* ===== 工具栏 ===== */
.fb-tools {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 10px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  flex-shrink: 0;
}
.fb-tools__left,
.fb-tools__right {
  display: flex;
  align-items: center;
  gap: 8px;
}
.fb-search {
  position: relative;
  display: flex;
  align-items: center;
}
.fb-search__ico {
  position: absolute;
  left: 9px;
  width: 15px;
  height: 15px;
  color: var(--el-text-color-placeholder);
  pointer-events: none;
}
.fb-search__ico :deep(svg) {
  width: 15px;
  height: 15px;
}
.fb-search__input {
  width: 200px;
  padding-left: 30px;
}

/* 视图切换分段 */
.fb-seg {
  display: inline-flex;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--app-radius-sm);
  overflow: hidden;
  flex-shrink: 0;
}
.fb-seg__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border: none;
  background: var(--el-bg-color);
  color: var(--el-text-color-placeholder);
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.fb-seg__btn + .fb-seg__btn {
  border-left: 1px solid var(--el-border-color-lighter);
}
.fb-seg__btn:hover {
  color: var(--el-text-color-primary);
  background: var(--el-fill-color-light);
}
.fb-seg__btn.is-active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.fb-seg__btn :deep(svg),
.fb-seg__btn :deep(.iconify) {
  width: 16px;
  height: 16px;
}

/* ===== 更多操作下拉面板（AppDropdown 内容）===== */
.fb-menu {
  display: flex;
  flex-direction: column;
  padding: 4px;
}
.fb-menu__item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 7px 9px;
  border: none;
  border-radius: var(--app-radius-sm);
  background: transparent;
  color: var(--el-text-color-regular);
  font-size: 0.8125rem;
  text-align: left;
  white-space: nowrap;
  cursor: pointer;
  transition: background 0.15s, color 0.15s;
}
.fb-menu__item:hover:not(:disabled) {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.fb-menu__item:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
.fb-menu__ico {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
}
.fb-menu__ico :deep(svg) {
  width: 16px;
  height: 16px;
}
.fb-menu__div {
  height: 1px;
  margin: 4px 2px;
  background: var(--el-border-color-lighter);
}

/* 筛选面板（AppDropdown 内容）*/
.fb-filter {
  padding: 8px 4px;
}
.fb-filter__item {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 8px;
  font-size: 0.8125rem;
  color: var(--el-text-color-regular);
  cursor: pointer;
}
.fb-filter__item input {
  width: 15px;
  height: 15px;
  accent-color: var(--el-color-primary);
  cursor: pointer;
}

/* ===== 选择条 ===== */
.fb-selbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  flex-wrap: wrap;
  padding: 8px 12px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: var(--el-color-primary-light-9);
}
.fb-selbar__count {
  font-size: 0.8125rem;
  font-weight: 600;
  color: var(--el-color-primary);
}
.fb-selbar__actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}

/* ===== 内容区 ===== */
.fb-content {
  flex: 1;
  min-height: 0;
  overflow: auto;
}

/* 骨架 */
.fb-skeleton {
  display: flex;
  flex-direction: column;
}
.fb-skeleton__row {
  height: 44px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  background: linear-gradient(
    100deg,
    var(--el-fill-color-light) 30%,
    var(--el-fill-color) 50%,
    var(--el-fill-color-light) 70%
  );
  background-size: 200% 100%;
  animation: fb-shimmer 1.4s ease-in-out infinite;
}
@keyframes fb-shimmer {
  from {
    background-position: 200% 0;
  }
  to {
    background-position: -200% 0;
  }
}

/* ===== 列表视图 ===== */
.fb-thead,
.fb-row {
  display: grid;
  grid-template-columns: 42px minmax(200px, 2fr) 120px 100px 168px 88px;
  align-items: center;
}
.fb-thead {
  position: sticky;
  top: 0;
  z-index: 1;
  height: 40px;
  background: var(--el-fill-color-lighter);
  border-bottom: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-secondary);
  font-size: 0.75rem;
  font-weight: 600;
}
.fb-col {
  padding: 0 10px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fb-thead .fb-col {
  display: flex;
  align-items: center;
  gap: 4px;
  height: 100%;
}
.fb-col--check {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}
.fb-col--ops {
  display: flex;
  align-items: center;
  gap: 2px;
  justify-content: flex-end;
}
.fb-sort {
  border: none;
  background: transparent;
  font: inherit;
  color: inherit;
  cursor: pointer;
}
.fb-sort:hover {
  color: var(--el-color-primary);
}
.fb-sort__ico {
  width: 13px;
  height: 13px;
  color: var(--el-color-primary);
}
.fb-sort__ico :deep(svg) {
  width: 13px;
  height: 13px;
}
.fb-row {
  height: 44px;
  border-bottom: 1px solid var(--el-border-color-lighter);
  color: var(--el-text-color-primary);
  font-size: 0.8125rem;
  cursor: pointer;
  transition: background 0.12s;
}
.fb-row:hover {
  background: var(--el-fill-color-light);
}
.fb-row.is-selected {
  background: var(--el-color-primary-light-9);
}
.fb-row__ico {
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  margin-right: 8px;
}
.fb-row__ico :deep(svg) {
  width: 18px;
  height: 18px;
}
/* 图标只继承文本色（01 §7）；文件夹不用琥珀色/彩色，选中行由行背景表达。 */
.fb-row__ico.is-folder {
  color: var(--el-text-color-regular);
}
.fb-row__ico.is-file {
  color: var(--el-text-color-placeholder);
}
.fb-col--name {
  display: flex;
  align-items: center;
  min-width: 0;
}
.fb-row__name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fb-col--type,
.fb-col--size,
.fb-col--time {
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}

/* ===== 网格视图 ===== */
.fb-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(120px, 1fr));
  gap: 8px;
  padding: 12px;
}
.fb-tile {
  position: relative;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 14px 8px 10px;
  border: 1px solid transparent;
  border-radius: var(--app-radius);
  cursor: pointer;
  transition: background 0.12s, border-color 0.12s;
}
.fb-tile:hover {
  background: var(--el-fill-color-light);
}
.fb-tile.is-selected {
  background: var(--el-color-primary-light-9);
  border-color: var(--el-color-primary);
}
.fb-tile__check {
  position: absolute;
  top: 6px;
  left: 6px;
  width: 15px;
  height: 15px;
  accent-color: var(--el-color-primary);
  opacity: 0;
  transition: opacity 0.12s;
}
.fb-tile:hover .fb-tile__check,
.fb-tile.is-selected .fb-tile__check {
  opacity: 1;
}
.fb-tile__more {
  position: absolute;
  top: 4px;
  right: 2px;
  opacity: 0;
  transition: opacity 0.12s;
}
.fb-tile:hover .fb-tile__more {
  opacity: 1;
}
.fb-tile__ico {
  width: 40px;
  height: 40px;
}
.fb-tile__ico :deep(svg) {
  width: 40px;
  height: 40px;
}
.fb-tile__ico.is-folder {
  color: var(--el-text-color-regular);
}
.fb-tile__ico.is-file {
  color: var(--el-text-color-placeholder);
}
.fb-tile__name {
  max-width: 100%;
  margin-top: 2px;
  font-size: 0.75rem;
  color: var(--el-text-color-primary);
  text-align: center;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fb-tile__meta {
  font-size: 0.6875rem;
  color: var(--el-text-color-placeholder);
  font-variant-numeric: tabular-nums;
}

/* 复选框令牌化 */
.fb-content input[type='checkbox'] {
  width: 15px;
  height: 15px;
  accent-color: var(--el-color-primary);
  cursor: pointer;
}

/* ===== 底部 ===== */
.fb-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 16px;
  flex-wrap: wrap;
  padding: 8px 12px;
  border-top: 1px solid var(--el-border-color-lighter);
  background: var(--el-bg-color);
  flex-shrink: 0;
}
.fb-foot__summary {
  font-size: 0.8125rem;
  color: var(--el-text-color-secondary);
  font-variant-numeric: tabular-nums;
}

/* 删除确认文案 */
.fb-confirm {
  margin: 0;
  font-size: 0.875rem;
  line-height: 1.6;
  color: var(--el-text-color-regular);
  word-break: break-word;
}

/* ===== 移动端：隐藏树，工具栏换行 ===== */
@media (width <= 768px) {
  .fb-nav {
    gap: 6px;
    padding: 8px;
  }
  .fb-source {
    width: 130px;
  }
  .fb-search__input {
    width: 150px;
  }
  .fb-tools {
    padding: 8px;
  }
  .fb-thead,
  .fb-row {
    grid-template-columns: 36px minmax(140px, 2fr) 92px 64px;
  }
  /* 提高特异性覆盖 .fb-thead .fb-col{display:flex}，否则表头 类型/时间 列不会被隐藏 */
  .fb-thead .fb-col--type,
  .fb-thead .fb-col--time,
  .fb-row .fb-col--type,
  .fb-row .fb-col--time {
    display: none;
  }
}
</style>
