<script lang="ts">
export interface FileManagerWorkbenchItem {
  id: string
  name: string
  path: string
  isDirectory: boolean
  extension?: string
  permission?: string
  ownerGroup?: string
  sizeLabel: string
  sizeValue: number
  updatedAtLabel: string
  updatedAtValue: number | string | Date
}
export interface FileManagerWorkbenchSearchPayload {
  path: string
  keyword: string
  includeSubdirectories: boolean
}
export interface FileManagerWorkbenchPageChangePayload {
  path: string
  page: number
  pageSize: number
}
export type FileManagerWorkbenchSortOrder = 'ascending' | 'descending' | null
export type FileManagerWorkbenchSortProp = 'name' | 'size' | 'updatedAt' | null
export type FileManagerWorkbenchSortableField = Exclude<FileManagerWorkbenchSortProp, null>
export interface FileManagerWorkbenchSortChangePayload {
  path: string
  prop: FileManagerWorkbenchSortProp
  order: FileManagerWorkbenchSortOrder
}
export interface FileManagerWorkbenchSortState {
  prop: FileManagerWorkbenchSortProp
  order: FileManagerWorkbenchSortOrder
}
export interface FileManagerWorkbenchPagination {
  page: number
  pageSize: number
  total?: number
}
export type FileManagerPaginationMode = 'client' | 'server'
export interface FileManagerWorkbenchStats {
  folders: number
  files: number
}
export interface FileManagerUploadPreSignPayload {
  path: string
  file: File
  fileName: string
  fileSize: number
  hash: string
}
export interface FileManagerUploadSession {
  uploadId: string
  uploadPartUrlPathTemplate: string
  partSize: number
  fileSize: number
  totalParts: number
  uploadedParts: Record<string, string>
  completed: boolean
  cancel: boolean
  expiredAt?: Date | string
}
export const FILE_MANAGER_ACTIONS = [
  'refresh',
  'createFolder',
  'createFile',
  'upload',
  'download',
  'copyTemporaryLink',
  'compress',
  'unzip',
  'open',
  'delete',
  'rename',
  'more',
  'copy',
  'cut',
  'paste',
] as const
export type FileManagerAction = (typeof FILE_MANAGER_ACTIONS)[number]
</script>

