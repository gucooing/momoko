// 主题内容模型：默认值 / 规范化 / 按页面与深浅色求出实际生效的 token、背景、横幅，以及生成 CSS。
// 主题 = 配色（浅色 + 深色 token 覆盖）+ 背景 + 细节（shape）+ 横幅（banner）+ 页面单独设置（pages）。
import type {
  ThemeBackground,
  ThemeBanner,
  ThemeContent,
  ThemePageOverride,
  ThemeShape,
} from '@/types/v1/theme'
import {
  DERIVED_TOKENS,
  PALETTE_TOKENS,
  TERM_DEFAULTS,
  TERM_TOKEN_KEYS,
  UI_TOKEN_KEYS,
  isPaletteKey,
  type PaletteKey,
  type SchemeKey,
  type TokenKey,
  type UiTokenKey,
} from './palettes'
import { contrastRatio, isHex, mixHex, luminance, ensureContrast } from './color'
import { SIZE_KEY_RE } from './sizeTokens'

export type { ThemeBackground, ThemeBanner, ThemeContent, ThemePageOverride, ThemeShape }

export const BG_TYPES = ['theme', 'solid', 'pattern', 'gradient', 'image'] as const
export type BgType = (typeof BG_TYPES)[number]
export const PATTERNS = ['dots', 'grid', 'stripe', 'polka', 'wave'] as const
export const PATTERN_COLORS = ['ink', 'p', 'a2'] as const
export const FITS = ['cover', 'contain', 'tile', 'center'] as const
export const FONTS = ['outfit', 'system', 'rounded', 'serif', 'mono'] as const
export const FONT_SIZES = ['s', 'm', 'l'] as const
export const SHADOWS = ['none', 'soft', 'strong'] as const
export const BUTTON_SHAPES = ['pill', 'round', 'square'] as const
export const SIDEBAR_STYLES = ['solid', 'clear', 'glass'] as const
export const PILL_STYLES = ['dot', 'fill'] as const
export const BANNER_TYPES = ['color', 'gradient', 'image'] as const
export const BANNER_INKS = ['auto', 'light', 'dark'] as const

/** 渐变预设（外观页的色块） */
export const GRADIENT_PRESETS: { key: string; stops: string[]; angle: number }[] = [
  { key: 'peach', stops: ['#FFD3E2', '#FFE9C7', '#D9E4FF'], angle: 135 },
  { key: 'lagoon', stops: ['#C9F2E6', '#D7E8FF', '#EBDDFF'], angle: 160 },
  { key: 'candy', stops: ['#FFB8D2', '#C8B6FF'], angle: 120 },
  { key: 'lemon', stops: ['#FFF4B8', '#C9F5D9'], angle: 150 },
  { key: 'berry', stops: ['#2B1747', '#5B1F4F', '#1B2350'], angle: 135 },
  { key: 'ocean', stops: ['#0E2340', '#13405A', '#1D1A45'], angle: 170 },
]

export const SOLID_PRESETS = ['#FFF6F9', '#F4F1FF', '#EFF8F3', '#FFF7EC', '#EEF3FA', '#1A1424', '#101820']

/** 横幅渐变预设（跟随主题色） */
export const BANNER_GRADIENTS = [
  'linear-gradient(120deg, var(--m-p), var(--m-a2))',
  'linear-gradient(120deg, var(--m-a2), var(--m-ok))',
  'linear-gradient(120deg, var(--m-p), var(--m-a))',
]

export const FONT_STACKS: Record<(typeof FONTS)[number], string> = {
  outfit: "'Outfit Variable', 'Outfit', 'HarmonyOS Sans SC', 'PingFang SC', 'Microsoft YaHei', 'Noto Sans SC', system-ui, sans-serif",
  system: "system-ui, -apple-system, 'Segoe UI', 'PingFang SC', 'Microsoft YaHei', 'Noto Sans SC', sans-serif",
  rounded: "'Varela Round', 'Nunito', 'Arial Rounded MT Bold', 'HarmonyOS Sans SC', 'PingFang SC', 'Microsoft YaHei', system-ui, sans-serif",
  serif: "'Noto Serif SC', 'Source Han Serif SC', 'Songti SC', 'SimSun', Georgia, 'Times New Roman', serif",
  mono: "'JetBrains Mono', 'Cascadia Code', ui-monospace, 'Sarasa Mono SC', Consolas, monospace",
}

