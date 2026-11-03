// 工具 · 网络（端口转发 / 内网穿透）共用的格式化与小工具。
type Translate = (key: string, named: Record<string, unknown>) => string

const UNITS = ['B', 'KB', 'MB', 'GB', 'TB', 'PB']

/** 字节数 → 可读大小，返回 [数值, 单位]，便于分开排版大数字与单位 */
export const splitBytes = (bytes?: unknown): [string, string] => {
  let v = Number(bytes) || 0
  let i = 0
  while (v >= 1024 && i < UNITS.length - 1) {
    v /= 1024
    i++
  }
  const digits = i === 0 ? 0 : v >= 100 ? 0 : 1
  return [v.toFixed(digits), UNITS[i]!]
}

export const formatBytes = (bytes?: unknown): string => {
  const [v, u] = splitBytes(bytes)
  return `${v} ${u}`
}

/** 字节/秒 → 可读速率；0 显示为「—」 */
export const formatRate = (bytesPerSecond?: unknown, zero = '—'): string => {
  const n = Number(bytesPerSecond) || 0
  if (n <= 0) return zero
  return `${formatBytes(n)}/s`
}

export const toDate = (value: unknown): Date | null => {
  if (!value) return null
  const d = value instanceof Date ? value : new Date(String(value))
  if (Number.isNaN(d.getTime()) || d.getFullYear() < 1971) return null
  return d
}

const pad2 = (n: number) => String(n).padStart(2, '0')

export const formatDateTime = (value: unknown): string => {
  const d = toDate(value)
  if (!d) return '—'
  return `${d.getFullYear()}-${pad2(d.getMonth() + 1)}-${pad2(d.getDate())} ${pad2(d.getHours())}:${pad2(d.getMinutes())}`
}

export const formatClock = (value: unknown): string => {
  const d = toDate(value)
  if (!d) return '—'
  return `${pad2(d.getHours())}:${pad2(d.getMinutes())}`
}

/** 时长（毫秒）→「2 小时 13 分」「48 分」「20 秒」「3 天 4 小时」 */
export const formatDuration = (ms: number, t: Translate): string => {
  const s = Math.max(0, Math.floor(ms / 1000))
  const d = Math.floor(s / 86400)
  const h = Math.floor((s % 86400) / 3600)
  const m = Math.floor((s % 3600) / 60)
  if (d > 0) return t('tools.time.dayHour', { d, h })
  if (h > 0) return t('tools.time.hourMin', { h, m })
  if (m > 0) return t('tools.time.min', { m })
  return t('tools.time.sec', { s })
}

/** 距今多久：「刚刚」「5 分钟前」「3 小时前」「2 天前」，超过 7 天显示日期 */
export const formatAgo = (value: unknown, t: Translate): string => {
  const d = toDate(value)
  if (!d) return '—'
  const s = Math.floor((Date.now() - d.getTime()) / 1000)
  if (s < 60) return t('tools.time.justNow', {})
  if (s < 3600) return t('tools.time.minAgo', { n: Math.floor(s / 60) })
  if (s < 86400) return t('tools.time.hourAgo', { n: Math.floor(s / 3600) })
  if (s < 7 * 86400) return t('tools.time.dayAgo', { n: Math.floor(s / 86400) })
  return formatDateTime(d)
}

/** 速率采样 → 迷你折线（SVG path，宽 w 高 h），全 0 时为底部直线 */
export const sparkPath = (values: number[] | undefined, w = 80, h = 28): { line: string; area: string } => {
  const data = (values || []).map((v) => Number(v) || 0)
  if (data.length < 2 || data.every((v) => v <= 0)) {
    const flat = `M0 ${h - 1} L${w} ${h - 1}`
    return { line: flat, area: '' }
  }
  const maxV = Math.max(...data) * 1.1
  const pts = data.map((v, k) => [(k / (data.length - 1)) * w, h - 1 - (v / maxV) * (h - 4)] as const)
  let d = `M${pts[0]![0].toFixed(1)} ${pts[0]![1].toFixed(1)}`
  for (let k = 1; k < pts.length; k++) {
    const [x0, y0] = pts[k - 1]!
    const [x1, y1] = pts[k]!
    const cx = ((x0 + x1) / 2).toFixed(1)
    d += ` C${cx} ${y0.toFixed(1)} ${cx} ${y1.toFixed(1)} ${x1.toFixed(1)} ${y1.toFixed(1)}`
  }
  return { line: d, area: `${d} L${w} ${h} L0 ${h} Z` }
}

/** 下载文本文件 */
export const downloadText = (filename: string, content: string, type = 'text/plain;charset=utf-8') => {
  const blob = new Blob([content], { type })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = filename
  a.click()
  URL.revokeObjectURL(url)
}

export const copyText = async (text: string) => {
  await navigator.clipboard.writeText(text)
}
