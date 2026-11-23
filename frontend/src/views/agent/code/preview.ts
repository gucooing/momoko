// HTML 预览站点的地址：后端把会话目录（新会话页是工作区）当静态站点给 iframe 加载，
// 地址是 {接口根}/public/agent/preview/{节点}/{签名}/{相对路径}（见后端 biz.AgentPreviewPrefix）。
// 签名 12 小时过期；按会话 / 工作区记着，快过期了重新要。
import { getAgentPreviewToken, type AgentFileScope } from '@/api/agent'
import { currentNodeId, LOCAL_NODE_ID } from '@/utils/nodeContext'
import { fileScope } from './fileState'

const tokens = new Map<string, Promise<{ token: string; exp: number }>>()
/** 剩不到这么久（秒）就换新的 */
const RENEW = 3600

const fetchToken = (scope: AgentFileScope) =>
  getAgentPreviewToken(scope).then(({ data }) => ({ token: data.token, exp: Number(data.expiresAt) }))

/** 预览站点的根地址（以 / 结尾） */
export const previewRoot = async (scope: AgentFileScope) => {
  const key = fileScope(scope.sessionId, scope.workspaceId)
  let hit = tokens.get(key)
  if (hit) {
    const cur = await hit.catch(() => null)
    if (!cur || cur.exp - Date.now() / 1000 < RENEW) hit = undefined
  }
  if (!hit) {
    hit = fetchToken(scope)
    tokens.set(key, hit)
    hit.catch(() => tokens.delete(key))
  }
  const { token } = await hit
  const api = new URL(import.meta.env.VITE_API_BASE_URL || '/api/v1', window.location.origin).href.replace(/\/+$/, '')
  return `${api}/public/agent/preview/${encodeURIComponent(currentNodeId.value || LOCAL_NODE_ID)}/${token}/`
}

/** 相对路径按段编码，放进地址里 */
export const encodePath = (path: string) => path.split('/').map(encodeURIComponent).join('/')

/** 和后端 previewShimPath 一致：沙箱里 localStorage 用不了时的替身脚本 */
export const PREVIEW_SHIM = '__momoko-preview/storage.js'

/**
 * 编辑中（没保存）的 HTML 放进 srcdoc 预览：开头插一个 <base> 指向它在预览站点里的位置，
 * 相对路径的链接、样式表、脚本、图片照样能取到；再先引存储替身脚本。
 */
export const withBase = (html: string, root: string, path: string) => {
  const dir = path.includes('/') ? `${encodePath(path.slice(0, path.lastIndexOf('/')))}/` : ''
  const head = `<base href="${root}${dir}"><script src="${root}${PREVIEW_SHIM}"></script>`
  const m = /<head[^>]*>/i.exec(html) ?? /<html[^>]*>/i.exec(html)
  return m ? html.slice(0, m.index + m[0].length) + head + html.slice(m.index + m[0].length) : head + html
}
