<template>
  <FileManagerPage
    :path="currentPath"
    :items="workbenchItems"
    :loading="loading"
    :note="workbenchNote"
    :stats="workbenchStats"
    :pagination="workbenchPagination"
    :sort-state="workbenchSortState"
    :sortable-fields="workbenchSortableFields"
    :hidden-actions="workbenchHiddenActions"
    pagination-mode="server"
    :on-refresh="reloadDirectory"
    :on-create-entry="handleCreateEntry"
    :on-delete-entries="handleDeleteEntries"
    :on-download-entries="handleDownloadEntries"
    :on-get-upload-pre-sign="handleGetUploadPreSign"
    :on-copy-temporary-link="handleCopyTemporaryLink"
    :on-compress-entries="handleCompressEntries"
    :on-unzip-entry="handleUnzipEntry"
    :on-rename-entry="handleRenameEntry"
    :on-open-entry="openEntry"
    :on-release-object-url="releaseObjectUrl"
    :on-copy-entries="handleCopyEntries"
    :on-cut-entries="handleCutEntries"
    @navigate="handleWorkbenchNavigate"
    @search="handleWorkbenchSearch"
    @page-change="handleWorkbenchPageChange"
    @sort-change="handleWorkbenchSortChange"
  />
</template>

<script setup lang="ts">
import dayjs from 'dayjs'
import { storeToRefs } from 'pinia'
import FileManagerPage, {
  type FileManagerDeleteResult,
  type FileManagerDownloadResult,
} from '@/components/fileManager/FileManagerPage.vue'
import type {
  FileManagerUploadPreSignPayload,
  FileManagerWorkbenchItem,
  FileManagerWorkbenchPageChangePayload,
  FileManagerWorkbenchSearchPayload,
  FileManagerWorkbenchSortChangePayload,
  FileManagerWorkbenchSortProp,
  FileManagerWorkbenchSortState,
  FileManagerWorkbenchSortableField,
} from '@/components/fileManager/index.vue'
import type { FileManagerAction } from '@/components/fileManager/index.vue'
import { useInstanceFileManagerStore } from '@/stores/instance/fileManager'
import { useTabsStore } from '@/stores/tabs'
import { FileSortField, type FileEntryInfo } from '@/types/v1/file'
import type { GetInstanceFileListRequest } from '@/types/v1/instance'
import { showRequestError } from '@/utils/request'
import { getInstanceInfoRequest } from '@/api/instance'

defineOptions({ name: 'InstanceFileManagerView' })

const route = useRoute()
const fileManagerStore = useInstanceFileManagerStore()
const tabsStore = useTabsStore()

const { directory, items, total, loading, query } = storeToRefs(fileManagerStore)
const {
  initialize,
  applyQuery,
  reloadDirectory,
  createEntry,
  deleteEntries,
  renameEntry,
  compressEntries,
  unzipEntry,
  downloadEntries,
  copyTemporaryLink,
  copyEntries,
  cutEntries,
  getUploadPreSign,
  openEntry,
  releaseObjectUrl,
} = fileManagerStore

const workbenchHiddenActions: FileManagerAction[] = ['more']
const workbenchSortableFields: FileManagerWorkbenchSortableField[] = ['name', 'updatedAt']
const workbenchNote = '当前接口支持名称和修改时间排序，点击文件可直接查看内容。'

const runSafely = (task: Promise<unknown>, fallback: string) => {
  void task.catch((error) => {
    showRequestError(error, fallback)
  })
}

const formatBytes = (bytes: number) => {
  if (!bytes) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let value = Number(bytes) || 0
  let unitIndex = 0
  while (value >= 1024 && unitIndex < units.length - 1) {
    value /= 1024
    unitIndex += 1
  }
  return `${value >= 100 ? value.toFixed(0) : value.toFixed(2).replace(/\.00$/, '')} ${units[unitIndex]}`
}

const formatUpdateTime = (value: Date | string | undefined) => {
  if (!value) return ''
  const parsed = dayjs(value instanceof Date ? value : String(value))
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm:ss') : String(value)
}

const toTimestamp = (value: Date | string | undefined) => {
  if (!value) return 0
  const parsed = dayjs(value instanceof Date ? value : String(value))
  return parsed.isValid() ? parsed.valueOf() : 0
}

const formatOwnerGroup = (entry: FileEntryInfo) =>
  `${entry.userName || '-'} (${entry.userId ?? '-'}) / ${entry.groupName || '-'} (${entry.groupId ?? '-'})`

const mapSortFieldToProp = (field: FileSortField): FileManagerWorkbenchSortProp | null => {
  if (field === FileSortField.FILE_SORT_FIELD_UPDATE_TIME) return 'updatedAt'
  if (field === FileSortField.FILE_SORT_FIELD_NAME) return 'name'
  return null
}

const mapSortPropToField = (prop: FileManagerWorkbenchSortProp | null) => {
  if (prop === 'updatedAt') return FileSortField.FILE_SORT_FIELD_UPDATE_TIME
  if (prop === 'name') return FileSortField.FILE_SORT_FIELD_NAME
  return FileSortField.FILE_SORT_FIELD_UNSPECIFIED
}

const currentPath = computed(
  () => directory.value?.parentPath ?? query.value.path ?? '',
)

