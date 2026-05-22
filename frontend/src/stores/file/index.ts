import { defineStore } from 'pinia'
import {
  compressFileSystemRequest,
  copyFileSystemRequest,
  createFileSystemRequest,
  cutFileSystemRequest,
  deleteFileSystemEntriesRequest,
  getFileSystemPreSignRequest,
  getFileSystemUploadPreSignRequest,
  getFileSystemListRequest,
  openFileSystemFileRequest,
  renameFileSystemRequest,
  unzipFileSystemRequest,
} from '@/api/file'
import type { FileManagerWorkbenchItem } from '@/components/fileManager/index.vue'
import {
  FileSortField,
  type FileDirectoryInfo,
  type FileEntryInfo,
  type GetFileSystemListRequest,
} from '@/types/v1/file'
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

const SYSTEM_FILE_DEFAULT_PATH = './servers'
const SYSTEM_FILE_DEFAULT_PAGE_SIZE = 10

const normalizeSystemPath = (value: unknown, fallback = SYSTEM_FILE_DEFAULT_PATH) => {
  if (typeof value !== 'string') {
    return fallback
  }

  const normalizedValue = value.trim()
  return normalizedValue || fallback
}

const getPreSignedFileUrl = async (path: string) => {
  const { data } = await getFileSystemPreSignRequest({ path })
  const resolvedUrl = resolvePreSignedFileUrl(data.downloadUrlPath)

  if (!resolvedUrl) {
    throw new Error('未获取到有效的临时链接')
  }

  return resolvedUrl
}

export const useFileIndexStore = defineStore('file-index', () => {
  const rootPath = ref(SYSTEM_FILE_DEFAULT_PATH)
  const directory = ref<FileDirectoryInfo>()
  const items = ref<FileEntryInfo[]>([])
  const total = ref(0)
  const loading = ref(false)
  const query = reactive<GetFileSystemListRequest>({
    path: SYSTEM_FILE_DEFAULT_PATH,
    page: 1,
    pageSize: SYSTEM_FILE_DEFAULT_PAGE_SIZE,
    keywords: undefined,
    includeSubDir: undefined,
    sortField: FileSortField.FILE_SORT_FIELD_NAME,
    isDesc: false,
  })

  const loadDirectory = async (nextQuery: Partial<GetFileSystemListRequest> = {}) => {
    const keywords = hasOwnQueryKey(nextQuery, 'keywords')
      ? normalizeKeywords(nextQuery.keywords)
      : normalizeKeywords(query.keywords)
    const includeSubDirSource = hasOwnQueryKey(nextQuery, 'includeSubDir')
      ? nextQuery.includeSubDir
      : query.includeSubDir

    Object.assign(query, {
      ...query,
      ...nextQuery,
      path: normalizeSystemPath(nextQuery.path ?? query.path, rootPath.value),
      keywords,
      includeSubDir: normalizeIncludeSubDir(keywords, includeSubDirSource),
    })

    loading.value = true

    try {
      const { data } = await getFileSystemListRequest({ ...query })
      directory.value = data.directory
      items.value = data.items
      total.value = normalizeNumber(data.total, 0)
      query.page = normalizeNumber(data.page, query.page)
      query.pageSize = normalizeNumber(data.pageSize, query.pageSize)

      return data.directory?.parentPath || query.path
    } finally {
      loading.value = false
    }
  }

  const initialize = async (params: { workdir?: unknown } = {}) => {
    rootPath.value = normalizeSystemPath(params.workdir)

    const resolvedPath = await loadDirectory({
      path: rootPath.value,
      page: 1,
      pageSize: query.pageSize,
      keywords: undefined,
      includeSubDir: undefined,
      sortField: FileSortField.FILE_SORT_FIELD_NAME,
      isDesc: false,
    })

    rootPath.value = resolvedPath || rootPath.value
  }

  const applyQuery = async (nextQuery: GetFileSystemListRequest) => {
    await loadDirectory(nextQuery)
  }

  const reloadDirectory = async (path: string) => {
    await loadDirectory({
      ...query,
      path,
    })
  }

  const createEntry = async (path: string, name: string, isDirectory: boolean) => {
    await createFileSystemRequest({
      info: {
        path: joinPath(path, name),
        isDir: isDirectory,
        content: new Uint8Array(),
      },
    })

    await reloadDirectory(path)
  }

  const deleteEntries = async (path: string, entries: FileManagerWorkbenchItem[]) => {
    const { data } = await deleteFileSystemEntriesRequest({
      paths: entries.map((item) => item.path),
    })

    await reloadDirectory(path)
    return data
  }

  const compressEntries = async (path: string, entries: FileManagerWorkbenchItem[], targetPath: string) => {
    const { data } = await compressFileSystemRequest({
      paths: entries.map((item) => item.path),
      targetPath: joinPath(path, targetPath),
    })

    await reloadDirectory(path)
    return data.outputPath
  }

  const renameEntry = async (path: string, entry: FileManagerWorkbenchItem, newName: string) => {
    const { data } = await renameFileSystemRequest({
      path: entry.path,
      newName,
    })

    await reloadDirectory(path)
    return data.path
  }

  const unzipEntry = async (path: string, entry: FileManagerWorkbenchItem[], targetPath: string) => {
    const { data } = await unzipFileSystemRequest({
      path: entry[0]!.path,
      targetPath: joinPath(path, targetPath),
    })

    await reloadDirectory(path)
    return data.outputPath
  }

  const copyEntries = async (path: string, entries: FileManagerWorkbenchItem[]) => {
    const { data } = await copyFileSystemRequest({
      paths: entries.map((item) => item.path),
      targetPath: path,
    })

    const taskId = data.task?.taskId
    if (!taskId) throw new Error('未获取到文件任务')
    const task = await waitFileTask(taskId)
    await reloadDirectory(path)
    return { items: task.items }
  }

  const cutEntries = async (path: string, entries: FileManagerWorkbenchItem[]) => {
    const { data } = await cutFileSystemRequest({
      paths: entries.map((item) => item.path),
      targetPath: path,
    })

    const taskId = data.task?.taskId
    if (!taskId) throw new Error('未获取到文件任务')
    const task = await waitFileTask(taskId)
    await reloadDirectory(path)
    return { items: task.items }
  }

  const downloadEntries = async (entries: FileManagerWorkbenchItem[]) => {
    const downloadTargets = await Promise.all(
      entries.map(async (entry) => ({
        entry,
        downloadUrl: await getPreSignedFileUrl(entry.path),
      })),
    )

    for (const item of downloadTargets) {
      await downloadFileFromUrl(item.downloadUrl, item.entry.name)
    }

    return downloadTargets
  }

  const copyTemporaryLink = async (entry: FileManagerWorkbenchItem) => {
    const temporaryUrl = await getPreSignedFileUrl(entry.path)
    await copyTextToClipboard(temporaryUrl)
    return temporaryUrl
  }

  const getUploadPreSign = async (params: {
    path: string
    fileName: string
    fileSize: number
    hash: string
  }) => {
    const { data } = await getFileSystemUploadPreSignRequest({
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
        objectUrl: await getPreSignedFileUrl(entry.path),
      }
    }

    const { data } = await openFileSystemFileRequest({ path: entry.path })

    return {
      kind: 'text',
      fileName: entry.name,
      filePath: entry.path,
      content: decodeFileContent(data.info),
    }
  }

  const releaseObjectUrl = (value?: string) => {
    if (value && isBlobUrl(value)) {
      URL.revokeObjectURL(value)
    }
  }

  return {
    rootPath,
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
    releaseObjectUrl,
  }
})
