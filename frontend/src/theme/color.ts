// 颜色小工具：hex / rgb / hsv 互转、混色、WCAG 对比度与自动修正。

export type RGB = [number, number, number]

export const HEX_RE = /^#[0-9a-f]{6}$/i

export const isHex = (v: unknown): v is string => typeof v === 'string' && HEX_RE.test(v)

export const normalizeHex = (v: string): string | null => {
  let s = v.trim()
  if (!s.startsWith('#')) s = `#${s}`
  if (/^#[0-9a-f]{3}$/i.test(s)) s = `#${s[1]}${s[1]}${s[2]}${s[2]}${s[3]}${s[3]}`
  return HEX_RE.test(s) ? s.toUpperCase() : null
}

export const hexToRgb = (hex: string): RGB => {
  const n = parseInt(hex.slice(1, 7), 16)
  return [(n >> 16) & 255, (n >> 8) & 255, n & 255]
}

const clamp255 = (n: number) => Math.max(0, Math.min(255, Math.round(n)))

export const rgbToHex = ([r, g, b]: RGB): string =>
  `#${[r, g, b].map((c) => clamp255(c).toString(16).padStart(2, '0')).join('')}`.toUpperCase()

/** a 占 weight（0-1），其余为 b */
export const mixHex = (a: string, b: string, weight: number): string => {
  const x = hexToRgb(a)
  const y = hexToRgb(b)
  return rgbToHex([0, 1, 2].map((i) => x[i]! * weight + y[i]! * (1 - weight)) as RGB)
}

const channel = (c: number) => {
  const s = c / 255
  return s <= 0.03928 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4
}

export const luminance = (hex: string) => {
  const [r, g, b] = hexToRgb(hex)
  return 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)
}

export const luminanceRgb = ([r, g, b]: RGB) => 0.2126 * channel(r) + 0.7152 * channel(g) + 0.0722 * channel(b)

export const contrastRatio = (a: string, b: string) => {
  const la = luminance(a)
  const lb = luminance(b)
  return (Math.max(la, lb) + 0.05) / (Math.min(la, lb) + 0.05)
}

export const formatRatio = (r: number) => `${(Math.floor(r * 10) / 10).toFixed(1)} : 1`

/**
 * 调整 fg 直到与 bg 的对比度 ≥ target：朝黑或白（取能达标、改动更小的方向）逐步混合。
 */
export const ensureContrast = (fg: string, bg: string, target = 4.5): string => {
  if (contrastRatio(fg, bg) >= target) return fg
  const tryToward = (end: string) => {
    for (let w = 0.95; w >= 0; w -= 0.05) {
      const c = mixHex(fg, end, w)
      if (contrastRatio(c, bg) >= target) return { c, cost: 1 - w }
    }
    return null
  }
  const dark = tryToward('#000000')
  const light = tryToward('#FFFFFF')
  if (dark && light) return dark.cost <= light.cost ? dark.c : light.c
  return (dark || light)?.c ?? (luminance(bg) > 0.4 ? '#000000' : '#FFFFFF')
}

export interface HSV {
  h: number
  s: number
  v: number
}

export const hexToHsv = (hex: string): HSV => {
  const [r, g, b] = hexToRgb(hex).map((c) => c / 255) as RGB
  const mx = Math.max(r, g, b)
  const mn = Math.min(r, g, b)
  const d = mx - mn
  let h = 0
  if (d) {
    if (mx === r) h = ((g - b) / d) % 6
    else if (mx === g) h = (b - r) / d + 2
    else h = (r - g) / d + 4
  }
  return { h: (h * 60 + 360) % 360, s: mx ? d / mx : 0, v: mx }
}

export const hsvToHex = ({ h, s, v }: HSV): string => {
  const c = v * s
  const x = c * (1 - Math.abs(((h / 60) % 2) - 1))
  const m = v - c
  let rgb: [number, number, number]
  if (h < 60) rgb = [c, x, 0]
  else if (h < 120) rgb = [x, c, 0]
  else if (h < 180) rgb = [0, c, x]
  else if (h < 240) rgb = [0, x, c]
  else if (h < 300) rgb = [x, 0, c]
  else rgb = [c, 0, x]
  return rgbToHex(rgb.map((n) => (n + m) * 255) as RGB)
}

/** 解析 getComputedStyle 给出的颜色（rgb()/rgba()/color(srgb …)），失败返回 null；alpha 为 0 视为无色 */
export const parseCssColor = (value: string): { rgb: RGB; alpha: number } | null => {
  const v = value.trim()
  let m = /^rgba?\(\s*([\d.]+)[,\s]+([\d.]+)[,\s]+([\d.]+)(?:\s*[,/]\s*([\d.]+%?))?\s*\)$/i.exec(v)
  if (m) {
    const a = m[4] == null ? 1 : m[4].endsWith('%') ? parseFloat(m[4]) / 100 : parseFloat(m[4])
    return { rgb: [Number(m[1]), Number(m[2]), Number(m[3])], alpha: a }
  }
  m = /^color\(srgb\s+([\d.e-]+)\s+([\d.e-]+)\s+([\d.e-]+)(?:\s*\/\s*([\d.]+%?))?\s*\)$/i.exec(v)
  if (m) {
    const a = m[4] == null ? 1 : m[4].endsWith('%') ? parseFloat(m[4]) / 100 : parseFloat(m[4])
    return { rgb: [Number(m[1]) * 255, Number(m[2]) * 255, Number(m[3]) * 255], alpha: a }
  }
  return null
}

export const rgbDistance = (a: RGB, b: RGB) => Math.max(Math.abs(a[0] - b[0]), Math.abs(a[1] - b[1]), Math.abs(a[2] - b[2]))
