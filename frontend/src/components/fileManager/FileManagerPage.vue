<script lang="ts">
export type FileManagerDialogMediaKind = 'image' | 'audio' | 'video'

export interface FileManagerOpenMediaResult {
  kind: 'media'
  mediaKind: FileManagerDialogMediaKind
  fileName: string
  filePath: string
  objectUrl: string
}

export interface FileManagerOpenTextResult {
  kind: 'text'
  fileName: string
  filePath: string
  content: string
}

export type FileManagerOpenResult = FileManagerOpenMediaResult | FileManagerOpenTextResult

export interface FileManagerDeleteResult {
  successMessage?: string
  warningMessage?: string
}

export interface FileManagerDownloadResult {
  count?: number
  firstEntryName?: string
  successMessage?: string
}
</script>

<template>
  <FileManager
    :path="props.path"
    :items="props.items"
    :loading="props.loading"
    :note="props.note"
    :stats="props.stats"
    :pagination="props.pagination"
    :page-sizes="props.pageSizes"
    :sort-state="props.sortState"
    :sortable-fields="props.sortableFields"
    :hidden-actions="props.hiddenActions"
    :searchable="props.searchable"
    :pagination-mode="props.paginationMode"
    :has-clipboard="hasClipboard"
    :paste-loading="pasteLoading"
    @action="handleWorkbenchAction"
    @navigate="emit('navigate', $event)"
    @search="emit('search', $event)"
    @page-change="emit('pageChange', $event)"
    @sort-change="emit('sortChange', $event)"
  />

  <FileUploadDialog
    v-model="uploadDialog.visible"
    :path="uploadDialog.path"
    :on-refresh="props.onRefresh"
    :on-get-upload-pre-sign="props.onGetUploadPreSign"
  />

  <BaseDialog
    v-model="createDialog.visible"
    :title="createDialogTitle"
    width="min(420px, calc(100vw - 24px))"
    :show-fullscreen-button="false"
    :resizable="false"
    :confirm-text="t('fileManager.confirmCreate')"
    :show-confirm-loading="false"
    @close="resetCreateDialog"
    @confirm="submitCreateDialog"
  >
    <div class="dialog-body">
      <el-input
        ref="createNameInput"
        v-model="createDialog.name"
        maxlength="255"
        :placeholder="createDialogPlaceholder"
        @keydown.enter.prevent="submitCreateDialog"
      />
      <p v-if="createDialog.errorText" class="dialog-error">{{ createDialog.errorText }}</p>
    </div>
  </BaseDialog>

  <BaseDialog
    v-model="deleteDialog.visible"
    :title="t('fileManager.confirmDelete')"
    width="min(420px, calc(100vw - 24px))"
    :show-fullscreen-button="false"
    :resizable="false"
    :show-footer="false"
    @close="resetDeleteDialog"
  >
    <div class="dialog-body">
      <p class="dialog-copy">{{ deleteDialogCopy }}</p>
      <div v-if="deleteDialog.entries.length" class="dialog-tags">
        <span
          v-for="entry in deleteDialog.entries.slice(0, 6)"
          :key="entry.path"
          class="dialog-tag"
        >
          {{ entry.name }}
        </span>
      </div>
    </div>
    <template #footer>
      <div class="dialog-footer">
        <el-button @click="deleteDialog.visible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="danger" :loading="deleteDialog.submitting" @click="submitDeleteDialog">
          {{ t('fileManager.confirmDelete') }}
        </el-button>
      </div>
    </template>
  </BaseDialog>

  <BaseDialog
    v-if="mediaDialog.visible && mediaDialog.kind === 'audio'"
    v-model="mediaDialog.visible"
    :title="mediaDialog.fileName"
    width="min(720px, calc(100vw - 32px))"
    :show-fullscreen-button="false"
    :show-footer="false"
    @close="closeMediaDialog"
  >
    <FileMediaDialogContent
      :kind="mediaDialog.kind"
      :object-url="mediaDialog.objectUrl"
    />
  </BaseDialog>

  <Transition name="media-overlay-fade">
    <div
      v-if="mediaDialog.visible && mediaDialog.kind !== 'audio'"
      class="media-fullscreen-overlay"
      @click.self="closeMediaDialog"
      @keydown.escape="closeMediaDialog"
    >
      <button class="media-fullscreen-close" :aria-label="t('common.close')" @click="closeMediaDialog">
        <el-icon size="22"><Close /></el-icon>
      </button>
      <FileMediaDialogContent
        :kind="mediaDialog.kind"
        :object-url="mediaDialog.objectUrl"
      />
    </div>
  </Transition>

  <el-dialog
    v-model="editorDialog.visible"
    class="file-editor-dialog"
    width="min(1160px, calc(100vw - 32px))"
    :show-close="false"
    destroy-on-close
    @closed="closeEditorDialog"
  >
    <FileEditorDialogContent
      v-if="editorDialog.visible"
      :file-name="editorDialog.fileName"
      :file-path="editorDialog.filePath"
      :content="editorDialog.content"
      :on-close="closeEditorDialog"
      :on-save="handleSaveEntry"
    />
  </el-dialog>

  <BaseDialog
    v-model="compressDialog.visible"
    :title="t('fileManager.compressTitle')"
    width="min(480px, calc(100vw - 24px))"
    :show-fullscreen-button="false"
    :resizable="false"
    :confirm-text="t('fileManager.confirmCompress')"
    :show-confirm-loading="false"
    @close="resetCompressDialog"
    @confirm="submitCompressDialog"
  >
    <div class="dialog-body">
      <p class="dialog-copy">{{ compressDialogCopy }}</p>
      <div v-if="compressDialog.entries.length" class="dialog-tags">
        <span
          v-for="entry in compressDialog.entries.slice(0, 6)"
          :key="entry.path"
          class="dialog-tag"
        >
          {{ entry.name }}
        </span>
      </div>
      <el-input
        ref="compressNameInput"
        v-model="compressDialog.targetName"
        maxlength="255"
        :placeholder="t('fileManager.compressPlaceholder')"
        @keydown.enter.prevent="submitCompressDialog"
      />
      <p v-if="compressDialog.errorText" class="dialog-error">{{ compressDialog.errorText }}</p>
    </div>
  </BaseDialog>

  <BaseDialog
    v-model="unzipDialog.visible"
    :title="t('fileManager.unzipTitle')"
    width="min(480px, calc(100vw - 24px))"
    :show-fullscreen-button="false"
    :resizable="false"
    :confirm-text="t('fileManager.confirmUnzip')"
    :show-confirm-loading="false"
    @close="resetUnzipDialog"
    @confirm="submitUnzipDialog"
  >
    <div class="dialog-body">
      <p class="dialog-copy">
        {{ t('fileManager.unzip') }} <strong>{{ unzipDialog.entry?.name }}</strong> {{ t('fileManager.to') }}
      </p>
      <el-input
        ref="unzipNameInput"
        v-model="unzipDialog.targetName"
        maxlength="255"
        :placeholder="t('fileManager.unzipPlaceholder')"
        @keydown.enter.prevent="submitUnzipDialog"
      />
      <p v-if="unzipDialog.errorText" class="dialog-error">{{ unzipDialog.errorText }}</p>
    </div>
  </BaseDialog>

  <BaseDialog
    v-model="renameDialog.visible"
    :title="t('fileManager.renameTitle')"
    width="min(480px, calc(100vw - 24px))"
    :show-fullscreen-button="false"
    :resizable="false"
    :confirm-text="t('fileManager.confirmRename')"
    :show-confirm-loading="false"
    @close="resetRenameDialog"
    @confirm="submitRenameDialog"
  >
    <div class="dialog-body">
      <p class="dialog-copy">
        {{ t('fileManager.rename') }} <strong>{{ renameDialog.entry?.name }}</strong>
      </p>
      <el-input
        ref="renameNameInput"
        v-model="renameDialog.newName"
        maxlength="255"
        :placeholder="t('fileManager.renamePlaceholder')"
        @keydown.enter.prevent="submitRenameDialog"
      />
      <p v-if="renameDialog.errorText" class="dialog-error">{{ renameDialog.errorText }}</p>
    </div>
  </BaseDialog>
