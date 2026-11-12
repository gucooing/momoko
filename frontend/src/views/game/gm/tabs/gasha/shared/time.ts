// 配置编辑用的时间换算：配置里的时间是 Unix 毫秒，页面一律按服务器时区（GM 上下文的 IANA 时区）显示和输入。
// 浏览器时区与服务器时区不同时也不会错位。

/** 可用的时区：服务器给的时区无效或为空时用浏览器时区。 */
export function resolveZone(tz: string | undefined): string {
  if (tz) {
    try {
      new Intl.DateTimeFormat('en-US', { timeZone: tz })
      return tz
    } catch {
      // 无效时区，退回浏览器时区
    }
  }
  return Intl.DateTimeFormat().resolvedOptions().timeZone || 'UTC'
}

const formatters = new Map<string, Intl.DateTimeFormat>()
function partsFormatter(tz: string): Intl.DateTimeFormat {
  let f = formatters.get(tz)
  if (!f) {
    f = new Intl.DateTimeFormat('en-US', {
      timeZone: tz,
      hourCycle: 'h23',
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit',
      second: '2-digit',
      weekday: 'short',
    })
    formatters.set(tz, f)
  }
  return f
}

export interface ZonedParts {
  year: number
  month: number
  day: number
  hour: number
  minute: number
  second: number
  /** 0 = 星期日 */
  weekday: number
}

const WEEKDAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat']

/** 某个时刻在时区里的年月日时分秒。 */
export function zonedParts(ms: number, tz: string): ZonedParts {
  const out: ZonedParts = { year: 1970, month: 1, day: 1, hour: 0, minute: 0, second: 0, weekday: 0 }
  for (const p of partsFormatter(tz).formatToParts(new Date(ms))) {
    switch (p.type) {
      case 'year':
        out.year = Number(p.value)
        break
      case 'month':
        out.month = Number(p.value)
        break
      case 'day':
        out.day = Number(p.value)
        break
      case 'hour':
        out.hour = Number(p.value) % 24
        break
      case 'minute':
        out.minute = Number(p.value)
        break
      case 'second':
        out.second = Number(p.value)
        break
      case 'weekday':
        out.weekday = Math.max(0, WEEKDAYS.indexOf(p.value))
        break
    }
  }
  return out
}

/** 时区相对 UTC 的偏移（毫秒）。 */
export function zoneOffset(ms: number, tz: string): number {
  const p = zonedParts(ms, tz)
  return Date.UTC(p.year, p.month - 1, p.day, p.hour, p.minute, p.second) - Math.floor(ms / 1000) * 1000
}

/** 时区里的年月日时分秒 → Unix 毫秒。 */
export function zonedToMs(tz: string, year: number, month: number, day: number, hour = 0, minute = 0, second = 0): number {
  const wall = Date.UTC(year, month - 1, day, hour, minute, second)
  let ms = wall - zoneOffset(wall, tz)
  // 夏令时切换附近再校正一次
  ms = wall - zoneOffset(ms, tz)
  return ms
}

const pad = (n: number, len = 2) => String(n).padStart(len, '0')

/** YYYY-MM-DD HH:mm（withSeconds 时带秒）。 */
export function formatZoned(ms: number, tz: string, withSeconds = false): string {
  const p = zonedParts(ms, tz)
  const base = `${p.year}-${pad(p.month)}-${pad(p.day)} ${pad(p.hour)}:${pad(p.minute)}`
  return withSeconds ? `${base}:${pad(p.second)}` : base
}

/** MM-DD HH:mm */
export function formatShort(ms: number, tz: string): string {
  const p = zonedParts(ms, tz)
  return `${pad(p.month)}-${pad(p.day)} ${pad(p.hour)}:${pad(p.minute)}`
}

/** MM/DD */
export function formatDay(ms: number, tz: string): string {
  const p = zonedParts(ms, tz)
  return `${pad(p.month)}/${pad(p.day)}`
}

/** YYYY-MM-DD */
export function formatDate(ms: number, tz: string): string {
  const p = zonedParts(ms, tz)
  return `${p.year}-${pad(p.month)}-${pad(p.day)}`
}

/** datetime-local 输入框的值（YYYY-MM-DDTHH:mm:ss）。 */
export function toInputValue(ms: number, tz: string): string {
  const p = zonedParts(ms, tz)
  return `${p.year}-${pad(p.month)}-${pad(p.day)}T${pad(p.hour)}:${pad(p.minute)}:${pad(p.second)}`
}

/** datetime-local 输入框的值 → Unix 毫秒（按服务器时区解释）；无效时返回 undefined。 */
export function fromInputValue(value: string, tz: string): number | undefined {
  const m = /^(\d{4})-(\d{2})-(\d{2})T(\d{2}):(\d{2})(?::(\d{2}))?/.exec(value)
  if (!m) return undefined
  const [, y, mo, d, h, mi, s] = m
  return zonedToMs(tz, Number(y), Number(mo), Number(d), Number(h), Number(mi), Number(s || 0))
}

/** UTC+8 / UTC+5:30 / UTC−3 */
export function offsetLabel(tz: string, at = Date.now()): string {
  const minutes = Math.round(zoneOffset(at, tz) / 60000)
  if (minutes === 0) return 'UTC'
  const sign = minutes > 0 ? '+' : '−'
  const abs = Math.abs(minutes)
  const h = Math.floor(abs / 60)
  const m = abs % 60
  return `UTC${sign}${h}${m ? `:${pad(m)}` : ''}`
}

/** 接口里的时间（protobuf Timestamp 的 JSON 字符串、Date 或毫秒）→ 毫秒；没有时返回 0。 */
export function toMs(value: unknown): number {
  if (value instanceof Date) return value.getTime()
  if (typeof value === 'number') return value
  if (typeof value === 'string' && value) {
    const ms = Date.parse(value)
    return Number.isNaN(ms) ? 0 : ms
  }
  return 0
}
