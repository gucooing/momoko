// 代理（只有 Linux 能配）：表单里是一个地址，加上沙箱里的时区和语言。
// 地址留空 = 不走代理；开头的 http:// 或 socks5:// 决定类型（不写默认 http）；账号密码写在地址里，
// 已保存的密码显示成 ******，原样不动就是保留原来的密码。
import type { AgentProxy, AgentSandbox } from '@/types/v1/agent'

export interface ProxyForm {
  url: string
  /** IANA 时区名，如 America/Los_Angeles；空 = 跟服务器一样 */
  timezone: string
  /** 语言区域，如 en_US.UTF-8；空 = 跟服务器一样 */
  locale: string
}

/** 已保存的密码在地址里的样子 */
const MASK = '******'
const TIMEZONE = /^[A-Za-z][A-Za-z0-9_+-]*(\/[A-Za-z0-9_+-]+)*$/
const LOCALE = /^[A-Za-z][A-Za-z0-9_.-]{0,39}$/

const empty = (): AgentProxy => ({ type: '', host: '', port: 0, username: '', password: '', hasPassword: false, timezone: '', locale: '' })

/** 已保存的代理 → 表单（密码用 ****** 代替） */
export const proxyForm = (p?: AgentProxy | null): ProxyForm => {
  if (!p?.type) return { url: '', timezone: '', locale: '' }
  const host = p.host.includes(':') ? `[${p.host}]` : p.host
  const auth = p.username ? `${encodeURIComponent(p.username)}${p.hasPassword ? `:${MASK}` : ''}@` : ''
  return { url: `${p.type}://${auth}${host}:${p.port}`, timezone: p.timezone, locale: p.locale }
}

/** 拆地址；留空是不走代理（type 为空），拆不开为 null */
const parse = (text: string): AgentProxy | null => {
  const raw = text.trim()
  if (!raw) return empty()
  let u: URL
  try {
    u = new URL(/^[a-z][a-z0-9+.-]*:\/\//i.test(raw) ? raw : `http://${raw}`)
  } catch {
    return null
  }
  const scheme = u.protocol.replace(':', '').toLowerCase()
  const type = scheme === 'http' ? 'http' : scheme === 'socks5' || scheme === 'socks5h' ? 'socks5' : ''
  const port = Number(u.port)
  const host = u.hostname.replace(/^\[|\]$/g, '')
  if (!type || !host || !(port > 0 && port <= 65535)) return null
  const password = decodeURIComponent(u.password)
  return {
    ...empty(),
    type,
    host,
    port,
    username: decodeURIComponent(u.username),
    // ****** 是没改的已保存密码：交空的给后端，表示保留
    password: password === MASK ? '' : password,
  }
}

/** 地址写得对（或者留空） */
export const proxyAddressOk = (f: ProxyForm) => parse(f.url) !== null
export const timezoneOk = (f: ProxyForm) => !f.timezone.trim() || TIMEZONE.test(f.timezone.trim())
export const localeOk = (f: ProxyForm) => !f.locale.trim() || LOCALE.test(f.locale.trim())

/** 整份都能提交 */
export const proxyReady = (f: ProxyForm) => proxyAddressOk(f) && timezoneOk(f) && localeOk(f)

/** 提交给后端的：不走代理时 type 为空（时区和语言跟着代理，不走代理就没有） */
export const proxyPayload = (f: ProxyForm): AgentProxy => {
  const p = parse(f.url)
  if (!p?.type) return empty()
  return { ...p, timezone: f.timezone.trim(), locale: f.locale.trim() }
}

/** 沙箱现在用不了的原因：建不了，或者 sing-box 还没下载；没有问题为空 */
export const sandboxProblem = (t: (key: string) => string, s?: AgentSandbox | null) => {
  if (!s?.supported) return ''
  if (s.unavailable) return s.unavailable
  return s.singBoxVersion ? '' : t('agent.proxy.noSingBox')
}

/** 行上显示的地址（不带账号密码） */
export const proxyLabel = (p?: AgentProxy | null) => {
  if (!p?.type) return ''
  const host = p.host.includes(':') ? `[${p.host}]` : p.host
  return `${p.type}://${host}:${p.port}`
}

/** 时区、语言的常用值（输入框的下拉建议） */
export const COMMON_TIMEZONES = [
  'America/Los_Angeles',
  'America/New_York',
  'America/Chicago',
  'Europe/London',
  'Europe/Berlin',
  'Asia/Tokyo',
  'Asia/Singapore',
  'Asia/Hong_Kong',
  'Asia/Taipei',
  'UTC',
]
export const COMMON_LOCALES = ['en_US.UTF-8', 'en_GB.UTF-8', 'ja_JP.UTF-8', 'zh_TW.UTF-8', 'zh_HK.UTF-8', 'C.UTF-8']