</template>

<script setup lang="ts">
import BaseDialog from '@/components/dialog/BaseDialog.vue'
import FileEditorDialogContent from '@/components/fileManager/FileEditorDialogContent.vue'
import FileManager, {
  type FileManagerAction,
  type FileManagerUploadPreSignPayload,
  type FileManagerUploadSession,
  type FileManagerWorkbenchItem,
  type FileManagerWorkbenchPageChangePayload,
  type FileManagerWorkbenchPagination,
  type FileManagerWorkbenchSearchPayload,
  type FileManagerWorkbenchSortChangePayload,
  type FileManagerWorkbenchSortableField,
  type FileManagerWorkbenchSortState,
  type FileManagerWorkbenchStats,
  type FileManagerPaginationMode,
} from '@/components/fileManager/index.vue'
import { Close } from '@element-plus/icons-vue'
import FileMediaDialogContent from '@/components/fileManager/FileMediaDialogContent.vue'
import FileUploadDialog from '@/components/fileManager/FileUploadDialog.vue'
import { showRequestError } from '@/utils/request'
import { useI18n } from 'vue-i18n'

defineOptions({ name: 'FileManagerPage' })

interface CreateEntryPayload {
  path: string
  name: string
  isDirectory: boolean
}

interface DeleteEntriesPayload {
  path: string
  entries: FileManagerWorkbenchItem[]
}

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
    unsupportedActionText?: string
    onRefresh?: (path: string) => Promise<void> | void
    onCreateEntry?: (payload: CreateEntryPayload) => Promise<void> | void
    onDeleteEntries?: (payload: DeleteEntriesPayload) => Promise<FileManagerDeleteResult | void> | void
    onDownloadEntries?: (entries: FileManagerWorkbenchItem[]) => Promise<FileManagerDownloadResult | void> | void
    onGetUploadPreSign?: (
      payload: FileManagerUploadPreSignPayload,
    ) => Promise<FileManagerUploadSession | void> | void
    onCopyTemporaryLink?: (entry: FileManagerWorkbenchItem) => Promise<void> | void
    onCompressEntries?: (path: string, entries: FileManagerWorkbenchItem[], targetPath: string) => Promise<string | void> | void
    onUnzipEntry?: (path: string, entry: FileManagerWorkbenchItem[], targetPath: string) => Promise<string | void> | void
    onRenameEntry?: (path: string, entry: FileManagerWorkbenchItem, newName: string) => Promise<string | void> | void
    onOpenEntry?: (entry: FileManagerWorkbenchItem) => Promise<FileManagerOpenResult | void> | void
    onSaveEntry?: (entry: FileManagerWorkbenchItem, content: string) => Promise<void> | void
    onReleaseObjectUrl?: (value?: string) => void
    onCopyEntries?: (path: string, entries: FileManagerWorkbenchItem[]) => Promise<void> | void
    onCutEntries?: (path: string, entries: FileManagerWorkbenchItem[]) => Promise<void> | void
  }>(),
  {
    loading: false,
    note: '',
    stats: undefined,
    pagination: undefined,
    pageSizes: undefined,
    sortState: undefined,
    sortableFields: undefined,
    hiddenActions: undefined,
    searchable: true,
    paginationMode: 'server',
    unsupportedActionText: '',
    onRefresh: undefined,
    onCreateEntry: undefined,
    onDeleteEntries: undefined,
    onDownloadEntries: undefined,
    onGetUploadPreSign: undefined,
    onCopyTemporaryLink: undefined,
    onCompressEntries: undefined,
    onUnzipEntry: undefined,
    onRenameEntry: undefined,
    onOpenEntry: undefined,
    onSaveEntry: undefined,
    onReleaseObjectUrl: undefined,
    onCopyEntries: undefined,
    onCutEntries: undefined,
  },
)

