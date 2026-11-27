// 代理（只有 Linux 能配）：表单里就是一个地址，留空不走代理；地址开头的 http:// 或 socks5:// 决定类型（不写默认 http），
// 账号密码写在地址里。已保存的密码显示成 ******，原样不动就是保留原来的密码。
import type { AgentProxy } from '@/types/v1/agent'

/** 表单里的代理地址 */
export type ProxyForm = string

/** 已保存的密码在地址里的样子 */
const MASK = '******'

/** 已保存的代理 → 表单里的地址（密码用 ****** 代替） */
export const proxyForm = (p?: AgentProxy | null): ProxyForm => {
  if (!p?.type) return ''
  const host = p.host.includes(':') ? `[${p.host}]` : p.host
  let auth = ''
  if (p.username) auth = `${encodeURIComponent(p.username)}${p.hasPassword ? `:${MASK}` : ''}@`
  return `${p.type}://${auth}${host}:${p.port}`
}

type Parsed = { ok: true; proxy: AgentProxy } | { ok: false }

/** 拆地址；留空是不走代理（type 为空） */
const parse = (text: string): Parsed => {
  const raw = text.trim()
  const empty: AgentProxy = { type: '', host: '', port: 0, username: '', password: '', hasPassword: false }
  if (!raw) return { ok: true, proxy: empty }
  const withScheme = /^[a-z][a-z0-9+.-]*:\/\//i.test(raw) ? raw : `http://${raw}`
  let u: URL
  try {
    u = new URL(withScheme)
  } catch {
    return { ok: false }
  }
  const scheme = u.protocol.replace(':', '').toLowerCase()
  const type = scheme === 'http' ? 'http' : scheme === 'socks5' || scheme === 'socks5h' ? 'socks5' : ''
  const port = Number(u.port)
  const host = u.hostname.replace(/^\[|\]$/g, '')
  if (!type || !host || !(port > 0 && port <= 65535)) return { ok: false }
  const password = decodeURIComponent(u.password)
  return {
    ok: true,
    proxy: {
      type,
      host,
      port,
      username: decodeURIComponent(u.username),
      // ****** 是没改的已保存密码：交空的给后端，表示保留
      password: password === MASK ? '' : password,
      hasPassword: false,
    },
  }
}

/** 提交给后端的：不走代理时 type 为空 */
export const proxyPayload = (f: ProxyForm): AgentProxy => {
  const r = parse(f)
  return r.ok ? r.proxy : { type: '', host: '', port: 0, username: '', password: '', hasPassword: false }
}

/** 地址能用：留空，或者是 http:// / socks5:// 加主机和端口 */
export const proxyReady = (f: ProxyForm) => parse(f).ok

/** 行上显示的地址（不带账号密码） */
export const proxyLabel = (p?: AgentProxy | null) => {
  if (!p?.type) return ''
  const host = p.host.includes(':') ? `[${p.host}]` : p.host
  return `${p.type}://${host}:${p.port}`
}