export const DEFAULT_RADIUS = 22
export const GLASS_TYPES: BgType[] = ['image', 'gradient']

const pick = <T extends string>(v: unknown, list: readonly T[], fallback: T): T =>
  list.includes(v as T) ? (v as T) : fallback
const clampInt = (v: unknown, min: number, max: number, fallback: number) => {
  const n = typeof v === 'number' && Number.isFinite(v) ? Math.round(v) : fallback
  return Math.max(min, Math.min(max, n))
}
const cleanTokens = (raw: unknown): Record<string, string> => {
  const out: Record<string, string> = {}
  if (!raw || typeof raw !== 'object') return out
  for (const [k, v] of Object.entries(raw as Record<string, unknown>)) {
    if ((UI_TOKEN_KEYS as readonly string[]).includes(k) || (TERM_TOKEN_KEYS as readonly string[]).includes(k)) {
      if (isHex(v)) out[k] = v.toUpperCase()
    }
  }
  return out
}

/** 组件尺寸覆盖：键符合命名规则、值为有限数字（未登记的键保留，可能来自更新版本的主题包） */
const cleanSizes = (raw: unknown): Record<string, number> => {
  const out: Record<string, number> = {}
  if (!raw || typeof raw !== 'object') return out
  for (const [k, v] of Object.entries(raw as Record<string, unknown>)) {
    const n = typeof v === 'string' ? Number(v) : v
    if (SIZE_KEY_RE.test(k) && typeof n === 'number' && Number.isFinite(n)) out[k] = Math.round(n * 2) / 2
  }
  return out
}

// ---------------------------------------------------------------------------
// 默认值与规范化
// ---------------------------------------------------------------------------

export const emptyBackground = (): ThemeBackground => ({
  type: 'theme',
  color: '',
  pattern: '',
  patternColor: '',
  gradientStops: [],
  gradientAngle: 0,
  gradientKind: '',
  gradientFlow: false,
  image: '',
  fit: '',
  blur: 0,
  dim: 0,
  cardOpacity: 100,
  autoDarkDim: false,
})

/** 切换背景类型时的初始值（保留同类型下已有的选择） */
export const backgroundOfType = (type: BgType, prev?: ThemeBackground | null): ThemeBackground => {
  const base = { ...emptyBackground(), type }
  const p = prev && prev.type === type ? prev : null
  switch (type) {
    case 'solid':
      return { ...base, color: p?.color || SOLID_PRESETS[1]! }
    case 'pattern':
      return { ...base, pattern: p?.pattern || 'dots', patternColor: p?.patternColor || 'ink' }
    case 'gradient': {
      const g = GRADIENT_PRESETS[0]!
      return {
        ...base,
        gradientStops: p?.gradientStops?.length ? [...p.gradientStops] : [...g.stops],
        gradientAngle: p?.gradientAngle ?? g.angle,
        gradientKind: p?.gradientKind || 'linear',
        gradientFlow: p?.gradientFlow ?? false,
        dim: p?.dim ?? 10,
        cardOpacity: p?.cardOpacity || 76,
      }
    }
    case 'image':
      return {
        ...base,
        image: p?.image || prev?.image || '',
        fit: p?.fit || 'cover',
        blur: p?.blur ?? 0,
        dim: p?.dim ?? 22,
        cardOpacity: p?.cardOpacity || 76,
        autoDarkDim: p ? p.autoDarkDim : true,
      }
    default:
      return base
  }
}