<template>
  <div class="file-manager-workbench">
    <section class="file-manager-nav">
      <div
        class="nav-address-bar"
        :class="{ 'is-editing': isEditingPath }"
        @click="handleAddressBarClick"
      >
        <input
          v-if="isEditingPath"
          ref="addressInput"
          v-model="editablePath"
          type="text"
          class="address-input"
          spellcheck="false"
          @click.stop
          @keydown.enter.prevent="submitAddressInput"
          @keydown.esc.prevent="cancelAddressInput"
          @blur="submitAddressInput"
        />
        <div v-else class="address-segments">
          <template v-for="(item, index) in breadcrumbItems" :key="item.path">
            <el-icon v-if="index > 0" class="address-separator" size="13">
              <component :is="ChevronRightIcon" />
            </el-icon>
            <button
              type="button"
              class="address-segment"
              :class="{ 'is-current': index === breadcrumbItems.length - 1 }"
              @click.stop="navigateToPath(item.path)"
            >
              {{ item.label }}
            </button>
          </template>
        </div>
      </div>

      <div v-if="props.searchable" class="nav-search-bar">
        <div class="nav-search-scope">
          <el-checkbox
            v-model="searchInSubdirectories"
            class="toolbar-checkbox"
            :disabled="!hasSearchKeyword"
          >
            子目录
          </el-checkbox>
        </div>
        <el-input
          v-model="searchKeyword"
          class="toolbar-search"
          clearable
          placeholder="在当前目录下查找"
          @clear="handleSearchClear"
          @keyup.enter="triggerSearch"
        >
          <template #suffix>
            <button
              type="button"
              class="search-trigger"
              :disabled="!hasSearchKeyword"
              @click="triggerSearch"
            >
              <el-icon size="14"><component :is="MagnifyingGlassIcon" /></el-icon>
            </button>
          </template>
        </el-input>
      </div>
    </section>

    <section class="file-manager-panel">
      <div class="panel-toolbar">
        <el-button class="toolbar-button" :disabled="!canGoBack || props.pasteLoading" @click="goBack">
          <span class="toolbar-button__content"
            ><el-icon class="toolbar-button__icon" size="15"><component :is="ArrowLeft" /></el-icon
            ><span>返回</span></span
          >
        </el-button>
        <el-button class="toolbar-button" :disabled="!canGoForward || props.pasteLoading" @click="goForward">
          <span class="toolbar-button__content"
            ><el-icon class="toolbar-button__icon" size="15"><component :is="ArrowRight" /></el-icon
            ><span>前进</span></span
          >
        </el-button>
        <el-button class="toolbar-button" :disabled="!canGoUp || props.pasteLoading" @click="goUp">
          <span class="toolbar-button__content"
            ><el-icon class="toolbar-button__icon" size="15"><component :is="Top" /></el-icon
            ><span>上级</span></span
          >
        </el-button>
        <el-button v-if="supportsAction('refresh')" class="toolbar-button" :disabled="props.pasteLoading" @click="handleRefresh">
          <el-icon class="toolbar-button__icon toolbar-button__icon--standalone" size="15"
            ><component :is="RefreshRight"
          /></el-icon>
          <span class="toolbar-button__content"><span>刷新</span></span>
        </el-button>

        <span v-if="toolbarHasActions" class="toolbar-divider" aria-hidden="true" />

        <el-dropdown v-if="createActionItems.length" :disabled="props.pasteLoading" @command="handleCreateCommand">
          <el-button class="toolbar-button" :disabled="props.pasteLoading">
            <span class="toolbar-button__content">
              <el-icon class="toolbar-button__icon" size="15"
                ><component :is="CirclePlus"
              /></el-icon>
              <span>新建</span>
              <el-icon class="toolbar-chevron" size="13"><component :is="ArrowDown" /></el-icon>
            </span>
          </el-button>
          <template #dropdown>
            <el-dropdown-menu>
              <el-dropdown-item
                v-for="item in createActionItems"
                :key="item.command"
                :command="item.command"
                >{{ item.label }}</el-dropdown-item
              >
            </el-dropdown-menu>
          </template>
        </el-dropdown>

        <el-button
          v-if="supportsAction('upload')"
          class="toolbar-button"
          :disabled="props.pasteLoading"
          @click="handleTransferCommand('upload')"
        >
          <span class="toolbar-button__content"
            ><el-icon class="toolbar-button__icon" size="15"><component :is="Upload" /></el-icon
            ><span>上传</span></span
          >
        </el-button>
        <template v-if="hasSelectedBatchActions && selectedEntries.length > 0">
          <span class="toolbar-divider" aria-hidden="true" />
          <el-button
            v-if="supportsAction('copy')"
            class="toolbar-button"
            :disabled="props.pasteLoading"
            @click="emit('action', 'copy', activePath, selectedEntries)"
          >
            <span class="toolbar-button__content"
              ><el-icon class="toolbar-button__icon" size="15"><component :is="DocumentDuplicateIcon" /></el-icon
              ><span>复制</span></span
            >
          </el-button>
          <el-button
            v-if="supportsAction('cut')"
            class="toolbar-button"
            :disabled="props.pasteLoading"
            @click="emit('action', 'cut', activePath, selectedEntries)"
          >
            <span class="toolbar-button__content"
              ><el-icon class="toolbar-button__icon" size="15"><component :is="ScissorsIcon" /></el-icon
              ><span>剪切</span></span
            >
          </el-button>
          <el-button
            v-if="supportsAction('compress')"
            class="toolbar-button"
            :disabled="props.pasteLoading"
            @click="emit('action', 'compress', activePath, selectedEntries)"
          >
            <span class="toolbar-button__content"
              ><el-icon class="toolbar-button__icon" size="15"><component :is="ArchiveBoxIcon" /></el-icon
              ><span>压缩</span></span
            >
          </el-button>
          <el-button v-if="supportsAction('delete')" class="toolbar-button" :disabled="props.pasteLoading" @click="handleRowMore('delete')">
            <span class="toolbar-button__content"
              ><el-icon class="toolbar-button__icon" size="15"><component :is="Delete" /></el-icon
              ><span>删除</span></span
            >
          </el-button>
        </template>
        <el-button
          v-if="supportsAction('paste') && props.hasClipboard"
          class="toolbar-button"
          :class="{ 'is-pasting': props.pasteLoading }"
          :loading="props.pasteLoading"
          :disabled="props.pasteLoading"
          @click="emit('action', 'paste', activePath, [])"
        >
          <span class="toolbar-button__content">
            <span class="paste-icon-stack" v-if="!props.pasteLoading">
              <el-icon class="toolbar-button__icon" size="15"><component :is="ClipboardDocumentIcon" /></el-icon>
            </span>
            <span>{{ props.pasteLoading ? '粘贴中...' : '粘贴' }}</span>
          </span>
        </el-button>
      </div>

      <div class="panel-note">{{ note }}</div>

      <Transition name="directory-switch" mode="out-in">
        <div
          :key="tableTransitionKey"
          :class="[
            'file-table-shell',
            { 'is-nested': isNestedPath, 'is-compact': visibleItems.length <= 4, 'is-processing': props.pasteLoading },
          ]"
        >
          <el-table
            v-loading="loading"
            :data="visibleItems"
            row-key="id"
            class="file-table"
            :empty-text="loading ? '目录加载中...' : '当前目录暂无文件'"
            @selection-change="handleSelectionChange"
            @row-click="handleRowClick"
            @sort-change="handleSortChange"
          >
            <el-table-column type="selection" width="48" />
            <el-table-column
              prop="name"
              label="名称"
              min-width="340"
              :sortable="getSortMode('name')"
            >
              <template #default="{ row }">
                <div class="file-name-cell">
                  <span class="file-kind-icon" :class="row.isDirectory ? 'is-folder' : 'is-file'">
                    <el-icon size="15"
                      ><component :is="row.isDirectory ? FolderIcon : DocumentTextIcon"
                    /></el-icon>
                  </span>
                  <div class="file-name-copy">
                    <span class="file-name">{{ row.name }}</span>
                    <span class="file-subpath">{{ row.path }}</span>
                  </div>
                </div>
              </template>
            </el-table-column>
            <el-table-column prop="extension" label="拓展名" width="85">
                <template #default="{ row }">
                  <span v-if="row.extension" class="file-extension">{{ row.extension }}</span>
                  <span v-else class="file-extension file-extension--muted">--</span>
                </template></el-table-column
              >
              <el-table-column prop="permission" label="权限" width="110" />
            <el-table-column prop="ownerGroup" label="用户 / 用户组" min-width="190" />
            <el-table-column prop="size" label="大小" width="120" :sortable="getSortMode('size')"
              ><template #default="{ row }">{{ row.sizeLabel }}</template></el-table-column
            >
            <el-table-column
              prop="updatedAt"
              label="修改时间"
              min-width="180"
              :sortable="getSortMode('updatedAt')"
              ><template #default="{ row }">{{ row.updatedAtLabel }}</template></el-table-column
            >
            <el-table-column label="操作" width="180" fixed="right">
              <template #default="{ row }">
                <div class="row-actions">
                  <button
                    v-if="canOpenEntry(row)"
                    type="button"
                    class="row-action"
                    :disabled="props.pasteLoading"
                    @click.stop="handleOpen(row)"
                  >
                    打开
                  </button>
                  <button
                    v-if="supportsAction('download') && !row.isDirectory"
                    type="button"
                    class="row-action"
                    :disabled="props.pasteLoading"
                    @click.stop="emit('action', 'download', activePath, [row])"
                  >
                    下载
                  </button>
                  <el-dropdown
                    v-if="getRowMoreActions(row).length"
                    :disabled="props.pasteLoading"
                    @command="(command) => handleRowMore(command, row)"
                  >
                    <button type="button" class="row-action" :disabled="props.pasteLoading" @click.stop>
                      更多 <el-icon size="12"><component :is="ArrowDown" /></el-icon>
                    </button>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item
                          v-for="item in getRowMoreActions(row)"
                          :key="item.command"
                          :command="item.command"
                          >{{ item.label }}</el-dropdown-item
                        >
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </template>
            </el-table-column>
          </el-table>
          <Transition name="paste-overlay-fade">
            <div v-if="props.pasteLoading" class="paste-processing-overlay">
              <div class="paste-processing-card">
                <span class="paste-processing-spinner" />
                <span class="paste-processing-text">正在粘贴，请稍候...</span>
              </div>
            </div>
          </Transition>
        </div>
      </Transition>

      <div class="panel-footer">
        <div class="footer-summary">
          共 {{ visibleStats.folders }} 个目录，{{ visibleStats.files }} 个文件
        </div>
        <TablePagination
          v-model:current-page="pagination.page"
          v-model:page-size="pagination.pageSize"
          :total="paginationTotal"
          :is-mobile="isMobile"
          :page-sizes="pageSizes"
          @change="handlePageChange"
        />
      </div>
    </section>
  </div>
