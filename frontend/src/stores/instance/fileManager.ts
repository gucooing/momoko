import { defineStore } from 'pinia'
import {
  getInstanceFileListRequest,
  createInstanceFileRequest,
  batchDeleteInstanceFileRequest,
  openInstanceFileRequest,
  editInstanceFileRequest,
  getInstanceFilePreSignRequest,
  getInstanceFilePreSignUploadRequest,
  batchCompressInstanceFileRequest,
  renameInstanceFileRequest,
  unzipInstanceFileRequest,
  copyInstanceFileRequest,
  cutInstanceFileRequest,
} from '@/api/instance'
import type { FileManagerWorkbenchItem } from '@/components/fileManager/index.vue'
import {
  FileSortField,
  type FileDirectoryInfo,
  type FileEntryInfo,
} from '@/types/v1/file'
import type { GetInstanceFileListRequest } from '@/types/v1/instance'
import {
  type LocalFileOpenResult,
  copyTextToClipboard,
  decodeFileContent,
  downloadFileFromUrl,
  hasOwnQueryKey,
  isBlobUrl,
  joinPath,
  normalizeIncludeSubDir,
  normalizeKeywords,
  normalizeNumber,
  resolveFilePreviewKind,
  resolvePreSignedFileUrl,
} from '@/utils/filePreview'
import { waitFileTask } from '@/utils/fileTask'
import { translate } from '@/locales'

const INSTANCE_FILE_DEFAULT_PAGE_SIZE = 10

const getPreSignedFileUrl = async (instanceId: string, path: string) => {
  const { data } = await getInstanceFilePreSignRequest({ id: instanceId, path })
  const resolvedUrl = resolvePreSignedFileUrl(data.downloadUrlPath)

  if (!resolvedUrl) {
    throw new Error(translate('instance.invalidTemporaryLink'))
  }

  return resolvedUrl
}