export const normalizeBackground = (raw?: Partial<ThemeBackground> | null): ThemeBackground => {
  if (!raw) return emptyBackground()
  let type = pick(raw.type, BG_TYPES, 'theme')
  if (type === 'image' && !raw.image) type = 'theme'
  const glass = GLASS_TYPES.includes(type)
  const stops = (raw.gradientStops || []).filter(isHex).slice(0, 5)
  return {
    type,
    color: isHex(raw.color) ? raw.color.toUpperCase() : type === 'solid' ? SOLID_PRESETS[1]! : '',
    pattern: pick(raw.pattern, PATTERNS, 'dots'),
    patternColor: pick(raw.patternColor, PATTERN_COLORS, 'ink'),
    gradientStops: stops.length >= 2 ? stops : [...GRADIENT_PRESETS[0]!.stops],
    gradientAngle: clampInt(raw.gradientAngle, 0, 360, 135),
    gradientKind: pick(raw.gradientKind, ['linear', 'radial'] as const, 'linear'),
    gradientFlow: !!raw.gradientFlow,
    image: typeof raw.image === 'string' ? raw.image : '',
    fit: pick(raw.fit, FITS, 'cover'),
    blur: clampInt(raw.blur, 0, 40, 0),
    dim: clampInt(raw.dim, 0, 80, 0),
    cardOpacity: raw.cardOpacity ? clampInt(raw.cardOpacity, 40, 100, 100) : glass ? 76 : 100,
    autoDarkDim: !!raw.autoDarkDim,
  }
}

export const emptyShape = (): ThemeShape => ({
  radius: undefined,
  font: 'outfit',
  fontAsset: '',
  fontName: '',
  fontSize: 'm',
  shadow: 'none',
  button: 'pill',
  sidebar: 'glass',
  pill: 'dot',
  cardBorder: undefined,
  stickerTilt: undefined,
})

export const normalizeShape = (raw?: Partial<ThemeShape> | null): ThemeShape => {
  const s = raw || {}
  const font = s.font === 'custom' && s.fontAsset ? 'custom' : pick(s.font, FONTS, 'outfit')
  return {
    radius: typeof s.radius === 'number' ? clampInt(s.radius, 0, 32, DEFAULT_RADIUS) : undefined,
    font,
    fontAsset: font === 'custom' ? s.fontAsset || '' : '',
    fontName: font === 'custom' ? (s.fontName || '').slice(0, 40) : '',
    fontSize: pick(s.fontSize, FONT_SIZES, 'm'),
    shadow: pick(s.shadow, SHADOWS, 'none'),
    button: pick(s.button, BUTTON_SHAPES, 'pill'),
    sidebar: pick(s.sidebar, SIDEBAR_STYLES, 'glass'),
    pill: pick(s.pill, PILL_STYLES, 'dot'),
    cardBorder: typeof s.cardBorder === 'boolean' ? s.cardBorder : undefined,
    stickerTilt: typeof s.stickerTilt === 'boolean' ? s.stickerTilt : undefined,
  }
}

export const emptyBanner = (): ThemeBanner => ({
  type: 'color',
  gradient: 0,
  image: '',
  dim: 35,
  ink: 'auto',
  focusX: 50,
  focusY: 50,
})

export const normalizeBanner = (raw?: Partial<ThemeBanner> | null): ThemeBanner => {
  if (!raw) return emptyBanner()
  let type = pick(raw.type, BANNER_TYPES, 'color')
  if (type === 'image' && !raw.image) type = 'color'
  return {
    type,
    gradient: clampInt(raw.gradient, 0, BANNER_GRADIENTS.length - 1, 0),
    image: typeof raw.image === 'string' ? raw.image : '',
    dim: clampInt(raw.dim, 0, 80, 35),
    ink: pick(raw.ink, BANNER_INKS, 'auto'),
    focusX: clampInt(raw.focusX, 0, 100, 50),
    focusY: clampInt(raw.focusY, 0, 100, 50),
  }
}

export const emptyPage = (): ThemePageOverride => ({
  tokensLight: {},
  tokensDark: {},
  background: undefined,
  cardOpacity: undefined,
  banner: undefined,
  sizes: {},
})

export const normalizePage = (raw?: Partial<ThemePageOverride> | null): ThemePageOverride => ({
  tokensLight: cleanTokens(raw?.tokensLight),
  tokensDark: cleanTokens(raw?.tokensDark),
  background: raw?.background ? normalizeBackground(raw.background) : undefined,
  cardOpacity: typeof raw?.cardOpacity === 'number' ? clampInt(raw.cardOpacity, 40, 100, 100) : undefined,
  banner: raw?.banner ? normalizeBanner(raw.banner) : undefined,
  sizes: cleanSizes(raw?.sizes),
})

