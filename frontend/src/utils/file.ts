import dayjs from 'dayjs'
import { resolveStaticResourceUrl } from '@/utils/assets'

// 文件管理通用工具：路径、媒体判定、格式化、下载、剪贴板、Monaco 语言识别。
// 与具体 scope（系统/实例）无关，纯函数，便于复用与测试。

const IMAGE_EXTENSIONS = new Set(['png', 'jpg', 'jpeg', 'gif', 'webp', 'bmp', 'svg', 'ico', 'avif'])
const VIDEO_EXTENSIONS = new Set(['mp4', 'webm', 'ogv', 'mov', 'mkv', 'avi', 'm4v'])
const AUDIO_EXTENSIONS = new Set(['mp3', 'wav', 'ogg', 'flac', 'aac', 'm4a'])

export type FilePreviewKind = 'image' | 'video' | 'audio'

// 取小写扩展名（不含点）；无扩展名返回空串。
export const getFileExtension = (name: string): string => {
  const base = name.split(/[\\/]/).pop() || name
  const dot = base.lastIndexOf('.')
  if (dot <= 0 || dot === base.length - 1) return ''
  return base.slice(dot + 1).toLowerCase()
}

// 按扩展名判断媒体类型，非媒体返回 null（媒体走 pre-sign 预览，编辑器对其给遮罩）。
export const resolveFilePreviewKind = (path: string): FilePreviewKind | null => {
  const ext = getFileExtension(path)
  if (!ext) return null
  if (IMAGE_EXTENSIONS.has(ext)) return 'image'
  if (VIDEO_EXTENSIONS.has(ext)) return 'video'
  if (AUDIO_EXTENSIONS.has(ext)) return 'audio'
  return null
}

export const isMediaFile = (path: string): boolean => resolveFilePreviewKind(path) !== null

export const MAX_EDITOR_FILE_SIZE = 5 * 1024 * 1024

export const isFileTooLargeForEditor = (
  path: string,
  size: number | string | undefined | null,
): boolean => {
  const bytes = Number(size)
  return !isMediaFile(path) && Number.isFinite(bytes) && bytes > MAX_EDITOR_FILE_SIZE
}

// 检测路径分隔符：含反斜杠视为 Windows 风格。
export const getPathSeparator = (path: string): '\\' | '/' => (path.includes('\\') ? '\\' : '/')

// 拼接目录与子项名，沿用根路径的分隔符风格。
export const joinPath = (root: string, segment: string): string => {
  if (!root) return segment
  const sep = getPathSeparator(root)
  const trimmed = root.replace(/[\\/]+$/, '')
  return `${trimmed}${sep}${segment}`
}

// 取末段名称（文件/目录名）。
export const getBaseName = (path: string): string => {
  const cleaned = path.replace(/[\\/]+$/, '')
  const parts = cleaned.split(/[\\/]/)
  return parts[parts.length - 1] || cleaned
}

// 取父级目录路径；盘符根（如 "D:\"）的上级是虚拟根 ""（此电脑），虚拟根再无上级。
export const getParentPath = (path: string): string => {
  if (!path) return ''
  const sep = getPathSeparator(path)
  const cleaned = path.replace(/[\\/]+$/, '')
  // 盘符根（"D:" / "D:\"）→ 虚拟根（此电脑）
  if (/^[a-zA-Z]:$/.test(cleaned)) return ''
  const idx = Math.max(cleaned.lastIndexOf('\\'), cleaned.lastIndexOf('/'))
  if (idx < 0) return ''
  // 保留 POSIX 根（/x → /）
  if (idx === 0) return sep
  const parent = cleaned.slice(0, idx)
  // 保留 Windows 盘符根（C:\）
  return /^[a-zA-Z]:$/.test(parent) ? `${parent}${sep}` : parent
}

// 把路径拆成面包屑段：[{ name, path }]，含根。
export interface PathSegment {
  name: string
  path: string
}

export const splitPathSegments = (path: string): PathSegment[] => {
  if (!path) return []
  const sep = getPathSeparator(path)
  const isPosixRoot = path.startsWith('/')
  const cleaned = path.replace(/[\\/]+$/, '')
  const rawParts = cleaned.split(/[\\/]/).filter((part, index) => part !== '' || index === 0)

  const segments: PathSegment[] = []
  let accumulated = ''
  rawParts.forEach((part, index) => {
    if (index === 0) {
      if (isPosixRoot) {
        segments.push({ name: sep, path: sep })
        accumulated = sep
        if (part !== '') {
          accumulated = `${sep}${part}`
          segments.push({ name: part, path: accumulated })
        }
        return
      }
      // Windows 盘符根
      accumulated = /^[a-zA-Z]:$/.test(part) ? `${part}${sep}` : part
      segments.push({ name: part || sep, path: accumulated })
      return
    }
    accumulated = joinPath(accumulated, part)
    segments.push({ name: part, path: accumulated })
  })

  return segments
}

