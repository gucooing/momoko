// GM 数据页共用的格式化：数字、时间（显示用浏览器本地时间）、时长，以及按服务器时区计算「今天」的边界。
import dayjs from 'dayjs'
import { useI18n } from 'vue-i18n'

/** 时间戳（毫秒、protobuf Timestamp 的 JSON 字符串或 Date）→ 毫秒；无效时为 0。 */
export function toMs(value: number | string | Date | null | undefined): number {
  if (value === null || value === undefined || value === '') return 0
  if (typeof value === 'number') return Number.isFinite(value) ? value : 0
  const ms = dayjs(value).valueOf()
  return Number.isFinite(ms) ? ms : 0
}

interface ZonedParts {
  year: number
  month: number
  day: number
  hour: number
  minute: number
  second: number
}

function zonedParts(tz: string, ms: number): ZonedParts | undefined {
  try {
    const f = new Intl.DateTimeFormat('en-US', {
      timeZone: tz,
      hourCycle: 'h23',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
    })
    const p: Record<string, number> = {}
    for (const part of f.formatToParts(new Date(ms))) {
      if (part.type !== 'literal') p[part.type] = Number(part.value)
    }
    return {
      year: p.year ?? 1970,
      month: p.month ?? 1,
      day: p.day ?? 1,
      hour: p.hour ?? 0,
      minute: p.minute ?? 0,
      second: p.second ?? 0,
    }
  } catch {
    return undefined
  }
}

/** 时区在某一时刻相对 UTC 的偏移（毫秒）。 */
function zoneOffset(tz: string, ms: number): number {
  const p = zonedParts(tz, ms)
  if (!p) return -new Date(ms).getTimezoneOffset() * 60_000
  return Date.UTC(p.year, p.month - 1, p.day, p.hour, p.minute, p.second) - Math.floor(ms / 1000) * 1000
}

/** 服务器时区（IANA，空串为浏览器时区）里某一天 0 点的时间戳；date 为 YYYY-MM-DD 或时间戳。 */
export function zonedDayStart(tz: string, date: number | string): number {
  if (!tz) return dayjs(date).startOf('day').valueOf()
  let y: number
  let m: number
  let d: number
  if (typeof date === 'string') {
    const [ys, ms, ds] = date.split('-')
    y = Number(ys)
    m = Number(ms)
    d = Number(ds)
  } else {
    const p = zonedParts(tz, date)
    if (!p) return dayjs(date).startOf('day').valueOf()
    y = p.year
    m = p.month
    d = p.day
  }
  const guess = Date.UTC(y, m - 1, d)
  const first = guess - zoneOffset(tz, guess)
  // 夏令时切换日：用求得的时刻再校正一次
  return guess - zoneOffset(tz, first)
}

/** 数字、时间与时长的显示（跟随界面语言）。 */
export function useGmFormat() {
  const { t } = useI18n()

  const num = (value: number | null | undefined) =>
    value === null || value === undefined || !Number.isFinite(value) ? '—' : value.toLocaleString()

  /** 今天 HH:mm / 昨天 HH:mm / MM-DD HH:mm / YYYY-MM-DD */
  const when = (value: number | string | Date | null | undefined) => {
    const ms = toMs(value)
    if (!ms) return '—'
    const d = dayjs(ms)
    const now = dayjs()
    if (d.isSame(now, 'day')) return t('gmData.time.today', { time: d.format('HH:mm') })
    if (d.isSame(now.subtract(1, 'day'), 'day')) return t('gmData.time.yesterday', { time: d.format('HH:mm') })
    if (d.year() === now.year()) return d.format('MM-DD HH:mm')
    return d.format('YYYY-MM-DD')
  }

  const dateTime = (value: number | string | Date | null | undefined) => {
    const ms = toMs(value)
    return ms ? dayjs(ms).format('YYYY-MM-DD HH:mm') : '—'
  }

  const date = (value: number | string | Date | null | undefined) => {
    const ms = toMs(value)
    return ms ? dayjs(ms).format('YYYY-MM-DD') : '—'
  }

  /** 时长：x 天 y 小时 / x 小时 y 分 / x 分 */
  const duration = (ms: number) => {
    const minutes = Math.max(0, Math.floor(ms / 60_000))
    if (minutes < 1) return t('gmData.time.lessMinute')
    const days = Math.floor(minutes / 1440)
    const hours = Math.floor((minutes % 1440) / 60)
    const mins = minutes % 60
    if (days) return hours ? t('gmData.time.daysHours', { d: days, h: hours }) : t('gmData.time.days', { d: days })
    if (hours) return mins ? t('gmData.time.hoursMinutes', { h: hours, m: mins }) : t('gmData.time.hours', { h: hours })
    return t('gmData.time.minutes', { m: mins })
  }

  /** 多久以前：刚刚 / x 分钟前 / x 小时前 / x 天前，超过 30 天显示日期 */
  const ago = (value: number | string | Date | null | undefined) => {
    const ms = toMs(value)
    if (!ms) return '—'
    const diff = Date.now() - ms
    if (diff < 60_000) return t('gmData.time.justNow')
    const minutes = Math.floor(diff / 60_000)
    if (minutes < 60) return t('gmData.time.minutesAgo', { n: minutes })
    const hours = Math.floor(minutes / 60)
    if (hours < 24) return t('gmData.time.hoursAgo', { n: hours })
    const days = Math.floor(hours / 24)
    if (days <= 30) return t('gmData.time.daysAgo', { n: days })
    return dayjs(ms).format('YYYY-MM-DD')
  }

  /** 带正负号的增减量 */
  const delta = (value: number) => (value > 0 ? `+${value.toLocaleString()}` : value < 0 ? `−${Math.abs(value).toLocaleString()}` : '±0')

  /** 百分比（0–100 的整数） */
  const pct = (part: number, whole: number) => (whole > 0 ? Math.round((part / whole) * 100) : 0)

  return { num, when, dateTime, date, duration, ago, delta, pct }
}