export const isPageEmpty = (p?: ThemePageOverride | null) =>
  !p ||
  (!Object.keys(p.tokensLight || {}).length &&
    !Object.keys(p.tokensDark || {}).length &&
    !p.background &&
    p.cardOpacity == null &&
    !p.banner &&
    !Object.keys(p.sizes || {}).length)

export const emptyContent = (base: PaletteKey = 'sakura'): ThemeContent => ({
  base,
  tokensLight: {},
  tokensDark: {},
  background: emptyBackground(),
  shape: emptyShape(),
  banner: emptyBanner(),
  pages: {},
  sizes: {},
})

export const normalizeContent = (raw?: Partial<ThemeContent> | null): ThemeContent => {
  const c = raw || {}
  const pages: Record<string, ThemePageOverride> = {}
  for (const [k, v] of Object.entries(c.pages || {})) {
    if (!k.startsWith('/')) continue
    const page = normalizePage(v)
    if (!isPageEmpty(page)) pages[k] = page
  }
  return {
    base: isPaletteKey(c.base) ? c.base : 'sakura',
    tokensLight: cleanTokens(c.tokensLight),
    tokensDark: cleanTokens(c.tokensDark),
    background: normalizeBackground(c.background),
    shape: normalizeShape(c.shape),
    banner: normalizeBanner(c.banner),
    pages,
    sizes: cleanSizes(c.sizes),
  }
}

export const cloneContent = (c: ThemeContent): ThemeContent => JSON.parse(JSON.stringify(c)) as ThemeContent

/** 页面单独设置的条数（token 按当前深浅色计） */
export const pageOverrideCount = (p: ThemePageOverride | undefined, scheme?: SchemeKey) => {
  if (!p) return 0
  const toks = scheme
    ? Object.keys(scheme === 'dark' ? p.tokensDark : p.tokensLight).length
    : new Set([...Object.keys(p.tokensLight), ...Object.keys(p.tokensDark)]).size
  return toks + (p.background ? 1 : 0) + (p.cardOpacity != null ? 1 : 0) + (p.banner ? 1 : 0) + Object.keys(p.sizes || {}).length
}

export const pagesWithOverrides = (c: ThemeContent) => Object.keys(c.pages).filter((k) => !isPageEmpty(c.pages[k]))

// ---------------------------------------------------------------------------
// 求出实际生效的值
// ---------------------------------------------------------------------------

export type ResolvedTokens = Record<TokenKey, string>

const derive = (key: UiTokenKey, from: string, t: Record<string, string>, scheme: SchemeKey): string => {
  const s = t.s!
  const ink = t.ink!
  switch (key) {
    case 'pInk':
      return scheme === 'dark' ? mixHex(from, '#FFFFFF', 0.72) : ensureContrast(mixHex(from, ink, 0.9), s, 4.5)
    case 'aOn':
      return luminance(from) > 0.45 ? (scheme === 'dark' ? t.bg! : ink) : '#FFFFFF'
    default:
      return mixHex(from, s, scheme === 'dark' ? 0.22 : 0.16)
  }
}

/** 全局 token 覆盖 + 页面覆盖 → 完整色表（含派生色和终端色） */
export const resolveTokens = (content: ThemeContent, scheme: SchemeKey, pageKey?: string | null): ResolvedTokens => {
  const base = PALETTE_TOKENS[isPaletteKey(content.base) ? content.base : 'sakura'][scheme]
  const page = pageKey ? content.pages[pageKey] : undefined
  const over: Record<string, string> = {
    ...(scheme === 'dark' ? content.tokensDark : content.tokensLight),
    ...(page ? (scheme === 'dark' ? page.tokensDark : page.tokensLight) : {}),
  }
  const out: Record<string, string> = { ...base, ...TERM_DEFAULTS, ...over }
  for (const d of DERIVED_TOKENS) {
    if (over[d.key] == null && over[d.from] != null) out[d.key] = derive(d.key, out[d.from]!, out, scheme)
  }
  return out as ResolvedTokens
}