const workbenchItems = computed<FileManagerWorkbenchItem[]>(() =>
  items.value.map((entry) => ({
    id: entry.path,
    name: entry.name,
    path: entry.path,
    isDirectory: entry.isDir,
    extension: entry.isDir ? undefined : entry.name.match(/\.[^./\\]+$/)?.[0] || '',
    permission: entry.permission,
    ownerGroup: formatOwnerGroup(entry),
    sizeLabel: entry.isDir ? '--' : formatBytes(Number(entry.size) || 0),
    sizeValue: entry.isDir ? 0 : Number(entry.size) || 0,
    updatedAtLabel: formatUpdateTime(entry.updateTime),
    updatedAtValue: toTimestamp(entry.updateTime),
  })),
)

const workbenchStats = computed(() => ({
  folders: directory.value?.dirCount || 0,
  files: directory.value?.fileCount || 0,
}))

const workbenchPagination = computed(() => ({
  page: query.value.page,
  pageSize: query.value.pageSize,
  total: total.value,
}))

const workbenchSortState = computed<FileManagerWorkbenchSortState>(() => ({
  prop: mapSortFieldToProp(query.value.sortField),
  order:
    query.value.sortField === FileSortField.FILE_SORT_FIELD_UNSPECIFIED
      ? null
      : query.value.isDesc
        ? 'descending'
        : 'ascending',
}))

watch(
  () => route.params.instanceId,
  (instanceId) => {
    const id = typeof instanceId === 'string' ? instanceId : ''
    if (!id) return

    runSafely(initialize({ instanceId: id }), '实例文件列表加载失败')

    getInstanceInfoRequest({ id }).then(({ data }) => {
      if (data.info?.name) {
        tabsStore.updateTabTitle(route.fullPath, data.info.name)
      }
    }).catch(() => {
      // tab title stays as-is on error
    })
  },
  { immediate: true },
)

const handleCreateEntry = async (payload: {
  path: string
  name: string
  isDirectory: boolean
}) => {
  await createEntry(payload.path, payload.name, payload.isDirectory)
}

const handleDeleteEntries = async (payload: {
  path: string
  entries: FileManagerWorkbenchItem[]
}): Promise<FileManagerDeleteResult> => {
  const { items: resultItems } = await deleteEntries(payload.path, payload.entries)
  const failedItems = resultItems.filter((item) => !item.success)

  if (failedItems.length) {
    return {
      warningMessage:
        failedItems[0]?.message ||
        `已删除 ${resultItems.length - failedItems.length} 项，部分条目删除失败`,
    }
  }

  return {
    successMessage: '删除成功',
  }
}

const handleCompressEntries = async (
  path: string,
  entries: FileManagerWorkbenchItem[],
  targetPath: string,
) => {
  return compressEntries(path, entries, targetPath)
}

const handleUnzipEntry = async (
  path: string,
  entries: FileManagerWorkbenchItem[],
  targetPath: string,
) => {
  return unzipEntry(path, entries, targetPath)
}

const handleRenameEntry = async (
  path: string,
  entry: FileManagerWorkbenchItem,
  newName: string,
) => {
  return renameEntry(path, entry, newName)
}

const handleCopyEntries = async (path: string, entries: FileManagerWorkbenchItem[]) => {
  await copyEntries(path, entries)
}

const handleCutEntries = async (path: string, entries: FileManagerWorkbenchItem[]) => {
  await cutEntries(path, entries)
}

const handleDownloadEntries = async (
  entries: FileManagerWorkbenchItem[],
): Promise<FileManagerDownloadResult> => {
  const downloadTargets = await downloadEntries(entries)
  return {
    count: downloadTargets.length,
    firstEntryName: downloadTargets[0]?.entry.name,
  }
}

const handleCopyTemporaryLink = async (entry: FileManagerWorkbenchItem) => {
  await copyTemporaryLink(entry)
}

const handleGetUploadPreSign = async (payload: FileManagerUploadPreSignPayload) => {
  return getUploadPreSign({
    path: payload.path,
    fileName: payload.fileName,
    fileSize: payload.fileSize,
    hash: payload.hash,
  })
}

const handleQueryChange = (nextQuery: GetInstanceFileListRequest) => {
  runSafely(applyQuery(nextQuery), '目录加载失败')
}

const handleWorkbenchNavigate = (path: string) => {
  handleQueryChange({
    ...query.value,
    path,
    page: 1,
    keywords: undefined,
    includeSubDir: undefined,
  })
}

const handleWorkbenchSearch = (payload: FileManagerWorkbenchSearchPayload) => {
  const keywords = payload.keyword.trim()
  handleQueryChange({
    ...query.value,
    path: payload.path,
    page: 1,
    keywords,
    includeSubDir: keywords ? payload.includeSubdirectories : undefined,
  })
}

const handleWorkbenchPageChange = (payload: FileManagerWorkbenchPageChangePayload) => {
  handleQueryChange({
    ...query.value,
    path: payload.path,
    page: payload.page,
    pageSize: payload.pageSize,
  })
}

const handleWorkbenchSortChange = (payload: FileManagerWorkbenchSortChangePayload) => {
  handleQueryChange({
    ...query.value,
    path: payload.path,
    page: 1,
    sortField: mapSortPropToField(payload.prop),
    isDesc: payload.order === 'descending',
  })
}
</script>