const emit = defineEmits<{
  navigate: [path: string]
  search: [payload: FileManagerWorkbenchSearchPayload]
  pageChange: [payload: FileManagerWorkbenchPageChangePayload]
  sortChange: [payload: FileManagerWorkbenchSortChangePayload]
}>()

const { t } = useI18n()

type CreateDialogMode = 'createFolder' | 'createFile'

const createNameInputRef = useTemplateRef<HTMLInputElement>('createNameInput')
const compressNameInputRef = useTemplateRef<HTMLInputElement>('compressNameInput')
const unzipNameInputRef = useTemplateRef<HTMLInputElement>('unzipNameInput')
const renameNameInputRef = useTemplateRef<HTMLInputElement>('renameNameInput')

const createDialog = reactive({
  visible: false,
  mode: 'createFolder' as CreateDialogMode,
  path: '',
  name: '',
  errorText: '',
  submitting: false,
})

const deleteDialog = reactive({
  visible: false,
  path: '',
  entries: [] as FileManagerWorkbenchItem[],
  submitting: false,
})

const uploadDialog = reactive({
  visible: false,
  path: '',
})

const mediaDialog = reactive({
  visible: false,
  kind: 'image' as FileManagerDialogMediaKind,
  fileName: '',
  filePath: '',
  objectUrl: '',
})