/** 生效的背景：页面单独设置 > 外观页未保存的修改 > 主题背景 */
export const resolveBackground = (
  content: ThemeContent,
  pageKey?: string | null,
  override?: ThemeBackground | null,
): { bg: ThemeBackground; cardOpacity: number; fromPage: boolean } => {
  const page = pageKey ? content.pages[pageKey] : undefined
  const bg = normalizeBackground(page?.background || override || content.background)
  const cardOpacity = page?.cardOpacity ?? bg.cardOpacity
  return { bg, cardOpacity: GLASS_TYPES.includes(bg.type as BgType) ? cardOpacity : Math.max(cardOpacity, 100), fromPage: !!page?.background }
}

/** 生效的组件尺寸覆盖：全局 + 当前页面 */
export const resolveSizes = (content: ThemeContent, pageKey?: string | null): Record<string, number> => {
  const page = pageKey ? content.pages[pageKey] : undefined
  return { ...(content.sizes || {}), ...(page?.sizes || {}) }
}

export const resolveBanner = (content: ThemeContent, pageKey?: string | null): ThemeBanner => {
  const page = pageKey ? content.pages[pageKey] : undefined
  return normalizeBanner(page?.banner || content.banner)
}

/** 深色模式下自动加深遮罩 */
export const effectiveDim = (bg: ThemeBackground, scheme: SchemeKey) =>
  bg.autoDarkDim && scheme === 'dark' ? Math.min(80, Math.round(bg.dim * 1.4 + 14)) : bg.dim

// ---------------------------------------------------------------------------
// CSS
// ---------------------------------------------------------------------------

export type AssetResolver = (ref: string) => string

const API_BASE = (import.meta.env.VITE_API_BASE_URL as string | undefined) || '/api/v1'
export const ASSET_ID_RE = /^[0-9a-f]{32}\.(webp|woff2)$/

