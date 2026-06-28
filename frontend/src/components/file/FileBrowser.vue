<template>
  <div class="file-module file-browser" :class="{ 'is-dark': isDark }">
    <!-- 顶部导航：来源切换 + 返回/前进/上级 + 面包屑地址 -->
    <div class="fb-navbar">
      <div v-if="isSystemScope" class="fb-source">
        <select
          v-model="currentSourceId"
          class="fm-input fb-source-select"
          :title="t('fileSource.switchSource')"
          @change="onSourceChange"
        >
          <option value="">{{ t('fileSource.localDisk') }}</option>
          <option v-for="s in sources" :key="s.id" :value="s.id">
            {{ s.name }}（{{ s.type.toUpperCase() }}）
          </option>
        </select>
      </div>
      <div class="fb-nav-actions">
        <button
          type="button"
          class="fm-icon-btn"
          :disabled="!canGoBack"
          :title="t('fileManager.back')"
          @click="goBack"
        >
          <el-icon><IconBack /></el-icon>
        </button>
        <button
          type="button"
          class="fm-icon-btn"
          :disabled="!canGoForward"
          :title="t('fileManager.forward')"
          @click="goForward"
        >
          <el-icon><IconForward /></el-icon>
        </button>
        <button
          type="button"
          class="fm-icon-btn"
          :disabled="!canGoUp"
          :title="t('fileManager.up')"
          @click="goUp"
        >
          <el-icon><IconUp /></el-icon>
        </button>
      </div>

      <div class="fb-breadcrumb" @click="startEditPath">
        <input
          v-if="editingPath"
          ref="pathInputRef"
          v-model="pathDraft"
          class="fm-input fb-path-input"
          @keyup.enter="commitPath"
          @keyup.esc="editingPath = false"
          @blur="editingPath = false"
          @click.stop
        />
        <template v-else>
          <button
            type="button"
            class="fb-crumb fb-crumb-root"
            :title="t('fileManager.rootDir')"
            @click.stop="navigateTo('')"
          >
            <el-icon><IconHome /></el-icon>
          </button>
          <template v-for="segment in pathSegments" :key="segment.path">
            <el-icon class="fb-crumb-sep"><IconChevronRight /></el-icon>
            <button type="button" class="fb-crumb" @click.stop="navigateTo(segment.path)">
              {{ segment.name }}
            </button>
          </template>
        </template>
      </div>

      <button
        type="button"
        class="fm-icon-btn"
        :title="t('fileManager.refresh')"
        :class="{ 'is-spinning': loading }"
        @click="refresh"
      >
        <el-icon><IconRefresh /></el-icon>
      </button>
    </div>

    <!-- 工具栏 -->
    <div class="fb-toolbar">
      <div class="fb-toolbar-left">
        <button type="button" class="fm-btn fm-btn--primary" @click="uploadOpen = true">
          <el-icon><IconUpload /></el-icon>{{ t('fileManager.uploadTitle') }}
        </button>
        <button type="button" class="fm-btn" @click="openCreateFolder">
          <el-icon><IconNewFolder /></el-icon>{{ t('fileManager.createFolder') }}
        </button>
        <button
          type="button"
          class="fm-btn"
          :disabled="!hasSelection"
          @click="downloadSelected"
        >
          <el-icon><IconDownload /></el-icon>{{ t('fileManager.download') }}
        </button>
        <button
          type="button"
          class="fm-btn fm-btn--danger"
          :disabled="!hasSelection"
          @click="deleteSelected"
        >
          <el-icon><IconDelete /></el-icon>{{ t('fileManager.delete') }}
        </button>
        <FileMenu :items="moreActions" @select="onMoreAction">
          <button type="button" class="fm-btn">
            {{ t('fileManager.moreActions') }}<el-icon><IconChevronDown /></el-icon>
          </button>
        </FileMenu>
      </div>

      <div class="fb-toolbar-right">
        <div class="fb-search">
          <el-icon class="fb-search-icon"><IconSearch /></el-icon>
          <input
            v-model="keywords"
            class="fm-input fb-search-input"
            :placeholder="t('fileManager.searchPlaceholder')"
          />
        </div>
        <div ref="filterRef" class="fb-filter">
          <button
            type="button"
            class="fm-icon-btn"
            :class="{ 'is-active': includeSubDir }"
            :title="t('fileManager.filter')"
            @click="filterOpen = !filterOpen"
          >
            <el-icon><IconFilter /></el-icon>
          </button>
          <transition name="fp-fade">
            <div v-if="filterOpen" class="fb-filter-panel">
              <label class="fb-filter-item">
                <input v-model="includeSubDir" type="checkbox" @change="onFilterChange" />
                <span>{{ t('fileManager.subdirectories') }}</span>
              </label>
            </div>
          </transition>
        </div>
      </div>
    </div>

    <!-- 表格 -->
    <div class="fb-table">
      <div class="fb-thead">
        <div class="fb-col fb-col-check">
          <input
            type="checkbox"
            :checked="allSelected"
            :indeterminate.prop="someSelected"
            @change="toggleSelectAll"
          />
        </div>
        <button type="button" class="fb-col fb-col-name fb-sortable" @click="sortBy('name')">
          {{ t('fileManager.tableName') }}
          <el-icon v-if="sortField === FileSortField.FILE_SORT_FIELD_NAME" class="fb-sort-icon">
            <component :is="isDesc ? IconChevronDown : IconChevronUp" />
          </el-icon>
        </button>
        <div class="fb-col fb-col-type">{{ t('fileManager.type') }}</div>
        <div class="fb-col fb-col-size">{{ t('fileManager.size') }}</div>
        <div class="fb-col fb-col-path">{{ t('fileManager.path') }}</div>
        <button type="button" class="fb-col fb-col-time fb-sortable" @click="sortBy('time')">
          {{ t('fileManager.updatedAt') }}
          <el-icon
            v-if="sortField === FileSortField.FILE_SORT_FIELD_UPDATE_TIME"
            class="fb-sort-icon"
          >
            <component :is="isDesc ? IconChevronDown : IconChevronUp" />
          </el-icon>
        </button>
        <div class="fb-col fb-col-ops">{{ t('fileManager.operation') }}</div>
      </div>

      <div class="fb-tbody">
        <div v-if="loading" class="fb-placeholder">{{ t('fileManager.directoryLoading') }}</div>
        <div v-else-if="!items.length" class="fb-placeholder">
          {{ t('fileManager.directoryEmpty') }}
        </div>
        <template v-else>
          <div
            v-for="row in items"
            :key="row.path"
            class="fb-row"
            :class="{ 'is-selected': selectedPaths.has(row.path) }"
            @click="onRowClick(row, $event)"
          >
            <div class="fb-col fb-col-check" @click.stop>
              <input
                type="checkbox"
                :checked="selectedPaths.has(row.path)"
                @change="toggleSelect(row)"
              />
            </div>
            <div class="fb-col fb-col-name">
              <el-icon class="fb-row-icon" :class="row.isDir ? 'is-folder' : 'is-file'">
                <component :is="rowIcon(row)" />
              </el-icon>
              <span class="fb-row-name" :title="row.name">{{ row.name }}</span>
            </div>
            <div class="fb-col fb-col-type">{{ typeLabel(row) }}</div>
            <div class="fb-col fb-col-size">{{ row.isDir ? '-' : formatFileSize(row.size) }}</div>
            <div class="fb-col fb-col-path" :title="row.path">{{ displayPath(row) }}</div>
            <div class="fb-col fb-col-time">{{ formatDateTime(row.updateTime) }}</div>
            <div class="fb-col fb-col-ops" @click.stop>
              <button
                type="button"
                class="fm-icon-btn"
                :title="t('fileManager.open')"
                @click="onRowOpen(row)"
              >
                <el-icon><IconRename /></el-icon>
              </button>
              <button
                type="button"
                class="fm-icon-btn"
                :title="t('fileManager.download')"
                :disabled="row.isDir"
                @click="downloadOne(row)"
              >
                <el-icon><IconDownload /></el-icon>
              </button>
              <button
                type="button"
                class="fm-icon-btn is-danger"
                :title="t('fileManager.delete')"
                @click="deleteRows([row])"
              >
                <el-icon><IconDelete /></el-icon>
              </button>
              <FileMenu :items="rowActions(row)" @select="(key) => onRowAction(key, row)">
                <button type="button" class="fm-icon-btn" :title="t('fileManager.more')">
                  <el-icon><IconMoreVertical /></el-icon>
                </button>
              </FileMenu>
            </div>
          </div>
        </template>
      </div>
    </div>

    <!-- 底部：汇总 + 分页 -->
    <div class="fb-footer">
      <span class="fb-summary">
        {{ t('fileManager.footerSummary', { folders: folderCount, files: fileCount }) }}
      </span>
      <FilePager
        v-model:page="page"
        v-model:page-size="pageSize"
        :total="total"
        @change="loadList"
      />
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

    <FileMediaDialog
      v-model="media.open"
      :name="media.name"
      :kind="media.kind"
      :url="media.url"
    />

    <FileUploadDialog
      v-model="uploadOpen"
      :client="client"
      :target-path="currentPath"
      @uploaded="loadList"
    />

    <FileEditor
      v-model="editor.open"
      :client="client"
      :path="editor.path"
      root-path=""
      @saved="loadList"
      @renamed="loadList"
      @deleted="loadList"
    />

    <ShareFormDialog
      v-model="shareOpen"
      :path="sharePath"
      :source-id="shareSourceId"
      :size="shareSize"
    />
  </div>
