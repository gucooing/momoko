// 性能监控页的格式化工具：百分比、字节、速率、时间刻度与迷你走势路径。
import dayjs from 'dayjs'

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']

export const num = (v: unknown): number => {
  const n = Number(v)
  return Number.isFinite(n) ? n : 0
}

export const fmtPct = (v?: number | null) => (v == null || !Number.isFinite(v) ? '—' : `${v.toFixed(1)}%`)

/** 字节数 → [数值, 单位] */
export const splitSize = (bytes?: number | null): [string, string] => {
  let v = Math.max(0, num(bytes))
  let i = 0
  while (v >= 1024 && i < UNITS.length - 1) {
    v /= 1024
    i++
  }
  return [v.toFixed(i === 0 || v >= 100 ? 0 : 1), UNITS[i]!]
}

/** 字节数 → 可读大小；int64 字段在 JSON 里是字符串，这里统一转数字 */
export const fmtSize = (bytes?: unknown) => {
  const n = Number(bytes)
  if (bytes == null || bytes === '' || !Number.isFinite(n)) return '—'
  const [v, u] = splitSize(n)
  return `${v} ${u}`
}

/** 带符号的大小（增长量），如 +4.1 GB / −120 MB */
export const fmtSignedSize = (bytes: number) => `${bytes < 0 ? '−' : '+'}${fmtSize(Math.abs(bytes))}`

export const fmtRate = (bps?: unknown) => {
  const n = Number(bps)
  if (bps == null || bps === '' || !Number.isFinite(n)) return '—'
  const [v, u] = splitSize(n)
  return `${v} ${u}/s`
}

/** 按最大值选一个固定单位（给坐标轴与副标题用），返回除数与单位名 */
export const rateUnit = (max: number): { div: number; label: string } => {
  let i = 0
  let v = Math.max(0, max)
  while (v >= 1024 && i < UNITS.length - 1) {
    v /= 1024
    i++
  }
  return { div: 1024 ** i, label: `${UNITS[i]}/s` }
}

export const fmtCount = (v?: number | null) => {
  if (v == null || !Number.isFinite(v)) return '—'
  if (v >= 10000) return `${(v / 1000).toFixed(1)}k`
  return Math.round(v).toLocaleString('en-US')
}

export const fmtPerSec = (v?: number | null) => (v == null || !Number.isFinite(v) ? '—' : `${fmtCount(v)}/s`)

/** 时间轴刻度：一天以内显示时分，跨天显示月-日（更长再带时分） */
export const axisTimeFormat = (spanSec: number) => (spanSec <= 26 * 3600 ? 'HH:mm' : spanSec <= 3 * 86400 ? 'MM-DD HH:mm' : 'MM-DD')

/** 悬停 / 峰值时间：一小时内的桶带秒 */
export const pointTime = (ms: number, stepSec: number) => {
  const d = dayjs(ms)
  const fmt = stepSec < 60 ? 'HH:mm:ss' : 'HH:mm'
  if (d.isSame(dayjs(), 'day')) return d.format(fmt)
  return d.format(`MM-DD ${fmt}`)
}

export const rangeText = (startSec: number, endSec: number) => {
  const s = dayjs.unix(startSec)
  const e = dayjs.unix(endSec)
  const sameDay = s.isSame(e, 'day')
  return `${s.format('MM-DD HH:mm')} – ${e.format(sameDay ? 'HH:mm' : 'MM-DD HH:mm')}`
}

/**
 * 迷你走势路径（viewBox 200×40）：空值处断开，max 不传时按数据自动缩放。
 */
export const sparkPaths = (values: (number | null)[], max?: number): { line: string; area: string } => {
  const n = values.length
  if (n < 2) return { line: '', area: '' }
  const top = max ?? Math.max(1e-9, ...values.map((v) => v ?? 0)) * 1.1
  const x = (k: number) => ((k / (n - 1)) * 200).toFixed(1)
  const y = (v: number) => (40 - Math.min(1, v / top) * 36).toFixed(1)
  let line = ''
  let area = ''
  let runStart = -1
  const closeRun = (end: number) => {
    if (runStart < 0) return
    area += `L${x(end)} 40L${x(runStart)} 40Z`
    runStart = -1
  }
  values.forEach((v, k) => {
    if (v == null) {
      closeRun(k - 1)
      return
    }
    const seg = `${x(k)} ${y(v)}`
    if (runStart < 0) {
      runStart = k
      line += `M${seg}`
      area += `M${seg}`
    } else {
      line += `L${seg}`
      area += `L${seg}`
    }
  })
  closeRun(n - 1)
  return { line, area }
}

/** 时长 → 「N 小时 / N 天 / N 个月 / N 年」的文案键与数值 */
export const durationParts = (sec: number): { key: 'minutes' | 'hours' | 'daysN' | 'months' | 'years'; n: number } => {
  if (sec < 3600) return { key: 'minutes', n: Math.max(1, Math.round(sec / 60)) }
  if (sec < 86400) return { key: 'hours', n: Math.round(sec / 3600) }
  if (sec < 60 * 86400) return { key: 'daysN', n: Math.round(sec / 86400) }
  if (sec < 730 * 86400) return { key: 'months', n: Math.round(sec / (30 * 86400)) }
  return { key: 'years', n: Math.round(sec / (365 * 86400)) }
}