</template>

<script setup lang="ts">
import {
  ArrowLeft,
  ArrowRight,
  Top,
  ArrowDown,
  CirclePlus,
  Upload,
  Delete,
  RefreshRight,
} from '@element-plus/icons-vue'
import {
  ArchiveBoxIcon,
  ChevronRightIcon,
  ClipboardDocumentIcon,
  DocumentDuplicateIcon,
  MagnifyingGlassIcon,
  FolderIcon,
  DocumentTextIcon,
  ScissorsIcon,
} from '@heroicons/vue/24/outline'
import { useWindowSize } from '@vueuse/core'
import TablePagination from '@/components/pagination/TablePagination.vue'

defineOptions({ name: 'FileManagerWorkbench' })

const props = withDefaults(
  defineProps<{
    path: string
    items: FileManagerWorkbenchItem[]
    loading?: boolean
    note?: string
    stats?: FileManagerWorkbenchStats
    pagination?: FileManagerWorkbenchPagination
    pageSizes?: number[]
    sortState?: FileManagerWorkbenchSortState
    sortableFields?: FileManagerWorkbenchSortableField[]
    hiddenActions?: FileManagerAction[]
    searchable?: boolean
    paginationMode?: FileManagerPaginationMode
    hasClipboard?: boolean
    pasteLoading?: boolean
  }>(),
  {
    loading: false,
    note: '支持搜索、排序和操作。',
    pageSizes: () => [5, 10, 20, 50, 100, 200, 500],
    sortableFields: () => ['name', 'size', 'updatedAt'],
    hiddenActions: () => [],
    searchable: true,
    paginationMode: 'server',
    hasClipboard: false,
    pasteLoading: false,
  },
)

const emit = defineEmits<{
  action: [action: FileManagerAction, path: string, entries: FileManagerWorkbenchItem[]]
  navigate: [path: string]
  search: [payload: FileManagerWorkbenchSearchPayload]
  pageChange: [payload: FileManagerWorkbenchPageChangePayload]
  sortChange: [payload: FileManagerWorkbenchSortChangePayload]
}>()

const { width } = useWindowSize()
const isMobile = computed(() => width.value <= 768)
const activePath = ref(props.path)
const selectedEntries = ref<FileManagerWorkbenchItem[]>([])
const searchKeyword = ref('')
const searchInSubdirectories = ref(false)
const isEditingPath = ref(false)
const editablePath = ref('')
const refreshKey = ref(0)
const pagination = reactive({
  page: props.pagination?.page || 1,
  pageSize: props.pagination?.pageSize || 100,
})
const sortState = reactive<FileManagerWorkbenchSortState>({
  prop: props.sortState?.prop ?? null,
  order: props.sortState?.order ?? null,
})
const pathHistory = ref<string[]>([activePath.value])
const historyCursor = ref(0)
const addressInputRef = useTemplateRef<HTMLInputElement>('addressInput')

const pageSizes = computed(() => props.pageSizes)
const hiddenActionSet = computed(() => new Set(props.hiddenActions))
const isClientPagination = computed(() => props.paginationMode === 'client')
const supportsAction = (action: FileManagerAction) => !hiddenActionSet.value.has(action)
const isSortable = (field: FileManagerWorkbenchSortableField) =>
  props.sortableFields.includes(field)
const getSortMode = (field: FileManagerWorkbenchSortableField) =>
  isSortable(field) ? 'custom' : false
const canOpenEntry = (entry: FileManagerWorkbenchItem) =>
  entry.isDirectory || supportsAction('open')
const createActionItems = computed(() =>
  [
    supportsAction('createFolder')
      ? { command: 'createFolder' as const, label: '新建文件夹' }
      : null,
    supportsAction('createFile') ? { command: 'createFile' as const, label: '新建文件' } : null,
  ].filter((item): item is { command: 'createFolder' | 'createFile'; label: string } =>
    Boolean(item),
  ),
)
const isZipFile = (row: FileManagerWorkbenchItem) =>
  !row.isDirectory && row.name.toLowerCase().endsWith('.zip')

