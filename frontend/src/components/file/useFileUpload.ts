import { ref, computed } from 'vue'
import { translate as t } from '@/locales'
import { resolveStaticResourceUrl } from '@/utils/assets'
import { cancelFileUpload, completeFileUpload, getFileUploadStatus } from '@/api/file'
import { waitFileTask } from '@/utils/fileTask'
import type { FileClient } from './types'

// 分片并发上传：scope 无关。预签名走 client（系统/实例各自路由），分片 PUT 直传签名 URL
// （/api/v1/upload/pre 对 PUT 免 JWT，sign 自校验），完成/取消/状态走系统级共享会话端点。

export type UploadStatus =
  | 'waiting'
  | 'hashing'
  | 'uploading'
  | 'verifying'
  | 'finishing'
  | 'done'
  | 'failed'
  | 'canceled'

export interface UploadItem {
  id: string
  file: File
  name: string
  size: number
  status: UploadStatus
  statusText: string
  progress: number
  uploadId?: string
  error?: string
}

const FAST_HASH_SAMPLE = 2 * 1024 * 1024 // 头尾各采样 2MB 做快速指纹

const toHex = (buffer: ArrayBuffer): string => {
  const bytes = new Uint8Array(buffer)
  let hex = ''
  for (const byte of bytes) hex += byte.toString(16).padStart(2, '0')
  return hex
}

// 退化哈希（无 SubtleCrypto 时，如非 localhost 的纯 http）。
const fnv1a = (bytes: Uint8Array): string => {
  let hash = 0x811c9dc5
  for (const byte of bytes) {
    hash ^= byte
    hash = Math.imul(hash, 0x01000193)
  }
  return (hash >>> 0).toString(16)
}

// 确定性快速哈希：大文件采样头尾，附带大小，保证同文件同值（支持秒传/断点续传）。
const computeFastHash = async (file: File): Promise<string> => {
  const blob =
    file.size <= FAST_HASH_SAMPLE * 2
      ? file
      : new Blob([file.slice(0, FAST_HASH_SAMPLE), file.slice(file.size - FAST_HASH_SAMPLE)])
  const buffer = await blob.arrayBuffer()
  if (globalThis.crypto?.subtle) {
    const digest = await globalThis.crypto.subtle.digest('SHA-256', buffer)
    return `${file.size}-${toHex(digest)}`
  }
  return `${file.size}-${fnv1a(new Uint8Array(buffer))}`
}

// 并发池：以 concurrency 个 worker 消费任务队列。
const runPool = async <T>(
  tasks: T[],
  concurrency: number,
  worker: (task: T) => Promise<void>,
): Promise<void> => {
  const queue = [...tasks]
  const size = Math.max(1, Math.min(concurrency, queue.length || 1))
  const runners = Array.from({ length: size }, async () => {
    for (;;) {
      const task = queue.shift()
      if (task === undefined) return
      await worker(task)
    }
  })
  await Promise.all(runners)
}

const isAbort = (error: unknown): boolean =>
  error instanceof DOMException && error.name === 'AbortError'

let idSeed = 0
const nextId = () => {
  idSeed += 1
  return `upload-${idSeed}`
}