</template>

<script setup lang="ts">
import { onClickOutside, useDebounceFn } from '@vueuse/core'
import { useI18n } from 'vue-i18n'
import { showRequestError } from '@/utils/request'
import { useThemeStore } from '@/stores/theme'
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
  type FilePreviewKind,
} from '@/utils/file'
import { createFileClient } from './fileClient'
import type { FileClient, FileClipboard, FileScope } from './types'
import FilePager from './FilePager.vue'
import FileMenu, { type FileMenuItem } from './FileMenu.vue'
import FilePromptDialog from './FilePromptDialog.vue'
import FileMediaDialog from './FileMediaDialog.vue'
import FileUploadDialog from './FileUploadDialog.vue'
import FileEditor from './FileEditor.vue'
import ShareFormDialog from '@/components/share/ShareFormDialog.vue'
import {
  IconBack,
  IconForward,
  IconUp,
  IconHome,
  IconRefresh,
  IconChevronRight,
  IconChevronDown,
  IconChevronUp,
  IconUpload,
  IconNewFolder,
  IconNewFile,
  IconDownload,
  IconDelete,
  IconSearch,
  IconFilter,
  IconMoreVertical,
  IconRename,
  IconFolder,
  IconFile,
  IconImage,
  IconVideo,
  IconAudio,
  IconShare,
  IconLink,
  IconCompress,
  IconUnzip,
  IconCopy,
  IconCut,
  IconPaste,
} from './icons'

