import { useThemeStore } from '@/stores/theme'

export interface PaletteColors {
  p: string
  pInk: string
  pSoft: string
  a: string
  a2: string
  ok: string
  warn: string
  bad: string
  ink: string
  ink2: string
  line: string
  s: string
  bg: string
}

const KEYS: (keyof PaletteColors)[] = [
  'p', 'pInk', 'pSoft', 'a', 'a2', 'ok', 'warn', 'bad', 'ink', 'ink2', 'line', 's', 'bg',
]

/**
 * 读取当前主题的实际色值（给 ECharts 这类只接受具体颜色、不认 CSS 变量的库用）。
 * 色值来自主题 store（含自定义主题的 token 覆盖与当前页面的单独设置），切换主题 / 深浅色 / 页面时自动更新。
 */
export const usePaletteColors = () => {
  const themeStore = useThemeStore()
  return computed<PaletteColors>(() => {
    const t = themeStore.tokens
    const out = {} as PaletteColors
    for (const k of KEYS) out[k] = t[k] || '#888888'
    return out
  })
}

/** #rrggbb + 透明度(0-1) → rgba()，非 6 位 hex 时原样返回 */
export const withAlpha = (hex: string, alpha: number) => {
  const m = /^#([0-9a-f]{6})$/i.exec(hex)
  if (!m) return hex
  const n = parseInt(m[1]!, 16)
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`
}
