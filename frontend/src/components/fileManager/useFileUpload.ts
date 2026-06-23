import {
  cancelFileUploadRequest,
  completeFileUploadRequest,
  getFileUploadStatusRequest,
} from '@/api/file'
import { sha256 } from '@noble/hashes/sha2.js'
import { bytesToHex } from '@noble/hashes/utils.js'
import type {
  FileManagerUploadPreSignPayload,
  FileManagerUploadSession,
} from '@/components/fileManager/index.vue'
import { resolveStaticResourceUrl } from '@/utils/assets'
import { translate } from '@/locales'

const QUICK_HASH_SAMPLE_SIZE = 512 * 1024
const DEFAULT_UPLOAD_CONCURRENCY = 8
const PART_HASH_CONCURRENCY = 4
const COMPLETE_STATUS_POLL_INTERVAL = 600
const COMPLETE_STATUS_POLL_RETRY = 6
const VERIFY_RETRY_LIMIT = 3

type UploadPhase = 'hashing' | 'uploading' | 'verifying' | 'finishing'

interface UploadPartDescriptor {
  partNumber: number
  start: number
  end: number
  size: number
}

type UploadedPartHashMap = Record<number, string>

interface UploadRuntime {
  canceled: boolean
  uploadId: string
  activeRequests: Set<XMLHttpRequest>
  cancelPromise: Promise<void> | null
  partHashCache: Map<number, Promise<string>>
}

interface UploadFileOptions {
  path: string
  file: File
  getUploadPreSign: (
    payload: FileManagerUploadPreSignPayload,
  ) => Promise<FileManagerUploadSession | void> | void
  concurrency?: number
  onPhaseChange?: (phase: UploadPhase) => void
  onProgress?: (progress: number) => void
  onSessionCreated?: (session: FileManagerUploadSession) => void
}

interface UploadFileResult {
  completedBeforeUpload: boolean
}

export interface UploadTask {
  promise: Promise<UploadFileResult>
  cancel: () => Promise<void>
}

const sleep = (ms: number) => new Promise<void>((resolve) => window.setTimeout(resolve, ms))

const normalizeNumber = (value: unknown, fallback = 0) => {
  const nextValue = Number(value)
  return Number.isFinite(nextValue) ? nextValue : fallback
}

const normalizeConcurrency = (value: unknown, fallback = DEFAULT_UPLOAD_CONCURRENCY) => {
  const nextValue = Math.trunc(normalizeNumber(value, fallback))
  return nextValue >= 1 ? nextValue : fallback
}

const clamp = (value: number, min: number, max: number) => Math.min(Math.max(value, min), max)

const createUploadCanceledError = () => {
  const error = new Error(translate('fileManager.uploadCanceled'))
  error.name = 'UploadCanceledError'
  return error
}

export const isUploadCanceledError = (error: unknown) =>
  error instanceof Error && error.name === 'UploadCanceledError'

const buildQuickHashRanges = (size: number) => {
  if (size <= QUICK_HASH_SAMPLE_SIZE * 3) {
    return [[0, size]] as const
  }

  const middleStart = Math.max(0, Math.floor(size / 2 - QUICK_HASH_SAMPLE_SIZE / 2))
  const tailStart = Math.max(0, size - QUICK_HASH_SAMPLE_SIZE)

  return [
    [0, QUICK_HASH_SAMPLE_SIZE],
    [middleStart, middleStart + QUICK_HASH_SAMPLE_SIZE],
    [tailStart, size],
  ] as const
}

const readBlobBytes = async (blob: Blob) => new Uint8Array(await blob.arrayBuffer())

const concatUint8Arrays = (chunks: Uint8Array[]) => {
  const totalLength = chunks.reduce((sum, chunk) => sum + chunk.byteLength, 0)
  const result = new Uint8Array(totalLength)
  let offset = 0

  for (const chunk of chunks) {
    result.set(chunk, offset)
    offset += chunk.byteLength
  }

  return result
}

const computeSha256Hex = async (value: Uint8Array) => bytesToHex(sha256(value))

const computeQuickHash = async (file: File) => {
  const encoder = new TextEncoder()
  const metadata = encoder.encode(`${file.name}|${file.size}|${file.type}|${file.lastModified}`)
  const sampleBuffers = await Promise.all(
    buildQuickHashRanges(file.size).map(async ([start, end]) =>
      readBlobBytes(file.slice(start, end)),
    ),
  )

  return computeSha256Hex(concatUint8Arrays([metadata, ...sampleBuffers]))
}