const props = defineProps<{
  scope: FileScope
  initialPath?: string
}>()

const { t } = useI18n()

// 文件模块自成体系：用自己的中性令牌，但跟随 app 浅/暗切换（非继承紫色主题）。
const themeStore = useThemeStore()
const isDark = computed(() => themeStore.isDarkTheme)

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

// scope 在组件生命周期内稳定；source_id 动态读取，切换来源无需重建 client。
const client: FileClient = createFileClient(props.scope, () => currentSourceId.value)

const loadSources = async () => {
  if (!isSystemScope) return
  try {
    const { data } = await listFileSourcesRequest({ enabledOnly: true })
    sources.value = data.items ?? []
  } catch {
    sources.value = []
  }
}

const onSourceChange = () => {
  // 切换来源：清空历史/选择，回到该来源根目录。
  history.value = []
  historyIndex.value = -1
  selectedPaths.value = new Set()
  clipboard.value = null
  keywords.value = ''
  page.value = 1
  currentPath.value = ''
  loadList()
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
const sharePath = ref('')
const shareSourceId = ref('')
const shareSize = ref(0)

// ---- 选择 ----
const selectedRows = computed(() => items.value.filter((item) => selectedPaths.value.has(item.path)))
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

// ---- 汇总 ----
const folderCount = computed(() => Number(directory.value?.dirCount ?? 0))
const fileCount = computed(() => Number(directory.value?.fileCount ?? 0))

// ---- 加载 ----
const loadList = async () => {
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
  } catch (error) {
    showRequestError(error, t('fileManager.refreshFailed'))
  } finally {
    loading.value = false
  }
}

const refresh = () => loadList()

