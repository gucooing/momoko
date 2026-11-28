// 会话里要从接口取回来显示的图片：消息附件和工具调用产出的图片（在会话附件目录里），以及回复里引用的本机图片。
// 取回来的 data: 地址按会话 + 来源 + 路径记着，最多记这么多张（最久没用的先丢）。
import { getAgentSessionLocalFile, getAgentSessionUpload } from '@/api/agent'

const CACHE_MAX = 60
const cache = new Map<string, Promise<string>>()

/** 取回一张图片；接口回空（文件不在）时为空字符串 */
const cached = (key: string, load: () => Promise<{ data: { mime: string; data: string } }>) => {
  const hit = cache.get(key)
  if (hit) {
    cache.delete(key)
    cache.set(key, hit)
    return hit
  }
  const p = load().then(({ data }) => (data.data ? `data:${data.mime};base64,${data.data}` : ''))
  p.catch(() => cache.delete(key))
  cache.set(key, p)
  if (cache.size > CACHE_MAX) cache.delete(cache.keys().next().value as string)
  return p
}

/** 会话附件目录里的图片：file 是事件里记的那个 */
export const uploadImage = (sessionId: string, file: string) =>
  cached(`${sessionId}\nupload\n${file}`, () => getAgentSessionUpload(sessionId, file))

/** 回复里引用的本机图片：path 是 Agent 写的绝对路径 */
export const localImage = (sessionId: string, path: string) =>
  cached(`${sessionId}\nlocal\n${path}`, () => getAgentSessionLocalFile(sessionId, path))

/** 图片地址指向本机文件时的路径（file:// 地址的路径部分，或盘符、/ 开头的绝对路径）；不是本机文件为空 */
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