const mapWithConcurrency = async <T, R>(
  items: T[],
  concurrency: number,
  worker: (item: T, index: number) => Promise<R>,
) => {
  if (!items.length) {
    return [] as R[]
  }

  const results = new Array<R>(items.length)
  let cursor = 0
  const workerCount = Math.max(1, Math.min(concurrency, items.length))

  await Promise.all(
    Array.from({ length: workerCount }, async () => {
      while (cursor < items.length) {
        const currentIndex = cursor
        cursor += 1
        results[currentIndex] = await worker(items[currentIndex]!, currentIndex)
      }
    }),
  )

  return results
}

const normalizeUploadedParts = (value: unknown): UploadedPartHashMap => {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    return {}
  }

  const normalized: UploadedPartHashMap = {}

  for (const [key, hash] of Object.entries(value as Record<string, unknown>)) {
    const partNumber = normalizeNumber(key, 0)
    if (partNumber <= 0 || typeof hash !== 'string' || !hash.trim()) {
      continue
    }

    normalized[partNumber] = hash.trim().toLowerCase()
  }

  return normalized
}

const normalizeUploadSession = (value: FileManagerUploadSession | void) => {
  if (!value?.uploadId || !value.uploadPartUrlPathTemplate) {
    throw new Error(translate('fileManager.invalidUploadSession'))
  }

  const partSize = normalizeNumber(value.partSize, 0)
  const fileSize = normalizeNumber(value.fileSize, 0)
  const totalParts = normalizeNumber(value.totalParts, 0)

  if (partSize <= 0 || totalParts <= 0) {
    throw new Error(translate('fileManager.invalidPartInfo'))
  }

  return {
    uploadId: value.uploadId,
    uploadPartUrlPathTemplate: value.uploadPartUrlPathTemplate,
    partSize,
    fileSize,
    totalParts,
    uploadedParts: normalizeUploadedParts(value.uploadedParts),
    completed: Boolean(value.completed),
    cancel: Boolean(value.cancel),
    expiredAt: value.expiredAt,
  }
}

const buildUploadParts = (fileSize: number, partSize: number, totalParts: number) => {
  const parts: UploadPartDescriptor[] = []

  for (let partNumber = 1; partNumber <= totalParts; partNumber += 1) {
    const start = (partNumber - 1) * partSize
    const end = Math.min(fileSize, start + partSize)

    if (start >= end) continue

    parts.push({
      partNumber,
      start,
      end,
      size: end - start,
    })
  }

  return parts
}

const resolveUploadPartUrl = (template: string, partNumber: number) => {
  const resolvedTemplate = resolveStaticResourceUrl(template)
  return resolvedTemplate.replace(/\{partNumber\}/g, String(partNumber))
}

const abortActiveRequests = (runtime: UploadRuntime) => {
  runtime.activeRequests.forEach((xhr) => {
    try {
      xhr.abort()
    } catch {
      // noop
    }
  })
}

const requestRemoteCancel = async (runtime: UploadRuntime) => {
  if (!runtime.uploadId) {
    return
  }

  if (!runtime.cancelPromise) {
    runtime.cancelPromise = cancelFileUploadRequest({ uploadId: runtime.uploadId }).then(
      () => undefined,
    )
  }

  return runtime.cancelPromise
}

const ensureNotCanceled = async (runtime: UploadRuntime) => {
  if (!runtime.canceled) {
    return
  }

  await requestRemoteCancel(runtime)
  throw createUploadCanceledError()
}

const uploadPart = (
  url: string,
  blob: Blob,
  runtime: UploadRuntime,
  onProgress?: (loaded: number) => void,
) =>
  new Promise<void>((resolve, reject) => {
    if (runtime.canceled) {
      reject(createUploadCanceledError())
      return
    }

    const xhr = new XMLHttpRequest()
    runtime.activeRequests.add(xhr)

    const cleanup = () => {
      runtime.activeRequests.delete(xhr)
    }

    xhr.open('PUT', url, true)
    xhr.withCredentials = false

    xhr.upload.onprogress = (event) => {
      onProgress?.(event.lengthComputable ? event.loaded : blob.size)
    }

    xhr.onload = () => {
      cleanup()

      if (xhr.status >= 200 && xhr.status < 300) {
        resolve()
        return
      }

      reject(new Error(translate('fileManager.partUploadFailedHttp', { status: xhr.status || 0 })))
    }

    xhr.onerror = () => {
      cleanup()
      reject(new Error(translate('fileManager.partUploadNetworkFailed')))
    }

    xhr.onabort = () => {
      cleanup()
      reject(runtime.canceled ? createUploadCanceledError() : new Error(translate('fileManager.partUploadAborted')))
    }

    xhr.send(blob)
  })