// ---- 导航 ----
const navigateTo = (path: string, pushHistory = true) => {
  if (pushHistory) {
    history.value = history.value.slice(0, historyIndex.value + 1)
    history.value.push(path)
    historyIndex.value = history.value.length - 1
  }
  currentPath.value = path
  page.value = 1
  keywords.value = ''
  loadList()
}
const goBack = () => {
  if (!canGoBack.value) return
  historyIndex.value -= 1
  currentPath.value = history.value[historyIndex.value] ?? currentPath.value
  page.value = 1
  loadList()
}
const goForward = () => {
  if (!canGoForward.value) return
  historyIndex.value += 1
  currentPath.value = history.value[historyIndex.value] ?? currentPath.value
  page.value = 1
  loadList()
}
const goUp = () => {
  if (!canGoUp.value) return
  navigateTo(getParentPath(currentPath.value))
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

// ---- 搜索/筛选 ----
const debouncedReload = useDebounceFn(() => {
  page.value = 1
  loadList()
}, 350)
watch(keywords, () => debouncedReload())

const filterRef = ref<HTMLElement | null>(null)
const filterOpen = ref(false)
onClickOutside(filterRef, () => (filterOpen.value = false))
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

// ---- 行展示 ----
const rowIcon = (row: FileEntryInfo) => {
  if (row.isDir) return IconFolder
  const kind = resolveFilePreviewKind(row.name)
  if (kind === 'image') return IconImage
  if (kind === 'video') return IconVideo
  if (kind === 'audio') return IconAudio
  return IconFile
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
const displayPath = (row: FileEntryInfo) => getParentPath(row.path)

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
    openMedia(row)
    return
  }
  editor.path = row.path
  editor.open = true
}

// ---- 媒体预览 ----
const openMedia = async (row: FileEntryInfo) => {
  try {
    const downloadPath = await client.preSignDownload(row.path, true)
    media.url = resolvePreSignedFileUrl(downloadPath)
    media.name = row.name
    media.kind = (resolveFilePreviewKind(row.name) || 'image') as FilePreviewKind
    media.open = true
  } catch (error) {
    showRequestError(error, t('fileManager.openFailed'))
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
    ElMessage.warning(t('fileManager.selectCompressTarget'))
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
      ElMessage.success(t('fileManager.folderCreateSuccess'))
    } else if (prompt.mode === 'createFile') {
      const newPath = joinPath(currentPath.value, value)
      await client.create({ path: newPath, isDir: false, content: '' })
      ElMessage.success(t('fileManager.fileCreateSuccess'))
      // 新建文件后直接进入编辑器（创建即编辑，符合直觉）
      editor.path = newPath
      editor.open = true
    } else if (prompt.mode === 'rename') {
      const target = prompt.targets[0]
      if (target) await client.rename(target.path, value)
      ElMessage.success(t('fileManager.renameSuccess'))
    } else if (prompt.mode === 'compress') {
      const name = /\.(zip|tar|gz|tgz)$/i.test(value) ? value : `${value}.zip`
      const output = await client.compress(
        prompt.targets.map((item) => item.path),
        joinPath(currentPath.value, name),
      )
      ElMessage.success(t('fileManager.compressDone', { path: t('fileManager.outputPath', { path: output }) }))
    } else if (prompt.mode === 'unzip') {
      const target = prompt.targets[0]
      if (!target) return
      const output = await client.unzip(target.path, joinPath(currentPath.value, value))
      ElMessage.success(t('fileManager.unzipDone', { path: t('fileManager.outputPath', { path: output }) }))
    }
    prompt.open = false
    loadList()
  } catch (error) {
    showRequestError(error, t('fileManager.refreshFailed'))
  } finally {
    prompt.confirming = false
  }
}

// ---- 删除 ----
const deleteRows = async (rows: FileEntryInfo[]) => {
  if (!rows.length) {
    ElMessage.warning(t('fileManager.selectDeleteTarget'))
    return
  }
  const message =
    rows.length === 1
      ? t('fileManager.deleteOneConfirm', { name: rows[0]?.name ?? '' })
      : t('fileManager.deleteManyConfirm', { count: rows.length })
  try {
    await ElMessageBox.confirm(message, t('fileManager.confirmDelete'), {
      type: 'warning',
      confirmButtonText: t('system.common.confirm'),
      cancelButtonText: t('system.common.cancel'),
    })
  } catch {
    return
  }
  try {
    await client.remove(rows.map((item) => item.path))
    ElMessage.success(t('fileManager.deleteSuccess'))
    loadList()
  } catch (error) {
    showRequestError(error, t('fileManager.deleteFailed'))
  }
}
const deleteSelected = () => deleteRows(selectedRows.value)