const getRowMoreActions = (row: FileManagerWorkbenchItem) =>
  [
    supportsAction('rename')
      ? { command: 'rename' as const, label: '重命名' }
      : null,
    supportsAction('copy')
      ? { command: 'copy' as const, label: '复制' }
      : null,
    supportsAction('cut')
      ? { command: 'cut' as const, label: '剪切' }
      : null,
    !row.isDirectory && supportsAction('copyTemporaryLink')
      ? { command: 'copyTemporaryLink' as const, label: '复制链接' }
      : null,
    isZipFile(row) && supportsAction('unzip')
      ? { command: 'unzip' as const, label: '解压' }
      : null,
    supportsAction('delete') ? { command: 'delete' as const, label: '删除' } : null,
    supportsAction('more') ? { command: 'more' as const, label: '属性' } : null,
  ].filter((item): item is { command: 'rename' | 'copy' | 'cut' | 'copyTemporaryLink' | 'unzip' | 'delete' | 'more'; label: string } =>
    Boolean(item),
  )
const hasSelectedBatchActions = computed(
  () =>
    supportsAction('copy') ||
    supportsAction('cut') ||
    supportsAction('compress') ||
    supportsAction('delete'),
)
const toolbarHasActions = computed(
  () =>
    createActionItems.value.length > 0 ||
    supportsAction('upload') ||
    supportsAction('refresh') ||
    supportsAction('paste'),
)
const trimTrailingSeparators = (path: string) => path.replace(/[\\/]+$/, '')
const normalizePath = (path: string) => trimTrailingSeparators(path).replace(/\\/g, '/')
const getPathSeparator = (path: string) => (path.includes('\\') ? '\\' : '/')
const splitPath = (path: string) => {
  const normalizedPath = normalizePath(path)
  if (!normalizedPath) {
    return { root: '', segments: [] as string[] }
  }

  const windowsDriveMatch = normalizedPath.match(/^[A-Za-z]:/)
  const root = windowsDriveMatch ? windowsDriveMatch[0] : normalizedPath.startsWith('/') ? '/' : ''
  const remainingPath = root
    ? normalizedPath.slice(root.length).replace(/^\/+/, '')
    : normalizedPath

  return {
    root,
    segments: remainingPath.split('/').filter(Boolean),
  }
}
const buildPath = (root: string, segments: string[], separator: string) => {
  if (!root) {
    return segments.join(separator)
  }

  if (!segments.length) {
    return root === '/' ? '/' : `${root}${separator}`
  }

  return root === '/'
    ? `/${segments.join(separator)}`
    : `${root}${separator}${segments.join(separator)}`
}
const isSamePath = (source: string, target: string) =>
  normalizePath(source) === normalizePath(target)
const resolveParentPath = (path: string) => {
  const { root, segments } = splitPath(path)
  if (!segments.length) {
    return ''
  }

  return buildPath(root, segments.slice(0, -1), getPathSeparator(path))
}
const clearSearchState = () => {
  searchKeyword.value = ''
  searchInSubdirectories.value = false
}
const breadcrumbItems = computed(() => {
  const { root, segments } = splitPath(activePath.value)
  if (!root && !segments.length) return []
  const separator = getPathSeparator(activePath.value)
  const items: Array<{ label: string; path: string }> = []
  if (root) items.push({ label: root, path: buildPath(root, [], separator) })
  for (const [index, segment] of segments.entries()) {
    items.push({
      label: segment,
      path: buildPath(root, segments.slice(0, index + 1), separator),
    })
  }
  return items
})
const hasSearchKeyword = computed(() => Boolean(searchKeyword.value.trim()))
const paginationTotal = computed(() =>
  isClientPagination.value ? props.items.length : (props.pagination?.total ?? props.items.length),
)
const visibleItems = computed(() =>
  !isClientPagination.value
    ? props.items
    : props.items.slice(
        (pagination.page - 1) * pagination.pageSize,
        (pagination.page - 1) * pagination.pageSize + pagination.pageSize,
      ),
)
const visibleStats = computed(() => {
  if (props.stats) return props.stats
  let folders = 0
  let files = 0
  for (const entry of props.items) {
    if (entry.isDirectory) folders += 1
    else files += 1
  }
  return { folders, files }
})
const canGoBack = computed(() => historyCursor.value > 0)
const canGoForward = computed(() => historyCursor.value < pathHistory.value.length - 1)
const upPath = computed(() => (props.loading ? '' : resolveParentPath(activePath.value)))
const canGoUp = computed(() => Boolean(upPath.value))
const isNestedPath = computed(() => breadcrumbItems.value.length > 1)
const tableTransitionKey = computed(() =>
  [
    activePath.value,
    pagination.page,
    pagination.pageSize,
    paginationTotal.value,
    sortState.prop,
    sortState.order,
    refreshKey.value,
  ].join('|'),
)
const resetHistory = (path: string) => {
  pathHistory.value = [path]
  historyCursor.value = 0
}
const startAddressEditing = () => {
  editablePath.value = activePath.value
  isEditingPath.value = true
  nextTick(() => {
    addressInputRef.value?.focus()
    addressInputRef.value?.select()
  })
}
const cancelAddressInput = () => {
  editablePath.value = activePath.value
  isEditingPath.value = false
}
const submitAddressInput = () => {
  const nextPath = editablePath.value.trim()
  isEditingPath.value = false

  if (!nextPath || isSamePath(nextPath, activePath.value)) {
    editablePath.value = activePath.value
    return
  }

  navigateToPath(nextPath)
}
const resetViewState = () => {
  selectedEntries.value = []
  clearSearchState()
  pagination.page = props.pagination?.page || 1
  pagination.pageSize = props.pagination?.pageSize || 100
  sortState.prop = props.sortState?.prop ?? null
  sortState.order = props.sortState?.order ?? null
}
watch(
  () => props.path,
  (path, previousPath) => {
    activePath.value = path
    if (!isEditingPath.value) {
      editablePath.value = path
    }
    if (!previousPath) {
      resetViewState()
      resetHistory(path)
      return
    }
    if (
      !isSamePath(path, previousPath) &&
      !isSamePath(path, pathHistory.value[historyCursor.value] || '')
    ) {
      selectedEntries.value = []
      clearSearchState()
      pagination.page = 1
      resetHistory(path)
    }
  },
  { immediate: true },
)
const handleAddressBarClick = (event: MouseEvent) => {
  const target = event.target instanceof HTMLElement ? event.target : null
  if (!target || target.closest('.address-segment')) return
  if (!isEditingPath.value) {
    startAddressEditing()
  }
}
watch(
  () => [props.pagination?.page, props.pagination?.pageSize] as const,
  ([page, pageSize]) => {
    if (typeof page === 'number') pagination.page = page
    if (typeof pageSize === 'number') pagination.pageSize = pageSize
  },
  { immediate: true },
)
watch(
  () => [props.sortState?.prop, props.sortState?.order] as const,
  ([prop, order]) => {
    sortState.prop = prop ?? null
    sortState.order = order ?? null
  },
  { immediate: true },
)
watch(
  () => props.items,
  () => {
    selectedEntries.value = []
  },
)
watch(hasSearchKeyword, (value) => {
  if (!value) searchInSubdirectories.value = false
})
watch(
  () => [paginationTotal.value, pagination.pageSize] as const,
  () => {
    const lastPage = Math.max(1, Math.ceil(paginationTotal.value / pagination.pageSize))
    if (pagination.page > lastPage) pagination.page = lastPage
  },
  { immediate: true },
)
const navigateToPath = (path: string, recordHistory = true) => {
  if (!path || isSamePath(path, activePath.value)) return
  activePath.value = path
  selectedEntries.value = []
  clearSearchState()
  if (recordHistory) {
    const nextHistory = pathHistory.value.slice(0, historyCursor.value + 1)
    if (!isSamePath(nextHistory[nextHistory.length - 1] || '', path)) nextHistory.push(path)
    pathHistory.value = nextHistory
    historyCursor.value = nextHistory.length - 1
  }
  emit('navigate', path)
}
const goBack = () => {
  if (!canGoBack.value) return
  historyCursor.value -= 1
  navigateToPath(pathHistory.value[historyCursor.value] || activePath.value, false)
}
const goForward = () => {
  if (!canGoForward.value) return
  historyCursor.value += 1
  navigateToPath(pathHistory.value[historyCursor.value] || activePath.value, false)
}
const goUp = () => {
  if (!upPath.value) return
  navigateToPath(upPath.value)
}
const handleSelectionChange = (rows: FileManagerWorkbenchItem[]) => {
  selectedEntries.value = rows
}
const emitSearch = (keyword: string) =>
  emit('search', {
    path: activePath.value,
    keyword,
    includeSubdirectories: keyword ? searchInSubdirectories.value : false,
  })
