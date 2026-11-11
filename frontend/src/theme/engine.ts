// 把一份主题内容套到整个文档上：
//   · <html data-palette> 选中所基于的内置配色（CSS 里已有完整色值）；
//   · 有 token 覆盖（全局或当前页面）时注入一段 <style>，写出浅色 / 深色两套完整 --m-* 色值；
//   · 细节（圆角、字体、字号、阴影、按钮形状、侧栏、状态标签、卡片描边、贴纸旋转）写成 CSS 变量和 <html> 类名；
//   · 组件尺寸（theme/sizeTokens.ts 注册表）写成 --cmp-* 变量（默认值在模块加载时写到 :root）；
//   · 背景只切换 has-theme-bg / is-glass 类与卡片不透明度，真正的背景层由 components/theme/ThemeBackdrop.vue 绘制。
import type { ThemeBackground, ThemeContent } from '@/types/v1/theme'
import {
  DEFAULT_RADIUS,
  FONT_STACKS,
  GLASS_TYPES,
  assetUrl,
  resolveBackground,
  resolveSizes,
  resolveTokens,
  tokensToCssVars,
  type AssetResolver,
  type BgType,
} from './content'
import { UI_TOKEN_KEYS } from './palettes'
import { sizeDefaultsCss, sizesToCssVars } from './sizeTokens'

const STYLE_ID = 'momoko-theme-runtime'
const SIZE_DEFAULTS_ID = 'momoko-size-defaults'

/** 组件尺寸默认值（来自 theme/sizeTokens.ts 注册表）在模块加载时写到 :root 上，组件样式直接用 var(--cmp-*) */
if (typeof document !== 'undefined' && !document.getElementById(SIZE_DEFAULTS_ID)) {
  const el = document.createElement('style')
  el.id = SIZE_DEFAULTS_ID
  el.textContent = sizeDefaultsCss()
  document.head.prepend(el)
}
const CUSTOM_FONT_FAMILY = 'Momoko Theme Font'

export interface ApplyThemeInput {
  content: ThemeContent
  pageKey: string | null
  override: ThemeBackground | null
  radiusPref: 'round' | 'standard'
  resolver?: AssetResolver
}

const hasTokenOverrides = (c: ThemeContent, pageKey: string | null) => {
  if (Object.keys(c.tokensLight).length || Object.keys(c.tokensDark).length) return true
  const page = pageKey ? c.pages[pageKey] : undefined
  return !!page && (Object.keys(page.tokensLight).length > 0 || Object.keys(page.tokensDark).length > 0)
}

const ensureStyle = () => {
  let el = document.getElementById(STYLE_ID) as HTMLStyleElement | null
  if (!el) {
    el = document.createElement('style')
    el.id = STYLE_ID
    document.head.appendChild(el)
  }
  return el
}

let loadedFont = ''
const loadCustomFont = (url: string) => {
  if (!url || loadedFont === url || typeof FontFace === 'undefined') return
  loadedFont = url
  const face = new FontFace(CUSTOM_FONT_FAMILY, `url("${url}") format("woff2")`, { display: 'swap' })
  face
    .load()
    .then((f) => document.fonts.add(f))
    .catch(() => {
      loadedFont = ''
    })
}

const px = (n: number) => `${Math.round(n * 10) / 10}px`

const SHADOWS: Record<string, string> = {
  soft: '0 2px 6px -2px color-mix(in srgb, var(--m-ink) 10%, transparent), 0 10px 24px -14px color-mix(in srgb, var(--m-ink) 22%, transparent)',
  strong: '0 4px 10px -4px color-mix(in srgb, var(--m-ink) 18%, transparent), 0 18px 40px -16px color-mix(in srgb, var(--m-ink) 38%, transparent)',
}