export const useFileUpload = (getClient: () => FileClient, getTargetPath: () => string) => {
  const items = ref<UploadItem[]>([])
  const threads = ref(3)
  const uploading = ref(false)
  const controllers = new Map<string, AbortController>()

  const hasPending = computed(() => items.value.some((item) => item.status === 'waiting'))
  const hasFailed = computed(() => items.value.some((item) => item.status === 'failed'))
  const successCount = computed(() => items.value.filter((item) => item.status === 'done').length)
  const failedCount = computed(() => items.value.filter((item) => item.status === 'failed').length)

  const addFiles = (files: FileList | File[]) => {
    const list = Array.from(files)
    for (const file of list) {
      const duplicate = items.value.some(
        (item) => item.name === file.name && item.size === file.size && item.status !== 'failed',
      )
      if (duplicate) {
        ElMessage.warning(t('fileManager.duplicateFile'))
        continue
      }
      items.value.push({
        id: nextId(),
        file,
        name: file.name,
        size: file.size,
        status: 'waiting',
        statusText: t('fileManager.waitUpload'),
        progress: 0,
      })
    }
  }

  const removeItem = (id: string) => {
    controllers.get(id)?.abort()
    controllers.delete(id)
    items.value = items.value.filter((item) => item.id !== id)
  }

  const clearFinished = () => {
    items.value = items.value.filter(
      (item) => item.status !== 'done' && item.status !== 'canceled',
    )
  }

  // 每片直传到 part_urls[chunk]：来源透明（OSS=来源预签名直链，绝对 URL；本地/FTP/WebDAV=momoko 签名相对路径）。
  const putPart = async (
    partUrls: { [key: number]: string },
    chunk: number,
    partSize: number,
    item: UploadItem,
    signal: AbortSignal,
  ) => {
    const start = (chunk - 1) * partSize
    const end = Math.min(start + partSize, item.size)
    const blob = item.file.slice(start, end)
    const raw = partUrls?.[chunk]
    if (!raw) {
      throw new Error(t('fileManager.partUploadFailedHttp', { status: 0 }))
    }
    const url = /^https?:\/\//i.test(raw) ? raw : resolveStaticResourceUrl(raw)
    const res = await fetch(url, { method: 'PUT', body: blob, signal })
    if (!res.ok) {
      throw new Error(t('fileManager.partUploadFailedHttp', { status: res.status }))
    }
  }

  const uploadOne = async (item: UploadItem) => {
    const client = getClient()
    const controller = new AbortController()
    controllers.set(item.id, controller)
    const { signal } = controller

    try {
      item.error = undefined
      item.status = 'hashing'
      item.statusText = t('fileManager.hashing')
      const hash = await computeFastHash(item.file)
      if (signal.aborted) throw new DOMException('aborted', 'AbortError')

      item.status = 'uploading'
      item.statusText = t('fileManager.chunkUploading')
      const info = await client.preSignUpload({
        path: getTargetPath(),
        fileName: item.name,
        fileSize: item.size,
        hash,
      })
      item.uploadId = info.uploadId

      if (info.completed) {
        item.progress = 100
        item.status = 'done'
        item.statusText = t('fileManager.fileExistsDone')
        return
      }

      const partSize = Number(info.partSize)
      const totalParts = Number(info.totalParts)
      const done = new Set<number>(Object.keys(info.uploadedParts || {}).map(Number))
      let completed = done.size
      const updateProgress = () => {
        item.progress = totalParts ? Math.round((completed / totalParts) * 100) : 100
      }
      updateProgress()

      const pending: number[] = []
      for (let chunk = 1; chunk <= totalParts; chunk += 1) {
        if (!done.has(chunk)) pending.push(chunk)
      }

      item.statusText = t('fileManager.chunkUploadingWithThreads', { count: threads.value })
      await runPool(pending, threads.value, async (chunk) => {
        if (signal.aborted) throw new DOMException('aborted', 'AbortError')
        await putPart(info.partUrls, chunk, partSize, item, signal)
        completed += 1
        updateProgress()
      })

      if (signal.aborted) throw new DOMException('aborted', 'AbortError')

      // 校验并补传缺失分片（一轮）
      item.status = 'verifying'
      item.statusText = t('fileManager.verifying')
      const missing = await findMissingParts(info.uploadId, totalParts)
      if (missing.length) {
        await runPool(missing, threads.value, async (chunk) => {
          if (signal.aborted) throw new DOMException('aborted', 'AbortError')
          await putPart(info.partUrls, chunk, partSize, item, signal)
        })
        const stillMissing = await findMissingParts(info.uploadId, totalParts)
        if (stillMissing.length) {
          throw new Error(t('fileManager.partVerifyFailed', { count: stillMissing.length }))
        }
      }

      item.status = 'finishing'
      item.statusText = t('fileManager.finishing')
      // 收尾：FTP/WebDAV 远端整流收尾会返回异步任务，轮询至终态；本地/OSS 瞬时收尾无任务。
      const { data: completeData } = await completeFileUpload(info.uploadId)
      if (completeData?.task?.taskId) {
        await waitFileTask(completeData.task.taskId, { initialTask: completeData.task })
      }

      item.progress = 100
      item.status = 'done'
      item.statusText = t('fileManager.uploadDone')
    } catch (error) {
      if (isAbort(error) || signal.aborted) {
        item.status = 'canceled'
        item.statusText = t('fileManager.uploadCanceled')
      } else {
        item.status = 'failed'
        item.error = (error as Error).message
        item.statusText = t('fileManager.uploadFailed')
      }
    } finally {
      controllers.delete(item.id)
    }
  }

  const findMissingParts = async (uploadId: string, totalParts: number): Promise<number[]> => {
    const { data } = await getFileUploadStatus(uploadId)
    const uploaded = new Set<number>(Object.keys(data?.info?.uploadedParts || {}).map(Number))
    const missing: number[] = []
    for (let chunk = 1; chunk <= totalParts; chunk += 1) {
      if (!uploaded.has(chunk)) missing.push(chunk)
    }
    return missing
  }

  // 串行处理各文件（每个文件内部分片并发），返回是否有成功项。
  const startAll = async (): Promise<boolean> => {
    if (uploading.value) return false
    uploading.value = true
    try {
      for (const item of items.value) {
        if (item.status === 'waiting' || item.status === 'failed') {
           
          await uploadOne(item)
        }
      }
      return successCount.value > 0
    } finally {
      uploading.value = false
    }
  }

  const retryFailed = async (): Promise<boolean> => {
    items.value.forEach((item) => {
      if (item.status === 'failed') {
        item.status = 'waiting'
        item.statusText = t('fileManager.waitUpload')
        item.progress = 0
      }
    })
    return startAll()
  }

  const cancel = async (item: UploadItem) => {
    controllers.get(item.id)?.abort()
    item.status = 'canceled'
    item.statusText = t('fileManager.uploadCanceled')
    if (item.uploadId) {
      try {
        await cancelFileUpload(item.uploadId)
      } catch {
        // best-effort
      }
    }
  }

  const cancelAll = () => {
    items.value.forEach((item) => {
      if (item.status !== 'done') cancel(item)
    })
  }

  return {
    items,
    threads,
    uploading,
    hasPending,
    hasFailed,
    successCount,
    failedCount,
    addFiles,
    removeItem,
    clearFinished,
    startAll,
    retryFailed,
    cancel,
    cancelAll,
  }
}