const triggerSearch = () => {
  const keyword = searchKeyword.value.trim()
  if (!keyword) {
    handleSearchClear()
    return
  }
  emitSearch(keyword)
}
const handleSearchClear = () => {
  clearSearchState()
  emitSearch('')
}
const shouldIgnoreRowClick = (target: EventTarget | null) => {
  const el = target instanceof HTMLElement ? target : null
  if (!el) return false
  return Boolean(
    el.closest(
      '.row-actions, .row-action, .el-checkbox, .el-checkbox__input, .el-table-column--selection',
    ),
  )
}
const handleRowClick = (row: FileManagerWorkbenchItem, _column: unknown, event: Event) => {
  if (props.pasteLoading) return
  if (shouldIgnoreRowClick(event.target)) return
  handleOpen(row)
}
const handleOpen = (row: FileManagerWorkbenchItem) => {
  if (row.isDirectory) {
    navigateToPath(row.path)
    return
  }
  if (!supportsAction('open')) return
  emit('action', 'open', activePath.value, [row])
}
const handleCreateCommand = (command: 'createFolder' | 'createFile') => {
  if (!supportsAction(command)) return
  emit('action', command, activePath.value, selectedEntries.value)
}
const handleTransferCommand = (command: 'upload' | 'download') => {
  if (!supportsAction(command)) return
  emit('action', command, activePath.value, selectedEntries.value)
}
const handleRowMore = (
  command: 'copyTemporaryLink' | 'unzip' | 'delete' | 'rename' | 'more' | 'copy' | 'cut',
  row?: FileManagerWorkbenchItem,
) => {
  if (!supportsAction(command)) return
  emit('action', command, activePath.value, row ? [row] : selectedEntries.value)
}
const handleRefresh = () => {
  selectedEntries.value = []
  refreshKey.value += 1
  emit('action', 'refresh', activePath.value, [])
}
const handlePageChange = () => {
  selectedEntries.value = []
  emit('pageChange', {
    path: activePath.value,
    page: pagination.page,
    pageSize: pagination.pageSize,
  })
}
const handleSortChange = (payload: {
  prop: string | null
  order: FileManagerWorkbenchSortOrder
}) => {
  const nextProp =
    payload.prop === 'name' || payload.prop === 'size' || payload.prop === 'updatedAt'
      ? (payload.prop as FileManagerWorkbenchSortProp)
      : null
  if (nextProp && !isSortable(nextProp)) return
  selectedEntries.value = []
  emit('sortChange', { path: activePath.value, prop: nextProp, order: payload.order })
}
</script>
<style scoped lang="scss">
.file-manager-workbench {
  --file-manager-control-height: 2.55rem;
  --file-manager-control-radius: 0.88rem;
  --file-manager-surface-bg: color-mix(in srgb, var(--el-bg-color-page) 90%, var(--el-bg-color));
  --file-manager-surface-border: color-mix(
    in srgb,
    var(--el-border-color-extra-light) 92%,
    transparent
  );
  --file-manager-surface-hover: color-mix(
    in srgb,
    var(--el-color-primary) 7%,
    var(--file-manager-surface-bg)
  );
  --file-manager-surface-active: color-mix(
    in srgb,
    var(--el-color-primary) 12%,
    var(--file-manager-surface-bg)
  );
  display: flex;
  flex: 1;
  flex-direction: column;
  min-height: 100%;
  margin: -1rem;
  padding: 1rem;
  background: var(--el-bg-color-page);
  box-sizing: border-box;
  gap: 0.8rem;
}