const ROOT_CLASSES = [
  'theme-btn-round',
  'theme-btn-square',
  'theme-shadow-soft',
  'theme-shadow-strong',
  'theme-sidebar-solid',
  'theme-sidebar-clear',
  'theme-sidebar-glass',
  'theme-pill-fill',
  'theme-no-card-border',
  'theme-no-tilt',
  'has-theme-bg',
  'is-glass',
]

export const applyThemeToDocument = ({ content, pageKey, override, radiusPref, resolver }: ApplyThemeInput) => {
  const root = document.documentElement
  root.dataset.palette = content.base
  const shape = content.shape!
  const { bg, cardOpacity } = resolveBackground(content, pageKey, override)
  const glass = GLASS_TYPES.includes(bg.type as BgType)

  const rules: string[] = []
  const custom = hasTokenOverrides(content, pageKey)
  if (custom) {
    const keys = [...UI_TOKEN_KEYS]
    rules.push(`html[data-theme-custom][data-palette] { ${tokensToCssVars(resolveTokens(content, 'light', pageKey), keys)} }`)
    rules.push(`html.dark[data-theme-custom][data-palette] { ${tokensToCssVars(resolveTokens(content, 'dark', pageKey), keys)} }`)
  }

  // 圆角：主题设定卡片圆角 R（默认 22），「标准」偏好整体收小一档
  const r = (shape.radius ?? DEFAULT_RADIUS) * (radiusPref === 'standard' ? 0.72 : 1)
  const vars: string[] = [
    `--app-radius-xs: ${px((r * 8) / 22)}`,
    `--app-radius-sm: ${px((r * 12) / 22)}`,
    `--app-radius: ${px((r * 16) / 22)}`,
    `--app-radius-lg: ${px(r)}`,
    `--app-radius-xl: ${px((r * 28) / 22)}`,
    `--app-radius-btn: ${shape.button === 'square' ? px(Math.min(6, r / 3)) : shape.button === 'round' ? px((r * 12) / 22) : '999px'}`,
    `--theme-card-a: ${cardOpacity}%`,
  ]
  if (shape.font !== 'outfit') {
    let stack = FONT_STACKS[shape.font as keyof typeof FONT_STACKS]
    if (shape.font === 'custom') {
      loadCustomFont(assetUrl(shape.fontAsset, resolver))
      stack = `'${CUSTOM_FONT_FAMILY}', ${FONT_STACKS.outfit}`
    }
    if (stack) vars.push(`--app-font-sans: ${stack}`, `--app-font-display: ${stack}`)
  }
  if (shape.fontSize === 's') vars.push('font-size: 15px')
  if (shape.fontSize === 'l') vars.push('font-size: 17px')
  if (SHADOWS[shape.shadow]) vars.push(`--app-shadow-card: ${SHADOWS[shape.shadow]}`)
  rules.push(`html[data-theme-shape][data-palette] { ${vars.join('; ')} }`)
  // 组件尺寸：全局 + 当前页面的覆盖
  const sizeVars = sizesToCssVars(resolveSizes(content, pageKey))
  if (sizeVars) rules.push(`html[data-theme-shape][data-palette] { ${sizeVars} }`)

  ensureStyle().textContent = rules.join('\n')
  root.toggleAttribute('data-theme-custom', custom)
  root.setAttribute('data-theme-shape', '')

  const on = new Set<string>()
  if (shape.button !== 'pill') on.add(`theme-btn-${shape.button}`)
  if (shape.shadow !== 'none') on.add(`theme-shadow-${shape.shadow}`)
  on.add(`theme-sidebar-${shape.sidebar}`)
  if (shape.pill === 'fill') on.add('theme-pill-fill')
  if (shape.cardBorder === false) on.add('theme-no-card-border')
  if (shape.stickerTilt === false) on.add('theme-no-tilt')
  if (bg.type !== 'theme') on.add('has-theme-bg')
  if (glass) on.add('is-glass')
  for (const c of ROOT_CLASSES) root.classList.toggle(c, on.has(c))
}
