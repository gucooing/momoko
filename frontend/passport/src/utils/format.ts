// 时间、数字与错误文案的格式化（全部经通行证自己的 i18n）。
import { PassportError } from '@passport/api/request'
import { i18n, t, te } from '@passport/locales'

/** 生成的 TS 类型把时间写成 Date，接口实际返回 RFC3339 字符串（或 null）。 */
export type TimeLike = Date | string | number | null | undefined

export function toDate(v: TimeLike): Date | null {
  if (v === null || v === undefined || v === '') return null
  const d = v instanceof Date ? v : new Date(v)
  return Number.isNaN(d.getTime()) ? null : d
}

const pad = (n: number) => String(n).padStart(2, '0')

function sameDay(a: Date, b: Date) {
  return a.getFullYear() === b.getFullYear() && a.getMonth() === b.getMonth() && a.getDate() === b.getDate()
}

/** 今天 21:12 / 昨天 23:01 / 11-03 20:15 / 2025-11-03。 */
export function fmtTime(v: TimeLike): string {
  const d = toDate(v)
  if (!d) return '—'
  const now = new Date()
  const hm = `${pad(d.getHours())}:${pad(d.getMinutes())}`
  if (sameDay(d, now)) return t('common.time.today', { time: hm })
  const y = new Date(now)
  y.setDate(now.getDate() - 1)
  if (sameDay(d, y)) return t('common.time.yesterday', { time: hm })
  if (d.getFullYear() === now.getFullYear()) return `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${hm}`
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** 2026-10-02。 */
export function fmtDate(v: TimeLike): string {
  const d = toDate(v)
  if (!d) return '—'
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

/** 刚刚 / 12 分钟前 / 2 小时前 / 3 天前；更早用日期。 */
export function fmtAgo(v: TimeLike): string {
  const d = toDate(v)
  if (!d) return '—'
  const sec = Math.round((Date.now() - d.getTime()) / 1000)
  if (sec < 60) return t('common.time.justNow')
  if (sec < 3600) return t('common.time.minutesAgo', { n: Math.floor(sec / 60) })
  if (sec < 86400) return t('common.time.hoursAgo', { n: Math.floor(sec / 3600) })
  if (sec < 86400 * 30) return t('common.time.daysAgo', { n: Math.floor(sec / 86400) })
  return fmtDate(d)
}

/** 还有多久：3 天 / 5 小时 / 12 分钟。 */
export function fmtLeft(v: TimeLike): string {
  const d = toDate(v)
  if (!d) return '—'
  const sec = Math.max(0, Math.round((d.getTime() - Date.now()) / 1000))
  if (sec >= 86400) return t('common.time.days', { n: Math.ceil(sec / 86400) })
  if (sec >= 3600) return t('common.time.hours', { n: Math.ceil(sec / 3600) })
  return t('common.time.minutes', { n: Math.max(1, Math.ceil(sec / 60)) })
}

/** 秒数 → 1 小时 / 30 分钟 / 2 天。 */
export function fmtDuration(seconds: number): string {
  if (seconds >= 86400 && seconds % 86400 === 0) return t('common.time.days', { n: seconds / 86400 })
  if (seconds >= 3600 && seconds % 3600 === 0) return t('common.time.hours', { n: seconds / 3600 })
  if (seconds >= 60) return t('common.time.minutes', { n: Math.round(seconds / 60) })
  return t('common.time.seconds', { n: seconds })
}

export function fmtNum(n: number | null | undefined): string {
  if (n === null || n === undefined || Number.isNaN(n)) return '—'
  return new Intl.NumberFormat(i18n.global.locale.value).format(n)
}

/** 接口错误 → 本地化文案：PASSPORT_* 代码查表，其余（游戏服务端、GM 规则）原样显示。 */
export function errorText(e: unknown): string {
  if (e instanceof PassportError) {
    const key = `errors.${e.code}`
    if (/^[A-Z][A-Z0-9_]+$/.test(e.code) && te(key)) return t(key)
    if (e.code === 'NETWORK_ERROR') return t('errors.NETWORK_ERROR')
    if (/^\d+$/.test(e.code)) return t('errors.HTTP', { status: e.code })
    return e.code
  }
  return t('errors.UNKNOWN')
}

/** 只显示邮箱的一部分。 */
export function maskEmail(email: string): string {
  const at = email.lastIndexOf('@')
  if (at <= 0) return email
  return `${email.slice(0, 1)}***${email.slice(at)}`
}