const editorDialog = reactive({
  visible: false,
  fileName: '',
  filePath: '',
  content: '',
})

const compressDialog = reactive({
  visible: false,
  path: '',
  entries: [] as FileManagerWorkbenchItem[],
  targetName: '',
  errorText: '',
  submitting: false,
})

const unzipDialog = reactive({
  visible: false,
  path: '',
  entry: null as FileManagerWorkbenchItem | null,
  targetName: '',
  errorText: '',
  submitting: false,
})

const renameDialog = reactive({
  visible: false,
  path: '',
  entry: null as FileManagerWorkbenchItem | null,
  newName: '',
  errorText: '',
  submitting: false,
})

interface ClipboardState {
  entries: FileManagerWorkbenchItem[]
  operation: 'copy' | 'cut'
}

const clipboard = ref<ClipboardState | null>(null)
const hasClipboard = computed(() => clipboard.value !== null)
const pasteLoading = ref(false)

const createDialogTitle = computed(() =>
  createDialog.mode === 'createFolder' ? t('fileManager.createFolder') : t('fileManager.createFile'),
)

const createDialogPlaceholder = computed(() =>
  createDialog.mode === 'createFolder'
    ? t('fileManager.createFolderPlaceholder')
    : t('fileManager.createFilePlaceholder'),
)

const deleteDialogCopy = computed(() => {
  if (!deleteDialog.entries.length) {
    return t('fileManager.noDeleteSelection')
  }

  if (deleteDialog.entries.length === 1) {
    return t('fileManager.deleteOneConfirm', { name: deleteDialog.entries[0]!.name })
  }

  return t('fileManager.deleteManyConfirm', { count: deleteDialog.entries.length })
})

const compressDialogCopy = computed(() => {
  if (!compressDialog.entries.length) return t('fileManager.noCompressSelection')
  if (compressDialog.entries.length === 1) {
    return t('fileManager.compressOne', { name: compressDialog.entries[0]!.name })
  }
  return t('fileManager.compressMany', { count: compressDialog.entries.length })
})

const defaultCompressName = (entries: FileManagerWorkbenchItem[]) => {
  if (entries.length === 1) {
    return `${entries[0]!.name}.zip`
  }
  return `archive.zip`
}

const defaultUnzipName = (entry: FileManagerWorkbenchItem) => {
  const name = entry.name
  if (name.toLowerCase().endsWith('.zip')) {
    return name.slice(0, -4)
  }
  return `${name}-${t('fileManager.unzippedSuffix')}`
}

const buildTargetText = (path: string, entries: FileManagerWorkbenchItem[]) => {
  if (entries.length > 0) {
    return t('fileManager.currentTarget', {
      target: entries.map((item) => item.name).join(t('fileManager.listDelimiter')),
    })
  }

  return t('fileManager.currentDirectory', { path })
}

const actionTextKeyMap: Record<FileManagerAction, string> = {
  refresh: 'fileManager.refresh',
  createFolder: 'fileManager.createFolder',
  createFile: 'fileManager.createFile',
  upload: 'fileManager.upload',
  download: 'fileManager.download',
  copyTemporaryLink: 'fileManager.copyLink',
  compress: 'fileManager.compress',
  unzip: 'fileManager.unzip',
  open: 'fileManager.open',
  delete: 'fileManager.delete',
  rename: 'fileManager.rename',
  more: 'fileManager.more',
  copy: 'fileManager.copy',
  cut: 'fileManager.cut',
  paste: 'fileManager.paste',
}

const getActionText = (action: FileManagerAction) => t(actionTextKeyMap[action])

const formatOutputPath = (outputPath?: string | void) =>
  outputPath ? t('fileManager.outputPath', { path: outputPath }) : ''

const showUnsupportedAction = (
  action: FileManagerAction,
  path: string,
  entries: FileManagerWorkbenchItem[],
) => {
  ElMessage.info(
    t('fileManager.unsupportedAction', {
      action: getActionText(action),
      unsupported: props.unsupportedActionText || t('fileManager.unsupported'),
      target: buildTargetText(path, entries),
    }),
  )
}

const resetCreateDialog = () => {
  createDialog.visible = false
  createDialog.mode = 'createFolder'
  createDialog.path = ''
  createDialog.name = ''
  createDialog.errorText = ''
  createDialog.submitting = false
}