const SIZE_UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']

// 人类可读大小；0/负数返回 '0 B'。
export const formatFileSize = (bytes: number | string | undefined | null): string => {
  const value = Number(bytes)
  if (!Number.isFinite(value) || value <= 0) return '0 B'
  const exponent = Math.min(Math.floor(Math.log(value) / Math.log(1024)), SIZE_UNITS.length - 1)
  const size = value / 1024 ** exponent
  const fixed = exponent === 0 ? size.toFixed(0) : size.toFixed(2)
  return `${fixed} ${SIZE_UNITS[exponent]}`
}

// 统一时间格式（接受 ISO 字符串 / Date / 时间戳）；无效返回 '-'。
export const formatDateTime = (value: unknown): string => {
  if (!value) return '-'
  const parsed = dayjs(value as string | number | Date)
  return parsed.isValid() ? parsed.format('YYYY-MM-DD HH:mm:ss') : '-'
}

// pre-sign 下载路径 → 可访问 URL。
export const resolvePreSignedFileUrl = (downloadUrlPath?: string | null): string =>
  resolveStaticResourceUrl(downloadUrlPath)

// 触发浏览器下载（通过临时 a[download]）。
export const downloadFileFromUrl = (url: string, fileName?: string): void => {
  const anchor = document.createElement('a')
  anchor.href = url
  if (fileName) anchor.download = fileName
  anchor.rel = 'noopener'
  anchor.style.display = 'none'
  document.body.appendChild(anchor)
  anchor.click()
  document.body.removeChild(anchor)
}

// 复制文本到剪贴板，降级到 execCommand。
export const copyTextToClipboard = async (text: string): Promise<void> => {
  try {
    if (navigator.clipboard?.writeText) {
      await navigator.clipboard.writeText(text)
      return
    }
  } catch {
    // 降级
  }
  const textarea = document.createElement('textarea')
  textarea.value = text
  textarea.style.position = 'fixed'
  textarea.style.opacity = '0'
  document.body.appendChild(textarea)
  textarea.select()
  try {
    document.execCommand('copy')
  } finally {
    document.body.removeChild(textarea)
  }
}

// 扩展名/文件名 → Monaco 语言 id（识别失败回退 plaintext）。
const LANGUAGE_BY_EXTENSION: Record<string, string> = {
  js: 'javascript',
  mjs: 'javascript',
  cjs: 'javascript',
  jsx: 'javascript',
  ts: 'typescript',
  mts: 'typescript',
  cts: 'typescript',
  tsx: 'typescript',
  json: 'json',
  json5: 'json',
  jsonc: 'json',
  html: 'html',
  htm: 'html',
  vue: 'html',
  xml: 'xml',
  svg: 'xml',
  css: 'css',
  scss: 'scss',
  less: 'less',
  md: 'markdown',
  markdown: 'markdown',
  yaml: 'yaml',
  yml: 'yaml',
  toml: 'ini',
  ini: 'ini',
  conf: 'ini',
  cfg: 'ini',
  env: 'ini',
  properties: 'ini',
  sh: 'shell',
  bash: 'shell',
  zsh: 'shell',
  ps1: 'powershell',
  bat: 'bat',
  cmd: 'bat',
  go: 'go',
  rs: 'rust',
  py: 'python',
  rb: 'ruby',
  php: 'php',
  java: 'java',
  kt: 'kotlin',
  c: 'c',
  h: 'c',
  cpp: 'cpp',
  cc: 'cpp',
  cxx: 'cpp',
  hpp: 'cpp',
  cs: 'csharp',
  sql: 'sql',
  dockerfile: 'dockerfile',
  lua: 'lua',
  swift: 'swift',
  dart: 'dart',
  log: 'plaintext',
  txt: 'plaintext',
}

const LANGUAGE_BY_FILENAME: Record<string, string> = {
  dockerfile: 'dockerfile',
  makefile: 'makefile',
  '.gitignore': 'ini',
  '.env': 'ini',
}

export const detectMonacoLanguage = (name: string): string => {
  const base = (name.split(/[\\/]/).pop() || name).toLowerCase()
  if (LANGUAGE_BY_FILENAME[base]) return LANGUAGE_BY_FILENAME[base]
  const ext = getFileExtension(base)
  return LANGUAGE_BY_EXTENSION[ext] || 'plaintext'
}

// 行尾风格识别。
export type EndOfLine = 'LF' | 'CRLF'

export const detectEndOfLine = (text: string): EndOfLine => (text.includes('\r\n') ? 'CRLF' : 'LF')

export const applyEndOfLine = (text: string, eol: EndOfLine): string => {
  const normalized = text.replace(/\r\n/g, '\n')
  return eol === 'CRLF' ? normalized.replace(/\n/g, '\r\n') : normalized
}