// ---- 下载 / 复制链接 ----
const downloadOne = async (row: FileEntryInfo) => {
  if (row.isDir) return
  try {
    const downloadPath = await client.preSignDownload(row.path)
    downloadFileFromUrl(resolvePreSignedFileUrl(downloadPath), row.name)
    ElMessage.success(t('fileManager.downloadStarted', { name: row.name }))
  } catch (error) {
    showRequestError(error, t('fileManager.downloadFailed'))
  }
}
const downloadSelected = async () => {
  const files = selectedRows.value.filter((row) => !row.isDir)
  if (!files.length) {
    ElMessage.warning(t('fileManager.selectDownloadFile'))
    return
  }
  for (const file of files) {
     
    await downloadOne(file)
  }
}
const copyLink = async (row: FileEntryInfo) => {
  if (row.isDir) {
    ElMessage.warning(t('fileManager.selectCopyLinkFile'))
    return
  }
  try {
    const downloadPath = await client.preSignDownload(row.path)
    await copyTextToClipboard(resolvePreSignedFileUrl(downloadPath))
    ElMessage.success(t('fileManager.copiedTemporaryLink', { name: row.name }))
  } catch (error) {
    showRequestError(error, t('fileManager.copyLinkFailed'))
  }
}

// ---- 复制 / 剪切 / 粘贴 ----
const setClipboard = (mode: 'copy' | 'cut', rows: FileEntryInfo[]) => {
  if (!rows.length) {
    ElMessage.warning(mode === 'copy' ? t('fileManager.selectCopyTarget') : t('fileManager.selectCutTarget'))
    return
  }
  clipboard.value = { mode, paths: rows.map((item) => item.path) }
  ElMessage.success(
    mode === 'copy'
      ? t('fileManager.copiedItems', { count: rows.length })
      : t('fileManager.cutItems', { count: rows.length }),
  )
}
const paste = async () => {
  if (!clipboard.value || !clipboard.value.paths.length) {
    ElMessage.warning(t('fileManager.copyOrCutFirst'))
    return
  }
  const { mode, paths } = clipboard.value
  const loadingInstance = ElMessage({
    message: t('fileManager.pasteProcessing'),
    type: 'info',
    duration: 0,
  })
  try {
    if (mode === 'copy') await client.copy(paths, currentPath.value)
    else await client.move(paths, currentPath.value)
    ElMessage.success(t('fileManager.pasteDone'))
    if (mode === 'cut') clipboard.value = null
    loadList()
  } catch (error) {
    showRequestError(error, t('fileManager.pasteFailed'))
  } finally {
    loadingInstance.close()
  }
}

// ---- 分享 ----
const openShare = (row: FileEntryInfo) => {
  sharePath.value = row.path
  shareSourceId.value = currentSourceId.value
  shareSize.value = Number(row.size) || 0
  shareOpen.value = true
}

// ---- 工具栏“更多操作” ----
const moreActions = computed<FileMenuItem[]>(() => {
  const caps = currentCaps.value
  const actions: FileMenuItem[] = [
    { key: 'createFile', label: t('fileManager.createFile'), icon: IconNewFile },
  ]
  if (caps.copy) {
    actions.push({ key: 'copy', label: t('fileManager.copy'), icon: IconCopy, disabled: !hasSelection.value })
  }
  if (caps.move) {
    actions.push({ key: 'cut', label: t('fileManager.cut'), icon: IconCut, disabled: !hasSelection.value })
  }
  if (caps.copy || caps.move) {
    actions.push({ key: 'paste', label: t('fileManager.paste'), icon: IconPaste, disabled: !clipboard.value })
  }
  actions.push({ key: 'divider-1', label: '', divider: true })
  if (caps.compress) {
    actions.push({ key: 'compress', label: t('fileManager.compress'), icon: IconCompress, disabled: !hasSelection.value })
  }
  // 分享支持任意系统来源（本地/OSS/FTP/WebDAV）；实例作用域不提供分享。
  if (isSystemScope) {
    actions.push({ key: 'share', label: t('fileManager.share'), icon: IconShare, disabled: selectedRows.value.length !== 1 })
  }
  return actions
})
const onMoreAction = (key: string) => {
  if (key === 'createFile') openCreateFile()
  else if (key === 'copy') setClipboard('copy', selectedRows.value)
  else if (key === 'cut') setClipboard('cut', selectedRows.value)
  else if (key === 'paste') paste()
  else if (key === 'compress') openCompress(selectedRows.value)
  else if (key === 'share') {
    if (selectedRows.value[0]) openShare(selectedRows.value[0])
  }
}