const resetDeleteDialog = () => {
  deleteDialog.visible = false
  deleteDialog.path = ''
  deleteDialog.entries = []
  deleteDialog.submitting = false
}

const closeUploadDialog = () => {
  uploadDialog.visible = false
  uploadDialog.path = ''
}

const closeMediaDialog = () => {
  props.onReleaseObjectUrl?.(mediaDialog.objectUrl)
  mediaDialog.visible = false
  mediaDialog.kind = 'image'
  mediaDialog.fileName = ''
  mediaDialog.filePath = ''
  mediaDialog.objectUrl = ''
}

const currentEditorEntry = ref<FileManagerWorkbenchItem | null>(null)

const closeEditorDialog = () => {
  editorDialog.visible = false
  editorDialog.fileName = ''
  editorDialog.filePath = ''
  editorDialog.content = ''
  currentEditorEntry.value = null
}

const handleSaveEntry = async (content: string) => {
  return props.onSaveEntry!(currentEditorEntry.value!, content)
}

const resetRenameDialog = () => {
  renameDialog.visible = false
  renameDialog.path = ''
  renameDialog.entry = null
  renameDialog.newName = ''
  renameDialog.errorText = ''
  renameDialog.submitting = false
}

const openRenameDialog = (path: string, entries: FileManagerWorkbenchItem[]) => {
  const entry = entries[0]
  if (!entry) {
    ElMessage.warning(t('fileManager.selectRenameTarget'))
    return
  }

  resetRenameDialog()
  renameDialog.visible = true
  renameDialog.path = path
  renameDialog.entry = entry
  renameDialog.newName = entry.name
  nextTick(() => {
    renameNameInputRef.value?.focus()
    renameNameInputRef.value?.select()
  })
}

const submitRenameDialog = async () => {
  if (renameDialog.submitting) return

  renameDialog.errorText = validateTargetName(renameDialog.newName)
  if (renameDialog.errorText) return

  const entry = renameDialog.entry
  if (!entry || !props.onRenameEntry) {
    showUnsupportedAction('rename', renameDialog.path, entry ? [entry] : [])
    resetRenameDialog()
    return
  }

  try {
    renameDialog.submitting = true
    const outputPath = await props.onRenameEntry(
      renameDialog.path,
      entry,
      renameDialog.newName.trim(),
    )
    ElMessage.success(t('fileManager.renameDone', { path: formatOutputPath(outputPath) }))
  } catch (error) {
    showRequestError(error, t('fileManager.renameFailed'))
  } finally {
    renameDialog.submitting = false
  }

  renameDialog.visible = false
}

const closeAllDialogs = () => {
  resetCreateDialog()
  resetDeleteDialog()
  closeUploadDialog()
  closeMediaDialog()
  closeEditorDialog()
  resetCompressDialog()
  resetUnzipDialog()
  resetRenameDialog()
}

const validateCreateName = (value: string) => {
  const nextValue = value.trim()
  if (!nextValue) return t('fileManager.nameRequired')
  if (nextValue.includes('/') || nextValue.includes('\\')) return t('fileManager.nameNoSeparator')
  return ''
}

const openCreateDialog = (mode: CreateDialogMode, path: string) => {
  resetCreateDialog()
  createDialog.visible = true
  createDialog.mode = mode
  createDialog.path = path
  nextTick(() => {
    createNameInputRef.value?.focus()
    createNameInputRef.value?.select()
  })
}

const openDeleteDialog = (path: string, entries: FileManagerWorkbenchItem[]) => {
  if (!entries.length) {
    ElMessage.warning(t('fileManager.selectDeleteTarget'))
    return
  }

  resetDeleteDialog()
  deleteDialog.visible = true
  deleteDialog.path = path
  deleteDialog.entries = [...entries]
}

const openUploadDialog = (path: string) => {
  uploadDialog.path = path
  uploadDialog.visible = true
}

const resetCompressDialog = () => {
  compressDialog.visible = false
  compressDialog.path = ''
  compressDialog.entries = []
  compressDialog.targetName = ''
  compressDialog.errorText = ''
  compressDialog.submitting = false
}

const resetUnzipDialog = () => {
  unzipDialog.visible = false
  unzipDialog.path = ''
  unzipDialog.entry = null
  unzipDialog.targetName = ''
  unzipDialog.errorText = ''
  unzipDialog.submitting = false
}