/** 资源引用 → 可用的 URL（资源 id / data URL / 导入预览里的包内路径） */
export const assetUrl = (ref: string, resolver?: AssetResolver): string => {
  if (!ref) return ''
  if (resolver) {
    const r = resolver(ref)
    if (r) return r
  }
  if (/^(data|blob):/.test(ref) || /^https?:\/\//.test(ref)) return ref
  if (ASSET_ID_RE.test(ref)) return `${API_BASE.replace(/\/$/, '')}/theme-asset/${ref}`
  return ''
}

const patternColorCss = (key: string, strength = 1) => {
  const k = key === 'p' || key === 'a2' ? key : 'ink'
  const pct = Math.round((k === 'ink' ? 14 : 30) * strength)
  return `color-mix(in srgb, var(--m-${k}) ${pct}%, transparent)`
}

export const patternCss = (pattern: string, color: string) => {
  switch (pattern) {
    case 'grid':
      return `linear-gradient(${color} 1px, transparent 1px) 0 0 / 26px 26px, linear-gradient(90deg, ${color} 1px, transparent 1px) 0 0 / 26px 26px`
    case 'stripe':
      return `repeating-linear-gradient(45deg, ${color} 0 1.5px, transparent 1.5px 14px)`
    case 'polka':
      return `radial-gradient(${color} 5px, transparent 5.5px) 0 0 / 34px 34px, radial-gradient(${color} 5px, transparent 5.5px) 17px 17px / 34px 34px`
    case 'wave':
      return `radial-gradient(circle at 50% 100%, transparent 9px, ${color} 9.5px 10.5px, transparent 11px) 0 0 / 24px 12px`
    default:
      return `radial-gradient(${color} 1.6px, transparent 1.8px) 0 0 / 18px 18px`
  }
}

export const gradientCss = (stops: string[], angle: number, kind: string) =>
  kind === 'radial'
    ? `radial-gradient(120% 90% at 20% 10%, ${stops.join(', ')})`
    : `linear-gradient(${angle}deg, ${stops.join(', ')})`

/** 背景层的 CSS background 简写（不含模糊、遮罩） */
export const backgroundCss = (bg: ThemeBackground, resolver?: AssetResolver, patternStrength = 1): string => {
  switch (bg.type) {
    case 'solid':
      return bg.color || 'var(--m-bg)'
    case 'pattern':
      return `${patternCss(bg.pattern, patternColorCss(bg.patternColor, patternStrength))}, var(--m-bg)`
    case 'gradient':
      return gradientCss(bg.gradientStops, bg.gradientAngle, bg.gradientKind)
    case 'image': {
      const url = assetUrl(bg.image, resolver)
      if (!url) return 'var(--m-bg)'
      const u = `url("${url}")`
      switch (bg.fit) {
        case 'contain':
          return `${u} center / contain no-repeat, var(--m-bg)`
        case 'tile':
          return `${u} 0 0 / auto repeat, var(--m-bg)`
        case 'center':
          return `${u} center / auto no-repeat, var(--m-bg)`
        default:
          return `${u} center / cover no-repeat, var(--m-bg)`
      }
    }
    default:
      return 'var(--m-bg)'
  }
}

/** 完整色表 → CSS 变量声明（--m-p: #…;） */
export const tokensToCssVars = (tokens: Partial<ResolvedTokens>, keys: readonly string[] = [...UI_TOKEN_KEYS, ...TERM_TOKEN_KEYS]) =>
  keys
    .filter((k) => tokens[k as TokenKey])
    .map((k) => `--m-${k}: ${tokens[k as TokenKey]};`)
    .join(' ')

/** 「复制主题代码」：浅色 / 深色两套 CSS 变量 */
export const themeCssCode = (name: string, content: ThemeContent) => {
  const block = (sel: string, scheme: SchemeKey) => {
    const t = resolveTokens(content, scheme)
    const lines = [...UI_TOKEN_KEYS, ...TERM_TOKEN_KEYS].map((k) => `  --m-${k}: ${t[k]};`)
    return `${sel} {\n${lines.join('\n')}\n}`
  }
  return `/* Momoko theme: ${name.replace(/\*\//g, '')} */\n${block(':root', 'light')}\n\n${block('html.dark', 'dark')}\n`
}

// ---------------------------------------------------------------------------
// 横幅
// ---------------------------------------------------------------------------

export interface BannerLook {
  background: string
  /** 文字颜色 */
  fg: string
  /** 压暗层 */
  shade: string
  /** 是否显示主色横幅的装饰圆 */
  deco: boolean
  /** 主按钮底色 / 文字色 */
  btnBg: string
  btnFg: string
  /** 横幅上的浅色半透明元素（次按钮、标签）用的颜色 */
  soft: string
  isImage: boolean
}

/**
 * @param luma 图片左侧（文字所在区域）平均亮度 0-1，未知时传 null
 */
export const bannerLook = (b: ThemeBanner, resolver?: AssetResolver, luma?: number | null): BannerLook => {
  if (b.type === 'gradient') {
    return {
      background: BANNER_GRADIENTS[b.gradient] || BANNER_GRADIENTS[0]!,
      fg: '#FFFFFF',
      shade: 'transparent',
      deco: false,
      btnBg: '#FFFFFF',
      btnFg: 'var(--m-p)',
      soft: '#FFFFFF',
      isImage: false,
    }
  }
  if (b.type === 'image') {
    const url = assetUrl(b.image, resolver)
    const dark = b.ink === 'dark' || (b.ink === 'auto' && luma != null && luma > 0.62 && b.dim < 30)
    const fg = dark ? '#1A1020' : '#FFFFFF'
    const a = b.dim / 100
    return {
      background: url ? `url("${url}") ${b.focusX}% ${b.focusY}% / cover no-repeat, var(--m-p)` : 'var(--m-p)',
      fg,
      shade: dark
        ? `linear-gradient(90deg, rgba(255,255,255,${a}), transparent 75%)`
        : `linear-gradient(90deg, rgba(12,6,24,${a}) 0%, rgba(12,6,24,${(a / 2.5).toFixed(3)}) 65%, transparent)`,
      deco: false,
      btnBg: fg,
      btnFg: dark ? '#FFFFFF' : 'var(--m-p)',
      soft: fg,
      isImage: true,
    }
  }
  return {
    background: 'var(--m-p)',
    fg: 'var(--m-pOn)',
    shade: 'transparent',
    deco: true,
    btnBg: 'var(--m-pOn)',
    btnFg: 'var(--m-p)',
    soft: 'var(--m-pOn)',
    isImage: false,
  }
}

// ---------------------------------------------------------------------------
// 对比度检查
// ---------------------------------------------------------------------------

export interface ContrastCheck {
  /** i18n 键：theme.checks.<key> */
  key: 'body' | 'muted' | 'button' | 'pill' | 'term' | 'backdrop'
  ratio: number
  ok: boolean
  /** 一键修正时改哪个 token（backdrop 改遮罩） */
  fix: TokenKey | 'dim' | null
}

export const MIN_CONTRAST = 4.5

/**
 * @param backdropBright 背景最亮处（已叠加遮罩）的颜色，只有图片 / 渐变背景时给
 */
export const contrastChecks = (
  t: ResolvedTokens,
  opts: { backdropBright?: string | null; terminal?: boolean } = {},
): ContrastCheck[] => {
  const mk = (key: ContrastCheck['key'], a: string, b: string, fix: ContrastCheck['fix']): ContrastCheck => {
    const ratio = contrastRatio(a, b)
    return { key, ratio, ok: ratio >= MIN_CONTRAST, fix }
  }
  const list = [
    mk('body', t.ink, t.s, 'ink'),
    mk('muted', t.ink2, t.s, 'ink2'),
    mk('button', t.pOn, t.p, 'pOn'),
    opts.terminal ? mk('term', t.termFg, t.termBg, 'termFg') : mk('pill', t.ok, t.okSoft, 'ok'),
  ]
  if (opts.backdropBright) list.push(mk('backdrop', t.ink, opts.backdropBright, 'dim'))
  return list
}

/** 一键修正某一项：返回要写入的 token 值（按钮文字优先在黑白之间选，选不出再调主色） */
export const fixContrast = (check: ContrastCheck, t: ResolvedTokens): { key: TokenKey; value: string }[] => {
  switch (check.key) {
    case 'body':
      return [{ key: 'ink', value: ensureContrast(t.ink, t.s) }]
    case 'muted':
      return [{ key: 'ink2', value: ensureContrast(t.ink2, t.s) }]
    case 'button': {
      const white = contrastRatio('#FFFFFF', t.p)
      const black = contrastRatio('#140A1E', t.p)
      if (Math.max(white, black) >= MIN_CONTRAST) return [{ key: 'pOn', value: white >= black ? '#FFFFFF' : '#140A1E' }]
      const on = white >= black ? '#FFFFFF' : '#140A1E'
      return [
        { key: 'pOn', value: on },
        { key: 'p', value: ensureContrast(t.p, on) },
      ]
    }
    case 'pill':
      return [{ key: 'ok', value: ensureContrast(t.ok, t.okSoft) }]
    case 'term':
      return [{ key: 'termFg', value: ensureContrast(t.termFg, t.termBg) }]
    default:
      return []
  }
}

/** 背景最亮处叠上遮罩后的颜色 */
export const backdropBrightColor = (bright: string, bg: string, dim: number) => mixHex(bg, bright, dim / 100)

/** 渐变背景最亮的色标 */
export const brightestStop = (stops: string[]) =>
  stops.reduce((best, c) => (luminance(c) > luminance(best) ? c : best), stops[0] || '#FFFFFF')

/** 让「文字 / 背景最亮处」达到 4.5 所需的最小遮罩 */
export const dimForContrast = (ink: string, bright: string, bg: string) => {
  for (let d = 0; d <= 80; d += 2) {
    if (contrastRatio(ink, backdropBrightColor(bright, bg, d)) >= MIN_CONTRAST) return d
  }
  return 80
}

export const UI_KEYS = UI_TOKEN_KEYS
export const TERM_KEYS = TERM_TOKEN_KEYS