// ---- 行“更多” ----
const rowActions = (row: FileEntryInfo): FileMenuItem[] => {
  const caps = currentCaps.value
  const actions: FileMenuItem[] = [{ key: 'rename', label: t('fileManager.rename'), icon: IconRename }]
  if (caps.copy) actions.push({ key: 'copy', label: t('fileManager.copy'), icon: IconCopy })
  if (caps.move) actions.push({ key: 'cut', label: t('fileManager.cut'), icon: IconCut })
  if (caps.compress) actions.push({ key: 'compress', label: t('fileManager.compress'), icon: IconCompress })
  if (caps.compress && !row.isDir && /\.(zip|tar|gz|tgz|rar|7z)$/i.test(row.name)) {
    actions.push({ key: 'unzip', label: t('fileManager.unzip'), icon: IconUnzip })
  }
  actions.push({ key: 'divider', label: '', divider: true })
  // 分享支持任意系统来源（本地/OSS/FTP/WebDAV）；实例作用域不提供分享。
  if (isSystemScope) {
    actions.push({ key: 'share', label: t('fileManager.share'), icon: IconShare })
  }
  if (!row.isDir) {
    actions.push({ key: 'copyLink', label: t('fileManager.copyLink'), icon: IconLink })
  }
  return actions
}
const onRowAction = (key: string, row: FileEntryInfo) => {
  if (key === 'rename') openRename(row)
  else if (key === 'copy') setClipboard('copy', [row])
  else if (key === 'cut') setClipboard('cut', [row])
  else if (key === 'compress') openCompress([row])
  else if (key === 'unzip') openUnzip(row)
  else if (key === 'share') openShare(row)
  else if (key === 'copyLink') copyLink(row)
}

onMounted(() => {
  loadSources()
  loadList()
})

defineExpose({ refresh, navigateTo })
</script>

<style scoped>
.file-browser {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  background: var(--fm-bg);
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius);
  overflow: hidden;
}

/* 导航栏 */
.fb-navbar {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  padding: 0.625rem 0.875rem;
  border-bottom: 1px solid var(--fm-border);
}
.fb-nav-actions {
  display: flex;
  gap: 0.125rem;
}
.fb-source {
  flex-shrink: 0;
}
.fb-source-select {
  height: 32px;
  max-width: 200px;
  cursor: pointer;
}
.fb-breadcrumb {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 0.125rem;
  min-width: 0;
  height: 32px;
  padding: 0 0.5rem;
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  background: var(--fm-subtle);
  overflow-x: auto;
  cursor: text;
  white-space: nowrap;
}
.fb-breadcrumb::-webkit-scrollbar {
  height: 0;
}
.fb-path-input {
  height: 26px;
  background: var(--fm-surface);
}
.fb-crumb {
  flex-shrink: 0;
  padding: 0.125rem 0.375rem;
  border: none;
  border-radius: var(--fm-radius-sm);
  background: transparent;
  color: var(--fm-text-2);
  font-size: 13px;
  cursor: pointer;
  transition:
    background 0.15s,
    color 0.15s;
}
.fb-crumb:hover {
  background: var(--fm-hover);
  color: var(--fm-accent);
}
.fb-crumb-sep {
  flex-shrink: 0;
  font-size: 13px;
  color: var(--fm-text-3);
}
.fb-crumb-root {
  display: inline-flex;
  align-items: center;
}
.fb-crumb-root .el-icon {
  font-size: 15px;
}

