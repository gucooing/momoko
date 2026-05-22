import { resolveStaticResourceUrl } from '@/utils/assets'

export const imageExtensions = new Set([
  '.apng', '.avif', '.bmp', '.gif', '.ico', '.jpeg', '.jpg', '.png', '.svg', '.tif', '.tiff', '.webp',
])

export const audioExtensions = new Set(['.aac', '.flac', '.m4a', '.mp3', '.oga', '.ogg', '.opus', '.wav'])
export const videoExtensions = new Set(['.m4v', '.mkv', '.mov', '.mp4', '.ogv', '.webm'])

export type LocalFilePreviewKind = 'image' | 'audio' | 'video'

export interface LocalFileMediaOpenResult {
  kind: 'media'
  mediaKind: LocalFilePreviewKind
  fileName: string
  filePath: string
  objectUrl: string
}

export interface LocalFileTextOpenResult {
  kind: 'text'
  fileName: string
  filePath: string
  content: string
}

export type LocalFileOpenResult = LocalFileMediaOpenResult | LocalFileTextOpenResult

export const normalizeKeywords = (value: unknown) =>
  typeof value === 'string' && value.trim() ? value.trim() : undefined

export const normalizeIncludeSubDir = (keywords: string | undefined, value: unknown) =>
  keywords ? Boolean(value) : undefined

export const normalizeNumber = (value: unknown, fallback: number) => {
  const nextValue = Number(value)
  return Number.isFinite(nextValue) ? nextValue : fallback
}

export const hasOwnQueryKey = <T extends object>(target: T, key: keyof T) =>
  Object.prototype.hasOwnProperty.call(target, key)

export const decodeBase64ToBytes = (value: string) => {
  try {
    const binary = window.atob(value)
    return Uint8Array.from(binary, (char) => char.charCodeAt(0))
  } catch {
    return null
  }
}

export const toUint8Array = (value: unknown) => {
  if (value instanceof Uint8Array) return value
  if (Array.isArray(value)) return Uint8Array.from(value)
  if (typeof value === 'string') {
    return decodeBase64ToBytes(value) || new TextEncoder().encode(value)
  }
  return new TextEncoder().encode(String(value ?? ''))
}

export const decodeFileContent = (value: unknown) => {
  return new TextDecoder().decode(toUint8Array(value))
}

export const getFileExtension = (path: string) => path.toLowerCase().match(/\.[^./\\]+$/)?.[0] || ''

export const resolveFilePreviewKind = (path: string): LocalFilePreviewKind | null => {
  const extension = getFileExtension(path)
  if (imageExtensions.has(extension)) return 'image'
  if (audioExtensions.has(extension)) return 'audio'
  if (videoExtensions.has(extension)) return 'video'
  return null
}

export const isBlobUrl = (value: string) => value.startsWith('blob:')

export const resolvePreSignedFileUrl = (value?: string | null) => {
  return resolveStaticResourceUrl(value)
}

export const triggerFileDownload = (blobUrl: string, fileName: string) => {
  const link = document.createElement('a')
  link.href = blobUrl
  link.download = fileName
  link.rel = 'noopener'
  link.style.display = 'none'

  document.body.appendChild(link)
  link.click()
  link.remove()
}

export const downloadFileFromUrl = async (url: string, fileName: string) => {
  const response = await fetch(url)

  if (!response.ok) {
    throw new Error(`下载失败：${response.status}`)
  }

  const blob = await response.blob()
  const blobUrl = URL.createObjectURL(blob)

  try {
    triggerFileDownload(blobUrl, fileName)
  } finally {
    window.setTimeout(() => {
      URL.revokeObjectURL(blobUrl)
    }, 1000)
  }
}

export const copyTextToClipboard = async (value: string) => {
  if (navigator.clipboard?.writeText) {
    await navigator.clipboard.writeText(value)
    return
  }

  const textarea = document.createElement('textarea')
  textarea.value = value
  textarea.setAttribute('readonly', 'true')
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  textarea.style.pointerEvents = 'none'

  document.body.appendChild(textarea)
  textarea.select()

  try {
    document.execCommand('copy')
  } finally {
    textarea.remove()
  }
}

export const joinPath = (root: string, segment: string) => {
  const separator = root.includes('\\') ? '\\' : '/'
  return `${root}${root.endsWith(separator) ? '' : separator}${segment}`
}