const pollUploadCompletion = async (uploadId: string) => {
  let latestStatus: ReturnType<typeof normalizeUploadSession> | null = null

  for (let attempt = 0; attempt < COMPLETE_STATUS_POLL_RETRY; attempt += 1) {
    try {
      const { data } = await getFileUploadStatusRequest({ uploadId })
      if (!data.info) {
        return latestStatus
      }

      latestStatus = normalizeUploadSession(data.info)
      if (latestStatus.completed || latestStatus.cancel) {
        return latestStatus
      }
    } catch {
      return latestStatus
    }

    await sleep(COMPLETE_STATUS_POLL_INTERVAL)
  }

  return latestStatus
}

const getRemotePartHash = (uploadedParts: UploadedPartHashMap, partNumber: number) => {
  return uploadedParts[partNumber]?.trim().toLowerCase() || ''
}

export const useFileUpload = () => {
  const cancelUploadSession = async (uploadId: string) => {
    if (!uploadId) {
      return
    }

    await cancelFileUploadRequest({ uploadId })
  }

  const createUploadTask = ({
    path,
    file,
    getUploadPreSign,
    concurrency = DEFAULT_UPLOAD_CONCURRENCY,
    onPhaseChange,
    onProgress,
    onSessionCreated,
  }: UploadFileOptions): UploadTask => {
    const runtime: UploadRuntime = {
      canceled: false,
      uploadId: '',
      activeRequests: new Set(),
      cancelPromise: null,
      partHashCache: new Map(),
    }
    const uploadConcurrency = normalizeConcurrency(concurrency)

    const getPartHash = (part: UploadPartDescriptor) => {
      const cachedHash = runtime.partHashCache.get(part.partNumber)
      if (cachedHash) {
        return cachedHash
      }

      const nextHashPromise = (async () => {
        const partBytes = await readBlobBytes(file.slice(part.start, part.end))
        return computeSha256Hex(partBytes)
      })()

      runtime.partHashCache.set(part.partNumber, nextHashPromise)
      return nextHashPromise
    }

    const partMap = new Map<number, UploadPartDescriptor>()
    const confirmedPartNumbers = new Set<number>()
    const progressByPart = new Map<number, number>()
    let totalBytes = Math.max(file.size, 1)

    const getConfirmedBytes = () => {
      let bytes = 0
      confirmedPartNumbers.forEach((partNumber) => {
        bytes += partMap.get(partNumber)?.size || 0
      })
      return bytes
    }

    const reportProgress = () => {
      const uploadingBytes = Array.from(progressByPart.entries()).reduce(
        (sum, [partNumber, loaded]) => {
          if (confirmedPartNumbers.has(partNumber)) {
            return sum
          }

          return sum + loaded
        },
        0,
      )
      const progress = Math.round(
        clamp(((getConfirmedBytes() + uploadingBytes) / totalBytes) * 100, 0, 99),
      )
      onProgress?.(progress)
    }

    const resolveMatchedPartNumbers = async (
      parts: UploadPartDescriptor[],
      uploadedParts: UploadedPartHashMap,
    ) => {
      const results = await mapWithConcurrency(parts, PART_HASH_CONCURRENCY, async (part) => {
        const remoteHash = getRemotePartHash(uploadedParts, part.partNumber)
        if (!remoteHash) {
          return {
            part,
            matched: false,
          }
        }

        const localHash = await getPartHash(part)
        return {
          part,
          matched: localHash === remoteHash,
        }
      })

      return {
        matchedPartNumbers: new Set(
          results.filter((item) => item.matched).map((item) => item.part.partNumber),
        ),
        mismatchedParts: results.filter((item) => !item.matched).map((item) => item.part),
      }
    }

    const uploadParts = async (
      parts: UploadPartDescriptor[],
      session: ReturnType<typeof normalizeUploadSession>,
    ) => {
      if (!parts.length) {
        return
      }

      onPhaseChange?.('uploading')
      const workerCount = Math.max(1, Math.min(uploadConcurrency, parts.length))
      let cursor = 0

      const worker = async () => {
        while (cursor < parts.length) {
          await ensureNotCanceled(runtime)

          const part = parts[cursor]
          cursor += 1

          if (!part) {
            return
          }

          confirmedPartNumbers.delete(part.partNumber)
          progressByPart.set(part.partNumber, 0)
          reportProgress()

          try {
            await uploadPart(
              resolveUploadPartUrl(session.uploadPartUrlPathTemplate, part.partNumber),
              file.slice(part.start, part.end),
              runtime,
              (loaded) => {
                progressByPart.set(part.partNumber, clamp(loaded, 0, part.size))
                reportProgress()
              },
            )

            confirmedPartNumbers.add(part.partNumber)
          } catch (error) {
            if (isUploadCanceledError(error)) {
              throw error
            }
          } finally {
            progressByPart.delete(part.partNumber)
            reportProgress()
          }
        }
      }

      await Promise.all(Array.from({ length: workerCount }, () => worker()))
    }

    const verifyUploadedParts = async (parts: UploadPartDescriptor[]) => {
      onPhaseChange?.('verifying')
      const { data } = await getFileUploadStatusRequest({ uploadId: runtime.uploadId })
      const latestStatus = normalizeUploadSession(data.info)

      if (latestStatus.cancel) {
        throw new Error(translate('fileManager.sessionCanceled'))
      }

      const { matchedPartNumbers, mismatchedParts } = await resolveMatchedPartNumbers(
        parts,
        latestStatus.uploadedParts,
      )

      confirmedPartNumbers.clear()
      matchedPartNumbers.forEach((partNumber) => {
        confirmedPartNumbers.add(partNumber)
      })
      reportProgress()

      return {
        latestStatus,
        mismatchedParts,
      }
    }

    const cancel = async () => {
      if (runtime.canceled) {
        await requestRemoteCancel(runtime)
        return
      }

      runtime.canceled = true
      abortActiveRequests(runtime)
      await requestRemoteCancel(runtime)
    }

    const promise = (async (): Promise<UploadFileResult> => {
      try {
        onPhaseChange?.('hashing')
        onProgress?.(0)

        const fileHash = await computeQuickHash(file)
        await ensureNotCanceled(runtime)

        const session = normalizeUploadSession(
          await getUploadPreSign({
            path,
            file,
            fileName: file.name,
            fileSize: file.size,
            hash: fileHash,
          }),
        )

        runtime.uploadId = session.uploadId
        onSessionCreated?.(session)

        if (session.cancel) {
          throw new Error(translate('fileManager.sessionCanceled'))
        }

        if (session.fileSize > 0 && session.fileSize !== file.size) {
          throw new Error(translate('fileManager.fileSizeMismatch'))
        }

        if (session.completed) {
          onProgress?.(100)
          return { completedBeforeUpload: true }
        }

        const allParts = buildUploadParts(file.size, session.partSize, session.totalParts)
        allParts.forEach((part) => {
          partMap.set(part.partNumber, part)
        })
        totalBytes = Math.max(
          allParts.reduce((sum, part) => sum + part.size, 0),
          1,
        )

        const initialComparison = await resolveMatchedPartNumbers(allParts, session.uploadedParts)
        initialComparison.matchedPartNumbers.forEach((partNumber) => {
          confirmedPartNumbers.add(partNumber)
        })
        reportProgress()

        let retryCount = 0
        let pendingParts = initialComparison.mismatchedParts

        while (true) {
          await ensureNotCanceled(runtime)
          await uploadParts(pendingParts, session)
          await ensureNotCanceled(runtime)

          const verification = await verifyUploadedParts(allParts)
          await ensureNotCanceled(runtime)

          if (!verification.mismatchedParts.length) {
            break
          }

          retryCount += 1
          if (retryCount >= VERIFY_RETRY_LIMIT) {
            throw new Error(
              translate('fileManager.partVerifyFailed', {
                count: verification.mismatchedParts.length,
              }),
            )
          }

          pendingParts = verification.mismatchedParts
        }

        await ensureNotCanceled(runtime)
        onPhaseChange?.('finishing')
        await completeFileUploadRequest({ uploadId: session.uploadId })
        await ensureNotCanceled(runtime)

        const latestStatus = await pollUploadCompletion(session.uploadId)
        if (latestStatus?.cancel) {
          throw new Error(translate('fileManager.sessionCanceled'))
        }

        onProgress?.(100)
        return { completedBeforeUpload: false }
      } catch (error) {
        if (runtime.canceled) {
          await requestRemoteCancel(runtime)
          throw createUploadCanceledError()
        }

        throw error
      } finally {
        abortActiveRequests(runtime)
      }
    })()

    return {
      promise,
      cancel,
    }
  }

  const uploadFile = async (options: UploadFileOptions) => {
    const task = createUploadTask(options)
    return task.promise
  }

  return {
    uploadFile,
    createUploadTask,
    cancelUploadSession,
  }
}