const openCompressDialog = (path: string, entries: FileManagerWorkbenchItem[]) => {
  if (!entries.length) {
    ElMessage.warning(t('fileManager.selectCompressTarget'))
    return
  }

  resetCompressDialog()
  compressDialog.visible = true
  compressDialog.path = path
  compressDialog.entries = [...entries]
  compressDialog.targetName = defaultCompressName(entries)
  nextTick(() => {
    compressNameInputRef.value?.focus()
    compressNameInputRef.value?.select()
  })
}

const openUnzipDialog = (path: string, entries: FileManagerWorkbenchItem[]) => {
  const entry = entries[0]
  if (!entry) {
    ElMessage.warning(t('fileManager.selectUnzipTarget'))
    return
  }

  resetUnzipDialog()
  unzipDialog.visible = true
  unzipDialog.path = path
  unzipDialog.entry = entry
  unzipDialog.targetName = defaultUnzipName(entry)
  nextTick(() => {
    unzipNameInputRef.value?.focus()
    unzipNameInputRef.value?.select()
  })
}

const validateTargetName = (value: string) => {
  const nextValue = value.trim()
  if (!nextValue) return t('fileManager.nameRequired')
  if (nextValue.includes('/') || nextValue.includes('\\')) return t('fileManager.nameNoSeparator')
  return ''
}

const submitCompressDialog = async () => {
  if (compressDialog.submitting) return

  compressDialog.errorText = validateTargetName(compressDialog.targetName)
  if (compressDialog.errorText) return

  if (!props.onCompressEntries) {
    showUnsupportedAction('compress', compressDialog.path, compressDialog.entries)
    resetCompressDialog()
    return
  }

  try {
    compressDialog.submitting = true
    const outputPath = await props.onCompressEntries(
      compressDialog.path,
      compressDialog.entries,
      compressDialog.targetName.trim(),
    )
    ElMessage.success(t('fileManager.compressDone', { path: formatOutputPath(outputPath) }))
  } catch (error) {
    showRequestError(error, t('fileManager.compressFailed'))
  } finally {
    compressDialog.submitting = false
  }

  compressDialog.visible = false
}

const submitUnzipDialog = async () => {
  if (unzipDialog.submitting) return

  unzipDialog.errorText = validateTargetName(unzipDialog.targetName)
  if (unzipDialog.errorText) return

  const entry = unzipDialog.entry
  if (!entry || !props.onUnzipEntry) {
    showUnsupportedAction('unzip', unzipDialog.path, entry ? [entry] : [])
    resetUnzipDialog()
    return
  }

  try {
    unzipDialog.submitting = true
    const outputPath = await props.onUnzipEntry(
      unzipDialog.path,
      [entry],
      unzipDialog.targetName.trim(),
    )
    ElMessage.success(t('fileManager.unzipDone', { path: formatOutputPath(outputPath) }))
  } catch (error) {
    showRequestError(error, t('fileManager.unzipFailed'))
  } finally {
    unzipDialog.submitting = false
  }

  unzipDialog.visible = false
}

const submitCreateDialog = async () => {
  if (createDialog.submitting) {
    return
  }

  createDialog.errorText = validateCreateName(createDialog.name)
  if (createDialog.errorText) {
    return
  }

  if (!props.onCreateEntry) {
    showUnsupportedAction(createDialog.mode, createDialog.path, [])
    resetCreateDialog()
    return
  }

  try {
    createDialog.submitting = true
    const isDirectory = createDialog.mode === 'createFolder'
    await props.onCreateEntry({
      path: createDialog.path,
      name: createDialog.name.trim(),
      isDirectory,
    })
    ElMessage.success(
      isDirectory ? t('fileManager.folderCreateSuccess') : t('fileManager.fileCreateSuccess'),
    )
  } catch (error) {
    showRequestError(
      error,
      createDialog.mode === 'createFolder'
        ? t('fileManager.folderCreateFailed')
        : t('fileManager.fileCreateFailed'),
    )
  } finally {
    createDialog.submitting = false
  }

  createDialog.visible = false
}

const submitDeleteDialog = async () => {
  if (deleteDialog.submitting) {
    return
  }

  if (!props.onDeleteEntries) {
    showUnsupportedAction('delete', deleteDialog.path, deleteDialog.entries)
    resetDeleteDialog()
    return
  }

  try {
    deleteDialog.submitting = true
    const result = await props.onDeleteEntries({
      path: deleteDialog.path,
      entries: deleteDialog.entries,
    })

    if (result?.warningMessage) {
      ElMessage.warning(result.warningMessage)
    } else {
      ElMessage.success(result?.successMessage || t('fileManager.deleteSuccess'))
    }
  } catch (error) {
    showRequestError(error, t('fileManager.deleteFailed'))
  } finally {
    deleteDialog.submitting = false
  }

  deleteDialog.visible = false
}