@media (width <= 768px) {
  .file-manager-workbench {
    margin: -1rem;
    padding: 1rem;
  }
}

.file-manager-nav,
.file-manager-panel {
  border-radius: 1rem;
  background: var(--el-bg-color);
  box-shadow: 0 1px 2px rgb(15 23 42 / 4%);
}

.file-manager-nav {
  padding: 0.55rem 0.6rem;
  display: flex;
  align-items: center;
  gap: 0.7rem;
}

.nav-address-bar {
  display: flex;
  align-items: center;
  min-height: var(--file-manager-control-height);
  min-width: 0;
  flex: 1;
  border: 1px solid var(--file-manager-surface-border);
  border-radius: var(--file-manager-control-radius);
  background: var(--file-manager-surface-bg);
  padding: 0.28rem 0.42rem 0.28rem 0.48rem;
  cursor: text;
}

.nav-address-bar.is-editing {
  padding: 0.18rem 0.3rem;
}

.address-segments {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  min-width: 0;
  flex: 1;
  overflow-x: auto;
  scrollbar-width: none;
}

.address-segments::-webkit-scrollbar {
  display: none;
}

.address-input {
  width: 100%;
  min-width: 0;
  border: none;
  outline: none;
  background: transparent;
  color: var(--el-text-color-primary);
  font-size: 0.86rem;
  font-weight: 500;
  line-height: 1.3;
}

.address-input::selection {
  background: color-mix(in srgb, var(--el-color-primary) 24%, transparent);
}

.nav-search-bar {
  display: flex;
  align-items: center;
  min-height: var(--file-manager-control-height);
  gap: 0.28rem;
  width: auto;
  flex-shrink: 0;
  border: 1px solid var(--file-manager-surface-border);
  border-radius: var(--file-manager-control-radius);
  background: var(--file-manager-surface-bg);
  padding: 0.24rem 0.3rem;
}

.nav-search-scope {
  display: inline-flex;
  align-items: center;
  min-height: calc(var(--file-manager-control-height) - 0.48rem);
  border-radius: calc(var(--file-manager-control-radius) - 0.18rem);
  background: color-mix(in srgb, var(--el-bg-color) 92%, var(--file-manager-surface-bg));
  padding: 0 0.72rem;
  flex-shrink: 0;
}

.nav-home-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: calc(var(--file-manager-control-height) - 0.72rem);
  height: calc(var(--file-manager-control-height) - 0.72rem);
  border-radius: calc(var(--file-manager-control-radius) - 0.2rem);
  background: color-mix(in srgb, var(--el-color-primary) 12%, var(--el-bg-color));
  color: color-mix(in srgb, var(--el-color-primary) 82%, var(--el-text-color-primary));
  flex-shrink: 0;
}

.address-segment {
  display: inline-flex;
  align-items: center;
  border: none;
  background: transparent;
  min-height: calc(var(--file-manager-control-height) - 0.76rem);
  padding: 0 0.68rem;
  border-radius: calc(var(--file-manager-control-radius) - 0.22rem);
  color: var(--el-text-color-primary);
  font-size: 0.82rem;
  font-weight: 500;
  line-height: 1;
  white-space: nowrap;
  cursor: pointer;
  transition:
    background-color 0.18s ease,
    color 0.18s ease;
}

.address-segment:hover {
  background: var(--file-manager-surface-hover);
  color: var(--el-color-primary);
}

.address-segment.is-current {
  background: var(--file-manager-surface-active);
  color: color-mix(in srgb, var(--el-color-primary) 80%, var(--el-text-color-primary));
  font-weight: 600;
}

.address-separator {
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}

.file-manager-panel {
  padding: 0.8rem 0.9rem;
}

.panel-toolbar {
  display: flex;
  align-items: center;
  gap: 0.18rem;
  flex-wrap: wrap;
  padding-block: 0.08rem 0.18rem;
}

.toolbar-divider {
  width: 1px;
  height: 1.5rem;
  margin: 0 0.26rem;
  background: var(--file-manager-surface-border);
  flex-shrink: 0;
}

.toolbar-button {
  min-width: auto;
  height: calc(var(--file-manager-control-height) - 0.52rem);
  padding: 0 0.76rem;
  border-radius: calc(var(--file-manager-control-radius) - 0.22rem);
  --el-button-bg-color: transparent;
  --el-button-border-color: transparent;
  --el-button-text-color: var(--el-text-color-secondary);
  --el-button-hover-bg-color: var(--file-manager-surface-hover);
  --el-button-hover-border-color: transparent;
  --el-button-hover-text-color: var(--el-text-color-primary);
  --el-button-active-bg-color: var(--file-manager-surface-active);
  --el-button-active-border-color: transparent;
  --el-button-active-text-color: var(--el-text-color-primary);
  --el-button-disabled-bg-color: transparent;
  --el-button-disabled-border-color: transparent;
  --el-button-disabled-text-color: #a0adbf;
  font-size: 0.79rem;
  font-weight: 500;
  box-shadow: none;
}

.toolbar-button__content {
  display: inline-flex;
  align-items: center;
  gap: 0.38rem;
  line-height: 1;
}

.toolbar-button__icon {
  color: color-mix(in srgb, var(--el-color-primary) 82%, var(--el-text-color-secondary));
}

.toolbar-button__icon--standalone {
  margin-right: 0.38rem;
}

.panel-toolbar :deep(.el-dropdown) {
  display: flex;
}

.panel-toolbar :deep(.el-dropdown > .el-tooltip__trigger) {
  display: flex;
}

