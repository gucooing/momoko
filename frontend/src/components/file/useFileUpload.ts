import { ref, computed } from 'vue'
import { translate as t } from '@/locales'
import { resolveStaticResourceUrl } from '@/utils/assets'
import { withNodeQuery } from '@/utils/nodeContext'
import { cancelFileUpload, completeFileUpload, getFileUploadStatus } from '@/api/file'
import { waitFileTask } from '@/utils/fileTask'
import { useFeedback } from '@/utils/feedback'
import type { FileClient } from './types'

// 分片并发上传队列：scope 无关。预签名走 client（系统/实例各自路由），分片 PUT 直传签名 URL
// （/api/v1/upload/pre 对 PUT 免 JWT，sign 自校验），完成/取消/状态走系统级共享会话端点。
// 加入队列即开始上传（逐个文件串行、文件内分片并发）；可暂停（中止在途分片，服务端会话保留）后断点续传。

export type UploadStatus =
  | 'waiting'
  | 'hashing'
  | 'uploading'
  | 'verifying'
  | 'finishing'
  | 'paused'
  | 'done'
  | 'failed'
  | 'canceled'

// 分片状态：0=未传 1=传输中 2=已完成
export type PartState = 0 | 1 | 2

export interface UploadItem {
  id: string
  file: File
  name: string
  size: number
  // 加入队列时冻结的目标目录 / 来源，排队期间切换目录或来源不影响它
  targetPath: string
  sourceId: string
  status: UploadStatus
  statusText: string
  progress: number
  uploadId?: string
  error?: string
  partSize: number
  totalParts: number
  parts: PartState[]
  uploadedBytes: number
  // 当前速度（字节/秒）与预计剩余（秒），仅上传中有意义
  speed: number
  eta: number
}

export interface UploadOptions {
  // 系统级来源 id（加入队列时读取）
  getSourceId?: () => string
  // 单个文件上传完成（含秒传）
  onUploaded?: (item: UploadItem) => void
}

const FAST_HASH_SAMPLE = 2 * 1024 * 1024 // 头尾各采样 2MB 做快速指纹
const ACTIVE: UploadStatus[] = ['hashing', 'uploading', 'verifying', 'finishing']

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
const abortError = () => new DOMException('aborted', 'AbortError')

let idSeed = 0
const nextId = () => {
  idSeed += 1
  return `upload-${idSeed}`
}