const handleDownloadAction = async (
  path: string,
  entries: FileManagerWorkbenchItem[],
) => {
  if (!props.onDownloadEntries) {
    showUnsupportedAction('download', path, entries)
    return
  }

  const downloadableEntries = entries.filter((entry) => !entry.isDirectory)
  if (!downloadableEntries.length) {
    ElMessage.warning(t('fileManager.selectDownloadFile'))
    return
  }

  try {
    const result = await props.onDownloadEntries(downloadableEntries)
    if (result?.successMessage) {
      ElMessage.success(result.successMessage)
      return
    }

    const count = result?.count ?? downloadableEntries.length
    if (count === 1) {
      ElMessage.success(
        t('fileManager.downloadStarted', {
          name: result?.firstEntryName || downloadableEntries[0]!.name,
        }),
      )
      return
    }

    ElMessage.success(t('fileManager.downloadManyStarted', { count }))
  } catch (error) {
    showRequestError(error, t('fileManager.downloadFailed'))
  }
}

const handleCopyTemporaryLinkAction = async (
  path: string,
  entries: FileManagerWorkbenchItem[],
) => {
  if (!props.onCopyTemporaryLink) {
    showUnsupportedAction('copyTemporaryLink', path, entries)
    return
  }

  const targetEntry = entries.find((entry) => !entry.isDirectory)
  if (!targetEntry) {
    ElMessage.warning(t('fileManager.selectCopyLinkFile'))
    return
  }

  try {
    await props.onCopyTemporaryLink(targetEntry)
    ElMessage.success(t('fileManager.copiedTemporaryLink', { name: targetEntry.name }))
  } catch (error) {
    showRequestError(error, t('fileManager.copyLinkFailed'))
  }
}

const handleOpenAction = async (
  path: string,
  entries: FileManagerWorkbenchItem[],
) => {
  if (!props.onOpenEntry) {
    showUnsupportedAction('open', path, entries)
    return
  }

  const targetEntry = entries[0]
  if (!targetEntry) {
    return
  }

  try {
    const result = await props.onOpenEntry(targetEntry)
    if (!result) {
      showUnsupportedAction('open', path, entries)
      return
    }

    if (result.kind === 'media') {
      closeMediaDialog()
      closeEditorDialog()
      mediaDialog.visible = true
      mediaDialog.kind = result.mediaKind
      mediaDialog.fileName = result.fileName
      mediaDialog.filePath = result.filePath
      mediaDialog.objectUrl = result.objectUrl
      return
    }

    closeEditorDialog()
    editorDialog.visible = true
    editorDialog.fileName = result.fileName
    editorDialog.filePath = result.filePath
    editorDialog.content = result.content
    currentEditorEntry.value = targetEntry
  } catch (error) {
    showRequestError(error, t('fileManager.openFailed'))
  }
}

const handleCompressAction = (
  path: string,
  entries: FileManagerWorkbenchItem[],
) => {
  if (!props.onCompressEntries) {
    showUnsupportedAction('compress', path, entries)
    return
  }
  openCompressDialog(path, entries)
}

const handleUnzipAction = (
  path: string,
  entries: FileManagerWorkbenchItem[],
) => {
  if (!props.onUnzipEntry) {
    showUnsupportedAction('unzip', path, entries)
    return
  }
  openUnzipDialog(path, entries)
}

