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

const read = (): PaletteColors => {
  const style = getComputedStyle(document.documentElement)
  const out = {} as PaletteColors
  for (const k of KEYS) out[k] = style.getPropertyValue(`--m-${k}`).trim() || '#888888'
  return out
}

/**
 * 读取当前配色的实际色值（给 ECharts 这类只接受具体颜色、不认 CSS 变量的库用）。
 * 配色或深浅色切换后自动刷新。
 */
export const usePaletteColors = () => {
  const themeStore = useThemeStore()
  const colors = ref<PaletteColors>(read())
  watch(
    () => [themeStore.palette, themeStore.isDarkTheme],
    () => nextTick(() => (colors.value = read())),
  )
  return colors
}

/** #rrggbb + 透明度(0-1) → rgba()，非 6 位 hex 时原样返回 */
export const withAlpha = (hex: string, alpha: number) => {
  const m = /^#([0-9a-f]{6})$/i.exec(hex)
  if (!m) return hex
  const n = parseInt(m[1]!, 16)
  return `rgba(${(n >> 16) & 255}, ${(n >> 8) & 255}, ${n & 255}, ${alpha})`
}
