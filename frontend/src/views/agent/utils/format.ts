// Agent 页面共用的格式化：相对时间、时长、字节、token 数。
import { translate } from '@/locales'

type TimeLike = Date | string | number | null | undefined

const toDate = (value: TimeLike): Date | null => {
  if (value == null || value === '') return null
  const d = value instanceof Date ? value : new Date(value)
  return Number.isNaN(d.getTime()) ? null : d
}

const pad = (n: number) => String(n).padStart(2, '0')
const hm = (d: Date) => `${pad(d.getHours())}:${pad(d.getMinutes())}`
const dayStart = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime()

/** 刚刚 / 3 分钟前 / 今天 10:24 / 昨天 21:40 / 周一 15:12 / 11 月 12 日 / 2025 年 3 月 1 日 */
export const relTime = (value: TimeLike): string => {
  const d = toDate(value)
  if (!d) return ''
  const now = new Date()
  const diff = now.getTime() - d.getTime()
  if (diff < 60_000) return translate('agent.common.justNow')
  if (diff < 3_600_000) return translate('agent.time.minutesAgo', { n: Math.floor(diff / 60_000) })
  const days = Math.round((dayStart(now) - dayStart(d)) / 86_400_000)
  if (days === 0) return translate('agent.time.today', { time: hm(d) })
  if (days === 1) return translate('agent.time.yesterday', { time: hm(d) })
  if (days < 7) return translate('agent.time.weekday', { day: translate(`agent.time.days.d${d.getDay()}`), time: hm(d) })
  if (d.getFullYear() === now.getFullYear()) return translate('agent.time.monthDay', { m: d.getMonth() + 1, d: d.getDate() })
  return translate('agent.time.fullDate', { y: d.getFullYear(), m: d.getMonth() + 1, d: d.getDate() })
}

/** 时刻：今天只写 14:30，一周内写「周四 14:30」，其它写日期 */
export const clockTime = (value: TimeLike): string => {
  const d = toDate(value)
  if (!d) return ''
  const now = new Date()
  const days = Math.round((dayStart(d) - dayStart(now)) / 86_400_000)
  if (days === 0) return hm(d)
  if (days > 0 && days < 7) return `${translate(`agent.time.days.d${d.getDay()}`)} ${hm(d)}`
  return translate('agent.time.monthDay', { m: d.getMonth() + 1, d: d.getDate() })
}

/** 时长：0.4 秒 / 6 秒 / 1 分 52 秒 / 12 分钟 / 3 小时 5 分 */
export const fmtDuration = (ms: number): string => {
  const s = Math.max(0, Math.round(ms / 1000))
  if (s < 60) return translate('agent.time.seconds', { n: ms < 10_000 ? Math.max(0.1, Math.round(ms / 100) / 10) : s })
  const m = Math.floor(s / 60)
  if (m < 60) return s % 60 ? translate('agent.time.minSec', { m, s: s % 60 }) : translate('agent.time.minutes', { n: m })
  const h = Math.floor(m / 60)
  return m % 60 ? translate('agent.time.hourMin', { h, m: m % 60 }) : translate('agent.time.hours', { n: h })
}

/** 从某个时刻到现在过了多久：刚启动 / 6 分钟 / 3 小时 5 分 */
export const sinceTime = (value: TimeLike): string => {
  const d = toDate(value)
  if (!d) return ''
  const ms = Date.now() - d.getTime()
  if (ms < 60_000) return translate('agent.time.justStarted')
  return fmtDuration(Math.floor(ms / 60_000) * 60_000)
}

export const fmtBytes = (bytes: number | string | null | undefined): string => {
  const n = Number(bytes || 0)
  if (n < 1024) return `${n} B`
  if (n < 1024 * 1024) return `${(n / 1024).toFixed(1)} KB`
  if (n < 1024 * 1024 * 1024) return `${Math.round(n / 1024 / 1024)} MB`
  return `${(n / 1024 / 1024 / 1024).toFixed(1)} GB`
}

/** token 数：986 / 18.4k / 1.20M */
export const fmtTokens = (value: number | string | null | undefined): string => {
  const n = Number(value || 0)
  if (n < 1000) return String(n)
  if (n < 1_000_000) return `${(n / 1000).toFixed(1)}k`
  return `${(n / 1_000_000).toFixed(2)}M`
}

/** 百分比，保留一位小数；分母为 0 时为空 */
export const fmtPercent = (part: number, whole: number): string => {
  if (!whole) return ''
  return `${((part / whole) * 100).toFixed(1)}%`
}

/** 右侧「文件」里的文件拖到输入框时带的数据类型：内容是相对会话目录的路径 */
export const AGENT_PATH_MIME = 'application/x-momoko-agent-path'

/** Agent 标记用的种类：claude / codex / antigravity / custom */
export const agentKindOf = (kind?: string) =>
  kind === 'claude' || kind === 'codex' || kind === 'antigravity' ? kind : 'custom'

/** 第三方接口的协议给哪一系 Agent 用：anthropic → claude，openai → codex */
export const protocolAgent = (protocol?: string) =>
  protocol === 'anthropic' ? 'claude' : protocol === 'openai' ? 'codex' : 'custom'