.toolbar-chevron {
  margin-left: 0.06rem;
  color: var(--el-text-color-placeholder);
}

.toolbar-checkbox {
  display: inline-flex;
  align-items: center;
  margin-right: 0;
}

.toolbar-checkbox :deep(.el-checkbox) {
  display: inline-flex;
  align-items: center;
  height: 100%;
}

.toolbar-checkbox :deep(.el-checkbox__label) {
  padding-left: 0.4rem;
  color: var(--el-text-color-secondary);
  font-size: 0.78rem;
  line-height: 1;
}

.toolbar-search {
  width: 15.5rem;
}

.toolbar-search :deep(.el-input__wrapper) {
  min-height: calc(var(--file-manager-control-height) - 0.48rem);
  border-radius: calc(var(--file-manager-control-radius) - 0.18rem);
  background: transparent;
  padding-inline: 0.68rem 0.32rem;
  box-shadow: none;
}

.toolbar-search :deep(.el-input__inner) {
  font-size: 0.82rem;
}

.search-trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  width: 1.85rem;
  height: 1.85rem;
  border-radius: calc(var(--file-manager-control-radius) - 0.22rem);
  background: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  padding: 0;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  transition:
    background-color 0.18s ease,
    color 0.18s ease;
}

.search-trigger:hover:not(:disabled) {
  background: color-mix(in srgb, var(--el-color-primary) 16%, transparent);
  color: var(--el-text-color-primary);
}

.search-trigger:disabled {
  background: transparent;
  color: color-mix(in srgb, var(--el-text-color-placeholder) 88%, transparent);
  cursor: not-allowed;
}

.panel-note {
  margin-top: 0.62rem;
  border-radius: 0.72rem;
  background: color-mix(in srgb, var(--el-color-primary) 4%, var(--el-bg-color-page));
  padding: 0.62rem 0.75rem;
  color: var(--el-text-color-secondary);
  font-size: 0.74rem;
}

.file-table-shell {
  margin-top: 0.72rem;
  overflow: hidden;
  border: 1px solid var(--el-border-color-extra-light);
  border-radius: 0.85rem;
  background: var(--el-bg-color-page);
  min-height: 22rem;
}

.file-table-shell.is-nested {
  min-height: 17rem;
}

.file-table-shell.is-compact {
  min-height: 8rem;
}

.file-table :deep(th.el-table__cell) {
  background: color-mix(in srgb, var(--el-bg-color-page) 88%, var(--el-bg-color));
  color: var(--el-text-color-secondary);
  font-weight: 600;
  padding-top: 0.5rem;
  padding-bottom: 0.5rem;
}

.file-table :deep(td.el-table__cell) {
  padding-top: 0.42rem;
  padding-bottom: 0.42rem;
  transition: background-color 0.18s ease;
}

.file-extension {
  display: inline-block;
  padding: 0.08rem 0.48rem;
  border-radius: 0.28rem;
  background: color-mix(in srgb, var(--el-color-primary) 8%, var(--el-bg-color-page));
  color: var(--el-text-color-regular);
  font-size: 0.74rem;
  font-weight: 500;
  letter-spacing: 0.01em;
}

.file-extension--muted {
  color: var(--el-text-color-placeholder);
  background: transparent;
}

.file-table :deep(.el-table__row) {
  cursor: pointer;
}

.file-table :deep(.el-table__row:hover > td.el-table__cell) {
  background: color-mix(in srgb, var(--el-color-primary) 4%, var(--el-bg-color));
}

.file-table :deep(.el-table__row:hover .file-name) {
  color: color-mix(in srgb, var(--el-color-primary) 82%, var(--el-text-color-primary));
}

.file-table :deep(.el-table__row:hover .file-kind-icon) {
  transform: translateY(-1px);
  box-shadow: 0 6px 14px rgb(15 23 42 / 10%);
}

.file-table :deep(.el-table__inner-wrapper::before) {
  display: none;
}

.file-name-cell {
  display: flex;
  align-items: center;
  gap: 0.5rem;
  min-width: 0;
}

.file-kind-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 1.52rem;
  height: 1.52rem;
  border-radius: 0.46rem;
  flex-shrink: 0;
  transition:
    transform 0.18s ease,
    box-shadow 0.18s ease,
    background-color 0.18s ease;
}

.file-kind-icon.is-folder {
  background: color-mix(in srgb, var(--el-color-primary) 10%, var(--el-bg-color));
  color: color-mix(in srgb, var(--el-color-primary) 82%, var(--el-text-color-primary));
}

