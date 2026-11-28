// 会话（新会话页还没有会话时是工作区）里的文件给页面用：一律是 Momoko 标准的签名下载链接（/api/v1/download/pre），
// 浏览器直接取原样字节，图片用 <img src>，要内容的 fetch 它。不经 JSON、不转 base64。
// 同一拍里要的链接合成一次请求（一个会话里的十张图只签一次）；签好的按「范围 + 路径」记着，快过期了再签。
import { signAgentFiles, type AgentFileScope } from '@/api/agent'
import { resolvePreSignedFileUrl } from '@/utils/file'

/** 后端签 12 小时，提前一小时换新的 */
const FRESH_MS = 11 * 3600 * 1000
const CACHE_MAX = 300

interface Want {
  scope: AgentFileScope
  uploads: boolean
  path: string
  done: (url: string) => void
  fail: (e: unknown) => void
}

const cache = new Map<string, { url: Promise<string>; at: number }>()
let pending: Want[] = []

const groupOf = (w: Pick<Want, 'scope' | 'uploads'>) => `${w.scope.sessionId || ''}\n${w.scope.workspaceId || ''}\n${w.uploads}`

const flush = () => {
  const batch = pending
  pending = []
  const groups = new Map<string, Want[]>()
  for (const w of batch) {
    const key = groupOf(w)
    groups.set(key, [...(groups.get(key) ?? []), w])
  }
  for (const list of groups.values()) {
    const first = list[0]
    if (!first) continue
    signAgentFiles(
      first.scope,
      list.map((w) => w.path),
      first.uploads,
    ).then(
      ({ data }) => list.forEach((w, i) => w.done(data.urls?.[i] ? resolvePreSignedFileUrl(data.urls[i]) : '')),
      (e) => list.forEach((w) => w.fail(e)),
    )
  }
}

/**
 * 一个文件的签名链接；文件不在、不在能访问的目录里为空字符串。
 * path 相对工作目录或者是绝对路径；uploads 时相对会话的附件目录（消息附件、工具调用产出的图片）。
 */
export const fileLink = (scope: AgentFileScope, path: string, uploads = false): Promise<string> => {
  const key = `${groupOf({ scope, uploads })}\n${path}`
  const hit = cache.get(key)
  if (hit && Date.now() - hit.at < FRESH_MS) {
    cache.delete(key)
    cache.set(key, hit)
    return hit.url
  }
  const url = new Promise<string>((done, fail) => {
    if (!pending.length) queueMicrotask(flush)
    pending.push({ scope, uploads, path, done, fail })
  })
  url.catch(() => cache.delete(key))
  cache.set(key, { url, at: Date.now() })
  if (cache.size > CACHE_MAX) cache.delete(cache.keys().next().value as string)
  return url
}

/** 地址指向本机文件时的路径（file:// 地址的路径部分，或盘符、/ 开头的绝对路径）；不是本机文件为空 */
export const localPathOf = (src: string) => {
  if (/^file:\/\//i.test(src)) {
    try {
      return decodeURIComponent(new URL(src).pathname)
    } catch {
      return ''
    }
  }
  // Markdown 会把地址里的反斜杠、空格这些编码掉
  let path = src
  try {
    path = decodeURIComponent(src)
  } catch {
    /* 原样 */
  }
  return /^[a-z]:[\\/]/i.test(path) || (path.startsWith('/') && !path.startsWith('//')) ? path : ''
}

/** 看扩展名是不是图片 */
export const isImagePath = (path: string) => /\.(png|jpe?g|gif|webp|svg|bmp|avif|ico)$/i.test(path)