.is-spinning {
  animation: fb-spin 0.8s linear infinite;
}
@keyframes fb-spin {
  from {
    transform: perspective(120px) rotateY(0deg);
  }
  to {
    transform: perspective(120px) rotateY(360deg);
  }
}

/* 工具栏 */
.fb-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
  padding: 0.75rem 0.875rem;
  border-bottom: 1px solid var(--fm-border);
}
.fb-toolbar-left,
.fb-toolbar-right {
  display: flex;
  align-items: center;
  gap: 0.5rem;
}
.fb-search {
  position: relative;
  display: flex;
  align-items: center;
}
.fb-search-icon {
  position: absolute;
  left: 0.5rem;
  font-size: 15px;
  color: var(--fm-text-3);
  pointer-events: none;
}
.fb-search-input {
  width: 220px;
  padding-left: 1.875rem;
  height: 32px;
}
.fb-filter {
  position: relative;
}
.fm-icon-btn.is-active {
  color: var(--fm-accent);
  background: var(--fm-accent-soft);
}
.fb-filter-panel {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  z-index: 30;
  padding: 0.625rem 0.75rem;
  background: var(--fm-surface);
  border: 1px solid var(--fm-border);
  border-radius: var(--fm-radius-sm);
  box-shadow: var(--fm-shadow);
  white-space: nowrap;
}
.fb-filter-item {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  font-size: 13px;
  color: var(--fm-text-2);
  cursor: pointer;
}

/* 表格 */
.fb-table {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: auto;
}
.fb-thead,
.fb-row {
  display: grid;
  grid-template-columns: 44px minmax(220px, 2fr) 130px 110px minmax(160px, 1.4fr) 180px 148px;
  align-items: center;
}
.fb-thead {
  position: sticky;
  top: 0;
  z-index: 1;
  background: var(--fm-header);
  border-bottom: 1px solid var(--fm-border);
  color: var(--fm-text-2);
  font-size: 12.5px;
  font-weight: 600;
}
.fb-col {
  padding: 0 0.625rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fb-thead .fb-col {
  height: 40px;
  display: flex;
  align-items: center;
  gap: 0.25rem;
}
.fb-col-check {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0;
}
.fb-col-ops {
  display: flex;
  align-items: center;
  gap: 0.125rem;
  justify-content: flex-end;
}
.fb-sortable {
  border: none;
  background: transparent;
  font: inherit;
  color: inherit;
  cursor: pointer;
}
.fb-sortable:hover {
  color: var(--fm-accent);
}
.fb-sort-icon {
  font-size: 13px;
  color: var(--fm-accent);
}

.fb-tbody {
  flex: 1;
}
.fb-row {
  height: 44px;
  border-bottom: 1px solid var(--fm-border);
  color: var(--fm-text);
  font-size: 13px;
  cursor: pointer;
  transition: background 0.12s;
}
.fb-row:hover {
  background: var(--fm-hover);
}
.fb-row.is-selected {
  background: var(--fm-active);
}
.fb-row-icon {
  flex-shrink: 0;
  font-size: 18px;
  margin-right: 0.5rem;
}
.fb-row-icon.is-folder {
  color: var(--fm-folder);
}
.fb-row-icon.is-file {
  color: var(--fm-text-3);
}
.fb-col-name {
  display: flex;
  align-items: center;
  min-width: 0;
}
.fb-row-name {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fb-col-type,
.fb-col-size,
.fb-col-path,
.fb-col-time {
  color: var(--fm-text-2);
}
.fb-placeholder {
  padding: 3rem 1rem;
  text-align: center;
  color: var(--fm-text-3);
  font-size: 13px;
}

/* 复选框浅色化 */
.fb-table input[type='checkbox'] {
  width: 15px;
  height: 15px;
  accent-color: var(--fm-accent);
  cursor: pointer;
}

/* 底部 */
.fb-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  flex-wrap: wrap;
  padding: 0.625rem 0.875rem;
  border-top: 1px solid var(--fm-border);
  background: var(--fm-surface);
}
.fb-summary {
  font-size: 13px;
  color: var(--fm-text-3);
}
.fp-fade-enter-active,
.fp-fade-leave-active {
  transition:
    opacity 0.15s,
    transform 0.15s;
}
.fp-fade-enter-from,
.fp-fade-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
</style>