export const useInstanceFileManagerStore = defineStore('instance-file-manager', () => {
  const instanceId = ref('')
  const directory = ref<FileDirectoryInfo>()
  const items = ref<FileEntryInfo[]>([])
  const total = ref(0)
  const loading = ref(false)
  const query = reactive<GetInstanceFileListRequest>({
    id: '',
    path: '',
    page: 1,
    pageSize: INSTANCE_FILE_DEFAULT_PAGE_SIZE,
    keywords: undefined,
    includeSubDir: undefined,
    sortField: FileSortField.FILE_SORT_FIELD_NAME,
    isDesc: false,
  })

  const loadDirectory = async (nextQuery: Partial<GetInstanceFileListRequest> = {}) => {
    const keywords = hasOwnQueryKey(nextQuery, 'keywords')
      ? normalizeKeywords(nextQuery.keywords)
      : normalizeKeywords(query.keywords)
    const includeSubDirSource = hasOwnQueryKey(nextQuery, 'includeSubDir')
      ? nextQuery.includeSubDir
      : query.includeSubDir

    Object.assign(query, {
      ...query,
      ...nextQuery,
      id: instanceId.value,
      path: nextQuery.path ?? query.path ?? '',
      keywords,
      includeSubDir: normalizeIncludeSubDir(keywords, includeSubDirSource),
    })

    loading.value = true

    try {
      const { data } = await getInstanceFileListRequest({ ...query })
      directory.value = data.directory
      items.value = data.items
      total.value = normalizeNumber(data.total, 0)
      query.page = normalizeNumber(data.page, query.page)
      query.pageSize = normalizeNumber(data.pageSize, query.pageSize)

      return data.directory?.parentPath ?? query.path
    } finally {
      loading.value = false
    }
  }

  const initialize = async (params: { instanceId?: unknown; dirPath?: unknown } = {}) => {
    instanceId.value = typeof params.instanceId === 'string' ? params.instanceId : ''
    const dirPath = typeof params.dirPath === 'string' && params.dirPath ? params.dirPath : ''

    const resolvedPath = await loadDirectory({
      id: instanceId.value,
      path: dirPath,
      page: 1,
      pageSize: query.pageSize,
      keywords: undefined,
      includeSubDir: undefined,
      sortField: FileSortField.FILE_SORT_FIELD_NAME,
      isDesc: false,
    })

    if (resolvedPath !== undefined) {
      query.path = resolvedPath
    }
  }

  const applyQuery = async (nextQuery: GetInstanceFileListRequest) => {
    await loadDirectory(nextQuery)
  }

  const reloadDirectory = async (path: string) => {
    await loadDirectory({
      ...query,
      path,
    })
  }

  const createEntry = async (path: string, name: string, isDirectory: boolean) => {
    await createInstanceFileRequest({
      id: instanceId.value,
      info: {
        path: joinPath(path, name),
        isDir: isDirectory,
        content: new Uint8Array(),
      },
    })

    await reloadDirectory(path)
  }

  const deleteEntries = async (path: string, entries: FileManagerWorkbenchItem[]) => {
    const { data } = await batchDeleteInstanceFileRequest({
      id: instanceId.value,
      paths: entries.map((item) => item.path),
    })

    await reloadDirectory(path)
    return data
  }

  const compressEntries = async (
    path: string,
    entries: FileManagerWorkbenchItem[],
    targetPath: string,
  ) => {
    const { data } = await batchCompressInstanceFileRequest({
      id: instanceId.value,
      paths: entries.map((item) => item.path),
      targetPath: joinPath(path, targetPath),
    })

    await reloadDirectory(path)
    return data.outputPath
  }

  const renameEntry = async (path: string, entry: FileManagerWorkbenchItem, newName: string) => {
    const { data } = await renameInstanceFileRequest({
      id: instanceId.value,
      path: entry.path,
      newName,
    })

    await reloadDirectory(path)
    return data.path
  }

  const unzipEntry = async (
    path: string,
    entry: FileManagerWorkbenchItem[],
    targetPath: string,
  ) => {
    const { data } = await unzipInstanceFileRequest({
      id: instanceId.value,
      path: entry[0]!.path,
      targetPath: joinPath(path, targetPath),
    })

    await reloadDirectory(path)
    return data.outputPath
  }

  const copyEntries = async (path: string, entries: FileManagerWorkbenchItem[]) => {
    const { data } = await copyInstanceFileRequest({
      id: instanceId.value,
      paths: entries.map((item) => item.path),
      targetPath: path,
    })

    const taskId = data.task?.taskId
    if (!taskId) throw new Error(translate('instance.missingFileTask'))
    const task = await waitFileTask(taskId)
    await reloadDirectory(path)
    return { items: task.items }
  }

  const cutEntries = async (path: string, entries: FileManagerWorkbenchItem[]) => {
    const { data } = await cutInstanceFileRequest({
      id: instanceId.value,
      paths: entries.map((item) => item.path),
      targetPath: path,
    })

    const taskId = data.task?.taskId
    if (!taskId) throw new Error(translate('instance.missingFileTask'))
    const task = await waitFileTask(taskId)
    await reloadDirectory(path)
    return { items: task.items }
  }

  const downloadEntries = async (entries: FileManagerWorkbenchItem[]) => {
    const downloadTargets = await Promise.all(
      entries.map(async (entry) => ({
        entry,
        downloadUrl: await getPreSignedFileUrl(instanceId.value, entry.path),
      })),
    )

    for (const item of downloadTargets) {
      await downloadFileFromUrl(item.downloadUrl, item.entry.name)
    }

    return downloadTargets
  }

  const copyTemporaryLink = async (entry: FileManagerWorkbenchItem) => {
    const temporaryUrl = await getPreSignedFileUrl(instanceId.value, entry.path)
    await copyTextToClipboard(temporaryUrl)
    return temporaryUrl
  }

  const getUploadPreSign = async (params: {
    path: string
    fileName: string
    fileSize: number
    hash: string
  }) => {
    const { data } = await getInstanceFilePreSignUploadRequest({
      id: instanceId.value,
      path: params.path,
      fileName: params.fileName,
      fileSize: params.fileSize,
      hash: params.hash,
    })

    return data.info
  }

  const openEntry = async (entry: FileManagerWorkbenchItem): Promise<LocalFileOpenResult> => {
    const previewKind = resolveFilePreviewKind(entry.path)

    if (previewKind) {
      return {
        kind: 'media',
        mediaKind: previewKind,
        fileName: entry.name,
        filePath: entry.path,
        objectUrl: await getPreSignedFileUrl(instanceId.value, entry.path),
      }
    }

    const { data } = await openInstanceFileRequest({
      id: instanceId.value,
      path: entry.path,
    })

    return {
      kind: 'text',
      fileName: entry.name,
      filePath: entry.path,
      content: decodeFileContent(data.info),
    }
  }

  const saveEntry = async (entry: FileManagerWorkbenchItem, content: string) => {
    const encoder = new TextEncoder()
    await editInstanceFileRequest({
      id: instanceId.value,
      path: entry.path,
      content: encoder.encode(content),
    })
  }

  const releaseObjectUrl = (value?: string) => {
    if (value && isBlobUrl(value)) {
      URL.revokeObjectURL(value)
    }
  }

  return {
    instanceId,
    directory,
    items,
    total,
    loading,
    query,
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
    saveEntry,
    releaseObjectUrl,
  }
})