.file-kind-icon.is-file {
  background: color-mix(in srgb, #000000 5%, var(--el-bg-color));
  color: var(--el-text-color-secondary);
}

.file-name-copy {
  display: grid;
  min-width: 0;
  gap: 0;
}

.file-name {
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 0.82rem;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
  transition: color 0.18s ease;
}

.file-subpath {
  overflow: hidden;
  color: var(--el-text-color-secondary);
  font-size: 0.68rem;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.row-actions {
  display: flex;
  align-items: center;
  gap: 0.42rem;
}

.row-action {
  display: inline-flex;
  align-items: center;
  gap: 0.18rem;
  border: none;
  background: transparent;
  padding: 0;
  color: var(--el-text-color-primary);
  font-size: 0.76rem;
  cursor: pointer;
  transition: color 0.2s ease;
}

.row-action:hover {
  color: color-mix(in srgb, var(--el-color-primary) 75%, var(--el-text-color-primary));
}

.row-action:disabled {
  color: var(--el-text-color-placeholder);
  cursor: not-allowed;
}

.panel-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 1rem;
  margin-top: 0.68rem;
  flex-wrap: wrap;
}

.footer-summary {
  color: var(--el-text-color-secondary);
  font-size: 0.78rem;
}

.directory-switch-enter-active,
.directory-switch-leave-active {
  transition:
    opacity 0.22s ease,
    transform 0.22s ease;
}

.directory-switch-enter-from,
.directory-switch-leave-to {
  opacity: 0;
  transform: translateY(10px);
}

:global(html.dark .file-manager-nav),
:global(html.dark .file-manager-panel) {
  background: color-mix(in srgb, var(--el-bg-color) 94%, #0f172a);
}

:global(html.dark .nav-address-bar),
:global(html.dark .nav-search-bar),
:global(html.dark .nav-search-scope),
:global(html.dark .toolbar-search .el-input__wrapper) {
  background: color-mix(in srgb, #ffffff 5%, #111827);
  border-color: color-mix(in srgb, #ffffff 10%, var(--el-border-color));
}

:global(html.dark .file-table-shell) {
  background: var(--el-bg-color);
  border-color: color-mix(in srgb, #ffffff 10%, var(--el-border-color));
}

:global(html.dark .toolbar-button) {
  --el-button-text-color: #ffffff;
  --el-button-hover-bg-color: color-mix(in srgb, #ffffff 10%, #111827);
  --el-button-hover-border-color: transparent;
  --el-button-hover-text-color: #ffffff;
  --el-button-active-bg-color: color-mix(in srgb, #ffffff 14%, #111827);
  --el-button-active-border-color: transparent;
  --el-button-active-text-color: #ffffff;
  --el-button-disabled-bg-color: transparent;
  --el-button-disabled-border-color: transparent;
  --el-button-disabled-text-color: #7f8da2;
}

:global(html.dark .file-table .el-table),
:global(html.dark .file-table .el-table__inner-wrapper),
:global(html.dark .file-table .el-table__header-wrapper),
:global(html.dark .file-table .el-table__body-wrapper) {
  background: transparent;
}

:global(
  html.dark .file-table .el-table__header-wrapper tr th.el-table-fixed-column--right.el-table__cell
),
:global(html.dark .file-table .el-table__header-wrapper tr th.el-table__fixed-right-patch) {
  background: color-mix(in srgb, var(--el-bg-color-page) 88%, var(--el-bg-color));
}

:global(
  html.dark .file-table .el-table__body-wrapper tr td.el-table-fixed-column--right.el-table__cell
),
:global(html.dark .file-table .el-table__body-wrapper tr td.el-table__fixed-right-patch) {
  background: var(--el-bg-color);
}

:global(
  html.dark
    .file-table
    .el-table__body-wrapper
    tr:hover
    > td.el-table-fixed-column--right.el-table__cell
) {
  background: color-mix(in srgb, var(--el-color-primary) 4%, var(--el-bg-color));
}

@media (width <= 1100px) {
  .file-manager-nav {
    flex-direction: column;
    align-items: stretch;
  }

  .panel-toolbar {
    gap: 0.24rem;
  }

  .toolbar-search {
    flex: 1;
    width: auto;
  }
}

@media (width <= 768px) {
  .file-manager-nav,
  .file-manager-panel {
    padding: 0.74rem;
  }

  .nav-address-bar {
    padding-inline: 0.42rem;
  }

  .toolbar-search {
    width: 100%;
  }

  .file-table-shell {
    min-height: 18rem;
  }

  .toolbar-divider {
    display: none;
  }
}

@media (width <= 576px) {
  .nav-search-bar {
    align-items: stretch;
    flex-direction: column;
  }

  .panel-toolbar {
    gap: 0.12rem;
  }

  .toolbar-search {
    width: 100%;
  }
}

.file-table-shell.is-processing {
  position: relative;
  overflow: hidden;
}

.file-table-shell.is-processing::after {
  content: '';
  position: absolute;
  inset: 0;
  border-radius: 0.85rem;
  border: 2px solid transparent;
  animation: paste-border-pulse 1.2s ease-in-out infinite;
  pointer-events: none;
  z-index: 10;
}

@keyframes paste-border-pulse {
  0%,
  100% {
    border-color: color-mix(in srgb, var(--el-color-primary) 12%, transparent);
    box-shadow: inset 0 0 0 0 color-mix(in srgb, var(--el-color-primary) 4%, transparent);
  }
  50% {
    border-color: color-mix(in srgb, var(--el-color-primary) 38%, transparent);
    box-shadow: inset 0 0 24px 0 color-mix(in srgb, var(--el-color-primary) 8%, transparent);
  }
}

.paste-processing-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: color-mix(in srgb, var(--el-bg-color-page) 78%, transparent);
  backdrop-filter: blur(2px);
  border-radius: 0.85rem;
  z-index: 20;
}

.paste-processing-card {
  display: flex;
  align-items: center;
  gap: 0.75rem;
  padding: 0.85rem 1.5rem;
  border-radius: 0.78rem;
  background: var(--el-bg-color);
  box-shadow: 0 8px 32px rgb(15 23 42 / 12%);
}

.paste-processing-spinner {
  display: inline-block;
  width: 1.15rem;
  height: 1.15rem;
  border-radius: 50%;
  border: 2px solid var(--el-border-color-light);
  border-top-color: var(--el-color-primary);
  animation: paste-spin 0.7s linear infinite;
}

@keyframes paste-spin {
  to {
    transform: rotate(360deg);
  }
}

.paste-processing-text {
  color: var(--el-text-color-secondary);
  font-size: 0.84rem;
  font-weight: 500;
}

.paste-overlay-fade-enter-active,
.paste-overlay-fade-leave-active {
  transition: opacity 0.25s ease;
}

.paste-overlay-fade-enter-from,
.paste-overlay-fade-leave-to {
  opacity: 0;
}

.toolbar-button.is-pasting {
  --el-button-bg-color: color-mix(in srgb, var(--el-color-primary) 10%, transparent);
  --el-button-border-color: color-mix(in srgb, var(--el-color-primary) 25%, transparent);
  --el-button-text-color: var(--el-color-primary);
}
</style>