const handleWorkbenchAction = (
  action: FileManagerAction,
  path: string,
  entries: FileManagerWorkbenchItem[],
) => {
  if (action === 'refresh') {
    if (!props.onRefresh) {
      showUnsupportedAction(action, path, entries)
      return
    }

    Promise.resolve(props.onRefresh(path)).catch((error) => {
      showRequestError(error, t('fileManager.refreshFailed'))
    })
    return
  }

  if (action === 'createFolder' || action === 'createFile') {
    openCreateDialog(action, path)
    return
  }

  if (action === 'delete') {
    openDeleteDialog(path, entries)
    return
  }

  if (action === 'download') {
    void handleDownloadAction(path, entries)
    return
  }

  if (action === 'upload') {
    if (!props.onGetUploadPreSign) {
      showUnsupportedAction(action, path, entries)
      return
    }

    openUploadDialog(path)
    return
  }

  if (action === 'copyTemporaryLink') {
    void handleCopyTemporaryLinkAction(path, entries)
    return
  }

  if (action === 'open') {
    void handleOpenAction(path, entries)
    return
  }

  if (action === 'compress') {
    handleCompressAction(path, entries)
    return
  }

  if (action === 'rename') {
    if (!props.onRenameEntry) {
      showUnsupportedAction('rename', path, entries)
      return
    }
    openRenameDialog(path, entries)
    return
  }

  if (action === 'unzip') {
    handleUnzipAction(path, entries)
    return
  }

  if (action === 'copy') {
    if (!entries.length) {
      ElMessage.warning(t('fileManager.selectCopyTarget'))
      return
    }
    clipboard.value = { entries: [...entries], operation: 'copy' }
    ElMessage.success(t('fileManager.copiedItems', { count: entries.length }))
    return
  }

  if (action === 'cut') {
    if (!entries.length) {
      ElMessage.warning(t('fileManager.selectCutTarget'))
      return
    }
    clipboard.value = { entries: [...entries], operation: 'cut' }
    ElMessage.success(t('fileManager.cutItems', { count: entries.length }))
    return
  }

  if (action === 'paste') {
    if (!clipboard.value) {
      ElMessage.warning(t('fileManager.copyOrCutFirst'))
      return
    }
    const { entries: clipEntries, operation } = clipboard.value
    pasteLoading.value = true
    if (operation === 'copy') {
      if (!props.onCopyEntries) {
        showUnsupportedAction('paste', path, entries)
        pasteLoading.value = false
        return
      }
      Promise.resolve(props.onCopyEntries(path, clipEntries))
        .then(() => {
          ElMessage.success(t('fileManager.pasteDone'))
        })
        .catch((error: unknown) => {
          showRequestError(error, t('fileManager.pasteFailed'))
        })
        .finally(() => {
          pasteLoading.value = false
        })
    } else {
      if (!props.onCutEntries) {
        showUnsupportedAction('paste', path, entries)
        pasteLoading.value = false
        return
      }
      Promise.resolve(props.onCutEntries(path, clipEntries))
        .then(() => {
          clipboard.value = null
          ElMessage.success(t('fileManager.pasteDone'))
        })
        .catch((error: unknown) => {
          showRequestError(error, t('fileManager.pasteFailed'))
        })
        .finally(() => {
          pasteLoading.value = false
        })
    }
    return
  }

  showUnsupportedAction(action, path, entries)
}

watch(
  () => createDialog.visible,
  (visible) => {
    if (!visible) return
    nextTick(() => {
      createNameInputRef.value?.focus()
      createNameInputRef.value?.select()
    })
  },
)

onBeforeUnmount(() => {
  closeAllDialogs()
})
</script>

<style scoped lang="scss">
.dialog-body {
  display: grid;
  gap: 0.85rem;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 0.75rem;
}

.dialog-copy {
  margin: 0;
  color: var(--el-text-color-primary);
  line-height: 1.7;
}

.dialog-error {
  margin: 0;
  color: var(--el-color-danger);
  font-size: 0.82rem;
}

.dialog-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
}

.dialog-tag {
  display: inline-flex;
  align-items: center;
  max-width: 100%;
  border-radius: 999px;
  background: color-mix(in srgb, var(--el-color-primary) 10%, var(--el-bg-color));
  padding: 0.3rem 0.7rem;
  color: var(--el-text-color-primary);
  font-size: 0.78rem;
}

.media-fullscreen-overlay {
  position: fixed;
  inset: 0;
  z-index: 2500;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgb(0 0 0 / 82%);
  backdrop-filter: blur(6px);
}

.media-fullscreen-close {
  position: absolute;
  top: 1.25rem;
  right: 1.25rem;
  z-index: 10;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 2.5rem;
  height: 2.5rem;
  border: none;
  border-radius: 50%;
  background: rgb(255 255 255 / 12%);
  color: #fff;
  cursor: pointer;
  transition: background 0.2s;
}

.media-fullscreen-close:hover {
  background: rgb(255 255 255 / 24%);
}

.media-overlay-fade-enter-active,
.media-overlay-fade-leave-active {
  transition: opacity 0.25s ease;
}

.media-overlay-fade-enter-from,
.media-overlay-fade-leave-to {
  opacity: 0;
}
</style>
