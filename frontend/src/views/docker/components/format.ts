// Docker 各页共用的展示格式化：字节、短 ID、相对时间、绝对日期。
type Translate = (key: string, named?: Record<string, unknown>) => string

export const formatBytes = (bytes?: number | string | unknown) => {
  const n = Number(bytes)
  if (!n || Number.isNaN(n)) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  let i = 0
  let v = n
  while (v >= 1024 && i < units.length - 1) {
    v /= 1024
    i++
  }
  return `${v.toFixed(i > 0 && v < 100 ? 1 : 0)} ${units[i]}`
}

export const shortId = (id?: unknown) => String(id || '').replace(/^sha256:/, '').slice(0, 12) || '-'

const toDate = (value: unknown): Date | null => {
  if (!value) return null
  const s = value instanceof Date ? value : String(value)
  if (typeof s === 'string' && s.startsWith('0001-')) return null
  const d = new Date(s)
  return Number.isNaN(d.getTime()) ? null : d
}

const pad = (n: number) => String(n).padStart(2, '0')

export const formatDate = (value: unknown) => {
  const d = toDate(value)
  if (!d) return '—'
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`
}

export const formatDateTime = (value: unknown) => {
  const d = toDate(value)
  if (!d) return '—'
  return `${formatDate(d)} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

export const formatClock = (value: unknown) => {
  const d = toDate(value)
  if (!d) return '—'
  const today = new Date()
  const sameDay = d.toDateString() === today.toDateString()
  return sameDay ? `${pad(d.getHours())}:${pad(d.getMinutes())}` : `${pad(d.getMonth() + 1)}-${pad(d.getDate())} ${pad(d.getHours())}:${pad(d.getMinutes())}`
}

/** 相对时间：刚刚 / N 分钟前 / N 小时前 / N 天前 / N 周前 / N 个月前 / N 年前 */
export const formatAgo = (value: unknown, t: Translate) => {
  const d = toDate(value)
  if (!d) return '—'
  const sec = Math.max(0, (Date.now() - d.getTime()) / 1000)
  if (sec < 60) return t('docker.ago.justNow')
  const min = sec / 60
  if (min < 60) return t('docker.ago.minutes', { n: Math.floor(min) })
  const hr = min / 60
  if (hr < 24) return t('docker.ago.hours', { n: Math.floor(hr) })
  const day = hr / 24
  if (day < 14) return t('docker.ago.days', { n: Math.floor(day) })
  if (day < 60) return t('docker.ago.weeks', { n: Math.floor(day / 7) })
  if (day < 365) return t('docker.ago.months', { n: Math.floor(day / 30) })
  return t('docker.ago.years', { n: Math.floor(day / 365) })
}

/** 贴纸配色轮换（按名字稳定取色） */
const STICKERS = ['m-sticker--p', 'm-sticker--a2', 'm-sticker--a', 'm-sticker--ok'] as const
export const stickerFor = (name: string) => {
  let h = 0
  for (const ch of name) h = (h * 31 + ch.charCodeAt(0)) >>> 0
  return STICKERS[h % STICKERS.length]
}

/** 时长：N 秒 / N 分钟 / N 小时 / N 天（取最大单位） */
export const formatDuration = (ms: number, t: Translate) => {
  const sec = Math.max(0, Math.floor(ms / 1000))
  if (sec < 60) return t('docker.page.dur.seconds', { n: sec })
  const min = Math.floor(sec / 60)
  if (min < 60) return t('docker.page.dur.minutes', { n: min })
  const hr = Math.floor(min / 60)
  if (hr < 48) return t('docker.page.dur.hours', { n: hr })
  return t('docker.page.dur.days', { n: Math.floor(hr / 24) })
}

/** 月日（随界面语言）：10月21日 / Oct 21；跨年时带年份 */
export const formatMonthDay = (value: unknown, locale: string) => {
  const d = toDate(value)
  if (!d) return '—'
  const opts: Intl.DateTimeFormatOptions = { month: 'short', day: 'numeric' }
  if (d.getFullYear() !== new Date().getFullYear()) opts.year = 'numeric'
  try {
    return new Intl.DateTimeFormat(locale, opts).format(d)
  } catch {
    return formatDate(d)
  }
}

/** 速度：4.8 MB/s */
export const formatRate = (bytesPerSecond: unknown) => `${formatBytes(bytesPerSecond)}/s`

/** 按简化的 shell 规则拆分命令行（支持单双引号与反斜杠转义），与后端 compose 解析一致 */
export const splitCommand = (text: string): string[] => {
  const out: string[] = []
  let cur = ''
  let quote = ''
  let escaped = false
  let has = false
  for (const ch of text) {
    if (escaped) {
      cur += ch
      escaped = false
    } else if (ch === '\\' && quote !== "'") {
      escaped = true
      has = true
    } else if (quote) {
      if (ch === quote) quote = ''
      else cur += ch
    } else if (ch === '"' || ch === "'") {
      quote = ch
      has = true
    } else if (/\s/.test(ch)) {
      if (has || cur) out.push(cur)
      cur = ''
      has = false
    } else {
      cur += ch
    }
  }
  if (has || cur) out.push(cur)
  return out
}

/** 把参数数组拼回命令行，含空白或引号的参数加引号 */
export const joinCommand = (args: string[]) =>
  args.map((a) => (a === '' ? '""' : /[\s"'\\$]/.test(a) ? `'${a.replace(/'/g, `'\\''`)}'` : a)).join(' ')
