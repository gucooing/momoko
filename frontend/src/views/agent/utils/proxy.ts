// 代理（只有 Linux 能配）：表单里是一个节点，加上沙箱里的时区和语言。
// 节点留空 = 不走代理；填分享链接（vless://、vmess://、trojan://、ss://……）或 sing-box 的出站 JSON，后端认出来再用 sing-box 检查一遍。
import type { AgentProxy, AgentSandbox } from '@/types/v1/agent'

export interface ProxyForm {
  node: string
  /** IANA 时区名，如 America/Los_Angeles；空 = 跟服务器一样 */
  timezone: string
  /** 语言区域，如 en_US.UTF-8；空 = 跟服务器一样 */
  locale: string
}

/** 认得的分享链接（和后端 pkg/agentproxy 一致） */
const SCHEMES = ['vless', 'vmess', 'trojan', 'ss', 'hysteria2', 'hy2', 'hysteria', 'tuic', 'anytls', 'wireguard', 'wg', 'socks', 'socks5', 'socks5h']
const TIMEZONE = /^[A-Za-z][A-Za-z0-9_+-]*(\/[A-Za-z0-9_+-]+)*$/
const LOCALE = /^[A-Za-z][A-Za-z0-9_.-]{0,39}$/

/** 已保存的代理 → 表单 */
export const proxyForm = (p?: AgentProxy | null): ProxyForm => ({
  node: p?.node || '',
  timezone: p?.node ? p.timezone : '',
  locale: p?.node ? p.locale : '',
})

/** 节点看着像样（留空也算）：sing-box 的出站 JSON，或认得的分享链接；细的由后端查 */
export const nodeOk = (f: ProxyForm) => {
  const n = f.node.trim()
  if (!n) return true
  if (n.startsWith('{')) {
    try {
      return typeof JSON.parse(n)?.type === 'string'
    } catch {
      return false
    }
  }
  const scheme = /^([a-z0-9]+):\/\//i.exec(n)?.[1]?.toLowerCase()
  return !!scheme && SCHEMES.includes(scheme)
}
export const timezoneOk = (f: ProxyForm) => !f.timezone.trim() || TIMEZONE.test(f.timezone.trim())
export const localeOk = (f: ProxyForm) => !f.locale.trim() || LOCALE.test(f.locale.trim())

/** 整份都能提交 */
export const proxyReady = (f: ProxyForm) => nodeOk(f) && timezoneOk(f) && localeOk(f)

/** 提交给后端的：不走代理时节点为空（时区和语言跟着代理，不走代理就没有） */
export const proxyPayload = (f: ProxyForm): AgentProxy => {
  const node = f.node.trim()
  return { node, protocol: '', server: '', timezone: node ? f.timezone.trim() : '', locale: node ? f.locale.trim() : '' }
}

/** 沙箱现在用不了的原因：建不了，或者 sing-box 还没下载；没有问题为空 */
export const sandboxProblem = (t: (key: string) => string, s?: AgentSandbox | null) => {
  if (!s?.supported) return ''
  if (s.unavailable) return s.unavailable
  return s.singBoxVersion ? '' : t('agent.proxy.noSingBox')
}

/** 行上显示的节点：协议和地址（不带密钥） */
export const proxyLabel = (p?: AgentProxy | null) => {
  if (!p?.protocol) return ''
  return p.server ? `${p.protocol}://${p.server}` : p.protocol
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