export const useFileUpload = (
  getClient: () => FileClient,
  getTargetPath: () => string,
  options: UploadOptions = {},
) => {
  const fb = useFeedback()
  const items = ref<UploadItem[]>([])
  const threads = ref(3)
  const uploading = ref(false)
  const controllers = new Map<string, AbortController>()
  // 主动暂停 / 取消的会话：中止信号到达时据此区分最终状态
  const pausing = new Set<string>()

  const hasPending = computed(() => items.value.some((item) => item.status === 'waiting'))
  const hasFailed = computed(() => items.value.some((item) => item.status === 'failed'))
  const successCount = computed(() => items.value.filter((item) => item.status === 'done').length)
  const failedCount = computed(() => items.value.filter((item) => item.status === 'failed').length)
  // 当前展示的主条目：进行中 > 已暂停 > 排队中
  const current = computed<UploadItem | undefined>(
    () =>
      items.value.find((item) => ACTIVE.includes(item.status)) ??
      items.value.find((item) => item.status === 'paused') ??
      items.value.find((item) => item.status === 'waiting'),
  )
  // 还没结束的条目（进行中 / 排队 / 暂停）
  const unfinishedCount = computed(
    () => items.value.filter((item) => ACTIVE.includes(item.status) || item.status === 'waiting' || item.status === 'paused').length,
  )
  // 整个队列的字节进度（0-100）
  const overallProgress = computed(() => {
    const list = items.value.filter((item) => item.status !== 'canceled')
    const total = list.reduce((sum, item) => sum + item.size, 0)
    if (!total) return list.length && list.every((item) => item.status === 'done') ? 100 : 0
    const done = list.reduce((sum, item) => sum + (item.status === 'done' ? item.size : item.uploadedBytes), 0)
    return Math.min(100, Math.round((done / total) * 100))
  })

  const setStatus = (item: UploadItem, status: UploadStatus, text: string) => {
    item.status = status
    item.statusText = text
  }

  const addFiles = (files: FileList | File[]) => {
    const list = Array.from(files)
    const targetPath = getTargetPath()
    const sourceId = options.getSourceId?.() ?? ''
    let added = 0
    for (const file of list) {
      const duplicate = items.value.some(
        (item) =>
          item.name === file.name &&
          item.size === file.size &&
          item.targetPath === targetPath &&
          item.status !== 'failed' &&
          item.status !== 'canceled',
      )
      if (duplicate) {
        fb.warning(t('fileManager.duplicateFile'))
        continue
      }
      items.value.push({
        id: nextId(),
        file,
        name: file.name,
        size: file.size,
        targetPath,
        sourceId,
        status: 'waiting',
        statusText: t('fileManager.waitUpload'),
        progress: 0,
        partSize: 0,
        totalParts: 0,
        parts: [],
        uploadedBytes: 0,
        speed: 0,
        eta: 0,
      })
      added += 1
    }
    if (added) void startAll()
  }

  const removeItem = (id: string) => {
    controllers.get(id)?.abort()
    controllers.delete(id)
    items.value = items.value.filter((item) => item.id !== id)
  }

  const clearFinished = () => {
    items.value = items.value.filter((item) => item.status !== 'done' && item.status !== 'canceled')
  }

  const partBytes = (item: UploadItem, chunk: number) => {
    const start = (chunk - 1) * item.partSize
    return Math.max(0, Math.min(item.partSize, item.size - start))
  }

  // 每片直传到 part_urls[chunk]：来源透明（OSS=来源预签名直链，绝对 URL；本地/FTP/WebDAV=momoko 签名相对路径）。
  const putPart = async (
    partUrls: { [key: number]: string },
    chunk: number,
    item: UploadItem,
    signal: AbortSignal,
  ) => {
    const start = (chunk - 1) * item.partSize
    const end = Math.min(start + item.partSize, item.size)
    const blob = item.file.slice(start, end)
    const raw = partUrls?.[chunk]
    if (!raw) {
      throw new Error(t('fileManager.partUploadFailedHttp', { status: 0 }))
    }
    const url = /^https?:\/\//i.test(raw) ? raw : withNodeQuery(resolveStaticResourceUrl(raw), raw)
    const res = await fetch(url, { method: 'PUT', body: blob, signal })
    if (!res.ok) {
      throw new Error(t('fileManager.partUploadFailedHttp', { status: res.status }))
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

  const uploadOne = async (item: UploadItem) => {
    const client = getClient()
    const controller = new AbortController()
    controllers.set(item.id, controller)
    const { signal } = controller

    try {
      item.error = undefined
      setStatus(item, 'hashing', t('fileManager.hashing'))
      const hash = await computeFastHash(item.file)
      if (signal.aborted) throw abortError()

      setStatus(item, 'uploading', t('fileManager.chunkUploading'))
      const info = await client.preSignUpload({
        path: item.targetPath,
        fileName: item.name,
        fileSize: item.size,
        hash,
        sourceId: item.sourceId,
      })
      item.uploadId = info.uploadId

      if (info.completed) {
        item.progress = 100
        item.uploadedBytes = item.size
        setStatus(item, 'done', t('fileManager.fileExistsDone'))
        options.onUploaded?.(item)
        return
      }

      item.partSize = Number(info.partSize) || 0
      item.totalParts = Number(info.totalParts) || 0
      const done = new Set<number>(Object.keys(info.uploadedParts || {}).map(Number))
      item.parts = Array.from({ length: item.totalParts }, (_, i) => (done.has(i + 1) ? 2 : 0))
      let completed = done.size
      item.uploadedBytes = [...done].reduce((sum, chunk) => sum + partBytes(item, chunk), 0)
      // 速度以本轮（含续传）起点计：已续传的字节不计入速度
      const startedAt = performance.now()
      const startedBytes = item.uploadedBytes
      const refresh = () => {
        item.progress = item.totalParts ? Math.round((completed / item.totalParts) * 100) : 100
        const elapsed = (performance.now() - startedAt) / 1000
        const sent = item.uploadedBytes - startedBytes
        item.speed = elapsed > 0.3 ? sent / elapsed : 0
        item.eta = item.speed > 0 ? (item.size - item.uploadedBytes) / item.speed : 0
      }
      refresh()

      const sendPart = async (chunk: number) => {
        if (signal.aborted) throw abortError()
        item.parts[chunk - 1] = 1
        try {
          await putPart(info.partUrls, chunk, item, signal)
        } catch (error) {
          item.parts[chunk - 1] = 0
          throw error
        }
        item.parts[chunk - 1] = 2
        completed += 1
        item.uploadedBytes += partBytes(item, chunk)
        refresh()
      }

      const pending: number[] = []
      for (let chunk = 1; chunk <= item.totalParts; chunk += 1) {
        if (!done.has(chunk)) pending.push(chunk)
      }
      item.statusText = t('fileManager.chunkUploadingWithThreads', { count: threads.value })
      await runPool(pending, threads.value, sendPart)
      if (signal.aborted) throw abortError()

      // 校验并补传缺失分片（一轮）
      setStatus(item, 'verifying', t('fileManager.verifying'))
      const missing = await findMissingParts(info.uploadId, item.totalParts)
      if (missing.length) {
        completed -= missing.length
        for (const chunk of missing) {
          item.parts[chunk - 1] = 0
          item.uploadedBytes -= partBytes(item, chunk)
        }
        await runPool(missing, threads.value, sendPart)
        const stillMissing = await findMissingParts(info.uploadId, item.totalParts)
        if (stillMissing.length) {
          throw new Error(t('fileManager.partVerifyFailed', { count: stillMissing.length }))
        }
      }

      setStatus(item, 'finishing', t('fileManager.finishing'))
      // 收尾：FTP/WebDAV 远端整流收尾会返回异步任务，轮询至终态；本地/OSS 瞬时收尾无任务。
      const { data: completeData } = await completeFileUpload(info.uploadId)
      if (completeData?.task?.taskId) {
        await waitFileTask(completeData.task.taskId, { initialTask: completeData.task })
      }

      item.progress = 100
      item.speed = 0
      item.eta = 0
      setStatus(item, 'done', t('fileManager.uploadDone'))
      options.onUploaded?.(item)
    } catch (error) {
      item.speed = 0
      item.eta = 0
      if (pausing.has(item.id)) {
        setStatus(item, 'paused', t('fileManager.uploadPaused'))
      } else if (isAbort(error) || signal.aborted) {
        setStatus(item, 'canceled', t('fileManager.uploadCanceled'))
      } else {
        item.error = (error as Error).message
        setStatus(item, 'failed', t('fileManager.uploadFailed'))
      }
      // 中止后在途分片回到未传
      item.parts = item.parts.map((p) => (p === 1 ? 0 : p))
    } finally {
      pausing.delete(item.id)
      controllers.delete(item.id)
    }
  }

  // 逐个消费排队中的文件（每个文件内部分片并发）；队列里随时加入/续传的条目也会被接着处理。
  const startAll = async (): Promise<boolean> => {
    if (uploading.value) return false
    uploading.value = true
    try {
      for (;;) {
        const next = items.value.find((item) => item.status === 'waiting')
        if (!next) break
        await uploadOne(next)
      }
      return successCount.value > 0
    } finally {
      uploading.value = false
    }
  }

  const retryFailed = async (): Promise<boolean> => {
    items.value.forEach((item) => {
      if (item.status === 'failed') {
        setStatus(item, 'waiting', t('fileManager.waitUpload'))
        item.progress = 0
      }
    })
    return startAll()
  }

  const retry = (item: UploadItem) => {
    if (item.status !== 'failed' && item.status !== 'canceled') return
    setStatus(item, 'waiting', t('fileManager.waitUpload'))
    void startAll()
  }

  // 暂停：中止在途分片，保留服务端会话（续传时按已传分片补齐）。排队中的条目直接挂起。
  const pause = (item: UploadItem) => {
    if (item.status === 'waiting') {
      setStatus(item, 'paused', t('fileManager.uploadPaused'))
      return
    }
    if (!ACTIVE.includes(item.status) || item.status === 'finishing') return
    pausing.add(item.id)
    controllers.get(item.id)?.abort()
  }

  const resume = (item: UploadItem) => {
    if (item.status !== 'paused') return
    setStatus(item, 'waiting', t('fileManager.waitUpload'))
    void startAll()
  }

  const cancel = async (item: UploadItem) => {
    controllers.get(item.id)?.abort()
    setStatus(item, 'canceled', t('fileManager.uploadCanceled'))
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
      if (item.status !== 'done' && item.status !== 'canceled') cancel(item)
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
    current,
    unfinishedCount,
    overallProgress,
    addFiles,
    removeItem,
    clearFinished,
    startAll,
    retryFailed,
    retry,
    pause,
    resume,
    cancel,
    cancelAll,
  }
}

export type FileUploadQueue = ReturnType<typeof useFileUpload>
